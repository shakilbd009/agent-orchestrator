# BRD-02: Platform-Native Orchestration Pipeline

> **Status:** Approved for Curation

---

## Overview

The platform provides native, project-scoped orchestration for tasks, dependencies, gates, agent handoffs, audit events, dashboard updates, and external event delivery. The Go/Echo backend owns operational orchestration state; Hermes Kanban is not the runtime source of truth for this product capability. This BRD defines the governed human/agent workflow required for Layer A planning, Layer B execution, strict gate enforcement, and observable project progress.

---

## User Stories

| ID | As a | I want | So that |
|----|------|--------|---------|
| US-02-001 | Human orchestrator | Create project-scoped tasks and view them on a project board | Work is organized by project rather than one global task queue |
| US-02-002 | Human orchestrator | See dependencies, child tasks, gates, and stale work for a project | I can understand orchestration progress and intervene before work stalls |
| US-02-003 | Layer A agent | Propose decomposed child tasks for a parent task | Complex work can be broken down without immediately mutating active execution scope |
| US-02-004 | Human or authorized Layer A approver | Review and approve proposed child tasks before activation | Decomposition is controlled, auditable, and aligned with project scope |
| US-02-005 | Layer B agent | Update only my assigned task and submit structured handoff evidence | Specialists can report completion without bypassing governance gates |
| US-02-006 | Human approver | Approve or reject project-level phase gates | Major phase transitions remain under human control |
| US-02-007 | Human or authorized Layer A approver | Approve or reject task-level gates | Task outputs cannot be accepted without the required review path |
| US-02-008 | Dashboard user | Receive real-time project board updates | I can monitor orchestration without manually refreshing the board |
| US-02-009 | Integration owner | Subscribe project-level webhook consumers to orchestration events | External systems can react to task, gate, and handoff changes |
| US-02-010 | System operator | Observe pipeline health and infrastructure health metrics | I can detect stalled work, broken event delivery, and service degradation |

---

## Functional Requirements

### Must Have

