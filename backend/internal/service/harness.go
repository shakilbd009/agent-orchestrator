package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/agent-orchestrator/backend/internal/models"
)

// This file defines the runtime-agnostic agent-execution harness adapter
// (BRD-04+). The interface is backend-agnostic so the Phase F provider-SDK-native
// backend can implement it unchanged. The DEV backend (Pi primary, OpenCode
// fallback) lives in harness_dev.go.
//
// Failure/idle semantics (per contracts/events.md agent.* topics):
//   - worker spawned for (agent, task)      -> agent.activated
//   - worker exits, no artifact, no handoff -> agent.idle
//   - worker cannot proceed / non-zero exit / timeout -> agent.blocked + TaskService.BlockTask
//   - worker produces artifacts + completion evidence  -> TaskService.CompleteTask (-> handoff.submitted)

// AgentProfile is the runtime configuration for a role: the system prompt that
// shapes behavior and the tool allowlist that constrains what the worker may do.
// Provider/Model are optional; empty means "use the runtime default".
type AgentProfile struct {
	Name         string   // role label, e.g. "developer", "architect"
	Layer        string   // "A" or "B"
	SystemPrompt string   // passed to the worker as --system-prompt
	Tools        []string // tool allowlist passed as --tools
	Provider     string   // optional; "" = runtime default
	Model        string   // optional; "" = runtime default
}

// HarnessTask is the task view the harness needs. It is a subset of
// models.OrchestrationTask so the adapter stays decoupled from the task model.
type HarnessTask struct {
	ID            string
	ProjectID     string
	Title         string
	Body          string
	Assignee      string // role label or agent instance id (durable identity seed)
	Layer         string // "A" or "B"
	WorkspacePath string // cwd for the worker process
}

// WorkerEventKind classifies a single lifecycle event streamed from a worker.
type WorkerEventKind int

const (
	// WorkerStep is an informational progress event (text delta / tool call).
	// The supervisor logs it; it maps to no agent.* topic.
	WorkerStep WorkerEventKind = iota
	// WorkerCompleted means the worker ran to completion with a non-empty result.
	// Drives TaskService.CompleteTask -> handoff.submitted.
	WorkerCompleted
	// WorkerBlocked means the worker cannot proceed (non-zero exit, timeout,
	// or error). Emits agent.blocked and calls TaskService.BlockTask.
	WorkerBlocked
	// WorkerIdle means the worker exited cleanly but produced no completion
	// evidence. Emits agent.idle.
	WorkerIdle
)

// WorkerEvent is a single lifecycle event from a running worker, streamed on the
// channel returned by Harness.Spawn. The terminal event (Completed/Blocked/Idle)
// is always the last value on the channel; the supervisor closes out after it.
type WorkerEvent struct {
	Kind                WorkerEventKind
	TaskID              string
	AgentName           string
	Layer               string
	Message             string   // human-readable detail (blocked reason / step text)
	Summary             string   // completion summary (WorkerCompleted)
	Artifacts           []string // artifact paths (WorkerCompleted)
	ValidationPerformed string   // handoff evidence (WorkerCompleted)
	RisksOrResidualIssues string // handoff evidence (WorkerCompleted)
	RecommendedNextGate string   // handoff evidence (WorkerCompleted)
	Err                 error    // underlying error (WorkerBlocked)
}

// Harness is the runtime-agnostic adapter. The DEV backend (pi/opencode) and
// the future Phase F provider-SDK-native backend both implement this interface.
type Harness interface {
	// Spawn launches one worker process for the task/profile pair and returns a
	// channel of WorkerEvent. The channel closes when the worker has fully exited.
	// A non-nil error means the worker could not be started at all.
	Spawn(ctx context.Context, task HarnessTask, profile AgentProfile) (<-chan WorkerEvent, error)
	// Runtime reports which backend is active, e.g. "pi" or "opencode".
	Runtime() string
}

// ---------------------------------------------------------------------------
// Agent profiles — derived from the Layer A/B roles in AGENTS.md.
// Layer A (orchestrator): architect, pm.
// Layer B (specialist):   developer, reviewer, qa, devops.
// Tool names are the pi built-in tool names (read, bash, edit, write, ...).
// Kept minimal for the MVP; expand via ADR when roles gain responsibilities.
// ---------------------------------------------------------------------------

