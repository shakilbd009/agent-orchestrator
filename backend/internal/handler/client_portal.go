package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/agent-orchestrator/backend/internal/middleware"
	"github.com/agent-orchestrator/backend/internal/models"
	"github.com/agent-orchestrator/backend/internal/observability"
	"github.com/agent-orchestrator/backend/internal/service"
	"github.com/agent-orchestrator/backend/internal/util"
)

// ClientPortalHandler holds all client portal HTTP handlers.
type ClientPortalHandler struct {
	svc     *service.ClientPortalService
	metrics *observability.Metrics
	logger  *observability.Logger
}

// NewClientPortalHandler creates a new client portal handler with observability.
func NewClientPortalHandler(svc *service.ClientPortalService, metrics *observability.Metrics, logger *observability.Logger) *ClientPortalHandler {
	return &ClientPortalHandler{
		svc:     svc,
		metrics: metrics,
		logger:  logger,
	}
}

// GetPortfolio handles GET /client-portal/portfolio
// Returns the portfolio summary for the authenticated client principal.
func (h *ClientPortalHandler) GetPortfolio(c echo.Context) error {
	ctx := c.Request().Context()
	actor := middleware.GetActor(c)

	start := time.Now()
	result, err := h.svc.GetPortfolio(ctx, actor.ID)
	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		h.metrics.RecordSubmissionFailed(ctx)
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "Internal server error",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		})
	}

	h.metrics.RecordPortfolioView(ctx)
	h.metrics.RecordPortfolioLoadDuration(ctx, durationMs)
	// PortfolioViewed log emitted by service layer (to capture real principal+project count)

	return c.JSON(http.StatusOK, result)
}

// GetProjectDetail handles GET /client-portal/projects/:projectId
// Returns the full project detail. Returns empty body (fail-closed) if unauthorized.
func (h *ClientPortalHandler) GetProjectDetail(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := c.Param("projectId")
	actor := middleware.GetActor(c)

	start := time.Now()
	result, err := h.svc.GetProjectDetail(ctx, projectID, actor.ID)
	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		h.metrics.RecordSubmissionFailed(ctx)
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "Internal server error",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		})
	}

	// Access check: if result is empty struct, access was denied
	if result.ID == "" {
		h.metrics.RecordAccessDenied(ctx)
		h.logger.AccessDenied(ctx, actor.ID, "project", projectID)
		return c.JSON(http.StatusOK, result) // fail-closed: empty body
	}

	h.metrics.RecordProjectView(ctx, projectID)
	h.metrics.RecordProjectLoadDuration(ctx, durationMs)

	return c.JSON(http.StatusOK, result)
}

// GetApprovalInbox handles GET /client-portal/approvals
// Returns pending approvals across all accessible projects for the authenticated principal.
func (h *ClientPortalHandler) GetApprovalInbox(c echo.Context) error {
	ctx := c.Request().Context()
	actor := middleware.GetActor(c)

	start := time.Now()
	result, err := h.svc.GetApprovalInbox(ctx, actor.ID)
	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		h.metrics.RecordSubmissionFailed(ctx)
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "Internal server error",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		})
	}

	h.metrics.RecordProjectView(ctx, "approval_inbox") // label for approval inbox view
	h.metrics.RecordProjectLoadDuration(ctx, durationMs)

	return c.JSON(http.StatusOK, result)
}

// DecideApproval handles POST /client-portal/approvals/:approvalId/decide
// Processes a client approval decision with state machine validation.
func (h *ClientPortalHandler) DecideApproval(c echo.Context) error {
	ctx := c.Request().Context()
	approvalID := c.Param("approvalId")
	actor := middleware.GetActor(c)

	var req models.ApprovalDecisionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/bad-request",
			Title:  "Bad request",
			Status: http.StatusBadRequest,
			Detail: err.Error(),
		})
	}

	result, err := h.svc.DecideApproval(ctx, approvalID, actor.ID, req)
	if err != nil {
		h.metrics.RecordSubmissionFailed(ctx)
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "Internal server error",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		})
	}

	if !result.Success {
		h.metrics.RecordSubmissionFailed(ctx)
		return c.JSON(http.StatusBadRequest, result)
	}

	// Note: client_portal.approval.submitted log + decision_outcome metric are
	// emitted by the service layer (the single authoritative source).
	// Service records the outcome counter for ALL outcomes including NMI.
	// Handler recordSubmissionFailed above covers failure path only.

	return c.JSON(http.StatusOK, result)
}

// Search handles GET /client-portal/search
// Performs client-safe search with forbidden content stripped.
func (h *ClientPortalHandler) Search(c echo.Context) error {
	ctx := c.Request().Context()
	actor := middleware.GetActor(c)

	query := c.QueryParam("q")
	healthFilter := c.QueryParam("health")
	statusFilter := c.QueryParam("status")

	start := time.Now()
	result, err := h.svc.SearchClientPortal(ctx, actor.ID, query, healthFilter, statusFilter)
	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		h.metrics.RecordSubmissionFailed(ctx)
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "Internal server error",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		})
	}

	h.metrics.RecordProjectView(ctx, "search")
	h.metrics.RecordProjectLoadDuration(ctx, durationMs)

	return c.JSON(http.StatusOK, result)
}

