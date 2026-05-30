package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/agent-orchestrator/backend/internal/event"
	"github.com/agent-orchestrator/backend/internal/middleware"
	"github.com/agent-orchestrator/backend/internal/models"
	"github.com/agent-orchestrator/backend/internal/repository"
	"github.com/agent-orchestrator/backend/internal/service"
)

// auditEventQuerier abstracts the GetAuditEventsAfter method for SSE replay.
type auditEventQuerier interface {
	GetAuditEventsAfter(ctx context.Context, projectID, lastEventID string) ([]models.AuditEvent, error)
}

// Handlers holds all HTTP handlers for BRD-02 orchestration endpoints.
type Handlers struct {
	projectSvc      *service.ProjectService
	taskSvc         *service.TaskService
	gateSvc         *service.GateService
	decompSvc       *service.DecompositionService
	webhookSvc      *service.WebhookService
	readySvc        *service.ReadyService
	eventSvc        *event.EventService
	projectRepo     *repository.ProjectRepository
	auditQuerier    auditEventQuerier
}

func (h *Handlers) withAuditQuerier(q auditEventQuerier) *Handlers {
	h.auditQuerier = q
	return h
}

// NewHandlers creates a new handlers instance.
func NewHandlers(
	projectSvc *service.ProjectService,
	taskSvc *service.TaskService,
	gateSvc *service.GateService,
	decompSvc *service.DecompositionService,
	webhookSvc *service.WebhookService,
	readySvc *service.ReadyService,
	eventSvc *event.EventService,
	projectRepo *repository.ProjectRepository,
) *Handlers {
	return &Handlers{
		projectSvc: projectSvc,
		taskSvc:    taskSvc,
		gateSvc:    gateSvc,
		decompSvc:  decompSvc,
		webhookSvc: webhookSvc,
		readySvc:   readySvc,
		eventSvc:   eventSvc,
		projectRepo: projectRepo,
		auditQuerier: projectRepo,
	}
}

// Project handlers

func (h *Handlers) ListProjects(c echo.Context) error {
	ctx := c.Request().Context()
	projects, err := h.projectSvc.ListProjects(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "Internal server error",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"projects": projects})
}

