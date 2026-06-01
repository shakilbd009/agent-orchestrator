# contracts/events.md — Async / Event Contract

**Project:** agent-orchestrator
**Phase:** 1 — BRD-02 Platform-Native Orchestration Pipeline
**Status:** Canonical — per BRD-02 (FR-02-022, FR-02-022A, FR-02-023, FR-02-024, FR-02-025, FR-02-028, FR-02-032)

---

## Purpose

This document records the async / event contract boundaries for the Agent Orchestrator Platform. All audit and integration events use the canonical envelope defined in FR-02-022A. Event transport is provided via two channels: a project-scoped Server-Sent Events (SSE) stream for first-party dashboard clients (FR-02-023) and registered outbound webhooks for external consumers (FR-02-024/025).

---

## Schema Versioning Convention

Orchestration event schemas use **v1alpha** for Phase 1 and Phase 2 implementations per OQ-291. Consumers SHOULD treat v1alpha as unstable and expect field additions or changes within the v1alpha lineage. A transition to `v1` will be announced via an ADR and will include a migration guide for existing consumers.

Schema version appears in every event envelope as the `schemaVersion` field. Consumers MUST tolerate the presence of additional fields not defined in their current schema version.

---

## Canonical Event Envelope

Every audit and integration event MUST use the following 11-field envelope:

| Field | Type | Description |
|-------|------|-------------|
| `eventId` | `string` (UUIDv4) | Globally unique event identifier; assigned by the platform at emission time |
| `schemaVersion` | `string` | Schema identifier for this event family; format `v1alpha` through Phase 2, e.g. `v1alpha` |
| `projectId` | `string` | Project-scoped ID for this event; required on all project/task/gate/handoff events |
| `topic` | `string` | Event topic string used as SSE `event` field; dot-separated hierarchical type, e.g. `task.created`, `gate.approved` |
| `actorId` | `string` | Principal ID of the actor who triggered this event; `system` when emitted by a background process |
| `actorRole` | `string` | Role of the actor at emission time: `human`, `layer_a`, `layer_b`, or `system` |
| `taskId` | `string \| null` | Related task ID if the event pertains to a specific task; null otherwise |
| `parentTaskId` | `string \| null` | Parent task ID for child task events; null for non-child events |
| `gateId` | `string \| null` | Related gate ID for gate lifecycle events; null otherwise |
| `timestamp` | `string` (ISO 8601) | UTC timestamp assigned at event emission; format `YYYY-MM-DDTHH:mm:ss.SSSZ` |
| `payload` | `object` | Event-type-specific payload; structure defined per topic below |

Fields that are not applicable to an event type SHOULD be set to `null` rather than omitted, unless the event schema for that topic explicitly omits the field.

---

## Envelope + Payload Pattern

Every event topic defines a specific `payload` shape. The envelope fields are always present; the payload varies by topic.

Example — `TaskCreated` event:

```json
{
  "eventId": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "schemaVersion": "v1alpha",
  "projectId": "proj_01J9X",
  "topic": "task.created",
  "actorId": "agent_planner_a",
  "actorRole": "layer_a",
  "taskId": "t_abc1234",
  "parentTaskId": "t_parent_xyz",
  "gateId": null,
  "timestamp": "2026-05-28T14:32:00.000Z",
  "payload": {
    "taskType": "feature",
    "assignee": "layer_b_specialist",
    "title": "Implement user auth endpoint",
    "executionStatus": "todo",
    "layer": "B"
  }
}
```

---

## Layer A vs Layer B Event Responsibilities

| Layer | Event Responsibility | Notes |
|-------|---------------------|-------|
| Layer A (orchestrator) | Emits task lifecycle events, gate transition events, decomposition events, agent handoff events | These drive the Kanban state machine and audit trail |
| Layer B (specialist) | Emits task completion signals with structured handoff evidence, quality gate delivery events | These feed into Layer A gate enforcement |
| Platform | Emits health, status, feature flag change events | These are infrastructure-level only |

---

## Event Topic Registry

All topics follow the pattern `<entity>.<action>` or `<entity>.<subcategory>.<action>`.

### Task Lifecycle Events

#### `task.created`
Platform emits when an active task is created under a project.

```json
{
  "payload": {
    "taskType": "string",
    "assignee": "string | null",
    "title": "string",
    "executionStatus": "todo | in_progress | blocked | done | cancelled",
    "layer": "A | B",
    "required": "boolean",
    "staleThresholdMinutes": "number | null"
  }
}
```

#### `task.status.changed`
Platform emits when task execution status changes.

