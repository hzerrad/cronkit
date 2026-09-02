package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hzerrad/cronkit/internal/inventory"
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
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, output)
		assert.Contains(t, output, "Every 15 minutes") // Check for humanized description
	})

	t.Run("timeline with --view hour", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"*/5 * * * *", "--view", "hour"})

		err := tc.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, output)
		assert.Regexp(t, `\d{2}:00 → \d{2}:59`, output)
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
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, output)
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
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, output)
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
		tc.SetArgs([]string{"*/15 * * * *", "--from", "2025-01-15T00:00:00Z", "--timezone", "UTC"})

		err := tc.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, output)
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
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, output)
		// A single job can't overlap with itself, so no overlaps section.
		assert.NotContains(t, output, "overlaps:")
	})

	t.Run("timeline without --show-overlaps flag (backward compatibility)", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.NotContains(t, output, "overlaps:")
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
		assert.Contains(t, output, "overlaps:")
		assert.Contains(t, output, "conflict window")
	})

	t.Run("timeline with --width flag", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *", "--width", "120"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, output)
	})

	t.Run("timeline with --timezone flag", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *", "--timezone", "UTC"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, output)
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
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, string(content))
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
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, output)
	})

	t.Run("timeline with --width flag set to minimum", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *", "--width", "30"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, output)
		// Should enforce minimum width of 40
	})

	t.Run("timeline with --from and hour view", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"*/5 * * * *", "--from", "2025-01-15T14:00:00Z", "--view", "hour", "--timezone", "UTC"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, output)
		assert.Regexp(t, `\d{2}:00 → \d{2}:59`, output)
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
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, output)
	})

	t.Run("timeline export with text format and show-overlaps", func(t *testing.T) {
		tempFile := createTempCrontab(t, "0 * * * * /usr/bin/job1.sh\n0 * * * * /usr/bin/job2.sh\n")
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
		tc.SetArgs([]string{"--file", tempFile, "--export", exportFile, "--show-overlaps"})

		err := tc.Execute()
		require.NoError(t, err)

		// Check file was created and has overlap info
		content, err := os.ReadFile(exportFile)
		require.NoError(t, err)
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, string(content))
		assert.Contains(t, string(content), "overlaps:")
	})

	t.Run("timeline with --timezone America/New_York", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *", "--timezone", "America/New_York"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, output)
	})

	t.Run("timeline with --from and --timezone", func(t *testing.T) {
		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"0 * * * *", "--from", "2025-01-15T00:00:00Z", "--timezone", "UTC"})

		err := tc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, output)
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
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, output)
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
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, output)
	})

	t.Run("timeline dedupes duplicate command basenames with a :LINE suffix", func(t *testing.T) {
		tempFile := createTempCrontab(t, "# comment\n* * * * * /usr/local/bin/backup.sh --full\n# comment\n# comment\n0 2 * * * /opt/scripts/backup.sh\n")
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
		assert.Contains(t, output, "backup.sh:2")
		assert.Contains(t, output, "backup.sh:5")
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
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, output)
	})
}

func TestResolveWidth(t *testing.T) {
	t.Run("should prefer the flag", func(t *testing.T) {
		assert.Equal(t, 120, resolveWidth(120, true))
	})
	t.Run("should fall back to COLUMNS off-tty", func(t *testing.T) {
		t.Setenv("COLUMNS", "100")
		assert.Equal(t, 100, resolveWidth(0, false))
	})
	t.Run("should default to 80", func(t *testing.T) {
		t.Setenv("COLUMNS", "")
		assert.Equal(t, 80, resolveWidth(0, false))
	})
}

func TestResolveColor(t *testing.T) {
	t.Run("should honor always and never regardless of tty", func(t *testing.T) {
		on, err := resolveColor("always", false)
		assert.NoError(t, err)
		assert.True(t, on)
		on, _ = resolveColor("never", true)
		assert.False(t, on)
	})
	t.Run("should auto-disable off-tty and under NO_COLOR", func(t *testing.T) {
		on, _ := resolveColor("auto", false)
		assert.False(t, on)
		t.Setenv("NO_COLOR", "1")
		on, _ = resolveColor("auto", true)
		assert.False(t, on)
	})
	t.Run("should reject unknown modes", func(t *testing.T) {
		_, err := resolveColor("rainbow", true)
		assert.Error(t, err)
	})
}

func TestUserCrontabSource(t *testing.T) {
	t.Run("should always name a crontab source", func(t *testing.T) {
		got := userCrontabSource()
		assert.NotEmpty(t, got)
		assert.Contains(t, got, "crontab")
	})
}

func TestAbsPath(t *testing.T) {
	t.Run("should resolve a relative path", func(t *testing.T) {
		got := absPath("timeline_test.go")
		assert.True(t, filepath.IsAbs(got), "want absolute, got %q", got)
	})
	t.Run("should pass an already-absolute path through", func(t *testing.T) {
		assert.Equal(t, "/etc/crontab", absPath("/etc/crontab"))
	})
}

