# evals/e2e/brd-02-platform-orchestration-sse.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** End-to-end scenario contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## E2E Scenario: SSE client connects and receives events

### Given
- Project `proj-alpha` with `platform-orchestration=true`
- Dashboard (BRD-03) client is authenticated and connecting

### When
Client connects:
```
GET /projects/proj-alpha/events/stream
Headers:
  Accept: text/event-stream
```

### Then
- Response `200 OK` with `Content-Type: text/event-stream`
- Connection remains open
- Client receives committed orchestration events as SSE `event` stream
- SSE `event` field uses event `topic` (e.g., `task.status.changed`)
- SSE `id` field uses `eventId` (UUID)
- SSE `data` field contains full canonical event envelope as JSON (per FR-02-023, FR-02-022A)
- Latency: events visible within 2 seconds of commit (NFR-02-008)

---

## E2E Scenario: SSE event delivery uses canonical envelope

### Given
- SSE client is connected to `proj-alpha` event stream

### When
A task status changes:
```
PATCH /tasks/task-x
{
  "executionStatus": "in_progress",
  "actorId": "layer_b:agent1",
  "actorRole": "layer_b"
}
```

### Then
- SSE `event` = `task.status.changed`
- SSE `id` = UUID eventId
- SSE `data` contains:
  ```json
  {
    "eventId": "<uuid>",
    "schemaVersion": "v1alpha",
    "projectId": "proj-alpha",
    "topic": "task.status.changed",
    "actorId": "layer_b:agent1",
    "actorRole": "layer_b",
    "taskId": "task-x",
    "parentTaskId": null,
    "gateId": null,
    "timestamp": "<timestamp>",
    "payload": {
      "taskId": "task-x",
      "fromStatus": "todo",
      "toStatus": "in_progress"
    }
  }
  ```

---

## E2E Scenario: SSE reconnect with Last-Event-ID receives missed events

### Given
- SSE client was connected to `proj-alpha`
- Client was disconnected after receiving events up to `Last-Event-ID: event-abc123`
- New events `event-def456`, `event-ghi789` were committed while client was disconnected

### When
Client reconnects:
```
GET /projects/proj-alpha/events/stream
Headers:
  Accept: text/event-stream
  Last-Event-ID: event-abc123
```

### Then
- Client receives catch-up batch: `event-def456`, `event-ghi789`
- Events returned in ascending eventId order (ordering preserved)
- After catch-up, client receives live events
- This is dashboard continuity, NOT event replay as operational source of truth (per FR-02-023A)

---

## E2E Scenario: SSE reconnect recovers without operational state replay

### Given
- SSE client disconnects and reconnects with `Last-Event-ID`

### When
Reconnect is processed

### Then
- Events are fetched from immutable audit log (append-only)
- Current-state records remain authoritative for reads
- Reconnect does not imply the audit log is the operational source of truth (per FR-02-002)

---

## E2E Scenario: SSE client at max connections (50) is rejected

### Given
- Project `proj-alpha` already has 50 SSE clients connected (NFR-02-005 limit)

### When
Client 51 attempts connection:
```
GET /projects/proj-alpha/events/stream
```

### Then
- Response `503 Service Unavailable` or `429 Too Many Requests`
- Metric `orch_sse_clients_current` reflects at-limit state
- Connection rejected; no fanout to 51st client

---

## E2E Scenario: SSE delivery latency under 2 seconds

### Given
- SSE client is connected to `proj-alpha`
- Latency measurement begins when event is committed to audit log

### When
An event is committed

### Then
- SSE client receives the event within 2 seconds (NFR-02-008)
- Measured via `orch_sse_delivery_duration_ms` histogram

---

## E2E Scenario: SSE client disconnect removes from fanout

### Given
- SSE client connected to `proj-alpha`
- Client disconnects (close, timeout, network loss)

### When
Disconnection is detected

### Then
- Client channel is removed from fanout
- `orch_sse_clients_current` gauge decremented
- Audit event `sse.client.disconnected` appended
- Goroutine cleanup initiated (no goroutine leak per implementation threat)

---

## E2E Scenario: SSE ping/keepalive for idle connections

### Given
- SSE client connected and idle (no events for 30 seconds)

### When
Server sends keepalive

### Then
- SSE comment or `: ping` line sent to client
- Connection remains alive
- After 2 missed pings, server disconnects stale client

---

## E2E Scenario: CORS preflight for SSE endpoint

### Given
- Browser-based dashboard (BRD-03) connects to SSE endpoint
- Origin: `https://dashboard.example.com`

### When
Browser sends preflight:
```
OPTIONS /projects/proj-alpha/events/stream
Origin: https://dashboard.example.com
Access-Control-Request-Method: GET
```

### Then
- Response `200 OK` with CORS headers
- `Access-Control-Allow-Origin: <validated-origin>` (per ADR-02-006)
- Wildcard `*` only for `localhost` development
- Production origins must be explicitly allow-listed per project

---

*End of SSE E2E scenarios*