- **FR-02-001: Project-scoped orchestration.** The platform MUST scope tasks, gates, dependency graphs, event streams, webhook registrations, and board views to a project. A task MUST belong to exactly one project.
- **FR-02-002: Current-state source of truth.** The operational source of truth MUST be durable current-state records for projects, tasks, dependencies, gates, assignments, and handoff evidence. The event log MUST be immutable and append-only for audit and integration, but BRD-02 MUST NOT require event replay as the primary state reconstruction mechanism.
- **FR-02-003: Task execution status.** Tasks MUST use the execution statuses `todo`, `in_progress`, `blocked`, `done`, and `cancelled`. Execution status MUST describe work progress only, not gate approval or decomposition proposal state.
- **FR-02-004: Separate governance/control state.** The platform MUST model governance state separately from execution status, including decomposition state, gate state, stale state, requiredness of child tasks, and approval state.
- **FR-02-005: Dependency graph.** Tasks MAY have parent/child dependency relationships within the same project. The platform MUST reject circular dependencies and cross-project parent/child edges.
- **FR-02-006: Required child semantics.** A child task MUST indicate whether it is required for parent completion. Required children MUST reach `done` before the parent can reach `done`, unless an authorized human or Layer A actor explicitly changes the child to non-required or replaces the child through an audited scope adjustment.
- **FR-02-007: Strict parent completion.** A parent task MUST NOT reach `done` while any required child is unresolved, cancelled without approved scope reduction, or blocked. Parent completion MUST also require all blocking task-level gates to be approved.
- **FR-02-008: Assisted decomposition.** A human or Layer A actor MAY request decomposition for a parent task. The platform MUST create proposed child tasks in a proposal state; proposed children MUST NOT affect assignment, readiness, parent completion, or active board workload until approved.
- **FR-02-009: Decomposition approval.** Proposed decompositions MUST be approved by a human or authorized Layer A actor before proposed child tasks become active tasks. Rejected decomposition proposals MUST retain the proposal details and rejection reason for audit.
- **FR-02-009A: Decomposition idempotency.** A task MUST NOT have more than one active decomposition proposal at the same time. Re-submitting an equivalent decomposition request MUST NOT create duplicate proposals. Rejected proposals MAY be superseded by a new proposal. Approved proposals MUST NOT be silently replaced; changes after approval require an audited scope adjustment.
- **FR-02-010: Decomposition limits.** Default decomposition limits MUST be project-configurable with defaults of max depth 3 and max 20 active children per parent. Hard caps MUST be max depth 5 and max 50 active children per parent.
- **FR-02-011: Bounded decomposition overrides.** Human or authorized Layer A actors MAY override project default decomposition limits up to the hard caps. Every override MUST include an actor identity, timestamp, affected task, old limit, requested limit, and audit reason. No override MAY exceed hard caps.
- **FR-02-012: Role-scoped internal authorization.** Every mutating action MUST include an actor identity and role claim. Minimum role claims are `human`, `layer_a`, `layer_b`, and `system`.
- **FR-02-013: Layer A permissions.** Authorized Layer A actors MAY propose decomposition, approve decomposition where configured, route child tasks, manage task-level gates where authorized, and perform audited scope adjustments. Layer A actors MUST NOT approve project-level phase gates unless a human has delegated that authority explicitly in project configuration.
- **FR-02-014: Layer B permissions.** Layer B actors MAY update only tasks assigned to them, submit structured handoff evidence, and request blocking or review. Layer B actors MUST NOT approve their own gates, approve project-level gates, override decomposition limits, complete parent tasks by side effect, or mutate tasks outside their assignment scope.
- **FR-02-015: Structured Layer B handoff evidence.** Before a Layer B actor can mark an assigned task `done`, the platform MUST require structured handoff evidence containing completion summary, artifact references where relevant, validation performed, known risks or residual issues, recommended next gate or reviewer, actor identity, and timestamp.
- **FR-02-016: Project-level phase gates.** The platform MUST model project-level phase gates such as G0 Foundation & Governance, G1 App Shell, and G2 Core Delivery. Project-level phase advancement MUST require human approval.
- **FR-02-017: Task-level gates.** The platform MUST model task-level gates for individual task outputs. Default gate types MUST include `scope_review`, `architecture_review`, `implementation_review`, `code_review`, `qa_review`, and `release_review`.
- **FR-02-017A: Gates independent of decomposition.** A task-level gate MAY exist on a task with no child tasks. Gate enforcement MUST NOT depend on whether a task has been decomposed.
- **FR-02-018: Gate configuration.** Projects MUST be able to enable or disable default task-level gates, configure which task types require which gates, configure which roles can approve each gate, and mark gates as advisory or blocking. Custom new gate types are out of scope for BRD-02.
- **FR-02-019: Gate enforcement.** Blocking gates MUST prevent the guarded task or project phase from advancing until approved. Gate rejection MUST require a reason and MUST leave the guarded task or phase in a non-advanced state.
- **FR-02-020: Stale task detection.** The platform MUST detect stale tasks using configurable inactivity thresholds. Stale detection MUST emit events and surface warnings in project board/API state, but MUST NOT automatically change an `in_progress` task to `blocked`.
- **FR-02-021: Manual blocked state.** A task MAY transition to `blocked` only through an explicit action by a human, authorized Layer A actor, assigned Layer B actor, or system rule that is explicitly configured outside stale detection. Blocking MUST require a reason.
- **FR-02-022: Immutable audit event log.** Every project/task/gate/decomposition/handoff/webhook-relevant state change MUST append an immutable event record. Events MUST include event ID, schema version, project ID, topic, actor identity, actor role, timestamp, related task/gate IDs where applicable, and event payload.
- **FR-02-022A: Canonical event envelope.** Audit and integration events MUST use a canonical envelope containing `eventId`, `schemaVersion`, `projectId`, `topic`, `actorId`, `actorRole`, `taskId`, `parentTaskId`, `gateId`, `timestamp`, and `payload`. Fields that are not applicable to an event MAY be null or omitted according to the finalized event schema convention.
- **FR-02-023: SSE event stream.** The backend MUST expose a project-scoped SSE stream for first-party dashboard clients at `GET /projects/{projectId}/events/stream`. Connected clients MUST receive committed orchestration events without polling. SSE `event` MUST use the event `topic`, SSE `id` MUST use `eventId`, and SSE `data` MUST contain the full event envelope as JSON.
- **FR-02-023A: SSE reconnect/catch-up.** SSE clients SHOULD be able to reconnect with `Last-Event-ID` and receive missed project events from the immutable audit log. This catch-up behavior is for dashboard continuity and MUST NOT imply that event replay is the operational source of truth.
- **FR-02-024: Outbound webhooks.** Projects MUST support registered outbound webhook consumers by event type or event prefix, such as `task.*`, `gate.*`, `task.decomposition.*`, or exact event topics. Webhook delivery MUST be asynchronous and MUST NOT block or roll back task/gate state changes.
- **FR-02-025: Webhook retries.** Webhook delivery MUST retry failed attempts with exponential backoff up to a configurable retry limit. The default retry count MUST be 3 attempts. Exhausted delivery MUST be logged, counted, and visible in project audit/event views.
- **FR-02-026: Project board API.** The platform MUST expose project board/read APIs that return tasks grouped by execution status, dependency relationships, gate state, decomposition proposal state, stale indicators, assignments, and latest handoff summary where available.
- **FR-02-027: Retention and export.** Task records, gate records, audit events, and handoff evidence MUST be retained for the lifetime of the project. Project archival or deletion MUST require explicit authorization and MUST support export before destructive deletion.
- **FR-02-028: Feature-flag gating.** Platform-native orchestration MUST be gated by a new master feature flag, `platform-orchestration`, default `false`. Related sub-capabilities MUST continue to respect `layer-a-agents`, `layer-b-agents`, `human-gates`, and `audit-trail` where applicable.

