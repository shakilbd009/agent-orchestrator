package models

import (
	"time"
)

// Project represents a BRD-02 project-scoped container for tasks, gates, and events.
type Project struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Owner       string         `json:"owner,omitempty"`
	Status      string         `json:"status"` // active, archived
	Phase       string         `json:"phase"`  // planning, decomposition, execution, validation, acceptance, closed
	PhaseGates  []ProjectPhaseGate `json:"phaseGates,omitempty"`
	Statistics  *ProjectStatistics `json:"statistics,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

// ProjectStatistics holds aggregate task counts for a project.
type ProjectStatistics struct {
	TotalTasks      int `json:"totalTasks"`
	TodoTasks       int `json:"todoTasks"`
	InProgressTasks int `json:"inProgressTasks"`
	DoneTasks       int `json:"doneTasks"`
	BlockedTasks    int `json:"blockedTasks"`
	ActiveGateCount int `json:"activeGateCount"`
}

// ProjectPhaseGate is a project-level phase gate (G0, G1, G2, etc.).
type ProjectPhaseGate struct {
	ID           string     `json:"id"`
	ProjectID    string     `json:"projectId"`
	PhaseIndex   int        `json:"phaseIndex"`
	Phase        string     `json:"phase"`
	State        string     `json:"state"` // open, passed, blocked
	Criteria     []string   `json:"criteria,omitempty"`
	PassCondition string    `json:"passCondition,omitempty"`
	PassedAt     *time.Time `json:"passedAt,omitempty"`
	PassedBy     string     `json:"passedBy,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

// OrchestrationTask is a BRD-02 task scoped to a project.
type OrchestrationTask struct {
	ID            string           `json:"id"`
	ProjectID     string           `json:"projectId"`
	Title         string           `json:"title"`
	Body          string           `json:"body,omitempty"`
	Status        string           `json:"status"` // todo, in_progress, blocked, done, cancelled
	Layer         string           `json:"layer"`   // A, B
	Assignee      string           `json:"assignee,omitempty"`
	Parents       []string         `json:"parents,omitempty"`
	Children      []string         `json:"children,omitempty"`
	Dependencies  []TaskDependency `json:"dependencies,omitempty"`
	Gates         []TaskGate       `json:"gates,omitempty"`
	Required      bool             `json:"required"`
	Priority      int              `json:"priority"`
	Stale         bool             `json:"stale,omitempty"`
	StaleThresholdMinutes int      `json:"staleThresholdMinutes,omitempty"`
	BlockedReason string           `json:"blockedReason,omitempty"`
	WorkspaceKind string           `json:"workspaceKind,omitempty"`
	WorkspacePath string           `json:"workspacePath,omitempty"`
	Tags          []string         `json:"tags,omitempty"`
	Metadata      map[string]any   `json:"metadata,omitempty"`
	CreatedAt     time.Time        `json:"createdAt"`
	UpdatedAt     time.Time        `json:"updatedAt"`
	CompletedAt   *time.Time       `json:"completedAt,omitempty"`
}

// TaskDependency models a dependency edge between two tasks.
type TaskDependency struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"projectId"`
	SourceTaskID string    `json:"sourceTaskId"` // task that has the dependency
	TargetTaskID string    `json:"targetTaskId"` // task that must complete first
	Type         string    `json:"type"`         // blocks, depends_on, handoff
	CreatedAt    time.Time `json:"createdAt"`
}

// TaskDependencyGraph holds parent and child dependency edges for a task.
type TaskDependencyGraph struct {
	TaskID  string           `json:"taskId"`
	Parents []TaskDependency `json:"parents"`
	Children []TaskDependency `json:"children"`
}

// TaskGate is a task-level quality gate.
type TaskGate struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"projectId"`
	TaskID      string     `json:"taskId"`
	Phase       string     `json:"phase"` // scope_review, architecture_review, implementation_review, code_review, qa_review, release_review
	State       string     `json:"state"` // open, passed, blocked
	Criteria    []string   `json:"criteria,omitempty"`
	Blocking    bool       `json:"blocking"`
	PassedAt    *time.Time `json:"passedAt,omitempty"`
	PassedBy    string     `json:"passedBy,omitempty"`
	OverrideNote string    `json:"overrideNote,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// DecompositionProposal is a proposed set of child tasks for a parent task.
type DecompositionProposal struct {
	ID           string           `json:"id"`
	ProjectID    string           `json:"projectId"`
	ParentTaskID string           `json:"parentTaskId"`
	Submitter    string           `json:"submitter"`
	State        string           `json:"state"` // draft, submitted, accepted, rejected
	ProposedTasks []DecomposedTaskSpec `json:"proposedTasks"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
}

