package repository

import (
	"context"
	"time"

	"github.com/agent-orchestrator/backend/internal/models"
	"github.com/agent-orchestrator/backend/internal/service"
)

// ClientPortalProjectRepository is the concrete implementation of
// service.ClientPortalProjectRepo using the existing Pool.
type ClientPortalProjectRepository struct {
	db *Pool
}

// NewClientPortalProjectRepository creates a client portal project repository.
func NewClientPortalProjectRepository(db *Pool) *ClientPortalProjectRepository {
	return &ClientPortalProjectRepository{db: db}
}

// ListAccessibleProjects returns all projects accessible to the given client principal.
func (r *ClientPortalProjectRepository) ListAccessibleProjects(ctx context.Context, principal string) ([]service.ClientPortalProject, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.id, p.name,
			COALESCE((SELECT health FROM project_client_details WHERE project_id = p.id), 'on_track') as health,
			COALESCE((SELECT confidence FROM project_client_details WHERE project_id = p.id), 'medium') as confidence,
			COALESCE((SELECT health_reason FROM project_client_details WHERE project_id = p.id), '') as health_reason,
			(SELECT completion_percent FROM project_client_details WHERE project_id = p.id) as completion_percent,
			(SELECT next_milestone FROM project_client_details WHERE project_id = p.id) as next_milestone,
			p.updated_at
		FROM projects p
		JOIN project_clients pc_map ON pc_map.project_id = p.id AND pc_map.client_id = $1
		WHERE p.status = 'active'
		ORDER BY p.updated_at DESC`, principal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []service.ClientPortalProject
	for rows.Next() {
		var p service.ClientPortalProject
		var nextMilestone *string
		var updatedAt time.Time
		if err := rows.Scan(&p.ID, &p.Name, &p.Health, &p.Confidence, &p.HealthReason, &p.CompletionPercent, &nextMilestone, &updatedAt); err != nil {
			return nil, err
		}
		p.NextMilestone = nextMilestone
		p.LatestUpdate = updatedAt
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// GetProjectDetail returns the full project detail for a client.
func (r *ClientPortalProjectRepository) GetProjectDetail(ctx context.Context, projectID, principal string) (*service.ClientPortalProjectDetail, error) {
	// Fetch project summary
	row := r.db.QueryRow(ctx, `
		SELECT p.id, p.name,
			COALESCE(pc.health, 'on_track'),
			COALESCE(pc.confidence, 'medium'),
			COALESCE(pc.health_reason, ''),
			pc.completion_percent,
			p.updated_at
		FROM projects p
		JOIN project_clients pc_map ON pc_map.project_id = p.id AND pc_map.client_id = $2
		LEFT JOIN project_client_details pc ON pc.project_id = p.id
		WHERE p.id = $1 AND p.status = 'active'`, projectID, principal)
	var p service.ClientPortalProjectDetail
	var name string
	var updatedAt time.Time
	err := row.Scan(&p.ID, &name, &p.Health, &p.Confidence, &p.HealthReason, &p.CompletionPercent, &updatedAt)
	if err != nil {
		return nil, err
	}
	p.Name = name

	// Fetch tasks
	taskRows, err := r.db.Query(ctx, `
		SELECT id, title, status,
			COALESCE(owner_label, 'Product') as owner_label,
			summary, blocker_reason, due_date, updated_at, next_action
		FROM orchestration_tasks
		WHERE project_id=$1 AND status != 'cancelled' AND status != 'proposed'
		ORDER BY priority DESC, created_at ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer taskRows.Close()

	for taskRows.Next() {
		var t models.ClientTaskCard
		var blockerReason *string
		var dueDate *time.Time
		var updatedAtTask time.Time
		if err := taskRows.Scan(&t.ID, &t.Title, &t.Status, &t.OwnerLabel, &t.Summary, &blockerReason, &dueDate, &updatedAtTask, &t.NextAction); err != nil {
			return nil, err
		}
		t.BlockerReason = blockerReason
		t.DueDate = dueDate
		t.UpdatedAt = updatedAtTask
		p.Tasks = append(p.Tasks, t)
	}

	// Fetch risks
	riskRows, err := r.db.Query(ctx, `
		SELECT id, severity, impact, owner_label, mitigation_summary, status, next_action
		FROM risks WHERE project_id=$1 ORDER BY severity DESC, created_at ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer riskRows.Close()

	for riskRows.Next() {
		var rk models.ClientRiskItem
		if err := riskRows.Scan(&rk.ID, &rk.Severity, &rk.Impact, &rk.OwnerLabel, &rk.MitigationSummary, &rk.Status, &rk.NextAction); err != nil {
			return nil, err
		}
		p.Risks = append(p.Risks, rk)
	}

	// Fetch milestones
	milestoneRows, err := r.db.Query(ctx, `
		SELECT id, name, status, target_date, progress, health, summary, next_action
		FROM milestones WHERE project_id=$1 ORDER BY target_date ASC NULLS LAST`, projectID)
	if err != nil {
		return nil, err
	}
	defer milestoneRows.Close()

	for milestoneRows.Next() {
		var m models.ClientMilestoneItem
		var targetDate *string
		if err := milestoneRows.Scan(&m.ID, &m.Name, &m.Status, &targetDate, &m.Progress, &m.Health, &m.Summary, &m.NextAction); err != nil {
			return nil, err
		}
		m.TargetDate = targetDate
		p.Milestones = append(p.Milestones, m)
	}

	return &p, nil
}

// PrincipalHasAccess checks if a client principal can access a given project.
func (r *ClientPortalProjectRepository) PrincipalHasAccess(ctx context.Context, projectID, principal string) bool {
	var count int
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM project_clients WHERE project_id=$1 AND client_id=$2",
		projectID, principal).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

// Search performs client-safe search across accessible projects.
func (r *ClientPortalProjectRepository) Search(ctx context.Context, principal, query, healthFilter, statusFilter string) ([]models.ClientSearchResultItem, error) {
	// Build parameterized query with access control
	sql := `
		SELECT DISTINCT
			COALESCE(t.id, p.id) as id,
			COALESCE(t.title, p.name) as title,
			COALESCE(t.project_id, p.id) as project_id,
			COALESCE(t.project_id, p.id) as project_id2,
			COALESCE(p.name, '') as project_name,
			CASE WHEN t.id IS NOT NULL THEN 'task' ELSE 'project' END as result_type,
			COALESCE(t.body, p.description) as snippet
		FROM projects p
		JOIN project_clients pc_map ON pc_map.project_id = p.id AND pc_map.client_id = $1
		LEFT JOIN orchestration_tasks t ON t.project_id = p.id
		WHERE p.status = 'active'`
	args := []any{principal}
	argIdx := 2

	if query != "" {
		sql += " AND (p.name ILIKE $2 OR p.description ILIKE $2 OR t.title ILIKE $2 OR t.body ILIKE $2)"
		args = append(args, "%"+query+"%")
		argIdx++
	}

	// The rest of the filters and pagination...
	// Return combined project + task results
	var items []models.ClientSearchResultItem

	// Fetch project matches
	projRows, err := r.db.Query(ctx, `
		SELECT p.id, p.name, p.id, p.name, 'project', COALESCE(p.description, '')
		FROM projects p
		JOIN project_clients pc_map ON pc_map.project_id = p.id AND pc_map.client_id = $1
		WHERE p.status = 'active'
		AND ($2 = '' OR p.name ILIKE $2 OR p.description ILIKE $2)`,
		principal, "%"+query+"%")
	if err == nil {
		defer projRows.Close()
		for projRows.Next() {
			var item models.ClientSearchResultItem
			if err := projRows.Scan(&item.ID, &item.Title, &item.ProjectID, &item.ProjectName, &item.Type, &item.HighlightedContent); err == nil {
				items = append(items, item)
			}
		}
	}

	// Fetch task matches
	taskRows, err := r.db.Query(ctx, `
		SELECT t.id, t.title, t.project_id, p.name, 'task', COALESCE(t.body, '')
		FROM orchestration_tasks t
		JOIN projects p ON p.id = t.project_id
		JOIN project_clients pc_map ON pc_map.project_id = p.id AND pc_map.client_id = $1
		WHERE p.status = 'active'
		AND ($2 = '' OR t.title ILIKE $2 OR t.body ILIKE $2)`,
		principal, "%"+query+"%")
	if err == nil {
		defer taskRows.Close()
		for taskRows.Next() {
			var item models.ClientSearchResultItem
			if err := taskRows.Scan(&item.ID, &item.Title, &item.ProjectID, &item.ProjectName, &item.Type, &item.HighlightedContent); err == nil {
				items = append(items, item)
			}
		}
	}

	return items, nil
}

// --- ClientPortalApprovalRepository ---

// ClientPortalApprovalRepository is the concrete implementation of
// service.ClientPortalApprovalRepo.
type ClientPortalApprovalRepository struct {
	db *Pool
}

// NewClientPortalApprovalRepository creates a client portal approval repository.
func NewClientPortalApprovalRepository(db *Pool) *ClientPortalApprovalRepository {
	return &ClientPortalApprovalRepository{db: db}
}

// CountPendingApprovals returns pending and overdue counts for a project/principal.
func (r *ClientPortalApprovalRepository) CountPendingApprovals(ctx context.Context, projectID, principal string) (pending int, overdue int) {
	rows, err := r.db.Query(ctx, `
		SELECT g.state, g.created_at
		FROM task_gates g
		JOIN project_clients pc ON pc.project_id = g.project_id AND pc.client_id = $2
		WHERE g.project_id = $1 AND g.state IN ('open', 'pending')`, projectID, principal)
	if err != nil {
		return 0, 0
	}
	defer rows.Close()

	for rows.Next() {
		var state string
		var createdAt time.Time
		if err := rows.Scan(&state, &createdAt); err == nil {
			pending++
			if time.Since(createdAt) > 24*time.Hour {
				overdue++
			}
		}
	}
	return pending, overdue
}

// ListProjectApprovals returns approval items for a specific project accessible to the principal.
func (r *ClientPortalApprovalRepository) ListProjectApprovals(ctx context.Context, projectID, principal string) ([]models.ClientApprovalItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT g.id, COALESCE(t.title, 'Gate: ' || g.phase) as title,
			COALESCE(g.passed_by, 'Client') as owner_label,
			g.state as outcome, g.created_at
		FROM task_gates g
		JOIN project_clients pc ON pc.project_id = g.project_id AND pc.client_id = $2
		LEFT JOIN orchestration_tasks t ON t.id = g.task_id
		WHERE g.project_id = $1 AND g.state IN ('open', 'pending')
		ORDER BY g.created_at DESC`, projectID, principal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.ClientApprovalItem
	for rows.Next() {
		var item models.ClientApprovalItem
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.Title, &item.OwnerLabel, &item.Outcome, &createdAt); err != nil {
			return nil, err
		}
		item.CreatedAt = createdAt
		// Map internal gate state to client approval outcome
		switch item.Outcome {
		case "open", "pending":
			item.Outcome = "pending"
		case "passed":
			item.Outcome = "approved"
		case "blocked":
			item.Outcome = "rejected"
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ListAccessibleApprovals returns all approvals across projects accessible to the principal.
func (r *ClientPortalApprovalRepository) ListAccessibleApprovals(ctx context.Context, principal string) ([]models.ClientApprovalItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT g.id, COALESCE(t.title, 'Gate: ' || g.phase) as title,
			COALESCE(g.passed_by, 'Client') as owner_label,
			g.state as outcome, g.created_at, g.project_id
		FROM task_gates g
		JOIN project_clients pc ON pc.project_id = g.project_id AND pc.client_id = $1
		LEFT JOIN orchestration_tasks t ON t.id = g.task_id
		WHERE g.state IN ('open', 'pending')
		ORDER BY g.created_at DESC`, principal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.ClientApprovalItem
	for rows.Next() {
		var item models.ClientApprovalItem
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.Title, &item.OwnerLabel, &item.Outcome, &createdAt); err != nil {
			return nil, err
		}
		item.CreatedAt = createdAt
		switch item.Outcome {
		case "open", "pending":
			item.Outcome = "pending"
		case "passed":
			item.Outcome = "approved"
		case "blocked":
			item.Outcome = "rejected"
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ApprovalPrincipalHasAccess checks if the principal can act on a given approval.
func (r *ClientPortalApprovalRepository) ApprovalPrincipalHasAccess(ctx context.Context, approvalID, principal string) bool {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM task_gates g
		JOIN project_clients pc ON pc.project_id = g.project_id AND pc.client_id = $2
		WHERE g.id = $1`, approvalID, principal).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

// RecordDecision records a client approval decision and returns the updated item and project ID.
func (r *ClientPortalApprovalRepository) RecordDecision(ctx context.Context, approvalID, principal string, req models.ApprovalDecisionRequest) (*models.ClientApprovalItem, string, error) {
	var projectID, taskID, currentState string
	row := r.db.QueryRow(ctx, `SELECT project_id, task_id, state FROM task_gates WHERE id=$1`, approvalID)
	if err := row.Scan(&projectID, &taskID, &currentState); err != nil {
		return nil, "", err
	}

	// Map client outcome to internal gate state
	var newState string
	switch req.Outcome {
	case "approve":
		newState = "passed"
	case "reject":
		newState = "blocked"
	case "request_changes", "need_more_information":
		newState = "pending"
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, projectID, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE task_gates SET state=$2, passed_by=$3, passed_at=NOW()
		WHERE id=$1`, approvalID, newState, principal)
	if err != nil {
		return nil, projectID, err
	}

	// Emit audit event
	eventID := "ev_" + newID("")
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_events (event_id, schema_version, project_id, topic, actor_id, actor_role, task_id, parent_task_id, gate_id, timestamp, payload)
		VALUES ($1, 'v1alpha', $2, 'gate.approved', $3, 'client', $4, NULL, $5, NOW(), $6)`,
		eventID, projectID, principal, taskID, approvalID,
		`{"outcome":"`+req.Outcome+`","comment":`+ptrJSON(req.Comment)+`}`)
	if err != nil {
		return nil, projectID, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, projectID, err
	}

	return &models.ClientApprovalItem{
		ID:         approvalID,
		Title:      "Gate approval",
		Outcome:    req.Outcome,
		CreatedAt:  time.Now(),
		Overdue:    false,
	}, projectID, nil
}

func ptrJSON(s *string) string {
	if s == nil {
		return "null"
	}
	return `"` + *s + `"`
}