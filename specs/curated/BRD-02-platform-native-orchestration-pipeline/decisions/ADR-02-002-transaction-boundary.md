# ADR-02-002: Transaction Boundary for Current-State and Audit Append

**ADR:** ADR-02-002  
**Subject:** Transaction boundary requirement for mutation + audit event append  
**Profile:** architect  
**Date:** 2026-05-28  
**Status:** Accepted

---

## Problem Statement

FR-02-002 says the platform "MUST commit current-state records and audit event append in one transactional boundary where possible." The "where possible" qualifier is ambiguous: it permits implementations to commit state and audit separately without defining the semantics of that deviation.

The risk (BRD-02 Risk R-03) is that current-state records and immutable audit events can drift — the database shows task X is DONE but the audit log has no corresponding `task.status.changed` event. This undermines the audit trail's reliability and makes diagnostic event replay useless.

---

## Options Considered

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| A | Always one DB transaction (required) | Strongest consistency; no drift possible | Requires DB engine with transaction support; all mutations must be in one DB |
| B | Best-effort with async consistency check | Simpler if DB lacks transaction support | Drift occurs and is detected after the fact |
| C | Separate commits allowed; diagnostic replay only | Simplest logically | Audit gap possible on crash; inconsistent historical record |

---

## Decision

**Option A** is selected as the default: task/gate/handoff mutations MUST commit current-state records and append the corresponding audit event in ONE database transaction.

**Specific rules:**
1. The mutating API handler (e.g., `POST /tasks/{id}/complete`) begins a DB transaction, performs all state changes, appends all audit events, and commits in one atomic operation.
2. If the DB engine in use does not support row-level transactions (e.g., specific SQLite configurations), both writes MUST occur in the same atomic batch via the engine's available atomicity primitive.
3. A consistency check runs as a background health probe available at `GET /ready` (the diagnostic query): `SELECT COUNT(*) FROM task_current_state ts LEFT JOIN audit_events ae ON ae.task_id = ts.id AND ae.topic = 'task.status.changed' WHERE ts.status = 'done' AND ae.id IS NULL`. If non-zero drift is detected, log a warning and surface in `/ready` response.
4. Deviation from the atomicity requirement requires an explicit ADR filed before implementation of that deviation begins.

**Validation plan:** Write a unit test that injects a failure after state commit but before audit append, confirming the entire transaction rolls back and the error is returned to the client.

---

## Consequences

**Positive:**
- Audit log is always consistent with current-state records
- No drift detection alerts needed in normal operation
- Diagnostic consistency query runs only as a background health check, not on every mutation

**Negative:**
- DB engine must support transactions; limits storage engine choices
- Rollback of state mutation always rolls back the audit event (which is correct)

**Neutral:**
- Webhook enqueue and SSE fanout are triggered AFTER the transaction commits, so webhook/SSE failures do not cause rollback (these are async non-transactional side effects — correct per NFR-02-010)
