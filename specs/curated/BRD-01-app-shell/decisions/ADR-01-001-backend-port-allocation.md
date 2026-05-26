# ADR-01-001: Backend Port Allocation

**Status:** Proposed  
**Date:** 2026-05-25  
**Deciders:** architect (subagent, BRD-01 refinement)  
**Supersedes:** None  
**Superseded by:** None  

---

## Context

BRD-01-001 establishes a Go/Echo health endpoint at a port that must be reachable from the frontend. OQ-2 was resolved to `http://localhost:3001` as the frontend base URL, but the port value itself was not formally recorded as an ADR-ee decision. This ADR captures that decision.

---

## Decision Drivers

1. OQ-2 resolved to `localhost:3001` as the backend API base URL
2. Docker Compose needs a predictable port for port-forwarding
3. Frontend `VITE_API_BASE_URL` must be stable across the project lifetime
4. Port 3001 is the resolved consensus value (not in conflict with common well-known ports)

---

## Options Considered

### Option A: Fixed port 3001 (Chosen)

Backend always binds to `:3001`. Frontend's `VITE_API_BASE_URL` defaults to `http://localhost:3001`.

**Pros:**
- Simple, zero configuration overhead
- Matches resolved OQ-2 value — no inconsistency between BRD and implementation
- Predictable for Docker Compose port mapping
- Common convention for backend APIs in monorepo setups

**Cons:**
- If port 3001 is occupied, startup fails (port conflict)
- No flexibility for running multiple backend instances

### Option B: Configurable port via `BACKEND_PORT` env var

Default 3001, overridable via environment variable.

**Pros:**
- Handles port conflict scenarios gracefully
- Flexible for different dev environments

**Cons:**
- Introduces configuration complexity in Phase 1 (minimal scope)
- Frontend would need dynamic port resolution for service discovery

### Option C: Dynamic port (`:0`)

Backend binds to a port assigned by the OS.

**Pros:**
- Zero port conflict risk
- Idiomatic for microservices

**Cons:**
- Announcing the chosen port requires coordination (service discovery, config service, or file)
- Frontend cannot configure `VITE_API_BASE_URL` statically
- Overcomplicates Phase 1 shell

---

## Decision

**Option A — Fixed port 3001** is adopted for Phase 1.

The backend Go/Echo server binds explicitly to `:3001`. This is the port the frontend uses for `VITE_API_BASE_URL`. This value may only be changed via a subsequent ADR.

---

## Consequences

### Positive
- Zero configuration complexity in Phase 1
- Frontend `VITE_API_BASE_URL` is a simple compile-time constant
- Docker Compose port mapping is deterministic (`3001:3001`)

### Negative
- Port 3001 conflict requires manual resolution (change the process using 3001, or use a wrapper script)
- Not suitable if we later need multiple backend instances

### Neutral
- Phase 2+ can introduce configurable ports via ADR without breaking Phase 1 decisions

---

## Implementation Notes

- Backend: `e.Start(":3001")` in Go/Echo
- Docker Compose: `ports: - "3001:3001"` in `backend` service
- Frontend: `VITE_API_BASE_URL=http://localhost:3001` in `.env` and/or `docker-compose.yml`

---

## Review Cadence

Reviewed at Phase 1 → Phase 2 transition. If port conflicts are common, this ADR will be superseded with Option B.

---

## References

- BRD-01-app-shell.md (FR-01-001, FR-01-002)
- OQ-2 resolution: `http://localhost:3001`
