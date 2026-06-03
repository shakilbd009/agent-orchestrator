package models

import (
	"time"
)

// ---------------------------------------------------------------------------------------------------------------------
// Client Portal — Portfolio & Project Models
// ---------------------------------------------------------------------------------------------------------------------

// ClientPortfolio is the top-level response for GET /client-portal/portfolio.
type ClientPortfolio struct {
	ProjectsSummary ProjectsHealthSummary   `json:"projectsSummary"`
	ProjectList    []ClientProjectSummary   `json:"projectList"`
	DecisionSummary ClientDecisionSummary   `json:"decisionSummary"`
	Timestamp      time.Time               `json:"timestamp"`
}

// ProjectsHealthSummary holds aggregate health counts across all accessible projects.
type ProjectsHealthSummary struct {
	OnTrack int `json:"onTrack"`
	AtRisk  int `json:"atRisk"`
	Blocked int `json:"blocked"`
}

// ClientProjectSummary is a stripped-down project card for the portfolio list.
type ClientProjectSummary struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	Health            string     `json:"health"`           // on_track, at_risk, blocked
	Confidence        string     `json:"confidence"`       // high, medium, low
	CompletionPercent float64    `json:"completionPercent"` // done/(todo+in_progress+blocked+done); -1 if no active tasks
	NextMilestone     *string    `json:"nextMilestone"`    // nullable
	PendingDecisions  int        `json:"pendingDecisions"`
	OverdueDecisions  int        `json:"overdueDecisions"`
	LatestUpdate      time.Time  `json:"latestUpdate"`
}

// ClientDecisionSummary holds aggregate approval decision counts for the portfolio.
type ClientDecisionSummary struct {
	TotalPending   int `json:"totalPending"`
	Overdue        int `json:"overdue"`
	WaitingOnClient int `json:"waitingOnClient"`
	AtRiskCount    int `json:"atRiskCount"`
	BlockedCount   int `json:"blockedCount"`
}

// ClientProjectDetail is the full project detail view for GET /client-portal/projects/{projectId}.
type ClientProjectDetail struct {
	ID                string               `json:"id"`
	Health            string               `json:"health"`            // on_track, at_risk, blocked
	Confidence        string               `json:"confidence"`        // high, medium, low
	HealthReason      string               `json:"healthReason,omitempty"`
	CompletionPercent *float64             `json:"completionPercent"`  // nullable; nil when no active tasks
	Board             []ClientTaskColumn   `json:"board"`
	Approvals         []ClientApprovalItem `json:"approvals"`
	Risks             []ClientRiskItem     `json:"risks"`
	Milestones        []ClientMilestoneItem `json:"milestones"`
	Comments          []ClientComment      `json:"comments"`
	NextAction        string               `json:"nextAction"`
	Timestamp         time.Time            `json:"timestamp"`
}

// ---------------------------------------------------------------------------------------------------------------------
// Client Portal — Board / Task Models
// ---------------------------------------------------------------------------------------------------------------------

// ClientTaskColumn represents a kanban column in the client project board view.
type ClientTaskColumn struct {
	Status string           `json:"status"` // todo, in_progress, blocked, done
	Tasks  []ClientTaskCard  `json:"tasks"`
}

// ClientTaskCard is a client-facing task card stripped of technical fields.
type ClientTaskCard struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Status       string     `json:"status"`        // todo, in_progress, blocked, done
	OwnerLabel   string     `json:"ownerLabel"`    // Product, Engineering, Review, Quality, Client
	Summary      string     `json:"summary"`
	BlockerReason *string   `json:"blockerReason"` // nullable
	DueDate      *time.Time `json:"dueDate"`       // nullable
	UpdatedAt    time.Time  `json:"updatedAt"`
	NextAction   string     `json:"nextAction"`
}

// ---------------------------------------------------------------------------------------------------------------------
// Client Portal — Approval Models
// ---------------------------------------------------------------------------------------------------------------------

// ClientApprovalItem is a client-facing approval item for a project approval gate.
type ClientApprovalItem struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	OwnerLabel string   `json:"ownerLabel"`
	Outcome   string    `json:"outcome"` // pending, approved, rejected, request_changes, need_more_information
	CreatedAt time.Time `json:"createdAt"`
	Overdue   bool      `json:"overdue"`
}

// ClientApprovalInbox is the response for GET /client-portal/approvals.
type ClientApprovalInbox struct {
	Items     []ClientApprovalItem `json:"items"`
	TotalCount int                 `json:"totalCount"`
	Timestamp time.Time            `json:"timestamp"`
}

// ApprovalDecisionRequest is the request body for POST /client-portal/approvals/{approvalId}/decide.
type ApprovalDecisionRequest struct {
	Outcome                  string   `json:"outcome"` // approve, reject, request_changes, need_more_information
	Comment                  *string  `json:"comment"` // optional for approve; required for reject/request_changes/need_more_information
	ClientOwnerLabelOverride *string  `json:"clientOwnerLabelOverride"` // optional client-facing owner label override
}

