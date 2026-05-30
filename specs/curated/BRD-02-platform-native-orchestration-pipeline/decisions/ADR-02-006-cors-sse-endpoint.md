# ADR-02-006: CORS Policy for SSE Endpoint

**ADR:** ADR-02-006  
**Subject:** Cross-Origin Resource Sharing headers for `GET /projects/{projectId}/events/stream`  
**Profile:** architect  
**Date:** 2026-05-28  
**Status:** Accepted

---

## Problem Statement

FR-02-023 defines the SSE endpoint: `GET /projects/{projectId}/events/stream`. The endpoint is consumed by "first-party dashboard clients" which are browser-based (SvelteKit per BRD-03). Browser-based clients require proper CORS headers when fetching cross-origin resources.

BRD-02 does not specify CORS configuration for this endpoint. This is a gap: in development, the dashboard may run on `localhost:5173` while the backend runs on `localhost:8080` — different origins. Without CORS headers, the browser blocks the SSE connection.

Furthermore, Go/Echo middleware must handle `OPTIONS` preflight requests and validate `Origin` against an allowed list to prevent unauthorized SSE subscription.

---

## Options Considered

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| A | Allow all origins (`Access-Control-Allow-Origin: *`) | Zero config; works for any consumer | Overbroad; any website can subscribe to SSE stream |
| B | `same-origin only` (no CORS headers) | Simplest; no CORS config | Dashboard must be on same origin; breaks dev setups with separate ports |
| C | Per-project configurable allowlist for allowed origins | Correct security; project-scoped model fits naturally | Requires project-level origin configuration |

---

## Decision

**Option C** is selected, with a permissive default for development.

**Specific rules:**
1. SSE endpoint responds to `OPTIONS` preflight requests with `200` and the headers below.
2. `Access-Control-Allow-Origin` value is taken from the `Origin` request header, validated against the project's allowed-origin list.
3. Allowed-origin list per project defaults to `["http://localhost", "http://127.0.0.1"]` (supports common dev setups with separate ports).
4. Project administrators (human role) can add explicit origins to the project config via the project configuration API. Added origins must use `https` in production.
5. No wildcard `Access-Control-Allow-Origin: *` in any non-development configuration. If no origin matches, the SSE endpoint returns `403 Forbidden` with an error body.
6. `Access-Control-Allow-Methods` for SSE: `GET, OPTIONS`.
7. `Access-Control-Allow-Headers`: `Last-Event-ID, Origin, Cache-Control, X-Requested-With`.
8. `Access-Control-Max-Age`: `86400` (24 hours) — reduces preflight overhead for dashboard clients.

**Implementation note:** The origin validation is a simple string prefix match for `http://localhost` and `http://127.0.0.1`. For other origins, exact string match against the configured allowlist. No regex or dynamic origin resolution (security boundary simplicity).

---

## Consequences

**Positive:**
- CORS is properly configured for dashboard clients (both dev and prod)
- Project-scoped origin allowlist aligns with BRD-02's project-scoped model
- Security boundary preserved (no cross-origin unauthorized SSE subscription)
- OPTIONS preflight cached 24h reduces latency overhead

**Negative:**
- Project config must store origin allowlist
- First deployment requires origin configuration in addition to database setup

**Neutral:**
- `/projects/{projectId}/events/stream` is the only SSE endpoint; CORS scope is narrow
- Go/Echo middleware for CORS should be a reusable middleware function
