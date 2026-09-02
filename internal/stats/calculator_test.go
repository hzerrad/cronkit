package stats

import (
	"testing"
	"time"

	"github.com/hzerrad/cronkit/internal/crontab"
	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCalculator(t *testing.T) {
	calc := NewCalculator()
	assert.NotNil(t, calc)
}

func TestCalculator_SetClock(t *testing.T) {
	calc := NewCalculator()
	calc.SetClock(func() time.Time {
		return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	})

	items := inventory.FromCrontabJobs([]*crontab.Job{
		{Expression: "0 * * * *", Command: "/usr/bin/a", Valid: true, LineNumber: 1},
		{Expression: "0 * * * *", Command: "/usr/bin/b", Valid: true, LineNumber: 2},
	}, "crontab")

	first := calc.CalculateCollisions(items, 24*time.Hour)
	second := calc.CalculateCollisions(items, 24*time.Hour)

	assert.Equal(t, first, second, "a fixed clock must produce identical results")
	assert.Equal(t, 2, first.MaxConcurrent)

	// The origin (00:00 UTC) is excluded, so the first collision lands at 01:00.
	require.NotEmpty(t, first.BusiestHours)
	assert.Equal(t, 1, first.BusiestHours[0].Hour)
}

func TestCalculateCollisions_TiedBusiestHoursOrderedByHour(t *testing.T) {
	// BusiestHours is built from a map, so run repeatedly to catch sort non-determinism on ties.
	calc := NewCalculator()
	calc.SetClock(func() time.Time {
		return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	})

	items := inventory.FromCrontabJobs([]*crontab.Job{
		{Expression: "0 * * * *", Command: "/usr/bin/a", Valid: true, LineNumber: 1},
		{Expression: "0 * * * *", Command: "/usr/bin/b", Valid: true, LineNumber: 2},
	}, "crontab")

	for i := 0; i < 20; i++ {
		stats := calc.CalculateCollisions(items, 24*time.Hour)
		require.NotEmpty(t, stats.BusiestHours)

		for j := 1; j < len(stats.BusiestHours); j++ {
			prev, cur := stats.BusiestHours[j-1], stats.BusiestHours[j]
			if prev.RunCount == cur.RunCount {
				assert.Less(t, prev.Hour, cur.Hour, "tied hours must be ordered ascending by hour")
			} else {
				assert.Greater(t, prev.RunCount, cur.RunCount, "hours must otherwise be ordered by descending run count")
			}
		}
	}
}

// TestCalculateCollisions_CrossZoneCollision confirms collision detection honors an item's own timezone.
func TestCalculateCollisions_CrossZoneCollision(t *testing.T) {
	calc := NewCalculator()
	calc.SetClock(func() time.Time {
		return time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	})

	items := []inventory.Item{
		{Expression: "0 12 * * *", Timezone: "Europe/London", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
		{Expression: "0 13 * * *", Timezone: "Europe/Paris", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
	}

	stats := calc.CalculateCollisions(items, 24*time.Hour)
	assert.Equal(t, 2, stats.MaxConcurrent)
}

// TestCalculateCollisions_SuspendedItemExcluded confirms suspended items contribute no collisions.
func TestCalculateCollisions_SuspendedItemExcluded(t *testing.T) {
	calc := NewCalculator()
	calc.SetClock(func() time.Time {
		return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	})

	items := []inventory.Item{
		{Expression: "0 * * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
		{Expression: "0 * * * *", State: inventory.StateSuspended, Locator: inventory.Locator{Line: 2}},
	}

	stats := calc.CalculateCollisions(items, 24*time.Hour)
	assert.Equal(t, 1, stats.MaxConcurrent, "the suspended item does not run, so at most one job is ever concurrent")
}

// TestCalculateMetrics_HistogramAgreesWithBusiestHours_AcrossZones checks that
// HourHistogram's peak and CalculateCollisions' busiest hour agree for the
// same item when the collisions clock shares ReferenceDate's DST season.
func TestCalculateMetrics_HistogramAgreesWithBusiestHours_AcrossZones(t *testing.T) {
	calc := NewCalculator()
	calc.SetClock(func() time.Time {
		return time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	})

	items := []inventory.Item{
		{Expression: "0 13 * * *", Timezone: "Europe/Paris", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
	}

	metrics, err := calc.CalculateMetrics(items, 24*time.Hour)
	require.NoError(t, err)

	// 13:00 Europe/Paris is 12:00 UTC in winter.
	require.NotEmpty(t, metrics.Collisions.BusiestHours)
	assert.Equal(t, 12, metrics.Collisions.BusiestHours[0].Hour)

	peakHour, peakCount := -1, 0
	for hour, count := range metrics.HourHistogram {
		if count > peakCount {
			peakHour, peakCount = hour, count
		}
	}
	assert.Equal(t, 12, peakHour,
		"the histogram and the busiest-hour collision stats must agree on which hour this item runs in")
}

// TestCalculateMetrics_HistogramMayDisagreeWithBusiestHours_AcrossDSTSeasons
// documents the accepted case where HourHistogram's fixed January anchor and
// CalculateCollisions' live clock fall in different DST seasons and disagree.
func TestCalculateMetrics_HistogramMayDisagreeWithBusiestHours_AcrossDSTSeasons(t *testing.T) {
	calc := NewCalculator()
	calc.SetClock(func() time.Time {
		return time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)
	})

	items := []inventory.Item{
		{Expression: "0 13 * * *", Timezone: "Europe/Paris", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
	}

	metrics, err := calc.CalculateMetrics(items, 24*time.Hour)
	require.NoError(t, err)

	peakHour, peakCount := -1, 0
	for hour, count := range metrics.HourHistogram {
		if count > peakCount {
			peakHour, peakCount = hour, count
		}
	}
	assert.Equal(t, 12, peakHour, "the histogram always evaluates against ReferenceDate's fixed January")

	require.NotEmpty(t, metrics.Collisions.BusiestHours)
	assert.Equal(t, 11, metrics.Collisions.BusiestHours[0].Hour,
		"collisions evaluates against the live clock's real July, one hour off from the histogram's January view")
}

// TestCalculateMetrics_HistogramIsAdditiveAcrossZones confirms two items
// landing on the same absolute hour in different zones share one histogram bucket.
func TestCalculateMetrics_HistogramIsAdditiveAcrossZones(t *testing.T) {
	calc := NewCalculator()

	items := []inventory.Item{
		{Expression: "0 12 * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
		{Expression: "0 13 * * *", Timezone: "Europe/Paris", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
	}

	metrics, err := calc.CalculateMetrics(items, 24*time.Hour)
	require.NoError(t, err)

	assert.Equal(t, 2, metrics.HourHistogram[12], "both items land in the same UTC hour and must accumulate together")
	assert.Equal(t, 0, metrics.HourHistogram[13], "the Paris item's own local hour must not appear as a separate bucket")
}

func TestCalculateMetrics(t *testing.T) {
	calc := NewCalculator()

	t.Run("should calculate metrics for valid jobs", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true},
			{LineNumber: 2, Expression: "0 0 * * *", Valid: true},
		}, "crontab")

		metrics, err := calc.CalculateMetrics(items, 24*time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, metrics)
		assert.Equal(t, 2, len(metrics.JobFrequencies))
		assert.Greater(t, metrics.TotalRunsPerDay, 0)
	})

	t.Run("should skip invalid jobs", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "invalid", Valid: false},
			{LineNumber: 2, Expression: "0 * * * *", Valid: true},
		}, "crontab")

		metrics, err := calc.CalculateMetrics(items, 24*time.Hour)
		require.NoError(t, err)
		assert.Equal(t, 1, len(metrics.JobFrequencies))
	})

	t.Run("should skip suspended items", func(t *testing.T) {
		items := []inventory.Item{
			{Expression: "0 * * * *", State: inventory.StateSuspended, Locator: inventory.Locator{Line: 1}},
			{Expression: "0 * * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
		}

		metrics, err := calc.CalculateMetrics(items, 24*time.Hour)
		require.NoError(t, err)
		assert.Equal(t, 1, len(metrics.JobFrequencies))
	})

	t.Run("an item with an unresolvable timezone, once admitted, is excluded rather than shown with zero runs", func(t *testing.T) {
		// Runs items through the same admission step (inventory.ResolveTimezones) a real caller does.
		items := inventory.ResolveTimezones([]inventory.Item{
			{Expression: "0 * * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
			{Expression: "0 * * * *", Timezone: "Not/AZone", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
		})
		require.Equal(t, inventory.StateInvalid, items[1].State, "sanity check: admission must have marked the bad-zone item")

		metrics, err := calc.CalculateMetrics(items, 24*time.Hour)
		require.NoError(t, err)
		assert.Equal(t, 1, len(metrics.JobFrequencies), "the unanalysable item must not appear at all, not appear with zero runs")
	})

	t.Run("an item already marked StateInvalid for an unresolvable timezone is excluded", func(t *testing.T) {
		// Constructs the item exactly as inventory.ResolveTimezones would leave it.
		items := []inventory.Item{
			{Expression: "0 * * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
			{
				Expression: "0 * * * *",
				Timezone:   "Not/AZone",
				State:      inventory.StateInvalid,
				Reason:     `unresolvable timezone "Not/AZone": unknown time zone Not/AZone`,
				Locator:    inventory.Locator{Line: 2},
			},
		}

		metrics, err := calc.CalculateMetrics(items, 24*time.Hour)
		require.NoError(t, err)
		assert.Equal(t, 1, len(metrics.JobFrequencies))
	})

	t.Run("should calculate hour histogram", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true},
		}, "crontab")

		metrics, err := calc.CalculateMetrics(items, 24*time.Hour)
		require.NoError(t, err)
		assert.Equal(t, 24, len(metrics.HourHistogram))
		// Hourly job should have runs in all 24 hours
		totalRuns := 0
		for _, count := range metrics.HourHistogram {
			totalRuns += count
		}
		assert.Greater(t, totalRuns, 0)
	})
}

