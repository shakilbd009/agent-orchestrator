# evals/e2e/brd-03-client-portal.md

**Project:** agent-orchestrator
**BRD:** BRD-03 — Client Portal and Business Project Board
**Type:** End-to-end scenario contracts
**Owner:** qa
**Status:** 🔴 **Failing** (before implementation)

---

## E2E Scenario: Portfolio landing — multi-project client access

### Given
- Client principal `client:carol` is authenticated
- `client:carol` has access to projects `proj-a`, `proj-b`, `proj-c`
- Project `proj-a` has health `on_track`, `proj-b` has `at_risk`, `proj-c` has `blocked`

### When
Carol opens the client portal portfolio landing page

### Then
- Portfolio summary shows health counts: On Track = 1, At Risk = 1, Blocked = 1
- Project list shows exactly `proj-a`, `proj-b`, `proj-c` (no other projects)
- Each project card shows: name, health, confidence, completion percentage, next milestone, pending decision count, overdue decision count, latest update timestamp

---

## E2E Scenario: Cross-project isolation — unauthorized project invisible

### Given
- Client principal `client:carol` has access only to `proj-a` and `proj-b`
- Project `proj-c` exists but `carol` has no access

### When
Carol navigates to the portfolio, project list, approval inbox, search, and SSE-connected project views

### Then
- `proj-c` does not appear in portfolio summary health counts
- `proj-c` does not appear in project list
- `proj-c` does not appear in global approval inbox
- Search for content that exists in `proj-c` returns no results for `carol`
- SSE stream for `proj-c` is not subscribed (no connection created)
- Attempting to navigate directly to `/projects/proj-c` returns empty result or redirect (not 403 — per ADR-03-001 security design)

---

## E2E Scenario: Portfolio landing latency at target scale

### Given
- Client principal has access to 50 projects (NFR-03-002 max)
- Each project has up to 10,000 tasks (NFR-03-003)
- All project data is populated

### When
Client principal opens the portfolio landing page

### Then
- High-level summary (health counts, decision counts) visible within 5 seconds (NFR-03-004)
- Progressive loading renders summary before lower-priority details
- Project list populates as data becomes available

---

## E2E Scenario: Project detail view renders within 2 seconds

### Given
- Client principal has access to `proj-alpha`
- `proj-alpha` has 500 tasks in various statuses

### When
Client clicks on `proj-alpha` in the project list

### Then
- Project detail summary (health, completion %, board) visible within 2 seconds (NFR-03-005)
- Task board renders with correct status groupings
- SSE subscription for `proj-alpha` established
- Live updates indicator shows connected state

---

## E2E Scenario: Completion percentage calculation

### Given
- Project `proj-alpha` has tasks in states: `todo` (10), `in_progress` (5), `blocked` (3), `done` (8), `cancelled` (4), `proposed` (2)

### When
Client views `proj-alpha` project detail

### Then
- Completion percentage = `done / (todo + in_progress + blocked + done)` = `8 / (10 + 5 + 3 + 8)` = 8/26 ≈ 31%
- `cancelled` tasks excluded from numerator and denominator
- `proposed` tasks excluded from numerator and denominator
- Cancelled toggle (FR-03-012) shows cancelled tasks with plain-language cancellation reason

---

## E2E Scenario: Zero active tasks shows empty state

### Given
- Project `proj-alpha` has only `done` tasks (denominator = 0 after exclusions)

### When
Client views `proj-alpha` project detail

### Then
- Completion percentage display shows "No active work yet" (not 0%, not 100%)
- Empty state matches FR-03-049 for no active work scenario

---

## E2E Scenario: Project board status mapping

### Given
- Project `proj-alpha` has tasks with BRD-02 canonical statuses: `todo`, `in_progress`, `blocked`, `done`, `cancelled`

### When
Client views the project board

### Then
- Board columns render with client-readable labels (To Do, In Progress, Blocked, Done)
- `cancelled` tasks hidden by default (FR-03-012)
- Enabling "show cancelled" reveals cancelled tasks with cancellation reason

---

## E2E Scenario: Business-language-only display — no raw technical content

### Given
- An internal task in `proj-alpha` has: title "Fix auth bug", internal owner "agent-a1b2c3", branch "feature/auth-fix", commit SHA "abc123def...", stack trace in description, file path `/backend/internal/auth/service.go`

### When
Client views the task card in the client portal

