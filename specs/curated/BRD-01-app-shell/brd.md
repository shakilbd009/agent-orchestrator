# BRD-01 — Build Specification: App Shell

**BRD:** BRD-01-app-shell
**Type:** Canonical build specification for implementation and QA
**Source artifacts:** requirements.md, progressive-deepening-L1.md, progressive-deepening-L2.md, edge-cases-L2.md, trade-offs.md, ADR-01-001–004
**Status:** Complete — all findings resolved; no unresolved OQs, TBDs, or validator findings

---

## 1. Overview

BRD-01-app-shell establishes a minimal runnable system with two services: a Go/Echo backend exposing a `/health` endpoint about call, and a SvelteKit frontend that calls it from the browser. No business logic, no database, no orchestration. The shell proves integration works end-to-end and provides a clean base for all subsequent phases.

**Out of scope (confirmed):** authentication, database connections, business logic, agent orchestration, quality gates, dashboard features.

---

## 2. Functional Requirements

### FR-01-001 — Backend Health Endpoint

The Go/Echo server exposes `GET /health` on port 3001.

Response shape:

```
{
  "status": "ok",
  "version": "0.1.0",
  "timestamp": "<RFC3339>"
}
```

The `version` field is injected at build time via `-ldflags`. If injection fails, the value is an empty string or "dev" — the response always succeeds with HTTP 200 in Phase 1.

Source: requirements.md (FR-01-001) · ADR-01-001 (port 3001) · ADR-01-003 (Echo v4.15.2 strict pin)

### FR-01-002 — Frontend Landing Route

SvelteKit renders the root route (`/`). The page calls `GET /health` via `VITE_API_BASE_URL` (value: `http://localhost:3001`) and displays the response fields `status`, `version`, and `timestamp`.

Error state: if the backend is unreachable, the page displays an error message. The exact error UX is deferred to Phase 2; Phase 1 may show a raw error string.

Source: requirements.md (FR-01-002) · progressive-deepening-L1.md (component map) · progressive-deepening-L2.md (error path)

### FR-01-003 — Docker Compose Development Environment

`docker-compose up -d` starts both the backend container and the frontend container. The backend starts before the frontend container is created. No healthcheck definition exists in Phase 1.

Source: requirements.md (FR-01-003) · ADR-01A-002 (depends_on without health checks)

---

## 3. Non-Functional Requirements

| ID | Requirement | Target | Source |
|----|-------------|--------|--------|
| NFR-01-001 | Backend starts without errors | Echo binds to port within 10s | requirements.md |
| NFR-01-002 | Frontend builds without errors | `pnpm build` exits 0 within 60s | requirements.md |
| NFR-01-003 | Docker Compose starts all services | All containers reach running state | requirements.md |
| NFR-01-004a | /health cold-start latency | Response within 500ms on first call | requirements.md |
| NFR-01-004b | /health steady-state latency | Response within 100ms P95 | requirements.md |
| NFR-01-005 | Client fetch timeout | Browser fetch times out at 5s; backend responds within 100ms | requirements.md |
| NFR-01-006 | CORS headers on /health | Preflight requests from browser require `Access-Control-Allow-Origin` | requirements.md · ADR-01-004 |
| NFR-01-007 | Frontend graceful degradation | If backend unreachable, show user-friendly message | requirements.md (Phase 2) |
| NFR-01-008 | Docker Compose startup time | All services healthy within 60s warm start, 120s cold start | requirements.md |
| NFR-01-009 | Frontend hot-reload latency | File change reflected in browser within 3s of save | requirements.md |
| NFR-01-010 | Backend container resource bounds | Memory ≤ 256MB; CPU limit set for CI parity | requirements.md |
| NFR-01-011 | Frontend container resource bounds | Memory ≤ 512MB; CPU limit set for CI parity | requirements.md |
| NFR-01-012 | /health endpoint throughput | 100 concurrent requests, P99 < 50ms | requirements.md (Phase 2 k8s readiness probe base) |

### Performance Findings — All Resolved

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| PERF-001 | medium | NFR-01-008 added (60s warm / 120s cold startup) | requirements.md disposition table |
| PERF-002 | low | Latency split into cold-start 500ms and steady-state 100ms P95 | requirements.md disposition table |
| PERF-003 | low | NFR-01-005 rewritten: client timeout 5s, backend target 100ms separated | requirements.md disposition table |
| PERF-004 | medium | NFR-01-002 (60s build) + NFR-01-009 (3s hot-reload) added | requirements.md disposition table |
| PERF-005 | medium | NFR-01-010 (≤256MB) + NFR-01-011 (≤512MB) added | requirements.md disposition table |
| PERF-006 | info | Correctly documented; Phase 2 service_healthy defers resolution | requirements.md · ADR-01-002 |
| PERF-007 | info | Row replaced with explicit Phase 2 healthcheck deferral | trade-offs.md |
| PERF-008 | low | Browser timeout corrected: typical 2–5 min; AbortController 5s recommended | edge-cases-L2.md |
| PERF-009 | medium | NFR-01-012 added: 100 concurrent, P99 < 50ms | requirements.md disposition table |
| PERF-010 | low | Graceful shutdown TCP termination risk stated as Phase 2, not Phase 1 blocker | progressive-deepening-L2.md |

