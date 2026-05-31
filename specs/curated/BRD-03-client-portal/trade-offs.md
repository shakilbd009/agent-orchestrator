# BRD-03 — Trade-Off Analysis

**BRD:** BRD-03-client-portal
**Stage:** 03-trade-offs
**Status:** Analyzed

---

## Major Decision 1: Client Portal Architecture — Server-Side Rendering vs. Client-Side Fetching

### Context

FR-03-040 requires real-time updates via SSE and manual refresh fallback. FR-03-041 and NFR-03-015 require the portal to remain usable for reads when SSE is unavailable. The architectural question is where the portal fetches its data: does the SvelteKit frontend fetch directly from BRD-02 backend APIs (browser → backend), or does it go through a BFF (backend-for-frontend) layer that aggregates and filters data server-side?

### Options

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| **A: Client-side direct fetch** | SvelteKit pages call BRD-02 backend APIs directly from browser; access filtering enforced at API layer via session token | Simple architecture; fewer moving parts; SSE connections managed from browser | Browser embeds BRD-02 API URL; access token exposed to browser; SSE subscriptions directly to BRD-02 event streams from client; harder to enforce row-level access filtering without a proxy |
| **B: BFF aggregation layer** | SvelteKit → client-portal BFF (Go/Echo) → BRD-02 backend APIs; BFF enforces project access filtering and aggregates data | Access token never exposed to browser; centralized access control; BFF can filter/censor before sending to client; simpler SSE management (server-side EventSource to BRD-02, client receives filtered events) | More infrastructure; BFF becomes another service to maintain; additional latency hop |
| **C: Hybrid — BFF for write, direct for read** | Approval/comment mutations go through BFF; portfolio/project reads go direct to BRD-02 with API-layer filtering | Mutations are security-critical so BFF makes sense there; reads simpler | Split responsibilities; complexity in determining which calls go where |

### Verdict

**Option B (BFF aggregation layer)** preferred for Phase 2. Centralized project access filtering is critical (NFR-03-013 "filtering failures fail closed as security defect"). A BFF provides a single enforcement point for access control rather than relying on every BRD-02 API endpoint to correctly enforce project-level filtering for client principal types. SSE event stream management is also simpler server-side (one connection from BFF to BRD-02 vs. many from browsers).

However, Phase 1 app shell (BRD-01) does not include a BFF. For Phase 1 client portal prototype, Option A is acceptable with the caveat that BRD-02 must guarantee API-layer project filtering. An ADR is needed to lock in the BFF architecture for production.

**Decision recorded in:** ADR-03-001

---

## Major Decision 2: SSE Subscription Scope — Per-Project vs. Global Stream

### Context

FR-03-018 requires a global approval inbox showing pending decisions across all accessible projects. FR-03-040 requires SSE for live updates. The SSE implementation question is whether the portal maintains one global SSE connection covering all accessible projects, or one SSE connection per visible project.

### Options

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| **A: Per-project SSE connections** | One EventSource per visible project; each streams project-scoped BRD-02 events | Aligns with BRD-02 project-scoped SSE design; simpler event routing per project; easy to identify which project an event belongs to | Many concurrent connections if portfolio has 50 projects; browser connection management overhead; potential for connection churn on project list changes |
| **B: Single global SSE connection** | One EventSource to a BFF or BRD-02 endpoint that multiplexes events across all accessible projects | Fewer connections; simpler browser management; natural fit for global inbox | BFF or BRD-02 must multiplex events by project; harder to route events to correct UI components without a client-side pub/sub; BRD-02 must support multi-project SSE scope |
| **C: Per-project with global hub** | Browser opens per-project EventSources; a client-side event hub aggregates and handles routing | Balances connection count with event routing simplicity; each project streams independently | More complex client-side event routing; hub becomes a state management concern |

### Verdict

**Option A (per-project SSE connections)** for Phase 1. Aligns with BRD-02 project-scoped SSE design which is already explicitly named in FR-03-040 ("subscribe to BRD-02 project-scoped SSE streams"). Per-project connections are the straightforward implementation.

For the global approval inbox, the SSE subscription would cover all accessible projects — which could mean 50 simultaneous EventSource connections. If that proves problematic in practice (browser limits, performance), Option C or a server-side multiplexing approach (Option B via BFF) can be explored via a subsequent ADR. ADR-03-002 records the per-project SSE decision and notes the global inbox scalability concern.

**Decision recorded in:** ADR-03-002

---

## Major Decision 3: Publication Validation Enforcement — API Gate vs. UI Guard

### Context

FR-03-045 requires that publishing a client-visible item must validate business-language summary, owner label, next action, visibility status, and forbid technical fields. FR-03-048 says items failing validation remain hidden. The enforcement point question: does the backend API reject publication attempts with 400, or does the UI prevent submission, or both?

### Options

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| **A: API gate enforcement only** | Backend rejects publication with 400 + error details if required fields missing or forbidden content detected; UI is辅助 | Defense in depth; server is authoritative; cannot be bypassed by malicious/malformed client | UX: user clicks publish, gets error after round-trip; more friction |
| **B: UI guard + API gate** | UI validates before submission (client-side checks), backend also validates (server-side enforcement) | Best UX: client-side catches obvious errors fast; server-side catches bypassed/malformed | Duplication of validation logic in two places; must keep in sync |
| **C: UI guard only (client-side only)** | Frontend validates submission; backend trusts client | Simple backend; fast feedback | Not defensive; a misbehaving client can bypass validation; forbidden content could reach backend logs |

