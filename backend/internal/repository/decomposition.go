package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/agent-orchestrator/backend/internal/models"
)

// DecompositionRepository handles decomposition proposal lifecycle.
type DecompositionRepository struct {
	db *Pool
}

// NewDecompositionRepository creates a decomposition repository.
func NewDecompositionRepository(db *Pool) *DecompositionRepository {
	return &DecompositionRepository{db: db}
}

// SubmitProposal submits a new decomposition proposal or overrides an existing one.
// FR-02-008/009: Proposed children do NOT affect assignment/readiness until approved.
// FR-02-009A: Only one active proposal per parent at a time.
// proposedDepth and totalChildren are computed by the service layer (via CheckDecompositionLimits)
// and passed in so the audit event records accurate values.
func (r *DecompositionRepository) SubmitProposal(ctx context.Context, dp *models.DecompositionProposal, proposedDepth, totalChildren int, override bool, actorID, actorRole, overrideReason string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check for existing active proposal
	row := tx.QueryRow(ctx, `
		SELECT id, state FROM decomposition_proposals
		WHERE parent_task_id=$1 AND state IN ('submitted', 'accepted')`, dp.ParentTaskID)

	var existingID, existingState string
	err = row.Scan(&existingID, &existingState)
	if err == nil {
		// Active proposal exists
		if !override {
			return fmt.Errorf("active proposal already exists for parent")
		}
		// Override: mark existing as rejected
		_, err = tx.Exec(ctx, `UPDATE decomposition_proposals SET state='rejected', updated_at=NOW() WHERE id=$1`, existingID)
		if err != nil {
			return fmt.Errorf("reject existing: %w", err)
		}
	}

	proposedTasksJSON, err := json.Marshal(dp.ProposedTasks)
	if err != nil {
		return fmt.Errorf("marshal proposed tasks: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO decomposition_proposals (id, project_id, parent_task_id, submitter, state, proposed_tasks, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		dp.ID, dp.ProjectID, dp.ParentTaskID, dp.Submitter, dp.State, proposedTasksJSON, dp.CreatedAt, dp.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert proposal: %w", err)
	}

	// Audit event (AC-02-008: depthAtProposal and activeChildrenCount from service-computed values)
	var topic string
	if override {
		topic = "task.decomposition.override_used"
	} else {
		topic = "task.decomposition.proposed"
	}

	var parentTaskID *string = &dp.ParentTaskID
	event := models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     dp.ProjectID,
		Topic:         topic,
		ActorID:       actorID,
		ActorRole:     actorRole,
		TaskID:        parentTaskID,
		ParentTaskID:  nil,
		GateID:        nil,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"parentTaskId":         dp.ParentTaskID,
			"depthAtProposal":       proposedDepth,
			"activeChildrenCount":   totalChildren,
			"override":              override,
			"overrideReason":        overrideReason,
			"actorId":               actorID,
			"actorRole":             actorRole,
		},
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_events (event_id, schema_version, project_id, topic, actor_id, actor_role, task_id, parent_task_id, gate_id, timestamp, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		event.EventID, event.SchemaVersion, event.ProjectID, event.Topic, event.ActorID, event.ActorRole,
		event.TaskID, event.ParentTaskID, event.GateID, event.Timestamp, event.Payload)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}

	return tx.Commit(ctx)
}

