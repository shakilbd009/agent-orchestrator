# evals/e2e/ — End-to-End Test Placeholders

**Project:** agent-orchestrator
**Phase:** 0

---

## Purpose

This directory contains end-to-end (E2E) / browser-based test scenarios.
E2E tests verify that the full system works from the user's perspective,
covering the complete request/response cycle across backend, frontend, and
agent orchestration.

---

## Phase 0 Status

Phase 0 has no source code and no runnable application. Test scenarios in this
directory are **placeholders** — they describe the scenario in plain language
and are marked with `[PLACEHOLDER — FAILING BEFORE IMPLEMENTATION]`.

Once Phase 1 creates the app shell and Phase 2 implements the orchestration
pipeline, these placeholders are replaced with actual Playwright test files.

---

## Running E2E Tests (Future)

```bash
# Install dependencies first
pnpm install

# Run E2E tests
pnpm exec playwright test

# Run with UI
pnpm exec playwright test --ui
```

---

## Placeholder Files

| File | Scenario | Phase |
|------|----------|-------|
| `placeholder.md` | (this file) | all |
