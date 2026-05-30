package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/agent-orchestrator/backend/internal/middleware"
	"github.com/agent-orchestrator/backend/internal/models"
)

// ---------------------------------------------------------------------------
// Repository interfaces — allow mock implementations in unit tests without
// pulling in the full repository package (avoids circular deps and DB deps).
// ---------------------------------------------------------------------------

// ProjectRepo abstracts the project repository for service-layer testing.
type ProjectRepo interface {
	Create(ctx context.Context, p *models.Project) error
	GetByID(ctx context.Context, id string) (*models.Project, error)
	Update(ctx context.Context, p *models.Project) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]models.Project, error)
	GetStatistics(ctx context.Context, id string) (*models.ProjectStatistics, error)
}

// TaskRepo abstracts the task repository for service-layer testing.
type TaskRepo interface {
	CreateTask(ctx context.Context, t *models.OrchestrationTask, actor *models.Actor) error
	GetTaskByID(ctx context.Context, projectID, taskID string) (*models.OrchestrationTask, error)
	ListTasksByProject(ctx context.Context, projectID, status, layer, assignee string) ([]models.OrchestrationTask, error)
	UpdateTaskStatus(ctx context.Context, projectID, taskID, newStatus string, actor *models.Actor, reason string) error
	BlockTask(ctx context.Context, projectID, taskID, reason string, actor *models.Actor) error
	CanCompleteParent(ctx context.Context, taskID string) error
	CountActiveChildren(ctx context.Context, parentTaskID string) (int, error)
	GetTaskDepth(ctx context.Context, taskID string) (int, error)
}

// GateRepo abstracts the gate repository for service-layer testing.
type GateRepo interface {
	CreateTaskGate(ctx context.Context, g *models.TaskGate, actor *models.Actor) error
	GetTaskGate(ctx context.Context, projectID, taskID, gateID string) (*models.TaskGate, error)
	UpdateGateState(ctx context.Context, projectID, taskID, gateID, newState, actorID, actorRole, note string) error
	ListProjectPhaseGates(ctx context.Context, projectID string) ([]models.ProjectPhaseGate, error)
	CreateProjectPhaseGate(ctx context.Context, g *models.ProjectPhaseGate) error
	GetProjectPhaseGate(ctx context.Context, projectID, gateID string) (*models.ProjectPhaseGate, error)
	UpdateProjectPhaseGate(ctx context.Context, g *models.ProjectPhaseGate) error
}

// DecompRepo abstracts the decomposition repository for service-layer testing.
type DecompRepo interface {
	SubmitProposal(ctx context.Context, dp *models.DecompositionProposal, proposedDepth, totalChildren int, override bool, actorID, actorRole, overrideReason string) error
	ApproveProposal(ctx context.Context, proposalID, actorID, actorRole string) error
	RejectProposal(ctx context.Context, proposalID, actorID, reason string) error
	GetProposal(ctx context.Context, id string) (*models.DecompositionProposal, error)
	GetActiveProposalForParent(ctx context.Context, parentTaskID string) (*models.DecompositionProposal, error)
}

// WebhookRepo abstracts the webhook repository for service-layer testing.
type WebhookRepo interface {
	Create(ctx context.Context, wh *models.WebhookRegistration) error
	ListActiveByProject(ctx context.Context, projectID string) ([]models.WebhookRegistration, error)
	Delete(ctx context.Context, id string) error
}

// EventSvc abstracts the event service for service-layer testing.
type EventSvc interface {
	Emit(ctx context.Context, ev models.AuditEvent)
}

// OrchestrationService is the main service for BRD-02 orchestration.
type OrchestrationService struct {
	projectRepo ProjectRepo
	taskRepo    TaskRepo
	gateRepo    GateRepo
	decompRepo  DecompRepo
	webhookRepo WebhookRepo
	eventSvc    EventSvc
}

// NewOrchestrationService creates a new orchestration service.
func NewOrchestrationService(
	projectRepo ProjectRepo,
	taskRepo TaskRepo,
	gateRepo GateRepo,
	decompRepo DecompRepo,
	webhookRepo WebhookRepo,
	eventSvc EventSvc,
) *OrchestrationService {
	return &OrchestrationService{
		projectRepo: projectRepo,
		taskRepo:    taskRepo,
		gateRepo:    gateRepo,
		decompRepo:  decompRepo,
		webhookRepo: webhookRepo,
		eventSvc:    eventSvc,
	}
}

