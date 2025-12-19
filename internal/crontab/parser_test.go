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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := ParseLine(tt.line, tt.lineNumber)

			assert.Equal(t, tt.wantType, entry.Type)
			require.NotNil(t, entry.Job, "Job should not be nil for EntryTypeJob")
			assert.Equal(t, tt.lineNumber, entry.LineNumber)
			assert.Equal(t, tt.wantExpr, entry.Job.Expression)
			assert.Equal(t, tt.wantCommand, entry.Job.Command)
			if tt.wantComment != "" {
				assert.Equal(t, tt.wantComment, entry.Job.Comment)
			}
		})
	}
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
}
