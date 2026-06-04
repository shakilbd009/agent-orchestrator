package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/agent-orchestrator/backend/internal/models"
	"github.com/agent-orchestrator/backend/internal/observability"
)

// --- mock implementations ---

type mockClientPortalProjectRepo struct {
	listAccessibleProjectsFn   func(ctx context.Context, principal string) ([]ClientPortalProject, error)
	getProjectDetailFn          func(ctx context.Context, projectID, principal string) (*ClientPortalProjectDetail, error)
	principalHasAccessFn         func(ctx context.Context, projectID, principal string) bool
	searchFn                     func(ctx context.Context, principal, query, healthFilter, statusFilter string) ([]models.ClientSearchResultItem, error)
}

// newTestClientPortalService is a test helper that creates a ClientPortalService
// with nil observability (metrics and logger), for tests that only verify business logic.
func newTestClientPortalService(
	projectRepo ClientPortalProjectRepo,
	approvalRepo ClientPortalApprovalRepo,
	commentRepo ClientPortalCommentRepo,
) *ClientPortalService {
	return NewClientPortalService(projectRepo, approvalRepo, commentRepo, nil, nil)
}

// newTestClientPortalServiceWithObservability creates a ClientPortalService with real observability.
func newTestClientPortalServiceWithObservability(
	projectRepo ClientPortalProjectRepo,
	approvalRepo ClientPortalApprovalRepo,
	commentRepo ClientPortalCommentRepo,
	logger *observability.Logger,
	metrics *observability.Metrics,
) *ClientPortalService {
	return NewClientPortalService(projectRepo, approvalRepo, commentRepo, logger, metrics)
}

func (m *mockClientPortalProjectRepo) ListAccessibleProjects(ctx context.Context, principal string) ([]ClientPortalProject, error) {
	if m.listAccessibleProjectsFn != nil {
		return m.listAccessibleProjectsFn(ctx, principal)
	}
	return nil, nil
}

func (m *mockClientPortalProjectRepo) GetProjectDetail(ctx context.Context, projectID, principal string) (*ClientPortalProjectDetail, error) {
	if m.getProjectDetailFn != nil {
		return m.getProjectDetailFn(ctx, projectID, principal)
	}
	return nil, nil
}

func (m *mockClientPortalProjectRepo) PrincipalHasAccess(ctx context.Context, projectID, principal string) bool {
	if m.principalHasAccessFn != nil {
		return m.principalHasAccessFn(ctx, projectID, principal)
	}
	return false
}

func (m *mockClientPortalProjectRepo) Search(ctx context.Context, principal, query, healthFilter, statusFilter string) ([]models.ClientSearchResultItem, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, principal, query, healthFilter, statusFilter)
	}
	return nil, nil
}

type mockClientPortalApprovalRepo struct {
	countPendingApprovalsFn      func(ctx context.Context, projectID, principal string) (pending int, overdue int)
	listProjectApprovalsFn       func(ctx context.Context, projectID, principal string) ([]models.ClientApprovalItem, error)
	listAccessibleApprovalsFn    func(ctx context.Context, principal string) ([]models.ClientApprovalItem, error)
	approvalPrincipalHasAccessFn func(ctx context.Context, approvalID, principal string) bool
	recordDecisionFn             func(ctx context.Context, approvalID, principal string, req models.ApprovalDecisionRequest) (*models.ClientApprovalItem, string, error)
}

func (m *mockClientPortalApprovalRepo) CountPendingApprovals(ctx context.Context, projectID, principal string) (pending int, overdue int) {
	if m.countPendingApprovalsFn != nil {
		return m.countPendingApprovalsFn(ctx, projectID, principal)
	}
	return 0, 0
}

func (m *mockClientPortalApprovalRepo) ListProjectApprovals(ctx context.Context, projectID, principal string) ([]models.ClientApprovalItem, error) {
	if m.listProjectApprovalsFn != nil {
		return m.listProjectApprovalsFn(ctx, projectID, principal)
	}
	return nil, nil
}

