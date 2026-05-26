# BRD-01 — Requirements Analysis

**BRD:** BRD-01-app-shell  
**Stage:** 02-requirements  
**Status:** Analyzed

---

## Functional Requirements Analysis

### FR-01-001: Backend Health Endpoint

| Field | Value |
|-------|-------|
| **What** | Go/Echo server exposes `GET /health` |
| **Response body** | `{ "status": "ok", "version": "0.1.0", "timestamp": "<RFC3339>" }` |
| **Acceptance criteria** | `curl http://localhost:PORT/health` returns JSON with `status` field |
| **Priority** | MUST |
| **Open issues** | Port is not explicitly fixed; see ADR-01-001 |

### FR-01-002: Frontend Landing Route

| Field | Value |
|-------|-------|
| **What** | SvelteKit renders root route confirming UI is connected to backend |
| **API call** | Uses `VITE_API_BASE_URL=http://localhost:3001` to call backend |
| **Acceptance criteria** | Frontend is accessible in browser at localhost; shows health response or equivalent |
| **Priority** | MUST |
| **Open issues** | Frontend doesn't explicitly specify WHAT it renders (just "confirming connection") |

### FR-01-003: Docker Compose Development Environment

| Field | Value |
|-------|-------|
| **What** | Docker Compose starts both backend and frontend services |
| **Acceptance criteria** | `docker-compose up -d` starts all containers without error |
| **Priority** | MUST |
| **Open issues** | Network mode (host vs bridge), service startup order, health checks |

---

## Non-Functional Requirements Analysis

| ID | Requirement | Target | Gap |
|----|-------------|--------|-----|
| NFR-01-001 | Backend starts without errors | Echo server binds to port within 10s | Port value not locked; see ADR-01-001 |
| NFR-01-002 | Frontend builds without errors | `pnpm build` exits 0 within 60s | No hot-reload latency target |
| NFR-01-003 | Docker Compose starts all services | All containers reach healthy state | No `healthcheck` definitions in current draft |
| NFR-01-004a | Backend /health cold-start latency | Response within 500ms on first call (container just started) | Original NFR-01-004 captured this; see below for steady-state |
| NFR-01-004b | Backend /health steady-state latency | Response within 100ms P95 under normal conditions | Separate from cold-start; addresses PERF-002 |
| NFR-01-005 | Client fetch timeout configuration | Client fetch to /health times out at 5s; backend responds within 100ms | Original NFR-01-005 conflated client timeout with backend target; now corrected |
| NFR-01-006 | CORS headers on /health API | Pre-flight requests from browser require CORS | See ADR-01-004 |
| NFR-01-007 | Frontend graceful degradation | If backend unreachable, show user-friendly message | Implementation deferred to Phase 2 |
| NFR-01-008 | Docker Compose startup time | All services healthy within 60s warm start, 120s cold start (first pull) | Addresses PERF-001 |
| NFR-01-009 | Frontend hot-reload latency | File change reflected in browser within 3s of save | Addresses PERF-004 |
| NFR-01-010 | Backend container resource bounds | Memory ≤ 256MB; CPU limit for CI parity | Addresses PERF-005 |
| NFR-01-011 | Frontend container resource bounds | Memory ≤ 512MB; CPU limit for CI parity | Addresses PERF-005 |
| NFR-01-012 | /health endpoint throughput | Supports 100 concurrent requests with P99 < 50ms | Addresses PERF-009; prepares for Phase 2 k8s readiness probe |

### Missing NFRs — Resolved:

| NFR | Proposed Value | Rationale | Finding |
|-----|----------------|-----------|---------|
| Startup time | 60s warm, 120s cold | Total Docker Compose time (image pull + container create + backend start + frontend start); `depends_on` waits for container start only, not port binding — see ADR-01-002 | PERF-001 |
| /health latency split | Cold 500ms, steady 100ms | 500ms appropriate for cold-start; 100ms P95 for steady-state is the true degradation signal | PERF-002 |
| /health timeout split | Backend < 100ms; client ≤ 5s | Backend target is 100ms; 5s is browser/client timeout configuration, not a backend guarantee | PERF-003 |
| Frontend build time | `pnpm build` completes within 60s | Minimal SvelteKit shell should build in < 30s on warm cache; 60s provides margin | PERF-004 |
| Frontend hot-reload | Visible in browser within 3s of file save | Vite HMR target for developer iteration | PERF-004 |
| Resource bounds | Backend ≤ 256MB RAM, Frontend ≤ 512MB RAM | Prevents runaway process from consuming host resources | PERF-005 |
| /health throughput | 100 concurrent requests, P99 < 50ms | Basis for Phase 2 Kubernetes readiness probe | PERF-009 |

