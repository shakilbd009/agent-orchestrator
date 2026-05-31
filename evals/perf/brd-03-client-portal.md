# evals/perf/brd-03-client-portal.md

**Project:** agent-orchestrator
**BRD:** BRD-03 — Client Portal and Business Project Board
**Type:** Performance test contracts
**Owner:** qa
**Status:** 🔴 **Failing** (before implementation)

---

## Performance Contract: Portfolio Landing Latency (< 5s)

### Target: NFR-03-004
**Requirement:** Initial portfolio summary visible within 5 seconds at target scale (50 accessible projects, 10,000 tasks per project).

### Test scenario: Portfolio load at max scale

**Given**
- Client principal has access to exactly 50 projects (NFR-03-002 max)
- Each project has up to 10,000 tasks
- All project data populated and accessible

**When**
Client opens portfolio landing page (cold load, no cache)

**Then**
- High-level summary (health counts, pending/overdue decision counts, confidence summary) rendered within 5 seconds
- `client_portal_portfolio_load_duration_ms` histogram recorded
- Target: p95 < 5000ms
- Progressive loading: high-level summary renders before lower-priority detail data

**Measurement:**
- Start: navigation/URL request initiated
- End: first meaningful paint of portfolio summary (health counts visible)
- Metrics: `client_portal_portfolio_load_duration_ms` histogram

---

## Performance Contract: Project Detail Latency (< 2s)

### Target: NFR-03-005
**Requirement:** Selected project detail summary and board visible within 2 seconds at target project scale.

### Test scenario: Project detail load at target scale

**Given**
- Client principal has access to `proj-alpha`
- `proj-alpha` has 500 tasks (below max scale but representative)
- Client navigates from portfolio to project detail

**When**
Client clicks on `proj-alpha` in project list

**Then**
- Project detail summary (health, completion %, health explanation) visible within 2 seconds
- Task board renders with correct status groupings within 2 seconds
- `client_portal_project_load_duration_ms` histogram recorded
- Target: p95 < 2000ms

**Measurement:**
- Start: project detail navigation initiated
- End: project health + board rendered and interactive
- Metrics: `client_portal_project_load_duration_ms` histogram

---

## Performance Contract: SSE Delivery Latency (< 2s)

### Target: NFR-03-006
**Requirement:** Relevant committed updates visible within 2 seconds when SSE is connected.

### Test scenario: SSE event end-to-end delivery

**Given**
- Client is SSE-connected to `proj-alpha`
- Client is viewing project detail board for `proj-alpha`
- Latency target: 2 seconds

**When**
Internal actor commits a task status change (e.g., `in_progress` → `done`)

**Then**
- SSE event emitted by BRD-02
- Event received by BFF, filtered, envelope fields stripped
- Event delivered to client browser via EventSource
- UI updated to reflect new status
- Total latency < 2000ms (p95)

**Measurement:**
- Start: internal task status update committed to BRD-02
- End: UI reflects new status (DOM updated)
- Metrics: latency histogram, `client_portal_project_load_duration_ms` for SSE-triggered updates

---

## Performance Contract: Portfolio API Latency

### Target: NFR-03-004
**Requirement:** Portfolio API response time sufficient to render summary within 5 seconds.

### Test scenario: BFF portfolio API response time

**Given**
- 50 accessible projects with full data
- BFF → BRD-02 API latency ~100ms per project (simulated)

**When**
BFF receives `GET /portal/portfolio` for client principal

**Then**
- BFF parallelizes project data fetches (up to concurrency limit)
- Response aggregate built from accessible project data
- Response returned to client within 4 seconds (leaving 1s for client rendering)

**Measurement:**
- `GET /portal/portfolio` response time p95 < 4000ms
- BFF internal latency < 3500ms (excluding network)

---

## Performance Contract: Search Latency

### Target: Implicit from NFR-03-004 and FR-03-038
**Requirement:** Text search across client-visible content completes within reasonable time.

### Test scenario: Search completes within 3 seconds

**Given**
- Client principal has access to 50 projects
- Search index populated across all accessible projects

**When**
Client searches for term that exists in multiple projects (e.g., "deployment" — 20 matches across 5 projects)

**Then**
- Search results returned within 3 seconds
- Results include `project_id`, `project_name` for context
- Results filtered to client-visible items and accessible projects only

**Measurement:**
- Search API response time p95 < 3000ms
- Client-side render time for results list included

---

## Performance Contract: SSE Connection Management

### Target: NFR-03-002
**Requirement:** System supports up to 50 accessible projects per client principal; SSE connections managed per-project.

### Test scenario: Multiple SSE connections at max portfolio scale

**Given**
- Client principal has access to 50 projects
- Client opens project detail for 3 projects sequentially (not all simultaneously)

**When**
Client views project A → close → view project B → close → view project C

**Then**
- At most 3 concurrent SSE EventSource connections at any time
- Each EventSource closed on component unmount
- No SSE connection to projects not currently being viewed
- BFF maintains one SSE connection to BRD-02 per project being viewed

**Measurement:**
- Max concurrent EventSource connections = min(visible_projects, 50)
- SSE connection memory footprint tracked
- BFF SSE multiplexer: 1 connection to BRD-02 per project, not per client

---

## Performance Contract: SSE Reconnection Backoff

### Target: ADR-03-002 / FR-03-041
**Requirement:** SSE reconnection uses exponential backoff with jitter; max 5 retries per minute.

### Test scenario: SSE reconnection behavior

**Given**
- SSE connection to `proj-alpha` established and functioning

