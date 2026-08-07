package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

)

// This file implements the DEV backend of the agent-execution harness:
//   - Pi (`pi -p --mode json`) is the primary runtime. Its headless JSON output
//     is newline-delimited JSON (NDJSON) with a stable lifecycle
//     (session -> agent_start -> turn_start -> message_* -> turn_end ->
//     agent_end -> agent_settled). Verified empirically against pi 0.83.0.
//   - OpenCode (`opencode run`) is the fallback. It has no documented JSON mode,
//     so the fallback reads stdout as text and infers completion from non-empty
//     output. The interface is identical; only the line interpretation differs.
//
// Both satisfy the runtime-agnostic Harness interface so the Phase F
// provider-SDK-native backend can replace this without touching callers.

// devHarness is the DEV Harness backend (pi primary, opencode fallback).
type devHarness struct {
	runtime  string // "pi" | "opencode"
	lookPath func(file string) (string, error)
	// buildCmd is the subprocess seam; defaults to exec.CommandContext. Injectable
	// so Spawn can be unit-tested without a real pi/opencode process or LLM call.
	buildCmd func(ctx context.Context, name string, args []string, dir string) cmdRunner
}

// cmdRunner is the subset of *exec.Cmd the harness uses. *exec.Cmd satisfies it.
type cmdRunner interface {
	Start() error
	Wait() error
	StdoutPipe() (io.ReadCloser, error)
}

// execCmd wraps exec.CommandContext to satisfy cmdRunner.
type execCmd struct{ *exec.Cmd }

func (c *execCmd) Start() error            { return c.Cmd.Start() }
func (c *execCmd) Wait() error             { return c.Cmd.Wait() }
func (c *execCmd) StdoutPipe() (io.ReadCloser, error) {
	return c.Cmd.StdoutPipe()
}

// NewDevHarness detects an available runtime (pi preferred, opencode fallback)
// and returns a DEV harness. Returns an error if neither is on PATH — callers
// should treat a nil harness as "agent execution disabled".
func NewDevHarness() (Harness, error) {
	return newDevHarnessWith(exec.LookPath)
}

func newDevHarnessWith(lookPath func(string) (string, error)) (Harness, error) {
	if _, err := lookPath("pi"); err == nil {
		return &devHarness{runtime: "pi", lookPath: lookPath, buildCmd: defaultBuildCmd}, nil
	}
	if _, err := lookPath("opencode"); err == nil {
		return &devHarness{runtime: "opencode", lookPath: lookPath, buildCmd: defaultBuildCmd}, nil
	}
	return nil, fmt.Errorf("no agent runtime found: install pi (primary) or opencode (fallback)")
}

func defaultBuildCmd(ctx context.Context, name string, args []string, dir string) cmdRunner {
	c := exec.CommandContext(ctx, name, args...)
	c.Dir = dir
	return &execCmd{Cmd: c}
}

func (h *devHarness) Runtime() string { return h.runtime }

// Spawn launches one worker process for the task and streams WorkerEvents.
// The returned channel closes once the worker has fully exited.
func (h *devHarness) Spawn(ctx context.Context, task HarnessTask, profile AgentProfile) (<-chan WorkerEvent, error) {
	sessionDir, err := ensureSessionDir(task)
	if err != nil {
		return nil, fmt.Errorf("harness session dir: %w", err)
	}
	sessionID := durableSessionID(task)
	prompt := taskPrompt(task)
	dir := task.WorkspacePath
	if dir == "" {
		dir = sessionDir
	}

	name, args := h.commandFor(profile, prompt, sessionID, sessionDir)
	events := make(chan WorkerEvent, 16)

	cmd := h.buildCmd(ctx, name, args, dir)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("harness stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("harness start %s: %w", h.runtime, err)
	}

	go func() {
		defer close(events)
		st := h.stream(ctx, stdout, events, task, profile)
		// Process has flushed; collect exit status, then emit the terminal event.
		waitErr := cmd.Wait()
		terminal := h.decideTerminal(ctx, waitErr, st, task, profile)
		events <- terminal
	}()

	return events, nil
}

