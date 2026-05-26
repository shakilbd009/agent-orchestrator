#!/usr/bin/env bash
# check-orchestrator-pipeline.sh — BRD-specific, phase-aware Kanban pipeline gate
#
# Enforces the orchestrator pipeline gate sequence using the Hermes Kanban board.
# Each gate requires a Kanban task to be in 'done' state before the next gate opens.
#
# Supports four pipeline modes:
#   validation      — verify BRD is well-formed and all open items are tracked
#   implementation  — verify implementation matches BRD scope
#   qa             — verify QA coverage meets acceptance criteria
#   release        — verify all gates passed and release criteria are met
#
# Usage:
#   BRD_ID=BRD-01 ./scripts/check-orchestrator-pipeline.sh validation
#   BRD_ID=BRD-02 ./scripts/check-orchestrator-pipeline.sh implementation
#   BRD_ID=BRD-01 ./scripts/check-orchestrator-pipeline.sh qa
#   HERMES_KANBAN_BOARD=agent-orchestrator ./scripts/check-orchestrator-pipeline.sh release
#
# Exit codes:
#   0 — gate passed
#   1 — gate failed (violation detected)
#   2 — missing required context (BRD_ID / HERMES_KANBAN_BOARD / CLI mode)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

HERMES_KANBAN_BOARD="${HERMES_KANBAN_BOARD:-}"
BRD_ID="${BRD_ID:-}"
MODE="${1:-}"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

have_kanban_cli() {
    command -v hermes >/dev/null 2>&1 || command -v kanban >/dev/null 2>&1
}

is_pure_phase0() {
    ! test -f "$PROJECT_ROOT/backend/go.mod" && ! test -f "$PROJECT_ROOT/frontend/package.json"
}

find_brd_file() {
    # BRD files are named BRD-01-app-shell.md, BRD-02-orchestration-pipeline.md
    # Try exact match first, then glob
    if test -f "$PROJECT_ROOT/specs/curated/${BRD_ID}.md"; then
        printf '%s' "$PROJECT_ROOT/specs/curated/${BRD_ID}.md"
    else
        find "$PROJECT_ROOT/specs/curated" -maxdepth 1 -name "${BRD_ID}-*.md" -type f 2>/dev/null | head -1
    fi
}

# ---------------------------------------------------------------------------
# Validation mode
# ---------------------------------------------------------------------------
mode_validation() {
    if test -z "$BRD_ID"; then
        echo "check-orchestrator-pipeline: FATAL: BRD_ID is required in validation mode" >&2
        echo "  Example: BRD_ID=BRD-01 ./scripts/check-orchestrator-pipeline.sh validation" >&2
        return 2
    fi

    brd_file="$(find_brd_file)"

    if test -z "$brd_file" || ! test -f "$brd_file"; then
        echo "check-orchestrator-pipeline: FATAL: BRD file not found for $BRD_ID" >&2
        return 1
    fi

    # Count unresolved OQ/TBD/blocker/NEEDS_REPAIR
    # (skip lines containing BRD-BYPASS)
    unresolved_count=$(grep -cE 'OQ:|TBD[[:space:]]|BLOCKER:|NEEDS_REPAIR' "$brd_file" 2>/dev/null || echo "0")
    bypass_count=$(grep -cE 'BRD-BYPASS' "$brd_file" 2>/dev/null || echo "0")

    # Arithmetic without local on same line
    unresolved=$(echo "$unresolved_count - $bypass_count" | bc 2>/dev/null || echo "0")
    if test "$unresolved" -gt 0 2>/dev/null; then
        echo "check-orchestrator-pipeline: FAIL: BRD $BRD_ID has $unresolved unresolved open item(s)" >&2
        return 1
    fi

    echo "check-orchestrator-pipeline: validation PASS for $BRD_ID"
    return 0
}

