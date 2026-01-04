# Development Guide

This guide covers the development workflow, testing practices, and contribution guidelines for Cronic.

## Development Setup

### Prerequisites

- **Go**: 1.25.2 or higher
- **Make**: For running build and test commands
- **golangci-lint**: For code linting (recommended)
- **Ginkgo v2**: For BDD testing (installed automatically via Makefile)

### Initial Setup

```bash
# Clone the repository
git clone https://github.com/hzerrad/cronic.git
cd cronic

# Install dependencies
go mod download

# Install development tools
make setup-hooks  # Install pre-commit hooks
```

## Development Workflow

### Running Tests

This project follows **Test-Driven Development (TDD)** and **Behavior-Driven Development (BDD)** practices.

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests
make test-integration

# Run E2E tests
make test-e2e

# Run tests with coverage
make test-coverage
# View coverage report: open bin/coverage.html

# Run tests in watch mode (requires ginkgo)
make test-watch
```

### Running Benchmarks

```bash
# Run all benchmarks
make benchmark

# Run specific benchmark
go test -bench=BenchmarkReadFile -benchmem ./internal/crontab

# Compare benchmark results (requires benchstat)
make benchmark-compare
```

### Code Quality Checks

```bash
# Format code
make fmt

# Run go vet
make vet

# Run linter
make lint

# All checks (run before committing)
make fmt && make vet && make lint
```

### Building

```bash
# Build binary
make build
# Binary: ./bin/cronic

# Build for all platforms
make build-all
# Binaries: ./dist/

# Install to GOPATH/bin
make install

# Run without building
make dev
```

## Code Style Guidelines

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` for formatting (enforced by pre-commit hooks)
- Write clear, descriptive names
- Comment all exported functions (godoc style)

### Naming Conventions

- **Commands**: lowercase, single word (e.g., `explain`, `list`, `next`)
- **Functions**: PascalCase for exported, camelCase for private
- **Variables**: camelCase
- **Constants**: PascalCase or UPPER_SNAKE_CASE
- **Test Functions**: `TestFeatureName` or `TestFeatureName_Scenario`

### File Organization

- One command per file: `internal/cmd/explain.go`, `internal/cmd/explain_test.go`
- Test files alongside source: `foo.go` → `foo_test.go`
- Shared test fixtures: `testdata/` directory

## Testing Strategy

### Three-Tier Testing Approach

1. **Unit Tests** (colocated with source)
   - Location: `internal/*/*_test.go`
   - Framework: Go testing + testify
   - Pattern: Table-driven tests with subtests
   - Coverage: Test individual functions and methods

2. **Integration Tests** (`test/integration/`)
   - Framework: Ginkgo v2 + Gomega
   - Pattern: BDD style (Describe/Context/It)
   - Coverage: Test CLI commands via binary execution

3. **E2E Tests** (`test/e2e/`)
   - Framework: Ginkgo v2 + Gomega
   - Pattern: BDD style with multi-step scenarios
   - Coverage: Test complete user workflows

### Test Coverage Requirements

- **Overall minimum**: 95% (no less than 95% test coverage required)
- **Critical paths**: 90%
- **New code**: 100% (all new code MUST have tests)

### Writing Tests

#### Unit Test Pattern

```go
func TestFeatureName(t *testing.T) {
    t.Run("should handle valid input", func(t *testing.T) {
        // Setup
        input := "test"
        
        // Execute
        result, err := SomeFunction(input)
        
        // Assert
        require.NoError(t, err)
        assert.Equal(t, expected, result)
    })
    
    t.Run("should return error for invalid input", func(t *testing.T) {
        _, err := SomeFunction(invalidInput)
        require.Error(t, err)
        assert.Contains(t, err.Error(), "expected message")
    })
}
```

#### BDD Test Pattern (Integration/E2E)

```go
package integration_test

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    "github.com/onsi/gomega/gbytes"
    "github.com/onsi/gomega/gexec"
)

var _ = Describe("Feature Name", func() {
    Context("when condition exists", func() {
        It("should produce expected outcome", func() {
            command := exec.Command(pathToCLI, "command", "args")
            session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
            Expect(err).NotTo(HaveOccurred())
            
            Eventually(session).Should(gexec.Exit(0))
            Expect(session.Out).To(gbytes.Say("expected output"))
        })
    })
})
```

