# BRD-01: Application Shell — Architecture Diagram

**Project:** agent-orchestrator
**BRD:** BRD-01 (App Shell)

---

## System Context

```mermaid
flowchart LR
    subgraph Docker["Docker Compose (project root)"]
        subgraph Backend["backend service\nGo/Echo :3001"]
            BE[Echo Server]
            Health[HealthHandler]
            BE --> Health
        end

        subgraph Frontend["frontend service\nSvelteKit :5173"]
            FE[SvelteKit App]
            Page[+page.svelte]
            FE --> Page
        end

        DB[(PostgreSQL\n:5432)]
        Cache[(Redis\n:6379)]

        Frontend -->|"VITE_API_BASE_URL=http://localhost:3001"| Backend
        Backend --> DB
        Backend --> Cache
    end

    User[("User\nBrowser")] -->|"http://localhost:5173"| Frontend
    User -->|"http://localhost:3001/health"| Backend
```

---

## Component Interaction Sequence

```mermaid
sequenceDiagram
    participant User
    participant Frontend as SvelteKit\n:5173
    participant Backend as Go/Echo\n:3001
    participant Docker as docker-compose

    User->>Frontend: GET /
    Frontend-->>User: 200 OK (HTML)

    Note over Frontend: onMount() fires

    User->>Backend: GET /health
    Backend-->>User: 200 { "status": "ok", "version": "0.1.0", "timestamp": "..." }

    Frontend->>Backend: fetch(/health)
    Backend-->>Frontend: JSON
    Frontend->>Frontend: Update status card
```

---

## Directory Structure

```mermaid
graph TD
    Root["agent-orchestrator/\n(project root)"]
    DC["docker-compose.yml"]
    Root --> DC

    Backend["backend/\n(Go/Echo)"]
    Root --> Backend
    Backend --> BE_Main["main.go"]
    Backend --> BE_Mod["go.mod / go.sum"]
    Backend --> BE_Docker["Dockerfile\n(multi-stage)"]

    Frontend["frontend/\n(SvelteKit)"]
    Root --> Frontend
    Frontend --> FE_Page["src/routes/+page.svelte"]
    Frontend --> FE_Cfg["svelte.config.js\nvite.config.ts"]
    Frontend --> FE_Pkg["package.json\npnpm-lock.yaml"]
    Frontend --> FE_Docker["Dockerfile\n(node → nginx)"]

    Evals["evals/"]
    Root --> Evals
    Evals --> E2E["e2e/placeholder.md"]
    Evals --> Arch["architecture/*.sh"]

    Specs["specs/curated/"]
    Root --> Specs
    Specs --> BRD01["BRD-01-app-shell.md"]
```

---

## Infrastructure Integration (Phase 0 → Phase 1)

```mermaid
graph LR
    subgraph Phase0["Phase 0 Services (existing)"]
        DB[(PostgreSQL 17\n:5432)]
        Cache[(Redis 7\n:6379)]
    end

    subgraph Phase1["Phase 1 Services (new)"]
        BE[backend\n:3001]
        FE[frontend\n:5173]
    end

    BE --> DB
    BE --> Cache
    FE --> BE
```

Phase 0 docker-compose.yml defines database and cache services. BRD-01 extends this with backend and frontend services, with `depends_on` clauses ensuring healthy startup order.