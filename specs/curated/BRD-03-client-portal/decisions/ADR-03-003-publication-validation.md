# ADR-03-003: Publication Validation — Shared Schema (UI Guard + API Gate)

**ADR:** ADR-03-003
**Title:** Publication Validation — Shared Schema (UI Guard + API Gate)
**Status:** Accepted
**Date:** 2026-05-30
**Decider:** architect
**BRD:** BRD-03-client-portal
**Review:** BRD-02 alignment confirmed. OQ-03-002 resolved.

---

## Context

FR-03-045 (publication validation) requires business-language summary, owner label, next action, visibility status, and forbidden technical field checks before an item becomes client-visible. FR-03-048 says items failing validation stay hidden.

The question: where is validation enforced? UI only (bypassable), API only (higher latency), or both (defense in depth)?

---

## Decision

**Option B — UI guard + API gate with shared validation schema.**

The frontend validates submission before sending (client-side, fast feedback). The backend also validates every publication request (server-side, authoritative). Both use the same shared validation schema so discrepancies between client and server validation rules cannot occur.

A discrepancy between client and server validation rules is treated as a security defect (NFR-03-013 class).

---

## Alternatives Considered

**Option A — API gate only (rejected for Phase 1):**
- User clicks publish, gets error after round-trip; higher friction
- Accepted for Phase 1 prototype; not acceptable for production

**Option C — UI guard only (rejected):**
- A misbehaving client can bypass client-side validation
- Forbidden content could reach backend logs or error messages
- Does not meet NFR-03-014 (business-language safety) bar

---

## Consequences

**Positive:**
- Defense in depth: server is authoritative even if client is bypassed
- Client-side validation provides fast feedback UX
- Shared schema ensures consistency between client and server

**Negative:**
- Validation logic must be maintained in two places (UI + API)
- Shared schema means a change to validation rules requires updating both

**Neutral:**
- The shared validation schema must be defined before implementation
- Schema should be versioned to prevent silent divergence

---

## Forbidden Technical Fields List

*Confirmed against BRD-02 platform conventions. BRD-02 review task t_6a231486.*

### Content Patterns (forbidden in any text field value)

| Pattern | Examples | Rationale |
|---------|----------|-----------|
| Stack traces | Lines starting with `at `, `Traceback`, `Exception`, `Error:`, `panic:` | Internal debug information — not client-safe |
| Internal agent IDs | `agent-<hex>` (e.g., `agent-a1b2c3`), `agentId`, `assignedAgent`, `executedBy` as field values | Internal accountability structure not visible to clients |
| Branch names | `refs/heads/*`, `feature/*`, `fix/*`, `hotfix/*`, `release/*` | Internal version control terminology |
| Commit SHAs | 40-character hex string matching `[a-f0-9]{40}` | Internal VCS identifiers |
| File paths | Paths containing `/src/`, `/internal/`, `/backend/`, `/agent/`, `/pkg/`, `/cmd/` | Internal code structure |
| Infrastructure jargon | `docker`, `kubernetes`, `pod`, `deployment`, `service mesh`, `container`, `namespace`, `kubeconfig`, `helm`, `ingress`, `sidecar` | Infrastructure-level terminology not relevant to business users |
| Structured log output | Lines starting with `[DEBUG]`, `[INFO]`, `[WARN]`, `[ERROR]`, `[TRACE]` | Internal observability output |
| Go runtime panic output | `panic:`, `goroutine`, `runtime.gopanic` | Internal runtime diagnostics |
| Internal role identifiers | String values `layer_a`, `layer_b` in any field | Internal agent tier designation |

### Envelope Fields (forbidden in client-visible SSE event payloads)

The following canonical event envelope fields MUST be stripped by the BFF before SSE fanout to client-visible subscriptions:

| Envelope Field | Rationale |
|----------------|------------|
| `actorId` | Internal identity — clients see owner labels, not raw agent IDs |
| `actorRole` | Internal role claim (`layer_a`, `layer_b`, `human`) — not client-relevant |
| `eventId` | Internal sequencing identifier — UUID v7 used for ordering only |
| `schemaVersion` | Internal version marker — not relevant to business content |
| `parentTaskId` | Internal task graph structure — exposes governance hierarchy |
| `gateId` | Internal governance identifier — not meaningful to clients |
| `layer` (Task field) | Internal agent tier designation — exposed in `layer` field of Task schema |

### Metadata Object Constraints

`Task.metadata` is defined in `contracts/openapi.yaml` as `additionalProperties: true`. This open schema allows any key-value pair to appear in metadata.

For client-visible publication, the following semantic constraints apply:
- Metadata field names must not be internal identifiers (e.g., `blockedReason`, `owner_override`, `internal_tags`, `execution_agent`)
- Metadata field values must not contain forbidden content patterns
- The BFF MUST NOT forward raw metadata to client-visible responses unless field names are on the client-safe allowlist

### Additional Scan Targets

The following free-text fields MUST also be scanned for forbidden patterns (in addition to dedicated fields):
- `blockedReason` — may contain agent IDs, branch names, or implementation details in natural language
- `body` (task description) — user-authored content that may inadvertently include technical references
- `summary` field in handoff evidence — must be reviewed before publication

### Removed Item

| Removed | Reason |
|---------|--------|
| "Raw API response payloads" | Replaced with explicit envelope field exclusions above; vague as written |

---

## Implementation Notes

- Shared validation schema defined in `contracts/publication-validation.yaml`
- Schema version embedded in both UI bundle and BFF build
- On validation failure: API returns `400 Bad Request` with structured error `{ field, reason, code }`
- UI shows field-level error messages derived from API response
- Audit log: `client_portal.publication_validation.failed` with reason category (not raw content)
- BFF SSE fanout layer strips envelope metadata fields (actorId, actorRole, eventId, schemaVersion, parentTaskId, gateId, layer) before client subscription delivery

---

## Verification

Publication validation test: Submit item with missing business-language summary → 400 returned, item stays hidden. Submit item with forbidden technical content (e.g., stack trace in description) → 400 returned, item stays hidden. Submit item with all required fields and no forbidden content → 201 returned, item becomes visible.

SSE envelope strip test: Subscribe to project SSE stream; verify envelope metadata fields (actorId, actorRole, eventId, schemaVersion, parentTaskId, gateId, layer) are absent from all event payloads received by client.

Metadata scan test: Submit item with forbidden pattern in metadata object → 400 returned, item stays hidden.

---

## Cross-BRD Contract Alignment Notes

BRD-02 `contracts/openapi.yaml` Task schema exposes `layer` (enum: A/B) and `metadata` (additionalProperties: true). These are correctly modeled as internal-only fields. The BFF access filtering layer (BRD-03 component 6) is responsible for stripping `layer` from client-visible responses. The publication validation service is responsible for scanning `metadata` values.

Flag for ops task t_447cbbee: `layer` field and open `metadata` schema should be explicitly addressed in the contract parity work to ensure the BFF has documented behavior for both.

---

## Open Questions Resolved

| OQ | Resolution |
|----|------------|
| OQ-03-002 | Forbidden fields list defined above — aligns with BRD-02 internal field definitions; no conflicts found |