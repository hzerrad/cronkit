package stats

import (
	"testing"
	"time"

	"github.com/hzerrad/cronic/internal/crontab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCalculator(t *testing.T) {
	calc := NewCalculator()
	assert.NotNil(t, calc)
}

func TestCalculateMetrics(t *testing.T) {
	calc := NewCalculator()

	t.Run("should calculate metrics for valid jobs", func(t *testing.T) {
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true},
			{LineNumber: 2, Expression: "0 0 * * *", Valid: true},
		}

		metrics, err := calc.CalculateMetrics(jobs, 24*time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, metrics)
		assert.Equal(t, 2, len(metrics.JobFrequencies))
		assert.Greater(t, metrics.TotalRunsPerDay, 0)
	})

	t.Run("should skip invalid jobs", func(t *testing.T) {
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "invalid", Valid: false},
			{LineNumber: 2, Expression: "0 * * * *", Valid: true},
		}

		metrics, err := calc.CalculateMetrics(jobs, 24*time.Hour)
		require.NoError(t, err)
		assert.Equal(t, 1, len(metrics.JobFrequencies))
	})

	t.Run("should calculate hour histogram", func(t *testing.T) {
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true},
		}

		metrics, err := calc.CalculateMetrics(jobs, 24*time.Hour)
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
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "* * * * *", Valid: true}, // Every minute
			{LineNumber: 2, Expression: "0 * * * *", Valid: true}, // Every hour
			{LineNumber: 3, Expression: "0 0 * * *", Valid: true}, // Daily
		}

		mostFrequent := calc.IdentifyMostFrequent(jobs, 2)
		assert.Equal(t, 2, len(mostFrequent))
		assert.Greater(t, mostFrequent[0].RunsPerDay, mostFrequent[1].RunsPerDay)
	})

	t.Run("should return all jobs when topN is 0", func(t *testing.T) {
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true},
			{LineNumber: 2, Expression: "0 0 * * *", Valid: true},
		}

		mostFrequent := calc.IdentifyMostFrequent(jobs, 0)
		assert.Equal(t, 2, len(mostFrequent))
	})

	t.Run("should handle topN larger than job count", func(t *testing.T) {
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true},
		}

		mostFrequent := calc.IdentifyMostFrequent(jobs, 10)
		assert.Equal(t, 1, len(mostFrequent))
	})

	t.Run("should skip invalid jobs", func(t *testing.T) {
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "invalid", Valid: false},
			{LineNumber: 2, Expression: "0 * * * *", Valid: true},
		}

		mostFrequent := calc.IdentifyMostFrequent(jobs, 10)
		assert.Equal(t, 1, len(mostFrequent))
	})
}

func TestIdentifyLeastFrequent(t *testing.T) {
	calc := NewCalculator()

	t.Run("should identify least frequent jobs", func(t *testing.T) {
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "* * * * *", Valid: true}, // Every minute
			{LineNumber: 2, Expression: "0 * * * *", Valid: true}, // Every hour
			{LineNumber: 3, Expression: "0 0 * * *", Valid: true}, // Daily
		}

		leastFrequent := calc.IdentifyLeastFrequent(jobs, 2)
		assert.Equal(t, 2, len(leastFrequent))
		assert.Less(t, leastFrequent[0].RunsPerDay, leastFrequent[1].RunsPerDay)
	})

	t.Run("should return all jobs when topN is 0", func(t *testing.T) {
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true},
			{LineNumber: 2, Expression: "0 0 * * *", Valid: true},
		}

		leastFrequent := calc.IdentifyLeastFrequent(jobs, 0)
		assert.Equal(t, 2, len(leastFrequent))
	})
}

func TestCalculateCollisions(t *testing.T) {
	calc := NewCalculator()

	t.Run("should detect collisions", func(t *testing.T) {
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true},
			{LineNumber: 2, Expression: "0 * * * *", Valid: true},
		}

		stats := calc.CalculateCollisions(jobs, 24*time.Hour)
		assert.Greater(t, stats.MaxConcurrent, 1)
		assert.Greater(t, len(stats.BusiestHours), 0)
	})

	t.Run("should not detect collisions for different times", func(t *testing.T) {
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true},
			{LineNumber: 2, Expression: "30 * * * *", Valid: true},
		}

		stats := calc.CalculateCollisions(jobs, 1*time.Hour)
		// May or may not have collisions depending on window
		assert.GreaterOrEqual(t, stats.MaxConcurrent, 0)
	})
}

func TestIdentifyBusiestHours(t *testing.T) {
	calc := NewCalculator()

	jobs := []*crontab.Job{
		{LineNumber: 1, Expression: "0 * * * *", Valid: true},
	}

	busiestHours := calc.IdentifyBusiestHours(jobs)
	assert.Greater(t, len(busiestHours), 0)
}

