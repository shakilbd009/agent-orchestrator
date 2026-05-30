# BRD-02 — Implementation Readiness

**BRD:** BRD-02-platform-native-orchestration-pipeline
**Stage:** Graduation evidence (implementation readiness gate)
**Source tasks:** t_f5b70b60 (systematic-refinement) · t_06e22673 (eval-readiness) · t_b481ed59 (security repair) · t_1bad7f24 (performance repair) · t_9fd392cb (webhook repair) · t_7cc74376 (PM approval) · t_6f5b3407 (eval-readiness re-run)
**Status:** Ready for implementation

---

## 1. Eval Contracts

### 1.1 Security Eval Contract

The security eval (`evals/security/brd-02-platform-orchestration-security.md`) defines 8 test contracts covering role-scoped authorization, Layer A/Layer B boundary enforcement, auth.mutation.denied event, unauthorized mutation fail-closed, webhook non-blocking, webhook failure isolation, audit immutability, and cross-project rejection.

Key testable behaviors:
- `401 Unauthorized` for missing/null/empty actorId or unrecognized role
- `403 Forbidden` for role-action mismatches (Layer B gate approval, Layer A phase gate without delegation, cross-project mutation)
- `auth.mutation.denied` audit event appended on every failed authorization
- Webhook failure never rolls back committed task/gate state
- Audit events are append-only (no UPDATE/DELETE in normal flows)

### 1.2 Performance Eval Contract

The performance eval (`evals/perf/brd-02-platform-orchestration-performance.md`) defines 8 test contracts covering board read latency, mutation latency, SSE delivery latency, webhook enqueue latency, SSE concurrent client scale, operational simplicity (no broker), webhook failure isolation, and read availability during webhook outage.

Key performance thresholds:
- Board read: p50 < 300ms, p95 < 500ms at 10,000 tasks
- Mutation latency: p50 < 500ms, p95 < 750ms (excluding async webhook)
- SSE delivery: p50 < 2s, p95 < 3s
- Webhook enqueue: p50 < 1s, p95 < 2s
- SSE concurrent: 50 clients max; 51st rejected with 503/429

### 1.3 Unit Eval Contracts

Nine unit eval files cover: authorization service, decomposition service, event envelope encoding, gate service, health/readiness endpoint, SSE fanout, task model, and webhook delivery.

### 1.4 Integration Eval Contracts

Eight integration eval files cover: audit event persistence, decomposition lifecycle, gate enforcement, health endpoints, project CRUD, SSE stream, task lifecycle, and webhook delivery/retry.

### 1.5 E2E Eval Contracts

Six E2E eval files cover: authorization workflows, decomposition end-to-end, gate workflows, project scoping, SSE streaming, task lifecycle, and webhook delivery.

---

## 2. Feature Flags

| Flag | Default | Current | Scope |
|------|---------|---------|-------|
| `platform-orchestration` | `false` | `false` | Master gate: all project/task/gate/orchestration capabilities |
| `layer-a-agents` | `false` | `false` | Layer A decomposition, routing, task-level gate management |
| `layer-b-agents` | `false` | `false` | Layer B assigned-task updates and handoff submission |
| `human-gates` | `false` | `false` | Human approval for project-level phase gates and configured task gates |
| `audit-trail` | `false` | `false` | Immutable audit event persistence and audit/event query APIs |
| `kanban-orchestrator` | `false` | `false` | Legacy/compatibility; new platform-native behavior uses `platform-orchestration` |

When `platform-orchestration=false`, all project-scoped orchestration endpoints reject or hide the disabled capability. Sub-capability flags gate within the enabled surface.

---

## 3. OpenAPI / Events Contracts

### 3.1 OpenAPI Contract

`contracts/openapi.yaml` defines the full BRD-02 API surface: project CRUD, task CRUD with project scope, dependency graph management, decomposition proposal lifecycle, gate CRUD and approval, structured handoff submission, board read API, SSE stream, webhook registration, and health endpoints.

Schema versioning: `0.2.0-brd02` at curation time; advances to `1.0.0` after BRD-02 implementation GA.

### 3.2 Events Contract