var roleProfiles = map[string]AgentProfile{
	"architect": {
		Name:   "architect",
		Layer:  "A",
		SystemPrompt: "You are the architect (Layer A orchestrator). Decompose work into child tasks, set quality gates, and record ADRs. Read existing decisions before proposing patterns. Do not implement code yourself; delegate implementation to Layer B specialists.",
		Tools:  []string{"read", "bash", "write"},
	},
	"pm": {
		Name:   "pm",
		Layer:  "A",
		SystemPrompt: "You are the project manager (Layer A). Curate artifacts, run PM gate reviews, and maintain the graduation evidence package. Decide approve/repair/escalate at each gate. Do not write implementation code.",
		Tools:  []string{"read", "bash", "write"},
	},
	"developer": {
		Name:   "developer",
		Layer:  "B",
		SystemPrompt: "You are a developer (Layer B specialist). Implement the assigned task surgically: touch only affected components, follow existing conventions, and verify intent with tests. Submit structured handoff evidence on completion.",
		Tools:  []string{"read", "bash", "edit", "write"},
	},
	"reviewer": {
		Name:   "reviewer",
		Layer:  "B",
		SystemPrompt: "You are a code reviewer (Layer B specialist). Review for correctness, security, and quality. Read-only: do not modify the code under review; report findings as handoff evidence.",
		Tools:  []string{"read", "bash"},
	},
	"qa": {
		Name:   "qa",
		Layer:  "B",
		SystemPrompt: "You are QA (Layer B specialist). Verify test coverage and BRD compliance against the shipped implementation. Author the smallest check that fails if the logic breaks. Report pass/fail as handoff evidence.",
		Tools:  []string{"read", "bash", "write"},
	},
	"devops": {
		Name:   "devops",
		Layer:  "B",
		SystemPrompt: "You are DevOps (Layer B specialist). Verify the production checklist and readiness against the shipped implementation before flag enablement. Read-only on app code; may edit infra/scripts.",
		Tools:  []string{"read", "bash", "write"},
	},
}

// ProfileForRole returns the AgentProfile for a known role label, or ok=false.
func ProfileForRole(role string) (AgentProfile, bool) {
	p, ok := roleProfiles[role]
	return p, ok
}

// profileForTask picks a profile for a task: the assignee's role if it is a known
// role label, otherwise a layer-based default (architect for Layer A, developer
// for Layer B). This keeps activation robust when Assignee is an opaque agent id.
func profileForTask(task *models.OrchestrationTask) AgentProfile {
	if p, ok := ProfileForRole(task.Assignee); ok {
		return p
	}
	if task.Layer == "A" {
		return roleProfiles["architect"]
	}
	return roleProfiles["developer"]
}

// roleForLayer maps a task layer to the actor role used when the harness drives
// task lifecycle calls (CompleteTask/BlockTask) on behalf of the worker.
func roleForLayer(layer string) string {
	if layer == "A" {
		return "layer_a"
	}
	return "layer_b"
}

// ---------------------------------------------------------------------------
// Worker supervision: maps the WorkerEvent stream onto the existing EventService
// (agent.* topics) and TaskService (CompleteTask/BlockTask). No new transport or
// envelope — it reuses the canonical AuditEvent path that already feeds SSE.
// ---------------------------------------------------------------------------

// Harness limits. Defaults are conservative; override via ConfigureHarness from
// env (AGENT_HARNESS_MAX_CONCURRENT / AGENT_HARNESS_TASK_TIMEOUT) in main.go.
// Call ConfigureHarness before the first activation (tests use the defaults).
const (
	defaultMaxConcurrentWorkers = 6
	defaultWorkerTimeout        = 30 * time.Minute
)

var (
	harnessTaskTimeout = defaultWorkerTimeout
	// workerSlots bounds concurrent worker processes (B1/M1: liveness + resource
	// ceiling). A buffered channel used as a counting semaphore.
	workerSlots = make(chan struct{}, defaultMaxConcurrentWorkers)
)

// ConfigureHarness sets the worker concurrency cap and per-task timeout. Must be
// called before the first ActivateTask (it replaces the semaphore channel).
func ConfigureHarness(maxConcurrent int, taskTimeout time.Duration) {
	if maxConcurrent > 0 {
		workerSlots = make(chan struct{}, maxConcurrent)
	}
	if taskTimeout > 0 {
		harnessTaskTimeout = taskTimeout
	}
}

// acquireWorkerSlot tries to take a concurrency slot. Returns false if the cap
// is reached (caller emits agent.blocked instead of spawning).
func acquireWorkerSlot() bool {
	select {
	case workerSlots <- struct{}{}:
		return true
	default:
		return false
	}
}

func releaseWorkerSlot() {
	select {
	case <-workerSlots:
	default:
	}
}