### Then
- Task card shows business-readable title (no agent IDs, branch names, SHAs, or file paths)
- No stack traces, raw logs, agent IDs, or internal terminology visible
- Owner label shows mapped label (e.g., "Engineering") via owner label mapping (FR-03-016/FR-03-017)
- `client_portal_access_denied_total` counter unchanged (no leak detected)

---

## E2E Scenario: Global approval inbox shows all pending decisions

### Given
- Client principal has access to `proj-a` (2 pending approvals) and `proj-b` (3 pending approvals)
- One approval in `proj-a` has been pending 26 hours (overdue)
- One approval in `proj-b` is in `need_more_information` state

### When
Client opens the global approval inbox from portfolio landing

### Then
- Global inbox shows total pending count = 5
- Overdue decision count = 1 (per ADR-03-004, per-item overdue threshold)
- `need_more_information` items shown with "waiting on response" label (FR-03-023)
- Each approval item shows: project name, item title, client-facing owner label, age, current state

---

## E2E Scenario: Approval consistency — global inbox and project detail show same state

### Given
- Client principal has access to `proj-alpha`
- `proj-alpha` has one pending approval item with outcome `pending`

### When
Client acts on the approval from the global inbox with outcome `approve` (no comment required)

### Then
- Approval state updates to `approved` in global inbox
- Global inbox reflects the updated state
- Client navigates to `proj-alpha` project detail
- Project detail approval inbox shows the same item with `approved` state (FR-03-020)
- Actor identity, outcome, timestamp recorded in audit metadata (FR-03-024)

---

## E2E Scenario: Approval outcome — reject requires comment

### Given
- Client principal has access to `proj-alpha`
- `proj-alpha` has one pending approval item in `pending` state

### When
Client submits `reject` on the approval without providing a comment

### Then
- API returns `400 Bad Request` with validation error indicating comment is required
- Approval state remains `pending` (no state change)
- No audit event recorded

---

## E2E Scenario: Approval outcome — reject with comment succeeds

### Given
- Client principal has access to `proj-alpha`
- `proj-alpha` has one pending approval item in `pending` state

### When
Client submits `reject` with comment "Please clarify the scope change"

### Then
- API returns `200 OK` with updated approval state
- Approval state updates to `rejected`
- Comment stored in audit metadata (FR-03-024)
- Project detail and global inbox both reflect `rejected` state

---

## E2E Scenario: Approval outcome — request_changes requires comment

### Given
- Client principal has access to `proj-alpha`
- `proj-alpha` has one pending approval item in `pending` state

### When
Client submits `request_changes` without a comment

### Then
- API returns `400 Bad Request`
- Approval state remains `pending`

---

## E2E Scenario: Approval outcome — request_changes with comment succeeds

### Given
- Client principal has access to `proj-alpha`
- `proj-alpha` has one pending approval item in `pending` state

### When
Client submits `request_changes` with comment "Can you address the risk items first?"

### Then
- API returns `200 OK`
- Approval state updates to `requested_changes`
- Item enters `waiting_on_response` or similar state (FR-03-023)
- Internal owner can address and republish; approval returns to `pending`

---

## E2E Scenario: Approval outcome — need_more_information requires question/comment

### Given
- Client principal has access to `proj-alpha`
- `proj-alpha` has one pending approval item in `pending` state

### When
Client submits `need_more_information` without a question/comment

### Then
- API returns `400 Bad Request`
- Approval state remains `pending`

---

## E2E Scenario: Approval outcome — need_more_information with comment succeeds and does not count as rejection

### Given
- Client principal has access to `proj-alpha`
- `proj-alpha` has one pending approval item in `pending` state

### When
Client submits `need_more_information` with comment "What is the expected delivery date?"

### Then
- API returns `200 OK`
- Approval state becomes `waiting_on_response` (non-terminal, FR-03-023)
- Item does NOT count toward rejected or overdue
- Overdue clock pauses (per implementation of ADR-03-004)
- When client provides information, item returns to `pending`

---

## E2E Scenario: Overdue decision threshold — per-item, 24 hours

### Given
- Project `proj-alpha` has 5 pending approval items
- Item 1 created 25 hours ago (overdue)
- Items 2–5 created within last 2 hours

### When
Client views the global approval inbox

### Then
- Overdue count = 1 (item 1 only — per ADR-03-004 per-item semantics)
- Items 2–5 shown as pending but not overdue
- Portfolio decision summary reflects overdue count = 1

---

## E2E Scenario: Comment creation and display

### Given
- Client principal has access to `proj-alpha`
- `proj-alpha` has one approval item, one risk, and one milestone

### When
Client creates comments on the approval, the risk, and the milestone

