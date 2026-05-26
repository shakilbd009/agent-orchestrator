.PHONY: help backend frontend test test-unit test-integration test-e2e eval-arch check-orchestrator-pipeline

# Makefile for agent-orchestrator
#
# Phase 0: governance and scaffolding only — no source code.
# Phase 1+: adds app shell targets.
#
# Docker: uses docker-compose (standalone v1), NOT "docker compose" v2 plugin.
#
# Usage:
#   make help
#   make doctor
#   make init-eval
#   make check-status
#   make check-pipeline
#   make seed
#   make seed-reset
#   make docker-up
#   make docker-down
#   make docker-logs
#   make backend        (Phase 1+, skips gracefully if backend/ is empty)
#   make frontend       (Phase 1+, skips gracefully if frontend/ is empty)
#   make test
#   make test-unit
#   make test-integration
#   make test-e2e

PROJECT_ROOT := $(dir $(lastword $(MAKEFILE_LIST)))
SCRIPTS_DIR := $(PROJECT_ROOT)/scripts
DOCKER_COMPOSE := docker-compose

# Phase detection: backend/frontend have source if they have non-gitkeep files
# POSIX-compatible: uses test and find, not bash [[ ]]
BACKEND_HAS_SOURCE := $(shell test -d backend && test -n "$$(find backend -type f ! -name .gitkeep 2>/dev/null)" && echo yes || echo no)
FRONTEND_HAS_SOURCE := $(shell test -d frontend && test -n "$$(find frontend -type f ! -name .gitkeep 2>/dev/null)" && echo yes || echo no)

# ---------------------------------------------------------------------------
# Help
# ---------------------------------------------------------------------------
help: ## Show this help message
	@echo "agent-orchestrator Makefile targets:"
	@echo ""
	@echo "  make help              Show this help"
	@echo "  make doctor            Pre-flight health checks"
	@echo "  make init-eval         Initialise eval environment and run static checks"
	@echo "  make check-status      Run status-sync check (spec/eval/flag parity)"
	@echo "  make eval-arch        Run architecture checks (Phase-gated)"
	@echo "  make check-pipeline    Run orchestrator pipeline gate"
	@echo "  make seed              Start docker services (postgres, redis)"
	@echo "  make seed-reset        Reset and reseed docker services"
	@echo "  make docker-up          Start docker-compose services"
	@echo "  make docker-down        Stop docker-compose services"
	@echo "  make docker-logs       Show docker-compose logs"
	@echo "  make backend           Build Go backend (Phase 1+, skips if empty)"
	@echo "  make frontend          Build SvelteKit frontend (Phase 1+, skips if empty)"
	@echo "  make test              Run all tests (Phase 1+)"
	@echo "  make test-unit         Run unit tests (Phase 1+)"
	@echo "  make test-integration  Run integration tests (Phase 1+)"
	@echo "  make test-e2e          Run E2E tests (Phase 1+, requires dev server)"

# ---------------------------------------------------------------------------
# Phase 0 scripts
# ---------------------------------------------------------------------------
doctor: ## Pre-flight health checks
	@echo "Running doctor.sh..."
	@"$(SCRIPTS_DIR)/doctor.sh"

init-eval: ## Initialise eval environment and run static checks
	@echo "Running init-eval.sh..."
	@"$(SCRIPTS_DIR)/init-eval.sh"

check-status: ## Run status-sync check
	@echo "Running check-status-sync.sh..."
	@"$(SCRIPTS_DIR)/check-status-sync.sh"

eval-arch: ## Run architecture checks (Phase-gated, SKIP in Phase 0)
	@echo "Running init-eval.sh (architecture checks)..."
	@"$(SCRIPTS_DIR)/init-eval.sh"

check-pipeline: ## Run orchestrator pipeline gate
	@echo "Running check-orchestrator-pipeline.sh..."
	@"$(SCRIPTS_DIR)/check-orchestrator-pipeline.sh" release

seed: ## Start docker services (postgres, redis)
	@echo "Starting docker services via seed-dev.sh..."
	@"$(SCRIPTS_DIR)/seed-dev.sh"

seed-reset: ## Reset and reseed docker services
	@echo "Resetting docker services via seed-dev.sh --reset..."
	@"$(SCRIPTS_DIR)/seed-dev.sh" --reset

docker-up: ## Start docker-compose services
	@echo "Starting docker-compose services..."
	$(DOCKER_COMPOSE) up -d

docker-down: ## Stop docker-compose services
	$(DOCKER_COMPOSE) down

docker-logs: ## Show docker-compose logs
	$(DOCKER_COMPOSE) logs -f

# ---------------------------------------------------------------------------
# Phase 1+ targets — skip gracefully when backend/frontend are empty
# ---------------------------------------------------------------------------
backend: ## Build Go backend (Phase 1+)
	@if [ "$(BACKEND_HAS_SOURCE)" = "yes" ]; then \
		echo "Building Go backend..."; \
		cd backend && go build -o bin/server ./...; \
	else \
		echo "make backend: backend/ is empty (Phase 0) — skipping"; \
	fi

frontend: ## Build SvelteKit frontend (Phase 1+)
	@if [ "$(FRONTEND_HAS_SOURCE)" = "yes" ]; then \
		echo "Building SvelteKit frontend..."; \
		cd frontend && pnpm run build; \
	else \
		echo "make frontend: frontend/ is empty (Phase 0) — skipping"; \
	fi

test: test-unit test-integration ## Run all tests (Phase 1+)

test-unit: ## Run unit tests (Phase 1+)
	@if [ "$(BACKEND_HAS_SOURCE)" = "yes" ] && [ -n "$$(find backend -type f -name '*_test.go' 2>/dev/null)" ]; then \
		echo "Running Go unit tests..."; \
		cd backend && go test ./...; \
	else \
		echo "make test-unit: no Go unit tests found — skipping"; \
	fi

test-integration: ## Run integration tests (Phase 1+)
	@if [ "$(BACKEND_HAS_SOURCE)" = "yes" ] && [ -n "$$(find backend -type f -name '*_integration_test.go' 2>/dev/null)" ]; then \
		echo "Running integration tests..."; \
		cd backend && go test ./...; \
	else \
		echo "make test-integration: no integration tests found — skipping"; \
	fi

test-e2e: ## Run E2E tests (Phase 1+, requires dev server)
	@if [ "$(FRONTEND_HAS_SOURCE)" = "yes" ] && [ -f frontend/package.json ]; then \
		echo "Running E2E tests..."; \
		cd frontend && pnpm exec playwright test; \
	else \
		echo "make test-e2e: frontend not initialized — skipping"; \
	fi
