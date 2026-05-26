# Observability Baseline — Phase 0

**Phase:** 0 (Governance Scaffolding)
**Status:** Stub — implementation deferred to Phase 1
**Audience:** Operators, DevOps, future implementers

---

## 1. Structured Logging

| Concern | Phase 0 stub | Phase 1 implementation owner |
|---|---|---|
| Format | JSON lines | Structured logger (e.g., zerolog, zap) |
| Required fields | `ts`, `level`, `msg`, `request_id` | Consistent field names across all services |
| Sensitive data | Never logged | Redaction library in Phase 1 |
| Log levels | DEBUG / INFO / WARN / ERROR / FATAL | Actionable levels; no DEBUG in production |
| Sampling | Not applicable | Optional high-volume endpoint sampling with explicit config |

**Principle:** Logs are for humans to read in dev and machines to parse in prod. Every log entry describes one discrete action or event.

---

## 2. Request and Correlation IDs

| Concern | Stub |
|---|---|
| Request ID generation | `X-Request-ID` header; generated at gateway |
| Propagation | Request ID passed to all downstream services and agent subprocesses |
| Correlation across agents | Each agent task run tagged with parent `task_id` + its own run ID |
| Log association | All log entries during a request carry the same `request_id` |

**Implementation owner:** Backend (Phase 1) + agent framework (Phase 2).

---

## 3. Health, Readiness, and Liveness Probes

| Probe | Path | Purpose | Failure behavior |
|---|---|---|---|
| Liveness | `GET /health/live` | Is the process alive? | Restart container |
| Readiness | `GET /health/ready` | Can it accept traffic? | Remove from load balancer |
| Database connectivity | Inside readiness | Is DB reachable? | Fail readiness |
| Redis connectivity | Inside readiness | Is cache reachable? | Fail readiness |

**Startup sequence:**
1. Process starts
2. Validate env vars
3. Connect to database
4. Connect to Redis
5. Mark `/health/ready` as OK
6. Begin accepting traffic

**Implementation owner:** Backend in Phase 1.

---

## 4. Frontend Error Boundary Expectations

| Concern | Phase 0 stub |
|---|---|
| Error boundary | Component-level error boundaries wrapping each major section |
| Uncaught exception | Global handler + toast notification + structured error report |
| API error surface | Errors shown in-context with actionable message; never raw stack traces |
| Error report payload | Timestamp, user_id (if authenticated), request_id, component, message |
| Error reporting service | Placeholder integration (e.g., Sentry) in Phase 1 |

**Principle:** Users never see technical error messages. Operators get structured error reports.

---

## 5. Metrics Placeholders

| Metric | Type | Phase 1 owner |
|---|---|---|
| `http_requests_total` | Counter | Backend |
| `http_request_duration_seconds` | Histogram | Backend |
| `agent_tasks_active` | Gauge | Agent framework |
| `agent_tasks_completed_total` | Counter (labeled by outcome) | Agent framework |
| `db_connections_active` | Gauge | Backend |
| `redis_connections_active` | Gauge | Backend |
| `queue_depth` | Gauge | Backend |

**Metrics endpoint:** `GET /metrics` (Prometheus scrape endpoint, behind auth).

**Dashboards (Phase 2):** System health, agent throughput, error rate, pipeline latency.

---

## 6. Local Debugging Workflow

| Scenario | Expected workflow |
|---|---|
| View structured logs | `docker-compose logs -f backend` with JSON parsing via `jq` |
| Trace a request | Filter by `X-Request-ID` |
| Inspect database | `docker-compose exec postgres psql -U agent-orchestrator` |
| Inspect Redis | `docker-compose exec redis redis-cli` |
| Simulate agent run | `make agent-debug` (Phase 1 target) |
| Kill/restart a service | `docker-compose restart <service>` |
| Full reset | `docker-compose down -v && docker-compose up -d` |

**Principle:** Local environment mirrors production as closely as possible. No proprietary debugging tools required.

---

## 7. Audit Trail Visibility

From BRD section 7, every meaningful action is recorded. This section defines the visibility layer.

| Actor | What they can see |
|---|---|
| End user | Own project events: task history, approvals given, decisions made |
| Project collaborator | Project audit trail (all project members) |
| Operator/admin | Subset of system-level events (not raw agent reasoning) |
| External auditor | Read-only audit log export via admin panel |

**Audit log API (Phase 1):**
- `GET /audit?project_id=X&from=ts&to=ts` — paginated, filtered
- `GET /audit/export?project_id=X&format=jsonl` — export for compliance

**Principle:** Audit trail is append-only. No delete capability on audit records.

---

## 8. Alerting Expectations

| Alert | Condition | Owner |
|---|---|---|
| High error rate | >1% of requests returning 5xx over 5 min | On-call |
| Service down | Liveness probe failing > 30s | On-call |
| Queue backlog | > 100 queued tasks for > 10 min | Project lead |
| Secrets scan failure | Critical CVE found in dependency scan | Security |
| Disk usage | Any volume > 80% | On-call |

**Alert routing:** PagerDuty or equivalent in Phase 1. On-call rotation defined in Phase 1 runbook.

---

## Open Questions

1. What is the metrics backend? Prometheus + Grafana self-hosted, or a cloud offering (Datadog, CloudWatch)?
2. Is there a log aggregation backend for multi-service production (ELK, Loki, etc.)?
3. What is the alert routing and on-call rotation?
4. Should agent reasoning traces be visible in the audit trail or only final outputs?
5. What is the sampling strategy for high-volume endpoints in production?
