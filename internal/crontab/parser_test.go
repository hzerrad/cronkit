package crontab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseLine_ValidJobs tests parsing valid cron job lines
func TestParseLine_ValidJobs(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		lineNumber  int
		wantType    EntryType
		wantExpr    string
		wantCommand string
		wantComment string
	}{
		{
			name:        "simple daily job",
			line:        "0 0 * * * /usr/bin/backup.sh",
			lineNumber:  1,
			wantType:    EntryTypeJob,
			wantExpr:    "0 0 * * *",
			wantCommand: "/usr/bin/backup.sh",
		},
		{
			name:        "job with inline comment",
			line:        "*/15 * * * * /usr/bin/check.sh # Health check",
			lineNumber:  5,
			wantType:    EntryTypeJob,
			wantExpr:    "*/15 * * * *",
			wantCommand: "/usr/bin/check.sh",
			wantComment: "Health check",
		},
		{
			name:        "job with complex command",
			line:        "0 2 * * * cd /var/log && tar -czf backup.tar.gz *.log",
			lineNumber:  10,
			wantType:    EntryTypeJob,
			wantExpr:    "0 2 * * *",
			wantCommand: "cd /var/log && tar -czf backup.tar.gz *.log",
		},
		{
			name:        "job with spaces in command",
			line:        `0 0 * * * /usr/bin/script.sh "arg with spaces"`,
			lineNumber:  3,
			wantType:    EntryTypeJob,
			wantExpr:    "0 0 * * *",
			wantCommand: `/usr/bin/script.sh "arg with spaces"`,
		},
		{
			name:        "job with alias",
			line:        "@daily /usr/bin/daily-task.sh",
			lineNumber:  7,
			wantType:    EntryTypeJob,
			wantExpr:    "@daily",
			wantCommand: "/usr/bin/daily-task.sh",
		},
		{
			name:        "job with @reboot",
			line:        "@reboot /usr/bin/startup.sh",
			lineNumber:  1,
			wantType:    EntryTypeJob,
			wantExpr:    "@reboot",
			wantCommand: "/usr/bin/startup.sh",
		},
		{
			name:        "job with @every",
			line:        "@every 1h /usr/bin/poll.sh",
			lineNumber:  1,
			wantType:    EntryTypeJob,
			wantExpr:    "@every 1h",
			wantCommand: "/usr/bin/poll.sh",
		},
		{
			name:        "job with only expression no command",
			line:        "0 0 * * *", // Only expression, no command - exprEnd will be 0
			lineNumber:  1,
			wantType:    EntryTypeInvalid,
			wantExpr:    "",
			wantCommand: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := ParseLine(tt.line, tt.lineNumber)

			assert.Equal(t, tt.wantType, entry.Type)
			if tt.wantType == EntryTypeJob {
				require.NotNil(t, entry.Job, "Job should not be nil for EntryTypeJob")
				assert.Equal(t, tt.lineNumber, entry.LineNumber)
				assert.Equal(t, tt.wantExpr, entry.Job.Expression)
				assert.Equal(t, tt.wantCommand, entry.Job.Command)
				if tt.wantComment != "" {
					assert.Equal(t, tt.wantComment, entry.Job.Comment)
				}
			}
		})
	}
}

