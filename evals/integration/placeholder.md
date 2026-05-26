# evals/integration/ — Integration Test Placeholders

**Project:** agent-orchestrator
**Phase:** 0

---

## Purpose

This directory contains integration test files that verify multiple components
work correctly together. Integration tests span module boundaries — for example,
testing that the Kanban API correctly creates a task and that the event emitter
fires the expected event.

---

## Phase 0 Status

Phase 0 has no source code. Integration test files in this directory are
**placeholders** marked with `[PLACEHOLDER — FAILING BEFORE IMPLEMENTATION]`.

---

## Running Integration Tests (Future)

```bash
# Backend integration tests (Go)
cd backend && go test ./...

# With coverage
go test -cover ./...
```

---

## Placeholder Files

| File | Scenario | Phase |
|------|----------|-------|
| `placeholder.md` | (this file) | all |
