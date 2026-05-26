# ADR-01-002: Docker Compose Service Startup Order

**Status:** Proposed  
**Date:** 2026-05-25  
**Deciders:** architect (subagent, BRD-01 refinement)  
**Supersedes:** None  
**Superseded by:** None  

---

## Context

BRD-01 FR-01-003 requires Docker Compose to start both backend and frontend services. The current BRD draft does not specify startup ordering or dependency management. Frontend makes HTTP calls to backend at startup to confirm connectivity — if backend is not ready, those calls fail.

---

## Decision Drivers

1. Frontend calls `VITE_API_BASE_URL/health` to confirm backend connectivity
2. Backend initializes in under 10s (NFR-01-001)
3. Phase 1 shell avoids complexity — health checks add configuration weight
4. `depends_on` is available in Docker Compose v3+ without extra configuration

---

## Options Considered

### Option A: No ordering constraints, fail-fast (No dependency declaration)

Services start independently. No `depends_on` or `healthcheck`.

**Pros:**
- Simplest Docker Compose configuration
- Services start as fast as possible (parallel)
- No circular dependency risk

**Cons:**
- Frontend may repeatedly 500/503 while backend initializes
- "Ready" vs "Started" — container running doesn't mean Echo has bound the port
- Repeated retry calls from frontend may cause confusing error states in UI

### Option B: `depends_on` with container start ordering (Chosen)

Frontend declares `depends_on: [backend]`.

**Pros:**
- Guarantees backend container starts before frontend container
- Backend Echo server is fast — typically binds within <2s
- No health check configuration required
- Docker Compose guarantees container start order

**Cons:**
- `depends_on` only waits for container to start, not for the process inside to be "ready"
- If backend process is slow to start, frontend still starts before backend is listening

### Option C: `depends_on` with `condition: service_healthy`

Requires backend to implement a Docker healthcheck ( wget curl against /health ) and frontend to wait for `service_healthy`.

**Pros:**
- Guarantees true readiness — frontend only starts when backend /health returns 200
- Idempotent startup validation

**Cons:**
- Requires Echo to serve health check on the same port
- More complex Docker Compose file
- Overkill for Phase 1 shell — adds configuration weight that obscures Phase 1 scope

---

## Decision

**Option B — `depends_on` without health checks** is adopted for Phase 1.

Docker Compose will use `depends_on: [backend]` on the frontend service. This is sufficient for Phase 1 because:
- Echo binds to port very quickly (<2s in practice)
- Phase 1 is explicitly a shell — frontend connectivity errors are acceptable as demonstrable defects
- Health check complexity is deferred to Phase 2+

---

## Consequences

### Positive
- Minimal Docker Compose configuration
- Correct semantics for Phase 1 — simpler than health checks
- Easy to upgrade to Option C in Phase 2 when more complex patterns emerge

### Negative
- Frontend may briefly show error state if backend takes >2s to bind
- Not a true "microservice readiness" contract (only container start order)

### Neutral
- SvelteKit dev server also restarts on file changes — this could race with backend restarts
- Phase 3+ may introduce service probes for more robust orchestration

---

## Implementation Notes

```yaml
services:
  backend:
    build: ./backend
    ports:
      - "3001:3001"
  frontend:
    build: ./frontend
    depends_on:
      - backend
    ports:
      - "5173:5173"
    environment:
      - VITE_API_BASE_URL=http://localhost:3001
```

**Note:** `depends_on` uses the default `service_started` condition. Future upgrade path:
```yaml
depends_on:
  backend:
    condition: service_healthy
```

---

## References

- BRD-01-app-shell.md (FR-01-003, NFR-01-001, NFR-01-002)
- Docker Compose `depends_on` documentation
