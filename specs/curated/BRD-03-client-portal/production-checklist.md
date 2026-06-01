# BRD-03 Client Portal — Production Readiness Checklist

**BRD:** BRD-03-client-portal
**Generated:** 2026-05-31
**Source:** curated artifacts (brd.md, decision-record.md, validator-findings.md, implementation-readiness.md, ADR-03-001 through ADR-03-005, openapi.yaml, events.md, feature-flags.md, eval contracts)
**Gate:** validator t_8c1b01df PASS → PM wrapper t_5986b725 handoff

---

## Infrastructure

- [ ] **Go/Echo BFF scaffold** — Create `backend/` BFF service skeleton with Echo routing, dependency injection, and config management. Reference `ADR-03-001` (Option B — BFF aggregation layer for Phase 2; Phase 1 direct fetch prototype).
- [ ] **BRD-02 API client integration** — BFF wraps BRD-02 project/task/approval/comment/milestone read APIs. Backend provides `GET /config/owner-mapping` endpoint per `ADR-03-005`.
- [ ] **Phase 1 prototype flag** — Phase 1 SvelteKit frontend fetches directly from BRD-02 APIs (browser → BRD-02) without BFF. BFF is added in Phase 2.
- [ ] **Server env var** — `FF_ENABLE_CLIENT_PORTAL` env var exists and defaults to `false`. BFF checks flag before serving any `/client-portal/*` route.
- [ ] **Browser env var** — `VITE_FF_ENABLE_CLIENT_PORTAL` env var exists and defaults to `false`. SvelteKit reads at build time; portal UI is conditionally rendered.
- [ ] **Project access filtering enforcement** — Every BFF API call extracts `principal` from session and passes `principal` filter to BRD-02 queries. Unauthorized project access returns empty result (NOT 403), per `implementation-readiness.md` (Access Filtering section).
- [ ] **Publication validation schema** — BFF enforces shared schema (from `ADR-03-003` and `implementation-readiness.md`): required fields (`business_summary`, `client_owner_label`, `next_action`, `visibility_status`) and forbidden patterns (stack traces, agent IDs, branch names, commit SHAs, file paths, infra terms, log lines).
- [ ] **Owner label mapping cache** — BFF caches `GET /config/owner-mapping` with 5-minute TTL; hardcoded fallback per `ADR-03-005`.

---

## Security

- [ ] **Access boundary — fail closed** — `client_portal_access_denied_total` metric emitted on any denied project/item access attempt. Cross-project filtering failures are treated as security defects (NFR-03-013).
- [ ] **Publication validation** — Items missing required fields or containing forbidden technical content MUST fail validation and remain hidden (FR-03-048). Test: submit item with `agent-a1b2c3` pattern → validation fails, item not visible in portal.
- [ ] **Forbidden content stripping** — BFF strips from ALL client-facing responses: stack traces (`^\s+at\s+` or `Traceback`), internal agent IDs (`agent-[0-9a-f]{6,}`), branch names (`refs/heads/`, `feature/`, `fix/`), commit SHAs (40-char hex), file paths (`/src/`, `/internal/`, `/backend/`, `/agent/`), infra terms (`docker`, `kubernetes`, `pod`, `deployment`, `service mesh`), raw log lines (`^[DEBUG]`, `^[INFO]`, `^[WARN]`, `^[ERROR]`).
- [ ] **Comment audit metadata** — Comment edit/delete audit events record create/edit/delete metadata, actor, timestamp, project, related item, and action. Deleted/edited body text is NOT retained (FR-03-029). Validation: audit store does not contain previous-deleted comment body.
- [ ] **Approval audit metadata** — Every approval outcome records: actor identity/display name, actor role or client principal type, project ID, related item ID, outcome, timestamp, and comment metadata where applicable (FR-03-024).
- [ ] **SSE envelope metadata stripping** — BFF strips `actorId`, `actorRole`, `eventId`, `schemaVersion`, `parentTaskId`, `gateId`, `layer` from all SSE event payloads before client subscription delivery (ADR-03-003, events.md line 288).
- [ ] **No BRD-03 notifications** — Portal shows visible indicators only; no email, push, Slack, webhooks, or in-app notification center delivery. Scope enforced.

