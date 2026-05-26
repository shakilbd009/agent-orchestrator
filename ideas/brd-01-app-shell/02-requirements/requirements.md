# BRD-01: Application Shell — Requirements Analysis

**Project:** agent-orchestrator
**BRD:** BRD-01 (App Shell)
**Owner:** developer
**Phase:** 1

---

## Problem Statement

Phase 0 left `backend/` and `frontend/` as empty directories. Without a runnable shell, there is no way to validate tooling, demonstrate integration, or provide a foundation for Phase 2 feature development.

---

## Target Users

- **Developers** onboarding to the project who need a working local environment
- **Agents** (architect, qa, reviewer) who need a runnable scaffold to validate integrations
- **Human stakeholders** who need a demo-ready artifact after Phase 1

---

## Functional Requirements

### FR-01-001: Backend Health Endpoint
The Go/Echo backend exposes a `GET /health` endpoint that returns JSON:
```json
{ "status": "ok", "version": "0.1.0", "timestamp": "<RFC3339>" }
```
- **Acceptance criteria:** `curl http://localhost:PORT/health` returns JSON with status field
- **Priority:** Must have

### FR-01-002: Frontend Landing Route
The SvelteKit frontend renders a single page at the root route confirming the UI is connected to the backend.
- **Acceptance criteria:** Frontend is accessible in browser at localhost
- **Priority:** Must have

### FR-01-003: Docker Compose Development Environment
Docker Compose starts both backend and frontend services with port forwarding configured.
- **Acceptance criteria:** `docker-compose up -d` starts both services without error; all containers reach healthy state
- **Priority:** Must have

---

## Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-01-001 | Backend starts without errors | Echo server binds to port within 10 seconds |
| NFR-01-002 | Frontend builds without errors | `pnpm build` completes with exit code 0 |
| NFR-01-003 | Docker Compose starts all services | All containers reach healthy state |
| NFR-01-004 | Backend responds to health check | `curl` returns JSON within 500ms |

---

## Out of Scope

- Any feature code beyond a single health/ping endpoint
- Authentication, database connections, or business logic
- Agent orchestration, quality gates, or dashboard features
- Production-grade deployment configurations

---

## Constraints

- **Tooling:** Go 1.25.6, Echo v4.15.2, Node v25.3.0, pnpm 10.32.1, SvelteKit sv CLI 0.15.3
- **Package manager:** pnpm (lockfile: `pnpm-lock.yaml`)
- **Docker:** standalone `docker-compose` v5.0.2 (not `docker compose` plugin)
- **Phase 0 rule:** No source code in Phase 0; backend/frontend empty; all governance artifacts must precede implementation
- **No separate CLAUDE.md** — AGENTS.md is the agent constitution

---

## Success Metrics

| Metric | Target |
|--------|--------|
| Backend /health returns 200 in < 500ms | 100% |
| Frontend loads at localhost:PORT | 100% |
| `docker-compose up -d` exits 0 | 100% |
| `pnpm build` in frontend/ exits 0 | 100% |
| Architecture eval scripts SKIP or PASS | 100% |

---

## Open Questions

_(Resolved)_
- ~~Should the health endpoint return additional metadata (version, timestamp)?~~ → **Yes.** Return `{ status: "ok", version: string, timestamp: string }`.
- ~~What base URL should the frontend use to call the backend API?~~ → **`http://localhost:3001`** (direct, no proxy in Phase 1).

_(Open)_
- Should the backend health endpoint also verify database/redis connectivity, or only confirm the server process is alive? (Decision: Phase 1 scope is process-liveness only; connection pooling verified in Phase 2 per BRD-05/BRD-13)