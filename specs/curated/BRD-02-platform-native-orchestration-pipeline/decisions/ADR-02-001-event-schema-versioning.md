# ADR-02-001: Event Schema Version Naming Convention

**ADR:** ADR-02-001  
**Subject:** Event schema version naming convention for canonical event envelope  
**Profile:** architect  
**Date:** 2026-05-28  
**Status:** Accepted

---

## Problem Statement

BRD-02 defines a canonical event envelope in FR-02-022A that includes a `schemaVersion` field. The format of this version string is not specified. Phase 0 placeholder events used `v1alpha` as a stub. Open Question 4 (BRD-02) asks whether schemas should start at `v1alpha` or `v1`. Neither the BRD nor the Phase 0 contract specifies a naming convention.

Without a defined convention:
- Consumers cannot programmatically determine event stability
- CI cannot validate breaking-change rules
- Schema evolution policy is undefined
- FR-02-032 (should-have schema version field) remains unsatisfied

---

## Options Considered

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| A | `v1alpha` during implementation; promote to `v1` at GA | Clear pre-GA signal; meaningful GA milestone | Requires version bump at GA |
| B | `v1` from day 1 of BRD-02 implementation | Simpler; no transitional version | False claim of stability before contract is frozen |
| C | `v1` incremented to `v2alpha` on breaking changes | Calver-style clarity | Requires coordination on version bumps |
| D | No schema version field | Simplest envelope | Violates FR-02-032 should-have; no consumer stability signal |

---

## Decision

**Option A** is selected: `v1alpha` during Phase 1 implementation; `v1` once BRD-02 implementation is complete, contract is frozen, and the curation gate passes.

**Specific rules:**
1. All events generated during Phase 1 implementation carry `schemaVersion: "v1alpha"`.
2. Upon BRD-02 implementation completion (all ACs passing), the schema version advances to `"v1"`.
3. Breaking changes after GA require a new major version bump (e.g., `v2alpha` → `v2`).
4. Non-breaking additive changes (new optional fields) do NOT increment the major version.
5. The schema version is injected at build time via `ldflags` from a version variable. No runtime injection from config.

**Implementation note:** Build pipeline must set `-X version.SchemaVersion=v1alpha` or `-X version.SchemaVersion=v1` at build time. Version variable in a `version` package imported by the event encoder.

---

## Consequences

**Positive:**
- Consumers can programmatically gate on `v1` vs `v1alpha` stability
- CI can enforce "no `v1` events until GA" pre-commit rule
- Schema evolution policy is explicit
- FR-02-032 satisfied with clear versioning semantics

**Negative:**
- Version promotion requires a deliberate build pipeline step at GA
- Consumers processing Phase 1 events may need `v1alpha` handling

**Neutral:**
- `v1alpha` events are not subject to breaking-change CI rules
- `v1` events after promotion ARE subject to breaking-change rules

---

## Verification

- Build injects schema version via `ldflags`; version string appears in all emitted events
- CI checks that no `v1alpha` events are emitted in `v1` builds (negative test)
- Events emitted in CI/test environment carry `v1alpha` by default unless override is set
- Audit log entries include schemaVersion field and value matching build tag