---

## Integrations / Contracts

- [ ] **OpenAPI contract** — All `/client-portal/*` endpoints defined in `contracts/openapi.yaml` (lines 444–628). POST `/client-portal/approvals/{approvalId}/decide` defined (lines 541–582) with `ApprovalDecisionRequest` and `ApprovalDecisionResponse` schemas (lines 910–941).
- [ ] **Event contract** — All client portal events defined in `contracts/events.md` (lines 145–279): approval events, publication events, comment events, access/portal health events. SSE envelope strip requirement documented (line 288).
- [ ] **BRD-02 dependency** — Project/task/gate/current-state APIs, SSE event stream, and owner mapping API required from BRD-02. OQ-03-001 (SSE envelope schema) is a cross-BRD deferral — BRD-02 defines the actual envelope format; BRD-03 requires minimum fields (`project_id`, `event_type`, `item_id`, `timestamp`).
- [ ] **BRD-16 dependency** — Risk data API assumed as `risks` endpoint in project detail. BRD-03 display contract defined; BRD-16 must deliver the API.
- [ ] **BRD-17 dependency** — Platform auth must provide `principal` context to BFF for project access filtering.
- [ ] **Feature flag registry** — `client-portal` flag registered in `specs/feature-flags.md` (line 57) with `default: false`, Phase 1. Existing `dashboard` flag preserved until later ADR deprecation decision.

---

## Monitoring / Observability

- [ ] **Metrics emitted** — All 20 metrics from `brd.md` Observability section (lines 137–159):
  - `client_portal_portfolio_view_total`
  - `client_portal_project_view_total`
  - `client_portal_portfolio_load_duration_ms`
  - `client_portal_project_load_duration_ms`
  - `client_portal_pending_approvals_current`
  - `client_portal_overdue_decisions_current`
  - `client_portal_oldest_pending_decision_age_ms`
  - `client_portal_decision_turnaround_ms`
  - `client_portal_decision_outcome_total` (labeled by outcome)
  - `client_portal_need_more_information_current`
  - `client_portal_requested_changes_current`
  - `client_portal_blocked_projects_current`
  - `client_portal_at_risk_projects_current`
  - `client_portal_publication_validation_failed_total`
  - `client_portal_comment_created_total`
  - `client_portal_comment_edited_total`
  - `client_portal_comment_deleted_total`
  - `client_portal_sse_disconnect_total`
  - `client_portal_manual_refresh_total`
  - `client_portal_access_denied_total`
- [ ] **Log events emitted** — All 16 log events from `brd.md` (lines 161–177) and `events.md` (lines 145–279): `client_portal.portfolio.viewed`, `client_portal.project.viewed`, `client_portal.approval.submitted`, `client_portal.approval.need_more_information`, `client_portal.comment.created/edited/deleted`, `client_portal.item.published/unpublished`, `client_portal.publication_validation.failed`, `client_portal.access.denied`, `client_portal.sse.connected/disconnected`, `client_portal.read_only_mode.entered`, `client_portal.reads.unavailable`.
- [ ] **Health endpoint** — `GET /ready` reports degraded/unavailable when current-state project reads, approval reads, comment reads, or publication state reads are unavailable. Distinguishes read availability from write/submission availability for read-only degraded mode.
- [ ] **Observability test contract** — `evals/integration/brd-03-client-portal.md` and `evals/perf/brd-03-client-portal.md` define metric and log event verification criteria.

---

## Testing / Evals

