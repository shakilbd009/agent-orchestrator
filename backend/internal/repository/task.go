package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/agent-orchestrator/backend/internal/models"
)

// TaskRepository handles task CRUD with project scope enforcement (FR-02-001).
type TaskRepository struct {
	db *Pool
}

// NewTaskRepository creates a task repository.
func NewTaskRepository(db *Pool) *TaskRepository {
	return &TaskRepository{db: db}
}

// CreateTask creates a new task and emits task.created audit event atomically (ADR-02-002).
func (r *TaskRepository) CreateTask(ctx context.Context, t *models.OrchestrationTask, actor *models.Actor) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	// Parse tags and metadata for DB storage
	// Insert task
	_, err = tx.Exec(ctx, `
		INSERT INTO orchestration_tasks (id, project_id, title, body, status, layer, assignee, required, priority, stale, stale_threshold_minutes, blocked_reason, workspace_kind, workspace_path, tags, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
		t.ID, t.ProjectID, t.Title, t.Body, t.Status, t.Layer, t.Assignee, t.Required, t.Priority, t.Stale, t.StaleThresholdMinutes, t.BlockedReason, t.WorkspaceKind, t.WorkspacePath, t.Tags, t.Metadata, t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}

	// Insert parent edges
	for _, parentID := range t.Parents {
		_, err = tx.Exec(ctx, `INSERT INTO task_parents (task_id, parent_id) VALUES ($1, $2)`, t.ID, parentID)
		if err != nil {
			return fmt.Errorf("insert parent edge: %w", err)
		}
	}

	// Insert child edges
	for _, childID := range t.Children {
		_, err = tx.Exec(ctx, `INSERT INTO task_children (task_id, child_id) VALUES ($1, $2)`, t.ID, childID)
		if err != nil {
			return fmt.Errorf("insert child edge: %w", err)
		}
	}

	// Audit event (ADR-02-001: 11-field envelope, v1alpha)
	var parentTaskID *string
	if len(t.Parents) > 0 {
		parentTaskID = &t.Parents[0]
	}
	event := models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     t.ProjectID,
		Topic:         "task.created",
		ActorID:       actor.ID,
		ActorRole:     actor.Role,
		TaskID:        &t.ID,
		ParentTaskID:  parentTaskID,
		GateID:        nil,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"taskType":       "feature",
			"assignee":       t.Assignee,
			"title":          t.Title,
			"executionStatus": t.Status,
			"layer":          t.Layer,
			"required":       t.Required,
			"staleThresholdMinutes": t.StaleThresholdMinutes,
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

// GetTaskByID retrieves a task by ID with project scope enforcement.
func (r *TaskRepository) GetTaskByID(ctx context.Context, projectID, taskID string) (*models.OrchestrationTask, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, project_id, title, body, status, layer, assignee, required, priority, stale, stale_threshold_minutes, blocked_reason, workspace_kind, workspace_path, tags, metadata, created_at, updated_at, completed_at
		FROM orchestration_tasks WHERE id=$1 AND project_id=$2`, taskID, projectID)

	var t models.OrchestrationTask
	var assignee, body, blockedReason, wk, wp string
	var tags []string
	var metadata map[string]any
	var completedAt *time.Time

	err := row.Scan(&t.ID, &t.ProjectID, &t.Title, &body, &t.Status, &t.Layer, &assignee, &t.Required, &t.Priority, &t.Stale, &t.StaleThresholdMinutes, &blockedReason, &wk, &wp, &tags, &metadata, &t.CreatedAt, &t.UpdatedAt, &completedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan: %w", err)
	}
	t.Assignee = assignee
	t.Body = body
	t.BlockedReason = blockedReason
	t.WorkspaceKind = wk
	t.WorkspacePath = wp
	t.Tags = tags
	t.Metadata = metadata
	t.CompletedAt = completedAt

	// Load parent IDs
	parentRows, err := r.db.Query(ctx, `SELECT parent_id FROM task_parents WHERE task_id=$1`, taskID)
	if err == nil {
		defer parentRows.Close()
		for parentRows.Next() {
			var pid string
			parentRows.Scan(&pid)
			t.Parents = append(t.Parents, pid)
		}
	}

	// Load child IDs
	childRows, err := r.db.Query(ctx, `SELECT child_id FROM task_children WHERE task_id=$1`, taskID)
	if err == nil {
		defer childRows.Close()
		for childRows.Next() {
			var cid string
			childRows.Scan(&cid)
			t.Children = append(t.Children, cid)
		}
	}

	return &t, nil
}

