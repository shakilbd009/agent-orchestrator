# AGENTS.md — Agent Orchestrator Platform

**Project:** agent-orchestrator  
**Phase:** 0 (governance scaffolding)  
**Governance scope:** All agent profiles operating in this workspace

---

## Layer A vs Layer B Separation

| | Layer A | Layer B |
|---|---|---|
| **Role** | Orchestrator / Planner | Specialist / Executor |
| **Scope** | Decomposition, routing, pipeline gates, handoff contracts | Implementation, review, validation, delivery |
| **Examples** | `architect`, `pm` | `developer`, `reviewer`, `qa`, `devops` |
| **May initiate** | New tasks, child tasks, dependency links | Only within assigned task |
| **Governance** | Sets quality gates, records ADRs | Must pass Layer A gates before advancing |

Layer A agents own the process contract. Layer B agents own the output quality. No Layer B agent may close a quality gate — that authority rests with Layer A or a human.

---

## Phase Boundaries

```
Phase 0 — Governance & Scaffolding
  └─ Artifacts: AGENTS.md, STATUS.md, docs/adr/0001.md, .gitignore, .env.example
  └─ No source code. No backend/frontend implementation.

Phase 1 — App Shell (BRD-01)
  └─ Empty backend/ and frontend/ directories exist
  └─ Minimal working scaffold: Go/Echo server, SvelteKit UI
  └─ Shell is runnable, not feature-complete

Phase 2 — Core Delivery (BRD-02 through BRD-N)
  └─ Full agent orchestration pipeline
  └─ Quality gates, audit trail, dashboard
  └─ Production-ready output
```

Phase 0 artifacts remain valid throughout Phase 1 and Phase 2 unless explicitly superseded by a new ADR.

---

## Pinned Tool Versions

These versions are locked for the lifetime of Phase 0. Phase 1 may update pins via a new ADR.

| Tool | Version | Notes |
|------|---------|-------|
| Go | `1.25.6` | darwin/arm64 |
| Node | `v25.3.0` | |
| npm | `11.7.0` | |
| pnpm | `10.32.1` | Lockfile: `pnpm-lock.yaml` |
| Docker Compose (standalone) | `v5.0.2` | Use `docker-compose`, NOT `docker compose` |
| Docker client | `29.2.1` | |
| GitHub CLI (`gh`) | `2.86.0` | |
| Make | `3.81` | GNU |
| Bash | `3.2.57` | |
| Zsh | `5.9` | |
| Git (Apple) | `2.50.1` | |
| Homebrew | `5.1.11` | |
| macOS | `26.2` | BuildVersion 25C56 |
| Echo (Go) | `v4.15.2` | Import: `github.com/labstack/echo/v4` |
| SvelteKit `sv` CLI | `0.15.3` | |
| Playwright | `1.60.0` | Fetched via npx; browsers via `npx playwright install` |

---

## Agent Operating Rules

### Rule 1 — Think Before Designing
State the goal, constraints, tradeoffs, failure modes. High-impact decisions need human input before proceeding.

### Rule 2 — Simplicity First
Design the minimum architecture that solves the confirmed problem. No speculative abstractions or future-proofing.

### Rule 3 — Surgical Changes
Touch only the components affected. Do not redesign adjacent systems.

### Rule 4 — Goal-Driven Execution
Define what success looks like: performance targets, scalability bounds, failure tolerance.

### Rule 5 — Use the Model Only for Judgment Calls
Use AI for comparing approaches, generating alternatives. Routing and deterministic transforms = plain code.

### Rule 6 — Manage Context Deliberately
Work in explicit batches (data layer, service layer, API layer, infra). After each: summarize design decisions, risks.

### Rule 7 — Surface Conflicts
If two architectural approaches conflict, document both, recommend one, explain why, flag the tradeoffs.

### Rule 8 — Read Before You Design
Read existing ADRs, architecture docs, team conventions before proposing new patterns.

### Rule 9 — Tests Verify Intent
Architecture must support testability. If a component cannot be tested, the design is incomplete.

### Rule 10 — Checkpoint After Significant Steps
Summarize design decisions, ADRs created, what was validated, what remains.

### Rule 11 — Match Architectural Conventions
Follow existing patterns for naming, layering, component boundaries.

### Rule 12 — Fail Loud
State explicitly what was not validated, what could go wrong, what assumptions were made.

### Rule 13 — Evidence Beats Claims
Every ADR must include: problem statement, options considered, tradeoffs, decision, consequences.

---

## Quick Reference

```bash
# Kanban
hermes kanban show <task_id>       # Read task
hermes kanban create "<title>"      # Create task
hermes kanban complete <task_id>    # Close task (with summary)

# Docker (use standalone on this Mac)
docker-compose up -d
docker-compose down

# SvelteKit init
npx sv create <path> --template minimal --types ts --install pnpm

# Go/Echo
go mod init <module>
go get github.com/labstack/echo/v4@v4.15.2

# Phase 0 rule: no source code in backend/ or frontend/
```

---

## PR-Only Workflow

All agent work enters the project via pull request. No agent pushes directly to a protected branch.

1. Agent creates a feature branch from `main`
2. Agent completes the scoped work
3. Agent opens a PR with a structured description
4. PR must pass CI gates (lint, test, build if applicable)
5. Human or designated approver merges
6. Branch is deleted post-merge

Kanban task completion does not substitute for a PR. A closed kanban task is an internal handoff signal; a merged PR is the only authorized delivery mechanism to `main`.

---

## Orchestrator Pipeline Gate Documentation

The orchestrator pipeline enforces a strict gate sequence. Each gate is a Kanban task dependency edge (parent → child). A child task does not promote to `ready` until all parent tasks reach `done`.

```
[G0] Foundation & Governance
  └─ Establishes AGENTS.md, STATUS.md, ADR 0001, .gitignore, .env.example
  └─ No source code gates exist here

[G1] App Shell Gate
  └─ backend/ and frontend/ scaffolds exist and are runnable
  └─ No feature code, no agent orchestration yet

[G2] Core Delivery Gate
  └─ Full orchestration pipeline is operational
  └─ Quality gates, audit trail, dashboard implemented
  └─ All BRD requirements addressed
```

Gates are not enforced by automation alone — each Phase transition requires a human decision to authorize the next Phase card.

---

## CLAUDE.md — No Separate File Required

This `AGENTS.md` serves as the agent operating constitution for the project. There is no separate `CLAUDE.md`.

If a sub-project or subdirectory needs its own agent guidance (e.g., `backend/CLAUDE.md`), that is created via a new ADR that records the rationale for the exception.

---

## Governance Ownership

| Artifact | Owner | Review Cadence |
|---|---|---|
| `AGENTS.md` | `architect` | On every Phase transition |
| `STATUS.md` | `architect` / `pm` | Every sprint |
| `docs/adr/0001.md` | `architect` | On governance change |
| `docs/adr/*.md` (future) | `architect` | On each decision |

ADR decisions are immutable once merged. Superseding an ADR requires opening a new ADR that explicitly superseded the old one.
