# BRD-03 — Progressive Deepening L2

**BRD:** BRD-03-client-portal
**Stage:** 05-design-l2
**Status:** Analyzed

---

## Component L2 Analysis

### 1. Portfolio Landing — L2

**How:** The landing page fetches all accessible projects in a single aggregated call via the BFF. The BFF returns project metadata including health, confidence, completion%, pending/overdue decision counts, and latest update timestamp. The page renders a responsive grid of project cards.

**Interactions:**
- Initial load: `GET /portfolio?principal=<id>` → BFF → BRD-02 project list API
- SSE: per-project EventSources opened for each visible project; counts updated live
- Search: `GET /portfolio/search?q=<query>&principal=<id>` → filtered to accessible projects
- Filters: query params `?health=on_track,at_risk&decisions=pending,overdue`

**Alternatives considered:**
- Separate per-project calls for landing page: rejected (would require 50+ calls for full portfolio)
- BFF caching project list: considered but not needed for initial load (data freshness requirement)

**Edge cases:**
- Empty project list: show empty state "No projects available — contact your administrator"
- Partial data (some projects fail to load): show available projects with warning "Some project data unavailable"
- Portfolio with 50 projects exceeds viewport: virtual scrolling or pagination with "Show more" control

**Risks:**
- Portfolio load latency with 50 projects at NFR-03-004 5s target: BFF must parallelize project fetches
- SSE connection churn: when portfolio changes (project added/removed from access list), existing EventSources must be closed and new ones opened

---

### 2. Project Detail View — L2

**How:** The project detail page fetches task board, approvals, risks, and milestones in parallel via the BFF. The BFF applies access filtering and owner label mapping. The board renders with BRD-02 canonical statuses mapped to client-readable labels.

**Interactions:**
- Initial load: `GET /projects/<id>` → BFF → parallel fetch tasks, approvals, risks, milestones from BRD-02
- Task board: `GET /projects/<id>/tasks?status=todo,in_progress,blocked,done&visible=true` (cancelled excluded unless toggled)
- Completion calculation: `done_tasks / (todo + in_progress + blocked + done)` tasks; cancelled/proposed excluded
- SSE: per-project EventSource for live board updates

**Alternatives considered:**
- Single combined API endpoint for project detail: simpler but less flexible; BFF could aggregate
- Client-side completion calculation vs. backend-computed: backend provides calculated field; client renders

**Edge cases:**
- Zero active tasks: completion shows empty state "No active work yet" (not 0% or 100%)
- Cancelled tasks: hidden by default; "show cancelled" toggle reveals with cancellation reason
- No risks: show "No risks identified" with suggested next action
- No milestones: show "No milestones defined" with suggested next action
- Long task list (10,000 tasks per NFR-03-003): virtual scrolling; pagination; status-grouped view with collapse

**Risks:**
- 10,000-task project rendering performance: virtual scrolling required; cannot render all DOM nodes
- Task detail for blocked items: dependency/blocker reason must be client-safe (no unpublished task titles)

---

### 3. Global Approval Inbox — L2

**How:** The inbox aggregates pending approval items across all accessible projects. The BFF fetches from BRD-02 approval API with project filter. SSE subscriptions cover all accessible projects to update inbox counts in real time.

**Interactions:**
- Initial load: `GET /approvals?principal=<id>&state=pending` → BFF → BRD-02 approval API
- Actions: `POST /approvals/<id>/decide` with `{ outcome, comment? }`
- Overdue: computed from `created_at` vs `now()` (per ADR-03-004)

**Approval state machine:**

