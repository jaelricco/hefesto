# Hefesto — developer entrypoints.
# `make up` must yield a seeded, working API with no further steps.

SHELL := /bin/bash
.DEFAULT_GOAL := help

COMPOSE     ?= docker compose
GOOSE_DIR   := db/migrations
PKG         := ./...
DB_URL      ?= $(shell grep -E '^DATABASE_URL=' .env 2>/dev/null | cut -d= -f2-)
LOCAL_DB_URL?= postgres://hefesto:$(shell grep -E '^POSTGRES_PASSWORD=' .env 2>/dev/null | cut -d= -f2-)@localhost:$${POSTGRES_PORT_HOST:-5433}/hefesto?sslmode=disable

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
	$(COMPOSE) exec postgres psql -U $${POSTGRES_USER:-hefesto} -d $${POSTGRES_DB:-hefesto}

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
	@test -d ios/Hefesto || (echo "ios project not scaffolded yet (Phase 5)"; exit 1)
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

## ----------------------------------------------------------------- production

# Production is reached through GitHub Actions, not from a laptop. These
# targets are thin wrappers over `gh` and `ssh` so that the sequence is
# written down rather than remembered.

PROD_HOST ?= $(shell git config --get hefesto.prodhost)
PROD_USER ?= deploy
PROD_DIR  ?= /srv/hefesto
PROD_COMPOSE := docker compose -f $(PROD_DIR)/docker-compose.prod.yml --env-file $(PROD_DIR)/.env.prod

.PHONY: release
release: ## Tag and push a release, which triggers the deploy: make release version=v0.2.0
	@test -n "$(version)" || (echo "usage: make release version=v0.2.0"; exit 1)
	@echo "$(version)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-.+)?$$' || (echo "version must look like v1.2.3"; exit 1)
	@test -z "$$(git status --porcelain)" || (echo "working tree is dirty — commit first"; exit 1)
	git tag -a $(version) -m "release $(version)"
	git push origin $(version)
	@echo "pushed $(version) — watch it with: make deploy-watch"

.PHONY: deploy
deploy: ## Deploy an existing tag (also how you roll back): make deploy tag=v0.1.9
	@test -n "$(tag)" || (echo "usage: make deploy tag=v0.1.9"; exit 1)
	gh workflow run deploy.yml -f tag=$(tag)
	@sleep 3
	$(MAKE) deploy-watch

.PHONY: deploy-watch
deploy-watch: ## Follow the most recent deploy run
	gh run watch $$(gh run list --workflow=deploy.yml --limit 1 --json databaseId --jq '.[0].databaseId')

.PHONY: deploy-check
deploy-check: ## Validate everything the deploy depends on, without deploying
	@# .env.prod.example leaves every secret blank on purpose, and the prod
	@# compose file uses ${VAR:?} so a missing one fails loudly. Fill the blanks
	@# with placeholders just to type-check the file.
	@sed -E 's/^([A-Z_]+)=$$/\1=placeholder/' .env.prod.example > /tmp/hefesto-check.env
	@echo 'HEFESTO_TAG=v0.0.0-check' >> /tmp/hefesto-check.env
	docker compose -f docker-compose.prod.yml --env-file /tmp/hefesto-check.env config --quiet
	@rm -f /tmp/hefesto-check.env
	@command -v shellcheck >/dev/null && shellcheck -e SC1091 scripts/*.sh || echo "shellcheck not installed — skipped"
	@command -v actionlint >/dev/null && actionlint || echo "actionlint not installed — skipped"
	@echo "deploy configuration is valid"

.PHONY: server-bootstrap
server-bootstrap: ## One-time provisioning of a fresh host: make server-bootstrap host=1.2.3.4
	@test -n "$(host)" || (echo "usage: make server-bootstrap host=1.2.3.4"; exit 1)
	ssh root@$(host) 'bash -s' < scripts/bootstrap-server.sh

.PHONY: prod-ps
prod-ps: ## What is running in production
	ssh $(PROD_USER)@$(PROD_HOST) '$(PROD_COMPOSE) ps'

.PHONY: prod-logs
prod-logs: ## Tail production api logs
	ssh -t $(PROD_USER)@$(PROD_HOST) '$(PROD_COMPOSE) logs -f --tail 200 api'

.PHONY: prod-psql
prod-psql: ## psql against the production database (read carefully before typing)
	ssh -t $(PROD_USER)@$(PROD_HOST) '$(PROD_COMPOSE) exec postgres psql -U hefesto -d hefesto'

.PHONY: prod-backup-now
prod-backup-now: ## Force a backup outside the schedule
	ssh $(PROD_USER)@$(PROD_HOST) '$(PROD_COMPOSE) restart backup'

.PHONY: prod-version
prod-version: ## Which tag is live
	ssh $(PROD_USER)@$(PROD_HOST) 'cat $(PROD_DIR)/.deployed-tag'
