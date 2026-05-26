# agent-orchestrator

An AI-native software delivery platform that turns business intent into production software through governed agent orchestration.

---

## Local Development

### Prerequisites

| Tool | Version | Notes |
|---|---|---|
| Docker Compose | v5.0.2 | Standalone only; not `docker compose` |
| Go | 1.25.6 | darwin/arm64 |
| Node | v25.3.0 | |
| pnpm | 10.32.1 | Lockfile: `pnpm-lock.yaml` |

### Setup

```bash
# Clone and enter
git clone <repo-url>
cd agent-orchestrator

# Copy environment defaults
cp .env.example .env.local

# Start infrastructure
docker-compose up -d

# Verify services
docker-compose ps
# Expected: postgres and redis show "healthy"

# Verify Go/Echo backend
cd backend && go build ./... && cd ..
# Expected: no errors (Phase 1 target: server starts)

# Verify SvelteKit frontend
cd frontend && pnpm install && pnpm build && cd ..
# Expected: no errors (Phase 1 target: dev server starts)
```

### Running Services

```bash
# All services (Phase 1 target)
docker-compose up -d

# Backend only
docker-compose up -d backend

# Frontend only (dev mode)
cd frontend && pnpm dev

# All services down
docker-compose down

# Full reset (destroys volumes)
docker-compose down -v
docker-compose up -d
```

### Environment Variables

All service configuration is via environment variables. See `.env.example` for the full list.

**Required for Phase 1 startup:**
- `DATABASE_URL` — PostgreSQL connection string
- `REDIS_URL` — Redis connection string
- `SESSION_SECRET` — Application secret (min 32 chars)
- `LLM_PROVIDER_API_KEY` — LLM provider key

> **FINALIZATION NOTE (Phase 0):** `.env.example` must be finalized in Phase 1 with all required and optional variables documented inline.

---

## UI-First Workflow

The platform is designed for human-in-the-loop operation. The UI is the primary interface for all stakeholders.

```
User (UI) --> Kanban Board --> Agent Dispatch --> Task Completion --> Human Approval --> Deployment
```

1. **Submit intent** via UI (plain-language idea)
2. **Track progress** on Kanban board (task states visible to all project members)
3. **Approve decisions** at quality gates before work advances
4. **Review outputs** (specs, code, tests) before merge
5. **Approve deployment** before release

> **FINALIZATION NOTE (Phase 0):** UI wireframes and component inventory are defined in BRD-01. UI implementation is Phase 1.

---

## Troubleshooting

### Database

```bash
# Connect to postgres
docker-compose exec postgres psql -U agent-orchestrator -d agent_orchestrator

# Check connection
docker-compose exec postgres pg_isready -U agent-orchestrator
```

### Redis

```bash
# Connect to redis
docker-compose exec redis redis-cli

# Test connectivity
docker-compose exec redis redis-cli ping
# Expected: PONG
```

### Common Issues

| Issue | Diagnosis | Resolution |
|---|---|---|
| `docker-compose up` fails | Port already in use | Stop other services on ports 5432/6379 |
| Postgres connection refused | Container not healthy yet | Wait for healthcheck or `docker-compose restart postgres` |
| `pnpm install` fails | Node version mismatch | Ensure `node --version` is `v25.3.0` |
| Go build fails | Missing Go deps | `go mod download` in backend directory |
| Frontend dev server won't start | Port 5173 in use | `lsof -i :5173` and kill the process |

### Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f backend
docker-compose logs -f

# Last 100 lines
docker-compose logs --tail=100
```

> **FINALIZATION NOTE (Phase 0):** Structured logging (JSON) is Phase 1. Plain text docker logs apply until then.

---

## Project Structure

```
agent-orchestrator/
├── AGENTS.md           — Agent operating constitution
├── STATUS.md           — Phase and gate status
├── agent-orchestrator.md  — Business requirements (BRD)
├── docker-compose.yml  — Phase 0 infrastructure
├── backend/            — Go/Echo server (Phase 1)
├── frontend/           — SvelteKit UI (Phase 1)
├── contracts/          — Agent handoff contracts
├── docs/
│   ├── adr/            — Architecture decision records
│   ├── security-baseline.md
│   └── observability.md
├── specs/
│   ├── feature-flags.md
│   ├── BRD-*.md        — Phase 1+ BRDs
│   └── curated/        — Graduated artifacts
├── evals/              — Quality gates and checks
└── references/         — Prior session artifacts
```

---

## Phase Gate Status

| Gate | Status | Notes |
|---|---|---|
| G0 — Foundation | Done | AGENTS.md, STATUS.md, ADR 0001, .gitignore, .env.example |
| G1 — App Shell | Pending | backend/ and frontend/ scaffolds, Phase 1 BRD-01 |
| G2 — Core Delivery | Pending | Full orchestration pipeline, audit trail, dashboard |

See `STATUS.md` for detailed gate tracking.

---

## Getting Help

- **Kanban board:** `hermes kanban` or your configured gateway UI
- **Architecture decisions:** `docs/adr/0001-record-architecture-decisions.md`
- **Business requirements:** `agent-orchestrator.md`
- **Agent rules:** `AGENTS.md`