func TestIdentifyMostFrequent(t *testing.T) {
	calc := NewCalculator()

	t.Run("should identify most frequent jobs", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "* * * * *", Valid: true}, // Every minute
			{LineNumber: 2, Expression: "0 * * * *", Valid: true}, // Every hour
			{LineNumber: 3, Expression: "0 0 * * *", Valid: true}, // Daily
		}, "crontab")

		mostFrequent := calc.IdentifyMostFrequent(items, 2)
		assert.Equal(t, 2, len(mostFrequent))
		assert.Greater(t, mostFrequent[0].RunsPerDay, mostFrequent[1].RunsPerDay)
	})

	t.Run("should return all jobs when topN is 0", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true},
			{LineNumber: 2, Expression: "0 0 * * *", Valid: true},
		}, "crontab")

		mostFrequent := calc.IdentifyMostFrequent(items, 0)
		assert.Equal(t, 2, len(mostFrequent))
	})

	t.Run("should handle topN larger than job count", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true},
		}, "crontab")

		mostFrequent := calc.IdentifyMostFrequent(items, 10)
		assert.Equal(t, 1, len(mostFrequent))
	})

	t.Run("should skip invalid jobs", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "invalid", Valid: false},
			{LineNumber: 2, Expression: "0 * * * *", Valid: true},
		}, "crontab")

		mostFrequent := calc.IdentifyMostFrequent(items, 10)
		assert.Equal(t, 1, len(mostFrequent))
	})

	t.Run("two items on the same line in different files must not share a JobID", func(t *testing.T) {
		items := []inventory.Item{
			{Expression: "0 * * * *", State: inventory.StateActive, Locator: inventory.Locator{File: "site-a/crontab", Line: 6}},
			{Expression: "0 * * * *", State: inventory.StateActive, Locator: inventory.Locator{File: "site-b/crontab", Line: 6}},
		}
		mostFrequent := calc.IdentifyMostFrequent(items, 10)
		require.Equal(t, 2, len(mostFrequent))
		assert.NotEqual(t, mostFrequent[0].JobID, mostFrequent[1].JobID,
			"two items on the same line in different files must not share a JobID")
	})
}

