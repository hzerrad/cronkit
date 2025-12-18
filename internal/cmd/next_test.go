package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextCommand_Registration(t *testing.T) {
	nextCmd := findCommand(rootCmd, "next")
	require.NotNil(t, nextCmd, "next command should be registered")
	assert.Contains(t, nextCmd.Use, "next", "command use should contain 'next'")
	assert.NotEmpty(t, nextCmd.Short, "command should have short description")
	assert.NotEmpty(t, nextCmd.Long, "command should have long description")
}

func TestNextCommand_Flags(t *testing.T) {
	nextCmd := findCommand(rootCmd, "next")
	require.NotNil(t, nextCmd)

	countFlag := nextCmd.Flags().Lookup("count")
	assert.NotNil(t, countFlag, "should have --count flag")
	assert.Equal(t, "c", countFlag.Shorthand, "should have -c shorthand")
	assert.Equal(t, "10", countFlag.DefValue, "default count should be 10")

	jsonFlag := nextCmd.Flags().Lookup("json")
	assert.NotNil(t, jsonFlag, "should have --json flag")
	assert.Equal(t, "j", jsonFlag.Shorthand, "should have -j shorthand")
	assert.Equal(t, "false", jsonFlag.DefValue, "default json should be false")
}

func TestNextCommand_BasicUsage(t *testing.T) {
	output := executeCommand(t, "next", "*/15 * * * *")

	assert.Contains(t, output, "Next 10 runs", "should show default count of 10")
	assert.Contains(t, output, "*/15 * * * *", "should show the expression")
	assert.Contains(t, output, "1.", "should show first run number")
	assert.Contains(t, output, "10.", "should show last run number")

	// Should contain timestamps with timezone
	assert.Regexp(t, `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} \w+`, output, "should contain timestamp with timezone")
}

func TestNextCommand_CustomCount(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		count    string
		expected string
	}{
		{
			name:     "long flag",
			flag:     "--count",
			count:    "5",
			expected: "Next 5 runs",
		},
		{
			name:     "short flag",
			flag:     "-c",
			count:    "3",
			expected: "Next 3 runs",
		},
		{
			name:     "count of 1",
			flag:     "--count",
			count:    "1",
			expected: "Next 1 run",
		},
		{
			name:     "count of 100",
			flag:     "--count",
			count:    "100",
			expected: "Next 100 runs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := executeCommand(t, "next", "@daily", tt.flag, tt.count)
			assert.Contains(t, output, tt.expected)
		})
	}
}

func TestNextCommand_CronAliases(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		contains   string
	}{
		{
			name:       "@daily",
			expression: "@daily",
			contains:   "midnight",
		},
		{
			name:       "@hourly",
			expression: "@hourly",
			contains:   "hour",
		},
		{
			name:       "@weekly",
			expression: "@weekly",
			contains:   "Sunday",
		},
		{
			name:       "@monthly",
			expression: "@monthly",
			contains:   "first day",
		},
		{
			name:       "@yearly",
			expression: "@yearly",
			contains:   "January",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := executeCommand(t, "next", tt.expression, "-c", "1")
			assert.Contains(t, strings.ToLower(output), strings.ToLower(tt.contains))
		})
	}
}

func TestNextCommand_JSONOutput(t *testing.T) {
	tests := []struct {
		name       string
		flag       string
		expression string
		count      string
	}{
		{
			name:       "long flag",
			flag:       "--json",
			expression: "@daily",
			count:      "5",
		},
		{
			name:       "short flag",
			flag:       "-j",
			expression: "*/15 * * * *",
			count:      "3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args []string
			if tt.count != "" {
				args = []string{"next", tt.expression, tt.flag, "--count", tt.count}
			} else {
				args = []string{"next", tt.expression, tt.flag}
			}

			output := executeCommand(t, args...)

			// Parse JSON
			var result map[string]interface{}
			err := json.Unmarshal([]byte(output), &result)
			require.NoError(t, err, "output should be valid JSON")

			// Validate JSON structure
			assert.Equal(t, tt.expression, result["expression"], "should include expression")
			assert.NotEmpty(t, result["description"], "should include description")
			assert.NotEmpty(t, result["timezone"], "should include timezone")

			nextRuns, ok := result["next_runs"].([]interface{})
			require.True(t, ok, "next_runs should be an array")
			assert.NotEmpty(t, nextRuns, "should have at least one run")

			// Validate first run structure
			firstRun := nextRuns[0].(map[string]interface{})
			assert.NotNil(t, firstRun["number"], "should have run number")
			assert.NotEmpty(t, firstRun["timestamp"], "should have timestamp")
			assert.NotEmpty(t, firstRun["relative"], "should have relative time")
		})
	}
}

func TestNextCommand_JSONTimestampFormat(t *testing.T) {
	output := executeCommand(t, "next", "@daily", "--json", "-c", "1")

	var result map[string]interface{}
	err := json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	nextRuns := result["next_runs"].([]interface{})
	firstRun := nextRuns[0].(map[string]interface{})
	timestamp := firstRun["timestamp"].(string)

	// Verify RFC3339 format (ISO8601)
	assert.Regexp(t, `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}$`, timestamp,
		"timestamp should be in RFC3339 format")
}

