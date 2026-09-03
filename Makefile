SHELL := /bin/bash
.DEFAULT_GOAL := help

BIN := bin


.PHONY: help
help: ## List available targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

## ---------------------------------------------------------------- codegen

.PHONY: generate
generate: ## Generate the database code from the schema and the queries
	go tool sqlc generate

## ---------------------------------------------------------------- build & run

.PHONY: build
build: generate ## Build the app
	go build -o $(BIN)/yaa ./cmd/

.PHONY: run
run: generate ## Run the app on the host
	go run ./cmd run

.PHONY: dev
dev: generate ## Migrate, then run the app (used inside the dev container)
	go run ./cmd migrate up
	go run ./cmd run

## ---------------------------------------------------------------- quality

.PHONY: check-no-lint
check-no-lint: generate ## tidy, vet, test, without lint - for CI
	go mod tidy -diff
	go vet ./...
	go tool gotestsum --format dots-v2 --format-hide-empty-pkg -- ./...

.PHONY: check
check: generate ## Everything: tidy, vet, test, lint
	go mod tidy -diff
	go vet ./...
	go tool gotestsum --format dots-v2 --format-hide-empty-pkg -- ./...
	golangci-lint run

## ---------------------------------------------------------------- database

.PHONY: migrate
migrate: generate ## Apply pending database migrations
	go run ./cmd migrate up

.PHONY: migrate-down
migrate-down: generate ## Roll back the most recent migration
	go run ./cmd migrate down

.PHONY: migrate-status
migrate-status: generate ## Show which migrations are applied
	go run ./cmd migrate status

.PHONY: migration
migration: ## Scaffold a migration: make migration name=add_reminders
	@test -n "$(name)" || { echo "usage: make migration name=add_reminders" >&2; exit 1; }
	go tool goose -dir migrations/sql create $(name) sql

.PHONY: psql
psql: ## Open a psql shell on the dev database
	docker compose exec db sh -c 'exec psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

## ---------------------------------------------------------------- docker

.env:
	@cp .env.example .env
	@echo ">> wrote .env from .env.example — fill in TG_TOKEN and PUBLIC_HTTP_HOST"

.PHONY: up
up: .env ## Run the app in docker
	docker compose up --build -d
	@echo ">> http://localhost:$$(grep -E '^HTTP_PORT=' .env | cut -d= -f2)"

.PHONY: down
down: ## Stop the docker stack
	docker compose down

.PHONY: logs
logs: ## Follow the app logs
	docker compose logs -f app

.PHONY: clean
clean: ## Remove the build output and the generated files
	rm -f $(BIN)/yaa
	find . -name '*_sqlc.go' -delete
