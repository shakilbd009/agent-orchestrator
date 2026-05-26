# BRD-01 — Edge Case Discovery & L2 Analysis

**BRD:** BRD-01-app-shell  
**Stage:** 05-design-L2  
**Status:** Analyzed

---

## Edge Cases by Category

### Category: Data Boundaries

| Edge Case | What Happens | How We Handle It | Testability |
|-----------|--------------|------------------|-------------|
| **`timestamp` field is empty** (time unavailable) | RFC3339 time would be blank | Go's `time.Now().Format(time.RFC3339)` always produces a value; not an edge case | Manual: curl /health |
| **`version` field is missing or "dev"** (build-time injection failed) | Version shows as "dev" or empty string | Implement version injection via `-ldflags`; fallback to "0.0.0" | Manual: curl /health after mock build |
| **Response body exceeds typical size** | N/A for /health (tiny JSON) | Not a concern for Phase 1 | N/A |
| **Content-Type is not application/json** | Browser interprets raw text | Echo sends `Content-Type: application/json` by default; not an edge case | Manual: check headers |

### Category: State Transitions

| Edge Case | What Happens | How We Handle It | Testability |
|-----------|--------------|------------------|-------------|
| **Backend starts, frontend starts before backend is listening** | Frontend health fetch fails; error shown briefly | `depends_on` ensures backend container first; within-container latency is low | `docker-compose up -d` timing test |
| **Backend restarts while frontend is running** | Frontend shows stale data or error; Vite dev server may reconnect | Manual: refresh page; no auto-retry in Phase 1 | Manual: restart backend container |
| **Frontend rebuilds on file change (HMR)** | Vite may lose API connection briefly | Acceptable for Phase 1; dev server behavior | Manual: save a file, observe |
| **Docker Compose restarts only one service** | `docker-compose restart backend` — backend down, frontend still running | User must manually refresh browser | Manual: docker-compose restart |

### Category: Timing

| Edge Case | What Happens | How We Handle It | Testability |
|-----------|--------------|------------------|-------------|
| **Backend doesn't bind within 10s** | NFR-01-001 would be violated | No automated handling; developer observes logs | Manual: time `docker-compose up -d` |
| **Frontend build takes >60s** | CI/CD or local build appears hung | No timeout spec; developer must interrupt | Manual: observe timing |
| **Browser fetch times out** | Browser fetch timeout for same-origin requests is typically 2–5 minutes; the effective timeout for health check failure is bounded by the browser's configured request timeout, not a protocol-level constraint. For /health calls, an explicit AbortController timeout of 5s is recommended to fail fast rather than wait for browser defaults. | Manual: stop backend mid-request; observe AbortController-based timeout behavior |
| **Slow network between browser and backend in Docker** | Bridge network is fast (<1ms latency); not realistic edge case | Not handled; acceptable for local Docker | Manual: add artificial delay (tc) |

### Category: Integration

| Edge Case | What Happens | How We Handle It | Testability |
|-----------|--------------|------------------|-------------|
| **Backend port is occupied** | Docker Compose fails to bind port; service exits 1 | Unique error message in logs; user resolves port conflict | Manual: occupy port 3001 |
| **Docker daemon not running** | `docker-compose` returns connection error immediately | Clear error message from Docker CLI | Manual: kill Docker daemon |
| **CORs preflight on /health** (browser hits from frontend) | OPTIONS request to backend | Backend has no CORS middleware configured in Phase 1; would fail if browser calls directly | Playwright: check preflight |
| **Backend returns non-JSON** ( coding error generates text) | `JSON.parse` fails in frontend | Error shown in catch block | Manual: corrupt backend temporarily |
| **Frontend VITE_API_BASE_URL misconfigured** | Points to wrong host/port; fetch fails silently | Visible error in browser console | Manual: set wrong URL, observe |

---

## CORS Gap Analysis

### The Problem

BRD-01 does not address CORS. The frontend calls `http://localhost:3001/health` from `http://localhost:5173`. The browserSame-Origin Policy blocks the request unless the backend sends appropriate CORS headers.

### Evidence

