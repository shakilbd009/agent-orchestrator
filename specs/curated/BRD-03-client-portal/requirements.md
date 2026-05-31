# BRD-03 — Requirements Analysis

**BRD:** BRD-03-client-portal
**Stage:** 02-requirements
**Status:** Analyzed

---

## Functional Requirements Decomposition

### FR Category: Portfolio (US-03-001)

| FR | What | Priority | Acceptance Criteria | Eval Method | Open Issues |
|----|------|----------|---------------------|-------------|-------------|
| FR-03-001 | Multi-project client portal | MUST | Landing page shows all projects accessible to current client principal | Access-control integration test | BRD-02 project API contract |
| FR-03-002 | Portfolio health summary | MUST | Aggregate counts for on_track/at_risk/blocked visible on landing | UI/API integration test | None |
| FR-03-003 | Portfolio confidence summary | MUST | high/medium/low confidence summarized across accessible projects | UI/API test | None |
| FR-03-004 | Portfolio decision summary | MUST | Total pending, overdue, waiting-on-client counts visible | UI/API integration test | 24h overdue threshold definition |
| FR-03-005 | Project list | MUST | Project name, health, confidence, completion%, milestone, decision counts, latest update timestamp | UI/API test | None |

### FR Category: Project Detail (US-03-002)

| FR | What | Priority | Acceptance Criteria | Eval Method | Open Issues |
|----|------|----------|---------------------|-------------|-------------|
| FR-03-006 | Project access filtering | MUST | Every portal read/board/search/approval/comment/SSE filtered to accessible projects | Negative access-control test | BRD-02 session contract |
| FR-03-007 | Cross-project isolation | MUST | Client NEVER sees unauthorized project names, tasks, risks, milestones, approvals, counts, search results | Negative tests across all views | None |
| FR-03-008 | Project detail view | MUST | Health summary, completion%, board, approvals, blockers, risks, milestones, comments, next actions | UI/API integration test | None |
| FR-03-009 | Delivery health representation | MUST | on_track/at_risk/blocked with high/medium/low confidence and plain-language reason | UI test | None |
| FR-03-010 | Completion percentage | MUST | done/(todo+in_progress+blocked+done); excludes cancelled/proposed; zero denominator shows empty state | Unit/integration test | None |
| FR-03-011 | Project board status mapping | MUST | BRD-02 canonical statuses with client-readable labels | UI test | None |
| FR-03-012 | Cancelled visibility | MUST | Hidden by default; toggle reveals with cancellation reason | UI integration test | None |
| FR-03-013 | Client-visible task detail | MUST | title, status, owner label, summary, blocker reason, due date, update timestamp, next action | UI inspection/integration test | None |
| FR-03-014 | Business-language-only display | MUST | No stack traces, logs, agent IDs, branch names, commit SHAs, file paths, infrastructure jargon | Content safety test | Publication validation needed |
| FR-03-015 | Safe fallback for missing summaries | MUST | "Internal work item awaiting summary" when no client-safe summary available | UI behavior test | None |

### FR Category: Owner Labels (US-03-006)

| FR | What | Priority | Acceptance Criteria | Eval Method | Open Issues |
|----|------|----------|---------------------|-------------|-------------|
| FR-03-016 | Client-facing owner labels | MUST | Hybrid model: auto-mapping default, internal override capability | API/UI integration test | Default mapping table |
| FR-03-017 | Default owner label mapping | SHOULD | Default labels: Product, Engineering, Review, Quality, Client | API inspection | None |

### FR Category: Approvals (US-03-003)

