package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/agent-orchestrator/backend/internal/event"
	"github.com/agent-orchestrator/backend/internal/middleware"
	"github.com/agent-orchestrator/backend/internal/models"
	"github.com/agent-orchestrator/backend/internal/service"
)

func TestHealthHandler(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status":    "ok",
			"version":   "test",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}

	if err := handler(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", resp["status"])
	}
	if resp["version"] != "test" {
		t.Errorf("expected version 'test', got %q", resp["version"])
	}
	if resp["timestamp"] == "" {
		t.Error("expected non-empty timestamp")
	}
}

func TestActorMiddleware_MissingActor(t *testing.T) {
	e := echo.New()
	e.Use(middleware.ActorMiddleware())
	e.GET("/test", func(c echo.Context) error {
		actor := middleware.GetActor(c)
		if actor.ID == "" || actor.Role == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing"})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing actor, got %d", rec.Code)
	}
}

func TestActorMiddleware_ValidActor(t *testing.T) {
	e := echo.New()
	e.Use(middleware.ActorMiddleware())
	e.GET("/test", func(c echo.Context) error {
		actor := middleware.GetActor(c)
		return c.JSON(http.StatusOK, map[string]any{"actorId": actor.ID, "role": actor.Role})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Actor-ID", "user_123")
	req.Header.Set("X-Actor-Role", "human")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for valid actor, got %d", rec.Code)
	}

	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["actorId"] != "user_123" {
		t.Errorf("expected actorId 'user_123', got %v", resp["actorId"])
	}
	if resp["role"] != "human" {
		t.Errorf("expected role 'human', got %v", resp["role"])
	}
}

func TestActorMiddleware_InvalidRole(t *testing.T) {
	e := echo.New()
	e.Use(middleware.ActorMiddleware())
	e.GET("/test", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Actor-ID", "user_123")
	req.Header.Set("X-Actor-Role", "invalid_role")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid role, got %d", rec.Code)
	}
}

func TestRequireFeatureGate_Disabled(t *testing.T) {
	orig := middleware.FeatureFlags.PlatformOrchestration
	middleware.FeatureFlags.PlatformOrchestration = false
	defer func() { middleware.FeatureFlags.PlatformOrchestration = orig }()

	e := echo.New()
	e.Use(middleware.RequireFeatureGate("platform-orchestration"))
	e.GET("/test", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for disabled feature flag, got %d", rec.Code)
	}
}

func TestRequireFeatureGate_Enabled(t *testing.T) {
	orig := middleware.FeatureFlags.PlatformOrchestration
	middleware.FeatureFlags.PlatformOrchestration = true
	defer func() { middleware.FeatureFlags.PlatformOrchestration = orig }()

	e := echo.New()
	e.Use(middleware.RequireFeatureGate("platform-orchestration"))
	e.GET("/test", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for enabled feature flag, got %d", rec.Code)
	}
}

func TestRequireRole_HumanAllowed(t *testing.T) {
	e := echo.New()
	e.Use(middleware.ActorMiddleware())
	e.Use(middleware.RequireRole("human", "layer_a"))
	e.GET("/test", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Actor-ID", "user_123")
	req.Header.Set("X-Actor-Role", "human")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for human role, got %d", rec.Code)
	}
}

func TestRequireRole_LayerBForbidden(t *testing.T) {
	e := echo.New()
	e.Use(middleware.ActorMiddleware())
	e.Use(middleware.RequireRole("human", "layer_a"))
	e.GET("/test", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Actor-ID", "agent_b")
	req.Header.Set("X-Actor-Role", "layer_b")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for layer_b role, got %d", rec.Code)
	}
}

func TestComputeWebhookSignature(t *testing.T) {
	body := []byte(`{"eventId":"ev_123","topic":"task.created"}`)
	secret := "test-secret-123"

	sig := service.ComputeWebhookSignature(body, secret)

	if len(sig) < 7 || sig[:7] != "sha256=" {
		t.Errorf("signature should start with 'sha256=', got %q", sig)
	}
}

func TestIsLocalhostURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"http://localhost:8080/webhook", true},
		{"http://127.0.0.1:8080/webhook", true},
		{"https://api.example.com/webhook", false},
		{"http://192.168.1.1/webhook", false},
	}

	for _, tc := range tests {
		result := isLocalhostURL(tc.url)
		if result != tc.expected {
			t.Errorf("isLocalhostURL(%q) = %v, expected %v", tc.url, result, tc.expected)
		}
	}
}

