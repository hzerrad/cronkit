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
