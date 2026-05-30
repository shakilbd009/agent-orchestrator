# evals/e2e/brd-02-platform-orchestration-decomposition.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** End-to-end scenario contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## E2E Scenario: Layer A can submit decomposition proposal

### Given
- Project `proj-alpha` with `platform-orchestration=true`
- Task `parent-task` in `proj-alpha` has no active decomposition proposal
- Actor `layer_a:bob` is authenticated

### When
Bob proposes decomposition:
```
POST /tasks/parent-task/decompose
{
  "children": [
    {"title": "Sub-task A"},
    {"title": "Sub-task B"}
  ],
  "actorId": "layer_a:bob",
  "actorRole": "layer_a"
}
```

### Then
- Response `201 Created` with proposal object
- Proposed children do NOT appear in active board read as executable tasks
- Proposal status is `proposed`
- Audit event `task.decomposition.proposed` is appended

---

## E2E Scenario: Proposed children are inactive before approval

### Given
- Task `parent-task` has an active decomposition proposal with 2 proposed children

### When
```
GET /projects/proj-alpha/board
```

### Then
- Proposed children do NOT appear in board task list
- Children are not assigned to any actor
- Children do not affect parent completion semantics (per FR-02-008)

---

## E2E Scenario: Decomposition idempotency — no duplicate proposals

### Given
- Task `parent-task` already has an active proposal with status `proposed`
- Actor `layer_a:bob` attempts re-submission with identical children

### When
```
POST /tasks/parent-task/decompose
{
  "children": [
    {"title": "Sub-task A"},
    {"title": "Sub-task B"}
  ],
  "actorId": "layer_a:bob",
  "actorRole": "layer_a"
}
```

### Then
- Response `200 OK` with existing proposal (no new proposal created)
- Or: response `409 Conflict` indicating active proposal exists
- No duplicate children are created (per FR-02-009A idempotency)
- Audit event records idempotent handling

---

## E2E Scenario: Decomposition approval activates children

### Given
- Task `parent-task` has a decomposition proposal with 2 proposed children
- Actor `human:alice` is authorized to approve

### When
Alice approves:
```
POST /tasks/parent-task/proposal/approve
{
  "actorId": "human:alice",
  "actorRole": "human"
}
```

### Then
- Response `200 OK`
- Children transition to active tasks with `status: todo`
- Children are linked to parent via `parent_id`
- Audit event `task.decomposition.approved` is appended
- Children now appear in project board

---

## E2E Scenario: Decomposition rejection retains audit record

### Given
- Task `parent-task` has a decomposition proposal
- Actor `human:alice` rejects with reason

### When
```
POST /tasks/parent-task/proposal/reject
{
  "reason": "Sub-task B scope is too large; needs further breakdown",
  "actorId": "human:alice",
  "actorRole": "human"
}
```

### Then
- Response `200 OK`
- Proposal status becomes `rejected`
- Rejection reason is stored and queryable in audit
- No active child tasks are created
- Audit event `task.decomposition.rejected` is appended with reason

---

## E2E Scenario: Decomposition limits enforced at proposal time

### Given
- Project `proj-alpha` has max depth=3, max fan-out=20

### When
Actor attempts decomposition exceeding fan-out:
```
POST /tasks/parent-task/decompose
{
  "children": [<21 child titles>],
  "actorId": "layer_a:bob",
  "actorRole": "layer_a"
}
```

### Then
- Response `422 Unprocessable Entity`
- No proposal created
- Error body includes current limit and requested count
- Audit event with reason "fan-out limit exceeded"

---

## E2E Scenario: Decomposition limits enforced at depth cap

### Given
- Project `proj-alpha` has max depth=5 (hard cap)

### When
Actor attempts decomposition at depth 5 (creating depth 6 children):
```
POST /tasks/depth-5-task/decompose
{
  "children": [{"title": "Would exceed depth"}],
  "actorId": "layer_a:bob",
  "actorRole": "layer_a"
}
```

### Then
- Response `422 Unprocessable Entity`
- Error body includes "hard cap depth 5 exceeded"
- Audit event with reason "depth hard cap exceeded"

---

## E2E Scenario: Override within hard caps requires actor identity and audit reason

### Given
- Project `proj-alpha` defaults: depth=3, fan-out=20

### When
```
PATCH /projects/proj-alpha/decomposition-limits
{
  "maxDepth": 4,
  "reason": "Complex security audit requires extra decomposition level",
  "actorId": "human:alice",
  "actorRole": "human"
}
```

### Then
- Response `200 OK`
- New limits active for project
- Audit event `task.decomposition.override_used` captures: actor identity, old limit, new limit, reason, timestamp

---

## E2E Scenario: Override exceeding hard caps is rejected

### Given
- Project `proj-alpha` has hard cap depth=5

### When
```
PATCH /projects/proj-alpha/decomposition-limits
{
  "maxDepth": 6,
  "reason": "Testing",
  "actorId": "human:alice",
  "actorRole": "human"
}
```

### Then
- Response `422 Unprocessable Entity`
- Hard cap is enforced (per FR-02-011)
- Error body: "requested depth 6 exceeds hard cap 5"

---

## E2E Scenario: Rejected proposal can be superseded by new proposal

### Given
- Task `parent-task` has a rejected decomposition proposal with reason recorded

### When
Actor submits a new decomposition proposal:
```
POST /tasks/parent-task/decompose
{
  "children": [{"title": "Revised sub-task A"}],
  "actorId": "layer_a:bob",
  "actorRole": "layer_a"
}
```

### Then
- Response `201 Created` (new proposal)
- New proposal status is `proposed`
- Old rejected proposal is superseded (per FR-02-009A)
- Audit trail shows transition from rejected to new proposal

---

*End of decomposition E2E scenarios*