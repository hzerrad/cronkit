# JSON Output Schemas

This document describes the JSON output format for all Cronic commands. All JSON outputs use camelCase for field names and include a `locale` field where applicable.

## Version

**Current Version**: v0.4.0

## Common Fields

All JSON outputs may include:
- `locale` (string) - Locale used for parsing (e.g., "en", "fr")

## Command Schemas

### `explain` Command

**Command:** `cronic explain <expression> --json`

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

**Command:** `cronic next <expression> --json [--timezone <zone>]`

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

**Command:** `cronic list --json [--all]`

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

**Command:** `cronic check [expression|--file <path>] --json [--verbose]`

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
      "hint": "string (optional)"
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
  - `lineNumber` - Line number in crontab (0 for single expression)
  - `expression` - Cron expression (if applicable)
  - `message` - Human-readable issue description
  - `hint` - Actionable suggestion for fixing the issue

**Example:**
```json
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
      "type": "warn"
    }
  ]
}
```

### `timeline` Command

**Command:** `cronic timeline [expression|--file <path>] --json [--timezone <zone>]`

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
  - `id` - Job identifier
  - `expression` - Cron expression
  - `description` - Human-readable description
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

## Backward Compatibility

### Deprecated Fields

- `next_runs` in `next` command - Changed to `nextRuns` in v0.2.0

### Version History

- **v0.3.0**: Added `doc` and `stats` command schemas, enhanced `check` command with new diagnostic codes (CRON-006 through CRON-012)
- **v0.2.0**: Added `locale` field to all outputs, standardized field naming (camelCase), added `timezone` to timeline output
- **v0.1.0**: Initial JSON schema

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

**Command:** `cronic doc --file <path> --format <format> --json`

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

**Command:** `cronic stats --file <path> --json [--verbose] [--top <number>]`

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

**Command:** `cronic diff [old-file] [new-file] --json [flags]`

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

**Command:** `cronic budget --file <path> --max-concurrent <number> --window <duration> --json [flags]`

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

## Version History

### v0.4.0
- Added `diff` command JSON schema
- Added `budget` command JSON schema

### v0.3.0
- Added `doc` command JSON schema
- Added `stats` command JSON schema

### v0.2.0
- Initial JSON schema documentation
- Added `explain`, `next`, `list`, `timeline`, and `check` command schemas