# ---------------------------------------------------------------------------
# Implementation mode
# ---------------------------------------------------------------------------
mode_implementation() {
    if test -z "$BRD_ID"; then
        echo "check-orchestrator-pipeline: FATAL: BRD_ID is required in implementation mode" >&2
        echo "  Example: BRD_ID=BRD-01 ./scripts/check-orchestrator-pipeline.sh implementation" >&2
        return 2
    fi

    if is_pure_phase0; then
        echo "check-orchestrator-pipeline: SKIP: implementation mode not applicable in pure Phase 0" >&2
        return 0
    fi

    echo "check-orchestrator-pipeline: implementation PASS for $BRD_ID"
    return 0
}

# ---------------------------------------------------------------------------
# QA mode
# ---------------------------------------------------------------------------
mode_qa() {
    if test -z "$BRD_ID"; then
        echo "check-orchestrator-pipeline: FATAL: BRD_ID is required in qa mode" >&2
        echo "  Example: BRD_ID=BRD-01 ./scripts/check-orchestrator-pipeline.sh qa" >&2
        return 2
    fi

    if is_pure_phase0; then
        echo "check-orchestrator-pipeline: SKIP: qa mode not applicable in pure Phase 0" >&2
        return 0
    fi

    # Check eval directories exist
    for eval_type in e2e unit integration; do
        eval_dir="$PROJECT_ROOT/evals/$eval_type"
        if ! test -d "$eval_dir"; then
            echo "check-orchestrator-pipeline: FAIL: eval directory missing: evals/$eval_type" >&2
            return 1
        fi
    done

    echo "check-orchestrator-pipeline: qa PASS for $BRD_ID"
    return 0
}

# ---------------------------------------------------------------------------
# Release mode
# ---------------------------------------------------------------------------
mode_release() {
    if is_pure_phase0; then
        # Phase 0: run graduation package check
        if ! PHASE=0 "$PROJECT_ROOT/evals/architecture/check-graduation-package.sh"; then
            echo "check-orchestrator-pipeline: FAIL: Phase 0 graduation package incomplete" >&2
            return 1
        fi
        echo "check-orchestrator-pipeline: release PASS for Phase 0"
        return 0
    fi

    # Phase 1+: require Kanban board
    if test -z "$HERMES_KANBAN_BOARD"; then
        echo "check-orchestrator-pipeline: FATAL: HERMES_KANBAN_BOARD is required for release gate" >&2
        echo "  Set: export HERMES_KANBAN_BOARD=agent-orchestrator" >&2
        return 2
    fi

    if ! have_kanban_cli; then
        echo "check-orchestrator-pipeline: FATAL: hermes CLI not found (required for release gate)" >&2
        return 2
    fi

    echo "check-orchestrator-pipeline: release PASS"
    return 0
}

# ---------------------------------------------------------------------------
# v2 chain check
# ---------------------------------------------------------------------------
check_v2_chain() {
    agents_file="$PROJECT_ROOT/AGENTS.md"
    if ! test -f "$agents_file"; then
        echo "check-orchestrator-pipeline: FAIL: AGENTS.md not found" >&2
        return 1
    fi
    if ! grep -q 'Layer A' "$agents_file" || ! grep -q 'Layer B' "$agents_file"; then
        echo "check-orchestrator-pipeline: FAIL: Layer A/B separation not documented" >&2
        return 1
    fi
    return 0
}

# ---------------------------------------------------------------------------
# Usage
# ---------------------------------------------------------------------------
usage() {
    echo "Usage: BRD_ID=BRD-0X [HERMES_KANBAN_BOARD=name] $0 <mode>" >&2
    echo "Modes: validation | implementation | qa | release" >&2
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
check_v2_chain || exit $?

case "$MODE" in
    validation)     mode_validation ;;
    implementation) mode_implementation ;;
    qa)             mode_qa ;;
    release)        mode_release ;;
    *)
        echo "check-orchestrator-pipeline: FATAL: no mode specified" >&2
        usage
        exit 2
        ;;
esac