func (m *mockClientPortalApprovalRepo) ListAccessibleApprovals(ctx context.Context, principal string) ([]models.ClientApprovalItem, error) {
	if m.listAccessibleApprovalsFn != nil {
		return m.listAccessibleApprovalsFn(ctx, principal)
	}
	return nil, nil
}

func (m *mockClientPortalApprovalRepo) ApprovalPrincipalHasAccess(ctx context.Context, approvalID, principal string) bool {
	if m.approvalPrincipalHasAccessFn != nil {
		return m.approvalPrincipalHasAccessFn(ctx, approvalID, principal)
	}
	return false
}

func (m *mockClientPortalApprovalRepo) RecordDecision(ctx context.Context, approvalID, principal string, req models.ApprovalDecisionRequest) (*models.ClientApprovalItem, string, error) {
	if m.recordDecisionFn != nil {
		return m.recordDecisionFn(ctx, approvalID, principal, req)
	}
	return nil, "", nil
}

type mockClientPortalCommentRepo struct {
	listByProjectAndItemFn func(ctx context.Context, projectID, itemID string) ([]models.ClientComment, error)
	createCommentFn        func(ctx context.Context, c *models.ClientComment) error
	editCommentFn          func(ctx context.Context, commentID, authorID, newBody string) (*models.ClientComment, error)
	deleteCommentFn        func(ctx context.Context, commentID, authorID string) (bool, error)
}

func (m *mockClientPortalCommentRepo) ListByProjectAndItem(ctx context.Context, projectID, itemID string) ([]models.ClientComment, error) {
	if m.listByProjectAndItemFn != nil {
		return m.listByProjectAndItemFn(ctx, projectID, itemID)
	}
	return nil, nil
}

func (m *mockClientPortalCommentRepo) CreateComment(ctx context.Context, c *models.ClientComment) error {
	if m.createCommentFn != nil {
		return m.createCommentFn(ctx, c)
	}
	return nil
}

func (m *mockClientPortalCommentRepo) EditComment(ctx context.Context, commentID, authorID, newBody string) (*models.ClientComment, error) {
	if m.editCommentFn != nil {
		return m.editCommentFn(ctx, commentID, authorID, newBody)
	}
	return nil, nil
}

func (m *mockClientPortalCommentRepo) DeleteComment(ctx context.Context, commentID, authorID string) (bool, error) {
	if m.deleteCommentFn != nil {
		return m.deleteCommentFn(ctx, commentID, authorID)
	}
	return false, nil
}

// --- tests ---

func TestGetPortfolio_Empty(t *testing.T) {
	projectRepo := &mockClientPortalProjectRepo{
		listAccessibleProjectsFn: func(ctx context.Context, principal string) ([]ClientPortalProject, error) {
			return []ClientPortalProject{}, nil
		},
	}
	approvalRepo := &mockClientPortalApprovalRepo{}
	svc := newTestClientPortalService(projectRepo, approvalRepo, nil)

	result, err := svc.GetPortfolio(context.Background(), "client-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.ProjectList) != 0 {
		t.Errorf("expected 0 projects, got %d", len(result.ProjectList))
	}
	if result.DecisionSummary.TotalPending != 0 {
		t.Errorf("expected 0 pending, got %d", result.DecisionSummary.TotalPending)
	}
}

