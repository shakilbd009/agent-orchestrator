# BRD-03 — Validator Findings

**BRD:** BRD-03-client-portal
**Stage:** validator-findings
**Status:** Complete

---

## Validation Chain

| Task | Profile | Verdict | Date |
|------|---------|---------|------|
| t_28482151 — validate-design GATE 2 | validator | PASS | 2026-05-30 |
| t_efbba3c9 — PM GATE 2 REPAIR decision | pm | REPAIR → repair chain created | 2026-05-30 |
| t_01cd552d — architect repair | architect | Complete | 2026-05-30 |
| t_d0374578 — validator re-run | validator | PASS | 2026-05-31 |
| t_1f6419fa — PM GATE 2 final APPROVE | pm | APPROVE | 2026-05-31 |

Evidence chain: `t_28482151 → t_efbba3c9 → t_01cd552d → t_d0374578 → t_1f6419fa`

---

## validate-design Findings (t_28482151)

**Gate:** validate-design GATE 2
**Verdict:** PASS — no blocking issues; 1 MEDIUM and several LOW/INFO findings

### Security — PASS

| FR | Requirement | Finding | Disposition |
|----|-------------|---------|-------------|
| FR-03-006/03-007 | Cross-project access filtering | BFF enforces `principal` filtering; unauthorized project access returns empty result (not 403) per implementation-readiness.md. Consistent with NFR-03-013 (fail-closed). | Accepted |
| FR-03-029 | Comment audit privacy | implementation-readiness.md explicitly specifies no body text retained in edit/delete audit events. Security eval contract covers this. | Accepted |
| FR-03-045/03-048 | Publication validation | Forbidden content patterns (stack traces, agent IDs, branch names, commit SHAs, file paths, infra terms, log lines) per ADR-03-003 implemented in both security eval contract and implementation-readiness.md schema. | Accepted |
| NFR-03-013 | Access boundary failures | `client_portal_access_denied_total` counter emitted; fail-closed behavior enforced. | Accepted |

**No security blocking issues.**

### Architecture — PASS (1 MEDIUM gap)

| FR | Finding | Severity | Disposition |
|----|---------|----------|-------------|
| ADR-03-001 | BFF aggregation pattern (Phase 1 direct fetch, Phase 2 BFF) consistent across brd.md, implementation-readiness.md, and ADR-03-001. | — | Accepted |
| ADR-03-002 | SSE per-project connections; events.md BRD-03 section defines client-facing events with envelope-stripping for actorId/actorRole. | — | Accepted |
| FR-03-040/03-041 | SSE live updates + manual refresh fallback: current-state APIs authoritative; "live updates paused" shown on disconnect. | MEDIUM | Accepted — design documented; Phase 1 SSE reliability to be validated during implementation |

### ADR Index Table

| Finding | Severity | Disposition |
|---------|----------|-------------|
| decision-record.md lines 13-17 showed ADRs as "Proposed" in index; authoritative ADR files show "Accepted". | COSMETIC | **Repaired** by t_01cd552d — index now shows Accepted for all 5 ADRs |

---

## validate-design Re-run Findings (t_d0374578)

**Scope:** Post-repair check against t_01cd552d artifacts

### Finding 1 — POST /decide Contract: PASS

| Contract | Location | Status |
|----------|----------|--------|
| `POST /client-portal/approvals/{approvalId}/decide` operationId | openapi.yaml lines 541-582 | ✅ Defined |
| `ApprovalDecisionRequest` schema (outcome enum, comment, clientOwnerLabelOverride) | openapi.yaml lines 910-941 | ✅ Defined |
| `ApprovalDecisionResponse` schema (success, updatedApproval, message) | openapi.yaml lines 910-941 | ✅ Defined |
| 200/400/403/404 responses | openapi.yaml lines 541-582 | ✅ Defined |
| FR-03-021, FR-03-022, FR-03-024 satisfied | — | ✅ Parity confirmed |
| AC-03-019 through AC-03-023 satisfied | — | ✅ Parity confirmed |

**Verdict: PARITY CONFIRMED** — OpenAPI fully matches BRD-03 approval semantics.

### Finding 2 — ADR Index Consistency: PASS

| ADR | Title | Status in Index | Status in File |
|-----|-------|-----------------|---------------|
| ADR-03-001 | Client Portal BFF Architecture | Accepted | Accepted |
| ADR-03-002 | SSE Subscription Scope — Per-Project Connections | Accepted | Accepted |
| ADR-03-003 | Publication Validation — Shared Schema | Accepted | Accepted |
| ADR-03-004 | Overdue Decision Threshold — Per Decision Item | Accepted | Accepted |
| ADR-03-005 | Owner Label Mapping — API-Provided with Hardcoded Fallback | Accepted | Accepted |

