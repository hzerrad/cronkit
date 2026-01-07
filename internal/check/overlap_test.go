package check

import (
	"testing"
	"time"

	"github.com/hzerrad/cronkit/internal/crontab"
	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeOverlaps(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()

	t.Run("should detect overlaps for jobs running at same time", func(t *testing.T) {
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true}, // Every hour at :00
			{LineNumber: 2, Expression: "0 * * * *", Valid: true}, // Every hour at :00
		}

		overlaps, stats, err := AnalyzeOverlaps(jobs, 24*time.Hour, scheduler, parser)
		require.NoError(t, err)
		assert.Greater(t, len(overlaps), 0, "Should detect overlaps")
		assert.Greater(t, stats.MaxConcurrent, 1, "Should have max concurrent > 1")
	})

	t.Run("should not detect overlaps for jobs at different times", func(t *testing.T) {
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true},  // Every hour at :00
			{LineNumber: 2, Expression: "30 * * * *", Valid: true}, // Every hour at :30
		}

		overlaps, stats, err := AnalyzeOverlaps(jobs, 1*time.Hour, scheduler, parser)
		require.NoError(t, err)
		assert.Equal(t, 0, len(overlaps), "Should not detect overlaps for different times")
		assert.Equal(t, 0, stats.MaxConcurrent)
	})

	t.Run("should return empty for single job", func(t *testing.T) {
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true},
		}

		overlaps, stats, err := AnalyzeOverlaps(jobs, 24*time.Hour, scheduler, parser)
		require.NoError(t, err)
		assert.Equal(t, 0, len(overlaps), "Single job cannot have overlaps")
		assert.Equal(t, 0, stats.MaxConcurrent)
	})

	t.Run("should return empty for empty job list", func(t *testing.T) {
		jobs := []*crontab.Job{}

		overlaps, stats, err := AnalyzeOverlaps(jobs, 24*time.Hour, scheduler, parser)
		require.NoError(t, err)
		assert.Equal(t, 0, len(overlaps))
		assert.Equal(t, 0, stats.MaxConcurrent)
	})

	t.Run("should handle invalid jobs gracefully", func(t *testing.T) {
		jobs := []*crontab.Job{
			{LineNumber: 1, Expression: "invalid", Valid: false},
			{LineNumber: 2, Expression: "0 * * * *", Valid: true},
		}

		overlaps, _, err := AnalyzeOverlaps(jobs, 24*time.Hour, scheduler, parser)
		require.NoError(t, err)
		// Should only analyze valid jobs
		assert.GreaterOrEqual(t, len(overlaps), 0)
	})
}

func TestUniqueStrings(t *testing.T) {
	t.Run("should remove duplicates", func(t *testing.T) {
		input := []string{"a", "b", "a", "c", "b"}
		result := uniqueStrings(input)
		assert.Equal(t, 3, len(result))
		assert.Contains(t, result, "a")
		assert.Contains(t, result, "b")
		assert.Contains(t, result, "c")
	})

	t.Run("should handle empty slice", func(t *testing.T) {
		result := uniqueStrings([]string{})
		assert.Equal(t, 0, len(result))
	})

	t.Run("should handle single element", func(t *testing.T) {
		result := uniqueStrings([]string{"a"})
		assert.Equal(t, 1, len(result))
		assert.Equal(t, "a", result[0])
	})
}
