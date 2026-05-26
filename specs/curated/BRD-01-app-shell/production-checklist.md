# Production Readiness Checklist — BRD-01-app-shell

**BRD:** BRD-01-app-shell
**Generated:** 2026-05-25
**Source:** specs/curated/BRD-01-app-shell/
**Purpose:** Deployment prerequisites before Phase 1 app shell goes live

---

## Pre-Flight: Tool Versions

Verify all prerequisites before running Docker Compose.

| # | Check | Command | Expected |
|---|-------|---------|----------|
| 1 | Docker daemon running | `docker info` | exits 0 |
| 2 | Docker Compose version | `docker-compose --version` | v5.0.2 |
| 3 | Go toolchain | `go version` | 1.25.6 |
| 4 | Node.js | `node --version` | v25.3.0 |
| 5 | npm | `npm --version` | 11.7.0 |
| 6 | pnpm | `pnpm --version` | 10.32.1 |
| 7 | Ports 3001 and 5173 free | `lsof -i :3001 -i :5173` | empty |

**Source:** implementation-readiness.md §4 (Production-Checklist Prerequisites)

---

## Build-Time Requirements

| # | Action | Detail | Source |
|---|--------|--------|--------|
| 8 | Inject version via `-ldflags` in backend Dockerfile | `-ldflags '-X main.version=0.1.0'` | progressive-deepening-L2.md §Build-Time Version Injection |
| 9 | Lock Echo to `v4.15.2` in `go.mod` | No `go get -u` without new ADR | ADR-01-003 / AGENTS.md |
| 10 | Set `VITE_API_BASE_URL=http://localhost:3001` in frontend container | In docker-compose.yml environment block | ADR-01-001 |

**Source:** brd.md §Functional Requirements · ADR-01-003 · progressive-deepening-L2.md

---

## Docker Compose

| # | Action | Detail | Source |
|---|--------|--------|--------|
| 11 | Backend `Dockerfile` in `./backend/` | Builds Go/Echo server | progressive-deepening-L2.md §Docker Pattern: Build context |
| 12 | Frontend `Dockerfile` in `./frontend/` | Builds SvelteKit app | progressive-deepening-L2.md §Docker Pattern: Build context |
| 13 | Backend binds port `3001:3001` in docker-compose.yml | `e.Start(":3001")` inside container | ADR-01-001 |
| 14 | Frontend binds port `5173:5173` in docker-compose.yml | Vite dev server | progressive-deepening-L2.md §Docker Pattern: Port forwarding |
| 15 | Frontend declares `depends_on: [backend]` | Container-start ordering only; no `condition: service_healthy` | ADR-01-002 |
| 16 | Inject `VITE_API_BASE_URL=http://localhost:3001` environment variable | In frontend service `environment:` block | progressive-deepening-L2.md §Docker Pattern: Environment |
| 17 | Do NOT add Docker `healthcheck` for backend | Deferred to Phase 2 | ADR-01-002 · brd.md §7 deferred items |

**Source:** ADR-01-001 · ADR-01-002 · brd.md §4

---

## Backend Implementation

| # | Action | Detail | Source |
|---|--------|--------|--------|
| 18 | Implement `GET /health` returning `{ "status": "ok", "version": "0.1.0", "timestamp": "<RFC3339>" }` | Always returns HTTP 200; no business logic | brd.md §2 (FR-01-001) |
| 19 | Add CORS middleware scoped to `http://localhost:5173` exactly | `middleware.CORSWithConfig({ AllowOrigins: []string{"http://localhost:5173"}, AllowMethods: []string{http.MethodGet, http.MethodOptions} })` | ADR-01-004 · edge-cases-L2.md §CORS Gap |
| 20 | Handle preflight `OPTIONS /health` via CORS middleware | Returns 204 No Content with CORS headers | ADR-01-004 |
| 21 | Handle port 3001 conflict (exit 1 if port in use) | Echo.Start() returns error on port conflict | progressive-deepening-L2.md §Edge Cases at L2 |
| 22 | Handle unknown routes (Echo 404, no body) | Default Echo behavior | progressive-deepening-L2.md §Edge Cases at L2 |
| 23 | Graceful shutdown NOT implemented in Phase 1 | `e.Close()` not called; Docker restart causes immediate TCP termination — Phase 2 risk | progressive-deepening-L2.md §Risks · brd.md §7 deferred |

**Source:** brd.md §5 (API Contract) · ADR-01-004 · progressive-deepening-L2.md §Edge Cases

---

## Frontend Implementation

| # | Action | Detail | Source |
|---|--------|--------|--------|
| 24 | Render root route `src/routes/+page.svelte` | Display health response fields | brd.md §2 (FR-01-002) |
| 25 | Fetch `VITE_API_BASE_URL + "/health"` via browser `fetch()` | CORS-enabled call to backend | brd.md §2 (FR-01-002) |
| 26 | Display `status`, `version`, and `timestamp` fields | From backend response | brd.md §2 (FR-01-002) |
| 27 | Handle backend unreachable: show error message | Try/catch around fetch; catch sets error message | progressive-deepening-L2.md §Frontend Response Handling |
| 28 | Do NOT implement user-friendly error state (deferred to Phase 2) | Phase 1 shows raw error string; acceptable | brd.md §7 deferred items |

**Source:** brd.md §2 (FR-01-002) · progressive-deepening-L2.md §Frontend Response Handling

---

## Performance & Resource Bounds

