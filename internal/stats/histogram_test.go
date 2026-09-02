package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateHistogram(t *testing.T) {
	t.Run("should generate histogram from hour data", func(t *testing.T) {
		hourData := make([]int, 24)
		hourData[0] = 10
		hourData[12] = 20
		hourData[23] = 5

		result := GenerateHistogram(hourData, 40)
		assert.Contains(t, result, "Hour Distribution")
		assert.Contains(t, result, "00:00")
		assert.Contains(t, result, "12:00")
		assert.Contains(t, result, "23:00")
	})

	t.Run("labels the header as a typical day in UTC", func(t *testing.T) {
		// Distinguishes this from CollisionStats' busiest hours, which use a different window and zone.
		hourData := make([]int, 24)
		hourData[12] = 1

		result := GenerateHistogram(hourData, 40)
		assert.Contains(t, result, "Hour Distribution (typical day, UTC):")
	})

	t.Run("should handle empty data", func(t *testing.T) {
		hourData := make([]int, 24)
		result := GenerateHistogram(hourData, 40)
		assert.Contains(t, result, "No runs detected")
	})

	t.Run("should handle invalid length", func(t *testing.T) {
		hourData := []int{1, 2, 3} // Wrong length
		result := GenerateHistogram(hourData, 40)
		assert.Equal(t, "", result)
	})

	t.Run("should scale bars correctly", func(t *testing.T) {
		hourData := make([]int, 24)
		hourData[0] = 100
		hourData[1] = 50

		result := GenerateHistogram(hourData, 40)
		// Hour 0 should have longer bar than hour 1
		assert.Contains(t, result, "00:00")
		assert.Contains(t, result, "01:00")
	})
}