// ProjectService handles project CRUD operations.
type ProjectService struct {
	repo     ProjectRepo
	eventSvc EventSvc
}

func NewProjectService(repo ProjectRepo, eventSvc EventSvc) *ProjectService {
	return &ProjectService{repo: repo, eventSvc: eventSvc}
}

func (s *ProjectService) CreateProject(ctx context.Context, req *models.Project, actor *models.Actor) (*models.Project, error) {
	if !middleware.FeatureFlags.PlatformOrchestration {
		return nil, fmt.Errorf("platform-orchestration flag must be enabled")
	}

	now := time.Now()
	if req.ID == "" {
		req.ID = newID("p")
	}
	req.CreatedAt = now
	req.UpdatedAt = now
	req.Status = "active"
	if req.Phase == "" {
		req.Phase = "planning"
	}

	if err := s.repo.Create(ctx, req); err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	// Emit event (async, non-blocking)
	go s.eventSvc.Emit(ctx, models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     req.ID,
		Topic:         "project.created",
		ActorID:       actor.ID,
		ActorRole:     actor.Role,
		TaskID:        nil,
		ParentTaskID:  nil,
		GateID:        nil,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"projectName":               req.Name,
			"createdBy":                 actor.ID,
			"staleThresholdMinutes":     nil,
			"decompositionDepthDefault": 3,
			"decompositionFanOutDefault": 20,
		},
	})

	return req, nil
}

func (s *ProjectService) GetProject(ctx context.Context, id string) (*models.Project, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	if p == nil {
		return nil, fmt.Errorf("project not found")
	}

	stats, err := s.repo.GetStatistics(ctx, id)
	if err != nil {
		log.Printf("stats error: %v", err)
	}
	p.Statistics = stats

	return p, nil
}

func (s *ProjectService) UpdateProject(ctx context.Context, id string, req *models.Project) error {
	req.ID = id
	return s.repo.Update(ctx, req)
}

func (s *ProjectService) DeleteProject(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *ProjectService) ListProjects(ctx context.Context) ([]models.Project, error) {
	return s.repo.List(ctx)
}

// TaskService handles task CRUD operations.
type TaskService struct {
	repo        TaskRepo
	projectRepo ProjectRepo
	eventSvc    EventSvc
}

func NewTaskService(repo TaskRepo, projectRepo ProjectRepo, eventSvc EventSvc) *TaskService {
	return &TaskService{repo: repo, projectRepo: projectRepo, eventSvc: eventSvc}
}

func (s *TaskService) CreateTask(ctx context.Context, projectID string, req *models.OrchestrationTaskCreateRequest, actor *models.Actor) (*models.OrchestrationTask, error) {
	if !middleware.FeatureFlags.PlatformOrchestration {
		return nil, fmt.Errorf("platform-orchestration flag must be enabled")
	}

	// Layer A only (FR-02-002 / BRD-02)
	if actor.Role != "human" && actor.Role != "layer_a" {
		return nil, fmt.Errorf("only human or layer_a can create tasks")
	}

	now := time.Now()
	task := &models.OrchestrationTask{
		ID:                   newID("t"),
		ProjectID:            projectID,
		Title:                req.Title,
		Body:                 req.Body,
		Status:               "todo",
		Layer:                req.Layer,
		Assignee:             req.Assignee,
		Required:             false, // default, can be overridden
		Priority:             req.Priority,
		Stale:                false,
		StaleThresholdMinutes: 0,
		WorkspaceKind:        req.WorkspaceKind,
		WorkspacePath:        "",
		Tags:                 req.Tags,
		Metadata:             map[string]any{},
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := s.repo.CreateTask(ctx, task, actor); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	// Emit event (async for webhooks, sync for SSE)
	go s.eventSvc.Emit(ctx, models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     projectID,
		Topic:         "task.created",
		ActorID:       actor.ID,
		ActorRole:     actor.Role,
		TaskID:        &task.ID,
		ParentTaskID:  nil,
		GateID:        nil,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"taskType":        "feature",
			"assignee":        task.Assignee,
			"title":           task.Title,
			"executionStatus": task.Status,
			"layer":           task.Layer,
			"required":        false,
		},
	})

	return task, nil
}

func (s *TaskService) GetTask(ctx context.Context, projectID, taskID string) (*models.OrchestrationTask, error) {
	task, err := s.repo.GetTaskByID(ctx, projectID, taskID)
	if err != nil || task == nil {
		return nil, fmt.Errorf("task not found")
	}
	return task, nil
}

