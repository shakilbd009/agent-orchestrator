# check-no-background-context — Architecture Fitness Function

**Project:** agent-orchestrator
**Owner:** qa

---

## What It Checks

This fitness function enforces that agent skill definitions and task handlers
do not capture or retain background context that was not explicitly passed
as a parameter.

**Rule:** No function, closure, or skill handler may read from a shared global
or ambient context variable that is not injected as a named parameter.

**Rationale:** Agent systems are most auditable and testable when data flow is
explicit. Hidden context capture (e.g., reading from a global `session` object
inside a skill handler) makes it impossible to reason about what a skill will
do without reading its entire closure chain. This rule prevents "context smuggling"
where an agent silently carries state from one task into an unrelated task.

**Specific patterns flagged:**
- Reading from an undeclared global variable inside a function body
- Closing over a variable that was not passed as a parameter ("captured variable")
  in a function that is registered as a skill or task handler
- Use of implicit ambient session or context objects (e.g., `session_get`,
  `get_context()`, `ctx.get()`) without the variable being a named function parameter

---

## Phase 0 Scope

In Phase 0, there is no source code. This script therefore:

1. Exits `0` immediately if `backend/` and `frontend/` do not exist or contain
   no source files of the target languages.
2. This is not a failure — Phase 0 deliberately has no source code.

---

## Failure Output

```
skills/my-skill.md:23: RULE: background-context-capture
  PATTERN: closure captures variable "session" that was not passed as parameter
REMEDIATION: Pass "session" as an explicit named parameter to this function/closure
```

---

## Escape Hatch

`// ARCH_OK` on the same line as the violation suppresses it.

Example:
```python
def handle_task(event, session):  # ARCH_OK: session is an explicit parameter
    cached = global_cache.read()  # still flagged separately if global_cache is implicit
```

---

## References

- `AGENTS.md` — Layer A / Layer B separation (context must be explicit)
- `agent-orchestrator.md` — Audit trail requirements
