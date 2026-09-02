# JSON Output Schemas

This document describes the JSON output format for all Cronkit commands. All JSON outputs use camelCase for field names and include a `locale` field where applicable.

## Version

**Current Version**: v0.6.0

## Common Fields

All JSON outputs may include:
- `locale` (string) - Locale used for parsing (e.g., "en", "fr")

## Command Schemas

### `explain` Command

**Command:** `cronkit explain <expression> --json`

**Schema:**
```json
{
  "expression": "string",
  "description": "string",
  "locale": "string"
}
```

**Example:**
```json
{
  "expression": "*/15 * * * *",
  "description": "Every 15 minutes",
  "locale": "en"
}
```

### `next` Command

**Command:** `cronkit next <expression> --json [--timezone <zone>]`

**Schema:**
```json
{
  "expression": "string",
  "description": "string",
  "timezone": "string",
  "locale": "string",
  "nextRuns": [
    {
      "number": "integer",
      "timestamp": "string (RFC3339)",
      "relative": "string"
    }
  ]
}
```

**Fields:**
- `timezone` - IANA timezone name (e.g., "UTC", "America/New_York")
- `nextRuns` - Array of scheduled run times
  - `number` - Sequential run number (1-based)
  - `timestamp` - ISO 8601 / RFC3339 formatted time
  - `relative` - Human-readable relative time (e.g., "in 2 hours")

