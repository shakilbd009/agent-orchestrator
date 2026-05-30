# BRD-02: Platform-Native Orchestration Pipeline — Systematic Refinement

**Profile:** architect  
**Date:** 2026-05-28  
**BRD:** specs/orchestration/brd-02-platform-native-orchestration-pipeline.md (approved for curation)

---

## Systematic Refinement — Applied Frameworks

This document applies the systematic-refinement skill to BRD-02. It covers:
- Requirements analysis (gap finding in FR/NFR)
- Trade-off analysis (implicit decisions that need ADRs)
- Progressive deepening L1 (component map, data flows)
- Edge cases L2 (data boundaries, state transitions, timing, integration)
- ADRs for every gap found

BRD-03 (Dashboard) is a placeholder — SSE/API consumers not yet fully defined.
BRD-08 (Quality Gates) is a placeholder — automated gate evaluation not yet designed.
contracts/events.md is Phase 0 placeholder — must be updated during BRD-02 curation.

---

## 1. Requirements Analysis

### 1.1 FR Coverage Check

| Heuristic | Finding |
|---|---|
| CORS | **GAP**: `GET /projects/{projectId}/events/stream` is browser-facing (SSE). Dashboard (BRD-03) consumes from browser. No CORS configuration specified anywhere in BRD-02. |
| Port allocation | **GAP**: No mention of port. Master feature flag `FF_ENABLE_PLATFORM_ORCHESTRATION` doesn't specify which port orchestration API runs on. `FF_ENABLE_PLATFORM_ORCHESTRATION` env vars listed but no port config. |
| Error state UX | **GAP**: FR-02-003 execution statuses defined but API 4xx/5xx error response shapes not specified. "Stale detection" (FR-02-020) emits events/warnings but doesn't define what the API returns for a stale task. |
| Missing NFRs | **PARTIAL**: SSE reconnects supported (FR-02-023A) but no backoff/retry budget for reconnect. Webhook failure isolation specified (NFR-02-010) but no explicit requirement for dead-letter visibility in API/audit. |
| Version injection | **GAP**: No mention of where version string comes from. API schema version field mentioned in FR-02-032 as should-have but not specified as must-have. |
| Healthcheck definitions | **PARTIAL**: `/live` and `/ready` semantics defined in FR-02-021 but readiness checks (DB reachability, event persistence, webhook queueing, SSE fanout) not enumerated as specific healthcheck probes. |

### 1.2 NFR Specificity Assessment

| NFR | Target | Assessment |
|---|---|---|
| NFR-02-006 Board read latency | < 300ms | Specific — directly testable |
| NFR-02-007 Mutation latency | < 500ms | Specific — directly testable |
| NFR-02-008 SSE delivery latency | < 2 seconds | Specific — but Dashboard NFR-03-001 initial render 3s means 1s for API+render after SSE delivery. Tight but workable. |
| NFR-02-009 Webhook enqueue latency | < 1 second | Specific |
| NFR-02-011 Retention | Project lifetime | Specific — directly auditable |
| NFR-02-012 Availability | "should remain usable" when webhooks unavailable | **VAGUE**: "should" is should-have language, not a hard SLA. |
| **NFR gap** | No CPU/memory container resource targets | Missing — needed for K8s/Docker Compose deployability |
| **NFR gap** | No event ordering guarantee for SSE reconnect | Ordering semantics undefined for reconnect with `Last-Event-ID` |
| **NFR gap** | No durability target for webhook queue | SQLite WAL? File-backed WAL? Not specified |

---

## 2. Trade-Off Analysis — 6 Implicit Decisions Requiring ADRs

### 2.1 ADR-02-001: Event Schema Version Naming Convention

**Problem:** FR-02-022A canonical envelope includes `schemaVersion` but format not specified. Phase 0 placeholder used `v1alpha`. Open Question 4 asks whether `v1alpha` or `v1`.

**Options:**

| Option | Pros | Cons |
|---|---|---|
| `v1alpha` during implementation, `v1` at GA | Clear pre-GA signal; consumers know what to expect | Version string changes at GA gate |
| Start at `v1` immediately | Simplified versioning; no transitional version | No explicit pre-GA signal |
| No version field | Simpler envelope; consumer handles flexible | Violates FR-02-032 should-have; no contract stability signal |

