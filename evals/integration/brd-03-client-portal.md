# evals/integration/brd-03-client-portal.md

**Project:** agent-orchestrator
**BRD:** BRD-03 — Client Portal and Business Project Board
**Type:** Integration test contracts
**Owner:** qa
**Status:** 🔴 **Failing** (before implementation)

---

## Integration: SSE subscription + per-project event filtering

### Scenario: SSE subscription established for each visible project

**Given**
- Client principal `client:carol` has access to `proj-a`, `proj-b`, `proj-c`
- Carol's browser is connected to the client portal

**When**
Carol navigates to portfolio landing page, then opens `proj-a` project detail

**Then**
- SSE EventSource created for `proj-a` project stream
- EventSource subscribes to BRD-02 SSE endpoint for `proj-a`
- Events from `proj-b` and `proj-c` not delivered to `proj-a` view
- Client-side event handler routes events by `project_id` in envelope

**Then**
- Carol navigates to `proj-b` detail
- New SSE EventSource created for `proj-b`
- `proj-c` SSE connection not opened (not yet viewed)
- On unmount, `proj-a` SSE connection closed

---

### Scenario: SSE event delivery with envelope fields stripped

**Given**
- SSE stream for `proj-alpha` is connected
- BRD-02 emits event with envelope fields: `actorId`, `actorRole`, `eventId`, `schemaVersion`, `parentTaskId`, `gateId`, `layer`

**When**
Event reaches client portal BFF

**Then**
- BFF strips envelope metadata fields before fanout to client
- Client receives event payload with business content only (no actorId, actorRole, eventId, schemaVersion, parentTaskId, gateId, layer)
- `client_portal_sse.connected` event logged on connection establishment
- `client_portal_sse.disconnected` event logged on disconnect

---

### Scenario: SSE delivery latency under 2 seconds

**Given**
- SSE stream connected and client viewing `proj-alpha`
- NFR-03-006 target: relevant committed updates visible within 2 seconds

**When**
Internal actor updates a task status in `proj-alpha`

**Then**
- SSE event arrives at client within 2 seconds
- UI reflects updated state
- `client_portal_project_load_duration_ms` histogram updated

---

### Scenario: SSE disconnect reconnects with backoff

**Given**
- SSE stream for `proj-alpha` is connected

**When**
Connection drops (network failure)

**Then**
- Client shows "Live updates paused" indicator
- BFF attempts reconnection with exponential backoff + jitter
- Max 5 retries per minute (per ADR-03-002)
- After reconnection, manual refresh reconciles state per FR-03-055

---

## Integration: Approval consistency between global inbox and project view

### Scenario: Approval state reflected in both locations after action

**Given**
- Client principal has access to `proj-alpha`
- `proj-alpha` has one pending approval item: `approval-item-1`
- Global inbox shows `approval-item-1` in `pending` state

**When**
Client submits `approve` (no comment) from global inbox

**Then**
- Global inbox updates to `approved` state
- Client navigates to `proj-alpha` project detail
- Project approval inbox shows `approval-item-1` with `approved` state
- No divergence between global inbox and project view
- Audit metadata recorded with actor, timestamp, outcome, project_id, item_id

---

### Scenario: Acting from project view updates global inbox

**Given**
- Client principal has access to `proj-alpha`
- `proj-alpha` has one pending approval item
- Client opens project detail approval inbox

**When**
Client submits `reject` with comment from project view

**Then**
- Project view updates to `rejected` state
- Client returns to portfolio landing
- Global approval inbox shows same item as `rejected`
- Consistency maintained via current-state reads and SSE updates

---

### Scenario: NMI state shown in both locations as waiting-for-response

**Given**
- Client principal has access to `proj-alpha`
- `proj-alpha` has one pending approval item

**When**
Client submits `need_more_information` with comment

**Then**
- Global inbox shows item in `waiting_on_response` state
- Project detail shows same item in `waiting_on_response` state
- Item labeled "Waiting on response" (not shown as rejected or overdue)
- `client_portal_approval.need_more_information` log event emitted

---

## Integration: Publication/republish workflow

### Scenario: Item with all required fields passes validation

**Given**
- Internal project owner submits item for publication to `proj-alpha`

**When**
Item has: `business_summary: "Customer authentication ready for review"`, `owner_label: "Engineering"`, `next_action: "Awaiting client approval"`, `visibility_status: "published"`, no forbidden content

**Then**
- API returns `201 Created` with validated item
- Item becomes visible in client portal
- `client_portal.item.published` audit event recorded
- `client_portal_publication_validation_failed_total` counter unchanged

---

### Scenario: Item missing business summary fails validation

**Given**
- Internal project owner submits item for publication to `proj-alpha`

**When**
Item has `business_summary: ""` (empty) and all other required fields