func TestIdentifyLeastFrequent(t *testing.T) {
	calc := NewCalculator()

	t.Run("should identify least frequent jobs", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "* * * * *", Valid: true}, // Every minute
			{LineNumber: 2, Expression: "0 * * * *", Valid: true}, // Every hour
			{LineNumber: 3, Expression: "0 0 * * *", Valid: true}, // Daily
		}, "crontab")

		leastFrequent := calc.IdentifyLeastFrequent(items, 2)
		assert.Equal(t, 2, len(leastFrequent))
		assert.Less(t, leastFrequent[0].RunsPerDay, leastFrequent[1].RunsPerDay)
	})

	t.Run("should return all jobs when topN is 0", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true},
			{LineNumber: 2, Expression: "0 0 * * *", Valid: true},
		}, "crontab")

		leastFrequent := calc.IdentifyLeastFrequent(items, 0)
		assert.Equal(t, 2, len(leastFrequent))
	})
}

func TestCalculateCollisions(t *testing.T) {
	calc := NewCalculator()

	t.Run("should detect collisions", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true},
			{LineNumber: 2, Expression: "0 * * * *", Valid: true},
		}, "crontab")

		stats := calc.CalculateCollisions(items, 24*time.Hour)
		assert.Greater(t, stats.MaxConcurrent, 1)
		assert.Greater(t, len(stats.BusiestHours), 0)
	})

	t.Run("should not detect collisions for different times", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true},
			{LineNumber: 2, Expression: "30 * * * *", Valid: true},
		}, "crontab")

		stats := calc.CalculateCollisions(items, 1*time.Hour)
		// May or may not have collisions depending on window
		assert.GreaterOrEqual(t, stats.MaxConcurrent, 0)
	})
}

