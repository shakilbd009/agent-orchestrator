# Feature Flag Registry

**Project:** agent-orchestrator  
**Phase:** 0  
**Owner:** architect  

---

## Overview

This registry records every feature flag for the Agent Orchestrator Platform. All flags default to `false`. A flag transitions to `true` only after its associated BRD exits draft status and has been approved for activation. No source code implements a flag-gated feature until the flag is registered here.

---

## Flag Lifecycle

| Stage | Description |
|-------|-------------|
| `false` | Flag is registered but feature is not implemented or is disabled. |
| `alpha` | Feature is implemented but private or unstable. Internal testing only. |
| `beta` | Feature is available to internal users. Known issues may exist. |
| `true` | Feature is generally available. Full support applies. |
| `deprecated` | Feature is supported but flagged for removal. No new work on this flag. |
| `removed` | Flag has been deleted. Feature no longer exists. |

---

## Registry

| Flag | Domain | Default | Current | Introduced | Deprecated | Removed | Notes |
|------|--------|---------|---------|------------|------------|---------|-------|
| `kanban-orchestrator` | Orchestration | `false` | `false` | Phase 0 | — | — | Kanban-first task orchestration pipeline |
| `layer-a-agents` | Orchestration | `false` | `false` | Phase 0 | — | — | Layer A agent profiles (orchestrator, pm, architect) |
| `layer-b-agents` | Orchestration | `false` | `false` | Phase 0 | — | — | Layer B agent profiles (developer, reviewer, qa, devops) |
| `human-gates` | Governance | `false` | `false` | Phase 0 | — | — | Human approval checkpoints in the pipeline |
| `audit-trail` | Governance | `false` | `false` | Phase 0 | — | — | Structured event log for all agent actions |
| `adr-governance` | Governance | `false` | `false` | Phase 0 | — | — | ADR-based decision recording |
| `dashboard` | UI | `false` | `false` | Phase 1 | — | — | Business-facing project status dashboard |
| `agent-dashboard` | UI | `false` | `false` | Phase 1 | — | — | Agent workstream and accountability view |
| `llm-provider` | Backend | `false` | `false` | Phase 0 | — | — | Configurable LLM provider backend |
| `agent-memory` | Backend | `false` | `false` | Phase 0 | — | — | Persistent cross-session agent memory |
| `brd-authoring` | Workflow | `false` | `false` | Phase 0 | — | — | Structured BRD creation workflow |
| `quality-gates` | Quality | `false` | `false` | Phase 1 | — | — | Automated quality gate enforcement |
| `code-review` | Quality | `false` | `false` | Phase 1 | — | — | Agent code review workflow |
| `security-review` | Quality | `false` | `false` | Phase 1 | — | — | Agent security review workflow |
| `qa-automation` | Quality | `false` | `false` | Phase 1 | — | — | Automated QA test generation and execution |
| `deployment-pipeline` | DevOps | `false` | `false` | Phase 1 | — | — | Automated deployment pipeline |
| `docker-scaffold` | DevOps | `false` | `false` | Phase 1 | — | — | Docker-based development environment |
| `playwright-testing` | QA | `false` | `false` | Phase 1 | — | — | Browser-based acceptance testing |
| `scope-control` | Governance | `false` | `false` | Phase 1 | — | — | Structured scope change management |
| `risk-tracking` | Governance | `false` | `false` | Phase 1 | — | — | Active risk identification and mitigation tracking |
| `multi-tenant` | Backend | `false` | `false` | Phase 2 | — | — | Multi-tenant project isolation |
| `collaboration` | Workflow | `false` | `false` | Phase 2 | — | — | Multi-user collaboration and team management |
| `change-history` | Workflow | `false` | `false` | Phase 2 | — | — | Full decision and artifact version history |
| `post-deployment` | Operations | `false` | `false` | Phase 2 | — | — | Post-deployment monitoring and issue triage |
| `notifications` | UX | `false` | `false` | Phase 2 | — | — | User-facing event notifications |
| `client-portal` | UI | `false` | `false` | Phase 1 | — | — | Client-facing multi-project portal, approval actions, comments, risks, milestones, search/filter, SSE, publication validation |
| `dashboard` | UI | `false` | `false` | Phase 1 | — | — | Business-facing project status dashboard (preserve until `client-portal` adopted; deprecation deferred to later ADR) |

---

## Transition Process

1. Flag is registered in this file with `default: false` before any implementation begins.
2. Implementation begins only after the owning BRD exits draft status.
3. Flag transitions are recorded in the `Current` column with the phase of activation.
4. Deprecation requires an ADR that explicitly names the flag and the removal timeline.
5. Removal requires confirmation that no code references the flag.

---

## Guidelines

- No feature flag may appear in source code that is not registered in this file.
- No flag may be hardcoded to `true` in the main branch without an approved BRD.
- Flag names use kebab-case: `feature-name`.
- Each flag belongs to exactly one domain.
- Flags introduced in Phase 0 reflect governance and structural capabilities, not user-facing features.
