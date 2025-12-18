package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExplainCommand_Registration(t *testing.T) {
	// Find the explain command
	explainCmd := findCommand(rootCmd, "explain")
	require.NotNil(t, explainCmd, "explain command should be registered")

	assert.Contains(t, explainCmd.Use, "explain", "Use should contain command name")
	assert.Contains(t, explainCmd.Short, "cron", "Short description should mention cron")
	assert.Contains(t, explainCmd.Long, "human-readable", "Long description should mention human-readable")
}

func TestExplainCommand_ValidExpressions(t *testing.T) {
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
			name:       "every 15 minutes",
			expression: "*/15 * * * *",
			expected:   "Every 15 minutes",
		},
		{
			name:       "weekdays at 9am",
			expression: "0 9 * * 1-5",
			expected:   "At 09:00 on weekdays (Mon-Fri)",
		},
		{
			name:       "daily alias",
			expression: "@daily",
			expected:   "At midnight every day",
		},
		{
			name:       "hourly alias",
			expression: "@hourly",
			expected:   "At the start of every hour",
		},
		{
			name:       "weekly alias",
			expression: "@weekly",
			expected:   "At midnight every Sunday",
		},
		{
			name:       "monthly alias",
			expression: "@monthly",
			expected:   "At midnight on the first day of every month",
		},
		{
			name:       "yearly alias",
			expression: "@yearly",
			expected:   "At midnight on January 1st",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := executeCommand(t, "explain", tt.expression)
			assert.Contains(t, output, tt.expected)
		})
	}
}

func TestExplainCommand_JSONFlag(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		flag       string
	}{
		{
			name:       "long flag",
			expression: "0 0 * * *",
			flag:       "--json",
		},
		{
			name:       "short flag",
			expression: "@daily",
			flag:       "-j",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := executeCommand(t, "explain", tt.expression, tt.flag)

			// Verify valid JSON
			var result map[string]string
			err := json.Unmarshal([]byte(output), &result)
			require.NoError(t, err, "Output should be valid JSON")

			// Verify required fields
			assert.Contains(t, result, "expression", "JSON should contain expression")
			assert.Contains(t, result, "description", "JSON should contain description")

			// Verify values
			assert.Equal(t, tt.expression, result["expression"])
			assert.NotEmpty(t, result["description"], "Description should not be empty")
		})
	}
}

func TestExplainCommand_InvalidExpressions(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		errorMsg   string
	}{
		{
			name:       "empty expression",
			expression: "",
			errorMsg:   "empty",
		},
		{
			name:       "invalid syntax",
			expression: "invalid",
			errorMsg:   "expected 5 fields",
		},
		{
			name:       "out of range minute",
			expression: "60 0 * * *",
			errorMsg:   "out of range",
		},
		{
			name:       "out of range hour",
			expression: "0 24 * * *",
			errorMsg:   "out of range",
		},
		{
			name:       "too many fields",
			expression: "0 0 * * * * extra",
			errorMsg:   "expected 5 fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeCommandWithError(t, "explain", tt.expression)
			require.Error(t, err, "Should return error for invalid expression")
			assert.Contains(t, strings.ToLower(output), strings.ToLower(tt.errorMsg),
				"Error message should be helpful")
		})
	}
}

func TestExplainCommand_NoArguments(t *testing.T) {
	_, err := executeCommandWithError(t, "explain")
	require.Error(t, err, "Should return error when no argument provided")
}

func TestExplainCommand_TooManyArguments(t *testing.T) {
	_, err := executeCommandWithError(t, "explain", "0 0 * * *", "extra")
	require.Error(t, err, "Should return error when too many arguments provided")
}

func TestExplainCommand_HelpfulErrors(t *testing.T) {
	output, err := executeCommandWithError(t, "explain", "invalid")
	require.Error(t, err)

	// Error should be user-friendly
	assert.NotContains(t, output, "panic", "Should not expose internal errors")
	assert.True(t, len(output) > 0, "Should provide error message")
}

// Helper functions

func findCommand(root *cobra.Command, name string) *cobra.Command {
	for _, cmd := range root.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}

func executeCommand(t *testing.T, args ...string) string {
	t.Helper()
	output, err := executeCommandWithError(t, args...)
	require.NoError(t, err)
	return output
}

func executeCommandWithError(t *testing.T, args ...string) (string, error) {
	t.Helper()

	// Create fresh command instances to avoid state pollution
	// Each test gets completely isolated commands with no shared state
	buf := new(bytes.Buffer)
	cmd := &cobra.Command{Use: "cronic"}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)

	// Add fresh command instances (not global singletons)
	cmd.AddCommand(newExplainCommand())
	cmd.AddCommand(newNextCommand())

	// Execute
	err := cmd.Execute()

	return buf.String(), err
}