**Then**
- API returns `400 Bad Request` with `{ field: "business_summary", reason: "missing_required" }`
- Item stays hidden from client portal
- `client_portal_publication_validation_failed_total` incremented with reason "missing_business_summary"
- `client_portal.publication_validation.failed` audit event logged

---

### Scenario: Item with forbidden technical content fails validation

**Given**
- Internal project owner submits item for publication to `proj-alpha`

**When**
Item body field contains: `at com.example.Service.method(File.java:42)` (stack trace)

**Then**
- API returns `400 Bad Request` with reason category "forbidden_content" (not raw content)
- Item stays hidden from client portal
- Forbidden content not logged or stored
- Audit event includes reason category only

---

### Scenario: Republished item after request_changes returns to pending

**Given**
- Client submitted `request_changes` with comment on `approval-item-1`
- Item is in `requested_changes` state

**When**
Internal owner addresses the feedback and republishes the item

**Then**
- Item state transitions from `requested_changes` to `pending`
- Client sees item back in pending state in global inbox and project view
- Overdue clock resets from republish timestamp

---

## Integration: Client portal BFF → BRD-02 API integration

### Scenario: BFF aggregates project list for client principal

**Given**
- Client principal `client:carol` has access to projects `proj-a`, `proj-b` (not `proj-c`)
- BRD-02 API returns full project list

**When**
BFF receives request for portfolio for `carol`