**Verdict: PARITY CONFIRMED** — Previous blocking finding (ADR index showed Proposed) is resolved.

### Finding 3 — Remaining OQs and Contract Gaps

#### Cross-BRD Dependencies (non-blocking, documented as deferred)

| ID | Severity | Description | Deferred To | Disposition |
|----|----------|-------------|------------|-------------|
| OQ-03-001 | HIGH | SSE event envelope schema undefined in BRD-02 events.md | BRD-02 | **Deferred** — documented in decision-record.md line 27; BRD-03 requires `project_id`, `event_type`, `item_id`, `timestamp` minimum |
| OQ-03-002 | HIGH | Forbidden technical fields list needs BRD-02 review against real API payloads | BRD-02 | **Deferred** — ADR-03-003 provides initial list; BRD-02 review still needed per decision-record.md lines 82-83 |

Both OQs are cross-BRD dependencies PM has visibility into. BRD-03 artifacts contain no TBDs or unresolved design decisions. PM GATE 2 APPROVE (t_1f6419fa) confirms these deferrals are acceptable.

#### Additional Gaps from refiner-gap-analysis.md (non-blocking)

| ID | Severity | Description | Disposition |
|----|----------|-------------|-------------|
| Gap-REF-01 | MEDIUM | Client "information provided" event missing from events.md | Documented; does not block BRD-03 |
| Gap-REF-02 | MEDIUM | Phase 1 direct-fetch vs. NFR-03-004 portfolio latency clarification | Documented; implementation must meet 5s target regardless of fetch pattern |
| Gap-REF-03 | MEDIUM | SSE reconnect exhaustion behavior undefined after max retries | Documented; implementation should use exponential backoff with jitter; max 5 retries/min per ADR-03-002 |
| Gap-REF-04 | LOW | Comment query semantics for deleted comments ambiguous | Documented; FR-03-028/03-029 behavior unambiguous (deleted hidden, no body retained) |
| Gap-REF-05 | LOW | BRD-02 search API contract missing | Deferred to BRD-02 |
| Gap-REF-06 | LOW | Publication republish event not distinguished | Documented; ItemPublished event covers both initial and republished |
| Gap-REF-07 | LOW | SSE latency budget allocation not specified | Documented; NFR-03-006 requires 2s freshness; implementation allocates within that budget |

### Feature Flag — PASS

`client-portal` flag registered in `specs/feature-flags.md` (line 57) with default `false`, Phase 1 introduction. No registration gaps.

### Contract Parity Summary

| Contract | BRD-03 Requirement | Status |
|----------|-------------------|--------|
| openapi.yaml — POST /decide | FR-03-021/022/024, AC-03-019 to AC-03-023 | ✅ Defined (lines 541-582, 910-941) |
| openapi.yaml — all client-portal endpoints | All FRs | ✅ Defined (lines 463-628) |
| events.md — client portal events | FR-03-025-030, FR-03-043-048 | ✅ Defined (lines 145-279) |
| feature-flags.md — client-portal | Feature Flag section | ✅ Registered (line 57) |
| ADR index | All ADRs Accepted | ✅ Consistent |

---

## Open Questions (Cross-BRD Deferrals)

| OQ | Question | Deferred To | Not a Blocker Because |
|----|----------|------------|----------------------|
| OQ-03-001 | SSE event envelope schema | BRD-02 | BRD-03 artifacts define minimum required fields; BRD-02 defines the actual event envelope format |
| OQ-03-002 | Forbidden technical fields list review | BRD-02 | ADR-03-003 provides an initial exhaustive list; BRD-02 API review may add additional patterns |

Both OQs are documented as cross-BRD dependencies in decision-record.md and are not BRD-03 artifact deficiencies. PM approved BRD-03 with these deferrals documented.

---

## Overall Verdict

**validate-design GATE 2: PASS**

No blocking issues remain on BRD-03 artifacts after repair chain:
- POST /decide OpenAPI contract: ✅
- ADR index consistency: ✅
- No TBDs in brd.md: ✅
- Feature flag registered: ✅

PM GATE 2 APPROVE (t_1f6419fa, 2026-05-31) grants final approval with cross-BRD deferrals documented.

---

*Validator findings complete — graduation package ready for production-checklist/implementation*
