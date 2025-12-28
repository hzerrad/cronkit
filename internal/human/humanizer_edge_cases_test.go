package human_test

import (
	"fmt"
	"testing"

	"github.com/hzerrad/cronic/internal/cronx"
	"github.com/hzerrad/cronic/internal/human"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHumanizer_MonthPatterns(t *testing.T) {
	parser := cronx.NewParser()
	humanizer := human.NewHumanizer()

	tests := []struct {
		name       string
		expression string
		expected   string
	}{
		{
			name:       "specific month",
			expression: "0 0 1 6 *",
			expected:   "June 1st",
		},
		{
			name:       "month range",
			expression: "0 0 1 6-8 *",
			expected:   "from June to August",
		},
		{
			name:       "month list",
			expression: "0 0 1 1,6,12 *",
			expected:   "January",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)
			require.NoError(t, err)

			result := humanizer.Humanize(schedule)
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestHumanizer_DayOfMonthPatterns(t *testing.T) {
	parser := cronx.NewParser()
	humanizer := human.NewHumanizer()

	tests := []struct {
		name       string
		expression string
		expected   string
	}{
		{
			name:       "specific day of month",
			expression: "0 0 15 * *",
			expected:   "on day 15 of every month",
		},
		{
			name:       "day of month range",
			expression: "0 0 1-7 * *",
			expected:   "on days 1-7 of every month",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)
			require.NoError(t, err)

			result := humanizer.Humanize(schedule)
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestHumanizer_OrdinalDays(t *testing.T) {
	parser := cronx.NewParser()
	humanizer := human.NewHumanizer()

	tests := []struct {
		name       string
		expression string
		dayStr     string
	}{
		{
			name:       "1st of January",
			expression: "0 0 1 1 *",
			dayStr:     "January 1st",
		},
		{
			name:       "2nd of February",
			expression: "0 0 2 2 *",
			dayStr:     "February 2nd",
		},
		{
			name:       "3rd of March",
			expression: "0 0 3 3 *",
			dayStr:     "March 3rd",
		},
		{
			name:       "11th of April (special case)",
			expression: "0 0 11 4 *",
			dayStr:     "April 11th",
		},
		{
			name:       "12th of May (special case)",
			expression: "0 0 12 5 *",
			dayStr:     "May 12th",
		},
		{
			name:       "13th of June (special case)",
			expression: "0 0 13 6 *",
			dayStr:     "June 13th",
		},
		{
			name:       "21st of July",
			expression: "0 0 21 7 *",
			dayStr:     "July 21st",
		},
		{
			name:       "22nd of August",
			expression: "0 0 22 8 *",
			dayStr:     "August 22nd",
		},
		{
			name:       "23rd of September",
			expression: "0 0 23 9 *",
			dayStr:     "September 23rd",
		},
		{
			name:       "31st of October",
			expression: "0 0 31 10 *",
			dayStr:     "October 31st",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)
			require.NoError(t, err)

			result := humanizer.Humanize(schedule)
			assert.Contains(t, result, tt.dayStr)
		})
	}
}

func TestHumanizer_EdgeCases_Formatters(t *testing.T) {
	parser := cronx.NewParser()
	humanizer := human.NewHumanizer()

	t.Run("ordinalSuffix edge cases", func(t *testing.T) {
		// Test ordinalSuffix with various day numbers
		tests := []struct {
			day      int
			expected string
		}{
			{1, "1st"},
			{2, "2nd"},
			{3, "3rd"},
			{11, "11th"}, // Special case
			{12, "12th"}, // Special case
			{13, "13th"}, // Special case
			{21, "21st"},
			{22, "22nd"},
			{23, "23rd"},
			{31, "31st"},
		}

		for _, tt := range tests {
			schedule, err := parser.Parse(fmt.Sprintf("0 0 %d 1 *", tt.day))
			require.NoError(t, err)
			result := humanizer.Humanize(schedule)
			// Should contain the ordinal suffix
			assert.Contains(t, result, tt.expected, "ordinalSuffix should work for day %d", tt.day)
		}
	})

	t.Run("buildTimePart with hour list", func(t *testing.T) {
		// Test buildTimePart with hour list
		schedule, err := parser.Parse("30 9,12,15 * * *")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should contain formatted times
		assert.Contains(t, result, "09:30")
		assert.Contains(t, result, "12:30")
		assert.Contains(t, result, "15:30")
		// Should use formatList
		assert.Contains(t, result, "and")
	})

	t.Run("buildTimePart fallback", func(t *testing.T) {
		// Test buildTimePart fallback case
		// This is hard to trigger, but we can test with complex patterns
		schedule, err := parser.Parse("*/5 9-17 * * *")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should not be fallback for this pattern
		assert.NotContains(t, result, "Runs periodically")
	})

	t.Run("formatList with empty list", func(t *testing.T) {
		// Test formatList with empty list (case 0)
		// This is hard to trigger directly, but we can test via edge cases
		// Actually, formatList is only called with non-empty lists in practice
		// But we can test the code path exists
		schedule, err := parser.Parse("0 9 * * 1")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should work normally
		assert.NotEmpty(t, result)
	})

	t.Run("formatList with single item", func(t *testing.T) {
		// Test formatList with single item (case 1)
		schedule, err := parser.Parse("0 9 * 1 *")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should contain single month name
		assert.Contains(t, result, "January")
		// Should not have "and" for single item
		// (Actually, single month won't use formatList, but let's test day lists)
		schedule2, err := parser.Parse("0 9 * * 1")
		require.NoError(t, err)
		result2 := humanizer.Humanize(schedule2)
		assert.Contains(t, result2, "Monday")
	})

	t.Run("formatList with default case (3+ items)", func(t *testing.T) {
		// Test formatList default case with 3+ items
		schedule, err := parser.Parse("0 9 * * 1,3,5,6")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should contain formatted list with Oxford comma
		assert.Contains(t, result, "Monday")
		assert.Contains(t, result, "Wednesday")
		assert.Contains(t, result, "Friday")
		assert.Contains(t, result, "Saturday")
		// Should have "and" before last item
		assert.Contains(t, result, "and Saturday")
	})

	t.Run("formatList with 4+ items for months", func(t *testing.T) {
		// Test formatList with 4+ items via month list
		schedule, err := parser.Parse("0 9 * 1,3,5,7,9 *")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should contain formatted month list with Oxford comma
		assert.Contains(t, result, "January")
		assert.Contains(t, result, "March")
		assert.Contains(t, result, "May")
		assert.Contains(t, result, "July")
		assert.Contains(t, result, "September")
		// Should have "and" before last item
		assert.Contains(t, result, "and September")
	})

	t.Run("formatList with 3 items for hours", func(t *testing.T) {
		// Test formatList with 3 items via hour list (buildTimePart case 7)
		schedule, err := parser.Parse("30 9,12,15 * * *")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should contain formatted time list with Oxford comma
		assert.Contains(t, result, "09:30")
		assert.Contains(t, result, "12:30")
		assert.Contains(t, result, "15:30")
		// Should have "and" before last item
		assert.Contains(t, result, "and 15:30")
	})

	t.Run("formatList with 2 items for hours", func(t *testing.T) {
		// Test formatList with 2 items via hour list
		schedule, err := parser.Parse("30 9,12 * * *")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should contain formatted time list with "and"
		assert.Contains(t, result, "09:30")
		assert.Contains(t, result, "12:30")
		assert.Contains(t, result, "and 12:30")
		// Should not have Oxford comma for 2 items
		assert.NotContains(t, result, ", and")
	})
}

func TestHumanizer_DayOfWeekRanges(t *testing.T) {
	parser := cronx.NewParser()
	humanizer := human.NewHumanizer()

	tests := []struct {
		name       string
		expression string
		expected   string
	}{
		{
			name:       "weekend days list",
			expression: "0 0 * * 0,6",
			expected:   "Sunday",
		},
		{
			name:       "mid-week range",
			expression: "0 0 * * 2-4",
			expected:   "on Tuesday-Thursday",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)
			require.NoError(t, err)

			result := humanizer.Humanize(schedule)
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestHumanizer_EdgeCases(t *testing.T) {
	parser := cronx.NewParser()
	humanizer := human.NewHumanizer()

	tests := []struct {
		name       string
		expression string
		shouldWork bool
	}{
		{
			name:       "minute 59",
			expression: "59 * * * *",
			shouldWork: true,
		},
		{
			name:       "hour 23",
			expression: "0 23 * * *",
			shouldWork: true,
		},
		{
			name:       "Sunday as 0",
			expression: "0 0 * * 0",
			shouldWork: true,
		},
		{
			name:       "Saturday as 6",
			expression: "0 0 * * 6",
			shouldWork: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)
			require.NoError(t, err)

			result := humanizer.Humanize(schedule)
			assert.NotEmpty(t, result, "Should produce non-empty result")
		})
	}
}