### Should Have

- **FR-02-029: Legacy flag clarification.** The BRD should define `kanban-orchestrator` as legacy/compatibility naming and state that new platform-native behavior is controlled by `platform-orchestration`.
- **FR-02-030: Project defaults.** Projects should define defaults for stale thresholds, decomposition depth, decomposition fan-out, enabled gate templates, and default approver roles.
- **FR-02-031: Webhook signing.** Webhook delivery should support a per-webhook shared secret or signature mechanism if implementation cost is low; unsigned local-only webhooks may be acceptable for first internal use if clearly marked.
- **FR-02-032: Event schema version field.** Audit and integration events should include a schema version so downstream consumers can detect contract changes.
- **FR-02-033: Readiness details.** Readiness output should identify whether database access, event persistence, webhook queueing, and SSE fanout are available.

### Could Have

- Diagnostic event replay for comparing current-state records against the audit log.
- Manual webhook replay or dead-letter controls.
- Custom project-defined gate types.
- Detailed agent performance analytics such as per-agent cycle time and utilization.
- Enterprise multi-tenant policies, legal hold, or compliance-specific retention controls.
- Broker-backed event delivery for multi-host deployments.

### Out of Scope

- Full OIDC/JWT production authentication and external identity provider integration.
- Dedicated message broker infrastructure such as NATS, Kafka, or RabbitMQ.
- Event store as the primary operational source of truth.
- Required replay-on-startup state reconstruction.
- Automatic decomposition when a task enters `in_progress`.
- Automatic stale-task mutation to `blocked`.
- Hermes Kanban as the runtime orchestration source of truth.
- Agent performance dashboard details, owned by BRD-04 where applicable.
- LLM provider behavior, owned by BRD-05.
- Agent memory behavior, owned by BRD-06.
- Automated quality gate evaluation beyond configured blocking/approval rules, owned by BRD-08 where applicable.
- Notification UX beyond outbound webhook delivery, owned by BRD-21 where applicable.

