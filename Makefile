SHELL := /bin/bash
.DEFAULT_GOAL := help

BIN := bin


.PHONY: help
help: ## List available targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the app
	go build -o $(BIN)/yaa .

.PHONY: run
run: ## Run the app on the host
	go run .

.PHONY: check
check: ## Everything: tidy, vet, test, lint
	go mod tidy -diff
	go vet ./...
	go tool gotestsum --format dots-v2 --format-hide-empty-pkg -- ./...
	golangci-lint run
