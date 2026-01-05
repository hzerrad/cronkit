.PHONY: help build install clean test test-unit test-integration test-e2e test-bdd test-coverage test-watch lint fmt run dev build-all setup-hooks vet benchmark benchmark-compare test-large profile examples docs dev-setup

# Variables
BINARY_NAME=cronic
MAIN_PATH=./cmd/cronic
BUILD_DIR=./bin
DIST_DIR=./dist

# Build information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Linker flags
LDFLAGS=-ldflags "-X github.com/hzerrad/cronic/internal/cmd.version=$(VERSION) \
                  -X github.com/hzerrad/cronic/internal/cmd.commit=$(COMMIT) \
                  -X github.com/hzerrad/cronic/internal/cmd.date=$(DATE)"

help: ## Display this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\n"} /^[a-zA-Z0-9_-]+:.*##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

install: ## Install the binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) $(MAIN_PATH)
	@echo "Installation complete"

clean: ## Remove build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@go clean
	@echo "Clean complete"

test: ## Run all tests (unit + BDD)
	@echo "Running all tests..."
	@$(MAKE) test-unit
	@$(MAKE) test-bdd

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	go test -v -race -short ./internal/...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@which ginkgo > /dev/null || (echo "Installing ginkgo..." && go install github.com/onsi/ginkgo/v2/ginkgo@latest)
	ginkgo -v -race ./test/integration/...

test-e2e: ## Run end-to-end tests
	@echo "Running E2E tests..."
	@which ginkgo > /dev/null || (echo "Installing ginkgo..." && go install github.com/onsi/ginkgo/v2/ginkgo@latest)
	ginkgo -v ./test/e2e/...

test-bdd: ## Run all BDD tests (integration + e2e)
	@echo "Running BDD tests..."
	@$(MAKE) test-integration
	@$(MAKE) test-e2e

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@mkdir -p $(BUILD_DIR)
	go test -v -race -coverprofile=$(BUILD_DIR)/coverage.out ./internal/...
	go tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "Coverage report generated: $(BUILD_DIR)/coverage.html"
	@go tool cover -func=$(BUILD_DIR)/coverage.out | tail -1

test-watch: ## Run tests in watch mode (requires ginkgo)
	@echo "Running tests in watch mode..."
	@which ginkgo > /dev/null || (echo "Installing ginkgo..." && go install github.com/onsi/ginkgo/v2/ginkgo@latest)
	ginkgo watch -r ./...

benchmark: ## Run all benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./internal/...

benchmark-compare: ## Compare benchmark results (requires benchstat)
	@echo "Running benchmarks for comparison..."
	@which benchstat > /dev/null || (echo "Installing benchstat..." && go install golang.org/x/perf/cmd/benchstat@latest)
	go test -bench=. -benchmem ./internal/... > /tmp/bench-current.txt 2>&1
	@echo "Benchmark results saved to /tmp/bench-current.txt"
	@echo "Use 'benchstat old.txt /tmp/bench-current.txt' to compare"

lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	@echo "Format complete"

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...
	@echo "go vet complete"

setup-hooks: ## Install git hooks (pre-commit and commit-msg)
	@echo "Setting up git hooks..."
	@mkdir -p .git/hooks
	@if [ ! -f .githooks/pre-commit ]; then \
		echo "ERROR: .githooks/pre-commit not found"; \
		exit 1; \
	fi
	@if [ ! -f .githooks/commit-msg ]; then \
		echo "ERROR: .githooks/commit-msg not found"; \
		exit 1; \
	fi
	@cp .githooks/pre-commit .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@cp .githooks/commit-msg .git/hooks/commit-msg
	@chmod +x .git/hooks/commit-msg
	@echo "Git hooks installed successfully!"
	@echo "Pre-commit hook: go fmt, go vet, golangci-lint, and tests"
	@echo "Commit-msg hook: conventional commit format validation"

run: build ## Build and run the application
	@$(BUILD_DIR)/$(BINARY_NAME)

dev: ## Run the application without building (go run)
	go run $(MAIN_PATH) $(ARGS)

test-large: build ## Test with large crontabs (100+ jobs)
	@echo "Creating large test crontab..."
	@cat /dev/null > /tmp/large-test.cron
	@for i in $$(seq 1 100); do \
		echo "0 * * * * /usr/bin/job$$i.sh" >> /tmp/large-test.cron; \
	done
	@echo "Testing with large crontab..."
	@time ./bin/cronic check --file /tmp/large-test.cron
	@time ./bin/cronic list --file /tmp/large-test.cron --json > /dev/null
	@rm -f /tmp/large-test.cron
	@echo "Large crontab tests passed"

test-performance: ## Run performance integration tests
	@echo "Running performance tests..."
	@which ginkgo > /dev/null || (echo "Installing ginkgo..." && go install github.com/onsi/ginkgo/v2/ginkgo@latest)
	@ginkgo -v ./test/integration/performance_test.go

docs-check: ## Verify documentation completeness
	@echo "Checking documentation..."
	@test -f README.md || (echo "ERROR: README.md missing" && exit 1)
	@test -f docs/JSON_SCHEMAS.md || (echo "ERROR: docs/JSON_SCHEMAS.md missing" && exit 1)
	@test -f docs/TROUBLESHOOTING.md || (echo "ERROR: docs/TROUBLESHOOTING.md missing" && exit 1)
	@echo "✓ All required documentation files exist"

profile: ## Generate performance profiles
	@echo "Generating CPU profile..."
	@mkdir -p $(BUILD_DIR)
	@go test -bench=. -cpuprofile=$(BUILD_DIR)/cpu.prof ./internal/crontab
	@go test -bench=. -cpuprofile=$(BUILD_DIR)/cpu-parser.prof ./internal/cronx
	@go test -bench=. -cpuprofile=$(BUILD_DIR)/cpu-validator.prof ./internal/check
	@echo "Generating memory profile..."
	@go test -bench=. -memprofile=$(BUILD_DIR)/mem.prof ./internal/crontab
	@go test -bench=. -memprofile=$(BUILD_DIR)/mem-parser.prof ./internal/cronx
	@go test -bench=. -memprofile=$(BUILD_DIR)/mem-validator.prof ./internal/check
	@echo "Profiles generated in $(BUILD_DIR)/"
	@echo "View with: go tool pprof $(BUILD_DIR)/cpu.prof"
	@echo "Or use: go tool pprof -http=:8080 $(BUILD_DIR)/cpu.prof"

examples: ## Validate example crontabs
	@echo "Validating example crontabs..."
	@for file in examples/crontabs/*.cron; do \
		echo "Checking $$file..."; \
		./bin/cronic check --file $$file || exit 1; \
	done
	@echo "All example crontabs are valid"

docs: ## Generate documentation (placeholder for future doc generation)
	@echo "Documentation is in docs/ directory"
	@echo "Available docs:"
	@ls -1 docs/*.md

dev-setup: ## Complete development environment setup
	@echo "Setting up development environment..."
	@$(MAKE) setup-hooks
	@go mod download
	@echo "Development environment ready!"
	@echo "Run 'make test' to verify setup"

build-all: ## Build for multiple platforms
	@echo "Building for multiple platforms..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "Build complete for all platforms in $(DIST_DIR)/"

.DEFAULT_GOAL := help
