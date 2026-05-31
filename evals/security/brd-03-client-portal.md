# evals/security/brd-03-client-portal.md

**Project:** agent-orchestrator
**BRD:** BRD-03 — Client Portal and Business Project Board
**Type:** Security test contracts
**Owner:** qa
**Status:** 🔴 **Failing** (before implementation)

---

## Security Contract: XSS Prevention in Business-Language Rendering

### Vulnerability: Script injection via project name, task title, or comment body

**Attack vector:**
- Attacker creates project with name `<script>alert('xss')</script>`
- Attacker creates task title containing `<img src=x onerror=alert(1)>`
- Attacker posts comment with `javascript:alert(1)` in body

**Attack surface:**
- All user-authored text fields rendered in client-facing views
- Project name, task title, owner label, risk description, milestone summary, comment body

**Required defenses:**
1. All text rendered via SvelteKit's automatic HTML escaping (default)
2. No `v-html` or equivalent unsanitized rendering of user content
3. BFF strips any forbidden content before storing/forwarding (publication validation)
4. XSS test payload in any field → 400 on submission, item stays hidden

**Test cases:**
| Input | Expected behavior |
|-------|------------------|
| `<script>alert('xss')</script>` in project name | Rendered as `&lt;script&gt;...` — not executed |
| `<img src=x onerror=alert(1)>` in task title | Rendered safely or stripped |
| `javascript:` URL in comment body href | Neutralized |
| `<svg onload=alert(1)>` in risk description | Rendered safely |
| Unicode emoji in all fields | Preserved, no XSS |

---

## Security Contract: Project Access Filtering Enforcement

### Vulnerability: Cross-project data leak via API or SSE

**Attack vector:**
- Client principal `carol` has access to `proj-a` and `proj-b`
- Attacker (or misconfiguration) attempts to access `proj-c` (no access)
- Or: SSE event arrives for `proj-c` and is inadvertently displayed

**Required defenses:**
1. BFF enforces access filtering on ALL endpoints — every read, search, count, SSE event
2. BFF returns empty result (not 403/404) — avoids existence signal
3. SSE event filter drops events for inaccessible projects
4. `client_portal_access_denied_total` counter incremented on each attempted access
5. `client_portal.access.denied` audit event logged

**Test cases:**
| Scenario | Expected result |
|----------|----------------|
| GET `/projects/proj-c/tasks` for unauthorized project | Empty result, no 403/404 |
| GET `/projects/proj-c/search?q=test` for unauthorized project | Empty result |
| SSE event for `proj-c` arrives for client without access | Event dropped, no UI update |
| Attempt to navigate directly to `/projects/proj-c` | Empty result or redirect, not 403 |
| Count endpoint for unauthorized project | Returns 0, not error |

**Negative test — authorization:**
- Verify BFF rejects requests without valid session principal
- Verify SSE subscription rejected for unauthorized project scope

---

## Security Contract: CORS for SSE Endpoint

### Vulnerability: Unauthorized SSE subscription from cross-origin browser

**Attack vector:**
- Malicious page on `attacker.com` attempts SSE connection to client portal BFF for victim's projects
- Browser automatically sends cookies on cross-origin SSE request

**Required defenses:**
1. SSE endpoint validates origin header against allowed origins
2. SSE endpoint validates that requesting principal has access to requested project scope
3. No credentials or tokens in SSE URL query params (use headers or cookies)
4. CORS preflight rejected for SSE endpoint
5. Allowed origins explicitly configured, not wildcard in production

**Test cases:**
| Scenario | Expected result |
|----------|----------------|
| SSE request from `https://app.agent-orchestrator.com` (allowed origin) | Connection accepted |
| SSE request from `https://attacker.com` (not allowed) | Connection rejected with CORS error |
| SSE request with no `Origin` header | Connection rejected |
| SSE request with forged `Origin` | Connection rejected |

---

## Security Contract: Comment Audit Privacy

### Vulnerability: Comment body retained in audit after delete, or previous version after edit

**Attack vector:**
- Client posts comment with sensitive information
- Client edits comment — old version retained in audit
- Client deletes comment — body retained in audit log

**Required defenses:**
1. Audit records: create event with comment_id, actor, timestamp, project, item, action="created" — NO body
2. Audit records: edit event with comment_id, actor, timestamp, action="edited" — NO previous body
3. Audit records: delete event with comment_id, actor, timestamp, action="deleted" — NO deleted body
4. Comment body not retained in any system component after delete
5. Audit log is append-only — no edits to historical audit records

**Test cases:**
| Scenario | Expected result |
|----------|----------------|
| Create comment → check audit record | No body text in audit |
| Edit comment → check audit record | No previous body text |
| Delete comment → check audit record | No deleted body text |
| Query audit for deleted comment by ID | Record exists (action=deleted) but body field absent |

---

## Security Contract: Publication Validation — Forbidden Technical Content

### Vulnerability: Internal technical content leaks to client via publication

**Attack vector:**
- Internal user publishes item with stack trace in summary
- Internal user publishes item with agent ID in owner field
- Internal user publishes item with branch name/commit SHA in description
- SSE event payload contains envelope metadata fields (actorId, actorRole, etc.)

**Required defenses:**
1. UI guard + API gate with shared validation schema (ADR-03-003)
2. API gate rejects items with forbidden content patterns
3. BFF strips envelope metadata fields (actorId, actorRole, eventId, schemaVersion, parentTaskId, gateId, layer) before SSE fanout
4. BFF scans metadata object for forbidden field names and patterns
5. Forbidden content never reaches client, never logged with raw content

