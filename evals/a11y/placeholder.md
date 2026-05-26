# evals/a11y/ — Accessibility Test Placeholders

**Project:** agent-orchestrator
**Phase:** 0

---

## Purpose

This directory contains accessibility (a11y) test scenarios that verify the
frontend is usable by people with disabilities. This includes keyboard
navigation, screen reader compatibility, color contrast, and focus management.

---

## Phase 0 Status

Phase 0 has no frontend. Accessibility test files in this directory are
**placeholders** marked with `[PLACEHOLDER — FAILING BEFORE IMPLEMENTATION]`.

---

## Accessibility Standards

All frontend must conform to:

- **WCAG 2.1 AA** — Web Content Accessibility Guidelines
- **Keyboard navigable** — no mouse-only interactions
- **Screen reader tested** — semantic HTML, ARIA where needed
- **Color contrast** — minimum 4.5:1 for normal text, 3:1 for large text

---

## Running A11y Tests (Future)

```bash
# Uses Playwright with @playwright/test and axe-core
pnpm exec playwright test --project=a11y

# Standalone axe check
npx playwright open --axe <url>
```

---

## Placeholder Files

| File | Scenario | Phase |
|------|----------|-------|
| `placeholder.md` | (this file) | all |