| FR | What | Priority | Acceptance Criteria | Eval Method | Open Issues |
|----|------|----------|---------------------|-------------|-------------|
| FR-03-018 | Global approval inbox | MUST | Portfolio landing includes or links to global inbox with all pending decisions | UI/API integration test | SSE scope for cross-project inbox |
| FR-03-019 | Project approval inbox | MUST | Project detail shows pending decisions for that project | UI/API test | None |
| FR-03-020 | Approval consistency | MUST | Global and project inbox show same underlying state; acting in one updates the other | UI state integration test | SSE reconciliation |
| FR-03-021 | Approval outcomes | MUST | Four outcomes: approve, reject, request_changes, need_more_information | Integration test | None |
| FR-03-022 | Approval comments | MUST | reject/request_changes/need_more_info require comment; approve has optional comment | Validation/integration test | Comment format/spec |
| FR-03-023 | Need More Information semantics | MUST | nmi → waiting-for-response; not rejection; blocks completion until resolved | State machine test | None |
| FR-03-024 | Approval audit metadata | MUST | Actor identity, role, project ID, item ID, outcome, timestamp, comment metadata | Audit inspection | None |

### FR Category: Comments (US-03-004)

| FR | What | Priority | Acceptance Criteria | Eval Method | Open Issues |
|----|------|----------|---------------------|-------------|-------------|
| FR-03-025 | Simple comments | MUST | Clients can comment on approvals, blockers, risks, milestones; visible to client and internal | Comment integration test | None |
| FR-03-026 | Comment ordering | MUST | Newest-first within each item | UI behavior test | None |
| FR-03-027 | Comment fields | MUST | Author display name, timestamp, updated timestamp, edited indicator, project/item, body | UI inspection test | None |
| FR-03-028 | Comment edit/delete | MUST | Authors edit/delete own comments; deleted hidden from normal view | Comment integration test | None |
| FR-03-029 | Comment audit privacy | MUST | Audit events exclude deleted/edited body text | Audit inspection test | None |
| FR-03-030 | Comment retention | MUST | Visible comments/decisions retained for project lifetime | Retention behavior test | BRD-02 retention contract |

### FR Category: Risks & Milestones

| FR | What | Priority | Acceptance Criteria | Eval Method | Open Issues |
|----|------|----------|---------------------|-------------|-------------|
| FR-03-031 | Risk list | MUST | Severity, impact, owner label, mitigation summary, status, next action | UI/API integration test | BRD-16 risk data contract |
| FR-03-032 | Risk visibility default | MUST | All risks client-visible by default unless governance explicitly changes | Risk visibility test | None |
| FR-03-033 | Risk comments | MUST | Same simple comment model as approval/blocker comments | Comment integration test | None |
| FR-03-034 | Milestone list | MUST | Name, status, due/target date, progress, health, summary, next action | UI/API test | None |
| FR-03-035 | Milestones independent | MUST | Milestone progress does NOT change task-count completion % | Unit/integration test | None |
| FR-03-036 | Due dates | MUST | Milestone targets, task due dates, approval due dates shown when available; missing not error | UI behavior test | None |
| FR-03-037 | Overdue client decisions | MUST | 24h pending threshold; only visibility signal; no auto-reject/approve/advance | Time-controlled integration test | None |

### FR Category: Search, Filter, Real-time

| FR | What | Priority | Acceptance Criteria | Eval Method | Open Issues |
|----|------|----------|---------------------|-------------|-------------|
| FR-03-038 | Search | MUST | Text search across client-visible project/task/risk/milestone/approval/blocker content; filtered to accessible projects | Search integration test | BRD-02 search API contract |
| FR-03-039 | Basic filters | MUST | Health, task status, decisions needed, blocked, at-risk, overdue decisions | Filter integration test | None |
| FR-03-040 | Real-time updates | MUST | Initial load from current-state APIs; SSE subscription for live updates | SSE integration/performance test | BRD-02 SSE event contract |
| FR-03-041 | Manual refresh fallback | MUST | SSE disconnect → "live updates paused" + manual refresh via current-state APIs | Failure-mode integration test | None |
| FR-03-042 | Current state authoritative | MUST | Current-state APIs authoritative for initial load, manual refresh, reconciliation | Integration test | None |

### FR Category: Publication/Review (US-03-005)