- FR-01-002 uses `VITE_API_BASE_URL=http://localhost:3001` to call backend from browser
- Browser enforces same-origin policy; cross-origin fetch requires `Access-Control-Allow-Origin`
- Echo v4 does NOT add CORS headers by default
- If frontend makes a real browser fetch (not server-side), this will fail for localhost → localhost cross-origin

### Options

| Option | Approach | Pros | Cons |
|--------|----------|------|------|
| **A: Add CORS middleware to Echo** | `e.Use(middleware.CORS())` with `AllowOrigins: "*"` | Simple; widely documented | `\0` origin is risky for future; too permissive |
| **B: CORS with specific frontend origin** | `AllowOrigins: "http://localhost:5173"` | Correctly scoped | Hardcoded; port change requires code update |
| **C: Proxy through SvelteKit dev server** | SvelteKit proxies `/api/*` to backend | No CORS needed; consistent with SvelteKit patterns | Adds complexity; proxy config needed |
| **D: No CORS, use server-side fetch** | SvelteKit page fetches on server (SSR) | No CORS concerns at all | Changes architecture (SSR vs SPA) |

### Recommendation

**Option B (CORS with specific frontend origin)** for Phase 1. This is explicit, scoped, and avoids proxy complexity while remaining correct for the localhost dev environment.

**ADR-01-004 should be created to formalize this.**

---

## Error Handling Gap

| Gap | Current State | Needed |
|-----|--------------|--------|
| Backend unreachable at runtime | Frontend fetch times out and shows error | No graceful degradation defined |
| Backend returns 500 or error JSON | Not handled; frontend would try to JSON.parse error body | Should display user-friendly message |
| Version injection failure | Silently falls back to "dev" or empty | Should be explicit; visibility to developer |
| Docker healthcheck failure | Not defined | Phase 2 can implement Docker HEALTHCHECK |

---

## L2 Failure Mode Table

| Component | Failure Mode | Cause | Impact | Detection | Recovery |
|-----------|-------------|-------|--------|-----------|----------|
| Backend | Echo fails to bind port | Port 3001 occupied | Service does not start | `docker-compose logs backend` | Free port 3001 |
| Backend | Go runtime panic | Unhandled goroutine panic, bad dependency | Container exits | `docker-compose ps` shows "Exited" | Restart container; log and fix |
| Backend | /health returns non-200 | Route handler incorrectly implemented | Frontend cannot confirm connectivity | Manual curl test | Redeploy backend |
| Frontend | Vite HMR connection breaks | Docker networking; file change race | Dev server may not reload | Manual observation | Restart frontend container |
| Frontend | fetch fails | Network, CORS, or backend down | Page renders error state | Browser console | Refresh after backend fix |
| Docker daemon | Daemon unavailable | Docker not running | All services fail to start | `docker-compose` connection error | Start Docker |
| Docker network | Services can't reach each other | Bridge network misconfigured | Frontend cannot call backend | Browser shows network error | Restart with clean network |

---

## Security Considerations (Phase 1 — Minimal)

| Concern | Threat | Mitigation | Status |
|---------|--------|-----------|--------|
| /health exposes internal version | Reconnaissance | Phase 1 okay; future phases restrict internal endpoints | OK |
| CORS allows all origins | Data exfiltration | Only if `AllowOrigins: "*"` is used; must pin to localhost | Need ADR-01-004 |
| Port 3001 exposed on host | Local service conflict | Only binds to localhost:3001 inside container | OK |
| No authentication on /health | Unauthorized probe | N/A for Phase 1; expected to be internal only | OK |

---

## Open Items for L3

| Item | Category | Why It Matters |
|------|----------|----------------|
| CORS configuration for /health | Integration | Frontend cannot call backend without it |
| Version injection via `-ldflags` | Build | "dev" fallback is untrackable |
| Frontend error state design | UX | Phase 1 shows raw errors; needs spec |
| Docker healthchecks | Operations | NFR-01-003 has no healthcheck definition |
| Graceful degradation | UX | Browser errors should be user-friendly |

---

*Edge case discovery complete: Stage 05 L2 done*  
*Next: Verify open items are addressed or explicitly deferred*