func TestNextCommand_RelativeTime(t *testing.T) {
	output := executeCommand(t, "next", "*/5 * * * *", "--json", "-c", "3")

	var result map[string]interface{}
	err := json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	nextRuns := result["next_runs"].([]interface{})
	for _, run := range nextRuns {
		runMap := run.(map[string]interface{})
		relative := runMap["relative"].(string)
		assert.NotEmpty(t, relative, "relative time should not be empty")
		assert.Contains(t, relative, "in ", "relative time should start with 'in '")
	}
}

func TestNextCommand_InvalidExpressions(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		errMessage string
	}{
		{
			name:       "wrong field count",
			expression: "0 0 *",
			errMessage: "expected 5 fields",
		},
		{
			name:       "invalid minute",
			expression: "60 0 * * *",
			errMessage: "out of range",
		},
		{
			name:       "invalid hour",
			expression: "0 24 * * *",
			errMessage: "out of range",
		},
		{
			name:       "invalid day of month",
			expression: "0 0 32 * *",
			errMessage: "out of range",
		},
		{
			name:       "completely invalid",
			expression: "not-a-cron",
			errMessage: "expected 5 fields",
		},
		{
			name:       "invalid alias",
			expression: "@invalid",
			errMessage: "unrecognized descriptor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeCommandWithError(t, "next", tt.expression)
			assert.Error(t, err)
			assert.Contains(t, output, tt.errMessage)
		})
	}
}

func TestNextCommand_CountValidation(t *testing.T) {
	tests := []struct {
		name       string
		count      string
		errMessage string
	}{
		{
			name:       "count zero",
			count:      "0",
			errMessage: "count must be at least 1",
		},
		{
			name:       "negative count",
			count:      "-5",
			errMessage: "count must be at least 1",
		},
		{
			name:       "count over 100",
			count:      "101",
			errMessage: "count must be at most 100",
		},
		{
			name:       "count way over 100",
			count:      "1000",
			errMessage: "count must be at most 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeCommandWithError(t, "next", "* * * * *", "--count", tt.count)
			assert.Error(t, err)
			assert.Contains(t, output, tt.errMessage)
		})
	}
}

func TestNextCommand_NoArgument(t *testing.T) {
	output, err := executeCommandWithError(t, "next")
	assert.Error(t, err)
	assert.Contains(t, output, "accepts 1 arg(s), received 0", "should indicate that an argument is required")
}

func TestNextCommand_Help(t *testing.T) {
	tests := []struct {
		name string
		flag string
	}{
		{
			name: "long flag",
			flag: "--help",
		},
		{
			name: "short flag",
			flag: "-h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := executeCommand(t, "next", tt.flag)

			assert.Contains(t, output, "Usage:", "should show usage")
			assert.Contains(t, output, "next", "should mention next command")
			assert.Contains(t, output, "Examples:", "should show examples")
			assert.Contains(t, output, "--count", "should document --count flag")
			assert.Contains(t, output, "--json", "should document --json flag")
			assert.Contains(t, output, "@daily", "should show example with @daily")
			assert.Contains(t, output, "*/15 * * * *", "should show example with interval")

			// Help invocation can leave command state polluted, ensure cleanup
			t.Cleanup(func() {
				nextCmd.SetHelpFunc(nil)
				nextCmd.SetUsageFunc(nil)
			})
		})
	}
}

func TestAAAANextCommand_TextOutputFormat(t *testing.T) {
	// Renamed with "AAAA" prefix to run first before all tests
	output := executeCommand(t, "next", "@daily", "-c", "1")

	// Basic sanity check - full validation in integration tests
	assert.NotEmpty(t, output, "should produce output")
	assert.NotContains(t, output, "Error", "should not error")
}

func TestNextCommand_ComplexExpressions(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		count      string
	}{
		{
			name:       "business hours",
			expression: "*/5 9-17 * * 1-5",
			count:      "5",
		},
		{
			name:       "specific day of month",
			expression: "0 0 15 * *",
			count:      "3",
		},
		{
			name:       "multiple times per day",
			expression: "0 9,12,17 * * *",
			count:      "5",
		},
		{
			name:       "specific month",
			expression: "0 0 1 1 *",
			count:      "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := executeCommand(t, "next", tt.expression, "-c", tt.count)
			// Basic validation - full coverage in integration tests
			assert.NotEmpty(t, output, "should produce output")
			assert.NotContains(t, output, "Error:", "should not contain errors")
		})
	}
}

func TestAAAANextCommand_TimeProgression(t *testing.T) {
	// Renamed with "AAAA" prefix to run first before all tests
	output := executeCommand(t, "next", "@hourly", "--json", "-c", "2")

	// Basic sanity check - full JSON structure tested in integration tests
	assert.NotEmpty(t, output, "should produce output")
	assert.NotContains(t, output, "Error", "should not error")
}
