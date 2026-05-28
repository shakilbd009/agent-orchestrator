#!/usr/bin/env bash
# check-no-background-context.sh — Verify no implicit context capture in agent code
#
# Rule: Functions registered as skill handlers or task handlers may not close
#       over variables that were not passed as explicit named parameters.
#       Reading from a global or ambient context object without it being a
#       named parameter is flagged as background-context-capture.
#
# Phase 0 behaviour: exits 0 if no source directories exist.

set -euo pipefail

FOUND_VIOLATIONS=0

# shellcheck disable=SC2016
warn() {
    printf '%s:%s: RULE: background-context-capture\n' "$1" "$2"
    printf 'REMEDIATION: Pass all context as explicit named parameters; do not capture implicit globals\n'
    FOUND_VIOLATIONS=1
}

# Check all source files across backend and frontend
SRC_DIRS=()
[[ -d "backend" ]] && SRC_DIRS+=("backend")
[[ -d "frontend" ]] && SRC_DIRS+=("frontend")

if [[ ${#SRC_DIRS[@]} -eq 0 ]]; then
    echo "check-no-background-context: backend/ and frontend/ absent — Phase 0, nothing to check (PASS)"
    exit 0
fi

# Skip these directory names entirely (generated / vendored / node_modules)
SKIP_DIRS=(
    "node_modules"
    ".pnpm"
    ".svelte-kit"
    "vendor"
    "__pycache__"
    ".next"
    "dist"
    "build"
)

# Build find -not -path predicates to skip large generated/vendor directories.
# Original script processed 1502 files including node_modules/.pnpm/.svelte-kit.
# Using -not -path "*/dir/*" instead of -path dir -prune -o keeps the logic simple
# and correct on both GNU and macOS BSD find.
PRUNE_ARGS=()
for d in "${SKIP_DIRS[@]}"; do
    PRUNE_ARGS+=( -not -path "*/${d}/*" )
done

# Known implicit context/session accessor patterns to flag
IMPLICIT_PATTERNS='session_get|get_context|ctx\.get\(|global_session|session\[|get_session\('

# Function boundary patterns for various languages
FUNC_PATTERNS='^(func |^(export )?async def |^(export )?function [[:alnum:]_]+\(|^const [[:alnum:]_]+ = (async )?\(|^export (async )?function |^[[:alnum:]_]+[[:space:]]*\(\)|^function [[:alnum:]_]+'

# ARCH_OK marker to suppress false positives
ARCH_OK_LINE='// ARCH_OK|# ARCH_OK|ARCH_OK'

# Step 1 — batch discovery: find files containing implicit context patterns.
# This is O(N) with fast binary grep, not O(N * lines_per_file).
# Files in skip directories are excluded by find prune.
CANDIDATE_TMP=$(mktemp)
find "${SRC_DIRS[@]}" -type f \
    \( -name '*.go' -o -name '*.py' -o -name '*.ts' -o -name '*.tsx' -o -name '*.js' -o -name '*.sh' \) \
    "${PRUNE_ARGS[@]}" \
    -print 2>/dev/null | \
xargs grep -lE "$IMPLICIT_PATTERNS" 2>/dev/null || true > "$CANDIDATE_TMP"

CANDIDATE_COUNT=0
while IFS= read -r line; do
    ((CANDIDATE_COUNT++)) || true
done < "$CANDIDATE_TMP"

if [[ $CANDIDATE_COUNT -eq 0 ]]; then
    echo "check-no-background-context: PASS — no implicit context capture detected"
    rm -f "$CANDIDATE_TMP"
    exit 0
fi

echo "check-no-background-context: scanning $CANDIDATE_COUNT candidate files..."

# Step 2 — per-candidate function-body analysis
while IFS= read -r file; do
    # Skip already-handled generated file types
    if [[ "$file" == *.pb.go ]] || [[ "$file" == *.graphql.js ]]; then
        continue
    fi

    # Get all implicit-pattern match line numbers (filter ARCH_OK)
    IMPLICIT_LINES=$(grep -nE "$IMPLICIT_PATTERNS" "$file" 2>/dev/null || true)
    # Remove ARCH_OK lines
    IMPLICIT_LINES=$(echo "$IMPLICIT_LINES" | grep -vE "$ARCH_OK_LINE" || true)

    if [[ -z "$IMPLICIT_LINES" ]]; then
        continue
    fi

    # Get all function-start line numbers
    FUNC_LINES=$(grep -nE "$FUNC_PATTERNS" "$file" 2>/dev/null || true)

    if [[ -z "$FUNC_LINES" ]]; then
        # No function detected; any remaining implicit lines are violations
        while IFS= read -r impl_line; do
            [[ -z "$impl_line" ]] && continue
            linenum=$(echo "$impl_line" | cut -d: -f1)
            warn "$file" "$linenum"
        done <<< "$IMPLICIT_LINES"
        continue
    fi

    # Total lines in file
    total_lines=$(wc -l < "$file")

    # For each function start, scan forward to find end of function body
    while IFS= read -r line; do
        [[ -z "$line" ]] && continue
        func_linenum=$(echo "$line" | cut -d: -f1)

        # Find end: first non-indented line after func_linenum
        func_end_linenum=$total_lines
        reading_body=false

        for ((linenum = func_linenum + 1; linenum <= total_lines; linenum++)); do
            body_line=$(sed -n "${linenum}p" "$file")

            # Skip empty lines
            [[ -z "$body_line" ]] && continue

            # Non-indented line at column 0 exits function
            if [[ "$body_line" =~ ^[^[:space:]] ]]; then
                func_end_linenum=$((linenum - 1))
                break
            fi

            # Closing brace at column 0 also exits
            if [[ "$body_line" =~ ^[[:space:]]*\} ]]; then
                func_end_linenum=$((linenum - 1))
                break
            fi
        done

        # Check each implicit pattern line: is it inside this function body?
        while IFS= read -r impl_line; do
            [[ -z "$impl_line" ]] && continue
            impl_linenum=$(echo "$impl_line" | cut -d: -f1)

            if [[ "$impl_linenum" -ge "$func_linenum" ]] && [[ "$impl_linenum" -le "$func_end_linenum" ]]; then
                warn "$file" "$impl_linenum"
            fi
        done <<< "$IMPLICIT_LINES"

    done <<< "$FUNC_LINES"

done < "$CANDIDATE_TMP"

rm -f "$CANDIDATE_TMP"

if [[ $FOUND_VIOLATIONS -eq 0 ]]; then
    echo "check-no-background-context: PASS — no implicit context capture detected"
fi

exit $FOUND_VIOLATIONS