# Cronic — Human-Friendly Cron CLI  
**Version:** v0.1.0 (spec + roadmap reference)  
**Status:** Bootstrapped (design + planning phase)  
**License:** Apache 2.0 (recommended)

---

## 🧭 Overview

**Cronic** is an open-source command-line tool that makes cron jobs **human-readable**, **auditable**, and **visual**.  
It turns confusing cron syntax into plain English, generates upcoming run schedules, and draws ASCII timelines for better understanding and documentation.

It aims to be **the most accessible cron companion** for developers, DevOps engineers, and SREs who work with schedules daily.

---

## 🎯 Core Principles

| Principle | Description |
|------------|--------------|
| **Clarity** | Make cron expressions readable by humans. |
| **Safety** | Read-only by default. Never execute or modify crontabs. |
| **Portability** | Single static binary for all major OSes. |
| **Accessibility** | Works in all terminals, width-aware, color optional. |
| **Extensibility** | JSON schema and modular internals for future expansion. |
| **Community-first** | Easy to contribute to and adapt for other schedulers. |

---

## ⚙️ v0.1.0 Specification

### ✳️ Features

1. **Explain** a cron expression  
   - Converts any standard 5-field cron expression into plain English.  
   - Supports aliases (`@daily`, `@hourly`, etc.)  
   - Outputs both human-readable and JSON formats.  

2. **Next runs**  
   - Shows the next N occurrences (default: 10).  
   - Timezone-aware, ISO8601 output.

3. **List**  
   - Parses and lists cron jobs from a file or user crontab (`crontab -l`).  
   - Displays expression, plain-English summary, next run, and command.

4. **Timeline**  
   - Draws ASCII timeline (hour/day) showing job density and overlaps.  
   - Auto-scales based on terminal width.

5. **Check**  
   - Validates expressions and crontab files.  
   - Detects invalid fields, DOM/DOW conflicts, and empty schedules.

6. **JSON Mode (`--json`)**  
   - Stable, machine-readable schema for automation and CI pipelines.

---

### 🧩 CLI Overview

| Command | Description |
|----------|--------------|
| `cronic explain [EXPR]` | Explain a cron expression in English. |
| `cronic next [EXPR]` | Show next N run times. |
| `cronic list` | Parse and summarize crontab jobs. |
| `cronic timeline` | Display ASCII timeline for one day or hour. |
| `cronic check` | Validate crontab syntax and structure. |

#### Global Flags
```
--json             Output JSON
--tz <ZONE>        Timezone override
--width <COLS>     Terminal width override
--locale <LANG>    Language (default: en)
--no-color         Disable colored output
--version          Show version
```

#### Example
```bash
$ cronic explain "*/15 2-5 * * 1-5"
Runs every 15 minutes between 02:00–05:59 on weekdays (Mon–Fri).
Next runs: 02:00, 02:15, 02:30, ...
```

---

### 🧱 Supported Cron Dialect
- **Vixie cron (5-field):** `minute hour dom month dow`
- Supports aliases: `@hourly`, `@daily`, `@weekly`, `@monthly`, `@yearly`
- Case-insensitive month/day names (JAN–DEC, MON–SUN)
- Optional seconds field (`--with-seconds`)
- Not supported yet: `L`, `W`, `LW`, `#` (Quartz extensions)

---

### 🧮 Humanization Rules (EN v1)
- “Every 15 minutes between 02:00–05:59 on weekdays.”
- “At 00:00 daily.”
- “At 23:00 on the last day of each month.” *(future enhancement)*  
- “Every 2 hours on weekends.”

Each rule is template-based and locale-ready.

---

### 🧰 Internal Architecture

```
cronic/
├── cmd/cronic/          # CLI entry (Cobra or urfave/cli)
├── internal/cronx/      # Parser abstraction (wraps robfig/cron)
├── internal/human/      # Humanization templates
├── internal/plan/       # Next-run calculations, timeline logic
├── internal/render/     # Terminal table & timeline renderer
├── internal/crontab/    # Reader (system/user/file)
├── internal/check/      # Validation & linting
├── pkg/cronic/          # (optional) exported library
└── testdata/            # Cron fixtures
```

---

### 🧠 Language Choice: Go

**Chosen over Rust** for:
- Rapid iteration and cross-platform static builds.
- Mature cron and terminal libraries (`robfig/cron`, `tablewriter`).
- Easier contribution for the broader open-source community.
- Simple CI/CD pipeline via `goreleaser`.

Rust remains an option for future TUI or engine-level rewrite (v0.7+).

---

