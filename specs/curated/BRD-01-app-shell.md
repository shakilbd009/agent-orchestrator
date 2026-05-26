# BRD-01: Application Shell and Minimal Scaffold

**Project:** agent-orchestrator  
**Domain:** Orchestration  
**Phase:** 1  
**Owner:** developer  
**Status:** draft

---

## Goal

Establish a minimal, runnable application scaffold that demonstrates the backend and frontend are connectable, with tooling in place, before any feature code is written.

---

## Problem Statement

Phase 0 left `backend/` and `frontend/` as empty directories. Without a runnable shell, there is no way to validate tooling, demonstrate integration, or provide a foundation for Phase 2 feature development.

---

## Scope

### In Scope
- Go/Echo server in `backend/` that responds on a health endpoint
- SvelteKit application in `frontend/` with a single route
- Docker Compose configuration that runs both services
- Basic project tooling: TypeScript config, linting, package manager lockfiles

### Out of Scope
- Any feature code beyond a single health/ping endpoint
- Authentication, database connections, or business logic
- Agent orchestration, quality gates, or dashboard features

---

## Functional Requirements

### FR-01-001: Backend Health Endpoint
The Go/Echo backend exposes a `GET /health` endpoint that returns JSON:
```json
{ "status": "ok", "version": "0.1.0", "timestamp": "<RFC3339>" }
```

### FR-01-002: Frontend Landing Route
The SvelteKit frontend renders a single page at the root route confirming the UI is connected to the backend. The frontend uses `VITE_API_BASE_URL=http://localhost:3001` to call the backend API.

### FR-01-003: Docker Compose Development Environment
Docker Compose starts both backend and frontend services with port forwarding configured.

---

## Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-01-001 | Backend starts without errors | Echo server binds to port within 10 seconds |
| NFR-01-002 | Frontend builds without errors | `pnpm build` completes with exit code 0 |
| NFR-01-003 | Docker Compose starts all services | All containers reach healthy state |

---

## Acceptance Criteria

| ID | Criterion | Verification Method |
|----|-----------|---------------------|
| AC-01-001 | `curl http://localhost:PORT/health` returns JSON with status field | Demonstration |
| AC-01-002 | Frontend is accessible in browser at localhost | Demonstration |
| AC-01-003 | `docker-compose up -d` starts both services without error | Demonstration |

---

## Error Cases

| Scenario | Expected Behavior |
|----------|-------------------|
| Backend port is already in use | Docker Compose reports binding error; does not start |
| Frontend build fails | Build exits non-zero; error message visible in output |
| Docker daemon is not running | `docker-compose` command returns connection error |

---

## Edge Cases

| Scenario | Expected Behavior |
|----------|-------------------|
| Node or Go not installed | Tooling verification fails early with clear error |
| Port conflict between services | Docker Compose reports port binding failure |

---

## Open Questions

*(Resolved)*
- ~~Should the health endpoint return additional metadata (version, timestamp)?~~ → **Yes.** Return `{ status: "ok", version: string, timestamp: string }`.
- ~~What base URL should the frontend use to call the backend API?~~ → **`http://localhost:3001`** (direct, no proxy in Phase 1).

---

## Dependencies

- Phase 0 governance artifacts (AGENTS.md, STATUS.md, ADR-0001)
- Tool version pins in AGENTS.md (Go 1.25.6, Node v25.3.0, pnpm 10.32.1, Echo v4.15.2, sv CLI 0.15.3)

---

## Metadata

| Field | Value |
|-------|-------|
| Created | Phase 0 |
| Revised | Phase 1 (refined-on: 2026-05-25) |
| Version | 1 |
| Status | refined |

## ADR Index

| ADR | Topic | Status |
|-----|-------|--------|
| ADR-01-001 | Backend Port Allocation (3001 fixed) | Proposed |
| ADR-01-002 | Docker Compose Startup Order (depends_on) | Proposed |
| ADR-01-003 | Echo Version Stability Policy (strict pin v4.15.2) | Proposed |
| ADR-01-004 | CORS Configuration (scoped to localhost:5173) | Proposed |

*Refined via systematic-refinement (Stages 02-05). Artifacts in `specs/curated/BRD-01-app-shell/`.*
