package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/agent-orchestrator/backend/internal/event"
	"github.com/agent-orchestrator/backend/internal/handler"
	appmiddleware "github.com/agent-orchestrator/backend/internal/middleware"
	"github.com/agent-orchestrator/backend/internal/repository"
	"github.com/agent-orchestrator/backend/internal/service"
)

// readyPoolAdapter adapts *repository.Pool to service.ReadyPool so the concrete DB
// implementation satisfies the service-layer interface without leaking repository
// types into the service package.
type readyPoolAdapter struct {
	db *repository.Pool
}

func (a readyPoolAdapter) Ping(ctx context.Context) error {
	return a.db.Ping(ctx)
}

func (a readyPoolAdapter) Exec(ctx context.Context, query string) (interface{}, error) {
	return a.db.Exec(ctx, query)
}

var version = "dev"

func main() {
	ctx := context.Background()

	// Database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	dbPool, err := repository.NewPool(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Run migrations
	if err := runMigrations(ctx, dbPool); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// Initialize repositories
	projectRepo := repository.NewProjectRepository(dbPool)
	taskRepo := repository.NewTaskRepository(dbPool)
	gateRepo := repository.NewGateRepository(dbPool)
	decompRepo := repository.NewDecompositionRepository(dbPool)
	webhookRepo := repository.NewWebhookRepository(dbPool)

	// Initialize event system
	webhookQueue := event.NewWebhookQueue()
	webhookQueue.SetWebhookRepo(webhookRepo)
	eventSvc := event.NewEventService(webhookQueue)

	// Wire audit emitter for auth.mutation.denied events (NFR-02-014)
	appmiddleware.SetAuditEmitter(webhookRepo.InsertAuthDeniedEvent)

	// Start webhook delivery worker (background, non-blocking)
	go runWebhookWorker(ctx, webhookQueue, webhookRepo)

	// Initialize services
	projectSvc := service.NewProjectService(projectRepo, eventSvc)
	taskSvc := service.NewTaskService(taskRepo, projectRepo, eventSvc)
	gateSvc := service.NewGateService(gateRepo, eventSvc)
	decompSvc := service.NewDecompositionService(decompRepo, taskRepo, eventSvc)
	webhookSvc := service.NewWebhookService(webhookRepo, eventSvc)
	readySvc := service.NewReadyService(readyPoolAdapter{dbPool}, func() bool {
		// Webhook queue is always available (in-process/DB-backed)
		return true
	})

	// Initialize handlers
	h := handler.NewHandlers(projectSvc, taskSvc, gateSvc, decompSvc, webhookSvc, readySvc, eventSvc, projectRepo)

	// Initialize Echo
	e := echo.New()
	e.HideBanner = true

	// CORS for BRD-02 SSE (ADR-02-006)
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:5173", "http://localhost:3000"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, "X-Actor-ID", "X-Actor-Role", "Last-Event-ID"},
	}))

	// Recover middleware
	e.Use(middleware.Recover())

	// Request ID middleware
	e.Use(middleware.RequestID())

	// Feature flag loading from environment
	loadFeatureFlags()

	// Health endpoints (no auth required)
	e.GET("/health", h.Health)
	e.GET("/ready", h.Ready)
	e.GET("/live", h.Live)

	// Project-scoped orchestration endpoints (BRD-02)
	projects := e.Group("/projects")
	projects.Use(appmiddleware.ActorMiddleware())
	projects.Use(appmiddleware.RequireFeatureGate("platform-orchestration"))

	// Project CRUD
	projects.GET("", h.ListProjects)
	projects.POST("", h.CreateProject)
	projects.GET("/:projectId", h.GetProject)
	projects.PATCH("/:projectId", h.UpdateProject)
	projects.DELETE("/:projectId", h.DeleteProject)

	// Task CRUD (project-scoped)
	projects.GET("/:projectId/tasks", h.ListProjectTasks)
	projects.POST("/:projectId/tasks", h.CreateProjectTask)
	projects.GET("/:projectId/tasks/:taskId", h.GetProjectTask)
	projects.POST("/:projectId/tasks/:taskId/block", h.BlockProjectTask)
	projects.POST("/:projectId/tasks/:taskId/complete", h.CompleteProjectTask)

	// Task dependencies
	projects.GET("/:projectId/tasks/:taskId/dependencies", h.GetTaskDependencies)

	// Task gates
	projects.POST("/:projectId/tasks/:taskId/gate", h.CreateTaskGate)
	projects.GET("/:projectId/tasks/:taskId/gate/:gateId", h.GetTaskGate)
	projects.PATCH("/:projectId/tasks/:taskId/gate/:gateId", h.UpdateTaskGate)

	// Decomposition
	projects.POST("/:projectId/tasks/:taskId/decomposition", h.ProposeDecomposition)

	// Phase gates
	projects.GET("/:projectId/phase-gates", h.ListProjectPhaseGates)
	projects.POST("/:projectId/phase-gates", h.CreateProjectPhaseGate)
	projects.GET("/:projectId/phase-gates/:gateId", h.GetProjectPhaseGate)
	projects.PATCH("/:projectId/phase-gates/:gateId", h.UpdateProjectPhaseGate)

	// SSE stream (FR-02-023, ADR-02-006)
	projects.GET("/:projectId/events/stream", h.StreamProjectEvents)

	// Webhooks
	projects.GET("/:projectId/webhooks", h.ListProjectWebhooks)
	projects.POST("/:projectId/webhooks", h.RegisterProjectWebhook)
	projects.DELETE("/:projectId/webhooks/:webhookId", h.DeleteProjectWebhook)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	log.Printf("starting server on :%s (version %s)", port, version)
	if err := e.Start(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func loadFeatureFlags() {
	// Load feature flags from environment variables
	// FF_ENABLE_PLATFORM_ORCHESTRATION=true enables the master gate
	if os.Getenv("FF_ENABLE_PLATFORM_ORCHESTRATION") == "true" {
		appmiddleware.FeatureFlags.PlatformOrchestration = true
	}
	if os.Getenv("FF_ENABLE_LAYER_A_AGENTS") == "true" {
		appmiddleware.FeatureFlags.LayerAAgents = true
	}
	if os.Getenv("FF_ENABLE_LAYER_B_AGENTS") == "true" {
		appmiddleware.FeatureFlags.LayerBAgents = true
	}
	if os.Getenv("FF_ENABLE_HUMAN_GATES") == "true" {
		appmiddleware.FeatureFlags.HumanGates = true
	}
	if os.Getenv("FF_ENABLE_AUDIT_TRAIL") == "true" {
		appmiddleware.FeatureFlags.AuditTrail = true
	}
}

func runMigrations(ctx context.Context, db *repository.Pool) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS projects (id TEXT PRIMARY KEY, name TEXT NOT NULL, description TEXT DEFAULT '', owner TEXT DEFAULT '', status TEXT NOT NULL DEFAULT 'active', phase TEXT NOT NULL DEFAULT 'planning', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status)`,
		`CREATE TABLE IF NOT EXISTS orchestration_tasks (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, title TEXT NOT NULL, body TEXT DEFAULT '', status TEXT NOT NULL DEFAULT 'todo', layer TEXT NOT NULL, assignee TEXT DEFAULT '', required BOOLEAN NOT NULL DEFAULT false, priority INTEGER NOT NULL DEFAULT 0, stale BOOLEAN NOT NULL DEFAULT false, stale_threshold_minutes INTEGER, blocked_reason TEXT DEFAULT '', workspace_kind TEXT DEFAULT '', workspace_path TEXT DEFAULT '', tags TEXT[] DEFAULT '{}', metadata JSONB DEFAULT '{}', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), completed_at TIMESTAMPTZ)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON orchestration_tasks(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON orchestration_tasks(project_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_execution_status ON orchestration_tasks(project_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_parent_task_id ON orchestration_tasks(project_id)`,
		`CREATE TABLE IF NOT EXISTS task_parents (task_id TEXT NOT NULL, parent_id TEXT NOT NULL, PRIMARY KEY (task_id, parent_id))`,
		`CREATE TABLE IF NOT EXISTS task_children (task_id TEXT NOT NULL, child_id TEXT NOT NULL, PRIMARY KEY (task_id, child_id))`,
		`CREATE TABLE IF NOT EXISTS task_dependencies (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, source_task_id TEXT NOT NULL, target_task_id TEXT NOT NULL, type TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_deps_project_id ON task_dependencies(project_id)`,
		`CREATE TABLE IF NOT EXISTS task_gates (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, task_id TEXT NOT NULL, phase TEXT NOT NULL, state TEXT NOT NULL DEFAULT 'open', criteria TEXT[] DEFAULT '{}', blocking BOOLEAN NOT NULL DEFAULT true, passed_at TIMESTAMPTZ, passed_by TEXT DEFAULT '', override_note TEXT DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_gates_project_id ON task_gates(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_gates_task_id ON task_gates(task_id)`,
		`CREATE TABLE IF NOT EXISTS project_phase_gates (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, phase_index INTEGER NOT NULL, phase TEXT NOT NULL, state TEXT NOT NULL DEFAULT 'open', criteria TEXT[] DEFAULT '{}', pass_condition TEXT DEFAULT '', passed_at TIMESTAMPTZ, passed_by TEXT DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE(project_id, phase_index))`,
		`CREATE TABLE IF NOT EXISTS decomposition_proposals (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, parent_task_id TEXT NOT NULL, submitter TEXT NOT NULL, state TEXT NOT NULL DEFAULT 'submitted', proposed_tasks JSONB NOT NULL DEFAULT '[]', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_decomp_parent ON decomposition_proposals(parent_task_id)`,
		`CREATE TABLE IF NOT EXISTS webhook_registrations (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, url TEXT NOT NULL, events TEXT[] NOT NULL DEFAULT '{}', active BOOLEAN NOT NULL DEFAULT true, secret_hash TEXT NOT NULL, delivery_last_attempt_at TIMESTAMPTZ, delivery_last_success_at TIMESTAMPTZ, delivery_failure_count INTEGER NOT NULL DEFAULT 0, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_webhooks_project_id ON webhook_registrations(project_id)`,
		`CREATE TABLE IF NOT EXISTS handoff_evidence (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, task_id TEXT NOT NULL, from_agent TEXT NOT NULL, to_agent TEXT DEFAULT '', summary TEXT NOT NULL, artifacts TEXT[] DEFAULT '{}', validation_performed TEXT DEFAULT '', risks_or_residual_issues TEXT DEFAULT '', recommended_next_gate TEXT DEFAULT '', metadata JSONB DEFAULT '{}', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_handoff_project ON handoff_evidence(project_id)`,
		`CREATE TABLE IF NOT EXISTS audit_events (event_id TEXT PRIMARY KEY, schema_version TEXT NOT NULL DEFAULT 'v1alpha', project_id TEXT NOT NULL, topic TEXT NOT NULL, actor_id TEXT NOT NULL, actor_role TEXT NOT NULL, task_id TEXT, parent_task_id TEXT, gate_id TEXT, timestamp TIMESTAMPTZ NOT NULL, payload JSONB NOT NULL DEFAULT '{}')`,
		`CREATE INDEX IF NOT EXISTS idx_audit_project_id ON audit_events(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_events(project_id, timestamp)`,
		`CREATE TABLE IF NOT EXISTS feature_flags (name TEXT PRIMARY KEY, enabled BOOLEAN NOT NULL DEFAULT false, updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE TABLE IF NOT EXISTS sse_clients (client_id TEXT PRIMARY KEY, project_id TEXT NOT NULL, remote_addr TEXT DEFAULT '', connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), last_event_id TEXT DEFAULT '')`,
		`CREATE INDEX IF NOT EXISTS idx_sse_project ON sse_clients(project_id)`,
		`CREATE TABLE IF NOT EXISTS webhook_delivery_queue (id SERIAL PRIMARY KEY, webhook_id TEXT NOT NULL, event_id TEXT NOT NULL, payload JSONB NOT NULL, attempt_count INTEGER NOT NULL DEFAULT 0, next_retry_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`,
		`CREATE INDEX IF NOT EXISTS idx_wdq_next_retry ON webhook_delivery_queue(next_retry_at)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(ctx, m); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

// runWebhookWorker processes the webhook delivery queue in the background.
func runWebhookWorker(ctx context.Context, queue *event.WebhookQueue, repo *repository.WebhookRepository) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			queue.ProcessQueue(ctx)
		}
	}
}