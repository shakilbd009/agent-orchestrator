# evals/integration/brd-02-platform-orchestration-sse-integration.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** Integration test contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## Integration Test: SSE client receives events within 2 seconds

### Setup
- Backend server running
- Project `proj-alpha` with `platform-orchestration=true`
- SSE client connects to `GET /projects/proj-alpha/events/stream`

### Steps
1. SSE client connected
2. Execute task mutation:
   ```
   POST /tasks/task-x/complete
   {
     "summary": "Done",
     "validationPerformed": "Tests pass",
     "risks": "None",
     "residuals": "None",
     "actorId": "layer_b:agent1",
     "actorRole": "layer_b",
     "timestamp": "2026-05-28T12:00:00Z"
   }
   ```
3. Measure time from commit to SSE delivery

### Assertions
- SSE `data` received within 2 seconds of commit (NFR-02-008)
- SSE `event` = `task.status.changed`
- SSE `id` = eventId
- SSE `data` contains full canonical event envelope

---

## Integration Test: SSE reconnect with Last-Event-ID receives missed events

### Setup
- SSE client connected, receives events up to `event-abc123`
- Client disconnects
- New events `event-def456`, `event-ghi789` committed

### Steps
1. Client reconnects with header `Last-Event-ID: event-abc123`

### Assertions
- Client receives catch-up: `event-def456`, `event-ghi789`
- Events in ascending eventId order
- After catch-up, live events stream
- Ordering preserved throughout (per FR-02-023A)

---

## Integration Test: SSE client limit (50) enforced

### Setup
- Project `proj-alpha` with 50 SSE clients connected

### Steps
1. 51st client attempts connection

### Assertions
- Response `503 Service Unavailable` or `429 Too Many Requests`
- Metric `orch_sse_clients_current` at limit
- 51st connection rejected; existing 50 clients unaffected

---

## Integration Test: SSE event format correctness

### Setup
- SSE client connected to `proj-alpha`

### Steps
1. Create task:
   ```
   POST /projects/proj-alpha/tasks
   ```

### Assertions
- SSE `event:` line contains task topic (e.g., `task.created`)
- SSE `id:` line contains eventId (UUID)
- SSE `data:` line contains full canonical envelope as JSON
- Proper SSE formatting: fields terminated with `\n`, double newline at end

---

## Integration Test: SSE disconnect cleanup

### Setup
- SSE client connected to `proj-alpha`

### Steps
1. Client closes connection (TCP FIN or timeout)

### Assertions
- Client removed from fanout list
- `orch_sse_clients_current` gauge decremented
- Audit event `sse.client.disconnected` appended

---

## Integration Test: SSE CORS preflight

### Setup
- Browser dashboard (BRD-03) connecting to SSE endpoint
- Origin: `https://dashboard.example.com`
- Project has allow-list with `https://dashboard.example.com`

### Steps
1. Browser sends OPTIONS preflight

### Assertions
- Response `200 OK`
- `Access-Control-Allow-Origin: https://dashboard.example.com`
- `Access-Control-Allow-Methods: GET`
- Wildcard only for localhost (per ADR-02-006)

---

## Integration Test: SSE delivery latency measurement

### Setup
- SSE client connected
- `orch_sse_delivery_duration_ms` histogram available

### Steps
1. Emit event
2. Record timestamp at commit
3. Record timestamp when SSE client receives

### Assertions
- Delta < 2000ms (NFR-02-008)
- Metric recorded in histogram

---

*End of SSE integration tests*