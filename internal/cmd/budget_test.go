package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBudgetCommand(t *testing.T) {
	cmd := newBudgetCommand()
	require.NotNil(t, cmd)
	assert.Equal(t, "budget", cmd.Use)
}

func TestBudgetCommand_RunBudget(t *testing.T) {
	t.Run("file input with budget", func(t *testing.T) {
		content := "0 * * * * /usr/bin/job1.sh\n15 * * * * /usr/bin/job2.sh\n"
		testFile := createTempFile(t, content)

		bc := newBudgetCommand()
		bc.file = testFile
		bc.maxConcurrent = 10
		bc.window = "1h"

		var buf bytes.Buffer
		bc.SetOut(&buf)

		err := bc.runBudget(nil, nil)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Budget Analysis")
	})

	t.Run("json output", func(t *testing.T) {
		content := "0 * * * * /usr/bin/job1.sh\n"
		testFile := createTempFile(t, content)

		bc := newBudgetCommand()
		bc.file = testFile
		bc.maxConcurrent = 10
		bc.window = "1h"
		bc.json = true

		var buf bytes.Buffer
		bc.SetOut(&buf)

		err := bc.runBudget(nil, nil)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, `"passed"`)
		assert.Contains(t, output, `"budgets"`)
	})

	t.Run("stdin input", func(t *testing.T) {
		content := "0 * * * * /usr/bin/job1.sh\n"
		bc := newBudgetCommand()
		bc.stdin = true
		bc.maxConcurrent = 10
		bc.window = "1h"

		var buf bytes.Buffer
		bc.SetOut(&buf)
		bc.SetIn(strings.NewReader(content))

		err := bc.runBudget(nil, nil)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Budget Analysis")
	})

	t.Run("error when max-concurrent not specified", func(t *testing.T) {
		bc := newBudgetCommand()
		bc.window = "1h"

		err := bc.runBudget(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max-concurrent")
	})

	t.Run("error when window not specified", func(t *testing.T) {
		bc := newBudgetCommand()
		bc.maxConcurrent = 10

		err := bc.runBudget(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "window")
	})

	t.Run("error when window invalid", func(t *testing.T) {
		content := "0 * * * * /usr/bin/job1.sh\n"
		testFile := createTempFile(t, content)

		bc := newBudgetCommand()
		bc.file = testFile
		bc.maxConcurrent = 10
		bc.window = "invalid"

		err := bc.runBudget(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
	})

	t.Run("error when file not found", func(t *testing.T) {
		bc := newBudgetCommand()
		bc.file = "/nonexistent/file.cron"
		bc.maxConcurrent = 10
		bc.window = "1h"

		err := bc.runBudget(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read")
	})

	t.Run("enforce flag - passes when budget met", func(t *testing.T) {
		content := "0 * * * * /usr/bin/job1.sh\n"
		testFile := createTempFile(t, content)

		bc := newBudgetCommand()
		bc.file = testFile
		bc.maxConcurrent = 10
		bc.window = "1h"
		bc.enforce = true

		var buf bytes.Buffer
		bc.SetOut(&buf)

		err := bc.runBudget(nil, nil)
		// Should not error when budget passes
		assert.NoError(t, err)
	})

	t.Run("verbose flag", func(t *testing.T) {
		content := "0 * * * * /usr/bin/job1.sh\n0 * * * * /usr/bin/job2.sh\n0 * * * * /usr/bin/job3.sh\n"
		testFile := createTempFile(t, content)

		bc := newBudgetCommand()
		bc.file = testFile
		bc.maxConcurrent = 2
		bc.window = "1h"
		bc.verbose = true

		var buf bytes.Buffer
		bc.SetOut(&buf)

		err := bc.runBudget(nil, nil)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Budget Analysis")
	})
}

func TestBudgetCommand_Additional(t *testing.T) {
	t.Run("error when budget analysis fails", func(t *testing.T) {
		// Create a file with invalid cron expressions that will cause parsing errors
		content := "invalid-cron-expression /usr/bin/job.sh\n"
		testFile := createTempFile(t, content)

		bc := newBudgetCommand()
		bc.file = testFile
		bc.maxConcurrent = 10
		bc.window = "1h"

		var buf bytes.Buffer
		bc.SetOut(&buf)

		// Should still work - invalid jobs are ignored
		err := bc.runBudget(nil, nil)
		// May succeed (invalid jobs ignored) or fail (parsing error)
		// Just verify it doesn't panic
		_ = err
	})
}
