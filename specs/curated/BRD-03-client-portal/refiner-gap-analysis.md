# BRD-03 — Refiner Gap Analysis

**BRD:** BRD-03-client-portal
**Refiner:** systematic-refinement (parallel track)
**Date:** 2026-05-30
**Status:** Gap analysis complete — blockers for eval-contracts confirmed, additional gaps identified

---

## Executive Summary

The architect's systematic-refinement (t_700e4ad0) produced 13 artifacts with thorough coverage: 52 FRs, 15 NFRs, 35 edge cases, 5 ADRs, approval state machine, data model, and SSE processing flow. Two HIGH-severity blockers identified by the architect are confirmed correct (OQ-03-001, OQ-03-002). The refiner analysis finds 4 additional gaps across 3 severity levels that must be addressed before implementation, though they do not block the eval-contracts gate.

**No cross-document contradictions found.** The BRD and its curated artifacts are internally consistent.

---

## Confirmed Blockers (Architect's OQs — Unchanged)

### OQ-03-001: SSE Event Envelope Schema — HIGH (BLOCKS eval-contracts)
**Status:** Unresolved. BRD-02 `events.md` is a Phase 0 placeholder lacking all 11 canonical envelope fields. BRD-03 requires minimum `project_id`, `event_type`, `item_id`, `timestamp` per implementation-readiness.md §4. Cannot author SSE eval contracts without this. The architect correctly flagged this as blocking.

### OQ-03-002: Forbidden Technical Fields List Review — HIGH (BLOCKS eval-contracts)
**Status:** Partially resolved. ADR-03-003 defines an initial list (stack traces, agent IDs, branch names, commit SHAs, file paths, infra terms, raw log lines). The list is substantive but needs BRD-02 API review for completeness before eval contract authoring. The architect correctly flagged this as blocking.

---

## Item 8 Analysis: NFR-03-006 vs. NFR-03-001 vs. BRD-02 NFRs

**Task requirement:** Flag any tension between NFR-03-006 (< 2s SSE latency) + NFR-03-001 (< 3s dashboard render) and BRD-02 NFRs.

**Finding: No tension exists.** These NFRs measure different things:
- **NFR-03-001 (< 3s dashboard render):** Time for initial portfolio page to show visible content (first paint of summary data)
- **NFR-03-006 (< 2s SSE latency):** Time from BRD-02 committing an update to that update appearing in the client portal UI via SSE

The architect correctly states NFR-03-006 "matches BRD-02 target." Both the SSE latency and dashboard render targets are independently achievable — SSE is an incremental update after initial load, not on the critical path for first contentful paint. **No NFR conflict identified.**

---

## Additional Gaps Found

### Gap-REF-01: Client "Information Provided" Event Missing from Event Model

**Severity:** MEDIUM
**Category:** Event/State Model completeness
**Location:** BRD-03 observability log events; progressive-deepening-L2.md state machine

**Finding:**

The approval state machine (progressive-deepening-L2.md) correctly shows:
```
need_more_information → waiting_on_response → (client responds) → pending
```

The log events include:
- `client_portal.approval.need_more_information` — when client asks for more info

**Gap:** No event is defined for when the client subsequently provides the requested information and the item returns to `pending` state. The BRD-02 event contract must include this event type, or the NMI resolution flow cannot be tested via SSE.

**Quote from state machine (progressive-deepening-L2.md, lines 90–101):**
```
waiting_on_response     │
     │                  │
     │ (client responds with info)           │
     │                  └──────────────────┘ │
     │                                              ▼
     │                                    (back to pending)
```

**Quote from observability log events (BRD-03, line 166):**
```
client_portal.approval.need_more_information — when a client asks for more information
```

**Gap:** `client_portal.approval.information_provided` (or equivalent) is absent from the log events list and the SSE event types in implementation-readiness.md.

**Impact:** Eval contracts for the NMI resolution flow cannot be authored without defining this event. The BRD-02 event contract is incomplete for the full approval lifecycle.

