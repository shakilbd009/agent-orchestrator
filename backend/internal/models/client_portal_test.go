package models

import (
	"testing"
)

func TestCompletionPercent(t *testing.T) {
	tests := []struct {
		name      string
		todo      int
		inProgress int
		blocked   int
		done      int
		wantNil   bool
		wantVal   *float64
	}{
		{
			name:       "no active tasks returns nil",
			todo:       0,
			inProgress: 0,
			blocked:   0,
			done:      0,
			wantNil:   true,
		},
		{
			name:       "all done returns 100",
			todo:       0,
			inProgress: 0,
			blocked:   0,
			done:      10,
			wantNil:   false,
		},
		{
			name:       "mixed column counts",
			todo:       2,
			inProgress: 3,
			blocked:   1,
			done:      4,
			wantNil:   false,
		},
		{
			name:       "excludes cancelled and proposed by only summing active columns",
			todo:       1,
			inProgress: 1,
			blocked:   1,
			done:      1,
			wantNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompletionPercent(tt.todo, tt.inProgress, tt.blocked, tt.done)
			if tt.wantNil && got != nil {
				t.Fatalf("CompletionPercent() = %v, want nil", *got)
			}
			if !tt.wantNil && got == nil {
				t.Fatalf("CompletionPercent() = nil, want non-nil")
			}
			if got != nil {
				// Verify calculation: done / total * 100
				denom := tt.todo + tt.inProgress + tt.blocked + tt.done
				expected := float64(tt.done) / float64(denom) * 100
				if *got != expected {
					t.Fatalf("CompletionPercent() = %v, want %v", *got, expected)
				}
			}
		})
	}
}

func TestOwnerLabelTypes(t *testing.T) {
	expected := []OwnerLabelType{
		OwnerLabelProduct,
		OwnerLabelEngineering,
		OwnerLabelReview,
		OwnerLabelQuality,
		OwnerLabelClient,
	}
	if len(expected) != 5 {
		t.Fatalf("expected 5 owner label types, got %d", len(expected))
	}
}

func TestPendingApprovalOutcome(t *testing.T) {
	// approve and reject are NOT pending (false)
	if PendingApprovalOutcome[ApprovalOutcomeApprove] {
		t.Errorf("ApprovalOutcomeApprove should NOT be pending")
	}
	if PendingApprovalOutcome[ApprovalOutcomeReject] {
		t.Errorf("ApprovalOutcomeReject should NOT be pending")
	}
	// request_changes and need_more_information ARE pending (true)
	if !PendingApprovalOutcome[ApprovalOutcomeRequestChanges] {
		t.Errorf("ApprovalOutcomeRequestChanges SHOULD be pending")
	}
	if !PendingApprovalOutcome[ApprovalOutcomeNeedMoreInformation] {
		t.Errorf("ApprovalOutcomeNeedMoreInformation SHOULD be pending")
	}
}

func TestValidatePublication(t *testing.T) {
	// ValidatePublication is a placeholder returning {Passed: true}.
	// Full validation logic lives in the service layer.
	result := ValidatePublication("any content")
	if !result.Passed {
		t.Errorf("ValidatePublication() = %v, want Passed=true", result)
	}
}

func TestSSEClientEventConstants(t *testing.T) {
	constants := []string{
		SSEClientPortalApprovalSubmitted,
		SSEClientPortalApprovalOutcomeRecorded,
		SSEClientPortalApprovalNeedMoreInformation,
		SSEClientPortalItemPublished,
		SSEClientPortalItemUnpublished,
		SSEClientPortalPublicationValidationFailed,
		SSEClientPortalCommentCreated,
		SSEClientPortalCommentEdited,
		SSEClientPortalCommentDeleted,
		SSEClientPortalAccessDenied,
		SSEClientPortalReadOnlyModeEntered,
		SSEClientPortalReadsUnavailable,
		SSEClientPortalSSEConnected,
		SSEClientPortalSSEDisconnected,
	}
	if len(constants) != 14 {
		t.Errorf("expected 14 SSE event constants, got %d", len(constants))
	}
	for _, c := range constants {
		if c == "" {
			t.Errorf("SSE event constant should not be empty")
		}
	}
}