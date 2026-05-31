# ADR-03-005: Owner Label Mapping — API-Provided with Hardcoded Fallback

**ADR:** ADR-03-005
**Title:** Owner Label Mapping — API-Provided with Hardcoded Fallback
**Status:** Accepted
**Date:** 2026-05-30
**Accepted:** 2026-05-30
**Decider:** architect
**BRD:** BRD-03-client-portal

---

## Context

FR-03-016 and FR-03-017 define a hybrid owner label model: default auto-mapping (Product, Engineering, Review, Quality, Client) with internal project owner override capability. The question is where the mapping table lives and how it is served.

---

## Decision

**Option D — API-provided with hardcoded fallback.**

The client portal BFF fetches the owner label mapping from the BRD-02 backend API at startup or on first render. If the API is unavailable, the portal uses hardcoded default labels.

Phase 1 prototype may use hardcoded mapping only with API-based mapping as the target for Phase 2.

---

## Alternatives Considered

**Option A — Hardcoded in portal implementation (rejected for production):**
- Changing mapping requires code change + redeploy
- Cannot support per-project override without configuration infrastructure

**Option B — Config file per deployment (rejected):**
- Different mappings per environment could cause client confusion
- File must be kept in sync across environments

**Option C — API-provided at runtime (accepted as target):**
- Consistent across environments; changeable without redeploy
- Single source of truth; supports per-project override naturally

---

## Consequences

**Positive:**
- Runtime flexibility: mapping changes without redeploy
- Consistent across environments (dev/staging/prod)
- Supports per-project override without extra infrastructure
- Hardcoded fallback ensures portal remains functional if API is temporarily unavailable

**Negative:**
- Extra API call on startup (mitigated: cached in BFF with 5min TTL)
- Must define API contract for mapping fetch

**Neutral:**
- Phase 1 prototype may use hardcoded only; migration path is clear

---

## Default Mapping Table

| Internal Role/Metadata | Default Client-Facing Label |
|------------------------|-----------------------------|
| `product_manager`, `product_owner`, `pm` | Product |
| `engineering`, `backend`, `frontend`, `developer` | Engineering |
| `reviewer`, `qa`, `quality_assurance`, `testing` | Review |
| `quality`, `architect`, `design` | Quality |
| `client`, `client_stakeholder`, `external` | Client |
| (unmapped roles, fallback) | Product |

The default mapping is the starting point. Internal project owners can override per-item via the override mechanism defined in FR-03-016.

---

## Implementation Notes

- BFF caches owner label mapping with 5-minute TTL
- On cache miss: fetch from BRD-02 `GET /config/owner-mapping`
- On API failure: fall back to hardcoded defaults
- Per-item override stored in BRD-02 backend as item metadata `client_owner_label_override`
- Override takes precedence over default mapping when present

---

## API Contract (BRD-02)

```
GET /config/owner-mapping
Response: {
  "version": "1",
  "mapping": [
    { "internal_role": "engineering", "client_label": "Engineering" },
    { "internal_role": "product_manager", "client_label": "Product" },
    ...
  ],
  "updated_at": "2026-05-30T00:00:00Z"
}
```

---

## Verification

API/UI integration test: Set item owner to `engineering`. Verify client-facing label is "Engineering". Set item `client_owner_label_override` to "Dev Team". Verify client-facing label is "Dev Team". Simulate API failure; verify fallback to hardcoded defaults.