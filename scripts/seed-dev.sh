#!/usr/bin/env bash
# seed-dev.sh — Seed the local development environment
#
# Runs docker-compose to start Phase 0 services (postgres, redis) and optionally
# initialises the database schema if backend source is present.
#
# Phase 0: starts postgres and redis only.
# Phase 1+: also runs database migrations if backend is present.
#
# Usage:
#   ./scripts/seed-dev.sh           # start services
#   ./scripts/seed-dev.sh --reset   # destroy volumes and re-seed

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
RESET="${1:-}"

# Resolve to project root
cd "$PROJECT_ROOT"

echo "seed-dev: seeding local development environment..."

# ---------------------------------------------------------------------------
# Phase 0: just start postgres + redis via docker-compose
# ---------------------------------------------------------------------------
start_phase0_services() {
    if [[ ! -f "$PROJECT_ROOT/docker-compose.yml" ]]; then
        echo "seed-dev: docker-compose.yml not found — skipping service start" >&2
        return 0
    fi

    echo "seed-dev: pulling images..."
    if ! docker-compose pull 2>&1 | tail -5; then
        echo "seed-dev: WARNING: docker-compose pull had issues" >&2
    fi

    echo "seed-dev: starting Phase 0 services (postgres, redis)..."
    docker-compose up -d

    echo "seed-dev: waiting for services to be healthy..."
    local max_wait=30
    local count=0

    # Wait for postgres
    while [[ $count -lt $max_wait ]]; do
        if docker-compose exec -T postgres pg_isready -U agentorch >/dev/null 2>&1; then
            echo "seed-dev: postgres is ready"
            break
        fi
        ((count++))
        sleep 1
    done

    if [[ $count -eq $max_wait ]]; then
        echo "seed-dev: WARNING: postgres did not become ready in ${max_wait}s" >&2
    fi

    # Wait for redis
    count=0
    while [[ $count -lt $max_wait ]]; do
        if docker-compose exec -T redis redis-cli ping 2>/dev/null | grep -q PONG; then
            echo "seed-dev: redis is ready"
            break
        fi
        ((count++))
        sleep 1
    done

    if [[ $count -eq $max_wait ]]; then
        echo "seed-dev: WARNING: redis did not become ready in ${max_wait}s" >&2
    fi

    echo "seed-dev: Phase 0 services started"
    echo "seed-dev:  postgres: localhost:5432 (user: agentorch)"
    echo "seed-dev:  redis:    localhost:6379"
}

# ---------------------------------------------------------------------------
# Reset: destroy volumes and re-seed
# ---------------------------------------------------------------------------
reset_environment() {
    echo "seed-dev: RESET requested — destroying volumes..."
    docker-compose down -v 2>/dev/null || true
    echo "seed-dev: volumes destroyed"
    start_phase0_services
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
if [[ "$RESET" == "--reset" ]]; then
    reset_environment
else
    # Check if services are already running
    if docker-compose ps 2>/dev/null | grep -q 'Up'; then
        echo "seed-dev: services already running — skipping start"
        echo "seed-dev: run with --reset to restart from scratch"
    else
        start_phase0_services
    fi
fi

echo "seed-dev: DONE"
echo ""
echo "Services:"
docker-compose ps 2>/dev/null || echo "  (docker-compose not available)"
