# BRD-02 — Build Specification: Platform-Native Orchestration Pipeline

**BRD:** BRD-02-platform-native-orchestration-pipeline
**Type:** Canonical build specification for implementation and QA
**Source artifacts:** brd-02-platform-native-orchestration-pipeline.md (approved raw BRD), requirements.md, progressive-deepening-L1.md, edge-cases-L2.md, trade-offs.md, ADR-02-001 through ADR-02-006
**Status:** Complete — all findings resolved; no unresolved OQs, TBDs, or validator findings

---

## 1. Overview

BRD-02 establishes platform-native, project-scoped orchestration for tasks, dependencies, gates, agent handoffs, audit events, dashboard updates, and external event delivery. The Go/Echo backend owns operational orchestration state. Hermes Kanban is not the runtime source of truth for this product capability.

The governed workflow covers Layer A planning, Layer B execution, strict gate enforcement, and observable project progress. Every project-scoped action emits immutable audit events via SSE and outbound webhooks.

**Out of scope (confirmed):** full OIDC/JWT production authentication; dedicated message broker infrastructure (NATS, Kafka, RabbitMQ); event store as primary operational source of truth; Hermes Kanban as runtime orchestration source; automatic decomposition; automatic stale-task mutation to blocked.

---

## 2. Design Intent

### 2.1 Two-Tier State Model

The platform separates execution status from governance state.

Execution status (`todo`, `in_progress`, `blocked`, `done`, `cancelled`) describes work progress only. Governance state — decomposition proposal, gate state, stale indicator, requiredness, approval state — is modeled independently. This ensures the board API reflects real progress while governance constraints are auditable and enforceable.

### 2.2 Role Scoped Authorization

Every mutating action requires an actor identity and role claim. Minimum role set: `human`, `layer_a`, `layer_b`, `system`.

- `human`: full project operations, phase gates, task gates
- `layer_a`: decomposition proposal, task routing, task-level gate management, scope adjustments; MUST NOT approve project-level phase gates without explicit human delegation
- `layer_b`: assigned task updates and structured handoff submission only; MUST NOT approve gates, override limits, or touch unassigned tasks
- `system`: infrastructure-only — emit audit events and feature-flag-change events; all other mutations are forbidden

### 2.3 Current-State Source of Truth

The Go/Echo backend maintains durable current-state records for projects, tasks, dependencies, gates, assignments, and handoff evidence. The immutable audit log is append-only for audit and integration. BRD-02 does not require event replay as the primary state reconstruction mechanism.

State mutations and audit event appends MUST commit in one database transaction. A diagnostic consistency check runs as a background health probe on `/ready` to detect any drift (ADR-02-002).

### 2.4 Event-Driven Observability

Every project/task/gate/decomposition/handoff state change emits a canonical event envelope via two channels: a project-scoped SSE stream for first-party dashboard clients, and registered outbound webhooks for external consumers. Webhook delivery is asynchronous and MUST NOT block or roll back the originating mutation.

---

## 3. Functional Requirements

### 3.1 Project Scoping

FR-02-001: All tasks, gates, dependency graphs, event streams, webhook registrations, and board views are scoped to exactly one project.

FR-02-005: Circular dependencies and cross-project parent/child edges are rejected.

### 3.2 Task Lifecycle

FR-02-003: Execution statuses are `todo`, `in_progress`, `blocked`, `done`, `cancelled`.

FR-02-006: Required children must reach `done` before the parent can reach `done`, unless an authorized actor explicitly changes the child to non-required or performs an audited scope adjustment.

FR-02-007: A parent task MUST NOT reach `done` while any required child is unresolved or while any blocking task-level gate is open or rejected.

FR-02-015: Layer B completion requires structured handoff evidence: completion summary, artifact references, validation performed, known risks or residual issues, recommended next gate or reviewer, actor identity, timestamp.

FR-02-020: Stale detection emits events and surfaces warnings; it does NOT automatically transition `in_progress` to `blocked`.

FR-02-021: Manual blocked transition requires an explicit action and a reason.

### 3.3 Assisted Decomposition

FR-02-008: A human or Layer A actor may request decomposition. Proposed children are created in proposal state; they do not affect assignment, readiness, parent completion, or active board workload until approved.

FR-02-009: Proposed decompositions require approval from a human or authorized Layer A actor before becoming active tasks.

FR-02-009A: Only one active decomposition proposal per parent at a time. Rejected proposals may be superseded; approved proposals cannot be silently replaced.

FR-02-010: Default limits are project-configurable: max depth 3, max 20 active children per parent. Hard caps: max depth 5, max 50 active children per parent.

FR-02-011: Override above project defaults up to hard caps requires actor identity, timestamp, affected task, old limit, new limit, and audit reason.

### 3.4 Gates

FR-02-016: Project-level phase gates (G0 Foundation, G1 App Shell, G2 Core Delivery) require human approval for advancement.

FR-02-017: Task-level gate types: `scope_review`, `architecture_review`, `implementation_review`, `code_review`, `qa_review`, `release_review`.

