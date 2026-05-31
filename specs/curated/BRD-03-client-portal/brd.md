# BRD-03: Client Portal and Business Project Board

> **Status:** Approved for Curation

---

## Overview

The Client Portal gives client stakeholders a multi-project, business-readable view of delivery progress, pending decisions, risks, milestones, and project board status. It translates internal orchestration state into plain business language so clients can understand what is happening, what needs their attention, and whether delivery is on track without seeing raw agent activity, technical logs, or implementation details. BRD-03 owns the client-facing portal experience; BRD-04 owns internal agent workstream visibility, BRD-18 owns richer collaboration, and BRD-21 owns notifications.

---

## User Stories

| ID | As a | I want | So that |
|----|------|--------|---------|
| US-03-001 | Client stakeholder | See a portfolio summary across all projects I can access | I can quickly understand overall delivery health and where my attention is needed |
| US-03-002 | Client stakeholder | Open a project and view a business-readable project board | I can understand progress, blockers, risks, milestones, and next actions without internal technical detail |
| US-03-003 | Client stakeholder | Approve, reject, request changes, or ask for more information from either a global inbox or project detail view | I can make decisions without hunting through each project manually |
| US-03-004 | Client stakeholder | Comment on approvals, blockers, risks, and milestones | I can provide clarification and feedback in context |
| US-03-005 | Internal project owner | Publish only reviewed, client-safe project content | Clients see accurate business-language status without raw internal work leaking into the portal |
| US-03-006 | Internal project owner | Override client-facing owner labels when needed | Ownership is presented in terms clients understand rather than internal agent or team names |
| US-03-007 | Client stakeholder using a small screen | Use the same portal capabilities from desktop, tablet, or mobile browser | I can review status and decisions from whichever browser device is convenient |

---

## Functional Requirements

### Must Have

