# BRD-02 — Decision Record

**BRD:** BRD-02-platform-native-orchestration-pipeline
**Stage:** 05-decision-record
**Status:** Complete

---

## 1. PM Gate Decisions

### t_7cc74376 — PM Approval of Repair Path

| Field | Value |
|-------|-------|
| **Task** | t_7cc74376 — PM approve repair path |
| **Verdict** | `APPROVED` — repair path approved for eval findings |
| **Date** | 2026-05-28 |
| **Approver** | pm (profile) |
| **Source-task** | t_7cc74376 |
| **Context** | Approved the repair strategy for security eval (t_b481ed59), performance eval (t_1bad7f24), and webhook header (t_9fd392cb) |

### t_6f5b3407 — Eval-Readiness Re-run Approval

| Field | Value |
|-------|-------|
| **Task** | t_6f5b3407 — eval-readiness re-run approval |
| **Verdict** | `APPROVED` — eval-readiness gate passed after repair |
| **Date** | 2026-05-28 |
| **Approver** | pm (profile) |
| **Source-task** | t_6f5b3407 |
| **Context** | t_06e22673 findings repaired; BRD-02 eval coverage confirmed at 25/25 required eval files |

---

## 2. OQ Resolutions

### OQ-1: `platform-orchestration` Flag Addition

| Field | Value |
|-------|-------|
| **OQ** | Should `platform-orchestration` be added immediately as a new registry flag while leaving `kanban-orchestrator` untouched? |
| **Resolution** | Add `platform-orchestration` as the master flag. Label `kanban-orchestrator` legacy/compatibility. Both coexist in feature-flags.md. |
| **Documented in** | FR-02-028, FR-02-029, feature-flags.md |
| **Source** | brd.md (approved) · requirements.md · trade-offs.md |
| **Date** | 2026-05-28 |

### OQ-3: Webhook Signing

| Field | Value |
|-------|-------|
| **OQ** | Should webhook signing be mandatory or acceptable as a should-have for local/internal use? |
| **Resolution** | Mandatory signing with `X-Webhook-Signature: HMAC-SHA256` for all webhook registrations except `localhost` and `127.0.0.1` in development. |
| **Documented in** | ADR-02-003 · contracts/events.md · requirements.md |
| **Source** | trade-offs.md · systematic-refinement (t_f5b70b60) |
| **Date** | 2026-05-28 |

### OQ-4: Event Schema Version Naming

| Field | Value |
|-------|-------|
| **OQ** | Should orchestration event schemas be labeled `v1alpha` during Phase 1/2, or start at `v1`? |
| **Resolution** | `v1alpha` during Phase 1/2 implementation; `v1` after BRD-02 implementation GA. |
| **Documented in** | ADR-02-001 · contracts/events.md |
| **Source** | systematic-refinement (t_f5b70b60) |
| **Date** | 2026-05-28 |

### OQ-7: Decomposition Idempotency

| Field | Value |
|-------|-------|
| **OQ** | Should re-submitting an equivalent decomposition request create duplicates? |
| **Resolution** | No. FR-02-009A specifies idempotency: only one active proposal per parent. Atomically check-and-create in one DB transaction. |
| **Documented in** | FR-02-009A · requirements.md · progressive-deepening-L1.md |
| **Source** | brd.md (approved) |
| **Date** | 2026-05-28 |

---

## 3. Trade-Off Decisions

### Trade-Off 1: Event Schema Version Naming

| Field | Value |
|-------|-------|
| **Decision** | Event schema version naming convention |
| **Chosen option** | `v1alpha` during Phase 1/2; `v1` at BRD-02 GA |
| **Source** | requirements.md · systematic-refinement |
| **Approver** | architect (subagent) |
| **Date** | 2026-05-28 |
| **ADR ref** | ADR-02-001 |
| **Rationale** | `v1alpha` signals pre-GA instability to consumers. Starting at `v1` would be a false claim of stability before the contract is frozen. GA promotion is a meaningful signal. |

### Trade-Off 2: Transaction Boundary

| Field | Value |
|-------|-------|
| **Decision** | Atomic commit of current-state mutation + audit event append |
| **Chosen option** | One DB transaction as default; diagnostic consistency check on `/ready` if engine lacks row-level transactions |
| **Source** | requirements.md · systematic-refinement |
| **Approver** | architect (subagent) |
| **Date** | 2026-05-28 |
| **ADR ref** | ADR-02-002 |
| **Rationale** | Strongest consistency; eliminates drift between current-state and audit log. `where possible` qualifier handles edge cases. |

### Trade-Off 3: Webhook Signing

| Field | Value |
|-------|-------|
| **Decision** | Mandatory webhook signing with HMAC-SHA256 |
| **Chosen option** | Mandatory signing for all webhook registrations except localhost/127.0.0.1 dev URLs |
| **Source** | requirements.md · systematic-refinement |
| **Approver** | architect (subagent) |
| **Date** | 2026-05-28 |
| **ADR ref** | ADR-02-003 |
| **Rationale** | External webhook delivery without signing is a vector for injection attacks. Localhost exemption reduces dev friction while maintaining security for production. |

### Trade-Off 4: `/ready` Degradation Semantics

