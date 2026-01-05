# Cronic

> Make cron human again.

**Cronic** is a command-line tool that makes cron jobs human-readable, auditable, and visual. It converts confusing cron syntax into plain English, generates upcoming run schedules, provides ASCII timeline visualizations, and validates crontabs with severity levels and diagnostic codes.

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25.2%2B-blue.svg)](https://golang.org/)
[![codecov](https://codecov.io/gh/hzerrad/cronic/branch/main/graph/badge.svg)](https://codecov.io/gh/hzerrad/cronic)

## Why Cronic?

- **Works offline** - No internet connection required. Perfect for isolated environments, air-gapped systems, and CI/CD pipelines where external services aren't accessible.
- **CI/CD ready** - Machine-readable JSON output, deterministic exit codes, and comprehensive validation make it ideal for automated checks in pre-commit hooks, PR pipelines, and deployment workflows.
- **Visual timelines** - ASCII timeline visualizations help you understand schedule density, identify overlaps, and spot resource contention at a glance.
- **Enterprise-grade auditing** - Advanced linting with severity levels, diagnostic codes, frequency analysis, command hygiene checks, and concurrency budget analysis for production-ready validation.

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

# With timezone support
$ cronic next "0 9 * * *" --timezone America/New_York --count 3
Next 3 runs for "0 9 * * *" (At 09:00 daily):

1. 2025-12-29 09:00:00 EST
2. 2025-12-30 09:00:00 EST
3. 2025-12-31 09:00:00 EST
```

### List Crontab Jobs

```bash
$ cronic list --file /etc/crontab
LINE  EXPRESSION        DESCRIPTION                          COMMAND
────  ────────────────  ───────────────────────────────────  ────────────────────────
1     0 2 * * *         At 02:00 daily                       /usr/bin/backup.sh
2     */15 * * * *      Every 15 minutes                     /usr/bin/check-disk.sh

# Read from stdin
$ cat /etc/crontab | cronic list
# or
$ cronic list --stdin < /etc/crontab
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

# Group issues by severity
$ cronic check --file jobs.cron --group-by severity --verbose
━━━ Error Issues (2 issue(s)) ━━━
  ...

━━━ Warning Issues (1 issue(s)) ━━━
  ...

# Use in CI/CD with fail-on
$ cronic check --file jobs.cron --fail-on warn --verbose
# Exits with code 2 if warnings are found

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
- `--timezone <zone>` - Timezone for calculations (e.g., 'America/New_York', 'UTC', defaults to local timezone)
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
- `--stdin` - Read crontab from standard input (automatic if stdin is not a terminal)
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
- `--timezone <zone>` - Timezone for timeline (e.g., 'America/New_York', 'UTC', defaults to local timezone)
- `--width <cols>` - Terminal width (0 = auto-detect, defaults to 80 if detection fails)
- `--export <path>` - Export timeline to file (format determined by extension: .txt, .json)
- `--show-overlaps` - Show detailed overlap information in output
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
- `--stdin` - Read crontab from standard input (automatic if stdin is not a terminal)
- `-v, --verbose` - Show warnings (DOM/DOW conflicts, etc.) with diagnostic codes and hints
- `--fail-on <level>` - Severity level to fail on: `error` (default), `warn`, or `info`
- `--group-by <mode>` - Group issues by: `none` (default), `severity`, `line`, or `job`
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
- `CRON-006` - Redundant pattern (warning, e.g., `*/1` → `*`)
- `CRON-007` - Excessive runs (warning, exceeds `--max-runs-per-day` threshold)
- `CRON-008` - Missing absolute path (warning)
- `CRON-009` - Missing output redirection (warning)
- `CRON-010` - Percent character usage (warning, cron newline semantics)
- `CRON-011` - Quoting/escaping issue (warning)
- `CRON-012` - Overlap detected (warning, multiple jobs running simultaneously)

Each diagnostic includes a **hint** with actionable suggestions for fixing the issue.

**Exit Codes:**
- `0` - All valid (no errors, or only issues below the `--fail-on` threshold)
- `1` - Errors found (or configured severity level reached)
- `2` - Warnings found (when `--fail-on warn` or `--fail-on info` is used, or with `--verbose` for backward compatibility)

**Note:** Exit codes are determined by the highest severity issue found and the `--fail-on` threshold. Use `--fail-on warn` to fail on warnings in CI/CD pipelines.

**Advanced Linting Flags:**
- `--enable-frequency-checks` - Enable frequency analysis (redundant patterns, excessive runs)
- `--max-runs-per-day <number>` - Threshold for excessive runs warning (default: 1000)
- `--enable-hygiene-checks` - Enable command hygiene checks (absolute paths, redirections, %, quoting)
- `--warn-on-overlap` - Enable overlap warnings (multiple jobs running simultaneously)
- `--overlap-window <duration>` - Time window for overlap analysis (default: 24h, e.g., 1h, 24h, 48h)

### `doc`

Generate human-readable documentation from crontab files in Markdown, HTML, or JSON format.

```bash
cronic doc [flags]
cronic doc --file /etc/crontab --output docs.md
cronic doc --file crontab.txt --format html --output docs.html
cronic doc --stdin --format json --include-next 5
cronic doc --file jobs.cron --format md --include-warnings --include-stats
```

**Flags:**
- `-f, --file <path>` - Path to crontab file (defaults to user's crontab if not specified)
- `--stdin` - Read crontab from standard input
- `--format <format>` - Output format: `md` (markdown, default), `html`, or `json`
- `--output <path>` - Output file path (defaults to stdout)
- `--include-next <number>` - Include next N runs per job (default: 0, disabled)
- `--include-warnings` - Include validation warnings in documentation
- `--include-stats` - Include frequency statistics in documentation

**Example Output (Markdown):**
```markdown
# Crontab Documentation

**Source:** /etc/crontab

## Summary
- Total Jobs: 3
- Valid Jobs: 3
- Invalid Jobs: 0

## Jobs

### Job 1
- **Expression:** `0 2 * * *`
- **Description:** At 02:00 daily
- **Command:** `/usr/local/bin/backup.sh`
- **Line:** 1
```

### `stats`

Calculate and display statistics about crontab jobs including run frequency metrics, collision analysis, and hour distribution.

```bash
cronic stats [flags]
cronic stats --file /etc/crontab
cronic stats --file crontab.txt --json
cronic stats --top 10 --verbose
cronic stats --stdin --aggregate
```

**Flags:**
- `-f, --file <path>` - Path to crontab file (defaults to user's crontab if not specified)
- `--stdin` - Read crontab from standard input
- `-j, --json` - Output in JSON format
- `--verbose` - Show detailed statistics including histogram and collision details
- `--top <number>` - Show top N most frequent jobs
- `--aggregate` - Aggregate statistics from multiple sources (future use)

### `diff`

Compare two crontabs semantically to see what actually changed (jobs added, removed, or modified).

```bash
cronic diff [old-file] [new-file] [flags]
cronic diff old.cron new.cron
cronic diff --old-file old.cron --new-file new.cron --json
cronic diff --old-stdin --new-file new.cron
cronic diff old.cron new.cron --format unified
```

**Flags:**
- `--old-file <path>` - Path to old crontab file
- `--new-file <path>` - Path to new crontab file
- `--old-stdin` - Read old crontab from standard input
- `--new-stdin` - Read new crontab from standard input
- `--format <format>` - Output format: `text` (default), `json`, or `unified`
- `-j, --json` - Output in JSON format (shorthand for `--format json`)
- `--ignore-comments` - Ignore comment-only changes
- `--ignore-env` - Ignore environment variable changes
- `--show-unchanged` - Show unchanged jobs (default: false)

**Example Output:**
```
Crontab Diff
═══════════════════════════════════════════════════════════════

Added Jobs (1):
─────────────────────────────────────────────────────────────
+ */15 * * * *  /usr/bin/check.sh

Removed Jobs (1):
─────────────────────────────────────────────────────────────
- 0 1 * * *  /usr/bin/old.sh

Summary: 1 added, 1 removed, 0 modified
```

### `budget`

Analyze crontab jobs against concurrency budgets to prevent resource exhaustion.

```bash
cronic budget [flags]
cronic budget --file /etc/crontab --max-concurrent 10 --window 1m
cronic budget --file crontab.txt --max-concurrent 50 --window 1h --json
cronic budget --file jobs.cron --max-concurrent 10 --window 1m --enforce
cronic budget --stdin --max-concurrent 5 --window 1h --verbose
```

**Flags:**
- `-f, --file <path>` - Path to crontab file (defaults to user's crontab if not specified)
- `--stdin` - Read crontab from standard input
- `--max-concurrent <number>` - Maximum concurrent jobs allowed (required)
- `--window <duration>` - Time window for budget (e.g., `1m`, `1h`, `24h`) (required)
- `--enforce` - Exit with error code if budget is violated (default: report only)
- `-j, --json` - Output in JSON format
- `-v, --verbose` - Show detailed violation information

**Exit Codes:**
- `0` - All budgets passed (or report-only mode, violations shown but not failing)
- `1` - Budget violated (only when `--enforce` is used)
- `2` - Error reading/parsing crontab or budget configuration

**Example Output:**
```
Budget Analysis
═══════════════════════════════════════════════════════════════

✓ All budgets passed

Budget: max-10-per-1h
  Limit: 10 concurrent jobs
  Found: 5 concurrent jobs (max)
  Status: ✓ PASSED
```

**Example Output:**

```
Crontab Statistics
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ file is too large to read

All commands support these global flags:

- `--locale <LANG>` - Locale for parsing day/month names (default: `en`)

**Note:** The `--locale` flag affects parsing of day/month names in cron expressions. It's also included in JSON output for reference.

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
  "description": "Every 15 minutes",
  "locale": "en"
}
```

**Example - Next (with timezone):**
```bash
$ cronic next "@daily" --timezone UTC --json -c 2
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
$ cronic check "0 0 1 * 1" --json --verbose
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
$ echo "0 2 * * * /usr/bin/backup.sh" | cronic list --json
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