func (h *Handlers) CreateProject(c echo.Context) error {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Owner       string `json:"owner,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/bad-request",
			Title:  "Bad request",
			Status: http.StatusBadRequest,
			Detail: err.Error(),
		})
	}

	actor := middleware.GetActor(c)
	project := &models.Project{
		Name:        req.Name,
		Description: req.Description,
		Owner:       actor.ID,
	}

	result, err := h.projectSvc.CreateProject(c.Request().Context(), project, actor)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "Internal server error",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		})
	}
	return c.JSON(http.StatusCreated, result)
}

func (h *Handlers) GetProject(c echo.Context) error {
	projectID := c.Param("projectId")
	ctx := c.Request().Context()

	project, err := h.projectSvc.GetProject(ctx, projectID)
	if err != nil {
		return c.JSON(http.StatusNotFound, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/not-found",
			Title:  "Project not found",
			Status: http.StatusNotFound,
		})
	}
	return c.JSON(http.StatusOK, project)
}

func (h *Handlers) UpdateProject(c echo.Context) error {
	projectID := c.Param("projectId")
	var req models.ProjectUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/bad-request",
			Title:  "Bad request",
			Status: http.StatusBadRequest,
		})
	}

	project := &models.Project{
		ID:          projectID,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.projectSvc.UpdateProject(c.Request().Context(), projectID, project); err != nil {
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "Internal server error",
			Status: http.StatusInternalServerError,
		})
	}
	return c.JSON(http.StatusOK, project)
}

func (h *Handlers) DeleteProject(c echo.Context) error {
	projectID := c.Param("projectId")
	if err := h.projectSvc.DeleteProject(c.Request().Context(), projectID); err != nil {
		return c.JSON(http.StatusConflict, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/conflict",
			Title:  "Cannot delete project",
			Status: http.StatusConflict,
			Detail: err.Error(),
		})
	}
	return c.NoContent(http.StatusNoContent)
}

// Task handlers

func (h *Handlers) ListProjectTasks(c echo.Context) error {
	projectID := c.Param("projectId")
	status := c.QueryParam("status")
	layer := c.QueryParam("layer")
	assignee := c.QueryParam("assignee")

	tasks, err := h.taskSvc.ListTasks(c.Request().Context(), projectID, status, layer, assignee)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "Internal server error",
			Status: http.StatusInternalServerError,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"tasks": tasks})
}

func (h *Handlers) CreateProjectTask(c echo.Context) error {
	projectID := c.Param("projectId")
	var req models.OrchestrationTaskCreateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/bad-request",
			Title:  "Bad request",
			Status: http.StatusBadRequest,
		})
	}

	actor := middleware.GetActor(c)
	task, err := h.taskSvc.CreateTask(c.Request().Context(), projectID, &req, actor)
	if err != nil {
		return c.JSON(http.StatusForbidden, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/forbidden",
			Title:  "Forbidden",
			Status: http.StatusForbidden,
			Detail: err.Error(),
		})
	}
	return c.JSON(http.StatusCreated, task)
}

func (h *Handlers) GetProjectTask(c echo.Context) error {
	projectID := c.Param("projectId")
	taskID := c.Param("taskId")

	task, err := h.taskSvc.GetTask(c.Request().Context(), projectID, taskID)
	if err != nil {
		return c.JSON(http.StatusNotFound, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/not-found",
			Title:  "Task not found",
			Status: http.StatusNotFound,
		})
	}
	return c.JSON(http.StatusOK, task)
}

func (h *Handlers) BlockProjectTask(c echo.Context) error {
	projectID := c.Param("projectId")
	taskID := c.Param("taskId")

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/bad-request",
			Title:  "Bad request",
			Status: http.StatusBadRequest,
		})
	}

	actor := middleware.GetActor(c)
	if err := h.taskSvc.BlockTask(c.Request().Context(), projectID, taskID, req.Reason, actor); err != nil {
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "Internal server error",
			Status: http.StatusInternalServerError,
		})
	}

	task, _ := h.taskSvc.GetTask(c.Request().Context(), projectID, taskID)
	return c.JSON(http.StatusOK, task)
}

func (h *Handlers) CompleteProjectTask(c echo.Context) error {
	projectID := c.Param("projectId")
	taskID := c.Param("taskId")

	var req models.TaskCompleteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/bad-request",
			Title:  "Bad request",
			Status: http.StatusBadRequest,
		})
	}

	actor := middleware.GetActor(c)
	handoff := &models.HandoffEvidence{
		Summary:               req.Handoff.Summary,
		Artifacts:             req.Handoff.Artifacts,
		ValidationPerformed:   req.Handoff.ValidationPerformed,
		RisksOrResidualIssues: req.Handoff.RisksOrResidualIssues,
		RecommendedNextGate:   req.Handoff.RecommendedNextGate,
	}

	if err := h.taskSvc.CompleteTask(c.Request().Context(), projectID, taskID, handoff, actor); err != nil {
		return c.JSON(http.StatusConflict, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/conflict",
			Title:  "Cannot complete task",
			Status: http.StatusConflict,
			Detail: err.Error(),
		})
	}

	task, _ := h.taskSvc.GetTask(c.Request().Context(), projectID, taskID)
	return c.JSON(http.StatusOK, task)
}

// Task dependency handlers

func (h *Handlers) GetTaskDependencies(c echo.Context) error {
	projectID := c.Param("projectId")
	taskID := c.Param("taskId")

	task, err := h.taskSvc.GetTask(c.Request().Context(), projectID, taskID)
	if err != nil {
		return c.JSON(http.StatusNotFound, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/not-found",
			Title:  "Task not found",
			Status: http.StatusNotFound,
		})
	}

	// Return dependency graph
	return c.JSON(http.StatusOK, map[string]any{
		"taskId":  taskID,
		"parents": task.Parents,
		"children": task.Children,
	})
}

// Gate handlers

func (h *Handlers) CreateTaskGate(c echo.Context) error {
	projectID := c.Param("projectId")
	taskID := c.Param("taskId")

	var req models.TaskGateCreateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/bad-request",
			Title:  "Bad request",
			Status: http.StatusBadRequest,
		})
	}

	actor := middleware.GetActor(c)
	gate, err := h.gateSvc.CreateTaskGate(c.Request().Context(), projectID, taskID, &req, actor)
	if err != nil {
		return c.JSON(http.StatusForbidden, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/forbidden",
			Title:  "Forbidden",
			Status: http.StatusForbidden,
			Detail: err.Error(),
		})
	}
	return c.JSON(http.StatusCreated, gate)
}

func (h *Handlers) GetTaskGate(c echo.Context) error {
	projectID := c.Param("projectId")
	taskID := c.Param("taskId")
	gateID := c.Param("gateId")

	gate, err := h.gateSvc.GetTaskGate(c.Request().Context(), projectID, taskID, gateID)
	if err != nil {
		return c.JSON(http.StatusNotFound, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/not-found",
			Title:  "Gate not found",
			Status: http.StatusNotFound,
		})
	}
	return c.JSON(http.StatusOK, gate)
}

func (h *Handlers) UpdateTaskGate(c echo.Context) error {
	projectID := c.Param("projectId")
	taskID := c.Param("taskId")
	gateID := c.Param("gateId")

	var req struct {
		State        string `json:"state"` // open, passed, blocked
		OverrideNote string `json:"overrideNote,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/bad-request",
			Title:  "Bad request",
			Status: http.StatusBadRequest,
		})
	}

	actor := middleware.GetActor(c)
	var err error
	if req.State == "passed" {
		err = h.gateSvc.ApproveGate(c.Request().Context(), projectID, taskID, gateID, actor, req.OverrideNote)
	} else if req.State == "blocked" {
		err = h.gateSvc.RejectGate(c.Request().Context(), projectID, taskID, gateID, actor, req.OverrideNote)
	}

	if err != nil {
		return c.JSON(http.StatusForbidden, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/forbidden",
			Title:  "Forbidden",
			Status: http.StatusForbidden,
			Detail: err.Error(),
		})
	}

	gate, _ := h.gateSvc.GetTaskGate(c.Request().Context(), projectID, taskID, gateID)
	return c.JSON(http.StatusOK, gate)
}

