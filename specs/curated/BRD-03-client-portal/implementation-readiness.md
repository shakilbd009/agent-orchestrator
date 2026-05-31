# BRD-03 — Implementation Readiness

**BRD:** BRD-03-client-portal
**Stage:** implementation-readiness
**Status:** Ready for Implementation (pending eval-contracts gate)

---

## What Implementors Need to Know

### Architecture Summary

Client portal is a BFF (backend-for-frontend) pattern in front of BRD-02 backend:
- **Phase 1 prototype**: SvelteKit frontend fetches directly from BRD-02 APIs (browser → BRD-02)
- **Phase 2 production**: Go/Echo BFF sits between browser and BRD-02, enforcing access control

### Critical Interfaces

**1. Portfolio API (BFF → BRD-02)**
```
GET /portfolio?principal=<client_principal_id>
Response: {
  projects: [
    {
      id, name, health, confidence, confidence_reason,
      completion_percentage (null if no active tasks),
      next_milestone: { name, due_date, status } | null,
      pending_decision_count, overdue_decision_count,
      latest_client_safe_update
    }
  ],
  summary: {
    total_on_track, total_at_risk, total_blocked,
    total_pending_decisions, total_overdue_decisions,
    total_waiting_on_client
  }
}
```

**2. Project Detail API**
```
GET /projects/<project_id>?principal=<client_principal_id>
Response: {
  project: { id, name, health, confidence, confidence_reason, completion_percentage, ... },
  tasks: [ { id, title, status, owner_label, summary, blocker_reason, due_date, updated_at, next_action, is_cancelled, cancellation_reason } ],
  approvals: [ { id, title, type, project_id, created_at, is_overdue, outcome_state, ... } ],
  risks: [ { id, severity, impact, owner_label, mitigation_summary, status, next_action } ],
  milestones: [ { id, name, status, due_date, progress, health, summary, next_action } ]
}
```

**3. Approval Action API**
```
POST /approvals/<approval_id>/decide
Body: { outcome: "approve" | "reject" | "request_changes" | "need_more_information", comment?: string }
Response: { success, updated_state }
Note: reject, request_changes, need_more_information require comment per FR-03-022
```

**4. SSE Event Envelope (required from BRD-02)**
```
Minimum required fields:
{
  "project_id": string,
  "event_type": "task.updated" | "approval.created" | "approval.updated" | "risk.updated" | "milestone.updated" | "item.published" | "item.unpublished",
  "item_id": string,
  "timestamp": "2026-05-30T00:00:00Z",
  "payload": { ... }  // event-type-specific
}
```

### Owner Label Mapping

Default mapping served by `GET /config/owner-mapping` from BRD-02. Cached in BFF with 5-minute TTL.

| Internal Role | Client Label |
|---------------|-------------|
| engineering, backend, frontend, developer | Engineering |
| product_manager, product_owner, pm | Product |
| reviewer, qa, quality_assurance, testing | Review |
| quality, architect, design | Quality |
| client, client_stakeholder, external | Client |
| (fallback for unmapped) | Product |

Override: item metadata `client_owner_label_override` takes precedence.

### Publication Validation Schema

```yaml
required_fields:
  - business_summary      # plain-language, 10-500 chars
  - client_owner_label    # from allowed label set
  - next_action            # exactly one, max 200 chars
  - visibility_status      # must be "client_visible"

forbidden_patterns:
  - stack_trace_lines: '^\s+at\s+' or 'Traceback'
  - agent_ids: 'agent-[0-9a-f]{6,}'
  - branch_names: 'refs/heads/|feature/|fix/'
  - commit_shas: 40-char hex
  - file_paths: '/src/|/internal/|/backend/|/agent/'
  - infra_terms: 'docker|kubernetes|pod|deployment|service mesh'
  - log_lines: '^\[DEBUG\]|^\[INFO\]|^\[WARN\]|^\[ERROR\]'
```

### Forbidden Content Stripping

BFF strips the following from ALL client-facing responses:
- Stack traces (lines matching `^\s+at\s+` or containing `Traceback`)
- Internal agent IDs (pattern `agent-<hex>`)
- Branch names (patterns: `refs/heads/`, `feature/`, `fix/`)
- Commit SHAs (40-char hex strings)
- File paths containing `/src/`, `/internal/`, `/backend/`, `/agent/`
- Infrastructure terms: `docker`, `kubernetes`, `pod`, `deployment`, `service mesh`
- Raw log lines starting with `[DEBUG]`, `[INFO]`, `[WARN]`, `[ERROR]`

