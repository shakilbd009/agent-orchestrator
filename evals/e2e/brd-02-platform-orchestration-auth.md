# evals/e2e/brd-02-platform-orchestration-auth.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** End-to-end scenario contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## E2E Scenario: Unauthenticated mutation is rejected

### Given
- Project `proj-alpha` exists
- No actor identity provided

### When
```
POST /projects/proj-alpha/tasks
{
  "title": "Test task",
  "actorRole": "human"
  // no actorId
}
```

### Then
- Response `401 Unauthorized` (fail closed per FR-02-012)
- No state change occurs
- Security-relevant audit event may be appended (per Open Question 9 resolution)

---

## E2E Scenario: Null actorId fails closed

### Given
- Request with `actorId: null` or missing

### When
Any mutating endpoint is called

### Then
- Response `401 Unauthorized`
- Fail closed — reject with clear authorization error (per edge case in L2)

---

## E2E Scenario: Layer A can perform decomposition and routing but not phase gates

### Given
- Actor `layer_a:bob` is authenticated with role `layer_a`
- Project `proj-alpha` does NOT have explicit human delegation for phase gate approval

### When
Bob attempts to approve a project-level phase gate:
```
POST /projects/proj-alpha/gates/{gateId}/approve
{
  "actorId": "layer_a:bob",
  "actorRole": "layer_a"
}
```

### Then
- Response `403 Forbidden`
- Layer A cannot approve project-level phase gates unless human has delegated authority (per FR-02-013)

---

## E2E Scenario: Layer A can propose decomposition and manage task gates

### Given
- Actor `layer_a:bob` is authenticated

### When
Bob proposes decomposition:
```
POST /tasks/task-x/decompose
{
  "children": [{"title": "Child task"}],
  "actorId": "layer_a:bob",
  "actorRole": "layer_a"
}
```

### Then
- Response `201 Created`
- Proposal is created

---

## E2E Scenario: system role can emit audit events but cannot mutate tasks

### Given
- Actor `system:orchestration-service` is authenticated with role `system`

### When
System attempts to complete a task:
```
POST /tasks/task-x/complete
{
  "summary": "Auto-complete",
  "actorId": "system:orchestration-service",
  "actorRole": "system"
}
```

### Then
- Response `403 Forbidden`
- `system` role is infrastructure-only (per ADR-02-005): permitted for audit events and feature-flag-change events only
- No task mutation permitted

---

## E2E Scenario: system role can update feature flag state

### Given
- Actor `system:feature-flag-service` is authenticated with role `system`

### When
```
POST /projects/proj-alpha/feature-flags/platform-orchestration
{
  "enabled": true,
  "actorId": "system:feature-flag-service",
  "actorRole": "system"
}
```

### Then
- Response `200 OK`
- Feature flag state updated
- Audit event `feature_flag.updated` appended (per ADR-02-005)

---

## E2E Scenario: Unauthorized mutation attempt fails closed and appends audit event

### Given
- Actor `layer_b:agent1` attempts to mutate task assigned to `layer_b:agent2`

### When
```
PATCH /tasks/task-2/status
{
  "executionStatus": "done",
  "actorId": "layer_b:agent1",
  "actorRole": "layer_b"
}
```

### Then
- Response `403 Forbidden`
- Task state unchanged
- Audit event `auth.mutation.denied` appended with actor identity and attempted action

---

## E2E Scenario: Role hierarchy used for authorization decisions

### Given
- Actor presents credentials claiming both `layer_a` and `layer_b` roles

### When
```
POST /tasks/task-x/complete
{
  "actorId": "agent1",
  "actorRole": "layer_a,layer_b"
}
```

### Then
- System uses most-privileged role for this action
- For task completion as Layer B assignee: `layer_b` applies
- For gate approval as Layer A: `layer_a` applies
- Most-privileged role selected per action type (per edge case L2)

---

## E2E Scenario: Layer B cannot complete task assigned to different Layer B

### Given
- Task `task-x` is assigned to `layer_b:agent1`
- Actor `layer_b:agent2` is authenticated

### When
Agent2 attempts to complete task-x:
```
POST /tasks/task-x/complete
{
  "summary": "Done by agent2",
  "actorId": "layer_b:agent2",
  "actorRole": "layer_b"
}
```

### Then
- Response `403 Forbidden`
- Only the current assignee may complete their assigned task (per edge case L2)

---

*End of auth E2E scenarios*