// ApproveProposal approves a decomposition proposal, activating child tasks.
func (r *DecompositionRepository) ApproveProposal(ctx context.Context, proposalID, actorID, actorRole string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get proposal
	row := tx.QueryRow(ctx, `
		SELECT id, project_id, parent_task_id, proposed_tasks FROM decomposition_proposals
		WHERE id=$1 AND state='submitted'`, proposalID)
	var proposal struct {
		ID          string
		ProjectID   string
		ParentTaskID string
		TasksJSON   []byte
	}
	if err := row.Scan(&proposal.ID, &proposal.ProjectID, &proposal.ParentTaskID, &proposal.TasksJSON); err != nil {
		return fmt.Errorf("proposal not found: %w", err)
	}

	// Update proposal state
	_, err = tx.Exec(ctx, `UPDATE decomposition_proposals SET state='accepted', updated_at=NOW() WHERE id=$1`, proposalID)
	if err != nil {
		return fmt.Errorf("update proposal: %w", err)
	}

	event := models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     proposal.ProjectID,
		Topic:         "task.decomposition.approved",
		ActorID:       actorID,
		ActorRole:     actorRole,
		TaskID:        &proposal.ParentTaskID,
		ParentTaskID:  nil,
		GateID:        nil,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"parentTaskId":  proposal.ParentTaskID,
			"approvedBy":    actorID,
			"approvedByRole": actorRole,
			"depthUsed":     1,
		},
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_events (event_id, schema_version, project_id, topic, actor_id, actor_role, task_id, parent_task_id, gate_id, timestamp, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		event.EventID, event.SchemaVersion, event.ProjectID, event.Topic, event.ActorID, event.ActorRole,
		event.TaskID, event.ParentTaskID, event.GateID, event.Timestamp, event.Payload)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}

	return tx.Commit(ctx)
}

// RejectProposal rejects a decomposition proposal.
func (r *DecompositionRepository) RejectProposal(ctx context.Context, proposalID, actorID, reason string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `SELECT id, project_id, parent_task_id FROM decomposition_proposals WHERE id=$1`, proposalID)
	var proposal struct {
		ID          string
		ProjectID   string
		ParentTaskID string
	}
	if err := row.Scan(&proposal.ID, &proposal.ProjectID, &proposal.ParentTaskID); err != nil {
		return fmt.Errorf("proposal not found: %w", err)
	}

	_, err = tx.Exec(ctx, `UPDATE decomposition_proposals SET state='rejected', updated_at=NOW() WHERE id=$1`, proposalID)
	if err != nil {
		return fmt.Errorf("update proposal: %w", err)
	}

	var parentTaskID *string = &proposal.ParentTaskID
	event := models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     proposal.ProjectID,
		Topic:         "task.decomposition.rejected",
		ActorID:       actorID,
		ActorRole:     "human",
		TaskID:        parentTaskID,
		ParentTaskID:  nil,
		GateID:        nil,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"parentTaskId":    proposal.ParentTaskID,
			"rejectedBy":      actorID,
			"rejectionReason": reason,
			"proposalRetained": false,
		},
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_events (event_id, schema_version, project_id, topic, actor_id, actor_role, task_id, parent_task_id, gate_id, timestamp, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		event.EventID, event.SchemaVersion, event.ProjectID, event.Topic, event.ActorID, event.ActorRole,
		event.TaskID, event.ParentTaskID, event.GateID, event.Timestamp, event.Payload)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}

	return tx.Commit(ctx)
}

// GetProposal retrieves a decomposition proposal by ID.
func (r *DecompositionRepository) GetProposal(ctx context.Context, id string) (*models.DecompositionProposal, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, project_id, parent_task_id, submitter, state, proposed_tasks, created_at, updated_at
		FROM decomposition_proposals WHERE id=$1`, id)

	var dp models.DecompositionProposal
	var tasksJSON []byte
	err := row.Scan(&dp.ID, &dp.ProjectID, &dp.ParentTaskID, &dp.Submitter, &dp.State, &tasksJSON, &dp.CreatedAt, &dp.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(tasksJSON, &dp.ProposedTasks); err != nil {
		return nil, fmt.Errorf("unmarshal tasks: %w", err)
	}
	return &dp, nil
}

// GetActiveProposalForParent returns the active proposal for a parent task (FR-02-009A).
func (r *DecompositionRepository) GetActiveProposalForParent(ctx context.Context, parentTaskID string) (*models.DecompositionProposal, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, project_id, parent_task_id, submitter, state, proposed_tasks, created_at, updated_at
		FROM decomposition_proposals WHERE parent_task_id=$1 AND state='submitted' LIMIT 1`, parentTaskID)

	var dp models.DecompositionProposal
	var tasksJSON []byte
	err := row.Scan(&dp.ID, &dp.ProjectID, &dp.ParentTaskID, &dp.Submitter, &dp.State, &tasksJSON, &dp.CreatedAt, &dp.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(tasksJSON, &dp.ProposedTasks); err != nil {
		return nil, fmt.Errorf("unmarshal tasks: %w", err)
	}
	return &dp, nil
}