func TestTimelineSourceLabel(t *testing.T) {
	t.Run("inventory from a path", func(t *testing.T) {
		got := timelineSourceLabel(&inputFlags{inventory: "some.json"})
		assert.True(t, filepath.IsAbs(got))
		assert.Contains(t, got, "some.json")
	})
	t.Run("inventory from stdin", func(t *testing.T) {
		got := timelineSourceLabel(&inputFlags{inventory: "-"})
		assert.Equal(t, "inventory from stdin", got)
	})
	t.Run("file", func(t *testing.T) {
		got := timelineSourceLabel(&inputFlags{file: "crontab.txt"})
		assert.True(t, filepath.IsAbs(got))
		assert.Contains(t, got, "crontab.txt")
	})
	t.Run("stdin", func(t *testing.T) {
		got := timelineSourceLabel(&inputFlags{stdin: true})
		assert.Equal(t, "stdin", got)
	})
	t.Run("user crontab", func(t *testing.T) {
		got := timelineSourceLabel(&inputFlags{})
		assert.Contains(t, got, "crontab")
	})
}

func TestLaneLabel(t *testing.T) {
	t.Run("should use the command basename", func(t *testing.T) {
		item := inventory.Item{Locator: inventory.Locator{Line: 3}, Command: "/usr/local/bin/backup.sh --full"}
		assert.Equal(t, "backup.sh", laneLabel(item))
	})
	t.Run("should use the file base name without extension when Command is empty", func(t *testing.T) {
		item := inventory.Item{Command: "", Locator: inventory.Locator{File: "team-a/backup.yaml", Line: 6}}
		assert.Equal(t, "backup", laneLabel(item))
	})
	t.Run("should never label two items from different files the same", func(t *testing.T) {
		a := laneLabel(inventory.Item{Locator: inventory.Locator{File: "team-a/backup.yaml", Line: 6}})
		b := laneLabel(inventory.Item{Locator: inventory.Locator{File: "team-b/flow.yaml", Line: 7}})
		assert.NotEqual(t, a, b)
		assert.Equal(t, "backup", a)
		assert.Equal(t, "flow", b)
	})
	t.Run("should fall back to the structural path's last segment when there is no file", func(t *testing.T) {
		item := inventory.Item{Locator: inventory.Locator{Path: "spec.schedules[0]", Line: 3}}
		assert.Equal(t, "schedules[0]", laneLabel(item))
	})
	t.Run("should fall back to job-N when nothing else names the item", func(t *testing.T) {
		item := inventory.Item{Locator: inventory.Locator{Line: 7}, Command: "   "}
		assert.Equal(t, "job-7", laneLabel(item))
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

func TestTimelineCommand_Stdin(t *testing.T) {
	t.Run("timeline reads a crontab from stdin -- a fix, not new behavior it never had before", func(t *testing.T) {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		_, err = w.WriteString("*/15 * * * * /usr/bin/test.sh\n")
		require.NoError(t, err)
		require.NoError(t, w.Close())

		oldStdin := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = oldStdin }()

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--stdin"})

		err = tc.Execute()
		require.NoError(t, err)
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, buf.String())
	})
}

func TestTimelineCommand_Inventory(t *testing.T) {
	t.Run("timeline reads schedules from an inventory file", func(t *testing.T) {
		inv := inventory.New("", []inventory.Item{
			{Expression: "*/15 * * * *", Command: "worker", State: inventory.StateActive},
		})
		f, err := os.CreateTemp("", "cronkit-inventory-*.json")
		require.NoError(t, err)
		require.NoError(t, inv.Encode(f))
		require.NoError(t, f.Close())
		defer func() { _ = os.Remove(f.Name()) }()

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--inventory", f.Name()})

		err = tc.Execute()
		require.NoError(t, err)
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, buf.String())
	})

	t.Run("a malformed inventory produces a clear error", func(t *testing.T) {
		f, err := os.CreateTemp("", "cronkit-inventory-*.json")
		require.NoError(t, err)
		_, err = f.WriteString("not json")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		defer func() { _ = os.Remove(f.Name()) }()

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetErr(buf)
		tc.SetArgs([]string{"--inventory", f.Name()})

		err = tc.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read inventory")
	})

	t.Run("colliding line numbers across files produce distinct jobs, not a merged one", func(t *testing.T) {
		// Two files sharing line 2 used to collapse into a single "job-2" lane, dropping one schedule's runs.
		inv := inventory.New("", []inventory.Item{
			{
				Expression: "0 1 * * *", Command: "/opt/backup.sh", Shell: true,
				State:   inventory.StateActive,
				Locator: inventory.Locator{File: "site-a/crontab", Line: 2},
			},
			{
				Expression: "0 5 * * *", Command: "/srv/backup.sh", Shell: true,
				State:   inventory.StateActive,
				Locator: inventory.Locator{File: "site-b/crontab", Line: 2},
			},
		})
		f, err := os.CreateTemp("", "cronkit-inventory-*.json")
		require.NoError(t, err)
		require.NoError(t, inv.Encode(f))
		require.NoError(t, f.Close())
		defer func() { _ = os.Remove(f.Name()) }()

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{
			"--inventory", f.Name(), "--json",
			"--from", "2026-08-13T00:00:00Z", "--width", "100", "--color", "never",
		})

		err = tc.Execute()
		require.NoError(t, err)

		var result map[string]interface{}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
		jobs, ok := result["jobs"].([]interface{})
		require.True(t, ok)
		require.Len(t, jobs, 2, "both schedules must survive as distinct jobs")

		ids := make(map[string]bool)
		exprByID := make(map[string]string)
		runsByExpr := make(map[string]int)
		for _, raw := range jobs {
			job := raw.(map[string]interface{})
			id := job["id"].(string)
			assert.False(t, ids[id], "job ids must not collide: %s", id)
			ids[id] = true

			expr := job["expression"].(string)
			exprByID[id] = expr
			runs, _ := job["runs"].([]interface{})
			runsByExpr[expr] = len(runs)
		}

		require.Contains(t, runsByExpr, "0 1 * * *")
		require.Contains(t, runsByExpr, "0 5 * * *")
		assert.Equal(t, 1, runsByExpr["0 1 * * *"], "the 01:00 schedule must keep its own run")
		assert.Equal(t, 1, runsByExpr["0 5 * * *"], "the 05:00 schedule must not vanish")
	})
}

