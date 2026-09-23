# Lodestar — developer entrypoints.
# `make up` must yield a seeded, working API with no further steps.

SHELL := /bin/bash
.DEFAULT_GOAL := help

COMPOSE     ?= docker compose
GOOSE_DIR   := db/migrations
PKG         := ./...
DB_URL      ?= $(shell grep -E '^DATABASE_URL=' .env 2>/dev/null | cut -d= -f2-)
LOCAL_DB_URL?= postgres://lodestar:$(shell grep -E '^POSTGRES_PASSWORD=' .env 2>/dev/null | cut -d= -f2-)@localhost:$${POSTGRES_PORT_HOST:-5433}/lodestar?sslmode=disable

## ---------------------------------------------------------------- environment

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nTargets:\n"} /^[a-zA-Z0-9_.-]+:.*?##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
	@echo ""

.env: .env.example
	@test -f .env || (cp .env.example .env && echo "created .env from .env.example — fill in the CHANGE_ME values")

.PHONY: up
up: .env ## Start the full stack (postgres, migrate, api, minio, mailpit, pgweb) and seed content
	$(COMPOSE) up -d --build
	$(MAKE) seed
	@echo "api      http://localhost:$${API_PORT_HOST:-8080}/healthz"
	@echo "pgweb    http://localhost:$${PGWEB_PORT_HOST:-8081}"
	@echo "mailpit  http://localhost:$${MAILPIT_UI_PORT_HOST:-8025}"
	@echo "minio    http://localhost:$${MINIO_CONSOLE_PORT_HOST:-9001}"

.PHONY: down
down: ## Stop the stack (keeps volumes)
	$(COMPOSE) down

.PHONY: reset
reset: ## Stop the stack and destroy all volumes, then bring it back up seeded
	$(COMPOSE) down -v
	$(MAKE) up

.PHONY: logs
logs: ## Tail api logs
	$(COMPOSE) logs -f api

.PHONY: psql
psql: ## Open psql against the dev database
	$(COMPOSE) exec postgres psql -U $${POSTGRES_USER:-lodestar} -d $${POSTGRES_DB:-lodestar}

## ----------------------------------------------------------------- migrations

.PHONY: migrate-up
migrate-up: ## Apply all pending migrations
	$(COMPOSE) run --rm migrate goose up

.PHONY: migrate-down
migrate-down: ## Roll back the most recent migration
	$(COMPOSE) run --rm migrate goose down

.PHONY: migrate-status
migrate-status: ## Show migration status
	$(COMPOSE) run --rm migrate goose status

.PHONY: migrate-new
migrate-new: ## Create a new migration: make migrate-new name=add_foo
	@test -n "$(name)" || (echo "usage: make migrate-new name=add_foo"; exit 1)
	goose -dir $(GOOSE_DIR) create $(name) sql

## -------------------------------------------------------------------- codegen

.PHONY: sqlc
sqlc: ## Regenerate typed query code from db/queries
	sqlc generate

.PHONY: gen-openapi
gen-openapi: ## Validate and bundle the OpenAPI spec
	npx --yes @redocly/cli@latest lint api/openapi.yaml
	npx --yes @redocly/cli@latest bundle api/openapi.yaml -o api/openapi.bundled.yaml

.PHONY: gen-ios-client
gen-ios-client: ## Regenerate the Swift client from api/openapi.yaml
	@test -d ios/Lodestar || (echo "ios project not scaffolded yet (Phase 5)"; exit 1)
	cd ios && swift run swift-openapi-generator generate ../api/openapi.yaml

## -------------------------------------------------------------------- content

.PHONY: content-validate
content-validate: ## Schema-check content/, verify slugs, edges and DAG acyclicity
	go run ./cmd/contentlint -dir ./content

.PHONY: seed
seed: ## Idempotent upsert of content/ into the database
	$(COMPOSE) run --rm -e DATABASE_URL="$(DB_URL)" api go run ./cmd/seed -dir ./content

## ----------------------------------------------------------------------- code

.PHONY: build
build: ## Build all binaries into ./bin
	go build -trimpath -o ./bin/api ./cmd/api
	go build -trimpath -o ./bin/seed ./cmd/seed
	go build -trimpath -o ./bin/contentlint ./cmd/contentlint

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run $(PKG)

.PHONY: fmt
fmt: ## Format and tidy
	go fmt $(PKG)
	go mod tidy

.PHONY: test
test: ## Unit tests (no database)
	go test -race -short $(PKG)

.PHONY: test-integration
test-integration: ## Integration tests against a real Postgres via testcontainers
	go test -race -tags=integration -count=1 $(PKG)

.PHONY: cover
cover: ## Coverage report (opens coverage.html)
	go test -race -coverprofile=coverage.out -covermode=atomic $(PKG)
	go tool cover -html=coverage.out -o coverage.html
	@echo "wrote coverage.html"

.PHONY: check
check: lint test content-validate ## What CI runs on every push