---

## Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-02-001 | Human user scale | Support up to 10 human users in the initial internal platform target |
| NFR-02-002 | Agent scale | Support up to 25 active agents across Layer A and Layer B roles |
| NFR-02-003 | Project scale | Support up to 50 active projects |
| NFR-02-004 | Task scale | Support up to 10,000 tasks per project at target latency |
| NFR-02-005 | SSE scale | Support up to 50 concurrent SSE clients |
| NFR-02-006 | Board read latency | Project board read responds in under 300ms at target scale |
| NFR-02-007 | Mutation latency | Task and gate mutations commit current state and audit event in under 500ms at target scale, excluding asynchronous webhook delivery |
| NFR-02-008 | SSE delivery latency | Committed orchestration events are visible to connected SSE clients within 2 seconds |
| NFR-02-009 | Webhook enqueue latency | Webhook delivery jobs are enqueued within 1 second of the committed event |
| NFR-02-010 | Webhook failure isolation | Webhook receiver failure never rolls back committed task, gate, decomposition, or handoff state |
| NFR-02-011 | Retention | Task records, gate records, audit events, and handoff evidence are retained for the lifetime of the project |
| NFR-02-012 | Availability | Local/internal platform should remain usable for reads when webhook consumers are unavailable |
| NFR-02-013 | Audit immutability | Audit events are append-only and are not updated, deleted, or reordered through normal application flows |
| NFR-02-014 | Authorization enforcement | Unauthorized mutation attempts fail closed and append a security-relevant audit/log event where safe |
| NFR-02-015 | Operational simplicity | BRD-02 requires no dedicated broker or distributed worker system |

---

## Observability

### Metrics to emit

- `orch_project_created_total` — counter for created projects.
- `orch_task_created_total` — counter for active task creation, labeled by project, actor role, and task type where available.
- `orch_task_status_changed_total` — counter for task status transitions, labeled by from/to status.
- `orch_task_completed_total` — counter for tasks marked `done`.
- `orch_task_blocked_total` — counter for tasks explicitly moved to `blocked`.
- `orch_task_cancelled_total` — counter for tasks moved to `cancelled`.
- `orch_task_stale_total` — counter for stale detections.
- `orch_task_stale_current` — gauge for currently stale tasks by project.
- `orch_task_cycle_time_ms` — histogram for time from active task creation or activation to `done`.
- `orch_gate_opened_total` — counter for opened gates by gate type and level.
- `orch_gate_approved_total` — counter for approved gates by gate type and approver role.
- `orch_gate_rejected_total` — counter for rejected gates by gate type and approver role.
- `orch_gate_wait_duration_ms` — histogram for time from gate open to approval/rejection.
- `orch_decomposition_proposed_total` — counter for decomposition proposals.
- `orch_decomposition_approved_total` — counter for approved decomposition proposals.
- `orch_decomposition_rejected_total` — counter for rejected decomposition proposals.
- `orch_decomposition_review_duration_ms` — histogram for time from decomposition proposal creation to approval or rejection.
- `orch_decomposition_activation_duration_ms` — histogram for time from decomposition request to active child task creation.
- `orch_children_per_decomposition` — histogram for child count in decomposition proposals.
- `orch_handoff_submitted_total` — counter for Layer B structured handoffs.
- `orch_api_request_duration_ms` — histogram for orchestration API request duration, labeled by route and method.
- `orch_db_operation_duration_ms` — histogram for database operation latency.
- `orch_sse_clients_current` — gauge for connected SSE clients.
- `orch_sse_delivery_duration_ms` — histogram for event commit to SSE delivery latency.
- `orch_webhook_enqueued_total` — counter for webhook jobs enqueued.
- `orch_webhook_delivery_succeeded_total` — counter for successful webhook deliveries.
- `orch_webhook_delivery_failed_total` — counter for exhausted webhook deliveries.
- `orch_webhook_retry_total` — counter for webhook retry attempts.

