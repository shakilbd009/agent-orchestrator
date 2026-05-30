# evals/security/brd-02-platform-orchestration-security.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** Security test contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## Contract: Role-scoped authorization enforcement — FR-02-012

### Input
- Actor identity + role claim present in mutation request
- Endpoint requires minimum role: `human`, `layer_a`, `layer_b`, or `system`
- Actor role matches (or outranks) required role for this action

### Output
- Mutation proceeds
- Actor identity and role recorded in audit envelope

### Evidence
- FR-02-012: "Every mutating action MUST include an actor identity and role claim. Minimum role claims are `human`, `layer_a`, `layer_b`, and `system`."
- FR-02-013, FR-02-014 define per-role action allowances

### Edge cases
- No actorId + no role claim: `401 Unauthorized` (fail closed)
- actorId null or empty string: `401 Unauthorized`
- Role claim not in {human, layer_a, layer_b, system}: `401 Unauthorized`
- Role claim valid but action not in role's allowed set: `403 Forbidden`

---

## Contract: Layer A boundary enforcement — FR-02-013 and FR-02-014

### Input
- Actor with `layer_a` role
- Attempted actions over N=4 scope boundaries: phase gate approval (non-delegated), self gate approval, cross-project mutation, approved scope reduction

### Output
- Phase gate approval without explicit human delegation: `403 Forbidden`
- Self gate approval: `403 Forbidden`
- Decomposition proposal, routing, task-level gate management, audited scope adjustments: permitted

### Evidence
- FR-02-013: Layer A MAY propose decomposition, approve decomposition (where configured), route child tasks, manage task-level gates (where authorized), perform audited scope adjustments. MUST NOT approve project-level phase gates unless human has delegated that authority explicitly.
- FR-02-014: Layer B MUST NOT approve their own gates, approve project-level gates, override decomposition limits, complete parent tasks by side effect, or mutate tasks outside their assignment scope.

### Edge cases
- Layer A attempting to approve project-level phase gate without delegation: `403 Forbidden`
- Layer A approving task-level gate where explicitly authorized: permitted
- Layer A attempting to complete parent via side effect: `403 Forbidden`
- Layer A overriding decomposition limits within hard caps: permitted (audit required)

---

## Contract: Layer B boundary enforcement — FR-02-014

### Input
- Actor with `layer_b` role
- Attempted mutation: update status of unassigned task, complete task not assigned to actor, approve any gate, override decomposition limits, create cross-project dependency, emit audit event for another actor

### Output
- Update unassigned task: `403 Forbidden`
- Complete unassigned task: `403 Forbidden`
- Approve own task gate: `403 Forbidden`
- Approve project-level gate: `403 Forbidden`
- Override decomposition limits: `403 Forbidden`
- Cross-project mutation: `403 Forbidden` (FR-02-005 rejects cross-project edges)

### Evidence
- FR-02-014: Layer B MUST NOT approve their own gates, approve project-level gates, override decomposition limits, complete parent tasks by side effect, or mutate tasks outside their assignment scope.
- FR-02-015: Layer B handoff requires structured evidence before `done` transition
- AC-02-014: "Layer B can update only its assigned task and cannot mutate unassigned tasks"
- AC-02-015: "Layer B cannot approve its own task gate, approve a project phase gate, or override decomposition limits"

### Edge cases
- Layer B assigned to task X attempts to update task Y status: `403 Forbidden`
- Layer B assigned to task X attempts to complete task X without structured handoff: validation fail, `422` or `400`
- Layer B assigned to task X provides complete handoff and attempts to complete task X: permitted
- Layer B self-approving a gate on assigned task: `403 Forbidden`

---

## Contract: auth.mutation.denied event on forbidden mutation — NFR-02-014

### Input
- Actor attempts a mutation that fails authorization (both 401 and 403 cases)
- Audit system is available (FF_ENABLE_PLATFORM_ORCHESTRATION=true, FF_ENABLE_AUDIT_TRAIL=true)

### Output
- Mutation rejected before any state change
- Response: `401 Unauthorized` (no identity) or `403 Forbidden` (insufficient role/scope)
- Audit event `auth.mutation.denied` appended to immutable audit log:
  ```json
  {
    "eventId": "string",
    "schemaVersion": "string",
    "projectId": "string | null",
    "topic": "auth.mutation.denied",
    "actorId": "string",
    "actorRole": "string",
    "taskId": "string | null",
    "parentTaskId": "string | null",
    "gateId": "string | null",
    "timestamp": "ISO8601",
    "payload": {
      "attemptedAction": "string",
      "deniedReason": "string"
    }
  }
  ```
- Event is append-only (NFR-02-013): no update, no delete, no reorder

### Evidence
- NFR-02-014: "Unauthorized mutation attempts fail closed and append a security-relevant audit/log event where safe"
- Log event table: `auth.mutation.denied` listed as a committed log event
- contracts/events.md: `auth.mutation.denied` envelope defined

### Edge cases
- Audit system unavailable at mutation time: mutation still fails closed; audit event may be dropped or logged locally as a fallback
- Actor presents forged credentials: treat as no-identity (fail closed at 401)
- Attempted action is a created resource (webhook registration, project creation): `403 Forbidden` still applies regardless of what was created

---

## Contract: Unauthorized mutation fails closed — NFR-02-014

