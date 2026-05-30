# ADR-02-004: `/ready` Degradation Semantics

**ADR:** ADR-02-004  
**Subject:** Readiness endpoint failure behavior when webhook infrastructure is unavailable  
**Profile:** architect  
**Date:** 2026-05-28  
**Status:** Accepted

---

## Problem Statement

FR-02-021 describes the `/ready` endpoint:
- Returns `200` when "orchestration storage is reachable, current-state reads are available, audit event persistence is available, webhook enqueueing is available, and SSE fanout is initialized"
- Returns non-200 when "current-state persistence or audit event persistence is unavailable"
- Says webhook receiver outages "MUST NOT make the platform unready" but is silent on webhook queue unavailability

"Anb" says readiness "SHOULD fail or degrade" when webhook queue is unavailable. "Should" is ambiguous. Different implementers will make different calls, leading to inconsistent behavior.

---

## Options Considered

| Option | Behavior | Pros | Cons |
|--------|----------|------|------|
| A | Hard fail: 503 when webhook queue unavailable | Clear infrastructure alert; ops action required | All reads continue but platform marked unhealthy |
| B | Soft degrade: 200 with `status: "degraded"` | Softer signal; reads continue | Warning signal might be missed; ambiguous for consumers |
| C | Webhook queue not part of readiness | Simplest to implement | Consumer can't distinguish degraded from healthy |

---

## Decision

**Option A** with a two-tier model: return `200` only when ALL four subsystems are available. Return `503` with JSON body identifying the specific failing subsystem when any subsystem is unavailable.

**Specific rules:**
1. `GET /ready` returns `200` only when ALL of the following are true:
   - Orchestration storage reachable (DB ping succeeds)
   - Current-state reads available (`SELECT 1` on tasks table succeeds)
   - Audit event persistence available (append operation succeeds)
   - Webhook enqueueing available (queue can accept a test job within 1 second)
2. When any of the above checks fails, `GET /ready` returns `503 Service Unavailable` with JSON body:
   ```json
   {
     "status": "unavailable",
     "failingSubsystem": "<storage|currentState|auditPersistence|webhookQueue>",
     "message": "<human-readable reason>",
     "timestamp": "<ISO 8601>"
   }
   ```
3. Webhook RECEIVER outages do NOT cause a 503. Only webhook QUEUE unavailability causes a 503. This distinction is because the queue (not the delivery) is part of the request commit path; receiver outages are isolated per NFR-02-010.
4. SSE fanout initialization failure: If SSE fanout is not initialized at startup, `/live` returns 200 (process is alive) but `/ready` returns 503 until fanout is ready. This prevents premature load balancer registration.
5. The specific failing subsystem allows ops dashboards to route the alert to the right team.

---

## Consequences

**Positive:**
- Behavior is explicit and testable: every subsystem has a clear pass/fail
- Two-tier model (storage vs. webhook) correctly distinguishes infra tiers
- Ops teams get actionable 503s that immediately identify the failing subsystem

**Negative:**
- More complex `/ready` implementation than a simple "check DB" probe
- Full `/ready` (webhook queue included) may legitimately fail even when core reads work

**Neutral:**
- `/live` remains simple (process alive → 200 regardless of infra state)