// StreamApprovalInbox handles GET /client-portal/stream
// SSE endpoint for real-time approval inbox updates per ADR-03-002.
func (h *ClientPortalHandler) StreamApprovalInbox(c echo.Context) error {
	ctx := c.Request().Context()
	actor := middleware.GetActor(c)
	projectID := c.QueryParam("projectId") // optional: filter by project

	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	// Emit SSE connected log event
	h.logger.SSEConnected(ctx, projectID)

	// Send initial connection event
	connectEvent := models.SSEClientEvent{
		EventType: models.SSEClientPortalSSEConnected,
		ProjectID: projectID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   map[string]interface{}{"principal": actor.ID},
	}
	envelope, _ := json.Marshal(connectEvent)
	fmt.Fprintf(c.Response(), "data: %s\n\n", envelope)

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		h.metrics.RecordSSEDisconnect(ctx)
		h.logger.SSEDisconnected(ctx, projectID, "flusher_unavailable")
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "Streaming unavailable",
			Status: http.StatusInternalServerError,
		})
	}
	flusher.Flush()

	// Heartbeat tick to keep connection alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			h.metrics.RecordSSEDisconnect(ctx)
			h.logger.SSEDisconnected(ctx, projectID, "context_cancelled")
			disconnectEvent := models.SSEClientEvent{
				EventType: models.SSEClientPortalSSEDisconnected,
				ProjectID: projectID,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Payload:   map[string]interface{}{"reason": "context_cancelled"},
			}
			env, _ := json.Marshal(disconnectEvent)
			fmt.Fprintf(c.Response(), "data: %s\n\n", env)
			return nil
		case <-ticker.C:
			// Keep-alive ping
			fmt.Fprintf(c.Response(), ": ping\n\n")
			flusher.Flush()
		}
	}
}

// ValidatePublication is the handler-level publication validation wrapper.
func ValidatePublication(c echo.Context) error {
	var req struct {
		Title   string   `json:"title"`
		Body    string   `json:"body"`
		OwnerID string   `json:"ownerId"`
		Tags    []string `json:"tags"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/bad-request",
			Title:  "Bad request",
			Status: http.StatusBadRequest,
			Detail: err.Error(),
		})
	}

	result := util.ValidatePublication(util.PublicationValidationInput{
		Title:   req.Title,
		Body:    req.Body,
		OwnerID: req.OwnerID,
		Tags:    req.Tags,
	})

	return c.JSON(http.StatusOK, result)
}

// CreateComment handles POST /client-portal/comments
// Creates a new client-visible comment.
func (h *ClientPortalHandler) CreateComment(c echo.Context) error {
	ctx := c.Request().Context()
	actor := middleware.GetActor(c)

	var req struct {
		ProjectID    string `json:"projectId"`
		RelatedItemID string `json:"relatedItemId"`
		Body         string `json:"body"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/bad-request",
			Title:  "Bad request",
			Status: http.StatusBadRequest,
			Detail: err.Error(),
		})
	}

	if strings.TrimSpace(req.Body) == "" {
		return c.JSON(http.StatusBadRequest, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/bad-request",
			Title:  "Bad request",
			Status: http.StatusBadRequest,
			Detail: "body required",
		})
	}

	comment := &models.ClientComment{
		ID:            fmt.Sprintf("cmt_%d", time.Now().UnixNano()),
		ProjectID:     req.ProjectID,
		RelatedItemID: req.RelatedItemID,
		AuthorName:    actor.ID,
		Body:          req.Body,
		CreatedAt:     time.Now().UTC(),
	}

	if err := h.svc.CreateComment(ctx, comment); err != nil {
		h.metrics.RecordSubmissionFailed(ctx)
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "Internal server error",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		})
	}
	// CommentCreated metric + log emitted by service layer

	return c.JSON(http.StatusCreated, comment)
}

// EditComment handles PUT /client-portal/comments/:commentId
// Edits an existing comment body.
func (h *ClientPortalHandler) EditComment(c echo.Context) error {
	ctx := c.Request().Context()
	actor := middleware.GetActor(c)
	commentID := c.Param("commentId")

	var req struct {
		Body string `json:"body"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/bad-request",
			Title:  "Bad request",
			Status: http.StatusBadRequest,
			Detail: err.Error(),
		})
	}

	if strings.TrimSpace(req.Body) == "" {
		return c.JSON(http.StatusBadRequest, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/bad-request",
			Title:  "Bad request",
			Status: http.StatusBadRequest,
			Detail: "body required",
		})
	}

	updated, err := h.svc.EditComment(ctx, commentID, actor.ID, req.Body)
	if err != nil {
		h.metrics.RecordSubmissionFailed(ctx)
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "Internal server error",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		})
	}
	if updated == nil {
		return c.JSON(http.StatusNotFound, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/not-found",
			Title:  "Not found",
			Status: http.StatusNotFound,
		})
	}
	// CommentEdited metric + log emitted by service layer

	return c.JSON(http.StatusOK, updated)
}

// DeleteComment handles DELETE /client-portal/comments/:commentId
// Deletes a comment.
func (h *ClientPortalHandler) DeleteComment(c echo.Context) error {
	ctx := c.Request().Context()
	actor := middleware.GetActor(c)
	commentID := c.Param("commentId")

	deleted, err := h.svc.DeleteComment(ctx, commentID, actor.ID)
	if err != nil {
		h.metrics.RecordSubmissionFailed(ctx)
		return c.JSON(http.StatusInternalServerError, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/internal",
			Title:  "Internal server error",
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		})
	}
	if !deleted {
		return c.JSON(http.StatusNotFound, models.Error{
			Type:   "https://api.agentorchestrator.example.com/errors/not-found",
			Title:  "Not found",
			Status: http.StatusNotFound,
		})
	}
	// CommentDeleted metric + log emitted by service layer

	return c.NoContent(http.StatusNoContent)
}