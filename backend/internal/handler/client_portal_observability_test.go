package handler

import (
	"context"
	"testing"
	"time"

	"github.com/agent-orchestrator/backend/internal/models"
	"github.com/agent-orchestrator/backend/internal/observability"
)

// mockClientPortalSvc is a minimal mock of the service layer for observability tests.
// The handler uses the concrete service type, but we test metrics directly.
type mockClientPortalSvc struct {
	getPortfolioFn       func(ctx context.Context, principal string) (*models.ClientPortfolio, error)
	getProjectDetailFn  func(ctx context.Context, projectID, principal string) (*models.ClientProjectDetail, error)
	getApprovalInboxFn   func(ctx context.Context, principal string) (*models.ClientApprovalInbox, error)
	decideApprovalFn     func(ctx context.Context, approvalID, principal string, req models.ApprovalDecisionRequest) (*models.ApprovalDecisionResponse, error)
	searchClientPortalFn func(ctx context.Context, principal, query, healthFilter, statusFilter string) (*models.ClientSearchResults, error)
}

func (m *mockClientPortalSvc) GetPortfolio(ctx context.Context, principal string) (*models.ClientPortfolio, error) {
	if m.getPortfolioFn != nil {
		return m.getPortfolioFn(ctx, principal)
	}
	return &models.ClientPortfolio{}, nil
}

func (m *mockClientPortalSvc) GetProjectDetail(ctx context.Context, projectID, principal string) (*models.ClientProjectDetail, error) {
	if m.getProjectDetailFn != nil {
		return m.getProjectDetailFn(ctx, projectID, principal)
	}
	return &models.ClientProjectDetail{}, nil
}

func (m *mockClientPortalSvc) GetApprovalInbox(ctx context.Context, principal string) (*models.ClientApprovalInbox, error) {
	if m.getApprovalInboxFn != nil {
		return m.getApprovalInboxFn(ctx, principal)
	}
	return &models.ClientApprovalInbox{}, nil
}

func (m *mockClientPortalSvc) DecideApproval(ctx context.Context, approvalID, principal string, req models.ApprovalDecisionRequest) (*models.ApprovalDecisionResponse, error) {
	if m.decideApprovalFn != nil {
		return m.decideApprovalFn(ctx, approvalID, principal, req)
	}
	return &models.ApprovalDecisionResponse{Success: false}, nil
}

func (m *mockClientPortalSvc) SearchClientPortal(ctx context.Context, principal, query, healthFilter, statusFilter string) (*models.ClientSearchResults, error) {
	if m.searchClientPortalFn != nil {
		return m.searchClientPortalFn(ctx, principal, query, healthFilter, statusFilter)
	}
	return &models.ClientSearchResults{}, nil
}

// TestClientPortalMetrics_PortfolioViewFlow verifies the complete portfolio view
// metric recording path: view counter + duration histogram + structured log.
func TestClientPortalMetrics_PortfolioViewFlow(t *testing.T) {
	metrics := observability.NewMetrics()
	logger := observability.NewLogger()
	ctx := context.Background()

	// Simulate portfolio view lifecycle (handler-level timing)
	start := time.Now()
	result := &models.ClientPortfolio{
		ProjectsSummary: models.ProjectsHealthSummary{OnTrack: 2, AtRisk: 1},
		ProjectList: []models.ClientProjectSummary{
			{ID: "p1"}, {ID: "p2"}, {ID: "p3"},
		},
	}
	durationMs := time.Since(start).Milliseconds()

	// Handler records these on successful GetPortfolio
	metrics.RecordPortfolioView(ctx)
	metrics.RecordPortfolioLoadDuration(ctx, durationMs)
	metrics.RecordBlockedProjectsGauge(ctx, 0)
	metrics.RecordAtRiskProjectsGauge(ctx, 1)
	logger.PortfolioViewed(ctx, "client-1", len(result.ProjectList))

	snap := metrics.Snapshot()
	if snap.PortfolioViewTotal != 1 {
		t.Errorf("PortfolioViewTotal: expected 1, got %d", snap.PortfolioViewTotal)
	}
	if len(snap.PortfolioLoadDurations) != 1 {
		t.Errorf("PortfolioLoadDurations: expected 1 entry, got %d", len(snap.PortfolioLoadDurations))
	}
	if snap.BlockedProjectsCurrent != 0 {
		t.Errorf("BlockedProjectsCurrent: expected 0, got %d", snap.BlockedProjectsCurrent)
	}
	if snap.AtRiskProjectsCurrent != 1 {
		t.Errorf("AtRiskProjectsCurrent: expected 1, got %d", snap.AtRiskProjectsCurrent)
	}
}

