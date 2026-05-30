package event

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/agent-orchestrator/backend/internal/models"
)

// Fanout manages SSE client connections for a project.
// SSE scale: max 50 concurrent clients per project (NFR-02-005).
type Fanout struct {
	mu      sync.RWMutex
	clients map[string]*SSEClient // clientID -> client
}

type SSEClient struct {
	ID        string
	ProjectID string
	EventCh   chan []byte
	CreatedAt time.Time
	LastEventID string
}

// NewFanout creates a new event fanout system.
func NewFanout() *Fanout {
	return &Fanout{clients: make(map[string]*SSEClient)}
}

// AddClient registers a new SSE client for a project.
// Returns the new client or an error if the project limit is reached.
func (f *Fanout) AddClient(clientID, projectID, lastEventID string) (*SSEClient, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Count existing clients for this project
	count := 0
	for _, c := range f.clients {
		if c.ProjectID == projectID {
			count++
		}
	}
	if count >= 50 {
		return nil, fmt.Errorf("max SSE clients reached (50)")
	}

	client := &SSEClient{
		ID:         clientID,
		ProjectID:  projectID,
		EventCh:   make(chan []byte, 64),
		CreatedAt: time.Now(),
		LastEventID: lastEventID,
	}
	f.clients[clientID] = client
	return client, nil
}

// RemoveClient disconnects a client.
func (f *Fanout) RemoveClient(clientID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if c, ok := f.clients[clientID]; ok {
		close(c.EventCh)
		delete(f.clients, clientID)
	}
}

// Broadcast sends an event to all clients subscribed to the project.
func (f *Fanout) Broadcast(projectID string, topic string, event models.AuditEvent) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("event marshal error: %v", err)
		return
	}

	for _, client := range f.clients {
		if client.ProjectID == projectID {
			select {
			case client.EventCh <- data:
			default:
				// Channel full; skip this client
			}
		}
	}
}

// ClientCount returns the number of connected clients for a project.
func (f *Fanout) ClientCount(projectID string) int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	count := 0
	for _, c := range f.clients {
		if c.ProjectID == projectID {
			count++
		}
	}
	return count
}

// EventService handles event emission to SSE and webhooks.
type EventService struct {
	fanout   *Fanout
	webhooks *WebhookQueue
}

// NewEventService creates a new event service.
func NewEventService(webhookQueue *WebhookQueue) *EventService {
	return &EventService{
		fanout:   NewFanout(),
		webhooks: webhookQueue,
	}
}

// Emit sends an event to SSE clients and enqueues webhook deliveries.
// This is async/non-blocking for webhooks (NFR-02-012).
func (s *EventService) Emit(ctx context.Context, event models.AuditEvent) {
	// SSE broadcast (sync, within latency budget)
	s.fanout.Broadcast(event.ProjectID, event.Topic, event)

	// Webhook enqueue (async, non-blocking per NFR-02-010)
	s.webhooks.Enqueue(event)
}

// GetFanout returns the SSE fanout for SSE stream handlers.
func (s *EventService) GetFanout() *Fanout {
	return s.fanout
}

// SSEClientCount returns the number of connected SSE clients for a project.
func (s *EventService) SSEClientCount(projectID string) int {
	return s.fanout.ClientCount(projectID)
}

// WebhookRepo defines the subset of WebhookRepository methods needed by WebhookQueue.
type WebhookRepo interface {
	ListActiveByProject(ctx context.Context, projectID string) ([]models.WebhookRegistration, error)
	EnqueueDelivery(ctx context.Context, webhookID, eventID string, payload []byte) error
	GetPendingDeliveries(ctx context.Context, limit int) ([]models.DeliveryRecord, error)
	DeleteDelivery(ctx context.Context, id int) error
	RescheduleDelivery(ctx context.Context, id int, attempt int) error
	UpdateDeliveryStatus(ctx context.Context, webhookID string, success bool) error
}

// DeliveryRecord represents a pending delivery row from the DB.
// (Type alias kept here for internal use; logically it's models.DeliveryRecord.)
type DeliveryRecord = models.DeliveryRecord

