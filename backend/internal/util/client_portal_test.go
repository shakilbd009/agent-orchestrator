package util

import (
	"testing"
	"time"
)

func TestIsForbiddenContent_Empty(t *testing.T) {
	if IsForbiddenContent("") {
		t.Error("empty string should not be forbidden")
	}
}

func TestIsForbiddenContent_StackTrace(t *testing.T) {
	if !IsForbiddenContent("at com.auth.AuthService.verifyToken(AuthService.java:42)") {
		t.Error("stack trace line should be forbidden")
	}
}

func TestIsForbiddenContent_BranchName(t *testing.T) {
	if !IsForbiddenContent("feature/user-auth-overhaul") {
		t.Error("feature branch name should be forbidden")
	}
	if !IsForbiddenContent("bugfix/payment-race-condition") {
		t.Error("bugfix branch name should be forbidden")
	}
	if !IsForbiddenContent("/compare/feature-auth-overhaul") {
		t.Error("branch in URL path should be forbidden")
	}
}

func TestIsForbiddenContent_GitSHA(t *testing.T) {
	// Real 40-char hex SHA (proper git SHA)
	sha := "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3" // 40-char
	if !IsForbiddenContent(sha) {
		t.Error("40-char hex SHA should be forbidden")
	}
	// With surrounding context
	if !IsForbiddenContent("commit "+sha+" changed the file") {
		t.Error("40-char hex SHA in context should be forbidden")
	}
}

func TestIsForbiddenContent_AgentID(t *testing.T) {
	if !IsForbiddenContent("agent-abc12345") {
		t.Error("agent ID should be forbidden")
	}
	if !IsForbiddenContent("agent_9f3a2b1c") {
		t.Error("agent ID with underscore should be forbidden")
	}
	if !IsForbiddenContent("assigned to agent-9f3a2b1c for processing") {
		t.Error("agent ID in sentence should be forbidden")
	}
}

func TestIsForbiddenContent_FilePath(t *testing.T) {
	if !IsForbiddenContent("/Users/shakilakram/projects/agent-orchestrator/backend/main.go") {
		t.Error("unix file path should be forbidden")
	}
	if !IsForbiddenContent("C:\\Users\\shakil\\Projects\\agent-orchestrator\\backend\\main.go") {
		t.Error("windows file path should be forbidden")
	}
}

func TestIsForbiddenContent_InfraTerms(t *testing.T) {
	if !IsForbiddenContent("kubernetes cluster eks-prod") {
		t.Error("kubernetes should be forbidden")
	}
	if !IsForbiddenContent("arn:aws:iam::123456789012:role/Admin") {
		t.Error("AWS ARN should be forbidden")
	}
	if !IsForbiddenContent("vpc-0abc123def0abc123") {
		t.Error("VPC ID should be forbidden")
	}
	if !IsForbiddenContent("s3://my-bucket-name") {
		t.Error("S3 URI should be forbidden")
	}
}

func TestIsForbiddenContent_EnvelopeMetadata(t *testing.T) {
	if !IsForbiddenContent(`"actorId":"user-123","actorRole":"human","eventId":"ev-456"`) {
		t.Error("envelope metadata fields should be forbidden")
	}
}

func TestIsForbiddenContent_CleanText(t *testing.T) {
	clean := []string{
		"Project on track, next milestone is Q3 launch",
		"Waiting for client approval on feature design",
		"Review completed, all tests passing",
		"Task completed ahead of schedule",
	}
	for _, s := range clean {
		if IsForbiddenContent(s) {
			t.Errorf("clean text should not be forbidden: %q", s)
		}
	}
}

func TestStripForbiddenContent_StackTrace(t *testing.T) {
	input := `Authentication failed
at com.auth.AuthService.verifyToken(AuthService.java:42)
at com.auth.AuthHandler.handleRequest(AuthHandler.java:15)
Please review`
	stripped, changed := StripForbiddenContent(input)
	if !changed {
		t.Error("expected content to be changed")
	}
	if stripped == input {
		t.Error("stack trace lines should be removed")
	}
	if len(stripped) >= len(input) {
		t.Error("stripped content should be shorter")
	}
}

func TestStripForbiddenContent_AgentID(t *testing.T) {
	input := "Task assigned to agent-abc12345 for processing"
	stripped, changed := StripForbiddenContent(input)
	if !changed {
		t.Error("expected content to be changed")
	}
	if stripped == input {
		t.Error("expected agent ID to be stripped")
	}
}

func TestValidatePublication_ValidInput(t *testing.T) {
	result := ValidatePublication(PublicationValidationInput{
		Title:   "Q3 Project Status",
		Body:    "All milestones on track. Client approval needed for next phase.",
		OwnerID: "product-manager",
		Tags:    []string{"status", "client"},
	})
	if !result.Passed {
		t.Errorf("expected passed=true, got false: %v", result.FailureReason)
	}
}

func TestValidatePublication_EmptyTitle(t *testing.T) {
	result := ValidatePublication(PublicationValidationInput{
		Title: "   ",
		Body:  "Some content",
	})
	if result.Passed {
		t.Error("expected passed=false for empty title")
	}
	if result.FailureReason == nil || *result.FailureReason != "title is required" {
		t.Errorf("expected 'title is required', got %v", result.FailureReason)
	}
}

