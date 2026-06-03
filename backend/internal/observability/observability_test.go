package observability

import (
	"context"
	"testing"
	"time"
)

func TestMetrics_Counters(t *testing.T) {
	m := NewMetrics()
	ctx := context.Background()

	// Portfolio view counter
	m.RecordPortfolioView(ctx)
	m.RecordPortfolioView(ctx)
	if got := m.PortfolioViewTotal(); got != 2 {
		t.Errorf("PortfolioViewTotal: expected 2, got %d", got)
	}

	// Project view counter
	m.RecordProjectView(ctx, "proj-a")
	m.RecordProjectView(ctx, "proj-b")
	if got := m.ProjectViewTotal(); got != 2 {
		t.Errorf("ProjectViewTotal: expected 2, got %d", got)
	}

	// Publication failed counter
	m.RecordPublicationValidationFailed(ctx, "missing_business_summary")
	if got := m.PublicationFailedTotal(); got != 1 {
		t.Errorf("PublicationFailedTotal: expected 1, got %d", got)
	}

	// Comment counters
	m.RecordCommentCreated(ctx)
	m.RecordCommentCreated(ctx)
	m.RecordCommentEdited(ctx)
	m.RecordCommentDeleted(ctx)
	if got := m.CommentCreatedTotal(); got != 2 {
		t.Errorf("CommentCreatedTotal: expected 2, got %d", got)
	}
	if got := m.CommentEditedTotal(); got != 1 {
		t.Errorf("CommentEditedTotal: expected 1, got %d", got)
	}
	if got := m.CommentDeletedTotal(); got != 1 {
		t.Errorf("CommentDeletedTotal: expected 1, got %d", got)
	}

	// SSE disconnect counter
	m.RecordSSEDisconnect(ctx)
	m.RecordSSEDisconnect(ctx)
	if got := m.SSEDisconnectTotal(); got != 2 {
		t.Errorf("SSEDisconnectTotal: expected 2, got %d", got)
	}

	// Manual refresh counter
	m.RecordManualRefresh(ctx, "fallback")
	if got := m.ManualRefreshTotal(); got != 1 {
		t.Errorf("ManualRefreshTotal: expected 1, got %d", got)
	}

	// Submission failed counter
	m.RecordSubmissionFailed(ctx)
	if got := m.SubmissionFailedTotal(); got != 1 {
		t.Errorf("SubmissionFailedTotal: expected 1, got %d", got)
	}

	// Access denied counter
	m.RecordAccessDenied(ctx)
	m.RecordAccessDenied(ctx)
	if got := m.AccessDeniedTotal(); got != 2 {
		t.Errorf("AccessDeniedTotal: expected 2, got %d", got)
	}
}

func TestMetrics_DecisionOutcomes(t *testing.T) {
	m := NewMetrics()
	ctx := context.Background()

	m.RecordDecisionOutcome(ctx, "approve")
	m.RecordDecisionOutcome(ctx, "approve")
	m.RecordDecisionOutcome(ctx, "reject")
	m.RecordDecisionOutcome(ctx, "need_more_information")
	m.RecordDecisionOutcome(ctx, "request_changes")

	if got := m.DecisionOutcome("approve"); got != 2 {
		t.Errorf("DecisionOutcome(approve): expected 2, got %d", got)
	}
	if got := m.DecisionOutcome("reject"); got != 1 {
		t.Errorf("DecisionOutcome(reject): expected 1, got %d", got)
	}
	if got := m.DecisionOutcome("need_more_information"); got != 1 {
		t.Errorf("DecisionOutcome(need_more_information): expected 1, got %d", got)
	}
	if got := m.DecisionOutcome("request_changes"); got != 1 {
		t.Errorf("DecisionOutcome(request_changes): expected 1, got %d", got)
	}
	if got := m.DecisionOutcome("unknown"); got != 0 {
		t.Errorf("DecisionOutcome(unknown): expected 0, got %d", got)
	}
}

