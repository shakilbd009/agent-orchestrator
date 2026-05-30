# evals/unit/brd-02-platform-orchestration-authorization.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** Unit test contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## Contract: AuthorizationMiddleware.EnforceRole

### Input
- Request with actor identity and role claim
- Required role for the endpoint

### Output
- If actor has required role: request proceeds
- If actor lacks required role: response `403 Forbidden`

### Edge cases:
- No actorId provided: `401 Unauthorized` (fail closed)
- actorId null: `401 Unauthorized`
- Role claim does not match required: `403 Forbidden`
- Multiple roles claimed: use most-privileged role for this action

---

## Contract: Layer A permissions — decomposition and routing

### Input
- Actor with `layer_a` role
- Request to: propose decomposition, approve decomposition (where configured), route child tasks, manage task-level gates (where authorized)

### Output
- Action permitted
- Audit event appended with actor role

### Edge cases:
- Layer A approving project-level phase gate (not delegated): `403 Forbidden`
- Layer A overriding decomposition limits: permitted if within hard caps
- Layer A self-approving own task: not applicable (Layer A doesn't get assigned tasks by default)

---

## Contract: Layer B permissions — assigned task only

### Input
- Actor with `layer_b` role
- Task assigned to this actor

### Output
- Task status update permitted
- Handoff submission permitted

### Edge cases:
- Layer B updating unassigned task: `403 Forbidden`
- Layer B approving own gate: `403 Forbidden`
- Layer B approving project-level gate: `403 Forbidden`
- Layer B overriding decomposition limits: `403 Forbidden`

---

## Contract: Human permissions — full scope within project

### Input
- Actor with `human` role
- Request: create tasks, approve phase gates, manage project config, override decomposition limits

### Output
- All actions permitted
- Audit events record human actor identity

### Edge cases:
- Human delegating phase gate approval to Layer A: requires explicit project configuration
- Human removing own access: not applicable (self-removal out of scope)

---

## Contract: system role — infrastructure only

### Input
- Actor with `system` role
- Request: emit audit events, update feature flag state

### Output
- Audit event emission: permitted
- Feature flag state update: permitted

### Edge cases:
- system attempting task mutation: `403 Forbidden`
- system attempting gate approval: `403 Forbidden`
- system attempting project creation: `403 Forbidden`
- system attempting any mutation beyond enumerated: `403 Forbidden`

---

## Contract: Decomposition idempotency enforcement

### Input
- Active proposal exists for parent task
- New decomposition request arrives

### Output
- Response: `200 OK` with existing proposal (idempotent) or `409 Conflict` (active proposal exists)
- No duplicate children created

### Edge cases:
- Re-submission with identical children: no-op or 409
- Re-submission with different children: rejected while active proposal exists
- Proposal was rejected: new proposal can supersede (allowed)

---

## Contract: Required child deletion protection

### Input
- Child task is marked `required` for parent
- Delete request for child task arrives

### Output
- Response `409 Conflict`
- Child not deleted
- Error: "cannot delete required child; perform scope adjustment first"

### Edge cases:
- Child marked non-required before delete: delete allowed
- Scope adjustment approved before delete: delete allowed

---

*End of authorization unit contracts*