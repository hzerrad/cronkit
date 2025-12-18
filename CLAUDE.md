# Cronic Project Guide for Claude Code

## Project Overview

**Cronic** is an open-source command-line tool that makes cron jobs human-readable, auditable, and visual. It turns confusing cron syntax into plain English, generates upcoming run schedules, and draws ASCII timelines for better understanding and documentation.

**Current Status:** Bootstrapped (design + planning phase)
**Version:** v0.1.0 (in development)
**Language:** Go 1.25.2+
**License:** Apache 2.0

**Tagline:** "Make cron human again."

### v0.1.0 Goals

Implement five core commands:
1. `explain` - Convert cron expressions to plain English
2. `next` - Show next N run times (default: 10)
3. `list` - Parse and summarize crontab jobs
4. `timeline` - ASCII visualization of job schedules
5. `check` - Validate crontab syntax and structure

All commands support `--json` mode for machine-readable output.

## Core Principles

| Principle | Description |
|-----------|-------------|
| **Clarity** | Make cron expressions readable by humans |
| **Safety** | Read-only by default; never execute or modify crontabs |
| **Portability** | Single static binary for all major OSes |
| **Accessibility** | Works in all terminals, width-aware, color optional |
| **Extensibility** | JSON schema and modular internals for future expansion |
| **Community-first** | Easy to contribute to and adapt for other schedulers |

**CRITICAL:** Cronic is read-only. It must NEVER execute or modify crontabs.

## Architecture

### Internal Package Structure

```
cronic/
├── cmd/cronic/          # CLI entry point
│   └── main.go
├── internal/            # Private application code
│   ├── cronx/          # Parser abstraction (wraps robfig/cron)
│   ├── human/          # Humanization templates
│   ├── plan/           # Next-run calculations, timeline logic
│   ├── render/         # Terminal table & timeline renderer
│   ├── crontab/        # Reader (system/user/file)
│   ├── check/          # Validation & linting
│   └── cmd/            # Command implementations (Cobra)
│       ├── root.go
│       ├── version.go
│       └── *_test.go
├── pkg/                # Public libraries (optional, future)
├── test/
│   ├── integration/    # Integration tests (Ginkgo)
│   └── e2e/           # End-to-end tests (Ginkgo)
├── testdata/          # Cron fixtures
└── docs/              # Documentation
    ├── CRONIC_REFERENCE.md
    └── BDD_TUTORIAL.md
```

### Architectural Patterns

- **Single-boundary integration:** All third-party cron parsing goes through `internal/cronx` wrapper
- **Separation of concerns:** Parser, humanizer, planner, renderer are independent modules
- **Dependency injection:** Commands receive dependencies via constructor/init patterns
- **Template-based i18n:** Humanization uses templates for localization readiness

## Development Workflow

### Prerequisites

- Go 1.25.2 or higher
- Make
- Git
- golangci-lint (recommended, installed via `brew install golangci-lint` on macOS)

### Quick Start

```bash
# Clone and setup
git clone https://github.com/hzerrad/cronic.git
cd cronic
make setup-hooks  # Install pre-commit hooks

# Build
make build        # Binary: ./bin/cronic

# Run without building
make dev          # go run cmd/cronic/main.go

# Test
make test         # Run all tests
make test-unit    # Unit tests only
make test-bdd     # Integration + E2E tests

# Code quality
make fmt          # Format code
make vet          # Run go vet
make lint         # Run golangci-lint
```

### Pre-Commit Hooks

Pre-commit hooks automatically run on every commit:
1. `go fmt` - Enforces formatting (blocks commit if not formatted)
2. `go vet` - Detects common errors
3. `golangci-lint` - Comprehensive linting (if installed)

Install with: `make setup-hooks`

## Testing Strategy (TDD/BDD)

### Three-Tier Testing Approach

#### 1. Unit Tests
- **Location:** Colocated with source code (`internal/cmd/*_test.go`)
- **Framework:** Go testing + testify
- **Pattern:** Table-driven tests with subtests
- **Coverage:** Test individual functions and methods

#### 2. Integration Tests
- **Location:** `test/integration/`
- **Framework:** Ginkgo v2 + Gomega
- **Pattern:** BDD style (Describe/Context/It)
- **Coverage:** Test CLI commands via binary execution

#### 3. E2E Tests
- **Location:** `test/e2e/`
- **Framework:** Ginkgo v2 + Gomega
- **Pattern:** BDD style with multi-step scenarios
- **Coverage:** Test complete user workflows

### Coverage Requirements

- **Overall minimum:** 80%
- **Critical paths:** 90%
- **New code:** 100% (all new code MUST have tests)

### Testing Workflow (TDD)

1. **Red:** Write failing test first
2. **Green:** Write minimal code to pass
3. **Refactor:** Clean up while keeping tests green

### Unit Test Pattern