### Input
- Any request to a mutating endpoint that omits, nullifies, or misspells the `actorId` or `actorRole` field

### Output
- Response `401 Unauthorized`
- No state change in current-state records
- No state change in audit log (or audit event appended if system available per NFR-02-014 "where safe")
- Rate limiting SHOULD NOT be the primary fail-closed mechanism for auth (per defense in depth)

### Evidence
- FR-02-012: must-include actor identity and role claim on every mutating action
- Edge case in e2e auth eval: "Null actorId fails closed" scenario

### Edge cases
- `actorId` present but `actorRole` omitted: also `401 Unauthorized` (both required)
- Both present but role not in allowlist: `403 Forbidden`
- Expired credentials treated as no credentials (fail closed at 401)

---

## Contract: Webhook non-blocking — NFR-02-010

### Input
- Mutating endpoint commits task, gate, decomposition, or handoff state
- FF_ENABLE_PLATFORM_ORCHESTRATION=true
- Registered webhook consumer for the event topic
- Webhook receiver is unavailable (timeouts, connection refused, or 5xx)

### Output
- Task/gate mutation response returns `200 OK` (or appropriate success)
- Current-state record committed
- Audit event appended
- Webhook delivery job enqueued asynchronously
- **Webhook failure does not roll back committed state**

### Evidence
- NFR-02-010: "Webhook receiver failure never rolls back committed task, gate, decomposition, or handoff state"
- NFR-02-012: "Local/internal platform should remain usable for reads when webhook consumers are unavailable"
- FR-02-024: "Webhook delivery MUST be asynchronous and MUST NOT block or roll back task/gate state changes"
- AC-02-025: "Webhook receiver outage does not roll back or fail a successfully committed task/gate mutation"

### Edge cases
- Webhook queue unavailable: read-path remains available; webhook events may be lost until queue recovers (logged)
- Webhook consumer registered for event that never fires: no-op (not a failure)
- Webhook delivery succeeds after N retries: success counted in metric `orch_webhook_delivery_succeeded_total`

---

## Contract: Webhook failure isolation — NFR-02-010, NFR-02-012, NFR-02-013

### Input
- A webhook receiver is slow, returns errors, or exceeds the retry budget
- Events from multiple projects/deliveries are in flight simultaneously

### Output
- Slow/failed webhook does not stall SSE delivery to other clients
- Slow/failed webhook does not cause `GET /ready` to fail (webhook receiver outage is not blocking readiness)
- Exhausted webhook delivery: `webhook.delivery.failed` event appended with attemptCount, lastError, eventId; metric `orch_webhook_delivery_failed_total` incremented
- No cross-webhook bleed: one consumer's failure does not affect another's registered consumers

### Evidence
- NFR-02-010: failure isolation
- NFR-02-012: reads remain available during webhook outage
- contracts/events.md: `webhook.delivery.failed` envelope with `attemptCount`, `lastError`, `eventId`

### Edge cases
- Webhook consumer unregisters during active delivery: current delivery completes or is abandoned (not a state error)
- Retry budget exhausted on one event: other events for same consumer still enqueued (separate retry state per event)
- Retry budget exhausted globally across all consumers: system continues accepting mutations

---

## Contract: Audit append-only immutability — NFR-02-013

### Input
- Any state-change mutation commits to current-state store and audit log simultaneously
- Actor identity, role, timestamp, related IDs, and event payload are included in the audit envelope

### Output
- Audit event is appended to immutable log (append-only; no UPDATE or DELETE issued by application flows)
- Event ordering is preserved (no reorder of committed events)
- Event ID is unique and monotonically increasing (or UUIDv7 for time-order)
- Schema version field present in envelope for downstream consumers

### Evidence
- NFR-02-013: "Audit events are append-only and are not updated, deleted, or reordered through normal application flows"
- FR-02-022: Immutable audit event log; every state change appends a record
- FR-02-022A: Canonical event envelope with all required fields
- AC-02-021: "Every successful task/gate/decomposition/handoff mutation appends an immutable audit event using the canonical envelope fields"

### Edge cases
- Application attempt to mutate or delete an audit record: rejected at DB layer (or application error)
- Audit table missing required columns (schema version, eventId): schema validation fails; alert emitted
- Replay of events for debugging: read-only replay on a separate read replica is acceptable; replay is not the operational source of truth (FR-02-002)

---

## Contract: Authorization test pattern for cross-project rejection

### Input
- Actor authenticated in project A context
- Attempted mutation targets a resource in project B (task, gate, webhook registration)

### Output
- `403 Forbidden`
- No state change in project B
- `auth.mutation.denied` audit event appended to project A (or project B audit log if applicable per deployment)

### Evidence
- FR-02-001: "A task MUST belong to exactly one project"
- FR-02-005: "The platform MUST reject circular dependencies and cross-project parent/child edges"
- Cross-project mutation is a superset of cross-project edge; rejection is consistent enforcement

### Edge cases
- Actor has `human` role in project A but no membership in project B: `403 Forbidden`
- Service account with `system` role attempts cross-project mutation: `403 Forbidden` per FR-02-014 system-role boundaries
- Actor presents valid project A credentials but spoofs project B resource ID in request body: authorization check fails (fail closed)

---

*End of BRD-02 security contracts*
