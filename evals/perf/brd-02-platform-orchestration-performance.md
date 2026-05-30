# evals/perf/brd-02-platform-orchestration-performance.md

**Project:** agent-orchestrator
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline
**Type:** Performance test contracts
**Owner:** qa
**Status:** 🔴 **Failing** (before implementation)

---

## Performance Test: Board read latency under 300ms at 10,000 tasks

**NFR:** NFR-02-006
**AC:** AC-02-026
**Test type:** Load/stress test

### Setup
- Project `proj-perf` created with `FF_ENABLE_PLATFORM_ORCHESTRATION=true`
- Project populated with 10,000 tasks across all execution statuses
- Tasks include: parent/child dependencies, gate states, stale indicators, assignments, and handoff summaries
- Database indexes in place (project_id, execution_status, parent_task_id)

### Load profile
- Single client measuring round-trip latency
- Test measures cold query latency (no warm-up caching assumed for Phase 1)
- 10 consecutive measurements; report median and p95

### Acceptance threshold
- p50 latency < 300ms
- p95 latency < 500ms
- No 5xx errors during measurement window

### Assertions
- Response body contains tasks grouped by execution status
- Response body includes dependency relationships, gate states, stale indicators, assignments, and handoff summaries
- Response is structurally valid (parseable JSON, all expected top-level fields present)

### Metrics collected
- `orch_api_request_duration_ms` histogram (route: board read)
- `orch_db_operation_duration_ms` histogram
- Wall-clock round-trip time per request

---

## Performance Test: Task and gate mutation latency under 500ms

**NFR:** NFR-02-007
**AC:** AC-02-027
**Test type:** Throughput + latency

### Setup
- Project `proj-perf-mut` with existing tasks and open gates
- Actor identity: `human:perftest` with role `human`
- Baseline audit event count recorded

### Load profile
- Sequential mutations: task status change, gate approval, task creation
- Measure each mutation's round-trip from request sent to 200/201 response received
- 20 consecutive mutations; report median and p95

### Acceptance threshold
- p50 latency < 500ms (excluding async webhook delivery)
- p95 latency < 750ms
- All mutations produce exactly one audit event via the canonical envelope

### Assertions
- Mutating endpoint returns within threshold
- Current-state record updated (query to verify)
- Immutable audit event appended (query events API or event log table)
- No partial state (mutation commit + audit append are atomic or consistently ordered)

### Metrics collected
- `orch_api_request_duration_ms` histogram (route: task mutation, gate mutation)
- `orch_db_operation_duration_ms` histogram (write operations)
- `orch_task_status_changed_total` counter
- `orch_gate_approved_total` counter

---

## Performance Test: SSE delivery latency under 2 seconds

**NFR:** NFR-02-008
**AC:** AC-02-022
**Test type:** Event delivery latency

### Setup
- Project `proj-perf-sse` with 1 SSE client already connected
- Actor identity: `human:perftest`
- SSE client records wall-clock time of event receipt

### Steps
1. Client connects to `GET /projects/proj-perf-sse/events/stream`
2. Client sends task mutation:
   ```
   PATCH /projects/proj-perf-sse/tasks/{taskId}/status
   { "status": "in_progress", "actorId": "human:perftest", "actorRole": "human" }
   ```
3. Measure delta from step 2 response received to SSE event received by client

### Acceptance threshold
- p50 event delivery latency < 2 seconds
- p95 event delivery latency < 3 seconds

### Assertions
- SSE `event` field equals the canonical event topic
- SSE `id` field equals `eventId`
- SSE `data` field contains full event envelope (eventId, schemaVersion, projectId, topic, actorId, actorRole, timestamp, taskId, payload)

### Metrics collected
- `orch_sse_delivery_duration_ms` histogram
- `orch_sse_clients_current` gauge

---

## Performance Test: Webhook enqueue latency under 1 second

**NFR:** NFR-02-009
**AC:** AC-02-023
**Test type:** Async job enqueue latency

### Setup
- Project `proj-perf-wh` with at least one registered webhook consumer
- Webhook consumer URL reachable (mock or local endpoint)
- Actor identity: `human:perftest`

### Steps
1. Record current webhook enqueue counter: `orch_webhook_enqueued_total`
2. Send task mutation that has subscribed webhook events
3. Poll or wait for `orch_webhook_enqueued_total` to increment
4. Record wall-clock time delta from step 2 response received to increment confirmed

### Acceptance threshold
- p50 enqueue latency < 1 second
- p95 enqueue latency < 2 seconds

### Assertions
- Webhook job appears in enqueue queue within threshold
- Webhook delivery is asynchronous (original mutation returns before delivery)
- Original mutation succeeds even if webhook receiver is slow or unavailable (per NFR-02-012)

### Metrics collected
- `orch_webhook_enqueued_total` counter delta
- Wall-clock enqueue time
- `orch_webhook_retry_total` on failure scenarios

---

