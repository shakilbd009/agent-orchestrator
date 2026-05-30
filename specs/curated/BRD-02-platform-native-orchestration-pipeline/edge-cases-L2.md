# BRD-02: Edge Cases L2

**Profile:** architect  
**Date:** 2026-05-28  
**BRD:** specs/orchestration/brd-02-platform-native-orchestration-pipeline.md

---

## Overview

Edge case discovery for BRD-02 across four categories: Data Boundaries, State Transitions, Timing, and Integration.

---

## Category 1: Data Boundaries

| ID | Edge Case | Trigger | Handling | Test Approach |
|----|-----------|----------|----------|----------------------------|
| DB-01 | Empty task body | `body: null` at task creation | Allowed — `title` is required, `body` is optional | POST /tasks with null body → 201 |
| DB-02 | Max title length | `title > DB column limit` | Spec-defined limit (recommend 512); reject at 422 if exceeded | POST with 513-char title → 422 |
| DB-03 | Max children hard cap (50) | Decomposition creates child 51 | Reject with 422; include `{limit: 50, current: 50}` in error | Counter test with 51 children |
| DB-04 | Max depth hard cap (5) | Decomposition chain depth = 5 → more | Reject with 422; include current depth and limit | Counter test: depth 6 attempt |
| DB-05 | Zero children proposed | `children: []` in decomposition request | Reject with 422; minimum one child per proposal | POST empty children → 422 |
| DB-06 | Null actor ID | Mutation has no `actorId` in auth context | Fail closed — reject with 401 | No actorId header → 401 |
| DB-07 | Duplicate role claim | Actor claims `layer_a + layer_b` simultaneously | Use most-privileged role for this action | Request with dual claim → use Layer A |
| DB-08 | Duplicate actorId | Same `actorId` in different role claims | First matching role wins; consistent per-request | No double-claim allowed |
| DB-09 | Schema version missing | Event received with no `schemaVersion` field | Accept (backward compat); log warning for v1 events | Test: emit event with no schemaVersion |
| DB-10 | Very large event payload | `payload` JSONB very large (>1MB) | Limit event payload size; reject mutation if event would exceed; include in observability metrics | Large payload → 413 |
| DB-11 | Max project name | Project name at DB column limit | Recommend 256 chars; reject at 422 | POST with 257-char name → 422 |
| DB-12 | Webhook registration URL too long | `consumer_url` field exceeds 2048 chars | Reject at registration time with 422 | Long URL → 422 |
| DB-13 | Non-allowed webhook URL scheme | `consumer_url: "ftp://..."` or `"file://..."` | Reject with 400; only `http`/`https` allowed | ftp URL → 400 |

---

## Category 2: State Transitions

