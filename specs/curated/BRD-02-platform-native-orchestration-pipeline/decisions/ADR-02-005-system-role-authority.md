# ADR-02-005: `system` Role Authority Enumeration

**ADR:** ADR-02-005  
**Subject:** Explicit enumeration of permitted mutations for the `system` role claim  
**Profile:** architect  
**Date:** 2026-05-28  
**Status:** Accepted

---

## Problem Statement

FR-02-012 introduces `system` as one of four minimum role claims (`human`, `layer_a`, `layer_b`, `system`). The authorization rules in FR-02-013 and FR-02-014 enumerate capabilities for `human`, `layer_a`, and `layer_b`. No equivalent enumeration exists for `system`.

Risk R-05 (BRD-02) flags this: "Role-scoped internal auth is under-specified and permits gate bypass." Without explicit enumeration, an implementation might permit `system` to perform gate approvals or task completions, violating the governance model.

Open Question 6 notes that `system` role authority for task-level gate approval without a human is also unresolved.

---

## Options Considered

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| A | `system` = infrastructure-only (audit events + FF changes only) | Clear least privilege; prevents scope creep | May need new role for automation use cases |
| B | `system` = automation super-role (Layer A-level for automated contexts) | Enables fraud detection, automated escalation | Overbroad; system can approve gates |
| C | Unspecified (current BRD state) | Simpler initially | Auth gap; risk R-05 is real |

---

## Decision

**Option A** is selected.

**`system` role is explicitly permitted to perform the following mutations:**
1. Emit audit events (any topic) — because audit events are platform-internal records
2. Record feature-flag-change events — because flag transitions are infrastructure-initiated

**`system` role is explicitly FORBIDDEN from performing:**
1. Task creation, update, or deletion
2. Task status transitions (including `done`, `blocked`, `cancelled`)
3. Gate approval or rejection (project-level or task-level)
4. Decomposition proposal creation or approval
5. Handoff evidence submission
6. Scope adjustment of any kind
7. Any mutation that requires a human or Layer A authorization

**Authorization enforcement:** The authorization middleware MUST validate the role claim on every mutation. If `system` is present on any of the above forbidden actions, the request returns `403 Forbidden` and an `auth.mutation.denied` security audit event is appended.

**Future automation use cases** (e.g., automated quality gate evaluation from BRD-08) SHOULD use a named service account with a `layer_a` or `layer_b` role claim rather than the generic `system` role. This prevents `system` from accumulating automation-specific capabilities.

---

## Consequences

**Positive:**
- `system` role is well-bounded; cannot accidentally be used to bypass governance
- Authorization matrix is complete across all four role types
- Risk R-05 is mitigated: explicit forbidden-action rules prevent gate bypass

**Negative:**
- Automation scenarios must use named service accounts with proper role claims
- `system` role cannot be granted special powers in edge cases without an ADR

**Neutral:**
- Standard human/Layer A/Layer B use cases are unaffected
- `system` role is rarely used in normal operation (infrastructure events only)