- **FR-03-001: Multi-project client portal.** The portal MUST provide a landing page for client stakeholders that summarizes all projects the current client principal is allowed to access.
- **FR-03-002: Portfolio health summary.** The landing page MUST show aggregate counts for projects by delivery health: `on_track`, `at_risk`, and `blocked`.
- **FR-03-003: Portfolio confidence summary.** The landing page MUST summarize delivery confidence across accessible projects using `high`, `medium`, and `low` confidence where confidence is available.
- **FR-03-004: Portfolio decision summary.** The landing page MUST show total pending client decisions, overdue client decisions, projects waiting on client action, and projects currently blocked or at risk.
- **FR-03-005: Project list.** The landing page MUST list accessible projects with project name, delivery health, confidence, completion percentage, next milestone, pending decision count, overdue decision count, and latest client-safe update timestamp.
- **FR-03-006: Project access filtering.** Every portal read, board view, search result, approval item, comment, risk, milestone, and SSE-driven update MUST be filtered to the projects the current client principal may access. BRD-03 does not define the underlying authentication mechanism; it inherits the platform/app-shell session and identity contract.
- **FR-03-007: Cross-project isolation.** A client MUST NOT see project names, task titles, risks, milestones, approvals, comments, counts, or search results for projects outside their accessible project set.
- **FR-03-008: Project detail view.** The portal MUST provide a per-project detail view containing a health summary, completion percentage, project board, approvals, blockers, risks, milestones, comments where relevant, and next actions.
- **FR-03-009: Delivery health representation.** Each project MUST show delivery health as one of `on_track`, `at_risk`, or `blocked`, with `high`, `medium`, or `low` confidence and a one-sentence plain-language reason.
- **FR-03-010: Completion percentage.** Client-visible completion percentage MUST be calculated as active `done` tasks divided by active tasks in `todo`, `in_progress`, `blocked`, or `done` status. `cancelled` tasks and proposed/inactive tasks MUST be excluded from both numerator and denominator. If the denominator is zero, the portal MUST show a plain-language empty state such as “No active work yet” rather than 0% or 100%.
- **FR-03-011: Project board status mapping.** The project board MUST use BRD-02 canonical task statuses: `todo`, `in_progress`, `blocked`, `done`, and `cancelled`. Client-facing labels MAY render as To Do, In Progress, Blocked, Done, and Cancelled.
- **FR-03-012: Cancelled visibility.** `cancelled` tasks MUST be hidden by default and visible only when the client enables a “show cancelled” control. Visible cancelled tasks MUST include a plain-language cancellation reason.
- **FR-03-013: Client-visible task detail.** Client-visible task cards MUST include title, status, client-facing owner label, plain-language summary, dependency or blocker reason when applicable, due date when available and client-visible, latest update timestamp, and next action.
- **FR-03-014: Business-language-only display.** The portal MUST NOT expose stack traces, raw logs, internal agent IDs, branch names, commit SHAs, implementation file paths, infrastructure jargon, or raw technical task details. Client-facing content MUST use plain business language.
- **FR-03-015: Safe fallback for missing summaries.** If an otherwise visible item lacks a client-safe summary, the portal MUST show a safe fallback such as “Internal work item awaiting summary” rather than exposing raw internal content.
- **FR-03-016: Client-facing owner labels.** Owner labels MUST use a hybrid model: automatically map internal role/task metadata to business labels by default, while allowing an internal project owner to override the client-facing owner label.
- **FR-03-017: Default owner label mapping.** The default mapping SHOULD use labels such as Product, Engineering, Review, Quality, and Client rather than internal agent profile names.
- **FR-03-018: Global approval inbox.** The portfolio landing page MUST include or link to a global approval inbox containing pending client decisions across all accessible projects.
- **FR-03-019: Project approval inbox.** Each project detail view MUST show pending client decisions for that project.
- **FR-03-020: Approval consistency.** An approval item shown in both the global inbox and project detail view MUST represent the same underlying decision state. Acting in one location MUST update the other location through current-state reads and live update mechanisms.
- **FR-03-021: Approval outcomes.** Client-facing decisions MUST support four outcomes: `approve`, `reject`, `request_changes`, and `need_more_information`.
- **FR-03-022: Approval comments.** `reject`, `request_changes`, and `need_more_information` outcomes MUST require a plain-language comment/reason. `approve` MAY include an optional comment.
- **FR-03-023: Need More Information semantics.** `need_more_information` MUST place the approval item into a waiting-for-response state and MUST NOT count as rejection. It MUST prevent the decision from being considered complete until the requested information is resolved.
- **FR-03-024: Approval audit metadata.** Every approval outcome MUST record actor identity/display name, actor role or client principal type, project ID, related item ID, outcome, timestamp, and comment metadata where applicable.
- **FR-03-025: Simple comments.** Clients MUST be able to create simple comments on approvals, blockers, risks, and milestones. Comments MUST be visible to both client and internal users on the relevant item.
- **FR-03-026: Comment ordering.** Comments MUST be shown newest-first within each relevant item.
- **FR-03-027: Comment fields.** Each visible comment MUST show author display name, timestamp, updated timestamp when edited, edited indicator when applicable, related project/item, and comment body.
- **FR-03-028: Comment edit/delete.** Comment authors MUST be able to edit and delete their own comments. Deleted comments MUST be hidden from the normal client view.
- **FR-03-029: Comment audit privacy.** Comment audit events MUST record create/edit/delete metadata, actor, timestamp, project, related item, and action. Audit events MUST NOT retain deleted comment bodies or previous edited comment body versions.
- **FR-03-030: Comment retention.** Visible comments and approval decisions MUST be retained for the lifetime of the project, matching BRD-02 retention expectations.
- **FR-03-031: Risk list.** The project detail view MUST include a client-visible risk list. Risks MUST show severity, impact, client-facing owner label, mitigation summary, status where available, and next action.
- **FR-03-032: Risk visibility default.** All risks are client-visible by default unless a later governance rule or project configuration explicitly changes visibility.
- **FR-03-033: Risk comments.** Clients MUST be able to comment on risks using the same simple comment model.
- **FR-03-034: Milestone list.** The project detail view MUST include a milestone list or timeline showing milestone name, status, due or target date, progress, health, summary, and next action.
- **FR-03-035: Milestones independent of completion calculation.** Milestone status MUST be displayed independently and MUST NOT drive the raw task-count completion percentage.
- **FR-03-036: Due dates.** The portal MUST show milestone target dates, task due dates, and approval/request due dates when available and client-visible. Missing due dates MUST NOT be treated as errors.
- **FR-03-037: Overdue client decisions.** A client decision MUST be considered overdue when it remains pending more than 24 hours after becoming visible and actionable to the client. Overdue state is a visibility signal only and MUST NOT automatically reject, approve, or advance work.
- **FR-03-038: Search.** The portal MUST support text search across client-visible project, task, risk, milestone, approval, and blocker content.
- **FR-03-039: Basic filters.** The portal MUST support basic filters for delivery health, task status, decisions needed, blocked items, at-risk projects/items, and overdue decisions.
- **FR-03-040: Real-time updates.** On initial load, the portal MUST read authoritative current state from platform APIs. It SHOULD subscribe to BRD-02 project-scoped SSE streams for live updates relevant to visible portfolio and project views.
- **FR-03-041: Manual refresh fallback.** If SSE disconnects or fails, the portal MUST remain usable, show a non-alarming “live updates paused” style state, and provide manual refresh using current-state APIs.
- **FR-03-042: Current state remains authoritative.** SSE events MUST be used for freshness and UI updates, but current-state APIs remain authoritative for initial load, manual refresh, and reconciliation.
- **FR-03-043: Internal publish/review step.** Projects, tasks, blockers, risks, milestones, and approval items MUST be explicitly published or approved internally before becoming visible to clients.
- **FR-03-044: Unpublish action.** Published items MAY be unpublished only through an audited internal action. Unpublished items MUST no longer appear in client portal views except through approved generic summaries.
- **FR-03-045: Publication validation.** Publishing a client-visible item MUST require a business-language summary, client-facing owner label, next action, visibility status, and validation that forbidden technical fields are not exposed.
- **FR-03-046: Generic unpublished blocker.** When an unpublished internal item blocks a published client-visible milestone, project, or item, the portal MUST show only a generic client-safe blocker such as “Internal dependency is blocking progress” and MUST NOT expose the unpublished task title or raw detail.
- **FR-03-047: Next action required.** Every published client-visible item MUST have exactly one current client-facing next action. If no client action is needed, the next action may describe an internal next step in client-safe language.
- **FR-03-048: Publication failure behavior.** Items missing required publication fields or containing forbidden technical fields MUST fail publication validation and remain hidden from the client portal.
- **FR-03-049: Empty states.** The portal MUST provide client-friendly empty states with suggested next action for no accessible projects, no active work, no approvals, no risks, no milestones, and empty search/filter results.
- **FR-03-050: Responsive browser experience.** The portal MUST work in desktop, tablet, and mobile browsers with the same core capabilities. No native mobile app is required.
- **FR-03-051: No BRD-03 export.** BRD-03 MUST NOT define CSV, PDF, or other export behavior. Export remains owned by BRD-02 and BRD-19.
- **FR-03-052: No BRD-03 notifications.** BRD-03 MUST NOT define email, push, Slack, external notification, or in-app notification center delivery. It may show visible indicators only. Notification delivery remains owned by BRD-21.

