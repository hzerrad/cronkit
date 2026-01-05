package cronx_test

import (
	"testing"

	"github.com/hzerrad/cronic/internal/cronx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseValue tests the parseValue function indirectly through field parsing
// Since parseValue is not exported, we test it through parseField which uses it
func TestParseValue_Unparseable(t *testing.T) {
	parser := cronx.NewParser()

	// Test that parseValue returns 0 for unparseable values
	// This happens when a field contains a value that's neither numeric nor a valid symbol
	// We test this through a range that includes an invalid symbol
	_, err := parser.Parse("0 0 * * MON-INVALID")
	// This should fail because "INVALID" is not a valid day name
	assert.Error(t, err, "Parse should fail for invalid symbol in range")

	// Test with a list containing invalid value
	_, err = parser.Parse("0 0 * * MON,INVALID")
	assert.Error(t, err, "Parse should fail for invalid symbol in list")
}

// TestIsStep tests the IsStep method with various scenarios
func TestIsStep(t *testing.T) {
	parser := cronx.NewParser()

	tests := []struct {
		name       string
		expression string
		field      func(*cronx.Schedule) cronx.Field
		expected   bool
	}{
		{
			name:       "field with step notation",
			expression: "*/15 * * * *",
			field:      func(s *cronx.Schedule) cronx.Field { return s.Minute },
			expected:   true,
		},
		{
			name:       "field without step notation",
			expression: "0 * * * *",
			field:      func(s *cronx.Schedule) cronx.Field { return s.Minute },
			expected:   false,
		},
		{
			name:       "field with wildcard and no step",
			expression: "* * * * *",
			field:      func(s *cronx.Schedule) cronx.Field { return s.Minute },
			expected:   false,
		},
		{
			name:       "field with range and step",
			expression: "0-59/5 * * * *",
			field:      func(s *cronx.Schedule) cronx.Field { return s.Minute },
			expected:   true,
		},
		{
			name:       "field with list containing step",
			expression: "*/5,*/10 * * * *",
			field:      func(s *cronx.Schedule) cronx.Field { return s.Minute },
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)
			require.NoError(t, err)
			field := tt.field(schedule)
			assert.Equal(t, tt.expected, field.IsStep())
		})
	}
}

// TestStep tests the Step method with various scenarios
func TestStep(t *testing.T) {
	parser := cronx.NewParser()

	tests := []struct {
		name       string
		expression string
		field      func(*cronx.Schedule) cronx.Field
		expected   int
	}{
		{
			name:       "field with step 15",
			expression: "*/15 * * * *",
			field:      func(s *cronx.Schedule) cronx.Field { return s.Minute },
			expected:   15,
		},
		{
			name:       "field without step",
			expression: "0 * * * *",
			field:      func(s *cronx.Schedule) cronx.Field { return s.Minute },
			expected:   1,
		},
		{
			name:       "field with wildcard and no step",
			expression: "* * * * *",
			field:      func(s *cronx.Schedule) cronx.Field { return s.Minute },
			expected:   1,
		},
		{
			name:       "field with range and step 5",
			expression: "0-59/5 * * * *",
			field:      func(s *cronx.Schedule) cronx.Field { return s.Minute },
			expected:   5,
		},
		{
			name:       "field with list containing step 10",
			expression: "*/10,*/20 * * * *",
			field:      func(s *cronx.Schedule) cronx.Field { return s.Minute },
			expected:   10, // Should return first step value
		},
		{
			name:       "field with step 30",
			expression: "*/30 * * * *",
			field:      func(s *cronx.Schedule) cronx.Field { return s.Minute },
			expected:   30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)
			require.NoError(t, err)
			field := tt.field(schedule)
			assert.Equal(t, tt.expected, field.Step())
		})
	}
}

// TestParseValue_SymbolParsing tests parseValue with symbol parsing
func TestParseValue_SymbolParsing(t *testing.T) {
	parser := cronx.NewParser()

	tests := []struct {
		name        string
		expression  string
		description string
	}{
		{
			name:        "day name in range",
			expression:  "0 0 * * MON-FRI",
			description: "Should parse MON and FRI as symbols",
		},
		{
			name:        "day name in list",
			expression:  "0 0 * * MON,WED,FRI",
			description: "Should parse day names in list",
		},
		{
			name:        "month name in range",
			expression:  "0 0 1 JAN-DEC *",
			description: "Should parse JAN and DEC as symbols",
		},
		{
			name:        "month name in list",
			expression:  "0 0 1 JAN,MAR,MAY *",
			description: "Should parse month names in list",
		},
		{
			name:        "day name with step",
			expression:  "0 0 * * */2",
			description: "Should handle step with numeric values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expression)
			require.NoError(t, err, tt.description)
			assert.NotNil(t, schedule)
		})
	}
}
