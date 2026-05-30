# evals/unit/brd-02-platform-orchestration-decomposition-service.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** Unit test contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## Contract: DecompositionService.CreateProposal — idempotency check

### Input
- Parent task ID
- Children list (2 child titles)
- Actor identity (layer_a:bob)

### Output
- Proposal created with status `proposed`
- Children created with status `proposed`

### Edge cases:
- Active proposal already exists: return existing proposal (idempotent) or return 409 Conflict
- Empty children list: reject as 422 (minimum 1 child per FR recommendation)
- Child title at DB column limit: reject at 422 with clear message

---

## Contract: DecompositionService.CreateProposal — limits enforcement

### Input
- Parent task with current depth level and fan-out count
- Children list exceeding project limits

### Output
- Proposal rejected with 422
- Error body includes: current limit, requested count, reason

### Edge cases:
- Depth exactly at hard cap (5): attempt to add child at depth 6 → REJECTED
- Fan-out exactly at hard cap (50): attempt to add 51st child → REJECTED
- Override active: use overridden limit, not project default

---

## Contract: DecompositionService.ApproveProposal

### Input
- Proposal ID with status `proposed`
- Approver identity (human:alice)

### Output
- Children status transitions from `proposed` to `todo`
- Proposal status transitions to `approved`
- Audit event `task.decomposition.approved` appended

### Edge cases:
- Proposal already `approved`: idempotent, no-op
- Proposal already `rejected`: return 409 Conflict (cannot approve rejected)
- Approver is not authorized: 403 Forbidden

---

## Contract: DecompositionService.RejectProposal

### Input
- Proposal ID with status `proposed`
- Rejection reason (required)
- Rejector identity (human:alice)

### Output
- Proposal status transitions to `rejected`
- Rejection reason stored and queryable
- Audit event `task.decomposition.rejected` appended with reason

### Edge cases:
- Empty rejection reason: reject as 422 (reason required per FR-02-019)
- Proposal already `approved`: return 409 Conflict

---

## Contract: DecompositionService.GetActiveProposal

### Input
- Parent task ID

### Output
- Returns active proposal if exists (status `proposed`)
- Returns null if no active proposal

### Edge cases:
- Multiple active proposals: should never occur; first check for atomic creation
- Proposal superseded: return current active proposal, not historical

---

## Contract: Decomposition limits — project defaults

### Input
- New project creation with default decomposition limits

### Output
- Project defaults: max_depth=3, max_children=20
- Limits apply to all tasks in project unless overridden

### Edge cases:
- Project config specifies lower limits: use configured values
- Project config specifies limits above hard caps: reject at 422

---

## Contract: Decomposition limits — override within hard caps

### Input
- Override request: max_depth from 3 to 4
- Override reason: "Complex security review"
- Actor identity (human:alice)

### Output
- Override applied; new limits active
- Audit event `task.decomposition.override_used` captures all fields

### Edge cases:
- Override without reason: reject as 422 (audit reason required per FR-02-011)
- Override to exact hard cap (depth=5): allowed
- Override exceeds hard cap (depth=6): rejected with 422

---

*End of decomposition service unit contracts*