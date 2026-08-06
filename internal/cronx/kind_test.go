package cronx_test

import (
	"testing"

	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldKind(t *testing.T) {
	parser := cronx.NewParser()

	tests := []struct {
		name       string
		expression string
		field      func(*cronx.Schedule) cronx.Field
		expected   cronx.FieldKind
	}{
		{"wildcard", "* * * * *", minuteOf, cronx.KindEvery},
		{"wildcard with unit step", "*/1 * * * *", minuteOf, cronx.KindEvery},
		{"step", "*/15 * * * *", minuteOf, cronx.KindStep},
		{"bounded step", "1-5/2 * * * *", minuteOf, cronx.KindBoundedStep},
		{"bounded step on hour", "0 9-17/2 * * *", hourOf, cronx.KindBoundedStep},
		{"range with unit step", "1-5/1 * * * *", minuteOf, cronx.KindRange},
		{"single", "30 * * * *", minuteOf, cronx.KindSingle},
		{"range", "1-5 * * * *", minuteOf, cronx.KindRange},
		{"single with step", "5/20 * * * *", minuteOf, cronx.KindBoundedStep},
		{"list", "1,31 * * * *", minuteOf, cronx.KindList},
		{"list containing step", "0,*/5 * * * *", minuteOf, cronx.KindList},
		{"list containing range", "1-5,30 * * * *", minuteOf, cronx.KindList},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, tt.field(schedule).Kind())
		})
	}
}

func TestFieldKindIsTotal(t *testing.T) {
	parser := cronx.NewParser()

	expressions := []string{
		"* * * * *", "*/15 * * * *", "1-5/2 * * * *", "30 * * * *",
		"1-5 * * * *", "1,31 * * * *", "0,*/5 * * * *", "*/1 * * * *",
	}

	valid := map[cronx.FieldKind]bool{
		cronx.KindEvery: true, cronx.KindStep: true, cronx.KindBoundedStep: true,
		cronx.KindSingle: true, cronx.KindRange: true, cronx.KindList: true,
	}

	for _, expr := range expressions {
		t.Run(expr, func(t *testing.T) {
			schedule, err := parser.Parse(expr)
			require.NoError(t, err)
			for _, f := range []cronx.Field{schedule.Minute, schedule.Hour, schedule.DayOfMonth, schedule.Month, schedule.DayOfWeek} {
				assert.True(t, valid[f.Kind()], "field %q produced unclassified kind %d", f.Raw(), f.Kind())
			}
		})
	}
}

func TestBoundedStepBounds(t *testing.T) {
	parser := cronx.NewParser()

	tests := []struct {
		name       string
		expression string
		field      func(*cronx.Schedule) cronx.Field
		start, end int
	}{
		{"explicit range", "1-5/2 * * * *", minuteOf, 1, 5},
		{"single with step runs to field max", "5/20 * * * *", minuteOf, 5, 59},
		{"hour single with step runs to field max", "0 9/2 * * *", hourOf, 9, 23},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)
			require.NoError(t, err)
			f := tt.field(schedule)
			assert.Equal(t, tt.start, f.RangeStart())
			assert.Equal(t, tt.end, f.RangeEnd())
		})
	}
}

func TestListValuesHonoursStep(t *testing.T) {
	parser := cronx.NewParser()

	tests := []struct {
		name       string
		expression string
		field      func(*cronx.Schedule) cronx.Field
		expected   []int
	}{
		{"stepped range", "1-5/2 * * * *", minuteOf, []int{1, 3, 5}},
		{"stepped range not landing on end", "10-20/5 * * * *", minuteOf, []int{10, 15, 20}},
		{"plain range unaffected", "1-4 * * * *", minuteOf, []int{1, 2, 3, 4}},
		{"stepped range inside list", "1-5/2,30 * * * *", minuteOf, []int{1, 3, 5, 30}},
		{"stepped hour range", "0 9-17/2 * * *", hourOf, []int{9, 11, 13, 15, 17}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, tt.field(schedule).ListValues())
		})
	}
}

func minuteOf(s *cronx.Schedule) cronx.Field { return s.Minute }
func hourOf(s *cronx.Schedule) cronx.Field   { return s.Hour }
