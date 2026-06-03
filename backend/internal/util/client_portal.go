package util

import (
	"regexp"
	"strings"
	"time"

	"github.com/agent-orchestrator/backend/internal/models"
)

// ---------------------------------------------------------------------------------------------------------------------
// Forbidden Content Stripper (ADR-03-003)
// ---------------------------------------------------------------------------------------------------------------------

// forbiddenPatterns collects all regex patterns for forbidden content detection.
var forbiddenPatterns = []*regexp.Regexp{
	// Stack trace lines: "at package.Class.method(File:line)"
	regexp.MustCompile(`(?m)^at\s+[\w.$]+\([\w./\\]+\:\d+\).*$`),
	// Agent IDs: agent_<hex> or agent-<hex> (1-8 hex chars)
	regexp.MustCompile(`agent[-_][a-f0-9]{1,8}`),
	// Git branch names (prefixed by / or at word boundary, followed by /)
	regexp.MustCompile(`(?:^|[/\s])feature/[a-zA-Z0-9_/-]+`),
	regexp.MustCompile(`(?:^|[/\s])bugfix/[a-zA-Z0-9_/-]+`),
	regexp.MustCompile(`(?:^|[/\s])hotfix/[a-zA-Z0-9_/-]+`),
	regexp.MustCompile(`(?:^|[/\s])refactor/[a-zA-Z0-9_/-]+`),
	regexp.MustCompile(`(?:^|[/\s])chore/[a-zA-Z0-9_/-]+`),
	regexp.MustCompile(`(?:^|[/\s])docs/[a-zA-Z0-9_/-]+`),
	regexp.MustCompile(`(?:^|[/\s])test/[a-zA-Z0-9_/-]+`),
	// Branch names embedded in compare URLs: /compare/feature-...
	regexp.MustCompile(`/compare/feature[-][a-zA-Z0-9_/-]+`),
	// 40-char hex SHAs (git commit IDs)
	regexp.MustCompile(`[a-f0-9]{40}`),
	// Unix paths with /Users/ or /home/ or /var/ or /tmp/
	regexp.MustCompile(`/Users/[\w./-]+`),
	regexp.MustCompile(`/home/[\w./-]+`),
	regexp.MustCompile(`/var/[\w./-]+`),
	regexp.MustCompile(`/tmp/[\w./-]+`),
	// Windows paths
	regexp.MustCompile(`[A-Za-z]:\\[\w\\.-]+`),
	// Internal package paths
	regexp.MustCompile(`(?:backend|frontend|internal|service|handler|repository|models)/(?:[\w/]+)`),
	// Infrastructure terms
	regexp.MustCompile(`\bkubernetes\b`),
	regexp.MustCompile(`\bdocker\b`),
	regexp.MustCompile(`\bhelm\b`),
	regexp.MustCompile(`\bterraform\b`),
	regexp.MustCompile(`arn:aws:[\w:]+`),
	regexp.MustCompile(`vpc-[a-f0-9]{17}`),
	regexp.MustCompile(`s3://[\w.-]+`),
	regexp.MustCompile(`\beks-\w+`),
	regexp.MustCompile(`\becs-\w+`),
	regexp.MustCompile(`\bkube-[a-z0-9-]+`),
	// Log lines with ERROR/WARN/INFO/DEBUG prefixes
	regexp.MustCompile(`(?m)^(?:ERROR|WARN|INFO|DEBUG)\s+[\w:]+:\s+`),
	// JSON object with internal metadata fields
	regexp.MustCompile(`"actorId"\s*:\s*"[^"]+"`),
	regexp.MustCompile(`"actorRole"\s*:\s*"[^"]+"`),
	regexp.MustCompile(`"eventId"\s*:\s*"[^"]+"`),
	regexp.MustCompile(`"schemaVersion"\s*:\s*"[^"]+"`),
	regexp.MustCompile(`"parentTaskId"\s*:\s*"[^"]+"`),
	regexp.MustCompile(`"gateId"\s*:\s*"[^"]+"`),
}

// IsForbiddenContent returns true if the string contains any forbidden patterns.
func IsForbiddenContent(s string) bool {
	for _, pat := range forbiddenPatterns {
		if pat.MatchString(s) {
			return true
		}
	}
	return false
}

// StripForbiddenContent removes all forbidden patterns from the input string.
// Returns the stripped string and true if any content was removed.
func StripForbiddenContent(s string) (string, bool) {
	original := s
	for _, pat := range forbiddenPatterns {
		s = pat.ReplaceAllString(s, "[redacted]")
	}
	return s, s != original
}