// WebhookQueue manages the webhook delivery queue (DB-backed per NFR-02-015).
type WebhookQueue struct {
	mu          sync.Mutex
	pending     []models.AuditEvent
	webhookRepo WebhookRepo
	maxRetries  int
}

// NewWebhookQueue creates a webhook queue.
func NewWebhookQueue() *WebhookQueue {
	return &WebhookQueue{
		pending:    []models.AuditEvent{},
		maxRetries:  3,
	}
}

// NewWebhookQueueWithRepo creates a webhook queue with the given repo injected.
// Use this in tests to inject a mock WebhookRepo.
func NewWebhookQueueWithRepo(repo WebhookRepo) *WebhookQueue {
	return &WebhookQueue{
		pending:    []models.AuditEvent{},
		webhookRepo: repo,
		maxRetries:  3,
	}
}

// SetWebhookRepo sets the webhook repository for looking up registrations and queue operations.
func (q *WebhookQueue) SetWebhookRepo(repo WebhookRepo) {
	q.webhookRepo = repo
}

// Enqueue adds an event to the webhook delivery queue.
// Webhook delivery is async and MUST NOT block the caller (FR-02-024).
func (q *WebhookQueue) Enqueue(event models.AuditEvent) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pending = append(q.pending, event)
}

// ProcessQueue delivers pending webhooks from the in-process queue AND retries from DB.
// Called by the background worker.
func (q *WebhookQueue) ProcessQueue(ctx context.Context) {
	q.processInProcessQueue(ctx)
	q.processDBQueue(ctx)
}

// processInProcessQueue drains the in-process pending queue.
func (q *WebhookQueue) processInProcessQueue(ctx context.Context) {
	q.mu.Lock()
	pending := q.pending
	q.pending = []models.AuditEvent{}
	q.mu.Unlock()

	for _, event := range pending {
		q.deliverEvent(ctx, event)
	}
}

// processDBQueue selects due deliveries, attempts delivery, handles retry or deletion.
func (q *WebhookQueue) processDBQueue(ctx context.Context) {
	if q.webhookRepo == nil {
		return
	}

	deliveries, err := q.webhookRepo.GetPendingDeliveries(ctx, 20)
	if err != nil {
		log.Printf("webhook: failed to get pending deliveries: %v", err)
		return
	}

	for _, d := range deliveries {
		q.attemptDelivery(ctx, d)
	}
}