func TestGetPortfolio_SumsHealth(t *testing.T) {
	projectRepo := &mockClientPortalProjectRepo{
		listAccessibleProjectsFn: func(ctx context.Context, principal string) ([]ClientPortalProject, error) {
			return []ClientPortalProject{
				{ID: "p1", Name: "Project 1", Health: "on_track"},
				{ID: "p2", Name: "Project 2", Health: "at_risk"},
				{ID: "p3", Name: "Project 3", Health: "at_risk"},
				{ID: "p4", Name: "Project 4", Health: "blocked"},
			}, nil
		},
	}
	approvalRepo := &mockClientPortalApprovalRepo{
		countPendingApprovalsFn: func(ctx context.Context, projectID, principal string) (int, int) {
			return 2, 1
		},
	}
	svc := newTestClientPortalService(projectRepo, approvalRepo, nil)

	result, err := svc.GetPortfolio(context.Background(), "client-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ProjectsSummary.OnTrack != 1 {
		t.Errorf("expected OnTrack=1, got %d", result.ProjectsSummary.OnTrack)
	}
	if result.ProjectsSummary.AtRisk != 2 {
		t.Errorf("expected AtRisk=2, got %d", result.ProjectsSummary.AtRisk)
	}
	if result.ProjectsSummary.Blocked != 1 {
		t.Errorf("expected Blocked=1, got %d", result.ProjectsSummary.Blocked)
	}
	if result.DecisionSummary.TotalPending != 8 {
		t.Errorf("expected TotalPending=8 (4 projects * 2), got %d", result.DecisionSummary.TotalPending)
	}
	if result.DecisionSummary.Overdue != 4 {
		t.Errorf("expected Overdue=4 (4 projects * 1), got %d", result.DecisionSummary.Overdue)
	}
}

func TestGetProjectDetail_FailClosed(t *testing.T) {
	projectRepo := &mockClientPortalProjectRepo{
		principalHasAccessFn: func(ctx context.Context, projectID, principal string) bool {
			return false // deny
		},
	}
	svc := newTestClientPortalService(projectRepo, nil, nil)

	result, err := svc.GetProjectDetail(context.Background(), "p1", "client-1")
	if err != nil {
		t.Fatalf("expected no error for fail-closed, got %v", err)
	}
	if result.ID != "" {
		t.Errorf("expected empty result for unauthorized, got ID=%s", result.ID)
	}
}

func TestGetProjectDetail_UnauthorizedReturnsEmpty(t *testing.T) {
	projectRepo := &mockClientPortalProjectRepo{
		principalHasAccessFn: func(ctx context.Context, projectID, principal string) bool {
			return false
		},
		getProjectDetailFn: func(ctx context.Context, projectID, principal string) (*ClientPortalProjectDetail, error) {
			return &ClientPortalProjectDetail{ID: projectID}, nil
		},
	}
	svc := newTestClientPortalService(projectRepo, nil, nil)
	result, _ := svc.GetProjectDetail(context.Background(), "p1", "client-1")
	if result.ID != "" {
		t.Errorf("fail-closed: expected empty result, got ID=%s", result.ID)
	}
}