FR-02-017A: Gates are independent of decomposition. A task-level gate may exist on a task with no child tasks.

FR-02-018: Projects can enable/disable default task-level gates, configure per-task-type gate requirements, configure approver roles, and mark gates advisory or blocking.

FR-02-019: Blocking gates prevent advancement. Rejection requires a reason and leaves the guarded item non-advanced.

### 3.5 Audit Event Log

FR-02-022: Every project/task/gate/decomposition/handoff state change appends an immutable audit event record.

FR-02-022A: All events use the canonical envelope: `eventId`, `schemaVersion`, `projectId`, `topic`, `actorId`, `actorRole`, `taskId`, `parentTaskId`, `gateId`, `timestamp`, `payload`.

Schema version is `v1alpha` during Phase 1/2 implementation, `v1` after GA (ADR-02-001).

### 3.6 SSE Event Stream

FR-02-023: Backend exposes `GET /projects/{projectId}/events/stream`. SSE `event` = topic, SSE `id` = eventId, SSE `data` = full event envelope as JSON.

FR-02-023A: Reconnecting clients may send `Last-Event-ID` to receive missed project events from the audit log. Catch-up is for dashboard continuity; current-state remains authoritative.

CORS: SSE endpoint responds to `OPTIONS` preflight with `Access-Control-Allow-Origin` validated against the project's allowed-origin list. Defaults to `["http://localhost", "http://127.0.0.1"]` for development. Wildcard `*` is not permitted in non-dev configurations (ADR-02-006).

### 3.7 Outbound Webhooks

FR-02-024: Projects support registered outbound webhook consumers by event type or prefix. Delivery is asynchronous and MUST NOT block or roll back task/gate state changes.

FR-02-025: Failed deliveries retry with exponential backoff (1s, 2s, 4s). Default retry count: 3. Exhausted deliveries are logged and visible in audit views.

Webhook signing (ADR-02-003): All webhook registrations MUST provide a `X-Webhook-Secret` at registration time. Every outbound delivery includes `X-Webhook-Signature: HMAC-SHA256(<raw_body>, <secret_hash>)`. The `localhost` and `127.0.0.1` URL exemption applies only in development; production internal URLs require signing.

### 3.8 Feature Flag Gating

FR-02-028: Master flag `platform-orchestration` (default `false`, env `FF_ENABLE_PLATFORM_ORCHESTRATION`, browser `VITE_FF_ENABLE_PLATFORM_ORCHESTRATION`) gates all platform-native orchestration capabilities.

FR-02-029: `kanban-orchestrator` is legacy/compatibility naming from Phase 0. New platform-native behavior is controlled exclusively by `platform-orchestration`.

Sub-capability flags continue to gate within the enabled surface: `layer-a-agents`, `layer-b-agents`, `human-gates`, `audit-trail`.

---

## 4. Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-02-001 | Human user scale | Up to 10 human users (initial internal platform) |
| NFR-02-002 | Agent scale | Up to 25 active Layer A/Layer B agents |
| NFR-02-003 | Project scale | Up to 50 active projects |
| NFR-02-004 | Task scale | Up to 10,000 tasks per project at target latency |
| NFR-02-005 | SSE scale | Up to 50 concurrent SSE clients per project |
| NFR-02-006 | Board read latency | Under 300ms at target scale |
| NFR-02-007 | Mutation latency | Under 500ms at target scale, excluding async webhook delivery |
| NFR-02-008 | SSE delivery latency | Committed events visible to SSE clients within 2 seconds |
| NFR-02-009 | Webhook enqueue latency | Webhook jobs enqueued within 1 second of committed event |
| NFR-02-010 | Webhook failure isolation | Webhook receiver failure never rolls back committed state |
| NFR-02-011 | Retention | Task records, gates, events, and handoff evidence retained for project lifetime |
| NFR-02-012 | Availability | Platform remains usable for reads when webhook consumers are unavailable |
| NFR-02-013 | Audit immutability | Audit events are append-only; no update, delete, or reorder through normal flows |
| NFR-02-014 | Authorization enforcement | Unauthorized mutation attempts fail closed; security audit event appended where safe |
| NFR-02-015 | Operational simplicity | No dedicated broker or distributed worker system required |

---

## 5. `/ready` Degradation Semantics

`GET /ready` returns `200` only when ALL of the following are available: orchestration storage reachable, current-state reads available, audit event persistence available, webhook enqueueing available.

When any subsystem fails, `GET /ready` returns `503` with a JSON body identifying the failing subsystem: `storage`, `currentState`, `auditPersistence`, or `webhookQueue`.

Webhook receiver outages do NOT cause a 503. Only webhook queue unavailability (not delivery) triggers a 503.

`GET /live` returns 200 when the service process is running and able to respond to HTTP requests.

---

## 6. Metrics and Log Events