```go
func TestExampleCommand(t *testing.T) {
    t.Run("should execute with custom name", func(t *testing.T) {
        // Setup: Capture output
        buf := new(bytes.Buffer)
        rootCmd.SetOut(buf)
        rootCmd.SetErr(buf)

        // Execute: Run command with args
        rootCmd.SetArgs([]string{"example", "--name", "Test"})
        err := rootCmd.Execute()

        // Assert: Verify behavior
        require.NoError(t, err)
        assert.Contains(t, buf.String(), "Hello, Test!")
    })
}
```

**Key patterns:**
- Use `t.Run()` for subtests
- Use `require.*` for critical checks (fails fast)
- Use `assert.*` for non-critical checks (continues)
- Capture output with `bytes.Buffer`
- Test both success and error paths

### BDD Test Pattern (Integration/E2E)

```go
var _ = Describe("Version Command", func() {
    Context("when running 'cronic version'", func() {
        It("should display version information", func() {
            // Execute
            command := exec.Command(pathToCLI, "version")
            session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
            Expect(err).NotTo(HaveOccurred())

            // Assert
            Eventually(session).Should(gexec.Exit(0))
            Expect(session.Out).To(gbytes.Say("cronic"))
        })
    })
})
```

**Key patterns:**
- `Describe()` - Feature grouping
- `Context()` - Scenario conditions
- `It()` - Expected behavior
- `BeforeSuite()/AfterSuite()` - One-time setup/teardown
- `BeforeEach()/AfterEach()` - Per-test setup/teardown
- `gexec.Build()` - Compile binary for testing
- `gexec.Start()` - Execute binary
- `Eventually()` - Wait for async operations
- `gbytes.Say()` - Pattern match output

### Multi-Step E2E Pattern

```go
It("should handle complete workflow", func() {
    By("checking version first")
    // ... step 1 ...

    By("running main command")
    // ... step 2 ...

    By("verifying output")
    // ... step 3 ...
})
```

Use `By()` to document steps in complex scenarios.

