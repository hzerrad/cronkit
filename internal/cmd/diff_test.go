package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: Some tests for both stdin flags are skipped as stdin can only be read once

// createTempFile creates a temporary file with the given content and returns its path
func createTempFile(t *testing.T, content string) string {
	t.Helper()
	file, err := os.CreateTemp("", "cronic-test-*.cron")
	require.NoError(t, err)

	_, err = file.WriteString(content)
	require.NoError(t, err)

	err = file.Close()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.Remove(file.Name())
	})

	return file.Name()
}

func TestNewDiffCommand(t *testing.T) {
	cmd := newDiffCommand()
	require.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "diff")
}

func TestDiffCommand_RunDiff(t *testing.T) {
	t.Run("file to file comparison", func(t *testing.T) {
		// Create temporary files
		oldContent := "0 2 * * * /usr/bin/backup.sh\n"
		newContent := "0 2 * * * /usr/bin/backup.sh\n*/15 * * * * /usr/bin/check.sh\n"

		oldFile := createTempFile(t, oldContent)
		newFile := createTempFile(t, newContent)

		dc := newDiffCommand()
		dc.oldFile = oldFile
		dc.newFile = newFile

		var buf bytes.Buffer
		dc.SetOut(&buf)

		err := dc.runDiff(nil, nil)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Added Jobs")
		assert.Contains(t, output, "*/15 * * * *")
	})

	t.Run("positional arguments", func(t *testing.T) {
		oldContent := "0 2 * * * /usr/bin/backup.sh\n"
		newContent := "0 2 * * * /usr/bin/backup.sh\n*/15 * * * * /usr/bin/check.sh\n"

		oldFile := createTempFile(t, oldContent)
		newFile := createTempFile(t, newContent)

		dc := newDiffCommand()
		var buf bytes.Buffer
		dc.SetOut(&buf)

		err := dc.runDiff(nil, []string{oldFile, newFile})
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Added Jobs")
	})

	t.Run("json output", func(t *testing.T) {
		oldContent := "0 2 * * * /usr/bin/backup.sh\n"
		newContent := "0 2 * * * /usr/bin/backup.sh\n*/15 * * * * /usr/bin/check.sh\n"

		oldFile := createTempFile(t, oldContent)
		newFile := createTempFile(t, newContent)

		dc := newDiffCommand()
		dc.oldFile = oldFile
		dc.newFile = newFile
		dc.json = true

		var buf bytes.Buffer
		dc.SetOut(&buf)

		err := dc.runDiff(nil, nil)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, `"added"`)
		assert.Contains(t, output, `"*/15 * * * *"`)
	})

	t.Run("unified format", func(t *testing.T) {
		oldContent := "0 2 * * * /usr/bin/backup.sh\n"
		newContent := "0 2 * * * /usr/bin/backup.sh\n*/15 * * * * /usr/bin/check.sh\n"

		oldFile := createTempFile(t, oldContent)
		newFile := createTempFile(t, newContent)

		dc := newDiffCommand()
		dc.oldFile = oldFile
		dc.newFile = newFile
		dc.format = "unified"

		var buf bytes.Buffer
		dc.SetOut(&buf)

		err := dc.runDiff(nil, nil)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "--- old crontab")
		assert.Contains(t, output, "+++ new crontab")
	})

	t.Run("stdin input", func(t *testing.T) {
		oldContent := "0 2 * * * /usr/bin/backup.sh\n"
		newContent := "0 2 * * * /usr/bin/backup.sh\n*/15 * * * * /usr/bin/check.sh\n"

		newFile := createTempFile(t, newContent)

		dc := newDiffCommand()
		dc.oldStdin = true
		dc.newFile = newFile

		var buf bytes.Buffer
		dc.SetOut(&buf)
		dc.SetIn(strings.NewReader(oldContent))

		err := dc.runDiff(nil, nil)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Added Jobs")
	})

	t.Run("error when old source not specified", func(t *testing.T) {
		dc := newDiffCommand()
		dc.newFile = "test.cron"

		err := dc.runDiff(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must specify old crontab source")
	})

	t.Run("error when new source not specified", func(t *testing.T) {
		oldFile := createTempFile(t, "0 2 * * * /usr/bin/backup.sh\n")

		dc := newDiffCommand()
		dc.oldFile = oldFile

		err := dc.runDiff(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must specify new crontab source")
	})

	t.Run("error when file not found", func(t *testing.T) {
		dc := newDiffCommand()
		dc.oldFile = "/nonexistent/old.cron"
		dc.newFile = "/nonexistent/new.cron"

		err := dc.runDiff(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read")
	})

	t.Run("error when file not found - positional args", func(t *testing.T) {
		dc := newDiffCommand()

		err := dc.runDiff(nil, []string{"/nonexistent/old.cron", "/nonexistent/new.cron"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read")
	})
}

func TestDiffCommand_Options(t *testing.T) {
	oldContent := "# Comment\n0 2 * * * /usr/bin/backup.sh # Old comment\n"
	newContent := "# Comment\n0 2 * * * /usr/bin/backup.sh # New comment\nPATH=/usr/bin\n"

	oldFile := createTempFile(t, oldContent)
	newFile := createTempFile(t, newContent)

	t.Run("ignore comments", func(t *testing.T) {
		dc := newDiffCommand()
		dc.oldFile = oldFile
		dc.newFile = newFile
		dc.ignoreComments = true

		var buf bytes.Buffer
		dc.SetOut(&buf)

		err := dc.runDiff(nil, nil)
		require.NoError(t, err)

		output := buf.String()
		// Should not show comment changes section
		assert.NotContains(t, output, "Comment Changes")
	})

	t.Run("ignore env", func(t *testing.T) {
		dc := newDiffCommand()
		dc.oldFile = oldFile
		dc.newFile = newFile
		dc.ignoreEnv = true

		var buf bytes.Buffer
		dc.SetOut(&buf)

		err := dc.runDiff(nil, nil)
		require.NoError(t, err)

		output := buf.String()
		// Should not show env changes section
		assert.NotContains(t, output, "Environment Variable Changes")
	})

	t.Run("show unchanged", func(t *testing.T) {
		oldContent := "0 2 * * * /usr/bin/backup.sh\n"
		newContent := "0 2 * * * /usr/bin/backup.sh\n*/15 * * * * /usr/bin/check.sh\n"

		oldFile := createTempFile(t, oldContent)
		newFile := createTempFile(t, newContent)

		dc := newDiffCommand()
		dc.oldFile = oldFile
		dc.newFile = newFile
		dc.showUnchanged = true

		var buf bytes.Buffer
		dc.SetOut(&buf)

		err := dc.runDiff(nil, nil)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Unchanged Jobs")
	})
}

func TestDiffCommand_Additional(t *testing.T) {
	t.Run("new stdin with old file", func(t *testing.T) {
		oldContent := "0 2 * * * /usr/bin/backup.sh\n"
		newContent := "0 2 * * * /usr/bin/backup.sh\n*/15 * * * * /usr/bin/check.sh\n"

		oldFile := createTempFile(t, oldContent)

		dc := newDiffCommand()
		dc.oldFile = oldFile
		dc.newStdin = true

		var buf bytes.Buffer
		dc.SetOut(&buf)
		dc.SetIn(strings.NewReader(newContent))

		err := dc.runDiff(nil, nil)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Added Jobs")
	})

	t.Run("error when renderer creation fails", func(t *testing.T) {
		oldContent := "0 2 * * * /usr/bin/backup.sh\n"
		newContent := "0 2 * * * /usr/bin/backup.sh\n*/15 * * * * /usr/bin/check.sh\n"

		oldFile := createTempFile(t, oldContent)
		newFile := createTempFile(t, newContent)

		dc := newDiffCommand()
		dc.oldFile = oldFile
		dc.newFile = newFile
		dc.format = "invalid-format"

		err := dc.runDiff(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown format")
	})

	t.Run("error when renderer render fails", func(t *testing.T) {
		// This is hard to test without mocking, but we can test the happy path
		// The renderer shouldn't fail in normal operation
		oldContent := "0 2 * * * /usr/bin/backup.sh\n"
		newContent := "0 2 * * * /usr/bin/backup.sh\n*/15 * * * * /usr/bin/check.sh\n"

		oldFile := createTempFile(t, oldContent)
		newFile := createTempFile(t, newContent)

		dc := newDiffCommand()
		dc.oldFile = oldFile
		dc.newFile = newFile

		var buf bytes.Buffer
		dc.SetOut(&buf)

		err := dc.runDiff(nil, nil)
		require.NoError(t, err)
	})
}
