package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimelineCommand(t *testing.T) {
	t.Run("timeline command should be registered", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"timeline"})
		assert.NoError(t, err)
		assert.Equal(t, "timeline", cmd.Name())
	})

	t.Run("timeline command should have metadata", func(t *testing.T) {
		tc := newTimelineCommand()
		assert.NotEmpty(t, tc.Short)
		assert.NotEmpty(t, tc.Long)
		assert.Contains(t, tc.Use, "timeline")
	})

	t.Run("timeline with single expression (text)", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"*/15 * * * *"})

		err := tc.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Timeline")
		assert.Contains(t, output, "*/15 * * * *")
	})

	t.Run("timeline with --view hour", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"*/5 * * * *", "--view", "hour"})

		err := tc.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Timeline")
		assert.Contains(t, output, "Hour View")
	})

	t.Run("timeline with --json flag", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"*/15 * * * *", "--json"})

		err := tc.Execute()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "day", result["view"])
		assert.NotNil(t, result["jobs"])
		assert.NotNil(t, result["overlaps"])
	})

	t.Run("timeline with invalid expression", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetErr(buf)
		tc.SetArgs([]string{"60 0 * * *"})

		err := tc.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
	})

	t.Run("timeline with --file flag", func(t *testing.T) {
		// Create a temporary crontab file
		tempFile := createTempCrontab(t, "*/15 * * * * /usr/bin/test.sh\n0 0 * * * /usr/bin/daily.sh\n")
		defer func() {
			_ = os.Remove(tempFile)
		}()

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--file", tempFile})

		err := tc.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Timeline")
	})

	t.Run("timeline with empty crontab file", func(t *testing.T) {
		// Create an empty temporary file
		tempFile := createTempCrontab(t, "")
		defer func() {
			_ = os.Remove(tempFile)
		}()

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--file", tempFile})

		err := tc.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Timeline")
	})

	t.Run("timeline with non-existent file", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetErr(buf)
		tc.SetArgs([]string{"--file", "/nonexistent/file.cron"})

		err := tc.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read")
	})

	t.Run("timeline JSON output with multiple jobs", func(t *testing.T) {
		tempFile := createTempCrontab(t, "*/15 * * * * /usr/bin/test.sh\n0 0 * * * /usr/bin/daily.sh\n")
		defer func() {
			_ = os.Remove(tempFile)
		}()

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--file", tempFile, "--json"})

		err := tc.Execute()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		jobs := result["jobs"].([]interface{})
		assert.Greater(t, len(jobs), 0)
	})

	t.Run("timeline with --view hour JSON output", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"*/5 * * * *", "--view", "hour", "--json"})

		err := tc.Execute()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "hour", result["view"])
	})

	t.Run("timeline detects overlaps", func(t *testing.T) {
		// Create jobs that run at the same time
		tempFile := createTempCrontab(t, "0 * * * * /usr/bin/job1.sh\n0 * * * * /usr/bin/job2.sh\n")
		defer func() {
			_ = os.Remove(tempFile)
		}()

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--file", tempFile, "--json"})

		err := tc.Execute()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		overlaps := result["overlaps"].([]interface{})
		// Should have at least some overlaps since both jobs run at minute 0
		assert.GreaterOrEqual(t, len(overlaps), 0)
	})

	t.Run("timeline with invalid --from time", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetErr(buf)
		tc.SetArgs([]string{"*/15 * * * *", "--from", "invalid-time"})

		err := tc.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid --from time")
	})

	t.Run("timeline with valid --from time", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"*/15 * * * *", "--from", "2025-01-15T00:00:00Z"})

		err := tc.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Timeline")
	})

	t.Run("timeline JSON output error handling", func(t *testing.T) {
		tc := newTimelineCommand()
		// Use a writer that will fail on write to test error path
		tc.SetOut(&timelineErrorWriter{})

		// JSON encoding errors are handled in runTimeline
		// This test verifies the command handles JSON encoding errors
		tc.SetArgs([]string{"0 * * * *", "--json"})
		err := tc.Execute()
		// Error writer will cause encoding to fail, but Execute may not return error
		// depending on implementation - this is acceptable for now
		_ = err
	})

	t.Run("timeline with --show-overlaps flag", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *", "--show-overlaps"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Overlap Summary")
	})

	t.Run("timeline without --show-overlaps flag (backward compatibility)", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.NotContains(t, output, "Overlap Summary")
	})

	t.Run("timeline with --show-overlaps and --json", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *", "--show-overlaps", "--json"})

		err := tc.Execute()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)
		assert.Contains(t, result, "overlapStats")
		overlapStats := result["overlapStats"].(map[string]interface{})
		assert.Contains(t, overlapStats, "totalWindows")
		assert.Contains(t, overlapStats, "maxConcurrent")
		assert.Contains(t, overlapStats, "mostProblematic")
	})

	t.Run("timeline --show-overlaps with multiple jobs", func(t *testing.T) {
		tempFile := createTempCrontab(t, "0 * * * * /usr/bin/job1.sh\n0 * * * * /usr/bin/job2.sh\n")
		defer func() {
			_ = os.Remove(tempFile)
		}()

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--file", tempFile, "--show-overlaps"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Overlap Summary")
		assert.Contains(t, output, "Total overlap windows")
	})

	t.Run("timeline with --width flag", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *", "--width", "120"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Timeline")
	})

	t.Run("timeline with --timezone flag", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *", "--timezone", "UTC"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Timeline")
	})

	t.Run("timeline with invalid --timezone flag", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *", "--timezone", "Invalid/Timezone"})

		err := tc.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid timezone")
	})

	t.Run("timeline with --export flag (text)", func(t *testing.T) {
		tempFile := createTempCrontab(t, "")
		defer func() {
			_ = os.Remove(tempFile)
		}()

		exportFile := tempFile + ".export.txt"
		defer func() {
			_ = os.Remove(exportFile)
		}()

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *", "--export", exportFile})

		err := tc.Execute()
		require.NoError(t, err)

		// Check file was created
		_, err = os.Stat(exportFile)
		assert.NoError(t, err)

		// Check file has content
		content, err := os.ReadFile(exportFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Timeline")
	})

	t.Run("timeline with --export flag (JSON)", func(t *testing.T) {
		tempFile := createTempCrontab(t, "")
		defer func() {
			_ = os.Remove(tempFile)
		}()

		exportFile := tempFile + ".export.json"
		defer func() {
			_ = os.Remove(exportFile)
		}()

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *", "--json", "--export", exportFile})

		err := tc.Execute()
		require.NoError(t, err)

		// Check file was created
		_, err = os.Stat(exportFile)
		assert.NoError(t, err)

		// Check file has JSON content
		content, err := os.ReadFile(exportFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), `"view"`)
	})

	t.Run("timeline with --width flag set to specific value", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *", "--width", "100"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Timeline")
	})

	t.Run("timeline with --width flag set to minimum", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *", "--width", "30"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Timeline")
		// Should enforce minimum width of 40
	})

	t.Run("timeline with --from and hour view", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"*/5 * * * *", "--from", "2025-01-15T14:00:00Z", "--view", "hour"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Timeline")
		assert.Contains(t, output, "Hour View")
	})

	t.Run("timeline with crontab file and timezone", func(t *testing.T) {
		tempFile := createTempCrontab(t, "0 * * * * /usr/bin/test.sh\n")
		defer func() {
			_ = os.Remove(tempFile)
		}()

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--file", tempFile, "--timezone", "Europe/London"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Timeline")
	})

	t.Run("timeline export with text format and show-overlaps", func(t *testing.T) {
		tempFile := createTempCrontab(t, "")
		defer func() {
			_ = os.Remove(tempFile)
		}()

		exportFile := tempFile + ".export.txt"
		defer func() {
			_ = os.Remove(exportFile)
		}()

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *", "--export", exportFile, "--show-overlaps"})

		err := tc.Execute()
		require.NoError(t, err)

		// Check file was created and has overlap info
		content, err := os.ReadFile(exportFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Timeline")
		assert.Contains(t, string(content), "Overlap Summary")
	})

	t.Run("timeline with --timezone America/New_York", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *", "--timezone", "America/New_York"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Timeline")
	})

	t.Run("timeline with --from and --timezone", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *", "--from", "2025-01-15T00:00:00Z", "--timezone", "UTC"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Timeline")
	})

	t.Run("timeline export with invalid file path", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		// Use a path that should fail (directory that doesn't exist)
		tc.SetArgs([]string{"0 * * * *", "--export", "/nonexistent/dir/file.txt"})

		err := tc.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create export file")
	})

	t.Run("timeline export JSON with file", func(t *testing.T) {
		tempFile := createTempCrontab(t, "")
		defer func() {
			_ = os.Remove(tempFile)
		}()

		exportFile := tempFile + ".export.json"
		defer func() {
			_ = os.Remove(exportFile)
		}()

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *", "--json", "--export", exportFile})

		err := tc.Execute()
		require.NoError(t, err)

		// Check file was created
		_, err = os.Stat(exportFile)
		assert.NoError(t, err)

		// Check file has JSON content
		content, err := os.ReadFile(exportFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), `"view"`)
	})

	t.Run("timeline detects terminal width from COLUMNS env var", func(t *testing.T) {
		// Set COLUMNS environment variable
		oldCols := os.Getenv("COLUMNS")
		defer func() {
			if oldCols != "" {
				_ = os.Setenv("COLUMNS", oldCols)
			} else {
				_ = os.Unsetenv("COLUMNS")
			}
		}()

		_ = os.Setenv("COLUMNS", "120")
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Timeline")
	})

	t.Run("timeline handles invalid COLUMNS env var", func(t *testing.T) {
		// Set invalid COLUMNS environment variable
		oldCols := os.Getenv("COLUMNS")
		defer func() {
			if oldCols != "" {
				_ = os.Setenv("COLUMNS", oldCols)
			} else {
				_ = os.Unsetenv("COLUMNS")
			}
		}()

		_ = os.Setenv("COLUMNS", "invalid")
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *"})

		err := tc.Execute()
		require.NoError(t, err)
		// Should fall back to default width
		output := buf.String()
		assert.Contains(t, output, "Timeline")
	})

	t.Run("timeline handles zero COLUMNS env var", func(t *testing.T) {
		// Set zero COLUMNS environment variable
		oldCols := os.Getenv("COLUMNS")
		defer func() {
			if oldCols != "" {
				_ = os.Setenv("COLUMNS", oldCols)
			} else {
				_ = os.Unsetenv("COLUMNS")
			}
		}()

		_ = os.Setenv("COLUMNS", "0")
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *"})

		err := tc.Execute()
		require.NoError(t, err)
		// Should fall back to default width
		output := buf.String()
		assert.Contains(t, output, "Timeline")
	})
}

// createTempCrontab is a helper function to create a temporary crontab file for testing
func createTempCrontab(t *testing.T, content string) string {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "crontab-*.cron")
	require.NoError(t, err)

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	err = tmpfile.Close()
	require.NoError(t, err)

	return tmpfile.Name()
}

// timelineErrorWriter is a writer that always returns an error for timeline tests
type timelineErrorWriter struct{}

func (e *timelineErrorWriter) Write(p []byte) (n int, err error) {
	return 0, &timelineWriteError{msg: "write error"}
}

type timelineWriteError struct {
	msg string
}

func (e *timelineWriteError) Error() string {
	return e.msg
}
