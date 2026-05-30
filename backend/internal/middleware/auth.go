package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/agent-orchestrator/backend/internal/event"
	"github.com/agent-orchestrator/backend/internal/models"
)

// auditEmitter is set during initialization by main.go.
var auditEmitter *event.AuditEmitter

// SetAuditEmitter wires the audit emitter so EmitAuthDenied can write auth.mutation.denied events.
// fn should be a function that inserts an auth.mutation.denied event into the DB.
func SetAuditEmitter(fn event.EmitAuthDeniedFunc) {
	auditEmitter = event.NewAuditEmitterWithFn(fn)
}

// FeatureFlags holds the current feature flag states.
// These are read from environment variables or database at startup.
var FeatureFlags = models.FeatureFlags{
	PlatformOrchestration: false, // master gate, default false per FR-02-028
	LayerAAgents:          false,
	LayerBAgents:          false,
	HumanGates:            false,
	AuditTrail:            false,
}

// RequireFeatureGate returns a middleware that blocks requests when the flag is disabled.
// For project-scoped BRD-02 endpoints, requires platform-orchestration=true.
func RequireFeatureGate(flagName string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			enabled := false
			switch flagName {
			case "platform-orchestration":
				enabled = FeatureFlags.PlatformOrchestration
			case "layer-a-agents":
				enabled = FeatureFlags.LayerAAgents
			case "layer-b-agents":
				enabled = FeatureFlags.LayerBAgents
			case "human-gates":
				enabled = FeatureFlags.HumanGates
			case "audit-trail":
				enabled = FeatureFlags.AuditTrail
			}
			if !enabled {
				return c.JSON(http.StatusForbidden, models.Error{
					Type:   "https://api.agentorchestrator.example.com/errors/feature-disabled",
					Title:  "Feature flag disabled",
					Status: http.StatusForbidden,
					Detail: "platform-orchestration feature flag must be enabled to access this endpoint",
				})
			}
			return next(c)
		}
	}
}

// RequirePlatformOrchestration is a convenience middleware for all BRD-02 endpoints.
var RequirePlatformOrchestration = RequireFeatureGate("platform-orchestration")

// ActorMiddleware extracts actor identity and role from request headers.
// Headers: X-Actor-ID, X-Actor-Role
// Returns 401 if missing or invalid role (fail-closed per ADR-02-005).
func ActorMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			actorID := c.Request().Header.Get("X-Actor-ID")
			actorRole := c.Request().Header.Get("X-Actor-Role")

			if actorID == "" || actorRole == "" {
				c.Set("actor", &models.Actor{ID: "anonymous", Role: "system"})
				return c.JSON(http.StatusUnauthorized, models.Error{
					Type:   "https://api.agentorchestrator.example.com/errors/unauthorized",
					Title:  "Missing actor identity",
					Status: http.StatusUnauthorized,
					Detail: "X-Actor-ID and X-Actor-Role headers are required",
				})
			}

			// Validate role (fail-closed on unknown roles per ADR-02-005)
			if !isValidRole(actorRole) {
				c.Set("actor", &models.Actor{ID: actorID, Role: actorRole})
				EmitAuthDenied(c, actorID, actorRole, "role_validation", "invalid role")
				return c.JSON(http.StatusUnauthorized, models.Error{
					Type:   "https://api.agentorchestrator.example.com/errors/invalid-role",
					Title:  "Invalid actor role",
					Status: http.StatusUnauthorized,
					Detail: "Actor role must be one of: human, layer_a, layer_b, system",
				})
			}

			// system role is infrastructure-only (ADR-02-005)
			if actorRole == "system" {
				// Allow only audit event emission and feature flag changes
				// The handler must validate the specific action
			}

			c.Set("actor", &models.Actor{ID: actorID, Role: actorRole})
			return next(c)
		}
	}
}

// isValidRole checks if the role is one of the four recognized roles.
func isValidRole(role string) bool {
	switch role {
	case "human", "layer_a", "layer_b", "system":
		return true
	default:
		return false
	}
}

// GetActor extracts the actor from the Echo context.
func GetActor(c echo.Context) *models.Actor {
	actor, ok := c.Get("actor").(*models.Actor)
	if !ok {
		return &models.Actor{ID: "anonymous", Role: "system"}
	}
	return actor
}

// RequireRole returns a middleware that enforces role-based access.
// allowedRoles is a list of roles that are permitted for this endpoint.
func RequireRole(allowedRoles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			actor := GetActor(c)
			if actor.Role == "" {
				return c.JSON(http.StatusUnauthorized, models.Error{
					Type:   "https://api.agentorchestrator.example.com/errors/unauthorized",
					Title:  "Missing actor",
					Status: http.StatusUnauthorized,
				})
			}

			// Check if actor's role is in allowed roles
			allowed := false
			for _, role := range allowedRoles {
				if actor.Role == role {
					allowed = true
					break
				}
			}
			if !allowed {
				EmitAuthDenied(c, actor.ID, actor.Role, "role_authorization", "insufficient role")
				return c.JSON(http.StatusForbidden, models.Error{
					Type:   "https://api.agentorchestrator.example.com/errors/forbidden",
					Title:  "Insufficient permissions",
					Status: http.StatusForbidden,
					Detail: "This action requires one of: " + strings.Join(allowedRoles, ", "),
				})
			}

			return next(c)
		}
	}
}

// EmitAuthDenied emits an auth.mutation.denied event for failed auth attempts.
// This is called by the authorization middleware when a mutation is denied.
// The event is inserted synchronously to guarantee the audit trail (NFR-02-014).
func EmitAuthDenied(c echo.Context, actorID, actorRole, attemptedAction, deniedReason string) {
	if auditEmitter == nil {
		// Not initialized; skip rather than panic
		return
	}
	projectID := c.Param("projectId")
	if projectID == "" {
		projectID = "unknown"
	}
	auditEmitter.EmitAuthDenied(c.Request().Context(), projectID, actorID, actorRole, attemptedAction, deniedReason)
}

// RequireLayerA is a convenience middleware requiring layer_a role.
var RequireLayerA = RequireRole("human", "layer_a")

// RequireLayerB is a convenience middleware requiring layer_b role.
var RequireLayerB = RequireRole("layer_b")

// RequireHuman is a convenience middleware requiring human role.
var RequireHuman = RequireRole("human")

// RequireSystem is a convenience middleware requiring system role (for audit-only endpoints).
var RequireSystem = RequireRole("system")

// RequireHumanOrLayerA is used for decomposition proposal endpoints.
var RequireHumanOrLayerA = RequireRole("human", "layer_a")

// ------- test‐only helpers -------
// These live here so the handler package can set/restore the audit emitter in unit tests
// without importing the event package (which would create a cycle).

var testAuditEmitter *event.AuditEmitter

// SetAuditEmitterForTest replaces the global audit emitter (used by EmitAuthDenied).
// Caller must call RestoreAuditEmitterForTest to revert.
func SetAuditEmitterForTest(fn event.EmitAuthDeniedFunc) {
	testAuditEmitter = event.NewAuditEmitterWithFn(fn)
	auditEmitter = testAuditEmitter
}

// GetAuditEmitterForTest returns the current test audit emitter.
func GetAuditEmitterForTest() *event.AuditEmitter {
	return testAuditEmitter
}

// RestoreAuditEmitterForTest restores the previous audit emitter.
func RestoreAuditEmitterForTest(prev *event.AuditEmitter) {
	testAuditEmitter = prev
	auditEmitter = prev
}
