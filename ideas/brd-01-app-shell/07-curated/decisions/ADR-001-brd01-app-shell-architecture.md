# ADR-001: BRD-01 App Shell Architecture Decisions

**Status:** Accepted
**Date:** 2026-05-25
**Deciders:** architect (systematic-refinement session)
**BRD:** BRD-01 (App Shell)
**Supersedes:** None

---

## Context

BRD-01 (App Shell) is the first Phase 1 implementation task. It creates a minimal runnable scaffold with two services (Go/Echo backend, SvelteKit frontend) and their Docker orchestration. Several architectural decisions must be settled before implementation begins.

---

## Decision 1: Go Module Name

**Decision:** The backend Go module is named `agent-orchestrator-backend`.

**Rationale:** Using `agent-orchestrator-backend` as the module name:
- Clearly associates the module with the backend service
- Avoids confusion with the project root module (which is not a Go module)
- Prevents circular dependency risk if the project root ever imports the backend

**Alternatives considered:**
- `agent-orchestrator` — rejected; too generic, risks confusion with the overall project

---

## Decision 2: Backend Health Endpoint Path

**Decision:** The health endpoint is `GET /health` (not versioned).

**Rationale:** `/health` is the standard path for container orchestration health checks (Docker, Kubernetes). Versioning (`/api/v1/health`) is deferred to Phase 2 when other API endpoints exist and versioned routing becomes necessary.

**Alternatives considered:**
- `/api/v1/health` — rejected; premature versioning for a single diagnostic endpoint

---

## Decision 3: Frontend-to-Backend Communication

**Decision:** Frontend calls the backend via direct URL via `VITE_API_BASE_URL` env var.

**Rationale:** The simplest approach for Phase 1. CORS is enabled on the backend (no credentials in Phase 1). SvelteKit's built-in proxy (`$service` or custom handle hook) is deferred to Phase 2 when the dashboard requires authenticated same-origin API calls.

**Alternatives considered:**
- SvelteKit server-side proxy — rejected; adds complexity without Phase 1 benefit

---

## Decision 4: Docker Multi-Stage Builds

**Decision:** Both backend and frontend images use multi-stage builds.

**Rationale:** Standard production practice. The Go backend builds in a `golang:alpine` stage and copies the binary to a minimal `alpine` image. The Node/frontend builds in a `node:alpine` stage and copies output to `nginx:alpine`. This keeps production images small (~20MB for Go, ~15MB for nginx) and excludes build toolchains.

**Alternatives considered:**
- Single-stage builds — rejected; results in large images with exposed toolchains

---

## Consequences

### Positive
- Minimal, clear scaffold that Phase 2 agents can extend
- Docker images are production-sized from day one
- Health endpoint path matches container orchestration conventions
- No circular dependency risk between backend and frontend

### Negative
- `VITE_API_BASE_URL` must be set correctly per environment (handled via docker-compose environment block)
- Multi-stage Docker builds require Docker BuildKit (default on modern Docker, may need explicit enable on older versions)

### Neutral
- Frontend health card shows "unreachable" if backend is down — this is intentional Phase 1 behavior (no retry logic)
- Backend `/health` does not verify database/Redis connectivity — this is Phase 2 (per BRD-05, BRD-13)

---

## Review Cadence

This ADR is reviewed at the BRD-01 validation gate. If any decision needs to change before BRD-01 is marked `approved`, a new ADR supersedes this one.