| # | Metric | Target | Verification |
|---|--------|--------|-------------|
| 29 | Backend cold-start latency | First response within 500ms of container start | Time from `docker-compose up -d` to first `curl http://localhost:3001/health` response |
| 30 | Backend steady-state P95 latency | < 100ms under normal conditions | Load test or `ab -n 1000 -c 10 http://localhost:3001/health` |
| 31 | /health throughput | 100 concurrent requests, P99 < 50ms | Load test; basis for Phase 2 k8s readiness probe |
| 32 | Client fetch timeout | ≤ 5s (AbortController recommended) | Edge-case: stop backend mid-request; verify 5s timeout |
| 33 | Frontend build time | `pnpm build` exits 0 within 60s | NFR-01-002 |
| 34 | Docker startup (warm) | All services running within 60s warm start | Timer from `docker-compose up -d` to `docker-compose ps` all running |
| 35 | Docker startup (cold) | All services running within 120s cold start | Timer after `docker-compose down -v`; first image pull |
| 36 | Frontend hot-reload latency | File change visible in browser within 3s | NFR-01-009 |
| 37 | Backend container memory | ≤ 256MB at steady state | `docker stats --no-stream` |
| 38 | Frontend container memory | ≤ 512MB at steady state | `docker stats --no-stream` |

**Source:** brd.md §3 (NFR table) · requirements.md §Performance Findings

---

## Acceptance Criteria Verification

Run these after `docker-compose up -d`:

| # | Criterion | Command | Expected |
|---|-----------|---------|----------|
| 39 | Backend /health returns JSON with `status` | `curl -f http://localhost:3001/health` | HTTP 200; `{ "status": "ok", ... }` |
| 40 | CORS preflight returns headers | `curl -X OPTIONS -i http://localhost:3001/health -H "Origin: http://localhost:5173" -H "Access-Control-Request-Method: GET"` | `Access-Control-Allow-Origin: http://localhost:5173` |
| 41 | Frontend loads and displays health response | Playwright or browser at `http://localhost:5173` | Page shows status, version, timestamp fields |
| 42 | Both containers running | `docker-compose ps` | backend and frontend both "Up" |
| 43 | Backend P95 latency | ab/brew test or script | P95 < 100ms |
| 44 | Backend memory ≤ 256MB | `docker stats --no-stream` | backend col < 256MB |
| 45 | Frontend memory ≤ 512MB | `docker stats --no-stream` | frontend col < 512MB |

**Source:** brd.md §8 (Acceptance Criteria) · implementation-readiness.md §8 (Verification Checklist)

---

## Rollback Procedures

| Scenario | Action |
|----------|--------|
| Backend fails to start | `docker-compose down` — no persistent state |
| Frontend fails to build | `docker-compose down && docker-compose build --no-cache frontend && docker-compose up -d` |
| CORS misconfiguration | Revert `backend/main.go` CORS changes; `docker-compose down && docker-compose up -d --build backend` |
| Version injection broken | Rebuild backend with correct `-ldflags`; no config files to restore |
| Docker Compose cluster fails | `docker-compose down -v` removes all containers and volumes; start fresh |

**Note:** Phase 1 has no persistent state. Rollback = stop and restart from clean build.

**Source:** implementation-readiness.md §6 (Rollback Notes)

---

## Deferred to Phase 2

The following are acknowledged but NOT in Phase 1 scope. Do not implement these as part of BRD-01-app-shell:

| Item | Reason Deferred | Impact if Deferred |
|------|-----------------|-------------------|
| Frontend user-friendly error state | Not specified in requirements; needs design | Phase 1 shows raw error string |
| Docker `healthcheck` for backend | `depends_on` sufficient for Phase 1 | Docker cannot auto-detect backend failure |
| `depends_on` with `condition: service_healthy` | Same as above | Same as above |
| Docker volume mounts for hot reload | Phase 1 dev uses dev server; rebuild acceptable | File changes require container rebuild |
| Backend graceful shutdown (`e.Close()` on SIGTERM) | Echo `e.Close()` not called in Phase 1 | Docker restart may cause immediate TCP termination |
| SSR / server-side fetch for frontend | SPA fetch pattern is intentional | CORS remains in scope for Phase 1 |

**Source:** brd.md §7 (Deferred to Phase 2) · trade-offs.md

---

## Constraints (Implementation Handoff Rules)

| Constraint | Source | Requirement |
|------------|--------|-------------|
| CORS origin must be exactly `http://localhost:5173` | ADR-01-004 | Do NOT use wildcard `*` |
| Echo version must remain `v4.15.2` | ADR-01-003 / AGENTS.md | No `go get -u` without new ADR |
| Backend port must be `:3001` | ADR-01-001 | Do not change without ADR |
| `depends_on` must NOT include `condition: service_healthy` | ADR-01-002 | Phase 1 uses container-start ordering only |
| No authentication or database connections | brd.md §1 | Phase 1 shell is unauthenticated; no DB |
| No business logic in `/health` response | brd.md §5 | Endpoint always returns 200 `"ok"` |
| No Phase 2 features in Phase 1 scope | brd.md §7 | 6 items deferred — do not anticipate |

**Source:** implementation-readiness.md §7 (Implementation Handoff Constraints)

---

*Generated by production-checklist skill — BRD-01-app-shell*
*Next gate: Before Phase 2 dispatch, verify all [ ] items above are checked or explicitly deferred*
