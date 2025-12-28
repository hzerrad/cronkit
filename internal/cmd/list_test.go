package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/hzerrad/cronic/internal/crontab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCommand(t *testing.T) {
	t.Run("list command should be registered", func(t *testing.T) {
		cmd := rootCmd.Commands()
		var found bool
		for _, c := range cmd {
			if c.Name() == "list" {
				found = true
				break
			}
		}
		assert.True(t, found, "list command should be registered")
	})

	t.Run("list command should have metadata", func(t *testing.T) {
		assert.NotEmpty(t, listCmd.Short, "Short description should not be empty")
		assert.NotEmpty(t, listCmd.Long, "Long description should not be empty")
		assert.NotEmpty(t, listCmd.Use, "Use should not be empty")
	})

	t.Run("list crontab file with valid jobs", func(t *testing.T) {
		// Setup: Create command with output capture
		buf := new(bytes.Buffer)
		cmd := newListCommand()
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		// Get test fixture path
		testFile := filepath.Join("..", "..", "testdata", "crontab", "sample.cron")

		// Execute: Run command with test file
		cmd.SetArgs([]string{"--file", testFile})
		err := cmd.Execute()

		// Assert: Should succeed
		require.NoError(t, err)
		output := buf.String()

		// Should contain job information
		assert.Contains(t, output, "backup")
		assert.Contains(t, output, "check-disk")
		assert.Contains(t, output, "0 2 * * *")
		assert.Contains(t, output, "*/15 * * * *")
	})

	t.Run("list crontab file with JSON output", func(t *testing.T) {
		// Setup: Create command with output capture
		buf := new(bytes.Buffer)
		cmd := newListCommand()
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		// Get test fixture path
		testFile := filepath.Join("..", "..", "testdata", "crontab", "sample.cron")

		// Execute: Run command with --json flag
		cmd.SetArgs([]string{"--file", testFile, "--json"})
		err := cmd.Execute()

		// Assert: Should succeed and output JSON
		require.NoError(t, err)
		output := buf.String()

		// Should contain JSON structure
		assert.Contains(t, output, `"jobs"`)
		assert.Contains(t, output, `"expression"`)
		assert.Contains(t, output, `"command"`)
		assert.Contains(t, output, `"lineNumber"`)
	})

	t.Run("list empty crontab file", func(t *testing.T) {
		// Setup: Create command with output capture
		buf := new(bytes.Buffer)
		cmd := newListCommand()
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		// Get test fixture path
		testFile := filepath.Join("..", "..", "testdata", "crontab", "empty.cron")

		// Execute: Run command with empty file
		cmd.SetArgs([]string{"--file", testFile})
		err := cmd.Execute()

		// Assert: Should succeed with no jobs message
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "No cron jobs found")
	})

	t.Run("list non-existent file", func(t *testing.T) {
		// Setup: Create command with output capture
		buf := new(bytes.Buffer)
		cmd := newListCommand()
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		// Execute: Run command with non-existent file
		cmd.SetArgs([]string{"--file", "/path/to/nonexistent.cron"})
		err := cmd.Execute()

		// Assert: Should fail with error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read crontab")
	})

	t.Run("list with invalid crontab entries", func(t *testing.T) {
		// Setup: Create command with output capture
		buf := new(bytes.Buffer)
		cmd := newListCommand()
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		// Get test fixture path
		testFile := filepath.Join("..", "..", "testdata", "crontab", "invalid.cron")

		// Execute: Run command with invalid entries
		cmd.SetArgs([]string{"--file", testFile})
		err := cmd.Execute()

		// Assert: Should succeed (parses what it can)
		require.NoError(t, err)
		output := buf.String()

		// Should show valid jobs only
		assert.NotEmpty(t, output)
	})

	t.Run("list without --file flag should try user crontab", func(t *testing.T) {
		// Setup: Create command with output capture
		buf := new(bytes.Buffer)
		cmd := newListCommand()
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		// Execute: Run command without --file flag
		cmd.SetArgs([]string{})
		err := cmd.Execute()

		// Assert: Should either succeed or fail gracefully
		// (depending on whether user has a crontab)
		if err != nil {
			// If error, should be informative
			output := buf.String()
			assert.NotEmpty(t, output)
		} else {
			// If success, should show jobs or "no jobs" message
			output := buf.String()
			assert.NotEmpty(t, output)
		}
	})

	t.Run("list with --all flag should show comments and env vars", func(t *testing.T) {
		// Setup: Create command with output capture
		buf := new(bytes.Buffer)
		cmd := newListCommand()
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		// Get test fixture path
		testFile := filepath.Join("..", "..", "testdata", "crontab", "sample.cron")

		// Execute: Run command with --all flag
		cmd.SetArgs([]string{"--file", testFile, "--all"})
		err := cmd.Execute()

		// Assert: Should succeed and show all entries
		require.NoError(t, err)
		output := buf.String()

		// Should contain comments and env vars
		assert.Contains(t, output, "SHELL")
		assert.Contains(t, output, "PATH")
		assert.Contains(t, output, "MAILTO")
	})

	t.Run("list command uses locale from GetLocale", func(t *testing.T) {
		// Setup: Create command with output capture
		buf := new(bytes.Buffer)
		cmd := newListCommand()
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		// Create temp crontab with day names
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.cron")
		content := "0 9 * * MON /usr/bin/weekly-report.sh"
		err := os.WriteFile(tmpFile, []byte(content), 0644)
		require.NoError(t, err)

		// Execute: Run command (locale is handled by GetLocale())
		cmd.SetArgs([]string{"--file", tmpFile})
		err = cmd.Execute()

		// Assert: Should succeed and parse MON correctly
		require.NoError(t, err)
		output := buf.String()
		assert.NotEmpty(t, output)
		// Output should contain the job (MON is parsed internally)
		assert.Contains(t, output, "weekly-report")
	})

	t.Run("list with --all flag and JSON output", func(t *testing.T) {
		// Setup: Create command with output capture
		buf := new(bytes.Buffer)
		cmd := newListCommand()
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		// Get test fixture path
		testFile := filepath.Join("..", "..", "testdata", "crontab", "sample.cron")

		// Execute: Run command with --all and --json flags
		cmd.SetArgs([]string{"--file", testFile, "--all", "--json"})
		err := cmd.Execute()

		// Assert: Should succeed and output JSON with entries
		require.NoError(t, err)
		output := buf.String()

		// Should contain JSON structure with entries
		assert.Contains(t, output, `"entries"`)
		assert.Contains(t, output, `"type"`)
		assert.Contains(t, output, `"JOB"`)
		assert.Contains(t, output, `"COMMENT"`)
		assert.Contains(t, output, `"ENV"`)
	})

	t.Run("entryTypeString covers all types", func(t *testing.T) {
		// Test all entry types are covered
		types := []struct {
			entryType crontab.EntryType
			expected  string
		}{
			{crontab.EntryTypeJob, "JOB"},
			{crontab.EntryTypeComment, "COMMENT"},
			{crontab.EntryTypeEnvVar, "ENV"},
			{crontab.EntryTypeEmpty, "EMPTY"},
			{crontab.EntryTypeInvalid, "INVALID"},
		}

		for _, tt := range types {
			result := entryTypeString(tt.entryType)
			assert.Equal(t, tt.expected, result, "entryTypeString should return correct string for %v", tt.entryType)
		}

		// Test default case with invalid EntryType value
		invalidType := crontab.EntryType(999)
		result := entryTypeString(invalidType)
		assert.Equal(t, "UNKNOWN", result, "entryTypeString should return UNKNOWN for invalid EntryType")
	})
}
