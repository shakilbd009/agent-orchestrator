# BRD-03 — Edge Cases L2

**BRD:** BRD-03-client-portal
**Stage:** 05-edge-cases
**Status:** Analyzed

---

## Edge Case Categories

### Category 1: Data Boundaries

#### E1.1 — Empty portfolio (no accessible projects)
**What happens:** Portfolio landing shows empty state with suggested next action.
**Handling:** "You don't have access to any projects yet. Contact your administrator to request access."
**How to test:** Create client principal with zero accessible projects; verify empty state.

#### E1.2 — Zero active tasks in project
**What happens:** Completion percentage shows empty state "No active work yet" instead of 0% or 100%.
**Handling:** Backend returns `null` for completion_percentage when denominator is zero.
**How to test:** Project with only `done` tasks (denominator = 0) → "No active work yet". Project with only `cancelled` tasks → same.

#### E1.3 — Maximum scale — 50 projects, 10,000 tasks per project
**What happens:** Portfolio page loads within 5s (NFR-03-004). Project detail loads within 2s (NFR-03-005). Task board uses virtual scrolling.
**Handling:** BFF parallelizes project fetches. Virtual scrolling for large task lists. SSE connections managed per-project.
**How to test:** Load test with 50 accessible projects; performance test with 10,000-task project.

#### E1.4 — Project name with special characters
**What happens:** Project name rendered correctly with HTML-escaped special chars (no XSS).
**Handling:** BFFescapes output. Frontend uses SvelteKit's automatic HTML escaping.
**How to test:** Inject `<script>` in project name; verify not executed in browser.

#### E1.5 — Very long task title (>500 chars)
**What happens:** Task title truncated in card view with "...". Full title visible in task detail.
**Handling:** UI truncation with title attribute for full text. No data loss.
**How to test:** Create task with 1000-char title; verify truncation and tooltip.

---

### Category 2: State Transitions

#### E2.1 — Approval state transition: pending → need_more_information → pending
**What happens:** Client submits `need_more_information`. Item enters `waiting_on_response` state. Client provides information. Item returns to `pending`. NMI does not count as rejection.
**Handling:** State machine tracks `waiting_on_response` as a distinct non-terminal state. Only `approve`, `reject`, `request_changes` are terminal.
**How to test:** Submit nmi → verify state = `waiting_on_response`. Provide info → verify state returns to `pending`. Verify overdue clock resets.

#### E2.2 — Approval state transition: request_changes → pending (resubmission)
**What happens:** Client submits `request_changes` with comment. Item enters `requested_changes` state. Internal owner addresses changes and republishes. Item returns to `pending` for client re-review.
**Handling:** State machine allows `requested_changes` → `pending` transition on republished item.
**How to test:** Submit request_changes → verify state. Republish item → verify state returns to `pending`.

#### E2.3 — Concurrent approval actions by same client
**What happens:** Client submits `approve` on item at T=0. Before server confirms, client submits `reject` at T=1. Server order: reject arrives first, then approve (race condition).
**Handling:** Last-write-wins with server-side timestamp ordering. Client refreshes to see authoritative state.
**How to test:** Submit concurrent approvals; verify final state reflects server order; verify client shows updated state.

#### E2.4 — Concurrent approval actions by different clients (same principal)
**What happens:** Two browser sessions for same client principal. Both submit different outcomes on same item.
**Handling:** Same as E2.3 — server processes both; last-write-wins; both sessions reconcile on next refresh.
**How to test:** Two sessions; different outcomes; verify server resolves one winner.

#### E2.5 — Task status transition while client is viewing board
**What happens:** Task moves from `in_progress` to `done` via SSE. Client sees update without refresh.
**Handling:** SSE event contains `{ event_type: 'task.updated', task_id, new_status, updated_at }`. Client store updates; UI re-renders.
**How to test:** Open project detail; trigger task status change; verify SSE update reflects within 2s.

#### E2.6 — Item unpublished while client is viewing it
**What happens:** Internal owner unpublishes an item client is viewing. Client sees item disappear (or generic unavailable message if it was a blocker).
**Handling:** SSE event `{ event_type: 'item.unpublished', item_id, project_id }`. Client removes item from view.
**How to test:** Publish item; view in client portal; unpublish via internal action; verify item disappears from client view.

---

### Category 3: Timing

#### E3.1 — Overdue decision threshold exactly at 24h
**What happens:** Decision pending exactly 24h + 1 second becomes overdue.
**Handling:** Server-side check `now() - created_at > 24 * 3600 * 1000`. Uses wall-clock, not business hours.
**How to test:** Create approval item; advance clock 24h+1s in test environment; verify overdue flag set.