func (s *TaskService) ListTasks(ctx context.Context, projectID, status, layer, assignee string) ([]models.OrchestrationTask, error) {
	return s.repo.ListTasksByProject(ctx, projectID, status, layer, assignee)
}

func (s *TaskService) UpdateTaskStatus(ctx context.Context, projectID, taskID, newStatus string, actor *models.Actor, reason string) error {
	if err := s.repo.UpdateTaskStatus(ctx, projectID, taskID, newStatus, actor, reason); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	var topic string
	switch newStatus {
	case "done":
		topic = "task.completed"
	case "blocked":
		topic = "task.blocked"
	case "cancelled":
		topic = "task.cancelled"
	default:
		topic = "task.status.changed"
	}

	go s.eventSvc.Emit(ctx, models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     projectID,
		Topic:         topic,
		ActorID:       actor.ID,
		ActorRole:     actor.Role,
		TaskID:        &taskID,
		ParentTaskID:  nil,
		GateID:        nil,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload:       map[string]any{"toStatus": newStatus, "reason": reason},
	})

	return nil
}

func (s *TaskService) BlockTask(ctx context.Context, projectID, taskID, reason string, actor *models.Actor) error {
	return s.repo.BlockTask(ctx, projectID, taskID, reason, actor)
}

func (s *TaskService) CompleteTask(ctx context.Context, projectID, taskID string, handoff *models.HandoffEvidence, actor *models.Actor) error {
	// Check required children and blocking gates (FR-02-006, FR-02-007)
	if err := s.repo.CanCompleteParent(ctx, taskID); err != nil {
		return fmt.Errorf("cannot complete: %w", err)
	}

	// Layer B scope: only assigned agent can complete
	task, err := s.repo.GetTaskByID(ctx, projectID, taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	if actor.Role == "layer_b" && task.Assignee != actor.ID {
		return fmt.Errorf("layer_b can only complete assigned tasks")
	}

	// Update status
	if err := s.repo.UpdateTaskStatus(ctx, projectID, taskID, "done", actor, "completed with handoff"); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	// Emit handoff.submitted event
	go s.eventSvc.Emit(ctx, models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     projectID,
		Topic:         "handoff.submitted",
		ActorID:       actor.ID,
		ActorRole:     actor.Role,
		TaskID:        &taskID,
		ParentTaskID:  nil,
		GateID:        nil,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"taskId":                  taskID,
			"submittedBy":             actor.ID,
			"summary":                 handoff.Summary,
			"artifacts":              handoff.Artifacts,
			"validationPerformed":    handoff.ValidationPerformed,
			"risksOrResidualIssues":   handoff.RisksOrResidualIssues,
			"recommendedNextGate":    handoff.RecommendedNextGate,
		},
	})

	return nil
}

// GateService handles gate operations.
type GateService struct {
	repo     GateRepo
	eventSvc EventSvc
}

func NewGateService(repo GateRepo, eventSvc EventSvc) *GateService {
	return &GateService{repo: repo, eventSvc: eventSvc}
}

func (s *GateService) CreateTaskGate(ctx context.Context, projectID, taskID string, req *models.TaskGateCreateRequest, actor *models.Actor) (*models.TaskGate, error) {
	if !middleware.FeatureFlags.PlatformOrchestration {
		return nil, fmt.Errorf("platform-orchestration flag must be enabled")
	}

	gate := &models.TaskGate{
		ID:        newID("tg"),
		ProjectID: projectID,
		TaskID:    taskID,
		Phase:     req.Phase,
		State:     "open",
		Criteria:  req.Criteria,
		Blocking:  req.Blocking,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateTaskGate(ctx, gate, actor); err != nil {
		return nil, fmt.Errorf("create gate: %w", err)
	}

	go s.eventSvc.Emit(ctx, models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     projectID,
		Topic:         "gate.opened",
		ActorID:       actor.ID,
		ActorRole:     actor.Role,
		TaskID:        &taskID,
		ParentTaskID:  nil,
		GateID:        &gate.ID,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"gateType":  gate.Phase,
			"gateLevel": "task",
			"blocking":  gate.Blocking,
			"openedBy":  actor.ID,
		},
	})

	return gate, nil
}

func (s *GateService) GetTaskGate(ctx context.Context, projectID, taskID, gateID string) (*models.TaskGate, error) {
	gate, err := s.repo.GetTaskGate(ctx, projectID, taskID, gateID)
	if err != nil {
		return nil, fmt.Errorf("gate not found")
	}
	return gate, nil
}

