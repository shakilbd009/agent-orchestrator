# ADR-03-001: Client Portal BFF Architecture

**ADR:** ADR-03-001
**Title:** Client Portal Backend-for-Frontend (BFF) Architecture
**Status:** Accepted
**Date:** 2026-05-30
**Accepted:** 2026-05-30
**Decider:** architect
**BRD:** BRD-03-client-portal

---

## Context

FR-03-006 (project access filtering) and NFR-03-013 (cross-project filtering failures fail closed as security defects) require that every portal read, board view, search result, approval item, comment, risk, milestone, and SSE-driven update is filtered to projects the current client principal may access. The architectural question is whether the SvelteKit frontend fetches directly from BRD-02 backend APIs (browser → BRD-02) or through a BFF aggregation layer (browser → BFF → BRD-02).

BRD-01 (app shell) does not include a BFF. Phase 1 will be a direct-fetch prototype.

---

## Decision

**Option B — BFF aggregation layer for production.**

SvelteKit pages call a client-portal BFF (Go/Echo) which enforces project access filtering and aggregates data from BRD-02 backend APIs. The BFF is the authoritative access control boundary. The access token is never exposed directly to the browser — it lives in BFF session context.

**Phase 1 (prototype):** Direct fetch from BRD-02 APIs is acceptable with explicit documentation that BRD-02 must guarantee API-layer project filtering for client principal types.

**Phase 2 (production):** BFF becomes the access control boundary with centralized project filtering enforcement.

---

## Alternatives Considered

**Option A — Client-side direct fetch (rejected for production):**
- Access token exposed to browser; harder to enforce row-level access filtering
- SSE subscriptions from browsers to BRD-02 event streams complicate security model
- Every BRD-02 API endpoint must correctly enforce project-level filtering for client principal types — single point of failure

**Option C — Hybrid BFF (write only) (rejected):**
- Split responsibilities create confusion about which layer enforces what
- Access control consistency harder to reason about

---

## Consequences

**Positive:**
- Centralized access control boundary; single enforcement point for project filtering
- BFF can inspect and censor responses before sending to client
- BFF manages SSE connections to BRD-02 (one connection per project vs. per-browser)
- Access token never exposed to browser
- BFF can implement project-level caching without exposing internal state

**Negative:**
- Additional latency hop (browser → BFF → BRD-02 vs. browser → BRD-02)
- BFF is another service to deploy, monitor, and maintain
- BFF must be deployed before client portal is operational in Phase 2

**Neutral:**
- Phase 1 prototype uses direct fetch; Phase 2 migration to BFF requires interface audit

---

## Implementation Notes

- BFF uses same Echo framework as BRD-02 backend (per AGENTS.md pin)
- BFF session auth integrates with BRD-01 app shell session contract
- BFF implements health endpoint for readiness probes
- BFF SSE multiplexes: one EventSource to BRD-02 per project, clients subscribe to BFF
- Forbidden technical fields are stripped by BFF before response reaches client

---

## Verification

Integration test: A client principal with access to projects A, B, C attempts to access project D (unauthorized). BFF returns empty result (not 403, which leaks project existence). `client_portal_access_denied_total` counter increments. Access denied event is logged.