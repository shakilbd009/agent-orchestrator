# evals/e2e/brd-02-platform-orchestration-webhooks.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** End-to-end scenario contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## E2E Scenario: Webhook receives event without blocking original mutation

### Given
- Project `proj-alpha` has a webhook consumer registered for `task.*` events
- Consumer URL is `https://consumer.example.com/webhook`
- Task `task-x` is assigned to `layer_b:agent1`

### When
Agent1 completes task:
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

### Then
- Response `200 OK` returned immediately (webhook delivery is async)
- Task status transitions to `done`
- Webhook delivery job is enqueued within 1 second (NFR-02-009)
- Original mutation is NOT blocked or rolled back regardless of webhook delivery outcome (per NFR-02-010)

---

## E2E Scenario: Webhook delivers with HMAC-SHA256 signature

### Given
- Project `proj-alpha` has a webhook consumer registered
- Consumer has a shared secret configured

### When
An orchestration event occurs and webhook delivery is attempted

### Then
- Request includes header `X-Webhook-Signature: HMAC-SHA256=<computed_signature>`
- Consumer can verify signature using shared secret (per ADR-02-003)
- `localhost` and `127.0.0.1` URLs are exempt during development

---

## E2E Scenario: Webhook retry with exponential backoff

### Given
- Project `proj-alpha` has a webhook consumer for `task.status.changed`
- Consumer URL returns `500 Internal Server Error`

### When
Webhook delivery fails on first attempt

### Then
- Retry 1: after 1 second
- Retry 2: after 2 seconds
- Retry 3: after 4 seconds
- After 3 exhausted attempts (default retry count per AC-02-024)
- `orch_webhook_delivery_failed_total` metric incremented
- Audit event `webhook.delivery.failed` is appended

---

## E2E Scenario: Webhook receiver outage does not roll back committed state

### Given
- Task `task-x` completion is committed to durable storage
- Webhook consumer is down/returning 500

### When
Webhook retries are exhausted

### Then
- Task status remains `done`
- Original mutation is NOT rolled back
- Failure is logged and visible in project audit/event views
- `GET /projects/proj-alpha/tasks/task-x` shows `done` status

---

## E2E Scenario: Project can register webhook by event topic prefix

### Given
- Project `proj-alpha`
- Actor `human:alice` registers webhook

### When
```
POST /projects/proj-alpha/webhooks
{
  "consumerUrl": "https://consumer.example.com/orch",
  "topicPrefix": "task.decomposition",
  "actorId": "human:alice",
  "actorRole": "human"
}
```

### Then
- Response `201 Created`
- Consumer receives events matching `task.decomposition.proposed`, `task.decomposition.approved`, `task.decomposition.rejected`
- Consumer does NOT receive `task.created` or `gate.approved` events

---

## E2E Scenario: Project webhook receives `task.*` wildcard subscription

### Given
- Project `proj-alpha` has webhook registered for topic `task.*`

### When
A task event occurs: `task.stale.detected`, `task.blocked`, `task.status.changed`, etc.

### Then
- Consumer receives delivery for all `task.*` events
- Subscription is prefix-based: `task.` matches all task events

---

## E2E Scenario: Webhook delivery enqueued within 1 second

### Given
- Project `proj-alpha` has an active webhook consumer
- An audit event is committed to the event log

### When
Time passes from event commit

### Then
- Webhook delivery job is enqueued within 1 second (NFR-02-009)
- Measured via `orch_webhook_enqueued_total` metric / timing histogram

---

## E2E Scenario: Webhook 301 redirect followed

### Given
- Webhook consumer URL returns `301 Redirect` to a new location

### When
Delivery is attempted

### Then
- Follow redirect (max 1 hop)
- Retry at final URL
- If redirect chain > 1 hop, fail with permanent error

---

## E2E Scenario: Webhook timeout treated as failure

### Given
- Webhook consumer is slow (>30s response time)

### When
Delivery is attempted

### Then
- Treated as delivery failure
- Retry with backoff initiated
- After exhaustion: `webhook.delivery.failed` event logged

---

## E2E Scenario: Webhook consumer non-2xx response triggers retry

### Given
- Webhook consumer returns `503 Service Unavailable`

### When
Delivery is attempted

### Then
- Retry with exponential backoff
- Metric `orch_webhook_retry_total` incremented on each retry attempt

---

*End of webhook E2E scenarios*