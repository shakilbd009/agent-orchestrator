# BRD-03 — Decision Record

**BRD:** BRD-03-client-portal
**Stage:** decision-record
**Status:** Complete

---

## ADR Index

| ADR | Title | Decider | Date | Status |
|-----|-------|---------|------|--------|
| ADR-03-001 | Client Portal BFF Architecture | architect | 2026-05-30 | **Accepted** |
| ADR-03-002 | SSE Subscription Scope — Per-Project Connections | architect | 2026-05-30 | **Accepted** |
| ADR-03-003 | Publication Validation — Shared Schema (UI Guard + API Gate) | architect | 2026-05-30 | **Accepted** |
| ADR-03-004 | Overdue Decision Threshold — Per Decision Item | architect | 2026-05-30 | **Accepted** |
| ADR-03-005 | Owner Label Mapping — API-Provided with Hardcoded Fallback | architect | 2026-05-30 | **Accepted** |

All ADRs live in `specs/curated/BRD-03-client-portal/decisions/`.

---

## OQ Resolutions

| OQ | Question | Resolution | Documented In | Date |
|----|----------|------------|---------------|------|
| OQ-03-001 | SSE event envelope schema for project-scoped updates | **Deferred to BRD-02** — BRD-03 defines minimum required fields (`project_id`, `event_type`, `item_id`, `timestamp`); BRD-02 must define the actual SSE event envelope format | requirements.md | 2026-05-30 |
| OQ-03-002 | Forbidden technical fields list for publication validation | **Deferred to BRD-02** — ADR-03-003 provides initial exhaustive list; BRD-02 review needed against real API payloads before SSE eval contract authoring | ADR-03-003 | 2026-05-30 |
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

### PM GATE 2 — Final Approval

**Decision:** APPROVE
**Task:** t_1f6419fa
**Date:** 2026-05-31
**Approver:** pm

**Evidence chain:** `t_28482151 → t_efbba3c9 → t_01cd552d → t_d0374578 → t_1f6419fa`

**Repair chain disposition:**
- t_28482151 (validate-design GATE 2): PASS — 1 MEDIUM, several LOW/INFO findings
- t_efbba3c9 (PM GATE 2 REPAIR): OpenAPI approval-decision contract missing; ADR index showed Proposed instead of Accepted
- t_01cd552d (architect repair): Added POST /client-portal/approvals/{approvalId}/decide with schemas; patched ADR index to Accepted
- t_d0374578 (validator re-run): PASS — contract parity and ADR status parity confirmed
- t_1f6419fa (PM GATE 2 final): APPROVE — repairs resolved; OQ-03-001 and OQ-03-002 are documented cross-BRD deferrals

**Stale language resolved:** Previous version of this record (pre-repair) stated "decision is REPAIR" and "pending PM approval". This version reflects the final APPROVE decision from t_1f6419fa.

Pipeline gate chain passed:
```
scaffold → scaffold-review → BRD drafting → systematic-refinement →
eval-contracts + flag/contract parity → eval-readiness-gate →
completeness-score → PM gate review → validate-design →
PM GATE 2 REPAIR → architect repair → validator re-run → PM GATE 2 APPROVE →
graduation evidence package → production-checklist → implementation
```

---

## Cross-BRD Dependencies

| BRD | Dependency | Status | Notes |
|-----|------------|--------|-------|
| BRD-01 | App shell session/auth integration | Needed | Required for BFF auth integration |
| BRD-02 | Project/task/gate/current-state APIs + SSE event stream | **DEFERRED** | OQ-03-001 (SSE envelope schema) deferred to BRD-02 API contract definition; OQ-03-002 (forbidden fields review) deferred to BRD-02 payload review |
| BRD-04 | Scope boundary — internal agent dashboard excluded | Clear separation documented | BRD-03 out of scope in brd.md |
| BRD-16 | Risk data contract | Needed for FR-03-031 | BRD-03 assumes `risks` endpoint in project detail |
| BRD-17 | Project access filtering enforcement | Needed for NFR-03-013 | Platform auth must provide `principal` context to BFF |
| BRD-18 | Scope boundary — simple comments only, no threads/mentions | Clear separation documented | BRD-03 out of scope in brd.md |
| BRD-19 | Comment/decision retention policy | Aligned | Aligned with BRD-02 retention expectations |
| BRD-21 | Scope boundary — visible indicators only, no notification delivery | Clear separation documented | BRD-03 out of scope in brd.md |
| contracts/openapi.yaml | Client portal read/action endpoint definitions | ✅ Defined | Lines 444-585; POST /decide added by t_01cd552d |
| contracts/events.md | Client portal decision/comment/publication event definitions | ✅ Defined | Lines 145-279 |
| specs/feature-flags.md | `client-portal` flag registration | ✅ Registered | Line 57, default false |

---

*Decision record complete — PM GATE 2 APPROVED (t_1f6419fa, 2026-05-31). Graduation evidence package complete. Ready for production-checklist/implementation.*