### Should Have

- **FR-03-053: Read-only degraded mode.** If approval/comment submission is unavailable but read APIs are available, the portal SHOULD remain usable in read-only mode and disable submission controls with a plain-language explanation.
- **FR-03-054: Progressive loading.** The portfolio page SHOULD render high-level summary before lower-priority details when aggregation is slow, while still meeting the portfolio latency target.
- **FR-03-055: Reconciliation after SSE reconnect.** After SSE reconnect or manual refresh, the portal SHOULD reconcile visible state against current-state APIs to avoid stale approval/comment/board state.
- **FR-03-056: Client-safe status explanations.** Health, confidence, blockers, and risks SHOULD include concise explanations that can be understood without knowing internal workflow terminology.
- **FR-03-057: Accessible mobile board layout.** On smaller screens, board columns SHOULD stack or transform into an accessible grouped list while preserving canonical status grouping.

### Could Have

- Saved client filters or per-user default filter state.
- Configurable landing preference between portfolio summary, approval inbox, or project list.
- Optional richer milestone visualization beyond list/timeline basics.
- Client-visible trend indicators for progress over time.
- A future ADR or cleanup BRD to deprecate the generic `dashboard` flag after `client-portal` is adopted.

### Out of Scope

- Internal agent workstream dashboard, agent handoff chains, and per-agent status detail; owned by BRD-04.
- Full collaboration workspace including threaded discussions, mentions, attachments, team membership, and complex comment permissions; owned by BRD-18.
- Notification delivery through email, push, Slack, webhooks for users, or in-app notification center; owned by BRD-21.
- Full multi-tenant administration, tenant lifecycle, enterprise permission policy, and external identity provider integration; owned by BRD-17 or the platform auth layer.
- Deep risk lifecycle management, risk creation workflow, risk scoring methodology, and mitigation governance; owned by BRD-16.
- CSV/PDF/status report export; owned by BRD-02 and BRD-19.
- Raw technical logs, stack traces, infrastructure metrics, branch/commit displays, or developer debugging views.
- Offline-first behavior or local cached stale-data mode when read APIs are unavailable.

