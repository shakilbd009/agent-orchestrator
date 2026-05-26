# BRD-01 — Trade-Off Analysis

**BRD:** BRD-01-app-shell  
**Stage:** 03-trade-offs  
**Status:** Analyzed

---

## Major Decision 1: Backend Port Allocation

### Context

FR-01-001 requires a health endpoint. BRD-01 does not explicitly fix the port number — it only resolves OQ-2 ("What base URL should the frontend use to call the backend API?") to `http://localhost:3001`. The port number itself (3001) is an implicit assumption but not formally recorded as a decision.

### Options

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| **A: Fixed port 3001** | Backend always binds to `:3001` | Simple, predictable, matches resolved OQ-2 | Port conflicts if other services use 3001 |
| **B: Configurable port via env var** | `BACKEND_PORT=3001` defaulting to 3001 | Flexible for dev environments with port conflicts | Adds env var complexity in Phase 1 (minimal scope) |
| **C: Dynamic port allocation** | Backend binds to `:0` and announces port | Zero port conflict risk | Frontend can't configure API URL statically; needs service discovery |

### Verdict

**Option A (Fixed port 3001)** for Phase 1. Simplicity principle favors fixed port for the shell phase. Any port conflict should be documented as an environment issue, not a design flaw. Future phases (BRD-02+) can introduce dynamic port allocation via ADR.

---

## Major Decision 2: Echo Version Stability

### Context

ADR-0001 pins Echo to `v4.15.2`. BRD-01 references Echo as the backend framework but does not call out specific routing patterns, middleware, or API design for future phases. Choosing Echo v4 means we are locked to that major version family.

### Options

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| **A: Stick with Echo v4.x (pinned)** | Never upgrade minor/patch versions without ADR | Maximum stability; version nunca drifts | May miss bug fixes and security patches |
| **B: Allow minor/patch upgrades within v4** | `~> v4.15` in go.mod allows 4.15.2 → 4.15.x | Security patches flow in automatically | Subtle breakage possible on minor updates |
| **C: Allow any version** | `require github.com/labstack/echo/v4` (no version pin) | Latest always available | Breaks reproducibility; goes against rule 12 (Fail Loud) |

### Verdict

**Option A (Strict pin)**. Echo v4 API is stable. Pinned minor+patch gives reproducibility without sacrificing features (v5 would be a separate major version which would require an ADR anyway).

---

## Major Decision 3: Frontend Build Target in Docker

### Context

NFR-01-002 specifies `pnpm build` completes successfully. But `docker-compose up` typically starts a dev server, not a production build. The BRD does not specify whether Docker runs `dev` or `build`.

### Options

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| **A: Dev server in Docker** | `pnpm dev` via `docker-compose up` | Fast iteration, hot reload works | Dev server relies on Vite's dev server, different from prod |
| **B: Production build in Docker** | `pnpm build && pnpm preview` in container | Closest to production behavior | Slower builds in loop, no hot reload |
| **C: Hybrid** | Dev for local, production image for CI | Best match to environments | Two Docker configs to maintain |

### Verdict

**Option A (Dev server)** for Phase 1 shell. The goal is demonstration of connectivity, not production parity. Development server is adequate.

---

## Major Decision 4: Docker Compose Service Startup Order

### Context

FR-01-003 says "Docker Compose starts both services". No ordering constraint is specified. In real scenarios, frontend often depends on backend being healthy before it can successfully call the API.

### Options

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| **A: No ordering constraints, fail-fast** | Services start simultaneously | Simple | Frontend may repeatedly fail while backend initializes |
| **B: `depends_on` with explicit ordering** | `depends_on: [backend]` for frontend | Frontend waits for backend | No health check; just "started" not "ready" |
| **C: `depends_on` + healthcheck** | `condition: service_healthy` for both | True readiness guarantee | More complex; healthcheck needed for backend |

### Verdict

**Option B (`depends_on`)** for Phase 1 shell. Simple and sufficient — doesn't require Echo to implement a health check endpoint beyond the `/health` we already have. Phase 3+ can add `service_healthy` conditions.

---

## Cross-Cutting Trade-Offs

| Decision | Trade-off | Mitigation |
|----------|-----------|------------|
| Frontend calls backend directly | CORS implications in browser | Backend must include CORS headers |
| No database in Phase 1 | Stateless design for shell | No persistence concerns |
| Phase 1 only uses `localhost` | Won't test actual Docker networking | Explicit in BRD scope |
| Docker healthcheck not implemented | NFR-01-003 says 'all containers reach healthy state' but no Docker HEALTHCHECK is defined for backend. Docker cannot detect a degraded backend in Phase 1. | Phase 2 must implement Docker HEALTHCHECK for backend and use `condition: service_healthy` in `depends_on`. ADR-01-002 records this as deferred to Phase 2. | PERF-007 |

---

## Red Flags Checklist

| Flag | Status | Notes |
|------|--------|-------|
| Only 1 approach considered | ❌ FIXED | 2-3 approaches for major decisions |
| Rubber-stamping | ✅ PASS | All options have genuine pros/cons |
| Vague rationale | ✅ PASS | Rationale explicit per decision |
| Hand-waving complexity | ✅ PASS | All consequences listed |
| No validation plan | ✅ PASS | Verification method tied to each AC |

---

*Trade-offs analyzed: Stage 03*  
*Recommendation: Proceed to Stage 04 design L1*