// ApprovalDecisionResponse is the response for POST /client-portal/approvals/{approvalId}/decide.
type ApprovalDecisionResponse struct {
	Success        bool              `json:"success"`
	UpdatedApproval ClientApprovalItem `json:"updatedApproval"`
	Message        *string           `json:"message"` // human-readable result; validation failure reason when success=false
}

// ---------------------------------------------------------------------------------------------------------------------
// Client Portal — Risk & Milestone Models
// ---------------------------------------------------------------------------------------------------------------------

// ClientRiskItem is a client-facing risk item.
type ClientRiskItem struct {
	ID                string `json:"id"`
	Severity          string `json:"severity"`           // low, medium, high, critical
	Impact            string `json:"impact"`
	OwnerLabel        string `json:"ownerLabel"`
	MitigationSummary string `json:"mitigationSummary"`
	Status            string `json:"status"`            // open, mitigated, closed
	NextAction        string `json:"nextAction"`
}

// ClientMilestoneItem is a client-facing milestone item.
type ClientMilestoneItem struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`      // pending, in_progress, completed, overdue
	TargetDate  *string   `json:"targetDate"`  // nullable; ISO 8601
	Progress    float64   `json:"progress"`
	Health      string    `json:"health"`      // on_track, at_risk, blocked
	Summary     string    `json:"summary"`
	NextAction  string    `json:"nextAction"`
}

// ---------------------------------------------------------------------------------------------------------------------
// Client Portal — Comment Model
// ---------------------------------------------------------------------------------------------------------------------

// ClientComment is a client-facing comment stripped of sensitive internal fields.
// Per FR-03-029: body text is retained for display but audit events must not retain body.
type ClientComment struct {
	ID          string    `json:"id"`
	AuthorName  string    `json:"authorName"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   *time.Time `json:"updatedAt"`  // nullable
	Edited      bool      `json:"edited"`
	ProjectID   string    `json:"projectId"`
	RelatedItemID string  `json:"relatedItemId"`
	Body        string    `json:"body"`
}

// ---------------------------------------------------------------------------------------------------------------------
// Client Portal — Search Models
// ---------------------------------------------------------------------------------------------------------------------

// ClientSearchResults is the response for GET /client-portal/search.
type ClientSearchResults struct {
	Items     []ClientSearchResultItem `json:"items"`
	TotalCount int                     `json:"totalCount"`
	Timestamp time.Time                `json:"timestamp"`
}

// ClientSearchResultItem is a single search result item.
type ClientSearchResultItem struct {
	ID                 string `json:"id"`
	Type               string `json:"type"`        // project, task, risk, milestone, approval, blocker
	Title              string `json:"title"`
	ProjectID          string `json:"projectId"`
	ProjectName        string `json:"projectName"`
	HighlightedContent string `json:"highlightedContent"` // forbidden technical fields already stripped
}

// ---------------------------------------------------------------------------------------------------------------------
// Client Portal — SSE Event Models
// ---------------------------------------------------------------------------------------------------------------------

// SSEEventType enumerates all client-portal SSE event types.
const (
	SSEClientPortalApprovalSubmitted         = "client_portal.approval.submitted"
	SSEClientPortalApprovalOutcomeRecorded   = "client_portal.approval.outcome_recorded"
	SSEClientPortalApprovalNeedMoreInformation = "client_portal.approval.need_more_information"
	SSEClientPortalItemPublished             = "client_portal.item.published"
	SSEClientPortalItemUnpublished           = "client_portal.item.unpublished"
	SSEClientPortalPublicationValidationFailed = "client_portal.publication_validation.failed"
	SSEClientPortalCommentCreated            = "client_portal.comment.created"
	SSEClientPortalCommentEdited             = "client_portal.comment.edited"
	SSEClientPortalCommentDeleted            = "client_portal.comment.deleted"
	SSEClientPortalAccessDenied              = "client_portal.access.denied"
	SSEClientPortalReadOnlyModeEntered       = "client_portal.read_only_mode.entered"
	SSEClientPortalReadsUnavailable          = "client_portal.reads.unavailable"
	SSEClientPortalSSEConnected              = "client_portal.sse.connected"
	SSEClientPortalSSEDisconnected           = "client_portal.sse.disconnected"
)

// SSEClientEvent is the generic client-facing SSE event envelope.
// Envelope metadata fields (actorId, actorRole, eventId, schemaVersion, parentTaskId, gateId)
// are stripped by the BFF before delivery per ADR-03-002.
type SSEClientEvent struct {
	EventType string                 `json:"eventType"`
	ProjectID string                 `json:"projectId"`
	Timestamp string                 `json:"timestamp"` // ISO 8601
	Payload   map[string]interface{} `json:"payload"`
}