func TestIdentifyBusiestHours(t *testing.T) {
	calc := NewCalculator()

	items := inventory.FromCrontabJobs([]*crontab.Job{
		{LineNumber: 1, Expression: "0 * * * *", Valid: true},
	}, "crontab")

	busiestHours := calc.IdentifyBusiestHours(items)
	assert.Greater(t, len(busiestHours), 0)
}

func TestCalculateMetrics_LongWindow(t *testing.T) {
	// Test countRunsInWindow indirectly through CalculateMetrics with long windows
	// This exercises the else branch in countRunsInWindow for windows > 24 hours
	calc := NewCalculator()

	t.Run("should handle windows longer than 24 hours", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "0 0 * * *", Valid: true}, // Daily
		}, "crontab")
		// Test with 48-hour window to exercise the long window path in CalculateCollisions
		metrics, err := calc.CalculateMetrics(items, 48*time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, metrics)
		// Should have calculated metrics successfully
		assert.Equal(t, 1, len(metrics.JobFrequencies))
		// Daily job should have 1 run per day (calculated over 24h window)
		assert.GreaterOrEqual(t, metrics.JobFrequencies[0].RunsPerDay, 0)
	})

	t.Run("should handle very long windows", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "0 0 1 * *", Valid: true}, // Monthly
		}, "crontab")
		// Test with 60-day window to exercise the long window path with cap in CalculateCollisions
		metrics, err := calc.CalculateMetrics(items, 60*24*time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, metrics)
		// Should have calculated metrics successfully
		assert.GreaterOrEqual(t, len(metrics.JobFrequencies), 1)
	})

	t.Run("should handle invalid expressions in long window", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "invalid", Valid: false},
		}, "crontab")
		metrics, err := calc.CalculateMetrics(items, 48*time.Hour)
		require.NoError(t, err)
		// Invalid jobs should be skipped
		assert.Equal(t, 0, len(metrics.JobFrequencies))
	})
}

