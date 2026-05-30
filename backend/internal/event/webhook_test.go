package event

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/agent-orchestrator/backend/internal/models"
)

// mockWebhookRepo is a mock implementation of WebhookRepo for testing.
type mockWebhookRepo struct {
	mu                 sync.Mutex
	enqueued           []enqueuedCall
	pendingDeliveries  []models.DeliveryRecord
	deletedIDs         []int
	rescheduled        []struct{ id, attempt int }
	updateStatusCalls  []struct{ webhookID string; success bool }
	webhooks           []models.WebhookRegistration
}

type enqueuedCall struct {
	webhookID string
	eventID   string
	payload   []byte
}

func newMockWebhookRepo() *mockWebhookRepo {
	return &mockWebhookRepo{}
}

func (m *mockWebhookRepo) ListActiveByProject(ctx context.Context, projectID string) ([]models.WebhookRegistration, error) {
	return m.webhooks, nil
}

func (m *mockWebhookRepo) SetWebhooks(webhooks []models.WebhookRegistration) {
	m.webhooks = webhooks
}

func (m *mockWebhookRepo) EnqueueDelivery(ctx context.Context, webhookID, eventID string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enqueued = append(m.enqueued, enqueuedCall{webhookID: webhookID, eventID: eventID, payload: payload})
	return nil
}

func (m *mockWebhookRepo) GetPendingDeliveries(ctx context.Context, limit int) ([]models.DeliveryRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pendingDeliveries, nil
}

func (m *mockWebhookRepo) DeleteDelivery(ctx context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deletedIDs = append(m.deletedIDs, id)
	return nil
}

func (m *mockWebhookRepo) RescheduleDelivery(ctx context.Context, id int, attempt int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rescheduled = append(m.rescheduled, struct{ id, attempt int }{id, attempt})
	return nil
}

func (m *mockWebhookRepo) UpdateDeliveryStatus(ctx context.Context, webhookID string, success bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateStatusCalls = append(m.updateStatusCalls, struct{ webhookID string; success bool }{webhookID, success})
	return nil
}

// --- Tests for enqueueDelivery persistence ---

func TestEnqueueDelivery_PersistsViaRepo(t *testing.T) {
	repo := newMockWebhookRepo()
	q := NewWebhookQueueWithRepo(repo)

	wh := models.WebhookRegistration{
		ID:        "wh_test123",
		ProjectID: "project_1",
		URL:       "http://localhost:8080/webhook",
		Events:    []string{"task.created"},
		Secret:    "test-secret",
	}
	ev := models.AuditEvent{
		EventID:       "ev_123",
		SchemaVersion: "v1alpha",
		ProjectID:     "project_1",
		Topic:         "task.created",
		ActorID:       "user_1",
		ActorRole:     "human",
		Timestamp:    "2026-01-01T00:00:00Z",
		Payload:      map[string]any{"title": "Test task"},
	}

	q.enqueueDelivery(context.Background(), wh, ev)

	repo.mu.Lock()
	hasEnqueue := len(repo.enqueued) > 0
	repo.mu.Unlock()

	if !hasEnqueue {
		t.Error("expected EnqueueDelivery to be called on repo, but it was not")
	}
}

func TestEnqueueDelivery_PassesCorrectFields(t *testing.T) {
	repo := newMockWebhookRepo()
	q := NewWebhookQueueWithRepo(repo)

	wh := models.WebhookRegistration{
		ID:        "wh_abc",
		ProjectID: "p1",
		URL:       "http://localhost:8080/wh",
		Events:    []string{"task.*"},
	}
	ev := models.AuditEvent{
		EventID:       "ev_456",
		SchemaVersion: "v1alpha",
		ProjectID:     "p1",
		Topic:         "task.created",
		ActorID:       "user_1",
		ActorRole:     "human",
		Timestamp:    "2026-01-01T00:00:00Z",
		Payload:      map[string]any{"key": "value"},
	}

	q.enqueueDelivery(context.Background(), wh, ev)

	repo.mu.Lock()
	if len(repo.enqueued) != 1 {
		t.Fatalf("expected 1 enqueued call, got %d", len(repo.enqueued))
	}
	call := repo.enqueued[0]
	if call.webhookID != "wh_abc" {
		t.Errorf("expected webhookID 'wh_abc', got %q", call.webhookID)
	}
	if call.eventID != "ev_456" {
		t.Errorf("expected eventID 'ev_456', got %q", call.eventID)
	}
	// Verify payload is valid JSON containing the event
	var parsed map[string]any
	if err := json.Unmarshal(call.payload, &parsed); err != nil {
		t.Errorf("payload is not valid JSON: %v", err)
	}
	if parsed["eventId"] != "ev_456" {
		t.Errorf("expected eventId 'ev_456' in payload, got %v", parsed["eventId"])
	}
	repo.mu.Unlock()
}

func TestProcessQueue_DeletesOrphanedDelivery(t *testing.T) {
	repo := newMockWebhookRepo()
	repo.pendingDeliveries = []models.DeliveryRecord{
		{ID: 1, WebhookID: "wh_del", EventID: "ev_xyz", Payload: []byte(`{}`), Attempts: 0},
	}
	q := NewWebhookQueueWithRepo(repo)
	q.ProcessQueue(context.Background())

	repo.mu.Lock()
	deleted := repo.deletedIDs
	repo.mu.Unlock()

	// Since no webhooks are registered via ListActiveByProject, attemptDelivery
	// will treat it as orphaned and call DeleteDelivery
	if len(deleted) == 0 {
		t.Error("expected DeleteDelivery to be called for orphaned delivery")
	}
}

func TestIsLocalhostURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"http://localhost:8080/webhook", true},
		{"http://127.0.0.1:8080/webhook", true},
		{"https://api.example.com/webhook", false},
		{"http://192.168.1.1/webhook", false},
	}

	for _, tc := range tests {
		result := isLocalhostURL(tc.url)
		if result != tc.expected {
			t.Errorf("isLocalhostURL(%q) = %v, expected %v", tc.url, result, tc.expected)
		}
	}
}
