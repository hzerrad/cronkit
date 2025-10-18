.PHONY: help build install clean test test-coverage lint fmt run dev build-all setup-hooks vet

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
	@awk 'BEGIN {FS = ":.*##"; printf "\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

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

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@mkdir -p $(BUILD_DIR)
	go test -v -coverprofile=$(BUILD_DIR)/coverage.out ./...
	go tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "Coverage report generated: $(BUILD_DIR)/coverage.html"

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

setup-hooks: ## Install git pre-commit hooks
	@echo "Setting up git hooks..."
	@mkdir -p .githooks
	@cat > .githooks/pre-commit <<'EOF'
#!/bin/bash
# Pre-commit hook for Go projects
# Enforces go fmt, go vet, and golangci-lint

set -e

echo "Running pre-commit checks..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get list of staged Go files
STAGED_GO_FILES=$$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$$' || true)

if [ -z "$$STAGED_GO_FILES" ]; then
    echo -e "$${GREEN}✓$${NC} No Go files to check"
    exit 0
fi

echo "Checking Go files: $$STAGED_GO_FILES"

# Check 1: go fmt
echo -e "\n$${YELLOW}Running go fmt...$${NC}"
UNFORMATTED_FILES=$$(gofmt -l $$STAGED_GO_FILES)
if [ -n "$$UNFORMATTED_FILES" ]; then
    echo -e "$${RED}✗$${NC} The following files are not formatted:"
    echo "$$UNFORMATTED_FILES"
    echo ""
    echo "Please run: make fmt"
    echo "Or run: gofmt -w $$UNFORMATTED_FILES"
    exit 1
fi
echo -e "$${GREEN}✓$${NC} All files are properly formatted"

# Check 2: go vet
echo -e "\n$${YELLOW}Running go vet...$${NC}"
if ! go vet ./...; then
    echo -e "$${RED}✗$${NC} go vet found issues"
    echo "Please fix the issues before committing"
    exit 1
fi
echo -e "$${GREEN}✓$${NC} go vet passed"

# Check 3: golangci-lint (if available)
echo -e "\n$${YELLOW}Running golangci-lint...$${NC}"
if ! command -v golangci-lint &> /dev/null; then
    echo -e "$${YELLOW}⚠$${NC}  golangci-lint not found, skipping lint check"
    echo "Install it from: https://golangci-lint.run/usage/install/"
else
    if ! golangci-lint run ./...; then
        echo -e "$${RED}✗$${NC} golangci-lint found issues"
        echo "Please fix the issues before committing"
        exit 1
    fi
    echo -e "$${GREEN}✓$${NC} golangci-lint passed"
fi

# All checks passed
echo -e "\n$${GREEN}✓ All pre-commit checks passed!$${NC}\n"
exit 0
EOF
	@chmod +x .githooks/pre-commit
	@cp .githooks/pre-commit .git/hooks/pre-commit
	@echo "Git hooks installed successfully!"
	@echo "Pre-commit hook will run: go fmt, go vet, and golangci-lint"

run: build ## Build and run the application
	@$(BUILD_DIR)/$(BINARY_NAME)

dev: ## Run the application without building (go run)
	go run $(MAIN_PATH)

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
