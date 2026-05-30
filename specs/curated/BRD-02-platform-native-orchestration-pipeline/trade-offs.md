# BRD-02: Trade-Off Analysis

**Profile:** architect  
**Date:** 2026-05-28  
**BRD:** specs/orchestration/brd-02-platform-native-orchestration-pipeline.md

---

## Overview

BRD-02 defines platform-native project-scoped orchestration. Six implicit decisions were identified that require explicit adjudication via ADR. This document captures each trade-off space.

---

## Trade-Off 1: Event Schema Version Naming Convention

**Implicit decision:** Format of `schemaVersion` field in canonical event envelope.

**Decision space:**
- `v1alpha` during implementation → `v1` at GA (recommended by this trade-off)
- `v1` from day 1
- No schema version field

**Trade-offs:**

| Option | Pros | Cons |
|---|---|---|
| `v1alpha` during impl, `v1` at GA | Consumers know stability level; clear GA signal | Requires version bump at GA milestone |
| `v1` from day 1 | Simpler; no transitional version | No pre-GA signal; consumers assume stability |
| No schema version | Simplest envelope | No stability signal; breaks FR-02-032 |

**Selected:** `v1alpha` during implementation; `v1` at GA.

**Rationale:** The Phase 0 placeholder used `v1alpha`. Starting `v1` immediately would be a false claim of stability before the contract is frozen. GA marking is a meaningful signal to consumers.

---

## Trade-Off 2: Transaction Boundary for Current-State + Audit Append

**Implicit decision:** Whether task/gate/handoff mutations must atomically commit both current-state and audit event in one DB transaction.

**Decision space:**
- Always one DB transaction (required)
- Best-effort with async consistency check
- Separate transactions allowed (no atomicity requirement)

**Trade-offs:**

| Option | Pros | Cons |
|---|---|---|
| Always one DB transaction | Strongest consistency; no drift between state and audit | Requires DB engine with transaction support; SQLite in WAL mode works fine |
| Best-effort + async check | Simpler to implement if DB lacks transactions | Drift can occur; detected but not prevented |
| Separate transactions | Simplest implementation | Risk of audit gap on rollback; inconsistent historical record |

**Selected:** Atomic commit as default requirement. Diagnostic consistency check on `/ready` if DB engine lacks row-level transactions.

**Rationale:** FR-02-002 says "where possible." The "where possible" qualifier is the mitigation to the drift risk. Making atomicity the default (required) addresses the risk directly. The qualifier handles edge cases where a non-transactional engine might be used.

---

## Trade-Off 3: Webhook Signing — Mandatory vs. Should-Have

**Implicit decision:** Whether webhook signing is a security requirement or a nice-to-have.

**Decision space:**
- Mandatory for all registered webhooks
- Mandatory for external URLs; localhost exempt
- Should-have (per current BRD text)

**Trade-offs:**

| Option | Pros | Cons |
|---|---|---|
| All webhooks signed | Strongest security posture; no exceptions to reason about | Key distribution infra needed; friction for first internal use |
| External URLs mandatory; localhost exempt | Proportionate security; internal dev friction reduced | Exemption code paths; need to validate URL schemes |
| Should-have | Zero friction | Security posture ambiguous; unresolved open question |

**Selected:** Mandatory signing except localhost/127.0.0.1 during development. HMAC-SHA256 signature header required. Production internal URLs (e.g., `http://backend:8080`) also require signing unless explicitly allow-listed for that environment.

**Rationale:** External webhook delivery is a vector for injection attacks if unsigned. Internal service-to-service calls are also a vector within a private network. The only safe exemption is explicit `localhost` development. This is the minimum bar for security.

---

## Trade-Off 4: `/ready` Degradation Semantics

**Implicit decision:** What happens when webhook queue is unavailable but other subsystems are healthy.

**Decision space:**
- Hard fail: return 503 when webhook queue is unavailable
- Soft degrade: return 200 with degraded body
- Webhook queue not part of readiness

**Trade-offs:**

| Option | Ops Impact | Consumer Impact |
|---|---|---|
| Hard fail (503) | Clear alert signal; ops knows webhook infra need attention | All reads/boards continue; only webhook delivery fails |
| Soft degrade (200 + warning) | Less alarming; might be ignored | Consumer assumes platform is healthy; webhook delivery silently failing |
| Queue not in readiness | Simplest; no ops change needed | Consumer cannot distinguish degraded from healthy |

**Selected:** Hard fail (503) with specific failing subsystem in body. Two-tier readiness: minimum tier (DB + state + audit) vs. full tier (+ webhook queue). Webhook receiver (not queue) failures do NOT make platform unready.

**Rationale:** NFR-02-012 says "should remain usable" when webhook consumers are unavailable. This means webhook failures should not cascade. But webhook queue unavailability IS a real infrastructure degradation that ops needs to know about. A 503 with clear JSON messaging is the right signal.

---

## Trade-Off 5: `system` Role Authority

**Implicit decision:** What specific mutations the `system` role is permitted to perform.

**Decision space:**
- `system` can do anything a human/Layer A can do in automation contexts
- `system` is infrastructure-only (audit events + feature flag changes only)
- `system` authority is unspecified (current state)

**Trade-offs:**

| Option | Pros | Cons |
|---|---|---|
| `system` = automation super-role | Enables full automation scenarios | Overbroad; `system` could approve gates or create tasks without human oversight |
| `system` = infrastructure-only | Clear least privilege; aligns with "system as infrastructure actor" concept | May need new role for future automation scenarios |
| Unspecified | Simpler initially | Auth gaps; risk R-05 is about this unspecified gap |

**Selected:** `system` is infrastructure-only: permitted to emit audit events and update feature-flag-change events. All other actions require `human`, `layer_a`, or `layer_b`. Explicit enumeration prevents scope creep.

**Rationale:** FR-02-012 introduces `system` as a minimum role claim without defining its authority. This is the gap Risk R-05 calls out. The infrastructure-only interpretation is the safest default. Future automation scenarios can be addressed with a new role, not an overbroad `system` role.

---

## Trade-Off 6: CORS Policy for SSE Endpoint

**Implicit decision:** CORS headers for `GET /projects/{projectId}/events/stream` (browser-facing SSE).

**Decision space:**
- No CORS headers (same-origin only)
- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Origin: <configurable per-project>`

**Trade-offs:**

| Option | Pros | Cons |
|---|---|---|
| Same-origin only | Simplest; no CORS config needed | Dashboard must be same-origin; fragile in dev where frontend and backend on different ports |
| Wildcard `*` | Simplest cross-origin config | Too open; webhook URL validation may conflict with signing |
| Per-project configurable origin | Correct security; aligns with project-scoped model | Requires project-level origin allowlist; more config |

**Selected:** Per-project configurable allowed-origin list for SSE. Wildcard `*` only acceptable for `localhost` development. SSE must implement `OPTIONS` preflight handling. Origins are validated against project-level allowlist.

**Rationale:** FR-02-023 says SSE is for "first-party dashboard clients." Dashboard is browser-based (SvelteKit by BRD-03). CORS is required. The project-scoped model makes per-project origin allowlist the natural fit. Wildcard is a security risk for production.

---

*Trade-off analysis complete. ADRs for items 1, 2, 3, 4, 5, 6 are in the decisions/ directory.*