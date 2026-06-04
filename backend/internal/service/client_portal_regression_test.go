package service

import (
	"context"
	"testing"
	"time"

	"github.com/agent-orchestrator/backend/internal/models"
	"github.com/agent-orchestrator/backend/internal/observability"
)

// TestDecideApproval_DoesNotRecordDecisionTurnaround verifies that DecideApproval
// does NOT record decision_turnaround on the read path — turnaround is only recorded
// when a decision is actually completed (actionable-to-outcome duration).
//
// Deferral: BRD-03 AC-03-024 defines the 24h overdue threshold and FR-03-019
// defines decision_turnaround as "time from decision becoming client-actionable to
// client outcome." However, the system does not currently store the "actionable time"
// (when the decision became client-actionable), so we cannot compute the
// actionable→outcome duration. This is an explicit PM/architect deferral:
// - FR-03-019 is a should-have (not in current implementation scope)
// - No current production path populates the actionable time field
// - The metric would require BRD-02 decision record extension to track actionable time
//
// This test documents that the metric is intentionally zero while deferred.
// If this test fails, it means someone wired RecordDecisionTurnaround incorrectly
// (e.g., using decision creation time instead of actionable time).
func TestDecideApproval_DoesNotRecordDecisionTurnaround(t *testing.T) {
	updatedItem := models.ClientApprovalItem{ID: "a1", Title: "Approve me", Outcome: "approved"}
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

	_, err := svc.DecideApproval(context.Background(), "a1", "client-1", models.ApprovalDecisionRequest{
		Outcome: "approve",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	snap := metrics.Snapshot()
	// The decision_turnaround histogram should NOT be populated by DecideApproval —
	// we don't have the "actionable time" (when the decision became actionable) stored,
	// so we cannot compute actionable→outcome duration correctly.
	// If a turnaround sample appears here, it means the implementation is computing
	// something wrong (e.g., using decision age instead of actionable-to-outcome duration).
	if len(snap.DecisionTurnaroundMs) > 0 {
		t.Errorf("DecisionTurnaroundMs: expected 0 samples after DecideApproval (turnaround must not be recorded on write path), got %d samples", len(snap.DecisionTurnaroundMs))
	}
}

// TestGetApprovalInbox_DoesNotRecordDecisionTurnaround verifies that GetApprovalInbox
// (read path) does NOT emit decision_turnaround samples — turnaround must only be
// recorded on decision completion with the actual actionable-to-outcome duration.
//
// Deferral: same as TestDecideApproval_DoesNotRecordDecisionTurnaround. The
// decision_turnaround metric requires FR-03-019 actionable time tracking which
// is not in the current implementation scope. GetApprovalInbox is a read path
// and correctly does not emit turnaround samples. If this test fails, the
// implementation has incorrectly wired the metric.
func TestGetApprovalInbox_DoesNotRecordDecisionTurnaround(t *testing.T) {
	oldItem := models.ClientApprovalItem{
		ID:        "a1",
		Title:     "Old approval",
		Outcome:   "pending",
		CreatedAt: time.Now().Add(-48 * time.Hour),
	}
	approvalRepo := &mockClientPortalApprovalRepo{
		listAccessibleApprovalsFn: func(ctx context.Context, principal string) ([]models.ClientApprovalItem, error) {
			return []models.ClientApprovalItem{oldItem}, nil
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
	// GetApprovalInbox records oldest_pending_decision_age_ms as a gauge, but
	// MUST NOT record decision_turnaround (that's only for completed decisions).
	if len(snap.DecisionTurnaroundMs) > 0 {
		t.Errorf("DecisionTurnaroundMs: expected 0 samples on read path (only written on decision completion), got %d", len(snap.DecisionTurnaroundMs))
	}
	if snap.OldestPendingDecisionAgeMs == 0 {
		t.Error("OldestPendingDecisionAgeMs: expected non-zero gauge for 48h-old pending item")
	}
}

// TestDecideApproval_RecordsOutcomeCounterOnce verifies that a successful decision
// results in exactly ONE increment to the decision outcome counter, not two.
func TestDecideApproval_RecordsOutcomeCounterOnce(t *testing.T) {
	updatedItem := models.ClientApprovalItem{ID: "a1", Title: "Approve me", Outcome: "approved"}
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

	_, err := svc.DecideApproval(context.Background(), "a1", "client-1", models.ApprovalDecisionRequest{
		Outcome: "approve",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	snap := metrics.Snapshot()
	if snap.DecisionOutcome["approve"] != 1 {
		t.Errorf("DecisionOutcome[approve]: expected 1, got %d (should be recorded once by service, not duplicated by handler)", snap.DecisionOutcome["approve"])
	}
}

// TestCreateComment_RecordsMetricOnce verifies that creating a comment results in
// exactly one CommentCreatedTotal increment (service layer only, no handler duplicate).
func TestCreateComment_RecordsMetricOnce(t *testing.T) {
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

	snap := metrics.Snapshot()
	if snap.CommentCreatedTotal != 1 {
		t.Errorf("CommentCreatedTotal: expected 1, got %d (metric should be emitted once by service, not duplicated by handler)", snap.CommentCreatedTotal)
	}
}

// TestEditComment_RecordsMetricOnce verifies that editing a comment results in
// exactly one CommentEditedTotal increment.
func TestEditComment_RecordsMetricOnce(t *testing.T) {
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

	snap := metrics.Snapshot()
	if snap.CommentEditedTotal != 1 {
		t.Errorf("CommentEditedTotal: expected 1, got %d", snap.CommentEditedTotal)
	}
}

// TestDeleteComment_RecordsMetricOnce verifies that deleting a comment results in
// exactly one CommentDeletedTotal increment.
func TestDeleteComment_RecordsMetricOnce(t *testing.T) {
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
		t.Fatal("expected deleted=true")
	}

	snap := metrics.Snapshot()
	if snap.CommentDeletedTotal != 1 {
		t.Errorf("CommentDeletedTotal: expected 1, got %d", snap.CommentDeletedTotal)
	}
}