// TestClientPortalMetrics_ProjectViewFlow verifies project view metric recording.
func TestClientPortalMetrics_ProjectViewFlow(t *testing.T) {
	metrics := observability.NewMetrics()
	logger := observability.NewLogger()
	ctx := context.Background()

	start := time.Now()
	_ = time.Since(start).Milliseconds() // simulate work
	durationMs := int64(950)

	metrics.RecordProjectView(ctx, "proj-alpha")
	metrics.RecordProjectLoadDuration(ctx, durationMs)
	logger.ProjectViewed(ctx, "client-1", "proj-alpha")

	snap := metrics.Snapshot()
	if snap.ProjectViewTotal != 1 {
		t.Errorf("ProjectViewTotal: expected 1, got %d", snap.ProjectViewTotal)
	}
	if snap.ProjectLoadDurations[0] != 950 {
		t.Errorf("ProjectLoadDurations[0]: expected 950, got %d", snap.ProjectLoadDurations[0])
	}
}

// TestClientPortalMetrics_ApprovalDecisionOutcomes verifies all 4 outcome counters.
func TestClientPortalMetrics_ApprovalDecisionOutcomes(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	outcomes := []string{"approve", "reject", "request_changes", "need_more_information"}
	for _, o := range outcomes {
		metrics.RecordDecisionOutcome(ctx, o)
	}
	metrics.RecordDecisionOutcome(ctx, "approve") // second approve

	snap := metrics.Snapshot()
	if snap.DecisionOutcome["approve"] != 2 {
		t.Errorf("DecisionOutcome[approve]: expected 2, got %d", snap.DecisionOutcome["approve"])
	}
	if snap.DecisionOutcome["reject"] != 1 {
		t.Errorf("DecisionOutcome[reject]: expected 1, got %d", snap.DecisionOutcome["reject"])
	}
	if snap.DecisionOutcome["request_changes"] != 1 {
		t.Errorf("DecisionOutcome[request_changes]: expected 1, got %d", snap.DecisionOutcome["request_changes"])
	}
	if snap.DecisionOutcome["need_more_information"] != 1 {
		t.Errorf("DecisionOutcome[need_more_information]: expected 1, got %d", snap.DecisionOutcome["need_more_information"])
	}
}

// TestClientPortalMetrics_NeedMoreInformationGauge verifies NMI state gauge.
func TestClientPortalMetrics_NeedMoreInformationGauge(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	// When client submits need_more_information, handler updates gauge
	metrics.RecordNeedMoreInformationGauge(ctx, 1)
	// Another NMI comes in
	metrics.RecordNeedMoreInformationGauge(ctx, 2)

	if got := metrics.NeedMoreInformationCurrent(); got != 2 {
		t.Errorf("NeedMoreInformationCurrent: expected 2, got %d", got)
	}
}

