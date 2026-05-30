# evals/unit/brd-02-platform-orchestration-event-envelope.md

**Project:** agent-orchestrator  
**BRD:** BRD-02 — Platform-Native Orchestration Pipeline  
**Type:** Unit test contracts  
**Owner:** qa  
**Status:** 🔴 **Failing** (before implementation)

---

## Contract: CanonicalEventEnvelope — all required fields present

### Input
Any event emission (task created, gate approved, decomposition proposed, etc.)

### Output
- Event envelope contains all required fields per FR-02-022A:
  - `eventId` (UUID)
  - `schemaVersion` (v1alpha during implementation)
  - `projectId`
  - `topic`
  - `actorId`
  - `actorRole`
  - `taskId` (nullable when not task-specific)
  - `parentTaskId` (nullable)
  - `gateId` (nullable)
  - `timestamp` (ISO 8601)
  - `payload` (event-specific JSON object)

### Edge cases:
- Event not task-specific: taskId, parentTaskId, gateId are null or omitted
- Schema version: must be `v1alpha` during Phase 1 (per ADR-02-001)
- `schemaVersion` field missing: accept for v1alpha; log warning

---

## Contract: AuditEventLog.Append — immutability

### Input
Audit event to append

### Output
- Event appended to immutable log
- Event record is NOT updated, deleted, or reordered through normal application flows (per NFR-02-013)
- Event ID is monotonically increasing or UUID

### Edge cases:
- Duplicate eventId: reject or handle as idempotent
- Event with future timestamp: accepted with warning
- Event with null payload: allowed (use empty object {})

---

## Contract: Event envelope serialization

### Input
Canonical event envelope object

### Output
- JSON serialization contains all required fields
- Deserialization produces identical envelope
- SSE `data` field contains full JSON envelope string

### Edge cases:
- Payload with special characters: properly escaped in JSON
- Very large payload: accepted (no size limit specified in BRD-02)
- Non-UTF8 payload: rejected as 422

---

## Contract: SSE event format

### Input
Event from fanout to SSE stream

### Output
- SSE `event` field = event `topic` (e.g., `task.status.changed`)
- SSE `id` field = `eventId` (UUID string)
- SSE `data` field = full canonical envelope JSON string

### Edge cases:
- Topic contains special characters: SSE event field allows any string
- Very long event ID: SSE id field truncated to reasonable length (implementation choice)
- Null topic: not expected; reject at encoding time

---

## Contract: Schema version injection at build time

### Input
Build with `-X version.SchemaVersion=v1alpha` ldflags

### Output
- All emitted events carry `schemaVersion: "v1alpha"` in envelope
- Version appears in audit log entries
- `/ready` or health endpoint can report current schema version

### Edge cases:
- Version not injected: default to `v1alpha` or fail fast with clear error
- Version override at runtime: not supported; version is build-time only (per ADR-02-001)

---

*End of event envelope unit contracts*