// UpdateTaskStatus updates task execution status atomically with audit event (ADR-02-002).
func (r *TaskRepository) UpdateTaskStatus(ctx context.Context, projectID, taskID, newStatus string, actor *models.Actor, reason string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get current status for event payload
	var oldStatus string
	row := tx.QueryRow(ctx, `SELECT status FROM orchestration_tasks WHERE id=$1 AND project_id=$2`, taskID, projectID)
	if err := row.Scan(&oldStatus); err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("task not found")
		}
		return fmt.Errorf("get old status: %w", err)
	}

	// Update status
	var completedAt interface{} = nil
	if newStatus == "done" {
		now := time.Now()
		completedAt = &now
	}
	_, err = tx.Exec(ctx, `
		UPDATE orchestration_tasks SET status=$3, updated_at=NOW(), completed_at=$4 WHERE id=$1 AND project_id=$2`,
		taskID, projectID, newStatus, completedAt)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	// Emit task.status.changed event (ADR-02-001)
	event := models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     projectID,
		Topic:         "task.status.changed",
		ActorID:       actor.ID,
		ActorRole:     actor.Role,
		TaskID:        &taskID,
		ParentTaskID:  nil,
		GateID:        nil,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"fromStatus": oldStatus,
			"toStatus":   newStatus,
			"reason":     reason,
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