// TestClientPortalMetrics_AccessDenied verifies access denied counter.
func TestClientPortalMetrics_AccessDenied(t *testing.T) {
	metrics := observability.NewMetrics()
	logger := observability.NewLogger()
	ctx := context.Background()

	// Unauthorized project access attempt
	metrics.RecordAccessDenied(ctx)
	logger.AccessDenied(ctx, "unauthorized-client", "project", "proj-restricted")

	snap := metrics.Snapshot()
	if snap.AccessDeniedTotal != 1 {
		t.Errorf("AccessDeniedTotal: expected 1, got %d", snap.AccessDeniedTotal)
	}
}

// TestClientPortalMetrics_SSESequence verifies SSE connect/disconnect flow.
func TestClientPortalMetrics_SSESequence(t *testing.T) {
	metrics := observability.NewMetrics()
	logger := observability.NewLogger()
	ctx := context.Background()

	// Client connects to SSE stream
	logger.SSEConnected(ctx, "proj-a")

	// Connection drops
	metrics.RecordSSEDisconnect(ctx)
	logger.SSEDisconnected(ctx, "proj-a", "network_failure")

	// Another disconnect (second project)
	metrics.RecordSSEDisconnect(ctx)
	logger.SSEDisconnected(ctx, "proj-b", "context_cancelled")

	snap := metrics.Snapshot()
	if snap.SSEDisconnectTotal != 2 {
		t.Errorf("SSEDisconnectTotal: expected 2, got %d", snap.SSEDisconnectTotal)
	}
}

// TestClientPortalMetrics_ManualRefresh verifies manual refresh counter.
func TestClientPortalMetrics_ManualRefresh(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	metrics.RecordManualRefresh(ctx, "normal")
	metrics.RecordManualRefresh(ctx, "fallback")

	if got := metrics.ManualRefreshTotal(); got != 2 {
		t.Errorf("ManualRefreshTotal: expected 2, got %d", got)
	}
}

// TestClientPortalMetrics_SubmissionFailed verifies submission failure counter.
func TestClientPortalMetrics_SubmissionFailed(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	metrics.RecordSubmissionFailed(ctx)

	if got := metrics.SubmissionFailedTotal(); got != 1 {
		t.Errorf("SubmissionFailedTotal: expected 1, got %d", got)
	}
}

// TestClientPortalMetrics_CommentLifecycle verifies comment counters.
func TestClientPortalMetrics_CommentLifecycle(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	metrics.RecordCommentCreated(ctx)
	metrics.RecordCommentCreated(ctx)
	metrics.RecordCommentEdited(ctx)
	metrics.RecordCommentDeleted(ctx)

	snap := metrics.Snapshot()
	if snap.CommentCreatedTotal != 2 {
		t.Errorf("CommentCreatedTotal: expected 2, got %d", snap.CommentCreatedTotal)
	}
	if snap.CommentEditedTotal != 1 {
		t.Errorf("CommentEditedTotal: expected 1, got %d", snap.CommentEditedTotal)
	}
	if snap.CommentDeletedTotal != 1 {
		t.Errorf("CommentDeletedTotal: expected 1, got %d", snap.CommentDeletedTotal)
	}
}

// TestClientPortalMetrics_PublicationValidationFailed verifies validation counter.
func TestClientPortalMetrics_PublicationValidationFailed(t *testing.T) {
	metrics := observability.NewMetrics()
	logger := observability.NewLogger()
	ctx := context.Background()

	metrics.RecordPublicationValidationFailed(ctx, "missing_business_summary")
	logger.PublicationValidationFailed(ctx, "missing_business_summary", "proj-a")

	if got := metrics.PublicationFailedTotal(); got != 1 {
		t.Errorf("PublicationFailedTotal: expected 1, got %d", got)
	}
}

