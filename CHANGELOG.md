# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Project renamed from `cronkit` to `cronkit`

## [0.1.0] - 2026-01-05
### Added
- Initial release
- `explain` command - Convert cron expressions to plain English
- `next` command - Show next N scheduled run times
- `list` command - Parse and summarize cron jobs from crontab files
- `timeline` command - Visualize job schedules with ASCII timelines
- `check` command - Validate crontab syntax with severity levels and diagnostic codes
- `doc` command - Generate comprehensive documentation (Markdown, HTML, JSON)
- `stats` command - Calculate fleet statistics including run frequency metrics
- `diff` command - Compare crontabs semantically
- `budget` command - Analyze concurrency budgets
- JSON output support for all commands via `--json` flag
- Comprehensive test coverage (95%+)
- CI/CD integration with GitHub Actions
- Codecov integration

