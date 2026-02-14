.PHONY: build test clean install release deps fmt lint help

# Variables
BINARY_NAME=gocloud
BUILD_DIR=bin
DIST_DIR=dist
VERSION=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD)

# Default target
.DEFAULT_GOAL := help

# Help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Dependencies
deps: ## Download and tidy dependencies
	go mod download
	go mod tidy

# Build for development (all platforms)
build: ## Build for all platforms
	@echo "Building $(BINARY_NAME) for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)" -o $(BUILD_DIR)/gocloud main.go
	
	# Windows AMD64 (Intel)
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)" -o $(BUILD_DIR)/gocloud-win.exe main.go
	
	# Linux AMD64 (Intel)
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)" -o $(BUILD_DIR)/gocloud-linux main.go
	
	@echo "Build complete:"
	@echo "  macOS ARM64: $(BUILD_DIR)/gocloud"
	@echo "  Windows Intel: $(BUILD_DIR)/gocloud-win.exe"
	@echo "  Linux Intel: $(BUILD_DIR)/gocloud-linux"

# Build for current platform only
build-current: ## Build for current platform only
	@echo "Building $(BINARY_NAME) for current platform..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)" -o $(BUILD_DIR)/$(BINARY_NAME) main.go
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for release (cross-platform)
release: ## Build for all platforms
	@echo "Building $(BINARY_NAME) for all platforms..."
	@mkdir -p $(DIST_DIR)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)" -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 main.go
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)" -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 main.go
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 main.go
	
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 main.go
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)" -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe main.go
	
	@echo "Release builds complete in $(DIST_DIR)/"

# Testing
test: ## Run tests
	go test -v ./...

# Testing with coverage
test-coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Linting
lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --no-config --enable=errcheck,staticcheck,unused,govet; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Linting with auto-fix
lint-fix: ## Run linter with auto-fix
	@echo "Running linter with auto-fix..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --no-config --fix --enable=errcheck,staticcheck,unused,govet; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Dependency check
deps-check: ## Check for outdated dependencies
	go list -u -m all

# Format code
fmt: ## Format code
	go fmt ./...

# Install locally
install: ## Install locally
	go install .

# Clean build artifacts
clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR) $(DIST_DIR) coverage.out coverage.html

# Run the CLI
run: build ## Build and run the CLI
	./$(BUILD_DIR)/$(BINARY_NAME)

# Demo
demo: build ## Run demo
	./$(BUILD_DIR)/$(BINARY_NAME) config init test-project

# Show version
version: build ## Show version information
	./$(BUILD_DIR)/$(BINARY_NAME) --version

# Show help
cli-help: build ## Show CLI help
	./$(BUILD_DIR)/$(BINARY_NAME) --help

# Development setup
dev-setup: deps fmt lint test ## Setup development environment
	@echo "Development environment setup complete"

# Pre-commit checks
pre-commit: fmt lint test ## Run pre-commit checks
	@echo "Pre-commit checks passed"

# All quality checks
quality: fmt lint test deps-check ## Run all quality checks
	@echo "All quality checks passed"

# Docker build (if needed)
docker-build: ## Build Docker image
	docker build -t gocloud-cli:$(VERSION) .

# Docker run (if needed)
docker-run: ## Run in Docker
	docker run --rm gocloud-cli:$(VERSION)