---

## Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-03-001 | Client user scale | Support up to 10 concurrent client users in the initial target environment |
| NFR-03-002 | Project portfolio scale | Support up to 50 accessible projects per client principal |
| NFR-03-003 | Project task scale | Support project detail views for projects with up to 10,000 tasks, aligned with BRD-02 |
| NFR-03-004 | Portfolio landing latency | Initial portfolio summary visible within 5 seconds at target scale |
| NFR-03-005 | Project detail latency | Selected project detail summary and board visible within 2 seconds at target scale |
| NFR-03-006 | Live update freshness | Relevant committed updates visible within 2 seconds when SSE is connected, matching BRD-02 target |
| NFR-03-007 | Manual refresh fallback | Manual refresh returns current-state data or a clear failure message without requiring SSE |
| NFR-03-008 | Read-only degraded mode | If read APIs work but approval/comment submission fails, dashboard remains readable and disables write controls with explanation |
| NFR-03-009 | Read failure behavior | If read APIs are unavailable, portal shows a clear unavailable state and does not present stale data as current |
| NFR-03-010 | Accessibility | Client-facing UI meets WCAG 2.1 AA for supported browser experiences |
| NFR-03-011 | Responsiveness | Core portal capabilities usable on desktop, tablet, and mobile browser widths |
| NFR-03-012 | Data retention | Visible comments and approval decisions retained for the lifetime of the project |
| NFR-03-013 | Access boundary | Cross-project access filtering failures are treated as security defects and must fail closed |
| NFR-03-014 | Business-language safety | Published client-visible content must pass publication validation before becoming visible |
| NFR-03-015 | Operational dependency | Portal remains usable for reads when SSE is unavailable, as long as current-state read APIs are available |

---

## Observability

### Metrics to emit

- `client_portal_portfolio_view_total` — counter for portfolio landing views, labeled by client principal type where safe.
- `client_portal_project_view_total` — counter for project detail views, labeled by project ID where safe for internal metrics.
- `client_portal_portfolio_load_duration_ms` — histogram for portfolio summary load duration.
- `client_portal_project_load_duration_ms` — histogram for project detail load duration.
- `client_portal_pending_approvals_current` — gauge for current pending client approvals across accessible projects.
- `client_portal_overdue_decisions_current` — gauge for decisions pending more than 24 hours.
- `client_portal_oldest_pending_decision_age_ms` — gauge for age of oldest visible pending client decision.
- `client_portal_decision_turnaround_ms` — histogram for time from decision becoming client-actionable to client outcome.
- `client_portal_decision_outcome_total` — counter for approval outcomes, labeled by `approve`, `reject`, `request_changes`, and `need_more_information`.
- `client_portal_need_more_information_current` — gauge for current approval items waiting on more information.
- `client_portal_requested_changes_current` — gauge for current approval items in requested-changes state.
- `client_portal_blocked_projects_current` — gauge for accessible projects currently blocked.
- `client_portal_at_risk_projects_current` — gauge for accessible projects currently at risk.
- `client_portal_publication_validation_failed_total` — counter for failed publication validation attempts, labeled by reason category.
- `client_portal_comment_created_total` — counter for client-visible comments created.
- `client_portal_comment_edited_total` — counter for comment edits.
- `client_portal_comment_deleted_total` — counter for comment deletions.
- `client_portal_sse_disconnect_total` — counter for SSE disconnects while portal views are active.
- `client_portal_manual_refresh_total` — counter for manual refresh actions, labeled by normal or SSE-fallback context.
- `client_portal_submission_failed_total` — counter for failed approval/comment submissions.
- `client_portal_access_denied_total` — counter for denied attempts to access projects or items outside the client principal's allowed project set.

### Log events

