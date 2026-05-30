# evals/integration/brd-02-platform-orchestration-webhook-integration.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** Integration test contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## Integration Test: Webhook delivery does not block original mutation

### Setup
- Project `proj-alpha` with webhook consumer for `task.*`
- Consumer URL: `https://consumer.example.com/webhook` (returns 500)
- Task `task-x` assigned to `layer_b:agent1`

### Steps
1. Complete task:
   ```
   POST /tasks/task-x/complete
   ```
2. Measure response time

### Assertions
- Response returns immediately (async webhook delivery)
- Task status → `done`
- Webhook delivery is enqueued within 1 second (NFR-02-009)
- Original mutation NOT blocked regardless of webhook outcome (per NFR-02-010)

---

## Integration Test: Webhook retry with exponential backoff

### Setup
- Webhook consumer returns `500 Internal Server Error`
- Default retry count: 3

### Steps
1. Trigger event that webhook consumer would fail to process
2. Observe retry attempts and timing

### Assertions
- Retry 1: ~1 second after initial failure
- Retry 2: ~2 seconds after retry 1
- Retry 3: ~4 seconds after retry 2
- After 3 failures: exhausted; `orch_webhook_delivery_failed_total` incremented
- `webhook.delivery.failed` audit event appended

---

## Integration Test: Webhook 301 redirect followed (max 1 hop)

### Setup
- Webhook consumer URL returns `301 Redirect` to another URL

### Steps
1. Trigger event

### Assertions
- Delivery followed to final URL (max 1 hop)
- Delivery succeeds at final URL if it returns 2xx
- If redirect chain > 1 hop: permanent failure, logged

---

## Integration Test: Webhook timeout treated as failure

### Setup
- Webhook consumer is slow (>30s response time)

### Steps
1. Trigger event

### Assertions
- Delivery treated as failure after timeout
- Retry initiated with backoff
- After exhaustion: failure logged

---

## Integration Test: Webhook receiver outage does not roll back committed state

### Setup
- Task `task-x` completed and committed
- Webhook consumer is down (unreachable or returning 500)

### Steps
1. Complete task
2. Webhook retries exhausted

### Assertions
- Task status remains `done` (not rolled back)
- Webhook failure visible in audit/event views
- `GET /projects/proj-alpha/tasks/task-x` shows `done`

---

## Integration Test: Webhook topic prefix subscription

### Setup
- Project `proj-alpha` has webhook registered for `task.decomposition` prefix

### Steps
1. Create decomposition proposal (emits `task.decomposition.proposed`)
2. Create task (emits `task.created`)
3. Approve gate (emits `gate.approved`)

### Assertions
- Consumer receives delivery for `task.decomposition.proposed`
- Consumer does NOT receive `task.created`
- Consumer does NOT receive `gate.approved`

---

## Integration Test: Webhook signature verification

### Setup
- Webhook consumer registered with shared secret
- Consumer URL not localhost

### Steps
1. Trigger event

### Assertions
- Request includes `X-Webhook-Signature: HMAC-SHA256=<signature>`
- Consumer can verify using shared secret (per ADR-02-003)
- localhost/127.0.0.1 exempt from signing

---

## Integration Test: Webhook enqueue latency under 1 second

### Setup
- Webhook consumer registered
- Event committed to audit log

### Steps
1. Commit event
2. Measure time until webhook job enqueued

### Assertions
- Enqueue latency < 1 second (NFR-02-009)
- `orch_webhook_enqueued_total` metric incremented

---

*End of webhook integration tests*