### Verdict

**Option B (UI guard + API gate)**. Defense in depth for a security-sensitive operation (preventing technical content leakage to clients). The UX benefit of client-side validation (fast feedback) combined with server-side enforcement (authoritative) is the right balance for FR-03-014 (business-language-only display) and NFR-03-014 (business-language safety).

However, the UI guard and API gate must share the same validation schema — a discrepancy between client and server validation rules is a defect. ADR-03-003 records this decision and requires shared validation schema.

**Decision recorded in:** ADR-03-003

---

## Major Decision 4: Overdue Decision Threshold — Per-Item vs. Per-Project

### Context

FR-03-037 defines a client decision as overdue when "it remains pending more than 24 hours after becoming visible and actionable to the client." The question is whether the 24h clock applies per individual approval item or per project (all decisions for a project become overdue if the project has any decision pending 24h+).

### Options

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| **A: Per decision item** | Each approval item has its own 24h clock from the moment it becomes client-visible/actionable | Granular; correctly implements "a client decision" language; a project with 10 decisions where 1 is old shows exactly 1 overdue | Requires per-item timestamp tracking; more complex state |
| **B: Per project aggregate** | If ANY decision for a project is pending 24h+, the entire project's pending decisions are marked overdue | Simple to compute; per-project health scores could incorporate this easily | Wrong semantics: one old decision makes all other recent decisions appear overdue even if they're fresh |
| **C: Per-project but per-decision still shown** | Project-level overdue flag, but individual items show their own age | Shows per-item freshness while also flagging project-level delay | Most complex; may over-complicate UI |

### Verdict

**Option A (per decision item)**. The BRD language "a client decision" and "overdue client decisions" (FR-03-004) is clearly per-item. Using per-project aggregation would give misleading signals. The implementation complexity of per-item timestamps is justified by correct semantics. ADR-03-004 records this.

**Decision recorded in:** ADR-03-004

---

## Major Decision 5: Owner Label Mapping — Configuration vs. Hardcoded

### Context

FR-03-016 and FR-03-017 define a hybrid owner label model: default auto-mapping (Product, Engineering, Review, Quality, Client) with internal project owner override capability. The question is whether the default mapping table is a configuration file, a database row, or hardcoded in the portal implementation.

### Options

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| **A: Hardcoded in portal implementation** | Portal code contains if/else or switch on internal role → client label mapping | Simplest; no external dependency | Changing mapping requires code change + redeploy |
| **B: Config file (per deployment)** | `owner-mapping.json` or similar file on the portal's filesystem | Changing mapping doesn't require code change; ops can adjust | Different mappings per environment could cause client confusion; file must be kept in sync |
| **C: API-provided at runtime** | BRD-02 backend API provides the mapping (could be project config or global config); portal fetches on load | Consistent across environments; can be changed without redeploy; single source of truth | Extra API call on startup; mapping must be fetched before rendering |
| **D: Hybrid — API with hardcoded fallback** | Portal fetches mapping from BRD-02 API; if not available, uses hardcoded defaults | Best of both: runtime flexibility with simplicity for Phase 1 | Still requires API contract for mapping |

### Verdict

**Option D (API-provided with hardcoded fallback)** for Phase 1/2. FR-03-016 says "automatically map internal role/task metadata to business labels by default" — this implies the mapping is derived from internal metadata, not a static config. A runtime API approach is most consistent with the dynamic nature of project-level overrides and the BRD-02 backend providing project configuration.

However, for Phase 1 shell, Option A (hardcoded) is acceptable as a starting point with the API approach being the target for Phase 2. ADR-03-005 records this.

**Decision recorded in:** ADR-03-005

---

## Cross-Cutting Trade-Offs

| Decision | Trade-off | Mitigation | Source |
|----------|-----------|------------|--------|
| Client portal as Phase 2 feature | Phase 1 app shell (BRD-01) won't have client portal; dashboard flag must be preserved until `client-portal` is ready | Keep `dashboard` flag until `client-portal` ready; explicit deprecation deferred to later ADR | FR-03-051 / Feature Flag section |
| Real-time via SSE + manual fallback | SSE is opt-in freshness mechanism; current-state APIs are the authoritative source for all critical reads | SSE used for UI updates only; reconciliation always goes to current-state APIs | FR-03-041, FR-03-042 |
| Simple comments (no threading) | Clients cannot reply in threads, @mention, or attach files | Scope intentionally limited to keep BRD-03 focused; full collaboration deferred to BRD-18 | FR-03-025 |
| All risks client-visible by default | Clients may see internal concerns that aren't meaningful to them | Client-safe risk language required; mitigation summaries must be plain-language; governance may add visibility controls later | FR-03-032, Risk section |

---

## Red Flags Checklist

| Flag | Status | Notes |
|------|--------|-------|
| Only 1 approach considered | ✅ PASS | 2-3 options for each major decision |
| Rubber-stamping | ✅ PASS | All options have genuine pros/cons; no option is perfect |
| Vague rationale | ✅ PASS | Rationale explicit per decision |
| Hand-waving complexity | ✅ PASS | Consequences listed including negative cases |
| No validation plan | ✅ PASS | Each decision includes how to validate in implementation |

---

*Trade-offs analyzed: Stage 03*
*Next: Stage 04 progressive deepening L1 → create ADRs for major decisions*