// attemptDelivery attempts HTTP delivery for one delivery record.
// On success: deletes the row. On failure: reschedules or gives up.
func (q *WebhookQueue) attemptDelivery(ctx context.Context, d models.DeliveryRecord) {
	webhooks, err := q.webhookRepo.ListActiveByProject(ctx, d.WebhookID)
	if err != nil || len(webhooks) == 0 {
		// Webhook deleted or project gone — clean up the delivery
		q.webhookRepo.DeleteDelivery(ctx, d.ID)
		return
	}
	wh := webhooks[0]

	// Attempt HTTP POST
	body := bytes.NewReader(d.Payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, body)
	if err != nil {
		q.handleFailedDelivery(ctx, d, wh)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// ADR-02-003: sign non-localhost webhooks
	if !isLocalhostURL(wh.URL) && wh.Secret != "" {
		sig := computeSignature(d.Payload, wh.Secret)
		req.Header.Set("X-Webhook-Signature", sig)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		q.handleFailedDelivery(ctx, d, wh)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Success
		q.webhookRepo.DeleteDelivery(ctx, d.ID)
		q.webhookRepo.UpdateDeliveryStatus(ctx, wh.ID, true)
		// Emit webhook.delivered event
		q.emitWebhookEvent(ctx, wh, d, "webhook.delivery.succeeded")
	} else {
		// Unexpected response — treat as failure
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("webhook: delivery failed for %s: status %d body %s", wh.ID, resp.StatusCode, string(bodyBytes))
		q.handleFailedDelivery(ctx, d, wh)
	}
}

// handleFailedDelivery handles retry logic for a failed delivery.
// After max retries, emits webhook.delivery.failed and deletes the row.
func (q *WebhookQueue) handleFailedDelivery(ctx context.Context, d models.DeliveryRecord, wh models.WebhookRegistration) {
	nextAttempt := d.Attempts + 1
	if nextAttempt > q.maxRetries {
		// Exhausted retries — emit failure event and drop
		q.webhookRepo.DeleteDelivery(ctx, d.ID)
		q.webhookRepo.UpdateDeliveryStatus(ctx, wh.ID, false)
		q.emitWebhookEvent(ctx, wh, d, "webhook.delivery.failed")
	} else {
		// Reschedule with exponential backoff
		q.webhookRepo.RescheduleDelivery(ctx, d.ID, nextAttempt)
	}
}

// emitWebhookEvent emits a webhook delivery event via SSE only.
// Called without a webhook queue to prevent recursive enqueue.
func (q *WebhookQueue) emitWebhookEvent(ctx context.Context, wh models.WebhookRegistration, d models.DeliveryRecord, topic string) {
	// Broadcast only to SSE — no recursive webhook enqueue (avoid cycles)
	event := models.AuditEvent{
		EventID:       newWebhookEventID(),
		SchemaVersion: "v1alpha",
		ProjectID:     wh.ProjectID,
		Topic:         topic,
		ActorID:       "system",
		ActorRole:     "system",
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"webhookId": wh.ID,
			"eventId":  d.EventID,
		},
	}
	fanout := NewFanout()
	fanout.Broadcast(wh.ProjectID, topic, event)
}

// deliverEvent delivers an event to all registered webhooks for the project.
func (q *WebhookQueue) deliverEvent(ctx context.Context, event models.AuditEvent) {
	if q.webhookRepo == nil {
		return
	}

	webhooks, err := q.webhookRepo.ListActiveByProject(ctx, event.ProjectID)
	if err != nil {
		log.Printf("webhook lookup failed: %v", err)
		return
	}

	for _, wh := range webhooks {
		if MatchesTopic(wh.Events, event.Topic) {
			q.enqueueDelivery(ctx, wh, event)
		}
	}
}

// matchesTopic returns true if the event topic matches the webhook's event selector.
func MatchesTopic(events []string, topic string) bool {
	for _, e := range events {
		if e == topic || e == "*" {
			return true
		}
		// Prefix match (e.g., "task.*" matches "task.created")
		if len(e) > 0 && e[len(e)-1] == '*' && len(topic) >= len(e)-1 {
			prefix := e[:len(e)-1]
			if len(topic) >= len(prefix) && topic[:len(prefix)] == prefix {
				return true
			}
		}
	}
	return false
}

// enqueueDelivery stores a webhook delivery in the DB queue (non-blocking).
func (q *WebhookQueue) enqueueDelivery(ctx context.Context, wh models.WebhookRegistration, event models.AuditEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("webhook: failed to marshal event %s: %v", event.EventID, err)
		return
	}
	err = q.webhookRepo.EnqueueDelivery(ctx, wh.ID, event.EventID, payload)
	if err != nil {
		log.Printf("webhook: failed to enqueue delivery for webhook %s event %s: %v", wh.ID, event.EventID, err)
	}
}

// computeSignature computes HMAC-SHA256 signature for webhook delivery (ADR-02-003).
func computeSignature(body []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// isLocalhostURL returns true if the URL is localhost or 127.0.0.1.
func isLocalhostURL(url string) bool {
	return strings.HasPrefix(url, "http://localhost") ||
		strings.HasPrefix(url, "http://127.0.0.1") ||
		strings.HasPrefix(url, "https://localhost") ||
		strings.HasPrefix(url, "https://127.0.0.1")
}

func newWebhookEventID() string {
	return fmt.Sprintf("ev_wb_%d_%08x", time.Now().UnixNano(), time.Now().UnixNano()&0xffffffff)
}
