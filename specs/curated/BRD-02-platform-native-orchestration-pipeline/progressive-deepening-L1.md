# BRD-02: Progressive Deepening L1

**Profile:** architect  
**Date:** 2026-05-28  
**BRD:** specs/orchestration/brd-02-platform-native-orchestration-pipeline.md

---

## Overview

Progressive deepening L1 for BRD-02 covers the major component responsibilities, their relationships, primary data flows (happy paths and error paths), and basic error scenarios.

BRD-03 (Dashboard) is a Phase 1 placeholder. BRD-08 (Quality Gates) is a Phase 1 placeholder. SSE/API consumer details are drawn from FR-02-023 and the implied SvelteKit browser client architecture.

---

## Component Map

### Orchestration Service Layer (Go/Echo Backend)

**1. TaskService**
- **What:** Central task lifecycle manager
- **Why:** Every task in the system needs a single accountable service for status transitions, validation, and state enforcement
- **Core responsibility:** Validate actor authorization, enforce preconditions (required children, blocking gates), commit state transitions in one atomic DB transaction, emit audit events, trigger SSE fanout and async webhook enqueue
- **Initial questions raised:**
  - Can `task.title` be updated after creation? (Proposal state vs. active task)
  - What happens if a required child is re-parented to another project?
  - How does staleness interact with blocked state?

**2. GateService**
- **What:** Gate lifecycle and approval enforcement
- **Why:** Gates are governance anchors — they must block task completion and phase advancement until approved
- **Core responsibility:** CRUD for project-level and task-level gates; enforce blocking semantics (gate open → task/phase cannot advance); emit `gate.*` audit events
- **Initial questions raised:**
  - Can a gate be reopened after approval? (Only via explicit action per FR-02-019)
  - Who unblocks a rejected gate — the same approver or escalated?
  - Do advisory gates emit events without blocking?

**3. ProjectService**
- **What:** Project-scoped configuration and state
- **Why:** All other entities are project-scoped per FR-02-001; project-level controls (defaults, gate config, phase) must be centrally managed
- **Core responsibility:** Project CRUD; phase gate advancement; decomposition depth/fan-out defaults management; project-level configuration (allowed SSE origins, gate templates, stale thresholds)
- **Initial questions raised:**
  - Can project phase be downgraded? (No — phase advancement only per FR-02-016)
  - Project deletion — what entities are cascade-deleted vs. retained for audit?
  - How does `/ready` aggregate project-level health?

**4. DecompositionService**
- **What:** Assisted task decomposition with proposal/approval workflow
- **Why:** Decomposition is a governance-critical action — it changes the task graph and active work scope; must be controlled and auditable
- **Core responsibility:** Create proposed children in proposal state; enforce idempotency (one active proposal per parent per FR-02-009A); enforce decomposition limits; emit `task.decomposition.*` events; expose proposal for human/Layer A review
- **Initial questions raised:**
  - What is the UX for reviewing a decomposition proposal? (External to BRD-02 — consumed by BRD-03 Dashboard)
  - Can approved children be modified after activation? (Scope adjustment flow, not decomposition flow)
  - Does proposal creation trigger any gate checks?

**5. HandoffService**
- **What:** Structured Layer B completion evidence collection
- **Why:** Layer B task completion requires structured evidence (FR-02-015); this is the gate for quality — evidence drives accountability and downstream reviewer readiness
- **Core responsibility:** Validate handoff evidence schema (summary, artifacts, validation performed, risks/residuals, actorId, timestamp); store evidence; emit `handoff.submitted` event
- **Initial questions raised:**
  - Can handoff evidence be amended or retracted after submission? (No — immutable audit context)
  - How does HandoffService interact with gate approval? (Separate flows; handoff is Layer B completion, gate approval is Layer A/human governance)

### Event Publisher