// Phase gate handlers

func (h *Handlers) ListProjectPhaseGates(c echo.Context) error {
	projectID := c.Param("projectId")
	ctx := c.Request().Context()

	gates, err := h.gateSvc.ListProjectPhaseGates(ctx, projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "Internal server error",
			Status: http.StatusInternalServerError,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"gates": gates})
}

func (h *Handlers) CreateProjectPhaseGate(c echo.Context) error {
	projectID := c.Param("projectId")

	var req struct {
		PhaseIndex   int      `json:"phaseIndex"`
		Phase        string   `json:"phase"`
		Criteria     []string `json:"criteria,omitempty"`
		PassCondition string  `json:"passCondition,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/bad-request",
			Title:  "Bad request",
			Status: http.StatusBadRequest,
		})
	}

	gate := &models.ProjectPhaseGate{
		ID:            fmt.Sprintf("pg_%d", time.Now().UnixNano()),
		ProjectID:     projectID,
		PhaseIndex:    req.PhaseIndex,
		Phase:         req.Phase,
		State:         "open",
		Criteria:      req.Criteria,
		PassCondition: req.PassCondition,
		CreatedAt:     time.Now(),
	}

	if err := h.gateSvc.CreateProjectPhaseGate(c.Request().Context(), gate); err != nil {
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "Internal server error",
			Status: http.StatusInternalServerError,
		})
	}
	return c.JSON(http.StatusCreated, gate)
}

func (h *Handlers) GetProjectPhaseGate(c echo.Context) error {
	projectID := c.Param("projectId")
	gateID := c.Param("gateId")

	gate, err := h.gateSvc.GetProjectPhaseGate(c.Request().Context(), projectID, gateID)
	if err != nil {
		return c.JSON(http.StatusNotFound, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/not-found",
			Title:  "Gate not found",
			Status: http.StatusNotFound,
		})
	}
	return c.JSON(http.StatusOK, gate)
}

func (h *Handlers) UpdateProjectPhaseGate(c echo.Context) error {
	projectID := c.Param("projectId")
	gateID := c.Param("gateId")

	var req struct {
		State     string `json:"state"`
		PassedBy  string `json:"passedBy,omitempty"`
		OverrideNote string `json:"overrideNote,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/bad-request",
			Title:  "Bad request",
			Status: http.StatusBadRequest,
		})
	}

	actor := middleware.GetActor(c)

	// AC-02-017: Project-level phase gates require human authorization.
	// Only human actors may advance phase gates to state=passed.
	// layer_a without delegation is not permitted per FR-02-013 / BRD-02.
	if req.State == "passed" {
		if actor.Role != "human" {
			middleware.EmitAuthDenied(c, actor.ID, actor.Role, "phase_gate_authorization", "human role required for phase gate advancement")
			return c.JSON(http.StatusForbidden, models.Error{
				Type:   "https://api.agentorchestrator.example.com/errors/forbidden",
				Title:  "Forbidden",
				Status: http.StatusForbidden,
				Detail: "Project-level phase gate advancement requires human authorization.",
			})
		}
	}

	gate, err := h.gateSvc.GetProjectPhaseGate(c.Request().Context(), projectID, gateID)
	if err != nil {
		return c.JSON(http.StatusNotFound, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/not-found",
			Title:  "Gate not found",
			Status: http.StatusNotFound,
		})
	}

	gate.State = req.State
	if req.State == "passed" {
		now := time.Now()
		gate.PassedAt = &now
		gate.PassedBy = actor.ID
	}

	if err := h.gateSvc.UpdateProjectPhaseGate(c.Request().Context(), gate); err != nil {
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "Internal server error",
			Status: http.StatusInternalServerError,
		})
	}
	return c.JSON(http.StatusOK, gate)
}

