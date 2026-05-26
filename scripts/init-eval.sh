#!/usr/bin/env bash
# init-eval.sh — Initialise the local eval environment
#
# Verifies that all architecture-check scripts are present and executable,
# then runs a Phase-gated static check to confirm the workspace is ready.
#
# Usage:
#   ./scripts/init-eval.sh          # Check current phase automatically
#   ./scripts/init-eval.sh --phase 0   # Force Phase 0 checks

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
EVAL_ARCH_DIR="$PROJECT_ROOT/evals/architecture"

# Parse --phase flag
FORCE_PHASE=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --phase)
            FORCE_PHASE="$2"
            shift 2
            ;;
        *)
            echo "init-eval: unknown argument: $1" >&2
            echo "Usage: $0 [--phase N]" >&2
            exit 1
            ;;
    esac
done

# Determine phase
if [[ -n "$FORCE_PHASE" ]]; then
    PHASE="$FORCE_PHASE"
elif [[ -d "$PROJECT_ROOT/backend" ]] && [[ -n "$(find "$PROJECT_ROOT/backend" -type f ! -name '.gitkeep' 2>/dev/null)" ]]; then
    PHASE=1
elif [[ -f "$PROJECT_ROOT/backend/go.mod" ]]; then
    PHASE=1
elif [[ -f "$PROJECT_ROOT/frontend/package.json" ]]; then
    PHASE=1
else
    PHASE=0
fi

echo "init-eval: Phase $PHASE — checking eval environment..."

# 1. Verify architecture check scripts exist and are executable
REQUIRED_CHECKS=(
    "check-graduation-package.sh"
    "check-brd-open-items.sh"
    "check-feature-flags.sh"
    "check-no-panic.sh"
    "check-no-background-context.sh"
)

for check in "${REQUIRED_CHECKS[@]}"; do
    path="$EVAL_ARCH_DIR/$check"
    if [[ ! -f "$path" ]]; then
        echo "init-eval: FATAL: missing eval check: $path" >&2
        exit 1
    fi
    if [[ ! -x "$path" ]]; then
        echo "init-eval: WARNING: $check is not executable — fixing" >&2
        chmod +x "$path"
    fi
done
echo "init-eval: architecture check scripts OK"

# 2. Run Phase-gated static checks
PASS=0
FAIL=0

run_check() {
    local name="$1"
    local cmd="$2"
    echo ""
    echo "=== $name ==="
    if eval "$cmd"; then
        echo "init-eval: $name — PASS"
        PASS=$((PASS + 1))
    else
        local exitcode=$?
        echo "init-eval: $name — FAIL (exit $exitcode)"
        FAIL=$((FAIL + 1))
    fi
}

# Graduation package check — always runs
run_check "graduation-package (Phase $PHASE)" \
    "PHASE=$PHASE '$EVAL_ARCH_DIR/check-graduation-package.sh'"

# BRD open items check — always runs in Phase 0 (no source needed)
run_check "brd-open-items" \
    "'$EVAL_ARCH_DIR/check-brd-open-items.sh'"

# Feature flags check — exits 0 in Phase 0 if no source dirs exist
run_check "feature-flags" \
    "'$EVAL_ARCH_DIR/check-feature-flags.sh'"

# No-panic check — Phase 1+ only (requires Go source)
if [[ "$PHASE" -ge 1 ]] && [[ -d "$PROJECT_ROOT/backend" ]]; then
    run_check "no-panic" "'$EVAL_ARCH_DIR/check-no-panic.sh'"
elif [[ "$PHASE" -ge 1 ]]; then
    echo "init-eval: no-panic — SKIP (backend/ empty)"
fi

# No-background-context check — Phase 1+ only
if [[ "$PHASE" -ge 1 ]] && [[ -d "$PROJECT_ROOT/backend" ]]; then
    run_check "no-background-context" "'$EVAL_ARCH_DIR/check-no-background-context.sh'"
elif [[ "$PHASE" -ge 1 ]]; then
    echo "init-eval: no-background-context — SKIP (backend/ empty)"
fi

# 3. Summary
echo ""
echo "=== init-eval summary ==="
echo "Phase: $PHASE"
echo "Passed: $PASS"
echo "Failed: $FAIL"

if [[ $FAIL -gt 0 ]]; then
    echo "init-eval: FAIL — one or more checks failed" >&2
    exit 1
fi

echo "init-eval: DONE — eval environment is ready"
exit 0
