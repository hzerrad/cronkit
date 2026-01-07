package check

import (
	"testing"

	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateRunsPerDay(t *testing.T) {
	scheduler := cronx.NewScheduler()

	t.Run("should calculate runs for hourly job", func(t *testing.T) {
		runs, err := CalculateRunsPerDay("0 * * * *", scheduler)
		require.NoError(t, err)
		assert.Equal(t, 24, runs, "Hourly job should run 24 times per day")
	})

	t.Run("should calculate runs for daily job", func(t *testing.T) {
		runs, err := CalculateRunsPerDay("0 0 * * *", scheduler)
		require.NoError(t, err)
		assert.Equal(t, 1, runs, "Daily job should run 1 time per day")
	})

	t.Run("should calculate runs for every 15 minutes", func(t *testing.T) {
		runs, err := CalculateRunsPerDay("*/15 * * * *", scheduler)
		require.NoError(t, err)
		assert.Equal(t, 96, runs, "Every 15 minutes should run 96 times per day (4 per hour * 24)")
	})

	t.Run("should calculate runs for every minute", func(t *testing.T) {
		runs, err := CalculateRunsPerDay("* * * * *", scheduler)
		require.NoError(t, err)
		assert.Equal(t, 1440, runs, "Every minute should run 1440 times per day")
	})

	t.Run("should return error for invalid expression", func(t *testing.T) {
		_, err := CalculateRunsPerDay("invalid", scheduler)
		require.Error(t, err)
	})
}

func TestDetectRedundantPattern(t *testing.T) {
	parser := cronx.NewParser()

	t.Run("should detect redundant */1 pattern", func(t *testing.T) {
		schedule, err := parser.Parse("*/1 * * * *")
		require.NoError(t, err)
		assert.True(t, DetectRedundantPattern(schedule), "*/1 should be detected as redundant")
	})

	t.Run("should not detect non-redundant step pattern", func(t *testing.T) {
		schedule, err := parser.Parse("*/15 * * * *")
		require.NoError(t, err)
		assert.False(t, DetectRedundantPattern(schedule), "*/15 should not be redundant")
	})

	t.Run("should not detect wildcard as redundant", func(t *testing.T) {
		schedule, err := parser.Parse("* * * * *")
		require.NoError(t, err)
		assert.False(t, DetectRedundantPattern(schedule), "* should not be redundant")
	})

	t.Run("should detect redundant pattern in hour field", func(t *testing.T) {
		schedule, err := parser.Parse("0 */1 * * *")
		require.NoError(t, err)
		assert.True(t, DetectRedundantPattern(schedule), "*/1 in hour field should be redundant")
	})
}

func TestEstimateRunFrequency(t *testing.T) {
	scheduler := cronx.NewScheduler()

	t.Run("should estimate frequency for hourly job", func(t *testing.T) {
		runsPerDay, runsPerHour, err := EstimateRunFrequency("0 * * * *", scheduler)
		require.NoError(t, err)
		assert.Equal(t, 24, runsPerDay)
		assert.Equal(t, 1, runsPerHour)
	})

	t.Run("should estimate frequency for every 15 minutes", func(t *testing.T) {
		runsPerDay, runsPerHour, err := EstimateRunFrequency("*/15 * * * *", scheduler)
		require.NoError(t, err)
		assert.Equal(t, 96, runsPerDay)
		assert.Equal(t, 4, runsPerHour)
	})

	t.Run("should return error for invalid expression", func(t *testing.T) {
		_, _, err := EstimateRunFrequency("invalid", scheduler)
		require.Error(t, err)
	})
}

func TestGetRedundantPatternSuggestion(t *testing.T) {
	parser := cronx.NewParser()

	t.Run("should suggest simplification for */1 pattern", func(t *testing.T) {
		schedule, err := parser.Parse("*/1 * * * *")
		require.NoError(t, err)
		suggestion := GetRedundantPatternSuggestion("*/1 * * * *", schedule)
		assert.Equal(t, "* * * * *", suggestion)
	})

	t.Run("should suggest simplification for multiple */1 patterns", func(t *testing.T) {
		schedule, err := parser.Parse("*/1 */1 * * *")
		require.NoError(t, err)
		suggestion := GetRedundantPatternSuggestion("*/1 */1 * * *", schedule)
		assert.Equal(t, "* * * * *", suggestion)
	})

	t.Run("should not change non-redundant patterns", func(t *testing.T) {
		schedule, err := parser.Parse("*/15 * * * *")
		require.NoError(t, err)
		suggestion := GetRedundantPatternSuggestion("*/15 * * * *", schedule)
		assert.Equal(t, "*/15 * * * *", suggestion)
	})

	t.Run("should handle non-standard format", func(t *testing.T) {
		schedule, err := parser.Parse("0 * * * *")
		require.NoError(t, err)
		suggestion := GetRedundantPatternSuggestion("invalid format", schedule)
		assert.Equal(t, "invalid format", suggestion)
	})
}
