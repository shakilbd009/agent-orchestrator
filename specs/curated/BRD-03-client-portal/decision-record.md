# BRD-03 — Decision Record

**BRD:** BRD-03-client-portal
**Stage:** decision-record
**Status:** Complete

---

## ADR Index

| ADR | Title | Decider | Date | Status |
|-----|-------|---------|------|--------|
| ADR-03-001 | Client Portal BFF Architecture | architect | 2026-05-30 | Proposed |
| ADR-03-002 | SSE Subscription Scope — Per-Project Connections | architect | 2026-05-30 | Proposed |
| ADR-03-003 | Publication Validation — Shared Schema (UI Guard + API Gate) | architect | 2026-05-30 | Proposed |
| ADR-03-004 | Overdue Decision Threshold — Per Decision Item | architect | 2026-05-30 | Proposed |
| ADR-03-005 | Owner Label Mapping — API-Provided with Hardcoded Fallback | architect | 2026-05-30 | Proposed |

All ADRs live in `specs/curated/BRD-03-client-portal/decisions/`.

---

## OQ Resolutions

| OQ | Question | Resolution | Documented In | Date |
|----|----------|------------|---------------|------|
| OQ-03-001 | SSE event envelope schema for project-scoped updates | Deferred to BRD-02 API contract definition; BRD-03 requires `project_id`, `event_type`, `item_id`, `timestamp` minimum | requirements.md | 2026-05-30 |
| OQ-03-002 | Forbidden technical fields list for publication validation | Defined in ADR-03-003: stack traces, agent IDs, branch names, commit SHAs, file paths, infrastructure terms, raw log lines | ADR-03-003 | 2026-05-30 |
| OQ-03-003 | Global approval inbox SSE subscription scope | Per-project SSE connections (ADR-03-002); global inbox multiplexes client-side across all project streams | ADR-03-002 | 2026-05-30 |
| OQ-03-004 | SSE reconnect strategy and backoff policy | Exponential backoff with jitter; max 5 retries per minute; reconnect on project detail view mount | ADR-03-002 | 2026-05-30 |
| OQ-03-005 | 24h overdue threshold — per item or per project | Per decision item (ADR-03-004); correct FR-03-037 semantics | ADR-03-004 | 2026-05-30 |

---

## Trade-Off Decisions

| Decision | Chosen Option | Source | ADR |
|----------|---------------|--------|-----|
| Client portal BFF architecture | Option B — BFF aggregation layer (Phase 2); direct fetch Phase 1 prototype | trade-offs.md (Major Decision 1) | ADR-03-001 |
| SSE subscription scope | Option A — Per-project SSE connections | trade-offs.md (Major Decision 2) | ADR-03-002 |
| Publication validation enforcement | Option B — UI guard + API gate with shared schema | trade-offs.md (Major Decision 3) | ADR-03-003 |
| Overdue decision threshold | Option A — Per decision item | trade-offs.md (Major Decision 4) | ADR-03-004 |
| Owner label mapping | Option D — API-provided with hardcoded fallback | trade-offs.md (Major Decision 5) | ADR-03-005 |

---

## PM Gate Review

*No PM gate review has occurred yet. This BRD is in the systematic-refinement stage.*

Pipeline gate chain this BRD must pass:
```
scaffold → scaffold-review → BRD drafting → systematic-refinement → eval-contracts + flag/contract parity → eval-readiness-gate → completeness-score → PM gate review → validate-design → PM gate review → graduation evidence package → production-checklist → implementation
```

This decision record will be updated when PM gate reviews occur.

---

## Cross-BRD Dependencies Status

| BRD | Dependency | Status |
|-----|------------|--------|
| BRD-01 | App shell session/auth integration | Needed for BFF auth integration |
| BRD-02 | Project/task/gate/current-state APIs + SSE event stream | **BLOCKING**: OQ-03-001 (SSE envelope schema) must be resolved before eval contract authoring |
| BRD-04 | Scope boundary — internal agent dashboard excluded | Clear separation documented |
| BRD-16 | Risk data contract | Needed for FR-03-031 risk display |
| BRD-17 | Project access filtering enforcement | Needed for NFR-03-013 access boundary |
| BRD-18 | Scope boundary — simple comments only, no threads/mentions | Clear separation documented |
| BRD-19 | Comment/decision retention policy | Aligned with BRD-02 retention expectations |
| BRD-21 | Scope boundary — visible indicators only, no notification delivery | Clear separation documented |
| contracts/openapi.yaml | Client portal read/action endpoint definitions | Pending BRD-03 approval |
| contracts/events.md | Client portal decision/comment/publication event definitions | Pending BRD-03 approval |
| specs/feature-flags.md | `client-portal` flag registration | Must add when BRD-03 approved |

---

## Blockers for Downstream Eval Contract Authoring

| Blocker | Severity | Detail |
|---------|----------|--------|
| OQ-03-001: SSE event envelope schema | HIGH | Eval contract authoring for SSE-driven updates cannot proceed without BRD-02 defining event envelope (project_id, event_type, item_id, timestamp). Must resolve before `eval-contracts` gate. |
| OQ-03-002: Forbidden technical fields list | HIGH | Publication validation test cannot be authored without explicit forbidden fields list. ADR-03-003 provides initial list; needs BRD-02 review to ensure completeness. |

---

*Decision record complete — BRD-03 systematic-refinement output artifacts generated.*
*Pending: PM gate review, validate-design, completion-score gates before graduation.*