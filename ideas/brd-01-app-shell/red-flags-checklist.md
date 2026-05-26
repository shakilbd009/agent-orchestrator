# BRD-01: Application Shell — Red Flags Checklist

**Project:** agent-orchestrator
**BRD:** BRD-01 (App Shell)
**Stage:** L1 design review

---

## Design Red Flags Checklist

### Architecture Level
- [x] No single component doing "too much" (god object)
  - **Status:** PASS. Backend is Echo server + health handler. Frontend is single SvelteKit page. Both are intentionally minimal.
- [x] Clear separation of concerns
  - **Status:** PASS. Backend owns HTTP API. Frontend owns UI. Docker Compose owns orchestration.
- [x] No circular dependencies conceptually
  - **Status:** PASS. Frontend → Backend (read-only). Backend → PostgreSQL/Redis (infra). No reverse dependencies.
- [x] Scalability paths identified
  - **Status:** PASS. Horizontal scaling handled by Docker Compose in Phase 1; Kubernetes patterns deferred to Phase 2 (BRD-12).
- [x] Failure modes considered
  - **Status:** PASS. Error scenarios documented for port conflicts, backend unreachable, Docker daemon not running, build failures.

### Decision Quality
- [x] No "we'll figure it out later" on critical choices
  - **Status:** PASS. All 4 key decisions documented in ADR-001 with rationale and alternatives.
- [x] All major decisions have documented rationale
  - **Status:** PASS. Go module name, health endpoint path, frontend-backend communication, Docker build strategy — all have explicit rationale.
- [x] Trade-offs explicitly acknowledged
  - **Status:** PASS. Each trade-off section includes explicit "Trade-offs Accepted" statement.
- [x] Alternatives were actually considered (not rubber-stamped)
  - **Status:** PASS. Each decision lists 2-3 alternatives with pros/cons and explicit rejection rationale.

### Completeness
- [x] No obvious edge cases ignored
  - **Status:** PASS. Edge cases documented for both backend and frontend: port conflicts, network errors, non-JSON responses, missing env vars.
- [x] Error scenarios identified for all major flows
  - **Status:** PASS. Health check happy path and error path documented. Frontend loading/unreachable/connected states documented.
- [x] Integration points specified
  - **Status:** PASS. VITE_API_BASE_URL, Docker depends_on clauses, docker-compose port bindings — all specified.
- [x] Data flows complete (not just happy paths)
  - **Status:** PASS. Mermaid sequence diagrams show both happy path and error path.
- [x] Security implications considered
  - **Status:** PASS. CORS enabled on backend (intentional Phase 1 tradeoff). No auth/credentials in Phase 1 scope.

### Feasibility
- [x] No hand-waving on technical complexity
  - **Status:** PASS. Specific tool versions (Go 1.25.6, Echo v4.15.2, sv CLI 0.15.3), specific commands (multi-stage Dockerfile patterns), specific ports (3001, 5173).
- [x] No "just use AI" without specifics
  - **Status:** PASS. No AI calls in Phase 1 scope.
- [x] External dependencies realistic
  - **Status:** PASS. Dependencies are: Docker, Go toolchain, Node toolchain, pnpm. All confirmed in AGENTS.md with pinned versions.
- [x] Timeline expectations grounded
  - **Status:** PASS. Not timeline-driven; Phase 1 quality gates are structural (eval scripts pass, docker-compose starts).

---

## Critical Questions (L2/L3 depth required before implementation)

1. **CORS configuration:** The backend will have CORS enabled for `http://localhost:5173` in Phase 1. Is this acceptable, or should CORS be restricted to specific origins only?
   - **Status:** OPEN. Currently assumed broad CORS for demo purposes. Harden in Phase 2.

2. **Database/Redis connectivity:** Backend health check only verifies the Echo server is alive — not that database or cache connections are working. BRD-05 (LLM Provider) and BRD-13 (Docker Environment) address this in Phase 2. Should BRD-01 specify a _separate_ readiness check, or is `/health` liveness-only acceptable for Phase 1?
   - **Status:** OPEN. Currently `/health` is liveness-only. Readiness probes (database + cache connectivity check) deferred to Phase 2.

3. **Version injection:** Backend version is hardcoded as `"0.1.0"`. Should this be injected at build time via ldflags (e.g., `-ldflags "-X main.version=$VERSION"`), or is hardcoded acceptable for Phase 1?
   - **Status:** OPEN. Hardcoded for Phase 1; build-time injection via ldflags planned for Phase 2.

---

## Verdict

**The L1 design is ready for implementation.** All critical decisions are documented. Edge cases and error paths are identified. The scope is intentionally minimal — this is a feature-free scaffold.

**Blocking issues:** None.
**Open questions:** 3 (all documented above with resolution ownership).
**Next step:** Author the BRD-01 artifact in `specs/curated/BRD-01-app-shell.md` with full implementation details.