// DecomposedTaskSpec describes a single child in a decomposition proposal.
type DecomposedTaskSpec struct {
	Title       string   `json:"title"`
	Body        string   `json:"body,omitempty"`
	Layer       string   `json:"layer"` // A, B
	Assignee    string   `json:"assignee,omitempty"`
	Order       int      `json:"order"`
	Dependencies []string `json:"dependencies,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// WebhookRegistration is a registered outbound webhook consumer.
type WebhookRegistration struct {
	ID            string            `json:"id"`
	ProjectID     string            `json:"projectId"`
	URL           string            `json:"url"`
	Events        []string          `json:"events"` // event types to subscribe to
	Active        bool              `json:"active"`
	Secret        string            `json:"secret,omitempty"` // never returned after creation
	DeliveryStatus *WebhookDeliveryStatus `json:"deliveryStatus,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
}

// WebhookDeliveryStatus tracks recent delivery attempts.
type WebhookDeliveryStatus struct {
	LastAttemptAt *time.Time `json:"lastAttemptAt,omitempty"`
	LastSuccessAt *time.Time `json:"lastSuccessAt,omitempty"`
	FailureCount  int        `json:"failureCount"`
}

// DeliveryRecord represents a pending webhook delivery row from the DB queue.
// Used by the webhook worker to process retries.
type DeliveryRecord struct {
	ID        int
	WebhookID string
	EventID   string
	Payload   []byte
	Attempts  int
}

// HandoffEvidence is structured completion evidence for a Layer B task.
type HandoffEvidence struct {
	ID         string         `json:"id"`
	ProjectID  string         `json:"projectId"`
	TaskID     string         `json:"taskId"`
	FromAgent  string         `json:"fromAgent"`
	ToAgent    string         `json:"toAgent,omitempty"`
	Summary    string         `json:"summary"`
	Artifacts  []string       `json:"artifacts,omitempty"`
	ValidationPerformed string `json:"validationPerformed,omitempty"`
	RisksOrResidualIssues string `json:"risksOrResidualIssues,omitempty"`
	RecommendedNextGate   string `json:"recommendedNextGate,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
}

// AuditEvent is the canonical 11-field event envelope per ADR-02-001.
type AuditEvent struct {
	EventID       string         `json:"eventId"`
	SchemaVersion string         `json:"schemaVersion"` // always "v1alpha"
	ProjectID     string         `json:"projectId"`
	Topic         string         `json:"topic"`
	ActorID       string         `json:"actorId"`
	ActorRole     string         `json:"actorRole"` // human, layer_a, layer_b, system
	TaskID        *string        `json:"taskId,omitempty"`
	ParentTaskID  *string        `json:"parentTaskId,omitempty"`
	GateID        *string        `json:"gateId,omitempty"`
	Timestamp     string         `json:"timestamp"` // ISO 8601
	Payload       map[string]any `json:"payload"`
}

// Actor is the actor identity and role extracted from request headers.
type Actor struct {
	ID   string `json:"actorId"`
	Role string `json:"role"` // human, layer_a, layer_b, system
}

// FeatureFlags holds all platform feature flag states.
type FeatureFlags struct {
	PlatformOrchestration bool `json:"platform-orchestration"`
	LayerAAgents          bool `json:"layer-a-agents"`
	LayerBAgents          bool `json:"layer-b-agents"`
	HumanGates            bool `json:"human-gates"`
	AuditTrail            bool `json:"audit-trail"`
}

// OrchestrationTaskCreateRequest is the request body for creating a task.
type OrchestrationTaskCreateRequest struct {
	Title       string   `json:"title"`
	Body        string   `json:"body,omitempty"`
	Assignee    string   `json:"assignee,omitempty"`
	Layer       string   `json:"layer"`
	Parents     []string `json:"parents,omitempty"`
	Priority    int      `json:"priority"`
	WorkspaceKind string `json:"workspaceKind,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// TaskGateCreateRequest is the request body for creating a task gate.
type TaskGateCreateRequest struct {
	Phase    string   `json:"phase"`
	Criteria []string `json:"criteria,omitempty"`
	Blocking bool     `json:"blocking"`
}

// DecompositionProposalSubmitRequest is the request body for submitting a decomposition proposal.
type DecompositionProposalSubmitRequest struct {
	ProposedTasks []DecomposedTaskSpec `json:"proposedTasks"`
	Override     bool                 `json:"override"`
	OverrideReason string             `json:"overrideReason,omitempty"`
}

// WebhookRegistrationRequest is the request body for registering a webhook.
type WebhookRegistrationRequest struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Secret string   `json:"secret"`
}

// ProjectUpdateRequest is the request body for updating a project.
type ProjectUpdateRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// TaskCompleteRequest is the request body for completing a task with handoff evidence.
type TaskCompleteRequest struct {
	Handoff HandoffEvidenceInput `json:"handoff"`
}

type HandoffEvidenceInput struct {
	Summary               string   `json:"summary"`
	Artifacts             []string `json:"artifacts,omitempty"`
	ValidationPerformed   string   `json:"validationPerformed,omitempty"`
	RisksOrResidualIssues string   `json:"risksOrResidualIssues,omitempty"`
	RecommendedNextGate   string   `json:"recommendedNextGate,omitempty"`
}

// Error represents an RFC 7807 problem detail.
type Error struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}