```json
{
  "payload": {
    "fromStatus": "string",
    "toStatus": "string",
    "reason": "string | null"
  }
}
```

#### `task.stale.detected`
Platform emits when a task exceeds its configured stale threshold without status mutation.

```json
{
  "payload": {
    "staleThresholdMinutes": "number",
    "lastActivityAt": "string (ISO 8601)"
  }
}
```

#### `task.blocked`
Platform emits when a task transitions to `blocked` explicitly.

```json
{
  "payload": {
    "blockedBy": "string",
    "reason": "string"
  }
}
```

#### `task.cancelled`
Platform emits when a task is cancelled.

```json
{
  "payload": {
    "cancelledBy": "string",
    "wasRequiredForParent": "boolean",
    "reason": "string | null"
  }
}
```

#### `task.completed`
Platform emits when a task reaches `done` with Layer B structured handoff evidence.

```json
{
  "payload": {
    "completedBy": "string",
    "summary": "string",
    "artifacts": "string[]",
    "validationPerformed": "string",
    "risksOrResidualIssues": "string | null",
    "recommendedNextGate": "string | null"
  }
}
```

### Decomposition Events

#### `task.decomposition.proposed`
Layer A or human actor proposes child tasks for a parent.

```json
{
  "payload": {
    "parentTaskId": "string",
    "proposedChildren": [
      {
        "taskId": "string",
        "title": "string",
        "taskType": "string",
        "assignee": "string | null",
        "layer": "A | B",
        "required": "boolean"
      }
    ],
    "depthAtProposal": "number",
    "activeChildrenCount": "number"
  }
}
```

#### `task.decomposition.approved`
Proposed decomposition is approved; children become active tasks.

```json
{
  "payload": {
    "parentTaskId": "string",
    "approvedBy": "string",
    "approvedByRole": "human | layer_a",
    "childTaskIds": ["string"],
    "depthUsed": "number"
  }
}
```

#### `task.decomposition.rejected`
Proposed decomposition is rejected with reason.

```json
{
  "payload": {
    "parentTaskId": "string",
    "rejectedBy": "string",
    "rejectionReason": "string",
    "proposalRetained": "boolean"
  }
}
```

#### `task.decomposition.override_used`
Project decomposition limits are overridden.

```json
{
  "payload": {
    "parentTaskId": "string",
    "overrideBy": "string",
    "overrideType": "depth | fan_out | both",
    "oldLimit": "number",
    "newLimit": "number",
    "auditReason": "string"
  }
}
```

### Gate Lifecycle Events

#### `gate.opened`
A project-level or task-level gate opens.

```json
{
  "payload": {
    "gateType": "scope_review | architecture_review | implementation_review | code_review | qa_review | release_review | phase_gate",
    "gateLevel": "project | task",
    "blocking": "boolean",
    "openedBy": "string"
  }
}
```

#### `gate.approved`
A gate is approved.

```json
{
  "payload": {
    "approvedBy": "string",
    "approverRole": "human | layer_a",
    "gateType": "string",
    "gateLevel": "project | task"
  }
}
```

#### `gate.rejected`
A gate is rejected with reason.

```json
{
  "payload": {
    "rejectedBy": "string",
    "rejectionReason": "string",
    "gateType": "string",
    "gateLevel": "project | task"
  }
}
```

### Agent Events

#### `agent.activated`
Agent begins work on a task.

```json
{
  "payload": {
    "agentName": "string",
    "layer": "A | B",
    "taskId": "string | null"
  }
}
```

#### `agent.idle`
Agent has no active task assignment.

```json
{
  "payload": {
    "agentName": "string",
    "layer": "A | B",
    "idleSince": "string (ISO 8601) | null"
  }
}
```

#### `agent.blocked`
Agent cannot proceed due to a task-level blocker.

```json
{
  "payload": {
    "agentName": "string",
    "layer": "A | B",
    "taskId": "string",
    "reason": "string"
  }
}
```

### Handoff Events

#### `handoff.submitted`
Layer B submits structured completion evidence for an assigned task.

```json
{
  "payload": {
    "taskId": "string",
    "submittedBy": "string",
    "summary": "string",
    "artifacts": "string[]",
    "validationPerformed": "string",
    "risksOrResidualIssues": "string | null",
    "recommendedNextGate": "string | null"
  }
}
```

### Project Events

#### `project.created`
A new project is created.

```json
{
  "payload": {
    "projectName": "string",
    "createdBy": "string",
    "staleThresholdMinutes": "number",
    "decompositionDepthDefault": "number",
    "decompositionFanOutDefault": "number"
  }
}
```

### Platform Events

#### `platform.health.changed`
Platform health status changes.

