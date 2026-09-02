# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `CRON-013` warns when a schedule lands on a day-of-month that not every
  month has. `0 0 31 * *` runs seven times a year rather than monthly, and
  the five months it skips are silent today

### Fixed
- `explain` no longer claims a schedule on day 29, 30 or 31 runs in every
  month. It now names the days a short month lacks, e.g. `0 1 1/15 * *` reads
  "on days 1, 16, and 31 of every month (31 only in months that have it)".
  Only those days are qualified, since the rest of the schedule still runs
  that month

## [0.3.1] - 2026-09-02

### Changed
- Arch users get `cronkit-bin` from the AUR again, at the current version. AUR
  uploads were skipped while pushes were frozen upstream after the August
  malware waves, which left the package stranded on 0.1.0
- `scan` no longer opens a file no source recognises. A path is offered to
  each source by name before anything is stat'ed or read, so scanning a large
  repository stops reading 8 KiB of every file in it. An unreadable or
  oversized file that no source would have parsed is no longer reported as a
  per-file problem either — it was noise, not a diagnostic

### Fixed
- A malformed YAML file reported its decode failure once per source that
  matched it by name, so a broken manifest printed the same error twice
  (`k8s` and `argo` both claim `.yaml`, and share one decode). A source that
  fails differently still gets its own problem
- `scan --help` and the command reference listed exit code `1` twice instead
  of once with both causes

## [0.3.0] - 2026-09-02

### Added
- `scan` discovers cron schedules across a whole repository instead of
  one crontab at a time. It walks crontabs, Kubernetes CronJobs, GitHub
  Actions workflows, and Argo CronWorkflows, and reports what it finds as
  a table or as JSON
- `check`, `list`, `stats`, `budget`, and `timeline` can all read the
  inventory that `scan` produces with `--inventory <path|->`, so a
  whole repository gets the same linting, listing, statistics,
  timelines, and concurrency-budget analysis a single crontab always
  had. The pipeline is `cronkit scan . --json | cronkit check --inventory -`
- `check`'s shell-hygiene rules apply only to a schedule with a real
  shell command behind it, so a Kubernetes container image or a GitHub
  Actions workflow name is never flagged as one. An overlap warning is
  also suppressed when every schedule involved already forbids running
  concurrently, since the platform serialises those runs itself
- A schedule whose timezone cannot be resolved is now reported as
  invalid, with the reason, instead of vanishing silently. A suspended
  or templated schedule is excluded from analysis and reported as
  such, rather than drawn on a `timeline` chart as though it runs
- `next --json` gains a `note` field, set only for a schedule with no
  computable next runs (currently `@reboot`, a boot trigger rather than a
  wall-clock schedule), explaining why `nextRuns` is empty

### Changed
- `timeline`'s plain-text footer now reports invalid crontab lines as
  e.g. "1 invalid job excluded" instead of silently dropping them, the
  same way a suspended or unresolvable schedule already was
- `timeline --json` now includes an optional per-job `locator` (and
  `aggregated` on a collapsed lane) for crontab input too, not only
  `--inventory` — additive, so an existing consumer is unaffected unless
  it rejects unrecognised keys
- A job id published in `budget --json`, `stats --json`, and
  `timeline --json` is now the schedule's address in the input rather
  than its line number alone: `line-<file>:<line>`, suffixed
  `#<structural path>` for a source that has one. The same schedule
  resolves to the same id on every run — previously an id could change
  when an unrelated schedule elsewhere in the repository happened to
  land on the same line number, and two schedules on one line (an Argo
  `schedules: ["...", "..."]` sequence, say) were told apart only by a
  positional suffix that moved whenever a schedule was added above them.
  This affects `--file` crontab input for these three commands;
  `--stdin` and the user's own crontab are unaffected, since neither
  carries a file to fold in
- A `--file` path no longer reaches the job id as typed: `x.cron`,
  `./x.cron` and an absolute path to the same file now all produce the
  same id, where they previously produced three
- `timeline --json` publishes the structural path on a job's `locator`,
  so every part of an id can be related back to where it came from
- An overlap finding (CRON-012) names the jobs that collided instead of
  only counting them, which is what makes it actionable once the input
  spans more than one file

### Fixed
- `@every` schedules were described as "every minute" instead of their actual
  interval. `explain "@every 1h"` said "Every minute", and `next` printed
  correct run times under that wrong description
- `@reboot` could not be explained at all; it failed to parse with an error
  instead of being recognised as a boot trigger
- An `@every` line in a crontab was silently dropped or misparsed instead
  of being recognised as an interval schedule, depending on how many words
  followed it — the line parser did not know the `@every` alias at all. A
  short line such as `@every 5m /usr/local/bin/poll.sh`, the common case,
  had too few tokens to match any line shape and vanished entirely:
  `list` reported "No cron jobs found" and `check` said "All valid". A
  line with six or more whitespace-separated tokens was instead read as
  an ordinary five-field expression, leaving both the expression and the
  command wrong — `list` showed a garbled row and `check` reported
  CRON-003
- A timezone written inline in a schedule, e.g. Kubernetes'
  `CRON_TZ=Asia/Tokyo 0 2 * * *`, is now reported in the timezone field
  instead of being left inside the expression, where it made the job
  indistinguishable from one running in UTC

### Upgrade notes
- A job id in `budget --json`, `stats --json` and `timeline --json` changed
  shape. Anything matching `line-<number>` needs updating: with a file in play
  the id is now `line-<file>:<line>`, suffixed `#<path>` for a source that
  addresses a schedule structurally. `--stdin` and the user's own crontab are
  unaffected
- `--inventory` is additive. Existing `--file` and `--stdin` flows are unchanged
- `timeline` text output gained a footer line for excluded jobs; `--json`
  consumers are unaffected unless they reject unrecognised keys

## [0.2.0] - 2026-08-06

### Changed
- `timeline` draws a lane per job instead of stacked density rows. Each job gets
  its own labelled lane, so you can tell which job runs when, and markers sit at
  their true proportional position rather than being nudged into a free cell
- The header names what is being shown — the absolute file path, the expression
  and what it means, or whose crontab — followed by the window range and the
  resolved timezone. `Local` is gone in favour of the zone and offset, e.g.
  `CET (UTC+01:00)`
- Conflicts get a row of their own beneath the lanes, marked wherever two or more
  jobs share a minute
- The chart stops widening once more columns stop adding information: a day view
  holds at 96 cells (15 minutes each), an hour view at 60 (one per minute)
- Text output is not compatible with 0.1.x. `--json` is unchanged

### Added
- `--color auto|always|never`. Auto means colour only on a terminal, and never
  when `NO_COLOR` or `TERM=dumb` is set. `--export` always writes a clean file
- `--ascii` for terminals and log pipelines that mangle box drawing
- Terminal width is read from the terminal, falling back to `$COLUMNS` and then
  80. Off a terminal the output is byte-identical between runs, so CI diffs are
  stable

### Removed
- The `░▒▓█` density scale and the legend line that explained it

### Upgrade notes
- Anything grepping timeline text needs a look. The header no longer begins with
  "cronkit timeline" and the legend line is gone
- `--json` consumers are unaffected

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

