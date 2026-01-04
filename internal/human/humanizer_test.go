package human_test

import (
	"fmt"
	"testing"

	"github.com/hzerrad/cronic/internal/cronx"
	"github.com/hzerrad/cronic/internal/human"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test formatters directly (they're not exported, but we can test them via humanizer)
func TestFormatters_EdgeCases(t *testing.T) {
	parser := cronx.NewParser()
	humanizer := human.NewHumanizer()

	t.Run("formatList with empty list", func(t *testing.T) {
		// Test via humanizer with expression that might trigger formatList
		schedule, err := parser.Parse("0 9 * * 1,2,3")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should contain formatted day list
		assert.Contains(t, result, "Monday")
		assert.Contains(t, result, "Tuesday")
		assert.Contains(t, result, "Wednesday")
	})

	t.Run("dayName with valid day numbers", func(t *testing.T) {
		// Test via humanizer with different day numbers
		days := []string{"0 9 * * 0", "0 9 * * 1", "0 9 * * 2", "0 9 * * 3", "0 9 * * 4", "0 9 * * 5", "0 9 * * 6"}
		expected := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
		for i, expr := range days {
			schedule, err := parser.Parse(expr)
			require.NoError(t, err)
			result := humanizer.Humanize(schedule)
			assert.Contains(t, result, expected[i], "dayName should return correct day for %s", expr)
		}
	})

	t.Run("formatMonth with valid month numbers", func(t *testing.T) {
		// Test via humanizer with different month numbers
		months := []string{"0 9 1 1 *", "0 9 1 2 *", "0 9 1 3 *", "0 9 1 4 *", "0 9 1 5 *", "0 9 1 6 *",
			"0 9 1 7 *", "0 9 1 8 *", "0 9 1 9 *", "0 9 1 10 *", "0 9 1 11 *", "0 9 1 12 *"}
		expected := []string{"January", "February", "March", "April", "May", "June",
			"July", "August", "September", "October", "November", "December"}
		for i, expr := range months {
			schedule, err := parser.Parse(expr)
			require.NoError(t, err)
			result := humanizer.Humanize(schedule)
			assert.Contains(t, result, expected[i], "formatMonth should return correct month for %s", expr)
		}
	})

	t.Run("formatList with multiple items", func(t *testing.T) {
		// Test formatList with 3+ items via humanizer
		schedule, err := parser.Parse("0 9 * * 1,3,5")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should contain formatted list with Oxford comma
		assert.Contains(t, result, "Monday")
		assert.Contains(t, result, "Wednesday")
		assert.Contains(t, result, "Friday")
		// Should have "and" before last item
		assert.Contains(t, result, "and Friday")
	})

	t.Run("formatList with two items", func(t *testing.T) {
		// Test formatList with 2 items via humanizer
		schedule, err := parser.Parse("0 9 * * 1,2")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should contain "and" between two items
		assert.Contains(t, result, "Monday")
		assert.Contains(t, result, "Tuesday")
		assert.Contains(t, result, "and Tuesday")
	})

	t.Run("buildDayPart with day of month range", func(t *testing.T) {
		// Test formatDayOfMonth with range
		schedule, err := parser.Parse("0 9 1-5 * *")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should contain day range
		assert.Contains(t, result, "days 1-5")
	})

	t.Run("buildDayPart with day of month single value", func(t *testing.T) {
		// Test formatDayOfMonth with single value (not 1)
		schedule, err := parser.Parse("0 9 15 * *")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should contain day number
		assert.Contains(t, result, "day 15")
	})

	t.Run("buildDayPart with day of month first day", func(t *testing.T) {
		// Test formatDayOfMonth with value 1 (special case)
		schedule, err := parser.Parse("0 9 1 * *")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should contain "first day"
		assert.Contains(t, result, "first day")
	})

	t.Run("formatDayOfWeek with range not Mon-Fri", func(t *testing.T) {
		// Test formatDayOfWeek with range that's not 1-5
		schedule, err := parser.Parse("0 9 * * 0-2")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should contain day range
		assert.Contains(t, result, "Sunday")
		assert.Contains(t, result, "Tuesday")
	})

	t.Run("buildMonthPart with month list", func(t *testing.T) {
		// Test buildMonthPart with month list
		schedule, err := parser.Parse("0 9 * 1,3,5 *")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should contain month list
		assert.Contains(t, result, "January")
		assert.Contains(t, result, "March")
		assert.Contains(t, result, "May")
	})

	t.Run("buildMonthPart with month range", func(t *testing.T) {
		// Test buildMonthPart with month range
		schedule, err := parser.Parse("0 9 * 1-3 *")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should contain month range
		assert.Contains(t, result, "from January to March")
	})

	t.Run("formatDayOfMonth with list returns empty", func(t *testing.T) {
		// Test formatDayOfMonth with list (should return empty, not handled)
		// This tests the return "" path
		schedule, err := parser.Parse("0 9 1,15 * *")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should still work, but formatDayOfMonth returns "" for lists
		// The humanizer should handle this gracefully
		assert.NotEmpty(t, result)
	})

	t.Run("formatDayOfMonth with step returns empty", func(t *testing.T) {
		// Test formatDayOfMonth with step (should return empty, not handled)
		schedule, err := parser.Parse("0 9 */5 * *")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should still work
		assert.NotEmpty(t, result)
	})

	t.Run("buildDayPart with both wildcards", func(t *testing.T) {
		// Test buildDayPart when both dayOfWeek and dayOfMonth are wildcards
		schedule, err := parser.Parse("0 9 * * *")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should contain "every day" or be part of simple pattern
		assert.NotEmpty(t, result)
	})

	t.Run("buildDayPart with dayOfWeek priority", func(t *testing.T) {
		// Test that dayOfWeek has priority over dayOfMonth
		schedule, err := parser.Parse("0 9 15 * 1")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should show day of week, not day of month
		assert.Contains(t, result, "Monday")
		assert.NotContains(t, result, "day 15")
	})

	t.Run("buildDayPart fallback to every day", func(t *testing.T) {
		// Test the final fallback case where both dayOfWeek and dayOfMonth are wildcards
		// This should return "every day"
		schedule, err := parser.Parse("0 9 * * *")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Both are wildcards, so should return "every day"
		assert.Contains(t, result, "every day", "Should return 'every day' when both dayOfWeek and dayOfMonth are wildcards")
	})

	t.Run("formatDayOfWeek with step returns empty", func(t *testing.T) {
		// Test formatDayOfWeek with step (should return empty, not handled)
		// This tests the return "" path at the end of formatDayOfWeek
		schedule, err := parser.Parse("0 9 * * */2")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should still work, but formatDayOfWeek returns "" for steps
		// The humanizer should handle this gracefully
		assert.NotEmpty(t, result)
	})

	t.Run("formatDayOfWeek with every Sunday", func(t *testing.T) {
		// Test formatDayOfWeek with Sunday (0) - special case
		schedule, err := parser.Parse("0 9 * * 0")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should contain "every Sunday"
		assert.Contains(t, result, "every Sunday")
	})

	t.Run("formatDayOfWeek with single day not Sunday", func(t *testing.T) {
		// Test formatDayOfWeek with single day that's not Sunday
		schedule, err := parser.Parse("0 9 * * 3")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should contain "every Wednesday"
		assert.Contains(t, result, "every Wednesday")
	})

	t.Run("buildMonthPart returns empty for every month", func(t *testing.T) {
		// Test buildMonthPart with wildcard month
		schedule, err := parser.Parse("0 9 * * *")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should not contain month info
		assert.NotContains(t, result, "in January")
		assert.NotContains(t, result, "from")
	})

	t.Run("buildMonthPart returns empty for unhandled cases", func(t *testing.T) {
		// Test buildMonthPart with step (should return empty)
		schedule, err := parser.Parse("0 9 * */2 *")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		// Should still work
		assert.NotEmpty(t, result)
	})
}

func TestHumanizer_Humanize_StandardExpressions(t *testing.T) {
	parser := cronx.NewParser()
	humanizer := human.NewHumanizer()

	tests := []struct {
		name       string
		expression string
		expected   string
	}{
		{
			name:       "every minute",
			expression: "* * * * *",
			expected:   "Every minute",
		},
		{
			name:       "every 15 minutes",
			expression: "*/15 * * * *",
			expected:   "Every 15 minutes",
		},
		{
			name:       "daily at midnight",
			expression: "0 0 * * *",
			expected:   "At midnight every day",
		},
		{
			name:       "hourly",
			expression: "0 * * * *",
			expected:   "At the start of every hour",
		},
		{
			name:       "weekdays at 9am",
			expression: "0 9 * * 1-5",
			expected:   "At 09:00 on weekdays (Mon-Fri)",
		},
		{
			name:       "every 5 minutes during business hours on weekdays",
			expression: "*/5 9-17 * * 1-5",
			expected:   "Every 5 minutes between 09:00 and 17:59 on weekdays (Mon-Fri)",
		},
		{
			name:       "every 15 minutes in early morning on weekdays",
			expression: "*/15 2-5 * * 1-5",
			expected:   "Every 15 minutes between 02:00 and 05:59 on weekdays (Mon-Fri)",
		},
		{
			name:       "specific time 2:30pm",
			expression: "30 14 * * *",
			expected:   "At 14:30 every day",
		},
		{
			name:       "midnight and noon",
			expression: "0 0,12 * * *",
			expected:   "At 00:00 and 12:00 every day",
		},
		{
			name:       "9am, noon, and 5pm",
			expression: "0 9,12,17 * * *",
			expected:   "At 09:00, 12:00, and 17:00 every day",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)
			require.NoError(t, err)

			result := humanizer.Humanize(schedule)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHumanizer_Humanize_Aliases(t *testing.T) {
	parser := cronx.NewParser()
	humanizer := human.NewHumanizer()

	tests := []struct {
		name     string
		alias    string
		expected string
	}{
		{
			name:     "daily alias",
			alias:    "@daily",
			expected: "At midnight every day",
		},
		{
			name:     "hourly alias",
			alias:    "@hourly",
			expected: "At the start of every hour",
		},
		{
			name:     "weekly alias",
			alias:    "@weekly",
			expected: "At midnight every Sunday",
		},
		{
			name:     "monthly alias",
			alias:    "@monthly",
			expected: "At midnight on the first day of every month",
		},
		{
			name:     "yearly alias",
			alias:    "@yearly",
			expected: "At midnight on January 1st",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.alias)
			require.NoError(t, err)

			result := humanizer.Humanize(schedule)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHumanizer_Humanize_SpecificTimes(t *testing.T) {
	parser := cronx.NewParser()
	humanizer := human.NewHumanizer()

	tests := []struct {
		name       string
		expression string
		expected   string
	}{
		{
			name:       "specific time",
			expression: "30 14 * * *",
			expected:   "At 14:30 every day",
		},
		{
			name:       "specific time on specific days",
			expression: "0 9 * * 1,3,5",
			expected:   "At 09:00 on Monday, Wednesday, and Friday",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)
			require.NoError(t, err)

			result := humanizer.Humanize(schedule)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHumanizer_Humanize_DayPatterns(t *testing.T) {
	parser := cronx.NewParser()
	humanizer := human.NewHumanizer()

	tests := []struct {
		name       string
		expression string
		expected   string
	}{
		{
			name:       "every Sunday",
			expression: "0 0 * * 0",
			expected:   "At midnight every Sunday",
		},
		{
			name:       "first of month",
			expression: "0 0 1 * *",
			expected:   "At midnight on the first day of every month",
		},
		{
			name:       "2am daily",
			expression: "0 2 * * *",
			expected:   "At 02:00 every day",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)
			require.NoError(t, err)

			result := humanizer.Humanize(schedule)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHumanizer_Humanize_IntervalPatterns(t *testing.T) {
	parser := cronx.NewParser()
	humanizer := human.NewHumanizer()

	tests := []struct {
		name       string
		expression string
		expected   string
	}{
		{
			name:       "every 10 minutes in business hours",
			expression: "*/10 8-18 * * *",
			expected:   "Every 10 minutes between 08:00 and 18:59 every day",
		},
		{
			name:       "every 30 minutes",
			expression: "*/30 * * * *",
			expected:   "Every 30 minutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)
			require.NoError(t, err)

			result := humanizer.Humanize(schedule)
			assert.Equal(t, tt.expected, result)
		})
	}
}

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
		tests := []struct {
			day      int
			expected string
		}{
			{1, "1st"},
			{2, "2nd"},
			{3, "3rd"},
			{4, "4th"},
			{11, "11th"},
			{21, "21st"},
			{22, "22nd"},
			{23, "23rd"},
			{31, "31st"},
		}

		for _, tt := range tests {
			schedule, err := parser.Parse(fmt.Sprintf("0 0 %d 1 *", tt.day))
			require.NoError(t, err)
			result := humanizer.Humanize(schedule)
			assert.Contains(t, result, tt.expected)
		}
	})

	t.Run("formatList with 3+ items", func(t *testing.T) {
		schedule, err := parser.Parse("0 9 * * 1,3,5,6")
		require.NoError(t, err)
		result := humanizer.Humanize(schedule)
		assert.Contains(t, result, "Monday")
		assert.Contains(t, result, "and Saturday")
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)
			require.NoError(t, err)

			result := humanizer.Humanize(schedule)
			assert.NotEmpty(t, result)
		})
	}
}