#### E3.2 — SSE disconnect during active session
**What happens:** SSE connection drops. Client shows "Live updates paused" indicator. Portal remains usable via current-state APIs.
**Handling:** On SSE disconnect event: show indicator, enable manual refresh. BFF re-establishes SSE connection with backoff.
**How to test:** Disconnect SSE; verify "Live updates paused" shown. Use manual refresh; verify current data returned.

#### E3.3 — SSE reconnect after extended disconnect
**What happens:** SSE reconnects after 5 minutes. Client reconciles state against current-state APIs per FR-03-055.
**Handling:** On reconnect: fetch `GET /projects/<id>` to reconcile. Merge SSE events from reconnect window. Detect and handle any state conflicts.
**How to test:** Disconnect SSE for 5min; reconnect; verify state matches current-state API; verify no duplicate events in UI.

#### E3.4 — Manual refresh during SSE-connected session
**What happens:** Client clicks refresh. Current-state API called. SSE continues running. SSE events continue to arrive.
**Handling:** Manual refresh updates state. SSE events update state. Final state = current-state API result + SSE events since load.
**How to test:** SSE-connected; trigger manual refresh; verify page updates; SSE continues running.

#### E3.5 — Late-arriving SSE event (event for item client no longer has access to)
**What happens:** Client's access to project is revoked mid-session. SSE event arrives for that project.
**Handling:** BFF filters events by accessible project set. If project no longer in principal's accessible set, event dropped. Access change recognized on next navigation.
**How to test:** Revoke project access; trigger SSE event for that project; verify event dropped; verify client cannot navigate to that project.

---

### Category 4: Integration

#### E4.1 — BRD-02 project API returns 503
**What happens:** Portfolio page shows "Project data temporarily unavailable" with retry button. No stale data shown.
**Handling:** BFF catches 503 from BRD-02. Returns error response. Frontend shows unavailable state. Manual refresh retries.
**How to test:** Mock BRD-02 503; verify unavailable state; verify refresh button works.

#### E4.2 — BRD-02 approval API unavailable but project read API works
**What happens:** Portal shows project detail (reads work) but approval inbox shows "Approvals temporarily unavailable" with explanation. Write controls disabled (read-only degraded mode per FR-03-053).
**Handling:** Read-only degraded mode. `client_portal_read_only_mode.entered` event logged.
**How to test:** Mock approval API failure; verify read-only mode; verify write controls disabled with explanation.

#### E4.3 — BRD-02 SSE event stream unavailable
**What happens:** SSE connection fails. Client shows "Live updates paused". Manual refresh continues to work via current-state APIs.
**Handling:** `client_portal_sse_disconnect_total` incremented. Reconnection attempted with backoff.
**How to test:** Mock SSE stream failure; verify disconnect indicator; verify manual refresh works.

#### E4.4 — Publication validation fails due to forbidden technical content
**What happens:** Internal user tries to publish item containing stack trace in summary. API returns 400 with reason.
**Handling:** Item stays hidden. User sees error with field-level feedback. Item not visible in client portal.
**How to test:** Submit item with stack trace; verify 400; verify item not visible to client.

#### E4.5 — Publication validation fails due to missing required fields
**What happens:** Internal user tries to publish item with empty business summary. API returns 400 with missing field details.
**Handling:** Item stays hidden. User sees field-level error message.
**How to test:** Submit item with missing fields; verify 400; verify item not visible.

#### E4.6 — Comment submission fails but read works
**What happens:** Comment POST returns error. Existing comments visible. New comment input shows error with retry.
**Handling:** `client_portal_submission_failed_total` incremented. User sees error. Can retry.
**How to test:** Mock comment POST failure; verify error shown; verify existing comments still visible.

#### E4.7 — BRD-02 API returns non-200 with error details
**What happens:** API returns `{ error: "project_not_found", message: "..." }`. BFF returns same to client with appropriate status code.
**Handling:** BFF passes through error details; client handles gracefully. Unauthorized access returns empty result (not 404/403 per security design).
**How to test:** Mock various API error responses; verify client handles gracefully.

#### E4.8 — Search returns results from multiple accessible projects
**What happens:** Search `q=deployment` returns tasks across projects A, B, C (all accessible). Each result includes project context.
**Handling:** BFF searches BRD-02 with principal filter. Results include `project_id`, `project_name` for context. Client renders grouped by project or flat with project badge.
**How to test:** Search for term that exists in multiple projects; verify results from all accessible projects; verify no results from inaccessible projects.

