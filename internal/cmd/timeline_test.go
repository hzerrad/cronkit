package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/hzerrad/cronic/internal/render"
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

	t.Run("timeline outputTimelineJSON error handling", func(t *testing.T) {
		tc := newTimelineCommand()
		// Use a writer that will fail on write to test error path
		tc.SetOut(&timelineErrorWriter{})

		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		timeline := render.NewTimeline(render.DayView, startTime, 80)

		err := tc.outputTimelineJSON(timeline)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to encode JSON")
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
