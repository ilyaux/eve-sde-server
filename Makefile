.PHONY: help run build test migrate migrate-down clean docker docker-run download-sde

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

run: ## Run the server locally
	@mkdir -p data
	@go run cmd/server/main.go

build: ## Build the binary
	@mkdir -p bin
	@go build -o bin/eve-sde-server cmd/server/main.go
	@echo "Binary built: bin/eve-sde-server"

test: ## Run tests
	@go test -v -race ./...

test-coverage: ## Run tests with coverage
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

bench: ## Run benchmarks
	@go test -bench=. -benchmem ./internal/repository

migrate: ## Run database migrations
	@mkdir -p data
	@go run cmd/migrate/main.go

import-sde: ## Download and import full EVE SDE
	@go run cmd/import-sde/main.go

import-sde-skip: ## Import SDE from existing files (skip download)
	@go run cmd/import-sde/main.go -skip-download

migrate-down: ## Rollback last migration
	@goose -dir internal/database/migrations sqlite3 data/sde.db down

migrate-status: ## Show migration status
	@goose -dir internal/database/migrations sqlite3 data/sde.db status

clean: ## Clean build artifacts and data
	@rm -rf bin/ data/ coverage.out coverage.html
	@echo "Cleaned build artifacts"

docker: ## Build Docker image
	@docker build -t eve-sde-server .

docker-run: ## Run Docker container
	@docker run -p 8080:8080 -v $$(pwd)/data:/app/data eve-sde-server

docker-compose-up: ## Start with docker-compose
	@docker-compose up --build -d

docker-compose-down: ## Stop docker-compose
	@docker-compose down

download-sde: ## Download SDE from CCP (requires curl/wget)
	@mkdir -p data/sde
	@echo "Downloading SDE (this may take a while, ~400MB)..."
	@curl -L -o data/sde.zip https://eve-static-data-export.s3-eu-west-1.amazonaws.com/tranquility/sde.zip
	@echo "Extracting..."
	@unzip -q data/sde.zip -d data/sde
	@rm data/sde.zip
	@echo "SDE downloaded to data/sde/"

fmt: ## Format Go code
	@go fmt ./...
	@echo "Code formatted"

lint: ## Run linter (requires golangci-lint)
	@golangci-lint run

mod-tidy: ## Tidy Go modules
	@go mod tidy
	@echo "Modules tidied"

install-tools: ## Install development tools
	@go install github.com/pressly/goose/v3/cmd/goose@latest
	@echo "Tools installed"
