package cronx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseValue tests decoding of numbers and symbols
func TestParseValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "numeric value",
			input:    "15",
			expected: 15,
		},
		{
			name:     "day name",
			input:    "MON",
			expected: 1,
		},
		{
			name:     "month name",
			input:    "DEC",
			expected: 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := parseValue(tt.input, DefaultSymbolRegistry)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, value)
		})
	}
}

// TestParseValue_Rejected tests that undecodable values are reported instead of defaulting to zero
func TestParseValue_Rejected(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "unknown symbol",
			input: "LUN",
		},
		{
			name:  "empty value",
			input: "",
		},
		{
			name:  "quartz last-day marker",
			input: "L",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseValue(tt.input, DefaultSymbolRegistry)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unrecognized value")
		})
	}
}

// TestParsePart_Rejected tests that malformed field components are reported
func TestParsePart_Rejected(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "non-numeric step",
			input:    "*/x",
			contains: "unrecognized step",
		},
		{
			name:     "repeated step",
			input:    "*/5/2",
			contains: "too many steps",
		},
		{
			name:     "repeated bounds",
			input:    "1-2-3",
			contains: "too many bounds",
		},
		{
			name:     "unknown symbol as range start",
			input:    "LUN-FRI",
			contains: "unrecognized value",
		},
		{
			name:     "unknown symbol as range end",
			input:    "MON-VEN",
			contains: "unrecognized value",
		},
		{
			name:     "unknown symbol as single value",
			input:    "LUN",
			contains: "unrecognized value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parsePart(tt.input, DefaultSymbolRegistry)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.contains)
		})
	}
}

// TestParseField_Rejected tests that a bad component fails the whole field
func TestParseField_Rejected(t *testing.T) {
	t.Run("bad value in a list", func(t *testing.T) {
		_, err := parseField("MON,LUN", MinDayOfWeek, MaxDayOfWeek, DefaultSymbolRegistry)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unrecognized value "LUN"`)
	})

	t.Run("good values are kept", func(t *testing.T) {
		f, err := parseField("MON,FRI", MinDayOfWeek, MaxDayOfWeek, DefaultSymbolRegistry)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 5}, f.ListValues())
	})
}