**When**
Network failure causes SSE disconnect

**Then**
- Client shows "Live updates paused" within 5 seconds
- Reconnection attempted with exponential backoff + jitter
- Backoff sequence: ~1s, ~2s, ~4s, ~8s, ~16s (jitter added)
- After 5 failures in one minute, backoff continues at 1 per minute
- Manual refresh available at any time

**Measurement:**
- Time to "Live updates paused" indicator < 5 seconds
- Reconnection attempts tracked in `client_portal_sse_disconnect_total`

---

## Performance Contract: BFF Cache Performance

### Target: ADR-03-005
**Requirement:** BFF caches owner label mapping with 5-minute TTL; cache reduces API load.

### Test scenario: Owner label mapping served from cache

**Given**
- BFF has owner label mapping cached (5min TTL)
- Cache is warm

**When**
Client views 20 tasks in project detail, each requiring owner label resolution

**Then**
- All 20 owner labels resolved from cache (no BRD-02 API call for mapping)
- API response time for project detail not impacted by owner label resolution
- Cache hit rate tracked

**Measurement:**
- Owner label API call: 0 (cache hit)
- Cache miss: API call made, cache populated
- TTL expiry: next request triggers fresh fetch

---

## Performance Contract: Concurrent Client Load

### Target: NFR-03-001
**Requirement:** Support up to 10 concurrent client users in the initial target environment.

### Test scenario: 10 concurrent client principals

**Given**
- 10 client principals, each with access to 20 projects
- Each principal viewing portfolio landing simultaneously

**When**
All 10 clients load portfolio landing at the same time

**Then**
- Each portfolio page loads within 5 seconds (NFR-03-004)
- BFF handles 10 concurrent requests without errors
- SSE subscriptions: up to 10 principals × N projects = managed connections
- No request queuing or degradation observed

**Measurement:**
- All 10 clients complete portfolio load within target time
- BFF CPU/memory under acceptable load
- Response time distribution across concurrent requests

---

## Performance Contract: Large Task Count Rendering

### Target: NFR-03-003
**Requirement:** Project detail view renders for projects with up to 10,000 tasks, aligned with BRD-02.

### Test scenario: Project board renders large task list

**Given**
- Project `proj-alpha` has 10,000 tasks
- Client opens project detail

**When**
Project detail page loads with task board

**Then**
- Board renders within 2 seconds (NFR-03-005)
- Virtual scrolling or pagination used to avoid rendering all 10,000 DOM nodes simultaneously
- Task cards show required fields: title, status, owner label, summary, next action
- UI remains responsive during scroll

**Measurement:**
- Project detail first meaningful paint < 2000ms
- Scroll performance: 60fps during board scroll
- DOM node count: bounded by virtual scrolling

---

## Performance Contract: Manual Refresh Fallback

### Target: NFR-03-007
**Requirement:** Manual refresh returns current-state data or clear failure message without requiring SSE.

### Test scenario: Manual refresh succeeds when SSE is disconnected

**Given**
- SSE connection to `proj-alpha` is disconnected
- "Live updates paused" indicator shown
- Client clicks manual refresh button

**When**
Manual refresh request sent to current-state API

**Then**
- Current-state data returned within 3 seconds
- UI updated with latest data
- SSE connection remains disconnected (no automatic reconnection triggered by refresh)
- `client_portal_manual_refresh_total` counter incremented with label "fallback"

**Measurement:**
- Manual refresh response time p95 < 3000ms
- Data freshness: current-state data is authoritative

---

## Performance Contract: Read Failure Behavior

### Target: NFR-03-009
**Requirement:** If read APIs are unavailable, portal shows unavailable state and does not present stale data as current.

### Test scenario: Read API unavailable shows unavailable state

**Given**
- Client is viewing project detail for `proj-alpha`

**When**
BRD-02 read API becomes unavailable

**Then**
- Portal shows "Project data temporarily unavailable" state
- No stale data shown as current
- Manual refresh button retries the request
- `client_portal.reads.unavailable` event logged

**Measurement:**
- Time to unavailable state: < 5 seconds from API failure detection
- No stale data rendered after failure detection

---

## Performance Contract: Progressive Loading

### Target: FR-03-054
**Requirement:** Portfolio page SHOULD render high-level summary before lower-priority details when aggregation is slow.

### Test scenario: Progressive render of portfolio summary

**Given**
- Client principal has access to 50 projects
- Network latency to BRD-02 is high (simulated ~200ms per project)

**When**
Client loads portfolio landing

**Then**
- Health counts and decision summary rendered first (within 2s)
- Project list populates progressively as data arrives
- No blocking on full data before showing summary
- Progressive loading does not exceed 5s total for summary visibility

**Measurement:**
- Time to first meaningful paint (health counts): p95 < 2000ms
- Time to full portfolio render: p95 < 5000ms

---

*End of BRD-03 performance contracts — 13 performance contracts covering all NFR latency and scale targets*

**NFR Coverage:**
- NFR-03-001: 10 concurrent clients → concurrent client load test
- NFR-03-002: 50 accessible projects → SSE connection management test
- NFR-03-003: 10,000 tasks per project → large task count rendering test
- NFR-03-004: 5s portfolio landing → portfolio landing latency test
- NFR-03-005: 2s project detail → project detail latency test
- NFR-03-006: 2s SSE delivery → SSE delivery latency test
- NFR-03-007: manual refresh fallback → manual refresh fallback test
- NFR-03-009: read failure behavior → read failure behavior test
- FR-03-038 search → search latency test
- FR-03-054 progressive loading → progressive loading test