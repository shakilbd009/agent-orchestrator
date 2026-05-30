# evals/integration/brd-02-platform-orchestration-project-integration.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** Integration test contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## Integration Test: Project creation and task scoping

### Setup
- Backend server running with `platform-orchestration=true`
- Feature flag registry accessible
- SQLite storage initialized

### Steps
1. Create project `proj-alpha`:
   ```
   POST /projects
   {
     "name": "Alpha Project",
     "phase": "G0",
     "actorId": "human:alice",
     "actorRole": "human"
   }
   ```
2. Create task under `proj-alpha`:
   ```
   POST /projects/proj-alpha/tasks
   {
     "title": "Task in Alpha",
     "actorId": "human:alice",
     "actorRole": "human"
   }
   ```
3. Create project `proj-beta`
4. Create task under `proj-beta`
5. Query board for `proj-alpha`
6. Query board for `proj-beta`

### Assertions
- `proj-alpha` board contains only `proj-alpha` tasks
- `proj-beta` board contains only `proj-beta` tasks
- No cross-project visibility
- Audit events for both projects are independent

---

## Integration Test: Cross-project dependency edge rejected

### Setup
- Two projects: `proj-alpha`, `proj-beta`
- Task `task-1` in `proj-alpha`
- Task `task-2` in `proj-beta`

### Steps
1. Attempt to create child relationship: `task-1` → `task-2` (cross-project)

### Assertions
- Response `409 Conflict` or `422 Unprocessable Entity`
- No dependency edge persisted
- Audit event `task.decomposition.dependency_rejected` with reason "cross-project not permitted"

---

## Integration Test: Circular dependency detection

### Setup
- Project `proj-alpha`
- Tasks `A`, `B`, `C` in `proj-alpha`
- Existing edge: `A` → `B`

### Steps
1. Attempt: `B` → `C`
2. Attempt: `C` → `A` (would create A→B→C→A cycle)

### Assertions
- Step 2: `409 Conflict`
- Cycle not created
- Audit event records cycle rejection

---

## Integration Test: Project export before archival

### Setup
- Project `proj-alpha` with multiple tasks, gates, events, handoffs

### Steps
1. Request project export:
   ```
   POST /projects/proj-alpha/export
   {
     "actorId": "human:alice",
     "actorRole": "human"
   }
   ```

### Assertions
- Response `200 OK`
- Export payload contains all project records
- Export is authorized (actor has admin role)
- Export must precede archival/deletion

---

## Integration Test: Project board read latency under 300ms

### Setup
- Project with representative load: up to 10,000 tasks (per NFR-02-006)
- Populated with tasks at various execution statuses, with dependencies and gates

### Steps
1. Measure round-trip time for `GET /projects/proj-alpha/board`

### Assertions
- Response latency < 300ms at target scale
- Response includes: tasks grouped by status, dependency relationships, gate states, stale indicators, assignments, handoff summaries

---

## Integration Test: Feature flag `platform-orchestration=false` hides capability

### Setup
- Server running with `FF_ENABLE_PLATFORM_ORCHESTRATION=false`

### Steps
1. Attempt to create project-scoped task:
   ```
   POST /projects/proj-alpha/tasks
   ```

### Assertions
- Response `404 Not Found` or capability indicator shows orchestration disabled
- No task created
- Behavior consistent with other feature-flag-disabled capabilities (per AC-02-001)

---

*End of project-scoped integration tests*