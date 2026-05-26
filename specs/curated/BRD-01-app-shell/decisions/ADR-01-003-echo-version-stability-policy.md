# ADR-01-003: Echo Version Stability Policy

**Status:** Proposed  
**Date:** 2026-05-25  
**Deciders:** architect (subagent, BRD-01 refinement)  
**Supersedes:** None  
**Superseded by:** None  

---

## Context

ADR-0001 pinned Echo to `v4.15.2`. This ADR formalizes the policy for how that pin interacts with minor/patch updates. Echo v4 is stable and widely used; however, the pin's intent needs explicit documentation to prevent drift.

---

## Decision Drivers

1. AGENTS.md pins Echo to `v4.15.2` (minor+patch) — not just major version
2. Echo v4 API surface is stable; v5 would require a separate ADR (significant scope)
3. Rule 12 (Fail Loud) discourages undefined version behavior
4. Security patches in patch releases are desirable
5. Reproducibility is a core project value

---

## Options Considered

### Option A: Strict pin `v4.15.2` — no automatic updates (Chosen)

The exact version `v4.15.2` is locked in `go.mod`. No `go get -u` runs without a new ADR.

**Pros:**
- Maximum reproducibility — every environment gets identical behavior
- No surprise transitive dependency changes
- Clear audit trail — version change requires ADR and code review

**Cons:**
- Misses potential bug fixes in `v4.15.3+`
- Security patches requires manual ADR-driven update each time

### Option B: Minor-version flexible `v4.15.2` → `v4.15.x`

```
github.com/labstack/echo/v4 v4.15.2 // locked or loosely pinned?
```
Using `go.mod` replace or `go get -u github.com/labstack/echo/v4@v4.15` to allow 4.15.2 → 4.15.latest.

**Pros:**
- Security patches auto-flow
- Minimal breakage risk within minor version

**Cons:**
- Slightly less reproducibility (difference between 4.15.2 and 4.15.10)
- Transitive dependencies may shift subtly

### Option C: Major-version flexible `v4.x` → `v5.x`

Allow any v4 or v5 version without constraint.

**Pros:**
- Latest features and fixes

**Cons:**
- v5 has breaking changes; would require implementation work
- Goes against ADR-0001 which pins specific minor version
- Incompatible with systematic-refinement — version would change beneath us

---

## Decision

**Option A — Strict pin with no automatic updates** is adopted for Phase 1.

Echo v4.15.2 is locked in `go.mod`. Every version change (patch or minor) requires:
1. New ADR proposing the update
2. Verification that existing tests pass
3. Explicit decision rationale (security patch, bug fix, or feature need)

This is consistent with Rule 12 and the project's governance model where "immutable once merged" applies to tool chains as well as architecture.

---

## Consequences

### Positive
- Complete reproducibility — `go mod download` always fetches identical version
- Governance is consistent across all pinned tools (Go, Node, pnpm, Echo)
- ADR audit trail shows why each version change happened

### Negative
- Manual overhead for patch updates (ADR + review cycle)
- Security patches lag until someone triggers the update

### Neutral
- In Phase 2+, a separate ADR can create a "rapid update" exception for security patches only
- Echo v4.15.x has been stable in production across many projects — update cadence is low risk

---

## Implementation Notes

`go.mod` must include:
```
require github.com/labstack/echo/v4 v4.15.2
```

No `//go:embed` directives or replace rules should shadow this version.

If a security patch is needed urgently:
1. Open ADR with "Emergency patch" prefix in title
2. Fast-track review (same day if possible)
3. Merge and update version pin immediately

---

## References

- ADR-0001: Record Architecture Decisions (Echo version pin source)
- AGENTS.md (Echo v4.15.2 entry in pinned tools table)
- Rule 12 (Fail Loud)