`contracts/events.md` defines the canonical 11-field envelope (`eventId`, `schemaVersion`, `projectId`, `topic`, `actorId`, `actorRole`, `taskId`, `parentTaskId`, `gateId`, `timestamp`, `payload`), the envelope+payload pattern, schema versioning policy (`v1alpha` during Phase 1/2, `v1` at GA), SSE stream specification, webhook delivery specification, and the complete event topic registry.

Key transport decisions:
- SSE: `GET /projects/{projectId}/events/stream`; SSE `event` = topic, `id` = eventId, `data` = full envelope JSON
- SSE reconnect: `Last-Event-ID` header; catch-up from audit log
- Webhook: async, non-blocking, exponential backoff (1s, 2s, 4s), 3 retry attempts default
- Webhook signing: `X-Webhook-Signature: HMAC-SHA256` mandatory except localhost dev

---

## 4. Production-Checklist Prerequisites

Before BRD-02 implementation begins, the following must be available:

| Prerequisite | Required Version | Verification |
|-------------|-----------------|---------------|
| Go toolchain | 1.25.6 | `go version` |
| Docker daemon | Running | `docker info` exits 0 |
| Docker Compose (standalone) | v5.0.2 | `docker-compose --version` |
| Node.js | v25.3.0 | `node --version` |
| npm | 11.7.0 | `npm --version` |
| pnpm | 10.32.1 | `pnpm --version` |
| Go/Echo backend (BRD-01) | Running on :3001 | `curl http://localhost:3001/health` |
| SvelteKit frontend (BRD-01) | Running on :5173 | `curl http://localhost:5173` |
| SQLite (or chosen DB engine) | Latest stable | DB can be initialized |

Implementation must not begin until: (a) BRD-01 app shell is complete and verified, (b) `contracts/openapi.yaml` BRD-02 section schema is finalized, (c) `contracts/events.md` canonical envelope and event registry are finalized, (d) `specs/feature-flags.md` has `platform-orchestration` registered.

---

## 5. Security and Performance Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| CORS not configured for SSE | High | Dashboard SSE connections blocked in browser | ADR-02-006: explicit CORS policy; OPTIONS handler required |
| Webhook signing not implemented | High | Webhook delivery insecure; eval contract fails | ADR-02-003: mandatory signing except localhost dev |
| Decomposition idempotency not atomic | Medium | Duplicate proposals created under race | ADR-02-002: atomic check-and-create in one DB transaction |
| SSE goroutine leak on disconnect | Medium | Memory grows over time | Context cancellation on disconnect; connection count gauge metric |
| Role auth gaps enabling gate bypass | Medium | Governance model violated | ADR-02-005: explicit `system` role enumeration; fail-closed |
| Event schema version drift | Medium | Consumer breakage | ADR-02-001: `v1alpha` during impl; breaking-change CI check |
| Transaction boundary drift (current-state vs audit) | Medium | Inconsistent audit history | ADR-02-002: atomic commit; diagnostic check on `/ready` |
| Webhook queue in-memory lost on restart | Medium | Webhook deliveries lost | Queue persisted to DB/WAL before commit returns success |

---

## 6. Rollback Notes

| Scenario | Rollback Action |
|----------|----------------|
| Webhook signing broken | Revert webhook delivery code; re-register consumers without secret requirement |
| SSE CORS misconfigured | Revert CORS middleware; restrict SSE to same-origin temporarily |
| Transaction atomicity broken | Immediate DB transaction review; diagnostic query on `/ready` surfaces drift |
| `platform-orchestration` flag prevents all orchestration | Set flag to `false`; all orchestration endpoints return 403 or hide capability |
| DB schema migration failure | Migration scripts must be reversible; maintain down-migration path |
| Event envelope schema change breaks consumers | Version bump; `v1alpha` signals pre-stability; migration guide required |

Rollback at the orchestration layer: disabling `platform-orchestration=false` should make the platform revert to Phase 0 behavior (no project/task/gate orchestration surface visible). No persistent state is deleted.

---

## 7. Next Gate Constraints

Implementation is gated by ALL of the following:

1. **production-checklist (ops):** Ops must verify pre-implementation prerequisites are met before backend/frontend work begins. This is a mandatory gate per AGENTS.md.

