.PHONY: build test lint fmt vet clean run install generate help
.PHONY: test-domain test-infra test-app test-pkg policy-test
.PHONY: build-linux build-darwin build-windows build-all
.PHONY: docker-build docker-run docker-up docker-down docker-logs
.PHONY: demo-up demo-down demo-reset
.PHONY: security tools
.PHONY: release-plan release-notes release-validate

# Build variables
BINARY_NAME=bridge
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Directories
CMD_DIR=./cmd/bridge
BUILD_DIR=./bin

# Default target
all: check build

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

install: build ## Install binary to GOPATH/bin
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

run: build ## Build and run
	$(BUILD_DIR)/$(BINARY_NAME)

# Testing targets
test: ## Run tests
	$(GOTEST) -v -race -cover ./...

test-coverage: ## Run tests with coverage report
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-unit: ## Run unit tests only
	$(GOTEST) -v -short ./...

test-integration: ## Run integration tests
	$(GOTEST) -v -run Integration ./...

test-domain: ## Run domain layer tests
	$(GOTEST) -v ./internal/domain/...

test-infra: ## Run infrastructure layer tests
	$(GOTEST) -v ./internal/infrastructure/...

test-app: ## Run application layer tests
	$(GOTEST) -v ./internal/application/...

test-pkg: ## Run pkg tests
	$(GOTEST) -v ./pkg/...

# Code quality
lint: ## Run linter
	@which golangci-lint > /dev/null 2>&1 || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

fmt: ## Format code
	$(GOFMT) -s -w .
	@which goimports > /dev/null 2>&1 || go install golang.org/x/tools/cmd/goimports@latest
	goimports -w .

vet: ## Run go vet
	$(GOVET) ./...

check: fmt vet lint test ## Run all checks

# Policy testing
policy-test: ## Run OPA policy tests
	@which opa > /dev/null 2>&1 || echo "OPA not installed. Install from https://www.openpolicyagent.org/docs/latest/#1-download-opa"
	opa test ./policies

# Cleanup
clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Dependencies
deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

deps-update: ## Update dependencies
	$(GOCMD) get -u ./...
	$(GOMOD) tidy

deps-verify: ## Verify dependencies
	$(GOMOD) verify

# Code generation
generate: sqlc ## Run code generation (sqlc, etc.)
	@echo "Running code generation..."
	$(GOCMD) generate ./...

sqlc: ## Generate sqlc code
	@which sqlc > /dev/null 2>&1 || (echo "Installing sqlc..." && go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest)
	sqlc generate

# Cross-platform builds
build-all: build-linux build-darwin build-windows ## Build for all platforms

build-linux: ## Build for Linux
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)

build-darwin: ## Build for macOS
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)

build-windows: ## Build for Windows
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)

# Development helpers
dev: ## Run in development mode with hot reload (requires air)
	@which air > /dev/null 2>&1 || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	air

watch-test: ## Watch and run tests on change (requires watchexec)
	watchexec -e go -- make test-unit

# Docker targets
docker-build: ## Build Docker image
	docker build -t $(BINARY_NAME):$(VERSION) .

docker-run: ## Run Docker container
	docker run --rm -it $(BINARY_NAME):$(VERSION)

docker-up: ## Start development environment
	docker-compose up -d

docker-down: ## Stop development environment
	docker-compose down

docker-logs: ## View container logs
	docker-compose logs -f

# Demo environment
demo-up: ## Start demo environment with seeded data
	docker-compose -f docker-compose.demo.yml up -d

demo-down: ## Stop demo environment
	docker-compose -f docker-compose.demo.yml down

demo-reset: ## Reset demo environment (fresh data)
	docker-compose -f docker-compose.demo.yml down -v
	docker-compose -f docker-compose.demo.yml up -d

# Security scanning
security: ## Run security scans
	@which gosec > /dev/null 2>&1 || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	@which govulncheck > /dev/null 2>&1 || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	gosec -severity=high ./...
	govulncheck ./...

# Tool installation
tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/air-verse/air@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "All tools installed successfully"

# Release (via relicta)
release-plan: ## Plan next release
	@which relicta > /dev/null 2>&1 || (echo "Installing relicta..." && go install github.com/felixgeelhaar/relicta@latest)
	relicta plan --analyze

release-notes: ## Generate release notes
	@which relicta > /dev/null 2>&1 || (echo "Installing relicta..." && go install github.com/felixgeelhaar/relicta@latest)
	relicta notes --audience developers

release-validate: ## Validate release requirements
	@which relicta > /dev/null 2>&1 || (echo "Installing relicta..." && go install github.com/felixgeelhaar/relicta@latest)
	relicta validate