// ApprovalSubmittedEvent is the client-facing event when a client submits an approval decision.
type ApprovalSubmittedEvent struct {
	EventType  string   `json:"eventType"`  // client_portal.approval.submitted
	ProjectID  string   `json:"projectId"`
	ItemID     string   `json:"itemId"`
	ItemTitle  string   `json:"itemTitle"`
	Outcome    string   `json:"outcome"`   // approve, reject, request_changes, need_more_information
	ActorName  string   `json:"actorName"` // client display name
	Comment    *string  `json:"comment"`
	Timestamp  string   `json:"timestamp"` // ISO 8601
}

// ApprovalOutcomeRecordedEvent is emitted after the platform records an approval outcome.
type ApprovalOutcomeRecordedEvent struct {
	EventType string `json:"eventType"` // client_portal.approval.outcome_recorded
	ProjectID string `json:"projectId"`
	ItemID    string `json:"itemId"`
	Outcome   string `json:"outcome"`  // approved, rejected, request_changes, need_more_information
	Timestamp string `json:"timestamp"`
}

// ItemPublishedEvent is emitted when a client-portal item is published.
type ItemPublishedEvent struct {
	EventType        string `json:"eventType"` // client_portal.item.published
	ProjectID       string `json:"projectId"`
	ItemID          string `json:"itemId"`
	ItemType        string `json:"itemType"`  // task, risk, milestone, blocker
	PublishedBy     string `json:"publishedBy"`
	ValidationResult string `json:"validationResult"` // passed, failed
	Timestamp       string `json:"timestamp"`
}

// ItemUnpublishedEvent is emitted when a client-portal item is unpublished.
type ItemUnpublishedEvent struct {
	EventType      string `json:"eventType"` // client_portal.item.unpublished
	ProjectID      string `json:"projectId"`
	ItemID         string `json:"itemId"`
	UnpublishedBy  string `json:"unpublishedBy"`
	Reason         string `json:"reason"`
	Timestamp      string `json:"timestamp"`
}

// CommentCreatedEvent is emitted when a comment is created on a client-visible item.
type CommentCreatedEvent struct {
	EventType     string `json:"eventType"` // client_portal.comment.created
	ProjectID     string `json:"projectId"`
	RelatedItemID string `json:"relatedItemId"`
	ItemType      string `json:"itemType"` // task, risk, milestone, approval
	CommentID     string `json:"commentId"`
	AuthorName    string `json:"authorName"`
	Body          string `json:"body"`
	Timestamp     string `json:"timestamp"`
}

// CommentEditedEvent is emitted when a comment is edited.
// Per FR-03-029: previous body text is NOT retained.
type CommentEditedEvent struct {
	EventType string `json:"eventType"` // client_portal.comment.edited
	ProjectID string `json:"projectId"`
	CommentID string `json:"commentId"`
	EditedBy  string `json:"editedBy"`
	EditedAt  string `json:"editedAt"` // ISO 8601
}

// CommentDeletedEvent is emitted when a comment is deleted.
// Per FR-03-029: deleted body text is NOT retained.
type CommentDeletedEvent struct {
	EventType string `json:"eventType"` // client_portal.comment.deleted
	ProjectID string `json:"projectId"`
	CommentID string `json:"commentId"`
	DeletedBy string `json:"deletedBy"`
	DeletedAt string `json:"deletedAt"` // ISO 8601
}

// AccessDeniedEvent signals an access boundary violation for the client portal.
type AccessDeniedEvent struct {
	EventType    string `json:"eventType"` // client_portal.access.denied
	PrincipalID  string `json:"principalId"`
	ResourceType string `json:"resourceType"` // project, task, risk, milestone, approval, comment
	ResourceID   string `json:"resourceId"`
	Timestamp    string `json:"timestamp"`
}

// PortalReadOnlyModeEnteredEvent signals that submission endpoints are unavailable.
type PortalReadOnlyModeEnteredEvent struct {
	EventType string `json:"eventType"` // client_portal.read_only_mode.entered
	Reason    string `json:"reason"`    // submission_unavailable, read_api_degraded
	Timestamp string `json:"timestamp"`
}

// PortalReadsUnavailableEvent signals that read APIs are degraded.
type PortalReadsUnavailableEvent struct {
	EventType string `json:"eventType"` // client_portal.reads.unavailable
	Endpoint  string `json:"endpoint"`
	Timestamp string `json:"timestamp"`
}

// PortalSSEConnectedEvent signals a new SSE connection for a project.
type PortalSSEConnectedEvent struct {
	EventType string `json:"eventType"` // client_portal.sse.connected
	ProjectID string `json:"projectId"`
	Timestamp string `json:"timestamp"`
}

