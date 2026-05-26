# BRD-01-app-shell — Validator Findings

**BRD:** BRD-01-app-shell
**Stage:** Graduation evidence (validator findings gate)
**Source tasks:** t_0e13d59d (performance repair) · t_506eba0b (PM validate-design gate)
**Status:** All findings dispositioned — no blockers

---

## 1. Completeness Score

| Dimension | Score | Evidence |
|-----------|-------|----------|
| Functional requirements (FR-01-001 to FR-01-003) | 3/3 | All three FRs have acceptance criteria and are traced to source docs |
| Non-functional requirements (NFR-01-001 to NFR-01-012) | 12/12 | All 12 NFRs defined with targets; gaps addressed by repair task t_0e13d59d |
| Open questions (OQs) | 0 unresolved | All historical OQs resolved via ADR or explicit deferral to Phase 2 |
| TBDs | 0 unresolved | No open TBDs in curated artifacts |
| Performance findings (PERF-001 to PERF-010) | 10/10 dispositioned | All 10 findings from t_0e13d59d are dispositioned below |
| ADR coverage | 4/4 | ADR-01-001 through ADR-01-004 all adopted and cited |
| Deferred items | 6 acknowledged | All deferred items listed in brd.md §7 with Phase 2 rationale |

**Overall completeness: 100%** — graduation evidence package is complete.

Source: brd.md · requirements.md disposition table · decision-record.md

---

## 2. validate-design Findings — Disposition Record

All findings below originate from task **t_0e13d59d** (performance repair) and are confirmed dispositioned in the curated artifact set. No finding remains open.

| Finding | Severity | Artifact | Disposition |
|---------|----------|----------|-------------|
| **PERF-001** | medium | requirements.md (NFR-01-008) | **Fixed.** NFR-01-008 added: Docker Compose all services reach healthy state within 60s warm start, 120s cold start (first pull). Explicitly separates `depends_on` container-start wait from port-binding readiness. |
| **PERF-002** | low | requirements.md (NFR-01-004a/b) | **Fixed.** /health latency split into two NFRs: NFR-01-004a (cold-start ≤ 500ms on first call) and NFR-01-004b (steady-state ≤ 100ms P95). Corrects original NFR-01-004 that conflated cold and warm targets. |
| **PERF-003** | low | requirements.md (NFR-01-005) | **Fixed.** NFR-01-005 rewritten: client-side fetch timeout ≤ 5s (browser AbortController configuration) is now separated from the backend target of ≤ 100ms. Clear distinction between client-boundary requirement and server-side performance target. |
| **PERF-004** | medium | requirements.md (NFR-01-002, NFR-01-009) | **Fixed.** NFR-01-002 updated to include 60s build-time bound for `pnpm build`. NFR-01-009 added: file change reflected in browser within 3s of save (Vite HMR target for developer iteration). |
| **PERF-005** | medium | requirements.md (NFR-01-010, NFR-01-011) | **Fixed.** NFR-01-010 added: backend container memory ≤ 256MB with CPU limit for CI parity. NFR-01-011 added: frontend container memory ≤ 512MB with CPU limit for CI parity. Prevents runaway processes from consuming host resources. |
| **PERF-006** | info | ADR-01-002 | **No action.** Correctly documented in ADR-01-002 as the chosen design option (depends_on without health checks). Resolution deferred to Phase 2 via `service_healthy` condition. No change required; Phase 1 shell operates as specified. |
| **PERF-007** | info | trade-offs.md | **Fixed.** The trade-offs.md row that previously read generically "health endpoint is also a probe" has been replaced with explicit text: healthcheck implementation is deferred to Phase 2 when `service_healthy` conditions are available in Docker Compose. |
| **PERF-008** | low | edge-cases-L2.md | **Fixed.** Browser fetch timeout corrected from the incorrect "5–10 minute default" to the accurate statement: browser fetch timeouts are typically 2–5 minutes; explicit AbortController with 5s timeout is recommended for /health calls. |
| **PERF-009** | medium | requirements.md (NFR-01-012) | **Fixed.** NFR-01-012 added: /health endpoint must support 100 concurrent requests with P99 < 50ms. This provides the quantitative basis for Phase 2 Kubernetes readiness probe configuration. |
| **PERF-010** | low | progressive-deepening-L2.md | **Fixed.** Graceful shutdown row now explicitly states that TCP connection termination without a proper FIN sequence is a Phase 2 risk, documented as such, and is not a Phase 1 graduation blocker. Phase 1 accepts Docker restart as potentially causing immediate TCP termination. |

