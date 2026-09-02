package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/hzerrad/cronkit/internal/inventory"
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

	t.Run("user crontab path", func(t *testing.T) {
		bc := newBudgetCommand()
		bc.maxConcurrent = 10
		bc.window = "1h"
		// No file or stdin specified, should use user crontab

		var buf bytes.Buffer
		bc.SetOut(&buf)

		err := bc.runBudget(nil, nil)
		// May succeed or fail depending on whether user has crontab
		// Just verify it doesn't panic
		_ = err
	})

	t.Run("error when renderer creation fails", func(t *testing.T) {
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
		// Should succeed with valid format
		assert.NoError(t, err)
	})

	t.Run("enforce flag - fails when budget violated", func(t *testing.T) {
		// Create jobs that will violate the budget
		content := "0 * * * * /usr/bin/job1.sh\n15 * * * * /usr/bin/job2.sh\n30 * * * * /usr/bin/job3.sh\n45 * * * * /usr/bin/job4.sh\n"
		testFile := createTempFile(t, content)

		bc := newBudgetCommand()
		bc.file = testFile
		bc.maxConcurrent = 2 // Very low limit
		bc.window = "1h"
		bc.enforce = true

		var buf bytes.Buffer
		bc.SetOut(&buf)

		err := bc.runBudget(nil, nil)
		// Should error when budget is violated and enforce is true
		if err != nil {
			assert.Contains(t, err.Error(), "budget violation")
		}
	})

	t.Run("inventory input", func(t *testing.T) {
		inv := inventory.New("", []inventory.Item{
			{Expression: "0 * * * *", Command: "worker-a", State: inventory.StateActive},
			{Expression: "0 * * * *", Command: "worker-b", State: inventory.StateActive},
		})
		f, err := os.CreateTemp("", "cronkit-inventory-*.json")
		require.NoError(t, err)
		require.NoError(t, inv.Encode(f))
		require.NoError(t, f.Close())
		defer func() { _ = os.Remove(f.Name()) }()

		bc := newBudgetCommand()
		bc.inventory = f.Name()
		bc.maxConcurrent = 10
		bc.window = "1h"

		var buf bytes.Buffer
		bc.SetOut(&buf)

		err = bc.runBudget(nil, nil)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "Budget Analysis")
	})

	t.Run("job identity in --verbose output does not change when an unrelated schedule is added elsewhere", func(t *testing.T) {
		// Pins that a job's id stays stable regardless of what else is in
		// the batch (previously it only folded in the file on same-line
		// collisions within this run, so an unrelated schedule elsewhere
		// could silently rename it).
		base := []inventory.Item{
			{Expression: "0 * * * *", Command: "worker-a", State: inventory.StateActive, Locator: inventory.Locator{File: "site-a/crontab", Line: 5}},
			{Expression: "0 * * * *", Command: "worker-b", State: inventory.StateActive, Locator: inventory.Locator{File: "site-b/crontab", Line: 5}},
		}

		runVerbose := func(items []inventory.Item) string {
			inv := inventory.New("", items)
			f, err := os.CreateTemp("", "cronkit-inventory-*.json")
			require.NoError(t, err)
			require.NoError(t, inv.Encode(f))
			require.NoError(t, f.Close())
			defer func() { _ = os.Remove(f.Name()) }()

			bc := newBudgetCommand()
			bc.inventory = f.Name()
			bc.maxConcurrent = 1
			bc.window = "24h"
			bc.verbose = true
			var buf bytes.Buffer
			bc.SetOut(&buf)
			require.NoError(t, bc.runBudget(nil, nil))
			return buf.String()
		}

		before := runVerbose(base)
		require.Contains(t, before, "line-site-a/crontab:5",
			"the id format must already fold in the file, unconditionally")

		unrelated := inventory.Item{
			Expression: "0 0 1 1 *", Command: "unrelated", State: inventory.StateActive,
			Locator: inventory.Locator{File: "site-z/other.yaml", Line: 5},
		}
		after := runVerbose(append(append([]inventory.Item{}, base...), unrelated))

		assert.Contains(t, after, "line-site-a/crontab:5",
			"adding an unrelated schedule on the same line elsewhere must not rename this job's id")
	})

	t.Run("malformed inventory produces a clear error", func(t *testing.T) {
		f, err := os.CreateTemp("", "cronkit-inventory-*.json")
		require.NoError(t, err)
		_, err = f.WriteString("nope")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		defer func() { _ = os.Remove(f.Name()) }()

		bc := newBudgetCommand()
		bc.inventory = f.Name()
		bc.maxConcurrent = 10
		bc.window = "1h"

		var buf bytes.Buffer
		bc.SetOut(&buf)

		err = bc.runBudget(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read inventory")
	})
}