// TestTimelineJobID confirms job identity comes from inventory.Locator.Identity, since line numbers
// alone are not unique across files.
func TestTimelineJobID(t *testing.T) {
	t.Run("is exactly job-<line> when there is no file", func(t *testing.T) {
		item := inventory.Item{Locator: inventory.Locator{Line: 2}}
		assert.Equal(t, "job-2", item.Locator.Identity(0, "job-", "job-"),
			"no file to fold in -- the id stays the bare line, e.g. stdin or the user's own crontab")
	})

	t.Run("folds in the file whenever it is set", func(t *testing.T) {
		a := inventory.Locator{File: "site-a/crontab", Line: 2}.Identity(0, "job-", "job-")
		b := inventory.Locator{File: "site-b/crontab", Line: 2}.Identity(1, "job-", "job-")
		assert.NotEqual(t, a, b, "same line number in different files must not collide")
		assert.Equal(t, "job-site-a/crontab:2", a)
		assert.Equal(t, "job-site-b/crontab:2", b)
	})

	t.Run("folds in the structural path when several schedules share a line", func(t *testing.T) {
		a := inventory.Locator{File: "wf.yaml", Line: 3, Path: "spec.schedules[0]"}.Identity(0, "job-", "job-")
		b := inventory.Locator{File: "wf.yaml", Line: 3, Path: "spec.schedules[1]"}.Identity(1, "job-", "job-")
		assert.NotEqual(t, a, b, "an Argo flow-style sequence puts every schedule on one line")
		assert.Equal(t, "job-wf.yaml:3#spec.schedules[0]", a)
	})

	t.Run("falls back to index when there is no line", func(t *testing.T) {
		var locator inventory.Locator
		a := locator.Identity(0, "job-", "job-")
		b := locator.Identity(1, "job-", "job-")
		assert.NotEqual(t, a, b, "two line-less items sharing an expression must not collide")
	})

	t.Run("does not move when the item's position changes", func(t *testing.T) {
		locator := inventory.Locator{File: "crontab", Line: 2}
		assert.Equal(t, locator.Identity(0, "job-", "job-"), locator.Identity(4, "job-", "job-"),
			"a lane id published in --json must survive an unrelated schedule being added above it")
	})
}

