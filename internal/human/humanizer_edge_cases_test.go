package human_test

import (
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
