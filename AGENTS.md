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
Phase 0 — Governance & Scaffolding                            [DONE]
  └─ Artifacts: AGENTS.md, STATUS.md, docs/adr/0001.md, .gitignore, .env.example

Phase 1 — App Shell (BRD-01)                                  [DONE]
  └─ Go/Echo server (backend/) + SvelteKit UI (frontend/) scaffold
  └─ Shell is runnable; see backend/main.go and frontend/

Phase 2 — Core Delivery (BRD-02 through BRD-N)                [IN PROGRESS]
  └─ BRD-02 Platform-Native Orchestration Pipeline — implemented
  └─ BRD-03 Client Portal & Business Project Board — implemented
  └─ BRD-04+ (agent-execution harness, LLM provider, agent memory) — not yet built
```

Phase 0 artifacts remain valid throughout Phase 1 and Phase 2 unless explicitly superseded by a new ADR.

---

## Pinned Tool Versions

These versions were locked for Phase 0. Phase 1/2 updates require a new ADR.

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
| Pi (`pi`) | `0.83.0` | Agent-execution harness DEV primary runtime (`pi -p --mode json`); headless NDJSON lifecycle |
| OpenCode (`opencode`) | `1.18.10` | Harness DEV fallback runtime (`opencode run`); no JSON mode — text-mode adapter |

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

## Architecture baseline

`ARCHITECTURE.md` (repo root) is the **approved reference architecture**: system diagram + components table + approval boundaries. It is the baseline every PR is reviewed against — see it there rather than this section for component detail.

- **Architecture-impacting PRs update the baseline in the same PR.** Any PR that adds, changes, or removes a component, dependency, trust boundary, or external integration MUST include an `ARCHITECTURE.md` update, guided by the `architecture-impact` skill's `review` mode (which proposes the patch). The baseline becomes approved architecture only when merged to `main`.
- **Approval boundaries require explicit captain sign-off before merge**: new services, datastores, external systems/vendors, cross-domain writes, auth-boundary changes, breaking public contracts, and destructive migrations (these mirror the "Approval boundaries" section of `ARCHITECTURE.md`).
- The `architecture-impact` skill enforces this — `init` to establish the baseline, `review` on each PR. Its safety rules are **static reads + git only**; never install dependencies or run builds to produce or check the baseline.

---

## Orchestrator Pipeline Gate Chain

Every BRD moves through this gate sequence before implementation begins. Gates are Kanban task dependency edges (parent → child). A child task does not promote to `ready` until all parent tasks reach `done`. A validator task returning `done` is not approval — its output is findings; a PM gate-review task must explicitly approve before downstream work begins.

```
scaffold                            → pm (Phase 0 project bootstrap)
    ↓
scaffold-review                     → validator (post-scaffold governance gate)
    ↓
BRD drafting / brainstorming         → Shakil + orchestrator (top-level session only)
                                       NO Kanban task to pm/spec-writer/brainstormer for raw BRD
                                       If Shakil opens a separate brainstormer session,
                                       bring output back here before downstream gating
    ↓
systematic-refinement                → architect + refiner (parallel)
    ↓
subagent-driven-development          → architect + refiner (parallel sub-task coordination)
    ↓
curating-artifacts                   → pm
    ↓
eval-contracts + flag/contract parity → qa + ops
                                       (must exist before validate-design and implementation)
    ↓
eval-readiness-gate                  → validator
                                       Verifies all eval files exist on disk,
                                       cover every BRD FR/NFR, match OpenAPI/events contracts.
                                       Catches missing evals before implementation begins.
    ↓
completeness-score                   → validator (GATE 1 — findings only)
    ↓
PM gate review                       → pm (decides: approve / repair / escalate)
    ↓
validate-design                      → validator (GATE 2 — findings only)
    ↓
PM gate review                       → pm (must explicitly approve)
    ↓
graduation evidence package          → pm
                                       Single-folder audit trail in
                                       specs/curated/brd-XX-<slug>/ with:
                                         brd.md — canonical build spec
                                         decision_record.md — PM gate decisions, OQ resolutions
                                         validator-findings.md — consolidated findings + disposition
                                         implementation-readiness.md — eval/flag/contract evidence
    ↓
production-checklist                 → ops (pre-implementation readiness)
    ↓
IMPLEMENTATION                       → backend / frontend-eng (ONLY NOW)
    ↓
backend code review                  → backend-reviewer
                                       MANDATORY after backend implementation, before QA.
                                       Not QA, not optional.
    ↓
QA: test coverage                    → qa (separate child task)
QA: BRD compliance                   → qa (separate child task)
    ↓
repair loop if needed                → original implementer, then backend-reviewer/QA again
    ↓
ops: verify production checklist     → ops
    against shipped implementation      MANDATORY GATE — before flag enablement
    ↓
ops: verify production readiness     → ops
    before flag enablement               MANDATORY GATE
    ↓
FF_ENABLE_*=true in production
```

**No implementation task may be created until all preceding gates pass.**

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

---

## Maintaining this file

Keep this file for knowledge useful to almost every future agent session in this project.
Do not repeat what the codebase already shows; point to the authoritative file or command instead.
Prefer rewriting or pruning existing entries over appending new ones.
When updating this file, preserve this bar for all agents and keep entries concise.
