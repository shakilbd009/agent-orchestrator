# STATUS.md — Agent Orchestrator Platform

**Project:** agent-orchestrator  
**Last Updated:** Phase 0 batch 2 (governance artifacts)  
**Governance Phase:** 0 — Foundation & Governance  

---

## Phase Status

| Phase | Name | Status | Notes |
|-------|------|--------|-------|
| 0 | Foundation & Governance | In Progress | Artifacts being created |
| 1 | App Shell (BRD-01) | Pending | Blocked on Phase 0 |
| 2 | Core Delivery (BRD-02+) | Pending | Blocked on Phase 1 |

---

## Decision Log

| Decision | Rationale | Trade-off | Mitigation |
|---|---|---|---|
| **D-0001: Phase 0 produces no source code** | Governance artifacts must be finalized before implementation to prevent retrofitting constraints. | Slows initial delivery; no running software until Phase 1. | Artifacts are lightweight; Phase 1 scaffold is fast. |
| **D-0002: Use `docker-compose` (standalone v5.0.2) not `docker compose` (v2 plugin)** | `docker compose` (v2) is not installed on this Mac. `docker-compose` (standalone v1) is available at `/usr/local/bin/docker-compose`. | Scripts and documentation must consistently use the standalone command name. | All compose commands pinned to `docker-compose`; documented in AGENTS.md. |
| **D-0003: pnpm as workspace package manager** | Phase 0 session confirmed pnpm 10.32.1 available. `sv` CLI supports `--install pnpm`. Consistent with project-scaffold convention. | Team members accustomed to npm/yarn must adopt pnpm. | Lockfile pinned; `AGENTS.md` records exact version. |
| **D-0004: Go 1.25.6 + Echo v4.15.2 for backend** | Go 1.25.6 is the newest available and confirmed working with Echo v4. Echo v4 is the current stable major version. | Go 1.25.6 is very new; some ecosystem packages may lag. | Pin in `AGENTS.md`; update via ADR when stability confirmed. |
| **D-0005: SvelteKit with `sv` CLI 0.15.3, minimal template, TypeScript** | `sv` CLI is the official SvelteKit scaffolding tool. Minimal template avoids scope creep in Phase 1. TypeScript is the standard for this project. | `sv` CLI is relatively new (0.15.x); tooling may evolve. | Pin `sv` CLI version; `--add playwright` deferred to Phase 1. |
| **D-0006: AGENTS.md as single agent constitution (no separate CLAUDE.md)** | Single source of truth for agent operating rules, version pins, phase boundaries, and governance. Avoids drift between two documents. | No per-directory overrides unless recorded via ADR. | Sub-project overrides require an ADR exception (e.g., `backend/CLAUDE.md`). |
| **D-0007: PR-only workflow, no direct pushes to main** | All work enters via PR to enable CI gates, code review, and audit trail. | Slightly slower for trivial changes. | Small fixes can use conventional PRs with fast-track review. |
| **D-0008: Kanban task completion does not substitute for PR** | Kanban tracks internal handoffs between agents. PR is the authoritative delivery mechanism to `main`. | Agents must do two things (close task + merge PR). | Dual-track is intentional — separates process accountability (Kanban) from code delivery (PR). |
| **D-0009: ADR-based governance, immutable once merged** | Architectural decisions are recorded and frozen to prevent constant re-litigation. | Early mistakes can become expensive to supersede. | New ADRs can explicitly supersede old ones; process is lightweight. |
| **D-0010: Layer A (orchestrator) vs Layer B (specialist) separation** | Clear role separation prevents agents from closing their own quality gates or initiating work outside their contract. | Layer A agents can become bottlenecks. | Layer A must checkpoint frequently; human can authorize Layer B autonomy per gate. |
| **D-0011: No `version:` key in Docker Compose files** | Docker Compose warns or ignores `version:` in modern versions. | Removes a field that some teams use for documentation. | Version contract lives in `AGENTS.md` and `.env.example` instead. |
| **D-0012: Browser-based acceptance testing with Playwright 1.60.0** | Phase 0 session confirmed Playwright available via npx. `sv add playwright` is the SvelteKit integration path. | Playwright test execution requires dev server running separately (not via `webServer` in config). | Dev server started in background; pattern documented in AGENTS.md. |
| **D-0013: `ubuntu-latest` as default GitHub Actions runner** | Default for `setup-node` and most `gh` actions. No Windows or macOS runners needed in Phase 0. | No coverage of platform-specific behavior in CI. | Platform-specific tests can opt into matrix builds via ADR. |

---

## Current Work

| Task | Assignee | Status |
|---|---|---|
| Phase 0 batch 1: scaffold + env | `ops` | Done |
| Phase 0 batch 2: governance artifacts | `architect` | In Progress |
| Phase 0 batch 3: kanban board setup | `pm` | Pending |

---

## Blockers

| Blocker | Owner | Status |
|---|---|---|
| Phase 0 governance artifacts not finalized | `architect` | In Progress |
| Kanban board not yet created for this project | `pm` | Pending |

---

## Risks (Phase 0)

| Risk | Severity | Mitigation |
|---|---|---|
| `docker compose` vs `docker-compose` divergence across environments | Medium | Documented; all scripts use `docker-compose` |
| gh auth not configured for GitHub API calls | Medium | `GH_TOKEN` required before CI scripting |
| Tool versions drift if not pinned | Medium | All pins in `AGENTS.md`; no separate `CLAUDE.md` |
| No project kanban board exists yet | Low | `pm` owns board creation in batch 3 |

---

## Handoff Notes

- Phase 0 batch 1 created `backend/.gitkeep` and `frontend/.gitkeep` — both confirmed empty.
- `references/agent-orchestrator-phase0-session.md` contains verified tool versions from the ops session.
- All Phase 0 artifacts should be committed to `main` before Phase 1 begins.