func (s *GateService) ApproveGate(ctx context.Context, projectID, taskID, gateID string, actor *models.Actor, overrideNote string) error {
	// Only human or layer_a can approve (FR-02-016)
	if actor.Role == "layer_b" {
		return fmt.Errorf("layer_b cannot approve gates")
	}
	if actor.Role == "system" {
		return fmt.Errorf("system cannot approve gates")
	}

	if err := s.repo.UpdateGateState(ctx, projectID, taskID, gateID, "passed", actor.ID, actor.Role, overrideNote); err != nil {
		return fmt.Errorf("approve gate: %w", err)
	}

	go s.eventSvc.Emit(ctx, models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     projectID,
		Topic:         "gate.approved",
		ActorID:       actor.ID,
		ActorRole:     actor.Role,
		TaskID:        &taskID,
		ParentTaskID:  nil,
		GateID:        &gateID,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"approvedBy":   actor.ID,
			"approverRole": actor.Role,
			"gateType":     "task_gate",
			"gateLevel":    "task",
		},
	})

	return nil
}

func (s *GateService) RejectGate(ctx context.Context, projectID, taskID, gateID string, actor *models.Actor, reason string) error {
	if actor.Role != "human" && actor.Role != "layer_a" {
		return fmt.Errorf("only human or layer_a can reject gates")
	}

	if err := s.repo.UpdateGateState(ctx, projectID, taskID, gateID, "blocked", actor.ID, actor.Role, reason); err != nil {
		return fmt.Errorf("reject gate: %w", err)
	}

	go s.eventSvc.Emit(ctx, models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     projectID,
		Topic:         "gate.rejected",
		ActorID:       actor.ID,
		ActorRole:     actor.Role,
		TaskID:        &taskID,
		ParentTaskID:  nil,
		GateID:        &gateID,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"rejectedBy":      actor.ID,
			"rejectionReason": reason,
			"gateType":        "task_gate",
			"gateLevel":       "task",
		},
	})

	return nil
}

// ListProjectPhaseGates returns all phase gates for a project.
func (s *GateService) ListProjectPhaseGates(ctx context.Context, projectID string) ([]models.ProjectPhaseGate, error) {
	return s.repo.ListProjectPhaseGates(ctx, projectID)
}

// CreateProjectPhaseGate creates a project-level phase gate.
func (s *GateService) CreateProjectPhaseGate(ctx context.Context, gate *models.ProjectPhaseGate) error {
	return s.repo.CreateProjectPhaseGate(ctx, gate)
}

// GetProjectPhaseGate retrieves a specific phase gate.
func (s *GateService) GetProjectPhaseGate(ctx context.Context, projectID, gateID string) (*models.ProjectPhaseGate, error) {
	return s.repo.GetProjectPhaseGate(ctx, projectID, gateID)
}

// UpdateProjectPhaseGate updates a phase gate.
func (s *GateService) UpdateProjectPhaseGate(ctx context.Context, gate *models.ProjectPhaseGate) error {
	return s.repo.UpdateProjectPhaseGate(ctx, gate)
}

// DecompositionService handles decomposition proposal lifecycle.
type DecompositionService struct {
	repo     DecompRepo
	taskRepo TaskRepo
	eventSvc EventSvc
}

func NewDecompositionService(repo DecompRepo, taskRepo TaskRepo, eventSvc EventSvc) *DecompositionService {
	return &DecompositionService{repo: repo, taskRepo: taskRepo, eventSvc: eventSvc}
}

// DecompositionLimits holds the depth and fanout limits for a project.
type DecompositionLimits struct {
	DefaultDepth   int
	DefaultFanout  int
	HardDepthCap   int
	HardFanoutCap  int
}

// DefaultDecompositionLimits returns the hard-coded default limits per BRD-02 FR-02-010.
func DefaultDecompositionLimits() DecompositionLimits {
	return DecompositionLimits{
		DefaultDepth:   3,
		DefaultFanout:  20,
		HardDepthCap:   5,
		HardFanoutCap:  50,
	}
}

