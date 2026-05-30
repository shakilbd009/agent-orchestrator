package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/agent-orchestrator/backend/internal/models"
)

// GateRepository handles gate CRUD.
type GateRepository struct {
	db *Pool
}

// NewGateRepository creates a gate repository.
func NewGateRepository(db *Pool) *GateRepository {
	return &GateRepository{db: db}
}

// CreateTaskGate creates a task-level gate.
func (r *GateRepository) CreateTaskGate(ctx context.Context, g *models.TaskGate, actor *models.Actor) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO task_gates (id, project_id, task_id, phase, state, criteria, blocking, passed_at, passed_by, override_note, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		g.ID, g.ProjectID, g.TaskID, g.Phase, g.State, g.Criteria, g.Blocking, g.PassedAt, g.PassedBy, g.OverrideNote, g.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert gate: %w", err)
	}

	event := models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     g.ProjectID,
		Topic:         "gate.opened",
		ActorID:       actor.ID,
		ActorRole:     actor.Role,
		TaskID:        &g.TaskID,
		ParentTaskID:  nil,
		GateID:        &g.ID,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"gateType":  g.Phase,
			"gateLevel": "task",
			"blocking":  g.Blocking,
			"openedBy":  actor.ID,
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

// GetTaskGate retrieves a specific gate.
func (r *GateRepository) GetTaskGate(ctx context.Context, projectID, taskID, gateID string) (*models.TaskGate, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, project_id, task_id, phase, state, criteria, blocking, passed_at, passed_by, override_note, created_at
		FROM task_gates WHERE id=$1 AND task_id=$2 AND project_id=$3`, gateID, taskID, projectID)

	var g models.TaskGate
	var passedBy string
	var passedAt *time.Time
	err := row.Scan(&g.ID, &g.ProjectID, &g.TaskID, &g.Phase, &g.State, &g.Criteria, &g.Blocking, &passedAt, &passedBy, &g.OverrideNote, &g.CreatedAt)
	if err != nil {
		return nil, err
	}
	g.PassedAt = passedAt
	g.PassedBy = passedBy
	return &g, nil
}

// UpdateGateState updates a gate's state (approve/reject).
func (r *GateRepository) UpdateGateState(ctx context.Context, projectID, taskID, gateID, newState, actorID, actorRole, overrideNote string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	var passedAt interface{} = nil
	if newState == "passed" {
		now := time.Now()
		passedAt = &now
	}

	_, err = tx.Exec(ctx, `
		UPDATE task_gates SET state=$4, passed_at=$5, passed_by=$6, override_note=$7
		WHERE id=$1 AND task_id=$2 AND project_id=$3`,
		gateID, taskID, projectID, newState, passedAt, actorID, overrideNote)
	if err != nil {
		return fmt.Errorf("update gate: %w", err)
	}

	topic := "gate.approved"
	if newState == "blocked" {
		topic = "gate.rejected"
	}
	event := models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     projectID,
		Topic:         topic,
		ActorID:       actorID,
		ActorRole:     actorRole,
		TaskID:        &taskID,
		ParentTaskID:  nil,
		GateID:        &gateID,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"approvedBy":  actorID,
			"approverRole": actorRole,
			"gateType":    "task_gate",
			"gateLevel":   "task",
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

// CreateProjectPhaseGate creates a project-level phase gate.
func (r *GateRepository) CreateProjectPhaseGate(ctx context.Context, g *models.ProjectPhaseGate) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO project_phase_gates (id, project_id, phase_index, phase, state, criteria, pass_condition, passed_at, passed_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		g.ID, g.ProjectID, g.PhaseIndex, g.Phase, g.State, g.Criteria, g.PassCondition, g.PassedAt, g.PassedBy, g.CreatedAt)
	return err
}

// ListProjectPhaseGates returns all phase gates for a project.
func (r *GateRepository) ListProjectPhaseGates(ctx context.Context, projectID string) ([]models.ProjectPhaseGate, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, project_id, phase_index, phase, state, criteria, pass_condition, passed_at, passed_by, created_at
		FROM project_phase_gates WHERE project_id=$1 ORDER BY phase_index ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gates []models.ProjectPhaseGate
	for rows.Next() {
		var g models.ProjectPhaseGate
		var passedAt *time.Time
		var passedBy string
		err := rows.Scan(&g.ID, &g.ProjectID, &g.PhaseIndex, &g.Phase, &g.State, &g.Criteria, &g.PassCondition, &passedAt, &passedBy, &g.CreatedAt)
		if err != nil {
			return nil, err
		}
		g.PassedAt = passedAt
		g.PassedBy = passedBy
		gates = append(gates, g)
	}
	return gates, nil
}

// GetProjectPhaseGate retrieves a specific phase gate.
func (r *GateRepository) GetProjectPhaseGate(ctx context.Context, projectID, gateID string) (*models.ProjectPhaseGate, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, project_id, phase_index, phase, state, criteria, pass_condition, passed_at, passed_by, created_at
		FROM project_phase_gates WHERE id=$1 AND project_id=$2`, gateID, projectID)

	var g models.ProjectPhaseGate
	var passedAt *time.Time
	var passedBy string
	err := row.Scan(&g.ID, &g.ProjectID, &g.PhaseIndex, &g.Phase, &g.State, &g.Criteria, &g.PassCondition, &passedAt, &passedBy, &g.CreatedAt)
	if err != nil {
		return nil, err
	}
	g.PassedAt = passedAt
	g.PassedBy = passedBy
	return &g, nil
}

// UpdateProjectPhaseGate updates a phase gate.
func (r *GateRepository) UpdateProjectPhaseGate(ctx context.Context, g *models.ProjectPhaseGate) error {
	_, err := r.db.Exec(ctx, `
		UPDATE project_phase_gates SET state=$2, criteria=$3, pass_condition=$4, passed_at=$5, passed_by=$6
		WHERE id=$1 AND project_id=$7`,
		g.ID, g.State, g.Criteria, g.PassCondition, g.PassedAt, g.PassedBy, g.ProjectID)
	return err
}