| ID | Edge Case | Trigger | Handling | Test Approach |
|----|-----------|----------|----------|----------------------------|
| ST-01 | Decomposition + approval race | Two concurrent calls: (a) approve proposal, (b) resubmit same decomposition | (a) succeeds, (b) sees active proposal → 409 Conflict | Concurrent approve + resubmit → first wins, second gets 409 |
| ST-02 | Parent done + required child still in_progress | Parent called with child in active required child still in_progress | Reject 409; reject reason lists blocking child | Complete parent while child in_progress → 409 |
| ST-03 | Handoff submitted twice | Layer B calls complete endpoint twice with same evidence | First call commits; second call returns current task state (idempotent) | Double submit → first 200, second 200 current state |
| ST-04 | Gate approved + task cancelled concurrently | Gate approval + task cancellation intersect | Both events recorded in audit log sequentially; task completion (done) blocked by gate if gate was approved before cancellation → task returns to pre-completion state | Concurrent approve + cancel → sequence captured in audit |
| ST-05 | Required child deleted | Attempt to DELETE a required child task | Platform MUST reject with 409; deletion requires removing required-for-parent relationship first | Delete required child → 409 |
| ST-06 | Re-assigned Layer B completes old task | Task reassigned to Layer B-2; Layer B-1 calls complete | Fail closed — 403 Forbidden | Re-assigned task; original assignee calls complete → 403 |
| ST-07 | Active proposal superseded | New proposal before old is approved (re-submit) | Idempotency check: if active proposal exists, reject with 409 | Re-submit before approve → 409 |
| ST-08 | Layer B attempts gate approval | Layer B calls POST /gates/{id}/approve | 403 Forbidden; `auth.mutation.denied` event | Layer B approves own gate → 403 + audit |
| ST-09 | Layer B approves project phase gate | Layer B calls phase advancement | 403 Forbidden | Layer B phase advance → 403 + audit |
| ST-10 | Unauthorized cross-project task access | Actor in project A attempts to mutate task in project B | 404 Not Found (fail closed — project not found vs. unauthorized) | Cross-project mutation → 404 |
| ST-11 | Decomposition depth override accepted | Human override sets depth to 4 when project default is 3 | Allowed; audit log records override identity, timestamp, old, new, reason | Override within hard caps → 201 |
| ST-12 | Decomposition depth override beyond hard cap | Human override sets depth to 6 (exceeds hard cap of 5) | Reject with 422 | Override beyond hard cap → 422 |
| ST-13 | Decomposition fan-out override beyond hard cap | Human override sets fan-out to 51 (exceeds hard cap of 50) | Reject with 422 | Fan-out override 51 → 422 |
| ST-14 | Task-level gate on task with no children | FR-02-017A: gate can exist without decomposition | Gate enforced independently; no dependency on children | Open gate on leaf task → 201; attempt complete → blocked by gate |
| ST-15 | Proposal rejected then approved | Same proposal re-submitted after rejection | New proposal created; rejection reason cleared from old proposal | Re-submit rejected → new proposal active |
| ST-16 | Open gate on already-done task | Gate opens after task is already DONE | Allowed; gate appears in task gate list but task is already complete | Done task gate approval → gate state updates but task stays done |

---

## Category 3: Timing

| ID | Edge Case | Trigger | Handling | Test Approach |
|----|-----------|----------|----------|----------------------------|
| T-01 | SSE client connects but never reads | Browser opens connection, tab goes idle | Ping/keepalive every 30s via comment-event; disconnect after 2 missed pings; goroutine cleaned | Keepalive test: stop reading stream → disconnect after ~60s |
| T-02 | Rapid reconnect before previous stream closes | Client reconnects with `Last-Event-ID` before old connection EOF | Each connection gets fresh channel; idempotent delivery via eventId deduplication | Rapid reconnect → distinct channels; no duplicate events |
| T-03 | Webhook retry exhaustion | 3 delivery attempts all fail | Exhausted delivery logged; metric incremented; no rollback | Mock receiver 500s → exhausted after 3 retries |
| T-04 | Stale detection at exact threshold | Task inactive exactly N seconds (N = stale threshold) | Emit stale detection event once; not double-emitted on subsequent checks | Advance clock to threshold → event emitted |
| T-05 | `/ready` returns before webhook queue initialized | Service boot: DB up, queue still initializing | `/ready` returns 503 until queue can accept a test job; prevents premature LB registration | Start service → `/ready` 503 until queue ready |
| T-06 | Layer B completes during open gate review | POST /complete arrives while gate is still `open` | Reject 409; blocking gate must be approved first | Layer B complete + open gate → 409 |
| T-07 | Layer B completes during project phase gate | Layer B calls complete while phase gate is open | Task completion is blocked by project-level gate (as well as task-level gates); check both | Complete + open phase gate → 409 |
| T-08 | Long-running decomposition proposal | Proposal pending for days without review | No expiry defined; async stale logic could surface metrics; not auto-rejected | No expiry test: stale proposal remains |
| T-09 | Reconnect with very old `Last-Event-ID` | Client disconnected for extended period; reconnects with very old Last-Event-ID | Server returns all events since lastSeenId; if many events, stream starts with historical batch then live | Many events accumulated → catch-up batch first |
| T-10 | SSE connection closed during event fanout | Client disconnects mid-fanout for event 42 | Fanout goroutine detects closed channel; skips; emits disconnect log | Disconnect mid-stream → goroutine exits cleanly |
| T-11 | Webhook delivery timeout | Consumer returns 200 after 50s (longer than recommended 30s) | Treated as failure on timeout; recommended consumer timeout is 30s; log warning | Long timeout consumer → retry |
| T-12 | Mutually-exclusive role claim tokens | Two requests with same session but conflicting role claims | Most-privileged role wins for each action; token is for identity not role-switching | No action here; this is identity design |