---

### Category 5: User Behavior

#### E5.1 — Client submits approval with optional comment on approve
**What happens:** Client approves with comment. Approval recorded with outcome `approve` and comment. Both stored in audit.
**Handling:** `approve` accepts optional comment. Audit log includes comment.
**How to test:** Submit approve with comment; verify stored; verify comment visible in approval history.

#### E5.2 — Client submits rejection without required comment
**What happens:** Client submits `reject` without comment. API returns 400 with validation error.
**Handling:** Comment required for reject (FR-03-022). Validation prevents empty comment.
**How to test:** Submit reject without comment; verify 400; verify no state change.

#### E5.3 — Client edits own comment within 5 minutes
**What happens:** Author edits comment. `updated_at` set. `is_edited` flag set. Audit log records edit without previous body.
**Handling:** Edit within 5min allowed (reasonable window). After 5min, edit disabled? (BRD-03 doesn't specify a time limit; implementation decision.)
**How to test:** Create comment; edit within window; verify `updated_at` changed; verify `is_edited` true.

#### E5.4 — Client deletes own comment
**What happens:** Author deletes comment. `is_deleted` true. Comment hidden from normal view. Audit log records delete without body.
**Handling:** Audit metadata only. Deleted body not retained.
**How to test:** Create comment; delete; verify hidden from normal view; verify audit record exists with delete metadata.

#### E5.5 — Client tries to edit another client's comment
**What happens:** Edit/delete controls shown only for own comments (enforced at UI and API level).
**Handling:** API rejects edit/delete for non-author. UI doesn't show controls for other authors' comments.
**How to test:** Client A creates comment; Client B attempts edit; verify 403; verify controls not rendered.

#### E5.6 — Client uses search with special regex characters
**What happens:** Search query contains `.*+?^${}()|`. Treated as literal string, not regex (FR-03-038 text search).
**Handling:** Search API escapes special characters or disables regex mode.
**How to test:** Search for `test*`; verify literal `*` search; verify no regex evaluation.

#### E5.7 — Client filters by multiple criteria simultaneously
**What happens:** Filters: health=at_risk, blocked AND decisions=pending AND overdue=true. Combined filter shows intersection.
**Handling:** Filters combined with AND. If no results, empty state shown.
**How to test:** Apply multiple filters; verify results match all criteria; verify empty state with suggested next action.

---

### Category 6: Content Safety

#### C6.1 — Unpublished task blocks published milestone
**What happens:** Client views milestone. Unpublished internal task blocks it. Client sees only "Internal dependency is blocking progress" — no task title, no raw detail.
**Handling:** BFF detects unpublished blocker. Returns generic message. Never exposes unpublished task title.
**How to test:** Create published milestone blocked by unpublished task; verify client sees generic message; verify no task title in response.

#### C6.2 — Internal agent ID appears in SSE event payload
**What happens:** BRD-02 SSE event contains internal agent ID. BFF strips forbidden content before forwarding to client.
**Handling:** BFF event processor removes agent IDs, commit SHAs, branch names, file paths from event payload.
**How to test:** Inject event with agent ID; verify stripped from client-facing event.

#### C6.3 — Task title contains raw log output
**What happens:** Internal user sets task title to `[ERROR] Exception in thread main`. BFF detects forbidden content on publication validation.
**Handling:** Publication validation fails. Item stays hidden. User sees error with forbidden content type.
**How to test:** Submit task with log-like title; verify publication fails; verify item not visible.

#### C6.4 — Safe fallback used for item with no client-safe summary
**What happens:** Task has no `client_safe_summary` field populated. Client sees "Internal work item awaiting summary".
**Handling:** BFF detects missing summary. Returns fallback. Client renders fallback text.
**How to test:** Create task with no summary; verify fallback shown; verify raw internal content not exposed.

---

## Summary Table

| Category | Count | Highest Severity |
|----------|-------|-------------------|
| Data Boundaries | 5 | Low |
| State Transitions | 6 | High (concurrent approval race) |
| Timing | 5 | Medium |
| Integration | 8 | High (read-only degraded mode) |
| User Behavior | 7 | Low |
| Content Safety | 4 | Critical (technical content leak) |

**Critical edge cases (must pass):**
- E2.3 (concurrent approvals)
- E3.5 (late-arriving event for revoked access)
- C6.1 (unpublished blocker leak)
- C6.2 (agent ID in SSE event)

---

*Edge case discovery complete — 35 edge cases identified across 6 categories*
*Next: Stage 06 design L3 (if required) or finalize for graduation*