- `client_portal.portfolio.viewed` — when a client principal opens the portfolio landing page, including accessible project count.
- `client_portal.project.viewed` — when a client principal opens a project detail view, including project ID.
- `client_portal.approval.submitted` — when a client submits an approval outcome, including project ID, related item ID, outcome, actor identity, and timestamp.
- `client_portal.approval.need_more_information` — when a client asks for more information, including related item ID and timestamp.
- `client_portal.comment.created` — when a client-visible comment is created, including project ID, related item ID, actor identity, and timestamp.
- `client_portal.comment.edited` — when a comment author edits their comment, excluding previous body text.
- `client_portal.comment.deleted` — when a comment author deletes their comment, excluding deleted body text.
- `client_portal.item.published` — when an internal actor publishes a client-visible item, including validation result and actor identity.
- `client_portal.item.unpublished` — when an internal actor unpublishes an item, including actor identity and reason.
- `client_portal.publication_validation.failed` — when publication validation fails, including reason category but not forbidden raw content.
- `client_portal.access.denied` — when a client principal attempts to access unauthorized project or item data.
- `client_portal.sse.connected` — when the portal connects to a project SSE stream.
- `client_portal.sse.disconnected` — when the portal loses a project SSE stream.
- `client_portal.read_only_mode.entered` — when write controls are disabled because submission is unavailable while reads remain available.
- `client_portal.reads.unavailable` — when current-state read APIs are unavailable and the portal cannot show current data.

### Health/readiness endpoints

- BRD-03 does not require a new health endpoint separate from the app shell and BRD-02 backend readiness endpoints.
- `GET /ready` impact: readiness for the client portal SHOULD report degraded or unavailable when current-state project reads, approval reads, comment reads, or publication state reads are unavailable.
- `GET /ready` SHOULD distinguish read availability from write/submission availability so the UI can enter read-only degraded mode when appropriate.
- `GET /live` remains the process-level liveness endpoint inherited from the app shell/backend platform.

---

## Feature Flag

- **Flag name:** `client-portal`
- **Type:** boolean
- **Default:** `false`
- **Server env:** `FF_ENABLE_CLIENT_PORTAL`
- **Browser env:** `VITE_FF_ENABLE_CLIENT_PORTAL`
- **Scope:** Enables the multi-project client portal, portfolio landing page, client-facing project board, approval actions, simple comments, client-visible risks, milestones, search/filtering, publication validation, and SSE/manual-refresh client portal behavior.

Feature flag registry impact:

- Add `client-portal` to `specs/feature-flags.md` with default `false`, current `false`, domain `UI`, introduced in Phase 1, and notes identifying it as the client-facing multi-project portal.
- Keep the existing `dashboard` flag until a later ADR or cleanup BRD decides whether to deprecate or repurpose the generic Phase 0 dashboard naming.
- Do not gate BRD-03 behavior solely on the existing generic `dashboard` flag.

---

## Acceptance Criteria