### Log events

- `project.created` — when a project is created, including project ID and actor identity.
- `task.created` — when an active task is created, including project ID, task ID, parent IDs, assignee, and actor identity.
- `task.status.changed` — when task execution status changes, including from/to status and reason if provided.
- `task.stale.detected` — when a task exceeds its stale threshold without mutating status.
- `task.blocked` — when a task is explicitly blocked, including blocker and reason.
- `task.cancelled` — when a task is cancelled, including whether it was required for a parent.
- `task.decomposition.proposed` — when proposed children are created for review.
- `task.decomposition.approved` — when a proposed decomposition is activated.
- `task.decomposition.rejected` — when a proposed decomposition is rejected, including rejection reason.
- `task.decomposition.override_used` — when project decomposition defaults are overridden up to hard caps.
- `gate.opened` — when a project-level or task-level gate opens.
- `gate.approved` — when a gate is approved, including approver role.
- `gate.rejected` — when a gate is rejected, including rejection reason.
- `handoff.submitted` — when Layer B completion evidence is submitted.
- `auth.mutation.denied` — when an actor attempts a forbidden mutation.
- `webhook.delivery.failed` — when all webhook retry attempts are exhausted.
- `sse.client.connected` — when an SSE client connects to a project stream.
- `sse.client.disconnected` — when an SSE client disconnects from a project stream.

### Health/readiness endpoints

- `GET /live` — returns 200 when the service process is running and able to respond to HTTP requests.
- `GET /ready` — returns 200 only when orchestration storage is reachable, current-state reads are available, audit event persistence is available, webhook enqueueing is available, and SSE fanout is initialized.
- `GET /ready` — returns a non-200 status when current-state persistence or audit event persistence is unavailable. Webhook receiver outages MUST NOT make the platform unready, but webhook queue unavailability SHOULD make readiness fail or degrade based on implementation severity.

---

## Feature Flag

- **Flag name:** `platform-orchestration`
- **Type:** boolean
- **Default:** `false`
- **Server env:** `FF_ENABLE_PLATFORM_ORCHESTRATION`
- **Browser env:** `VITE_FF_ENABLE_PLATFORM_ORCHESTRATION`
- **Scope:** Enables platform-native project/task/gate/dependency orchestration, board APIs, SSE orchestration streams, and project-level webhook orchestration events.

Related existing flags:

| Flag | Relationship to BRD-02 |
|------|-------------------------|
| `layer-a-agents` | Enables Layer A agent actions such as decomposition proposal, routing, and configured gate actions |
| `layer-b-agents` | Enables Layer B assigned-task updates and structured handoff submission |
| `human-gates` | Enables enforcement of human approval for project-level phase gates and configured human-only task gates |
| `audit-trail` | Enables immutable orchestration audit event persistence and audit/event views |
| `kanban-orchestrator` | Legacy/compatibility naming from Phase 0; new platform-native behavior is controlled by `platform-orchestration` |

Feature flag registry impact:

- Add `platform-orchestration` to `specs/feature-flags.md` with default `false` and current `false`.
- Keep `kanban-orchestrator` until a later ADR or cleanup BRD removes or renames legacy references.
- Do not gate new platform-native runtime behavior solely on `kanban-orchestrator`.

---

## Acceptance Criteria

