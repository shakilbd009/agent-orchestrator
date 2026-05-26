# BRD-01-app-shell — Decision Record

**BRD:** BRD-01-app-shell
**Stage:** 05-decision-record
**Status:** Complete

---

## PM Gate Decisions

### t_506eba0b — validate-design (PM Gate)

| Field | Value |
|-------|-------|
| **Task** | t_506eba0b — PM validate-design gate |
| **Verdict** | `REPAIR_REQUIRED_BEFORE_GRADUATION` |
| **Date** | 2026-05-25 |
| **Approver** | pm (profile) |
| **Source-task** | t_506eba0b |
| **Finding count** | 10 PERF findings (PERF-001 through PERF-010) |

PM did not approve direct graduation. Created repair task t_0e13d59d as additional parent.

### t_0e13d59d — Performance Repair (Resolved)

| Field | Value |
|-------|-------|
| **Task** | t_0e13d59d — Performance repair |
| **Parent** | t_506eba0b |
| **Date resolved** | 2026-05-25 |
| **Approver** | pm (profile) — via repair completion |
| **Source-task** | t_0e13d59d |
| **Findings resolved** | PERF-001, PERF-002, PERF-003, PERF-004, PERF-005, PERF-006, PERF-007, PERF-008, PERF-009, PERF-010 |
| **Artifacts modified** | requirements.md, edge-cases-L2.md, trade-offs.md, progressive-deepening-L2.md |

After repair completed, no open PM decision remains.

---

## OQ Resolutions

### OQ-2 — Frontend Base URL

| Field | Value |
|-------|-------|
| **OQ** | OQ-2: What base URL should the frontend use to call the backend API? |
| **Resolution** | `http://localhost:3001` |
| **Documented in** | ADR-01-001 (backend port allocation) |
| **Source** | requirements.md, OQ-2 row |
| **Date** | 2026-05-25 |

OQ-2 also resolves port 3001 as the fixed backend port. Frontend `VITE_API_BASE_URL` is set to `http://localhost:3001`.

---

## Trade-Off Decisions

### Trade-Off 1: Backend Port Allocation

| Field | Value |
|-------|-------|
| **Decision** | Backend port allocation |
| **Chosen option** | Option A — Fixed port 3001 |
| **Source** | trade-offs.md (Major Decision 1) |
| **Approver** | architect (subagent) |
| **Date** | 2026-05-25 |
| **ADR ref** | ADR-01-001 |
| **Verdict rationale** | Simplicity; matches resolved OQ-2; zero config overhead for Phase 1 shell |

### Trade-Off 2: Echo Version Stability

| Field | Value |
|-------|-------|
| **Decision** | Echo version stability policy |
| **Chosen option** | Option A — Strict pin v4.15.2 |
| **Source** | trade-offs.md (Major Decision 2) |
| **Approver** | architect (subagent) |
| **Date** | 2026-05-25 |
| **ADR ref** | ADR-01-003 |
| **Verdict rationale** | Maximum reproducibility; consistent with AGENTS.md pin; Rule 12 (Fail Loud) |

### Trade-Off 3: Frontend Build Target in Docker

| Field | Value |
|-------|-------|
| **Decision** | Frontend build target in Docker |
| **Chosen option** | Option A — Dev server in Docker (`pnpm dev`) |
| **Source** | trade-offs.md (Major Decision 3) |
| **Approver** | architect (subagent) |
| **Date** | 2026-05-25 |
| **ADR ref** | None (Phase 1 scope — ADR not created) |
| **Verdict rationale** | Phase 1 goal is demonstration of connectivity, not production parity |

### Trade-Off 4: Docker Compose Service Startup Order

| Field | Value |
|-------|-------|
| **Decision** | Docker Compose service startup order |
| **Chosen option** | Option B — `depends_on` without health checks |
| **Source** | trade-offs.md (Major Decision 4) |
| **Approver** | architect (subagent) |
| **Date** | 2026-05-25 |
| **ADR ref** | ADR-01-002 |
| **Verdict rationale** | Sufficient for Phase 1; Echo binds port <2s; true readiness check deferred to Phase 2 |

