package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/agent-orchestrator/backend/internal/models"
)

// WebhookRepository handles webhook registration CRUD.
type WebhookRepository struct {
	db *Pool
}

// NewWebhookRepository creates a webhook repository.
func NewWebhookRepository(db *Pool) *WebhookRepository {
	return &WebhookRepository{db: db}
}

// DB returns the underlying pool for use by other components.
func (r *WebhookRepository) DB() *Pool { return r.db }

// InsertAuthDeniedEvent inserts an auth.mutation.denied event directly into the DB.
// Called by the middleware to record authorization failures (NFR-02-014).
func (r *WebhookRepository) InsertAuthDeniedEvent(ctx context.Context, projectID, actorID, actorRole, attemptedAction, deniedReason string) error {
	eventID := fmt.Sprintf("ev_%d", time.Now().UnixNano())
	timestamp := time.Now().UTC().Format(time.RFC3339Nano)
	payload := map[string]any{
		"actorId":         actorID,
		"actorRole":       actorRole,
		"attemptedAction": attemptedAction,
		"deniedReason":    deniedReason,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO audit_events (event_id, schema_version, project_id, topic, actor_id, actor_role, task_id, parent_task_id, gate_id, timestamp, payload)
		VALUES ($1, $2, $3, $4, $5, $6, NULL, NULL, NULL, $7, $8)`,
		eventID, "v1alpha", projectID, "auth.mutation.denied", actorID, actorRole, timestamp, payloadJSON)
	return err
}

// Create creates a new webhook registration.
func (r *WebhookRepository) Create(ctx context.Context, w *models.WebhookRegistration) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO webhook_registrations (id, project_id, url, events, active, secret_hash, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		w.ID, w.ProjectID, w.URL, w.Events, w.Active, w.Secret, w.CreatedAt)
	return err
}

// ListActiveByProject returns all active webhook registrations for a project.
func (r *WebhookRepository) ListActiveByProject(ctx context.Context, projectID string) ([]models.WebhookRegistration, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, project_id, url, events, active, delivery_last_attempt_at, delivery_last_success_at, delivery_failure_count, created_at
		FROM webhook_registrations WHERE project_id=$1 AND active=true`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []models.WebhookRegistration
	for rows.Next() {
		var w models.WebhookRegistration
		var ds models.WebhookDeliveryStatus
		var lastAttempt, lastSuccess *string
		if err := rows.Scan(&w.ID, &w.ProjectID, &w.URL, &w.Events, &w.Active, &lastAttempt, &lastSuccess, &ds.FailureCount, &w.CreatedAt); err != nil {
			return nil, err
		}
		w.DeliveryStatus = &ds
		webhooks = append(webhooks, w)
	}
	return webhooks, nil
}

// Delete removes a webhook registration.
func (r *WebhookRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM webhook_registrations WHERE id=$1`, id)
	return err
}

// UpdateDeliveryStatus updates the delivery status after a delivery attempt.
func (r *WebhookRepository) UpdateDeliveryStatus(ctx context.Context, id string, success bool) error {
	if success {
		_, err := r.db.Exec(ctx, `
			UPDATE webhook_registrations SET delivery_last_success_at=NOW() WHERE id=$1`, id)
		return err
	}
	_, err := r.db.Exec(ctx, `
		UPDATE webhook_registrations SET delivery_last_attempt_at=NOW(), delivery_failure_count=delivery_failure_count+1 WHERE id=$1`, id)
	return err
}

// EnqueueDelivery adds a webhook delivery to the queue table.
func (r *WebhookRepository) EnqueueDelivery(ctx context.Context, webhookID, eventID string, payload []byte) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO webhook_delivery_queue (webhook_id, event_id, payload, attempt_count, next_retry_at)
		VALUES ($1, $2, $3, 0, NOW())`, webhookID, eventID, payload)
	return err
}

// GetPendingDeliveries returns deliveries due for retry.
func (r *WebhookRepository) GetPendingDeliveries(ctx context.Context, limit int) ([]models.DeliveryRecord, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, webhook_id, event_id, payload, attempt_count
		FROM webhook_delivery_queue
		WHERE next_retry_at <= NOW()
		ORDER BY next_retry_at ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []models.DeliveryRecord
	for rows.Next() {
		var d models.DeliveryRecord
		if err := rows.Scan(&d.ID, &d.WebhookID, &d.EventID, &d.Payload, &d.Attempts); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, d)
	}
	return deliveries, nil
}

// DeleteDelivery removes a delivery record.
func (r *WebhookRepository) DeleteDelivery(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM webhook_delivery_queue WHERE id=$1`, id)
	return err
}

// RescheduleDelivery reschedules a failed delivery for retry with exponential backoff.
func (r *WebhookRepository) RescheduleDelivery(ctx context.Context, id int, attempt int) error {
	backoff := 1 << attempt // 1s, 2s, 4s
	_, err := r.db.Exec(ctx, `
		UPDATE webhook_delivery_queue SET attempt_count=$2, next_retry_at=NOW() + ($3 || ' seconds')::interval
		WHERE id=$1`, id, attempt, backoff)
	return err
}