**Recommendation:** `v1alpha` during Phase 1 implementation. `v1` once BRD-02 implementation is complete and contract is frozen. When `v1` events ship, increment to `v2alpha` on breaking changes.

---

### 2.2 ADR-02-002: Transaction Boundary for State + AuditAppend

**Problem:** FR-02-002 says "where possible" for transactional boundary between current-state mutation and audit event append. This permits drift without specifying the acceptable cases.

**Options:**

| Option | Pros | Cons |
|---|---|---|
| Always in one DB transaction | Strongest consistency; no drift possible | Requires DB engine with transaction support; may limit storage choices |
| Best-effort + async consistency check | Simpler implementation; drift detected but not prevented | Can实践中允许不一致 |
| Separate commits allowed; check in `/ready` | Flexible; easy to implement | Drift may not be detected until later |

**Recommendation:** Default: one DB transaction for mutation + audit append. If DB engine lacks row-level transactions, both writes must occur in the same atomic batch via DB-level transaction. A diagnostic consistency check runs as a background health probe on `/ready`. Deviation from atomic requirement requires explicit ADR.

---

### 2.3 ADR-02-003: Webhook Signing — Mandatory vs. Should-Have

**Problem:** FR-02-031 says "should-have" for signing. Open Question 3 is unresolved. Unsigned webhooks are a security risk for external consumers.

**Options:**

| Option | Pros | Cons |
|---|---|---|
| Mandatory for all registered webhooks | Strongest security; consistent policy | Requires secret management infrastructure; friction for first internal consumers |
| Mandatory for external URLs; localhost exempt | Appropriate security boundary; internal dev friction reduced | Exemption handling in code; need allowlist for localhost |
| Should-have (current BRD text) | Frictionless for internal use | Security posture unclear; open question unresolved |

**Recommendation:** Mandatory `X-Webhook-Signature: HMAC-SHA256` header for all webhook registrations except `localhost` and `127.0.0.1` during development. Production internal URLs (e.g., `http://backend:8080`) also require signing. Secrets stored bcrypt-hashed. Registration API accepts plaintext secret once, stores hash.

---

### 2.4 ADR-02-004: `/ready` Degradation Semantics

**Problem:** FR-02-021 says readiness "SHOULD fail or degrade" when webhook queue unavailable. "Should" is ambiguous — different implementers will make different calls.

**Options:**

| Option | Behavior |
|---|---|
| Hard fail: 503 when webhook queue unavailable | `webhookQueue: false` in body; clearly signals infrastructure issue |
| Soft degrade: 200 with `status: "degraded"` | Reads continue; operations staff alerted differently than hard fail |
| Webhook queue optional for readiness | Too lenient; consumer cannot distinguish healthy from degraded |

**Recommendation:** `GET /ready` returns `200` only when: (1) orchestration storage reachable AND (2) current-state reads available AND (3) audit event persistence available AND (4) webhook enqueueing available. If any subsystem is down, return `503` with JSON body identifying the failing subsystem. Webhook Receiver outages (as opposed to queue unavailability) do NOT make the platform unready. This creates a two-tier readiness: storage-availability (minimum) vs. webhook-availability (full).

---

### 2.5 ADR-02-005: `system` Role Authority Enumeration

**Problem:** FR-02-012 defines minimum role claims including `system` but does not enumerate what `system` can do. Risk R-05 flags this as under-specified.

**Options:**

| Role | Permitted Mutations |
|---|---|
| `system` → emit audit events | Yes |
| `system` → update feature flag state | Yes |
| `system` → approve gates | No — only human/Layer A |
| `system` → create/delete tasks | No |
| `system` → move task to blocked | No — stale detection explicitly does NOT auto-block (FR-02-020) |
| `system` → complete tasks | No — only assigned Layer B |

**Recommendation:** Enumerate explicitly in BRD-02 that `system` role is infrastructure-only: permitted to emit audit events and update feature-flag-change events. All other mutations require `human`, `layer_a`, or `layer_b`. Any `system`-role mutation beyond enumerated actions must be rejected with a security audit event.