**Forbidden patterns (ADR-03-003):**
| Pattern | Examples |
|---------|----------|
| Stack traces | `at `, `Traceback`, `Exception`, `Error:`, `panic:` |
| Agent IDs | `agent-<hex>` pattern |
| Branch names | `refs/heads/*`, `feature/*`, `fix/*` |
| Commit SHAs | 40-char hex `[a-f0-9]{40}` |
| File paths | `/src/`, `/internal/`, `/backend/`, `/agent/`, `/pkg/`, `/cmd/` |
| Infrastructure | `docker`, `kubernetes`, `pod`, `deployment`, `service mesh` |
| Log output | `[DEBUG]`, `[INFO]`, `[WARN]`, `[ERROR]`, `[TRACE]` |
| Go runtime | `panic:`, `goroutine`, `runtime.gopanic` |
| Internal roles | `layer_a`, `layer_b` as string values |

**Test cases:**
| Scenario | Expected result |
|----------|----------------|
| Submit item with stack trace in description | 400 Bad Request, item stays hidden |
| Submit item with `agent-a1b2c3` in owner field | 400 Bad Request |
| Submit item with `feature/auth-fix` in field | 400 Bad Request |
| Submit item with commit SHA `abc123def...` (40 hex) | 400 Bad Request |
| SSE event arrives with `actorId: "agent-x"` field | BFF strips before fanout to client |
| Submit item with `blockedReason` containing agent ID | 400 Bad Request |

---

## Security Contract: BFF Access Control Boundary

### Vulnerability: BFF fails to filter responses, exposing unauthorized project data

**Attack vector:**
- Client requests project list or item data
- BFF query to BRD-02 returns data for unauthorized projects
- BFF forwards full response to client without filtering

**Required defenses:**
1. BFF is the access control boundary — all client requests go through BFF
2. BFF maintains principal's accessible project set from session
3. BFF filters every response to accessible project set
4. BFF never returns raw 403/404 for unauthorized access (returns empty result)
5. `client_portal_access_denied_total` incremented on filter events

**Test cases:**
| Scenario | Expected result |
|----------|----------------|
| BFF receives response with `proj-c` tasks (no access) | BFF filters out `proj-c`, returns only accessible |
| BFF receives response with mixed accessible/inaccessible projects | Only accessible projects in response |
| BFF receives SSE event for inaccessible project | Event dropped, not forwarded |

---

## Security Contract: Client Portal Feature Flag Isolation

### Vulnerability: Feature flag bypass reveals client portal to unauthorized users

**Attack vector:**
- `client-portal` flag set to `false` but routes still accessible
- Or: flag checked only on frontend, backend allows all requests

**Required defenses:**
1. Feature flag enforced at both frontend and backend
2. Flag checked before rendering client portal routes
3. API endpoints return disabled response when flag = false
4. No client portal data exposed when flag = false

**Test cases:**
| Scenario | Expected result |
|----------|----------------|
| `client-portal=false` → access `/portal/portfolio` | 404 or disabled-state response |
| `client-portal=false` → API request for client portal data | Empty response or 404 |
| `client-portal=false` → SSE subscription attempt | Connection rejected |

---

## Security Contract: Session/Auth Isolation for Client Principal

### Vulnerability: Client principal session leaks to other clients

**Attack vector:**
- Client A's session token used by Client B
- Session token exposed in URL or logs
- SSE connection hijacked by another principal

**Required defenses:**
1. Session token validated on every BFF request
2. SSE connections authenticated per-project
3. No sensitive data in URL query params (use headers/cookies)
4. SSE connection tied to authenticated principal's accessible project set

---

## Security Contract: Read-Only Degraded Mode Security

### Vulnerability: Read-only mode bypass exposes write operations

**Attack vector:**
- Portal enters read-only mode (submission unavailable)
- Attacker tries to force write operations through

**Required defenses:**
1. Write controls disabled in read-only mode
2. API write endpoints return error when in read-only mode
3. No write state changes possible during read-only degraded mode
4. `client_portal_read_only_mode.entered` event logged

---

## Security Contract: Empty Result vs. Not Found vs. Forbidden

### Vulnerability: Distinguishing between "doesn't exist" and "not authorized" leaks project existence

**Attack vector:**
- Attacker probes for projects by name or ID
- Different error responses reveal whether project exists

**Required defenses:**
1. For unauthorized access: return same empty result as non-existent project
2. No 403 or 404 for project-level access failures (per ADR-03-001)
3. Count endpoints return 0 for inaccessible projects, not error
4. SSE subscriptions return no event for inaccessible project, not error

---

## Security Contract: Observability Does Not Leak Sensitive Data

### Vulnerability: Metrics or log events expose internal details, project names to unauthorized parties

**Attack vector:**
- Metrics include project IDs visible to all client portal users
- Logs include internal agent IDs, commit SHAs, technical details

**Required defenses:**
1. Metrics labeled by client principal type where safe (not by internal actor)
2. Project IDs in metrics only for internal-facing metrics (not client-visible)
3. Log events for client actions include project_id but not internal technical content
4. Access denied events do not include raw request details that could reveal system state

---

*End of BRD-03 security contracts — 11 security contracts covering all security-related FRs and NFRs (XSS prevention, access filtering, CORS, audit privacy, publication validation, BFF boundary, feature flag isolation, session isolation, read-only mode, empty-result semantics, observability safety)*

**Note:** OQ-03-002 (forbidden fields list) was resolved in the parent task t_6a231486 (ADR-03-003 Accepted). Security evals for publication validation are finalized and can proceed. Blocking OQ cleared.*