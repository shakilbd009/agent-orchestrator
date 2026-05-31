# evals/unit/brd-03-client-portal.md

**Project:** agent-orchestrator
**BRD:** BRD-03 — Client Portal and Business Project Board
**Type:** Unit test contracts
**Owner:** qa
**Status:** 🔴 **Failing** (before implementation)

---

## Contract: CompletionPercentageCalculator

### Input
- List of task objects with `status` field
- Task statuses: `todo`, `in_progress`, `blocked`, `done`, `cancelled`, `proposed`

### Output
- If denominator `(todo + in_progress + blocked + done) > 0`: percentage = `done_count / denominator * 100`, rounded to integer
- If denominator == 0: `null` (indicates empty state)

### Edge cases:
- All tasks `cancelled`: denominator = 0 → return `null`
- All tasks `proposed`: denominator = 0 → return `null`
- Mix of `done` and `cancelled` only: denominator counts only `done` → returns 100% only if `done > 0`
- `proposed` tasks excluded from both numerator and denominator
- `cancelled` tasks excluded from both numerator and denominator

### Verification
- Input: [todo:5, in_progress:3, blocked:1, done:4, cancelled:2, proposed:1] → Output: 4/(5+3+1+4) = 4/13 ≈ 31%
- Input: [done:10] → Output: 10/10 = 100%
- Input: [cancelled:5] → Output: `null`
- Input: [proposed:3] → Output: `null`
- Input: [] → Output: `null`

---

## Contract: ApprovalStateMachine

### States
- `pending` — initial state, awaiting client action
- `approved` — terminal state
- `rejected` — terminal state
- `requested_changes` — non-terminal, waiting for internal revision
- `waiting_on_response` — non-terminal (NMI), waiting for client-provided information

### Valid transitions
- `pending` → `approved` (approve action, comment optional)
- `pending` → `rejected` (reject action, comment required)
- `pending` → `requested_changes` (request_changes action, comment required)
- `pending` → `waiting_on_response` (need_more_information action, comment required)
- `requested_changes` → `pending` (internal owner republishes item)
- `waiting_on_response` → `pending` (client provides requested information)

### Invalid transitions
- `approved` → any state (terminal)
- `rejected` → any state (terminal)
- `pending` → `approved` without actor identity
- Any state → `pending` except via allowed transitions above

### Edge cases:
- Concurrent actions on same item: server timestamp ordering, last-write-wins
- NMI does not count as rejection in any aggregate or count metric

---

## Contract: OwnerLabelMapper

### Input
- Internal role string from task metadata
- Override label string (optional)

### Output
- If `client_owner_label_override` present and non-empty: return override label
- Else: return default mapped label from table

### Default mapping table (ADR-03-005)
| Internal Role | Default Client-Facing Label |
|--------------|-----------------------------|
| `product_manager`, `product_owner`, `pm` | Product |
| `engineering`, `backend`, `frontend`, `developer` | Engineering |
| `reviewer`, `qa`, `quality_assurance`, `testing` | Review |
| `quality`, `architect`, `design` | Quality |
| `client`, `client_stakeholder`, `external` | Client |
| (unmapped) | Product (fallback) |

### Edge cases:
- Role string with whitespace padding: trimmed before lookup
- Case-insensitive role matching
- Empty override string: falls back to default mapping
- Null override: falls back to default mapping

---

## Contract: SSEEventFilter

### Input
- SSE event payload with envelope fields and business content
- Client principal's accessible project set
- Project ID from event

### Output
- If `event.project_id` in accessible project set: pass event through
- If `event.project_id` NOT in accessible project set: drop event
- Envelope metadata fields stripped before client delivery (actorId, actorRole, eventId, schemaVersion, parentTaskId, gateId, layer)

### Edge cases:
- Event with no project_id: log warning, drop event
- Event for project client just lost access to: drop, no notification
- BFF SSE multiplexer: one connection to BRD-02 per project, fanout to clients

---

## Contract: PublicationValidator

### Input
- Item with fields: `business_summary`, `owner_label`, `next_action`, `visibility_status`, `body`, `blocked_reason`, `summary`

### Output
- If all required fields present and no forbidden content: `{ valid: true }`
- If missing required fields: `{ valid: false, reason: "missing_field", fields: ["business_summary"] }`
- If forbidden content detected: `{ valid: false, reason: "forbidden_content", pattern: "stack_trace" }`

### Required fields
- `business_summary`: non-empty string, no forbidden patterns
- `owner_label`: non-empty string
- `next_action`: non-empty string
- `visibility_status`: must be `published` or `unpublished`

### Forbidden content patterns (ADR-03-003)
- Stack traces: lines starting with `at `, `Traceback`, `Exception`, `Error:`, `panic:`
- Agent IDs: `agent-<hex>` pattern
- Branch names: `refs/heads/*`, `feature/*`, `fix/*`, `hotfix/*`, `release/*`
- Commit SHAs: 40-character hex `[a-f0-9]{40}`
- File paths: containing `/src/`, `/internal/`, `/backend/`, `/agent/`, `/pkg/`, `/cmd/`
- Infrastructure jargon: `docker`, `kubernetes`, `pod`, `deployment`, `service mesh`, `container`, `namespace`, `kubeconfig`, `helm`, `ingress`, `sidecar`
- Log output: lines starting with `[DEBUG]`, `[INFO]`, `[WARN]`, `[ERROR]`, `[TRACE]`
- Go runtime: `panic:`, `goroutine`, `runtime.gopanic`
- Internal role values: `layer_a`, `layer_b` as string values
- Envelope metadata fields in payload: `actorId`, `actorRole`, `eventId`, `schemaVersion`, `parentTaskId`, `gateId`

