#!/usr/bin/env bash
# doctor.sh — Pre-flight health checks for the local development environment
#
# Checks: Docker, docker-compose, Go, Node, pnpm, GitHub CLI, Hermes Agent.
# Treats port conflicts as warnings (not errors) so the script completes.
#
# Usage:
#   ./scripts/doctor.sh          # all checks
#   ./scripts/doctor.sh --fast  # skip slow checks (docker pull, etc.)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
FAST="${1:-}"

# Track overall health (global)
ISSUES=0

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
section() {
    printf '\n=== %s ===\n' "$1"
}

check_tool() {
    local name="$1"
    local cmd="$2"
    local version_flag="${3:-}"

    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "  [MISSING] $name"
        echo "  Install or add to PATH: $cmd" >&2
        ((ISSUES++))
        return 1
    fi

    local version
    if [[ -n "$version_flag" ]]; then
        version=$("$cmd" $version_flag 2>/dev/null) || version="(unknown)"
    else
        version=$("$cmd" --version 2>/dev/null | head -1) || version="(unknown)"
    fi
    echo "  [OK] $name — $version"
    return 0
}

check_port() {
    local port="$1"
    local service="$2"

    # Capture to variable — avoids set -e pitfall in command substitution
    local result
    result=$(lsof -ti:"$port" 2>/dev/null || true)

    if [[ -n "$result" ]]; then
        echo "  [WARN] Port $port ($service) is already in use"
        echo "  Process PIDs: $result" >&2
        echo "  This is a warning — stop the service or choose a different port." >&2
        # NOT an error — port conflicts are warnings only
        return 0
    fi

    echo "  [OK] Port $port ($service) is free"
    return 0
}

# ---------------------------------------------------------------------------
# Sections as functions (local vars are valid inside functions)
# ---------------------------------------------------------------------------
check_docker() {
    section "Docker"
    if ! command -v docker >/dev/null 2>&1; then
        echo "  [MISSING] docker"
        echo "  Install Docker Desktop or Docker Engine." >&2
        ((ISSUES++))
        return
    fi

    local docker_version
    docker_version=$(docker --version 2>/dev/null | head -1) || docker_version="(unknown)"
    echo "  [OK] docker — $docker_version"

    # Check docker daemon is reachable
    if ! docker info >/dev/null 2>&1; then
        echo "  [WARN] docker daemon may not be running — 'docker info' failed" >&2
        echo "  Start Docker Desktop or dockerd." >&2
    fi
}

check_docker_compose() {
    section "Docker Compose"
    if ! command -v docker-compose >/dev/null 2>&1; then
        echo "  [MISSING] docker-compose (standalone v1)"
        echo "  This project requires docker-compose (NOT 'docker compose' v2 plugin)." >&2
        echo "  Install: https://docs.docker.com/compose/install/" >&2
        ((ISSUES++))
        return
    fi

    local dc_version
    dc_version=$(docker-compose --version 2>/dev/null | head -1) || dc_version="(unknown)"
    echo "  [OK] docker-compose — $dc_version"
}

check_docker_services() {
    section "Docker services (docker-compose)"
    if [[ ! -f "$PROJECT_ROOT/docker-compose.yml" ]]; then
        echo "  [WARN] docker-compose.yml not found in project root"
        return
    fi

    echo "  docker-compose.yml found"

    if grep -qE 'postgres|redis' "$PROJECT_ROOT/docker-compose.yml" 2>/dev/null; then
        echo "  [OK] Phase 0 services (postgres, redis) defined"
    fi

    if command -v docker-compose >/dev/null 2>&1; then
        local config_ok
        config_ok=$(docker-compose config --quiet 2>&1) || true
        if [[ -z "$config_ok" ]]; then
            echo "  [OK] docker-compose.yml is valid"
        else
            echo "  [WARN] docker-compose.yml has issues: $config_ok" >&2
        fi
    fi
}

check_go() {
    section "Go"
    if ! command -v go >/dev/null 2>&1; then
        echo "  [MISSING] go"
        echo "  Install Go 1.25 or later: https://go.dev/dl/" >&2
        ((ISSUES++))
        return
    fi

    local go_version
    go_version=$(go version 2>/dev/null | head -1) || go_version="(unknown)"
    echo "  [OK] go — $go_version"

    local go_minor
    go_minor=$(go version 2>/dev/null | sed -E 's/.*go1\.([0-9]+).*/\1/' | head -1) || go_minor="0"
    if [[ "$go_minor" -lt 25 ]]; then
        echo "  [WARN] Go 1.25+ recommended; found 1.$go_minor" >&2
    fi
}

