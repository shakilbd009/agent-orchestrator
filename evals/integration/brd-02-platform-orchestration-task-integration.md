# evals/integration/brd-02-platform-orchestration-task-integration.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** Integration test contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## Integration Test: Required child blocks parent completion

### Setup
- Project `proj-alpha`
- Parent task `parent-1`
- Required child `child-1` with status `in_progress`

### Steps
1. Attempt parent completion:
   ```
   POST /tasks/parent-1/done
   {
     "actorId": "layer_a:bob",
     "actorRole": "layer_a"
   }
   ```

### Assertions
- Response `409 Conflict`
- Parent remains in current status
- Error body identifies blocking child task

---

## Integration Test: Required cancelled child blocks parent

### Setup
- Project `proj-alpha`
- Parent `parent-1` with required child `child-1`
- `child-1` is `cancelled` without approved scope reduction

### Steps
1. Attempt parent completion

### Assertions
- Response `409 Conflict`
- Parent blocked until: child marked non-required, or replaced with approved scope adjustment (per AC-02-012)

---

## Integration Test: Blocking task-level gate prevents parent completion

### Setup
- Project `proj-alpha`
- Task `task-x` with blocking `code_review` gate in state `open`
- All required children are `done`

### Steps
1. Attempt:
   ```
   POST /tasks/task-x/done
   ```

### Assertions
- Response `409 Conflict`
- Error identifies blocking gate

---

## Integration Test: Stale detection does not auto-block

### Setup
- Project with stale threshold = 24 hours
- Task `task-x` has been `in_progress` with no mutations for 25 hours

### Steps
1. Stale detection runs
2. Query board API for `task-x`

### Assertions
- Task status remains `in_progress` (NOT auto-blocked)
- Stale event `task.stale.detected` emitted
- Board API shows stale indicator on task

---

## Integration Test: Manual blocked state requires reason

### Setup
- Task `task-x` in `in_progress`
- Actor `human:alice`

### Steps
1. Block task:
   ```
   POST /tasks/task-x/block
   {
     "executionStatus": "blocked",
     "reason": "Waiting on external dependency",
     "actorId": "human:alice",
     "actorRole": "human"
   }
   ```

### Assertions
- Response `200 OK`
- Status becomes `blocked`
- Reason stored in audit event `task.blocked`

---

## Integration Test: Layer B can only update assigned task

### Setup
- Task `task-x` assigned to `layer_b:agent1`
- Task `task-y` assigned to `layer_b:agent2`

### Steps
1. `agent1` attempts to update `task-y`:
   ```
   PATCH /tasks/task-y
   ```

### Assertions
- Response `403 Forbidden`
- `auth.mutation.denied` audit event appended

---

## Integration Test: Structured handoff with all required fields

### Setup
- Task `task-x` assigned to `layer_b:agent1`
- All required children `done`
- All blocking gates `approved`

### Steps
1. Submit handoff:
   ```
   POST /tasks/task-x/complete
   {
     "summary": "Auth service implemented and tested",
     "validationPerformed": "Unit 98%, integration all pass",
     "risks": "Production monitoring recommended",
     "residuals": "Performance baseline to be collected",
     "actorId": "layer_b:agent1",
     "actorRole": "layer_b",
     "timestamp": "2026-05-28T12:00:00Z"
   }
   ```

### Assertions
- Response `200 OK`
- Task status → `done`
- Audit event `handoff.submitted` appended

---

## Integration Test: Structured handoff rejected without required fields

### Setup
- Task `task-x` with complete handoff fields missing (e.g., no `validationPerformed`)

### Steps
1. Submit incomplete handoff

### Assertions
- Response `422 Unprocessable Entity`
- Task remains in current status
- Error body lists missing required fields (per AC-02-016)

---

*End of task lifecycle integration tests*