// Decomposition handlers

func (h *Handlers) ProposeDecomposition(c echo.Context) error {
	projectID := c.Param("projectId")
	taskID := c.Param("taskId")

	var req models.DecompositionProposalSubmitRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/bad-request",
			Title:  "Bad request",
			Status: http.StatusBadRequest,
		})
	}

	actor := middleware.GetActor(c)
	proposal, err := h.decompSvc.ProposeDecomposition(c.Request().Context(), projectID, taskID, &req, actor)
	if err != nil {
		return c.JSON(http.StatusForbidden, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/forbidden",
			Title:  "Forbidden",
			Status: http.StatusForbidden,
			Detail: err.Error(),
		})
	}
	return c.JSON(http.StatusCreated, proposal)
}

// SSE stream handler (FR-02-023, ADR-02-006)
func (h *Handlers) StreamProjectEvents(c echo.Context) error {
	projectID := c.Param("projectId")
	lastEventID := c.Request().Header.Get("Last-Event-ID")

	ctx := c.Request().Context()
	fanout := h.eventSvc.GetFanout()

	// Generate client ID
	clientID := fmt.Sprintf("sse_%d", time.Now().UnixNano())

	// Check project SSE limit (NFR-02-005: max 50 concurrent clients)
	client, err := fanout.AddClient(clientID, projectID, lastEventID)
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/.max-sse-clients",
			Title:  "Max SSE clients reached",
			Status: http.StatusServiceUnavailable,
			Detail: "Maximum 50 concurrent SSE clients per project",
		})
	}
	defer fanout.RemoveClient(clientID)

	// Set SSE headers
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("X-Accel-Buffering", "no")

	// CORS headers for SSE (ADR-02-006)
	c.Response().Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
	c.Response().Header().Set("Access-Control-Allow-Headers", "Last-Event-ID")

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "SSE not supported",
			Status: http.StatusInternalServerError,
		})
	}

	// AC-02-022A: Replay missed events from audit log when Last-Event-ID is provided.
	// Query events where project_id = projectID AND event_id > lastEventID.
	if lastEventID != "" {
		historicalEvents, err := h.auditQuerier.GetAuditEventsAfter(ctx, projectID, lastEventID)
		if err != nil {
			log.Printf("SSE replay: failed to fetch audit events after %s: %v", lastEventID, err)
			// Non-fatal: fall through to live fanout; client can recover on next reconnect.
		} else {
			for _, ev := range historicalEvents {
				data, err := json.Marshal(ev)
				if err != nil {
					continue
				}
				// SSE format: event: <topic>\nid: <eventId>\ndata: <json>\n\n
				fmt.Fprintf(c.Response().Writer, "event: %s\nid: %s\ndata: %s\n\n", ev.Topic, ev.EventID, string(data))
				flusher.Flush()
			}
		}
	}

	// Send initial comment to keep connection alive
	c.Response().Write([]byte(":ok\n\n"))
	flusher.Flush()

	// Stream events until client disconnects
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case data, ok := <-client.EventCh:
			if !ok {
				return nil
			}

			// SSE format: event:<topic>\nid:<eventId>\ndata:<json>\n\n
			var event map[string]any
			if err := json.Unmarshal(data, &event); err != nil {
				continue
			}

			topic, _ := event["topic"].(string)
			eventID, _ := event["eventId"].(string)

			fmt.Fprintf(c.Response().Writer, "event: %s\nid: %s\ndata: %s\n\n", topic, eventID, string(data))
			flusher.Flush()

		case <-ticker.C:
			// Keep-alive comment
			c.Response().Write([]byte(": keepalive\n\n"))
			flusher.Flush()
		}
	}
}

