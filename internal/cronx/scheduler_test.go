package cronx_test

import (
	"testing"
	"time"

	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduler_Next_BasicExpression(t *testing.T) {
	scheduler := cronx.NewScheduler()
	from := time.Date(2025, 12, 18, 17, 0, 0, 0, time.UTC)

	times, err := scheduler.Next("*/15 * * * *", from, 3)

	require.NoError(t, err)
	assert.Len(t, times, 3)
	assert.Equal(t, time.Date(2025, 12, 18, 17, 15, 0, 0, time.UTC), times[0])
	assert.Equal(t, time.Date(2025, 12, 18, 17, 30, 0, 0, time.UTC), times[1])
	assert.Equal(t, time.Date(2025, 12, 18, 17, 45, 0, 0, time.UTC), times[2])
}

func TestScheduler_Next_IntervalPatterns(t *testing.T) {
	scheduler := cronx.NewScheduler()
	from := time.Date(2025, 12, 18, 17, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		expression string
		count      int
		expected   []time.Time
	}{
		{
			name:       "every 5 minutes",
			expression: "*/5 * * * *",
			count:      3,
			expected: []time.Time{
				time.Date(2025, 12, 18, 17, 5, 0, 0, time.UTC),
				time.Date(2025, 12, 18, 17, 10, 0, 0, time.UTC),
				time.Date(2025, 12, 18, 17, 15, 0, 0, time.UTC),
			},
		},
		{
			name:       "every 30 minutes",
			expression: "*/30 * * * *",
			count:      2,
			expected: []time.Time{
				time.Date(2025, 12, 18, 17, 30, 0, 0, time.UTC),
				time.Date(2025, 12, 18, 18, 0, 0, 0, time.UTC),
			},
		},
		{
			name:       "every hour",
			expression: "0 * * * *",
			count:      3,
			expected: []time.Time{
				time.Date(2025, 12, 18, 18, 0, 0, 0, time.UTC),
				time.Date(2025, 12, 18, 19, 0, 0, 0, time.UTC),
				time.Date(2025, 12, 18, 20, 0, 0, 0, time.UTC),
			},
		},
		{
			name:       "every minute",
			expression: "* * * * *",
			count:      3,
			expected: []time.Time{
				time.Date(2025, 12, 18, 17, 1, 0, 0, time.UTC),
				time.Date(2025, 12, 18, 17, 2, 0, 0, time.UTC),
				time.Date(2025, 12, 18, 17, 3, 0, 0, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			times, err := scheduler.Next(tt.expression, from, tt.count)
			require.NoError(t, err)
			assert.Len(t, times, tt.count)
			for i, expected := range tt.expected {
				assert.Equal(t, expected, times[i], "mismatch at index %d", i)
			}
		})
	}
}

func TestScheduler_Next_CronAliases(t *testing.T) {
	scheduler := cronx.NewScheduler()
	from := time.Date(2025, 12, 18, 17, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		expression string
		count      int
		expected   []time.Time
	}{
		{
			name:       "@daily",
			expression: "@daily",
			count:      3,
			expected: []time.Time{
				time.Date(2025, 12, 19, 0, 0, 0, 0, time.UTC),
				time.Date(2025, 12, 20, 0, 0, 0, 0, time.UTC),
				time.Date(2025, 12, 21, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:       "@hourly",
			expression: "@hourly",
			count:      3,
			expected: []time.Time{
				time.Date(2025, 12, 18, 18, 0, 0, 0, time.UTC),
				time.Date(2025, 12, 18, 19, 0, 0, 0, time.UTC),
				time.Date(2025, 12, 18, 20, 0, 0, 0, time.UTC),
			},
		},
		{
			name:       "@weekly",
			expression: "@weekly",
			count:      2,
			expected: []time.Time{
				time.Date(2025, 12, 21, 0, 0, 0, 0, time.UTC), // Sunday
				time.Date(2025, 12, 28, 0, 0, 0, 0, time.UTC), // Next Sunday
			},
		},
		{
			name:       "@monthly",
			expression: "@monthly",
			count:      3,
			expected: []time.Time{
				time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:       "@yearly",
			expression: "@yearly",
			count:      2,
			expected: []time.Time{
				time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			times, err := scheduler.Next(tt.expression, from, tt.count)
			require.NoError(t, err)
			assert.Len(t, times, tt.count)
			for i, expected := range tt.expected {
				assert.Equal(t, expected, times[i], "mismatch at index %d", i)
			}
		})
	}
}

func TestScheduler_Next_WeekdayPatterns(t *testing.T) {
	scheduler := cronx.NewScheduler()
	from := time.Date(2025, 12, 18, 17, 0, 0, 0, time.UTC) // Thursday

	tests := []struct {
		name       string
		expression string
		count      int
		validate   func(t *testing.T, times []time.Time)
	}{
		{
			name:       "weekdays (Monday-Friday)",
			expression: "0 9 * * 1-5",
			count:      5,
			validate: func(t *testing.T, times []time.Time) {
				// Should be Friday, Monday, Tuesday, Wednesday, Thursday
				assert.Equal(t, time.Date(2025, 12, 19, 9, 0, 0, 0, time.UTC), times[0]) // Friday
				assert.Equal(t, time.Date(2025, 12, 22, 9, 0, 0, 0, time.UTC), times[1]) // Monday
				assert.Equal(t, time.Date(2025, 12, 23, 9, 0, 0, 0, time.UTC), times[2]) // Tuesday
				assert.Equal(t, time.Date(2025, 12, 24, 9, 0, 0, 0, time.UTC), times[3]) // Wednesday
				assert.Equal(t, time.Date(2025, 12, 25, 9, 0, 0, 0, time.UTC), times[4]) // Thursday
			},
		},
		{
			name:       "specific days (Monday, Wednesday, Friday)",
			expression: "0 9 * * 1,3,5",
			count:      3,
			validate: func(t *testing.T, times []time.Time) {
				assert.Equal(t, time.Date(2025, 12, 19, 9, 0, 0, 0, time.UTC), times[0]) // Friday
				assert.Equal(t, time.Date(2025, 12, 22, 9, 0, 0, 0, time.UTC), times[1]) // Monday
				assert.Equal(t, time.Date(2025, 12, 24, 9, 0, 0, 0, time.UTC), times[2]) // Wednesday
			},
		},
		{
			name:       "Sunday only",
			expression: "0 0 * * 0",
			count:      2,
			validate: func(t *testing.T, times []time.Time) {
				assert.Equal(t, time.Date(2025, 12, 21, 0, 0, 0, 0, time.UTC), times[0])
				assert.Equal(t, time.Date(2025, 12, 28, 0, 0, 0, 0, time.UTC), times[1])
			},
		},
		{
			name:       "Saturday only",
			expression: "0 0 * * 6",
			count:      2,
			validate: func(t *testing.T, times []time.Time) {
				assert.Equal(t, time.Date(2025, 12, 20, 0, 0, 0, 0, time.UTC), times[0])
				assert.Equal(t, time.Date(2025, 12, 27, 0, 0, 0, 0, time.UTC), times[1])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			times, err := scheduler.Next(tt.expression, from, tt.count)
			require.NoError(t, err)
			assert.Len(t, times, tt.count)
			tt.validate(t, times)
		})
	}
}

func TestScheduler_Next_SpecificTimes(t *testing.T) {
	scheduler := cronx.NewScheduler()
	from := time.Date(2025, 12, 18, 17, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		expression string
		count      int
		expected   []time.Time
	}{
		{
			name:       "midnight daily",
			expression: "0 0 * * *",
			count:      3,
			expected: []time.Time{
				time.Date(2025, 12, 19, 0, 0, 0, 0, time.UTC),
				time.Date(2025, 12, 20, 0, 0, 0, 0, time.UTC),
				time.Date(2025, 12, 21, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:       "specific time (14:30)",
			expression: "30 14 * * *",
			count:      2,
			expected: []time.Time{
				time.Date(2025, 12, 19, 14, 30, 0, 0, time.UTC),
				time.Date(2025, 12, 20, 14, 30, 0, 0, time.UTC),
			},
		},
		{
			name:       "multiple times per day",
			expression: "0 9,12,17 * * *",
			count:      4,
			expected: []time.Time{
				time.Date(2025, 12, 19, 9, 0, 0, 0, time.UTC),
				time.Date(2025, 12, 19, 12, 0, 0, 0, time.UTC),
				time.Date(2025, 12, 19, 17, 0, 0, 0, time.UTC),
				time.Date(2025, 12, 20, 9, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			times, err := scheduler.Next(tt.expression, from, tt.count)
			require.NoError(t, err)
			assert.Len(t, times, tt.count)
			for i, expected := range tt.expected {
				assert.Equal(t, expected, times[i], "mismatch at index %d", i)
			}
		})
	}
}

func TestScheduler_Next_DayOfMonthPatterns(t *testing.T) {
	scheduler := cronx.NewScheduler()
	from := time.Date(2025, 12, 18, 17, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		expression string
		count      int
		validate   func(t *testing.T, times []time.Time)
	}{
		{
			name:       "first day of month",
			expression: "0 0 1 * *",
			count:      3,
			validate: func(t *testing.T, times []time.Time) {
				assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), times[0])
				assert.Equal(t, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), times[1])
				assert.Equal(t, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), times[2])
			},
		},
		{
			name:       "15th of every month",
			expression: "0 0 15 * *",
			count:      3,
			validate: func(t *testing.T, times []time.Time) {
				assert.Equal(t, 15, times[0].Day())
				assert.Equal(t, 15, times[1].Day())
				assert.Equal(t, 15, times[2].Day())
			},
		},
		{
			name:       "last day of month (31st where valid)",
			expression: "0 0 31 * *",
			count:      3,
			validate: func(t *testing.T, times []time.Time) {
				// Should only match months with 31 days
				assert.Equal(t, 31, times[0].Day())
				assert.Equal(t, 31, times[1].Day())
				assert.Equal(t, 31, times[2].Day())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			times, err := scheduler.Next(tt.expression, from, tt.count)
			require.NoError(t, err)
			assert.Len(t, times, tt.count)
			tt.validate(t, times)
		})
	}
}

func TestScheduler_Next_ComplexPatterns(t *testing.T) {
	scheduler := cronx.NewScheduler()
	from := time.Date(2025, 12, 18, 17, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		expression string
		count      int
		validate   func(t *testing.T, times []time.Time)
	}{
		{
			name:       "business hours (9-5 weekdays)",
			expression: "*/5 9-17 * * 1-5",
			count:      10,
			validate: func(t *testing.T, times []time.Time) {
				for _, tm := range times {
					hour := tm.Hour()
					assert.True(t, hour >= 9 && hour <= 17, "hour should be between 9 and 17")
					assert.True(t, tm.Weekday() >= time.Monday && tm.Weekday() <= time.Friday,
						"should be weekday")
				}
			},
		},
		{
			name:       "specific month and day",
			expression: "0 0 1 1 *",
			count:      3,
			validate: func(t *testing.T, times []time.Time) {
				for _, tm := range times {
					assert.Equal(t, time.January, tm.Month())
					assert.Equal(t, 1, tm.Day())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			times, err := scheduler.Next(tt.expression, from, tt.count)
			require.NoError(t, err)
			assert.Len(t, times, tt.count)
			tt.validate(t, times)
		})
	}
}

func TestScheduler_Next_CountValidation(t *testing.T) {
	scheduler := cronx.NewScheduler()
	from := time.Date(2025, 12, 18, 17, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		expression string
		count      int
		wantLen    int
	}{
		{
			name:       "count of 1",
			expression: "*/15 * * * *",
			count:      1,
			wantLen:    1,
		},
		{
			name:       "count of 10 (default)",
			expression: "@daily",
			count:      10,
			wantLen:    10,
		},
		{
			name:       "count of 100 (maximum)",
			expression: "0 * * * *",
			count:      100,
			wantLen:    100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			times, err := scheduler.Next(tt.expression, from, tt.count)
			require.NoError(t, err)
			assert.Len(t, times, tt.wantLen)
		})
	}
}

func TestScheduler_Next_InvalidExpressions(t *testing.T) {
	scheduler := cronx.NewScheduler()
	from := time.Date(2025, 12, 18, 17, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		expression string
		count      int
		wantError  string
	}{
		{
			name:       "wrong field count",
			expression: "0 0 *",
			count:      1,
			wantError:  "expected 5 fields",
		},
		{
			name:       "invalid minute value",
			expression: "60 0 * * *",
			count:      1,
			wantError:  "out of range",
		},
		{
			name:       "invalid hour value",
			expression: "0 24 * * *",
			count:      1,
			wantError:  "out of range",
		},
		{
			name:       "invalid day of month",
			expression: "0 0 32 * *",
			count:      1,
			wantError:  "out of range",
		},
		{
			name:       "invalid month",
			expression: "0 0 1 13 *",
			count:      1,
			wantError:  "out of range",
		},
		{
			name:       "invalid day of week",
			expression: "0 0 * * 7",
			count:      1,
			wantError:  "out of range",
		},
		{
			name:       "completely invalid",
			expression: "not-a-cron",
			count:      1,
			wantError:  "expected 5 fields",
		},
		{
			name:       "invalid alias",
			expression: "@invalid",
			count:      1,
			wantError:  "unrecognized descriptor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			times, err := scheduler.Next(tt.expression, from, tt.count)
			assert.Error(t, err)
			assert.Nil(t, times)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestScheduler_Next_EdgeCases(t *testing.T) {
	scheduler := cronx.NewScheduler()

	tests := []struct {
		name       string
		expression string
		from       time.Time
		count      int
		validate   func(t *testing.T, times []time.Time)
	}{
		{
			name:       "leap year February 29",
			expression: "0 0 29 2 *",
			from:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			count:      2,
			validate: func(t *testing.T, times []time.Time) {
				// Should skip non-leap years
				assert.Equal(t, 29, times[0].Day())
				assert.Equal(t, time.February, times[0].Month())
				assert.Equal(t, 2028, times[0].Year()) // Next leap year
			},
		},
		{
			name:       "end of month transition",
			expression: "0 0 31 * *",
			from:       time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC),
			count:      5,
			validate: func(t *testing.T, times []time.Time) {
				// Should only match months with 31 days
				for _, tm := range times {
					assert.Equal(t, 31, tm.Day())
				}
			},
		},
		{
			name:       "year transition",
			expression: "0 0 1 1 *",
			from:       time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC),
			count:      3,
			validate: func(t *testing.T, times []time.Time) {
				assert.Equal(t, 2026, times[0].Year())
				assert.Equal(t, 2027, times[1].Year())
				assert.Equal(t, 2028, times[2].Year())
			},
		},
		{
			name:       "minute 59",
			expression: "59 * * * *",
			from:       time.Date(2025, 12, 18, 17, 0, 0, 0, time.UTC),
			count:      3,
			validate: func(t *testing.T, times []time.Time) {
				for _, tm := range times {
					assert.Equal(t, 59, tm.Minute())
				}
			},
		},
		{
			name:       "hour 23",
			expression: "0 23 * * *",
			from:       time.Date(2025, 12, 18, 17, 0, 0, 0, time.UTC),
			count:      3,
			validate: func(t *testing.T, times []time.Time) {
				for _, tm := range times {
					assert.Equal(t, 23, tm.Hour())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			times, err := scheduler.Next(tt.expression, tt.from, tt.count)
			require.NoError(t, err)
			assert.Len(t, times, tt.count)
			tt.validate(t, times)
		})
	}
}

func TestScheduler_Next_TimeProgression(t *testing.T) {
	scheduler := cronx.NewScheduler()
	from := time.Date(2025, 12, 18, 17, 0, 0, 0, time.UTC)

	times, err := scheduler.Next("*/15 * * * *", from, 10)
	require.NoError(t, err)

	// Verify times are in ascending order
	for i := 1; i < len(times); i++ {
		assert.True(t, times[i].After(times[i-1]),
			"time at index %d should be after time at index %d", i, i-1)
	}

	// Verify times are all in the future relative to 'from'
	for i, tm := range times {
		assert.True(t, tm.After(from),
			"time at index %d should be after 'from' time", i)
	}
}