**Then**
- BFF filters projects to only `proj-a`, `proj-b` (carol's accessible set)
- BFF aggregates health counts, decision counts, confidence across accessible projects
- `proj-c` not included in response (not in accessible set)
- Response does NOT contain a 403 or 404 for `proj-c` — no existence signal

---

### Scenario: BFF enforces project access filtering on all endpoints

**Given**
- Client principal `client:carol` has access to `proj-a` only
- BFF receives request for `/projects/proj-b/tasks`

**When**
Request arrives at BFF with `carol`'s session

**Then**
- BFF returns empty result (not 403/404)
- `client_portal_access_denied_total` counter incremented
- `client_portal.access.denied` audit event logged
- No project name or data for `proj-b` included in response

---

### Scenario: BFF falls back to hardcoded owner labels when API unavailable

**Given**
- Client portal BFF cannot reach BRD-02 `/config/owner-mapping` API

**When**
BFF needs to resolve owner label for a task with internal role `engineering`

**Then**
- BFF uses hardcoded default mapping (ADR-03-005)
- Task owner label rendered as "Engineering"
- BFF does NOT fail the request — fallback is graceful
- 5-minute cache TTL ensures repeated API failures don't cause repeated fetch attempts

---

### Scenario: BFF SSE multiplexer — one BRD-02 connection per project

**Given**
- Client portal BFF has 10 connected clients
- Clients collectively have access to 5 projects: `proj-a` through `proj-e`

**When**
BRD-02 project event occurs for `proj-c`

**Then**
- BFF has one SSE connection to BRD-02 for `proj-c`
- BFF fans out event to all subscribed clients with `proj-c` in their accessible set
- Event delivery latency tracked via `client_portal_project_load_duration_ms`

---

## Integration: Comment lifecycle across client and internal views

### Scenario: Comment created by client visible to internal users

**Given**
- Client principal `client:carol` has access to `proj-alpha`
- `proj-alpha` has a risk item

**When**
Carol adds a comment to the risk item

**Then**
- Comment stored with: author `carol`, project `proj-alpha`, item_id, body, timestamp
- Internal users can see the comment on the same item
- Comment shows author display name, timestamp, body, project/item context
- `client_portal_comment_created_total` counter incremented

---

### Scenario: Comment edit visible to both client and internal

**Given**
- Client principal `client:carol` has an existing comment on a risk

**When**
Carol edits the comment

**Then**
- Comment `updated_at` set to edit timestamp
- Comment `is_edited` flag set to true
- Both client view and internal view show edited indicator
- Audit log records edit without previous body
- `client_portal_comment_edited_total` counter incremented

---

### Scenario: Comment delete hides from normal view but audit preserved

**Given**
- Client principal `client:carol` has an existing comment

**When**
Carol deletes the comment

**Then**
- Comment `is_deleted` = true
- Comment hidden from normal client view and internal view
- Audit log records delete action without deleted body
- `client_portal_comment_deleted_total` counter incremented

---

## Integration: Search with project filtering

### Scenario: Search returns only accessible project results

**Given**
- Client principal `client:carol` has access to `proj-a` and `proj-b`
- Term "deployment" appears in `proj-a` (5 matches), `proj-b` (3 matches), `proj-c` (8 matches, no access)

**When**
Carol searches for "deployment"

**Then**
- Results contain 8 total items (5 from `proj-a` + 3 from `proj-b`)
- No results from `proj-c` (carol has no access)
- Each result includes `project_id`, `project_name` for context
- Results filtered to client-visible items only (published, no forbidden content)

---

### Scenario: Search with special regex characters treated as literals

**Given**
- Client principal has access to `proj-alpha`

**When**
Client searches for `test*` (literal asterisk, not regex)

**Then**
- Search treats `*` as literal character, not wildcard
- Returns items containing "test*" literally
- No regex evaluation errors

---

## Integration: Filter combinations

### Scenario: Multiple filters combined with AND

**Given**
- Client principal has access to `proj-alpha`
- Project has items matching various filter criteria

**When**
Client applies filters: health=`at_risk` AND blocked=`true` AND decisions=`pending` AND overdue=`true`

**Then**
- Results include only items satisfying ALL conditions
- Empty state shown with suggested next action when no items match
- Each result is client-visible, within accessible project set

---

## Integration: Read-only degraded mode

### Scenario: Submission fails but reads succeed

**Given**
- Client is viewing project detail for `proj-alpha`
- Read APIs (project data, approvals, comments) are working
- Write API (approval submission, comment POST) is unavailable

**When**
Client attempts to submit an approval outcome

**Then**
- Write request returns error
- Portal enters read-only mode
- Write controls (submit buttons, comment input) disabled
- Explanation shown: "Approval submission temporarily unavailable"
- `client_portal_read_only_mode.entered` event logged
- Read operations unaffected

---

### Scenario: Read APIs unavailable shows unavailable state

**Given**
- Client is viewing the client portal

**When**
All current-state read APIs become unavailable

**Then**
- Portal shows "Project data temporarily unavailable" state
- No stale data shown as current
- Manual refresh button retries
- `client_portal.reads.unavailable` event logged

---

## Integration: Time-controlled overdue test

### Scenario: Approval item becomes overdue after 24 hours

**Given**
- Approval item `approval-1` created in `proj-alpha`
- Item is in `pending` state

**When**
System clock advances to 24h + 1s after item creation

**Then**
- `approval-1` marked as overdue
- Overdue count increments by 1
- `client_portal_overdue_decisions_current` gauge updated
- Portfolio decision summary reflects overdue = 1

**When**
Clock advances another 24h (item now 48h old)

**Then**
- Item remains overdue (overdue is persistent state, not just at threshold)
- Second item created 25h ago becomes overdue
- Overdue count = 2

---

### Scenario: NMI clock behavior — overdue clock pauses

**Given**
- Approval item `approval-1` is 20h old, not yet overdue

**When**
Client submits `need_more_information` with comment

**Then**
- Item state becomes `waiting_on_response`
- Overdue clock pauses (item does not become overdue while waiting)
- `client_portal_approval.need_more_information` event logged

**When**
Client provides requested information 30h after original creation

**Then**
- Item returns to `pending` state
- Overdue clock resumes from the 20h mark (or resets per implementation decision — document which)
- Item becomes overdue at 44h total (20h + 24h) from original creation

---

## Integration: Cross-project isolation across all read paths

### Scenario: Project list filtered to accessible projects

**Given**
- Client principal has access to 2 of 5 projects in the system

**When**
Client requests portfolio project list

**Then**
- Only the 2 accessible projects appear in the list
- Health counts reflect only accessible projects
- No signal about existence of the other 3 projects

---

### Scenario: SSE events for inaccessible projects dropped silently

**Given**
- Client has access to `proj-a` and `proj-b`
- Client's SSE subscription covers all accessible projects

**When**
BRD-02 emits event for `proj-c` (no access)

**Then**
- BFF drops the event
- No event delivered to client
- No notification to client about `proj-c`
- Client cannot infer `proj-c` exists

---

## Integration: Observability — metrics and log events

### Scenario: Portfolio view emits metrics and log events

**Given**
- Client principal `client:carol` opens portfolio landing

**When**
Portfolio page loads and renders

**Then**
- `client_portal_portfolio_view_total` counter incremented
- `client_portal_portfolio_load_duration_ms` histogram recorded
- `client_portal.portfolio.viewed` log event emitted with accessible project count

---

### Scenario: Project view emits metrics and log events

**Given**
- Client principal `client:carol` opens project detail for `proj-alpha`

**When**
Project detail page loads and renders

**Then**
- `client_portal_project_view_total` counter incremented
- `client_portal_project_load_duration_ms` histogram recorded
- `client_portal.project.viewed` log event emitted with project ID

---

### Scenario: Access denied emits counter and log event

**Given**
- Client principal attempts to access unauthorized project

**When**
BFF returns empty result for unauthorized access attempt

**Then**
- `client_portal_access_denied_total` counter incremented
- `client_portal.access.denied` audit event logged

---

*End of BRD-03 integration contracts — 23 integration test scenarios covering all FRs and NFRs*