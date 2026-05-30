# evals/integration/brd-02-platform-orchestration-audit-integration.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** Integration test contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## Integration Test: Audit event appended on task mutation

### Setup
- Project `proj-alpha`
- Clean audit log

### Steps
1. Create task:
   ```
   POST /projects/proj-alpha/tasks
   {
     "title": "Test task",
     "actorId": "human:alice",
     "actorRole": "human"
   }
   ```
2. Query audit log for project

### Assertions
- Audit event appended with:
  - `eventId`: UUID
  - `schemaVersion`: "v1alpha"
  - `projectId`: "proj-alpha"
  - `topic`: "task.created"
  - `actorId`: "human:alice"
  - `actorRole`: "human"
  - `taskId`: "<new-task-id>"
  - `timestamp`: present
  - `payload`: contains task details

---

## Integration Test: Audit event appended on gate state change

### Setup
- Project `proj-alpha`
- Task `task-x` with open gate

### Steps
1. Approve gate:
   ```
   POST /gates/{gateId}/approve
   {
     "actorId": "human:alice",
     "actorRole": "human"
   }
   ```
2. Query audit log

### Assertions
- `gate.approved` event appended with canonical envelope
- Event includes: gateId, projectId, actor, role, timestamp

---

## Integration Test: Audit event on decomposition proposal

### Setup
- Project `proj-alpha`
- Task `parent-1` with no active proposal

### Steps
1. Propose decomposition:
   ```
   POST /tasks/parent-1/decompose
   ```

### Assertions
- `task.decomposition.proposed` event appended
- Event includes: parentTaskId, child count, actor, role

---

## Integration Test: Immutable audit log — no updates or deletes

### Setup
- Audit log with existing events
- Actor attempts to update or delete an audit event

### Steps
1. Attempt:
   ```
   PUT /audit-events/{eventId}
   ```
   Or:
   ```
   DELETE /audit-events/{eventId}
   ```

### Assertions
- Response `405 Method Not Allowed` or `403 Forbidden`
- Audit event record unchanged (immutable per NFR-02-013)

---

## Integration Test: Audit event includes all canonical envelope fields

### Setup
- Various event types: task creation, gate approval, decomposition, handoff

### Steps
1. Emit each event type
2. Inspect audit log entries

### Assertions
- Every event includes: eventId, schemaVersion, projectId, topic, actorId, actorRole, timestamp
- taskId, parentTaskId, gateId present when applicable, null/omitted when not applicable
- `payload` contains event-specific data

---

## Integration Test: Authorization denial audit event

### Setup
- Task `task-x` assigned to `layer_b:agent1`
- Actor `layer_b:agent2`

### Steps
1. Attempt to update unassigned task:
   ```
   PATCH /tasks/task-x
   ```

### Assertions
- Response `403 Forbidden`
- `auth.mutation.denied` audit event appended with: actor identity, attempted action, reason

---

## Integration Test: Canonical event envelope for SSE and webhooks

### Setup
- SSE client connected
- Webhook consumer registered

### Steps
1. Create task in project

### Assertions
- SSE `data` field contains canonical event envelope
- Webhook delivery payload contains canonical event envelope
- Both match the audit log entry (single source of truth for event data)

---

*End of audit integration tests*