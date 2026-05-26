# check-feature-flags — Architecture Fitness Function

**Project:** agent-orchestrator
**Owner:** qa

---

## What It Checks

This fitness function enforces that all feature flags referenced in source code
are registered in `specs/feature-flags.md`.

**Rule:** No feature flag may appear in any source file that is not registered
in the Feature Flag Registry (`specs/feature-flags.md`).

**Rationale:** Unregistered flags indicate governance drift — a developer may have
introduced a flag without architect review and without updating the registry.
This is a Phase 0 governance check that will evolve into a source-code-static
analysis check in Phase 1.

---

## Phase 0 Scope

In Phase 0, there is no source code. This script therefore:

1. Verifies that `specs/feature-flags.md` exists and is non-empty.
2. Verifies the registry contains at least the Phase 0 flags defined in ADR-0001.
3. Exits `0` immediately if `backend/` and `frontend/` do not exist (no code to check).
   This is not a failure — Phase 0 deliberately has no source code.

Once Phase 1 creates code, the check activates and validates that any feature flag
referenced in `.go`, `.ts`, `.tsx`, `.js`, `.jsx`, `.py`, or `.sh` files
is registered in the Feature Flag Registry. Prose `.md` files are excluded to
avoid false positives on natural-language mentions of concepts like "dashboard" or
"collaboration".

---

## Failure Output

```
FILE:LINE: RULE: unregistered-feature-flag
REMEDIATION: Add "flag-name" to specs/feature-flags.md with a phase and domain before using it in source
```

---

## Escape Hatch

Not applicable in Phase 0 (no source code). In Phase 1, the escape hatch
mechanism (`// ARCH_OK`) will suppress flag violations on a per-line basis.

---

## References

- `specs/feature-flags.md` — Feature Flag Registry
- `docs/adr/0001-record-architecture-decisions.md` — Decision to use feature flags
- `agent-orchestrator.md` — BRD (open items reference)
