# evals/integration/brd-02-platform-orchestration-gate-integration.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** Integration test contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## Integration Test: Task-level gate on task with no children

### Setup
- Project `proj-alpha`
- Task `task-x` with no children (not decomposed)
- `scope_review` gate template enabled

### Steps
1. Open gate:
   ```
   POST /tasks/task-x/gates
   {
     "gateType": "scope_review",
     "level": "task",
     "blocking": true,
     "actorId": "layer_a:bob",
     "actorRole": "layer_a"
   }
   ```

### Assertions
- Response `201 Created`
- Gate state is `open`
- Gate independent of decomposition state (per FR-02-017A, AC-02-019A)

---

## Integration Test: Rejected gate requires reason

### Setup
- Task `task-x` with blocking `implementation_review` gate in state `open`
- Actor `human:alice` authorized to reject

### Steps
1. Reject:
   ```
   POST /gates/{gateId}/reject
   {
     "reason": "Memory leak in event handler must be fixed before approval",
     "actorId": "human:alice",
     "actorRole": "human"
   }
   ```

### Assertions
- Response `200 OK`
- Gate state → `rejected`
- Rejection reason stored
- Guarded task remains in non-advanced state

---

## Integration Test: Rejected gate leaves task non-advanced

### Setup
- Task `task-x` with blocking gate in state `rejected`

### Steps
1. Attempt task completion:
   ```
   POST /tasks/task-x/done
   ```

### Assertions
- Response `409 Conflict`
- Task remains in current status

---

## Integration Test: Project-level phase gate requires human approval

### Setup
- Project `proj-alpha` at phase `G0`
- Phase gate for `G1 App Shell` open
- Actor `layer_a:bob`

### Steps
1. Attempt:
   ```
   POST /projects/proj-alpha/gates/{gateId}/approve
   ```

### Assertions
- Response `403 Forbidden` (unless human explicitly delegated)
- Phase remains `G0`

---

## Integration Test: Human approves project-level phase gate

### Setup
- Project `proj-alpha` at phase `G0`
- Phase gate open
- Actor `human:alice`

### Steps
1. Approve:
   ```
   POST /projects/proj-alpha/gates/{gateId}/approve
   {
     "actorId": "human:alice",
     "actorRole": "human"
   }
   ```

### Assertions
- Response `200 OK`
- Project phase → `G1`
- Audit event `gate.approved` with human approver

---

## Integration Test: Gate configuration — enable/disable and advisory

### Setup
- Project `proj-alpha`

### Steps
1. Disable `qa_review`, mark `release_review` as advisory:
   ```
   PATCH /projects/proj-alpha/gate-config
   {
     "gates": {
       "qa_review": {"enabled": false},
       "release_review": {"enabled": true, "blocking": false}
     }
   }
   ```

### Assertions
- Response `200 OK`
- `qa_review` gate not required for tasks in project
- `release_review` gate opens if configured but does not block advancement

---

## Integration Test: AC-02-016 — Human phase approval is manual PM gate

### Setup
- Project `proj-alpha` has phase gate `G2 Core Delivery`
- PM `human:alice` performs manual approval

### Steps
1. Approve phase gate:
   ```
   POST /projects/proj-alpha/gates/{gateId}/approve
   {
     "actorId": "human:alice",
     "actorRole": "human",
     "approvalType": "phase_advancement"
   }
   ```

### Assertions
- Response `200 OK`
- Phase advances
- Human approval NOT automatable — this is a manual PM gate (per AC-02-016 eval method)
- Audit event captures: actor=human, role=human, approvalType=phase_advancement

---

*End of gate integration tests*