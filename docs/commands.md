# Cronkit Command Reference

Complete reference for every Cronkit command, flag, and output format.

### `explain`

Convert a cron expression to plain English.

```bash
cronkit explain <cron-expression>
cronkit explain "*/15 * * * *"
cronkit explain "@daily"
cronkit explain "0 9 * * 1-5" --json
```

### `next`

Show the next N scheduled run times for a cron expression.

```bash
cronkit next <cron-expression> [flags]
cronkit next "*/15 * * * *"              # Next 10 runs (default)
cronkit next "@daily" --count 5          # Next 5 runs
cronkit next "0 9 * * 1-5" -c 3          # Next 3 runs
cronkit next "0 14 * * *" --json          # JSON output
```

**Flags:**
- `-c, --count <number>` - Number of runs to show (1-100, default: 10)
- `--timezone <zone>` - Timezone for calculations (e.g., 'America/New_York', 'UTC', defaults to local timezone)
- `-j, --json` - Output as JSON

### `list`

Parse and list cron jobs from a crontab file or the user's crontab.

```bash
cronkit list [flags]
cronkit list                              # List user's crontab
cronkit list --file /etc/crontab         # List from file
cronkit list --all                        # Include comments and env vars
cronkit list --json                       # JSON output
```

**Flags:**
- `-f, --file <path>` - Path to crontab file
- `--stdin` - Read crontab from standard input (automatic if stdin is not a terminal)
- `--inventory <path|->` - Read schedules from a `cronkit scan --json` inventory instead of a crontab: a JSON file path, or `-` for standard input. See [Inventory input](#inventory-input). Rejected together with `--all`
- `-a, --all` - Show all entries including comments and environment variables
- `-j, --json` - Output as JSON

### `scan`

Discover cron schedules across a repository: crontabs, Kubernetes CronJobs, GitHub Actions workflows, and Argo CronWorkflows.

`scan` discovers; it does not audit. There is deliberately no `--fail-on`
here — severity gating belongs to `cronkit check`. The pipeline for
auditing everything a repository holds is
`cronkit scan . --json | cronkit check --inventory -`: `scan --json` emits
the inventory contract (see [JSON_SCHEMAS.md](JSON_SCHEMAS.md)), and every
schedule-consuming command — `check`, `list`, `stats`, `budget`, and
`timeline` — reads it back with `--inventory <path|->`. See
[Inventory input](#inventory-input) below.

```bash
cronkit scan [paths...] [flags]
cronkit scan                          # scan the current directory
cronkit scan ./services ./infra       # scan multiple roots
cronkit scan --json                   # emit the inventory JSON contract
cronkit scan --source crontab,k8s     # only run these sources
cronkit scan --exclude-source gha     # run every source except this one
cronkit scan --exclude build,'*.bak'  # skip a directory (any depth) and a suffix, anywhere
cronkit scan --exclude build/gen.txt  # skip that one path only
cronkit scan --strict                 # fail the exit code on any per-file problem
```

**Flags:**
- `-j, --json` - Emit the inventory as JSON
- `--source <ids>` - Only run these sources: `crontab`, `k8s`, `argo`, or `gha` (comma-separated)
- `--exclude-source <ids>` - Run every source except these: `crontab`, `k8s`, `argo`, or `gha` (comma-separated)
- `--exclude <patterns>` - Skip matching paths (comma-separated): a pattern with a `/` matches one full path, a pattern without one matches any path segment (e.g. a directory name, at any depth)
- `--no-ignore` - Do not honour `.gitignore`
- `--max-file-size <bytes>` - Skip files larger than this many bytes (0 means no limit)
- `--strict` - Exit non-zero if the walk reported any per-file problem

**Exit Codes:**
- `0` - The walk completed, including when it found no schedules at all (a repository with none is a true answer, not a failure)
- `1` - The walk could not run at all: an unreadable root, an unknown `--source`/`--exclude-source` id, or a malformed flag; or, with `--strict`, the walk completed but reported one or more per-file problems

**Example Output:**

Run from inside a repository root laid out like the project's
`testdata/scan` fixture (a crontab file, `.github/workflows/`, a
Kubernetes manifest under `deploy/`, an Argo CronWorkflow under
`workflows/`) — not from inside the cronkit checkout itself, where
scanning `testdata/scan` in place would inherit the checkout's own
`.git` and put `.github/workflows` too many directories deep for the
`gha` source to recognise it:

```
$ cronkit scan .
LINE  PATH              SOURCE   EXPRESSION            DESCRIPTION
────  ────────────────  ───────  ────────────────────  ─────────────────────────

.github/workflows/ci.yml
5     ...edule[0].cron  gha      0 4 * * *             At 04:00 every day
6     ...edule[1].cron  gha      0 16 * * *            At 16:00 every day

crontab
2                       crontab  0 2 * * *             At 02:00 every day

deploy/cronjob.yaml
6     spec.schedule     k8s      30 3 * * *            At 03:30 every day

workflows/cronworkflow.yaml
7     ...schedules[0]   argo     0 1 * * *             At 01:00 every day
8     ...schedules[1]   argo     0 13 * * *            At 13:00 every day

6 schedule(s) across 4 file(s) from 4 source(s) (gha, crontab, k8s, argo)
0 suspended, 0 unresolved, 0 invalid
scan: deploy/broken.yaml: discover: k8s: failed to decode deploy/broken.yaml: failed to decode yaml: yaml: line 1: did not find expected ',' or ']'
scan: deploy/broken.yaml: discover: argo: failed to decode deploy/broken.yaml: failed to decode yaml: yaml: line 1: did not find expected ',' or ']'
```

This runs at the default 80-column width (no terminal attached,
`$COLUMNS` unset); the table adapts down to 40 columns, shedding the
SOURCE column and then PATH as the width narrows. Problems — here,
`deploy/broken.yaml`, a deliberately malformed manifest in the fixture —
always print to stderr, after the table, never mixed into it: a
consumer piping `--json` into `jq` never sees a diagnostic land inside
the stream. The `--json` form follows the inventory contract documented
in [JSON_SCHEMAS.md](JSON_SCHEMAS.md).

### Inventory input

`check`, `list`, `stats`, `budget`, and `timeline` all accept
`--inventory <path|->`: a JSON file produced by `cronkit scan --json`,
or `-` to read one from standard input. This is what lets any of them
audit a whole repository instead of one crontab:

```bash
$ cronkit scan . --json | cronkit check --inventory - --verbose
✓ All valid
  6 job(s) validated
```

Run from the same `testdata/scan`-shaped fixture as the `scan` example
above. `check`'s exit code and issue set are unaffected by the source —
an inventory item is checked exactly like a crontab job.

Every one of these commands resolves its input by the same precedence:
`--inventory` > `--file` > `--stdin` > the user's own crontab. Whichever
flag is given, `--inventory` wins:

```bash
$ cronkit check --inventory scan.json --file decoy.cron --verbose
# reads scan.json; decoy.cron is never opened
```

`--stdin` still means crontab text, and nothing sniffs the content to
tell the two apart — feeding an inventory JSON document to `--stdin`
(instead of `--inventory -`) is read as a very strange crontab, not
detected and redirected:

```bash
$ cronkit scan . --json | cronkit check --stdin --verbose
✗ Found 6 error(s)
  Total jobs: 6
  Valid: 0
  Invalid: 6

  Line 6: ✗ ERROR: Invalid cron expression: failed to parse expression: ...
```

This is a deliberate decision, not an oversight: `--stdin` and
`--inventory -` are two different contracts, and guessing which one a
given stream holds would make failures harder to diagnose, not easier.

A run against a multi-file `--inventory` also changes how issues are
printed and, in `--json`, adds a `locator` alongside the existing
`lineNumber`: see [`check`'s rule applicability](#check) below and
[JSON_SCHEMAS.md](JSON_SCHEMAS.md) for what `locator` carries.

`list --all` is rejected together with `--inventory`: `--all` shows
comments and environment variables, and an inventory item carries
neither (they exist only in a crontab). `diff` and `doc` do not accept
`--inventory` at all — see their sections below for why.

### `timeline`

Display an ASCII lane chart of cron job schedules — one lane per job on a shared time axis, with a `conflicts` lane marking runs that collide.

```bash
cronkit timeline [cron-expression] [flags]
cronkit timeline "*/15 * * * *"              # Timeline for single expression
cronkit timeline --file /etc/crontab         # Timeline for crontab file
cronkit timeline "*/5 * * * *" --view hour   # Hour view timeline
cronkit timeline --file jobs.cron --json     # JSON output
```

**Flags:**
- `-f, --file <path>` - Path to crontab file (defaults to user's crontab)
- `--stdin` - Read crontab from standard input
- `--inventory <path|->` - Read schedules from a `cronkit scan --json` inventory instead of a crontab: a JSON file path, or `-` for standard input. See [Inventory input](#inventory-input)
- `--view <type>` - Timeline view: `day` (24 hours) or `hour` (60 minutes, default: `day`)
- `--from <time>` - Start time for timeline (RFC3339 format, defaults to current time)
- `--timezone <zone>` - Timezone for timeline (e.g., 'America/New_York', 'UTC', defaults to local timezone)
- `--width <cols>` - Terminal width (0 = auto-detect, defaults to 80 if detection fails)
- `--export <path>` - Export timeline to file (format determined by extension: .txt, .json)
- `--show-overlaps` - Show detailed overlap information in output
- `--color <mode>` - Color output: `auto` (default, on when stdout is a TTY), `always`, or `never` (forced off when `--export` is set, even with `always`, to keep the exported file clean)
- `--ascii` - Use plain 7-bit ASCII glyphs instead of Unicode box-drawing characters
- `--expand` - Always draw one lane per job, even past 20 active jobs (default: collapse to one aggregate lane per file). See [Collapsing a large chart](#collapsing-a-large-chart)
- `--top <N>` - Cap the chart to the busiest N lanes by run count, ties broken by locator (file, then line); `0` (default) means no cap. See [Collapsing a large chart](#collapsing-a-large-chart)
- `-j, --json` - Output as JSON

**Example Output:**
```
$ cronkit timeline "*/15 * * * *" --from 2025-01-15T00:00:00Z --timezone UTC
cronkit timeline — 2025-01-15 · day · UTC

Every 15 minutes┤╷┃┃┃┃┃┃┃┃┃┃┃┃┃┃┃┃┃┃┃┃┃┃┃╷┃┃┃┃┃┃┃┃┃┃┃┃┃┃┃┃┃┃┃┃┃┃┃╷├ */15 * * * *
                └┬───────────┬───────────┬───────────┬───────────┬┘
                00:00      06:00       12:00       18:00      23:59

1 job · 95 runs · no conflicts
```

With `--show-overlaps` and more than one job, an `overlaps:` section lists each colliding time and the jobs involved. When the window has no runs, the command prints `no runs in this window` and exits `0`.

**CI/non-interactive behavior:** when stdout isn't a terminal, output is plain (no ANSI escapes) and defaults to 80 columns unless `--width` or `$COLUMNS` say otherwise.

**Suspended and unresolved schedules** (a Kubernetes `spec.suspend: true`,
or an expression still holding a template placeholder like
`{{ .Values.cron }}`) never reach the chart — they can't be drawn as
though they run — but they aren't dropped silently either: the footer
counts them, at the default 80-column width used above:

```
$ cronkit scan . --json | cronkit timeline --inventory - --from 2026-08-13T00:00:00Z --timezone UTC
inventory from stdin
2026-08-13  00:00 → 23:59 · UTC

active  ┤     ╷                                                      ├ 0 2 * * *
        └┬──────────────┬──────────────┬──────────────┬─────────────┬┘
        00:00         06:00          12:00          18:00        23:59

1 job · 1 run · no conflicts · 1 suspended job, 1 unresolved job excluded
```

from a scan of three Kubernetes manifests: one plain schedule, one
`suspend: true`, and one with a templated `spec.schedule`.

**Cross-file and cross-zone provenance.** Once a chart spans more than
one file, the right-hand gutter switches from the expression to
`file:line` (truncated with a leading `...` when it doesn't fit), and
each lane label is deduped against the others by locator instead of
just the command basename. When an item declares its own `timezone`
that differs from the chart's axis zone, its runs are converted onto
the axis before plotting, and the window line names every zone that
happened to, so the conversion is visible rather than silent:

```
$ cronkit scan . --json | cronkit timeline --inventory - --from 2026-08-13T00:00:00Z --timezone UTC
inventory from stdin
2026-08-13  00:00 → 23:59 · UTC · converted from America/New_York, Etc/UTC

ci:5          ┤       ╷                                      ├ ...flows/ci.yml:5
ci:6          ┤                              ╷               ├ ...flows/ci.yml:6
backup.sh     ┤   ╷                                          ├ crontab:2
cronjob       ┤      ╷                                       ├ ...cronjob.yaml:6
cronworkflow:7┤         ╷                                    ├ ...orkflow.yaml:7
cronworkflow:8┤                                ╷             ├ ...orkflow.yaml:8
              └┬──────────┬───────────┬──────────┬──────────┬┘
              00:00     06:00       12:00      18:00     23:59

6 jobs · 6 runs · no conflicts
```

at the default 80-column width, from the same `testdata/scan` fixture
used by the `scan` command's example above (run the same way: from
inside a repository root laid out like that fixture, not from inside
the cronkit checkout itself). The Argo CronWorkflow's two schedules
declare `America/New_York`; every other source in this fixture is UTC
(the Kubernetes CronJob's `Etc/UTC` is named too, since it differs in
spelling even though not in offset) — only zones that differ from the
axis are listed, and a chart with none declares no `converted from`
clause at all, unchanged from before this existed.

`--json` carries the same per-job `locator` (`file`, `line`) the text
gutter draws from, plus an `aggregated` count on a collapsed per-file
lane — both additive fields; see [JSON_SCHEMAS.md](JSON_SCHEMAS.md).

### Collapsing a large chart

Past 20 active jobs, `timeline` stops drawing one lane per job — a
wall of lanes stops reading as a chart — and collapses to one
aggregate lane per file instead, each lane's runs the union of every
job in that file. The footer says so, and names `--expand` as the way
back:

```
$ cronkit timeline --inventory jobs.json --width 150 --from 2026-08-13T00:00:00Z --timezone UTC
/home/user/jobs.json
2026-08-13  00:00 → 23:59 · UTC

service-00/crontab┤┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ├ 7 jobs
service-01/crontab┤┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ├ 6 jobs
service-02/crontab┤┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ├ 6 jobs
service-03/crontab┤┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ├ 6 jobs
                  └┬───────────────────────┬───────────────────────┬───────────────────────┬──────────────────────┬┘
                  00:00                  06:00                   12:00                   18:00                 23:59

4 jobs · 599 runs · no conflicts · 25 jobs collapsed into 4 file lanes (--expand to show all)
```

at 150 columns, from a 25-job inventory spread across 4 files (`jobs.json`
here stands in for whatever `--inventory` path or `-` a reader uses — this
run used one). Exactly 20 active jobs stays per-job; the threshold is
"more than 20," not "20 or more."

`--expand` forces per-job lanes back, regardless of how many result —
this is the same 25-job inventory, at the default 80-column width:

```
$ cronkit timeline --inventory jobs.json --expand --from 2026-08-13T00:00:00Z --timezone UTC
/home/user/jobs.json
2026-08-13  00:00 → 23:59 · UTC

job-000.sh┤  ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷  ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷  ├ ...e-00/crontab:1
job-004.sh┤╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷  ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷  ╷ ├ ...e-00/crontab:5
...
job-023.sh┤╷ ╷ ╷  ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷  ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ╷ ├ ...-03/crontab:24
          └┬───────────┬────────────┬───────────┬───────────┬┘
          00:00      06:00        12:00       18:00      23:59

25 jobs · 599 runs · no conflicts
```

(all 25 lanes shown when run for real; elided here with `...` since
every row follows the same shape). `--top N` caps whichever set of
lanes was about to render — collapsed or expanded — to the `N`
busiest by run count, ties broken by locator, and reports how many
were left out:

```
$ cronkit timeline --inventory jobs8.json --top 3 --width 150 --from 2026-08-13T00:00:00Z --timezone UTC
/home/user/jobs8.json
2026-08-13  00:00 → 23:59 · UTC

service-00/crontab┤╷┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ┃┃  ├ 4 jobs
service-01/crontab┤┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ├ 3 jobs
service-02/crontab┤┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ┃╷  ├ 3 jobs
                  └┬───────────────────────┬───────────────────────┬───────────────────────┬──────────────────────┬┘
                  00:00                  06:00                   12:00                   18:00                 23:59

3 jobs · 239 runs · no conflicts · 25 jobs collapsed into 8 file lanes (--expand to show all) · 5 lanes hidden (--top 3)
```

(`jobs8.json` here is the same shape of inventory, spread across 8
files instead of 4, to make `--top` visibly cut lanes rather than
leave all of them showing). `--top` applies after the collapse
decision, not instead of it: it caps whichever set — file lanes here,
job lanes under `--expand` — was already going to render.

### `check`

Validate crontab syntax and detect common issues with severity levels and diagnostic codes.

```bash
cronkit check [cron-expression|--file <path>] [flags]
cronkit check "0 0 * * *"                  # Validate single expression
cronkit check --file /etc/crontab         # Validate crontab file
cronkit check "0 0 1 * 1" --verbose       # Show warnings with diagnostic codes
cronkit check --file jobs.cron --json     # JSON output
```

**Flags:**
- `-f, --file <path>` - Path to crontab file
- `--stdin` - Read crontab from standard input (automatic if stdin is not a terminal)
- `--inventory <path|->` - Read schedules from a `cronkit scan --json` inventory instead of a crontab: a JSON file path, or `-` for standard input. See [Inventory input](#inventory-input)
- `-v, --verbose` - Show warnings (DOM/DOW conflicts, etc.) with diagnostic codes and hints
- `--fail-on <level>` - Severity level to fail on: `error` (default), `warn`, or `info`
- `--group-by <mode>` - Group issues by: `none` (default), `severity`, `line`, or `job`
- `-j, --json` - Output as JSON

**Severity Levels:**
- **Error** (`✗ ERROR`) - Invalid expressions or critical issues that prevent execution
- **Warning** (`⚠ WARNING`) - Potential issues that may cause unexpected behavior
- **Info** (`ℹ INFO`) - Informational messages

**Diagnostic Codes:**
- `CRON-001` - DOM/DOW conflict (warning)
- `CRON-002` - Empty schedule (error)
- `CRON-003` - Parse error (error)
- `CRON-004` - File read error (error)
- `CRON-005` - Invalid crontab structure (error)
- `CRON-006` - Redundant pattern (warning, e.g., `*/1` → `*`)
- `CRON-007` - Excessive runs (warning, exceeds `--max-runs-per-day` threshold)
- `CRON-008` - Missing absolute path (info)
- `CRON-009` - Missing output redirection (info)
- `CRON-010` - Percent character usage (warning, cron newline semantics)
- `CRON-011` - Quoting/escaping issue (warning)
- `CRON-012` - Overlap detected (warning, multiple jobs running simultaneously)
- `CRON-013` - Day-of-month a short month lacks (warning, e.g. `0 0 31 * *` runs seven times a year, not twelve)

Each diagnostic includes a **hint** with actionable suggestions for fixing the issue.

**Rule applicability across sources:** a schedule read from `--inventory`
does not always have a real shell command behind it — a Kubernetes
CronJob names a container image, a GitHub Actions workflow names itself
— so two of these codes are gated by source, not just by flag:

- `CRON-008` through `CRON-011` (the `--enable-hygiene-checks` group)
  only run against a schedule whose inventory item says `shell: true`.
  A crontab job is always `shell: true`; a Kubernetes CronJob or GitHub
  Actions workflow is not, so its `command` — an image name, a workflow
  name — is never linted as if it were a shell command
- `CRON-012` (`--warn-on-overlap`) is suppressed for an overlapping
  group only when *every* schedule in it declares
  `concurrencyPolicy: Forbid` (or the Argo/Kubernetes equivalent): the
  platform already serialises those runs against each other, so nothing
  can actually pile up, and reporting it would be a finding with no
  schedule to change. An overlap that mixes a forbidding schedule with
  one that allows or replaces concurrent runs — or a crontab entry,
  which has no concurrency policy at all — is still reported, since
  that mix genuinely can run at once. For example, two `Forbid`
  schedules colliding at the same minute produce no finding, but adding
  a plain crontab job at that same minute does:

  ```bash
  $ cronkit check --inventory overlap-mixed.json --warn-on-overlap --verbose
  ⚠ Found 1 warning(s)
    Total jobs: 2
    Valid: 2
    Invalid: 0

    ⚠ WARNING: Overlap detected: 2 jobs scheduled at 2026-08-14 03:00: line-crontab:2, line-deploy/cronjob.yaml:6#spec.schedule [CRON-012]
      Hint: Multiple jobs are scheduled to run at the same time. This may cause resource contention. Consider adjusting schedules to distribute load.
  ```

  The finding names the jobs it involves, by the same id `budget --json`
  and `timeline --json` publish, so a collision in a repository-wide scan
  points at the two files that caused it. Where `overlap-mixed.json`
  pairs one `Forbid` Kubernetes schedule with one plain crontab entry at
  the same time; two `Forbid` schedules
  alone at that same minute report `✓ All valid` instead. `check` has
  no `--from`, so the date it prints is whenever the command runs, not
  a fixed one — only the date changes between runs, not the finding.

**Issue locators:** once an issue's source can't be told apart from its
line number alone — a `--inventory` run spanning more than one file —
`check` prefixes each line with `file:line` (and, when the source
recorded one, ` (structural.path)`) instead of `Line N`, and `--json`
adds a `locator` object beside the existing `lineNumber`:

```bash
$ cronkit check --inventory locator-demo.json --verbose
⚠ Found 2 warning(s)
  Total jobs: 2
  Valid: 2
  Invalid: 0

  crontab:2: ⚠ WARNING: Both day-of-month and day-of-week specified (runs if either condition is met) [CRON-001]
    Expression: 0 0 1 * 1
    Hint: Consider using only day-of-month OR day-of-week, not both. Cron uses OR logic (runs if either condition is met).
  deploy/cronjob.yaml:6 (spec.schedule): ⚠ WARNING: Both day-of-month and day-of-week specified (runs if either condition is met) [CRON-001]
    Expression: 0 0 1 * 1
    Hint: Consider using only day-of-month OR day-of-week, not both. Cron uses OR logic (runs if either condition is met).
```

A single-source run — `--file`, `--stdin`, or the user's own crontab —
keeps the plain `Line N` form and no `locator` key, byte-identical to
output from before `--inventory` existed. See
[JSON_SCHEMAS.md](JSON_SCHEMAS.md) for the `locator` object's fields.

**A schedule whose timezone cannot be resolved** is reported as an
invalid schedule with the reason, not silently dropped:

```bash
$ cronkit check --inventory bad-tz.json --verbose
✗ Found 1 error(s)
  Total jobs: 1
  Valid: 0
  Invalid: 1

  Line 6: ✗ ERROR: Invalid cron expression: unresolvable timezone "Nowhere/Fake": unknown time zone Nowhere/Fake [CRON-003]
    Expression: 0 2 * * *
    Hint: Fix the syntax error in the cron expression. Ensure all 5 fields are present and valid.
```

A suspended schedule (Kubernetes `spec.suspend: true`) or one whose
expression is still a template placeholder (e.g. `{{ .Values.cron }}`)
is counted as valid but excluded from every check that needs to
evaluate a schedule — DOM/DOW, empty-schedule, frequency, hygiene, and
overlap — since it isn't wrong, it just isn't running right now.

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

**`doc` does not accept `--inventory`.** Unlike `check`, `list`,
`stats`, `budget`, and `timeline`, it was kept on `--file`/`--stdin`
(or the user's own crontab) in this round, reading the full parsed
crontab structure the same way `diff` does — see [`diff`](#diff) below
for the shape of that structure and why it resists collapsing into a
single `inventory.Item` per schedule.

```bash
cronkit doc [flags]
cronkit doc --file /etc/crontab --output docs.md
cronkit doc --file crontab.txt --format html --output docs.html
cronkit doc --stdin --format json --include-next 5
cronkit doc --file jobs.cron --format md --include-warnings --include-stats
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
cronkit stats [flags]
cronkit stats --file /etc/crontab
cronkit stats --file crontab.txt --json
cronkit stats --top 10 --verbose
cronkit stats --stdin --aggregate
```

**Flags:**
- `-f, --file <path>` - Path to crontab file (defaults to user's crontab if not specified)
- `--stdin` - Read crontab from standard input
- `--inventory <path|->` - Read schedules from a `cronkit scan --json` inventory instead of a crontab: a JSON file path, or `-` for standard input. See [Inventory input](#inventory-input)
- `-j, --json` - Output in JSON format
- `--verbose` - Show detailed statistics including histogram and collision details
- `--top <number>` - Show top N most frequent jobs
- `--aggregate` - Aggregate statistics from multiple sources (future use)

### `diff`

Compare two crontabs semantically to see what actually changed (jobs added, removed, or modified).

**`diff` does not accept `--inventory`.** It reports environment
variable changes (`envChanges`) and standalone comment-line changes
(`commentChanges`) by comparing the two crontabs' full parsed structure
— `crontab.Entry`, which has its own entries for `ENV` and `COMMENT`
lines, not just jobs. An `inventory.Item` represents a single schedule,
with no room for a line that isn't one, so migrating `diff` onto it
would silently drop both kinds of change from its output. `diff` keeps
its own `--old-file`/`--new-file`/`--old-stdin`/`--new-stdin` flags.

```bash
cronkit diff [old-file] [new-file] [flags]
cronkit diff old.cron new.cron
cronkit diff --old-file old.cron --new-file new.cron --json
cronkit diff --old-stdin --new-file new.cron
cronkit diff old.cron new.cron --format unified
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
cronkit budget [flags]
cronkit budget --file /etc/crontab --max-concurrent 10 --window 1m
cronkit budget --file crontab.txt --max-concurrent 50 --window 1h --json
cronkit budget --file jobs.cron --max-concurrent 10 --window 1m --enforce
cronkit budget --stdin --max-concurrent 5 --window 1h --verbose
```

**Flags:**
- `-f, --file <path>` - Path to crontab file (defaults to user's crontab if not specified)
- `--stdin` - Read crontab from standard input
- `--inventory <path|->` - Read schedules from a `cronkit scan --json` inventory instead of a crontab: a JSON file path, or `-` for standard input. See [Inventory input](#inventory-input)
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

All commands support these global flags:

- `--locale <LANG>` - Locale for parsing day/month names (default: `en`)

**Note:** The `--locale` flag affects parsing of day/month names in cron expressions. It's also included in JSON output for reference.