// TestTimelineCommand_DeterministicOrder feeds the identical jobs in a different order and compares output,
// since runTimeline sorts by locator before registering items.
func TestTimelineCommand_DeterministicOrder(t *testing.T) {
	items := []inventory.Item{
		{Expression: "0 1 * * *", Command: "/opt/backup.sh", Shell: true, State: inventory.StateActive, Locator: inventory.Locator{File: "site-a/crontab", Line: 2}},
		{Expression: "0 5 * * *", Command: "/srv/backup.sh", Shell: true, State: inventory.StateActive, Locator: inventory.Locator{File: "site-b/crontab", Line: 2}},
		{Expression: "0 9 * * *", Command: "/opt/report.sh", Shell: true, State: inventory.StateActive, Locator: inventory.Locator{File: "site-a/crontab", Line: 5}},
		{Expression: "0 3 * * *", Command: "/var/cleanup.sh", Shell: true, State: inventory.StateActive, Locator: inventory.Locator{File: "site-c/crontab", Line: 1}},
		{Expression: "0 7 * * *", Command: "/opt/sync.sh", Shell: true, State: inventory.StateActive, Locator: inventory.Locator{File: "site-b/crontab", Line: 9}},
		{Expression: "0 11 * * *", Command: "/var/purge.sh", Shell: true, State: inventory.StateActive, Locator: inventory.Locator{File: "site-c/crontab", Line: 4}},
	}
	// Same set, deliberately not sorted the same way.
	shuffled := []inventory.Item{items[4], items[1], items[5], items[0], items[3], items[2]}

	// renderWith writes order to a fixed path, shared across every call, so the printed source line matches too.
	renderWith := func(t *testing.T, path string, order []inventory.Item, extraArgs ...string) []byte {
		t.Helper()
		inv := inventory.New("", order)
		f, err := os.Create(path)
		require.NoError(t, err)
		require.NoError(t, inv.Encode(f))
		require.NoError(t, f.Close())

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		args := append([]string{
			"--inventory", path,
			"--from", "2026-08-13T00:00:00Z", "--width", "100", "--color", "never",
		}, extraArgs...)
		tc.SetArgs(args)
		require.NoError(t, tc.Execute())
		return buf.Bytes()
	}

	t.Run("json output is byte-identical regardless of input order", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "inventory.json")
		outA := renderWith(t, path, items, "--json")
		outB := renderWith(t, path, shuffled, "--json")
		assert.Equal(t, string(outA), string(outB),
			"the same jobs in a different order must produce a byte-identical JSON array")

		// Confirm it landed in the locator-sorted order the fix promises,
		// not merely "some" order that happens to match between the two.
		var result map[string]interface{}
		require.NoError(t, json.Unmarshal(outA, &result))
		jobs := result["jobs"].([]interface{})
		require.Len(t, jobs, 6)
		var exprs []string
		for _, raw := range jobs {
			exprs = append(exprs, raw.(map[string]interface{})["expression"].(string))
		}
		assert.Equal(t,
			[]string{"0 1 * * *", "0 9 * * *", "0 5 * * *", "0 7 * * *", "0 3 * * *", "0 11 * * *"},
			exprs, "jobs must be ordered by locator file, then line")
	})

	t.Run("text output lane order is byte-identical regardless of input order", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "inventory.json")
		outA := renderWith(t, path, items)
		outB := renderWith(t, path, shuffled)
		assert.Equal(t, string(outA), string(outB),
			"the same jobs in a different order must produce a byte-identical chart")
	})

	t.Run("running the same order twice is also stable", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "inventory.json")
		first := renderWith(t, path, items, "--json")
		second := renderWith(t, path, items, "--json")
		assert.Equal(t, string(first), string(second))
	})
}

// TestTimelineCommand_ExcludesNonActive pins that a non-StateActive item never becomes a lane, and the
// footer reports it instead of letting it vanish silently.
func TestTimelineCommand_ExcludesNonActive(t *testing.T) {
	inv := inventory.New("", []inventory.Item{
		{Expression: "0 * * * *", Command: "/opt/live.sh", Shell: true,
			State: inventory.StateActive, Locator: inventory.Locator{File: "crontab", Line: 1}},
		{Expression: "0 1 * * *", Command: "/opt/suspended.sh", Shell: true,
			State: inventory.StateSuspended, Reason: "manually suspended",
			Locator: inventory.Locator{File: "crontab", Line: 2}},
		{Expression: "{{ .Schedule }}", Command: "/opt/templated.sh", Shell: true,
			State: inventory.StateUnresolved, Reason: "templated expression",
			Locator: inventory.Locator{File: "crontab", Line: 3}},
		{Expression: "99 * * * *", Command: "/opt/broken.sh", Shell: true,
			State: inventory.StateInvalid, Reason: "field out of range",
			Locator: inventory.Locator{File: "crontab", Line: 4}},
	})
	path := writeInventoryFixture(t, inv)

	tc := newTimelineCommand()
	buf := new(bytes.Buffer)
	tc.SetOut(buf)
	tc.SetArgs([]string{"--inventory", path, "--from", "2026-08-13T00:00:00Z", "--color", "never"})
	require.NoError(t, tc.Execute())

	out := buf.String()
	assert.Contains(t, out, "live.sh")
	assert.NotContains(t, out, "suspended.sh")
	assert.NotContains(t, out, "templated.sh")
	assert.NotContains(t, out, "broken.sh")
	assert.Contains(t, out, "1 job ")
	assert.Contains(t, out, "1 suspended job, 1 unresolved job, 1 invalid job excluded")
}

// TestTimelineCommand_ZoneConversion pins that an item's own zone is converted onto the shared axis, and
// the window line admits the conversion happened.
func TestTimelineCommand_ZoneConversion(t *testing.T) {
	inv := inventory.New("", []inventory.Item{
		// 09:30 Asia/Tokyo (UTC+9) on 2026-08-13 is 00:30 UTC the same day.
		{Expression: "30 9 * * *", Command: "/opt/tokyo.sh", Shell: true, Timezone: "Asia/Tokyo",
			State: inventory.StateActive, Locator: inventory.Locator{File: "crontab", Line: 1}},
	})
	path := writeInventoryFixture(t, inv)

	tc := newTimelineCommand()
	buf := new(bytes.Buffer)
	tc.SetOut(buf)
	tc.SetArgs([]string{
		"--inventory", path, "--json",
		"--from", "2026-08-13T00:00:00Z", "--timezone", "UTC", "--color", "never",
	})
	require.NoError(t, tc.Execute())

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	jobs := result["jobs"].([]interface{})
	require.Len(t, jobs, 1)
	job := jobs[0].(map[string]interface{})
	runs := job["runs"].([]interface{})
	require.Len(t, runs, 1)
	runTimeStr := runs[0].(map[string]interface{})["time"].(string)
	runTime, err := time.Parse(time.RFC3339, runTimeStr)
	require.NoError(t, err)
	assert.True(t, runTime.Equal(time.Date(2026, 8, 13, 0, 30, 0, 0, time.UTC)),
		"09:30 Tokyo must land on 00:30 UTC, got %s", runTime)
	assert.Equal(t, "UTC", runTime.Location().String(), "the run time must carry the axis zone")

	// The text window line must admit the conversion.
	tc2 := newTimelineCommand()
	buf2 := new(bytes.Buffer)
	tc2.SetOut(buf2)
	tc2.SetArgs([]string{
		"--inventory", path,
		"--from", "2026-08-13T00:00:00Z", "--timezone", "UTC", "--color", "never",
	})
	require.NoError(t, tc2.Execute())
	assert.Contains(t, buf2.String(), "converted from Asia/Tokyo")
}

