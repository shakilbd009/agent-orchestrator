# STATUS.md — Agent Orchestrator Platform

**Project:** agent-orchestrator  
**Last Updated:** 2026-08-06 — Phase 2 (BRD-02/03) implemented; SSE live-update client repaired; agent-execution harness adapter (Phase C) implemented behind `agent-harness` flag  
**Governance Phase:** 2 — Core Delivery (Phase 0 + Phase 1 complete; BRD-02/03 shipped)  

---

## Phase Status

| Phase | Name | Status | Notes |
|-------|------|--------|-------|
| 0 | Foundation & Governance | Done | Artifacts merged; ADR-0001 in place |
| 1 | App Shell (BRD-01) | Done | Go/Echo backend + SvelteKit frontend scaffold runnable |
| 2 | Core Delivery (BRD-02/03) | Implemented | BRD-02 orchestration pipeline + BRD-03 client portal shipped. SSE live-update client repaired (was 404 + wrong event name). Agent-execution harness (BRD-04+), LLM provider (BRD-05), agent memory (BRD-06) still unbuilt |

---

## Decision Log

| Decision | Rationale | Trade-off | Mitigation |
|---|---|---|---|
| **D-0001: Phase 0 produced no source code (completed)** | Governance artifacts were finalized before implementation. Phase 0 is now done; `backend/` and `frontend/` contain real Phase 1/2 code (~11.7k LOC Go, ~9k LOC frontend). | Slowed initial delivery until Phase 1. | Superseded in practice by Phase 1/2 delivery; no longer a binding constraint. |
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
| **D-0014: Agent-execution harness is a runtime-agnostic adapter; DEV backend = Pi/OpenCode only; production provider-SDK-native backend deferred to Phase F** | Captain decision (Phase C): turn role-label agents into real executing processes that stream live activity. Pi (`pi -p --mode json`, verified NDJSON lifecycle) is primary; OpenCode (`opencode run`, text-mode fallback) is secondary. Never Claude/Codex. | OpenCode lacks a documented JSON mode → its fallback infers completion from non-empty stdout text. | Interface (`Harness.Spawn`) stays backend-agnostic so Phase F implements it unchanged. Gated by `agent-harness` flag; emits `agent.activated`/`idle`/`blocked` via the existing `EventService` and drives `CompleteTask`/`BlockTask`. |

---

## Current Work

| Task | Assignee | Status |
|---|---|---|
| Phase 0 scaffolding + governance | `ops` / `architect` | Done |
| Phase 1 app shell (BRD-01) | `developer` | Done |
| Phase 2 BRD-02 orchestration pipeline | `developer` | Done |
| Phase 2 BRD-03 client portal | `developer` | Done |
| SSE live-update client repair (404 path + named-event dispatch) | crewmate | Done (this branch) |
| Agent-execution harness adapter (Phase C) — runtime-agnostic interface + DEV pi/opencode backend, `agent.activated`/`agent.idle`/`agent.blocked` + `CompleteTask`/`BlockTask` wiring, behind `agent-harness` flag | crewmate | Implemented (`fm/orchestrator-harness-adapter`) |
| Agent-execution harness production backend (provider-SDK-native) | TBD | Phase F — not built |

---

## Backlog

| BRD | Title | Status | Artifact |
|---|---|---|---|
| BRD-02 | Platform-Native Orchestration Pipeline | Implemented | `specs/curated/BRD-02-orchestration-pipeline.md` |
| BRD-03 | Client Portal and Business Project Board | Implemented | `specs/ui/brd-03-client-portal.md` |
| BRD-04 | Agent Workstream Dashboard | Not started (placeholder) | — |
| BRD-05 | LLM Provider Integration | Not started (placeholder) | `specs/curated/BRD-05-llm-provider.md` |
| BRD-06 | Agent Memory and State | Not started (placeholder) | `specs/curated/BRD-06-agent-memory.md` |
| — | Agent-execution harness | Implemented (DEV backend) | `backend/internal/service/harness.go`, `harness_dev.go`; Phase F SDK-native backend pending |

---

## Blockers

| Blocker | Owner | Status |
|---|---|---|
| _(none active)_ | — | — |

Open items below are planned work, not blockers: agent-execution harness, BRD-04/05/06.

---

## Risks

| Risk | Severity | Mitigation |
|---|---|---|
| `docker compose` vs `docker-compose` divergence across environments | Medium | Documented; all scripts use `docker-compose` |
| gh auth not configured for GitHub API calls | Medium | `GH_TOKEN` required before CI scripting |
| Tool versions drift if not pinned | Medium | All pins in `AGENTS.md`; no separate `CLAUDE.md` |
| No project kanban board exists yet | Low | `pm` owns board creation in batch 3 |

---

## Handoff Notes

- Phase 0 is complete; `backend/` and `frontend/` now hold real Phase 1/2 implementation (~11.7k LOC Go incl. tests, ~9k LOC frontend, 32 commits on `main`).
- BRD-02 (orchestration pipeline) and BRD-03 (client portal) are implemented behind feature flags (`platform-orchestration`, `client-portal`).
- The SSE live-update client (`frontend/src/lib/api/client.ts`) was repaired: the `/projects/:id/stream` 404 and the `message`-listener-vs-named-event mismatch are fixed; named events now render in the `/orchestration` ticker.
- Agent-execution harness and BRD-04/05/06 remain unbuilt — do not claim them in docs.