// TestParseLine_EveryDescriptor tests "@every <duration> command", the only descriptor spanning two tokens.
func TestParseLine_EveryDescriptor(t *testing.T) {
	t.Run("valid interval with inline comment", func(t *testing.T) {
		entry := ParseLine("@every 1h /usr/bin/poll.sh # hourly poll", 1)

		require.Equal(t, EntryTypeJob, entry.Type)
		require.NotNil(t, entry.Job)
		assert.Equal(t, "@every 1h", entry.Job.Expression)
		assert.Equal(t, "/usr/bin/poll.sh", entry.Job.Command)
		assert.Equal(t, "hourly poll", entry.Job.Comment)
		assert.True(t, entry.Job.Valid)
		assert.Empty(t, entry.Job.Error)
	})

	t.Run("tabs and repeated spaces between tokens", func(t *testing.T) {
		entry := ParseLine("@every\t1h30m\t\t/usr/bin/poll.sh", 1)

		require.Equal(t, EntryTypeJob, entry.Type)
		require.NotNil(t, entry.Job)
		assert.Equal(t, "@every 1h30m", entry.Job.Expression)
		assert.Equal(t, "/usr/bin/poll.sh", entry.Job.Command)
		assert.True(t, entry.Job.Valid)
	})

	t.Run("missing duration and command is reported invalid, not dropped", func(t *testing.T) {
		entry := ParseLine("@every", 1)

		require.Equal(t, EntryTypeJob, entry.Type, "an @every line must never be silently dropped")
		require.NotNil(t, entry.Job)
		assert.Equal(t, "@every", entry.Job.Expression)
		assert.Empty(t, entry.Job.Command)
		assert.False(t, entry.Job.Valid)
		assert.NotEmpty(t, entry.Job.Error)
	})

	t.Run("missing duration but a command follows is reported invalid, not dropped", func(t *testing.T) {
		entry := ParseLine("@every /usr/bin/poll.sh", 1)

		require.Equal(t, EntryTypeJob, entry.Type, "an @every line must never be silently dropped")
		require.NotNil(t, entry.Job)
		assert.Equal(t, "@every /usr/bin/poll.sh", entry.Job.Expression)
		assert.False(t, entry.Job.Valid)
		assert.NotEmpty(t, entry.Job.Error)
	})

	t.Run("unparseable duration is reported invalid, not dropped", func(t *testing.T) {
		entry := ParseLine("@every nonsense /usr/bin/poll.sh", 1)

		require.Equal(t, EntryTypeJob, entry.Type, "an @every line must never be silently dropped")
		require.NotNil(t, entry.Job)
		assert.Equal(t, "@every nonsense", entry.Job.Expression)
		assert.Equal(t, "/usr/bin/poll.sh", entry.Job.Command)
		assert.False(t, entry.Job.Valid)
		assert.NotEmpty(t, entry.Job.Error)
	})

	t.Run("valid duration but no command is reported invalid, not silently active", func(t *testing.T) {
		entry := ParseLine("@every 1h", 1)

		require.Equal(t, EntryTypeJob, entry.Type)
		require.NotNil(t, entry.Job)
		assert.Equal(t, "@every 1h", entry.Job.Expression)
		assert.Empty(t, entry.Job.Command)
		assert.False(t, entry.Job.Valid, "a schedule with no command is invalid, the same as a bare @reboot")
		assert.Contains(t, entry.Job.Error, "no command")
	})

	t.Run("other aliases are unaffected", func(t *testing.T) {
		entry := ParseLine("@hourly /usr/bin/hourly-task.sh", 1)

		require.Equal(t, EntryTypeJob, entry.Type)
		require.NotNil(t, entry.Job)
		assert.Equal(t, "@hourly", entry.Job.Expression)
		assert.Equal(t, "/usr/bin/hourly-task.sh", entry.Job.Command)
		assert.True(t, entry.Job.Valid)
	})
}

// TestParseLine_Comments tests parsing comment lines
func TestParseLine_Comments(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		lineNumber int
	}{
		{
			name:       "simple comment",
			line:       "# This is a comment",
			lineNumber: 1,
		},
		{
			name:       "comment with leading spaces",
			line:       "  # Indented comment",
			lineNumber: 5,
		},
		{
			name:       "comment with special chars",
			line:       "# TODO: Fix this @midnight job!",
			lineNumber: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := ParseLine(tt.line, tt.lineNumber)

			assert.Equal(t, EntryTypeComment, entry.Type)
			assert.Equal(t, tt.lineNumber, entry.LineNumber)
			assert.Equal(t, tt.line, entry.Raw)
			assert.Nil(t, entry.Job, "Job should be nil for comments")
		})
	}
}

// TestParseLine_EnvVars tests parsing environment variable lines
func TestParseLine_EnvVars(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		lineNumber int
	}{
		{
			name:       "simple env var",
			line:       "PATH=/usr/local/bin:/usr/bin",
			lineNumber: 1,
		},
		{
			name:       "SHELL env var",
			line:       "SHELL=/bin/bash",
			lineNumber: 2,
		},
		{
			name:       "env var with spaces",
			line:       `MAILTO="admin@example.com"`,
			lineNumber: 3,
		},
		{
			name:       "HOME env var",
			line:       "HOME=/home/user",
			lineNumber: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := ParseLine(tt.line, tt.lineNumber)

			assert.Equal(t, EntryTypeEnvVar, entry.Type)
			assert.Equal(t, tt.lineNumber, entry.LineNumber)
			assert.Equal(t, tt.line, entry.Raw)
			assert.Nil(t, entry.Job, "Job should be nil for env vars")
		})
	}
}

// TestParseLine_EmptyLines tests parsing empty or whitespace lines
func TestParseLine_EmptyLines(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		lineNumber int
	}{
		{
			name:       "completely empty",
			line:       "",
			lineNumber: 1,
		},
		{
			name:       "only spaces",
			line:       "    ",
			lineNumber: 2,
		},
		{
			name:       "only tabs",
			line:       "\t\t",
			lineNumber: 3,
		},
		{
			name:       "mixed whitespace",
			line:       " \t  \t ",
			lineNumber: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := ParseLine(tt.line, tt.lineNumber)

			assert.Equal(t, EntryTypeEmpty, entry.Type)
			assert.Equal(t, tt.lineNumber, entry.LineNumber)
			assert.Nil(t, entry.Job, "Job should be nil for empty lines")
		})
	}
}

