SHELL := /bin/bash
.DEFAULT_GOAL := help

BIN := bin


.PHONY: help
help: ## List available targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the app
	go build -o $(BIN)/yaa ./cmd/

.PHONY: run
run: ## Run the app on the host
	go run ./cmd run

.PHONY: check-no-lint
check-no-lint: ## tidy, vet, test, without lint - for CI
	go mod tidy -diff
	go vet ./...
	go tool gotestsum --format dots-v2 --format-hide-empty-pkg -- ./...

.PHONY: check
check: ## Everything: tidy, vet, test, lint
	go mod tidy -diff
	go vet ./...
	go tool gotestsum --format dots-v2 --format-hide-empty-pkg -- ./...
	golangci-lint run

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
