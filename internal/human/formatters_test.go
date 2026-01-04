package human

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatHour(t *testing.T) {
	t.Run("should format hour correctly", func(t *testing.T) {
		assert.Equal(t, "00:00", formatHour(0))
		assert.Equal(t, "12:00", formatHour(12))
		assert.Equal(t, "23:00", formatHour(23))
	})

	t.Run("should format single digit hours with leading zero", func(t *testing.T) {
		assert.Equal(t, "01:00", formatHour(1))
		assert.Equal(t, "09:00", formatHour(9))
	})
}

func TestFormatHourEnd(t *testing.T) {
	t.Run("should format hour end correctly", func(t *testing.T) {
		assert.Equal(t, "00:59", formatHourEnd(0))
		assert.Equal(t, "12:59", formatHourEnd(12))
		assert.Equal(t, "23:59", formatHourEnd(23))
	})

	t.Run("should format single digit hours with leading zero", func(t *testing.T) {
		assert.Equal(t, "01:59", formatHourEnd(1))
		assert.Equal(t, "09:59", formatHourEnd(9))
	})
}

func TestFormatTime(t *testing.T) {
	t.Run("should format time correctly", func(t *testing.T) {
		assert.Equal(t, "00:00", formatTime(0, 0))
		assert.Equal(t, "12:30", formatTime(12, 30))
		assert.Equal(t, "23:59", formatTime(23, 59))
	})

	t.Run("should format single digit values with leading zeros", func(t *testing.T) {
		assert.Equal(t, "01:05", formatTime(1, 5))
		assert.Equal(t, "09:09", formatTime(9, 9))
	})
}

func TestFormatList(t *testing.T) {
	t.Run("should format empty list", func(t *testing.T) {
		result := formatList([]string{})
		assert.Empty(t, result)
	})

	t.Run("should format single item", func(t *testing.T) {
		result := formatList([]string{"apple"})
		assert.Equal(t, "apple", result)
	})

	t.Run("should format two items", func(t *testing.T) {
		result := formatList([]string{"apple", "banana"})
		assert.Equal(t, "apple and banana", result)
	})

	t.Run("should format three items with Oxford comma", func(t *testing.T) {
		result := formatList([]string{"apple", "banana", "cherry"})
		assert.Equal(t, "apple, banana, and cherry", result)
	})

	t.Run("should format multiple items with Oxford comma", func(t *testing.T) {
		result := formatList([]string{"apple", "banana", "cherry", "date"})
		assert.Equal(t, "apple, banana, cherry, and date", result)
	})

	t.Run("should format many items", func(t *testing.T) {
		result := formatList([]string{"one", "two", "three", "four", "five"})
		assert.Equal(t, "one, two, three, four, and five", result)
	})
}

func TestDayName(t *testing.T) {
	t.Run("should return correct day names", func(t *testing.T) {
		assert.Equal(t, "Sunday", dayName(0))
		assert.Equal(t, "Monday", dayName(1))
		assert.Equal(t, "Tuesday", dayName(2))
		assert.Equal(t, "Wednesday", dayName(3))
		assert.Equal(t, "Thursday", dayName(4))
		assert.Equal(t, "Friday", dayName(5))
		assert.Equal(t, "Saturday", dayName(6))
	})

	t.Run("should handle out of range values", func(t *testing.T) {
		assert.Equal(t, "day-1", dayName(-1))
		assert.Equal(t, "day7", dayName(7))
		assert.Equal(t, "day100", dayName(100))
	})
}

func TestFormatMonth(t *testing.T) {
	t.Run("should return correct month names", func(t *testing.T) {
		assert.Equal(t, "January", formatMonth(1))
		assert.Equal(t, "February", formatMonth(2))
		assert.Equal(t, "March", formatMonth(3))
		assert.Equal(t, "April", formatMonth(4))
		assert.Equal(t, "May", formatMonth(5))
		assert.Equal(t, "June", formatMonth(6))
		assert.Equal(t, "July", formatMonth(7))
		assert.Equal(t, "August", formatMonth(8))
		assert.Equal(t, "September", formatMonth(9))
		assert.Equal(t, "October", formatMonth(10))
		assert.Equal(t, "November", formatMonth(11))
		assert.Equal(t, "December", formatMonth(12))
	})

	t.Run("should handle out of range values", func(t *testing.T) {
		assert.Equal(t, "month0", formatMonth(0))
		assert.Equal(t, "month13", formatMonth(13))
		assert.Equal(t, "month100", formatMonth(100))
	})
}

func TestOrdinalSuffix(t *testing.T) {
	t.Run("should return correct suffix for 1st", func(t *testing.T) {
		assert.Equal(t, "st", ordinalSuffix(1))
		assert.Equal(t, "st", ordinalSuffix(21))
		assert.Equal(t, "st", ordinalSuffix(31))
	})

	t.Run("should return correct suffix for 2nd", func(t *testing.T) {
		assert.Equal(t, "nd", ordinalSuffix(2))
		assert.Equal(t, "nd", ordinalSuffix(22))
		assert.Equal(t, "nd", ordinalSuffix(32))
	})

	t.Run("should return correct suffix for 3rd", func(t *testing.T) {
		assert.Equal(t, "rd", ordinalSuffix(3))
		assert.Equal(t, "rd", ordinalSuffix(23))
		assert.Equal(t, "rd", ordinalSuffix(33))
	})

	t.Run("should return th for 11th, 12th, 13th", func(t *testing.T) {
		assert.Equal(t, "th", ordinalSuffix(11))
		assert.Equal(t, "th", ordinalSuffix(12))
		assert.Equal(t, "th", ordinalSuffix(13))
	})

	t.Run("should return th for other numbers", func(t *testing.T) {
		assert.Equal(t, "th", ordinalSuffix(4))
		assert.Equal(t, "th", ordinalSuffix(5))
		assert.Equal(t, "th", ordinalSuffix(6))
		assert.Equal(t, "th", ordinalSuffix(7))
		assert.Equal(t, "th", ordinalSuffix(8))
		assert.Equal(t, "th", ordinalSuffix(9))
		assert.Equal(t, "th", ordinalSuffix(10))
		assert.Equal(t, "th", ordinalSuffix(14))
		assert.Equal(t, "th", ordinalSuffix(20))
		assert.Equal(t, "th", ordinalSuffix(24))
	})

	t.Run("should handle edge cases", func(t *testing.T) {
		assert.Equal(t, "th", ordinalSuffix(0))
		assert.Equal(t, "th", ordinalSuffix(100))
		assert.Equal(t, "th", ordinalSuffix(111)) // 111 ends in 11, so "th"
		assert.Equal(t, "st", ordinalSuffix(101)) // 101 ends in 1, so "st"
		assert.Equal(t, "th", ordinalSuffix(112)) // 112 ends in 12, so "th"
		assert.Equal(t, "th", ordinalSuffix(113)) // 113 ends in 13, so "th"
		assert.Equal(t, "nd", ordinalSuffix(102)) // 102 ends in 2, so "nd"
		assert.Equal(t, "rd", ordinalSuffix(103)) // 103 ends in 3, so "rd"
	})
}