| ID | Criterion | Eval method |
|----|-----------|-------------|
| AC-03-001 | With `client-portal=false`, client portal routes and client portal API behavior are hidden or disabled according to project feature-flag conventions | Feature flag integration test |
| AC-03-002 | A client principal with access to multiple projects sees only those projects in the portfolio summary and project list | Access-control integration test |
| AC-03-003 | A client principal cannot retrieve or search projects, tasks, risks, milestones, approvals, comments, counts, or SSE updates for an unauthorized project | Negative access-control integration test |
| AC-03-004 | Portfolio landing shows health counts for On Track, At Risk, and Blocked projects, plus pending and overdue client decision counts | UI/API integration test |
| AC-03-005 | Portfolio summary becomes visible within 5 seconds at target scale of 50 accessible projects | Performance test |
| AC-03-006 | Selecting one project renders project detail summary and board within 2 seconds at target project scale | Performance test |
| AC-03-007 | Project completion percentage equals `done / (todo + in_progress + blocked + done)` and excludes cancelled and proposed/inactive tasks | Unit/integration test |
| AC-03-008 | When a project has no active tasks, completion displays a plain-language empty state rather than 0% or 100% | UI behavior test |
| AC-03-009 | Project board displays BRD-02 statuses with client-readable labels and hides cancelled tasks by default | UI integration test |
| AC-03-010 | Enabling “show cancelled” reveals cancelled tasks with a plain-language cancellation reason | UI integration test |
| AC-03-011 | Client-visible task cards show title, status, owner label, summary, dependency/blocker reason when applicable, due date when available, latest update, and next action | UI inspection/integration test |
| AC-03-012 | Raw technical content such as stack traces, internal agent IDs, commit SHAs, branch names, file paths, or infrastructure logs is not rendered in client-facing views | Content safety test |
| AC-03-013 | Publishing an item without business-language summary, owner label, next action, visibility status, or technical-leak validation fails and keeps the item hidden | Publication validation test |
| AC-03-014 | A published item can be unpublished only by an audited internal action and then disappears from client portal views | Integration plus audit inspection |
| AC-03-015 | An unpublished internal task blocking a published item appears only as a generic client-safe blocker and does not expose the unpublished task title or detail | Content safety integration test |
| AC-03-016 | Each published client-visible item has exactly one current client-facing next action | Publication validation test |
| AC-03-017 | Client-facing owner labels are generated by default mapping and can be overridden by an internal project owner | API/UI integration test |
| AC-03-018 | Global approval inbox and project-level approval inbox show consistent state for the same approval item | UI/API integration test |
| AC-03-019 | Submitting `approve` records an approved outcome and optional comment metadata | Approval integration plus audit inspection |
| AC-03-020 | Submitting `reject` without a reason is rejected; submitting with a reason records the rejected outcome | Validation/integration test |
| AC-03-021 | Submitting `request_changes` without a requested-change comment is rejected; submitting with a comment records requested-changes state | Validation/integration test |
| AC-03-022 | Submitting `need_more_information` without a question/comment is rejected; submitting with one places the item into waiting-for-response state without marking it rejected | Validation/integration test |
| AC-03-023 | Acting on an approval from the global inbox updates the project detail view, and acting from project detail updates the global inbox | UI state integration test |
| AC-03-024 | A client decision pending more than 24 hours after becoming visible/actionable appears as overdue and increments overdue counts | Time-controlled integration test |
| AC-03-025 | Clients can create comments on approvals, blockers, risks, and milestones, and comments are visible to both client and internal users | Comment integration test |
| AC-03-026 | Comments are displayed newest-first on the relevant item | UI behavior test |
| AC-03-027 | Comment authors can edit their own comments; edited comments show edited indicator and updated timestamp | Comment integration test |
| AC-03-028 | Comment authors can delete their own comments; deleted comments are hidden from normal client view | Comment integration test |
| AC-03-029 | Comment edit/delete audit metadata excludes prior edited body text and deleted comment body text | Audit inspection test |
| AC-03-030 | Visible comments and approval decisions remain available for the lifetime of the project unless project archival/deletion behavior from another BRD applies | Retention behavior test |
| AC-03-031 | Project detail shows risks with severity, impact, owner label, mitigation summary, status where available, and next action | UI/API integration test |
| AC-03-032 | All risks are client-visible by default unless later governance explicitly changes visibility | Risk visibility test |
| AC-03-033 | Milestone list shows status, due/target date, progress, health, summary, and next action | UI/API integration test |
| AC-03-034 | Milestone progress does not change the raw task-count completion percentage | Unit/integration test |
| AC-03-035 | Text search returns only client-visible matching projects/items within the client's accessible project set | Search integration test |
| AC-03-036 | Filters for health, task status, decisions needed, blocked, at risk, and overdue decisions affect only client-visible results | Filter integration test |
| AC-03-037 | A connected portal reflects relevant committed updates within 2 seconds when SSE is connected | SSE integration/performance test |
| AC-03-038 | If SSE disconnects, the portal shows live updates paused, remains readable, and manual refresh retrieves current-state data | Failure-mode integration test |
| AC-03-039 | If approval/comment submission is unavailable while reads work, the portal enters read-only degraded mode and disables write controls with explanation | Degraded-mode integration test |
| AC-03-040 | If current-state reads are unavailable, the portal shows an unavailable state and does not present stale data as current | Failure-mode integration test |
| AC-03-041 | Empty states appear with client-friendly explanation and suggested next action for no projects, no active work, no approvals, no risks, no milestones, and no search results | UI behavior test |
| AC-03-042 | Client-facing pages meet WCAG 2.1 AA checks for supported desktop, tablet, and mobile browser layouts | Accessibility audit |
| AC-03-043 | Core portal actions and information remain usable on desktop, tablet, and mobile browser widths | Responsive UI test |
| AC-03-044 | BRD-03 does not introduce CSV/PDF export controls or notification delivery behavior | Scope inspection |
| AC-03-045 | Metrics and log events listed in Observability are emitted for representative portfolio view, project view, approval, comment, publication validation, SSE disconnect, and access denied flows | Observability integration test |