// commandFor builds the argv for the active runtime.
func (h *devHarness) commandFor(profile AgentProfile, prompt, sessionID, sessionDir string) (string, []string) {
	switch h.runtime {
	case "opencode":
		args := []string{"run"}
		if profile.Model != "" {
			args = append(args, "-m", profile.Model)
		}
		if sessionID != "" {
			args = append(args, "-s", sessionID)
		}
		// opencode run takes the message as trailing positional args.
		args = append(args, prompt)
		return "opencode", args
	default: // pi
		args := []string{
			"-p",
			"--mode", "json",
			"--session-id", sessionID,
			"--session-dir", sessionDir,
			"--system-prompt", profile.SystemPrompt,
		}
		if len(profile.Tools) > 0 {
			args = append(args, "--tools", strings.Join(profile.Tools, ","))
		}
		if profile.Provider != "" {
			args = append(args, "--provider", profile.Provider)
		}
		if profile.Model != "" {
			args = append(args, "--model", profile.Model)
		}
		args = append(args, prompt)
		return "pi", args
	}
}

// stream reads worker stdout line by line and emits WorkerStep events. It tracks
// just enough state (piState) to classify the terminal outcome in decideTerminal.
func (h *devHarness) stream(ctx context.Context, stdout io.Reader, events chan<- WorkerEvent, task HarnessTask, profile AgentProfile) *piState {
	st := &piState{runtime: h.runtime}
	sc := bufio.NewScanner(stdout)
	// A single NDJSON record can be large (full message bodies); raise the limit.
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		ev := h.parseLine(line, st, task, profile)
		if ev != nil {
			select {
			case events <- *ev:
			case <-ctx.Done():
				return st
			}
		}
	}
	return st
}

// parseLine interprets one stdout line for the active runtime and updates state.
// Returns a WorkerStep event when the line carries useful progress, nil otherwise.
func (h *devHarness) parseLine(line []byte, st *piState, task HarnessTask, profile AgentProfile) *WorkerEvent {
	if h.runtime != "pi" {
		// opencode: accumulate text; no structured lifecycle to parse.
		txt := string(line)
		st.finalText += txt
		if strings.TrimSpace(txt) == "" {
			return nil
		}
		return &WorkerEvent{Kind: WorkerStep, TaskID: task.ID, AgentName: profile.Name, Layer: profile.Layer, Message: txt}
	}

	// pi: each line is one NDJSON record.
	var rec map[string]any
	if err := json.Unmarshal(line, &rec); err != nil {
		// Not JSON (e.g. a stray banner line). Skip rather than fail the run.
		return nil
	}
	return parsePiRecord(rec, st, task, profile)
}

// decideTerminal classifies the worker's final outcome from exit status + state.
func (h *devHarness) decideTerminal(ctx context.Context, waitErr error, st *piState, task HarnessTask, profile AgentProfile) WorkerEvent {
	base := WorkerEvent{TaskID: task.ID, AgentName: profile.Name, Layer: profile.Layer}

	// Cancellation / timeout.
	if ctx.Err() != nil {
		base.Kind = WorkerBlocked
		base.Message = "worker timed out or was canceled"
		base.Err = ctx.Err()
		return base
	}
	// Non-zero exit or start failure -> blocked, regardless of partial output.
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			base.Message = fmt.Sprintf("%s exited non-zero (code %d)", h.runtime, exitErr.ExitCode())
		} else {
			base.Message = fmt.Sprintf("%s run failed: %v", h.runtime, waitErr)
		}
		base.Kind = WorkerBlocked
		base.Err = waitErr
		return base
	}

	// Clean exit: completion requires non-empty evidence.
	finalText := ""
	if st != nil {
		finalText = st.finalText
	}
	if strings.TrimSpace(finalText) != "" {
		base.Kind = WorkerCompleted
		base.Summary = strings.TrimSpace(finalText)
		return base
	}
	// Clean exit, nothing produced -> idle.
	base.Kind = WorkerIdle
	return base
}

// ---------------------------------------------------------------------------
// Pi NDJSON record parsing. Stateful: the parser accumulates the assistant's
// final text so decideTerminal can tell completion from idle.
// ---------------------------------------------------------------------------