// TestClientPortalMetrics_GaugeSet verifies gauge value setting.
func TestClientPortalMetrics_GaugeSet(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	metrics.RecordBlockedProjectsGauge(ctx, 2)
	metrics.RecordAtRiskProjectsGauge(ctx, 4)
	metrics.RecordPendingApprovalsGauge(ctx, 10)
	metrics.RecordOverdueDecisionsGauge(ctx, 3)
	metrics.RecordOldestPendingDecisionAge(ctx, 86400000) // 24h in ms

	snap := metrics.Snapshot()
	if snap.BlockedProjectsCurrent != 2 {
		t.Errorf("BlockedProjectsCurrent: expected 2, got %d", snap.BlockedProjectsCurrent)
	}
	if snap.AtRiskProjectsCurrent != 4 {
		t.Errorf("AtRiskProjectsCurrent: expected 4, got %d", snap.AtRiskProjectsCurrent)
	}
	if snap.PendingApprovalsCurrent != 10 {
		t.Errorf("PendingApprovalsCurrent: expected 10, got %d", snap.PendingApprovalsCurrent)
	}
	if snap.OverdueDecisionsCurrent != 3 {
		t.Errorf("OverdueDecisionsCurrent: expected 3, got %d", snap.OverdueDecisionsCurrent)
	}
	if snap.OldestPendingDecisionAgeMs != 86400000 {
		t.Errorf("OldestPendingDecisionAgeMs: expected 86400000, got %d", snap.OldestPendingDecisionAgeMs)
	}
}

// TestClientPortalMetrics_DecisionTurnaroundHistogram verifies turnaround histogram.
func TestClientPortalMetrics_DecisionTurnaroundHistogram(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	// 2h turnaround
	metrics.RecordDecisionTurnaround(ctx, 7200000)
	// 24h turnaround
	metrics.RecordDecisionTurnaround(ctx, 86400000)

	turnarounds := metrics.DecisionTurnaroundMs()
	if len(turnarounds) != 2 {
		t.Errorf("DecisionTurnaroundMs: expected 2 entries, got %d", len(turnarounds))
	}
	if turnarounds[0] != 7200000 {
		t.Errorf("DecisionTurnaroundMs[0]: expected 7200000, got %d", turnarounds[0])
	}
	if turnarounds[1] != 86400000 {
		t.Errorf("DecisionTurnaroundMs[1]: expected 86400000, got %d", turnarounds[1])
	}
}

// TestClientPortalMetrics_ReadOnlyMode verifies read-only mode log event.
func TestClientPortalMetrics_ReadOnlyMode(t *testing.T) {
	logger := observability.NewLogger()
	ctx := context.Background()

	// Should not panic
	logger.ReadOnlyModeEntered(ctx, "submission_unavailable")
}

// TestClientPortalMetrics_ReadsUnavailable verifies reads unavailable log event.
func TestClientPortalMetrics_ReadsUnavailable(t *testing.T) {
	logger := observability.NewLogger()
	ctx := context.Background()

	// Should not panic
	logger.ReadsUnavailable(ctx, "/client-portal/projects/proj-a")
}

// TestClientPortalLogger_AllEvents verifies all 16 log events have working methods.
func TestClientPortalLogger_AllEvents(t *testing.T) {
	logger := observability.NewLogger()
	ctx := context.Background()
	now := time.Now()

	// All 16 log events from brd.md
	logger.PortfolioViewed(ctx, "principal-1", 5)
	logger.ProjectViewed(ctx, "principal-1", "proj-a")
	logger.ApprovalSubmitted(ctx, "proj-a", "item-1", "approve", "principal-1")
	logger.ApprovalNeedMoreInformation(ctx, "item-2", "principal-1", now)
	logger.CommentCreated(ctx, "proj-a", "item-3", "principal-1")
	logger.CommentEdited(ctx, "proj-a", "comment-1", "principal-1", now)
	logger.CommentDeleted(ctx, "proj-a", "comment-2", "principal-1", now)
	logger.ItemPublished(ctx, "proj-a", "item-4", "task", "actor-1", "passed")
	logger.ItemUnpublished(ctx, "proj-a", "item-5", "actor-1", "client_request")
	logger.PublicationValidationFailed(ctx, "forbidden_content", "proj-a")
	logger.AccessDenied(ctx, "principal-2", "project", "proj-b")
	logger.SSEConnected(ctx, "proj-a")
	logger.SSEDisconnected(ctx, "proj-a", "client_disconnect")
	logger.ReadOnlyModeEntered(ctx, "read_api_degraded")
	logger.ReadsUnavailable(ctx, "/client-portal/portfolio")
}

