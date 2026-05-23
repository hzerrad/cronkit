package budget

import (
	"testing"
	"time"

	"github.com/hzerrad/cronkit/internal/crontab"
	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeBudget(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()

	t.Run("budget passes when no violations", func(t *testing.T) {
		jobs := []*crontab.Job{
			{
				Expression: "0 * * * *",
				Command:    "/usr/bin/job1.sh",
				Valid:      true,
			},
			{
				Expression: "15 * * * *",
				Command:    "/usr/bin/job2.sh",
				Valid:      true,
			},
		}

		budgets := []Budget{
			{
				MaxConcurrent: 10,
				TimeWindow:    1 * time.Hour,
				Name:          "test-budget",
			},
		}

		report, err := AnalyzeBudget(jobs, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.True(t, report.Passed)
		assert.Len(t, report.Budgets, 1)
		assert.True(t, report.Budgets[0].Passed)
		assert.LessOrEqual(t, report.Budgets[0].MaxFound, budgets[0].MaxConcurrent)
	})

	t.Run("budget fails when violations found", func(t *testing.T) {
		// Three jobs that fire every minute, so they always run concurrently.
		jobs := []*crontab.Job{
			{
				LineNumber: 1,
				Expression: "* * * * *",
				Command:    "/usr/bin/job1.sh",
				Valid:      true,
			},
			{
				LineNumber: 2,
				Expression: "* * * * *",
				Command:    "/usr/bin/job2.sh",
				Valid:      true,
			},
			{
				LineNumber: 3,
				Expression: "* * * * *",
				Command:    "/usr/bin/job3.sh",
				Valid:      true,
			},
		}

		budgets := []Budget{
			{
				MaxConcurrent: 2,
				TimeWindow:    1 * time.Hour,
				Name:          "test-budget",
			},
		}

		report, err := AnalyzeBudget(jobs, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.Len(t, report.Budgets, 1)

		// MaxFound is 3 (all three every minute), over the limit of 2.
		assert.False(t, report.Budgets[0].Passed)
		assert.False(t, report.Passed)
		assert.Greater(t, len(report.Budgets[0].Violations), 0)
	})

	t.Run("multiple budgets", func(t *testing.T) {
		jobs := []*crontab.Job{
			{
				LineNumber: 1,
				Expression: "* * * * *",
				Command:    "/usr/bin/job1.sh",
				Valid:      true,
			},
			{
				LineNumber: 2,
				Expression: "* * * * *",
				Command:    "/usr/bin/job2.sh",
				Valid:      true,
			},
		}

		budgets := []Budget{
			{
				MaxConcurrent: 10,
				TimeWindow:    1 * time.Hour,
				Name:          "hourly-budget",
			},
			{
				MaxConcurrent: 1,
				TimeWindow:    1 * time.Hour,
				Name:          "strict-budget",
			},
		}

		report, err := AnalyzeBudget(jobs, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.Len(t, report.Budgets, 2)
		assert.True(t, report.Budgets[0].Passed) // First budget passes (limit 10, found 2)

		// Both jobs fire every minute, so the strict budget (limit 1) fails.
		assert.False(t, report.Budgets[1].Passed)
		assert.False(t, report.Passed)
	})

	t.Run("error when no budgets specified", func(t *testing.T) {
		jobs := []*crontab.Job{
			{
				Expression: "0 * * * *",
				Command:    "/usr/bin/job1.sh",
				Valid:      true,
			},
		}

		_, err := AnalyzeBudget(jobs, []Budget{}, scheduler, parser)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no budgets specified")
	})

	t.Run("ignores invalid jobs", func(t *testing.T) {
		jobs := []*crontab.Job{
			{
				Expression: "0 * * * *",
				Command:    "/usr/bin/job1.sh",
				Valid:      true,
			},
			{
				Expression: "invalid",
				Command:    "/usr/bin/job2.sh",
				Valid:      false,
			},
		}

		budgets := []Budget{
			{
				MaxConcurrent: 1,
				TimeWindow:    1 * time.Hour,
				Name:          "test-budget",
			},
		}

		report, err := AnalyzeBudget(jobs, budgets, scheduler, parser)
		require.NoError(t, err)
		// Should pass because only one valid job
		assert.True(t, report.Passed)
	})

	t.Run("empty jobs list", func(t *testing.T) {
		budgets := []Budget{
			{
				MaxConcurrent: 1,
				TimeWindow:    1 * time.Hour,
				Name:          "test-budget",
			},
		}

		report, err := AnalyzeBudget([]*crontab.Job{}, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.True(t, report.Passed)
		assert.Equal(t, 0, report.Budgets[0].MaxFound)
	})
}

func TestAnalyzeSingleBudget(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()

	t.Run("finds violations correctly", func(t *testing.T) {
		jobs := []*crontab.Job{
			{
				LineNumber: 1,
				Expression: "* * * * *",
				Command:    "/usr/bin/job1.sh",
				Valid:      true,
			},
			{
				LineNumber: 2,
				Expression: "* * * * *",
				Command:    "/usr/bin/job2.sh",
				Valid:      true,
			},
			{
				LineNumber: 3,
				Expression: "* * * * *",
				Command:    "/usr/bin/job3.sh",
				Valid:      true,
			},
		}

		budget := Budget{
			MaxConcurrent: 2,
			TimeWindow:    1 * time.Hour,
			Name:          "test-budget",
		}

		result, err := analyzeSingleBudget(jobs, budget, scheduler, parser)
		require.NoError(t, err)

		// All jobs fire every minute, so MaxFound is 3 — over the limit of 2.
		assert.False(t, result.Passed)
		assert.Greater(t, len(result.Violations), 0)

		violation := result.Violations[0]
		assert.Greater(t, violation.Count, budget.MaxConcurrent)
		assert.Greater(t, len(violation.Jobs), budget.MaxConcurrent)
	})
}