### Then
- All three comments created successfully
- Comments are visible to both client and internal users on the relevant items (FR-03-025)
- Comments display newest-first within each item (FR-03-026)
- Each comment shows: author display name, timestamp, `is_edited` indicator when applicable, `updated_at` when edited, project/item context, body

---

## E2E Scenario: Comment edit by author

### Given
- Client principal `client:carol` creates a comment on an approval item

### When
Carol edits her own comment within the allowed window

### Then
- Comment `updated_at` timestamp is set
- Comment `is_edited` flag is `true`
- Audit log records edit metadata without previous body text (FR-03-029)
- Comment remains visible in normal client view

---

## E2E Scenario: Comment delete by author

### Given
- Client principal `client:carol` creates a comment on an approval item

### When
Carol deletes her own comment

### Then
- Comment `is_deleted` flag is `true`
- Comment hidden from normal client view
- Audit log records delete metadata without deleted body text (FR-03-029)
- Comment not visible in project detail or global inbox

---

## E2E Scenario: Comment audit privacy — no body retained in audit

### Given
- Client principal `client:carol` creates a comment "Initial feedback"
- Carol edits the comment to "Revised feedback"
- Carol deletes the comment

### When
Internal administrator reviews audit trail for this comment lifecycle

### Then
- Audit records: create event (timestamp, actor, item, action = "created")
- Audit records: edit event (timestamp, actor, item, action = "edited") — no previous body
- Audit records: delete event (timestamp, actor, item, action = "deleted") — no deleted body
- No version of the comment body text is retained in audit records (FR-03-029)

---

## E2E Scenario: Risk list in project detail

### Given
- Client principal has access to `proj-alpha`
- `proj-alpha` has 3 risks: one `high` severity, one `medium`, one `low`

### When
Client views project detail for `proj-alpha`

### Then
- Risk list shows all three risks (all risks client-visible by default, FR-03-032)
- Each risk shows: severity, impact, client-facing owner label, mitigation summary, status where available, next action
- Risks are displayed independently of task completion percentage (FR-03-035)

---

## E2E Scenario: Milestone list in project detail

### Given
- Client principal has access to `proj-alpha`
- `proj-alpha` has milestones in states: `upcoming`, `in_progress`, `completed`, `at_risk`

### When
Client views project detail for `proj-alpha`

### Then
- Milestone list shows all milestones
- Each milestone shows: name, status, due/target date, progress, health, summary, next action
- Milestone progress does NOT change the task-count completion percentage (FR-03-035)

---

## E2E Scenario: Search across accessible projects only

### Given
- Client principal has access to `proj-a` and `proj-b` (not `proj-c`)
- Term "deployment" appears in tasks in all three projects

### When
Client searches for "deployment"

### Then
- Results include matching tasks from `proj-a` and `proj-b` only
- Results from `proj-c` are not included (no access)
- Each result includes project context (`project_id`, `project_name`)
- Results filtered to client-visible items only

---

## E2E Scenario: Basic filters affect only client-visible results

### Given
- Client principal has access to `proj-alpha`
- Project has tasks in various statuses, risks, and milestones

### When
Client applies filters: health = `at_risk`, task status = `in_progress`, decisions needed = `pending`, overdue = `true`

### Then
- Filtered results match all selected criteria simultaneously
- Results show only client-visible items (no unpublished, no unauthorized)
- Empty state shown with suggested next action when no results match
- No results from inaccessible projects

---

## E2E Scenario: SSE real-time updates — delivery within 2 seconds

### Given
- Client principal has access to `proj-alpha` and is viewing the project detail board
- Client is SSE-connected to `proj-alpha`

### When
Internal actor updates a task status from `in_progress` to `done`

### Then
- SSE event arrives at client within 2 seconds (NFR-03-006)
- UI updates to reflect new task status
- `client_portal_project_view_total` counter incremented

---

## E2E Scenario: SSE disconnect shows live updates paused

### Given
- Client is connected to `proj-alpha` SSE stream and viewing project detail

### When
SSE connection is dropped (network-level disconnect)

### Then
- Portal shows "Live updates paused" indicator (FR-03-041)
- Portal remains usable for reads via current-state APIs
- Manual refresh button is available and functional
- `client_portal_sse_disconnect_total` counter incremented

---

## E2E Scenario: SSE reconnect reconciliation

### Given
- Client was SSE-connected to `proj-alpha`, disconnected for 2 minutes, reconnected

### When
SSE reconnection succeeds and client triggers manual refresh