| ID | Criterion | Eval method |
|----|-----------|-------------|
| AC-02-001 | With `platform-orchestration=false`, platform-native project/task/gate mutation endpoints reject or hide the disabled capability according to project feature-flag conventions | Feature flag integration test |
| AC-02-002 | A project can be created, and tasks created under that project appear only in that project's board/read API | API integration test |
| AC-02-003 | Attempting to create a cross-project dependency edge is rejected and no active dependency is created | API integration test |
| AC-02-004 | Attempting to create a circular dependency is rejected and no active dependency is created | API integration test |
| AC-02-005 | A Layer A actor can submit a decomposition proposal, and proposed children do not appear as active executable tasks before approval | API integration test |
| AC-02-006 | An authorized approver can approve a decomposition proposal, causing proposed children to become active tasks with parent links and audit events | API integration test plus audit inspection |
| AC-02-007 | A rejected decomposition proposal remains available for audit with rejection reason and does not create active child tasks | API integration test plus audit inspection |
| AC-02-007A | Re-submitting an equivalent decomposition request while a proposal is active does not create duplicate active proposals or duplicate child tasks | API integration test |
| AC-02-008 | Default decomposition limits are enforced at max depth 3 and max 20 active children per parent unless project configuration changes them | Unit/integration test |
| AC-02-009 | Overrides above project defaults but within hard caps require actor identity and audit reason | API integration test plus audit inspection |
| AC-02-010 | Overrides beyond hard caps of depth 5 or 50 active children per parent are rejected | API integration test |
| AC-02-011 | A parent task cannot be marked `done` while a required child is `todo`, `in_progress`, `blocked`, or `cancelled` without approved scope reduction | API integration test |
| AC-02-012 | A required cancelled child prevents parent completion until an authorized actor replaces it, marks it non-required, or performs an audited scope adjustment | API integration test |
| AC-02-013 | A parent task cannot be marked `done` while a blocking task-level gate is open or rejected | API integration test |
| AC-02-014 | Layer B can update only its assigned task and cannot mutate unassigned tasks | Authorization test |
| AC-02-015 | Layer B cannot approve its own task gate, approve a project phase gate, or override decomposition limits | Authorization test |
| AC-02-016 | Layer B task completion is rejected unless structured handoff evidence includes summary, validation performed, risk/residual issue field, actor identity, and timestamp | API validation test |
| AC-02-017 | Project-level phase advancement requires human authorization | Authorization/integration test |
| AC-02-018 | Project configuration can enable/disable built-in task gate templates and mark gates advisory or blocking | API integration test |
| AC-02-019 | Rejected gates require a rejection reason and leave the guarded task or phase unadvanced | API integration test |
| AC-02-019A | A task-level gate can be opened and enforced on a task with no child tasks | API integration test |
| AC-02-020 | A stale `in_progress` task emits/surfaces stale state without automatically transitioning to `blocked` | Time-controlled integration test |
| AC-02-021 | Every successful task/gate/decomposition/handoff mutation appends an immutable audit event using the canonical envelope fields, including event ID, schema version, project ID, topic, actor, role, timestamp, related IDs where applicable, and payload | Audit inspection test |
| AC-02-022 | A connected SSE client receives committed orchestration events for its project within 2 seconds at target scale, with SSE `event` equal to topic, `id` equal to eventId, and `data` containing the full event envelope | SSE integration/performance test |
| AC-02-022A | An SSE client reconnecting with `Last-Event-ID` receives missed project events from the audit log without requiring operational state replay | SSE reconnect integration test |
| AC-02-023 | A project's webhook consumer receives asynchronous event delivery for subscribed events without blocking the original task/gate mutation | Webhook integration test |
| AC-02-024 | Failed webhook delivery retries with exponential backoff up to the configured retry count, default 3, then records an exhausted failure event/metric | Webhook failure integration test |
| AC-02-025 | Webhook receiver outage does not roll back or fail a successfully committed task/gate mutation | Failure injection test |
| AC-02-026 | Project board read responds under 300ms for a representative project with 10,000 tasks | Performance test |
| AC-02-027 | Task/gate mutation commits current state and audit event under 500ms at target scale, excluding async webhook delivery | Performance test |
| AC-02-028 | `GET /ready` fails or reports degraded readiness when orchestration storage or audit event persistence is unavailable | Health check integration test |
| AC-02-029 | Project export is available before authorized archival or deletion | API integration test |
| AC-02-030 | Events and handoff evidence remain queryable for the project lifetime until explicit archival/deletion flow is authorized | Retention behavior test |

---

## Dependencies / Related BRDs