Metrics emitted: `orch_project_created_total`, `orch_task_created_total`, `orch_task_status_changed_total`, `orch_task_completed_total`, `orch_task_blocked_total`, `orch_task_cancelled_total`, `orch_task_stale_total`, `orch_task_stale_current`, `orch_task_cycle_time_ms`, `orch_gate_opened_total`, `orch_gate_approved_total`, `orch_gate_rejected_total`, `orch_gate_wait_duration_ms`, `orch_decomposition_proposed_total`, `orch_decomposition_approved_total`, `orch_decomposition_rejected_total`, `orch_decomposition_review_duration_ms`, `orch_decomposition_activation_duration_ms`, `orch_children_per_decomposition`, `orch_handoff_submitted_total`, `orch_api_request_duration_ms`, `orch_db_operation_duration_ms`, `orch_sse_clients_current`, `orch_sse_delivery_duration_ms`, `orch_webhook_enqueued_total`, `orch_webhook_delivery_succeeded_total`, `orch_webhook_delivery_failed_total`, `orch_webhook_retry_total`.

Key log events: `project.created`, `task.created`, `task.status.changed`, `task.stale.detected`, `task.blocked`, `task.cancelled`, `task.decomposition.proposed`, `task.decomposition.approved`, `task.decomposition.rejected`, `task.decomposition.override_used`, `gate.opened`, `gate.approved`, `gate.rejected`, `handoff.submitted`, `auth.mutation.denied`, `webhook.delivery.failed`, `sse.client.connected`, `sse.client.disconnected`.

---

## 7. API Contract Summary

Full OpenAPI contract: `contracts/openapi.yaml` (brd-02 section).

Key endpoint families: project CRUD, task CRUD with project scope, parent/child dependency management, decomposition proposal lifecycle, gate CRUD and approval, structured handoff submission, board read API, SSE event stream, webhook registration, health/readiness.

All project-scoped endpoints require a valid project context. Cross-project mutations are rejected.

---

## 8. ADR Coverage

| ADR | Title | Status |
|-----|-------|--------|
| ADR-02-001 | Event Schema Version Naming Convention — `v1alpha` during implementation, `v1` at GA | Accepted |
| ADR-02-002 | Transaction Boundary — atomic state mutation + audit append in one DB transaction | Accepted |
| ADR-02-003 | Webhook Signing — mandatory `X-Webhook-Signature` HMAC-SHA256 except localhost dev | Accepted |
| ADR-02-004 | `/ready` Degradation — 503 with failing subsystem; webhook queue in readiness check | Accepted |
| ADR-02-005 | `system` Role Authority — infrastructure-only; forbidden from all task/gate/decomposition mutations | Accepted |
| ADR-02-006 | CORS for SSE Endpoint — per-project origin allowlist; `localhost` dev default; no wildcard | Accepted |

---

## 9. Acceptance Criteria Summary

All 30 acceptance criteria (AC-02-001 through AC-02-030) are defined in the raw BRD. Implementation must satisfy every criterion. Eval contracts in `evals/` provide testable specifications for each AC.

Key criteria: `platform-orchestration=false` hides capability; project isolation enforced; circular and cross-project dependencies rejected; decomposition proposals remain inactive until approved; required child enforcement blocks parent completion; Layer B scope strictly enforced; structured handoff required; project-level phase gates require human; SSE delivers within 2s; webhook failures never roll back state; `/ready` fails specifically when subsystems are unavailable.

---

## 10. Dependencies

| Dependency | Relationship |
|------------|--------------|
| BRD-01 App Shell | Backend/frontend shell where BRD-02 APIs and dashboard surfaces run |
| BRD-03 Dashboard | Consumes project board APIs and project-scoped SSE events |
| BRD-04 Agent Dashboard | Consumes assignment, handoff, and agent-task state |
| BRD-05 LLM Provider | Owns LLM inference behavior; BRD-02 orchestrates around agent tasks only |
| BRD-06 Agent Memory | Owns persistent agent memory; BRD-02 stores task/gate/handoff/audit records only |
| BRD-08 Quality Gates | May add automated evaluation later; BRD-02 defines configured gate state and approval enforcement |
| BRD-21 Notifications | May later convert events to user-facing notifications; BRD-02 provides SSE and outbound webhooks |
| `contracts/events.md` | Canonical event envelope, SSE spec, webhook spec, schema versioning |
| `contracts/openapi.yaml` | Project-scoped task/gate/dependency/webhook endpoint contracts |
| `specs/feature-flags.md` | `platform-orchestration` and `kanban-orchestrator` legacy relationship |

---

## 11. Deferred to Later BRDs

Task type taxonomy mapping to gate templates (OQ-2): deferred to BRD-08 or project configuration.

Archival/deletion workflow (OQ-5): export API required before deletion; full workflow deferred.

Automated task-level gate approval without human (OQ-6): Layer A by default; `scope_review` gates require human.

Custom gate types: out of scope for BRD-02.

---

*Canonical build spec — BRD-02-platform-native-orchestration-pipeline — Implementation and QA agents should treat this as the authoritative reference.*
