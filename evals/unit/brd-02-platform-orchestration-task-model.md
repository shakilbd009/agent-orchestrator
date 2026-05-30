# evals/unit/brd-02-platform-orchestration-task-model.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** Unit test contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## Contract: Task model execution statuses

### Input
Task creation with execution status set to one of: `todo`, `in_progress`, `blocked`, `done`, `cancelled`

### Output
- Task record stored with correct execution status
- Status transition events appended to audit log
- Board API reflects correct status grouping

### Edge cases:
- `null` status: rejected as 422
- Invalid status string: rejected as 422
- Status at column limit: handled per DB limits (recommend 512 chars)

---

## Contract: Governance state is separate from execution status

### Input
Task with execution status `in_progress` and governance state (decomposition `proposed`, gate `open`, stale `true`)

### Output
- Execution status and governance state stored in separate fields
- Board API returns both independently
- Changing execution status does not reset governance state

### Edge cases:
- Governance state changed while task is `done`: allowed (post-completion gate review)
- Task `cancelled` with open gate: gate remains in audit record

---

## Contract: Required child semantics

### Input
- Parent task `parent-1` with children `child-1` (required=true) and `child-2` (required=false)

### Output
- `child-1` must reach `done` before `parent-1` can complete
- `child-2` is optional; parent can complete regardless of `child-2` state

### Edge cases:
- `child-1` cancelled: parent completion blocked until scope adjustment (per AC-02-012)
- `child-1` status changed to non-required after creation: parent completion allowed
- Required child deleted: platform rejects deletion with 409 Conflict (per edge case L2)

---

## Contract: Strict parent completion

### Input
Parent task with all required children `done` and all blocking gates approved

### Output
Parent can be marked `done`

### Edge cases:
- One required child is `in_progress`: parent completion REJECTED with 409
- One required child is `blocked`: parent completion REJECTED
- One required child is `cancelled` without approved scope reduction: REJECTED
- One blocking gate is `open`: REJECTED
- One blocking gate is `rejected`: REJECTED

---

## Contract: Task workspace and priority

### Input
Task creation with workspace kind (`scratch`, `dir`, `worktree`) and priority

### Output
- Workspace kind stored and returned in task record
- Priority used for ordering in board read
- Workspace path validated as absolute when kind is `dir` or `worktree`

### Edge cases:
- Relative path for `dir` workspace: rejected as 422
- Negative priority: allowed (lower priority)
- Priority at boundary (9999): handled without overflow

---

*End of task model unit contracts*