// ---------------------------------------------------------------------------------------------------------------------
// Publication Validator (ADR-03-003)
// ---------------------------------------------------------------------------------------------------------------------

// PublicationValidationInput is the input for ValidatePublication.
type PublicationValidationInput struct {
	Title   string
	Body    string
	OwnerID string
	Tags    []string
}

// ValidatePublication validates a publication against required fields and forbidden patterns.
// Returns a result where failureReason does NOT include raw forbidden content (per ADR-03-003).
func ValidatePublication(in PublicationValidationInput) *models.PublicationValidationResult {
	// Check required fields
	if strings.TrimSpace(in.Title) == "" {
		return &models.PublicationValidationResult{
			Passed:        false,
			FailureReason: ptrStr("title is required"),
		}
	}
	if strings.TrimSpace(in.Body) == "" {
		return &models.PublicationValidationResult{
			Passed:        false,
			FailureReason: ptrStr("body is required"),
		}
	}

	// Check body for forbidden content
	bodyForbidden := IsForbiddenContent(in.Body)
	titleForbidden := IsForbiddenContent(in.Title)

	if bodyForbidden || titleForbidden {
		// Collect matched terms for server-side logging only (never sent to client)
		var terms []string
		if bodyForbidden {
			terms = append(terms, collectForbiddenTerms(in.Body)...)
		}
		if titleForbidden {
			terms = append(terms, collectForbiddenTerms(in.Title)...)
		}

		return &models.PublicationValidationResult{
			Passed:         false,
			FailureReason:  ptrStr("content contains forbidden patterns"),
			ForbiddenTerms: deduplicateTerms(terms),
		}
	}

	return &models.PublicationValidationResult{Passed: true}
}

func collectForbiddenTerms(s string) []string {
	var terms []string
	for _, pat := range forbiddenPatterns {
		matches := pat.FindAllString(s, -1)
		for _, m := range matches {
			terms = append(terms, m)
		}
	}
	return terms
}

func deduplicateTerms(terms []string) []string {
	seen := make(map[string]bool)
	var unique []string
	for _, t := range terms {
		if !seen[t] {
			seen[t] = true
			unique = append(unique, t)
		}
	}
	return unique
}

// ---------------------------------------------------------------------------------------------------------------------
// Owner Label Mapper (ADR-03-005)
// ---------------------------------------------------------------------------------------------------------------------

// MapOwnerLabel returns the client-facing label for an internal owner.
// Override precedence (ADR-03-005): client decision override > project-level override > internal mapping.
func MapOwnerLabel(internalRole, overrideLabel string) string {
	if overrideLabel != "" {
		return overrideLabel
	}
	switch internalRole {
	case "product", "product_manager", "pm":
		return string(models.OwnerLabelProduct)
	case "engineering", "dev", "backend", "frontend":
		return string(models.OwnerLabelEngineering)
	case "review", "reviewer", "qa":
		return string(models.OwnerLabelReview)
	case "quality", "tester":
		return string(models.OwnerLabelQuality)
	case "client", "customer":
		return string(models.OwnerLabelClient)
	default:
		return string(models.OwnerLabelProduct)
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// Decision Overdue Checker (ADR-03-004)
// ---------------------------------------------------------------------------------------------------------------------

// IsDecisionOverdue returns true if the decision is overdue (>24h since creation).
func IsDecisionOverdue(createdAt int64) bool {
	if createdAt == 0 {
		return false
	}
	age := time.Since(time.Unix(createdAt, 0))
	return age > time.Duration(models.OverdueThresholdHours)*time.Hour
}

// ---------------------------------------------------------------------------------------------------------------------
// Completion Percentage Calculator (ADR-03-001)
// ---------------------------------------------------------------------------------------------------------------------

// TaskStatusCounts holds task counts by status for completion calculation.
type TaskStatusCounts struct {
	Todo       int
	InProgress int
	Blocked    int
	Done       int
	Cancelled  int
	Proposed   int
}

// CalculateCompletionPercent computes done/(todo+in_progress+blocked+done) * 100.
// Returns nil if denom is zero (no active tasks).
// Excludes cancelled and proposed per ADR-03-001.
func CalculateCompletionPercent(counts TaskStatusCounts) *float64 {
	denom := counts.Todo + counts.InProgress + counts.Blocked + counts.Done
	if denom == 0 {
		return nil
	}
	pct := float64(counts.Done) / float64(denom) * 100
	return &pct
}

// ---------------------------------------------------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------------------------------------------------

func ptrStr(s string) *string { return &s }