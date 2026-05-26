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

# Known implicit context/session accessor patterns to flag
# These are heuristics: in Phase 1 these will be refined via static analysis.
IMPLICIT_PATTERNS='("session_get|get_context|ctx\.get\(|global_session|session\[|get_session\()'

# For each source file, look for patterns that suggest implicit context capture
# in what looks like a skill handler or task handler function.
while IFS= read -r -d '' file; do
    # Skip vendor, generated, node_modules
    if [[ "$file" == *"/vendor/"* ]] || \
       [[ "$file" == *"node_modules/"* ]] || \
       [[ "$file" == *".pb.go"* ]] || \
       [[ "$file" == *"*.graphql.js"* ]]; then
        continue
    fi

    # We scan for function definitions that contain implicit context reads
    # This is a heuristic check: it flags functions that contain both a
    # function definition keyword and one of the implicit context patterns,
    # without the pattern being on a parameter line.
    #
    # Strategy: for each function definition in the file, collect its body lines
    # and check if any implicit context access occurs outside an ARCH_OK line.

    local linenum=0
    local in_function=0
    local func_name=""
    local func_body_lines=""

    while IFS= read -r line; do
        linenum=$((linenum + 1))

        # Detect function definitions in various languages
        # Go: func name(
        # Python: def name(
        # TypeScript/JS: function name( or const name = ( or async name(
        # Shell: name() {  or  function name {
        if echo "$line" | grep -qoE '^(func |^(export )?async def |^(export )?function [[:alnum:]_]+\(|^const [[:alnum:]_]+ = (async )?\(|^export (async )?function |^[[:alnum:]_]+[[:space:]]*\(\)|^function [[:alnum:]_]+)'; then
            # Start of a new function — reset state
            in_function=1
            func_name="$line"
            func_body_lines=""
            continue
        fi

        # End of function: blank line or line not indented (top-level)
        if [[ $in_function -eq 1 ]]; then
            # If this line is not indented and not empty and not a closing brace, we exited the function
            if echo "$line" | grep -qoE '^[^[:space:]]'; then
                in_function=0
            else
                func_body_lines="${func_body_lines}${line}"$'\n'
            fi
        fi

        # Now check this line for implicit context patterns
        if echo "$line" | grep -qoE "$IMPLICIT_PATTERNS"; then
            # Skip ARCH_OK lines
            if echo "$line" | grep -qoE '// ARCH_OK|# ARCH_OK|ARCH_OK'; then
                continue
            fi

            warn "$file" "$linenum"
        fi
    done < "$file"

done < <(find "${SRC_DIRS[@]}" -type f \( -name '*.go' -o -name '*.py' -o -name '*.ts' -o -name '*.tsx' -o -name '*.js' -o -name '*.sh' \) -print0 2>/dev/null)

if [[ $FOUND_VIOLATIONS -eq 0 ]]; then
    echo "check-no-background-context: PASS — no implicit context capture detected"
fi

exit $FOUND_VIOLATIONS