---

## Dependencies / Related BRDs

| Dependency | Relationship |
|------------|--------------|
| BRD-01 App Shell | Provides frontend/backend shell, browser runtime, session/auth integration points, and health endpoint conventions |
| BRD-02 Platform-Native Orchestration Pipeline | Provides project/task/gate/current-state APIs, SSE event stream, audit events, canonical statuses, retention expectations, and project-scoped orchestration data |
| BRD-04 Agent Workstream Dashboard | Owns internal agent-specific status, handoff chains, and operational workstream visibility excluded from BRD-03 |
| BRD-16 Risk Tracking | Owns deeper risk lifecycle and governance; BRD-03 displays client-visible risk information |
| BRD-17 Multi-Tenant Isolation | Owns full tenant lifecycle and enterprise access policy; BRD-03 requires project access filtering for client-visible portal data |
| BRD-18 Collaboration and Team Management | Owns threaded discussion, mentions, attachments, teams, and richer collaboration beyond simple comments |
| BRD-19 Change History | Owns broader artifact/version history and export/change-history policy; BRD-03 records client action/comment metadata and lifetime retention |
| BRD-21 Notifications and Alerts | Owns notification delivery; BRD-03 only shows visible indicators |
| `contracts/openapi.yaml` | Must eventually define client portal read/action endpoints after BRD-03 is approved/curated |
| `contracts/events.md` | Must eventually include or reference client portal decision/comment/publication events where applicable |
| `specs/feature-flags.md` | Must add `client-portal` and preserve/deprecate `dashboard` only through an explicit later decision |

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| BRD-03 scope expands into full collaboration workspace | Medium | High | Keep comments simple, newest-first, non-threaded, no mentions, no attachments; leave full collaboration to BRD-18 |
| Client sees raw internal or technical content | Medium | High | Require publish/review step, publication validation, forbidden technical-field checks, and safe fallback summaries |
| Cross-project access filtering leaks unauthorized data | Medium | High | Treat access boundary failures as security defects; require negative access tests across views, search, counts, comments, approvals, and SSE updates |
| Raw task-count completion misleads clients about actual effort | Medium | Medium | Make calculation transparent, exclude cancelled/proposed items, show health/confidence/milestones alongside percentage |
| Publication workflow adds overhead and delays visibility | Medium | Medium | Require only essential fields: business summary, owner label, next action, visibility status, and no forbidden technical content |
| SSE reliability issues make client portal appear stale | Medium | Medium | Use current-state APIs as authoritative source, show live updates paused, and provide manual refresh fallback |
| Approval states diverge between global inbox and project detail | Medium | High | Treat approval state as a single current-state record and require consistency acceptance tests |
| Comment edit/delete privacy creates audit ambiguity | Low | Medium | Explicitly record audit metadata only and do not retain previous/deleted comment bodies in BRD-03 |
| All risks visible by default exposes sensitive internal concerns | Medium | Medium | Require client-safe risk language and mitigation summaries; later BRD/governance may add explicit risk visibility controls if needed |
| Mobile board layout becomes hard to use at large task counts | Medium | Medium | Require responsive grouped-list behavior and search/filtering; avoid forcing desktop-only board interactions on small screens |
| Overdue decision threshold creates pressure/noise | Low | Medium | Use 24-hour overdue only as a visibility signal; no automatic rejection, approval, or escalation in BRD-03 |

---

## Open Questions

None. The following are intentionally deferred rather than open for BRD-03 implementation:

- Whether to deprecate the existing generic `dashboard` flag is deferred to a later ADR or cleanup BRD.
- Full authentication, external identity, and tenant administration details are deferred to the platform auth layer and BRD-17.
- Export behavior is deferred to BRD-02 and BRD-19.
- Notification delivery is deferred to BRD-21.
- Full collaboration capabilities are deferred to BRD-18.