// CheckDecompositionLimits validates depth and fanout against project limits.
// Returns an error describing the first violation encountered.
func (s *DecompositionService) CheckDecompositionLimits(ctx context.Context, parentTaskID string, proposedChildCount int, override bool, overrideReason string, actor *models.Actor) (int, int, error) {
	if s.taskRepo == nil {
		return 0, 0, fmt.Errorf("task repository not available")
	}

	// Compute current depth of parent task (root = depth 1)
	depth, err := s.taskRepo.GetTaskDepth(ctx, parentTaskID)
	if err != nil {
		return 0, 0, fmt.Errorf("compute parent depth: %w", err)
	}

	// Compute current active children (non-cancelled tasks in task_children join where child is not done/cancelled)
	activeChildren, err := s.taskRepo.CountActiveChildren(ctx, parentTaskID)
	if err != nil {
		return 0, 0, fmt.Errorf("compute active children: %w", err)
	}

	limits := DefaultDecompositionLimits()
	proposedDepth := depth + 1
	totalChildren := activeChildren + proposedChildCount

	// Hard cap: depth > 5 always rejected
	if proposedDepth > limits.HardDepthCap {
		return proposedDepth, totalChildren, fmt.Errorf("decomposition depth %d exceeds hard cap %d", proposedDepth, limits.HardDepthCap)
	}

	// Hard cap: fanout > 50 always rejected
	if totalChildren > limits.HardFanoutCap {
		return proposedDepth, totalChildren, fmt.Errorf("decomposition fanout %d exceeds hard cap %d", totalChildren, limits.HardFanoutCap)
	}

	// Soft cap: depth > 3 (default) requires override with actor+reason
	if proposedDepth > limits.DefaultDepth {
		if !override || overrideReason == "" {
			return proposedDepth, totalChildren, fmt.Errorf("decomposition depth %d exceeds project default %d; override required with reason", proposedDepth, limits.DefaultDepth)
		}
		// Override with actor+reason
	}

	// Soft cap: fanout > 20 (default) requires override with actor+reason
	if totalChildren > limits.DefaultFanout {
		if !override || overrideReason == "" {
			return proposedDepth, totalChildren, fmt.Errorf("decomposition fanout %d exceeds project default %d; override required with reason", totalChildren, limits.DefaultFanout)
		}
		// Override with actor+reason
	}

	return proposedDepth, totalChildren, nil
}

func (s *DecompositionService) ProposeDecomposition(ctx context.Context, projectID, taskID string, req *models.DecompositionProposalSubmitRequest, actor *models.Actor) (*models.DecompositionProposal, error) {
	if !middleware.FeatureFlags.PlatformOrchestration {
		return nil, fmt.Errorf("platform-orchestration flag must be enabled")
	}
	if !middleware.FeatureFlags.LayerAAgents && actor.Role != "human" {
		return nil, fmt.Errorf("layer-a-agents flag must be enabled for layer_a actors")
	}

	// Enforce decomposition limits (AC-02-008, AC-02-009, AC-02-010)
	// Actor passed to CheckDecompositionLimits so it can be included in audit event on override
	proposedDepth, totalChildren, err := s.CheckDecompositionLimits(ctx, taskID, len(req.ProposedTasks), req.Override, req.OverrideReason, actor)
	if err != nil {
		return nil, fmt.Errorf("decomposition limit check failed: %w", err)
	}

	dp := &models.DecompositionProposal{
		ID:            newID("dp"),
		ProjectID:     projectID,
		ParentTaskID:  taskID,
		Submitter:     actor.ID,
		State:         "submitted",
		ProposedTasks: req.ProposedTasks,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.SubmitProposal(ctx, dp, proposedDepth, totalChildren, req.Override, actor.ID, actor.Role, req.OverrideReason); err != nil {
		return nil, fmt.Errorf("submit proposal: %w", err)
	}

	eventTopic := "task.decomposition.proposed"
	if req.Override {
		eventTopic = "task.decomposition.override_used"
	}
	go s.eventSvc.Emit(ctx, models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     projectID,
		Topic:         eventTopic,
		ActorID:       actor.ID,
		ActorRole:     actor.Role,
		TaskID:        &taskID,
		ParentTaskID:  nil,
		GateID:        nil,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"parentTaskId":        taskID,
			"depthAtProposal":     proposedDepth,
			"activeChildrenCount": totalChildren,
			"override":            req.Override,
			"overrideReason":      req.OverrideReason,
			"actorId":             actor.ID,
			"actorRole":           actor.Role,
		},
	})

	return dp, nil
}

func (s *DecompositionService) ApproveProposal(ctx context.Context, proposalID string, actor *models.Actor) error {
	if actor.Role != "human" && actor.Role != "layer_a" {
		return fmt.Errorf("only human or layer_a can approve decomposition")
	}

	if err := s.repo.ApproveProposal(ctx, proposalID, actor.ID, actor.Role); err != nil {
		return fmt.Errorf("approve proposal: %w", err)
	}
	return nil
}