---

## 4. Architecture

### Component Structure

```
agent-orchestrator/
├── backend/              ← Go/Echo server; binds :3001
│   └── main.go           ← Entry point + GET /health handler
├── frontend/             ← SvelteKit application
│   └── src/routes/       ← Root route (+page.svelte)
├── docker-compose.yml    ← Two-service orchestration
└── .env / .env.example   ← VITE_API_BASE_URL=http://localhost:3001
```

### Data Flow

```
Browser (localhost:5173)
  └─ HTTP GET /
        └─ SvelteKit +page.svelte mounts
              └─ fetch(VITE_API_BASE_URL + "/health")
                    └─ Backend Echo (localhost:3001) GET /health
                          └─ 200 { status: "ok", version: "0.1.0", timestamp: "..." }
```

### CORS

The backend allows cross-origin requests from `http://localhost:5173` only. Preflight `OPTIONS` requests are handled by Echo CORS middleware. This is required because the frontend makes browser-side fetch calls directly to the backend on a different origin.

Source: ADR-01-004 (Option B — fixed origin)

### Docker Startup Ordering

The `backend` service starts first. The `frontend` service declares `depends_on: [backend]`, which waits for the backend container to start (not for the port to be bound). This is the Phase 1 behavior; Phase 2 will implement `service_healthy` conditions.

Source: ADR-01-002 (Option B — depends_on without health checks)

---

## 5. API Contract

### `GET /health`

**Request:** None.

**Response:** `200 OK` with content type `application/json`.

```json
{
  "status": "ok",
  "version": "0.1.0",
  "timestamp": "2026-05-25T12:00:00Z"
}
```

**CORS:** Endpoint responds to `OPTIONS` preflight with `Access-Control-Allow-Origin: http://localhost:5173`.

**Error cases (Phase 1):** Not applicable — the endpoint always returns 200 in Phase 1.

---

## 6. ADR Coverage

| ADR Decision | Artifact | Status |
|-------------|----------|--------|
| Port 3001 fixed | ADR-01-001 | Adopted |
| `depends_on` without health checks | ADR-01-002 | Adopted |
| Echo v4.15.2 strict pin | ADR-01-003 | Adopted |
| CORS scoped to `http://localhost:5173` | ADR-01-004 | Adopted |

---

## 7. Deferred to Phase 2

All items below are acknowledged but excluded from Phase 1 scope.

| Item | Reason Deferred | Impact if Deferred |
|------|-----------------|-------------------|
| Frontend user-friendly error state UX | Not specified in requirements; needs design decision | Phase 1 shows raw error string |
| Docker `healthcheck` for backend | `depends_on` is sufficient for Phase 1 shell | Docker cannot auto-detect backend failure |
| `depends_on` with `condition: service_healthy` | Same as above | Same as above |
| Docker volume mounts for hot reload | Phase 1 dev uses dev server; rebuild acceptable | File changes require container rebuild |
| Backend graceful shutdown (FIN before TCP termination) | Echo `e.Close()` not called in Phase 1 | Docker restart may cause immediate TCP termination |
| SSR / server-side fetch for frontend | Current SPA fetch pattern is intentional | CORS remains in scope for Phase 1 |

---

## 8. Acceptance Criteria

| ID | Criterion | Verification |
|----|-----------|--------------|
| AC-01 | `curl http://localhost:3001/health` returns JSON with `status` field | Manual or Playwright |
| AC-02 | Frontend loads in browser at `http://localhost:5173` and displays health response | Manual or Playwright |
| AC-03 | `docker-compose up -d && docker-compose ps` shows both containers running | Manual |
| AC-04 | Preflight `OPTIONS` request to `localhost:3001/health` returns 204 with CORS headers | Playwright or curl |
| AC-05 | Backend responds to `/health` within 100ms P95 under steady-state | Load test |
| AC-06 | Backend container memory ≤ 256MB at steady state | `docker stats` |
| AC-07 | Frontend container memory ≤ 512MB at steady state | `docker stats` |

---

*Canonical build spec — BRD-01-app-shell — Implementation and QA agents should treat this as the authoritative reference.*