### Test Utilities

Use the `internal/testutil` package for common test helpers:

```go
import "github.com/hzerrad/cronic/internal/testutil"

// Create temporary crontab file
file, cleanup := testutil.CreateTempCrontab(t, "0 2 * * * /usr/bin/backup.sh")
defer cleanup()

// Load test fixture
path := testutil.LoadTestCrontab("sample.cron")
```

## Commit Message Format

Follow conventional commit format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test additions/changes
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Maintenance tasks

**Examples:**
```
feat(next): add timezone support for next command

Add --timezone flag to next command to allow timezone-aware
calculations. Includes unit, integration, and E2E tests.

Closes #123
```

```
fix(check): correct exit code calculation with --fail-on flag

The exit code was not correctly calculated when using --fail-on
flag with warnings. Now properly returns exit code 2 for warnings
when --fail-on warn is set.
```

## Pull Request Process

1. **Create a feature branch:**
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Write tests first (TDD):**
   - Write failing tests
   - Run: `make test-unit`

3. **Implement the feature:**
   - Write code to pass tests
   - Run: `make test`

4. **Ensure code quality:**
   ```bash
   make fmt
   make vet
   make lint
   make test-coverage
   ```

5. **Update documentation:**
   - Update README.md if needed
   - Update relevant docs in `docs/`
   - Add examples if applicable

6. **Create pull request:**
   - Include description of changes
   - Reference related issues
   - Ensure CI checks pass

## Release Process

1. **Update version:**
   - Update version in relevant files
   - Update CHANGELOG.md

2. **Create release tag:**
   ```bash
   git tag -a v0.2.0 -m "Release v0.2.0"
   git push origin v0.2.0
   ```

3. **Verify release:**
   - Check GitHub Actions build
   - Test release artifacts
   - Update release notes

## Project Structure

```
cronic/
├── cmd/cronic/          # CLI entry point (main.go)
├── internal/            # Private application code
│   ├── cmd/            # Command implementations (Cobra)
│   ├── cronx/          # Parser abstraction (wraps robfig/cron)
│   ├── human/          # Humanization templates
│   ├── crontab/        # Reader (system/user/file)
│   ├── check/          # Validation & linting
│   ├── render/         # Timeline rendering
│   └── testutil/       # Test helper utilities
├── test/
│   ├── integration/    # Integration tests (Ginkgo)
│   └── e2e/           # End-to-end tests (Ginkgo)
├── testdata/          # Test fixtures
├── examples/          # Example crontabs and scripts
└── docs/              # Documentation
```

## Debugging

### Running with Debug Output

```bash
# Run with verbose output
go run ./cmd/cronic <command> --verbose

# Run specific test with debug
go test -v ./internal/cmd -run TestNextCommand
```

### Profiling

```bash
# CPU profile
go test -bench=. -cpuprofile=cpu.prof ./internal/crontab
go tool pprof cpu.prof

# Memory profile
go test -bench=. -memprofile=mem.prof ./internal/crontab
go tool pprof mem.prof
```

## Common Tasks

### Adding a New Command

1. Create command file: `internal/cmd/newcommand.go`
2. Create test file: `internal/cmd/newcommand_test.go`
3. Add integration tests: `test/integration/newcommand_test.go`
4. Register in `internal/cmd/root.go`
5. Update README.md

### Adding a New Validation Rule

1. Add diagnostic code: `internal/check/codes.go`
2. Implement validation: `internal/check/validator.go`
3. Add tests: `internal/check/validator_test.go`
4. Update documentation

### Updating JSON Schema

1. Update command JSON output
2. Update `docs/JSON_SCHEMAS.md`
3. Add backward compatibility notes if needed
4. Update tests

## Resources

- [Go Documentation](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Cobra Documentation](https://github.com/spf13/cobra)
- [Ginkgo Documentation](https://onsi.github.io/ginkgo/)
- [Testify Documentation](https://github.com/stretchr/testify)

## Getting Help

If you need help:

1. Check this documentation
2. Review existing code and tests for examples
3. Check GitHub issues and discussions
4. Ask in team discussions


