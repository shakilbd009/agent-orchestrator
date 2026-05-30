# evals/unit/brd-02-platform-orchestration-health-readiness.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** Unit test contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## Contract: HealthService.CheckLiveness

### Input
- GET /live request to orchestrator service

### Output
- Response `200 OK` when service process is running and able to respond to HTTP requests

### Edge cases:
- Service process down: connection refused or 503
- Goroutine leak causing hang: timeout triggers failure

---

## Contract: HealthService.CheckReadiness — all subsystems up

### Input
- GET /ready request
- All subsystems available: orchestration storage, current-state reads, audit event persistence, webhook enqueueing, SSE fanout

### Output
- Response `200 OK`
- Body may include subsystem status details

### Edge cases:
- All subsystems up: ready
- Any subsystem down: return 503 with JSON identifying failing subsystem (per ADR-02-004)

---

## Contract: HealthService.CheckReadiness — webhook queue unavailable

### Input
- Orchestration storage reachable
- Current-state reads available
- Audit event persistence available
- Webhook enqueueing unavailable (receiver down)

### Output
- Response `503 Service Unavailable`
- Body: `{"webhookQueue": false, ...}`
- Webhook receiver outage (as opposed to queue unavailability) does NOT make platform unready (per FR-02-021)

---

## Contract: HealthService.CheckReadiness — storage unavailable

### Input
- Orchestration storage unreachable
- Other subsystems may or may not be available

### Output
- Response `503 Service Unavailable`
- Body identifies failing subsystem(s)
- Storage unavailability is minimum threshold for platform readiness

---

## Contract: HealthService.CheckReadiness — audit event persistence unavailable

### Input
- Storage reachable
- Current-state reads available
- Audit event persistence unavailable

### Output
- Response `503 Service Unavailable`
- Audit event persistence is required for readiness (per FR-02-021)

---

## Contract: Readiness degradation — webhook receiver outage (not queue)

### Input
- Webhook consumer URL returns 500 (receiver down)
- Webhook queue (enqueue capability) is still available

### Output
- Response `200 OK` with `status: "degraded"` or similar
- Platform remains usable for reads
- Webhook delivery may be delayed but not blocked
- This is the two-tier readiness model: minimum (storage) vs. full (webhook) (per ADR-02-004)

### Edge cases:
- Queue itself unavailable: hard fail (503)
- Receiver temporarily down: soft degrade

---

## Contract: Readiness probe initialization order

### Input
- Service boots: DB initializes before webhook queue

### Output
- `/ready` returns 503 until all required subsystems are initialized
- Prevents premature load balancer registration (per implementation threat)

### Edge cases:
- DB ready, queue still init: 503
- Queue ready after DB: transition from 503 to 200
- Webhook receiver down post-init: degrade gracefully, not hard fail

---

*End of health/readiness unit contracts*