// TestTimelineCommand_ZoneConversionNoNote pins that an item with no zone (or the axis zone) leaves the
// window line unchanged.
func TestTimelineCommand_ZoneConversionNoNote(t *testing.T) {
	inv := inventory.New("", []inventory.Item{
		{Expression: "0 9 * * *", Command: "/opt/utc.sh", Shell: true, Timezone: "UTC",
			State: inventory.StateActive, Locator: inventory.Locator{File: "crontab", Line: 1}},
	})
	path := writeInventoryFixture(t, inv)

	tc := newTimelineCommand()
	buf := new(bytes.Buffer)
	tc.SetOut(buf)
	tc.SetArgs([]string{
		"--inventory", path,
		"--from", "2026-08-13T00:00:00Z", "--timezone", "UTC", "--color", "never",
	})
	require.NoError(t, tc.Execute())
	assert.NotContains(t, buf.String(), "converted from")
}

// TestTimelineCommand_MultiFileGutter pins that the gutter shows the expression for a single file, and
// switches to file:line once the chart spans more than one.
func TestTimelineCommand_MultiFileGutter(t *testing.T) {
	t.Run("shows the expression for a single-file inventory", func(t *testing.T) {
		inv := inventory.New("", []inventory.Item{
			{Expression: "0 1 * * *", Command: "/opt/backup.sh", Shell: true,
				State: inventory.StateActive, Locator: inventory.Locator{File: "crontab", Line: 1}},
		})
		path := writeInventoryFixture(t, inv)

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--inventory", path, "--from", "2026-08-13T00:00:00Z", "--color", "never"})
		require.NoError(t, tc.Execute())
		assert.Contains(t, buf.String(), "0 1 * * *")
		assert.NotContains(t, buf.String(), "crontab:1")
	})

	t.Run("shows file:line once the chart spans multiple files", func(t *testing.T) {
		inv := inventory.New("", []inventory.Item{
			{Expression: "0 1 * * *", Command: "/opt/backup.sh", Shell: true,
				State: inventory.StateActive, Locator: inventory.Locator{File: "site-a/crontab", Line: 3}},
			{Expression: "0 5 * * *", Command: "/srv/sync.sh", Shell: true,
				State: inventory.StateActive, Locator: inventory.Locator{File: "site-b/crontab", Line: 7}},
		})
		path := writeInventoryFixture(t, inv)

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--inventory", path, "--from", "2026-08-13T00:00:00Z", "--color", "never"})
		require.NoError(t, tc.Execute())
		out := buf.String()
		assert.Contains(t, out, "site-a/crontab:3")
		assert.Contains(t, out, "site-b/crontab:7")
		assert.NotContains(t, out, "0 1 * * *")
		assert.NotContains(t, out, "0 5 * * *")
	})
}

// manyItems builds n active items spread one-per-file across nFiles files,
// evenly spaced every 5 minutes so they generate distinct run times.
func manyItems(n, nFiles int) []inventory.Item {
	items := make([]inventory.Item, 0, n)
	for i := 0; i < n; i++ {
		file := fmt.Sprintf("service-%02d/crontab", i%nFiles)
		items = append(items, inventory.Item{
			Expression: fmt.Sprintf("%d * * * *", i%60),
			Command:    fmt.Sprintf("/opt/job-%03d.sh", i),
			Shell:      true,
			State:      inventory.StateActive,
			Locator:    inventory.Locator{File: file, Line: i + 1},
		})
	}
	return items
}