---

### 2.6 ADR-02-006: CORS Policy for SSE Endpoint

**Problem:** FR-02-023 defines `GET /projects/{projectId}/events/stream` as SSE for first-party dashboard clients. Dashboard (BRD-03) is browser-based SvelteKit. CORS not specified.

**Options:**

| Option | Pros | Cons |
|---|---|---|
| Restrict to same-origin | Simplest; no CORS headers needed if same-origin | Dashboard may be on different port in dev vs. prod |
| `Access-Control-Allow-Origin: *` | Easy; works for any consumer | Too open for webhook consumers; conflicts with signing requirement |
| `Access-Control-Allow-Origin: <dashboard-origin>` | Correct security posture | Requires dashboard origin configuration |

**Recommendation:** SSE endpoint MUST respond to `OPTIONS` preflight with `Access-Control-Allow-Origin` reflecting the `Origin` request header, validated against an allowed-origin list configured at project scope. Wildcard `*` is only acceptable for `localhost` development. Production origins must be explicitly allow-listed per project.

---

## 3. Progressive Deepening L1

### 3.1 Component Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                  Orchestration Platform                       │
│                                                               │
│  ┌─────────────┐   ┌─────────────┐   ┌──────────────────┐    │
│  │ Task Model  │   │ Gate Model  │   │  Project Model   │    │
│  │             │   │             │   │                  │    │
│  │ Id, Title   │   │ Id, Type    │   │ Id, Phase        │    │
│  │ Status      │   │ Level       │   │ Gate Config      │    │
│  │ Execution   │   │ State       │   │ Decomposition    │    │
│  │ Governance  │   │ Blocking   │   │ Defaults         │    │
│  └──────┬──────┘   └──────┬─────┘   └────────┬─────────┘    │
│         │                  │                  │               │
│         └──────────────────┼──────────────────┘               │
│                            │                                  │
│  ┌─────────────────────────▼────────────────────────────────┐│
│  │              Orchestration Service                          ││
│  │                                                           ││
│  │  TaskService        — task CRUD, status transitions       ││
│  │  GateService       — gate CRUD, approval enforcement      ││
│  │  ProjectService    — project CRUD, phase gates           ││
│  │  DecompositionService — proposal lifecycle + limits      ││
│  │  HandoffService    — structured evidence + completion    ││
│  └─────────────────────────┬────────────────────────────────┘│
│                            │                                  │
│  ┌─────────────────────────▼────────────────────────────────┐│
│  │              Event Publisher                              ││
│  │                                                           ││
│  │  AuditEventLog  — immutable append-only                  ││
│  │  SSEFanout      — per-project stream fanout (max 50)      ││
│  │  WebhookQueue   — async delivery with exponential backoff ││
│  └───────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────┘
```

### 3.2 Key Data Flows

**Layer B Task Completion — Happy Path:**

```
Layer B agent
    │ POST /tasks/{taskId}/complete  (structured handoff evidence)
    ▼
TaskService
  1. Verify caller = assignee
  2. Validate handoff: summary, validation, risks, residuals, actorId, timestamp
  3. Verify all required children STATUS = DONE
  4. Verify all blocking task-level gates STATE = approved
  5. BEGIN TRANSACTION
  6. Update task status → DONE
  7. Append audit event (HandoffSubmitted)
  8. COMMIT TRANSACTION
    │
    ├─► SSEFanout ──► Dashboard (all connected clients ≤50)
    └─► WebhookQueue (async, non-blocking)
```

**Parent Completion with Required Children — Rejection Path:**

```
POST /tasks/{parentId}/done
    │
TaskService
  1. Fetch all children where required=true
    ├─ If any child STATUS ∉ {DONE, cancelled_without_approved_scope_reduction} → REJECT 409
  2. Fetch all blocking task gates
    └─ If any gate STATE ∈ {open, rejected} → REJECT 409
  3. BEGIN TRANSACTION...
```

**Decomposition Proposal Flow:**

```
Layer A or Human requests decomposition
    │
    ▼
