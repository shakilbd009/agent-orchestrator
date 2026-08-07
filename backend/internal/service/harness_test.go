package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agent-orchestrator/backend/internal/middleware"
	"github.com/agent-orchestrator/backend/internal/models"
)

// ---------------------------------------------------------------------------
// Profiles
// ---------------------------------------------------------------------------

func TestProfileForRole_KnownRoles(t *testing.T) {
	for _, role := range []string{"architect", "pm", "developer", "reviewer", "qa", "devops"} {
		p, ok := ProfileForRole(role)
		if !ok {
			t.Errorf("expected profile for role %q", role)
			continue
		}
		if p.Name != role {
			t.Errorf("role %q: name=%q", role, p.Name)
		}
		if p.SystemPrompt == "" {
			t.Errorf("role %q: empty system prompt", role)
		}
		if len(p.Tools) == 0 {
			t.Errorf("role %q: empty tool allowlist", role)
		}
	}
}

func TestProfileForRole_Unknown(t *testing.T) {
	if _, ok := ProfileForRole("nope"); ok {
		t.Fatal("expected unknown role to miss")
	}
}

func TestProfileForTask_AssigneeRoleWins(t *testing.T) {
	task := &models.OrchestrationTask{Assignee: "qa", Layer: "B"}
	p := profileForTask(task)
	if p.Name != "qa" {
		t.Fatalf("expected qa, got %s", p.Name)
	}
}

func TestProfileForTask_LayerFallback(t *testing.T) {
	// Opaque assignee -> layer-based default.
	a := profileForTask(&models.OrchestrationTask{Assignee: "agent_007", Layer: "A"})
	if a.Name != "architect" {
		t.Errorf("Layer A default = %s, want architect", a.Name)
	}
	b := profileForTask(&models.OrchestrationTask{Assignee: "agent_007", Layer: "B"})
	if b.Name != "developer" {
		t.Errorf("Layer B default = %s, want developer", b.Name)
	}
}

func TestRoleProfiles_ReviewerIsReadOnly(t *testing.T) {
	// Reviewer must not carry write/edit tools (read-only review).
	p, _ := ProfileForRole("reviewer")
	for _, tool := range p.Tools {
		if tool == "edit" || tool == "write" {
			t.Errorf("reviewer must be read-only, but allows %q", tool)
		}
	}
}

func TestRoleForLayer(t *testing.T) {
	if roleForLayer("A") != "layer_a" {
		t.Error("A -> layer_a")
	}
	if roleForLayer("B") != "layer_b" {
		t.Error("B -> layer_b")
	}
}

// ---------------------------------------------------------------------------
// Pi NDJSON parsing — driven by the real captured headless fixture.
// ---------------------------------------------------------------------------