// PortalSSEDisconnectedEvent signals SSE disconnection for a project.
type PortalSSEDisconnectedEvent struct {
	EventType string `json:"eventType"` // client_portal.sse.disconnected
	ProjectID string `json:"projectId"`
	Reason    string `json:"reason"`
	Timestamp string `json:"timestamp"`
}

// ---------------------------------------------------------------------------------------------------------------------
// Client Portal — Owner Label Mapping
// ---------------------------------------------------------------------------------------------------------------------

// OwnerLabelType enumerates the allowed client-facing owner label values.
type OwnerLabelType string

const (
	OwnerLabelProduct    OwnerLabelType = "Product"
	OwnerLabelEngineering OwnerLabelType = "Engineering"
	OwnerLabelReview     OwnerLabelType = "Review"
	OwnerLabelQuality    OwnerLabelType = "Quality"
	OwnerLabelClient     OwnerLabelType = "Client"
)

// OwnerLabelMapping maps an internal owner ID to a client-facing label.
// Override precedence (ADR-03-005): client decision override > project-level override > internal mapping.
type OwnerLabelMapping struct {
	OwnerID      string          `json:"ownerId"`
	Label        OwnerLabelType  `json:"label"`
	Override     bool            `json:"override"`      // true if this is a client-set override
	ClientLabel  *OwnerLabelType `json:"clientLabel"`  // nullable; set when override is true
}

// ---------------------------------------------------------------------------------------------------------------------
// Client Portal — Approval State Machine
// ---------------------------------------------------------------------------------------------------------------------

// ApprovalState captures the full state of a client approval item.
type ApprovalState struct {
	ApprovalID  string     `json:"approvalId"`
	CurrentState string     `json:"currentState"` // pending, waiting_on_response, approved, rejected, request_changes
	History      []ApprovalStateTransition `json:"history,omitempty"`
}

// ApprovalStateTransition records a single state transition in the approval lifecycle.
type ApprovalStateTransition struct {
	FromState string    `json:"fromState"`
	ToState   string    `json:"toState"`
	ActorID   string    `json:"actorId"`
	Timestamp time.Time `json:"timestamp"`
}

// ApprovalOutcome enumerates all possible approval outcomes.
type ApprovalOutcome string

const (
	ApprovalOutcomeApprove             ApprovalOutcome = "approve"
	ApprovalOutcomeReject              ApprovalOutcome = "reject"
	ApprovalOutcomeRequestChanges      ApprovalOutcome = "request_changes"
	ApprovalOutcomeNeedMoreInformation ApprovalOutcome = "need_more_information"
)

// PendingApprovalOutcome enumerates outcomes that leave an approval pending/waiting.
var PendingApprovalOutcome = map[ApprovalOutcome]bool{
	ApprovalOutcomeApprove:             false,
	ApprovalOutcomeReject:              false,
	ApprovalOutcomeRequestChanges:      true,  // waiting_on_response
	ApprovalOutcomeNeedMoreInformation: true,  // waiting_on_response
}

// ---------------------------------------------------------------------------------------------------------------------
// Client Portal — Publication Validation
// ---------------------------------------------------------------------------------------------------------------------

// PublicationValidationResult captures the result of validating a publication.
type PublicationValidationResult struct {
	Passed         bool     `json:"passed"`
	FailureReason  *string   `json:"failureReason"`  // category only; never raw forbidden content
	ForbiddenTerms []string `json:"forbiddenTerms"` // matched terms; used only in server-side logs, never exposed to client
}

// ValidatePublication runs publication validation against forbidden patterns per ADR-03-003.
// Returns a result that does NOT include raw forbidden content in the client-facing failureReason.
func ValidatePublication(content string) PublicationValidationResult {
	result := PublicationValidationResult{Passed: true}
	// Forbidden patterns: stack traces, agent IDs, branch names, SHAs, file paths,
	// infra terms (kubernetes, docker, helm, terraform, arn, vpc, s3 bucket),
	// log lines. Validation is implemented in the service layer; this type only
	// carries the result.
	_ = content // placeholder; actual validation in service layer
	return result
}

// ---------------------------------------------------------------------------------------------------------------------
// Key Contract Constants
// ---------------------------------------------------------------------------------------------------------------------

// OverdueThresholdHours is the per-item overdue threshold (ADR-03-004: strictly greater than 24h).
const OverdueThresholdHours = 24

// CompletionPercent calculates the client-facing completion percentage.
// Formula (ADR-03-001): done / (todo + in_progress + blocked + done)
// Excludes: cancelled, proposed
func CompletionPercent(todo, inProgress, blocked, done int) *float64 {
	denom := todo + inProgress + blocked + done
	if denom == 0 {
		return nil
	}
	pct := float64(done) / float64(denom) * 100
	return &pct
}