func TestDecideApproval_InvalidOutcome(t *testing.T) {
	approvalRepo := &mockClientPortalApprovalRepo{}
	svc := newTestClientPortalService(nil, approvalRepo, nil)

	result, err := svc.DecideApproval(context.Background(), "a1", "client-1", models.ApprovalDecisionRequest{
		Outcome: "invalid_outcome",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Success {
		t.Error("expected success=false for invalid outcome")
	}
	if result.Message == nil || !strings.Contains(*result.Message, "invalid outcome") {
		t.Errorf("expected 'invalid outcome' message, got %v", result.Message)
	}
}

func TestDecideApproval_CommentRequiredForReject(t *testing.T) {
	approvalRepo := &mockClientPortalApprovalRepo{}
	svc := newTestClientPortalService(nil, approvalRepo, nil)

	// reject without comment
	result, err := svc.DecideApproval(context.Background(), "a1", "client-1", models.ApprovalDecisionRequest{
		Outcome: "reject",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Success {
		t.Error("expected success=false for reject without comment")
	}
	if result.Message == nil || !strings.Contains(*result.Message, "comment required") {
		t.Errorf("expected 'comment required' message, got %v", result.Message)
	}

	// reject WITH comment — should pass the comment check
	comment := "needs rework"
	result, err = svc.DecideApproval(context.Background(), "a1", "client-1", models.ApprovalDecisionRequest{
		Outcome: "reject",
		Comment: &comment,
	})
	if err != nil {
		t.Fatalf("expected no error when comment provided, got %v", err)
	}
	// Now it fails on access check (no mock), not validation
}

func TestDecideApproval_ApproveWithoutCommentOk(t *testing.T) {
	updatedItem := models.ClientApprovalItem{ID: "a1", Title: "Approve me", Outcome: "approved"}
	approvalRepo := &mockClientPortalApprovalRepo{
		approvalPrincipalHasAccessFn: func(ctx context.Context, approvalID, principal string) bool {
			return true
		},
		recordDecisionFn: func(ctx context.Context, approvalID, principal string, req models.ApprovalDecisionRequest) (*models.ClientApprovalItem, string, error) {
			return &updatedItem, "proj-1", nil
		},
	}
	svc := newTestClientPortalService(nil, approvalRepo, nil)

	result, err := svc.DecideApproval(context.Background(), "a1", "client-1", models.ApprovalDecisionRequest{
		Outcome: "approve",
		// Comment is optional for approve
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.Success {
		t.Errorf("expected success=true, got false: %v", result.Message)
	}
}

func TestDecideApproval_AccessDenied(t *testing.T) {
	approvalRepo := &mockClientPortalApprovalRepo{
		approvalPrincipalHasAccessFn: func(ctx context.Context, approvalID, principal string) bool {
			return false // deny
		},
	}
	svc := newTestClientPortalService(nil, approvalRepo, nil)

	result, err := svc.DecideApproval(context.Background(), "a1", "client-1", models.ApprovalDecisionRequest{
		Outcome: "approve",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Success {
		t.Error("expected success=false for access denied")
	}
}

func TestGetApprovalInbox_SetsOverdue(t *testing.T) {
	oldItem := models.ClientApprovalItem{
		ID:        "a1",
		Title:     "Old approval",
		Outcome:   "pending",
		CreatedAt: time.Now().Add(-48 * time.Hour), // 48h old → overdue
	}
	recentItem := models.ClientApprovalItem{
		ID:        "a2",
		Title:     "Recent approval",
		Outcome:   "pending",
		CreatedAt: time.Now().Add(-1 * time.Hour), // 1h old → not overdue
	}
	approvalRepo := &mockClientPortalApprovalRepo{
		listAccessibleApprovalsFn: func(ctx context.Context, principal string) ([]models.ClientApprovalItem, error) {
			return []models.ClientApprovalItem{oldItem, recentItem}, nil
		},
	}
	svc := newTestClientPortalService(nil, approvalRepo, nil)

	result, err := svc.GetApprovalInbox(context.Background(), "client-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.TotalCount != 2 {
		t.Errorf("expected TotalCount=2, got %d", result.TotalCount)
	}
	if !result.Items[0].Overdue {
		t.Error("expected first item (48h old) to be overdue")
	}
	if result.Items[1].Overdue {
		t.Error("expected second item (1h old) to NOT be overdue")
	}
}

func TestSearchClientPortal_StripsForbidden(t *testing.T) {
	projectRepo := &mockClientPortalProjectRepo{
		searchFn: func(ctx context.Context, principal, query, healthFilter, statusFilter string) ([]models.ClientSearchResultItem, error) {
			return []models.ClientSearchResultItem{
				{
					ID:                 "result-1",
					Type:               "task",
					Title:              "Fix authentication bug",
					ProjectID:          "p1",
					ProjectName:        "Auth Service",
					HighlightedContent: "at com.auth.AuthService.verifyToken(AuthService.java:42)\nStack trace truncated",
				},
				{
					ID:                 "result-2",
					Type:               "task",
					Title:              "Deploy to kubernetes",
					ProjectID:          "p2",
					ProjectName:        "DevOps",
					HighlightedContent: "Deployed via helm chart to eks-prod cluster with arn:aws:iam::123456789012:role/kube-admin",
				},
			}, nil
		},
	}
	svc := newTestClientPortalService(projectRepo, nil, nil)

	result, err := svc.SearchClientPortal(context.Background(), "client-1", "authentication", "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Items))
	}
	// First result: stack trace line should be stripped
	if strings.Contains(result.Items[0].HighlightedContent, "at ") {
		t.Error("expected stack trace 'at ' to be stripped from highlighted content")
	}
	// Second result: infra terms should be stripped
	if strings.Contains(result.Items[1].HighlightedContent, "kubernetes") ||
		strings.Contains(result.Items[1].HighlightedContent, "eks-") ||
		strings.Contains(result.Items[1].HighlightedContent, "arn:aws:") {
		t.Error("expected infra terms to be stripped from highlighted content")
	}
}

func TestIsOverdue_ZeroTime(t *testing.T) {
	if isOverdue(time.Time{}) {
		t.Error("zero time should not be overdue")
	}
}

func TestIsOverdue_23Hours(t *testing.T) {
	recent := time.Now().Add(-23 * time.Hour)
	if isOverdue(recent) {
		t.Error("23h old should NOT be overdue (threshold is strictly >24h)")
	}
}

func TestIsOverdue_25Hours(t *testing.T) {
	old := time.Now().Add(-25 * time.Hour)
	if !isOverdue(old) {
		t.Error("25h old SHOULD be overdue")
	}
}

func TestDerefFloat64_Nil(t *testing.T) {
	if derefFloat64(nil) != 0 {
		t.Error("nil *float64 should return 0")
	}
}

func TestDerefFloat64_Value(t *testing.T) {
	v := 42.5
	if derefFloat64(&v) != 42.5 {
		t.Error("expected 42.5")
	}
}

func TestBuildBoardColumns(t *testing.T) {
	tasks := []models.ClientTaskCard{
		{ID: "t1", Title: "Task 1", Status: "todo"},
		{ID: "t2", Title: "Task 2", Status: "in_progress"},
		{ID: "t3", Title: "Task 3", Status: "todo"},
		{ID: "t4", Title: "Task 4", Status: "done"},
	}
	board := buildBoardColumns(tasks)
	if len(board) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(board))
	}
	// Check column order: todo, in_progress, blocked, done
	if board[0].Status != "todo" {
		t.Errorf("expected column 0 to be 'todo', got %s", board[0].Status)
	}
	if board[1].Status != "in_progress" {
		t.Errorf("expected column 1 to be 'in_progress', got %s", board[1].Status)
	}
	if board[3].Status != "done" {
		t.Errorf("expected column 3 to be 'done', got %s", board[3].Status)
	}
	// Check counts
	if len(board[0].Tasks) != 2 {
		t.Errorf("expected 2 tasks in todo, got %d", len(board[0].Tasks))
	}
	if len(board[1].Tasks) != 1 {
		t.Errorf("expected 1 task in in_progress, got %d", len(board[1].Tasks))
	}
	if len(board[3].Tasks) != 1 {
		t.Errorf("expected 1 task in done, got %d", len(board[3].Tasks))
	}
}

func TestBuildBoardColumns_Empty(t *testing.T) {
	board := buildBoardColumns(nil)
	if len(board) != 4 {
		t.Fatalf("expected 4 columns even for empty tasks, got %d", len(board))
	}
	for _, col := range board {
		if len(col.Tasks) != 0 {
			t.Errorf("expected 0 tasks in %s column, got %d", col.Status, len(col.Tasks))
		}
	}
}

// --- wiring tests: verify instrumentation is emitted through service call paths ---

func TestGetApprovalInbox_WiresMetrics(t *testing.T) {
	oldItem := models.ClientApprovalItem{
		ID:        "a1",
		Title:     "Old approval",
		Outcome:   "pending",
		CreatedAt: time.Now().Add(-48 * time.Hour),
	}
	recentItem := models.ClientApprovalItem{
		ID:        "a2",
		Title:     "Recent approval",
		Outcome:   "pending",
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}
	approvalRepo := &mockClientPortalApprovalRepo{
		listAccessibleApprovalsFn: func(ctx context.Context, principal string) ([]models.ClientApprovalItem, error) {
			return []models.ClientApprovalItem{oldItem, recentItem}, nil
		},
	}
	metrics := observability.NewMetrics()
	logger := observability.NewLogger()
	svc := newTestClientPortalServiceWithObservability(nil, approvalRepo, nil, logger, metrics)

	_, err := svc.GetApprovalInbox(context.Background(), "client-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	snap := metrics.Snapshot()
	if snap.PendingApprovalsCurrent != 2 {
		t.Errorf("PendingApprovalsCurrent: expected 2, got %d", snap.PendingApprovalsCurrent)
	}
	if snap.OverdueDecisionsCurrent != 1 {
		t.Errorf("OverdueDecisionsCurrent: expected 1, got %d", snap.OverdueDecisionsCurrent)
	}
	if snap.OldestPendingDecisionAgeMs == 0 {
		t.Error("OldestPendingDecisionAgeMs: expected non-zero")
	}
	// GetApprovalInbox records oldest_pending_decision_age_ms as a gauge, but
	// MUST NOT record decision_turnaround (histogram) — turnaround samples are
	// only recorded on decision completion (actionable-to-outcome duration).
	if len(snap.DecisionTurnaroundMs) != 0 {
		t.Errorf("DecisionTurnaroundMs: expected 0 samples on read path, got %d", len(snap.DecisionTurnaroundMs))
	}
}

func TestDecideApproval_WiresApprovalSubmittedLog(t *testing.T) {
	updatedItem := models.ClientApprovalItem{ID: "a1", Title: "Approve me", Outcome: "approved"}
	approvalRepo := &mockClientPortalApprovalRepo{
		approvalPrincipalHasAccessFn: func(ctx context.Context, approvalID, principal string) bool {
			return true
		},
		recordDecisionFn: func(ctx context.Context, approvalID, principal string, req models.ApprovalDecisionRequest) (*models.ClientApprovalItem, string, error) {
			return &updatedItem, "proj-42", nil
		},
	}
	metrics := observability.NewMetrics()
	logger := observability.NewLogger()
	svc := newTestClientPortalServiceWithObservability(nil, approvalRepo, nil, logger, metrics)

	result, err := svc.DecideApproval(context.Background(), "a1", "client-1", models.ApprovalDecisionRequest{
		Outcome: "approve",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success=true, got false")
	}
	// Verify decision outcome metric was recorded (approve counter incremented)
	snap := metrics.Snapshot()
	if snap.DecisionOutcome["approve"] != 1 {
		t.Errorf("DecisionOutcome[approve]: expected 1, got %d", snap.DecisionOutcome["approve"])
	}
}

func TestDecideApproval_WiresNeedMoreInformationGauge(t *testing.T) {
	updatedItem := models.ClientApprovalItem{ID: "a1", Title: "NMI", Outcome: "need_more_information"}
	comment := "please clarify"
	approvalRepo := &mockClientPortalApprovalRepo{
		approvalPrincipalHasAccessFn: func(ctx context.Context, approvalID, principal string) bool {
			return true
		},
		recordDecisionFn: func(ctx context.Context, approvalID, principal string, req models.ApprovalDecisionRequest) (*models.ClientApprovalItem, string, error) {
			return &updatedItem, "proj-1", nil
		},
	}
	metrics := observability.NewMetrics()
	logger := observability.NewLogger()
	svc := newTestClientPortalServiceWithObservability(nil, approvalRepo, nil, logger, metrics)

	result, err := svc.DecideApproval(context.Background(), "a1", "client-1", models.ApprovalDecisionRequest{
		Outcome: "need_more_information",
		Comment: &comment,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success=true")
	}
	snap := metrics.Snapshot()
	if snap.NeedMoreInformationCurrent == 0 {
		t.Error("NeedMoreInformationCurrent: expected non-zero after NMI decision")
	}
	// RecordDecisionOutcome is called for every outcome
	if snap.DecisionOutcome["need_more_information"] != 1 {
		t.Errorf("DecisionOutcome[need_more_information]: expected 1, got %d", snap.DecisionOutcome["need_more_information"])
	}
}

func TestDecideApproval_WiresRequestedChangesGauge(t *testing.T) {
	updatedItem := models.ClientApprovalItem{ID: "a1", Title: "RC", Outcome: "request_changes"}
	comment := "please rework"
	approvalRepo := &mockClientPortalApprovalRepo{
		approvalPrincipalHasAccessFn: func(ctx context.Context, approvalID, principal string) bool {
			return true
		},
		recordDecisionFn: func(ctx context.Context, approvalID, principal string, req models.ApprovalDecisionRequest) (*models.ClientApprovalItem, string, error) {
			return &updatedItem, "proj-1", nil
		},
	}
	metrics := observability.NewMetrics()
	logger := observability.NewLogger()
	svc := newTestClientPortalServiceWithObservability(nil, approvalRepo, nil, logger, metrics)

	result, err := svc.DecideApproval(context.Background(), "a1", "client-1", models.ApprovalDecisionRequest{
		Outcome: "request_changes",
		Comment: &comment,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success=true")
	}
	// Verify requested-changes gauge was recorded
	if snap := metrics.Snapshot(); snap.RequestedChangesCurrent == 0 {
		t.Error("RequestedChangesCurrent: expected non-zero after request_changes decision")
	}
}

func TestCreateComment_WiresMetrics(t *testing.T) {
	commentRepo := &mockClientPortalCommentRepo{
		createCommentFn: func(ctx context.Context, c *models.ClientComment) error {
			return nil
		},
	}
	metrics := observability.NewMetrics()
	logger := observability.NewLogger()
	svc := newTestClientPortalServiceWithObservability(nil, nil, commentRepo, logger, metrics)

	err := svc.CreateComment(context.Background(), &models.ClientComment{
		ID:            "cmt_test",
		ProjectID:     "proj-1",
		RelatedItemID: "item-1",
		AuthorName:    "client-1",
		Body:          "test comment",
		CreatedAt:     time.Now(),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if snap := metrics.Snapshot(); snap.CommentCreatedTotal != 1 {
		t.Errorf("CommentCreatedTotal: expected 1, got %d", snap.CommentCreatedTotal)
	}
}

func TestEditComment_WiresMetrics(t *testing.T) {
	commentRepo := &mockClientPortalCommentRepo{
		editCommentFn: func(ctx context.Context, commentID, authorID, newBody string) (*models.ClientComment, error) {
			return &models.ClientComment{
				ID:        commentID,
				ProjectID: "proj-1",
				Body:      newBody,
			}, nil
		},
	}
	metrics := observability.NewMetrics()
	logger := observability.NewLogger()
	svc := newTestClientPortalServiceWithObservability(nil, nil, commentRepo, logger, metrics)

	_, err := svc.EditComment(context.Background(), "cmt_test", "client-1", "updated body")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if snap := metrics.Snapshot(); snap.CommentEditedTotal != 1 {
		t.Errorf("CommentEditedTotal: expected 1, got %d", snap.CommentEditedTotal)
	}
}

func TestDeleteComment_WiresMetrics(t *testing.T) {
	commentRepo := &mockClientPortalCommentRepo{
		deleteCommentFn: func(ctx context.Context, commentID, authorID string) (bool, error) {
			return true, nil
		},
	}
	metrics := observability.NewMetrics()
	logger := observability.NewLogger()
	svc := newTestClientPortalServiceWithObservability(nil, nil, commentRepo, logger, metrics)

	deleted, err := svc.DeleteComment(context.Background(), "cmt_test", "client-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !deleted {
		t.Error("expected deleted=true")
	}
	if snap := metrics.Snapshot(); snap.CommentDeletedTotal != 1 {
		t.Errorf("CommentDeletedTotal: expected 1, got %d", snap.CommentDeletedTotal)
	}
}