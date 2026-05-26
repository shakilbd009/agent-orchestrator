# BRD-01: Application Shell — Trade-Off Analysis

**Project:** agent-orchestrator
**BRD:** BRD-01 (App Shell)

---

## Decision 1: Frontend State Management — To Use Reactivity or Vanilla JS?

### Context

The landing page needs to call the backend `/health` endpoint and display the result. The question is how to manage this client-side state.

### Options Considered

#### Option A: Svelte Native Reactivity (chosen)
Use Svelte's native reactive declarations (`$:`) and `fetch` in the component's `onMount`.

**Pros:**
- Zero additional dependencies
- Svelte's reactive model is a natural fit for this use case
- `onMount` ensures no SSR issues with `fetch`

**Cons:**
- Mixes data-fetching logic with presentation
- No loading/error state UI out of the box

**Complexity:** Low
**Risk:** Low

---

#### Option B: SvelteKit `load` Function + Page Data
Use `+page.ts` to fetch data server-side and pass to the page component.

**Pros:**
- Data fetched before page renders (no loading flicker)
- Clean separation between data and presentation

**Cons:**
- Overkill for a single landing page that changes on every reload
- Adds file (`+page.ts`) for trivial fetching

**Complexity:** Medium
**Risk:** Low

---

#### Option C: Svelte Store + API Layer
Create a dedicated store and API module for the health check.

**Pros:**
- Reusable pattern for future API calls
- Easy to add loading/error states

**Cons:**
- Brings in store boilerplate for one endpoint
- Adds a layer of indirection for minimal benefit in Phase 1

**Complexity:** Medium
**Risk:** Low

---

### Decision

**Option A — Svelte native reactivity** for Phase 1.

### Trade-offs Accepted

Trading some structural rigor for simplicity. The landing page is intentionally minimal; the store pattern can be introduced in Phase 2 when more complex UI interactions are needed.

---

## Decision 2: Backend API Path — /health or /api/health?

### Context

The backend exposes a health endpoint. The path determines whether it's a standalone diagnostic or a versioned API endpoint.

### Options Considered

#### Option A: `GET /health` (chosen)
Health check at root path, not under `/api/` prefix.

**Pros:**
- Standard practice for container health checks (Kubernetes, Docker)
- Simpler curl command for manual testing
- No routing prefix logic needed

**Cons:**
- Path is not versioned alongside other API endpoints
- Could conflict with a future `/health` route for monitoring

**Complexity:** Low
**Risk:** Low

---

#### Option B: `GET /api/v1/health`
Versioned API path for the health endpoint.

**Pros:**
- Aligns with OpenAPI `servers: /v1` prefix
- Future monitoring tools can route differently

**Cons:**
- More verbose for manual testing
- BRD-01 has no other v1 endpoints yet — versioning is premature

**Complexity:** Low
**Risk:** Low

---

### Decision

**Option A — `GET /health`** for Phase 1.

### Trade-offs Accepted

Health endpoint is a special case (not a product API). Versioning is a Phase 2 concern when other API endpoints exist.

---

## Decision 3: Frontend-to-Backend Communication — Direct URL vs SvelteKit Proxy

### Context

The frontend needs to call the backend API. Options include a hardcoded direct URL or a SvelteKit proxy layer.

### Options Considered

#### Option A: Direct `VITE_API_BASE_URL` (chosen)
Frontend uses `import.meta.env.VITE_API_BASE_URL` to call backend at a direct URL.

**Pros:**
- Simplest possible setup for Phase 1
- Works identically in Docker and local dev
- No SvelteKit server-side proxy configuration needed

**Cons:**
- CORS must be configured on the backend
- URL must be updated if backend host changes
- Hardcodes a specific backend origin in the browser

**Complexity:** Low
**Risk:** Low

---

#### Option B: SvelteKit Server Proxy
Use SvelteKit's `handle` hook or `proxy` config to forward `/api/*` to the backend.

**Pros:**
- Browser only talks to same-origin `/api` — no CORS issues
- Backend URL abstracted from client

**Cons:**
- More complex setup for a Phase 1 that doesn't need it
- Proxy configuration is non-obvious

**Complexity:** Medium
**Risk:** Low

---

### Decision

**Option A — Direct `VITE_API_BASE_URL`** for Phase 1.

### Trade-offs Accepted

CORS is enabled in Phase 1 (no sensitive credentials yet). SvelteKit proxy is introduced in Phase 2 when the dashboard needs authenticated API calls.

---

## Decision 4: Go Module Name

### Context

Go modules require a module path. The path should be meaningful and stable.

### Options Considered

#### Option A: `agent-orchestrator-backend` (chosen)
Matches the directory name and the project's general naming.

**Pros:**
- Clear association with backend service
- Doesn't conflict with `agent-orchestrator` (the CLI/monorepo root if it exists)

**Cons:**
- Long module name in import statements

**Complexity:** Low
**Risk:** Low

---

#### Option B: `agent-orchestrator`
Use the project root module name for the backend.

**Pros:**
- Shortest import paths

**Cons:**
- Risk of confusion with the project root module
- Circular dependency risk if root tries to import backend

**Complexity:** Low
**Risk:** Medium

---

### Decision

**Option A — `agent-orchestrator-backend`**.

### Trade-offs Accepted

Longer import paths are a minor cost; clarity and no circular dependency risk are worth it.

---

## Decision 5: Docker Build Strategy — Multi-stage vs Single-stage

### Context

How to build the backend Docker image: single `FROM golang` stage or multi-stage with a minimal runtime image.

### Options Considered

#### Option A: Multi-stage (FROM golang → FROM alpine) (chosen)
Build binary in Go container, copy to minimal alpine image.

**Pros:**
- Final image is small (~20MB vs ~800MB)
- No Go toolchain in production image
- Standard production pattern

**Cons:**
- Requires Docker buildkit for efficiency (default in modern Docker)
- Slightly more complex Dockerfile

**Complexity:** Medium
**Risk:** Low

---

#### Option B: Single-stage (FROM golang)
Build and run in same container.

**Pros:**
- Simpler Dockerfile
- Works on all Docker versions

**Cons:**
- Large image size
- Go toolchain exposed in production

**Complexity:** Low
**Risk:** Low

---

### Decision

**Option A — Multi-stage build** for both backend and frontend.

### Trade-offs Accepted

Multi-stage builds are standard practice and expected by the project's DevOps conventions (BRD-12, BRD-13). Phase 0 already established this pattern intent in the feature flag registry.