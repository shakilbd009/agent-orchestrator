# evals/e2e/brd-02-platform-orchestration-project-scoped.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** End-to-end scenario contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## E2E Scenario: Project-scoped task creation

### Given
- A project `proj-alpha` exists with `platform-orchestration=true`
- Actor `human:alice` is authenticated with role `human`

### When
Alice creates a task under `proj-alpha`:
```
POST /projects/proj-alpha/tasks
{
  "title": "Build auth service",
  "actorId": "alice",
  "actorRole": "human"
}
```

### Then
- Response `201 Created` with task object containing `projectId: proj-alpha`
- Task does NOT appear in any other project's board
- `GET /projects/proj-alpha/tasks` includes the new task
- `GET /projects/proj-beta/tasks` does NOT include the new task

---

## E2E Scenario: Cross-project dependency rejected

### Given
- Project `proj-alpha` exists
- Project `proj-beta` exists
- Task `task-1` exists in `proj-alpha`

### When
An actor attempts to create a child relationship from `task-1` to a task in `proj-beta`:
```
POST /tasks/task-1/children
{
  "childTaskId": "task-X-in-proj-beta",
  "actorId": "layer_a:bob",
  "actorRole": "layer_a"
}
```

### Then
- Response `409 Conflict` or `422 Unprocessable Entity`
- No dependency edge is created
- Audit event `task.decomposition.dependency_rejected` is appended with reason "cross-project edge not permitted"

---

## E2E Scenario: Circular dependency rejected

### Given
- Project `proj-alpha` exists
- Task `parent` exists in `proj-alpha`
- Task `child` exists in `proj-alpha` as a child of `parent`

### When
An actor attempts to make `parent` a child of `child`:
```
POST /tasks/child/children
{
  "childTaskId": "parent",
  "actorId": "layer_a:bob",
  "actorRole": "layer_a"
}
```

### Then
- Response `409 Conflict` or `422 Unprocessable Entity`
- No circular edge is created
- Audit event `task.dependency.rejected` is appended with reason "circular dependency not permitted"

---

## E2E Scenario: Feature flag disables platform-native orchestration

### Given
- `platform-orchestration=false` (default)
- Project `proj-alpha` exists

### When
A mutation request is sent:
```
POST /projects/proj-alpha/tasks
{
  "title": "Should be rejected",
  "actorId": "alice",
  "actorRole": "human"
}
```

### Then
- Response `404 Not Found` or `200 OK` with capability indicator showing orchestration disabled
- No task is created in the platform-native store
- API behavior is consistent with other feature-flag-disabled capabilities (per AC-02-001)

---

## E2E Scenario: Project board read returns scoped tasks only

### Given
- Project `proj-alpha` has 3 tasks: `task-a1`, `task-a2`, `task-a3`
- Project `proj-beta` has 2 tasks: `task-b1`, `task-b2`
- Actor `human:alice` is authenticated

### When
Alice calls:
```
GET /projects/proj-alpha/board
```

### Then
- Response contains exactly 3 tasks grouped by execution status
- None of `task-b1`, `task-b2` appear in the response
- Dependency relationships, gate states, and stale indicators are included for each task
- Response latency < 300ms (NFR-02-006)

---

## E2E Scenario: Project export before archival

### Given
- Project `proj-alpha` has tasks, gates, audit events, and handoff records
- Actor `human:alice` has authorization to archive projects

### When
Alice initiates project export:
```
POST /projects/proj-alpha/export
{
  "actorId": "alice",
  "actorRole": "human"
}
```

### Then
- Response `200 OK` with a complete export of all project records (tasks, gates, events, handoffs)
- Export is available before archival/deletion proceeds (per AC-02-029)
- Authorization is verified before export begins

---

## E2E Scenario: Project archival requires export first

### Given
- Project `proj-alpha` exists
- Actor `human:alice` attempts direct deletion without prior export

### When
```
DELETE /projects/proj-alpha
{
  "actorId": "alice",
  "actorRole": "human"
}
```

### Then
- Response `409 Conflict` indicating export must precede deletion
- Or: deletion is rejected with message requiring export confirmation
- No project data is deleted without export artifact confirmed (per FR-02-027)

---

*End of project-scoped E2E scenarios*