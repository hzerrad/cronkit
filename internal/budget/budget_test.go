package budget

import (
	"sort"
	"testing"
	"time"

	"github.com/hzerrad/cronkit/internal/crontab"
	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeBudget(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("budget passes when no violations", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
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
		}, "crontab")

		budgets := []Budget{
			{
				MaxConcurrent: 10,
				TimeWindow:    1 * time.Hour,
				Name:          "test-budget",
			},
		}

		report, err := AnalyzeBudget(items, from, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.True(t, report.Passed)
		assert.Len(t, report.Budgets, 1)
		assert.True(t, report.Budgets[0].Passed)
		assert.LessOrEqual(t, report.Budgets[0].MaxFound, budgets[0].MaxConcurrent)
	})

	t.Run("budget fails when violations found", func(t *testing.T) {
		// Three jobs that fire every minute, so they always run concurrently.
		items := inventory.FromCrontabJobs([]*crontab.Job{
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
		}, "crontab")

		budgets := []Budget{
			{
				MaxConcurrent: 2,
				TimeWindow:    1 * time.Hour,
				Name:          "test-budget",
			},
		}

		report, err := AnalyzeBudget(items, from, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.Len(t, report.Budgets, 1)

		// MaxFound is 3 (all three every minute), over the limit of 2.
		assert.False(t, report.Budgets[0].Passed)
		assert.False(t, report.Passed)
		assert.Greater(t, len(report.Budgets[0].Violations), 0)
	})

	t.Run("multiple budgets", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
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
		}, "crontab")

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

		report, err := AnalyzeBudget(items, from, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.Len(t, report.Budgets, 2)
		assert.True(t, report.Budgets[0].Passed) // First budget passes (limit 10, found 2)

		// Both jobs fire every minute, so the strict budget (limit 1) fails.
		assert.False(t, report.Budgets[1].Passed)
		assert.False(t, report.Passed)
	})

	t.Run("error when no budgets specified", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{
				Expression: "0 * * * *",
				Command:    "/usr/bin/job1.sh",
				Valid:      true,
			},
		}, "crontab")

		_, err := AnalyzeBudget(items, from, []Budget{}, scheduler, parser)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no budgets specified")
	})

	t.Run("ignores invalid jobs", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
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
		}, "crontab")

		budgets := []Budget{
			{
				MaxConcurrent: 1,
				TimeWindow:    1 * time.Hour,
				Name:          "test-budget",
			},
		}

		report, err := AnalyzeBudget(items, from, budgets, scheduler, parser)
		require.NoError(t, err)
		// Should pass because only one valid job
		assert.True(t, report.Passed)
	})

	t.Run("ignores suspended items", func(t *testing.T) {
		// A suspended Kubernetes CronJob does not run, so it cannot consume a
		// concurrency budget.
		items := []inventory.Item{
			{Expression: "0 * * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
			{Expression: "0 * * * *", State: inventory.StateSuspended, Locator: inventory.Locator{Line: 2}},
		}

		budgets := []Budget{
			{MaxConcurrent: 1, TimeWindow: 1 * time.Hour, Name: "test-budget"},
		}

		report, err := AnalyzeBudget(items, from, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.True(t, report.Passed, "the suspended item must not count toward concurrency")
	})

	t.Run("an unresolvable timezone widens the worst case once admitted", func(t *testing.T) {
		// Runs items through the same admission step (inventory.ResolveTimezones) a real caller does.
		items := inventory.ResolveTimezones([]inventory.Item{
			{Expression: "* * * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
			{Expression: "* * * * *", Timezone: "Not/AZone", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
		})
		budgets := []Budget{
			{MaxConcurrent: 1, TimeWindow: 1 * time.Hour, Name: "strict"},
		}

		report, err := AnalyzeBudget(items, from, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.False(t, report.Passed, "the bad-zone item must not silently vanish from the analysis")
		assert.Equal(t, 2, report.Budgets[0].MaxFound,
			"the one resolvable job plus the one unanalysable job must both count toward the worst case")
	})

	t.Run("an unresolvable timezone alone still widens the worst case even with no resolvable items", func(t *testing.T) {
		items := inventory.ResolveTimezones([]inventory.Item{
			{Expression: "* * * * *", Timezone: "Not/AZone", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
		})
		budgets := []Budget{
			{MaxConcurrent: 0, TimeWindow: 1 * time.Hour, Name: "strict"},
		}

		report, err := AnalyzeBudget(items, from, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.False(t, report.Passed)
		assert.Equal(t, 1, report.Budgets[0].MaxFound)
	})

	t.Run("an unresolvable timezone is reported on the result, not just excluded from the arithmetic", func(t *testing.T) {
		// Constructs the second item exactly as inventory.ResolveTimezones would leave it.
		items := []inventory.Item{
			{Expression: "* * * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
			{
				Expression: "* * * * *",
				Timezone:   "Not/AZone",
				State:      inventory.StateInvalid,
				Reason:     `unresolvable timezone "Not/AZone": unknown time zone Not/AZone`,
				Locator:    inventory.Locator{File: "ops/crontab", Line: 2},
			},
		}
		budgets := []Budget{
			{MaxConcurrent: 1, TimeWindow: 1 * time.Hour, Name: "strict"},
		}

		report, err := AnalyzeBudget(items, from, budgets, scheduler, parser)
		require.NoError(t, err)
		require.Len(t, report.Budgets[0].Unresolved, 1)
		unresolved := report.Budgets[0].Unresolved[0]
		assert.Equal(t, "* * * * *", unresolved.Expression)
		assert.Equal(t, inventory.Locator{File: "ops/crontab", Line: 2}, unresolved.Locator)
		assert.Contains(t, unresolved.Reason, "Not/AZone")
	})

	t.Run("empty jobs list", func(t *testing.T) {
		budgets := []Budget{
			{
				MaxConcurrent: 1,
				TimeWindow:    1 * time.Hour,
				Name:          "test-budget",
			},
		}

		report, err := AnalyzeBudget(nil, from, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.True(t, report.Passed)
		assert.Equal(t, 0, report.Budgets[0].MaxFound)
	})

	t.Run("jobs without line numbers count separately", func(t *testing.T) {
		// Two ad-hoc jobs share a schedule and have no line number. They must
		// still count as two concurrent jobs, not collapse into one.
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{Expression: "* * * * *", Command: "/usr/bin/a.sh", Valid: true},
			{Expression: "* * * * *", Command: "/usr/bin/b.sh", Valid: true},
		}, "crontab")
		budgets := []Budget{
			{MaxConcurrent: 1, TimeWindow: 1 * time.Hour, Name: "strict"},
		}

		report, err := AnalyzeBudget(items, from, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.Equal(t, 2, report.Budgets[0].MaxFound)
		assert.False(t, report.Passed)
	})

	t.Run("a reboot job does not inflate the conservative fallback", func(t *testing.T) {
		// MaxFound here comes from the "no runs found" fallback; @reboot must not inflate it.
		from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		budgets := []Budget{
			{MaxConcurrent: 10, TimeWindow: 1 * time.Hour, Name: "test-budget"},
		}
		ordinary := &crontab.Job{LineNumber: 1, Expression: "0 0 1 1 *", Command: "/usr/bin/yearly.sh", Valid: true}
		reboot := &crontab.Job{LineNumber: 2, Expression: "@reboot", Command: "/usr/bin/boot.sh", Valid: true}

		without, err := AnalyzeBudget(inventory.FromCrontabJobs([]*crontab.Job{ordinary}, "crontab"), from, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.Equal(t, 1, without.Budgets[0].MaxFound)

		with, err := AnalyzeBudget(inventory.FromCrontabJobs([]*crontab.Job{ordinary, reboot}, "crontab"), from, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.Equal(t, without.Budgets[0].MaxFound, with.Budgets[0].MaxFound,
			"a boot trigger with no computable runs must not change the conservative estimate")
	})

	t.Run("a crontab of only reboot jobs has no concurrency at all", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "@reboot", Command: "/usr/bin/a.sh", Valid: true},
			{LineNumber: 2, Expression: "@reboot", Command: "/usr/bin/b.sh", Valid: true},
		}, "crontab")
		budgets := []Budget{
			{MaxConcurrent: 1, TimeWindow: 1 * time.Hour, Name: "test-budget"},
		}

		report, err := AnalyzeBudget(items, from, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.Equal(t, 0, report.Budgets[0].MaxFound)
		assert.True(t, report.Passed)
	})
}

// TestAnalyzeBudget_CrossZoneCollision confirms same-instant items in different zones violate budget 1.
func TestAnalyzeBudget_CrossZoneCollision(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()
	from := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)

	items := []inventory.Item{
		{
			Expression: "0 12 * * *",
			Timezone:   "Europe/London",
			State:      inventory.StateActive,
			Locator:    inventory.Locator{Line: 1},
		},
		{
			Expression: "0 13 * * *",
			Timezone:   "Europe/Paris",
			State:      inventory.StateActive,
			Locator:    inventory.Locator{Line: 2},
		},
	}

	budgets := []Budget{
		{MaxConcurrent: 1, TimeWindow: 24 * time.Hour, Name: "strict"},
	}

	report, err := AnalyzeBudget(items, from, budgets, scheduler, parser)
	require.NoError(t, err)
	assert.Equal(t, 2, report.Budgets[0].MaxFound, "the two items collide at the same instant")
	assert.False(t, report.Budgets[0].Passed)
	assert.False(t, report.Passed)
	require.Len(t, report.Budgets[0].Violations, 1)
	assert.ElementsMatch(t, []string{"line-1", "line-2"}, report.Budgets[0].Violations[0].Jobs)
}

// TestAnalyzeBudget_SameWallClockDifferentZonesDoesNotCollide: same wall clock, different zones, no hit.
func TestAnalyzeBudget_SameWallClockDifferentZonesDoesNotCollide(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()
	from := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)

	items := []inventory.Item{
		{Expression: "0 12 * * *", Timezone: "Europe/London", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
		{Expression: "0 12 * * *", Timezone: "Europe/Paris", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
	}

	budgets := []Budget{
		{MaxConcurrent: 1, TimeWindow: 24 * time.Hour, Name: "strict"},
	}

	report, err := AnalyzeBudget(items, from, budgets, scheduler, parser)
	require.NoError(t, err)
	assert.Equal(t, 1, report.Budgets[0].MaxFound, "the same wall-clock hour in different zones is a different instant")
	assert.True(t, report.Passed)
}

// TestAnalyzeBudget_DaylightSavingTransition pins DST-aware zone handling, same as the check package.
func TestAnalyzeBudget_DaylightSavingTransition(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()

	items := []inventory.Item{
		{Expression: "0 9 * * *", Timezone: "America/New_York", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
		{Expression: "0 14 * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
	}
	budgets := []Budget{
		{MaxConcurrent: 1, TimeWindow: 24 * time.Hour, Name: "strict"},
	}

	t.Run("before the transition, 09:00 EST is 14:00 UTC and the budget is violated", func(t *testing.T) {
		from := time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC)
		report, err := AnalyzeBudget(items, from, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.Equal(t, 2, report.Budgets[0].MaxFound)
		assert.False(t, report.Passed)
	})

	t.Run("after the transition, 09:00 EDT is 13:00 UTC and the budget passes", func(t *testing.T) {
		from := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
		report, err := AnalyzeBudget(items, from, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.Equal(t, 1, report.Budgets[0].MaxFound)
		assert.True(t, report.Passed)
	})
}

// TestAnalyzeBudget_HalfHourOffset pins down that a half-hour UTC offset is honored exactly.
func TestAnalyzeBudget_HalfHourOffset(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()
	from := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	budgets := []Budget{
		{MaxConcurrent: 1, TimeWindow: 24 * time.Hour, Name: "strict"},
	}

	t.Run("the exact half-hour match violates the budget", func(t *testing.T) {
		items := []inventory.Item{
			{Expression: "0 12 * * *", Timezone: "Asia/Kolkata", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
			{Expression: "30 6 * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
		}
		report, err := AnalyzeBudget(items, from, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.Equal(t, 2, report.Budgets[0].MaxFound)
		assert.False(t, report.Passed)
	})

	t.Run("rounding to the nearest whole hour does not collide", func(t *testing.T) {
		items := []inventory.Item{
			{Expression: "0 12 * * *", Timezone: "Asia/Kolkata", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
			{Expression: "0 6 * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
			{Expression: "0 7 * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 3}},
		}
		report, err := AnalyzeBudget(items, from, budgets, scheduler, parser)
		require.NoError(t, err)
		assert.Equal(t, 1, report.Budgets[0].MaxFound)
		assert.True(t, report.Passed)
	})
}

// TestAnalyzeBudget_EmptyTimezoneUsesInvocationDefault: no Timezone falls back to from's location.
func TestAnalyzeBudget_EmptyTimezoneUsesInvocationDefault(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()

	paris, err := time.LoadLocation("Europe/Paris")
	require.NoError(t, err)
	from := time.Date(2026, 1, 5, 0, 0, 0, 0, paris)

	items := []inventory.Item{
		{Expression: "0 12 * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
		{Expression: "0 12 * * *", Timezone: "Europe/Paris", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
	}

	budgets := []Budget{
		{MaxConcurrent: 1, TimeWindow: 24 * time.Hour, Name: "strict"},
	}

	report, err := AnalyzeBudget(items, from, budgets, scheduler, parser)
	require.NoError(t, err)
	assert.Equal(t, 2, report.Budgets[0].MaxFound)
	assert.False(t, report.Budgets[0].Passed)
}

func TestAnalyzeSingleBudget(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("finds violations correctly", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
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
		}, "crontab")

		budget := Budget{
			MaxConcurrent: 2,
			TimeWindow:    1 * time.Hour,
			Name:          "test-budget",
		}

		result, err := analyzeSingleBudget(items, from, budget, scheduler, parser)
		require.NoError(t, err)

		// All jobs fire every minute, so MaxFound is 3 — over the limit of 2.
		assert.False(t, result.Passed)
		assert.Greater(t, len(result.Violations), 0)

		violation := result.Violations[0]
		assert.Greater(t, violation.Count, budget.MaxConcurrent)
		assert.Greater(t, len(violation.Jobs), budget.MaxConcurrent)
	})
}

func TestAnalyzeSingleBudget_ViolationJobsOrderIsDeterministic(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Violation.Jobs is built from a map, so run the analysis repeatedly and require the same order every time.
	items := inventory.FromCrontabJobs([]*crontab.Job{
		{LineNumber: 1, Expression: "* * * * *", Command: "/usr/bin/job1.sh", Valid: true},
		{LineNumber: 2, Expression: "* * * * *", Command: "/usr/bin/job2.sh", Valid: true},
		{LineNumber: 3, Expression: "* * * * *", Command: "/usr/bin/job3.sh", Valid: true},
		{LineNumber: 4, Expression: "* * * * *", Command: "/usr/bin/job4.sh", Valid: true},
		{LineNumber: 5, Expression: "* * * * *", Command: "/usr/bin/job5.sh", Valid: true},
	}, "crontab")

	budget := Budget{
		MaxConcurrent: 1,
		// Wide enough to catch a violation given analysis excludes the origin minute itself.
		TimeWindow: 2 * time.Minute,
		Name:       "test-budget",
	}

	var want []string
	for i := 0; i < 20; i++ {
		result, err := analyzeSingleBudget(items, from, budget, scheduler, parser)
		require.NoError(t, err)
		require.NotEmpty(t, result.Violations)

		got := append([]string(nil), result.Violations[0].Jobs...)
		assert.True(t, sort.StringsAreSorted(got), "jobs must be sorted: %v", got)

		if want == nil {
			want = got
			continue
		}
		assert.Equal(t, want, got, "violation job order must be stable across repeated calls")
	}
}

// TestAnalyzeBudget_SameLineDifferentFiles confirms job identity comes from file+line, not line alone.
func TestAnalyzeBudget_SameLineDifferentFiles(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	items := []inventory.Item{
		{Expression: "0 * * * *", State: inventory.StateActive, Locator: inventory.Locator{File: "site-a/crontab", Line: 6}},
		{Expression: "0 * * * *", State: inventory.StateActive, Locator: inventory.Locator{File: "site-b/crontab", Line: 6}},
	}

	budgets := []Budget{{MaxConcurrent: 1, TimeWindow: 2 * time.Hour, Name: "strict"}}

	report, err := AnalyzeBudget(items, from, budgets, scheduler, parser)
	require.NoError(t, err)
	assert.Equal(t, 2, report.Budgets[0].MaxFound, "two distinct items on the same line in different files must both count toward concurrency")
	assert.False(t, report.Passed, "a budget of 1 must be violated by two genuinely concurrent items")
}