---

## Category 4: Integration

| ID | Edge Case | Trigger | Handling | Test Approach |
|----|-----------|----------|----------|----------------------------|
| INT-01 | Webhook receiver non-2xx | Consumer returns 500 | Exponential backoff retry (1s, 2s, 4s); exhausted after 3 | Mock 500 → retry sequence |
| INT-02 | Webhook receiver timeout | Consumer slow >30s | Treat as timeout; retry with backoff | Mock slow 500 → timeout → retry |
| INT-03 | Webhook 301 redirect | Consumer returns 301 to new URL | Follow redirect (max 1 hop); retry at final URL | 301 redirect → follow and deliver |
| INT-04 | Webhook redirect loop | Consumer redirects in a loop (>1 hop) | Fail with permanent error; log redirect chain | 2-hop redirect → 400 + error |
| INT-05 | DB unreachable mid-transaction | DB connection drops during COMMIT | Full transaction rollback; no event emitted; client receives 503 | Kill DB mid-transaction → 503 + rollback |
| INT-06 | 51st SSE connection attempted | Client 51 attempts connection | Reject with 503 Too Many Connections; `orch_sse_clients_current` at max | 51st connection → 503 |
| INT-07 | BRD-08 automated gate evaluator calls BRD-02 API | BRD-08 agent calls POST /gates/{id}/approve | If authenticated as `layer_a` role → approved; BRD-02 doesn't distinguish automated from human | Layer A agent approves gate → 200 |
| INT-08 | BRD-03 Dashboard lost connection during critical update | Dashboard SSE stream disconnects right before a gate opens | Client reconnect with Last-Event-ID gets missed gate event (catch-up); gate state is current-state authoritative, so no permanent data loss | Disconnect → reconnect → gate.* event delivered |
| INT-09 | Feature flag toggled mid-request | `platform-orchestration` toggled off during active request | Request completes under original flag state (evaluated at request start); flag change is atomic at request boundary | Toggle flag mid-streaming-request → existing request completes |
| INT-10 | OpenAPI contract drift | API spec says one behavior but implementation does another | OpenAPI spec is contract; CI should validate spec coverage; drift caught in review not runtime | Spec vs. impl mismatch → CI failure |
| INT-11 | Contract/events.md schema vs. actual event | Phase 0 placeholder events differ from BRD-02 canonical envelope | BRD-02 curation MUST update contracts/events.md to reflect FR-02-022A envelope; BRD-02 refiner noted this as required dependency | events.md vs. actual envelope → curated events.md |

---

## Edge Case Handling Summary

| Category | Coverage |
|---|---|
| Data Boundaries | 13 edge cases across null/empty/max/scheme validation |
| State Transitions | 16 edge cases covering decomposition, completion, authorization, and gate concurrency |
| Timing | 12 edge cases covering SSE lifecycle, webhook retry, stale detection, and clock sensitivity |
| Integration | 11 edge cases covering webhook consumers, DB failures, concurrent SSE, and cross-BRD integration |

**Total: 52 edge cases catalogued**

All should-have and must-have behaviors have corresponding test approaches defined. The absence of "figure it out later" items is confirmed — every listed edge case has a defined handling approach and test method.