## Code Conventions

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` for formatting (enforced by pre-commit hooks)
- Write clear, descriptive names
- Comment all exported functions (godoc style)

### Cobra Command Pattern

```go
var myCmd = &cobra.Command{
    Use:   "mycommand",
    Short: "One-line description",
    Long: `Detailed explanation with usage examples.

Can span multiple lines.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation with error handling
        return nil
    },
}

func init() {
    rootCmd.AddCommand(myCmd)

    // Flags
    myCmd.Flags().StringP("name", "n", "", "Description")
    myCmd.Flags().BoolP("json", "j", false, "Output JSON")
}
```

**Command conventions:**
- Use lowercase for command names
- Use `RunE` for commands that can return errors
- Register in `init()` via `rootCmd.AddCommand()`
- Define flags with long and short forms
- Provide clear help text

### Test Naming

```go
// Unit tests
func TestFeatureName(t *testing.T)
func TestVersionCommand_ShouldReturnVersion(t *testing.T)

// BDD tests
Describe("Feature Name", func() { ... })
Context("when condition exists", func() { ... })
It("should produce expected outcome", func() { ... })
```

### File Organization

- One command per file: `internal/cmd/explain.go`, `internal/cmd/explain_test.go`
- Test files alongside source: `foo.go` → `foo_test.go`
- Shared test fixtures: `testdata/` directory

## Dependencies

### Approved Dependencies (v0.1.0)

| Package | Purpose | License |
|---------|---------|---------|
| `github.com/spf13/cobra` | CLI framework | Apache 2.0 |
| `robfig/cron/v3` | Cron parsing (wrapped) | MIT |
| `go-pretty/table` | Terminal tables | MIT |
| `fatih/color` | Colored output | MIT |
| `tzdata` | Embedded timezone DB | BSD |

### Dependency Policy

- All dependencies must be Apache/MIT/BSD licensed
- Vendor dependencies for reproducibility
- Wrap third-party libs under `internal/cronx` (single import boundary)
- Pin versions in `go.mod`
- Run security audits: `govulncheck ./...`
- Regular updates via Renovate/Dependabot

### Adding New Dependencies

1. Check license compatibility (Apache/MIT/BSD only)
2. Verify active maintenance
3. Run `go get <package>`
4. Update `go.mod` and `go.sum`
5. Vendor if needed: `go mod vendor`
6. Document in architecture section

## Quality Gates

### Pre-Commit (Automatic)

- `go fmt` - Code formatting
- `go vet` - Error detection
- `golangci-lint` - Comprehensive linting

### Pre-Push (Manual)

- `make test` - All tests pass
- `make test-coverage` - Coverage thresholds met
- `make lint` - No linting issues

### Pull Request Requirements

- All tests pass (unit + integration + E2E)
- Coverage: 80% overall, 90% critical paths, 100% new code
- No linting errors
- Documentation updated
- Pre-commit hooks passing

## Adding New Commands

### Step-by-Step Process

1. **Write tests first (TDD)**
   ```bash
   # Create test file
   touch internal/cmd/mycommand_test.go

   # Write failing tests
   # Run tests: make test-unit
   ```

2. **Implement command**
   ```bash
   # Create command file
   touch internal/cmd/mycommand.go

   # Implement to pass tests
   ```

3. **Add integration tests**
   ```bash
   # Add to test/integration/cli_integration_test.go
   # Run: make test-integration
   ```

4. **Run quality checks**
   ```bash
   make fmt
   make vet
   make lint
   ```

5. **Verify coverage**
   ```bash
   make test-coverage
   # Check HTML report: open bin/coverage.html
   ```

6. **Manual testing**
   ```bash
   make build
   ./bin/cronic mycommand --help
   ```

### Example Command Implementation

See `internal/cmd/example.go` for reference implementation demonstrating:
- Command structure
- Flag handling
- Output formatting
- Error handling
- Testing patterns

## Make Targets Reference

### Building
```bash
make build          # Build binary (./bin/cronic)
make build-all      # Cross-platform builds
make install        # Install to GOPATH/bin
make clean          # Remove build artifacts
```

### Testing
```bash
make test           # All tests (unit + BDD)
make test-unit      # Unit tests only
make test-integration  # Integration tests
make test-e2e       # E2E tests
make test-bdd       # Integration + E2E
make test-coverage  # Generate coverage report
make test-watch     # Continuous testing
```

### Code Quality
```bash
make fmt            # Format code
make vet            # Run go vet
make lint           # Run golangci-lint
make setup-hooks    # Install pre-commit hooks
```

### Development
```bash
make dev            # Run without building
make run            # Build then run
```

## Dos and Don'ts

### DO

- ✅ Write tests first (TDD approach)
- ✅ Use BDD style for integration/E2E tests
- ✅ Wrap third-party dependencies under `internal/cronx`
- ✅ Vendor dependencies for reproducibility
- ✅ Document all exported functions
- ✅ Keep binary lightweight (<10 MB)
- ✅ Support both human and JSON output modes
- ✅ Respect `NO_COLOR` environment variable
- ✅ Respect terminal width
- ✅ Ensure read-only operation (never modify crontabs)
- ✅ Validate crontab syntax comprehensively
- ✅ Run `make fmt` and `make vet` before committing
- ✅ Maintain 80%+ test coverage
- ✅ Use table-driven tests for multiple cases
- ✅ Test both success and error paths

### DON'T

- ❌ Execute or modify crontabs (safety principle)
- ❌ Add external APIs or network calls
- ❌ Add unpinned or unvetted dependencies
- ❌ Break backward compatibility of `--json` output
- ❌ Skip tests or documentation
- ❌ Commit without running pre-commit hooks
- ❌ Add dependencies with non-permissive licenses
- ❌ Ignore terminal accessibility (NO_COLOR, width)
- ❌ Support Quartz extensions (L, W, #) in v0.1.0
- ❌ Add complexity beyond v0.1.0 scope
- ❌ Use mocks without clear benefit
- ❌ Write implementation-dependent tests
- ❌ Commit code without tests

## Cron Dialect Support

### Supported (v0.1.0)

- Standard 5-field Vixie cron: `minute hour dom month dow`
- Aliases: `@hourly`, `@daily`, `@weekly`, `@monthly`, `@yearly`
- Case-insensitive day/month names: `MON-SUN`, `JAN-DEC`
- Ranges: `1-5`, `MON-FRI`
- Steps: `*/15`, `0-23/2`
- Lists: `1,3,5`, `MON,WED,FRI`

### Not Supported (v0.1.0)

- Quartz extensions: `L`, `W`, `LW`, `#`
- Seconds field (planned for v0.2 with `--with-seconds`)
- Non-standard dialects

## Global CLI Flags

All commands should support:

```bash
--json              # Output JSON (machine-readable)
--tz <ZONE>        # Timezone override
--width <COLS>     # Terminal width override
--locale <LANG>    # Language (default: en)
--no-color         # Disable colored output
```

## Troubleshooting

### Tests Failing

```bash
# Run specific test
go test -v ./internal/cmd -run TestVersionCommand

# Run with verbose output
make test-unit

# Check coverage
make test-coverage
```

### Build Issues

```bash
# Clean and rebuild
make clean
make build

# Check Go version
go version  # Should be 1.25.2+

# Update dependencies
go mod tidy
```

### Pre-Commit Hook Issues

```bash
# Reinstall hooks
make setup-hooks

# Run checks manually
make fmt
make vet
make lint
```

## Resources

- **Architecture:** `docs/CRONIC_REFERENCE.md`
- **Testing Guide:** `TESTING.md`
- **BDD Tutorial:** `docs/BDD_TUTORIAL.md`
- **Contributing:** `CONTRIBUTING.md`
- **Roadmap:** `docs/CRONIC_REFERENCE.md` (v0.2-v1.0 section)

## Version Information

This CLAUDE.md file corresponds to:
- **Cronic:** v0.1.0 (in development)
- **Last Updated:** 2025-12-18
- **Go Version:** 1.25.2+

---

**Remember:** Cronic's mission is to make cron human-readable, safe, and accessible. Every change should align with these core principles.