DecompositionService (idempotent — one active proposal at a time)
  1. Check limit: no existing active proposal for this parent
  2. Enforce limits: depth ≤max_depth, fan-out ≤max_children
  3. Create child tasks with status=proposed (inactive)
  4. Emit DecompositionProposed event
    │
    ▼ (proposal pending — no active children)
Human or Layer A approver reviews
    │
    ├─ APPROVE:
    │   children status: proposed → active
    │   emit DecompositionApproved
    └─ REJECT:
        proposal kept with reason
        emit DecompositionRejected
```

### 3.3 SSE Fanout Architecture

```
GET /projects/{projectId}/events/stream
    │
    ▼
per-project SSEManager:
  projectId → []clientChannels (max 50 per project)
  │
  │ on event committed to AuditEventLog
  │
  ├─ range clientChannels → send SSE event
  │    event = topic string
  │    id    = eventId (UUID)
  │    data  = full canonical envelope JSON
  │
  └─ client disconnects → remove from clientChannels
                         start goroutine cleanup
```

**Reconnect with Last-Event-ID:**
- Client sends `Last-Event-ID: <eventId>` header on reconnect
- Server queries AuditEventLog for all events where eventId > lastSeenId
- Server sends catch-up batch, then resumes live fanout
- Ordering preserved (events returned in ascending eventId order)

### 3.4 Webhook Delivery Architecture

```
Event committed → WebhookQueue (enqueued within 1s per NFR-02-009)
    │
WebhookDeliveryWorker (background, non-blocking)
  For each registered webhook consumer (by topic or prefix match):
    Attempt delivery with HMAC-SHA256 signature header
    On failure:
      Exponential backoff: 1s, 2s, 4^s (default 3 attempts)
      On exhaustion:
        emit webhook.delivery.failed log event
        increment orch_webhook_delivery_failed_total metric
        NO rollback of original task/gate mutation
