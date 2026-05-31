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

## BRD-03 — Client Portal Events

*BRD-03 Client Portal introduces client-facing events for approvals, publications, comments, and access enforcement. These events are emitted by the BFF layer and by the platform on behalf of client actions. SSE is the primary transport for live portal updates.*

### Client Approval Events

```
ApprovalSubmitted
  eventType:    "client_portal.approval.submitted"
  projectId:    string
  itemId:       string              # Related task/approval item ID
  itemTitle:    string
  outcome:      "approve" | "reject" | "request_changes" | "need_more_information"
  actorId:      string              # Internal — stripped before client SSE delivery
  actorName:    string              # Client display name
  actorRole:    string              # Internal — stripped before client SSE delivery
  comment:      string | null
  timestamp:    ISO 8601
  # Envelope fields (stripped from client-visible SSE):
  #   actorId, actorRole, eventId, schemaVersion, parentTaskId, gateId

ApprovalOutcomeRecorded
  eventType:    "client_portal.approval.outcome_recorded"
  projectId:    string
  itemId:       string
  outcome:      "approved" | "rejected" | "request_changes" | "need_more_information"
  timestamp:    ISO 8601
  # Envelope fields stripped from client-visible SSE

NeedMoreInformationRequested
  eventType:    "client_portal.approval.need_more_information"
  projectId:    string
  itemId:       string
  question:     string
  timestamp:    ISO 8601
  # Places item in waiting-for-response state; does not count as rejection
  # Envelope fields stripped from client-visible SSE
```

### Publication Events

```
ItemPublished
  eventType:    "client_portal.item.published"
  projectId:    string
  itemId:        string
  itemType:      "task" | "risk" | "milestone" | "blocker"
  publishedBy:   string              # Internal actor identity
  validationResult: "passed" | "failed"
  timestamp:    ISO 8601
  # Envelope fields stripped from client-visible SSE

ItemUnpublished
  eventType:    "client_portal.item.unpublished"
  projectId:    string
  itemId:        string
  unpublishedBy: string
  reason:        string
  timestamp:     ISO 8601
  # Envelope fields stripped from client-visible SSE

PublicationValidationFailed
  eventType:    "client_portal.publication_validation.failed"
  projectId:    string
  itemId:        string
  failureReason: string              # Category only — not raw forbidden content
  timestamp:    ISO 8601
  # FR-03-048: items failing validation stay hidden; failureReason must not expose raw content
```

### Comment Events

```
CommentCreated
  eventType:    "client_portal.comment.created"
  projectId:    string
  relatedItemId: string              # Task, risk, milestone, or approval ID
  itemType:      "task" | "risk" | "milestone" | "approval"
  commentId:    string
  authorName:    string
  body:          string
  timestamp:     ISO 8601
  # Envelope fields stripped from client-visible SSE

CommentEdited
  eventType:    "client_portal.comment.edited"
  projectId:    string
  commentId:    string
  editedBy:      string              # Author name
  editedAt:      ISO 8601
  # Audit: previous body text not retained (FR-03-029)
  # Envelope fields stripped from client-visible SSE

CommentDeleted
  eventType:    "client_portal.comment.deleted"
  projectId:    string
  commentId:    string
  deletedBy:     string              # Author name
  deletedAt:     ISO 8601
  # Audit: deleted body text not retained (FR-03-029)
  # Deleted comments hidden from normal client view
  # Envelope fields stripped from client-visible SSE
```

### Access and Portal Health Events

```
AccessDenied
  eventType:    "client_portal.access.denied"
  principalId:   string
  resourceType:  "project" | "task" | "risk" | "milestone" | "approval" | "comment"
  resourceId:    string
  timestamp:     ISO 8601
  # NFR-03-013: access boundary failures treated as security defects
  # Envelope fields stripped from client-visible SSE

PortalReadOnlyModeEntered
  eventType:    "client_portal.read_only_mode.entered"
  reason:       string              # "submission_unavailable" | "read_api_degraded"
  timestamp:    ISO 8601
  # FR-03-053: read-only degraded mode when submission unavailable but reads work

PortalReadsUnavailable
  eventType:    "client_portal.reads.unavailable"
  endpoint:     string              # Which read API became unavailable
  timestamp:    ISO 8601
  # NFR-03-009: show unavailable state, do not present stale data as current

PortalSSEConnected
  eventType:    "client_portal.sse.connected"
  projectId:    string
  timestamp:    ISO 8601

PortalSSEDisconnected
  eventType:    "client_portal.sse.disconnected"
  projectId:    string
  reason:       string
  timestamp:    ISO 8601
  # FR-03-041: "live updates paused" shown; manual refresh available
```

### BRD-03 Feature Flag Dependency

| Flag | Controls |
|------|----------|
| `client-portal` | All BRD-03 events above |

*BRD-03 SSE envelope strip requirement: The BFF MUST strip `actorId`, `actorRole`, `eventId`, `schemaVersion`, `parentTaskId`, `gateId`, and `layer` from all SSE event payloads before client subscription delivery (ADR-03-003, OQ-03-002 resolved).*

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