### Edge cases:
- Field value is `null`: treated as missing, validation fails
- Empty string: validation fails for required fields
- Whitespace-only string: validation fails
- Case-insensitive pattern matching for forbidden content
- Metadata object: keys must not be `blockedReason`, `owner_override`, `internal_tags`, `execution_agent`

---

## Contract: OverdueDecisionThreshold

### Input
- Approval item with `created_at` timestamp (when item became visible/actionable)
- Current server timestamp

### Output
- If `now - created_at > 24 * 3600 * 1000` ms: `{ overdue: true, age_hours: N }`
- Else: `{ overdue: false, age_hours: N }`

### Edge cases:
- Exactly 24h (86400000ms): `overdue: false` (must be strictly greater)
- At 24h + 1ms: `overdue: true`
- Per-item semantics: each item has independent clock (ADR-03-004)
- NMI/waiting_on_response: overdue clock pauses at NMI submission (documented behavior)

---

## Contract: CommentPrivacyManager

### Input
- Comment lifecycle event: `create`, `edit`, `delete`
- Comment body text (for create/edit)
- Actor identity

### Output (audit record)
- Create: `{ action: "created", actor, timestamp, project_id, item_id, comment_id }` — no body stored in audit
- Edit: `{ action: "edited", actor, timestamp, project_id, item_id, comment_id }` — no previous body
- Delete: `{ action: "deleted", actor, timestamp, project_id, item_id, comment_id }` — no deleted body

### Edge cases:
- Deleted comment body: not retained anywhere after delete action completes
- Edited comment previous body: not retained after edit confirmed
- Audit log must be append-only, immutable

---

## Contract: XSSSanitizer

### Input
- Any string rendered in client-facing UI (project name, task title, owner label, summary, risk description, comment body, etc.)

### Output
- All HTML special characters escaped: `<` → `&lt;`, `>` → `&gt;`, `&` → `&amp;`, `"` → `&quot;`, `'` → `&#39;`
- Script tags removed or escaped
- Event handler attributes stripped
- SVG/math tags sanitized

### Edge cases:
- Project name with `<script>alert(1)</script>`: rendered as text, not executed
- Task title with `<img src=x onerror=alert(1)>`: sanitized, not executed
- Comment body with `javascript:` URLs in href: stripped or neutralized
- Unicode smiley face and emoji: preserved (no XSS vector)
- Long strings: truncation at 500 chars with `...` preserving safety

### Verification
- Input: `<script>alert('xss')</script>` → Output: `&lt;script&gt;alert('xss')&lt;/script&gt;`
- Input: `<img src=x onerror=alert(1)>` → Output: `&lt;img src=x onerror=alert(1)&gt;` (or stripped)

---

## Contract: BFFProjectAccessFilter

### Input
- Client principal identity
- Requested resource (project ID, task ID, item ID)
- BFF local access control state

### Output
- If resource belongs to accessible project set: pass through
- If resource does NOT belong to accessible project set: return empty result (not 403/404 — per ADR-03-001 security design)

### Edge cases:
- Request for non-existent project: return empty result
- Request for project client was granted access to mid-session: access recognized on next navigation
- SSE event for project now inaccessible: dropped silently
- BFF must never return 403/404 for unauthorized access (presence/absence of project revealed)

---

## Contract: EmptyStateRenderer

### Input
- Resource context: `portfolio`, `project_detail`, `task_board`, `approval_inbox`, `risk_list`, `milestone_list`, `search_results`

### Output
- Appropriate empty state message and suggested next action

### Expected messages:
- Portfolio with no accessible projects: "You don't have access to any projects yet. Contact your administrator to request access."
- Project with no active work: "No active work yet"
- Approval inbox empty: "No pending decisions"
- Risk list empty: "No risks identified"
- Milestone list empty: "No milestones defined"
- Search results empty: "No results for '[query]'. Try adjusting your filters."

---

## Contract: NextActionResolver

### Input
- Published client-visible item (task, risk, milestone, approval)

### Output
- Exactly one current client-facing next action string
- If no client action needed: internal next step in client-safe language

### Edge cases:
- Item with no next action defined: return "No action required" or equivalent
- Multiple next actions possible: return the single most urgent/appropriate
- Next action must be client-safe (no internal jargon)

---

## Contract: CancelledTaskVisibility

### Input
- Task with `status: cancelled`
- Client toggle state: `show_cancelled` (default: false)

### Output
- If `show_cancelled == false`: task hidden from board
- If `show_cancelled == true`: task visible with cancellation reason in plain language

### Edge cases:
- Cancellation reason absent: show generic "This task was cancelled"
- `cancelled` tasks excluded from completion percentage calculation regardless of toggle

---

*End of BRD-03 unit contracts — 12 contracts covering all unit-testable behavior*