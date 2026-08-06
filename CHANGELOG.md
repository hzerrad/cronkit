# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.2] - 2026-08-06

### Changed
- Homebrew now ships a cask instead of a formula. GoReleaser deprecated `brews`,
  and a cask is the correct package type for prebuilt binaries. Install with
  `brew install --cask cronkit`

### Upgrade notes
- Homebrew clients older than October 2025 cannot migrate a formula to a cask in
  the same tap automatically. If `brew upgrade` does not pick up the new version,
  run `brew uninstall cronkit && brew install --cask cronkit`
- The cask clears the macOS quarantine flag on install. The binaries are not
  signed or notarised, so without this macOS reports "cronkit is damaged"
- Arch packages remain at 0.1.1 while AUR pushes are disabled upstream

## [0.1.1] - 2026-08-06

### Fixed
- `explain` no longer falls back to "Runs periodically" for list and range minute
  fields, e.g. `1,31 * * * *` now reads "At 1 and 31 minutes past the hour" (#1)
- Stepped ranges keep their bounds. `1-5/2` was described as "Every 2 minutes"
  but only runs at minutes 1, 3 and 5
- `N/step` is treated as `N-max/step`, matching the scheduler. `5/20` runs at
  minutes 5, 25 and 45, not every 20 minutes
- Stepped and list day-of-week, day-of-month and month fields are no longer
  dropped from the description. `0 9 * * */2` read as a plain daily job but runs
  Sunday, Tuesday, Thursday and Saturday
- `1-5/2` in the day-of-week field is no longer described as "weekdays (Mon-Fri)"

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