---

## ADR Summary

| ADR | Title | Decider | Date | Status |
|-----|-------|---------|------|--------|
| ADR-01-001 | Backend Port Allocation | architect | 2026-05-25 | Proposed |
| ADR-01-002 | Docker Compose Startup Order | architect | 2026-05-25 | Proposed |
| ADR-01-003 | Echo Version Stability Policy | architect | 2026-05-25 | Proposed |
| ADR-01-004 | CORS Configuration | architect | 2026-05-25 | Proposed |

All ADRs are in `specs/curated/BRD-01-app-shell/decisions/`.

---

## Parent Task Handoff Summary

|| Task | Role | Key outcome |
|------|------|-------------|
| t_506eba0b | PM validate-design gate | REPAIR_REQUIRED_BEFORE_GRADUATION; spawned t_0e13d59d |
| t_0e13d59d | Performance repair | All 10 PERF findings resolved; artifacts updated |
| t_199317c4 | Graduation validator | REQUEST_CHANGES: 3 graduation files missing (decision-record.md, validator-findings.md, implementation-readiness.md); spawned repair task t_9f9eeefa |
| t_9f9eeefa | This repair task | Files created/updated; graduation package complete; no unresolved blockers |

---

## Repair Rationale

### Why this repair was needed

The original graduation package (task t_bb9ddc2a) produced only `brd.md`. The three mandatory graduation evidence files — `decision-record.md`, `validator-findings.md`, and `implementation-readiness.md` — were not created, blocking the PM graduation gate (t_df1a4628) and the production-checklist child task (t_e96c7292).

Validator task t_199317c4 identified the gap and issued `REQUEST_CHANGES` with `verdict: REQUEST_CHANGES`.

### Trade-offs considered

| Option | Action | Trade-off |
|--------|--------|-----------|
| Create only the missing files | Write 3 files from scratch referencing existing artifacts | No content loss; clean handoff; minimal churn |
| Rewrite brd.md + all 4 files | Full regeneration | Unnecessary — brd.md was already valid |

Chosen: Option A — create the three missing files referencing existing artifact content. `brd.md` preserved as-is.

### Source tasks cited in this package

|| Task | Contribution |
|------|-------------|
| t_506eba0b | PM validate-design gate; 10 PERF findings; decision to repair before graduation |
| t_0e13d59d | Performance repair; all PERF findings resolved; 4 artifacts modified |
| t_199317c4 | Graduation validator; identified 3 missing files; issued REQUEST_CHANGES |
| t_9f9eeefa | This repair; created/updated 3 graduation files |

---

## Approver and Status

|| Field | Value |
|-------|-------|
| **Approver** | pm (profile) — via PM gate t_506eba0b |
| **Repair approver** | spec-writer (this task) |
| **Graduation gate** | t_df1a4628 (PM) — awaiting completion of t_e96c7292 (production checklist) |
| **Status** | Complete — all mandatory graduation files exist; no unresolved TBD/OQ/REQUEST_CHANGES markers |

---

## Records Index

- `specs/curated/BRD-01-app-shell/decisions/ADR-01-001-backend-port-allocation.md`
- `specs/curated/BRD-01-app-shell/decisions/ADR-01-002-docker-compose-startup-order.md`
- `specs/curated/BRD-01-app-shell/decisions/ADR-01-003-echo-version-stability-policy.md`
- `specs/curated/BRD-01-app-shell/decisions/ADR-01-004-cors-configuration.md`
- `specs/curated/BRD-01-app-shell/requirements.md`
- `specs/curated/BRD-01-app-shell/trade-offs.md`

---

*Decision record complete — BRD-01-app-shell gate decisions, OQ resolutions, trade-offs, and ADR references recorded.*