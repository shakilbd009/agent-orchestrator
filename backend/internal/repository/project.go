package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/agent-orchestrator/backend/internal/models"
)

// Pool wraps pgxpool.Pool for repository methods.
type Pool struct {
	*pgxpool.Pool
}

// NewPool creates a new Pool from a PostgreSQL connection string.
func NewPool(ctx context.Context, connString string) (*Pool, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &Pool{Pool: pool}, nil
}

// Close closes the underlying pgx pool.
func (p *Pool) Close() { p.Pool.Close() }

func newID(prefix string) string {
	return prefix + "_" + uuid.New().String()[:12]
}

func timePtr(t time.Time) *time.Time { return &t }
func strPtr(s string) *string        { return &s }

// ProjectRepository handles project CRUD with project_id isolation (FR-02-001).
type ProjectRepository struct {
	db *Pool
}

// NewProjectRepository creates a project repository.
func NewProjectRepository(db *Pool) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create creates a new project and emits project.created audit event in one transaction (ADR-02-002).
func (r *ProjectRepository) Create(ctx context.Context, p *models.Project) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO projects (id, name, description, owner, status, phase, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		p.ID, p.Name, p.Description, p.Owner, p.Status, p.Phase, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert project: %w", err)
	}

	event := models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     p.ID,
		Topic:         "project.created",
		ActorID:       p.Owner,
		ActorRole:     "human",
		TaskID:        nil,
		ParentTaskID:  nil,
		GateID:        nil,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"projectName":               p.Name,
			"createdBy":                p.Owner,
			"staleThresholdMinutes":    nil,
			"decompositionDepthDefault": 3,
			"decompositionFanOutDefault": 20,
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

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// GetByID retrieves a project by ID.
func (r *ProjectRepository) GetByID(ctx context.Context, id string) (*models.Project, error) {
	row := r.db.QueryRow(ctx, `SELECT id, name, description, owner, status, phase, created_at, updated_at FROM projects WHERE id=$1`, id)
	var p models.Project
	var desc, owner string
	err := row.Scan(&p.ID, &p.Name, &desc, &owner, &p.Status, &p.Phase, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.Description = desc
	p.Owner = owner
	return &p, nil
}

// Update updates project metadata.
func (r *ProjectRepository) Update(ctx context.Context, p *models.Project) error {
	_, err := r.db.Exec(ctx, `
		UPDATE projects SET name=$2, description=$3, owner=$4, status=$5, phase=$6, updated_at=NOW()
		WHERE id=$1`, p.ID, p.Name, p.Description, p.Owner, p.Status, p.Phase)
	return err
}

// Delete archives a project (soft delete).
func (r *ProjectRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `UPDATE projects SET status='archived', updated_at=NOW() WHERE id=$1`, id)
	return err
}

// List returns all active projects.
func (r *ProjectRepository) List(ctx context.Context) ([]models.Project, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name, description, owner, status, phase, created_at, updated_at FROM projects WHERE status='active' ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		var desc, owner string
		if err := rows.Scan(&p.ID, &p.Name, &desc, &owner, &p.Status, &p.Phase, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Description = desc
		p.Owner = owner
		projects = append(projects, p)
	}
	return projects, nil
}

// GetStatistics returns aggregate task counts for a project.
func (r *ProjectRepository) GetStatistics(ctx context.Context, projectID string) (*models.ProjectStatistics, error) {
	row := r.db.QueryRow(ctx, `
		SELECT COUNT(*) as total,
			COUNT(*) FILTER (WHERE status='todo') as todo,
			COUNT(*) FILTER (WHERE status='in_progress') as in_progress,
			COUNT(*) FILTER (WHERE status='done') as done,
			COUNT(*) FILTER (WHERE status='blocked') as blocked
		FROM orchestration_tasks WHERE project_id=$1`, projectID)

	var stats models.ProjectStatistics
	err := row.Scan(&stats.TotalTasks, &stats.TodoTasks, &stats.InProgressTasks, &stats.DoneTasks, &stats.BlockedTasks)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// GetAuditEventsAfter returns audit events for a project that occurred after lastEventID.
// Used by SSE reconnect replay (AC-02-022A) to catch up missed events from the audit log.
// Query filters by project_id AND event_id > lastEventID, ordered by timestamp for causal order.
func (r *ProjectRepository) GetAuditEventsAfter(ctx context.Context, projectID, lastEventID string) ([]models.AuditEvent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT event_id, schema_version, project_id, topic, actor_id, actor_role,
		       task_id, parent_task_id, gate_id, timestamp, payload
		FROM audit_events
		WHERE project_id = $1 AND event_id > $2
		ORDER BY timestamp ASC`,
		projectID, lastEventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.AuditEvent
	for rows.Next() {
		var ev models.AuditEvent
		var taskID, parentTaskID, gateID *string
		var payloadMap []byte
		if err := rows.Scan(
			&ev.EventID, &ev.SchemaVersion, &ev.ProjectID, &ev.Topic,
			&ev.ActorID, &ev.ActorRole,
			&taskID, &parentTaskID, &gateID,
			&ev.Timestamp, &payloadMap,
		); err != nil {
			return nil, err
		}
		ev.TaskID = taskID
		ev.ParentTaskID = parentTaskID
		ev.GateID = gateID
		if err := json.Unmarshal(payloadMap, &ev.Payload); err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	return events, rows.Err()
}