| FR | What | Priority | Acceptance Criteria | Eval Method | Open Issues |
|----|------|----------|---------------------|-------------|-------------|
| FR-03-043 | Internal publish/review step | MUST | Items explicitly published internally before client-visible | Integration test | None |
| FR-03-044 | Unpublish action | MUST | Audited internal action; unpublished items disappear from portal views | Integration plus audit inspection | None |
| FR-03-045 | Publication validation | MUST | Business-language summary, owner label, next action, visibility status, no forbidden technical fields | Publication validation test | Forbidden technical fields list |
| FR-03-046 | Generic unpublished blocker | MUST | Unpublished internal task blocking published item → generic "Internal dependency is blocking progress" | Content safety integration test | None |
| FR-03-047 | Next action required | MUST | Every published client-visible item has exactly one current client-facing next action | Publication validation test | None |
| FR-03-048 | Publication failure behavior | MUST | Missing required fields or forbidden technical content → validation fails → item stays hidden | Publication validation test | None |

### FR Category: Empty States & UX

| FR | What | Priority | Acceptance Criteria | Eval Method | Open Issues |
|----|------|----------|---------------------|-------------|-------------|
| FR-03-049 | Empty states | MUST | Client-friendly empty states with suggested next action for no projects, no active work, no approvals, no risks, no milestones, empty search/filter | UI behavior test | None |
| FR-03-050 | Responsive browser experience | MUST | Core capabilities on desktop, tablet, mobile; no native app required | Responsive UI test | Mobile board layout at high task count |
| FR-03-053 | Read-only degraded mode | SHOULD | Read-only when submission unavailable but reads available; write controls disabled with explanation | Degraded mode integration test | None |
| FR-03-054 | Progressive loading | SHOULD | Portfolio renders high-level summary before lower-priority details when aggregation is slow | Performance test | None |
| FR-03-055 | Reconciliation after SSE reconnect | SHOULD | After reconnect/manual refresh, reconcile against current-state APIs | Integration test | None |
| FR-03-056 | Client-safe status explanations | SHOULD | Health/confidence/blockers/risks include concise plain-language explanations | UI content review | None |
| FR-03-057 | Accessible mobile board layout | SHOULD | Board columns stack/transform on small screens; canonical status grouping preserved | Responsive UI test | None |

### Out of Scope (confirmed)

- Internal agent workstream dashboard (BRD-04)
- Full collaboration workspace with threads, mentions, attachments (BRD-18)
- Notification delivery (BRD-21)
- Multi-tenant administration and external identity (BRD-17 + platform auth)
- Deep risk lifecycle and scoring (BRD-16)
- CSV/PDF export (BRD-02, BRD-19)
- Raw technical logs, stack traces, infrastructure metrics
- Offline-first behavior

---

## Non-Functional Requirements Analysis

| NFR | Requirement | Target | Gap / Notes |
|-----|-------------|--------|-------------|
| NFR-03-001 | Client user scale | 10 concurrent client users | No gap — clearly specified |
| NFR-03-002 | Project portfolio scale | 50 accessible projects per client principal | No gap — clearly specified |
| NFR-03-003 | Project task scale | Projects with up to 10,000 tasks | No gap — aligned with BRD-02 |
| NFR-03-004 | Portfolio landing latency | Visible within 5s at target scale | No gap — specific target |
| NFR-03-005 | Project detail latency | Visible within 2s at target scale | No gap — specific target |
| NFR-03-006 | Live update freshness | Within 2s via SSE | No gap — matches BRD-02 target |
| NFR-03-007 | Manual refresh fallback | Current-state data or clear failure without SSE | No gap — clearly specified |
| NFR-03-008 | Read-only degraded mode | Readable with write controls disabled | No gap — clearly specified |
| NFR-03-009 | Read failure behavior | Unavailable state; no stale as current | No gap — clearly specified |
| NFR-03-010 | Accessibility | WCAG 2.1 AA | No gap — standard specified |
| NFR-03-011 | Responsiveness | Core capabilities on desktop/tablet/mobile | No gap — standard specified |
| NFR-03-012 | Data retention | Comments/decisions for project lifetime | No gap — matches BRD-02 |
| NFR-03-013 | Access boundary | Filtering failures fail closed (security defect) | No gap — clearly specified |
| NFR-03-014 | Business-language safety | Publication validation before visibility | No gap — clearly specified |
| NFR-03-015 | Operational dependency | Reads work when SSE unavailable | No gap — clearly specified |

