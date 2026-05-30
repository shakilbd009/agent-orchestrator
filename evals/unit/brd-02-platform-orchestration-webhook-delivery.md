# evals/unit/brd-02-platform-orchestration-webhook-delivery.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** Unit test contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## Contract: WebhookService.RegisterConsumer

### Input
- Project ID
- Consumer URL
- Topic prefix (e.g., `task.*`, `gate.`, `task.decomposition`)

### Output
- Webhook registration created with status `active`
- Consumer receives test event on registration (optional)

### Edge cases:
- Duplicate registration same URL + topic: idempotent or 409 Conflict
- Invalid URL format: rejected as 422
- localhost/127.0.0.1 URL: exempt from signing requirement (per ADR-02-003)

---

## Contract: WebhookService.EnqueueDelivery

### Input
- Event envelope (canonical format per FR-02-022A)
- List of matching webhook consumers

### Output
- Delivery job enqueued for each consumer
- Job includes: consumer URL, event envelope, attempt count=0, next_attempt_at=now
- `orch_webhook_enqueued_total` metric incremented

### Edge cases:
- Consumer inactive (de-registered): skip, no job created
- Topic does not match any registration: no jobs enqueued
- Enqueue latency > 1 second: metric captures duration; NFR-02-009 violation flagged

---

## Contract: WebhookDeliveryWorker.AttemptDelivery

### Input
- Webhook job (consumer URL, event envelope, attempt count)

### Output
- HTTP POST to consumer URL
- Header `X-Webhook-Signature: HMAC-SHA256=<signature>` included
- Request timeout: 30 seconds recommended

### Edge cases:
- Consumer returns 2xx: delivery succeeded, job removed from queue
- Consumer returns 3xx redirect: follow (max 1 hop), retry at final URL
- Consumer returns 4xx: permanent failure, no retry
- Consumer returns 5xx: temporary failure, retry with backoff
- Consumer timeout: treated as temporary failure

---

## Contract: WebhookDeliveryWorker.ExponentialBackoff

### Input
- Failed delivery attempt (attempt_count=N)

### Output
- Next attempt scheduled:
  - Attempt 1 fail → retry after 1 second
  - Attempt 2 fail → retry after 2 seconds
  - Attempt 3 fail → retry after 4 seconds
  - After attempt 3 fail: exhaust; log failure; metric incremented

### Edge cases:
- Configurable retry count: default 3, can be configured per consumer
- Backoff formula: 2^attempt * base_interval (base=1s per AC-02-024)
- Retry budget exhausted: `orch_webhook_delivery_failed_total` incremented, `webhook.delivery.failed` audit event appended

---

## Contract: WebhookService.OnExhaustedDelivery

### Input
- Webhook job after all retries exhausted

### Output
- `orch_webhook_delivery_failed_total` incremented
- Audit event `webhook.delivery.failed` appended with: consumer URL, event topic, attempt history, failure reason
- Original task/gate state NOT rolled back (per NFR-02-010)

### Edge cases:
- Webhook consumer comes back online after exhaustion: no auto-replay (deferred to future BRD)
- Exhausted delivery visible in project audit views: yes (per AC-02-024)

---

## Contract: Webhook signing — HMAC-SHA256

### Input
- Webhook registration with shared secret
- Event to deliver

### Output
- `X-Webhook-Signature` header computed as HMAC-SHA256(secret, event_payload_json)
- Consumer can verify using stored secret

### Edge cases:
- Secret not configured: use empty string or skip header (per ADR-02-003, internal URLs exempt)
- Secret rotation: new deliveries use new secret; old secret until rotation completes
- SHA variant: HMAC-SHA256 required (per ADR-02-003)

---

*End of webhook delivery unit contracts*