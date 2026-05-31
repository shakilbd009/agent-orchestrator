# BRD-03 — Progressive Deepening L1

**BRD:** BRD-03-client-portal
**Stage:** 04-design-l1
**Status:** Analyzed

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Client Browser                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │  Portfolio   │  │   Project   │  │   Global Approval   │ │
│  │   Landing   │  │   Detail    │  │       Inbox          │ │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘ │
│         │                │                    │              │
│         │ SSE             │ SSE                │ SSE          │
│         │ (per-project)   │ (per-project)     │ (multi-     │
│         │                 │                    │  project)   │
└─────────┼────────────────┼────────────────────┼─────────────┘
          │                │                    │
          │ HTTP/REST      │ HTTP/REST          │ HTTP/REST
          │ (initial load, │ (initial load,     │ (initial load,
          │  manual refresh │  manual refresh   │  manual refresh
          ▼                ▼                    ▼
┌─────────────────────────────────────────────────────────────┐
│              Client Portal BFF (Go/Echo)                    │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Access Filtering Layer  ───  Project access enforced │  │
│  │  Publication Validation  ───  Business-language check │  │
│  │  SSE Event Multiplexer   ───  Per-project SSE routing  │  │
│  │  Owner Label Service     ───  Default mapping + override│  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────┬───────────────────────────────┘
                              │ HTTP/REST + SSE
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              BRD-02 Platform Backend                       │
│  ┌────────────┐  ┌────────────┐  ┌────────────────────┐  │
│  │  Project    │  │   Task      │  │   Approval         │  │
│  │  API        │  │   API       │  │   API              │  │
│  └────────────┘  └────────────┘  └────────────────────┘  │
│  ┌────────────┐  ┌────────────┐  ┌────────────────────┐  │
│  │  Risk      │  │ Milestone  │  │   SSE              │  │
│  │  API       │  │   API      │  │   Event Stream     │  │
│  └────────────┘  └────────────┘  └────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

---

## Component Inventory

### 1. Portfolio Landing Page

**What:** Entry point for client stakeholders. Shows all accessible projects in aggregate.

**Why:** US-03-001 requires portfolio-level summary before drilling into individual projects.

**Core responsibilities:**
- Fetch accessible project list from BRD-02 via BFF
- Aggregate health counts, confidence, decision counts
- Render project list with health indicators
- Maintain SSE subscription for real-time count updates
- Provide link to global approval inbox

**Initial questions raised:**
- How is the initial project list sorted? (alphabetical? health priority? most recently updated?)
- Does the landing page paginate if 50 projects exceed viewport?

---

### 2. Project Detail View

**What:** Per-project view showing board, approvals, risks, milestones, comments.

**Why:** US-03-002 requires business-readable project board without technical detail.

**Core responsibilities:**
- Fetch project state from BRD-02 via BFF
- Render task board with BRD-02 canonical statuses → client-readable labels
- Display completion percentage (done / active tasks)
- Show pending approvals with action controls
- Show risks and milestones
- Maintain per-project SSE subscription for live updates

**Initial questions raised:**
- Does the board show all tasks or paginate/filter?
- What is the cancellation reason field format?

---

### 3. Global Approval Inbox

**What:** Unified view of all pending client decisions across accessible projects.

**Why:** US-03-003 requires decision-making without hunting through each project.

**Core responsibilities:**
- Aggregate pending approval items across all accessible projects
- Group by project for context
- Provide approve/reject/request_changes/need_more_information actions
- Show overdue status for items pending >24h
- Provide navigation to project context for each item

**Initial questions raised:**
- How does SSE cover all accessible projects simultaneously? (50 connections? multiplexing?)
- How is overdue calculated when items arrive at different times?

---

### 4. Publication Validation Service

**What:** Backend service enforcing business-language rules before items become client-visible.

**Why:** FR-03-043/045/048 require explicit publish/review step and validation.

**Core responsibilities:**
- Accept publication request with item metadata
- Validate required fields: business-language summary, owner label, next action, visibility status
- Check for forbidden technical fields (stack traces, agent IDs, commit SHAs, branch names, file paths)
- Return validation result (success / failure with reason)
- Audit log publication/unpublication actions

**Initial questions raised:**
- What is the exact list of forbidden technical fields?
- Is publication a property of the item or a separate audit event?

---

### 5. Owner Label Service

**What:** Service mapping internal role/task metadata to client-facing labels.

**Why:** FR-03-016/017 require hybrid model with default auto-mapping and override capability.

**Core responsibilities:**
- Maintain default mapping table (Product, Engineering, Review, Quality, Client)
- Provide override mechanism for internal project owners
- Serve label lookups at render time

**Initial questions raised:**
- Where is the override stored? (project config? item metadata?)
- How does the BFF know which label to use for a given item?

---

### 6. Access Filtering Layer

**What:** BFF layer ensuring client can only see projects they are authorized to access.

**Why:** FR-03-006/007 and NFR-03-013 require cross-project isolation with security-defect-level failure.

**Core responsibilities:**
- Enforce project-level access on every API call
- Filter SSE events to only accessible projects
- Return empty/no-data for unauthorized projects (not 403 to avoid information leakage)
- Emit access_denied metrics/logs for security monitoring

**Initial questions raised:**
- Does the BFF cache access control decisions? (performance vs. consistency tradeoff)
- How does the BFF handle a client principal that has access to 50 projects (NFR-03-002)?

---

## Data Flow Diagrams

### Portfolio Load Flow

```
Client → BFF → BRD-02 Project API
                │
                ├─ GET /projects?principal=<id>
                │   Returns: [ {project_id, name, health, confidence, completion%, ...} ]
                │
Client ← BFF ←─┘
```

### Project Detail Load Flow

```
Client → BFF → BRD-02 Task API
                │
                ├─ GET /projects/<id>/tasks (filtered to client-visible)
                │   Returns: [ {id, title, status, owner_label, summary, next_action, ...} ]
                │
                ├─ GET /projects/<id>/approvals?state=pending
                │   Returns: [ {id, title, type, created_at, ...} ]
                │
                ├─ GET /projects/<id>/risks
                │   Returns: [ {id, severity, impact, mitigation_summary, ...} ]
                │
                ├─ GET /projects/<id>/milestones
                │   Returns: [ {id, name, status, due_date, progress, ...} ]
                │
Client ← BFF ←─┘
```

### Approval Action Flow

```
Client → BFF → BRD-02 Approval API
                │
                ├─ POST /approvals/<id>/decide
                │   Body: { outcome, comment? }
                │   Returns: { success, updated_state }
                │
                └─ Log: client_portal.approval.submitted
                        { project_id, item_id, outcome, actor, timestamp }
                        │
Client ← BFF ←─┘
```

### SSE Subscription Flow

```
Client → BFF → BRD-02 SSE /projects/<id>/events
                │
                ├─ Event: task.updated { task_id, new_status, ... }
                ├─ Event: approval.created { approval_id, ... }
                ├─ Event: risk.updated { risk_id, ... }
                └─ Event: milestone.updated { milestone_id, ... }
                        │
Client ← BFF ←─┘
                        │
                (BFF filters events to client-visible items only)
                (BFF maps internal owner labels → client labels)
                (BFF strips any forbidden technical content from event payload)
```

---

## Error Scenarios (Basic)

| Scenario | How it manifests | Expected behavior |
|----------|-----------------|-------------------|
| BRD-02 API unavailable | Portfolio/project pages return error | Show unavailable state; do not show stale data as current (NFR-03-009) |
| SSE disconnects | "Live updates paused" indicator shown | Portal remains usable; manual refresh via current-state APIs (FR-03-041) |
| Publication validation fails | Item stays hidden; API returns 400 with reason | User sees error with actionable guidance |
| Client principal has no accessible projects | Empty state on portfolio landing | "No projects available" with suggested next action |
| Approval submission fails (reads work) | Write controls disabled; explanation shown | Read-only degraded mode (FR-03-053) |
| Access boundary violation attempt | Client tries to access unauthorized project | Access denied logged; empty result returned (not 403) |

---

*Progressive deepening L1 complete*
*Next: Stage 05 design L2 + edge case discovery*