**6. AuditEventLog**
- **What:** Immutable append-only event store
- **Why:** FR-02-022 and FR-02-022A define the audit contract — every state change is recorded, immutable, and queryable for the project lifetime
- **Core responsibility:** Append events with canonical envelope (eventId UUID, schemaVersion, projectId, topic, actorId, actorRole, taskId, parentTaskId, gateId, timestamp, payload JSONB); expose query API for catch-up fanout
- **Initial questions raised:**
  - Event query API — by projectId? By topic prefix? Both? (For SSE catch-up per FR-02-023A)
  - Event retention — are old events ever paginated, or is the full log always returned? (Recommend cursor-based pagination for catch-up)
  - Ordering — is eventId (UUID) globally ordered or only per-project? (Must be project-scoped ordering; UUID v7 is time-ordered)

**7. SSEFanout**
- **What:** Per-project Server-Sent Events fanout manager
- **Why:** FR-02-023 requires committed events visible to connected SSE clients within 2 seconds; FR-02-023A requires reconnect-catch-up
- **Core responsibility:** Maintain per-project connection channels (≤50 clients per NFR-02-005); fanout committed events to all connected channels; handle `Last-Event-ID` catch-up query; manage ping/keepalive for stale detection
- **Initial questions raised:**
  - What happens when a client's channel is full (≥50)? (Connection rejected with 503 per FR-02-023)
  - Does SSE fanout require a separate goroutine per project or one global goroutine selecting? (Per-project goroutine OR channel-based multiplexing; both acceptable)
  - Ping/keepalive interval? (Recommended: 30s with disconnect after 2 missed pings)

**8. WebhookQueue**
- **What:** Async webhook delivery job queue
- **Why:** FR-02-024 requires asynchronous, non-blocking webhook delivery; FR-02-025 requires exponential backoff retry; NFR-02-010 requires failure isolation
- **Core responsibility:** Enqueue webhook delivery jobs on event commit; process jobs with HMAC-SHA256 signature; exponential backoff retry (1s, 2s, 4s); exhaust tracking; never roll back on failure
- **Initial questions raised:**
  - Queue persistence — in-memory (lost on restart), file-backed WAL, or DB-backed? (Recommendation: DB-backed job table so queue survives restart)
  - Concurrent webhook workers — how many? (Recommend 1 worker per project or shared pool with project-based ID tagging)
  - Dead-letter — is exhausted job state visible in API? (Audit event + metric; no dead-letter queue in BRD-02 scope per "out of scope")

### Shared Infrastructure

**9. AuthorizationMiddleware**
- **What:** Role-filtering middleware on every mutating endpoint
- **Why:** FR-02-012 through FR-02-014 define the authorization matrix; every action checks actorRole before processing
- **Core responsibility:** Extract actorId + role claim from auth context; enforce role-action matrix; reject unauthorized mutations with 403; emit `auth.mutation.denied` security event on rejection

**10. FeatureFlagMiddleware**
- **What:** Feature flag evaluation middleware
- **Why:** FR-02-028 introduces `platform-orchestration` master flag; related flags (`layer-a-agents`, `layer-b-agents`, `human-gates`, `audit-trail`) gate specific actions per AGENTS.md
- **Core responsibility:** Evaluate feature flags per request context; reject requests to disabled capabilities with 403 or appropriate error

---

## Primary Data Flows

### Happy Path: Layer B Task Completion

```
Layer B agent                          Orchestration Backend
───────────────                        ──────────────────────
POST /tasks/{taskId}/complete
(w/ structured handoff evidence)
  ──────────────────────────────────────► AuthorizationMiddleware
                                            Extract actorId, role
                                            Validate role == layer_b
                                      ──► TaskService.CompleteTask()
                                            1. Validate caller = assignee
                                            2. Validate required children DONE
                                            3. Validate blocking gates APPROVED
                                            4. BEGIN TRANSACTION
                                            5. Update task.status → DONE
                                            6. Emit AuditEventLog entry
                                               (HandoffSubmitted)
                                            7. COMMIT TRANSACTION
                                            8. Trigger SSEFanout (sync)
                                            9. Trigger WebhookQueue (async)
                                      ◄── Return 200 Task
  ◄────────────────────────────────────── 200 w/ updated task
```

