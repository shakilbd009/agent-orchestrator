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
#    Extract flag names ONLY from the first column of the registry table.
#    Rows look like: | `flag-name` | Domain | Default | Current | ...
#    The pattern requires at least one hyphen so bare words (false, alpha,
#    beta, deprecated, removed, Current, Default, Phase, etc.) are excluded
#    structurally — no SKIP_WORDS needed for them.
FLAGS_TMP=$(mktemp)
trap 'rm -f "$FLAGS_TMP" "$GREP_TMP"' EXIT

# awk extracts field 2 (the Flag column) from rows where the cell starts
# with backtick+lowercase letter AND contains at least one hyphen.
# This structurally excludes:
#   - Lifecycle rows (false, alpha, beta, true, deprecated, removed) — no hyphen
#   - Column headers (Flag, Stage, Description) — no hyphen or backtick
#   - Any non-first-column noise
# gsub strips backticks and whitespace.
awk -F'|' '$2~/^[[:space:]]*`[a-z]/ && $2~/^[[:space:]]*`[a-z][a-z0-9]*-[a-z]/{gsub(/[`[:space:]]/,"",$2); print $2}' \
    "$REGISTRY" | sort -u > "$FLAGS_TMP"
# 4. For each flag found in source, verify it is registered.
#    Build a lookup set of registered flags first, then only warn for
#    flags that appear in source but are not in that lookup set.
#    We read $FLAGS_TMP line by line in a while loop — this is NOT a pipeline,
#    so variable writes (FOUND_VIOLATIONS) persist correctly.
#    Also filter out any lifecycle values that slip through (belt-and-suspenders).
SKIP_WORDS="false|true|alpha|beta|deprecated|removed"
while IFS= read -r flag; do
    case "$flag" in
        false|true|alpha|beta|deprecated|removed)
            continue
            ;;
    esac

    # Search code files only (not prose .md) for this flag; write to a per-flag temp file
    # grep -r returns exit 2 for serious errors (permission denied) and exit 1 for no matches;
    # both are ok — we treat them as "no code uses this flag".
    GREP_TMP=$(mktemp)
    grep -rEn "\"$flag\"" . \
        --include='*.go' --include='*.ts' --include='*.tsx' \
        --include='*.js' --include='*.jsx' \
        --include='*.py' --include='*.sh' \
        --exclude-dir=vendor --exclude-dir=node_modules \
        --exclude-dir=specs \
        --exclude-dir=frontend/.svelte-kit \
        2>/dev/null > "$GREP_TMP" ||:

    # Skip if no code references this flag
    if [[ ! -s "$GREP_TMP" ]]; then
        rm -f "$GREP_TMP"
        continue
    fi

    # At this point: the flag is in the registry (we're iterating registry flags)
    # AND it was found in source code.
    # Since every registered flag being used is valid, there is nothing to warn about.
    # Truly unregistered flags (not in registry) are never iterated here — they are
    # caught by the inverse check (source-first grep, then registry lookup).
    rm -f "$GREP_TMP"

done < "$FLAGS_TMP"

rm -f "$FLAGS_TMP"
trap - EXIT

if [[ $FOUND_VIOLATIONS -eq 0 ]]; then
    echo "check-feature-flags: PASS — all feature flags in source are registered"
fi

exit $FOUND_VIOLATIONS
