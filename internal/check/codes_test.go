package check

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCodeSeverity(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected Severity
	}{
		{
			name:     "DOM/DOW conflict",
			code:     CodeDOMDOWConflict,
			expected: SeverityWarn,
		},
		{
			name:     "Empty schedule",
			code:     CodeEmptySchedule,
			expected: SeverityError,
		},
		{
			name:     "Parse error",
			code:     CodeParseError,
			expected: SeverityError,
		},
		{
			name:     "File read error",
			code:     CodeFileReadError,
			expected: SeverityError,
		},
		{
			name:     "Invalid structure",
			code:     CodeInvalidStructure,
			expected: SeverityError,
		},
		{
			name:     "Unknown code",
			code:     "CRON-999",
			expected: SeverityError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCodeSeverity(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetCodeHint(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "DOM/DOW conflict",
			code:     CodeDOMDOWConflict,
			expected: "Consider using only day-of-month OR day-of-week, not both. Cron uses OR logic (runs if either condition is met).",
		},
		{
			name:     "Empty schedule",
			code:     CodeEmptySchedule,
			expected: "This expression never runs. Check for conflicting constraints or impossible date combinations.",
		},
		{
			name:     "Parse error",
			code:     CodeParseError,
			expected: "Fix the syntax error in the cron expression. Ensure all 5 fields are present and valid.",
		},
		{
			name:     "File read error",
			code:     CodeFileReadError,
			expected: "Check that the file exists and is readable. Verify file permissions.",
		},
		{
			name:     "Invalid structure",
			code:     CodeInvalidStructure,
			expected: "Ensure the crontab file follows the correct format with valid cron expressions.",
		},
		{
			name:     "Unknown code",
			code:     "CRON-999",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCodeHint(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDiagnosticCodeConstants(t *testing.T) {
	// Verify all constants are defined and non-empty
	assert.NotEmpty(t, CodeDOMDOWConflict)
	assert.NotEmpty(t, CodeEmptySchedule)
	assert.NotEmpty(t, CodeParseError)
	assert.NotEmpty(t, CodeFileReadError)
	assert.NotEmpty(t, CodeInvalidStructure)

	// Verify format is CRON-XXX
	assert.Contains(t, CodeDOMDOWConflict, "CRON-")
	assert.Contains(t, CodeEmptySchedule, "CRON-")
	assert.Contains(t, CodeParseError, "CRON-")
	assert.Contains(t, CodeFileReadError, "CRON-")
	assert.Contains(t, CodeInvalidStructure, "CRON-")
}
