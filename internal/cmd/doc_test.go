package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocCommand(t *testing.T) {
	t.Run("doc command should be registered", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"doc"})
		assert.NoError(t, err)
		assert.Equal(t, "doc", cmd.Name())
	})

	t.Run("doc command should have metadata", func(t *testing.T) {
		dc := newDocCommand()
		assert.NotEmpty(t, dc.Short)
		assert.NotEmpty(t, dc.Long)
		assert.Contains(t, dc.Use, "doc")
	})

	t.Run("doc command should have all flags", func(t *testing.T) {
		dc := newDocCommand()
		assert.NotNil(t, dc.Flag("file"))
		assert.NotNil(t, dc.Flag("stdin"))
		assert.NotNil(t, dc.Flag("output"))
		assert.NotNil(t, dc.Flag("format"))
		assert.NotNil(t, dc.Flag("include-next"))
		assert.NotNil(t, dc.Flag("include-warnings"))
		assert.NotNil(t, dc.Flag("include-stats"))
	})

	t.Run("should generate markdown from file", func(t *testing.T) {
		dc := newDocCommand()
		buf := new(bytes.Buffer)
		dc.SetOut(buf)

		testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")
		dc.SetArgs([]string{"--file", testFile, "--format", "md"})

		err := dc.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "# Crontab Documentation")
		assert.Contains(t, output, "## Summary")
		assert.Contains(t, output, "## Jobs")
	})

	t.Run("should generate HTML from file", func(t *testing.T) {
		dc := newDocCommand()
		buf := new(bytes.Buffer)
		dc.SetOut(buf)

		testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")
		dc.SetArgs([]string{"--file", testFile, "--format", "html"})

		err := dc.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "<!DOCTYPE html>")
		assert.Contains(t, output, "<html>")
		assert.Contains(t, output, "<title>Crontab Documentation</title>")
	})

	t.Run("should generate JSON from file", func(t *testing.T) {
		dc := newDocCommand()
		buf := new(bytes.Buffer)
		dc.SetOut(buf)

		testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")
		dc.SetArgs([]string{"--file", testFile, "--format", "json"})

		err := dc.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, `"Source"`)
		assert.Contains(t, output, `"Jobs"`)
	})

	t.Run("should write to output file", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "output.md")

		dc := newDocCommand()
		testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")
		dc.SetArgs([]string{"--file", testFile, "--format", "md", "--output", outputFile})

		err := dc.Execute()
		require.NoError(t, err)

		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "# Crontab Documentation")
	})

	t.Run("should read from stdin", func(t *testing.T) {
		dc := newDocCommand()
		buf := new(bytes.Buffer)
		dc.SetOut(buf)

		crontabContent := "0 2 * * * /usr/bin/backup.sh\n*/15 * * * * /usr/bin/check.sh\n"
		dc.SetIn(strings.NewReader(crontabContent))
		dc.SetArgs([]string{"--stdin", "--format", "md"})

		err := dc.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "# Crontab Documentation")
		assert.Contains(t, output, "backup.sh")
		assert.Contains(t, output, "check.sh")
	})

	t.Run("should include next runs when requested", func(t *testing.T) {
		dc := newDocCommand()
		buf := new(bytes.Buffer)
		dc.SetOut(buf)

		testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")
		dc.SetArgs([]string{"--file", testFile, "--format", "md", "--include-next", "5"})

		err := dc.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Next Runs")
	})

	t.Run("should include warnings when requested", func(t *testing.T) {
		dc := newDocCommand()
		buf := new(bytes.Buffer)
		dc.SetOut(buf)

		testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")
		dc.SetArgs([]string{"--file", testFile, "--format", "md", "--include-warnings"})

		err := dc.Execute()
		require.NoError(t, err)

		output := buf.String()
		// May or may not have warnings depending on the crontab
		assert.Contains(t, output, "# Crontab Documentation")
	})

	t.Run("should include stats when requested", func(t *testing.T) {
		dc := newDocCommand()
		buf := new(bytes.Buffer)
		dc.SetOut(buf)

		testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")
		dc.SetArgs([]string{"--file", testFile, "--format", "md", "--include-stats"})

		err := dc.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Statistics")
	})

	t.Run("should reject invalid format", func(t *testing.T) {
		dc := newDocCommand()
		buf := new(bytes.Buffer)
		dc.SetErr(buf)

		testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")
		dc.SetArgs([]string{"--file", testFile, "--format", "invalid"})

		err := dc.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid format")
	})

	t.Run("should handle file not found", func(t *testing.T) {
		dc := newDocCommand()
		buf := new(bytes.Buffer)
		dc.SetErr(buf)

		dc.SetArgs([]string{"--file", "nonexistent.cron", "--format", "md"})

		err := dc.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read crontab")
	})

	t.Run("should handle empty file", func(t *testing.T) {
		dc := newDocCommand()
		buf := new(bytes.Buffer)
		dc.SetOut(buf)

		testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "empty.cron")
		dc.SetArgs([]string{"--file", testFile, "--format", "md"})

		err := dc.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "# Crontab Documentation")
	})

	t.Run("should handle invalid crontab entries gracefully", func(t *testing.T) {
		dc := newDocCommand()
		buf := new(bytes.Buffer)
		dc.SetOut(buf)

		crontabContent := "60 0 * * * /usr/bin/invalid.sh\n"
		dc.SetIn(strings.NewReader(crontabContent))
		dc.SetArgs([]string{"--stdin", "--format", "md"})

		err := dc.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "# Crontab Documentation")
	})

	t.Run("should handle output file creation error", func(t *testing.T) {
		dc := newDocCommand()
		testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")
		// Use a path that cannot be created (e.g., in a non-existent directory)
		dc.SetArgs([]string{"--file", testFile, "--format", "md", "--output", "/nonexistent/path/output.md"})

		err := dc.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create output file")
	})
}