## Performance Test: SSE concurrent client scale — 50 clients

**NFR:** NFR-02-005
**Test type:** Concurrency stress test

### Setup
- Project `proj-perf-scale` with 50 simulated SSE clients
- Each client connected to `GET /projects/proj-perf-scale/events/stream`
- Clients acknowledge receipt with a sequence number recorded server-side or by a test harness

### Load profile
- Single mutation triggers one event
- Measure time for 90% of clients to confirm receipt
- Test stability: connect and disconnect 10 clients repeatedly while broadcasting events

### Acceptance threshold
- All 50 clients receive the same event within the SSE delivery latency threshold (2 seconds)
- No clients silently dropped during sustained connection
- New connections rejected with 503 when at 50-client limit

### Assertions
- 51st concurrent connection attempt receives 503 or 429
- `orch_sse_clients_current` reflects actual connection count
- Clients reconnecting with `Last-Event-ID` receive missed events correctly

### Metrics collected
- `orch_sse_clients_current` gauge
- `orch_sse_delivery_duration_ms` histogram per broadcast
- Rejected connection counter

---

## Performance Test: Operational simplicity — no broker dependency

**NFR:** NFR-02-015
**Test type:** Dependency/infrastructure audit

### Setup
- Review implementation dependencies at build time and at runtime
- Document all infrastructure components required to run BRD-02 orchestration

### Acceptance threshold
- No external message broker (NATS, Kafka, RabbitMQ, Redis Streams, etc.) required at runtime
- No external job queue system required at runtime
- Webhook delivery uses in-process or database-backed queue (bounded retry)
- SSE fanout is in-process (no pub/sub service dependency)

### Assertions
1. `go.mod` does not include broker client libraries (nats, kafka, amqp, etc.)
2. `docker-compose.yml` (if present) does not define broker services
3. Runtime startup does not require connectivity to external broker hosts
4. `GET /ready` returns 200 without any broker connectivity

### Metrics collected
- List of runtime dependencies (processes or services required)
- Startup warning log if broker is detected as a configured option

---

## Performance Test: Webhook failure isolation — no mutation rollback

**NFR:** NFR-02-010
**Test type:** Failure injection

### Setup
- Project `proj-perf-isol` with at least one registered webhook consumer
- Webhook consumer configured and reachable before failure injection
- Baseline: normal mutation succeeds and webhook is delivered

### Steps
1. Make a task mutation (e.g., `PATCH /projects/proj-perf-isol/tasks/{taskId}/status`)
2. Confirm mutation returns 200
3. Immediately crash/inject failure into the webhook receiver
4. Confirm the task mutation is committed and visible via board read
5. Attempt subsequent mutations — confirm they succeed regardless of webhook receiver state

### Acceptance threshold
- Task mutation commits regardless of webhook receiver state
- No rollback of committed mutation state when webhook delivery fails or receiver is down

### Assertions
- Task status reflects the mutation from step 1
- `orch_webhook_delivery_failed_total` increments after retry exhaustion
- Subsequent mutations succeed (no backpressure from webhook failures)

### Metrics collected
- `orch_webhook_delivery_succeeded_total`
- `orch_webhook_delivery_failed_total`
- `orch_webhook_retry_total`

---

## Performance Test: Read availability during webhook outage

**NFR:** NFR-02-012
**Test type:** Degraded mode

### Setup
- SSE clients connected to `proj-perf-read`
- Webhook receiver deliberately taken offline or network-partitioned

### Steps
1. Confirm SSE client count > 0
2. Take webhook receiver offline
3. Perform board read and task mutation on `proj-perf-read`

### Acceptance threshold
- Board read returns 200 and correct data within latency threshold
- Task mutations succeed within latency threshold
- SSE event stream continues uninterrupted

### Assertions
- `GET /live` returns 200
- `GET /ready` returns 200 (or 200 with degraded flag, but not 503)
- No impact to in-progress SSE connections

---

## Performance Benchmark Summary

| NFR | Metric | Threshold | Test |
|-----|--------|-----------|------|
| NFR-02-005 | SSE concurrent clients | 50 max | SSE concurrent client scale test |
| NFR-02-006 | Board read latency | < 300ms (p50), < 500ms (p95) | Board read latency at 10k tasks |
| NFR-02-007 | Mutation latency | < 500ms (p50), < 750ms (p95) | Task/gate mutation latency |
| NFR-02-008 | SSE delivery latency | < 2s (p50), < 3s (p95) | SSE event delivery latency |
| NFR-02-009 | Webhook enqueue latency | < 1s (p50), < 2s (p95) | Webhook enqueue latency |
| NFR-02-010 | Webhook failure isolation | No rollback of committed state | Webhook failure injection |
| NFR-02-012 | Read availability during webhook outage | Board reads succeed | Read availability test |
| NFR-02-015 | No broker dependency | Zero external broker at runtime | Dependency audit |

---

*End of BRD-02 performance eval contracts*