// Webhook handlers

func (h *Handlers) ListProjectWebhooks(c echo.Context) error {
	projectID := c.Param("projectId")
	webhooks, err := h.webhookSvc.ListWebhooks(c.Request().Context(), projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "Internal server error",
			Status: http.StatusInternalServerError,
		})
	}
	// Strip secrets from response (ADR-02-003)
	for i := range webhooks {
		webhooks[i].Secret = ""
	}
	return c.JSON(http.StatusOK, map[string]any{"webhooks": webhooks})
}

func (h *Handlers) RegisterProjectWebhook(c echo.Context) error {
	projectID := c.Param("projectId")

	var req models.WebhookRegistrationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/bad-request",
			Title:  "Bad request",
			Status: http.StatusBadRequest,
		})
	}

	actor := middleware.GetActor(c)
	webhook, err := h.webhookSvc.RegisterWebhook(c.Request().Context(), projectID, &req, actor)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/bad-request",
			Title:  "Bad request",
			Status: http.StatusBadRequest,
			Detail: err.Error(),
		})
	}
	webhook.Secret = "" // Never return secret
	return c.JSON(http.StatusCreated, webhook)
}

func (h *Handlers) DeleteProjectWebhook(c echo.Context) error {
	webhookID := c.Param("webhookId")
	if err := h.webhookSvc.DeleteWebhook(c.Request().Context(), webhookID); err != nil {
		return c.JSON(http.StatusNotFound, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/not-found",
			Title:  "Webhook not found",
			Status: http.StatusNotFound,
		})
	}
	return c.NoContent(http.StatusNoContent)
}

// Health handlers

func (h *Handlers) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":    "ok",
		"version":   "0.1.0",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (h *Handlers) Ready(c echo.Context) error {
	ctx := c.Request().Context()
	status, subsystem, body := h.readySvc.CheckReady(ctx)

	if status != http.StatusOK {
		return c.JSON(status, map[string]string{
			"failingSubsystem": subsystem,
			"error":            body["error"],
		})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handlers) Live(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "alive"})
}