| Field | Value |
|-------|-------|
| **Decision** | Hard fail (503) with failing subsystem in body when webhook queue unavailable |
| **Chosen option** | Two-tier readiness: minimum tier (storage + state + audit) vs. full tier (+ webhook queue). Webhook receiver outages do NOT make platform unready. |
| **Source** | systematic-refinement · progressive-deepening-L1.md |
| **Approver** | architect (subagent) |
| **Date** | 2026-05-28 |
| **ADR ref** | ADR-02-004 |
| **Rationale** | Webhook queue unavailability is a real infrastructure degradation that ops needs to know about. A 503 with clear JSON messaging is the right signal. Webhook receiver (not queue) failures are isolated per design. |

### Trade-Off 5: `system` Role Authority

| Field | Value |
|-------|-------|
| **Decision** | `system` role is infrastructure-only |
| **Chosen option** | `system` may emit audit events and feature-flag-change events only. All other mutations require `human`, `layer_a`, or `layer_b`. |
| **Source** | systematic-refinement · requirements.md |
| **Approver** | architect (subagent) |
| **Date** | 2026-05-28 |
| **ADR ref** | ADR-02-005 |
| **Rationale** | Infrastructure-only is the safest default. Overbroad `system` authority creates gate bypass risk. Future automation uses named service accounts with proper role claims. |

### Trade-Off 6: CORS Policy for SSE Endpoint

| Field | Value |
|-------|-------|
| **Decision** | Per-project configurable allowed-origin list |
| **Chosen option** | SSE `OPTIONS` preflight returns `Access-Control-Allow-Origin` validated against project allowlist. Defaults to `["http://localhost", "http://127.0.0.1"]`. Wildcard `*` not permitted in non-dev. |
| **Source** | systematic-refinement · progressive-deepening-L1.md |
| **Approver** | architect (subagent) |
| **Date** | 2026-05-28 |
| **ADR ref** | ADR-02-006 |
| **Rationale** | Project-scoped model fits naturally. Dashboard (BRD-03) is browser-based and requires CORS. Wildcard is a security risk for production. |

---

## 4. ADR Summary

| ADR | Title | Decider | Date | Status |
|-----|-------|---------|------|--------|
| ADR-02-001 | Event Schema Version Naming Convention | architect | 2026-05-28 | Accepted |
| ADR-02-002 | Transaction Boundary for State + Audit Append | architect | 2026-05-28 | Accepted |
| ADR-02-003 | Webhook Signing Requirement | architect | 2026-05-28 | Accepted |
| ADR-02-004 | `/ready` Degradation Semantics | architect | 2026-05-28 | Accepted |
| ADR-02-005 | `system` Role Authority Enumeration | architect | 2026-05-28 | Accepted |
| ADR-02-006 | CORS Policy for SSE Endpoint | architect | 2026-05-28 | Accepted |

All ADRs are in `specs/curated/BRD-02-platform-native-orchestration-pipeline/decisions/` (same as existing refinement artifact path).

---

## 5. Parent Task Handoff Summary

| Task | Role | Key Outcome |
|------|------|-------------|
| t_f5b70b60 | systematic-refinement | Requirements analysis, 6 ADRs created, OQ resolutions, edge cases L2 |
| t_06e22673 | eval-readiness | Initial findings: security eval (SEC-3), performance eval (PERF-3), webhook header; repair spawned |
| t_b481ed59 | security eval repair | SEC findings repaired; security eval repaired to passing |
| t_1bad7f24 | performance eval repair | PERF findings repaired; performance eval repaired to passing |
| t_9fd392cb | webhook header repair | `X-Webhook-Signature` header mismatch corrected in webhook integration eval |
| t_7cc74376 | PM approve repair path | Repair strategy approved |
| t_6f5b3407 | eval-readiness re-run approval | Eval-readiness gate passed; BRD-02 eval coverage 25/25 |

---

## 6. Open Items with Disposition

| Item | Disposition | Rationale |
|------|-------------|-----------|
| OQ-2: Task type taxonomy to gate template mapping | Deferred to BRD-08 or project configuration | Not required for Phase 1 implementation |
| OQ-5: Archival/deletion workflow | Deferred: export API required before deletion; full workflow out of scope | Minimal export API defined in FR-02-027 and AC-02-029 |
| OQ-6: Task-level gate approvers without human | Deferred: Layer A by default; `scope_review` gates require human | Not blocking for Phase 1 gate model |
| Custom gate types | Out of scope for BRD-02 | Built-in templates only |
| LLM inference events | Deferred to BRD-05 | Not yet defined |
| Agent memory events | Deferred to BRD-06 | Not yet defined |
| Notification events | Deferred to BRD-21 | Not yet defined |

---

## 7. Source Tasks Cited in This Package

| Task | Contribution |
|------|-------------|
| t_f5b70b60 | Systematic refinement; 6 ADRs; OQ resolutions; edge cases |
| t_06e22673 | Eval-readiness initial findings (SEC-3, PERF-3, webhook header) |
| t_b481ed59 | Security eval repair |
| t_1bad7f24 | Performance eval repair |
| t_9fd392cb | Webhook header repair |
| t_7cc74376 | PM approval of repair path |
| t_6f5b3407 | Eval-readiness re-run approval |

---

## 8. Approver and Status

| Field | Value |
|-------|-------|
| **Approver** | pm (profile) — via PM gates t_7cc74376 and t_6f5b3407 |
| **Graduation gate** | t_1d41bc0d (validator) — awaiting this package |
| **Status** | Complete — all mandatory graduation files exist; no unresolved TBD/OQ/blockers |

---

*Decision record complete — BRD-02 gate decisions, OQ resolutions, trade-offs, and ADR references recorded.*