// TestParseLine_InvalidJobs tests parsing invalid cron job lines
func TestParseLine_InvalidJobs(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		lineNumber int
	}{
		{
			name:       "too few fields",
			line:       "0 0 * *",
			lineNumber: 1,
		},
		{
			name:       "only expression no command",
			line:       "0 0 * * *",
			lineNumber: 2,
		},
		{
			name:       "invalid minute value",
			line:       "60 0 * * * /usr/bin/test.sh",
			lineNumber: 3,
		},
		{
			name:       "garbage input",
			line:       "not a cron job at all",
			lineNumber: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := ParseLine(tt.line, tt.lineNumber)

			// Should either be Invalid or Job with Valid=false
			if entry.Type == EntryTypeJob {
				require.NotNil(t, entry.Job)
				assert.False(t, entry.Job.Valid, "Invalid job should have Valid=false")
				assert.NotEmpty(t, entry.Job.Error, "Invalid job should have Error message")
			} else {
				assert.Equal(t, EntryTypeInvalid, entry.Type)
			}
			assert.Equal(t, tt.lineNumber, entry.LineNumber)
		})
	}
}

// TestParseLine_EdgeCases tests edge cases
func TestParseLine_EdgeCases(t *testing.T) {
	t.Run("job with tab separators", func(t *testing.T) {
		line := "0\t0\t*\t*\t*\t/usr/bin/test.sh"
		entry := ParseLine(line, 1)

		assert.Equal(t, EntryTypeJob, entry.Type)
		require.NotNil(t, entry.Job)
		assert.Equal(t, "0 0 * * *", entry.Job.Expression)
		assert.Equal(t, "/usr/bin/test.sh", entry.Job.Command)
	})

	t.Run("job with multiple spaces between fields", func(t *testing.T) {
		line := "0    0    *    *    *    /usr/bin/test.sh"
		entry := ParseLine(line, 1)

		assert.Equal(t, EntryTypeJob, entry.Type)
		require.NotNil(t, entry.Job)
		assert.Equal(t, "0 0 * * *", entry.Job.Expression)
	})

	t.Run("comment that looks like job", func(t *testing.T) {
		line := "# 0 0 * * * /usr/bin/disabled.sh"
		entry := ParseLine(line, 1)

		assert.Equal(t, EntryTypeComment, entry.Type)
		assert.Nil(t, entry.Job)
	})

	t.Run("alias job with inline comment", func(t *testing.T) {
		line := "@daily /usr/bin/daily-task.sh # Daily backup"
		entry := ParseLine(line, 1)

		assert.Equal(t, EntryTypeJob, entry.Type)
		require.NotNil(t, entry.Job)
		assert.Equal(t, "@daily", entry.Job.Expression)
		assert.Equal(t, "/usr/bin/daily-task.sh", entry.Job.Command)
		assert.Equal(t, "Daily backup", entry.Job.Comment)
		assert.True(t, entry.Job.Valid)
	})

	t.Run("alias job without comment", func(t *testing.T) {
		line := "@hourly /usr/bin/hourly-task.sh"
		entry := ParseLine(line, 1)

		assert.Equal(t, EntryTypeJob, entry.Type)
		require.NotNil(t, entry.Job)
		assert.Equal(t, "@hourly", entry.Job.Expression)
		assert.Equal(t, "/usr/bin/hourly-task.sh", entry.Job.Command)
		assert.Empty(t, entry.Job.Comment)
		assert.True(t, entry.Job.Valid)
	})

	t.Run("alias job with invalid alias", func(t *testing.T) {
		line := "@invalid /usr/bin/test.sh"
		entry := ParseLine(line, 1)

		// Should be parsed as a job (alias format detected)
		if entry.Type == EntryTypeJob {
			require.NotNil(t, entry.Job)
			assert.Equal(t, "@invalid", entry.Job.Expression)
			// May or may not be valid depending on parser
			// The important thing is it's parsed
		} else {
			// Or might be invalid entry
			assert.Equal(t, EntryTypeInvalid, entry.Type)
		}
	})

	t.Run("alias job with only alias no command", func(t *testing.T) {
		// Reported as an EntryTypeJob with an invalid Job, not silently discarded.
		line := "@daily"
		entry := ParseLine(line, 1)

		require.Equal(t, EntryTypeJob, entry.Type)
		require.NotNil(t, entry.Job)
		assert.Equal(t, "@daily", entry.Job.Expression)
		assert.Empty(t, entry.Job.Command)
		assert.False(t, entry.Job.Valid)
		assert.Contains(t, entry.Job.Error, "no command")
	})
}