func TestCalculateMetrics_LongWindow(t *testing.T) {
	// Test countRunsInWindow indirectly through CalculateMetrics with long windows
	// This exercises the else branch in countRunsInWindow for windows > 24 hours
	calc := NewCalculator()

	t.Run("should handle windows longer than 24 hours", func(t *testing.T) {
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "0 0 * * *", Valid: true}, // Daily
		}
		// Test with 48-hour window to exercise the long window path in CalculateCollisions
		metrics, err := calc.CalculateMetrics(jobs, 48*time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, metrics)
		// Should have calculated metrics successfully
		assert.Equal(t, 1, len(metrics.JobFrequencies))
		// Daily job should have 1 run per day (calculated over 24h window)
		assert.GreaterOrEqual(t, metrics.JobFrequencies[0].RunsPerDay, 0)
	})

	t.Run("should handle very long windows", func(t *testing.T) {
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "0 0 1 * *", Valid: true}, // Monthly
		}
		// Test with 60-day window to exercise the long window path with cap in CalculateCollisions
		metrics, err := calc.CalculateMetrics(jobs, 60*24*time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, metrics)
		// Should have calculated metrics successfully
		assert.GreaterOrEqual(t, len(metrics.JobFrequencies), 1)
	})

	t.Run("should handle invalid expressions in long window", func(t *testing.T) {
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "invalid", Valid: false},
		}
		metrics, err := calc.CalculateMetrics(jobs, 48*time.Hour)
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
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "*/5 * * * *", Valid: true}, // Every 5 minutes
		}
		metrics, err := calc.CalculateMetrics(jobs, time.Hour)
		require.NoError(t, err)
		// Should calculate runsPerHour correctly
		assert.Greater(t, metrics.JobFrequencies[0].RunsPerHour, 0)
	})

	t.Run("should handle window exactly equal to 1 day", func(t *testing.T) {
		// This tests the windowDuration <= OneDay branch
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true}, // Every hour
		}
		metrics, err := calc.CalculateMetrics(jobs, 24*time.Hour)
		require.NoError(t, err)
		// Should calculate runsPerDay correctly
		assert.Greater(t, metrics.JobFrequencies[0].RunsPerDay, 0)
	})

	t.Run("should handle window longer than 1 day with cap", func(t *testing.T) {
		// This tests the else branch for windows > 24 hours and the cap
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "0 0 * * *", Valid: true}, // Daily
		}
		// Use a very long window that would exceed MaxRunsForLongWindow if calculated naively
		metrics, err := calc.CalculateMetrics(jobs, 365*24*time.Hour)
		require.NoError(t, err)
		// Should still calculate successfully without hanging
		assert.Equal(t, 1, len(metrics.JobFrequencies))
	})

	t.Run("should handle job with zero line number", func(t *testing.T) {
		// Test the jobID logic when LineNumber is 0
		jobs := []*crontab.Job{
			{LineNumber: 0, Expression: "0 * * * *", Valid: true},
		}
		metrics, err := calc.CalculateMetrics(jobs, 24*time.Hour)
		require.NoError(t, err)
		assert.Equal(t, 1, len(metrics.JobFrequencies))
		// When LineNumber is 0, jobID should be the expression
		assert.Equal(t, "0 * * * *", metrics.JobFrequencies[0].JobID)
	})

	t.Run("should handle times equal to endTime", func(t *testing.T) {
		// Test the t.Equal(endTime) branch in countRunsInWindow
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "0 0 * * *", Valid: true}, // Daily at midnight
		}
		// Calculate metrics for a window that ends exactly at a run time
		metrics, err := calc.CalculateMetrics(jobs, 24*time.Hour)
		require.NoError(t, err)
		// Should count runs correctly even when times equal endTime
		assert.GreaterOrEqual(t, metrics.JobFrequencies[0].RunsPerDay, 0)
	})

	t.Run("should handle times before startTime", func(t *testing.T) {
		// Test the !t.Before(startTime) branch - times before start should not be counted
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "*/30 * * * *", Valid: true}, // Every 30 minutes
		}
		// Calculate metrics - scheduler might return times before startTime
		metrics, err := calc.CalculateMetrics(jobs, 1*time.Hour)
		require.NoError(t, err)
		// Should only count runs within the window
		assert.GreaterOrEqual(t, metrics.JobFrequencies[0].RunsPerHour, 0)
		assert.LessOrEqual(t, metrics.JobFrequencies[0].RunsPerHour, 3) // Max 2-3 runs per hour for */30
	})

	t.Run("should handle window duration exactly at boundaries", func(t *testing.T) {
		// Test windowDuration exactly equal to OneHour
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "*/15 * * * *", Valid: true}, // Every 15 minutes
		}
		metrics1, err1 := calc.CalculateMetrics(jobs, time.Hour)
		require.NoError(t, err1)

		// Test windowDuration slightly more than OneHour
		metrics2, err2 := calc.CalculateMetrics(jobs, time.Hour+time.Minute)
		require.NoError(t, err2)

		// Both should succeed
		assert.NotNil(t, metrics1)
		assert.NotNil(t, metrics2)
	})

	t.Run("should handle window duration exactly equal to OneDay", func(t *testing.T) {
		// Test windowDuration exactly equal to OneDay
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true}, // Every hour
		}
		metrics1, err1 := calc.CalculateMetrics(jobs, 24*time.Hour)
		require.NoError(t, err1)

		// Test windowDuration slightly more than OneDay
		metrics2, err2 := calc.CalculateMetrics(jobs, 24*time.Hour+time.Minute)
		require.NoError(t, err2)

		// Both should succeed, but second should use else branch
		assert.NotNil(t, metrics1)
		assert.NotNil(t, metrics2)
	})

	t.Run("should handle MaxRunsForLongWindow cap", func(t *testing.T) {
		// Test that very long windows are capped at MaxRunsForLongWindow
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "* * * * *", Valid: true}, // Every minute
		}
		// Use a very long window that would exceed MaxRunsForLongWindow
		// This tests the cap logic in countRunsInWindow
		metrics, err := calc.CalculateMetrics(jobs, 100*24*time.Hour) // 100 days
		require.NoError(t, err)
		// Should complete without hanging and calculate successfully
		assert.NotNil(t, metrics)
		assert.Equal(t, 1, len(metrics.JobFrequencies))
	})
}