**Recommendation:** Add to `contracts/events.md`: `approval.updated` with `subtype: information_provided` or a dedicated `approval.information_provided` event type.

---

### Gap-REF-02: Phase 1 Direct-Fetch vs. NFR-03-004 Portfolio Landing Latency

**Severity:** MEDIUM (design clarification needed, not a contradiction)
**Category:** Architecture/NFR alignment
**Location:** ADR-03-001; requirements.md NFR-03-004 analysis; BRD-03 portfolio API

**Finding:**

ADR-03-001 defines Phase 1 as "direct fetch" (browser → BRD-02), Phase 2 as BFF (browser → BFF → BRD-02). The requirements analysis (requirements.md, NFR-03-004) declares "No gap — clearly specified."

The portfolio landing page at 50-project scale requires:
- 1 API call to get project list (aggregated)
- Up to 50 parallel calls for per-project health/decision counts
- SSE connections for all visible projects

Under direct fetch (Phase 1), these calls originate from the browser. HTTP/2 multiplexing from a single browser origin is typically limited to ~6–100 concurrent streams depending on server configuration. With 50 projects, all 50 API calls would contend for this limit simultaneously.

**Quote from progressive-deepening-L1.md (L1 component #1, lines 70–72):**
```
Initial questions raised:
- How is the initial project list sorted?
- Does the landing page paginate if 50 projects exceed viewport?
```

These questions were raised but not resolved in the artifacts.

**Clarification needed:** NFR-03-004 ("visible within 5 seconds") measures when high-level summary appears, not when all 50 per-project data points are loaded. Progressive loading (FR-03-054) partially addresses this. However, if all 50 project-level API calls must complete before the portfolio summary counts are shown, the 5s target may be unreachable in Phase 1 direct-fetch mode.

**Not a blocker for eval-contracts** because:
1. Eval contracts test behavior, not performance at scale
2. Phase 2 BFF architecture resolves this through server-side parallelization
3. FR-03-054 (progressive loading) provides a defined path

**Recommendation:** Before Phase 1 implementation, verify that the aggregated portfolio API (`GET /portfolio?principal=<id>`) returns complete counts in a single call rather than requiring 50 parallel browser-to-BRD-02 requests. If not, Phase 1 will need an aggregation endpoint.

---

### Gap-REF-03: SSE Reconnect Exhaustion — Behavior After Max Retries

**Severity:** MEDIUM
**Category:** Failure mode completeness
**Location:** ADR-03-002 (implementation notes); edge-cases-L2.md E3.2/E3.3

**Finding:**

ADR-03-002 states:
> Reconnection: exponential backoff with jitter; max 5 retries per minute

**Gap:** What happens after 5 retries-per-minute are exhausted? The options are:
1. Stop reconnecting entirely (portal remains in "live updates paused" indefinitely, user must manually refresh on next visit)
2. Reduce retry frequency and continue attempting (lower-frequency polling mode)
3. Switch to periodic polling via current-state API as a fallback

E3.2 (SSE disconnect during active session) describes the immediate behavior but not the post-exhaustion state.

**Quote from ADR-03-002 (implementation notes):**
```
On SSE disconnect: show "Live updates paused"; manual refresh available
Reconnection: exponential backoff with jitter; max 5 retries per minute
```

**Impact:** Without a defined post-exhaustion behavior, implementations will diverge. A reasonable default (polling fallback) may be assumed but not tested.

**Recommendation:** Add to ADR-03-002: after max retries exhausted, BFF transitions to polling mode at reduced frequency (e.g., 30s interval) using current-state APIs, until SSE connection is re-established. Or explicitly defer to implementation.

---

### Gap-REF-04: Comment Query Semantics for Deleted Comments

**Severity:** LOW
**Category:** Data model / API contract ambiguity
**Location:** progressive-deepening-L2.md data model (ClientVisibleComment entity); BRD-03 FR-03-028

**Finding:**

The data model defines:
```
ClientVisibleComment:
  is_deleted: boolean
```

FR-03-028 states deleted comments are "hidden from the normal client view."

**Gap:** The API contract does not specify whether `GET /projects/<id>/comments` (or equivalent) returns comments with `is_deleted: true` or excludes them entirely.

Two valid patterns:
- **Pattern A (exclude):** Deleted comments are not returned in query results. Client cannot see them at all. Audit system retains them separately.
- **Pattern B (include with flag):** Deleted comments are returned with `is_deleted: true`. Client UI filters them out, but the data is available if needed for audit/debugging.

**Quote from FR-03-028 (BRD-03):**
> Comment authors MUST be able to edit and delete their own comments. Deleted comments MUST be hidden from the normal client view.

"Normal client view" implies the API may still return them for internal/audit use. But the API contract must specify which pattern is implemented.

**Impact:** Low — the UI behavior is defined (hidden from normal client view), but eval contracts for comment queries need a definite answer.

**Recommendation:** Define in BRD-02 comment API contract: deleted comments are excluded from query results (Pattern A) unless the requestor has audit/special-role access. Add `is_deleted` to the data model only for internal audit queries.

---

### Gap-REF-05: BRD-02 Search API Contract Missing

**Severity:** LOW (not blocking eval-contracts)
**Category:** Cross-BRD dependency
**Location:** requirements.md FR-03-038 open issues; implementation-readiness.md dependencies table

**Finding:**

FR-03-038 (Search) has an open issue: "BRD-02 search API contract." The search API is required to implement cross-project search filtered to accessible projects, but BRD-02's search API contract is not defined in the BRD-03 artifacts.

**Quote from requirements.md (FR-03-038):**
```
Search | MUST | Text search across client-visible project/task/risk/milestone/approval/blocker
       |      | content; filtered to accessible projects | Search integration test | BRD-02 search API contract |
```

**Impact:** Search integration tests cannot be authored until BRD-02 defines the search API. This is a cross-BRD dependency, not a BRD-03 gap. BRD-02 must define search before eval-contracts for search can be written.

---

### Gap-REF-06: Publication "Republish" Event Missing

**Severity:** LOW
**Category:** Event model completeness
**Location:** BRD-03 observability log events; progressive-deepening-L2.md edge case E2.2

**Finding:**

E2.2 describes the `request_changes → pending` transition via republished item:
> Client submits `request_changes` with comment. Item enters `requested_changes` state. Internal owner addresses changes and **republishes**. Item returns to `pending` for client re-review.

**Gap:** No log event is defined for the internal owner republishing an item that was in `requested_changes` state. The log events include:
- `client_portal.item.published`
- `client_portal.item.unpublished`
- `client_portal.publication_validation.failed`

But republish (publish → unpublished → republish) is not distinguished from initial publish.

**Impact:** Low — eval contracts can treat republish as a publish event, but distinguishing it would improve audit clarity.

---

### Gap-REF-07: SSE Latency Budget Allocation Not Specified

**Severity:** LOW
**Category:** NFR clarity
**Location:** NFR-03-006; ADR-03-002

**Finding:**

NFR-03-006: "Relevant committed updates visible within 2 seconds when SSE is connected, matching BRD-02 target."

This 2-second requirement spans two hops:
1. BRD-02 commits update → BFF receives SSE event
2. BFF processes event (filters, transforms, strips forbidden content) → client receives via SSE

Neither the NFR nor ADR-03-002 specifies how the 2s budget is allocated between these hops. If BRD-02's SSE latency is already 1.8s, the BFF processing time must fit in 0.2s.

**Impact:** Low — implementation teams will discover this in load testing. Specifying a per-hop budget (e.g., BRD-02 → BFF < 1.5s, BFF → client < 0.5s) would make the NFR testable at integration boundaries.

---

## Confirmed Adequate Coverage (No Gaps)

### Approval State Machine — CONFIRMED COMPLETE
The architect's state machine correctly captures all transitions:
- `pending → approve` (terminal)
- `pending → reject` (terminal)
- `pending → request_changes` (terminal per the diagram, though E2.2 shows republish → pending)
- `pending → need_more_information` → `waiting_on_response` → `pending` (on client information provision)

Note: The state machine diagram shows `request_changes` as terminal. Edge case E2.2 (request_changes → pending on republish) contradicts this. The state machine in progressive-deepening-L2.md should explicitly show `requested_changes → pending` transition on republish.

**This is a minor documentation inconsistency, not a functional gap.** The edge case E2.2 is documented and handles the transition correctly.

### Comment Audit Privacy — CONFIRMED CORRECT
FR-03-029 is explicit: audit events must not retain deleted comment bodies or previous edited body versions. The BRD, requirements.md, and edge-cases-L2.md are consistent. AC-03-029 tests this directly. No gap.

### Completion Percentage Edge Cases — CONFIRMED HANDLED
- Empty project (no active tasks): denominator=0 → "No active work yet" (FR-03-010 + E1.2)
- All cancelled: denominator=0 → same empty state (E1.2)
- Mixed states: done/(todo+in_progress+blocked+done) = valid %, e.g., all-done = 100%

Architect addressed E1.2 correctly. No gap.

### Cross-Project Isolation — CONFIRMED ADEQUATE
FR-03-006, FR-03-007, NFR-03-013, and the BFF access filtering layer provide comprehensive coverage. E3.5 (late-arriving SSE event for revoked access) is correctly identified as critical. SSE event filtering by `project_id ∈ principal.accessible_project_ids` is defined in the SSE processing flow. No gap.

### Publication Validation — CONFIRMED COMPLETE
The forbidden fields list (ADR-03-003) and the publication validation schema (progressive-deepening-L2.md) are comprehensive. The UI guard + API gate design (ADR-03-003 Option B) is sound. No functional gap — only the OQ-03-002 review dependency remains.

---

## Summary Table

| Gap | Severity | Blocks eval-contracts? | Description |
|-----|----------|------------------------|-------------|
| OQ-03-001 | HIGH | YES | SSE event envelope schema undefined in BRD-02 |
| OQ-03-002 | HIGH | YES | Forbidden fields list needs BRD-02 review |
| Gap-REF-01 | MEDIUM | NO | Client "information provided" event missing from event model |
| Gap-REF-02 | MEDIUM | NO | Phase 1 direct-fetch vs. NFR-03-004 portfolio latency clarification needed |
| Gap-REF-03 | MEDIUM | NO | SSE reconnect exhaustion behavior undefined after max retries |
| Gap-REF-04 | LOW | NO | Comment query semantics for deleted comments ambiguous |
| Gap-REF-05 | LOW | NO | BRD-02 search API contract missing (cross-BRD dependency) |
| Gap-REF-06 | LOW | NO | Publication republish event not distinguished from initial publish |
| Gap-REF-07 | LOW | NO | SSE latency budget allocation across hops not specified |

---

## OQ-03-001 / OQ-03-002 Resolution Path

Both HIGH blockers require BRD-02 action:

1. **OQ-03-001:** BRD-02 team must finalize `events.md` canonical envelope schema with minimum `project_id`, `event_type`, `item_id`, `timestamp`. Until then, eval contracts for SSE-driven updates cannot be authored.

2. **OQ-03-002:** BRD-02 team reviews the forbidden technical fields list in ADR-03-003 for completeness against actual BRD-02 API response payloads. The list is substantive but not yet validated against real data shapes.

**Next gate:** eval-contracts + flag/contract parity. Both blockers must be resolved before eval-contracts can be authored.

---

## Minor Documentation Inconsistency (Non-Blocking)

The state machine diagram (progressive-deepening-L2.md) shows `request_changes` as terminal, but E2.2 describes a `requested_changes → pending` transition on republish. This should be reconciled in the state machine diagram for clarity, but the edge case documentation (E2.2) correctly handles the behavior. No functional impact.

---

*Gap analysis complete. Refiner confirms architect's work is thorough and the two HIGH blockers are accurately identified. Seven additional gaps found (4 MEDIUM, 3 LOW), none of which block the eval-contracts gate but must be resolved before Phase 1 implementation proceeds.*
