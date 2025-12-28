# Cronic

> Make cron human again.

**Cronic** is a command-line tool that makes cron jobs human-readable, auditable, and visual. It converts confusing cron syntax into plain English, generates upcoming run schedules, provides ASCII timeline visualizations, and validates crontabs with severity levels and diagnostic codes.

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25.2%2B-blue.svg)](https://golang.org/)
[![codecov](https://codecov.io/gh/hzerrad/cronic/branch/main/graph/badge.svg)](https://codecov.io/gh/hzerrad/cronic)

## Features

- **Explain** - Convert cron expressions to plain English
- **Next** - Show the next N scheduled run times
- **List** - Parse and summarize crontab jobs from files or user crontabs
- **Timeline** - Visualize job schedules with ASCII timelines showing density and overlaps
- **Check** - Validate crontab syntax with severity levels and diagnostic codes (DOM/DOW conflicts, empty schedules)
- **JSON Output** - Machine-readable output for all commands via `--json` flag
- **Read-Only** - Safe by design; never executes or modifies crontabs

## Installation

### Using Go Install

```bash
go install github.com/hzerrad/cronic/cmd/cronic@latest
```

### From Source

```bash
git clone https://github.com/hzerrad/cronic.git
cd cronic
make install
```

### Building from Source

```bash
git clone https://github.com/hzerrad/cronic.git
cd cronic
make build
# Binary will be in ./bin/cronic
```

## Quick Start

### Explain a Cron Expression

```bash
$ cronic explain "*/15 2-5 * * 1-5"
Runs every 15 minutes between 02:00–05:59 on weekdays (Mon–Fri).
```

### Show Next Run Times

```bash
$ cronic next "0 9 * * *" --count 5
Next 5 runs for "0 9 * * *" (At 09:00 daily):

1. 2025-12-29 09:00:00 UTC
2. 2025-12-30 09:00:00 UTC
3. 2025-12-31 09:00:00 UTC
4. 2026-01-01 09:00:00 UTC
5. 2026-01-02 09:00:00 UTC
```

### List Crontab Jobs

```bash
$ cronic list --file /etc/crontab
LINE  EXPRESSION        DESCRIPTION                          COMMAND
────  ────────────────  ───────────────────────────────────  ────────────────────────
1     0 2 * * *         At 02:00 daily                       /usr/bin/backup.sh
2     */15 * * * *      Every 15 minutes                     /usr/bin/check-disk.sh
```

### Visualize Timeline

```bash
$ cronic timeline "*/15 * * * *" --view day
Timeline for 2025-12-28 (Day View)
00:00 ──────────────────────────────────────────────────────────────── 24:00
      │                                                                    │
      │  ████  ████  ████  ████  ████  ████  ████  ████  ████  ████  ████  │
      │                                                                    │
      └──────────────────────────────────────────────────────────────────┘
      expr-*/15 * * * *: Every 15 minutes
```

### Validate Crontab

```bash
$ cronic check --file /etc/crontab
✓ All valid (2 jobs)

$ cronic check "0 0 1 * 1" --verbose
⚠ Found 1 warning(s)
  Total jobs: 1
  Valid: 1
  Invalid: 0

⚠ WARNING: Both day-of-month and day-of-week specified (runs if either condition is met) [CRON-001]
  Expression: 0 0 1 * 1
  Hint: Consider using only day-of-month OR day-of-week, not both. Cron uses OR logic (runs if either condition is met).

$ cronic check "60 0 * * *"
✗ Found 1 issue(s)
  Total jobs: 1
  Valid: 0
  Invalid: 1

✗ ERROR: Invalid cron expression: expected 5 fields [CRON-003]
  Expression: 60 0 * * *
  Hint: Fix the syntax error in the cron expression. Ensure all 5 fields are present and valid.
```

## Commands

### `explain`

Convert a cron expression to plain English.

```bash
cronic explain <cron-expression>
cronic explain "*/15 * * * *"
cronic explain "@daily"
cronic explain "0 9 * * 1-5" --json
```

### `next`

Show the next N scheduled run times for a cron expression.

```bash
cronic next <cron-expression> [flags]
cronic next "*/15 * * * *"              # Next 10 runs (default)
cronic next "@daily" --count 5          # Next 5 runs
cronic next "0 9 * * 1-5" -c 3          # Next 3 runs
cronic next "0 14 * * *" --json          # JSON output
```

**Flags:**
- `-c, --count <number>` - Number of runs to show (1-100, default: 10)
- `-j, --json` - Output as JSON

### `list`

Parse and list cron jobs from a crontab file or the user's crontab.

```bash
cronic list [flags]
cronic list                              # List user's crontab
cronic list --file /etc/crontab         # List from file
cronic list --all                        # Include comments and env vars
cronic list --json                       # JSON output
```

**Flags:**
- `-f, --file <path>` - Path to crontab file
- `-a, --all` - Show all entries including comments and environment variables
- `-j, --json` - Output as JSON

### `timeline`

Display ASCII timeline visualization of cron job schedules.

```bash
cronic timeline [cron-expression] [flags]
cronic timeline "*/15 * * * *"              # Timeline for single expression
cronic timeline --file /etc/crontab         # Timeline for crontab file
cronic timeline "*/5 * * * *" --view hour   # Hour view timeline
cronic timeline --file jobs.cron --json     # JSON output
```

**Flags:**
- `-f, --file <path>` - Path to crontab file (defaults to user's crontab)
- `--view <type>` - Timeline view: `day` (24 hours) or `hour` (60 minutes, default: `day`)
- `--from <time>` - Start time for timeline (RFC3339 format, defaults to current time)
- `-j, --json` - Output as JSON

### `check`

Validate crontab syntax and detect common issues with severity levels and diagnostic codes.

```bash
cronic check [cron-expression|--file <path>] [flags]
cronic check "0 0 * * *"                  # Validate single expression
cronic check --file /etc/crontab         # Validate crontab file
cronic check "0 0 1 * 1" --verbose       # Show warnings with diagnostic codes
cronic check --file jobs.cron --json     # JSON output
```

**Flags:**
- `-f, --file <path>` - Path to crontab file
- `-v, --verbose` - Show warnings (DOM/DOW conflicts, etc.) with diagnostic codes and hints
- `-j, --json` - Output as JSON

**Severity Levels:**
- **Error** (`✗ ERROR`) - Invalid expressions or critical issues that prevent execution
- **Warning** (`⚠ WARNING`) - Potential issues that may cause unexpected behavior
- **Info** (`ℹ INFO`) - Informational messages (future use)

**Diagnostic Codes:**
- `CRON-001` - DOM/DOW conflict (warning)
- `CRON-002` - Empty schedule (error)
- `CRON-003` - Parse error (error)
- `CRON-004` - File read error (error)
- `CRON-005` - Invalid crontab structure (error)

Each diagnostic includes a **hint** with actionable suggestions for fixing the issue.

**Exit Codes:**
- `0` - All valid (no errors or only info messages)
- `1` - Errors found
- `2` - Warnings found (only with `--verbose`)

## Global Flags

All commands support these global flags:

- `--locale <LANG>` - Locale for parsing day/month names (default: `en`)
- `--json, -j` - Output as JSON (machine-readable)

## Supported Cron Dialect

- **Standard 5-field Vixie cron**: `minute hour dom month dow`
- **Aliases**: `@hourly`, `@daily`, `@weekly`, `@monthly`, `@yearly`
- **Case-insensitive day/month names**: `MON-SUN`, `JAN-DEC`
- **Ranges**: `1-5`, `MON-FRI`
- **Steps**: `*/15`, `0-23/2`
- **Lists**: `1,3,5`, `MON,WED,FRI`

## JSON Output

All commands support `--json` flag for machine-readable output. The JSON schema is stable and documented for automation and CI/CD integration.

**Example - Explain:**
```bash
$ cronic explain "*/15 * * * *" --json
{
  "expression": "*/15 * * * *",
  "description": "Every 15 minutes"
}
```

**Example - Check (with severity and diagnostic codes):**
```bash
$ cronic check "0 0 1 * 1" --json --verbose
{
  "valid": true,
  "totalJobs": 1,
  "validJobs": 1,
  "invalidJobs": 0,
  "issues": [
    {
      "severity": "warn",
      "code": "CRON-001",
      "lineNumber": 0,
      "expression": "0 0 1 * 1",
      "message": "Both day-of-month and day-of-week specified (runs if either condition is met)",
      "hint": "Consider using only day-of-month OR day-of-week, not both. Cron uses OR logic (runs if either condition is met).",
      "type": "warning"
    }
  ]
}
```

The `type` field is deprecated but maintained for backward compatibility. Use `severity` instead.

## Safety

**Cronic is read-only by design.** It never executes or modifies crontabs. It's safe to use on production systems for auditing and documentation purposes.

## Requirements

- **Go**: 1.25.2 or higher (for building from source)
- **Platform**: Linux, macOS, Windows (single static binary)

## Development

### Prerequisites

- Go 1.25.2 or higher
- Make
- golangci-lint (recommended, for linting)

### Building

```bash
make build          # Build binary (./bin/cronic)
make build-all      # Cross-platform builds
make install        # Install to GOPATH/bin
```

### Testing

This project follows **Test-Driven Development (TDD)** and **Behavior-Driven Development (BDD)** practices with 95%+ test coverage.

```bash
make test           # All tests (unit + integration + E2E)
make test-unit      # Unit tests only
make test-integration  # Integration tests
make test-e2e       # E2E tests
make test-coverage  # Generate coverage report
```

**Documentation:**
- [TESTING.md](TESTING.md) - Comprehensive testing guidelines
- [BDD Tutorial](docs/BDD_TUTORIAL.md) - Hands-on BDD with Ginkgo tutorial

### Code Quality

```bash
make fmt            # Format code
make vet            # Run go vet
make lint           # Run golangci-lint
make setup-hooks    # Install pre-commit hooks
```

### Project Structure

```
cronic/
├── cmd/cronic/          # CLI entry point
├── internal/            # Private application code
│   ├── cmd/            # Command implementations
│   ├── cronx/          # Cron parser abstraction
│   ├── human/          # Humanization templates
│   ├── render/         # Timeline renderer
│   ├── crontab/        # Crontab reader
│   └── check/          # Validation logic
├── test/               # Integration and E2E tests
│   ├── integration/    # Integration tests (Ginkgo)
│   └── e2e/           # E2E tests (Ginkgo)
├── testdata/          # Test fixtures
└── docs/              # Documentation
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## Documentation

- [CRONIC_REFERENCE.md](docs/CRONIC_REFERENCE.md) - Complete specification and roadmap
- [TESTING.md](TESTING.md) - Testing guidelines
- [BDD_TUTORIAL.md](docs/BDD_TUTORIAL.md) - BDD testing tutorial

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Author

[hzerrad](https://github.com/hzerrad)

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI framework
- Cron parsing powered by [robfig/cron](https://github.com/robfig/cron/v3)
- Testing with [Ginkgo](https://onsi.github.io/ginkgo/) and [Gomega](https://onsi.github.io/gomega/)

---

**Made with ❤️ for developers who work with cron jobs**
