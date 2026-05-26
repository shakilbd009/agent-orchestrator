# ADR-01-004: CORS Configuration for Health Endpoint

**Status:** Proposed  
**Date:** 2026-05-25  
**Deciders:** architect (subagent, BRD-01 refinement)  
**Supersedes:** None  
**Superseded by:** None  

---

## Context

BRD-01 FR-01-002 describes the frontend calling the backend at `http://localhost:3001` using `VITE_API_BASE_URL`. When the SvelteKit app runs in the browser (`http://localhost:5173`) and calls `http://localhost:3001/health`, the browser Same-Origin Policy blocks the request unless the backend sends `Access-Control-Allow-Origin` headers.

Echo v4 does not add CORS headers by default. Phase 1 did not address this gap in the original BRD draft. This ADR fills that gap.

---

## Decision Drivers

1. Frontend makes browser-side fetch to `http://localhost:3001/health`
2. Cross-origin request requires CORS headers from backend
3. Production deployments may use a reverse proxy (no CORS needed); Phase 1 dev uses direct browser calls
4. SvelteKit's Vite proxy is available but adds configuration complexity
5. Phase 1 scope is deliberately minimal — but CORS is required for the shell to work

---

## Options Considered

### Option A: Wildcard CORS `AllowOrigins: "*"`

Echo middleware configured with permissive wildcard.

```go
e.Use(middleware.CORS())
```

**Pros:**
- Works for any origin; zero configuration needed
- Widely documented; easiest to implement

**Cons:**
- Risk of accidental data exposure in production
- Violates secure-by-default principle (Rule 2: Simplicity First also means secure)
- Not appropriate even for Phase 1 — we should practice correct defaults

### Option B: Fixed frontend origin `AllowOrigins: "http://localhost:5173"` (Chosen)

CORS middleware scoped to the specific frontend origin.

```go
e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins: []string{"http://localhost:5173"},
    AllowMethods: []string{http.MethodGet, http.MethodOptions},
}))
```

**Pros:**
- Correctly scoped for Phase 1 development environment
- Explicit origins are auditable
- Easy to update port when frontend port changes (via ADR)
- Minimal deviation from defaults

**Cons:**
- Hardcoded port — if frontend port changes, requires code update
- May not cover all development scenarios (multiple frontend instances)

### Option C: Proxy through SvelteKit dev server

Configure Vite to proxy `/api` requests to backend — browser always talks to same origin.

**Pros:**
- No CORS needed at all
- Consistent with SvelteKit production patterns (reverse proxy in prod)

**Cons:**
- Adds Vite proxy configuration — additional complexity in Phase 1
- Frontend and backend must share the same origin in Vite config
- Hides the CORS concern rather than fixing it properly

### Option D: No CORS, server-side fetch in SvelteKit

Make the fetch call from SvelteKit's server-side load function (SSR), not from browser JavaScript.

**Pros:**
- No CORS needed — server-to-server call
- Natural for data fetching in SvelteKit

**Cons:**
- Changes the architecture from SPA to SSR pattern
- Frontend error handling becomes more complex
- Frontend is not "demonstrating connectivity" in the same way

---

## Decision

**Option B — Fixed frontend origin for CORS** is adopted for Phase 1.

Echo will be configured with CORS middleware that allows `http://localhost:5173` specifically. This is sufficient for Phase 1 development and does not require SvelteKit proxy configuration or SSR changes.

---

## Consequences

### Positive
- Browser-based health check works with no additional configuration
- CORS policy is explicit and auditable
- Zero risk of cross-origin data leakage (origins are locked down)

### Negative
- Port is hardcoded — if frontend port changes, requires ADR to update
- Only works for the specific frontend origin; doesn't generalize

### Neutral
- Production deployment will likely use a reverse proxy (nginx, Caddy) which handles origin normalization
- Phase 2 can re-evaluate whether CORS or proxy pattern is preferred
- If other ports are needed for local development, another ADR can expand the allowed list

---

## Implementation Notes

The Echo CORS middleware should be added in `backend/main.go`:

```go
import (
   "github.com/labstack/echo/v4/middleware"
)

e := echo.New()
e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins: []string{"http://localhost:5173"},
    AllowMethods: []string{http.MethodGet, http.MethodOptions},
}))
```

Note: No `AllowHeaders` needed for simple GET (preflight is handled by browser on Accept headers). If POST is added later, `AllowHeaders: []string{"Content-Type"}` would be needed.

---

## References

- BRD-01-app-shell.md (FR-01-002)
- Echo CORS middleware docs: `github.com/labstack/echo/v4/middleware`
- SvelteKit Vite proxy: https://vitejs.dev/config/server-options.html#server-proxy