```json
{
  "payload": {
    "fromStatus": "healthy | degraded | maintenance",
    "toStatus": "healthy | degraded | maintenance",
    "changedBy": "string"
  }
}
```

#### `feature_flag.changed`
A feature flag value changes.

```json
{
  "payload": {
    "flagName": "string",
    "oldValue": "boolean",
    "newValue": "boolean",
    "changedBy": "string"
  }
}
```

### Audit / Security Events

#### `auth.mutation.denied`
Unauthorized mutation attempt is rejected.

```json
{
  "payload": {
    "actorId": "string",
    "actorRole": "string",
    "attemptedAction": "string",
    "deniedReason": "string"
  }
}
```

#### `webhook.delivery.failed`
Webhook exhausts all retry attempts.

```json
{
  "payload": {
    "webhookId": "string",
    "eventTopic": "string",
    "attemptCount": "number",
    "lastError": "string",
    "eventId": "string"
  }
}
```

### SSE Client Events

#### `sse.client.connected`
SSE client connects to a project event stream.

```json
{
  "payload": {
    "clientId": "string",
    "projectId": "string",
    "remoteAddr": "string"
  }
}
```

#### `sse.client.disconnected`
SSE client disconnects from a project event stream.

```json
{
  "payload": {
    "clientId": "string",
    "projectId": "string",
    "disconnectReason": "string | null"
  }
}
```

---

## SSE Event Stream Specification

**Endpoint:** `GET /projects/{projectId}/events/stream`

**Authentication:** Requires valid project-scoped session token.

**SSE Fields:**

| SSE Field | Source |
|-----------|--------|
| `event` | `topic` from the canonical envelope |
| `id` | `eventId` from the canonical envelope |
| `data` | Full JSON-serialized canonical envelope |
| `retry` | Server-suggested reconnect interval in milliseconds (default: 5000) |

**Behavior (FR-02-023):**
- Connected clients receive all committed orchestration events for their project without polling.
- Events are delivered in emission order.
- The platform MUST deliver events within 2 seconds of commitment (NFR-02-008).

**Reconnect / Catch-up (FR-02-023A):**
- Clients MAY reconnect with an `Last-Event-ID` header containing the last `eventId` received.
- The platform SHOULD replay missed project events from the immutable audit log to the reconnecting client.
- Catch-up replay does not imply event replay as the operational source of truth — it only provides dashboard continuity.
- The platform MAY limit replay to events from the last 24 hours or a configurable window.

**Client Example:**
```
GET /projects/proj_01J9X/events/stream
Headers:
  Authorization: Bearer <token>
  Accept: text/event-stream
```

```
event: task.created
id: f47ac10b-58cc-4372-a567-0e02b2c3d479
data: {"eventId":"f47ac10b-58cc-4372-a567-0e02b2c3d479","schemaVersion":"v1alpha","projectId":"proj_01J9X","topic":"task.created","actorId":"agent_planner_a","actorRole":"layer_a","taskId":"t_abc1234","parentTaskId":null,"gateId":null,"timestamp":"2026-05-28T14:32:00.000Z","payload":{...}}

event: gate.approved
id: 7c9e6679-7425-40de-944b-e07fc1f90ae7
data: {"eventId":"7c9e6679-7425-40de-944b-e07fc1f90ae7","schemaVersion":"v1alpha","projectId":"proj_01J9X","topic":"gate.approved","actorId":"h_human_01","actorRole":"human","taskId":"t_abc1234","parentTaskId":null,"gateId":"g_task_scope_01","timestamp":"2026-05-28T14:35:00.000Z","payload":{...}}
```

---

## Webhook Delivery Specification

**Registration:** Projects MAY register outbound webhook consumers via the webhook registration API. Registration specifies:
- `webhookUrl`: Target URL for delivery
- `eventSelector`: Event topic or prefix pattern to subscribe to (e.g., `task.*`, `gate.approved`, `task.decomposition.*`)
- `secret`: Required for non-dev webhook URLs (localhost/127.0.0.1 exempt). Used for HMAC-SHA256 request signing.

**Delivery (FR-02-024):**
- Webhook delivery is asynchronous and MUST NOT block or roll back the originating task/gate state change.
- Delivery latency target: webhook jobs enqueued within 1 second of the committed event (NFR-02-009).
- Delivery is fire-and-forget from the perspective of the originating mutation — webhook failure MUST NOT roll back committed state (NFR-02-010).