| Dependency | Relationship |
|------------|--------------|
| BRD-01 App Shell | Provides the backend/frontend shell where platform-native orchestration APIs and dashboard surfaces run |
| BRD-03 Dashboard | Consumes project board APIs and project-scoped SSE events for human-facing status views |
| BRD-04 Agent Dashboard | Consumes assignment, handoff, and agent-task state; detailed agent performance analytics remain out of scope for BRD-02 |
| BRD-05 LLM Provider | Owns LLM inference behavior; BRD-02 only defines orchestration around agent tasks |
| BRD-06 Agent Memory | Owns persistent agent memory behavior; BRD-02 only stores task/gate/handoff/audit records |
| BRD-08 Quality Gates | May later add automated quality evaluation; BRD-02 defines configured gate state and approval enforcement |
| BRD-21 Notifications | May later convert orchestration events into user-facing notifications; BRD-02 only provides SSE and outbound webhook delivery |
| `contracts/events.md` | Must be updated during curation to replace Phase 0 placeholder event transport with BRD-02's canonical envelope, SSE, and webhook decisions |
| `contracts/openapi.yaml` | Must be updated during curation to replace placeholder task/gate/dependency schemas with BRD-02 project-scoped orchestration contracts |
| `specs/feature-flags.md` | Must add `platform-orchestration` and document `kanban-orchestrator` as legacy/compatibility naming |

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Gate model becomes too complex for first implementation | Medium | High | Use built-in gate templates only; defer custom gate types and workflow-builder behavior |
| Feature flag naming conflicts with existing `kanban-orchestrator` references | High | Medium | Add `platform-orchestration` as the master flag and explicitly label `kanban-orchestrator` legacy/compatibility naming |
| Current-state records and immutable audit events drift | Medium | High | Commit state mutation and audit append in one transactional boundary where possible; add diagnostic consistency checks as a could-have |
| Webhook retry system grows into a message broker | Medium | Medium | Limit BRD-02 to bounded retry, metrics, and failure visibility; defer dead-letter queues, replay, and broker-backed delivery |
| Role-scoped internal auth is under-specified and permits gate bypass | Medium | High | Define explicit allowed/forbidden actions for human, Layer A, Layer B, and system roles; require fail-closed authorization tests |
| Assisted decomposition creates too much approval overhead | Medium | Medium | Allow Layer A approval where configured; keep human approval mandatory only for project phase gates and configured human-only task gates |
| Project-scoped model adds overhead before many projects exist | Low | Medium | Treat single-project usage as a valid case while preserving project ID boundaries in data/API contracts |
| Stale detection without auto-blocking may leave work stalled | Medium | Medium | Emit stale events, board warnings, and metrics; require human or Layer A intervention rather than silent state mutation |
| Lifetime retention grows storage unexpectedly | Medium | Medium | Require project export before archival/deletion; defer enterprise retention policies until compliance requirements exist |
| SSE fanout becomes unreliable at target concurrency | Low | Medium | Track SSE connection count and delivery latency; keep target at 50 clients and defer broker-backed fanout |

---

## Open Questions

- Should `platform-orchestration` be added immediately as a new registry flag while leaving `kanban-orchestrator` untouched, or should an ADR explicitly supersede `kanban-orchestrator` during curation?
- Which task type taxonomy should map to the default gate templates (`scope_review`, `architecture_review`, `implementation_review`, `code_review`, `qa_review`, `release_review`)?
- Should webhook signing/shared secrets be mandatory in BRD-02, or acceptable as a should-have for local/internal use?
- Should orchestration event schemas be labeled `v1alpha` during Phase 1/2, or start at `v1` once BRD-02 is curated?
- Does project archival/deletion need a complete BRD-02 workflow, or is export-before-deletion enough for the first version?
- Which actors, if any, may approve task-level gates without a human present when the gate is not project-level?
- Should failed authorization attempts be appended to the immutable audit event log, or only emitted as security logs/metrics to avoid noisy audit history?