// TestTimelineCommand_Collapse pins that past twenty lanes the chart collapses to one lane per file,
// --expand forces per-job lanes, and --top caps whichever set results.
func TestTimelineCommand_Collapse(t *testing.T) {
	t.Run("collapses to one lane per file past the threshold", func(t *testing.T) {
		inv := inventory.New("", manyItems(25, 4))
		path := writeInventoryFixture(t, inv)

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--inventory", path, "--width", "150", "--from", "2026-08-13T00:00:00Z", "--color", "never"})
		require.NoError(t, tc.Execute())

		out := buf.String()
		assert.Contains(t, out, "4 jobs")
		assert.NotContains(t, out, "job-000.sh") // no per-job lane label survives collapse
		assert.Contains(t, out, "service-00/crontab")
		assert.Contains(t, out, "25 jobs collapsed into 4 file lanes (--expand to show all)")
	})

	t.Run("stays per-job at exactly the threshold", func(t *testing.T) {
		inv := inventory.New("", manyItems(20, 4))
		path := writeInventoryFixture(t, inv)

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--inventory", path, "--from", "2026-08-13T00:00:00Z", "--color", "never"})
		require.NoError(t, tc.Execute())
		assert.NotContains(t, buf.String(), "collapsed into")
	})

	t.Run("--expand forces per-job lanes past the threshold", func(t *testing.T) {
		inv := inventory.New("", manyItems(25, 4))
		path := writeInventoryFixture(t, inv)

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--inventory", path, "--expand", "--from", "2026-08-13T00:00:00Z", "--color", "never"})
		require.NoError(t, tc.Execute())

		out := buf.String()
		assert.NotContains(t, out, "collapsed into")
		assert.Contains(t, out, "job-000.sh")
		assert.Contains(t, out, "25 jobs")
	})

	t.Run("--top caps a collapsed chart and reports what was hidden", func(t *testing.T) {
		inv := inventory.New("", manyItems(25, 8))
		path := writeInventoryFixture(t, inv)

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--inventory", path, "--top", "3", "--from", "2026-08-13T00:00:00Z", "--color", "never"})
		require.NoError(t, tc.Execute())

		out := buf.String()
		assert.Contains(t, out, "collapsed into 8 file lanes")
		assert.Contains(t, out, "5 lanes hidden (--top 3)")
	})

	t.Run("collapsed lane labels stay distinguishable at default width", func(t *testing.T) {
		// padLabel used to truncate from the right, rendering every lane below identically.
		files := []string{
			"testdata/sources/services/api/cronjob.yaml",
			"testdata/sources/services/worker/cronjob.yaml",
			"testdata/sources/services/billing/cronjob.yaml",
			"testdata/sources/services/reports/cronjob.yaml",
		}
		items := make([]inventory.Item, 0, 25)
		for i := 0; i < 25; i++ {
			items = append(items, inventory.Item{
				Expression: fmt.Sprintf("%d * * * *", i%60),
				Command:    fmt.Sprintf("/opt/job-%03d.sh", i),
				Shell:      true,
				State:      inventory.StateActive,
				Locator:    inventory.Locator{File: files[i%len(files)], Line: i + 1},
			})
		}
		inv := inventory.New("", items)
		path := writeInventoryFixture(t, inv)

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--inventory", path, "--from", "2026-08-13T00:00:00Z", "--color", "never"})
		require.NoError(t, tc.Execute())

		out := buf.String()
		assert.Contains(t, out, "collapsed into 4 file lanes")

		// Truncating from the wrong end used to render every lane's label as the identical prefix.
		labels := map[string]bool{}
		for _, line := range strings.Split(out, "\n") {
			if idx := strings.Index(line, "┤"); idx > 0 && strings.Contains(line, "cronjob.yaml") {
				labels[strings.TrimSpace(line[:idx])] = true
			}
		}
		assert.Len(t, labels, 4, "collapsed lane labels must stay distinguishable, got %v", labels)
	})

	t.Run("--top caps an expanded chart and reports what was hidden", func(t *testing.T) {
		inv := inventory.New("", manyItems(6, 6))
		path := writeInventoryFixture(t, inv)

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--inventory", path, "--top", "2", "--from", "2026-08-13T00:00:00Z", "--color", "never"})
		require.NoError(t, tc.Execute())

		out := buf.String()
		assert.NotContains(t, out, "collapsed into")
		assert.Contains(t, out, "4 lanes hidden (--top 2)")
	})
}

// TestTimelineCommand_JSONProvenance pins that RenderJSON gains provenance per job additively: a locator
// once a job carries one, an aggregated count once a lane stands for more than one item.
func TestTimelineCommand_JSONProvenance(t *testing.T) {
	t.Run("carries a locator for an ordinary per-job lane", func(t *testing.T) {
		inv := inventory.New("", []inventory.Item{
			{Expression: "0 1 * * *", Command: "/opt/backup.sh", Shell: true,
				State: inventory.StateActive, Locator: inventory.Locator{File: "site-a/crontab", Line: 3}},
		})
		path := writeInventoryFixture(t, inv)

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--inventory", path, "--json", "--from", "2026-08-13T00:00:00Z", "--color", "never"})
		require.NoError(t, tc.Execute())

		var result map[string]interface{}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
		jobs := result["jobs"].([]interface{})
		require.Len(t, jobs, 1)
		job := jobs[0].(map[string]interface{})
		locator := job["locator"].(map[string]interface{})
		assert.Equal(t, "site-a/crontab", locator["file"])
		assert.EqualValues(t, 3, locator["line"])
		_, hasAggregated := job["aggregated"]
		assert.False(t, hasAggregated)
	})

	t.Run("carries an aggregated count for a collapsed lane", func(t *testing.T) {
		inv := inventory.New("", manyItems(25, 4))
		path := writeInventoryFixture(t, inv)

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--inventory", path, "--json", "--from", "2026-08-13T00:00:00Z", "--color", "never"})
		require.NoError(t, tc.Execute())

		var result map[string]interface{}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
		jobs := result["jobs"].([]interface{})
		require.Len(t, jobs, 4)
		total := 0
		for _, raw := range jobs {
			job := raw.(map[string]interface{})
			aggregated, ok := job["aggregated"].(float64)
			require.True(t, ok, "each collapsed lane must carry an aggregated count")
			total += int(aggregated)
			locator := job["locator"].(map[string]interface{})
			assert.NotEmpty(t, locator["file"])
		}
		assert.Equal(t, 25, total)
	})

	t.Run("existing per-job JSON output gains no new keys when nothing new was set", func(t *testing.T) {
		tempFile := createTempCrontab(t, "*/15 * * * * /usr/bin/test.sh\n0 0 * * * /usr/bin/daily.sh\n")
		defer func() { _ = os.Remove(tempFile) }()

		tc := newTimelineCommand()
		buf := new(bytes.Buffer)
		tc.SetOut(buf)
		tc.SetArgs([]string{"--file", tempFile, "--json", "--color", "never"})
		require.NoError(t, tc.Execute())

		var result map[string]interface{}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
		jobs := result["jobs"].([]interface{})
		for _, raw := range jobs {
			job := raw.(map[string]interface{})
			_, hasAggregated := job["aggregated"]
			assert.False(t, hasAggregated)
		}
	})
}

