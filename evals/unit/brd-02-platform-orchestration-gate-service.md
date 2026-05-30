# evals/unit/brd-02-platform-orchestration-gate-service.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** Unit test contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## Contract: GateService.OpenGate

### Input
- Task ID or project ID
- Gate type from built-in templates: `scope_review`, `architecture_review`, `implementation_review`, `code_review`, `qa_review`, `release_review`
- Level: `task` or `project`
- Blocking: boolean

### Output
- Gate record created with state `open`
- Audit event `gate.opened` appended

### Edge cases:
- Gate already open on same task/phase: idempotent or 409 Conflict
- Invalid gate type: rejected as 422
- Gate on task with no children: allowed (per FR-02-017A independence)

---

## Contract: GateService.ApproveGate

### Input
- Gate ID
- Approver identity with role
- Approval timestamp

### Output
- Gate state transitions to `approved`
- Approver identity and timestamp recorded
- Audit event `gate.approved` appended with approver role

### Edge cases:
- Gate already `approved`: idempotent, no-op
- Gate already `rejected`: return 409 Conflict (must reopen or new gate)
- Approver not authorized for this gate type/level: 403 Forbidden
- Layer B attempting self-approval: 403 Forbidden (per FR-02-014)

---

## Contract: GateService.RejectGate

### Input
- Gate ID
- Rejection reason (required)
- Rejector identity

### Output
- Gate state transitions to `rejected`
- Rejection reason stored and queryable
- Guarded task/phase remains in non-advanced state

### Edge cases:
- Empty rejection reason: rejected as 422 (per FR-02-019)
- Gate already `approved`: return 409 Conflict
- Rejection does not automatically reopen gate (manual reopen required)

---

## Contract: GateService.GetBlockingGates

### Input
- Task ID

### Output
- Returns all gates where blocking=true and state in {open, rejected}

### Edge cases:
- No blocking gates: return empty list
- All blocking gates approved: return empty list
- Gate on task with no children: included in results if state is open/rejected

---

## Contract: GateService.EnforceBlockingGateOnTaskCompletion

### Input
- Task completion request
- Task's blocking gates

### Output
- If any blocking gate is open or rejected: task completion REJECTED
- Error response identifies the blocking gate(s)

### Edge cases:
- Advisory (non-blocking) gate: does not prevent completion
- Gate on task-level vs project-level: checked independently

---

## Contract: Project gate configuration — enable/disable templates

### Input
- Project ID
- Gate config update: disable `qa_review`, mark `release_review` as advisory

### Output
- Project gate configuration updated
- New tasks in project respect updated configuration

### Edge cases:
- Disable a gate type that has open gates: existing gates remain open
- Mark gate as blocking then advisory: existing blocking gates remain blocking until closed

---

## Contract: Project-level phase gate — human-only approval

### Input
- Phase gate ID
- Actor with `layer_a` role attempting approval

### Output
- Response `403 Forbidden`
- Phase gate remains open
- Only `human` role can approve unless explicitly delegated

### Edge cases:
- Human has delegated phase gate approval to Layer A: Layer A approval allowed
- Delegation recorded in project config: verifiable in audit

---

*End of gate service unit contracts*