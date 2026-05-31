# BRD-03-client-portal — Systematic Refinement Complete

**BRD:** BRD-03-client-portal
**Output folder:** `specs/curated/BRD-03-client-portal/`
**Stage:** systematic-refinement (done)
**Status:** Ready for `eval-contracts` + `flag/contract parity` gate

---

## Artifacts Produced

| File | Purpose |
|------|---------|
| `requirements.md` | 52 FRs (5 categories) + 15 NFRs decomposed with eval methods and open questions |
| `trade-offs.md` | 5 major trade-off decisions with options, verdict rationale, cross-cutting trade-offs |
| `progressive-deepening-L1.md` | Component inventory (6 components), architecture diagram, data flows, basic error scenarios |
| `progressive-deepening-L2.md` | L2 for all 6 components, approval state machine, data model, SSE event processing flow |
| `edge-cases-L2.md` | 35 edge cases across 6 categories with handling and test guidance |
| `decision-record.md` | ADR index, OQ resolutions, trade-off decisions, cross-BRD dependencies, blockers |
| `decisions/ADR-03-001.md` | BFF architecture decision |
| `decisions/ADR-03-002.md` | SSE subscription scope decision |
| `decisions/ADR-03-003.md` | Publication validation enforcement decision + forbidden fields list |
| `decisions/ADR-03-004.md` | Overdue decision threshold (per item) decision |
| `decisions/ADR-03-005.md` | Owner label mapping decision |

---

## Key Decisions Made

1. **BFF architecture** (Phase 1 prototype: direct fetch; Phase 2: BFF aggregation layer)
2. **Per-project SSE connections** (up to 50 concurrent; fallback to BFF multiplexer if needed)
3. **UI guard + API gate for publication validation** (shared schema; forbidden fields list defined)
4. **Per-decision-item overdue threshold** (24h from item becoming client-visible/actionable)
5. **API-provided owner label mapping with hardcoded fallback** (5-min BFF cache TTL)

---

## Blockers for Downstream Eval Contract Authoring

| Blocker | Severity | Detail |
|---------|----------|--------|
| OQ-03-001: SSE event envelope schema | **HIGH** | BRD-02 must define: `project_id`, `event_type`, `item_id`, `timestamp`. Cannot author SSE eval contracts without this. |
| OQ-03-002: Forbidden technical fields list | **HIGH** | ADR-03-003 defines initial list; needs BRD-02 API review for completeness before eval contract authoring. |

---

## Cross-BRD Dependencies

- **BRD-02**: Project/task/gate/current-state APIs + SSE event stream (BLOCKING for OQ-03-001)
- **BRD-01**: App shell session/auth for BFF integration
- **BRD-16/17/18/21**: Scope boundaries clearly defined; no overlap
- **contracts/openapi.yaml**: Must add client portal endpoints after BRD-03 approval
- **contracts/events.md**: Must add client portal decision/comment/publication events after approval

---

## Feature Flag

`client-portal` flag must be added to `specs/feature-flags.md` with default `false`, domain `UI`, introduced in Phase 1. Existing `dashboard` flag preserved until later ADR decides deprecation.

---

## Next Steps

1. Resolve OQ-03-001 (SSE envelope schema) with BRD-02 team
2. BRD-02 review of forbidden technical fields list (OQ-03-002)
3. Proceed to `eval-contracts` gate
4. Add `client-portal` to feature-flags.md via a PR

---

*Systematic refinement complete — all design artifacts generated, all implicit decisions ADR'd, all OQs surfaced with blockers identified.*