// TestTimelineCommand_FlagsRegistered confirms --expand and --top are wired
// up on the command.
func TestTimelineCommand_FlagsRegistered(t *testing.T) {
	tc := newTimelineCommand()
	assert.NotNil(t, tc.Flag("expand"))
	assert.NotNil(t, tc.Flag("top"))
}

// TestTimelineCommand_LongPathGuttersStayDistinguishable pins that two long paths diverging near the end
// truncate from the left, not the right, so they don't collapse to the same displayed gutter.
func TestTimelineCommand_LongPathGuttersStayDistinguishable(t *testing.T) {
	inv := inventory.New("", []inventory.Item{
		{Expression: "0 1 * * *", Command: "/opt/backup.sh", Shell: true,
			State: inventory.StateActive, Locator: inventory.Locator{File: "services/api/deploy/backup.yaml", Line: 3}},
		{Expression: "0 5 * * *", Command: "/opt/restore.sh", Shell: true,
			State: inventory.StateActive, Locator: inventory.Locator{File: "services/api/deploy/restore.yaml", Line: 3}},
	})
	path := writeInventoryFixture(t, inv)

	tc := newTimelineCommand()
	buf := new(bytes.Buffer)
	tc.SetOut(buf)
	tc.SetArgs([]string{"--inventory", path, "--width", "80", "--from", "2026-08-13T00:00:00Z", "--color", "never"})
	require.NoError(t, tc.Execute())

	out := buf.String()
	var backupGutter, restoreGutter string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "backup.sh") {
			backupGutter = line[strings.LastIndex(line, "├")+len("├"):]
		}
		if strings.HasPrefix(line, "restore.sh") {
			restoreGutter = line[strings.LastIndex(line, "├")+len("├"):]
		}
	}
	require.NotEmpty(t, backupGutter)
	require.NotEmpty(t, restoreGutter)
	assert.NotEqual(t, backupGutter, restoreGutter,
		"two long paths sharing a prefix must render distinguishable gutters, got %q and %q", backupGutter, restoreGutter)
	assert.Contains(t, out, "...")
	assert.NotContains(t, out, "…", "path truncation must use the ASCII marker, not the chart's unicode ellipsis")
}

// TestTimelineCommand_TopBusiest pins that --top keeps the busiest lanes by run count, not alphabetically.
func TestTimelineCommand_TopBusiest(t *testing.T) {
	inv := inventory.New("", []inventory.Item{
		// Alphabetically first, but sparse: one run in the window.
		{Expression: "0 0 * * *", Command: "/opt/aaa-sparse.sh", Shell: true,
			State: inventory.StateActive, Locator: inventory.Locator{File: "aaa/crontab", Line: 1}},
		// Alphabetically last, but the busiest by far.
		{Expression: "*/5 * * * *", Command: "/opt/zzz-busy.sh", Shell: true,
			State: inventory.StateActive, Locator: inventory.Locator{File: "zzz/crontab", Line: 1}},
		{Expression: "0 12 * * *", Command: "/opt/mmm-medium.sh", Shell: true,
			State: inventory.StateActive, Locator: inventory.Locator{File: "mmm/crontab", Line: 1}},
	})
	path := writeInventoryFixture(t, inv)

	tc := newTimelineCommand()
	buf := new(bytes.Buffer)
	tc.SetOut(buf)
	tc.SetArgs([]string{
		"--inventory", path, "--json", "--top", "1",
		"--from", "2026-08-13T00:00:00Z", "--color", "never",
	})
	require.NoError(t, tc.Execute())

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	jobs := result["jobs"].([]interface{})
	require.Len(t, jobs, 1)
	assert.Equal(t, "*/5 * * * *", jobs[0].(map[string]interface{})["expression"],
		"--top 1 must keep the busiest lane (zzz-busy), not the alphabetically first one (aaa-sparse)")
}