### Then
- Current-state API returns latest data
- Client reconciles SSE events since reconnect
- No duplicate events rendered in UI
- Final state matches authoritative current-state API

---

## E2E Scenario: Publication validation — missing required fields fails

### Given
- Internal project owner attempts to publish an item to client portal

### When
Item is submitted with business-language summary missing, owner label missing, or next action missing

### Then
- API returns `400 Bad Request` with field-level error
- Item stays hidden from client portal
- `client_portal_publication_validation_failed_total` incremented with reason category
- Audit event `client_portal.publication_validation.failed` logged

---

## E2E Scenario: Publication validation — forbidden technical content fails

### Given
- Internal project owner attempts to publish an item containing forbidden technical content

### When
Item fields include: stack trace (`at com.example.Foo.method`), branch name (`feature/auth`), commit SHA (`abc123def...`), file path (`/backend/internal/service.go`), agent ID (`agent-a1b2c3`), or structured log output (`[ERROR] Exception`)

### Then
- API returns `400 Bad Request` with reason category (not raw content)
- Item stays hidden from client portal
- BFF strips envelope metadata fields (actorId, actorRole, eventId, schemaVersion, parentTaskId, gateId, layer) from SSE fanout

---

## E2E Scenario: Unpublish removes item from client view

### Given
- An item is published and visible in the client portal

### When
Internal project owner unpublishes the item via audited internal action

### Then
- Item disappears from client portal views
- Audit event `client_portal.item.unpublished` recorded
- Generic unpublished blocker appears if item was blocking a published milestone (FR-03-046)

---

## E2E Scenario: Generic unpublished blocker — no raw detail exposed

### Given
- Published milestone `M1` is blocked by unpublished internal task `T-internal`

### When
Client views the milestone detail

### Then
- Client sees only generic message: "Internal dependency is blocking progress"
- No unpublished task title, raw detail, or internal content exposed
- Content safety verified — no raw technical details visible

---

## E2E Scenario: Read-only degraded mode when submission unavailable

### Given
- Client principal is viewing the client portal with read APIs available

### When
Approval/comment submission API becomes unavailable (write service down)

### Then
- Portal remains readable
- Write controls (submit approval, add comment) are disabled
- Plain-language explanation shown: "Approval submission temporarily unavailable"
- `client_portal_read_only_mode.entered` event logged
- Read operations (portfolio, project detail, search) continue to work

---

## E2E Scenario: Read failure behavior — no stale data as current

### Given
- Client principal is viewing the client portal

### When
Current-state read APIs become unavailable

### Then
- Portal shows "Project data temporarily unavailable" state
- No stale data presented as current
- Manual refresh button retries the request
- `client_portal.reads.unavailable` event logged

---

## E2E Scenario: Empty states across all contexts

### Given
- Client principal has access but various resources have no content

### When
Client views empty scenarios: no accessible projects, no active work, no approvals, no risks, no milestones, empty search results

### Then
- Each empty state shows client-friendly message and suggested next action (FR-03-049)
- Examples:
  - No projects: "You don't have access to any projects yet. Contact your administrator to request access."
  - No active work: "No active work yet"
  - No approvals: "No pending decisions"
  - No search results: "No results for '[query]'. Try adjusting your filters."

---

## E2E Scenario: Client portal feature flag — disabled state

### Given
- `client-portal` feature flag is set to `false`

### When
Client or internal user attempts to access client portal routes

### Then
- Client portal routes return `404` or disabled-state response
- API behavior consistent with project feature-flag conventions (AC-03-001)
- No client portal UI rendered

---

## E2E Scenario: Owner label mapping — default and override

### Given
- Project `proj-alpha` has task with internal owner metadata `engineering`
- Another task has `client_owner_label_override` set to "Dev Team"

### When
Client views both tasks in project detail

### Then
- First task shows owner label "Engineering" (from default mapping, ADR-03-005)
- Second task shows owner label "Dev Team" (override takes precedence)
- If API is unavailable and BFF falls back to hardcoded defaults, mapping still applies

---

## E2E Scenario: Cross-project SSE — per-project connections only

### Given
- Client principal has access to 3 projects
- Client is viewing portfolio landing

### When
Client opens project detail for `proj-alpha`

### Then
- SSE EventSource created for `proj-alpha` only
- SSE connections to `proj-beta` and `proj-proj-gamma` not opened until those projects are viewed
- On unmount/navigation away, `proj-alpha` SSE connection closed
- Up to 50 concurrent SSE connections supported (NFR-03-002)

---

*End of BRD-03 E2E scenarios — 42 scenarios covering all FRs and NFRs*