check_node_npm() {
    section "Node.js and npm"
    if ! command -v node >/dev/null 2>&1; then
        echo "  [MISSING] node"
        echo "  Install Node.js v25 or later: https://nodejs.org/" >&2
        ((ISSUES++))
    else
        local node_version
        node_version=$(node --version 2>/dev/null) || node_version="(unknown)"
        echo "  [OK] node — $node_version"
    fi

    if ! command -v npm >/dev/null 2>&1; then
        echo "  [MISSING] npm"
        ((ISSUES++))
    else
        local npm_version
        npm_version=$(npm --version 2>/dev/null) || npm_version="(unknown)"
        echo "  [OK] npm — $npm_version"
    fi
}

check_pnpm() {
    section "pnpm"
    if ! command -v pnpm >/dev/null 2>&1; then
        echo "  [MISSING] pnpm"
        echo "  Install: npm install -g pnpm" >&2
        ((ISSUES++))
        return
    fi

    local pnpm_version
    pnpm_version=$(pnpm --version 2>/dev/null) || pnpm_version="(unknown)"
    echo "  [OK] pnpm — $pnpm_version"
}

check_gh() {
    section "GitHub CLI"
    if ! command -v gh >/dev/null 2>&1; then
        echo "  [MISSING] gh (GitHub CLI)"
        echo "  Install: https://cli.github.com/" >&2
        ((ISSUES++))
        return
    fi

    local gh_version
    gh_version=$(gh --version 2>/dev/null | head -1) || gh_version="(unknown)"
    echo "  [OK] gh — $gh_version"

    if gh auth status 2>&1 | grep -qi "not logged in"; then
        echo "  [WARN] gh is not authenticated — run 'gh auth login' for CI/GitHub API access" >&2
    else
        echo "  [OK] gh authenticated"
    fi
}

check_hermes() {
    section "Hermes Agent"
    if ! command -v hermes >/dev/null 2>&1; then
        echo "  [WARN] hermes CLI not found in PATH" >&2
        echo "  Install: see https://hermes-agent.nousresearch.com/docs" >&2
        # Not a hard failure — Hermes may not be required for all workflows
        return
    fi

    local hermes_version
    hermes_version=$(hermes --version 2>/dev/null | head -1) || hermes_version="(unknown)"
    echo "  [OK] hermes — $hermes_version"
}

check_ports() {
    section "Port availability"
    check_port 5432 "postgres"
    check_port 6379 "redis"
    check_port 8080 "backend (Go/Echo)"
    check_port 5173 "frontend (SvelteKit)"
}

check_project_structure() {
    section "Project structure"
    if [[ -f "$PROJECT_ROOT/AGENTS.md" ]]; then
        echo "  [OK] AGENTS.md"
    else
        echo "  [MISSING] AGENTS.md" >&2
        ((ISSUES++))
    fi

    if [[ -f "$PROJECT_ROOT/STATUS.md" ]]; then
        echo "  [OK] STATUS.md"
    else
        echo "  [MISSING] STATUS.md" >&2
        ((ISSUES++))
    fi

    if [[ -f "$PROJECT_ROOT/.env.example" ]]; then
        echo "  [OK] .env.example"
    else
        echo "  [MISSING] .env.example" >&2
        ((ISSUES++))
    fi

    if [[ -f "$PROJECT_ROOT/docker-compose.yml" ]]; then
        echo "  [OK] docker-compose.yml"
    else
        echo "  [WARN] docker-compose.yml not found" >&2
    fi
}

check_slow() {
    section "Slow checks (docker pull)"
    if command -v docker >/dev/null 2>&1; then
        if docker info >/dev/null 2>&1; then
            echo "  Docker daemon is running — image pulls should work"
        else
            echo "  [WARN] Docker daemon not reachable" >&2
        fi
    fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
echo "doctor.sh: running pre-flight health checks..."
echo "Project: $PROJECT_ROOT"

check_docker
check_docker_compose
check_docker_services
check_go
check_node_npm
check_pnpm
check_gh
check_hermes
check_ports
check_project_structure

if [[ "$FAST" != "--fast" ]]; then
    check_slow
fi

section "Summary"
echo "Issues found: $ISSUES"

if [[ $ISSUES -eq 0 ]]; then
    echo "doctor.sh: ALL CHECKS PASSED"
    exit 0
else
    echo "doctor.sh: $ISSUES issue(s) found — review above output" >&2
    exit 1
fi
