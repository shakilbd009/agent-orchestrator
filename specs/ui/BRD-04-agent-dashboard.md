# BRD-04: Agent Workstream Dashboard

**Project:** agent-orchestrator  
**Domain:** UI  
**Phase:** 1  
**Owner:** developer  
**Status:** placeholder  

---

## Goal

Provide visibility into each agent's current task, recent outputs, next actions, and accountability relationships.

---

## Problem Statement

Users and Layer A orchestrators need to see which agent is working on what, whether outputs have passed validation, and who is blocked or waiting for approval.

---

## Scope

### In Scope
- Per-agent task status and current action
- Handoff chain visibility (who handed off to whom)
- Validation pass/fail status per agent output
- Blocked task indicators with reasons

### Out of Scope
- Non-agent human task assignment
- External system integration status
- Technical logs or debugging information

---

## Functional Requirements

### FR-04-001: Agent Task View
Each active agent displays its current task, the owning agent profile, and the task's position in the pipeline.

### FR-04-002: Handoff Chain
The view shows the chain of custody: which agent completed work, which agent received it, and whether the quality gate passed.

### FR-04-003: Blocked Task Indicators
Blocked tasks show the blocking reason and the owner responsible for unblocking.

---

## Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-04-001 | View updates to reflect task state changes | Real-time within 30 seconds of state change |
| NFR-04-002 | Supports viewing up to 20 concurrent agents | No performance degradation |

---

## Acceptance Criteria

| ID | Criterion | Verification Method |
|----|-----------|---------------------|
| AC-04-001 | Active agent shows current task title and status | Inspection |
| AC-04-002 | Handoff chain displays at least 3 prior handoffs | Inspection |
| AC-04-003 | Blocked indicator includes blocking reason | Inspection |

---

## Open Questions

- Should this view be visible to non-technical users, or restricted to orchestrators?

---

## Dependencies

- BRD-02 (Orchestration Pipeline)
- Feature flag: `agent-dashboard`

---

## Metadata

| Field | Value |
|-------|-------|
| Created | Phase 0 |
| Revised | Phase 0 |
| Version | 1 |

*Placeholder. Full BRD to be authored in Phase 1.*