### Overdue Calculation

- Per approval item: `now() - created_at > 24 hours`
- `created_at` = when item became visible/actionable to client
- Overdue is a visibility signal only; no auto-reject/approve/advance

### Access Filtering

Every BFF API call enforces project-level access:
- BFF extracts client principal from session
- BFF query includes `principal` filter
- BRD-02 returns only accessible projects
- Unauthorized project access returns empty result (NOT 403 — to avoid leaking project existence)
- `client_portal_access_denied_total` incremented for security monitoring

### SSE Implementation

- Per-project SSE EventSource connections (one per visible project, up to 50)
- BFF SSE multiplexer for global inbox (server-side multiplex → clients subscribe to BFF)
- Reconnection: exponential backoff with jitter; max 5 retries per minute
- On disconnect: show "Live updates paused"; manual refresh available

---

## Data Model Summary

| Entity | Key Fields | Source |
|--------|------------|--------|
| ClientPrincipal | id, name, accessible_project_ids[] | BRD-17 / platform auth |
| ClientVisibleProject | id, name, health, confidence, completion%, next_milestone, decision_counts | BRD-02 project API |
| ClientVisibleTask | id, title, status, owner_label, summary, blocker_reason, due_date, next_action, is_cancelled | BRD-02 task API |
| ClientVisibleApprovalItem | id, title, type, project_id, created_at, is_overdue, outcome_state, actor | BRD-02 approval API |
| ClientVisibleRisk | id, severity, impact, owner_label, mitigation_summary, status, next_action | BRD-16 via BRD-02 |
| ClientVisibleMilestone | id, name, status, due_date, progress, health, summary, next_action | BRD-02 milestone API |
| ClientVisibleComment | id, project_id, related_item_id, author_name, body, created_at, is_edited, is_deleted | BRD-02 comment API |

---

## Dependencies and Blockers

| Dependency | Status | Action Required |
|------------|--------|-----------------|
| BRD-02 project/task/approval SSE APIs | **PARTIAL** | OQ-03-001 (SSE envelope schema) must be resolved before SSE eval contract authoring |
| BRD-02 approval state machine | Defined | Backend must implement state transitions per state machine in progressive-deepening-L2.md |
| BRD-02 owner mapping API | Defined | Backend must implement `GET /config/owner-mapping` |
| BRD-16 risk data API | Defined | BRD-16 must define risk API contract; BRD-03 assumes `risks` endpoint in project detail |
| BRD-17 project access filtering | Needed | Platform auth must provide `principal` context to BFF; BFF enforces filtering |
| BRD-02 comment API | Defined | Must support create/edit/delete with audit metadata per FR-03-025 to FR-03-030 |
| Feature flag `client-portal` | Not registered | Must add to `specs/feature-flags.md` when BRD-03 exits draft status |

---

## Implementation Phasing

### Phase 1 (Prototype — direct fetch, no BFF)

1. SvelteKit frontend with direct BRD-02 API calls
2. Client-side publication validation (UI guard)
3. Per-project SSE connections
4. Hardcoded owner label mapping
5. Manual publication validation testing

### Phase 2 (Production — BFF added)

1. Go/Echo BFF with centralized access filtering
2. Server-side publication validation (API gate) with shared schema
3. BFF SSE multiplexer for global inbox
4. API-provided owner label mapping with hardcoded fallback
5. Full security testing (access boundary, forbidden content stripping)

---

## Verification Checklist

Before implementation is considered complete:

- [ ] All 52 FRs have corresponding test cases
- [ ] All 15 NFRs have corresponding performance/load/accessibility tests
- [ ] 45 ACs have corresponding integration tests
- [ ] Negative access-control tests pass (cannot access unauthorized projects)
- [ ] Publication validation fails on forbidden technical content
- [ ] SSE updates reflect within 2s (NFR-03-006)
- [ ] Portfolio loads within 5s at 50-project scale (NFR-03-004)
- [ ] Project detail loads within 2s at 10,000-task scale (NFR-03-005)
- [ ] Overdue calculation correct (per item, 24h threshold)
- [ ] Owner label mapping correctly applies override precedence
- [ ] Comment audit metadata excludes deleted/edited body text
- [ ] `client-portal` flag registered in feature-flags.md

---

*Implementation readiness assessed — ready for eval-contracts gate*
*Blockers: OQ-03-001 (SSE envelope schema), OQ-03-002 (forbidden fields list review)*