### Missing NFRs — None identified

All NFRs have specific, measurable targets. No hidden gaps found.

---

## Open Questions (OQs)

| OQ | Question | Impact | Status |
|----|----------|--------|--------|
| OQ-03-001 | What is the exact SSE event envelope schema for project-scoped updates that the client portal subscribes to? | High — needed for BRD-02 SSE integration and eval contract authoring | Unresolved — BRD-02 must define |
| OQ-03-002 | What is the forbidden technical fields list used in publication validation (FR-03-045)? | High — implementation requires explicit list | Unresolved — must be defined before implementation |
| OQ-03-003 | Does the global approval inbox SSE subscription scope cover all accessible projects or just a subset? | Medium — affects SSE resource usage and event filtering | Unresolved — depends on BRD-02 SSE scope definition |
| OQ-03-004 | What is the SSE reconnect strategy and backoff policy when a project SSE stream is lost? | Medium — affects client UX during connectivity issues | Unresolved — could be deferred to implementation |
| OQ-03-005 | Is the 24h overdue threshold applied per decision item or per project-level aggregate? | Low — visibility signal only, no enforcement action | Resolved: per decision item (individual approval item) |

---

## Cross-BRD Dependencies

| BRD | Dependency Type | What BRD-03 Needs |
|-----|-----------------|-------------------|
| BRD-02 | API contract | Project/task/gate/current-state APIs; SSE event stream; canonical statuses; retention; audit events |
| BRD-01 | Shell integration | Frontend/backend shell, session/auth integration, health endpoint conventions |
| BRD-04 | Scope boundary | BRD-04 owns internal agent workstream visibility; BRD-03 must not overlap |
| BRD-16 | Risk data | BRD-16 owns risk lifecycle and governance; BRD-03 displays client-visible risk info |
| BRD-17 | Access filtering | BRD-17 owns tenant lifecycle and enterprise access policy; BRD-03 inherits project access filtering |
| BRD-18 | Scope boundary | BRD-18 owns threaded discussion, mentions, attachments, teams; BRD-03 simple comments only |
| BRD-19 | Retention | BRD-19 owns artifact/version history and export; BRD-03 records client action metadata |
| BRD-21 | Scope boundary | BRD-21 owns notification delivery; BRD-03 shows visible indicators only |
| contracts/openapi.yaml | API contract | Must define client portal read/action endpoints after BRD-03 approved |
| contracts/events.md | Event contract | Must include client portal decision/comment/publication events |
| specs/feature-flags.md | Flag registry | Must add `client-portal` flag (preserve `dashboard` until later ADR) |

---

## Assumptions Made

| Assumption | Validation Plan |
|------------|-----------------|
| Client portal inherits session/auth contract from BRD-01 app shell | Verify during BRD-02 API contract definition |
| Project access filtering is enforced at the API layer (BRD-02), not at the portal UI | Verify during access control integration test |
| SSE subscriptions are project-scoped; one stream per visible project | Confirm with BRD-02 SSE scope definition |
| Publication validation is a backend concern, not just UI enforcement | Verify during publication validation test design |
| Overdue decision threshold uses wall-clock time from item becoming visible/actionable | Verify during time-controlled integration test |

---

## Red Flags Check

| Flag | Status | Notes |
|------|--------|-------|
| Solution bias | ✅ PASS | Describes WHAT (client-facing display), not HOW (implementation) |
| Vague requirements | ✅ PASS | All FRs have specific acceptance criteria |
| No success metrics | ✅ PASS | NFRs have specific targets (5s, 2s, 10 users, 50 projects) |
| Scope creep | ✅ PASS | Out of scope explicitly listed and separated from BRD-04/18/21 |
| "Figure it out later" | ⚠️ OQ-03-001, OQ-03-002 | SSE envelope schema and forbidden fields list undefined — block eval contract authoring |

---

*Analyzed: Stage 02 requirements*
*Next: Stage 03 trade-offs → identify implicit decisions for ADR*