func TestMetrics_Gauges(t *testing.T) {
	m := NewMetrics()
	ctx := context.Background()

	m.RecordPendingApprovalsGauge(ctx, 5)
	m.RecordOverdueDecisionsGauge(ctx, 2)
	m.RecordNeedMoreInformationGauge(ctx, 1)
	m.RecordRequestedChangesGauge(ctx, 3)
	m.RecordBlockedProjectsGauge(ctx, 1)
	m.RecordAtRiskProjectsGauge(ctx, 4)
	m.RecordOldestPendingDecisionAge(ctx, 3600000) // 1 hour in ms

	if got := m.PendingApprovalsCurrent(); got != 5 {
		t.Errorf("PendingApprovalsCurrent: expected 5, got %d", got)
	}
	if got := m.OverdueDecisionsCurrent(); got != 2 {
		t.Errorf("OverdueDecisionsCurrent: expected 2, got %d", got)
	}
	if got := m.NeedMoreInformationCurrent(); got != 1 {
		t.Errorf("NeedMoreInformationCurrent: expected 1, got %d", got)
	}
	if got := m.RequestedChangesCurrent(); got != 3 {
		t.Errorf("RequestedChangesCurrent: expected 3, got %d", got)
	}
	if got := m.BlockedProjectsCurrent(); got != 1 {
		t.Errorf("BlockedProjectsCurrent: expected 1, got %d", got)
	}
	if got := m.AtRiskProjectsCurrent(); got != 4 {
		t.Errorf("AtRiskProjectsCurrent: expected 4, got %d", got)
	}
	if got := m.OldestPendingDecisionAgeMs(); got != 3600000 {
		t.Errorf("OldestPendingDecisionAgeMs: expected 3600000, got %d", got)
	}
}

func TestMetrics_Histograms(t *testing.T) {
	m := NewMetrics()
	ctx := context.Background()

	m.RecordPortfolioLoadDuration(ctx, 1200)
	m.RecordPortfolioLoadDuration(ctx, 3500)
	m.RecordProjectLoadDuration(ctx, 800)
	m.RecordProjectLoadDuration(ctx, 1900)
	m.RecordDecisionTurnaround(ctx, 7200000) // 2h in ms
	m.RecordDecisionTurnaround(ctx, 86400000) // 24h in ms

	portfolioDurations := m.PortfolioLoadDurations()
	if len(portfolioDurations) != 2 {
		t.Errorf("PortfolioLoadDurations: expected 2 values, got %d", len(portfolioDurations))
	}
	if portfolioDurations[0] != 1200 {
		t.Errorf("PortfolioLoadDurations[0]: expected 1200, got %d", portfolioDurations[0])
	}

	projectDurations := m.ProjectLoadDurations()
	if len(projectDurations) != 2 {
		t.Errorf("ProjectLoadDurations: expected 2 values, got %d", len(projectDurations))
	}

	turnarounds := m.DecisionTurnaroundMs()
	if len(turnarounds) != 2 {
		t.Errorf("DecisionTurnaroundMs: expected 2 values, got %d", len(turnarounds))
	}
}

