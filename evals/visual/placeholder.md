# evals/visual/ — Visual / Regression Test Placeholders

**Project:** agent-orchestrator
**Phase:** 0

---

## Purpose

This directory contains visual regression test scenarios that verify the
frontend renders correctly and that intentional UI changes are detected
before shipping. Visual tests capture screenshots of pages and compare them
against baseline images.

---

## Phase 0 Status

Phase 0 has no frontend. Visual test files in this directory are
**placeholders** marked with `[PLACEHOLDER — FAILING BEFORE IMPLEMENTATION]`.

---

## Running Visual Tests (Future)

```bash
# Uses Playwright with @playwright/test and a visual comparison plugin
pnpm exec playwright test --project=visual

# Update baselines
UPDATE_VISUAL_BASELINES=true pnpm exec playwright test --project=visual
```

---

## Visual Test Scope

| Page / Component | Baseline Needed | Priority |
|-----------------|----------------|----------|
| Dashboard (Phase 2) | Yes | P1 |
| Kanban Board (Phase 2) | Yes | P1 |
| Agent Workstream View (Phase 2) | Yes | P2 |
| Settings / Config (Phase 2) | Yes | P3 |

---

## Placeholder Files

| File | Scenario | Phase |
|------|----------|-------|
| `placeholder.md` | (this file) | all |