### 🧩 Dependencies (v0.1.0)
| Package | Purpose | Note |
|----------|----------|------|
| `robfig/cron/v3` | Core cron parsing | Pinned, vendored |
| `go-pretty/table` | Tables and formatting | Width-aware |
| `fatih/color` | Colored output | Optional, respects `NO_COLOR` |
| `tzdata` | Embedded timezone DB | Reproducible builds |

All dependencies are **Apache/MIT licensed**, actively maintained, and pinned.

---

## 🔐 Dependency Policy
- Third-party libraries are wrapped under `internal/cronx` (single import boundary).
- Vendored for reproducibility.
- Verified with:
  - `go vet`, `go test ./...`, `govulncheck ./...`
  - Regular Renovate/Dependabot audits
- License compliance tracked in `NOTICE.md`.

---

## ⚖️ License
**Apache License 2.0** is recommended for:
- Broad adoption (corporate and OSS safe)
- Patent protection clause
- Allows permissive reuse in CI/CD and cloud tooling
- Compatible with MIT/BSD dependencies

---

## ✅ Definition of Done (v0.1.0)
- `explain`, `next`, `list`, `timeline`, `check` fully functional
- English templates implemented; French (beta) scaffolded
- JSON schema v1 stable and documented
- CLI works offline, zero network calls
- 100% reproducible build via `goreleaser`
- Unit + snapshot tests for all parsers and renderers
- Docs: README, LICENSE, this reference

---

## 🧪 Testing Strategy
- Unit tests: parser, planner, humanizer
- Property tests: randomized cron strings → at least 1 valid occurrence
- Snapshot tests: ASCII timeline output (fixed width)
- CLI E2E tests: JSON comparison

---

## 🧭 Roadmap

### v0.2 — Quality & Reach
- Auto-detect seconds field
- Markdown export (`cronic list --md`)
- Enhanced `check`: overlapping jobs, dense schedules
- Shell completions and packaging improvements

### v0.3 — Systemd & Imports
- Read systemd timers and merge with cron data
- Unified “scheduling report” view
- `--source` column in `list`

### v0.4 — TUI (Preview)
- `cronic ui` using BubbleTea
- Interactive filters, hour/day switch
- Export PNG/SVG (headless render)

### v0.5 — Teams & CI
- JSON schema v2 (with `$schema` URL)
- GitHub Action integration (comment with timeline/lint results)
- `cronic diff old.cron new.cron`

### v0.6 — i18n & Accessibility
- Full French translation; begin Arabic support (RTL)
- High-contrast mode
- Screen-reader-friendly rendering

### v0.7 — Power User Features
- `cronic when` to show all runs in a date window
- “Explain why not” for missed jobs
- Density & overlap stats

### v0.8 — Remote & Workspace
- SSH-based crontab reading
- `.cronic.yml` workspace definitions

### v1.0 — Stable Ecosystem
- Finalized CLI and JSON contracts
- Plugin interface (importers/renderers)
- Official website & playground
- Packages: Homebrew, apt, rpm, Scoop, Chocolatey

---

## 🧩 Contribution Guidelines (future)
- Minimal dependencies, strict version pinning
- PR must include test + docs
- `go fmt`, `go vet`, `golangci-lint` enforced
- Signed commits (optional)
- “good first issue” labeled tasks

---

## 📦 Release Targets
| Platform | Artifact | Notes |
|-----------|-----------|-------|
| Linux (amd64/arm64) | `.tar.gz` | Static binary |
| macOS (amd64/arm64) | `.tar.gz` + Homebrew tap | |
| Windows (amd64) | `.zip` + Scoop manifest | |

---

## 🔭 Future Possibilities
- Online parser/visualizer (web + WASM)
- Git pre-commit hook for cron sanity checks
- Integration with Kubernetes CronJobs
- Plugin to parse `at` or `systemd` timers
- Graph-based schedule diffing

---

## 💡 Tagline Ideas
> “Cronic — Make cron human again.”  
> “Explain. Visualize. Audit. Your crontab, demystified.”  
> “Your friendly neighborhood cron translator.”

---

## 🧱 Stack Summary

| Category | Choice |
|-----------|--------|
| Language | Go |
| Build | GoReleaser |
| Parser | robfig/cron (wrapped) |
| Tests | Go testing + golden files |
| Output | Rich tables + ASCII |
| License | Apache 2.0 |
| Platforms | Linux/macOS/Windows |
| CI | GitHub Actions |

---

## 🧩 Maintainer Notes
- Focus on clean architecture, test coverage, and UX polish.
- Avoid external APIs or analytics.
- Keep releases lightweight (<10 MB binary).
- Document every public CLI flag.
- Ensure backward compatibility of `--json` output.

---

*This document serves as the internal blueprint and historical reference for Cronic’s design, purpose, and roadmap. It is not meant for public distribution but should guide development consistency through the project’s early versions.*
