package cronx_test

import (
	"strings"
	"testing"

	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_Parse_ValidExpressions(t *testing.T) {
	parser := cronx.NewParser()

	tests := []struct {
		name       string
		expression string
		wantErr    bool
	}{
		{
			name:       "valid standard expression",
			expression: "0 0 * * *",
			wantErr:    false,
		},
		{
			name:       "valid with ranges",
			expression: "*/15 9-17 * * 1-5",
			wantErr:    false,
		},
		{
			name:       "valid with lists",
			expression: "0 0,12 * * *",
			wantErr:    false,
		},
		{
			name:       "every minute",
			expression: "* * * * *",
			wantErr:    false,
		},
		{
			name:       "complex expression",
			expression: "*/5 2-5 * * 1-5",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, schedule)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, schedule)
				assert.Equal(t, tt.expression, schedule.Original)
			}
		})
	}
}

func TestParser_Parse_InvalidExpressions(t *testing.T) {
	parser := cronx.NewParser()

	tests := []struct {
		name       string
		expression string
		errorMsg   string
	}{
		{
			name:       "empty expression",
			expression: "",
			errorMsg:   "empty expression",
		},
		{
			name:       "invalid syntax",
			expression: "invalid",
			errorMsg:   "expected 5 fields",
		},
		{
			name:       "too many fields",
			expression: "* * * * * *",
			errorMsg:   "expected 5 fields",
		},
		{
			name:       "out of range minute",
			expression: "60 * * * *",
			errorMsg:   "out of range",
		},
		{
			name:       "out of range hour",
			expression: "* 24 * * *",
			errorMsg:   "out of range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)

			require.Error(t, err)
			assert.Nil(t, schedule)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestParser_ParseAlias(t *testing.T) {
	parser := cronx.NewParser()

	tests := []struct {
		name    string
		alias   string
		wantErr bool
	}{
		{name: "daily alias", alias: "@daily", wantErr: false},
		{name: "hourly alias", alias: "@hourly", wantErr: false},
		{name: "weekly alias", alias: "@weekly", wantErr: false},
		{name: "monthly alias", alias: "@monthly", wantErr: false},
		{name: "yearly alias", alias: "@yearly", wantErr: false},
		{name: "annually alias", alias: "@annually", wantErr: false},
		{name: "invalid alias", alias: "@invalid", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.alias)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, schedule)
				assert.Equal(t, tt.alias, schedule.Original)
			}
		})
	}
}

func TestParser_ParseCaseInsensitive(t *testing.T) {
	parser := cronx.NewParser()

	tests := []struct {
		name       string
		expression string
	}{
		{name: "uppercase days", expression: "0 0 * * MON-FRI"},
		{name: "lowercase days", expression: "0 0 * * mon-fri"},
		{name: "mixed case days", expression: "0 0 * * Mon-Fri"},
		{name: "uppercase months", expression: "0 0 1 JAN-DEC *"},
		{name: "lowercase months", expression: "0 0 1 jan-dec *"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)
			require.NoError(t, err)
			assert.NotNil(t, schedule)

			if strings.Contains(tt.name, "days") {
				assert.Equal(t, 1, schedule.DayOfWeek.RangeStart())
				assert.Equal(t, 5, schedule.DayOfWeek.RangeEnd())
			}
			if strings.Contains(tt.name, "months") {
				assert.Equal(t, 1, schedule.Month.RangeStart())
				assert.Equal(t, 12, schedule.Month.RangeEnd())
			}
		})
	}
}

func TestSchedule_Fields(t *testing.T) {
	parser := cronx.NewParser()
	schedule, err := parser.Parse("*/15 9-17 * * 1-5")
	require.NoError(t, err)

	// Schedule should expose parsed field information
	assert.NotNil(t, schedule.Minute)
	assert.NotNil(t, schedule.Hour)
	assert.NotNil(t, schedule.DayOfMonth)
	assert.NotNil(t, schedule.Month)
	assert.NotNil(t, schedule.DayOfWeek)
}

func TestSchedule_Original(t *testing.T) {
	parser := cronx.NewParser()
	expression := "0 9 * * 1-5"
	schedule, err := parser.Parse(expression)
	require.NoError(t, err)

	// Should preserve original expression
	assert.Equal(t, expression, schedule.Original)
}

