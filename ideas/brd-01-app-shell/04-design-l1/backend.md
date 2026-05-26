# BRD-01: Application Shell — L1 Design

**Project:** agent-orchestrator
**BRD:** BRD-01 (App Shell)
**Stage:** L1 (first complete pass)
**Owner:** developer

---

## What

BRD-01 establishes a minimal, runnable application scaffold with two independent services:

1. **Backend:** Go/Echo HTTP server in `backend/` that exposes a health endpoint
2. **Frontend:** SvelteKit SPA in `frontend/` that displays a single landing page

A `docker-compose.yml` at project root orchestrates both services for local development.

---

## Why

Phase 0 produced no source code. BRD-01 is the first implementation gate:

- Validates that tooling (Go, Node, pnpm, Docker) works in this environment
- Proves backend and frontend are separately runnable
- Provides a foundation that Phase 2 agents (orchestration, dashboard, quality gates) can build upon
- Gives stakeholders a demo-ready artifact

---

## Key Insight

The app shell is intentionally thin. The goal is a **minimum runnable scaffold**, not a feature. Any business logic, database access, or agent orchestration that sneaks into this phase is out-of-scope creep.

---

## Component Overview

```
┌─────────────────────────────────────────────────────┐
│  Docker Compose (project root)                      │
│  ┌──────────────────┐   ┌────────────────────────┐  │
│  │ backend service  │   │ frontend service        │  │
│  │ Go/Echo :3001    │   │ SvelteKit :5173        │  │
│  │                  │◄──│ VITE_API_BASE_URL=      │  │
│  │                  │   │ http://localhost:3001   │  │
│  └──────────────────┘   └────────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

---

## Component: backend/

**Location:** `backend/`
**Language:** Go 1.25.6
**Framework:** Echo v4.15.2 (import: `github.com/labstack/echo/v4`)
**Port:** 3001

### What it does

- Starts an Echo HTTP server on port 3001
- Handles `GET /health` returning `{ "status": "ok", "version": "0.1.0", "timestamp": "<RFC3339>" }`
- All other routes return 404

### Key files

```
backend/
├── main.go              # Entry point; starts Echo server
├── go.mod               # Module: agent-orchestrator-backend; go 1.25.6
├── go.sum               # Dependency lock
├── Dockerfile           # Builds Go binary; runs on alpine
└── .gitkeep             # (Phase 0留下的空文件标记)
```

### Data flows

**Happy path — health check:**
```
Client → GET /health → Echo router → HealthHandler → JSON response
```

**Error path — unknown route:**
```
Client → GET /unknown → Echo router → 404 handler → 404 JSON response
```

### Error scenarios

| Scenario | Expected behavior |
|----------|-------------------|
| Port 3001 already in use | `listen tcp :3001: bind: address already in use`; process exits non-zero |
| Docker daemon not running | `docker-compose` returns connection error |
| Go build fails | `go build` exits non-zero; Dockerfile build layer fails |

---

## Component: frontend/

**Location:** `frontend/`
**Runtime:** Node v25.3.0
**Tool:** SvelteKit sv CLI 0.15.3
**Package manager:** pnpm 10.32.1
**Port:** 5173 (dev server)

### What it does

- Renders a single SvelteKit page at `/` (root route)
- Displays a simple status card showing backend connectivity status
- Uses `VITE_API_BASE_URL` env var to call backend API

### Key files

```
frontend/
├── src/
│   ├── routes/
│   │   └── +page.svelte     # Landing page component
│   └── app.html             # HTML shell
├── package.json             # pnpm dependencies
├── pnpm-lock.yaml           # Lockfile (pnpm 10.32.1)
├── svelte.config.js         # SvelteKit config
├── vite.config.ts           # Vite config; defines VITE_API_BASE_URL
├── Dockerfile               # Builds SvelteKit app; runs on nginx:alpine
├── playwright.config.ts     # Playwright config (added in Phase 1)
└── .gitkeep
```

### Data flows

**Happy path — page load:**
```
Browser → GET / → SvelteKit router → +page.svelte → renders landing UI
Browser → GET /health → fetch(VITE_API_BASE_URL/health) → status card updates
```

### Edge cases

| Scenario | Expected behavior |
|----------|-------------------|
| `VITE_API_BASE_URL` not set | Vite build warning; app defaults to `http://localhost:3001` |
| Backend not running | Frontend loads but health card shows "unreachable" state |
| Node not installed | `pnpm install` fails with clear error |

---

## Component: docker-compose.yml

**Location:** project root (already exists from Phase 0, will be extended)

### Phase 1 additions to existing Phase 0 infrastructure services

The Phase 0 `docker-compose.yml` defines `database` (PostgreSQL 17) and `cache` (Redis 7) services. BRD-01 adds:

```yaml
services:
  backend:
    build: ./backend
    ports:
      - "3001:3001"
    environment:
      - PORT=3001
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:3001/health"]
      interval: 10s
      timeout: 5s
      retries: 5
    depends_on:
      database:
        condition: service_healthy
      cache:
        condition: service_healthy

  frontend:
    build: ./frontend
    ports:
      - "5173:5173"
    environment:
      - VITE_API_BASE_URL=http://localhost:3001
    depends_on:
      - backend
```

### Data flows

**Startup sequence:**
```
docker-compose up → database starts → cache starts → backend starts → frontend starts
```

### Error scenarios

| Scenario | Expected behavior |
|----------|-------------------|
| Backend build fails | Docker build layer fails; `docker-compose up` exits non-zero |
| Frontend build fails | Docker build layer fails; `docker-compose up` exits non-zero |
| Port conflict on 3001 or 5173 | Docker reports port binding error |

---

## Initial Questions Raised

1. Should backend health check also verify database/Redis connectivity, or just confirm the process is alive?
   - **Decision needed:** Phase 1 scope is process-liveness only; connection pooling verified in Phase 2
2. Should the frontend use a proxy to forward `/api/*` requests to backend, or use direct `VITE_API_BASE_URL`?
   - **Decision needed:** Direct URL in Phase 1; SvelteKit proxy deferred to Phase 2 (BRD-03 dashboard)
3. What version string should the backend return? Hardcoded `"0.1.0"` or injected at build time?
   - **Decision needed:** Hardcoded `"0.1.0"` for Phase 1; build-time injection via ldflags in Phase 2

---

## Dependencies

- Phase 0 governance artifacts (AGENTS.md, STATUS.md, ADR-0001)
- Tool version pins in AGENTS.md (Go 1.25.6, Node v25.3.0, pnpm 10.32.1, Echo v4.15.2, sv CLI 0.15.3)
- Existing `docker-compose.yml` Phase 0 infrastructure services (database, cache)

---

## Verification Plan

| Check | Method |
|-------|--------|
| `curl http://localhost:3001/health` returns 200 with JSON | Manual demonstration |
| Frontend accessible at localhost:5173 | Manual demonstration |
| `docker-compose up -d` starts all services | `docker-compose ps` shows healthy |
| `pnpm build` in frontend/ exits 0 | Terminal output |
| Architecture eval scripts SKIP or PASS | `./evals/architecture/*.sh` |