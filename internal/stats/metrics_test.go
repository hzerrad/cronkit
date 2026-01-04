package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {
	t.Run("should create metrics with all fields", func(t *testing.T) {
		metrics := &Metrics{
			TotalRunsPerDay:  100,
			TotalRunsPerHour: 5,
			JobFrequencies: []JobFrequency{
				{
					JobID:       "job1",
					Expression:  "0 2 * * *",
					RunsPerDay:  1,
					RunsPerHour: 0,
				},
			},
			HourHistogram: make([]int, 24),
			Collisions: CollisionStats{
				BusiestHours:       []HourStats{},
				QuietWindows:       []TimeWindow{},
				CollisionFrequency: 0.0,
				MaxConcurrent:      0,
			},
		}

		assert.Equal(t, 100, metrics.TotalRunsPerDay)
		assert.Equal(t, 5, metrics.TotalRunsPerHour)
		assert.Len(t, metrics.JobFrequencies, 1)
		assert.Len(t, metrics.HourHistogram, 24)
		assert.Equal(t, 0.0, metrics.Collisions.CollisionFrequency)
	})

	t.Run("should create empty metrics", func(t *testing.T) {
		metrics := &Metrics{}

		assert.Equal(t, 0, metrics.TotalRunsPerDay)
		assert.Equal(t, 0, metrics.TotalRunsPerHour)
		assert.Nil(t, metrics.JobFrequencies)
		assert.Nil(t, metrics.HourHistogram)
	})
}

func TestJobFrequency(t *testing.T) {
	t.Run("should create job frequency with all fields", func(t *testing.T) {
		freq := JobFrequency{
			JobID:       "line-10",
			Expression:  "*/15 * * * *",
			RunsPerDay:  95,
			RunsPerHour: 3,
		}

		assert.Equal(t, "line-10", freq.JobID)
		assert.Equal(t, "*/15 * * * *", freq.Expression)
		assert.Equal(t, 95, freq.RunsPerDay)
		assert.Equal(t, 3, freq.RunsPerHour)
	})

	t.Run("should create job frequency with zero runs", func(t *testing.T) {
		freq := JobFrequency{
			JobID:       "line-5",
			Expression:  "0 9 * * 1",
			RunsPerDay:  0,
			RunsPerHour: 0,
		}

		assert.Equal(t, 0, freq.RunsPerDay)
		assert.Equal(t, 0, freq.RunsPerHour)
	})
}

func TestCollisionStats(t *testing.T) {
	t.Run("should create collision stats with all fields", func(t *testing.T) {
		stats := CollisionStats{
			BusiestHours: []HourStats{
				{Hour: 9, RunCount: 10, JobCount: 5},
				{Hour: 10, RunCount: 8, JobCount: 4},
			},
			QuietWindows: []TimeWindow{
				{
					Start:    time.Date(2025, 1, 1, 2, 0, 0, 0, time.UTC),
					End:      time.Date(2025, 1, 1, 3, 0, 0, 0, time.UTC),
					RunCount: 0,
					JobCount: 0,
				},
			},
			CollisionFrequency: 25.5,
			MaxConcurrent:      5,
		}

		assert.Len(t, stats.BusiestHours, 2)
		assert.Len(t, stats.QuietWindows, 1)
		assert.Equal(t, 25.5, stats.CollisionFrequency)
		assert.Equal(t, 5, stats.MaxConcurrent)
	})

	t.Run("should create empty collision stats", func(t *testing.T) {
		stats := CollisionStats{}

		assert.Nil(t, stats.BusiestHours)
		assert.Nil(t, stats.QuietWindows)
		assert.Equal(t, 0.0, stats.CollisionFrequency)
		assert.Equal(t, 0, stats.MaxConcurrent)
	})
}

func TestHourStats(t *testing.T) {
	t.Run("should create hour stats with all fields", func(t *testing.T) {
		hourStat := HourStats{
			Hour:     9,
			RunCount: 10,
			JobCount: 5,
		}

		assert.Equal(t, 9, hourStat.Hour)
		assert.Equal(t, 10, hourStat.RunCount)
		assert.Equal(t, 5, hourStat.JobCount)
	})

	t.Run("should create hour stats with zero values", func(t *testing.T) {
		hourStat := HourStats{}

		assert.Equal(t, 0, hourStat.Hour)
		assert.Equal(t, 0, hourStat.RunCount)
		assert.Equal(t, 0, hourStat.JobCount)
	})
}

func TestTimeWindow(t *testing.T) {
	t.Run("should create time window with all fields", func(t *testing.T) {
		start := time.Date(2025, 1, 1, 9, 0, 0, 0, time.UTC)
		end := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

		window := TimeWindow{
			Start:    start,
			End:      end,
			RunCount: 5,
			JobCount: 3,
		}

		assert.Equal(t, start, window.Start)
		assert.Equal(t, end, window.End)
		assert.Equal(t, 5, window.RunCount)
		assert.Equal(t, 3, window.JobCount)
	})

	t.Run("should create time window with zero values", func(t *testing.T) {
		window := TimeWindow{}

		assert.True(t, window.Start.IsZero())
		assert.True(t, window.End.IsZero())
		assert.Equal(t, 0, window.RunCount)
		assert.Equal(t, 0, window.JobCount)
	})
}
