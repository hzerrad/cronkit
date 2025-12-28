package check

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		expected string
	}{
		{
			name:     "SeverityInfo",
			severity: SeverityInfo,
			expected: "info",
		},
		{
			name:     "SeverityWarn",
			severity: SeverityWarn,
			expected: "warn",
		},
		{
			name:     "SeverityError",
			severity: SeverityError,
			expected: "error",
		},
		{
			name:     "Invalid severity",
			severity: Severity(999),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.severity.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSeverityFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Severity
	}{
		{
			name:     "info",
			input:    "info",
			expected: SeverityInfo,
		},
		{
			name:     "warn",
			input:    "warn",
			expected: SeverityWarn,
		},
		{
			name:     "warning",
			input:    "warning",
			expected: SeverityWarn,
		},
		{
			name:     "error",
			input:    "error",
			expected: SeverityError,
		},
		{
			name:     "invalid string",
			input:    "invalid",
			expected: -1,
		},
		{
			name:     "empty string",
			input:    "",
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SeverityFromString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSeverity_IsError(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		expected bool
	}{
		{
			name:     "SeverityError",
			severity: SeverityError,
			expected: true,
		},
		{
			name:     "SeverityWarn",
			severity: SeverityWarn,
			expected: false,
		},
		{
			name:     "SeverityInfo",
			severity: SeverityInfo,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.severity.IsError()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSeverity_IsWarning(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		expected bool
	}{
		{
			name:     "SeverityWarn",
			severity: SeverityWarn,
			expected: true,
		},
		{
			name:     "SeverityError",
			severity: SeverityError,
			expected: false,
		},
		{
			name:     "SeverityInfo",
			severity: SeverityInfo,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.severity.IsWarning()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSeverity_IsInfo(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		expected bool
	}{
		{
			name:     "SeverityInfo",
			severity: SeverityInfo,
			expected: true,
		},
		{
			name:     "SeverityWarn",
			severity: SeverityWarn,
			expected: false,
		},
		{
			name:     "SeverityError",
			severity: SeverityError,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.severity.IsInfo()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSeverity_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		expected string
	}{
		{
			name:     "SeverityInfo",
			severity: SeverityInfo,
			expected: `"info"`,
		},
		{
			name:     "SeverityWarn",
			severity: SeverityWarn,
			expected: `"warn"`,
		},
		{
			name:     "SeverityError",
			severity: SeverityError,
			expected: `"error"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.severity)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))
		})
	}
}

func TestSeverity_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Severity
		wantErr  bool
	}{
		{
			name:     "info",
			input:    `"info"`,
			expected: SeverityInfo,
			wantErr:  false,
		},
		{
			name:     "warn",
			input:    `"warn"`,
			expected: SeverityWarn,
			wantErr:  false,
		},
		{
			name:     "error",
			input:    `"error"`,
			expected: SeverityError,
			wantErr:  false,
		},
		{
			name:     "invalid severity",
			input:    `"invalid"`,
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "invalid JSON",
			input:    `not json`,
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Severity
			err := json.Unmarshal([]byte(tt.input), &s)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, s)
			}
		})
	}
}

func TestParseFailOnLevel(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  Severity
		wantError bool
	}{
		{
			name:      "error lowercase",
			input:     "error",
			expected:  SeverityError,
			wantError: false,
		},
		{
			name:      "error uppercase",
			input:     "ERROR",
			expected:  SeverityError,
			wantError: false,
		},
		{
			name:      "error mixed case",
			input:     "Error",
			expected:  SeverityError,
			wantError: false,
		},
		{
			name:      "warn lowercase",
			input:     "warn",
			expected:  SeverityWarn,
			wantError: false,
		},
		{
			name:      "warn uppercase",
			input:     "WARN",
			expected:  SeverityWarn,
			wantError: false,
		},
		{
			name:      "warn mixed case",
			input:     "Warn",
			expected:  SeverityWarn,
			wantError: false,
		},
		{
			name:      "warning lowercase",
			input:     "warning",
			expected:  SeverityWarn,
			wantError: false,
		},
		{
			name:      "warning uppercase",
			input:     "WARNING",
			expected:  SeverityWarn,
			wantError: false,
		},
		{
			name:      "info lowercase",
			input:     "info",
			expected:  SeverityInfo,
			wantError: false,
		},
		{
			name:      "info uppercase",
			input:     "INFO",
			expected:  SeverityInfo,
			wantError: false,
		},
		{
			name:      "info mixed case",
			input:     "Info",
			expected:  SeverityInfo,
			wantError: false,
		},
		{
			name:      "invalid string",
			input:     "invalid",
			expected:  Severity(-1),
			wantError: true,
		},
		{
			name:      "empty string",
			input:     "",
			expected:  Severity(-1),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseFailOnLevel(tt.input)
			if tt.wantError {
				require.Error(t, err)
				assert.Equal(t, Severity(-1), result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
