# evals/CONVENTIONS.md — Phase 0 Eval Conventions

**Project:** agent-orchestrator
**Phase:** 0 — Governance & Scaffolding
**Owner:** qa

---

## Purpose

This file records the conventions that govern all eval scripts in this project.
These conventions ensure eval scripts are safe, portable, auditable, and
consistent with the project's governance model.

---

## Shell Script Conventions

All eval scripts (`.sh`) in `evals/` must comply with the following:

1. **Shebang:** `#!/usr/bin/env bash` — portable across macOS and Linux.

2. **Strict mode:** Every script must open with:
   ```bash
   set -euo pipefail
   ```
   - `-e`: exit immediately on non-zero exit status
   - `-u`: treat unset variables as errors
   - `-f`: disable glob expansion (prevents accidental path expansion)
   - `pipefail`: fail if any command in a pipeline fails, not just the last

3. **Executable:** Every `.sh` file must have the executable bit set (`chmod +x`).

4. **No `grep -P`:** Perl-compatible regex (`grep -P`) is not portable and may not
   be available on macOS. Use `grep -oE` with POSIX extended regular expressions
   or POSIX character classes (`[[:alnum:]]`, `[[:space:]]`, etc.) instead.

5. **No external dependencies beyond stdlib:** Scripts must not require `jq`,
   `yq`, `fgrep`, or other non-built-in tools beyond what is in the POSIX/GNU
   coreutils and bash 3.2+ (the bash on macOS). If additional tools are needed,
   they must be detected and produce a clear error message if absent.

6. **Graceful handling of missing source directories:**
   When a script scans a source directory (e.g., `backend/`, `frontend/`),
   it must exit `0` if the directory does not exist. It is not a failure for
   Phase 0 — Phase 0 deliberately has no source code.

7. **Failure output format:** When a script detects a violation, it must print:
   ```
   FILE:LINE: RULE: <rule name>
   REMEDIATION: <one-line fix description>
   ```
   The script must exit 1 on violation, 0 on clean pass.

---

## Source-Level Check Conventions

For architecture fitness functions that inspect source code:

1. **Escape hatch:** Source-level checks must respect the `// ARCH_OK` inline
   comment as a per-line suppression mechanism. When a line contains the literal
   text `// ARCH_OK`, the rule should not fire on that line.

2. **Bypass comment:** BRD-open-items check supports inline bypass via a
   specially-formatted comment: `// BRD-BYPASS: <reason>`. This is reserved
   exclusively for the `check-brd-open-items` rule and must not be recognised
   by other rules.

3. **No false positives on generated files:** Eval scripts should skip
   obviously-generated files (e.g., files in `node_modules/`, `vendor/`,
   `*.pb.go`, `*.graphql.js`).

---

## BRD Open Items Check Conventions

The `check-brd-open-items` script enforces that every open item, risk, or
decision in the BRD has a corresponding tracking artifact.

1. An inline `// BRD-BYPASS: <reason>` comment on the same line suppresses the
   open item flag for that specific item only.

2. Every open item that is deferred (not implemented in the current phase)
   must be tagged with a `Status:` field (e.g., `Status: deferred`, `Status: open`)
   in the BRD body.

3. The script outputs one line per open item found without a matching artifact
   or explicit deferral status.

---

## E2E / Integration Test Placeholder Conventions

Placeholders for E2E and integration tests must follow these rules:

1. **Marked as failing before implementation:** Every scenario placeholder must
   include the literal string `[PLACEHOLDER — FAILING BEFORE IMPLEMENTATION]`
   in the test body or comment so CI clearly identifies it as non-functional.

2. **Descriptive scenario names:** Test names should describe the scenario in
   plain language (e.g., `agent-can-promote-task-with-human-gate-open`).

3. **Phase alignment:** Test placeholders should be placed in the directory
   corresponding to the phase in which their implementation is planned.

4. **No source code in Phase 0 tests:** Phase 0 has no source code. No test
   placeholder may reference `backend/`, `frontend/`, or any implementation path
   that would not exist at Phase 0.

---

## Eval Directory Structure

```
evals/
  CONVENTIONS.md          # This file
  architecture/
    check-feature-flags.sh           # Architecture: feature flag discipline
    check-feature-flags.md           # Documentation for the check
    check-no-panic.sh                # Go: no panic() in source
    check-no-panic.md
    check-no-background-context.sh  # No background context leaks in agent code
    check-no-background-context.md
    check-brd-open-items.sh          # BRD open items have tracking artifacts
    check-brd-open-items.md
    check-graduation-package.sh      # All required Phase N artifacts present
    check-graduation-package.md
  e2e/                   # End-to-end / browser / E2E test placeholders
  unit/                 # Unit test placeholders
  integration/          # Integration test placeholders
  a11y/                 # Accessibility test placeholders
  visual/               # Visual/regression test placeholders
```

---

## Running Evals

Architecture checks can be run individually:
```bash
./evals/architecture/check-feature-flags.sh
./evals/architecture/check-no-panic.sh
# etc.
```

Or all at once via a future top-level eval runner (not yet implemented in Phase 0).

---

## Maintenance

This conventions file is owned by the `qa` profile. Any changes to these
conventions require a PR reviewed by the `architect` or `qa` profile.

---

_Last updated: Phase 0 batch 5 (task `t_ab459e27`)_