// piState accumulates the bits of a pi run needed to classify its outcome.
type piState struct {
	runtime    string
	sawAgentEnd bool
	finalText   string
}

// parsePiRecord updates st from one pi NDJSON record and returns a WorkerStep
// when the record carries visible progress (assistant text). Lifecycle markers
// (agent_start/agent_end/agent_settled) only update state; they do not map to an
// agent.* topic here — agent.activated is emitted by ActivateTask, and the
// terminal agent.idle/agent.blocked are emitted by the supervisor from the
// final WorkerEvent.
func parsePiRecord(rec map[string]any, st *piState, task HarnessTask, profile AgentProfile) *WorkerEvent {
	typ, _ := rec["type"].(string)
	switch typ {
	case "agent_end", "agent_settled":
		st.sawAgentEnd = true
		// agent_end may carry the final messages array; prefer its last assistant
		// message text as the completion summary.
		if msgs, ok := rec["messages"].([]any); ok {
			st.finalText = lastAssistantText(msgs)
		}
	case "turn_end":
		// turn_end carries the final assistant message for the turn.
		if msg, ok := rec["message"].(map[string]any); ok {
			if t := messageText(msg); t != "" {
				st.finalText = t
			}
		}
	case "message_end":
		// Capture the assistant message text when it completes.
		if msg, ok := rec["message"].(map[string]any); ok {
			if role, _ := msg["role"].(string); role == "assistant" {
				if t := messageText(msg); t != "" {
					st.finalText = t
					return &WorkerEvent{Kind: WorkerStep, TaskID: task.ID, AgentName: profile.Name, Layer: profile.Layer, Message: t}
				}
			}
		}
	case "message_update":
		// Stream visible assistant text deltas as steps.
		if ev, ok := rec["assistantMessageEvent"].(map[string]any); ok {
			if sub, _ := ev["type"].(string); sub == "text_end" {
				if c, _ := ev["content"].(string); strings.TrimSpace(c) != "" {
					return &WorkerEvent{Kind: WorkerStep, TaskID: task.ID, AgentName: profile.Name, Layer: profile.Layer, Message: c}
				}
			}
		}
	}
	return nil
}

// messageText extracts concatenated text from a pi message object.
func messageText(msg map[string]any) string {
	parts, ok := msg["content"].([]any)
	if !ok {
		return ""
	}
	var b strings.Builder
	for _, p := range parts {
		seg, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := seg["type"].(string); t == "text" {
			if txt, _ := seg["text"].(string); txt != "" {
				b.WriteString(txt)
			}
		}
	}
	return b.String()
}

// lastAssistantText returns the text of the last assistant message in a pi
// messages array (used by the agent_end record).
func lastAssistantText(msgs []any) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		msg, ok := msgs[i].(map[string]any)
		if !ok {
			continue
		}
		if role, _ := msg["role"].(string); role == "assistant" {
			return messageText(msg)
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// helpers: prompt, session identity, session dir
// ---------------------------------------------------------------------------

func taskPrompt(task HarnessTask) string {
	if task.Body != "" {
		return task.Title + "\n\n" + task.Body
	}
	return task.Title
}

var nonID = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// durableSessionID derives a stable session id for the agent's durable identity,
// keyed by project + assignee. Stable across re-activations so the worker
// resumes the same pi session.
func durableSessionID(task HarnessTask) string {
	seed := task.ProjectID + "-" + task.Assignee
	return nonID.ReplaceAllString(seed, "-")
}

// ensureSessionDir creates and returns the project-scoped session directory.
// Prefers <workspace>/.agent-orchestrator/sessions; falls back to a per-project
// dir under the OS temp dir when no workspace path is set.
func ensureSessionDir(task HarnessTask) (string, error) {
	var dir string
	if task.WorkspacePath != "" {
		dir = filepath.Join(task.WorkspacePath, ".agent-orchestrator", "sessions")
	} else {
		dir = filepath.Join(os.TempDir(), "agent-orchestrator", task.ProjectID, "sessions")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// compile-time guard: devHarness must satisfy Harness.
var _ Harness = (*devHarness)(nil)