// activeWorkers prevents double-spawning a worker for the same task.
// ponytail: in-memory only; not durable across process restarts. If the backend
// restarts mid-run the task stays in_progress until re-activated or a human
// intervenes. Upgrade to a DB-backed registry if durability is required.
var activeWorkers sync.Map // taskID -> struct{}

// markWorkerActive records that a worker is running for a task. Returns false if
// one is already running (caller should treat as already-activated, not an error).
func markWorkerActive(taskID string) bool {
	_, loaded := activeWorkers.LoadOrStore(taskID, struct{}{})
	return !loaded
}

func releaseWorker(taskID string) {
	activeWorkers.Delete(taskID)
}

// superviseWorker consumes a worker's event stream and drives the task lifecycle.
// It runs in its own goroutine, launched by TaskService.ActivateTask. The context
// carries the per-task deadline (B1) so a hung worker is reaped, not wedged.
func (s *TaskService) superviseWorker(
	ctx context.Context,
	projectID, taskID, agentName, layer string,
	events <-chan WorkerEvent,
) {
	defer releaseWorker(taskID)
	defer releaseWorkerSlot()

	terminal := false
	for ev := range events {
		switch ev.Kind {
		case WorkerStep:
			// Informational only; no agent.* topic in the v1alpha contract.
			continue
		case WorkerCompleted:
			terminal = true
			actor := &models.Actor{ID: agentName, Role: roleForLayer(layer)}
			handoff := &models.HandoffEvidence{
				Summary:               ev.Summary,
				Artifacts:             ev.Artifacts,
				ValidationPerformed:   ev.ValidationPerformed,
				RisksOrResidualIssues: ev.RisksOrResidualIssues,
				RecommendedNextGate:   ev.RecommendedNextGate,
			}
			if err := s.CompleteTask(ctx, projectID, taskID, handoff, actor); err != nil {
				// Completion rejected (e.g. required children open, layer-B scope).
				// Surface as blocked so the orchestrator can re-route.
				reason := fmt.Sprintf("completion rejected: %v", err)
				_ = s.BlockTask(ctx, projectID, taskID, reason, actor)
				s.emitAgentEvent(ctx, "agent.blocked", &taskID, projectID, agentName, layer, map[string]any{
					"agentName": agentName, "layer": layer, "taskId": taskID, "reason": reason,
				})
			}
			return
		case WorkerBlocked:
			terminal = true
			actor := &models.Actor{ID: agentName, Role: roleForLayer(layer)}
			reason := ev.Message
			if reason == "" && ev.Err != nil {
				reason = ev.Err.Error()
			}
			if reason == "" {
				reason = "worker could not proceed"
			}
			_ = s.BlockTask(ctx, projectID, taskID, reason, actor)
			s.emitAgentEvent(ctx, "agent.blocked", &taskID, projectID, agentName, layer, map[string]any{
				"agentName": agentName, "layer": layer, "taskId": taskID, "reason": reason,
			})
			return
		case WorkerIdle:
			terminal = true
			now := time.Now().UTC().Format(time.RFC3339Nano)
			s.emitAgentEvent(ctx, "agent.idle", nil, projectID, agentName, layer, map[string]any{
				"agentName": agentName, "layer": layer, "idleSince": now,
			})
			return
		}
	}

	// m4 safety net: the channel closed without a terminal event (e.g. a future
	// stream early-return). Don't leave the task pinned in_progress silently.
	if !terminal {
		reason := "worker stream ended without a terminal event"
		actor := &models.Actor{ID: agentName, Role: roleForLayer(layer)}
		_ = s.BlockTask(ctx, projectID, taskID, reason, actor)
		s.emitAgentEvent(ctx, "agent.blocked", &taskID, projectID, agentName, layer, map[string]any{
			"agentName": agentName, "layer": layer, "taskId": taskID, "reason": reason,
		})
	}
}

// emitAgentEvent emits one agent.* event on the canonical envelope. topic is one
// of agent.activated / agent.idle / agent.blocked; taskID is nil for agent.idle
// per contracts/events.md. payload carries the topic-specific fields.
func (s *TaskService) emitAgentEvent(ctx context.Context, topic string, taskID *string, projectID, agentName, layer string, payload map[string]any) {
	s.eventSvc.Emit(ctx, models.AuditEvent{
		EventID:       newID("ev"),
		SchemaVersion: "v1alpha",
		ProjectID:     projectID,
		Topic:         topic,
		ActorID:       agentName,
		ActorRole:     roleForLayer(layer),
		TaskID:        taskID,
		ParentTaskID:  nil,
		GateID:        nil,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Payload:       payload,
	})
}
