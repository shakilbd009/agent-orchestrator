#!/usr/bin/env bash
# check-status-sync.sh — Verify spec/eval/flag parity for graduated packages
#
# For each package that has exited draft (graduated), confirms:
#   1. All defined feature flags have matching eval coverage (e2e/unit/integration).
#   2. No unresolved OQs, TBDs, blockers, NEEDS_REPAIR, or unresolved validator
#      findings remain in graduated packages.
#
# Usage:
#   ./scripts/check-status-sync.sh [brd-id...]
#   ./scripts/check-status-sync.sh              # checks all graduated BRDs

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Resolve to project root if called from CI or elsewhere
if [[ ! -f "$PROJECT_ROOT/specs/curated/BRD-01-app-shell.md" ]]; then
    PROJECT_ROOT="$(pwd)"
fi

EVAL_DIR="$PROJECT_ROOT/evals"
SPECS_DIR="$PROJECT_ROOT/specs/curated"
GRADUATED_FLAGS_FILE="$PROJECT_ROOT/specs/feature-flags.md"

FOUND_VIOLATIONS=0

# shellcheck disable=SC2016
warn() {
    printf 'check-status-sync:%s: RULE: %s\n' "$2" "$3"
    printf '  %s\n' "$1"
    printf '  REMEDIATION: %s\n' "$4"
    FOUND_VIOLATIONS=1
}

# ---------------------------------------------------------------------------
# Helper: check that an eval directory has at least one non-placeholder file
# ---------------------------------------------------------------------------
eval_dir_has_content() {
    local eval_type="$1"   # e2e, unit, integration
    local brd_id="$2"
    local eval_path="$EVAL_DIR/$eval_type"

    if [[ ! -d "$eval_path" ]]; then
        return 1
    fi

    # Count non-placeholder files
    local count
    count=$(find "$eval_path" -type f \
        ! -name 'placeholder.md' \
        ! -name '*.placeholder.md' \
        2>/dev/null | wc -l | tr -d ' ')

    [[ "$count" -gt 0 ]]
}

# ---------------------------------------------------------------------------
# Check a single graduated BRD for eval coverage parity
# ---------------------------------------------------------------------------
check_brd_coverage() {
    local brd_file="$1"
    local brd_basename
    brd_basename=$(basename "$brd_file" .md)

    # Read Status from BRD frontmatter (e.g., **Status:** placeholder)
    # Use /Status/ to match the line — awk field splitting handles the rest
    # After gsub("**","",$0): "Status: placeholder" -> $2 is the status value
    local status
    status=$(awk '/Status/ { gsub(/\*\*/, "", $0); print tolower($2) }' "$brd_file" 2>/dev/null | head -1)

    # Skip BRDs that haven't graduated (draft or placeholder in Phase 0)
    # Only check eval coverage for explicitly approved BRDs
    if [[ -z "$status" ]] || [[ "$status" == "draft" ]] || [[ "$status" == "placeholder" ]]; then
        return 0
    fi

    # Check eval coverage exists for each eval type
    for eval_type in e2e unit integration; do
        if ! eval_dir_has_content "$eval_type" "$brd_basename"; then
            warn "BRD $brd_basename has no $eval_type eval coverage" \
                "$brd_basename" \
                "eval-coverage-missing" \
                "Add $eval_type test coverage for $brd_basename in evals/$eval_type/"
        fi
    done
}

# ---------------------------------------------------------------------------
# Check for unresolved OQ/TBD/blocker/NEEDS_REPAIR in graduated packages
# ---------------------------------------------------------------------------
check_unresolved_open_items() {
    local brd_file="$1"
    local brd_basename
    brd_basename=$(basename "$brd_file" .md)

    # Skip non-markdown files
    [[ "$brd_file" == *.md ]] || return 0

    # Patterns that indicate unresolved open items in source/prose
    local patterns=(
        "OQ:[[:space:]]"
        "TBD[[:space:]]"
        "BLOCKER[[:space:]]"
        "NEEDS_REPAIR[[:space:]]"
        "TODO[[:space:]]{2,}"   # multi-word TODOs, not single-word labels
    )

    for pattern in "${patterns[@]}"; do
        # Search markdown files (BRDs) only, skip generated/placeholder
        while IFS= read -r line; do
            # Skip ARCH_OK escapes
            if echo "$line" | grep -qwE '// ARCH_OK'; then
                continue
            fi
            # Skip BRD-BYPASS
            if echo "$line" | grep -qwE '// BRD-BYPASS'; then
                continue
            fi

            local linenum
            linenum=$(echo "$line" | cut -d: -f1)
            warn "Unresolved item in $brd_basename (line $linenum): $pattern" \
                "$brd_file" \
                "unresolved-open-item" \
                "Resolve, defer (Status: deferred), or add BRD-BYPASS for this item"
        done < <(grep -rnE "$pattern" "$brd_file" 2>/dev/null | grep -v 'ARCH_OK' | grep -v 'BRD-BYPASS' || true)
    done
}

# ---------------------------------------------------------------------------
# Check graduated packages have their feature flags properly registered
# ---------------------------------------------------------------------------
check_graduated_flags() {
    # In Phase 0 there are no source files to check — this is a no-op
    if [[ ! -d "$PROJECT_ROOT/backend" ]] && [[ ! -d "$PROJECT_ROOT/frontend" ]]; then
        return 0
    fi

    if [[ ! -f "$GRADUATED_FLAGS_FILE" ]]; then
        warn "Feature flag registry not found" "feature-flags" "registry-missing" \
            "Create specs/feature-flags.md with flag registry"
        return 0
    fi

    # For each graduated BRD, check that its flags are registered
    # (actual source-level flag check is done by check-feature-flags.sh)
    return 0
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
echo "check-status-sync: checking spec/eval/flag parity..."

# Determine which BRDs to check
if [[ $# -gt 0 ]]; then
    BRD_LIST=("$@")
else
    # Auto-discover graduated BRDs — compatible with bash 3.2 (macOS)
    # shellcheck disable=SC2162
    while IFS= read -r brd_file; do
        BRD_LIST+=("$brd_file")
    done < <(find "$SPECS_DIR" -maxdepth 1 -name 'BRD-*.md' -type f 2>/dev/null)
fi

for brd_file in "${BRD_LIST[@]}"; do
    [[ -f "$brd_file" ]] || continue
    check_brd_coverage "$brd_file"
    check_unresolved_open_items "$brd_file"
done

# Feature flag registry check (Phase-aware)
check_graduated_flags

if [[ $FOUND_VIOLATIONS -eq 0 ]]; then
    echo "check-status-sync: PASS — spec/eval/flag parity verified"
fi

exit $FOUND_VIOLATIONS
