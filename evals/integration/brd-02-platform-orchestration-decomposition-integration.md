# evals/integration/brd-02-platform-orchestration-decomposition-integration.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** Integration test contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## Integration Test: Decomposition proposal creates proposed children

### Setup
- Project `proj-alpha` with decomposition limits: depth=3, fan-out=20
- Task `parent-1` with no active proposal

### Steps
1. Submit proposal:
   ```
   POST /tasks/parent-1/decompose
   {
     "children": [{"title": "Child A"}, {"title": "Child B"}],
     "actorId": "layer_a:bob",
     "actorRole": "layer_a"
   }
   ```

### Assertions
- Response `201 Created`
- Proposal status is `proposed`
- Children created with status `proposed` (inactive)
- Children do NOT appear in board as active tasks
- Audit event `task.decomposition.proposed` appended

---

## Integration Test: Decomposition approval activates children

### Setup
- Task `parent-1` with active decomposition proposal

### Steps
1. Approve:
   ```
   POST /tasks/parent-1/proposal/approve
   {
     "actorId": "human:alice",
     "actorRole": "human"
   }
   ```

### Assertions
- Response `200 OK`
- Children status → `todo` (active)
- Children appear in project board
- Audit event `task.decomposition.approved` appended

---

## Integration Test: Decomposition rejection retains audit record

### Setup
- Task `parent-1` with active decomposition proposal

### Steps
1. Reject:
   ```
   POST /tasks/parent-1/proposal/reject
   {
     "reason": "Scope too large, needs further breakdown",
     "actorId": "human:alice",
     "actorRole": "human"
   }
   ```

### Assertions
- Response `200 OK`
- Proposal status → `rejected`
- Rejection reason stored and queryable
- No active children created
- Audit event `task.decomposition.rejected` appended with reason

---

## Integration Test: Decomposition idempotency — no duplicate proposals

### Setup
- Task `parent-1` with active proposal (status `proposed`)

### Steps
1. Submit identical proposal again

### Assertions
- Response `200 OK` with existing proposal (idempotent)
- Or: `409 Conflict` indicating active proposal exists
- No duplicate children created (per FR-02-009A)

---

## Integration Test: Fan-out limit enforcement

### Setup
- Project `proj-alpha` with max_children=20

### Steps
1. Submit proposal with 21 children

### Assertions
- Response `422 Unprocessable Entity`
- Error body: "21 children exceeds max_children=20"
- No proposal created

---

## Integration Test: Depth limit enforcement

### Setup
- Project `proj-alpha` with max_depth=3

### Steps
1. Create chain: parent → child1 → child2 → child3 (depth 3)
2. Attempt decomposition at depth 3 (would create depth 4)

### Assertions
- Step 2: `422 Unprocessable Entity`
- Hard cap depth=5 enforced; depth 4 within cap but exceeds project config

---

## Integration Test: Hard cap depth=5 enforcement

### Setup
- Project `proj-alpha`
- Task at depth 5 (creating depth 6 would exceed hard cap)

### Steps
1. Attempt decomposition at depth 5 (would produce depth 6 children)

### Assertions
- Response `422 Unprocessable Entity`
- Error body: "depth 6 exceeds hard cap 5"

---

## Integration Test: Override within hard caps requires audit reason

### Setup
- Project `proj-alpha` defaults: depth=3, fan-out=20

### Steps
1. Override:
   ```
   PATCH /projects/proj-alpha/decomposition-limits
   {
     "maxDepth": 4,
     "reason": "Complex security audit requires extra level",
     "actorId": "human:alice",
     "actorRole": "human"
   }
   ```

### Assertions
- Response `200 OK`
- New limits active
- Audit event `task.decomposition.override_used` with: actor identity, old limit, new limit, reason, timestamp

---

## Integration Test: Override exceeding hard caps rejected

### Setup
- Hard cap: depth=5

### Steps
1. Attempt override: `maxDepth=6`

### Assertions
- Response `422 Unprocessable Entity`
- Hard cap enforced; limits unchanged

---

*End of decomposition integration tests*