# evals/e2e/brd-02-platform-orchestration-task-lifecycle.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** End-to-end scenario contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## E2E Scenario: Layer B can update only assigned task

### Given
- Project `proj-alpha` with `platform-orchestration=true`
- Task `task-x` is assigned to `layer_b:agent1`
- Task `task-y` is assigned to `layer_b:agent2`
- Actor `layer_b:agent1` is authenticated

### When
Agent1 attempts to update status of `task-y` (unassigned):
```
PATCH /tasks/task-y
{
  "executionStatus": "in_progress",
  "actorId": "layer_b:agent1",
  "actorRole": "layer_b"
}
```

### Then
- Response `403 Forbidden`
- No state change occurs
- Audit event `auth.mutation.denied` is appended

---

## E2E Scenario: Layer B cannot approve own gate

### Given
- Project `proj-alpha` has a task `task-x` with an open task-level gate
- Task `task-x` is assigned to `layer_b:agent1`
- Actor `layer_b:agent1` is authenticated

### When
Agent1 attempts to approve the gate on their own task:
```
POST /gates/{gateId}/approve
{
  "actorId": "layer_b:agent1",
  "actorRole": "layer_b"
}
```

### Then
- Response `403 Forbidden`
- Gate state remains open
- Audit event `gate.approval.denied` with reason "layer_b cannot self-approve"

---

## E2E Scenario: Layer B cannot approve project phase gate

### Given
- Project `proj-alpha` has a project-level phase gate `G1 App Shell`
- Actor `layer_b:agent1` is authenticated

### When
Agent1 attempts to approve the project-level phase gate:
```
POST /projects/proj-alpha/gates/{gateId}/approve
{
  "actorId": "layer_b:agent1",
  "actorRole": "layer_b"
}
```

### Then
- Response `403 Forbidden`
- Gate state remains open
- Only `human` or authorized `layer_a` may approve project-level phase gates (per FR-02-013)

---

## E2E Scenario: Layer B cannot override decomposition limits

### Given
- Project `proj-alpha` has decomposition limits: depth=3, fan-out=20
- Actor `layer_b:agent1` attempts an override

### When
```
PATCH /projects/proj-alpha/decomposition-limits
{
  "maxDepth": 6,
  "actorId": "layer_b:agent1",
  "actorRole": "layer_b"
}
```

### Then
- Response `403 Forbidden`
- Decomposition limits are unchanged
- Override requires `human` or `layer_a` role (per FR-02-011, AC-02-009)

---

## E2E Scenario: Structured handoff required for Layer B task completion

### Given
- Task `task-x` is assigned to `layer_b:agent1`
- All required children are `done`
- All blocking task-level gates are approved

### When
Agent1 attempts to complete without structured handoff evidence:
```
POST /tasks/task-x/complete
{
  "actorId": "layer_b:agent1",
  "actorRole": "layer_b"
  // missing required fields
}
```

### Then
- Response `422 Unprocessable Entity`
- Task status remains `in_progress`
- Error body indicates missing required handoff fields

---

## E2E Scenario: Structured handoff accepted with all required fields

### Given
- Task `task-x` is assigned to `layer_b:agent1`
- All required children are `done`
- All blocking task-level gates are approved
- Actor provides complete handoff evidence

### When
```
POST /tasks/task-x/complete
{
  "summary": "Completed auth service implementation",
  "validationPerformed": "Unit tests pass, integration tests pass",
  "risks": "None identified",
  "residuals": "Production monitoring recommended for first 24h",
  "actorId": "layer_b:agent1",
  "actorRole": "layer_b",
  "timestamp": "2026-05-28T12:00:00Z"
}
```

### Then
- Response `200 OK`
- Task status transitions to `done`
- Audit event `handoff.submitted` is appended
- SSE event is emitted to project stream within 2 seconds

---

## E2E Scenario: Parent cannot complete with unresolved required child

### Given
- Task `parent-task` in `proj-alpha`
- Required child `child-1` has status `in_progress`

### When
Actor attempts to complete parent:
```
POST /tasks/parent-task/done
{
  "actorId": "layer_a:bob",
  "actorRole": "layer_a"
}
```

### Then
- Response `409 Conflict`
- Parent task status remains unchanged
- Error body identifies the blocking child task(s)

---

## E2E Scenario: Parent cannot complete with open blocking gate

### Given
- Task `parent-task` in `proj-alpha`
- Task has a blocking `architecture_review` gate in state `open`

### When
```
POST /tasks/parent-task/done
{
  "actorId": "layer_a:bob",
  "actorRole": "layer_a"
}
```

### Then
- Response `409 Conflict`
- Parent task status remains unchanged
- Error body identifies the blocking gate

---

## E2E Scenario: Stale task detection does not auto-block

### Given
- Task `task-x` has been `in_progress` for longer than project's stale threshold
- No explicit `blocked` action has been taken

### When
Stale detection runs (configurable threshold, e.g., 24 hours of inactivity)

### Then
- Stale event is emitted: `task.stale.detected`
- Task appears in board API with stale indicator
- Task execution status remains `in_progress` (NOT auto-changed to `blocked`)
- Explicit action by human, Layer A, assigned Layer B, or system rule is required to block (per FR-02-020, FR-02-021)

---

## E2E Scenario: Manual blocked state requires reason

### Given
- Task `task-x` is `in_progress`
- Actor `human:alice` decides to block

### When
```
POST /tasks/task-x/block
{
  "executionStatus": "blocked",
  "reason": "Waiting for dependency from external team",
  "actorId": "human:alice",
  "actorRole": "human"
}
```

### Then
- Response `200 OK`
- Task status transitions to `blocked`
- Audit event `task.blocked` is appended with reason
- Reason is preserved and queryable in audit log

---

*End of task lifecycle E2E scenarios*