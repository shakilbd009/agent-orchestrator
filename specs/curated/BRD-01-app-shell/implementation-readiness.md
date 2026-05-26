# BRD-01-app-shell — Implementation Readiness

**BRD:** BRD-01-app-shell
**Stage:** Graduation evidence (implementation readiness gate)
**Source tasks:** t_506eba0b (PM gate) · t_0e13d59d (performance repair)
**Status:** Ready for implementation

---

## 1. Eval Contracts

### Health Endpoint Test Contract

| Field | Value |
|-------|-------|
| **Endpoint** | `GET http://localhost:3001/health` |
| **Expected response** | `200 OK`, `Content-Type: application/json`, body `{ "status": "ok", "version": "0.1.0", "timestamp": "<RFC3339>" }` |
| **Success condition** | Response has `status` field with value `"ok"` |
| **CORS preflight** | `OPTIONS /health` from `http://localhost:5173` returns `204 No Content` with `Access-Control-Allow-Origin: http://localhost:5173` |
| **Latency (cold-start)** | First response within 500ms of container start |
| **Latency (steady-state)** | P95 response time under 100ms |
| **Throughput** | 100 concurrent requests, P99 < 50ms |
| **Verification** | `curl -f http://localhost:3001/health` exits 0 |

Source: brd.md §5 (API Contract) · NFR-01-004a, NFR-01-004b, NFR-01-012

---

## 2. Feature Flags

No feature flags are active for Phase 1. The Phase 0 feature-flag-registry.md defines 26 flags, all with lifecycle state `false` — none are enabled for Phase 1 app shell.

| Flag | Phase 1 State |
|------|---------------|
| All 26 flags | Default `false`; no flags enabled |

Source: `specs/feature-flags.md` (Phase 0 baseline)

---

## 3. OpenAPI / Events Contracts

No OpenAPI spec or event contracts exist for Phase 1. The shell exposes a single `/health` endpoint. No API versioning, no request/response schemas beyond the informal JSON contract in brd.md §5.

| Contract | Status |
|----------|--------|
| OpenAPI spec | Not required — single endpoint, informal JSON shape |
| Event schemas | Not applicable — no event-driven architecture in Phase 1 |
| gRPC / proto | Not applicable |

---

## 4. Production-Checklist Prerequisites

The following must be available before the app shell can be started via Docker Compose:

| Prerequisite | Required version | Verification |
|--------------|------------------|---------------|
| Docker daemon | Running locally | `docker info` exits 0 |
| Docker Compose (standalone) | v5.0.2 | `docker-compose --version` |
| Go toolchain | 1.25.6 | `go version` |
| Node.js | v25.3.0 | `node --version` |
| npm | 11.7.0 | `npm --version` |
| pnpm | 10.32.1 | `pnpm --version` |
| Ports 3001 and 5173 | Available (no conflict) | `lsof -i :3001 -i :5173` returns empty |

Source: AGENTS.md (pinned tool versions)

---

## 5. Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **CORS not implemented** | High (ADR-01-004 is mandatory implementation) | Browser fetch to `/health` fails; frontend cannot confirm connectivity | Developer must implement CORS middleware in `backend/main.go` per ADR-01-004 |
| **Version string missing** | Low | `version` field falls back to empty string or "dev" | `-ldflags` injection must be present in `backend/Dockerfile` |
| **Port 3001 conflict** | Medium | Backend fails to bind; Docker Compose exits with error | Resolve port conflict or use wrapper script |
| **Frontend build timeout** | Low | `pnpm build` exceeds 60s NFR-01-002 | Increase timeout or reduce build scope |
| **Graceful shutdown absent** | Low | Docker restart causes immediate TCP termination without FIN; frontend may hold stale connections | Acceptable for Phase 1; Phase 2 implements `e.Close()` on SIGTERM |
| **`depends_on` only waits for container start** | Medium | Frontend starts before backend port is bound; brief error state | Acceptable for Phase 1; Phase 2 implements `service_healthy` |

---

## 6. Rollback Notes

| Scenario | Rollback action |
|----------|-----------------|
| Backend fails to start | `docker-compose down` — no persistent state exists |
| Frontend fails to build | `docker-compose down && docker-compose build --no-cache frontend` |
| CORS misconfiguration | Revert `backend/main.go` CORS middleware changes; restart container |
| Version injection broken | Rebuild backend with correct `-ldflags`; no config files to restore |
| Docker Compose cluster fails | `docker-compose down -v` removes all containers and volumes; start fresh |

Phase 1 has no persistent state. Rollback consists of stopping containers and restarting from a clean build.

---

## 7. Implementation Handoff Constraints

The following constraints must be respected during Phase 1 implementation:

| Constraint | Source | Requirement |
|------------|--------|-------------|
| **CORS origin must be exactly `http://localhost:5173`** | ADR-01-004 | Do not use wildcard `*`; do not vary from this origin |
| **Echo version must remain `v4.15.2`** | ADR-01-003 / AGENTS.md | No `go get -u` without a new ADR |
| **Backend port must be `:3001`** | ADR-01-001 | Do not change without ADR |
| **`depends_on` must NOT include `condition: service_healthy`** | ADR-01-002 | Phase 1 uses container-start ordering only |
| **No authentication or database connections** | brd.md §1 (out of scope) | Phase 1 shell is unauthenticated; no DB wiring |
| **No business logic in `/health` response** | brd.md §5 | Endpoint always returns 200 `"ok"` — no conditional logic |
| **No Phase 2 features in Phase 1 scope** | brd.md §7 | 6 items deferred explicitly — do not anticipate them |

---

## 8. Verification Checklist

Before marking Phase 1 complete, verify all of the following:

- [ ] `curl http://localhost:3001/health` returns JSON with `status: "ok"`
- [ ] Preflight `OPTIONS /health` returns CORS header `Access-Control-Allow-Origin: http://localhost:5173`
- [ ] Frontend loads in browser at `http://localhost:5173` and displays health response fields
- [ ] `docker-compose up -d && docker-compose ps` shows both containers in running state
- [ ] Backend steady-state P95 latency under 100ms (load test)
- [ ] Backend container memory under 256MB at steady state (`docker stats`)
- [ ] Frontend container memory under 512MB at steady state (`docker stats`)
- [ ] No implementation files modified outside `backend/` and `frontend/` directories

---

*Implementation readiness gate: t_506eba0b + t_0e13d59d → t_bb9ddc2a*
*This document is part of the BRD-01-app-shell graduation evidence package.*