func TestField_EveryPattern(t *testing.T) {
	parser := cronx.NewParser()
	schedule, err := parser.Parse("* * * * *")
	require.NoError(t, err)

	// All fields should be "every" (wildcard)
	assert.True(t, schedule.Minute.IsEvery())
	assert.True(t, schedule.Hour.IsEvery())
	assert.True(t, schedule.DayOfMonth.IsEvery())
	assert.True(t, schedule.Month.IsEvery())
	assert.True(t, schedule.DayOfWeek.IsEvery())
}

func TestField_StepPattern(t *testing.T) {
	parser := cronx.NewParser()
	schedule, err := parser.Parse("*/15 * * * *")
	require.NoError(t, err)

	minute := schedule.Minute
	assert.True(t, minute.IsStep())
	assert.Equal(t, 15, minute.Step())
}

func TestField_RangePattern(t *testing.T) {
	parser := cronx.NewParser()
	schedule, err := parser.Parse("0 9-17 * * *")
	require.NoError(t, err)

	hour := schedule.Hour
	assert.True(t, hour.IsRange())
	assert.Equal(t, 9, hour.RangeStart())
	assert.Equal(t, 17, hour.RangeEnd())
}

func TestField_ListPattern(t *testing.T) {
	parser := cronx.NewParser()
	schedule, err := parser.Parse("0 9,12,17 * * *")
	require.NoError(t, err)

	hour := schedule.Hour
	assert.True(t, hour.IsList())
	assert.Equal(t, []int{9, 12, 17}, hour.ListValues())
}

func TestField_SinglePattern(t *testing.T) {
	parser := cronx.NewParser()
	schedule, err := parser.Parse("30 14 * * *")
	require.NoError(t, err)

	minute := schedule.Minute
	assert.True(t, minute.IsSingle())
	assert.Equal(t, 30, minute.Value())

	hour := schedule.Hour
	assert.True(t, hour.IsSingle())
	assert.Equal(t, 14, hour.Value())
}

func TestField_Raw(t *testing.T) {
	parser := cronx.NewParser()
	schedule, err := parser.Parse("*/15 9-17 1,15 * 1-5")
	require.NoError(t, err)

	assert.Equal(t, "*/15", schedule.Minute.Raw())
	assert.Equal(t, "9-17", schedule.Hour.Raw())
	assert.Equal(t, "1,15", schedule.DayOfMonth.Raw())
	assert.Equal(t, "*", schedule.Month.Raw())
	assert.Equal(t, "1-5", schedule.DayOfWeek.Raw())
}

func TestField_EdgeCases(t *testing.T) {
	parser := cronx.NewParser()

	t.Run("RangeStart returns 0 for non-range fields", func(t *testing.T) {
		schedule, err := parser.Parse("0 * * * *")
		require.NoError(t, err)
		// Hour is wildcard, not a range
		assert.Equal(t, 0, schedule.Hour.RangeStart())
	})

	t.Run("RangeEnd returns 0 for non-range fields", func(t *testing.T) {
		schedule, err := parser.Parse("0 * * * *")
		require.NoError(t, err)
		// Hour is wildcard, not a range
		assert.Equal(t, 0, schedule.Hour.RangeEnd())
	})

	t.Run("Value returns 0 for non-single fields", func(t *testing.T) {
		schedule, err := parser.Parse("0 9-17 * * *")
		require.NoError(t, err)
		// Hour is a range, not a single value
		assert.Equal(t, 0, schedule.Hour.Value())
	})

	t.Run("ListValues with ranges in list", func(t *testing.T) {
		schedule, err := parser.Parse("0 9-11,15-17 * * *")
		require.NoError(t, err)
		// Hour is a list with ranges
		values := schedule.Hour.ListValues()
		assert.Contains(t, values, 9)
		assert.Contains(t, values, 10)
		assert.Contains(t, values, 11)
		assert.Contains(t, values, 15)
		assert.Contains(t, values, 16)
		assert.Contains(t, values, 17)
	})

	t.Run("ListValues with single values and ranges", func(t *testing.T) {
		schedule, err := parser.Parse("0 5,9-11,15 * * *")
		require.NoError(t, err)
		// Hour is a list with single value and ranges
		values := schedule.Hour.ListValues()
		assert.Contains(t, values, 5)
		assert.Contains(t, values, 9)
		assert.Contains(t, values, 10)
		assert.Contains(t, values, 11)
		assert.Contains(t, values, 15)
	})
}