func loadFixtureRecords(t *testing.T) []map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "pi_headless_fixture.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var f struct {
		Records []map[string]any `json:"records"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	if len(f.Records) == 0 {
		t.Fatal("fixture has no records")
	}
	return f.Records
}

// TestParsePiRecord_RealFixture replays the captured pi 0.83.0 headless run and
// asserts the parser accumulates the assistant's final text and sees agent_end.
func TestParsePiRecord_RealFixture(t *testing.T) {
	records := loadFixtureRecords(t)
	st := &piState{runtime: "pi"}
	task := HarnessTask{ID: "t1"}
	prof := roleProfiles["developer"]

	var steps int
	for _, rec := range records {
		ev := parsePiRecord(rec, st, task, prof)
		if ev != nil && ev.Kind == WorkerStep {
			steps++
		}
	}
	if !st.sawAgentEnd {
		t.Error("expected sawAgentEnd after replaying fixture")
	}
	if st.finalText != "PONG" {
		t.Errorf("finalText = %q, want PONG", st.finalText)
	}
	if steps == 0 {
		t.Error("expected at least one WorkerStep from assistant text")
	}
}

// TestParsePiRecord_TurnEndCarriesText confirms turn_end alone seeds finalText.
func TestParsePiRecord_TurnEndCarriesText(t *testing.T) {
	st := &piState{runtime: "pi"}
	parsePiRecord(map[string]any{
		"type":    "turn_end",
		"message": map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "text", "text": "hello"}}},
	}, st, HarnessTask{ID: "t1"}, roleProfiles["developer"])
	if st.finalText != "hello" {
		t.Fatalf("finalText=%q want hello", st.finalText)
	}
}

// ---------------------------------------------------------------------------
// decideTerminal — completion / idle / blocked classification
// ---------------------------------------------------------------------------

func TestDecideTerminal_NonZeroExitBlocked(t *testing.T) {
	h := &devHarness{runtime: "pi"}
	ev := h.decideTerminal(context.Background(), &exec.ExitError{}, &piState{finalText: "partial"}, HarnessTask{ID: "t1"}, roleProfiles["developer"])
	if ev.Kind != WorkerBlocked {
		t.Fatalf("want blocked, got %v", ev.Kind)
	}
}

func TestDecideTerminal_CanceledBlocked(t *testing.T) {
	h := &devHarness{runtime: "pi"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ev := h.decideTerminal(ctx, nil, &piState{finalText: "x"}, HarnessTask{ID: "t1"}, roleProfiles["developer"])
	if ev.Kind != WorkerBlocked {
		t.Fatalf("canceled ctx should be blocked, got %v", ev.Kind)
	}
}

func TestDecideTerminal_CleanExitWithTextCompleted(t *testing.T) {
	h := &devHarness{runtime: "pi"}
	ev := h.decideTerminal(context.Background(), nil, &piState{finalText: "did the thing"}, HarnessTask{ID: "t1"}, roleProfiles["developer"])
	if ev.Kind != WorkerCompleted {
		t.Fatalf("want completed, got %v", ev.Kind)
	}
	if ev.Summary != "did the thing" {
		t.Errorf("summary=%q", ev.Summary)
	}
}

func TestDecideTerminal_CleanExitNoTextIdle(t *testing.T) {
	h := &devHarness{runtime: "pi"}
	ev := h.decideTerminal(context.Background(), nil, &piState{finalText: ""}, HarnessTask{ID: "t1"}, roleProfiles["developer"])
	if ev.Kind != WorkerIdle {
		t.Fatalf("want idle, got %v", ev.Kind)
	}
}

func TestDecideTerminal_StartFailureBlocked(t *testing.T) {
	h := &devHarness{runtime: "pi"}
	ev := h.decideTerminal(context.Background(), errors.New("signal: killed"), &piState{}, HarnessTask{ID: "t1"}, roleProfiles["developer"])
	if ev.Kind != WorkerBlocked {
		t.Fatalf("run error should be blocked, got %v", ev.Kind)
	}
}

// ---------------------------------------------------------------------------
// NewDevHarness runtime detection
// ---------------------------------------------------------------------------

func TestNewDevHarness_PrefersPi(t *testing.T) {
	lookPath := func(name string) (string, error) {
		switch name {
		case "pi":
			return "/usr/local/bin/pi", nil
		default:
			return "", os.ErrNotExist
		}
	}
	h, err := newDevHarnessWith(lookPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Runtime() != "pi" {
		t.Errorf("runtime=%s want pi", h.Runtime())
	}
}

func TestNewDevHarness_OpenCodeFallback(t *testing.T) {
	lookPath := func(name string) (string, error) {
		switch name {
		case "opencode":
			return "/usr/local/bin/opencode", nil
		default:
			return "", os.ErrNotExist
		}
	}
	h, err := newDevHarnessWith(lookPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Runtime() != "opencode" {
		t.Errorf("runtime=%s want opencode", h.Runtime())
	}
}

func TestNewDevHarness_NoneAvailable(t *testing.T) {
	lookPath := func(string) (string, error) { return "", os.ErrNotExist }
	if _, err := newDevHarnessWith(lookPath); err == nil {
		t.Fatal("expected error when no runtime available")
	}
}

// commandFor builds runtime argv; sanity-check the pi flags the brief requires.
func TestDevHarnessCommandFor_PiFlags(t *testing.T) {
	h := &devHarness{runtime: "pi"}
	prof := AgentProfile{SystemPrompt: "be brief", Tools: []string{"read", "bash"}, Provider: "google", Model: "gemini-2.5-pro"}
	name, args := h.commandFor(prof, "do X", "sess-1", "/tmp/s")
	if name != "pi" {
		t.Fatalf("name=%s want pi", name)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{"-p", "--mode json", "--session-id sess-1", "--system-prompt be brief", "--tools read,bash", "--provider google", "--model gemini-2.5-pro"} {
		if !strings.Contains(joined, want) {
			t.Errorf("pi args missing %q; args=%s", want, joined)
		}
	}
}

func TestDevHarnessCommandFor_OpenCodeFallback(t *testing.T) {
	h := &devHarness{runtime: "opencode"}
	name, args := h.commandFor(AgentProfile{Model: "google/gemini-2.5-pro"}, "do X", "sess-1", "/tmp/s")
	if name != "opencode" {
		t.Fatalf("name=%s want opencode", name)
	}
	if args[0] != "run" {
		t.Errorf("first arg=%s want run", args[0])
	}
}

// ---------------------------------------------------------------------------
// Spawn end-to-end with a fake process — proves the harness pipes stdout into
// WorkerEvents and classifies the terminal outcome against the real NDJSON shape.
// ---------------------------------------------------------------------------

type fakeCmd struct {
	stdout ioReader
	waitErr error
	started bool
}

type ioReader struct{ data string }

func (r *ioReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

func (f *fakeCmd) Start() error            { f.started = true; return nil }
func (f *fakeCmd) Wait() error             { return f.waitErr }
func (f *fakeCmd) StdoutPipe() (io.ReadCloser, error) {
	return &nopCloser{&f.stdout}, nil
}

type nopCloser struct{ r *ioReader }

func (n *nopCloser) Read(p []byte) (int, error) { return n.r.Read(p) }
func (n *nopCloser) Close() error                { return nil }

func TestSpawn_PiFixtureYieldsCompleted(t *testing.T) {
	records := loadFixtureRecords(t)
	var sb strings.Builder
	for _, r := range records {
		b, _ := json.Marshal(r)
		sb.WriteString(string(b) + "\n")
	}

	h := &devHarness{
		runtime: "pi",
		buildCmd: func(ctx context.Context, name string, args []string, dir string) cmdRunner {
			return &fakeCmd{stdout: ioReader{data: sb.String()}}
		},
	}

	prof := roleProfiles["developer"]
	events, err := h.Spawn(context.Background(), HarnessTask{ID: "t1", ProjectID: "p1", WorkspacePath: ""}, prof)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	var terminal *WorkerEvent
	for ev := range events {
		if ev.Kind != WorkerStep {
			terminal = &ev
		}
	}
	if terminal == nil {
		t.Fatal("expected a terminal event")
	}
	if terminal.Kind != WorkerCompleted {
		t.Errorf("terminal=%v want completed", terminal.Kind)
	}
	if terminal.Summary != "PONG" {
		t.Errorf("summary=%q want PONG", terminal.Summary)
	}
}

func TestSpawn_NonZeroExitBlocked(t *testing.T) {
	h := &devHarness{
		runtime: "pi",
		buildCmd: func(ctx context.Context, name string, args []string, dir string) cmdRunner {
			return &fakeCmd{stdout: ioReader{}, waitErr: errors.New("exit status 1")}
		},
	}
	events, _ := h.Spawn(context.Background(), HarnessTask{ID: "t1", ProjectID: "p1"}, roleProfiles["developer"])
	var terminal *WorkerEvent
	for ev := range events {
		terminal = &ev
	}
	if terminal == nil || terminal.Kind != WorkerBlocked {
		t.Fatalf("expected blocked terminal, got %+v", terminal)
	}
}

// ---------------------------------------------------------------------------
// superviseWorker — maps WorkerEvent stream onto TaskService + agent.* events.
// ---------------------------------------------------------------------------

// captureEventService is a thread-safe event recorder for supervisor tests,
// since the reused CompleteTask path emits via a goroutine.
type captureEventService struct {
	mu      sync.Mutex
	events  []models.AuditEvent
}

func (c *captureEventService) Emit(_ context.Context, ev models.AuditEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, ev)
}

func (c *captureEventService) hasTopic(topic string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.events {
		if e.Topic == topic {
			return true
		}
	}
	return false
}

func (c *captureEventService) waitForTopic(t *testing.T, topic string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if c.hasTopic(topic) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for topic %q (saw: %v)", topic, c.topics())
}

func (c *captureEventService) topics() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []string
	for _, e := range c.events {
		out = append(out, e.Topic)
	}
	return out
}

func newTaskSvcWithCapture() (*TaskService, *mockTaskRepo, *captureEventService) {
	taskRepo := &mockTaskRepo{}
	evt := &captureEventService{}
	svc := NewTaskService(taskRepo, &mockProjectRepo{}, evt)
	return svc, taskRepo, evt
}

func TestSuperviseWorker_CompletionDrivesCompleteTask(t *testing.T) {
	svc, taskRepo, evt := newTaskSvcWithCapture()
	var completed bool
	taskRepo.canCompleteParentFn = func(context.Context, string) error { return nil }
	taskRepo.getTaskByIDFn = func(_ context.Context, pid, tid string) (*models.OrchestrationTask, error) {
		return &models.OrchestrationTask{ID: tid, ProjectID: pid, Assignee: "dev1", Layer: "B"}, nil
	}
	taskRepo.updateTaskStatusFn = func(_ context.Context, _, _, status string, _ *models.Actor, _ string) error {
		if status == "done" {
			completed = true
		}
		return nil
	}

	ch := make(chan WorkerEvent, 1)
	ch <- WorkerEvent{Kind: WorkerCompleted, Summary: "shipped it", Artifacts: []string{"main.go"}}
	close(ch)

	svc.superviseWorker(context.Background(), "p1", "t1", "dev1", "B", ch)

	if !completed {
		t.Error("expected CompleteTask to set status=done")
	}
	// agent.* emissions from the supervisor path + handoff from CompleteTask.
	evt.waitForTopic(t, "handoff.submitted", time.Second)
}

func TestSuperviseWorker_BlockedEmitsBlockedAndBlocksTask(t *testing.T) {
	svc, taskRepo, evt := newTaskSvcWithCapture()
	var blockedReason string
	taskRepo.blockTaskFn = func(_ context.Context, _, tid, reason string, _ *models.Actor) error {
		blockedReason = reason
		return nil
	}

	ch := make(chan WorkerEvent, 1)
	ch <- WorkerEvent{Kind: WorkerBlocked, Message: "tests failed"}
	close(ch)

	svc.superviseWorker(context.Background(), "p1", "t1", "dev1", "B", ch)

	if !strings.Contains(blockedReason, "tests failed") {
		t.Errorf("blockReason=%q", blockedReason)
	}
	if !evt.hasTopic("agent.blocked") {
		t.Errorf("expected agent.blocked; saw %v", evt.topics())
	}
}

func TestSuperviseWorker_IdleEmitsIdle(t *testing.T) {
	svc, _, evt := newTaskSvcWithCapture()
	ch := make(chan WorkerEvent, 1)
	ch <- WorkerEvent{Kind: WorkerIdle}
	close(ch)
	svc.superviseWorker(context.Background(), "p1", "t1", "dev1", "B", ch)
	if !evt.hasTopic("agent.idle") {
		t.Errorf("expected agent.idle; saw %v", evt.topics())
	}
}

// fakeHarness emits a fixed event stream and closes immediately.
type fakeHarness struct {
	runtime string
	events  []WorkerEvent
}

func (f *fakeHarness) Runtime() string { return f.runtime }
func (f *fakeHarness) Spawn(_ context.Context, _ HarnessTask, _ AgentProfile) (<-chan WorkerEvent, error) {
	ch := make(chan WorkerEvent, len(f.events)+1)
	for _, e := range f.events {
		ch <- e
	}
	close(ch)
	return ch, nil
}

func TestActivateTask_FlagOffDoesNotSpawn(t *testing.T) {
	setFlag(t, func() { middleware.FeatureFlags.AgentHarness = false })
	svc, _, evt := newTaskSvcWithCapture()
	svc.SetHarness(&fakeHarness{events: []WorkerEvent{{Kind: WorkerIdle}}})

	taskRepo := svc.repo.(*mockTaskRepo)
	taskRepo.getTaskByIDFn = func(_ context.Context, pid, tid string) (*models.OrchestrationTask, error) {
		return &models.OrchestrationTask{ID: tid, ProjectID: pid, Assignee: "dev1", Layer: "B", Status: "todo"}, nil
	}
	taskRepo.updateTaskStatusFn = func(context.Context, string, string, string, *models.Actor, string) error { return nil }

	task, err := svc.ActivateTask(context.Background(), "p1", "t1", &models.Actor{ID: "human1", Role: "human"})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if task.Status != "in_progress" {
		t.Errorf("status=%s want in_progress", task.Status)
	}
	if evt.hasTopic("agent.activated") {
		t.Error("flag off must not emit agent.activated")
	}
}

func TestActivateTask_FlagOnSpawnsAndEmitsActivated(t *testing.T) {
	setFlag(t, func() { middleware.FeatureFlags.AgentHarness = true })
	svc, taskRepo, evt := newTaskSvcWithCapture()
	svc.SetHarness(&fakeHarness{events: []WorkerEvent{{Kind: WorkerIdle}}})

	taskRepo.getTaskByIDFn = func(_ context.Context, pid, tid string) (*models.OrchestrationTask, error) {
		return &models.OrchestrationTask{ID: tid, ProjectID: pid, Assignee: "developer", Layer: "B", Status: "todo"}, nil
	}
	taskRepo.updateTaskStatusFn = func(context.Context, string, string, string, *models.Actor, string) error { return nil }

	_, err := svc.ActivateTask(context.Background(), "p1", "t1", &models.Actor{ID: "human1", Role: "human"})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	// agent.activated is emitted synchronously before Spawn returns.
	if !evt.hasTopic("agent.activated") {
		t.Errorf("expected agent.activated; saw %v", evt.topics())
	}
	// The supervisor goroutine then emits agent.idle for the idle worker.
	evt.waitForTopic(t, "agent.idle", time.Second)

	// Verify the agent.activated envelope payload shape per contracts/events.md.
	evt.mu.Lock()
	var got *models.AuditEvent
	for i := range evt.events {
		if evt.events[i].Topic == "agent.activated" {
			got = &evt.events[i]
		}
	}
	evt.mu.Unlock()
	if got == nil {
		t.Fatal("no agent.activated event")
	}
	if got.Payload["agentName"] != "developer" || got.Payload["layer"] != "B" || got.Payload["taskId"] != "t1" {
		t.Errorf("agent.activated payload mismatch: %+v", got.Payload)
	}
	if got.ActorRole != "layer_b" {
		t.Errorf("actorRole=%s want layer_b", got.ActorRole)
	}
}

func TestActivateTask_TerminalTaskRejected(t *testing.T) {
	setFlag(t, func() { middleware.FeatureFlags.AgentHarness = true })
	svc, taskRepo, _ := newTaskSvcWithCapture()
	taskRepo.getTaskByIDFn = func(_ context.Context, pid, tid string) (*models.OrchestrationTask, error) {
		return &models.OrchestrationTask{ID: tid, ProjectID: pid, Status: "done"}, nil
	}
	_, err := svc.ActivateTask(context.Background(), "p1", "t1", &models.Actor{ID: "h", Role: "human"})
	if err == nil {
		t.Fatal("expected error activating a done task")
	}
}

func TestActivateTask_NoDoubleSpawn(t *testing.T) {
	setFlag(t, func() { middleware.FeatureFlags.AgentHarness = true })
	// Pre-mark the worker active; ActivateTask must skip spawning (no error).
	markWorkerActive("t_nodup")
	defer releaseWorker("t_nodup")

	svc, taskRepo, evt := newTaskSvcWithCapture()
	spawned := 0
	svc.SetHarness(&countingHarness{onSpawn: func() { spawned++ }})
	taskRepo.getTaskByIDFn = func(_ context.Context, pid, tid string) (*models.OrchestrationTask, error) {
		return &models.OrchestrationTask{ID: tid, ProjectID: pid, Assignee: "dev", Layer: "B", Status: "in_progress"}, nil
	}
	_, err := svc.ActivateTask(context.Background(), "p1", "t_nodup", &models.Actor{ID: "h", Role: "human"})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if spawned != 0 {
		t.Errorf("expected no spawn when worker already active, got %d", spawned)
	}
	if evt.hasTopic("agent.activated") {
		t.Error("must not re-emit agent.activated for already-active worker")
	}
}

type countingHarness struct{ onSpawn func() }

func (c *countingHarness) Runtime() string { return "fake" }
func (c *countingHarness) Spawn(ctx context.Context, _ HarnessTask, _ AgentProfile) (<-chan WorkerEvent, error) {
	c.onSpawn()
	ch := make(chan WorkerEvent)
	go func() { close(ch) }()
	return ch, nil
}

// setFlag toggles the harness feature flags for the duration of a test and
// restores them. ActivateTask requires platform-orchestration (master gate) plus
// agent-harness, so both are set together.
func setFlag(t *testing.T, set func()) {
	t.Helper()
	prevPlat := middleware.FeatureFlags.PlatformOrchestration
	prevHarness := middleware.FeatureFlags.AgentHarness
	middleware.FeatureFlags.PlatformOrchestration = true
	set()
	t.Cleanup(func() {
		middleware.FeatureFlags.PlatformOrchestration = prevPlat
		middleware.FeatureFlags.AgentHarness = prevHarness
	})
}