```
                    ┌─────────────────────────────┐
                    │                             │
    ┌───────────────│      pending                │◄────────── (initial state)
    │               │                             │
    │               └──────────┬──────────────────┘
    │                          │
    │         ┌────────────────┼────────────────┐
    │         │                │                │
    │         ▼                ▼                ▼
    │  ┌─────────────┐  ┌───────────────┐  ┌───────────────┐
    │  │   approve   │  │ need_more_info │  │request_changes│
    │  └─────────────┘  └───────────────┘  └───────────────┘
    │                                                 │
    │                           ┌────────────────────┘
    │                           │
    │                           ▼
    │                   ┌───────────────┐
    │                   │ waiting_on_   │ (nmi does NOT count as rejection)
    │                   │ response      │
    │                   └───────────────┘
    │                            │
    │         (client responds with info)           │
    │                            │                  │
    │                            └──────────────────┘
    │                                              │
    │                                              ▼
    │                                    (back to pending)
    │
    │  ┌─────────────┐
    └──│   reject    │
       └─────────────┘
         (terminal)
```

**Edge cases:**
- Approval item becomes visible but client is offline: 24h clock starts regardless; item shows as overdue on return
- Client acts on item while SSE is disconnected: manual refresh reconciles current state per FR-03-042
- Multiple clients (same principal) act on same item simultaneously: last-write-wins with conflict detection; both see updated state after refresh

**Risks:**
- 50 projects with avg 5 pending approvals = 250 inbox items; inbox must paginate or virtual-scroll
- SSE events for approval state changes must propagate to both global inbox and project detail view simultaneously

---

### 4. Publication Validation Service — L2

**How:** Internal project owner initiates publication. BFF receives publication request with item metadata. Shared validation schema checks required fields and forbidden content. On success, item becomes client-visible. On failure, item stays hidden with error details.

**Validation schema:**
```yaml
required_fields:
  - business_summary        # plain-language description, 10-500 chars
  - client_owner_label      # from allowed label set
  - next_action             # exactly one, plain-language, max 200 chars
  - visibility_status       # "client_visible" (only valid value for BRD-03)
forbidden_patterns:
  - stack_trace_lines       # lines matching `^\s+at\s+` or `Traceback`
  - internal_agent_ids      # pattern `agent-[0-9a-f]{6,}`
  - branch_names            # pattern `refs/heads/|feature/|fix/`
  - commit_shas             # 40-char hex string pattern
  - file_paths              # paths containing /src/, /internal/, /backend/
  - infrastructure_terms    # docker, kubernetes, pod, deployment, service mesh
  - raw_log_lines           # lines starting with [DEBUG], [INFO], [WARN], [ERROR]
```

**Interactions:**
- Publish: `POST /internal/projects/<id>/items/<item_id>/publish` → BFF → validation → BRD-02
- Unpublish: `POST /internal/projects/<id>/items/<item_id>/unpublish` → audited → BRD-02
- Validation failure: `400 Bad Request` with `{ field, reason, code }`

**Edge cases:**
- Item published then internal content updated: republished version must re-validate
- Item blocked by unpublished dependency: generic "Internal dependency is blocking progress" shown (FR-03-046)
- Empty summary after trimming whitespace: treated as missing, validation fails

**Risks:**
- Validation regex must not have catastrophic backtracking (ReDoS)
- Forbidden patterns must be documented explicitly so internal users know what not to write

---

### 5. Owner Label Service — L2

**How:** BFF maintains owner label mapping cache (5-min TTL). On item render, BFF applies mapping: check override first (item metadata), then default mapping table. Unmapped roles fall back to "Product".

**Interactions:**
- BFF startup: fetch `GET /config/owner-mapping` from BRD-02; cache response
- Item render: `GET /projects/<id>/tasks/<task_id>` → BFF applies label mapping → client receives label
- Override set: internal owner sets `client_owner_label_override` on item in BRD-02

**Edge cases:**
- API failure on mapping fetch: fall back to hardcoded defaults; log warning
- Override set to empty string: treated as "no override"; revert to default mapping
- New internal role not in mapping table: falls back to "Product" with logged warning

---

### 6. Access Filtering Layer — L2

**How:** Every BFF API call enforces project access filtering. BFF extracts client principal from session. BFF query includes principal filter. BRD-02 returns only accessible projects. Unauthorized project access returns empty result (not 403, to avoid leaking project existence).

