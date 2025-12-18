package human_test

import (
	"testing"

	"github.com/hzerrad/cronic/internal/cronx"
	"github.com/hzerrad/cronic/internal/human"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			expected: "At midnight on the first day of every month in January",
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