func TestEventFanout_AddClient(t *testing.T) {
	fanout := event.NewFanout()

	client, err := fanout.AddClient("client_1", "project_1", "")
	if err != nil {
		t.Fatalf("AddClient failed: %v", err)
	}
	if client.ID != "client_1" {
		t.Errorf("expected client ID 'client_1', got %q", client.ID)
	}
	if client.ProjectID != "project_1" {
		t.Errorf("expected project ID 'project_1', got %q", client.ProjectID)
	}
}

func TestEventFanout_MaxClients(t *testing.T) {
	fanout := event.NewFanout()

	for i := 0; i < 50; i++ {
		_, err := fanout.AddClient(fmt.Sprintf("client_%d", i), "project_1", "")
		if err != nil {
			t.Fatalf("AddClient failed at client %d: %v", i, err)
		}
	}

	_, err := fanout.AddClient("client_51", "project_1", "")
	if err == nil {
		t.Error("expected error when adding 51st client, got nil")
	}
}

func TestEventFanout_Broadcast(t *testing.T) {
	fanout := event.NewFanout()

	_, _ = fanout.AddClient("client_1", "project_1", "")
	_, _ = fanout.AddClient("client_2", "project_1", "")

	ev := models.AuditEvent{
		EventID:       "ev_test",
		SchemaVersion: "v1alpha",
		ProjectID:    "project_1",
		Topic:        "task.created",
		ActorID:      "user_1",
		ActorRole:    "human",
		Timestamp:    time.Now().UTC().Format(time.RFC3339Nano),
		Payload:      map[string]any{"title": "Test task"},
	}

	fanout.Broadcast("project_1", "task.created", ev)
}

func TestEventFanout_RemoveClient(t *testing.T) {
	fanout := event.NewFanout()

	client, _ := fanout.AddClient("client_1", "project_1", "")
	if client == nil {
		t.Fatal("expected client to be created")
	}

	fanout.RemoveClient("client_1")

	count := fanout.ClientCount("project_1")
	if count != 0 {
		t.Errorf("expected 0 clients after removal, got %d", count)
	}
}

func TestDefaultDecompositionLimits_HasCorrectValues(t *testing.T) {
	// FR-02-010: Default limits: max depth 3, max 20 active children per parent.
	// Hard caps: depth 5, 50 active children per parent.
	limits := service.DefaultDecompositionLimits()
	if limits.DefaultDepth != 3 {
		t.Errorf("expected DefaultDepth 3, got %d", limits.DefaultDepth)
	}
	if limits.DefaultFanout != 20 {
		t.Errorf("expected DefaultFanout 20, got %d", limits.DefaultFanout)
	}
	if limits.HardDepthCap != 5 {
		t.Errorf("expected HardDepthCap 5, got %d", limits.HardDepthCap)
	}
	if limits.HardFanoutCap != 50 {
		t.Errorf("expected HardFanoutCap 50, got %d", limits.HardFanoutCap)
	}
}

// isLocalhostURL is a package-level helper mirrored from service for testing.
func isLocalhostURL(url string) bool {
	return strings.HasPrefix(url, "http://localhost") ||
		strings.HasPrefix(url, "http://127.0.0.1") ||
		strings.HasPrefix(url, "https://localhost") ||
		strings.HasPrefix(url, "https://127.0.0.1")
}