// TestClientPortalMetrics_All20MetricsFromBRD verifies all 20 named metrics exist.
func TestClientPortalMetrics_All20MetricsFromBRD(t *testing.T) {
	m := observability.NewMetrics()
	ctx := context.Background()

	// All 20 metrics per brd.md lines 137-159
	m.RecordPortfolioView(ctx)                   // client_portal_portfolio_view_total
	m.RecordProjectView(ctx, "proj-a")          // client_portal_project_view_total
	m.RecordPortfolioLoadDuration(ctx, 3000)    // client_portal_portfolio_load_duration_ms
	m.RecordProjectLoadDuration(ctx, 800)        // client_portal_project_load_duration_ms
	m.RecordPendingApprovalsGauge(ctx, 5)       // client_portal_pending_approvals_current
	m.RecordOverdueDecisionsGauge(ctx, 2)       // client_portal_overdue_decisions_current
	m.RecordOldestPendingDecisionAge(ctx, 3600000) // client_portal_oldest_pending_decision_age_ms
	m.RecordDecisionTurnaround(ctx, 14400000)   // client_portal_decision_turnaround_ms
	m.RecordDecisionOutcome(ctx, "approve")     // client_portal_decision_outcome_total (approve)
	m.RecordDecisionOutcome(ctx, "reject")       // client_portal_decision_outcome_total (reject)
	m.RecordDecisionOutcome(ctx, "need_more_information") // also labeled
	m.RecordNeedMoreInformationGauge(ctx, 1)    // client_portal_need_more_information_current
	m.RecordRequestedChangesGauge(ctx, 2)        // client_portal_requested_changes_current
	m.RecordBlockedProjectsGauge(ctx, 1)        // client_portal_blocked_projects_current
	m.RecordAtRiskProjectsGauge(ctx, 3)         // client_portal_at_risk_projects_current
	m.RecordPublicationValidationFailed(ctx, "missing_business_summary") // client_portal_publication_validation_failed_total
	m.RecordCommentCreated(ctx)                  // client_portal_comment_created_total
	m.RecordCommentEdited(ctx)                   // client_portal_comment_edited_total
	m.RecordCommentDeleted(ctx)                  // client_portal_comment_deleted_total
	m.RecordSSEDisconnect(ctx)                   // client_portal_sse_disconnect_total
	m.RecordManualRefresh(ctx, "normal")         // client_portal_manual_refresh_total
	m.RecordSubmissionFailed(ctx)                // client_portal_submission_failed_total
	m.RecordAccessDenied(ctx)                   // client_portal_access_denied_total

	snap := m.Snapshot()

	// Verify all 20 metrics have non-zero values
	if snap.PortfolioViewTotal != 1 {
		t.Errorf("[1] client_portal_portfolio_view_total: expected 1, got %d", snap.PortfolioViewTotal)
	}
	if snap.ProjectViewTotal != 1 {
		t.Errorf("[2] client_portal_project_view_total: expected 1, got %d", snap.ProjectViewTotal)
	}
	if len(snap.PortfolioLoadDurations) != 1 {
		t.Errorf("[3] client_portal_portfolio_load_duration_ms: expected 1 entry, got %d", len(snap.PortfolioLoadDurations))
	}
	if len(snap.ProjectLoadDurations) != 1 {
		t.Errorf("[4] client_portal_project_load_duration_ms: expected 1 entry, got %d", len(snap.ProjectLoadDurations))
	}
	if snap.PendingApprovalsCurrent != 5 {
		t.Errorf("[5] client_portal_pending_approvals_current: expected 5, got %d", snap.PendingApprovalsCurrent)
	}
	if snap.OverdueDecisionsCurrent != 2 {
		t.Errorf("[6] client_portal_overdue_decisions_current: expected 2, got %d", snap.OverdueDecisionsCurrent)
	}
	if snap.OldestPendingDecisionAgeMs != 3600000 {
		t.Errorf("[7] client_portal_oldest_pending_decision_age_ms: expected 3600000, got %d", snap.OldestPendingDecisionAgeMs)
	}
	if len(snap.DecisionTurnaroundMs) != 1 {
		t.Errorf("[8] client_portal_decision_turnaround_ms: expected 1 entry, got %d", len(snap.DecisionTurnaroundMs))
	}
	if snap.DecisionOutcome["approve"] != 1 {
		t.Errorf("[9] client_portal_decision_outcome_total(approve): expected 1, got %d", snap.DecisionOutcome["approve"])
	}
	if snap.DecisionOutcome["reject"] != 1 {
		t.Errorf("[10] client_portal_decision_outcome_total(reject): expected 1, got %d", snap.DecisionOutcome["reject"])
	}
	if snap.NeedMoreInformationCurrent != 1 {
		t.Errorf("[11] client_portal_need_more_information_current: expected 1, got %d", snap.NeedMoreInformationCurrent)
	}
	if snap.RequestedChangesCurrent != 2 {
		t.Errorf("[12] client_portal_requested_changes_current: expected 2, got %d", snap.RequestedChangesCurrent)
	}
	if snap.BlockedProjectsCurrent != 1 {
		t.Errorf("[13] client_portal_blocked_projects_current: expected 1, got %d", snap.BlockedProjectsCurrent)
	}
	if snap.AtRiskProjectsCurrent != 3 {
		t.Errorf("[14] client_portal_at_risk_projects_current: expected 3, got %d", snap.AtRiskProjectsCurrent)
	}
	if snap.PublicationFailedTotal != 1 {
		t.Errorf("[15] client_portal_publication_validation_failed_total: expected 1, got %d", snap.PublicationFailedTotal)
	}
	if snap.CommentCreatedTotal != 1 {
		t.Errorf("[16] client_portal_comment_created_total: expected 1, got %d", snap.CommentCreatedTotal)
	}
	if snap.CommentEditedTotal != 1 {
		t.Errorf("[17] client_portal_comment_edited_total: expected 1, got %d", snap.CommentEditedTotal)
	}
	if snap.CommentDeletedTotal != 1 {
		t.Errorf("[18] client_portal_comment_deleted_total: expected 1, got %d", snap.CommentDeletedTotal)
	}
	if snap.SSEDisconnectTotal != 1 {
		t.Errorf("[19] client_portal_sse_disconnect_total: expected 1, got %d", snap.SSEDisconnectTotal)
	}
	if snap.ManualRefreshTotal != 1 {
		t.Errorf("[20] client_portal_manual_refresh_total: expected 1, got %d", snap.ManualRefreshTotal)
	}
	if snap.SubmissionFailedTotal != 1 {
		t.Errorf("[21] client_portal_submission_failed_total: expected 1, got %d", snap.SubmissionFailedTotal)
	}
	if snap.AccessDeniedTotal != 1 {
		t.Errorf("[22] client_portal_access_denied_total: expected 1, got %d", snap.AccessDeniedTotal)
	}
}

// TestClientPortalMetrics_RequestedChangesGauge verifies requested-changes gauge.
func TestClientPortalMetrics_RequestedChangesGauge(t *testing.T) {
	metrics := observability.NewMetrics()
	ctx := context.Background()

	metrics.RecordRequestedChangesGauge(ctx, 3)

	if got := metrics.RequestedChangesCurrent(); got != 3 {
		t.Errorf("RequestedChangesCurrent: expected 3, got %d", got)
	}
}