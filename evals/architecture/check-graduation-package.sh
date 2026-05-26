#!/usr/bin/env bash
# check-graduation-package.sh — Verify all required Phase artifacts are present
#
# Rule: Before the project transitions from Phase N to Phase N+1, all artifacts
#       required for Phase N must exist and be correctly formed.
#
# Usage:
#   PHASE=0 ./check-graduation-package.sh   # Check Phase 0 graduation
#   PHASE=1 ./check-graduation-package.sh   # Check Phase 1 graduation
#   ./check-graduation-package.sh            # Defaults to Phase 0

set -euo pipefail

PHASE="${PHASE:-0}"
FOUND_VIOLATIONS=0

# shellcheck disable=SC2016
warn() {
    printf 'check-graduation-package:%s: RULE: phase-artifact-missing\n' "$2"
    printf '  ARTIFACT: %s\n' "$1"
    printf '  REQUIRED FOR: Phase %s\n' "$PHASE"
    printf '  REMEDIATION: Create the missing artifact before graduating to Phase %s\n' "$((PHASE + 1))"
    FOUND_VIOLATIONS=1
}

# Phase 0 required artifacts
PHASE0_ARTIFACTS=(
    "AGENTS.md:file"
    "STATUS.md:file"
    "docs/adr/0001-record-architecture-decisions.md:file"
    ".gitignore:file"
    ".env.example:file"
    "agent-orchestrator.md:file"
    "contracts/openapi.yaml:file"
    "contracts/events.md:file"
    "specs/feature-flags.md:file"
    "evals:directory"
)

# Phase 1 required artifacts (must also have Phase 0 complete)
PHASE1_ARTIFACTS=(
    "backend:directory"
    "frontend:directory"
)

check_artifact() {
    local artifact="$1"
    local type="$2"

    if [[ "$type" == "file" ]]; then
        if [[ ! -f "$artifact" ]]; then
            warn "$artifact" "required"
            return 1
        fi
        if [[ ! -s "$artifact" ]]; then
            warn "$artifact (empty file)" "required"
            return 1
        fi
    elif [[ "$type" == "directory" ]]; then
        if [[ ! -d "$artifact" ]]; then
            warn "$artifact (directory missing)" "required"
            return 1
        fi
    fi
    return 0
}

echo "check-graduation-package: checking Phase $PHASE graduation package..."

if [[ "$PHASE" == "0" ]]; then
    for entry in "${PHASE0_ARTIFACTS[@]}"; do
        IFS=':' read -r artifact type <<< "$entry"
        check_artifact "$artifact" "$type" || true
    done

    # Phase 0 must not have source code in backend/ or frontend/
    if [[ -d "backend" ]] && [[ -n "$(find backend -type f ! -name '.gitkeep' 2>/dev/null)" ]]; then
        printf 'check-graduation-package: RULE: phase0-source-code-present\n'
        printf '  ARTIFACT: backend/ contains source files\n'
        printf '  REQUIRED FOR: Phase 0 (no source code)\n'
        printf '  REMEDIATION: Remove source files from backend/ — Phase 0 governance only\n'
        FOUND_VIOLATIONS=1
    fi

    if [[ -d "frontend" ]] && [[ -n "$(find frontend -type f ! -name '.gitkeep' 2>/dev/null)" ]]; then
        printf 'check-graduation-package: RULE: phase0-source-code-present\n'
        printf '  ARTIFACT: frontend/ contains source files\n'
        printf '  REQUIRED FOR: Phase 0 (no source code)\n'
        printf '  REMEDIATION: Remove source files from frontend/ — Phase 0 governance only\n'
        FOUND_VIOLATIONS=1
    fi

elif [[ "$PHASE" == "1" ]]; then
    # First check Phase 0 artifacts
    for entry in "${PHASE0_ARTIFACTS[@]}"; do
        IFS=':' read -r artifact type <<< "$entry"
        check_artifact "$artifact" "$type" || true
    done

    # Then check Phase 1 artifacts
    for entry in "${PHASE1_ARTIFACTS[@]}"; do
        IFS=':' read -r artifact type <<< "$entry"
        check_artifact "$artifact" "$type" || true
    done

    # Phase 1 requires go.mod in backend and package.json in frontend
    if [[ -d "backend" ]] && [[ ! -f "backend/go.mod" ]]; then
        warn "backend/go.mod" "required"
    fi

    if [[ -d "frontend" ]] && [[ ! -f "frontend/package.json" ]]; then
        warn "frontend/package.json" "required"
    fi
fi

if [[ $FOUND_VIOLATIONS -eq 0 ]]; then
    echo "check-graduation-package: Phase $PHASE graduation package COMPLETE (PASS)"
fi

exit $FOUND_VIOLATIONS