---

## 3. Finding Summary by Severity

| Severity | Count | Status |
|----------|-------|--------|
| medium | 4 (PERF-001, PERF-004, PERF-005, PERF-009) | All fixed — NFRs added to requirements.md |
|| low | 4 (PERF-002, PERF-003, PERF-008, PERF-010) | All fixed — prose corrections in respective artifacts |
| info | 2 (PERF-006, PERF-007) | One no-action, one fixed |
| **Total** | **10** | **10 dispositioned · 0 open** |

---

## 4. Source Document Cross-Reference

| Finding | Primary Fix Artifact | Backup Citation |
|---------|---------------------|-----------------|
| PERF-001 | requirements.md §NFR-01-008 | brd.md §3 NFR table |
| PERF-002 | requirements.md §NFR-01-004a/b | brd.md §3 |
| PERF-003 | requirements.md §NFR-01-005 | brd.md §3 |
| PERF-004 | requirements.md §NFR-01-002, NFR-01-009 | brd.md §3 |
| PERF-005 | requirements.md §NFR-01-010, NFR-01-011 | brd.md §3 |
| PERF-006 | ADR-01-002 | requirements.md disposition table |
| PERF-007 | trade-offs.md | requirements.md disposition table |
| PERF-008 | edge-cases-L2.md | requirements.md disposition table |
| PERF-009 | requirements.md §NFR-01-012 | brd.md §3 |
| PERF-010 | progressive-deepening-L2.md | requirements.md disposition table |

---

## 5. Open Items (Not Blockers — Phase 2 Follow-Through)

The following items are acknowledged in brd.md §7 and are not graduation blockers. They are logged here for traceability.

|| Item | Phase 2 Action | Impact if Deferred |
|------|----------------|-------------------|
| Frontend user-friendly error state UX | Design decision required | Phase 1 shows raw error string |
| Docker `healthcheck` for backend container | Implement `service_healthy` condition in docker-compose.yml | Docker cannot auto-detect backend failure |
| `depends_on condition: service_healthy` | Upgrade docker-compose.yml | Service ordering uses container-start wait only |
| Docker volume mounts for hot reload | Add bind mounts to docker-compose.yml | File changes require container rebuild |
| Backend graceful shutdown (FIN before TCP termination) | Call Echo `e.Close()` on SIGTERM | Docker restart may cause immediate TCP termination |
| SSR / server-side fetch for frontend | Implement SvelteKit server-side load function | CORS remains in scope for Phase 1 |

---

## 6. Missing-File Findings (t_199317c4)

Validator task **t_199317c4** identified that the original graduation package produced by task t_bb9ddc2a was incomplete: only `brd.md` existed; three mandatory graduation evidence files were absent.

|| Finding | Source Task | Disposition |
|---------|------------|-------------|
| `decision-record.md` missing | t_199317c4 | **Created.** Added repair rationale, trade-offs, source-task citations (t_506eba0b, t_0e13d59d, t_199317c4, t_9f9eeefa), approver/status, and open items section. |
| `validator-findings.md` missing | t_199317c4 | **Created.** This document consolidates PERF-001–PERF-010 disposition from t_0e13d59d plus the missing-file findings from t_199317c4. |
| `implementation-readiness.md` missing | t_199317c4 | **Created.** Summarizes eval contracts, feature flags, production-checklist prerequisites, risks, rollback notes, and implementation constraints. |

**t_199317c4 verdict:** `REQUEST_CHANGES` — all three files now exist. No outstanding REQUEST_CHANGES markers remain.

Source: t_199317c4 (graduation validator)

---

*Validator findings gate: t_0e13d59d + t_506eba0b → t_bb9ddc2a*
*This document is part of the BRD-01-app-shell graduation evidence package.*