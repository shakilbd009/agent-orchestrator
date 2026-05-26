# BRD-02: Core Orchestration Pipeline

**Project:** agent-orchestrator  
**Domain:** Orchestration  
**Phase:** 1  
**Owner:** architect  
**Status:** placeholder  

---

## Goal

Define the kanban-first orchestration pipeline that routes work through Layer A planners and Layer B specialists, with human gate checkpoints and structured handoff contracts.

---

## Problem Statement

Without a defined orchestration pipeline, agents lack a consistent mechanism for decomposing work, routing tasks, and enforcing quality gates. The system needs a concrete pipeline definition before any agent roles can execute reliably.

---

## Scope

### In Scope
- Kanban task creation, promotion, and completion lifecycle
- Layer A decomposition and routing logic
- Layer B task execution and quality gate handoffs
- Human approval checkpoint integration
- Task dependency management

### Out of Scope
- Specific agent profile implementations (developer, reviewer, etc.)
- LLM provider integration
- Persistent storage or database layer

---

## Functional Requirements

### FR-02-001: Task Lifecycle Management
The orchestrator creates, assigns, and promotes tasks through defined states: `todo` → `in_progress` → `done`, with support for `blocked` and `cancelled` states.

### FR-02-002: Layer A Decomposition
Layer A agents decompose high-level goals into actionable child tasks with explicit dependency edges.

### FR-02-003: Layer B Execution
Layer B agents execute assigned tasks and hand off to quality gates, not directly to completion.

### FR-02-004: Human Gate Checkpoint
Human approval is required at defined pipeline gates before work proceeds to the next phase.

---

## Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-02-001 | Task handoff completes within one agent session | No orphaned tasks between handoffs |
| NFR-02-002 | Layer A decisions are auditable | All decompositions logged to audit trail |

---

## Acceptance Criteria

| ID | Criterion | Verification Method |
|----|-----------|---------------------|
| AC-02-001 | A kanban task created by Layer A spawns correct child tasks | Inspection |
| AC-02-002 | Layer B task completion does not bypass quality gate | Inspection |
| AC-02-003 | Human gate blocks task promotion until approval | Demonstration |

---

## Error Cases

| Scenario | Expected Behavior |
|----------|-------------------|
| Circular task dependency detected | Task creation rejected with error |
| Agent attempts to close its own quality gate | Operation rejected; gate remains open |

---

## Edge Cases

| Scenario | Expected Behavior |
|----------|-------------------|
| Parent task has no children after decomposition | Blocked state; requires Layer A intervention |
| Human reviewer rejects at gate | Task returns to Layer B with feedback |

---

## Open Questions

- What is the maximum depth of task decomposition before requiring human review?
- How does the orchestrator handle a blocked task that times out?

---

## Dependencies

- BRD-01 (App Shell)
- Feature flags: `kanban-orchestrator`, `layer-a-agents`, `layer-b-agents`, `human-gates`

---

## Metadata

| Field | Value |
|-------|-------|
| Created | Phase 0 |
| Revised | Phase 0 |
| Version | 1 |

*Placeholder. Full BRD to be authored in Phase 1.*