func (s *DecompositionService) RejectProposal(ctx context.Context, proposalID, reason string, actor *models.Actor) error {
	if err := s.repo.RejectProposal(ctx, proposalID, actor.ID, reason); err != nil {
		return fmt.Errorf("reject proposal: %w", err)
	}
	return nil
}

// WebhookService handles webhook registration and delivery.
type WebhookService struct {
	repo     WebhookRepo
	eventSvc EventSvc
}

func NewWebhookService(repo WebhookRepo, eventSvc EventSvc) *WebhookService {
	return &WebhookService{repo: repo, eventSvc: eventSvc}
}

func (s *WebhookService) RegisterWebhook(ctx context.Context, projectID string, req *models.WebhookRegistrationRequest, actor *models.Actor) (*models.WebhookRegistration, error) {
	if !middleware.FeatureFlags.PlatformOrchestration {
		return nil, fmt.Errorf("platform-orchestration flag must be enabled")
	}

	// ADR-02-003: secret required for non-localhost webhooks
	if !isLocalhostURL(req.URL) && req.Secret == "" {
		return nil, fmt.Errorf("webhook secret is required for non-localhost URLs")
	}

	wh := &models.WebhookRegistration{
		ID:        newID("wh"),
		ProjectID: projectID,
		URL:       req.URL,
		Events:    req.Events,
		Active:    true,
		Secret:    req.Secret,
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, wh); err != nil {
		return nil, fmt.Errorf("create webhook: %w", err)
	}

	return wh, nil
}

func (s *WebhookService) ListWebhooks(ctx context.Context, projectID string) ([]models.WebhookRegistration, error) {
	return s.repo.ListActiveByProject(ctx, projectID)
}

func (s *WebhookService) DeleteWebhook(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// ComputeWebhookSignature computes HMAC-SHA256 signature for webhook delivery (ADR-02-003).
func ComputeWebhookSignature(body []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// isLocalhostURL returns true if the URL is localhost or 127.0.0.1.
func isLocalhostURL(url string) bool {
	return strings.HasPrefix(url, "http://localhost") ||
		strings.HasPrefix(url, "http://127.0.0.1") ||
		strings.HasPrefix(url, "https://localhost") ||
		strings.HasPrefix(url, "https://127.0.0.1")
}

// ReadyService handles the /ready health check with proper degradation semantics (ADR-02-004).
type ReadyService struct {
	db        ReadyPool
	webhookOk func() bool // webhook queue availability check
}

// ReadyPool abstracts the DB pool for service-layer testing.
type ReadyPool interface {
	Ping(ctx context.Context) error
	Exec(ctx context.Context, query string) (interface{}, error)
}

func NewReadyService(db ReadyPool, webhookOk func() bool) *ReadyService {
	return &ReadyService{db: db, webhookOk: webhookOk}
}

// CheckReady returns 200 if all subsystems available, 503 with failing subsystem.
func (s *ReadyService) CheckReady(ctx context.Context) (int, string, map[string]string) {
	// Check storage (DB)
	if err := s.db.Ping(ctx); err != nil {
		return http.StatusServiceUnavailable, "storage", map[string]string{
			"failingSubsystem": "storage",
			"error":            err.Error(),
		}
	}

	// Check current-state reads
	_, err := s.db.Exec(ctx, "SELECT 1 FROM projects LIMIT 1")
	if err != nil {
		return http.StatusServiceUnavailable, "currentState", map[string]string{
			"failingSubsystem": "currentState",
			"error":            err.Error(),
		}
	}

	// Check audit event persistence
	_, err = s.db.Exec(ctx, "SELECT 1 FROM audit_events LIMIT 1")
	if err != nil {
		return http.StatusServiceUnavailable, "auditPersistence", map[string]string{
			"failingSubsystem": "auditPersistence",
			"error":            err.Error(),
		}
	}

	// Check webhook queue availability (NOT delivery — webhook receiver down does NOT cause 503)
	if !s.webhookOk() {
		return http.StatusServiceUnavailable, "webhookQueue", map[string]string{
			"failingSubsystem": "webhookQueue",
			"error":           "webhook queue unavailable",
		}
	}

	return http.StatusOK, "", nil
}

func newID(prefix string) string {
	return prefix + "_" + fmt.Sprintf("%d", time.Now().UnixNano())[:12]
}