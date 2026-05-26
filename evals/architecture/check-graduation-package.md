# check-graduation-package — Architecture Fitness Function

**Project:** agent-orchestrator
**Owner:** qa

---

## What It Checks

This fitness function enforces that a Phase is complete before it "graduates"
to the next Phase. Graduation is the formal act of confirming that all required
artifacts for the current Phase are present and correctly formed.

**Rule:** Before the project transitions from Phase N to Phase N+1, all
artifacts required for Phase N must exist and pass their structural checks.

**Phase 0 graduation package:**

| Artifact | Required | Location |
|----------|----------|----------|
| `AGENTS.md` | Yes | project root |
| `STATUS.md` | Yes | project root |
| `docs/adr/0001-record-architecture-decisions.md` | Yes | docs/adr/ |
| `.gitignore` | Yes | project root |
| `.env.example` | Yes | project root |
| `agent-orchestrator.md` | Yes | project root |
| `contracts/openapi.yaml` | Yes | contracts/ |
| `contracts/events.md` | Yes | contracts/ |
| `specs/feature-flags.md` | Yes | specs/ |
| `evals/` directory | Yes | evals/ |
| `backend/` (empty dir) | Yes | Phase 1 start |
| `frontend/` (empty dir) | Yes | Phase 1 start |

> Note: `backend/` and `frontend/` are created as empty directories in Phase 0
> as a signal that Phase 1 scaffold begins here. They must not contain source
> code in Phase 0.

**Phase 1 graduation package:**

| Artifact | Required | Location |
|----------|----------|----------|
| All Phase 0 artifacts | Yes | (above) |
| `backend/go.mod` | Yes | backend/ |
| `backend/main.go` (minimal) | Yes | backend/ |
| `frontend/` with SvelteKit scaffold | Yes | frontend/ |
| `docker-compose.yml` | Yes | project root |
| `docs/adr/0002-app-shell.md` | Yes | docs/adr/ |

---

## Failure Output

```
check-graduation-package:14: RULE: phase-artifact-missing
  ARTIFACT: contracts/openapi.yaml
  REQUIRED FOR: Phase 0
  REMEDIATION: Create the missing artifact before graduating to Phase 1
```

---

## Escape Hatch

Not applicable — graduation requirements are binary. An artifact is either
present or it is not.

---

## How to Run

```bash
# Check Phase 0 graduation package
./evals/architecture/check-graduation-package.sh

# Check with explicit phase
PHASE=0 ./evals/architecture/check-graduation-package.sh
PHASE=1 ./evals/architecture/check-graduation-package.sh
```

---

## References

- `AGENTS.md` — Phase boundaries definition
- `agent-orchestrator.md` — BRD Phase description
