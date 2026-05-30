# evals/e2e/brd-02-platform-orchestration-gates.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** End-to-end scenario contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## E2E Scenario: Task-level gate on task with no children

### Given
- Project `proj-alpha` with `platform-orchestration=true`
- Task `task-x` has no children (not decomposed)
- Project has `scope_review` gate template enabled

### When
Actor opens a gate on `task-x`:
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

### Then
- Response `201 Created`
- Gate state is `open`
- Gate is independent of any decomposition state (per FR-02-017A)
- Task can have a gate open with zero children

---

## E2E Scenario: Blocking gate prevents task completion

### Given
- Task `task-x` has a blocking `code_review` gate in state `open`
- All required children are `done`

### When
Actor attempts task completion:
```
POST /tasks/task-x/complete
{
  "summary": "Implementation done",
  "validationPerformed": "Unit tests pass",
  "risks": "None",
  "residuals": "None",
  "actorId": "layer_b:agent1",
  "actorRole": "layer_b",
  "timestamp": "2026-05-28T12:00:00Z"
}
```

### Then
- Response `409 Conflict`
- Task status remains `in_progress`
- Error body identifies the blocking gate (per AC-02-013)

---

## E2E Scenario: Rejected gate requires reason and preserves non-advanced state

### Given
- Task `task-x` has a blocking `implementation_review` gate in state `open`
- Actor `human:alice` is authorized to reject

### When
Alice rejects:
```
POST /gates/{gateId}/reject
{
  "reason": "Memory leak detected in the event handler; must fix before approval",
  "actorId": "human:alice",
  "actorRole": "human"
}
```

### Then
- Response `200 OK`
- Gate state becomes `rejected`
- Guarded task/phase remains in non-advanced state (per FR-02-019, AC-02-019)
- Rejection reason is stored and queryable

---

## E2E Scenario: Project-level phase gate requires human approval

### Given
- Project `proj-alpha` is at phase `G0 Foundation & Governance`
- Project has a phase gate for `G1 App Shell`

### When
Actor `layer_a:bob` attempts to approve the phase gate:
```
POST /projects/proj-alpha/gates/{gateId}/approve
{
  "actorId": "layer_a:bob",
  "actorRole": "layer_a"
}
```

### Then
- Response `403 Forbidden` (unless human has explicitly delegated authority in project config)
- Project phase remains `G0`
- Project-level phase advancement requires `human` role (per FR-02-016, AC-02-017)

---

## E2E Scenario: Human can approve project-level phase gate

### Given
- Project `proj-alpha` has phase gate for `G1 App Shell` in state `open`
- Actor `human:alice` is authenticated

### When
```
POST /projects/proj-alpha/gates/{gateId}/approve
{
  "actorId": "human:alice",
  "actorRole": "human"
}
```

### Then
- Response `200 OK`
- Project phase advances from `G0` to `G1`
- Audit event `gate.approved` is appended with approver role

---

## E2E Scenario: Project can enable/disable built-in task gate templates

### Given
- Project `proj-alpha` has built-in gate templates: `scope_review`, `architecture_review`, `implementation_review`, `code_review`, `qa_review`, `release_review`

### When
Project owner disables `qa_review` and marks `release_review` as advisory:
```
PATCH /projects/proj-alpha/gate-config
{
  "gates": {
    "qa_review": {"enabled": false},
    "release_review": {"enabled": true, "blocking": false}
  },
  "actorId": "human:alice",
  "actorRole": "human"
}
```

### Then
- Response `200 OK`
- Tasks in `proj-alpha` no longer require `qa_review` gate
- `release_review` gate opens if configured but does not block phase advancement

---

## E2E Scenario: Gate rejection leaves task in non-advanced state

### Given
- Task `task-x` has a blocking `scope_review` gate in state `open`
- Rejection reason: "scope too broad for single task"

### When
Gate is rejected by authorized approver

### Then
- Task does not advance (no implicit state change)
- Task remains in current execution status
- Task cannot be marked `done` while gate is `rejected`
- Re-approval of gate is required (or scope adjustment)

---

## E2E Scenario: AC-02-016 human phase approval — manual PM gate + audit inspection

### Given
- Project `proj-alpha` has a project-level phase gate `G2 Core Delivery`
- Actor `human:alice` (PM) is authenticated

### When
Alice performs phase approval:
```
POST /projects/proj-alpha/gates/{gateId}/approve
{
  "actorId": "human:alice",
  "actorRole": "human",
  "approvalType": "phase_advancement",
  "notes": "All G1 deliverables verified; PM sign-off"
}
```

### Then
- Response `200 OK`
- Phase advances
- Audit event `gate.approved` records: actor=human:alice, role=human, type=phase_advancement
- Human approval is NOT automatable (per AC-02-016 eval method: manual PM gate + audit inspection)
- This gate cannot be approved by Layer A or Layer B without explicit human delegation

---

*End of gates E2E scenarios*