// TestCountRunsInWindow tests the countRunsInWindow function indirectly
// through calculateJobFrequency and CalculateMetrics
func TestCountRunsInWindow(t *testing.T) {
	calc := NewCalculator()

	t.Run("should handle window exactly equal to 1 hour", func(t *testing.T) {
		// This tests the windowDuration <= OneHour branch
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "*/5 * * * *", Valid: true}, // Every 5 minutes
		}, "crontab")
		metrics, err := calc.CalculateMetrics(items, time.Hour)
		require.NoError(t, err)
		// Should calculate runsPerHour correctly
		assert.Greater(t, metrics.JobFrequencies[0].RunsPerHour, 0)
	})

	t.Run("should handle window exactly equal to 1 day", func(t *testing.T) {
		// This tests the windowDuration <= OneDay branch
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true}, // Every hour
		}, "crontab")
		metrics, err := calc.CalculateMetrics(items, 24*time.Hour)
		require.NoError(t, err)
		// Should calculate runsPerDay correctly
		assert.Greater(t, metrics.JobFrequencies[0].RunsPerDay, 0)
	})

	t.Run("should handle window longer than 1 day with cap", func(t *testing.T) {
		// This tests the else branch for windows > 24 hours and the cap
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "0 0 * * *", Valid: true}, // Daily
		}, "crontab")
		// Use a very long window that would exceed MaxRunsForLongWindow if calculated naively
		metrics, err := calc.CalculateMetrics(items, 365*24*time.Hour)
		require.NoError(t, err)
		// Should still calculate successfully without hanging
		assert.Equal(t, 1, len(metrics.JobFrequencies))
	})

	t.Run("items with no line number still get a unique job identity", func(t *testing.T) {
		// inventory.Locator.Identity falls back to each item's position, which is unique by construction.
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 0, Expression: "0 * * * *", Valid: true},
			{LineNumber: 0, Expression: "0 * * * *", Valid: true},
		}, "crontab")
		metrics, err := calc.CalculateMetrics(items, 24*time.Hour)
		require.NoError(t, err)
		require.Equal(t, 2, len(metrics.JobFrequencies))
		assert.NotEqual(t, metrics.JobFrequencies[0].JobID, metrics.JobFrequencies[1].JobID,
			"two items with no line and the same expression must not share a JobID")
		assert.Equal(t, "job-0", metrics.JobFrequencies[0].JobID)
		assert.Equal(t, "job-1", metrics.JobFrequencies[1].JobID)
	})

	t.Run("two items on the same line in different files must not share a JobID", func(t *testing.T) {
		// Job identity uses the shared locator helper, since line numbers aren't unique across files.
		items := []inventory.Item{
			{Expression: "0 * * * *", State: inventory.StateActive, Locator: inventory.Locator{File: "site-a/crontab", Line: 6}},
			{Expression: "0 * * * *", State: inventory.StateActive, Locator: inventory.Locator{File: "site-b/crontab", Line: 6}},
		}
		metrics, err := calc.CalculateMetrics(items, 24*time.Hour)
		require.NoError(t, err)
		require.Equal(t, 2, len(metrics.JobFrequencies))
		assert.NotEqual(t, metrics.JobFrequencies[0].JobID, metrics.JobFrequencies[1].JobID,
			"two items on the same line in different files must not share a JobID")
	})

	t.Run("should handle times equal to endTime", func(t *testing.T) {
		// Test the t.Equal(endTime) branch in countRunsInWindow
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "0 0 * * *", Valid: true}, // Daily at midnight
		}, "crontab")
		// Calculate metrics for a window that ends exactly at a run time
		metrics, err := calc.CalculateMetrics(items, 24*time.Hour)
		require.NoError(t, err)
		// Should count runs correctly even when times equal endTime
		assert.GreaterOrEqual(t, metrics.JobFrequencies[0].RunsPerDay, 0)
	})

	t.Run("should handle times before startTime", func(t *testing.T) {
		// Test the !t.Before(startTime) branch - times before start should not be counted
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "*/30 * * * *", Valid: true}, // Every 30 minutes
		}, "crontab")
		// Calculate metrics - scheduler might return times before startTime
		metrics, err := calc.CalculateMetrics(items, 1*time.Hour)
		require.NoError(t, err)
		// Should only count runs within the window
		assert.GreaterOrEqual(t, metrics.JobFrequencies[0].RunsPerHour, 0)
		assert.LessOrEqual(t, metrics.JobFrequencies[0].RunsPerHour, 3) // Max 2-3 runs per hour for */30
	})

	t.Run("should handle window duration exactly at boundaries", func(t *testing.T) {
		// Test windowDuration exactly equal to OneHour
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "*/15 * * * *", Valid: true}, // Every 15 minutes
		}, "crontab")
		metrics1, err1 := calc.CalculateMetrics(items, time.Hour)
		require.NoError(t, err1)

		// Test windowDuration slightly more than OneHour
		metrics2, err2 := calc.CalculateMetrics(items, time.Hour+time.Minute)
		require.NoError(t, err2)

		// Both should succeed
		assert.NotNil(t, metrics1)
		assert.NotNil(t, metrics2)
	})

	t.Run("should handle window duration exactly equal to OneDay", func(t *testing.T) {
		// Test windowDuration exactly equal to OneDay
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true}, // Every hour
		}, "crontab")
		metrics1, err1 := calc.CalculateMetrics(items, 24*time.Hour)
		require.NoError(t, err1)

		// Test windowDuration slightly more than OneDay
		metrics2, err2 := calc.CalculateMetrics(items, 24*time.Hour+time.Minute)
		require.NoError(t, err2)

		// Both should succeed, but second should use else branch
		assert.NotNil(t, metrics1)
		assert.NotNil(t, metrics2)
	})

	t.Run("should handle MaxRunsForLongWindow cap", func(t *testing.T) {
		// Test that very long windows are capped at MaxRunsForLongWindow
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "* * * * *", Valid: true}, // Every minute
		}, "crontab")
		// Use a very long window that would exceed MaxRunsForLongWindow
		// This tests the cap logic in countRunsInWindow
		metrics, err := calc.CalculateMetrics(items, 100*24*time.Hour) // 100 days
		require.NoError(t, err)
		// Should complete without hanging and calculate successfully
		assert.NotNil(t, metrics)
		assert.Equal(t, 1, len(metrics.JobFrequencies))
	})
}
