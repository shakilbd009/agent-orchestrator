# Security Baseline — Phase 0

**Phase:** 0 (Governance Scaffolding)
**Status:** Stub — implementation deferred to Phase 1
**Audience:** Operators, security reviewers, future implementers

---

## 1. Secrets Management

| Concern | Phase 0 stub | Phase 1 implementation owner |
|---|---|---|
| No secrets in source | Enforced by .gitignore and code review | Automated scanner in CI |
| Secrets injection | `.env.example` lists all required keys | Vault/KMS integration |
| Rotation | Not applicable | Automated rotation for service accounts |
| Error on missing required env | Not applicable | Server exits fast on missing `REQUIRED_*` env vars |

**Principle:** Secrets are never committed, never logged, never echoed back.

---

## 2. Environment Validation

On process startup, the platform validates:

- All `REQUIRED_*` environment variables are present and non-empty
- Known optional variables are structurally valid (URI format, integer range, etc.)
- No `DEBUG=true` in production-targeted deployments

**Implementation owner:** Backend bootstrap in Phase 1.

---

## 3. Dependency Scanning

| Artifact | Tool | Gate |
|---|---|---|
| Go modules | `govulncheck` | CI must pass before merge |
| npm/pnpm packages | `npm audit` / `pnpm audit` | CI must pass before merge |
| Docker base images | Trivy or Snyk | Daily scan, alert on critical CVEs |

**Principle:** Known critical/high CVEs in direct dependencies block merge.

---

## 4. CORS and Security Headers

Phase 0 stub — define expectations only.

| Header | Expected value |
|---|---|
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` |
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` or `SAMEORIGIN` |
| `Content-Security-Policy` | Restrictive default; inline scripts require explicit allowlist |
| `Access-Control-Allow-Origin` | Explicit origin list only; no wildcard in production |

**Implementation:** Middleware in Phase 1 (Go Echo framework).

---

## 5. Input Validation

| Layer | Scope |
|---|---|
| API gateway | Schema validation on all inbound requests (reject unknown fields) |
| Agent prompts | Structured prompt injection guards; no raw user strings concatenated into system prompts |
| File uploads | Type allowlist, size limits, filename sanitization |
| Database | Parameterized queries only; ORM defaults |

**Implementation owner:** Backend team in Phase 1.

---

## 6. Authentication and Session Boundaries

| Concern | Phase 0 stub |
|---|---|
| Auth model | Session-based; per-user, per-team, per-project isolation |
| Session lifecycle | Expiry, refresh, revocation defined in BRD-01 |
| Multi-tenancy isolation | Tenants must not see each other's data (BRD-17) |
| Agent identity | Each agent profile has a role; agents act within scope of assigned task |
| Human approval gates | No automated deployment or release without human approval |

**Principle:** Agents operate with least privilege — task-scoped tokens, not persistent admin credentials.

---

## 7. Rate Limiting

| Endpoint class | Limit |
|---|---|
| Unauthenticated API | 60 req/min per IP |
| Authenticated API | 300 req/min per user |
| Agent spawning | 10 concurrent agents per project |
| File upload | 20 req/min per user |

**Principle:** Rate limits are per-project configurable via feature flags. Default limits above; human can raise per-project by explicit approval.

---

## 8. Threat Model Stub

**Owner:** Security Reviewer agent (BRD-10)

| Threat category | Phase 0 status | Phase 1+ action |
|---|---|---|
| Prompt injection | Acknowledged | Prompt injection resistance testing |
| Unauthorized agent action | Mitigated by task-scoping and kanban gating | Audit every agent action |
| Data exfiltration | Mitigated by tenant isolation | Penetration test in Phase 2 |
| Secrets leakage via logs | Mitigated by secrets never in env or logs | Log redaction library in Phase 1 |
| Denial of service | Mitigated by rate limits | Load testing in Phase 2 |

**This threat model is a living document. A full session-based threat model is created in Phase 1 once the app shell exists.**

---

## 9. Privacy

- User ideas, requirements, and project data are private to the owning tenant
- No data used for model training without explicit user consent
- Data residency: all data stored in tenant-designated region
- Right to deletion: full tenant data purge capability documented

---

## 10. Least Privilege

| Actor | Default permission |
|---|---|
| End user | Read/write own project data only |
| Project collaborator | Read/write scoped to assigned project |
| Agent (per profile) | Task-scoped; cannot escalate or persist beyond assigned task |
| Operator/admin | Infrastructure access; no ad-hoc data access |

**Principle:** No standing admin credentials for application data access.

---

## 11. Auditability

Every meaningful action is recorded (see BRD section 7):

| Event type | Captured fields |
|---|---|
| Task created | id, title, assignee, parent, timestamp, created_by |
| Task promoted | id, from_status, to_status, timestamp |
| Task completed | id, summary, metadata, timestamp |
| Agent handoff | from_agent, to_agent, task_id, timestamp |
| Human approval | approver, task_id, decision, timestamp |
| Deployment triggered | environment, triggered_by, timestamp |
| Error/failure | task_id, error, timestamp |

**Implementation:** Audit log schema defined in Phase 1. Retention: 1 year rolling.

---

## 12. Retention

| Data class | Retention period | Action on expiry |
|---|---|---|
| Audit logs | 1 year rolling | Archive to cold storage |
| Project data | Until project deleted | Soft delete then hard delete after 30 days |
| Agent session logs | 90 days rolling | Deletion |
| Secrets | Rotated per rotation policy | Revoked immediately on rotation |

---

## Open Questions

1. Which secrets management solution (Vault, AWS KMS, GCP Secret Manager)?
2. Is there a compliance standard we must certify against (SOC 2, ISO 27001)?
3. Do we need a data processing agreement (DPA) for EU users?
4. What is the threat model review cadence post-Phase 0?
