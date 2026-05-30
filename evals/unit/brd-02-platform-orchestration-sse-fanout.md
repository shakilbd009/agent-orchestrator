# evals/unit/brd-02-platform-orchestration-sse-fanout.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** Unit test contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## Contract: SSEManager.AddClient

### Input
- Project ID
- SSE client channel
- Client connection metadata (connected_at, last_event_id for catch-up)

### Output
- Client registered in per-project channel list
- `orch_sse_clients_current` gauge incremented
- Audit event `sse.client.connected` appended

### Edge cases:
- 51st connection when limit is 50: reject with 503; metric incremented
- Duplicate client registration: idempotent or reject
- Invalid project ID: 404 Not Found

---

## Contract: SSEManager.RemoveClient

### Input
- Project ID
- Client channel

### Output
- Client channel removed from fanout list
- `orch_sse_clients_current` gauge decremented
- Goroutine cleanup initiated
- Audit event `sse.client.disconnected` appended

### Edge cases:
- Client not found in list: no-op (already removed)
- Remove during event broadcast: safe (broadcast completes before removal)

---

## Contract: SSEManager.BroadcastEvent

### Input
- Project ID
- Canonical event envelope

### Output
- Event sent to all registered client channels
- SSE format: `event: <topic>\nid: <eventId>\ndata: <full JSON>\n\n`
- `orch_sse_delivery_duration_ms` histogram recorded

### Edge cases:
- Slow client (buffer full): non-blocking; event not queued indefinitely
- Client disconnect during broadcast: caught and cleaned up
- Zero clients: no-op (no error)

---

## Contract: SSEManager.ReconnectCatchUp

### Input
- Reconnecting client with `Last-Event-ID: <eventId>`
- Project ID

### Output
- Query audit log for events where eventId > lastSeenId
- Send catch-up batch in ascending eventId order
- Resume live fanout after catch-up batch
- Ordering preserved throughout (per FR-02-023A)

### Edge cases:
- Last-Event-ID not found in audit log: send all events from oldest available
- Large gap (many missed events): batch size reasonable (implementation choice)
- Empty catch-up (no missed events): resume live immediately

---

## Contract: SSE ping/keepalive

### Input
- Idle SSE client (no events for 30 seconds)

### Output
- Server sends `: ping\n\n` comment or SSE comment line
- Connection kept alive

### Edge cases:
- 2 missed pings: disconnect stale client
- Client explicitly disabled keepalive: respect config (if provided)

---

## Contract: SSE max client limit

### Input
- 50 clients already connected to `proj-alpha`
- 51st connection request arrives

### Output
- Response `503 Service Unavailable` or `429 Too Many Requests`
- `orch_sse_clients_current` reflects at-limit state
- Metric for rejected connections incremented

### Edge cases:
- Client disconnects quickly after rejection: gauge drops; next connection may succeed
- Concurrent reconnect attempts: all handled atomically

---

*End of SSE fanout unit contracts*