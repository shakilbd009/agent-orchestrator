package util

import (
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/agent-orchestrator/backend/internal/models"
)

// ---------------------------------------------------------------------------------------------------------------------
// Contract: CompletionPercentageCalculator
// Formula (ADR-03-001): done / (todo + in_progress + blocked + done)
// Excludes: cancelled, proposed
// ---------------------------------------------------------------------------------------------------------------------

// CompletionPercent calculates client-facing completion percentage.
// Returns nil when denominator is zero (no active tasks).
func CompletionPercent(todo, inProgress, blocked, done int) *float64 {
	denom := todo + inProgress + blocked + done
	if denom == 0 {
		return nil
	}
	pct := float64(done) / float64(denom) * 100
	return &pct
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: ApprovalStateMachine
// ---------------------------------------------------------------------------------------------------------------------

// ValidApprovalTransitions defines allowed state transitions.
var ValidApprovalTransitions = map[string]map[string]bool{
	"pending": {
		"approved":           true,
		"rejected":           true,
		"requested_changes":  true,
		"waiting_on_response": true,
	},
	"requested_changes": {
		"pending": true, // internal owner republishes
	},
	"waiting_on_response": {
		"pending": true, // client provides requested information
	},
}

// TerminalStates lists states that cannot transition further.
var TerminalStates = map[string]bool{
	"approved": true,
	"rejected": true,
}

// IsValidTransition returns true if the transition is allowed.
func IsValidApprovalTransition(from, to string) bool {
	if TerminalStates[from] {
		return false
	}
	if nextStates, ok := ValidApprovalTransitions[from]; ok {
		return nextStates[to]
	}
	return false
}

// ApprovalRequiresComment returns true if the outcome requires a comment.
func ApprovalRequiresComment(outcome string) bool {
	return outcome == "reject" || outcome == "request_changes" || outcome == "need_more_information"
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: OwnerLabelMapper
// ---------------------------------------------------------------------------------------------------------------------

var defaultOwnerLabelMap = map[string]models.OwnerLabelType{
	"product_manager":     models.OwnerLabelProduct,
	"product_owner":        models.OwnerLabelProduct,
	"pm":                   models.OwnerLabelProduct,
	"engineering":         models.OwnerLabelEngineering,
	"backend":             models.OwnerLabelEngineering,
	"frontend":            models.OwnerLabelEngineering,
	"developer":           models.OwnerLabelEngineering,
	"reviewer":            models.OwnerLabelReview,
	"qa":                  models.OwnerLabelReview,
	"quality_assurance":   models.OwnerLabelReview,
	"testing":             models.OwnerLabelReview,
	"quality":             models.OwnerLabelQuality,
	"architect":           models.OwnerLabelQuality,
	"design":              models.OwnerLabelQuality,
	"client":              models.OwnerLabelClient,
	"client_stakeholder":  models.OwnerLabelClient,
	"external":            models.OwnerLabelClient,
}

// MapOwnerLabel returns the client-facing label for an internal role.
// Override takes precedence over default mapping.
func MapOwnerLabel(internalRole string, override *models.OwnerLabelType) models.OwnerLabelType {
	if override != nil && *override != "" {
		return *override
	}
	trimmed := strings.TrimSpace(strings.ToLower(internalRole))
	if label, ok := defaultOwnerLabelMap[trimmed]; ok {
		return label
	}
	return models.OwnerLabelProduct // fallback per ADR-03-005
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: SSEEventFilter
// ---------------------------------------------------------------------------------------------------------------------

// SSEEventEnvelope represents the BRD-02 SSE event envelope.
type SSEEventEnvelope struct {
	ProjectID     string                 `json:"project_id"`
	EventType     string                 `json:"event_type"`
	ItemID        string                 `json:"item_id"`
	Timestamp     string                 `json:"timestamp"`
	Payload       map[string]interface{} `json:"payload"`
	ActorID       string                 `json:"actorId"`
	ActorRole     string                 `json:"actorRole"`
	EventID       string                 `json:"eventId"`
	SchemaVersion string                 `json:"schemaVersion"`
	ParentTaskID  string                 `json:"parentTaskId"`
	GateID        string                 `json:"gateId"`
	Layer         string                 `json:"layer"`
}

// EnvelopeFields lists metadata fields that must be stripped before client delivery.
var EnvelopeFields = []string{"actorId", "actorRole", "eventId", "schemaVersion", "parentTaskId", "gateId", "layer"}

// FilterSSEEvent returns the event payload with envelope metadata stripped.
// Returns nil if projectID is not in the accessible set.
func FilterSSEEvent(event SSEEventEnvelope, accessibleProjects map[string]bool) map[string]interface{} {
	if event.ProjectID == "" {
		return nil
	}
	if !accessibleProjects[event.ProjectID] {
		return nil
	}
	// Strip envelope metadata from payload
	filtered := make(map[string]interface{})
	for k, v := range event.Payload {
		if isEnvelopeField(k) {
			continue
		}
		filtered[k] = v
	}
	return filtered
}

func isEnvelopeField(field string) bool {
	for _, ef := range EnvelopeFields {
		if field == ef {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: PublicationValidator
// ---------------------------------------------------------------------------------------------------------------------

// ForbiddenPatterns contains all patterns that fail publication validation.
var ForbiddenPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?im)^\s*at\s+`),                                  // stack trace lines (multiline mode)
	regexp.MustCompile(`(?i)(Traceback|Exception|Error:|panic:)`),        // traceback/panic
	regexp.MustCompile(`(?i)agent-[0-9a-f]{6,}`),                         // agent IDs
	regexp.MustCompile(`(?i)(refs/heads/|fix/|hotfix/|release/)`),        // branch names (excludes feature/ which is too broad)
	regexp.MustCompile(`(?i)feature/`),                                    // feature branch prefix
	regexp.MustCompile(`[a-f0-9]{40}`),                                     // commit SHAs
	regexp.MustCompile(`(?i)(/src/|/internal/|/backend/|/agent/|/pkg/|/cmd/)`), // file paths
	regexp.MustCompile(`(?i)(docker|kubernetes|pod|deployment|service mesh|container|namespace|kubeconfig|helm|ingress|sidecar)`), // infra terms
	regexp.MustCompile(`^\s*\[(DEBUG|INFO|WARN|ERROR|TRACE)\]`),           // log lines
	regexp.MustCompile(`(?i)(panic:|goroutine|runtime\.gopanic)`),         // go runtime
	regexp.MustCompile(`(?i)\blayer_a\b`),                                  // internal role values with word boundaries
	regexp.MustCompile(`(?i)\blayer_b\b`),
}

// ForbiddenMetadataKeys lists metadata keys that cause validation failure.
var ForbiddenMetadataKeys = []string{"blockedReason", "owner_override", "internal_tags", "execution_agent"}

// PublicationValidationInput holds fields required for validation.
type PublicationValidationInput struct {
	BusinessSummary  string
	OwnerLabel       string
	NextAction       string
	VisibilityStatus string
	Body             string
	BlockedReason    string
	Summary          string
	Metadata         map[string]string
}

// PublicationValidationResult holds the outcome of validation.
type PublicationValidationResult struct {
	Valid          bool
	Reason         string // "missing_field" | "forbidden_content" | ""
	Pattern        string
	MissingFields  []string
}

// ValidatePublication checks required fields and forbidden content.
func ValidatePublication(in PublicationValidationInput) PublicationValidationResult {
	// Check required fields
	var missing []string
	if strings.TrimSpace(in.BusinessSummary) == "" {
		missing = append(missing, "business_summary")
	}
	if strings.TrimSpace(in.OwnerLabel) == "" {
		missing = append(missing, "owner_label")
	}
	if strings.TrimSpace(in.NextAction) == "" {
		missing = append(missing, "next_action")
	}
	vis := strings.TrimSpace(in.VisibilityStatus)
	if vis != "published" && vis != "unpublished" {
		missing = append(missing, "visibility_status")
	}
	if len(missing) > 0 {
		return PublicationValidationResult{
			Valid:         false,
			Reason:        "missing_field",
			MissingFields: missing,
		}
	}

	// Check all text fields for forbidden content
	allText := joinNonEmpty(in.BusinessSummary, in.Body, in.BlockedReason, in.Summary, in.NextAction)
	for _, pattern := range ForbiddenPatterns {
		if pattern.MatchString(allText) {
			return PublicationValidationResult{
				Valid:   false,
				Reason:  "forbidden_content",
				Pattern: pattern.String(),
			}
		}
	}

	// Check metadata keys
	for k := range in.Metadata {
		for _, forbidden := range ForbiddenMetadataKeys {
			if k == forbidden {
				return PublicationValidationResult{
					Valid:   false,
					Reason:  "forbidden_content",
					Pattern: "forbidden_metadata_key:" + forbidden,
				}
			}
		}
	}

	return PublicationValidationResult{Valid: true}
}

func joinNonEmpty(parts ...string) string {
	var b strings.Builder
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			b.WriteString(trimmed)
			b.WriteString(" ")
		}
	}
	return b.String()
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: OverdueDecisionThreshold
// ---------------------------------------------------------------------------------------------------------------------

const overdueThresholdHours = 24

// OverdueResult holds the overdue check result.
type OverdueResult struct {
	Overdue  bool
	AgeHours float64
}

// CheckOverdue returns whether an approval is overdue (>24h strictly).
func CheckOverdue(createdAt time.Time, now time.Time) OverdueResult {
	elapsed := now.Sub(createdAt)
	ageHours := elapsed.Hours()
	return OverdueResult{
		Overdue:  ageHours > overdueThresholdHours,
		AgeHours: ageHours,
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: CommentPrivacyManager
// ---------------------------------------------------------------------------------------------------------------------

// CommentAuditRecord represents the audit record for a comment action.
// Body is NEVER stored — per FR-03-029.
type CommentAuditRecord struct {
	Action     string    `json:"action"` // created, edited, deleted
	ActorID    string    `json:"actorId"`
	ActorName  string    `json:"actorName"`
	ProjectID  string    `json:"projectId"`
	ItemID     string    `json:"itemId"`
	CommentID  string    `json:"commentId"`
	ItemType   string    `json:"itemType"` // task, risk, milestone, approval
	Timestamp  time.Time `json:"timestamp"`
}

// NewCommentAuditRecord creates an audit record without storing body.
func NewCommentAuditRecord(action, actorID, actorName, projectID, itemID, commentID, itemType string) CommentAuditRecord {
	return CommentAuditRecord{
		Action:    action,
		ActorID:   actorID,
		ActorName: actorName,
		ProjectID: projectID,
		ItemID:    itemID,
		CommentID: commentID,
		ItemType:  itemType,
		Timestamp: time.Now(),
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: XSSSanitizer
// ---------------------------------------------------------------------------------------------------------------------

// dangerousTags matches HTML tags that can execute code.
var dangerousTags = regexp.MustCompile(`(?i)<(script|img|svg|math|onerror|onload|onclick)\b`)
var dangerousAttr = regexp.MustCompile(`(?i)\s(onerror|onload|onclick|onmouse\w+)\s*=`)
var ampReplacer = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&#39;")

// SanitizeXSS neutralizes XSS vectors in user-provided strings.
func SanitizeXSS(input string) string {
	// First escape HTML entities
	sanitized := ampReplacer.Replace(input)
	// Remove event handler attributes
	sanitized = dangerousAttr.ReplaceAllString(sanitized, " ")
	// Remove script and dangerous tags
	sanitized = dangerousTags.ReplaceAllString(sanitized, "&lt;$1")
	return sanitized
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: BFFProjectAccessFilter
// ---------------------------------------------------------------------------------------------------------------------

// FilterByAccess returns the resource if accessible, otherwise nil.
// Per ADR-03-001: returns empty result NOT 403/404.
func FilterByAccess(resourceProjectID string, accessibleProjects map[string]bool) bool {
	return accessibleProjects[resourceProjectID]
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: EmptyStateRenderer
// ---------------------------------------------------------------------------------------------------------------------

var emptyStateMessages = map[string]string{
	"portfolio":        "You don't have access to any projects yet. Contact your administrator to request access.",
	"project_detail":   "No active work yet",
	"task_board":       "No active work yet",
	"approval_inbox":  "No pending decisions",
	"risk_list":       "No risks identified",
	"milestone_list":  "No milestones defined",
	"search_results":  "No results found. Try adjusting your filters.",
}

// GetEmptyStateMessage returns the appropriate empty state message.
func GetEmptyStateMessage(context string) string {
	if msg, ok := emptyStateMessages[context]; ok {
		return msg
	}
	return "Nothing to display"
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: NextActionResolver
// ---------------------------------------------------------------------------------------------------------------------

var noActionRequired = []string{
	"No action required",
	"Waiting for internal team",
	"Under review",
}

// ResolveNextAction returns the single current client-facing next action.
// Returns "No action required" if no action is defined.
func ResolveNextAction(nextActions []string) string {
	if len(nextActions) == 0 {
		return "No action required"
	}
	// Return the first non-empty, client-safe action
	for _, a := range nextActions {
		trimmed := strings.TrimSpace(a)
		if trimmed != "" {
			return trimmed
		}
	}
	return "No action required"
}

// ---------------------------------------------------------------------------------------------------------------------
// Contract: CancelledTaskVisibility
// ---------------------------------------------------------------------------------------------------------------------

// FilterCancelledTasks hides cancelled tasks when showCancelled is false.
func FilterCancelledTasks(tasks []models.ClientTaskCard, showCancelled bool) []models.ClientTaskCard {
	if !showCancelled {
		var filtered []models.ClientTaskCard
		for _, t := range tasks {
			if t.Status != "cancelled" {
				filtered = append(filtered, t)
			}
		}
		return filtered
	}
	return tasks
}

// GetCancellationReason returns a plain-language cancellation reason.
func GetCancellationReason(reason string) string {
	if strings.TrimSpace(reason) == "" {
		return "This task was cancelled"
	}
	return reason
}

// ---------------------------------------------------------------------------------------------------------------------
// Helper: isAlphanumeric checks if a rune is a letter or digit.
// ---------------------------------------------------------------------------------------------------------------------

func isAlphanumeric(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsDigit(c)
}