# BRD-03: Business Dashboard

**Project:** agent-orchestrator  
**Domain:** UI  
**Phase:** 1  
**Owner:** developer  
**Status:** placeholder  

---

## Goal

Provide non-technical users with a plain-language view of project status, pending approvals, blockers, risks, and quality gate results.

---

## Problem Statement

Non-technical users cannot interpret raw kanban output or agent logs. They need a business-friendly dashboard that explains what is happening, what is blocked, and what requires their decision.

---

## Scope

### In Scope
- Project status summary (phase, progress, milestone)
- Pending approval queue
- Blocker and risk visibility
- Quality gate pass/fail summary
- Plain-language descriptions without technical jargon

### Out of Scope
- Agent-specific workstream views
- Raw technical logs or stack traces
- Infrastructure or DevOps metrics

---

## Functional Requirements

### FR-03-001: Project Status View
The dashboard displays the current project phase, overall completion percentage, and next milestone in plain language.

### FR-03-002: Approval Queue
Users see a list of pending approvals with a one-sentence description of what decision is required and why.

### FR-03-003: Blocker Summary
The dashboard surfaces active blockers with an assigned owner and suggested resolution.

---

## Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-03-001 | Dashboard loads without authentication delay | Initial render within 3 seconds |
| NFR-03-002 | Content is readable on tablet-sized screens | Responsive layout at 768px width |

---

## Acceptance Criteria

| ID | Criterion | Verification Method |
|----|-----------|---------------------|
| AC-03-001 | User sees current project phase and status without navigating | Inspection |
| AC-03-002 | Pending approval list shows at most 10 items with one-line descriptions | Inspection |
| AC-03-003 | Blocker entries include owner and resolution hint | Inspection |

---

## Open Questions

- Should the dashboard support multiple projects simultaneously?
- How frequently should status data refresh?

---

## Dependencies

- BRD-02 (Orchestration Pipeline)
- Feature flag: `dashboard`

---

## Metadata

| Field | Value |
|-------|-------|
| Created | Phase 0 |
| Revised | Phase 0 |
| Version | 1 |

*Placeholder. Full BRD to be authored in Phase 1.*
