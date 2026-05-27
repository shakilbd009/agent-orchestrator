#!/usr/bin/env bash
# check-no-panic.sh — Verify no unguarded panic() calls in Go source
#
# Rule: panic() calls must be accompanied by // ARCH_OK on the same line.
#       panic() is only acceptable for unrecoverable development-state contract
#       violations, never in production service paths.
#
# Phase 0 behaviour: exits 0 if backend/ does not exist (no source to check).

set -euo pipefail

FOUND_VIOLATIONS=0

# shellcheck disable=SC2016
warn() {
    printf '%s:%s: RULE: panic-without-escape-hatch\n' "$1" "$2"
    printf 'REMEDIATION: Replace panic() with a structured error return; only panic for contract violations in development\n'
    FOUND_VIOLATIONS=1
}

# In Phase 0, no source directories exist
if [[ ! -d "backend" ]]; then
    echo "check-no-panic: backend/ absent — Phase 0, nothing to check (PASS)"
    exit 0
fi

# Find all .go files, skip vendor/, generated files
while IFS= read -r -d '' file; do
    # Skip generated protobuf files and vendor
    if [[ "$file" == *".pb.go"* ]] || [[ "$file" == *"/vendor/"* ]]; then
        continue
    fi

    # Find lines containing panic( that are not on the same line as ARCH_OK
    grep -nE 'panic\s*\(' "$file" 2>/dev/null || true | \
    while IFS= read -r line; do
        linenum=$(echo "$line" | cut -d: -f1)
        content=$(echo "$line" | cut -d: -f2-)

        # Escape hatch
        if echo "$content" | grep -qoE '// ARCH_OK'; then
            continue
        fi

        warn "$file" "$linenum"
    done
done < <(find backend -type f -name '*.go' -print0 2>/dev/null)

if [[ $FOUND_VIOLATIONS -eq 0 ]]; then
    echo "check-no-panic: PASS — no unguarded panic() calls found"
fi

exit $FOUND_VIOLATIONS
