# BRD-01: Application Shell — Frontend L1 Design

**Project:** agent-orchestrator
**BRD:** BRD-01 (App Shell)
**Stage:** L1
**Owner:** developer

---

## What

The frontend is a SvelteKit single-page application that renders a single landing route (`/`). It displays a status card that fetches and displays the backend health endpoint result.

---

## Why

Phase 1 needs a visible, demoable artifact. The landing page:

- Proves the frontend can reach the backend API
- Provides a placeholder UI structure that Phase 2 (dashboard, BRD-03/BRD-04) will extend
- Serves as the entry point for browser-based Playwright acceptance tests

---

## Key Insight

The landing page is intentionally static and simple. It is a **demo harness**, not a product UI. No navigation, no auth, no state management — just a proof of connectivity and a version display.

---

## Component: frontend/ Structure

```
frontend/
├── src/
│   ├── routes/
│   │   ├── +page.svelte       # Landing page — Phase 1 deliverable
│   │   └── +layout.svelte     # Root layout (minimal, if needed)
│   ├── app.html               # HTML shell
│   └── app.css                # Global styles (minimal)
├── static/
│   └── favicon.png            # (placeholder)
├── package.json               # { "name": "agent-orchestrator-frontend", "version": "0.1.0" }
├── pnpm-lock.yaml             # Lockfile
├── svelte.config.js           # SvelteKit config
├── vite.config.ts             # Vite config
├── tsconfig.json              # TypeScript config
├── Dockerfile                 # Multi-stage build: node build → nginx:alpine
└── .gitkeep
```

---

## Component: Landing Page (+page.svelte)

### What it does

- On mount, fetches `VITE_API_BASE_URL/health`
- Displays status: `"connected"`, `"unreachable"`, or `"loading"`
- Shows backend `version` and `timestamp` if connected
- Shows frontend version (hardcoded `"0.1.0"`) for symmetry

### Svelte implementation

```svelte
<script lang="ts">
  import { onMount } from 'svelte';

  let status: 'loading' | 'connected' | 'unreachable' = 'loading';
  let backendVersion = '';
  let backendTimestamp = '';
  let error = '';

  onMount(async () => {
    try {
      const res = await fetch(`${import.meta.env.VITE_API_BASE_URL}/health`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      backendVersion = data.version;
      backendTimestamp = data.timestamp;
      status = 'connected';
    } catch (e) {
      error = String(e);
      status = 'unreachable';
    }
  });
</script>

<main>
  <h1>Agent Orchestrator</h1>
  <p>Frontend: 0.1.0</p>

  {#if status === 'loading'}
    <p>Checking backend connectivity...</p>
  {:else if status === 'connected'}
    <p>Backend: {backendVersion} @ {backendTimestamp}</p>
  {:else}
    <p>Backend unreachable: {error}</p>
  {/if}
</main>
```

---

## Data Flows

**Happy path:**
```
Browser → GET / → +page.svelte mounts → fetch(/health) → status = 'connected'
```

**Error path:**
```
Browser → GET / → +page.svelte mounts → fetch fails → status = 'unreachable'
```

---

## Error Handling

| Scenario | Behavior |
|----------|----------|
| `VITE_API_BASE_URL` undefined | Vite warns at build time; defaults to `http://localhost:3001` |
| Backend not running | `fetch` throws; status = 'unreachable'; error displayed |
| HTTP error response | `res.ok` is false; thrown as `HTTP {status}`; caught as error |
| Network timeout | `fetch` times out (browser default); caught as error |

---

## Edge Cases

| Scenario | Behavior |
|----------|----------|
| `VITE_API_BASE_URL` points to wrong host | `fetch` fails; 'unreachable' shown; no crash |
| Backend returns non-JSON | `await res.json()` throws; caught as error; 'unreachable' |
| Backend returns `{ status: "error" }` | JSON parses; but status is not 'ok'; could show a degraded state |

---

## Testing Approach

- **Manual:** Load `http://localhost:5173`, observe status card
- **Playwright (Phase 1):** `agent-can-load-landing-page` scenario in `evals/e2e/`
- **Playwright (Phase 1):** `agent-health-check-shows-connected` scenario

---

## Open Questions

1. Should the landing page show any branding/logo?
   - **No.** Phase 1 scope is minimum scaffold. Branding is Phase 2 (BRD-03 dashboard).
2. Should there be a "retry" button when backend is unreachable?
   - **No.** Phase 1 is a demo harness. Retry logic is Phase 2.
3. Should the page auto-refresh the health check?
   - **No.** Phase 1 is static display. Refresh logic is Phase 2.

---

## Dependencies

- SvelteKit sv CLI 0.15.3 (installed globally or via npx)
- pnpm 10.32.1
- `VITE_API_BASE_URL` env var (defaults to `http://localhost:3001`)
- Backend `/health` endpoint must be responding

---

## Non-Functional Requirements

| Requirement | Target |
|-------------|--------|
| Frontend loads in < 2s on localhost | 100% |
| Health status shown within 1s of page load | 100% |
| `pnpm build` exits 0 | 100% |
| Playwright test `agent-can-load-landing-page` passes | 100% |