### Error Path: Parent Done with Unresolved Required Child

```
Layer A or Human                        Orchestration Backend
───────────────                        ──────────────────────
POST /tasks/{parentTaskId}/done
  ──────────────────────────────────────► AuthorizationMiddleware
                                            Check role ∈ {human, layer_a}
                                      ──► TaskService.CompleteTask(parentTaskId)
                                            1. Fetch required children
                                            2. For each required child:
                                                 check status == DONE ||
                                                 cancelled_with_scope_reduction
                                               If any NOT done:
                                                 REJECT 409 Conflict
                                                 Body: { blockingChildren: [...] }
                                            3. Fetch blocking task gates
                                               If any NOT approved:
                                                 REJECT 409 Conflict
                                                 Body: { blockingGates: [...] }
                                            4. No state change
                                      ◄── 409 w/ blocking details
  ◄────────────────────────────────────── 409 Conflict
```

---

## Architecture Diagram (Mermaid)

```mermaid
flowchart TD
    subgraph "Orchestration Platform Layer"
        E[(AuditEventLog\nappend-only)]

        subgraph "Orchestration Service"
            TS[TasKService]
            GS[GateService]
            PS[ProjectService]
            DS[DecompositionService]
            HS[HandoffService]
        end

        subgraph "Event Publisher"
            SSE[SSEFanout\nper-project ≤50 clients]
            WH[WebhookQueue\nasync retry]
        end
    end

    subgraph "External Consumers"
        DB[(Dashboard\nBRD-03\nSvelteKit)]
        WEB[(Webhook\nConsumers)]
        LH[(Layer B\nAgents)]
        HA[(Human\nApprovers)]
        LA[(Layer A\nAgents)]
    end

    HA -->|POST /tasks/{id}/complete| TS
    LA -->|POST /tasks/{id}/decompose| DS
    LA -->|POST /gates/{id}/approve| GS
    HA -->|POST /projects/{id}/phase-advance| PS
    LH -->|GET /projects/{id}/events/stream| SSE
    DB -->|GET /projects/{id}/events/stream| SSE
    WEB -->|Outbound Webhook Delivery| WH
    WH -->|async| WEB

    TS -->|Commit + Audit| E
    GS -->|Audit| E
    PS -->|Audit| E
    DS -->|Audit| E
    HS -->|Audit| E
    E -->|Fanout| SSE
    E -->|Enqueue Job| WH

    style SSE fill:#1a1a2e,color:#eee,stroke:#646cff
    style WH fill:#1a1a2e,color:#eee,stroke:#646cff
    style E fill:#1a1a2e,color:#eee,stroke:#4caf50
```

---

## Error Scenarios

| Scenario | What Happens | Detection |
|---|---|---|
| DB unreachable during mutation | Transaction rolls back; client receives 503; no event emitted | `/ready` DB probe |
| SSE client disconnects | Goroutine channel cleaned up; SSE connection gauge decrements | SSE goroutine logs disconnect |
| SSE client count ≥50 | Connection rejected with 503 Too Many Connections; metric incremented | `orch_sse_clients_current` gauge at max |
| Webhook queue full/unavailable | `/ready` returns 503; events still committed; webhook delivery deferred | `GET /ready` webhook probe |
| Auth middleware rejects mutation | 403; `auth.mutation.denied` security event appended | Security event log |
| Decomposition limit exceeded | 422; body includes current count, limit, and requested count | API validation |
| Decomposition proposal active (idempotency) | 409; body says proposal already exists | API response body |
| Missing required child (parent done) | 409; body lists blocking required children | API response body |
| Open blocking gate (parent done) | 409; body lists blocking gates | API response body |
| Handoff evidence schema invalid | 422; body returns validation errors | API response body |
| Orphaned SSE goroutine (client silent) | Ping/keepalive every 30s; disconnect after 2 missed pings | Goroutine timeout |
| Event envelope schema drift | Consumer receives event with unexpected schemaVersion | Consumer schema validation logs |
