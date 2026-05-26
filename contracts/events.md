# contracts/events.md — Async / Event Contract Placeholder

**Project:** agent-orchestrator
**Phase:** 0 — Governance & Scaffolding
**Status:** Placeholder (events not defined until BRDs are authored)

---

## Purpose

This document records the async / event contract boundaries for the Agent Orchestrator Platform. It establishes what is intentionally deferred, what is explicitly not applicable, and the placeholder shapes that downstream systems may rely on during Phase 0.

---

## Layer A vs Layer B Event Responsibilities

| Layer | Event Responsibility | Notes |
|-------|---------------------|-------|
| Layer A (orchestrator) | Emits task lifecycle events, gate transition events, agent handoff events | These drive the Kanban state machine and audit trail |
| Layer B (specialist) | Emits task completion signals, quality gate delivery events | These feed into Layer A gate enforcement |
| Platform | Emits health, status, feature flag change events | These are infrastructure-level only |

---

## Placeholder Event Shapes

The following event schemas are stubs. They define the event envelope structure only — field semantics are filled with placeholder descriptions. Full field definitions require their respective BRDs.

### Task Lifecycle Events

```
TaskCreated
  taskId:        string           # Kanban task ID
  parentId:      string | null    # Parent task ID if this is a child decomposition
  layer:         "A" | "B"
  assignee:      string
  timestamp:     ISO 8601

TaskPromoted
  taskId:        string
  fromGate:      string           # Gate ID (e.g. "G0", "G1")
  toGate:        string
  promotedBy:    string           # Agent profile name
  timestamp:     ISO 8601

TaskCompleted
  taskId:        string
  completedBy:   string           # Agent profile name
  summary:       string           # Human-readable handoff summary
  metadata:      object           # Arbitrary key/value from the completing agent
  timestamp:     ISO 8601

TaskBlocked
  taskId:        string
  blockedBy:     string
  reason:        string           # One-sentence blocker description
  timestamp:     ISO 8601
```

### Quality Gate Events

```
GateOpened
  gateId:        string           # e.g. "G0", "G1"
  taskId:        string           # Task that opened this gate
  timestamp:     ISO 8601

GateApproved
  gateId:         string
  approvedBy:    string           # Human principal or "architect" agent
  taskId:        string
  timestamp:     ISO 8601

GateRejected
  gateId:         string
  rejectedBy:    string
  taskId:        string
  feedback:      string           # Human or agent rejection reason
  timestamp:     ISO 8601
```

### Agent Events

```
AgentActivated
  agentName:     string
  layer:         "A" | "B"
  taskId:        string | null
  timestamp:     ISO 8601

AgentIdle
  agentName:     string
  layer:         "A" | "B"
  timestamp:     ISO 8601

AgentBlocked
  agentName:     string
  layer:         "A" | "B"
  taskId:        string
  reason:        string
  timestamp:     ISO 8601
```

### Platform Events

```
PlatformHealthChanged
  status:        "healthy" | "degraded" | "maintenance"
  timestamp:     ISO 8601

FeatureFlagChanged
  flagName:      string           # Matches keys in specs/feature-flags.md
  oldValue:      boolean
  newValue:      boolean
  changedBy:     string
  timestamp:     ISO 8601
```

---

## Explicit Non-Applicability

The following are intentionally **not defined** in Phase 0 and are blocked on their respective BRDs:

| Not Defined | Blocked On | Reason |
|------------|------------|--------|
| LLM inference event schemas (token usage, latency, errors) | BRD-05 (LLM Provider) | LLM provider abstraction not finalized |
| Agent memory read/write event contracts | BRD-06 (Agent Memory) | Memory store design not defined |
| BRD authoring workflow events | BRD-07 (BRD Authoring) | BRD lifecycle not designed |
| Quality gate automated enforcement events | BRD-08 (Quality Gates) | Automated gate logic deferred |
| Code review webhook / async events | BRD-09 (Code Review) | Review workflow not designed |
| Security scan async events | BRD-10 (Security Review) | Security tooling not selected |
| QA test run / pass / fail events | BRD-11 (QA Automation) | QA framework not selected |
| Deployment pipeline events | BRD-12 (Deployment Pipeline) | CI/CD not designed |
| Playwright test report events | BRD-14 (Playwright Testing) | Testing framework deferred |
| Collaboration (comment, mention, assignment) events | BRD-18 (Collaboration) | Collaboration features deferred |
| Notification delivery events | BRD-21 (Notifications) | Notification system deferred |

---

## Event Transport

**Phase 0 stance:** Event transport is intentionally unspecified.

Possible transports for future BRDs to evaluate:

| Transport | Candidates | Trade-off |
|-----------|-----------|-----------|
| Webhook | External receivers, CI systems | Simple, stateless |
| Message queue | Async workers, internal consumers | Decoupled, durable |
| Server-Sent Events (SSE) | Dashboard real-time updates | Unidirectional, HTTP-only |
| WebSocket | Interactive dashboards | Bidirectional, stateful |
| Polling | Status endpoints | Simplest; no infrastructure needed |

No transport decision is made here. BRD-02 (Orchestration Pipeline) will define the event bus topology as part of the task dependency graph implementation.

---

## Feature Flag Dependencies

Event emission is gated by the following feature flags (from `specs/feature-flags.md`):

| Flag | Controls |
|------|---------|
| `kanban-orchestrator` | All task lifecycle events (`TaskCreated`, `TaskPromoted`, `TaskCompleted`, `TaskBlocked`) |
| `layer-a-agents` | `AgentActivated`, `AgentIdle` for Layer A agents |
| `layer-b-agents` | `AgentActivated`, `AgentIdle` for Layer B agents |
| `human-gates` | `GateOpened`, `GateApproved`, `GateRejected` |
| `feature-flags` | `FeatureFlagChanged` events |

---

## Audit Trail Contract

Every event in this system is an **immutable audit record**. The event store must be append-only. No event may be retracted or rewritten after emission.

This contract is not enforced in Phase 0. Enforcement is blocked on:
- BRD-02 (Orchestration Pipeline) — defines the event store
- BRD-19 (Change History) — defines retention and immutability rules

---

## Phase 0 Summary

| Item | Status |
|------|--------|
| Event envelope schema (topic, taskId, timestamp, source) | Placeholder — defined above |
| Layer A / Layer B event ownership | Defined above |
| Task lifecycle event shapes | Placeholder |
| Quality gate event shapes | Placeholder |
| Agent event shapes | Placeholder |
| Platform event shapes | Placeholder |
| Event transport mechanism | **Not defined** — deferred to BRD-02 |
| LLM inference events | **Not applicable** — deferred to BRD-05 |
| Memory store events | **Not applicable** — deferred to BRD-06 |
| Notification events | **Not applicable** — deferred to BRD-21 |
| Audit trail immutability | **Not enforced** — deferred to BRD-19 |

---

*This document is a Phase 0 placeholder. Do not implement event handlers or consumers based on this contract without a BRD defining the event semantics and transport.*