func TestMetrics_Snapshot(t *testing.T) {
	m := NewMetrics()
	ctx := context.Background()

	m.RecordPortfolioView(ctx)
	m.RecordProjectView(ctx, "proj-x")
	m.RecordPublicationValidationFailed(ctx, "forbidden_content")
	m.RecordCommentCreated(ctx)
	m.RecordSSEDisconnect(ctx)
	m.RecordManualRefresh(ctx, "normal")
	m.RecordAccessDenied(ctx)
	m.RecordDecisionOutcome(ctx, "approve")
	m.RecordDecisionOutcome(ctx, "reject")
	m.RecordPendingApprovalsGauge(ctx, 7)
	m.RecordOverdueDecisionsGauge(ctx, 3)
	m.RecordBlockedProjectsGauge(ctx, 2)
	m.RecordAtRiskProjectsGauge(ctx, 5)
	m.RecordPortfolioLoadDuration(ctx, 2100)
	m.RecordProjectLoadDuration(ctx, 950)

	snap := m.Snapshot()

	if snap.PortfolioViewTotal != 1 {
		t.Errorf("PortfolioViewTotal: expected 1, got %d", snap.PortfolioViewTotal)
	}
	if snap.ProjectViewTotal != 1 {
		t.Errorf("ProjectViewTotal: expected 1, got %d", snap.ProjectViewTotal)
	}
	if snap.PublicationFailedTotal != 1 {
		t.Errorf("PublicationFailedTotal: expected 1, got %d", snap.PublicationFailedTotal)
	}
	if snap.CommentCreatedTotal != 1 {
		t.Errorf("CommentCreatedTotal: expected 1, got %d", snap.CommentCreatedTotal)
	}
	if snap.SSEDisconnectTotal != 1 {
		t.Errorf("SSEDisconnectTotal: expected 1, got %d", snap.SSEDisconnectTotal)
	}
	if snap.ManualRefreshTotal != 1 {
		t.Errorf("ManualRefreshTotal: expected 1, got %d", snap.ManualRefreshTotal)
	}
	if snap.AccessDeniedTotal != 1 {
		t.Errorf("AccessDeniedTotal: expected 1, got %d", snap.AccessDeniedTotal)
	}
	if snap.DecisionOutcome["approve"] != 1 {
		t.Errorf("DecisionOutcome[approve]: expected 1, got %d", snap.DecisionOutcome["approve"])
	}
	if snap.DecisionOutcome["reject"] != 1 {
		t.Errorf("DecisionOutcome[reject]: expected 1, got %d", snap.DecisionOutcome["reject"])
	}
	if snap.PendingApprovalsCurrent != 7 {
		t.Errorf("PendingApprovalsCurrent: expected 7, got %d", snap.PendingApprovalsCurrent)
	}
	if snap.OverdueDecisionsCurrent != 3 {
		t.Errorf("OverdueDecisionsCurrent: expected 3, got %d", snap.OverdueDecisionsCurrent)
	}
	if snap.BlockedProjectsCurrent != 2 {
		t.Errorf("BlockedProjectsCurrent: expected 2, got %d", snap.BlockedProjectsCurrent)
	}
	if snap.AtRiskProjectsCurrent != 5 {
		t.Errorf("AtRiskProjectsCurrent: expected 5, got %d", snap.AtRiskProjectsCurrent)
	}
	if len(snap.PortfolioLoadDurations) != 1 || snap.PortfolioLoadDurations[0] != 2100 {
		t.Errorf("PortfolioLoadDurations: expected [2100], got %v", snap.PortfolioLoadDurations)
	}
	if len(snap.ProjectLoadDurations) != 1 || snap.ProjectLoadDurations[0] != 950 {
		t.Errorf("ProjectLoadDurations: expected [950], got %v", snap.ProjectLoadDurations)
	}
}

func TestLogger_BasicEmission(t *testing.T) {
	l := NewLogger()
	ctx := context.Background()

	// Smoke test: no panics on any log method
	l.PortfolioViewed(ctx, "principal-1", 3)
	l.ProjectViewed(ctx, "principal-1", "proj-a")
	l.ApprovalSubmitted(ctx, "proj-a", "item-1", "approve", "principal-1")
	l.ApprovalNeedMoreInformation(ctx, "item-2", "principal-1", time.Now())
	l.CommentCreated(ctx, "proj-a", "item-3", "principal-1")
	l.CommentEdited(ctx, "proj-a", "comment-1", "principal-1", time.Now())
	l.CommentDeleted(ctx, "proj-a", "comment-2", "principal-1", time.Now())
	l.ItemPublished(ctx, "proj-a", "item-4", "task", "actor-1", "passed")
	l.ItemUnpublished(ctx, "proj-a", "item-5", "actor-1", "client_request")
	l.PublicationValidationFailed(ctx, "missing_business_summary", "proj-a")
	l.AccessDenied(ctx, "principal-2", "project", "proj-b")
	l.SSEConnected(ctx, "proj-a")
	l.SSEDisconnected(ctx, "proj-a", "context_cancelled")
	l.ReadOnlyModeEntered(ctx, "submission_unavailable")
	l.ReadsUnavailable(ctx, "/client-portal/projects/proj-a")
}

func TestElapsedMs(t *testing.T) {
	start := time.Now().Add(-1500 * time.Millisecond)
	if got := ElapsedMs(start); got < 1400 || got > 1600 {
		t.Errorf("ElapsedMs: expected ~1500, got %d", got)
	}
}