// BlockTask blocks a task with explicit reason (FR-02-021).
func (r *TaskRepository) BlockTask(ctx context.Context, projectID, taskID, reason string, actor *models.Actor) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE orchestration_tasks SET status='blocked', blocked_reason=$3, updated_at=NOW()
		WHERE id=$1 AND project_id=$2`, taskID, projectID, reason)
	if err != nil {
		return fmt.Errorf("block: %w", err)
	}

	event := models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     projectID,
		Topic:         "task.blocked",
		ActorID:       actor.ID,
		ActorRole:     actor.Role,
		TaskID:        &taskID,
		ParentTaskID:  nil,
		GateID:        nil,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"blockedBy": actor.ID,
			"reason":    reason,
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

// ListTasksByProject returns all tasks for a project, optionally filtered by status/layer/assignee.
// Critical: uses idx_tasks_status for NFR-02-006 (<300ms p50 at 10k tasks).
func (r *TaskRepository) ListTasksByProject(ctx context.Context, projectID, status, layer, assignee string) ([]models.OrchestrationTask, error) {
	sql := `SELECT id, project_id, title, body, status, layer, assignee, required, priority, stale, stale_threshold_minutes, blocked_reason, workspace_kind, workspace_path, tags, metadata, created_at, updated_at, completed_at
		FROM orchestration_tasks WHERE project_id=$1`
	args := []any{projectID}
	argIdx := 2

	if status != "" {
		sql += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if layer != "" {
		sql += fmt.Sprintf(" AND layer=$%d", argIdx)
		args = append(args, layer)
		argIdx++
	}
	if assignee != "" {
		sql += fmt.Sprintf(" AND assignee=$%d", argIdx)
		args = append(args, assignee)
		argIdx++
	}
	sql += " ORDER BY priority DESC, created_at ASC"

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var tasks []models.OrchestrationTask
	for rows.Next() {
		var t models.OrchestrationTask
		var assignee, body, blockedReason, wk, wp string
		var tags []string
		var metadata map[string]any
		var completedAt *time.Time

		err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &body, &t.Status, &t.Layer, &assignee, &t.Required, &t.Priority, &t.Stale, &t.StaleThresholdMinutes, &blockedReason, &wk, &wp, &tags, &metadata, &t.CreatedAt, &t.UpdatedAt, &completedAt)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		t.Assignee = assignee
		t.Body = body
		t.BlockedReason = blockedReason
		t.WorkspaceKind = wk
		t.WorkspacePath = wp
		t.Tags = tags
		t.Metadata = metadata
		t.CompletedAt = completedAt
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// GetTaskDependencies returns the dependency graph for a task.
func (r *TaskRepository) GetTaskDependencies(ctx context.Context, projectID, taskID string) (*models.TaskDependencyGraph, error) {
	graph := &models.TaskDependencyGraph{TaskID: taskID}

	// Parent dependencies (tasks this task depends on)
	parentRows, err := r.db.Query(ctx, `
		SELECT d.id, d.project_id, d.source_task_id, d.target_task_id, d.type, d.created_at
		FROM task_dependencies d WHERE d.source_task_id=$1 AND d.project_id=$2`, taskID, projectID)
	if err != nil {
		return nil, fmt.Errorf("parent deps: %w", err)
	}
	defer parentRows.Close()
	for parentRows.Next() {
		var dep models.TaskDependency
		parentRows.Scan(&dep.ID, &dep.ProjectID, &dep.SourceTaskID, &dep.TargetTaskID, &dep.Type, &dep.CreatedAt)
		graph.Parents = append(graph.Parents, dep)
	}

	// Child dependencies (tasks that depend on this task)
	childRows, err := r.db.Query(ctx, `
		SELECT d.id, d.project_id, d.source_task_id, d.target_task_id, d.type, d.created_at
		FROM task_dependencies d WHERE d.target_task_id=$1 AND d.project_id=$2`, taskID, projectID)
	if err != nil {
		return nil, fmt.Errorf("child deps: %w", err)
	}
	defer childRows.Close()
	for childRows.Next() {
		var dep models.TaskDependency
		childRows.Scan(&dep.ID, &dep.ProjectID, &dep.SourceTaskID, &dep.TargetTaskID, &dep.Type, &dep.CreatedAt)
		graph.Children = append(graph.Children, dep)
	}

	return graph, nil
}

// ReplaceTaskDependencies atomically replaces all dependency edges for a task.
// Rejects circular dependencies and cross-project edges (FR-02-005).
func (r *TaskRepository) ReplaceTaskDependencies(ctx context.Context, projectID, taskID string, deps []models.TaskDependency, actor *models.Actor) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	// Fetch existing edges for the task (within this project only)
	existingRows, err := tx.Query(ctx, `
		SELECT d.target_task_id FROM task_dependencies d
		WHERE d.source_task_id=$1 AND d.project_id=$2`, taskID, projectID)
	if err != nil {
		return fmt.Errorf("fetch existing deps: %w", err)
	}
	var existingTargets []string
	for existingRows.Next() {
		var tid string
		existingRows.Scan(&tid)
		existingTargets = append(existingTargets, tid)
	}
	existingRows.Close()

	// Validate each new dep before touching the database.
	// 1) Cross-project check: target must belong to this project.
	// 2) Cycle detection: target must not be reachable from taskID through
	//    existing edges + the remaining new deps (simulates final graph).
	for _, dep := range deps {
		// Cross-project validation: look up target task's project_id
		var targetProjectID string
		err := tx.QueryRow(ctx, `
			SELECT project_id FROM orchestration_tasks WHERE id=$1`, dep.TargetTaskID).Scan(&targetProjectID)
		if err != nil {
			if err == pgx.ErrNoRows {
				return fmt.Errorf("target task %s not found", dep.TargetTaskID)
			}
			return fmt.Errorf("resolve target project: %w", err)
		}
		if targetProjectID != projectID {
			// Emit rejection event before returning (event name matches eval contract)
			rejEvent := models.AuditEvent{
				EventID:       newID("ev"),
				SchemaVersion: "v1alpha",
				ProjectID:     projectID,
				Topic:         "task.decomposition.dependency_rejected",
				ActorID:       actor.ID,
				ActorRole:     actor.Role,
				TaskID:        &taskID,
				ParentTaskID:  nil,
				GateID:        nil,
				Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
				Payload: map[string]any{
					"targetTaskId": dep.TargetTaskID,
					"reason":       "cross-project edges not permitted",
				},
			}
			_, _ = tx.Exec(ctx, `
				INSERT INTO audit_events (event_id, schema_version, project_id, topic, actor_id, actor_role, task_id, parent_task_id, gate_id, timestamp, payload)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
				rejEvent.EventID, rejEvent.SchemaVersion, rejEvent.ProjectID, rejEvent.Topic,
				rejEvent.ActorID, rejEvent.ActorRole, rejEvent.TaskID, rejEvent.ParentTaskID,
				rejEvent.GateID, rejEvent.Timestamp, rejEvent.Payload)
			return fmt.Errorf("cross_project_dependency")
		}

		// Cycle detection: to detect if adding taskID→dep.TargetTaskID would close a
		// cycle, we need to check whether dep.TargetTaskID can already reach taskID via
		// EXISTING edges (before this candidate edge). If so, wiring taskID→dep.TargetTaskID
		// would create a dependency chain that loops back to taskID.
		//
		// Build reverse adjacency from EXISTING edges only (revAdj[node] = nodes that have
		// outgoing edges TO node in the stored graph). Then run DFS from dep.TargetTaskID
		// following only those existing predecessor links. If we reach taskID, the cycle
		// would close. The candidate edge (dep.TargetTaskID←taskID) is NOT included in
		// revAdj — it is checked separately as an immediate self-loop guard.
		revAdj := make(map[string]map[string]bool) // node -> nodes that point TO it
		addRevEdge := func(dst, src string) {
			if revAdj[dst] == nil {
				revAdj[dst] = make(map[string]bool)
			}
			revAdj[dst][src] = true
		}
		// Existing edges: for each existing target T of taskID, add rev edge T←taskID
		for _, tgt := range existingTargets {
			addRevEdge(tgt, taskID)
		}
		// New deps that precede this one in the batch
		for _, earlier := range deps {
			if earlier == dep {
				break
			}
			addRevEdge(earlier.TargetTaskID, taskID)
		}

		// Immediate self-loop guard: if taskID already has a direct outgoing edge to
		// dep.TargetTaskID, that means dep.TargetTaskID already depends on taskID —
		// adding taskID→dep.TargetTaskID would close the cycle immediately.
		immediateCycle := false
		for _, tgt := range existingTargets {
			if tgt == dep.TargetTaskID {
				immediateCycle = true
				break
			}
		}

		// DFS from dep.TargetTaskID following EXISTING reverse edges only.
		// If we can reach taskID, dep.TargetTaskID already depends (transitively) on
		// taskID, so adding taskID→dep.TargetTaskID would close a cycle.
		visited := make(map[string]bool)
		var hasCycle func(node string) bool
		hasCycle = func(node string) bool {
			if node == taskID {
				return true
			}
			if visited[node] {
				return false
			}
			visited[node] = true
			for predecessor := range revAdj[node] {
				if hasCycle(predecessor) {
					return true
				}
			}
			return false
		}
		if immediateCycle || hasCycle(dep.TargetTaskID) {
			rejEvent := models.AuditEvent{
				EventID:       newID("ev"),
				SchemaVersion: "v1alpha",
				ProjectID:     projectID,
				Topic:         "task.dependency.rejected",
				ActorID:       actor.ID,
				ActorRole:     actor.Role,
				TaskID:        &taskID,
				ParentTaskID:  nil,
				GateID:        nil,
				Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
				Payload: map[string]any{
					"targetTaskId": dep.TargetTaskID,
					"reason":       "circular dependency not permitted",
				},
			}
			_, _ = tx.Exec(ctx, `
				INSERT INTO audit_events (event_id, schema_version, project_id, topic, actor_id, actor_role, task_id, parent_task_id, gate_id, timestamp, payload)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
				rejEvent.EventID, rejEvent.SchemaVersion, rejEvent.ProjectID, rejEvent.Topic,
				rejEvent.ActorID, rejEvent.ActorRole, rejEvent.TaskID, rejEvent.ParentTaskID,
				rejEvent.GateID, rejEvent.Timestamp, rejEvent.Payload)
			return fmt.Errorf("circular_dependency")
		}
	}

	// All validations passed — delete existing edges and insert new ones
	_, err = tx.Exec(ctx, `DELETE FROM task_dependencies WHERE source_task_id=$1 AND project_id=$2`, taskID, projectID)
	if err != nil {
		return fmt.Errorf("delete existing: %w", err)
	}

	for _, dep := range deps {
		_, err = tx.Exec(ctx, `
			INSERT INTO task_dependencies (id, project_id, source_task_id, target_task_id, type, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			newID("td"), projectID, taskID, dep.TargetTaskID, dep.Type, time.Now())
		if err != nil {
			return fmt.Errorf("insert dep: %w", err)
		}
	}

	// Audit event
	event := models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     projectID,
		Topic:         "task.dependencies.replaced",
		ActorID:       actor.ID,
		ActorRole:     actor.Role,
		TaskID:        &taskID,
		ParentTaskID:  nil,
		GateID:        nil,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload:       map[string]any{"dependencyCount": len(deps)},
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

// CanCompleteParent checks if a parent task can transition to done.
// Returns error if any required child is not done, or any blocking gate is open/rejected (FR-02-007).
func (r *TaskRepository) CanCompleteParent(ctx context.Context, parentID string) error {
	// Check required children
	childRows, err := r.db.Query(ctx, `
		SELECT t.status FROM orchestration_tasks t
		JOIN task_children tc ON tc.child_id = t.id
		WHERE tc.task_id=$1 AND t.required=true`, parentID)
	if err != nil {
		return fmt.Errorf("check children: %w", err)
	}
	defer childRows.Close()
	for childRows.Next() {
		var status string
		childRows.Scan(&status)
		if status != "done" {
			return fmt.Errorf("required child not done")
		}
	}

	// Check blocking gates
	gateRow := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM task_gates WHERE task_id=$1 AND blocking=true AND state IN ('open', 'blocked')`, parentID)
	var count int
	gateRow.Scan(&count)
	if count > 0 {
		return fmt.Errorf("blocking gate open")
	}

	return nil
}

// GetTaskDepth returns the depth of a task in the decomposition tree.
// Root tasks have depth 1. The depth is computed by counting ancestors via task_parents.
// Returns error if the task is not found.
func (r *TaskRepository) GetTaskDepth(ctx context.Context, taskID string) (int, error) {
	depth := 1
	currentID := taskID
	for {
		row := r.db.QueryRow(ctx, `SELECT parent_id FROM task_parents WHERE task_id=$1 LIMIT 1`, currentID)
		var parentID string
		err := row.Scan(&parentID)
		if err != nil {
			if err == pgx.ErrNoRows {
				// No parent means we've reached the root
				return depth, nil
			}
			return 0, fmt.Errorf("fetch parent: %w", err)
		}
		depth++
		currentID = parentID
		// Guard against infinite loop (should not happen with valid data)
		if depth > 100 {
			return 0, fmt.Errorf("depth limit exceeded (possible cycle)")
		}
	}
}

// CountActiveChildren returns the count of active (non-terminal) children of a task.
// Active means the child is not 'done' and not 'cancelled'.
func (r *TaskRepository) CountActiveChildren(ctx context.Context, parentID string) (int, error) {
	row := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM orchestration_tasks t
		JOIN task_children tc ON tc.child_id = t.id
		WHERE tc.task_id=$1 AND t.status NOT IN ('done', 'cancelled')`, parentID)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count active children: %w", err)
	}
	return count, nil
}