**Retry Policy (FR-02-025):**
- Failed webhook deliveries retry with exponential backoff: 1s, 2s, 4s, 8s, ... up to the configured retry limit.
- Default retry count: 3 attempts.
- Exhausted deliveries are logged as `webhook.delivery.failed`, increment the `orch_webhook_delivery_failed_total` metric, and are visible in project audit/event views.

**Request Shape:**
```
POST <webhookUrl>
Headers:
  Content-Type: application/json
  X-Event-Id: <eventId>
  X-Event-Topic: <topic>
  X-Project-Id: <projectId>
  X-Delivery-Attempt: <attempt number>
  X-Webhook-Signature: <HMAC-SHA256(body, secret)>  [non-dev URLs always; localhost/127.0.0.1 exempt per ADR-02-003]
Body: <canonical envelope JSON>
```

**Response Handling:**
- HTTP 2xx within 10 seconds: delivery considered successful.
- HTTP non-2xx or timeout: delivery considered failed and eligible for retry.
- Connection errors (DNS failure, connection refused): eligible for retry.

---

## Feature Flag Dependencies

Event emission is gated by the following feature flags (from `specs/feature-flags.md`):

| Flag | Controls |
|------|---------|
| `platform-orchestration` (master gate) | All project/task/gate/orchestration events; must be `true` for any BRD-02 event to emit. Sub-capabilities continue to respect their own flags. |
| `layer-a-agents` | `AgentActivated`, `AgentIdle`, `agent.blocked` for Layer A agents; `task.decomposition.proposed` |
| `layer-b-agents` | `AgentActivated`, `AgentIdle`, `agent.blocked` for Layer B agents; `task.completed`, `handoff.submitted` |
| `human-gates` | `GateOpened`, `GateApproved`, `GateRejected` for human-controlled gates |
| `audit-trail` | All immutable audit event persistence and audit/event query APIs |
| `feature-flags` | `FeatureFlagChanged` events |

**`platform-orchestration` as master gate (FR-02-028):**
- When `platform-orchestration=false`, the platform rejects or hides all project-scoped orchestration capabilities, including the SSE stream and webhook delivery.
- `kanban-orchestrator` is a legacy/compatibility flag from Phase 0. New platform-native behavior is exclusively controlled by `platform-orchestration`.
- Sub-capability flags (`layer-a-agents`, `layer-b-agents`, `human-gates`, `audit-trail`) remain as secondary gates within the enabled orchestration surface.

---

## Explicit Non-Applicability

The following are intentionally **not defined** in BRD-02 and are blocked on their respective BRDs:

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

<<<<<<< Updated upstream
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

=======
>>>>>>> Stashed changes
## Audit Trail Contract

Every event in this system is an **immutable audit record**. The event store is append-only. No event may be retracted, rewritten, or reordered after emission.

Immutable audit event persistence is gated by the `audit-trail` feature flag. When `audit-trail=false`, the platform MAY omit audit event persistence but MUST NOT roll back or modify any committed task/gate/handoff state.

---

## BRD-02 Phase 1 Summary

| Item | Status |
|------|--------|
| Canonical 11-field event envelope | Defined — per FR-02-022A |
| Envelope + payload pattern | Defined — all topics use envelope+payload |
| Schema versioning convention | v1alpha for Phase 1/2 — per OQ-291 |
| SSE event stream spec | Defined — `GET /projects/{projectId}/events/stream`, FR-02-023 |
| SSE reconnect / catch-up | Defined — `Last-Event-ID` header, FR-02-023A |
| Webhook delivery spec | Defined — async, non-blocking, FR-02-024 |
| Webhook retry spec | Defined — exponential backoff, default 3, FR-02-025 |
| Master flag gating | `platform-orchestration` as master; `kanban-orchestrator` legacy |
| Layer A / Layer B event ownership | Defined |
| Task lifecycle event shapes | Canonical — envelope+payload |
| Decomposition event shapes | Canonical — envelope+payload |
| Gate lifecycle event shapes | Canonical — envelope+payload |
| Agent event shapes | Canonical — envelope+payload |
| Handoff event shapes | Canonical — envelope+payload |
| Platform event shapes | Canonical — envelope+payload |
| Audit/security event shapes | Canonical — envelope+payload |
| LLM inference events | Not applicable — deferred to BRD-05 |
| Memory store events | Not applicable — deferred to BRD-06 |
| Notification events | Not applicable — deferred to BRD-21 |
| Audit trail immutability | Deferred to BRD-02 implementation + BRD-19 |

---

*This document reflects BRD-02 canonical event contracts. All event shapes use the canonical 11-field envelope and envelope+payload pattern. Event transport is provided via SSE stream and outbound webhooks per FR-02-023, FR-02-024, and FR-02-025.*