**Interactions:**
- Every API call: `GET /projects/<id>/...` → BFF checks principal has access to project_id
- SSE events: BFF filters event stream to only events where `event.project_id` is in principal's accessible set
- Search: BFF appends `principal=<id>` to search query; BRD-02 filters results

**Edge cases:**
- Client's access list changes mid-session (admin revokes project access): existing page remains readable; next navigation reflects new access; SSE events for revoked project stop
- Access list with 50 projects: BFF handles N projects without performance degradation

**Risks:**
- NFR-03-013: access filtering failure is treated as security defect and fails closed
- BFF must never accidentally cache or return data from an unauthorized project

---

## Data Model — Client Portal

### Entities

**ClientPrincipal**
- `id`: string
- `name`: string
- `accessible_project_ids`: string[]

**Project (client-visible subset)**
- `id`: string
- `name`: string
- `health`: `on_track | at_risk | blocked`
- `confidence`: `high | medium | low | null`
- `confidence_reason`: string (plain-language)
- `completion_percentage`: number (0-100, null if no active tasks)
- `completion_empty_state`: boolean
- `next_milestone`: { name, due_date, status } | null
- `pending_decision_count`: number
- `overdue_decision_count`: number
- `latest_client_safe_update`: timestamp

**ClientVisibleTask**
- `id`: string
- `title`: string
- `status`: `todo | in_progress | blocked | done`
- `client_status_label`: `To Do | In Progress | Blocked | Done`
- `owner_label`: string (from owner label service)
- `summary`: string (client-safe)
- `summary_fallback`: boolean (true if safe fallback used)
- `blocker_reason`: string | null (client-safe, generic if unpublished blocker)
- `due_date`: timestamp | null
- `updated_at`: timestamp
- `next_action`: string (exactly one per published item)
- `is_cancelled`: boolean
- `cancellation_reason`: string | null (shown only when `show_cancelled` enabled)

**ClientVisibleApprovalItem**
- `id`: string
- `title`: string
- `type`: string
- `project_id`: string
- `project_name`: string
- `created_at`: timestamp
- `is_overdue`: boolean
- `outcome_state`: `pending | approved | rejected | requested_changes | waiting_on_response`
- `actor`: { name, role }
- `next_action`: string

**ClientVisibleRisk**
- `id`: string
- `severity`: string
- `impact`: string
- `owner_label`: string
- `mitigation_summary`: string (plain-language)
- `status`: string | null
- `next_action`: string

**ClientVisibleMilestone**
- `id`: string
- `name`: string
- `status`: `upcoming | in_progress | completed | missed`
- `due_date`: timestamp | null
- `progress`: number (0-100) | null
- `health`: string
- `summary`: string
- `next_action`: string

**ClientVisibleComment**
- `id`: string
- `project_id`: string
- `related_item_id`: string
- `related_item_type`: `approval | blocker | risk | milestone`
- `author_name`: string
- `body`: string
- `created_at`: timestamp
- `updated_at`: timestamp | null
- `is_edited`: boolean
- `is_deleted`: boolean

---

## L2 Data Flow — SSE Event Processing

```
BRD-02 SSE Event
      │
      ▼
BFF SSE Subscriber
      │
      ├─ Parse event envelope: { event_type, project_id, item_id, ... }
      │
      ├─ Check: project_id ∈ principal.accessible_project_ids?
      │    └─ No → drop event; log access_denied (security defect)
      │
      ├─ Filter: item_id is client-visible (published)?
      │    └─ No → drop event
      │
      ├─ Transform: apply owner label mapping
      │    └─ strip forbidden technical content from event payload
      │
      ├─ Route: dispatch to correct client SSE channel by project_id
      │
      └─ Client SSE: EventSource receives event
                       │
                       ▼
                 SvelteKit store update
                       │
                       ▼
                 UI re-render with new state
```

---

*Progressive deepening L2 complete*
*Next: edge case discovery (Stage 05)*