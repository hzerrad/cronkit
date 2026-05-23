# Cronkit

> Make cron human again.

**Cronkit** is a command-line tool that makes cron jobs human-readable, auditable, and visual. It converts confusing cron syntax into plain English, generates upcoming run schedules, provides ASCII timeline visualizations, and validates crontabs with severity levels and diagnostic codes.

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25.2%2B-blue.svg)](https://golang.org/)
[![codecov](https://codecov.io/gh/hzerrad/cronkit/branch/main/graph/badge.svg)](https://codecov.io/gh/hzerrad/cronkit)

![Cronkit demo](docs/demo.gif)

## Why Cronkit?

Cron syntax is hostile. You write `0 2 * * *` and three months later you're staring at it wondering what it does. You inherit a crontab with sixty lines and no comments. You miss the difference between `* * * * 1` (every Monday) and `0 0 1 * 1` (midnight on the 1st of the month *and* every Monday — because cron ORs day-of-month and day-of-week when both are set), and you find out at 3 AM on the wrong day.

Cronkit reads cron back to you in plain English, shows you when it'll actually run, and catches the dumb mistakes — DOM/DOW conflicts, missing absolute paths, missing output redirects, runaway frequencies — before they turn into a pager incident.

It runs offline, emits JSON for your pipelines, and never executes or modifies your crontabs. It's safe to drop into a pre-commit hook on day one.

- **Works offline** - No network, no external service. Drop it into air-gapped boxes and CI runners without a second thought.
- **Built for CI/CD** - Machine-readable JSON, deterministic exit codes, and a `--fail-on` severity gate that slots straight into pre-commit hooks and PR checks.
- **Visual timelines** - ASCII timelines show schedule density and overlapping jobs at a glance, so you catch contention before it bites.
- **Real auditing** - Severity-coded diagnostics, frequency analysis, command-hygiene checks, and concurrency-budget analysis — not just humanization.

## Features

- **Explain** - Convert cron expressions to plain English
- **Next** - Show the next N scheduled run times
- **List** - Parse and summarize crontab jobs from files or user crontabs
- **Timeline** - Visualize job schedules with ASCII timelines showing density and overlaps
- **Check** - Validate crontab syntax with severity levels and diagnostic codes, including advanced linting (frequency analysis, command hygiene, overlap detection)
- **Doc** - Generate comprehensive documentation (Markdown, HTML, JSON) from crontabs with optional sections
- **Stats** - Calculate fleet statistics including run frequency metrics, collision analysis, and hour distribution
- **Diff** - Compare crontabs semantically to see what actually changed (jobs added/removed/modified)
- **Budget** - Analyze concurrency budgets to prevent resource exhaustion from too many simultaneous jobs
- **JSON Output** - Machine-readable output for all commands via `--json` flag
- **Read-Only** - Safe by design; never executes or modifies crontabs

## How it compares

- **[crontab.guru](https://crontab.guru)** — Excellent web-based humanizer. Cronkit covers the same humanization in a CLI that works offline, and adds validation, timeline rendering, audit linting, diff, stats, and budget analysis.
- **[cronstrue](https://github.com/bradymholt/cronstrue)** — JavaScript library for humanizing cron. Library-shaped, not a CLI; humanization only.
- **[croniter](https://github.com/kiorky/croniter)** — Python library for computing next runs. Library, not a CLI.
- **[cronie](https://github.com/cronie-crond/cronie)** / **vixie-cron** — The cron daemon itself. Different scope; Cronkit is a read-only auditing tool that complements whatever daemon you're running.

Cronkit's distinguishing position: a single static binary, offline, with machine-readable JSON output and severity-coded diagnostics, designed to live in pre-commit hooks and CI pipelines.

## Installation

### Quick Install (Recommended)

#### Homebrew (macOS/Linux)

```bash
brew tap hzerrad/cronkit
brew install cronkit
```

#### Direct Binary Download

Download the pre-built binary for your platform from [GitHub Releases](https://github.com/hzerrad/cronkit/releases):

```bash
# Linux (amd64)
wget https://github.com/hzerrad/cronkit/releases/download/v0.1.0/cronkit_linux_amd64.tar.gz
tar -xzf cronkit_linux_amd64.tar.gz
sudo mv cronkit /usr/local/bin/

# macOS (Apple Silicon)
wget https://github.com/hzerrad/cronkit/releases/download/v0.1.0/cronkit_darwin_arm64.tar.gz
tar -xzf cronkit_darwin_arm64.tar.gz
sudo mv cronkit /usr/local/bin/

# macOS (Intel)
wget https://github.com/hzerrad/cronkit/releases/download/v0.1.0/cronkit_darwin_amd64.tar.gz
tar -xzf cronkit_darwin_amd64.tar.gz
sudo mv cronkit /usr/local/bin/
```

### Package Managers

#### APT (Debian/Ubuntu)

Add the repository once, then install and upgrade with `apt` like any other package:

```bash
echo "deb [trusted=yes] https://apt.fury.io/hzerrad-dev/ /" | sudo tee /etc/apt/sources.list.d/cronkit.list
sudo apt update
sudo apt install cronkit
```

<details>
<summary>Or install a standalone <code>.deb</code> without adding the repo</summary>

```bash
# amd64
wget https://github.com/hzerrad/cronkit/releases/download/v0.1.0/cronkit_0.1.0_amd64.deb
sudo dpkg -i cronkit_0.1.0_amd64.deb && sudo apt-get install -f

# arm64
wget https://github.com/hzerrad/cronkit/releases/download/v0.1.0/cronkit_0.1.0_arm64.deb
sudo dpkg -i cronkit_0.1.0_arm64.deb && sudo apt-get install -f
```

</details>

#### DNF/YUM (Fedora/RHEL/CentOS)

Add the repository once, then install and upgrade with `dnf` like any other package:

```bash
sudo tee /etc/yum.repos.d/cronkit.repo >/dev/null <<'EOF'
[cronkit]
name=Cronkit
baseurl=https://yum.fury.io/hzerrad-dev/
enabled=1
gpgcheck=0
EOF
sudo dnf install cronkit
```

<details>
<summary>Or install a standalone <code>.rpm</code> without adding the repo</summary>

```bash
# x86_64
sudo dnf install https://github.com/hzerrad/cronkit/releases/download/v0.1.0/cronkit-0.1.0-1.x86_64.rpm

# aarch64
sudo dnf install https://github.com/hzerrad/cronkit/releases/download/v0.1.0/cronkit-0.1.0-1.aarch64.rpm
```

</details>

#### Pacman/AUR (Arch Linux)

Install from the Arch User Repository (AUR) using an AUR helper:

```bash
# Using yay
yay -S cronkit-bin

# Using paru
paru -S cronkit-bin

# Or manually
git clone https://aur.archlinux.org/cronkit-bin.git
cd cronkit-bin
makepkg -si
```

### Go Install

Install directly from source using Go:

```bash
go install github.com/hzerrad/cronkit/cmd/cronkit@latest
```

**Note:** This requires Go 1.25.2 or higher to be installed.

### Build from Source

Clone the repository and build:

```bash
git clone https://github.com/hzerrad/cronkit.git
cd cronkit
make build
# Binary will be in ./bin/cronkit

# Or install directly
make install
```

### Verify Installation

After installation, verify it works:

```bash
cronkit version
```

You should see the version information printed.

## Quick Start

### Explain a Cron Expression

```bash
$ cronkit explain "*/15 2-5 * * 1-5"
Runs every 15 minutes between 02:00–05:59 on weekdays (Mon–Fri).
```

### Show Next Run Times

```bash
$ cronkit next "0 9 * * *" --count 5
Next 5 runs for "0 9 * * *" (At 09:00 daily):

1. 2025-12-29 09:00:00 UTC
2. 2025-12-30 09:00:00 UTC
3. 2025-12-31 09:00:00 UTC
4. 2026-01-01 09:00:00 UTC
5. 2026-01-02 09:00:00 UTC

# With timezone support
$ cronkit next "0 9 * * *" --timezone America/New_York --count 3
Next 3 runs for "0 9 * * *" (At 09:00 daily):

1. 2025-12-29 09:00:00 EST
2. 2025-12-30 09:00:00 EST
3. 2025-12-31 09:00:00 EST
```

### List Crontab Jobs

```bash
$ cronkit list --file /etc/crontab
LINE  EXPRESSION        DESCRIPTION                          COMMAND
────  ────────────────  ───────────────────────────────────  ────────────────────────
1     0 2 * * *         At 02:00 daily                       /usr/bin/backup.sh
2     */15 * * * *      Every 15 minutes                     /usr/bin/check-disk.sh

# Read from stdin
$ cat /etc/crontab | cronkit list
# or
$ cronkit list --stdin < /etc/crontab
```

### Visualize Timeline

```bash
$ cronkit timeline "*/15 * * * *" --view day
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
$ cronkit check --file /etc/crontab
✓ All valid (2 jobs)

$ cronkit check "0 0 1 * 1" --verbose
⚠ Found 1 warning(s)
  Total jobs: 1
  Valid: 1
  Invalid: 0

⚠ WARNING: Both day-of-month and day-of-week specified (runs if either condition is met) [CRON-001]
  Expression: 0 0 1 * 1
  Hint: Consider using only day-of-month OR day-of-week, not both. Cron uses OR logic (runs if either condition is met).

# Group issues by severity
$ cronkit check --file jobs.cron --group-by severity --verbose
━━━ Error Issues (2 issue(s)) ━━━
  ...

━━━ Warning Issues (1 issue(s)) ━━━
  ...

# Use in CI/CD with fail-on
$ cronkit check --file jobs.cron --fail-on warn --verbose
# Exits with code 2 if warnings are found

$ cronkit check "60 0 * * *"
✗ Found 1 issue(s)
  Total jobs: 1
  Valid: 0
  Invalid: 1

✗ ERROR: Invalid cron expression: expected 5 fields [CRON-003]
  Expression: 60 0 * * *
  Hint: Fix the syntax error in the cron expression. Ensure all 5 fields are present and valid.
```

## Commands

Cronkit provides nine commands for working with cron expressions and crontabs:

- `explain` — Convert a cron expression to plain English
- `next` — Show the next N scheduled run times
- `list` — Parse and summarize crontab jobs
- `timeline` — ASCII timeline visualization of schedules
- `check` — Validate syntax and lint for common issues
- `doc` — Generate documentation from crontabs (Markdown, HTML, JSON)
- `stats` — Frequency, collision, and distribution statistics
- `diff` — Semantic diff between two crontabs
- `budget` — Concurrency budget analysis

See [docs/commands.md](docs/commands.md) for the complete reference with all flags, examples, and exit codes.

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
$ cronkit explain "*/15 * * * *" --json
{
  "expression": "*/15 * * * *",
  "description": "Every 15 minutes",
  "locale": "en"
}
```

**Example - Next (with timezone):**
```bash
$ cronkit next "@daily" --timezone UTC --json -c 2
{
  "expression": "@daily",
  "description": "At midnight every day",
  "timezone": "UTC",
  "locale": "en",
  "nextRuns": [
    {
      "number": 1,
      "timestamp": "2025-12-29T00:00:00Z",
      "relative": "in 6 hours"
    },
    {
      "number": 2,
      "timestamp": "2025-12-30T00:00:00Z",
      "relative": "in 1 day"
    }
  ]
}
```

**Example - Check (with severity and diagnostic codes):**
```bash
$ cronkit check "0 0 1 * 1" --json --verbose
{
  "valid": true,
  "totalJobs": 1,
  "validJobs": 1,
  "invalidJobs": 0,
  "locale": "en",
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

**Example - List (with stdin):**
```bash
$ echo "0 2 * * * /usr/bin/backup.sh" | cronkit list --json
{
  "jobs": [
    {
      "lineNumber": 1,
      "expression": "0 2 * * *",
      "command": "/usr/bin/backup.sh",
      "description": "At 02:00 daily"
    }
  ],
  "locale": "en"
}
```

## Safety

**Cronkit is read-only by design.** It never executes or modifies crontabs. It's safe to use on production systems for auditing and documentation purposes.

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
make build          # Build binary (./bin/cronkit)
make build-all      # Cross-platform builds
make install        # Install to GOPATH/bin
```

### Testing

This project follows **Test-Driven Development (TDD)** and **Behavior-Driven Development (BDD)** practices with 90%+ test coverage.

```bash
make test           # All tests (unit + integration + E2E)
make test-unit      # Unit tests only
make test-integration  # Integration tests
make test-e2e       # E2E tests
make test-coverage  # Generate coverage report
make benchmark      # Run performance benchmarks
```

**Documentation:**
- [TESTING.md](TESTING.md) - Comprehensive testing guidelines

### Code Quality

```bash
make fmt            # Format code
make vet            # Run go vet
make lint           # Run golangci-lint
make setup-hooks    # Install pre-commit hooks
```

### Project Structure

```
cronkit/
├── cmd/cronkit/          # CLI entry point
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

- [TESTING.md](TESTING.md) - Testing guidelines

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
