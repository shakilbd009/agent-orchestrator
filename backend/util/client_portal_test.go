package util

import (
	"strings"
	"testing"
	"time"

	"github.com/agent-orchestrator/backend/internal/models"
)

// ---------------------------------------------------------------------------------------------------------------------
// Contract: CompletionPercentageCalculator
// ---------------------------------------------------------------------------------------------------------------------

func TestCompletionPercent(t *testing.T) {
	tests := []struct {
		name       string
		todo       int
		inProgress int
		blocked    int
		done       int
		wantNil    bool
	}{
		{"no active tasks returns nil", 0, 0, 0, 0, true},
		{"all done returns 100", 0, 0, 0, 10, false},
		{"mixed column counts", 2, 3, 1, 4, false},
		{"verification 4/13", 5, 3, 1, 4, false},
		{"cancelled excluded from denom", 1, 1, 1, 1, false},
		{"only cancelled", 0, 0, 0, 0, true},
		{"proposed excluded", 0, 0, 0, 0, true},
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
				denom := tt.todo + tt.inProgress + tt.blocked + tt.done
				if denom > 0 {
					expected := float64(tt.done) / float64(denom) * 100
					if *got != expected {
						t.Fatalf("CompletionPercent() = %v, want %v", *got, expected)
					}
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: ApprovalStateMachine
// ---------------------------------------------------------------------------------------------------------------------

func TestIsValidApprovalTransition(t *testing.T) {
	tests := []struct {
		from  string
		to    string
		valid bool
	}{
		// Valid from pending
		{"pending", "approved", true},
		{"pending", "rejected", true},
		{"pending", "requested_changes", true},
		{"pending", "waiting_on_response", true},
		// Invalid from pending
		{"pending", "pending", false},
		// Valid from requested_changes
		{"requested_changes", "pending", true},
		// Valid from waiting_on_response
		{"waiting_on_response", "pending", true},
		// Terminal states block all transitions
		{"approved", "pending", false},
		{"rejected", "pending", false},
		{"approved", "rejected", false},
	}

	for _, tt := range tests {
		t.Run(tt.from+"_"+tt.to, func(t *testing.T) {
			got := IsValidApprovalTransition(tt.from, tt.to)
			if got != tt.valid {
				t.Errorf("IsValidApprovalTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.valid)
			}
		})
	}
}

func TestApprovalRequiresComment(t *testing.T) {
	tests := []struct {
		outcome string
		required bool
	}{
		{"approve", false},
		{"reject", true},
		{"request_changes", true},
		{"need_more_information", true},
	}

	for _, tt := range tests {
		t.Run(tt.outcome, func(t *testing.T) {
			got := ApprovalRequiresComment(tt.outcome)
			if got != tt.required {
				t.Errorf("ApprovalRequiresComment(%q) = %v, want %v", tt.outcome, got, tt.required)
			}
		})
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: OwnerLabelMapper
// ---------------------------------------------------------------------------------------------------------------------

func TestMapOwnerLabel(t *testing.T) {
	productOverride := models.OwnerLabelEngineering

	tests := []struct {
		role     string
		override *models.OwnerLabelType
		want     models.OwnerLabelType
	}{
		// Override takes precedence
		{"engineering", &productOverride, models.OwnerLabelEngineering},
		{"some_role", &productOverride, models.OwnerLabelEngineering},
		// Default mappings
		{"engineering", nil, models.OwnerLabelEngineering},
		{"backend", nil, models.OwnerLabelEngineering},
		{"frontend", nil, models.OwnerLabelEngineering},
		{"developer", nil, models.OwnerLabelEngineering},
		{"product_manager", nil, models.OwnerLabelProduct},
		{"product_owner", nil, models.OwnerLabelProduct},
		{"pm", nil, models.OwnerLabelProduct},
		{"reviewer", nil, models.OwnerLabelReview},
		{"qa", nil, models.OwnerLabelReview},
		{"quality_assurance", nil, models.OwnerLabelReview},
		{"testing", nil, models.OwnerLabelReview},
		{"quality", nil, models.OwnerLabelQuality},
		{"architect", nil, models.OwnerLabelQuality},
		{"design", nil, models.OwnerLabelQuality},
		{"client", nil, models.OwnerLabelClient},
		{"client_stakeholder", nil, models.OwnerLabelClient},
		{"external", nil, models.OwnerLabelClient},
		// Whitespace trimming
		{"  engineering  ", nil, models.OwnerLabelEngineering},
		// Case insensitivity
		{"ENGINEERING", nil, models.OwnerLabelEngineering},
		{"Engineering", nil, models.OwnerLabelEngineering},
		// Unmapped falls back to Product
		{"unknown_role", nil, models.OwnerLabelProduct},
		{"", nil, models.OwnerLabelProduct},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			got := MapOwnerLabel(tt.role, tt.override)
			if got != tt.want {
				t.Errorf("MapOwnerLabel(%q, %v) = %v, want %v", tt.role, tt.override, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: SSEEventFilter
// ---------------------------------------------------------------------------------------------------------------------

func TestFilterSSEEvent(t *testing.T) {
	accessible := map[string]bool{"proj-a": true, "proj-b": true}

	tests := []struct {
		name       string
		event      SSEEventEnvelope
		wantNil    bool
		wantFields []string
	}{
		{
			name: "accessible project passes",
			event: SSEEventEnvelope{
				ProjectID: "proj-a",
				Payload:   map[string]interface{}{"title": "test", "actorId": "agent-123"},
			},
			wantNil:    false,
			wantFields: []string{"title"},
		},
		{
			name: "inaccessible project returns nil",
			event: SSEEventEnvelope{
				ProjectID: "proj-c",
				Payload:   map[string]interface{}{"title": "test"},
			},
			wantNil: true,
		},
		{
			name: "empty project id returns nil",
			event: SSEEventEnvelope{
				ProjectID: "",
				Payload:   map[string]interface{}{"title": "test"},
			},
			wantNil: true,
		},
		{
			name: "envelope fields stripped",
			event: SSEEventEnvelope{
				ProjectID: "proj-a",
				Payload: map[string]interface{}{
					"title":       "test",
					"actorId":     "agent-123",
					"actorRole":   "developer",
					"eventId":     "ev-1",
					"schemaVersion": "v1",
				},
			},
			wantNil:    false,
			wantFields: []string{"title"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterSSEEvent(tt.event, accessible)
			if tt.wantNil && got != nil {
				t.Errorf("FilterSSEEvent() = %v, want nil", got)
			}
			if !tt.wantNil && got == nil {
				t.Errorf("FilterSSEEvent() = nil, want non-nil")
			}
			if got != nil && len(tt.wantFields) > 0 {
				for _, f := range tt.wantFields {
					if _, ok := got[f]; !ok {
						t.Errorf("FilterSSEEvent() missing field %q", f)
					}
				}
				for _, ef := range EnvelopeFields {
					if _, ok := got[ef]; ok {
						t.Errorf("FilterSSEEvent() should not contain envelope field %q", ef)
					}
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: PublicationValidator
// ---------------------------------------------------------------------------------------------------------------------

func TestValidatePublication(t *testing.T) {
	tests := []struct {
		name        string
		input       PublicationValidationInput
		wantValid   bool
		wantReason  string
	}{
		{
			name: "valid publication",
			input: PublicationValidationInput{
				BusinessSummary:  "This is a valid summary",
				OwnerLabel:       "Product",
				NextAction:       "Review the proposal",
				VisibilityStatus: "published",
			},
			wantValid:  true,
			wantReason: "",
		},
		{
			name: "missing business_summary",
			input: PublicationValidationInput{
				BusinessSummary:  "",
				OwnerLabel:       "Product",
				NextAction:       "Review",
				VisibilityStatus: "published",
			},
			wantValid:  false,
			wantReason: "missing_field",
		},
		{
			name: "missing owner_label",
			input: PublicationValidationInput{
				BusinessSummary:  "Summary",
				OwnerLabel:       "",
				NextAction:       "Review",
				VisibilityStatus: "published",
			},
			wantValid:  false,
			wantReason: "missing_field",
		},
		{
			name: "invalid visibility_status",
			input: PublicationValidationInput{
				BusinessSummary:  "Summary",
				OwnerLabel:       "Product",
				NextAction:       "Review",
				VisibilityStatus: "invalid",
			},
			wantValid:  false,
			wantReason: "missing_field",
		},
		{
			name: "whitespace only fails",
			input: PublicationValidationInput{
				BusinessSummary:  "   ",
				OwnerLabel:       "Product",
				NextAction:       "Review",
				VisibilityStatus: "published",
			},
			wantValid:  false,
			wantReason: "missing_field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidatePublication(tt.input)
			if got.Valid != tt.wantValid {
				t.Errorf("ValidatePublication().Valid = %v, want %v", got.Valid, tt.wantValid)
			}
			if got.Reason != tt.wantReason {
				t.Errorf("ValidatePublication().Reason = %q, want %q", got.Reason, tt.wantReason)
			}
		})
	}
}

func TestValidatePublicationForbiddenContent(t *testing.T) {
	forbiddenInputs := []PublicationValidationInput{
		{BusinessSummary: "at com.example.MyClass.run(MyClass.java:42)", OwnerLabel: "Product", NextAction: "review", VisibilityStatus: "published"},
		{BusinessSummary: "Traceback (most recent call last)", OwnerLabel: "Product", NextAction: "review", VisibilityStatus: "published"},
		{BusinessSummary: "agent-a1b2c3d4e5f6 is processing", OwnerLabel: "Product", NextAction: "review", VisibilityStatus: "published"},
		{BusinessSummary: "Branch feature/my-feature deployed", OwnerLabel: "Product", NextAction: "review", VisibilityStatus: "published"},
		{BusinessSummary: "Commit 0f14c3b1e8d5a7f9c2e1d3b4a5c6e7f8a9b0c1d2 is ready", OwnerLabel: "Product", NextAction: "review", VisibilityStatus: "published"}, // 40-char hex
		{BusinessSummary: "Stack trace in /src/main.go:42", OwnerLabel: "Product", NextAction: "review", VisibilityStatus: "published"},
		{BusinessSummary: "Running on kubernetes cluster", OwnerLabel: "Product", NextAction: "review", VisibilityStatus: "published"},
		{BusinessSummary: "[DEBUG] Processing request", OwnerLabel: "Product", NextAction: "review", VisibilityStatus: "published"},
		{BusinessSummary: "goroutine deadlock detected", OwnerLabel: "Product", NextAction: "review", VisibilityStatus: "published"},
		{BusinessSummary: "Task assigned to layer_a agent", OwnerLabel: "Product", NextAction: "review", VisibilityStatus: "published"},
	}

	for i, input := range forbiddenInputs {
		t.Run("forbidden", func(t *testing.T) {
			got := ValidatePublication(input)
			if got.Valid {
				t.Errorf("ValidatePublication() test case %d: expected forbidden content to be detected", i)
			}
			if got.Reason != "forbidden_content" {
				t.Errorf("ValidatePublication() reason = %q, want forbidden_content", got.Reason)
			}
		})
	}
}

func TestValidatePublicationForbiddenMetadata(t *testing.T) {
	input := PublicationValidationInput{
		BusinessSummary:  "Valid summary",
		OwnerLabel:       "Product",
		NextAction:       "Review",
		VisibilityStatus: "published",
		Metadata:         map[string]string{"blockedReason": "some reason"},
	}
	got := ValidatePublication(input)
	if got.Valid {
		t.Errorf("ValidatePublication() with forbidden metadata key: expected validation to fail")
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: OverdueDecisionThreshold
// ---------------------------------------------------------------------------------------------------------------------

func TestCheckOverdue(t *testing.T) {
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		created  time.Time
		elapsed  time.Duration
		wantOverdue bool
	}{
		{"exactly 24h not overdue", base.Add(-24 * time.Hour), 0, false},
		{"24h + 1ms is overdue", base.Add(-24*time.Hour - 1), 0, true},
		{"1 hour not overdue", base.Add(-1 * time.Hour), 0, false},
		{"48h is overdue", base.Add(-48 * time.Hour), 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckOverdue(tt.created, base)
			if got.Overdue != tt.wantOverdue {
				t.Errorf("CheckOverdue() overdue = %v, want %v", got.Overdue, tt.wantOverdue)
			}
		})
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: CommentPrivacyManager
// ---------------------------------------------------------------------------------------------------------------------

func TestNewCommentAuditRecord(t *testing.T) {
	record := NewCommentAuditRecord("created", "actor-1", "John", "proj-1", "item-1", "comment-1", "task")
	if record.Action != "created" {
		t.Errorf("Action = %q, want created", record.Action)
	}
	if record.ProjectID != "proj-1" {
		t.Errorf("ProjectID = %q, want proj-1", record.ProjectID)
	}
	if record.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: XSSSanitizer
// ---------------------------------------------------------------------------------------------------------------------

func TestSanitizeXSS(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"script tag escaped", "<script>alert('xss')</script>", "&lt;script&gt;alert('xss')&lt;/script&gt;"},
		{"img onerror stripped", `<img src=x onerror=alert(1)>`, "&lt;img src=x =alert(1)&gt;"},
		{"plain text unchanged", "Hello, World!", "Hello, World!"},
		{"HTML entities escaped", "a < b & c > d", "a &lt; b &amp; c &gt; d"},
		{"svg tag escaped", "<svg onload=alert(1)>", "&lt;svg onload=alert(1)&gt;"},
		{"event handler stripped", `<div onclick="alert(1)">test</div>`, "&lt;div =\"alert(1)\"&gt;test&lt;/div&gt;"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeXSS(tt.input)
			// Check that script tags are escaped
			if strings.Contains(got, "<script>") {
				t.Errorf("SanitizeXSS() still contains <script>: %s", got)
			}
			// Check that < is escaped
			if strings.Contains(got, "<") && !strings.Contains(got, "&lt;") {
				t.Errorf("SanitizeXSS() should escape <: %s", got)
			}
		})
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: BFFProjectAccessFilter
// ---------------------------------------------------------------------------------------------------------------------

func TestFilterByAccess(t *testing.T) {
	accessible := map[string]bool{"proj-a": true, "proj-b": true}

	if !FilterByAccess("proj-a", accessible) {
		t.Errorf("FilterByAccess(proj-a) = false, want true")
	}
	if FilterByAccess("proj-c", accessible) {
		t.Errorf("FilterByAccess(proj-c) = true, want false")
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: EmptyStateRenderer
// ---------------------------------------------------------------------------------------------------------------------

func TestGetEmptyStateMessage(t *testing.T) {
	tests := []struct {
		context string
		want    string
	}{
		{"portfolio", "You don't have access to any projects yet. Contact your administrator to request access."},
		{"project_detail", "No active work yet"},
		{"task_board", "No active work yet"},
		{"approval_inbox", "No pending decisions"},
		{"risk_list", "No risks identified"},
		{"milestone_list", "No milestones defined"},
		{"search_results", "No results found. Try adjusting your filters."},
		{"unknown_context", "Nothing to display"},
	}

	for _, tt := range tests {
		t.Run(tt.context, func(t *testing.T) {
			got := GetEmptyStateMessage(tt.context)
			if got != tt.want {
				t.Errorf("GetEmptyStateMessage(%q) = %q, want %q", tt.context, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: NextActionResolver
// ---------------------------------------------------------------------------------------------------------------------

func TestResolveNextAction(t *testing.T) {
	tests := []struct {
		name   string
		actions []string
		want   string
	}{
		{"empty returns default", []string{}, "No action required"},
		{"single action", []string{"Review the document"}, "Review the document"},
		{"first non-empty", []string{"", "Second action"}, "Second action"},
		{"whitespace only returns default", []string{"   ", ""}, "No action required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveNextAction(tt.actions)
			if got != tt.want {
				t.Errorf("ResolveNextAction() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: CancelledTaskVisibility
// ---------------------------------------------------------------------------------------------------------------------

func TestFilterCancelledTasks(t *testing.T) {
	cancelled := models.ClientTaskCard{ID: "1", Status: "cancelled"}
	active := models.ClientTaskCard{ID: "2", Status: "todo"}
	tasks := []models.ClientTaskCard{cancelled, active}

	// hide cancelled
	filtered := FilterCancelledTasks(tasks, false)
	if len(filtered) != 1 {
		t.Errorf("FilterCancelledTasks(hide) len = %d, want 1", len(filtered))
	}
	if filtered[0].Status == "cancelled" {
		t.Errorf("FilterCancelledTasks(hide) still contains cancelled task")
	}

	// show cancelled
	filtered = FilterCancelledTasks(tasks, true)
	if len(filtered) != 2 {
		t.Errorf("FilterCancelledTasks(show) len = %d, want 2", len(filtered))
	}
}

func TestGetCancellationReason(t *testing.T) {
	if GetCancellationReason("") != "This task was cancelled" {
		t.Errorf("GetCancellationReason(empty) = %q, want default message", GetCancellationReason(""))
	}
	if GetCancellationReason("Budget cut") != "Budget cut" {
		t.Errorf("GetCancellationReason(custom) = %q, want custom message", GetCancellationReason("Budget cut"))
	}
}