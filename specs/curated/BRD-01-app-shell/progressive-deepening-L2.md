# BRD-01 — Progressive Deepening L2

**BRD:** BRD-01-app-shell  
**Stage:** 05-design-L2 (interactions + edge cases)  
**Status:** Analyzed

---

## How: Backend (Go/Echo)

### Mechanism

1. `main.go` initializes Echo router
2. `e.Start(":3001")` blocks, binding to port 3001
3. `GET /health` handler returns struct marshaled to JSON
4. CORS middleware (see ADR-01-004) handles cross-origin preflight

### Build-Time Version Injection

The `version` field is injected at build time via `-ldflags`:

```bash
go build -ldflags '-X main.version=0.1.0' -o backend ./backend
```

Go module path: `github.com/agent-orchestrator/backend` (or per-user equivalent during local setup).

### Interactions

| Interaction | Partner | Protocol | Notes |
|-------------|---------|----------|-------|
| `/health` responds to browser GET | SvelteKit (browser) | HTTP + CORS | Only public endpoint |
| Backend binds to `:3001` | Docker / host | TCP | Port fixed per ADR-01-001 |
| Version information | Build system | Build-time flag | Injected at compile |
| Graceful shutdown | Docker SIGTERM | Signal | `e.Close()` not called in Phase 1; no graceful shutdown yet. Without graceful shutdown, Docker Compose restart backend causes immediate TCP termination (no FIN), potentially leaving frontend with stale connections. This is a Phase 2 risk — not a Phase 1 blocker. |

### Edge Cases at L2

| Edge Case | Handling |
|-----------|----------|
| Port 3001 already in use | Echo.Start() returns error; process exits 1 |
| Go module unavailable | `go mod tidy` fails; build fails; error visible |
| CORS preflight (OPTIONS) to /health | Handled by middleware; returns 204 No Content |
| Request to unknown route | Echo 404; no body |
| Backend panic | goroutine exits; container exits 1 |

### Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Port conflict | Medium | Service fails to start | ADR-01-001 documents resolution |
| CORS misconfiguration | High (if ADR-01-004 not implemented) | /health unreachable from browser | ADR-01-004 is mandatory |
| Version string missing | Low | Falls back to "dev" | `-ldflags` must be in Dockerfile |

---

## How: Frontend (SvelteKit)

### Mechanism

1. `src/routes/+page.svelte` mounts
2. `onMount` lifecycle fires `fetch(VITE_API_BASE_URL + '/health')`
3. Response rendered in component template
4. Error caught in try/catch, displayed in error state

### Interactions

| Interaction | Partner | Protocol | Notes |
|-------------|---------|----------|-------|
| Fetch to backend | Go/Echo backend | HTTP + CORS | Browser enforces same-origin |
| Environment variable | `docker-compose.yml` | Env injection | `VITE_API_BASE_URL` set in container |
| HMR (file changes) | Vite dev server | WebSocket | Container filesystem watch via volume mount |
| Page navigation | Browser | Client-side | Phase 1 single route only |

### Frontend Response Handling

```svelte
<script>
  let data = null;
  let error = null;

  onMount(async () => {
    try {
      const res = await fetch(`${import.meta.env.VITE_API_BASE_URL}/health`);
      data = await res.json();
    } catch (e) {
      error = e.message;
    }
  });
</script>

{#if error}
  <p>Error: {error}</p>
{:else if data}
  <pre>{JSON.stringify(data, null, 2)}</pre>
{/if}
```

### Edge Cases at L2

| Edge Case | Handling |
|-----------|----------|
| Backend unreachable | `catch` sets error message; renders error state |
| `VITE_API_BASE_URL` undefined | SvelteKit may warn in console; fetch fails |
| Non-JSON response | `res.json()` throws; caught in catch |
| Backend returns 200 but malformed JSON | `JSON.parse` fails in frontend |

### Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| No error state spec defined | High | Developer sees raw text error | Explicit error UX spec needed (L3) |
| `VITE_API_BASE_URL` port mismatch | Medium | Fetch connects to wrong port | Verify ADR-01-001 port consistency |
| Vite dev server CORS | N/A | Vite dev serves same-origin | Only CORS matters for prod-style direct calls |

---

## How: Docker Compose

### Mechanism

1. `docker-compose up -d` reads `docker-compose.yml`
2. Builds backend image from `backend/Dockerfile`
3. Builds frontend image from `frontend/Dockerfile`
4. Starts `backend` first (no `depends_on` wait — just container start order)
5. Starts `frontend` after `backend` container is created

### Key Docker Patterns

| Pattern | Implementation | Notes |
|---------|---------------|-------|
| Build context | `./backend` and `./frontend` | Each has its own Dockerfile |
| Port forwarding | `ports: - "3001:3001"` and `- "5173:5173"` | Host:container mapping |
| Environment | `environment:` block | `VITE_API_BASE_URL` injected |
| Dependency | `depends_on: [backend]` | Controls container start order |
| Network | Default bridge | Services communicate via container names |

### Edge Cases at L2

| Edge Case | Handling |
|-----------|----------|
| Docker daemon not running | `docker-compose` exits with connection error |
| Build fails (bad Dockerfile) | Exit non-zero; build logs visible |
| Backend image build fails | `depends_on` does not help; frontend builds fine |
| Port forwarding failure (host port taken) | Docker logs port binding error; containers may still run |
| Volume mount path doesn't exist | Build context empty; build fails |

### Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| `depends_on` only waits for container start, not port binding | Medium | Frontend starts before backend is listening | Reasonable for Phase 1; manual refresh |
| No healthcheck for backend | Medium | Docker can't detect backend failure | ADR-01-002 defers to Phase 2 |
| Hot reload not working (no volume mount) | High | File changes require rebuild | Missing from Phase 1 docker-compose |

---

## Interactions Summary

```
Browser → Frontend (Vite dev server) → Backend (Echo /health)
   ↑           ↑                             ↑
   └───────────┴─────────────────────────────┘
              CORS (ADR-01-004)
              
Docker host ← Frontend :5173 (port forward)
           ← Backend :3001 (port forward)
```

---

## Resolved vs Deferred Items

| Item | Resolved | Deferred | ADR |
|------|----------|----------|-----|
| Port 3001 fixed | ✓ | | ADR-01-001 |
| Docker startup order | ✓ | | ADR-01-002 |
| Echo version policy | ✓ | | ADR-01-003 |
| CORS for /health | ✓ | | ADR-01-004 |
| Version injection via build flags | ✓ | | Here (not yet in ADR) |
| Frontend error state spec | | Deferred | Gap identified |
| Docker healthchecks | | Deferred | Phase 2 |
| Hot reload volume mounts | | Deferred | Phase 2 |
| Graceful degradation in frontend | | Deferred | Gap identified — Phase 2 |
| Backend graceful shutdown | | Deferred | Phase 2 risk; TCP termination without FIN on restart — see row above |

---

*L2 complete — progressive deepening, edge cases, and interactions documented*  
*Ready for Stage 06 verification or graduation review*