**Example:**
```json
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

### `list` Command

**Command:** `cronkit list --json [--all]`

**Schema (jobs only):**
```json
{
  "jobs": [
    {
      "lineNumber": "integer",
      "expression": "string",
      "command": "string",
      "comment": "string (optional)",
      "description": "string (optional)"
    }
  ],
  "locale": "string"
}
```

**Schema (with --all flag):**
```json
{
  "entries": [
    {
      "lineNumber": "integer",
      "type": "string (JOB|COMMENT|ENV|EMPTY|INVALID)",
      "raw": "string",
      "job": {
        "expression": "string",
        "command": "string",
        "comment": "string (optional)"
      }
    }
  ],
  "locale": "string"
}
```

**Example:**
```json
{
  "jobs": [
    {
      "lineNumber": 1,
      "expression": "0 2 * * *",
      "command": "/usr/local/bin/backup.sh",
      "description": "At 02:00 daily"
    }
  ],
  "locale": "en"
}
```

### `check` Command

**Command:** `cronkit check [expression|--file <path>|--inventory <path|->] --json [--verbose]`

**Schema:**
```json
{
  "valid": "boolean",
  "totalJobs": "integer",
  "validJobs": "integer",
  "invalidJobs": "integer",
  "locale": "string",
  "issues": [
    {
      "severity": "string (error|warn|info)",
      "code": "string (e.g., CRON-001)",
      "lineNumber": "integer",
      "expression": "string",
      "message": "string",
      "hint": "string (optional)",
      "locator": {
        "file": "string",
        "line": "integer (optional)",
        "document": "integer (optional)",
        "path": "string (optional)"
      }
    }
  ]
}
```

**Fields:**
- `valid` - `true` if no errors found (warnings don't affect this)
- `totalJobs` - Total number of jobs validated
- `validJobs` - Number of valid jobs
- `invalidJobs` - Number of invalid jobs
- `issues` - Array of validation issues
  - `severity` - Issue severity level
  - `code` - Diagnostic code (e.g., "CRON-001")
  - `lineNumber` - Line number in crontab (0 for single expression); kept
    unconditionally, unchanged from before `locator` existed
  - `expression` - Cron expression (if applicable)
  - `message` - Human-readable issue description
  - `hint` - Actionable suggestion for fixing the issue
  - `locator` (optional) - additive; present only for a `--inventory` run
    whose issues span more than one file, which is the one case
    `lineNumber` alone can't tell apart. Same shape as an
    [inventory item's `locator`](#scan-command): `file` (slash-separated,
    relative to the inventory's `root`), `line` (1-indexed, omitted when
    the source format can't attribute one), `document` (0-indexed index
    into a multi-document file, omitted for a single-document file's
    first document — see the `scan` command's field notes), and `path`
    (the structural path within the document, e.g. `spec.schedule`). A
    single-source run — `--file`, `--stdin`, or the user's own crontab —
    never carries this key, byte-identical to output from before it
    existed

**Example** (two issues from the same `0 0 1 * 1` DOM/DOW conflict, one
found in a crontab and one in a Kubernetes CronJob — `locator` appears
because this run spans two files):
```json
{
  "invalidJobs": 0,
  "issues": [
    {
      "code": "CRON-001",
      "expression": "0 0 1 * 1",
      "hint": "Consider using only day-of-month OR day-of-week, not both. Cron uses OR logic (runs if either condition is met).",
      "lineNumber": 2,
      "locator": {
        "file": "crontab",
        "line": 2
      },
      "message": "Both day-of-month and day-of-week specified (runs if either condition is met)",
      "severity": "warn"
    },
    {
      "code": "CRON-001",
      "expression": "0 0 1 * 1",
      "hint": "Consider using only day-of-month OR day-of-week, not both. Cron uses OR logic (runs if either condition is met).",
      "lineNumber": 6,
      "locator": {
        "file": "deploy/cronjob.yaml",
        "line": 6,
        "path": "spec.schedule"
      },
      "message": "Both day-of-month and day-of-week specified (runs if either condition is met)",
      "severity": "warn"
    }
  ],
  "locale": "en",
  "totalJobs": 2,
  "valid": false,
  "validJobs": 2
}
```

### `timeline` Command

**Command:** `cronkit timeline [expression|--file <path>|--inventory <path|->] --json [--timezone <zone>]`

**Schema:**
```json
{
  "view": "string (day|hour)",
  "startTime": "string (RFC3339)",
  "endTime": "string (RFC3339)",
  "width": "integer",
  "timezone": "string",
  "locale": "string",
  "jobs": [
    {
      "id": "string",
      "expression": "string",
      "description": "string",
      "locator": {
        "file": "string (optional)",
        "line": "integer (optional)"
      },
      "aggregated": "integer (optional)",
      "runs": [
        {
          "time": "string (RFC3339)",
          "overlaps": "integer"
        }
      ]
    }
  ],
  "overlaps": [
    {
      "time": "string (RFC3339)",
      "count": "integer",
      "jobs": ["string"]
    }
  ],
  "overlapStats": {
    "totalWindows": "integer",
    "maxConcurrent": "integer",
    "mostProblematic": [
      {
        "time": "string (RFC3339)",
        "count": "integer",
        "jobs": ["string"]
      }
    ]
  }
}
```

**Fields:**
- `view` - Timeline view type ("day" or "hour")
- `startTime` - Start time of timeline (RFC3339)
- `endTime` - End time of timeline (RFC3339)
- `width` - Terminal width used for rendering
- `timezone` - IANA timezone name
- `jobs` - Array of jobs with their scheduled runs
  - `id` - Job identifier: the job's address in the input, so the same
    schedule keeps the same id across runs. Built from `locator` as
    `job-<file>:<line>` and suffixed `#<path>` where the source has one,
    falling back to the job's position when the locator addresses nothing
  - `expression` - Cron expression
  - `description` - Human-readable description
  - `locator` (optional) - additive per-job provenance, present whenever
    the job has a line to attribute: every crontab-derived source (the
    user's own crontab, `--stdin`, `--file`) as well as `--inventory`.
    `file` is present only once the source recorded one (`--file` or
    `--inventory`); a crontab read with no named file (the user's own
    crontab, `--stdin`) carries `line` alone. `path` is present for a
    source that addresses a schedule structurally rather than by line,
    and is what separates two schedules a flow-style YAML sequence puts
    on the same line. A job from a single cron-expression argument
    carries none of these keys, unchanged from before this field existed
  - `aggregated` (optional) - additive; present only on a collapsed
    per-file lane (more than 20 active jobs and `--expand` not given),
    giving the number of schedules that lane's `runs` union together.
    Absent on an ordinary one-job-per-lane entry
  - `runs` - Array of scheduled run times
    - `time` - Run time (RFC3339)
    - `overlaps` - Number of other jobs running at the same time
- `overlaps` - Array of overlap windows
  - `time` - Time of overlap (RFC3339)
  - `count` - Number of concurrent jobs
  - `jobs` - Array of job IDs running at this time
- `overlapStats` - Overlap statistics
  - `totalWindows` - Total number of overlap windows
  - `maxConcurrent` - Maximum number of concurrent jobs
  - `mostProblematic` - Most problematic overlap windows

**Example:**
```json
{
  "view": "day",
  "startTime": "2025-12-28T00:00:00Z",
  "endTime": "2025-12-29T00:00:00Z",
  "width": 80,
  "timezone": "UTC",
  "locale": "en",
  "jobs": [
    {
      "id": "job-1",
      "expression": "0 * * * *",
      "description": "At the start of every hour",
      "runs": [
        {
          "time": "2025-12-28T00:00:00Z",
          "overlaps": 0
        }
      ]
    }
  ],
  "overlaps": [],
  "overlapStats": {
    "totalWindows": 0,
    "maxConcurrent": 1,
    "mostProblematic": []
  }
}
```

**Example (`--inventory`, provenance):** one job from a
`cronkit scan --json | cronkit timeline --inventory -` run against a
repository with a GitHub Actions workflow; `locator` names the file and
line the schedule came from (`jobs` trimmed to one entry and `overlaps`
cleared here for brevity — a real run against that fixture returns six):
```json
{
  "view": "day",
  "startTime": "2026-08-13T00:00:00Z",
  "endTime": "2026-08-14T00:00:00Z",
  "width": 80,
  "timezone": "UTC",
  "locale": "en",
  "jobs": [
    {
      "id": "job-.github/workflows/ci.yml:5#on.schedule[0].cron",
      "expression": "0 4 * * *",
      "description": "At 04:00 every day",
      "locator": {
        "file": ".github/workflows/ci.yml",
        "line": 5,
        "path": "on.schedule[0].cron"
      },
      "runs": [
        {
          "time": "2026-08-13T04:00:00Z",
          "overlaps": 0
        }
      ]
    }
  ],
  "overlaps": [],
  "overlapStats": {
    "totalWindows": 0,
    "maxConcurrent": 0,
    "mostProblematic": []
  }
}
```

## Backward Compatibility

### Deprecated Fields

- `next_runs` in `next` command - Changed to `nextRuns` in v0.2.0

### Version History

- **v0.3.0**: Added the `scan` inventory contract, and `--inventory` input to `check`, `list`, `stats`, `budget` and `timeline`. A job id in `budget`, `stats` and `timeline` is now the schedule's address rather than its line number alone
- **v0.2.0**: Added `locale` field to all outputs, standardized field naming (camelCase), added `timezone` to timeline output
- **v0.1.0**: Initial JSON schema, covering every command shipped in the initial release

## Error Responses

All commands return JSON error responses in a consistent format:

```json
{
  "error": "string",
  "message": "string"
}
```

However, most commands output errors to stderr in plain text format for better CLI usability.

### `doc` Command

**Command:** `cronkit doc --file <path> --format <format> --json`

**Schema:**
```json
{
  "Source": "string",
  "GeneratedAt": "string (RFC3339)",
  "Jobs": [
    {
      "LineNumber": "integer",
      "Expression": "string",
      "Description": "string",
      "Command": "string",
      "Comment": "string (optional)",
      "NextRuns": [
        {
          "Time": "string (RFC3339)",
          "Relative": "string"
        }
      ],
      "Warnings": [
        {
          "Severity": "string (error|warn|info)",
          "Code": "string",
          "Message": "string",
          "Hint": "string (optional)"
        }
      ],
      "Stats": {
        "RunsPerDay": "integer",
        "RunsPerHour": "number"
      }
    }
  ],
  "Summary": {
    "TotalJobs": "integer",
    "ValidJobs": "integer",
    "InvalidJobs": "integer"
  },
  "Warnings": [
    {
      "Severity": "string",
      "Code": "string",
      "Message": "string",
      "Hint": "string (optional)"
    }
  ],
  "Statistics": {
    "TotalRunsPerDay": "integer",
    "TotalRunsPerHour": "number"
  }
}
```

**Fields:**
- `Source` - Source of the crontab (file path, "stdin", or "user crontab")
- `GeneratedAt` - Timestamp when documentation was generated (RFC3339)
- `Jobs` - Array of job documentation entries
  - `NextRuns` - Included only if `--include-next` is specified
  - `Warnings` - Included only if `--include-warnings` is specified
  - `Stats` - Included only if `--include-stats` is specified
- `Summary` - Summary statistics
- `Warnings` - Global warnings (if `--include-warnings` is specified)
- `Statistics` - Global statistics (if `--include-stats` is specified)

**Example:**
```json
{
  "Source": "/etc/crontab",
  "GeneratedAt": "2025-12-28T12:00:00Z",
  "Jobs": [
    {
      "LineNumber": 1,
      "Expression": "0 2 * * *",
      "Description": "At 02:00 daily",
      "Command": "/usr/local/bin/backup.sh",
      "NextRuns": [
        {
          "Time": "2025-12-29T02:00:00Z",
          "Relative": "in 14 hours"
        }
      ]
    }
  ],
  "Summary": {
    "TotalJobs": 1,
    "ValidJobs": 1,
    "InvalidJobs": 0
  }
}
```

### `stats` Command

**Command:** `cronkit stats --file <path> --json [--verbose] [--top <number>]`

**Schema:**
```json
{
  "TotalJobs": "integer",
  "TotalRunsPerDay": "integer",
  "TotalRunsPerHour": "number",
  "JobFrequencies": [
    {
      "Expression": "string",
      "Command": "string",
      "RunsPerDay": "integer",
      "RunsPerHour": "number"
    }
  ],
  "HourHistogram": [
    {
      "Hour": "integer (0-23)",
      "Count": "integer"
    }
  ],
  "MostFrequent": [
    {
      "Expression": "string",
      "Command": "string",
      "RunsPerDay": "integer"
    }
  ],
  "LeastFrequent": [
    {
      "Expression": "string",
      "Command": "string",
      "RunsPerDay": "integer"
    }
  ],
  "Collisions": {
    "TotalWindows": "integer",
    "MaxConcurrent": "integer",
    "BusiestHours": [
      {
        "Hour": "integer (0-23)",
        "Count": "integer",
        "Jobs": ["string"]
      }
    ]
  }
}
```

**Fields:**
- `TotalJobs` - Total number of jobs analyzed
- `TotalRunsPerDay` - Sum of all runs per day across all jobs
- `TotalRunsPerHour` - Average runs per hour
- `JobFrequencies` - Array of frequency metrics per job
- `HourHistogram` - Distribution of runs across 24 hours (included with `--verbose`)
- `MostFrequent` - Top N most frequent jobs (if `--top` is specified)
- `LeastFrequent` - Top N least frequent jobs (if `--top` is specified)
- `Collisions` - Collision analysis (included with `--verbose`)
  - `TotalWindows` - Number of time windows with overlaps
  - `MaxConcurrent` - Maximum number of concurrent jobs
  - `BusiestHours` - Hours with the most concurrent jobs

**Example:**
```json
{
  "TotalJobs": 3,
  "TotalRunsPerDay": 288,
  "TotalRunsPerHour": 12.0,
  "JobFrequencies": [
    {
      "Expression": "*/15 * * * *",
      "Command": "/usr/bin/check.sh",
      "RunsPerDay": 96,
      "RunsPerHour": 4.0
    }
  ],
  "MostFrequent": [
    {
      "Expression": "*/15 * * * *",
      "Command": "/usr/bin/check.sh",
      "RunsPerDay": 96
    }
  ],
  "Collisions": {
    "TotalWindows": 2,
    "MaxConcurrent": 3,
    "BusiestHours": [
      {
        "Hour": 0,
        "Count": 3,
        "Jobs": ["job-1", "job-2", "job-3"]
      }
    ]
  }
}
```



### `diff` Command

**Command:** `cronkit diff [old-file] [new-file] --json [flags]`

**Schema:**
```json
{
  "added": [
    {
      "type": "added",
      "expression": "string",
      "command": "string",
      "comment": "string",
      "lineNumber": "integer"
    }
  ],
  "removed": [
    {
      "type": "removed",
      "expression": "string",
      "command": "string",
      "comment": "string",
      "lineNumber": "integer"
    }
  ],
  "modified": [
    {
      "type": "modified",
      "expression": "string",
      "command": "string",
      "comment": "string",
      "lineNumber": "integer",
      "fieldsChanged": ["string"],
      "oldExpression": "string",
      "oldCommand": "string",
      "oldComment": "string",
      "oldLineNumber": "integer"
    }
  ],
  "unchanged": [
    {
      "type": "unchanged",
      "expression": "string",
      "command": "string",
      "comment": "string",
      "lineNumber": "integer"
    }
  ],
  "envChanges": [
    {
      "type": "added|removed|modified",
      "key": "string",
      "oldValue": "string",
      "newValue": "string"
    }
  ],
  "commentChanges": [
    {
      "type": "added|removed",
      "oldLine": "string",
      "newLine": "string"
    }
  ],
  "summary": {
    "added": "integer",
    "removed": "integer",
    "modified": "integer"
  },
  "generatedAt": "string (RFC3339)"
}
```

**Example:**
```json
{
  "added": [
    {
      "type": "added",
      "expression": "*/15 * * * *",
      "command": "/usr/bin/check.sh",
      "lineNumber": 2
    }
  ],
  "removed": [
    {
      "type": "removed",
      "expression": "0 1 * * *",
      "command": "/usr/bin/old.sh",
      "lineNumber": 1
    }
  ],
  "modified": [],
  "summary": {
    "added": 1,
    "removed": 1,
    "modified": 0
  },
  "generatedAt": "2026-01-04T19:00:00Z"
}
```

### `budget` Command

**Command:** `cronkit budget --file <path> --max-concurrent <number> --window <duration> --json [flags]`

**Schema:**
```json
{
  "passed": "boolean",
  "budgets": [
    {
      "name": "string",
      "maxConcurrent": "integer",
      "timeWindow": "string",
      "maxFound": "integer",
      "passed": "boolean",
      "violations": [
        {
          "Time": "string (RFC3339)",
          "Count": "integer",
          "Jobs": ["string"],
          "Budget": {
            "MaxConcurrent": "integer",
            "TimeWindow": "string (duration)",
            "Name": "string"
          }
        }
      ]
    }
  ],
  "violations": [
    {
      "time": "string (RFC3339)",
      "count": "integer",
      "jobs": ["string"],
      "budget": {
        "name": "string",
        "maxConcurrent": "integer",
        "timeWindow": "string"
      }
    }
  ],
  "generatedAt": "string (RFC3339)"
}
```

**Example:**
```json
{
  "passed": false,
  "budgets": [
    {
      "name": "max-2-per-1h",
      "maxConcurrent": 2,
      "timeWindow": "1h",
      "maxFound": 3,
      "passed": false,
      "violations": [
        {
          "Time": "2026-01-04T20:00:00Z",
          "Count": 3,
          "Jobs": ["line-1", "line-2", "line-3"],
          "Budget": {
            "MaxConcurrent": 2,
            "TimeWindow": "1h",
            "Name": "max-2-per-1h"
          }
        }
      ]
    }
  ],
  "violations": [
    {
      "time": "2026-01-04T20:00:00Z",
      "count": 3,
      "jobs": ["line-1", "line-2", "line-3"],
      "budget": {
        "name": "max-2-per-1h",
        "maxConcurrent": 2,
        "timeWindow": "1h"
      }
    }
  ],
  "generatedAt": "2026-01-04T19:00:00Z"
}
```

### `scan` Command

**Command:** `cronkit scan [paths...] --json`

This schema — the discovery inventory — is a published contract in a stronger
sense than the other commands' JSON: other tools are meant to produce and
consume it too, not just read output cronkit itself just wrote.
`cronkit check --inventory -`, and `list`, `stats`, `budget`, and `timeline`
with the same flag, read exactly this shape; see
[commands.md](commands.md#inventory-input).

**Schema:**
```json
{
  "schemaVersion": "string",
  "root": "string (optional)",
  "items": [
    {
      "expression": "string",
      "source": "string",
      "dialect": "string",
      "command": "string (optional)",
      "shell": "boolean",
      "comment": "string (optional)",
      "runAs": "string (optional)",
      "timezone": "string (optional)",
      "state": "string (active|suspended|unresolved|invalid)",
      "reason": "string (optional)",
      "concurrency": "string (optional)",
      "locator": {
        "file": "string",
        "line": "integer (optional; absent or 0 means the format cannot attribute one)",
        "document": "integer (optional)",
        "path": "string (optional)"
      }
    }
  ]
}
```

**`schemaVersion` policy:** `schemaVersion` identifies the breaking contract
only. An additive field — a new optional field an existing reader can safely
ignore — never bumps it, and readers are expected to ignore unknown fields
for exactly that reason. Only a change an old reader would misinterpret
(removing a field, renaming one, or changing what an existing field means)
bumps the version. The current value is `"1"`. A decoder that receives a
`schemaVersion` it does not recognise must reject the document rather than
guess at its shape; a missing or empty `schemaVersion` is rejected the same
way, since a document with no version cannot be told apart from one written
before this contract existed.

**Fields:**
- `schemaVersion` - the inventory contract's version; see the policy above
- `root` (optional) - the directory every `locator.file` is relative to.
  Reported relative to the invocation directory when the scanned
  repository sits at or below it — `"."` in the common case of scanning a
  repository from its own root — and as an absolute path only when it does
  not, such as a root above the working directory or on another volume
  entirely. Unlike `items`, `root` is machine-local context (which
  directory *this* invocation happened to run from), not part of the
  payload two scans of the same commit are expected to agree on — a
  consumer comparing inventories across machines or checkouts should
  ignore it rather than treat it as comparable data
- `items` - the discovered schedules; always present, `[]` when the scan
  found none (never `null`)
  - `expression` - the schedule exactly as written in the source
  - `source` - the ID of the source that produced this item: `crontab`,
    `k8s`, `argo`, or `gha`
  - `dialect` - the grammar `expression` is written in, e.g. `vixie`
  - `command` (optional) - **a shell command only when `shell` is `true`;
    a display label otherwise.** A consumer that assumes `command` is
    always executable would hand a Kubernetes container image name, or
    similar, to a shell
  - `shell` - always present; reports whether `command` is a real shell
    command, which is what gates shell-specific hygiene checks
  - `comment` (optional) - the crontab inline comment; absent for every
    other source
  - `runAs` (optional) - the account the schedule runs as, where the
    source expresses one: crontab's `USER` field, Kubernetes'
    `serviceAccountName`, and equivalents belong in this one slot
  - `timezone` (optional) - an IANA name; absent means inherit the
    invocation's own default
  - `state` - always present, one of `active`, `suspended`, `unresolved`,
    or `invalid`
  - `reason` (optional) - explains `state` when it is not `active`
  - `concurrency` (optional) - the platform's overlapping-run policy,
    e.g. `forbid` or `replace`
  - `locator` - always present; where in the source this item was found
    - `file` - always present, slash-separated, relative to `root`
    - `line` (optional) - 1-indexed; absent when the format cannot
      attribute one
    - `document` (optional) - the index of the document within a
      multi-document file; 0-indexed, and `omitempty` the same way `line`
      is — absent means 0 (the file's first document), not "not a
      multi-document file" or 1-indexed. A single-document file's items
      simply never carry the key at all, exactly like the first document
      of a multi-document one
    - `path` (optional) - the structural path within the document, e.g.
      `spec.schedule`

**Example** (two items from a larger scan run from the repository's own
root — hence `"root": "."` — showing an executed crontab command and a
non-executable Kubernetes locator):
```json
{
  "schemaVersion": "1",
  "root": ".",
  "items": [
    {
      "expression": "0 2 * * *",
      "source": "crontab",
      "dialect": "vixie",
      "command": "/usr/bin/backup.sh",
      "shell": true,
      "state": "active",
      "locator": {
        "file": "crontab",
        "line": 2
      }
    },
    {
      "expression": "30 3 * * *",
      "source": "k8s",
      "dialect": "vixie",
      "shell": false,
      "timezone": "Etc/UTC",
      "state": "active",
      "concurrency": "forbid",
      "locator": {
        "file": "deploy/cronjob.yaml",
        "line": 6,
        "path": "spec.schedule"
      }
    }
  ]
}
```

## Version History

### v0.3.0
- Added `scan` command JSON schema (the discovery inventory contract)
- `check`, `list`, `stats`, `budget`, and `timeline` all accept
  `--inventory <path|->`, reading the `scan` command's inventory
  contract described below instead of a crontab
- Added an optional `locator` to each `check` issue, additive alongside
  the existing `lineNumber`, present once an issue's source can't be
  told apart from a line number alone (a `--inventory` run spanning
  more than one file)
- Added optional per-job `locator` and `aggregated` fields to `timeline`,
  additive; `locator` is populated whenever a line can be attributed --
  every crontab-derived source (the user's own crontab, `--stdin`,
  `--file`) as well as `--inventory` -- with `file` present only once the
  source recorded one (`--file` or `--inventory`)
- A job id in `budget`, `stats` and `timeline` is now the schedule's
  address -- file, line and structural path -- rather than its line
  number alone

### v0.2.0
- Added `locale` to all outputs, standardized field naming on camelCase,
  added `timezone` to `timeline`

### v0.1.0
- Initial JSON schema documentation
- Added `explain`, `next`, `list`, `timeline`, `check`, `doc`, `stats`, `diff` and
  `budget` command schemas
