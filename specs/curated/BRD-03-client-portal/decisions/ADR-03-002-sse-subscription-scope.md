# ADR-03-002: SSE Subscription Scope — Per-Project Connections

**ADR:** ADR-03-002
**Title:** SSE Subscription Scope — Per-Project Connections
**Status:** Accepted
**Date:** 2026-05-30
**Accepted:** 2026-05-30
**Decider:** architect
**BRD:** BRD-03-client-portal

---

## Context

FR-03-018 (global approval inbox) and FR-03-040 (real-time updates) require SSE for live updates. BRD-02 defines project-scoped SSE streams. The question is whether the portal maintains one global SSE connection covering all accessible projects, or one SSE connection per visible project.

The global approval inbox shows pending decisions across all accessible projects (up to 50 per NFR-03-002).

---

## Decision

**Option A — Per-project SSE connections.**

The portal opens one SSE EventSource per visible project, each subscribing to that project's BRD-02 event stream. For the global approval inbox, the client subscribes to the SSE streams of all accessible projects (potentially 50 concurrent connections).

If browser connection limits or performance issues arise at scale, this can be revisited with a server-side multiplexing approach (Option B — BFF SSE multiplexer) via a subsequent ADR.

---

## Alternatives Considered

**Option B — Single global SSE connection (rejected for now):**
- BRD-02 would need to support multi-project SSE scope
- Client-side event routing to correct UI components becomes complex without a client-side pub/sub hub

**Option C — Per-project with client-side hub (rejected for now):**
- Client-side event hub introduces state management complexity
- May be warranted at scale but adds unnecessary complexity for initial implementation

---

## Consequences

**Positive:**
- Aligns with BRD-02 project-scoped SSE design
- Simple event routing: each EventSource corresponds to one project
- No client-side pub/sub needed
- Failed connection on one project doesn't affect others

**Negative:**
- Up to 50 concurrent EventSource connections for a full portfolio
- Browser may have connection limits or throttling at scale
- More SSE connections means more BRD-02 backend resource usage

**Neutral:**
- If performance degrades, Option B (BFF SSE multiplexer) can be explored
- BFF can also aggregate SSE events from multiple projects server-side

---

## Implementation Notes

- Each SSE EventSource is created when project detail view mounts
- EventSource is closed when component unmounts or project changes
- On SSE disconnect: show "Live updates paused"; manual refresh available
- Reconnection: exponential backoff with jitter; max 5 retries per minute
- Client SSE handler routes events to correct project store based on `project_id` in event envelope

---

## Verification

Performance test: Portfolio with 50 accessible projects. Measure SSE connection count, memory usage, and event delivery latency (target: <2s per FR-03-040). If browser connection limits are hit, escalate to Option B evaluation.