func TestValidatePublication_EmptyBody(t *testing.T) {
	result := ValidatePublication(PublicationValidationInput{
		Title: "Valid Title",
		Body:  "   ",
	})
	if result.Passed {
		t.Error("expected passed=false for empty body")
	}
	if result.FailureReason == nil || *result.FailureReason != "body is required" {
		t.Errorf("expected 'body is required', got %v", result.FailureReason)
	}
}

func TestValidatePublication_ForbiddenInBody(t *testing.T) {
	result := ValidatePublication(PublicationValidationInput{
		Title: "Deployment Note",
		Body:  "Deployed to kubernetes via helm chart on eks-prod-cluster",
	})
	if result.Passed {
		t.Error("expected passed=false for forbidden content in body")
	}
	if result.FailureReason == nil || *result.FailureReason != "content contains forbidden patterns" {
		t.Errorf("expected 'content contains forbidden patterns', got %v", result.FailureReason)
	}
	if len(result.ForbiddenTerms) == 0 {
		t.Error("expected ForbiddenTerms to be populated for server-side logging")
	}
}

func TestValidatePublication_ForbiddenInTitle(t *testing.T) {
	// Title contains a file path which is a forbidden pattern
	result := ValidatePublication(PublicationValidationInput{
		Title: "Fix in /Users/developer/project/backend/main.go",
		Body:  "Regular update about the fix",
	})
	if result.Passed {
		t.Error("expected passed=false for forbidden content in title")
	}
}

func TestValidatePublication_NoForbiddenTermsInClientResponse(t *testing.T) {
	result := ValidatePublication(PublicationValidationInput{
		Title: "Test",
		Body:  "kubernetes docker terraform eks-prod s3://bucket",
	})
	if result.Passed {
		t.Fatal("expected validation to fail")
	}
	// failureReason must NOT contain raw forbidden terms
	if result.FailureReason != nil {
		for _, term := range []string{"kubernetes", "docker", "eks", "s3://"} {
			found := contains(*result.FailureReason, term)
			if found {
				t.Errorf("failureReason must not contain raw forbidden term %q: %s", term, *result.FailureReason)
			}
		}
	}
}

func TestMapOwnerLabel(t *testing.T) {
	cases := []struct {
		internal string
		expected string
	}{
		{"product", "Product"},
		{"product_manager", "Product"},
		{"pm", "Product"},
		{"engineering", "Engineering"},
		{"dev", "Engineering"},
		{"backend", "Engineering"},
		{"frontend", "Engineering"},
		{"review", "Review"},
		{"reviewer", "Review"},
		{"qa", "Review"},
		{"quality", "Quality"},
		{"tester", "Quality"},
		{"client", "Client"},
		{"customer", "Client"},
		{"unknown_role", "Product"},
	}
	for _, c := range cases {
		got := MapOwnerLabel(c.internal, "")
		if got != c.expected {
			t.Errorf("MapOwnerLabel(%q, ''): expected %q, got %q", c.internal, c.expected, got)
		}
	}
}

func TestMapOwnerLabel_WithOverride(t *testing.T) {
	got := MapOwnerLabel("engineering", "Custom Engineering Label")
	if got != "Custom Engineering Label" {
		t.Errorf("expected override label, got %q", got)
	}
}

func TestIsDecisionOverdue_Zero(t *testing.T) {
	if IsDecisionOverdue(0) {
		t.Error("zero timestamp should not be overdue")
	}
}

func TestIsDecisionOverdue_Recent(t *testing.T) {
	createdAt := time.Now().Add(-12 * time.Hour).Unix()
	if IsDecisionOverdue(createdAt) {
		t.Error("12h old decision should NOT be overdue")
	}
}

func TestIsDecisionOverdue_Old(t *testing.T) {
	createdAt := time.Now().Add(-30 * time.Hour).Unix()
	if !IsDecisionOverdue(createdAt) {
		t.Error("30h old decision SHOULD be overdue")
	}
}

func TestCalculateCompletionPercent_Normal(t *testing.T) {
	counts := TaskStatusCounts{
		Todo:       10,
		InProgress: 5,
		Blocked:    2,
		Done:       3,
	}
	pct := CalculateCompletionPercent(counts)
	if pct == nil {
		t.Fatal("expected non-nil")
	}
	if *pct != 15.0 {
		t.Errorf("expected 15.0, got %f", *pct)
	}
}

func TestCalculateCompletionPercent_ZeroDenom(t *testing.T) {
	counts := TaskStatusCounts{
		Cancelled: 5,
		Proposed:  3,
	}
	pct := CalculateCompletionPercent(counts)
	if pct != nil {
		t.Error("expected nil for zero denominator (no active tasks)")
	}
}

func TestCalculateCompletionPercent_AllDone(t *testing.T) {
	counts := TaskStatusCounts{
		Done: 10,
	}
	pct := CalculateCompletionPercent(counts)
	if pct == nil {
		t.Fatal("expected non-nil")
	}
	if *pct != 100.0 {
		t.Errorf("expected 100.0, got %f", *pct)
	}
}

func TestCalculateCompletionPercent_ExcludesCancelledProposed(t *testing.T) {
	counts := TaskStatusCounts{
		Todo:       5,
		InProgress: 5,
		Blocked:    5,
		Done:       5,
		Cancelled:  100,
		Proposed:   100,
	}
	pct := CalculateCompletionPercent(counts)
	if pct == nil {
		t.Fatal("expected non-nil")
	}
	if *pct != 25.0 {
		t.Errorf("expected 25.0, got %f", *pct)
	}
}

// --- helpers ---

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}