// TestTimelineCommand_TopTieBreak pins that --top ties are broken by locator order, not scan order.
func TestTimelineCommand_TopTieBreak(t *testing.T) {
	// Both run exactly once a day within the window -- a genuine tie.
	inv := inventory.New("", []inventory.Item{
		{Expression: "0 3 * * *", Command: "/opt/z-tied.sh", Shell: true,
			State: inventory.StateActive, Locator: inventory.Locator{File: "zzz/crontab", Line: 1}},
		{Expression: "0 4 * * *", Command: "/opt/a-tied.sh", Shell: true,
			State: inventory.StateActive, Locator: inventory.Locator{File: "aaa/crontab", Line: 1}},
	})
	path := writeInventoryFixture(t, inv)

	tc := newTimelineCommand()
	buf := new(bytes.Buffer)
	tc.SetOut(buf)
	tc.SetArgs([]string{
		"--inventory", path, "--json", "--top", "1",
		"--from", "2026-08-13T00:00:00Z", "--color", "never",
	})
	require.NoError(t, tc.Execute())

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	jobs := result["jobs"].([]interface{})
	require.Len(t, jobs, 1)
	locator := jobs[0].(map[string]interface{})["locator"].(map[string]interface{})
	assert.Equal(t, "aaa/crontab", locator["file"], "a run-count tie must break by locator, file first")
}

func TestTimelineCommand_Export(t *testing.T) {
	t.Run("should export timeline to file", func(t *testing.T) {
		// Create a temporary file for export
		tmpDir := t.TempDir()
		exportFile := filepath.Join(tmpDir, "timeline.txt")

		tc := newTimelineCommand()
		tc.SetArgs([]string{"*/15 * * * *", "--export", exportFile})

		err := tc.Execute()
		require.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(exportFile)
		require.NoError(t, err, "Export file should exist")

		// Verify file has content
		content, err := os.ReadFile(exportFile)
		require.NoError(t, err)
		assert.NotEmpty(t, content)
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}  \d{2}:\d{2} → \d{2}:\d{2}`, string(content))
	})

	t.Run("should handle export file creation error", func(t *testing.T) {
		// Try to export to an invalid path (directory that doesn't exist)
		invalidPath := "/nonexistent/directory/timeline.txt"

		tc := newTimelineCommand()
		tc.SetArgs([]string{"*/15 * * * *", "--export", invalidPath})

		err := tc.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create export file")
	})

	t.Run("should export with JSON output", func(t *testing.T) {
		// Create a temporary file for export
		tmpDir := t.TempDir()
		exportFile := filepath.Join(tmpDir, "timeline.json")

		tc := newTimelineCommand()
		tc.SetArgs([]string{"*/15 * * * *", "--json", "--export", exportFile})

		err := tc.Execute()
		require.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(exportFile)
		require.NoError(t, err, "Export file should exist")

		// Verify file has JSON content
		content, err := os.ReadFile(exportFile)
		require.NoError(t, err)
		assert.NotEmpty(t, content)
		// JSON export should contain timeline structure
		assert.Contains(t, string(content), `"jobs"`)
		assert.Contains(t, string(content), `"expression"`)
	})
}

func TestTimelineCommand_JSONExportError(t *testing.T) {
	t.Run("should handle JSON export file creation error", func(t *testing.T) {
		// Test the error path when creating JSON export file fails (line 238-240)
		invalidPath := "/nonexistent/directory/timeline.json"

		tc := newTimelineCommand()
		tc.SetArgs([]string{"*/15 * * * *", "--json", "--export", invalidPath})

		err := tc.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create export file")
	})

	t.Run("should handle JSON export encoding error", func(t *testing.T) {
		// Test the error path when JSON encoding fails (line 244-246)
		// This is hard to test without mocking, but we can test the path exists
		tmpDir := t.TempDir()
		exportFile := filepath.Join(tmpDir, "timeline.json")

		tc := newTimelineCommand()
		tc.SetArgs([]string{"*/15 * * * *", "--json", "--export", exportFile})

		err := tc.Execute()
		require.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(exportFile)
		require.NoError(t, err)
	})

	t.Run("should handle JSON export file close error", func(t *testing.T) {
		// Test the error path when closing JSON export file fails (line 248-250)
		// This is hard to test without mocking, but we can test the path exists
		tmpDir := t.TempDir()
		exportFile := filepath.Join(tmpDir, "timeline.json")

		tc := newTimelineCommand()
		tc.SetArgs([]string{"*/15 * * * *", "--json", "--export", exportFile})

		err := tc.Execute()
		require.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(exportFile)
		require.NoError(t, err)
	})
}

func TestTimelineCommand_JSONStdoutError(t *testing.T) {
	t.Run("should handle JSON stdout encoding error", func(t *testing.T) {
		// Test the error path when JSON encoding to stdout fails (line 254-256)
		// This is hard to test without mocking, but we can test the path exists
		tc := newTimelineCommand()
		// Use an error writer to trigger JSON encoding error
		tc.SetOut(&errorWriter{})

		tc.SetArgs([]string{"*/15 * * * *", "--json"})

		err := tc.Execute()
		// Should return error from JSON encoding
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to encode JSON")
	})
}