- [ ] **Unit tests** — `evals/unit/brd-03-client-portal.md`: completion percentage calculation (`done / (todo + in_progress + blocked + done)` excluding cancelled/proposed), owner label mapping with override precedence, overdue calculation (per item, 24h threshold).
- [ ] **Integration tests** — `evals/integration/brd-03-client-portal.md`: access-control (positive + negative), portfolio health summary, project detail view, approval outcomes, comment create/edit/delete, publication validation, SSE live updates, manual refresh fallback, read-only degraded mode.
- [ ] **E2E tests** — `evals/e2e/brd-03-client-portal.md`: full client journey from portfolio landing → project detail → approval action → comment → search/filter.
- [ ] **Security tests** — `evals/security/brd-03-client-portal.md`: cross-project access boundary (negative test — accessing unauthorized project returns empty), forbidden content stripping (publication validation fails on technical content), comment audit metadata excludes body text.
- [ ] **Performance tests** — `evals/perf/brd-03-client-portal.md`: portfolio landing ≤ 5s at 50-project scale (NFR-03-004), project detail ≤ 2s at 10,000-task scale (NFR-03-005), SSE freshness ≤ 2s (NFR-03-006).
- [ ] **45 AC coverage** — All acceptance criteria from `brd.md` (lines 207–253) mapped to eval contracts. Key coverage: AC-03-001 (flag-gated routes), AC-03-002/003 (access filtering), AC-03-011 (no raw technical content), AC-03-013 (publication validation), AC-03-019/020/021/022 (approval outcomes with required comments), AC-03-037 (SSE 2s freshness), AC-03-039 (read-only degraded mode), AC-03-042 (WCAG 2.1 AA accessibility).

---

## Rollback

- [ ] **Feature flag rollback** — Set `FF_ENABLE_CLIENT_PORTAL=false` (server) and `VITE_FF_ENABLE_CLIENT_PORTAL=false` (browser rebuild) to disable all client portal routes and API behavior. No data migration required; portal is purely read/write on existing BRD-02 data.
- [ ] **Rollback verification** — With flag false, `GET /client-portal/*` returns empty responses or is routed to fallback; SSE connections closed; portal UI not rendered.
- [ ] **Deployment rollback** — Revert BFF and frontend builds to previous stable version. Portal data (comments, approvals) remains in BRD-02 backend; no portal-specific storage to restore.

---

## Feature Flag Release

- [ ] **`client-portal` flag defaults to `false`** — Flag is registered in `specs/feature-flags.md` with `default: false`, `current: false`, Phase 1. It MUST remain `false` through implementation and QA gates.
- [ ] **Do NOT enable before gates pass** — `FF_ENABLE_CLIENT_PORTAL` and `VITE_FF_ENABLE_CLIENT_PORTAL` must remain `false` until: (a) implementation is complete, (b) all eval contracts pass (unit, integration, e2e, security, perf), (c) QA sign-off on access control, content safety, and SSE reliability.
- [ ] **Dashboard flag preservation** — Existing `dashboard` flag in `specs/feature-flags.md` is preserved; not replaced by `client-portal`. Deprecation decision deferred to a later ADR per `brd.md` (FR-03-098).
- [ ] **Flag transition documentation** — When flag transitions from `false` → `alpha` → `beta` → `true`, record the transition in `feature-flags.md` `Current` column with phase of activation.

---

## Cross-BRD Deferrals

> The following items are **NOT BRD-03 blockers**. They are documented cross-BRD dependencies that do not prevent implementation dispatch.

- [ ] **OQ-03-001 — SSE event envelope schema** — BRD-03 artifacts define minimum required fields (`project_id`, `event_type`, `item_id`, `timestamp`); BRD-02 must define the actual SSE event envelope format. Phase 1 prototype uses hardcoded envelope parsing; Phase 2 BFF SSE multiplexer is designed to handle the envelope once BRD-02 delivers it. **Owner: BRD-02**
- [ ] **OQ-03-002 — Forbidden technical fields list review** — ADR-03-003 provides initial exhaustive list; BRD-02 API payload review may identify additional patterns. Implementation proceeds with ADR-03-003 list; BRD-02 review feedback may generate a follow-up ADR patch. **Owner: BRD-02**

---

## Handoff

**Changed files:**
- `specs/curated/BRD-03-client-portal/production-checklist.md` (this file — new)

**Open items:**
- OQ-03-001 (SSE envelope schema): BRD-02 owner
- OQ-03-002 (forbidden fields list review): BRD-02 owner

**No blockers remain.** BRD-03 artifacts are complete (validator PASS, PM GATE 2 APPROVE). Implementation dispatch can proceed after PM creates implementation tasks per pipeline rules.

---

*Production checklist generated by ops profile — task t_f8ed956e*