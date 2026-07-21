.PHONY: help up down migrate migrate-down run build test test-unit test-integration \
        bench lint fmt vet mocks tidy logs ps clean

APP_NAME       := backend-security-playground
CMD_DIR        := ./cmd/api
BIN_DIR        := ./bin
DATABASE_URL   ?= postgres://playground:playground@localhost:5432/playground?sslmode=disable
MIGRATIONS_DIR := ./migrations

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

up: ## Start docker compose stack (postgres, redis, app)
	docker compose -f docker/docker-compose.yml up -d --build

down: ## Stop docker compose stack
	docker compose -f docker/docker-compose.yml down

logs: ## Tail docker compose logs
	docker compose -f docker/docker-compose.yml logs -f

ps: ## Show docker compose container status
	docker compose -f docker/docker-compose.yml ps

migrate: ## Apply all up migrations
	go run ./cmd/migrate -database "$(DATABASE_URL)" -path $(MIGRATIONS_DIR) up

migrate-down: ## Roll back the last migration
	go run ./cmd/migrate -database "$(DATABASE_URL)" -path $(MIGRATIONS_DIR) down 1

run: ## Run the API server locally
	go run $(CMD_DIR)

build: ## Build the API binary
	go build -o $(BIN_DIR)/api $(CMD_DIR)

test: ## Run all tests with race detector and coverage
	go test -race -covermode=atomic -coverprofile=coverage.out ./...

test-unit: ## Run unit tests only (skip integration-tagged tests)
	go test -race -short ./...

test-integration: ## Run integration tests (requires docker services)
	go test -race -tags=integration ./...

bench: ## Run benchmarks
	go test -bench=. -benchmem -run=^$$ ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Format code
	gofmt -s -w .
	go vet ./...

vet: ## Run go vet
	go vet ./...

mocks: ## Regenerate mocks via mockery
	mockery

tidy: ## Tidy go.mod/go.sum
	go mod tidy

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) coverage.out coverage.html