```

### 3.5 Database Schema Scope (for reference)

The following DB entities are implied by BRD-02 (full schema in implementation):

- `projects`: id, name, phase, configuration (JSON), created_at
- `tasks`: id, project_id, title, body, status, execution_status, layer, assignee, parent_id, workspace_kind, workspace_path, priority, created_at, updated_at, completed_at
- `task_children`: task_id, child_task_id, required (bool), created_at
- `decomposition_proposals`: id, parent_task_id, status (proposed/approved/rejected), created_by, created_at, reviewed_by, reviewed_at, rejection_reason
- `gates`: id, project_id OR task_id, gate_type, level, state, blocking, created_at, approved_at, rejected_at, approver
- `handoff_evidence`: id, task_id, summary, artifacts, validation_performed, risks, residuals, actor_id, timestamp
- `audit_events`: id (eventId), schema_version, project_id, topic, actor_id, actor_role, task_id, parent_task_id, gate_id, timestamp, payload (JSONB)
- `webhook_registrations`: id, project_id, consumer_url, topic_prefix, active, secret_hash, created_at
- `sse_connections`: id, project_id, connected_at, last_event_id

---

## 4. Edge Cases L2

### 4.1 Data Boundaries

| Edge Case | Trigger | Expected Behavior |
|---|---|---|
| Empty body | Task with title only, body=null | Allowed — body is optional |
| Title at DB column limit | Title length > column limit | Specified limit needed in implementation (recommend 512 chars) |
| Fan-out = 50 (hard cap) | Decomposition child 51 requested | Reject with 422; include current count and limit in error |
| Depth = 5 (hard cap) | Decomposition at depth 5 → child of child of child... | Reject with 422 |
| Zero children proposed | Children array empty | Reject as invalid (recommend minimum 1 per proposal) |
| Null actorId | Mutation without actorId | Fail closed — reject with 401 Unauthorized |
| Duplicate actorId + role claim | Same actor claims multiple roles | Use most-privileged role for this action |
| Schema version missing | Receiving event with no schemaVersion | Accept for v1alpha events; log schemaVersion missing as warning |
| Max project name | Project name at DB limit | Recommend 256 chars; reject at 422 if exceeded |

### 4.2 State Transitions

| Edge Case | Trigger | Expected Behavior |
|---|---|---|
| Decomposition + approval race | Re-proposal arrives simultaneously with approval | Active proposal check prevents duplicate; reject new proposal |
| Parent done + child still in_progress | Parent marked done while required child is in_progress | Parent completion REJECTED — cannot confirm completion with active required child |
| Handoff submitted twice | Two rapid POST /tasks/{id}/complete | First succeeds; second returns current state (idempotent) |
| Gate approved + task cancelled simultaneously | Concurrency | Both events recorded; gate approval does not reopen cancelled task |
| Required child deleted | DELETE /tasks/{childId} when required for parent | Platform MUST reject deletion of required child; return 409 Conflict |
| Re-assigned Layer B completes old task | Task reassigned to Layer B-2; Layer B-1 calls complete | Fail closed — 403 Only the current assignee may complete |
| Active proposal superseded | New proposal before old is approved | Idempotency: reject if active proposal exists; Open Question 7 resolved by this |
| Decomposition limits overridden then reverted | Override changes mind mid-proposal | Audit trail captures each change; use latest approved override |
| Task-level gate on task with no children | Gate opened per FR-02-017A | Supported; gate independent of decomposition state |

### 4.3 Timing

| Edge Case | Trigger | Expected Behavior |
|---|---|---|
| SSE client never reads | Browser opens connection, goes idle | Implement ping/keepalive every 30s; disconnect stale channels after 2 missed pings |
| Rapid reconnect | Client reconnects before old stream EOF | Each connection gets unique channel; idempotent delivery prevents duplicates |
| Webhook retry exhaustion | All 3 attempts fail | Exhausted delivery logged; `orch_webhook_delivery_failed_total` incremented; no rollback |
| Stale detection at exact threshold | Task inactive for exactly N seconds | Emit stale event at exactly the threshold; no double-emit |
| `/ready` checks before webhook queue init | Service boots, DB up, queue still initializing | Return 503 until queue can accept; preventing premature load balancer registration |
| Layer B completes during gate review | POST complete arrives while gate still open | Reject — blocking gate must be approved before task completion |
| Decomposition proposal expires | Long-running proposal without review | No expiry defined in BRD-02; recommend async reminder metric (future) |

### 4.4 Integration

| Edge Case | Trigger | Expected Behavior |
|---|---|---|
| Webhook receiver non-2xx | Consumer returns 500 | Retry with backoff, then exhaust |
| Webhook receiver timeout | Consumer slow (>30s timeout recommended) | Treat as failure; retry |
| Webhook 301/302 redirect | Consumer redirects | Follow redirect (max 1 hop); retry at final URL |
| Webhook URL too many redirects | Consumer chain > 1 hop | Fail with permanent error; log chain |
| DB unreachable mid-transaction | DB connection drops during COMMIT | Transaction rolls back fully; client receives 503; no partial event emitted |
| 51st SSE connection attempted | Client 51 connects | Reject with 503 Too Many Connections; metric incremented |
| BRD-08 automated gate evaluation calls BRD-02 API | BRD-08 agent calls /gates/{id}/approve | If authorized via `layer_a` credential, permitted; BRD-02 treats automated same as agent |
| Feature flag changed mid-request | `platform-orchestration` toggled off during active request | Request completes under original flag state; next request sees new state |

---

## 5. Risks Cross-Check

| BRD-02 Risk | Refinement Assessment |
|---|---|
| Gate model too complex | BRD-08 is placeholder; built-in templates only. Risk is accepted by BRD-02 scope. |
| Feature flag naming conflict | `platform-orchestration` master flag defined in FR-02-028. `kanban-orchestrator` labeled legacy in FR-02-029. Properly addressed. |
| Current-state + audit drift | "Where possible" qualifier — addressed by ADR-02-002 requiring atomic writes. |
| Webhook retry → message broker | BRD-02 defers dead-letter/replay. Risk documented. Good scope control. |
| Role-scoped auth under-specified | FR-02-012 through FR-02-014 provide good coverage, but `system` role gap addressed by ADR-02-005. |
| Assisted decomposition overhead | Layer A approval configurable. Risk accepted. |
| Project-scoped overhead | Single-project is valid. Good. |
| Stale work stalls | Events + warnings emitted; auto-block explicitly excluded. Good. |
| SSE unreliability at 50 clients | 50-client target maintained (NFR-02-005). Tracking metric included. Good. |
| Retention storage growth | Not a Phase 1 concern. |

---

## 6. Contradiction Check

| Pair | Issue | Resolution |
|---|---|---|
| FR-02-020 (stale MUST NOT auto-block) + FR-02-021 (manual blocked) | No contradiction. Stale is detection-only. Manual blocked requires explicit action. Consistent. |
| FR-02-023A (SSE reconnect catch-up) + "event replay not primary source of truth" | No contradiction. Catch-up is for dashboard continuity; current-state is authoritative. |
| FR-02-002 (current-state source of truth) + FR-02-022 (immutable audit log) | No contradiction. Audit log is append-only for audit. Current-state is authoritative for reads. |
| FR-02-048 (Layer B cannot approve own gate) + BRD-08 (automated evaluation) | BRD-08 is placeholder — future concern. Automated evaluation needs its own authenticated credential. Not an active contradiction. |
| NFR-02-008 (< 2s SSE latency) + NFR-03-001 (< 3s dashboard render) | Tight but not contradictory. After SSE delivery (2s), 1s remains for API call + render. Flag for BRD-03 implementers as a constraint. |

---

## 7. Open Questions — Resolved and Open

| OQ | Status | Resolution |
|---|---|---|
| OQ-1: `platform-orchestration` flag addition | **Resolved** | FR-02-028 specifies master flag; FR-02-029 clarifies `kanban-orchestrator` as legacy. Add via feature-flags.md update. |
| OQ-2: Task type → gate template mapping | **Open** | BRD-02 does not define task type taxonomy. Deferred to BRD-08 or project configuration. Recommend BRD-02 spec allow project configuration to specify per-task-type gate requirements. |
| OQ-3: Webhook signing mandatory | **Resolved (ADR)** | ADR-02-003: mandatory signing except localhost dev; HMAC-SHA256 required. |
| OQ-4: Schema version naming | **Resolved (ADR)** | ADR-02-001: `v1alpha` during implementation, `v1` at GA. |
| OQ-5: Archival/deletion workflow | **Open** | BRD-02 specifies export-before-deletion requirement (FR-02-027, AC-02-029) but not the full workflow. Minimal spec needed: export API endpoint, authorization check, deletion confirmation, audit event. |
| OQ-6: Task-level gate approvers without human | **Open** | FR-02-017A gates can be task-level without project-level. Implicit Layer A is approver. Recommend: Layer A by default, `scope_review` gates require human. Document in gate model. |
| OQ-7: Decomposition idempotency | **Resolved** | FR-02-009A already specifies this. Implementation must atomically check-and-create proposal. |

---

## 8. Implementation Threats

| Threat | L | I | Mitigation |
|---|---|---|---|
| Go/Echo starts before SQLite init | M | H | `/ready` blocks on DB reachability; container healthcheck before routing |
| SSE goroutine leak on disconnect | M | M | Context cancellation on disconnect; connection count gauge metric |
| Webhook queue in-memory lost on restart | M | M | Queue persisted to DB/WAL before commit returns success |
| Authorization tests written too late | M | H | Fail-closed eval contracts (AC-14/AC-15) must exist before implementation |
| Decomposition idempotency non-atomic | M | H | Proposal creation + idempotency check in one DB transaction |
| CORS not configured for SSE | H | H | ADR-02-006: explicit CORS policy; SSE OPTIONS handler required |
| Event schema version drift | M | M | ADR-02-001; OpenAPI contract validation with breaking-change CI check |

---

## 9. Dependencies for Implementation

The following must be updated during/after BRD-02 curation:

| File | Required Update |
|---|---|
| `contracts/events.md` | Replace Phase 0 placeholders with BRD-02 canonical envelope (FR-02-022A) + webhook delivery spec |
| `contracts/openapi.yaml` | Add project-scoped task/gate/dependency endpoints per "must be updated" note in Dependencies section |
| `specs/feature-flags.md` | Add `platform-orchestration` with default `false`; document `kanban-orchestrator` legacy relationship |

---

*Refinement complete.*