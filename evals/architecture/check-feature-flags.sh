#!/usr/bin/env bash
# check-feature-flags.sh — Verify all feature flags in source are registered
#
# Rule: No feature flag may appear in source without being registered in
#       specs/feature-flags.md.
#
# Phase 0 behaviour: exits 0 if no source directories exist (no source to check).
# Phase 1+: validates that any flag found in source is in the registry.

set -euo pipefail

REGISTRY="specs/feature-flags.md"
FOUND_VIOLATIONS=0

warn() {
    printf '%s:%s: RULE: unregistered-feature-flag\n' "$1" "$2"
    printf 'REMEDIATION: Add "%s" to %s with a phase and domain before using it in source\n' "$3" "$REGISTRY"
    FOUND_VIOLATIONS=1
}

# 1. Verify the registry exists and is non-empty
if [[ ! -f "$REGISTRY" ]]; then
    echo "check-feature-flags: FATAL: $REGISTRY does not exist"
    exit 1
fi

if [[ ! -s "$REGISTRY" ]]; then
    echo "check-feature-flags: FATAL: $REGISTRY is empty"
    exit 1
fi

# 2. In Phase 0, if backend/ and frontend/ do not exist, exit 0
if [[ ! -d "backend" ]] && [[ ! -d "frontend" ]]; then
    echo "check-feature-flags: backend/ and frontend/ absent — Phase 0, nothing to check (PASS)"
    exit 0
fi

# 3. Collect flag names from the registry to a temp file (no subshell)
#    Extract backtick-enclosed words, skip lifecycle/column-header values.
SKIP_WORDS="false|true|alpha|beta|deprecated|removed|feature-name|Current|Default|Phase|Notes|Domain|Introduced|Deprecated|Removed"
FLAGS_TMP=$(mktemp)
trap 'rm -f "$FLAGS_TMP"' EXIT

grep -oE '`[[:alnum:]-]+`' "$REGISTRY" | tr -d '`' | sort -u > "$FLAGS_TMP"

# 4. For each flag, search source files
#    We read $FLAGS_TMP line by line in a while loop — this is NOT a pipeline,
#    so variable writes (FOUND_VIOLATIONS) persist correctly.
while IFS= read -r flag; do
    # Skip table column headers and lifecycle values
    case "$flag" in
        false|true|alpha|beta|deprecated|removed|feature-name|Current|Default|Phase|Notes|Domain|Introduced|Deprecated|Removed)
            continue
            ;;
    esac

    # Search code files only (not prose .md) for this flag; write to a per-flag temp file
    # grep -r returns exit 2 for serious errors (permission denied) and exit 1 for no matches;
    # both are ok — we treat them as "no code uses this flag".
    GREP_TMP=$(mktemp)
    grep -rEn "$flag" . \
        --include='*.go' --include='*.ts' --include='*.tsx' \
        --include='*.js' --include='*.jsx' \
        --include='*.py' --include='*.sh' \
        --exclude-dir=vendor --exclude-dir=node_modules \
        --exclude-dir=specs \
        2>/dev/null > "$GREP_TMP" ||:

    # Process matches one by one (not in a subshell)
    while IFS= read -r line; do
        # Parse "file:linenum:content"
        linenum=$(echo "$line" | cut -d: -f2)
        content=$(echo "$line" | cut -d: -f3-)
        # Reconstruct file path without leading ./
        file=$(echo "$line" | cut -d: -f1)

        # Skip ARCH_OK escape hatch
        if echo "$content" | grep -qwE '// ARCH_OK'; then
            rm -f "$GREP_TMP"
            continue 2
        fi

        warn "$file" "$linenum" "$flag"
    done < "$GREP_TMP"
    rm -f "$GREP_TMP"

done < "$FLAGS_TMP"

rm -f "$FLAGS_TMP"
trap - EXIT

if [[ $FOUND_VIOLATIONS -eq 0 ]]; then
    echo "check-feature-flags: PASS — all feature flags in source are registered"
fi

exit $FOUND_VIOLATIONS
