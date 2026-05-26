# ADR-0001: Record Architecture Decisions

**Status:** Accepted  
**Date:** 2026-05-25  
**Deciders:** architect (Phase 0 governance batch)  
**Supersedes:** None  
**Superseded by:** None  

---

## Context

The Agent Orchestrator Platform is a new project starting from zero. Without a formal decision record, the project risks:

- Inconsistent tool choices as new agents join
- Drift between implementation and stated governance
- Reinvention of resolved questions in later phases
- No audit trail for why architectural choices were made

We need a lightweight, durable mechanism to record decisions that outlast any single agent session and survive team turnover.

---

## Decision Drivers

1. The project spans multiple agent profiles (architect, developer, reviewer, pm, etc.)
2. Phase 0 deliberately produces no source code — governance must be finalized first
3. Multiple human and AI actors will make decisions over the project's lifetime
4. The BRD defines a complex multi-phase delivery with quality gates
5. Tool versions must be pinned to prevent environmental drift

---

## Options Considered

### Option A: Per-Decision ADRs in `docs/adr/` (Chosen)

Each significant decision gets a numbered document (`0001`, `0002`, etc.) in `docs/adr/`. Each ADR records:

- Status (Proposed / Accepted / Deprecated / Superseded)
- Context and decision drivers
- Options considered
- Decision outcome and rationale
- Consequences (positive and negative)

**Pros:**
- Durable, human-readable, grep-able
- Each decision stands alone — no giant document to maintain
- Clear supersession chain via `Superseded by:` field
- Easy to audit why a choice was made six months later

**Cons:**
- Overhead for trivial decisions (but trivial decisions don't need ADRs)
- Risk of ADRs going stale if not kept current

### Option B: Single `ARCHITECTURE.md` with a decision section

All decisions in one file, newest first.

**Pros:**
- One file to rule them all

**Cons:**
- Grows unbounded; becomes hard to navigate
- No clear closure for decisions — everything looks "in progress"
- No supersession mechanism
- Hard to reference a specific decision without copying text

### Option C: Decision log only in `STATUS.md`

Decisions as a table in STATUS.md.

**Pros:**
- Convenient, co-located with project status

**Cons:**
- STATUS.md changes frequently (phase updates, blockers, risks)
- Version control noise when updating status
- No room for nuance in decision rationale
- No Options Considered section — just the outcome

### Option D: External decision tracker (Linear, Notion, etc.)

**Pros:**
- Already used for task tracking

**Cons:**
- Decisions detach from the codebase — harder to audit
- Requires tool access; not self-contained
- Fragile if the external tool changes schema or access levels

---

## Decision

**Option A — Per-decision ADRs in `docs/adr/` is adopted.**

The project root will contain `docs/adr/` with numbered markdown files. The first decision is this document (ADR-0001), which establishes the governance stance and project conventions.

### What this ADR records as settled:

| Topic | Decision |
|-------|----------|
| Decision recording format | Per-decision markdown files in `docs/adr/` |
| Agent operating constitution | `AGENTS.md` — single file, no separate `CLAUDE.md` |
| Workflow model | PR-only; no direct pushes to `main` |
| Kanban vs PR distinction | Kanban = internal handoff signal; PR = authoritative delivery |
| Layer A / Layer B separation | Orchestrator vs specialist roles; Layer A owns gates |
| Phase 0 scope | Governance only; no source code |
| Tool version pinning | Pinned in `AGENTS.md`; updated via new ADR |
| Docker command | Use `docker-compose` (standalone v5.0.2) not `docker compose` |
| Package manager | pnpm 10.32.1 |
| Backend framework | Go 1.25.6 + Echo v4.15.2 |
| Frontend framework | SvelteKit (sv CLI 0.15.3, minimal template, TypeScript) |
| Testing | Playwright 1.60.0 via npx |
| CI runner | `ubuntu-latest` default |

---

## Consequences

### Positive

- New agents can read the ADR directory and understand every significant choice
- Decisions are versioned alongside code — if we revert a commit, the ADR state at that point is coherent
- Supersession chain prevents zombie decisions from being relitigated
- `AGENTS.md` pinned versions eliminate "it worked on my machine" class of failures

### Negative

- ADR creation overhead for small decisions — mitigated by only creating ADRs for non-obvious choices
- ADRs can become stale — mitigated by `Superseded by:` field and STATUS.md review cadence

### Neutral

- Phase 0 governance is heavier than pure implementation would be — this is intentional
- Some decisions will be wrong and require supersession — this is expected and acceptable

---

## Process for Future ADRs

1. Any agent or human may propose a decision by creating `docs/adr/XXXX-name.md` with `Status: Proposed`
2. The `architect` profile reviews and promotes to `Accepted`
3. A decision is `Accepted` when it is merged to `main` via PR
4. A decision is `Deprecated` when a subsequent ADR explicitly supersedes it
5. ADRs are never deleted — deprecated decisions remain for historical audit

---

## Review Cadence

`architect` reviews all ADRs at every Phase transition. `STATUS.md` decision log is updated to reflect current ADRs.

---

## References

- `AGENTS.md` — Agent operating rules and version pins
- `STATUS.md` — Decision log summary and project status
- `agent-orchestrator.md` — Business Requirement Document (BRD)
- `references/agent-orchestrator-phase0-session.md` — Verified tool versions
