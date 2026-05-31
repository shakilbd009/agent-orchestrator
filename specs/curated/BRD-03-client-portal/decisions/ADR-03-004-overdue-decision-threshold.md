# ADR-03-004: Overdue Decision Threshold — Per Decision Item

**ADR:** ADR-03-004
**Title:** Overdue Decision Threshold — Per Decision Item
**Status:** Accepted
**Date:** 2026-05-30
**Accepted:** 2026-05-30
**Decider:** architect
**BRD:** BRD-03-client-portal

---

## Context

FR-03-037 defines a client decision as overdue when "it remains pending more than 24 hours after becoming visible and actionable to the client." The question is whether the 24h clock applies per individual approval item or per project.

---

## Decision

**Option A — Per decision item (individual approval item).**

Each approval item has its own 24-hour clock starting from the moment it becomes client-visible and actionable. A project with 10 pending decisions where 1 has been pending for 48 hours shows exactly 1 overdue decision.

---

## Alternatives Considered

**Option B — Per project aggregate (rejected):**
- Would mark all decisions on a project as overdue if ANY single decision is old
- Incorrect semantics: one old decision would make all other recent decisions appear overdue
- Misleading signal for clients

**Option C — Per-project but per-decision shown (rejected):**
- Most complex; per-project flag adds UI complexity without clear benefit
- Does not match BRD language "a client decision"

---

## Consequences

**Positive:**
- Correctly implements FR-03-037 semantics: "a client decision" not "a project"
- Accurate count: 1 old decision = 1 overdue count, not 10
- Clear signal: client knows exactly which decision is stale

**Negative:**
- Requires per-item timestamp tracking (item.created_at → when visible to client)
- More implementation complexity than per-project aggregation

**Neutral:**
- The `created_at` timestamp for approval items is the authoritative "visible/actionable" clock
- Overdue is a visibility signal only — does not auto-reject, auto-approve, or advance work (FR-03-037 explicitly states this)

---

## Implementation Notes

- `approval_item.created_at` is the timestamp when item became visible/actionable to client
- Overdue check: `now() - item.created_at > 24 hours`
- Overdue recalculated on every page load and SSE update
- `client_portal_overdue_decisions_current` gauge reflects per-item overdue count
- `client_portal_oldest_pending_decision_age_ms` gauge tracks maximum age of any pending item

---

## Verification

Time-controlled integration test: Create approval item at T=0. Verify not overdue at T=23h. Verify marked overdue at T=25h. Verify project with 5 items (1 old, 4 new) shows overdue count of 1, not 5.