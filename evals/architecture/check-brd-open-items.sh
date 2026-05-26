#!/usr/bin/env bash
# check-brd-open-items.sh — Verify all BRD open items have tracking artifacts
#
# Rule: Every open item, risk, or deferred feature in agent-orchestrator.md
#       must have a corresponding tracking artifact (kanban task or ADR) or an
#       explicit Status: deferred annotation in the BRD itself.
#
# Inline bypass: // BRD-BYPASS: <reason> on the same line suppresses the flag
#               for that specific item only.
#
# Phase 0 behaviour: validates that known Phase 0 open items are properly
#                   annotated with Status: deferred or have a tracking artifact.

set -euo pipefail

BRD="agent-orchestrator.md"
FOUND_VIOLATIONS=0

# shellcheck disable=SC2016
warn() {
    printf '%s:%s: RULE: brd-open-item-untracked\n' "$1" "$2"
    printf '  ITEM: %s\n' "$3"
    printf '  DEFERRED TO: %s\n' "$4"
    printf '  REMEDIATION: Create a kanban task or ADR for %s before proceeding\n' "$4"
    FOUND_VIOLATIONS=1
}

if [[ ! -f "$BRD" ]]; then
    echo "check-brd-open-items: FATAL: $BRD does not exist"
    exit 1
fi

# Known BRD open items that must have a deferral status or tracking artifact.
# Format: "item name|blocked-on|deferred-to"
# Items marked Status: deferred in BRD or with a BRD-BYPASS are exempt.
declare -a KNOWN_OPEN_ITEMS=(
    "LLM provider abstraction|BRD-05"
    "Agent memory store|BRD-06"
    "Audit trail implementation|BRD-19"
    "Multi-tenant isolation|BRD-20"
    "Notification system|BRD-21"
)

# Check each known open item
for entry in "${KNOWN_OPEN_ITEMS[@]}"; do
    IFS='|' read -r item deferred_to <<< "$entry"

    # Find lines in the BRD that contain this item in a table row (starts with |)
    # This avoids matching the document title or other headings.
    # We use grep -n to get line numbers within table rows only.
    while IFS= read -r linenum; do
        # Skip if ARCH_OK or BRD-BYPASS is on this line or within 3 lines after
        context=$(sed -n "${linenum},$((linenum + 3))p" "$BRD")

        if echo "$context" | grep -qoE 'Status: deferred|BRD-BYPASS'; then
            # Item is properly deferred or bypassed
            continue
        fi

        # Check if a corresponding ADR exists
        adr_slug=$(printf '%s' "$deferred_to" | tr '[:upper:]' '[:lower:]' | sed 's/-//')
        if ls docs/adr/ 2>/dev/null | grep -qiE "$adr_slug"; then
            # Found matching ADR
            continue
        fi

        warn "$BRD" "$linenum" "$item" "$deferred_to"
    done < <(grep -nE "^[[:space:]]*\\|" "$BRD" | grep -E "$item" | cut -d: -f1)
done

if [[ $FOUND_VIOLATIONS -eq 0 ]]; then
    echo "check-brd-open-items: PASS — all BRD open items have tracking artifacts or explicit deferral status"
fi

exit $FOUND_VIOLATIONS
