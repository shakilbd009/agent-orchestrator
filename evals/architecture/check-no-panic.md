# check-no-panic — Architecture Fitness Function

**Project:** agent-orchestrator
**Owner:** qa

---

## What It Checks

This fitness function enforces that Go source code does not contain unguarded
calls to `panic()`.

**Rule:** No `panic()` calls in any `.go` source file unless the line also
contains the `// ARCH_OK` escape-hatch comment.

**Rationale:** `panic()` is reserved for truly unrecoverable program states
(e.g., contract violations during development). A deployed service should
handle errors gracefully and return structured error responses. Panic in
production causes goroutines to crash and degrades the entire process.

---

## Phase 0 Scope

In Phase 0, there is no Go source code. This script therefore:

1. Exits `0` immediately if `backend/` does not exist or contains no `.go` files.
2. This is not a failure — Phase 0 deliberately has no source code.

---

## Failure Output

```
backend/somefile.go:42: RULE: panic-without-escape-hatch
REMEDIATION: Replace panic() with a structured error return; only panic for contract violations in development
```

---

## Escape Hatch

`// ARCH_OK` on the same line as a `panic()` suppresses the violation.

Example of an allowed guarded panic:
```go
panic("BUG: unrecoverable config state") // ARCH_OK: development-only safeguard, replaced in production
```

---

## References

- [Go Code Review Comments — Panic](https://github.com/golang/go/wiki/CodeReviewComments#panic)
- `AGENTS.md` — Phase conventions (no source code in Phase 0)