2. **completeness-score gate:** Must run against this graduation package. Must confirm 25/25 eval files exist, FR/NFR coverage complete, no unresolved TBD/OQ.

3. **validate-design gate (t_1d41bc0d):** Validator must approve the four graduation files before PM gate.

4. **PM gate review:** PM must explicitly approve before implementation begins.

5. **Feature flag registration:** `platform-orchestration` must be added to `specs/feature-flags.md` before any BRD-02 source code is written.

6. **OpenAPI/events contract freeze:** `contracts/openapi.yaml` and `contracts/events.md` must be in final canonical form before implementation begins.

7. **No Phase 2 dependencies in Phase 1 scope:** BRD-03 (Dashboard), BRD-05 (LLM Provider), BRD-06 (Agent Memory) are placeholders. BRD-02 implementation must not assume their existence.

---

## 8. Implementation Handoff Constraints

The following constraints must be respected during implementation:

| Constraint | Source | Requirement |
|------------|--------|-------------|
| `platform-orchestration` master flag controls all BRD-02 surface | FR-02-028 | All orchestration endpoints gated by flag |
| `kanban-orchestrator` is legacy; do not use for new features | FR-02-029 | New code uses `platform-orchestration` |
| Webhook signing mandatory except localhost dev | ADR-02-003 | `X-Webhook-Signature: HMAC-SHA256` header on all non-dev webhooks |
| SSE requires CORS preflight handling | ADR-02-006 | OPTIONS handler required; per-project origin allowlist |
| `system` role infrastructure-only | ADR-02-005 | `system` may only emit audit and feature-flag-change events |
| Atomic state + audit commit | ADR-02-002 | One DB transaction for mutation + audit append |
| `/ready` returns 503 with failing subsystem | ADR-02-004 | Webhook queue in readiness check; webhook receiver outages do NOT cause 503 |
| Event schema version `v1alpha` | ADR-02-001 | All events carry `schemaVersion: "v1alpha"` during Phase 1/2 |
| No dedicated message broker | NFR-02-015 | In-process or DB-backed queue; no NATS/Kafka/RabbitMQ |
| Board read under 300ms at 10k tasks | NFR-02-006 | Database indexes on project_id, execution_status, parent_task_id |
| Mutation under 500ms excluding async webhook | NFR-02-007 | Synchronous commit path must be fast |
| SSE delivery within 2 seconds | NFR-02-008 | Fanout must not be blocked by slow webhook delivery |
| No Phase 2 features in Phase 1 scope | brd.md | 7 deferred items listed; do not anticipate them |

---

## 9. Verification Checklist

Before marking BRD-02 implementation complete, verify all of the following:

- [ ] `platform-orchestration=false` hides all orchestration endpoints (AC-02-001)
- [ ] Project isolation: tasks from project A not visible in project B board read
- [ ] Cross-project dependency edge rejected with 409
- [ ] Circular dependency rejected with 409
- [ ] Decomposition proposed children do not appear as active tasks before approval
- [ ] Only one active decomposition proposal per parent (idempotency)
- [ ] Required child blocks parent `done` transition
- [ ] Layer B can only update/mutate assigned tasks
- [ ] Layer B cannot approve own gate, phase gate, or override limits
- [ ] Structured handoff required before Layer B task completion
- [ ] Project-level phase gate requires human approval
- [ ] Task-level gate on task with no children enforced correctly
- [ ] `auth.mutation.denied` event appended on every failed authorization
- [ ] SSE client receives events within 2 seconds of commit
- [ ] SSE reconnect with `Last-Event-ID` receives missed events
- [ ] Webhook delivery does not block task mutation response
- [ ] Webhook failure never rolls back committed state
- [ ] `GET /ready` returns 503 with correct failing subsystem when webhook queue unavailable
- [ ] `GET /ready` returns 200 when webhook queue is up even if webhook receiver is down
- [ ] No external message broker in `go.mod` or `docker-compose.yml`
- [ ] All 30 ACs pass their respective eval contracts

---

*Implementation readiness gate: t_f5b70b60 + t_06e22673 + repairs → t_6f5b3407 → t_376d5ec2*
*This document is part of the BRD-02-orchestration-pipeline graduation evidence package.*