---

## Out of Scope (confirmed)

- Authentication
- Database connections
- Business logic
- Agent orchestration
- Quality gates
- Dashboard features

---

## Assumptions Made

| Assumption | Validation Plan |
|------------|-----------------|
| Port 3001 is used for backend | Validate no port conflicts with existing services on dev machine |
| Go module path is `github.com/agent-orchestrator/backend` or equivalent | Confirm during scaffold creation |
| Frontend calls backend directly from browser (no server-side proxy) | Verify VITE_API_BASE_URL pattern |
| Docker runs in bridge network mode | Can be changed via ADR if host mode preferred |
| Both services start successfully without interdependency | Services can gracefully handle backend being unavailable |

---

## Success Metrics

| Metric | Measurement |
|--------|-------------|
| Backend health endpoint reachable | `curl -f http://localhost:3001/health` exits 0 |
| Frontend loads in browser | Playwright or manual verification |
| Docker Compose starts cleanly | `docker-compose up -d && docker-compose ps` shows all running |
| No port conflicts on standard ports | One-shot start validation |

---

## Actor Identification

| Actor | Needs | Pain Point if Unmet |
|-------|-------|---------------------|
| Developer | Fast local iteration | Can't verify own changes |
| CI pipeline | Reproducible environment | Tests can't run |
| Future phases | Clean base to build on | Retrofitting constraints is expensive |

---

## Red Flags Check

| Flag | Status | Notes |
|------|--------|-------|
| Solution bias | ✅ PASS | Describes WHAT, not HOW |
| Vague requirements | ⚠️ PARTIAL | Frontend "confirming connection" is underspecified |
| No success metrics | ⚠️ PARTIAL | AC defined but no explicit Pass/Fail thresholds |
| Scope creep | ✅ PASS | Out of scope explicitly listed |
| "Figure it out later" | ⚠️ FOUND | Port fixed vs configurable unresolved |

---

*Analyzed: Stage 02 requirements*
*Next: Stage 03 trade-offs → create ADRs*

---

## validate-design Performance Findings — Disposition Record

| Finding | Severity | Artifact | Disposition |
|---------|----------|----------|-------------|
| PERF-001 | medium | requirements.md (NFR-01-008) | Fixed: NFR-01-008 added — Docker Compose all-services healthy within 60s warm, 120s cold. |
| PERF-002 | low | requirements.md (NFR-01-004a/b) | Fixed: /health split into cold-start (500ms) and steady-state (100ms P95). |
| PERF-003 | low | requirements.md (NFR-01-005) | Fixed: NFR-01-005 rewritten as client timeout configuration (≤5s) with backend target (100ms) separated. |
| PERF-004 | medium | requirements.md (NFR-01-002, NFR-01-009) | Fixed: NFR-01-002 now includes 60s build time bound; NFR-01-009 added for hot-reload (3s). |
| PERF-005 | medium | requirements.md (NFR-01-010, NFR-01-011) | Fixed: NFR-01-010 and NFR-01-011 added — backend ≤256MB RAM, frontend ≤512MB RAM. |
| PERF-006 | info | ADR-01-002 | No action: correctly documented bottleneck; Phase 2 service_healthy defers resolution. |
| PERF-007 | info | trade-offs.md | Fixed: row replaced generic "health endpoint is also a probe" with explicit Phase 2 healthcheck deferral. |
| PERF-008 | low | edge-cases-L2.md | Fixed: browser fetch timeout corrected from "5-10 minute default" to "typically 2–5 min; explicit AbortController 5s recommended". |
| PERF-009 | medium | requirements.md (NFR-01-012) | Fixed: NFR-01-012 added — 100 concurrent requests, P99 < 50ms for Phase 2 k8s readiness probe basis. |
| PERF-010 | low | progressive-deepening-L2.md | Fixed: graceful shutdown row now explicitly states TCP termination risk without FIN as Phase 2 risk, not Phase 1 blocker. |
