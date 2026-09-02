package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hzerrad/cronkit/internal/check"
	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCommand(t *testing.T) {
	t.Run("check command should be registered", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"check"})
		assert.NoError(t, err)
		assert.Equal(t, "check", cmd.Name())
	})

	t.Run("check command should have metadata", func(t *testing.T) {
		cc := newCheckCommand()
		assert.NotEmpty(t, cc.Short)
		assert.NotEmpty(t, cc.Long)
		assert.Contains(t, cc.Use, "check")
	})

	t.Run("check valid expression", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"0 0 * * *"})

		// Don't let os.Exit kill the test
		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "All valid")
	})

	t.Run("check invalid expression", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetErr(buf)
		cc.SetArgs([]string{"60 0 * * *"})

		// Don't let os.Exit kill the test
		oldExit := osExit
		exitCode := 0
		osExit = func(code int) { exitCode = code }
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		// os.Exit should have been called with code 1
		assert.Equal(t, 1, exitCode, "Should exit with code 1 for errors")
		// Output should contain error information
		output := buf.String()
		assert.Contains(t, output, "error", "Should show errors in output")
		// Error should be nil because os.Exit was called
		assert.NoError(t, err)
	})

	t.Run("check expression with DOM/DOW conflict", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"0 0 1 * 1"})

		// Don't let os.Exit kill the test
		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		require.NoError(t, err)
		// Warnings are now shown by default (compact format)
		output := buf.String()
		assert.Contains(t, output, "warning")
	})

	t.Run("check expression with DOM/DOW conflict verbose", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"0 0 1 * 1", "--verbose"})

		// Don't let os.Exit kill the test
		oldExit := osExit
		exitCode := 0
		osExit = func(code int) { exitCode = code }
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		// With verbose, warnings should show
		output := buf.String()
		if err == nil {
			// If no error, check exit code
			if exitCode == 0 {
				assert.Contains(t, output, "warning", "Should show warnings with --verbose")
			}
		}
	})

	t.Run("check with JSON output", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"0 0 * * *", "--json"})

		// Don't let os.Exit kill the test
		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		assert.True(t, result["valid"].(bool))
		assert.Equal(t, float64(1), result["totalJobs"])
	})

	t.Run("check crontab file", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		testFile := filepath.Join("..", "..", "testdata", "crontab", "sample.cron")
		cc.SetArgs([]string{"--file", testFile})

		// Don't let os.Exit kill the test
		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		require.NoError(t, err)
		// Should not error on valid file
		assert.NotEmpty(t, buf.String())
	})

	t.Run("check invalid crontab file", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetErr(buf)
		testFile := filepath.Join("..", "..", "testdata", "crontab", "invalid.cron")
		cc.SetArgs([]string{"--file", testFile})

		// Don't let os.Exit kill the test
		oldExit := osExit
		exitCode := 0
		osExit = func(code int) { exitCode = code }
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		// Should have errors
		if err == nil {
			assert.Equal(t, 1, exitCode, "Should exit with code 1 for errors")
		}
		assert.Contains(t, buf.String(), "error")
	})

	t.Run("check non-existent file", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetErr(buf)
		cc.SetArgs([]string{"--file", "nonexistent.cron"})

		// Don't let os.Exit kill the test
		oldExit := osExit
		exitCode := 0
		osExit = func(code int) { exitCode = code }
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		// Should error on non-existent file
		if err == nil {
			assert.Equal(t, 1, exitCode, "Should exit with code 1 for errors")
		}
		assert.Contains(t, buf.String(), "Failed to read")
	})

	t.Run("check JSON output with warnings", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"0 0 1 * 1", "--json", "--verbose"})

		// Don't let os.Exit kill the test
		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		// Should have issues array with warnings
		issues, ok := result["issues"].([]interface{})
		if ok && len(issues) > 0 {
			issue := issues[0].(map[string]interface{})
			assert.Equal(t, "warn", issue["severity"])
			assert.Equal(t, check.CodeDOMDOWConflict, issue["code"])
			assert.Contains(t, issue, "hint")
		}
	})

	t.Run("check with no arguments (user crontab)", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{})

		// Don't let os.Exit kill the test
		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		// Should not error (even if no crontab exists)
		assert.NoError(t, err)
	})

	t.Run("check JSON output with exit code 2 for warnings", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"0 0 1 * 1", "--json", "--verbose"})

		// Don't let os.Exit kill the test
		oldExit := osExit
		exitCode := 0
		osExit = func(code int) { exitCode = code }
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		require.NoError(t, err)
		// Warnings should not cause exit code unless --fail-on is set
		assert.Equal(t, 0, exitCode, "Warnings should not cause exit with --fail-on error")

		var result map[string]interface{}
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)
	})

	t.Run("check JSON output with valid result", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"0 0 * * *", "--json"})

		// Don't let os.Exit kill the test
		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)
		assert.True(t, result["valid"].(bool))
		assert.Equal(t, float64(1), result["totalJobs"])
	})

	t.Run("check text output with warnings and verbose", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"0 0 1 * 1", "--verbose"})

		// Don't let os.Exit kill the test
		oldExit := osExit
		exitCode := 0
		osExit = func(code int) { exitCode = code }
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		require.NoError(t, err)
		// Warnings should not cause exit code unless --fail-on is set
		assert.Equal(t, 0, exitCode, "Warnings should not cause exit with --fail-on error")
		assert.Contains(t, buf.String(), "warning")
	})

	t.Run("check text output with errors", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetErr(buf)
		cc.SetArgs([]string{"60 0 * * *"})

		// Don't let os.Exit kill the test
		oldExit := osExit
		exitCode := 0
		osExit = func(code int) { exitCode = code }
		defer func() { osExit = oldExit }()

		_ = cc.Execute()
		// Should exit with code 1 for errors
		assert.Equal(t, 1, exitCode, "Should exit with code 1 for errors")
		assert.Contains(t, buf.String(), "error")
	})

	t.Run("check text output with valid result and jobs", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"0 0 * * *"})

		// Don't let os.Exit kill the test
		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "All valid")
		assert.Contains(t, buf.String(), "job(s) validated")
	})

	t.Run("check text output with warnings only", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.verbose = true

		// Create result with warnings but valid
		result := check.ValidationResult{
			Valid:     true,
			TotalJobs: 1,
			ValidJobs: 1,
			Issues: []check.Issue{
				{
					Severity:   check.SeverityWarn,
					Code:       check.CodeDOMDOWConflict,
					LineNumber: 0,
					Expression: "0 0 1 * 1",
					Message:    "Both day-of-month and day-of-week specified",
				},
			},
		}

		// Don't let os.Exit kill the test
		oldExit := osExit
		exitCode := 0
		osExit = func(code int) { exitCode = code }
		defer func() { osExit = oldExit }()

		err := cc.outputText(result, check.SeverityError, false)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "warning")
		assert.Equal(t, 0, exitCode, "Warnings should not cause exit with --fail-on error")
	})

	t.Run("check text output with no jobs", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)

		// Create result with no jobs
		result := check.ValidationResult{
			Valid:     true,
			TotalJobs: 0,
			ValidJobs: 0,
			Issues:    []check.Issue{},
		}

		// Don't let os.Exit kill the test
		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.outputText(result, check.SeverityError, false)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "All valid")
		// Should not show job count when 0
		assert.NotContains(t, buf.String(), "0 job(s)")
	})

	t.Run("check text output with line number in issue", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)

		// Create result with issue that has line number
		result := check.ValidationResult{
			Valid:       false,
			TotalJobs:   1,
			ValidJobs:   0,
			InvalidJobs: 1,
			Issues: []check.Issue{
				{
					Severity:   check.SeverityError,
					Code:       check.CodeParseError,
					LineNumber: 5,
					Expression: "60 0 * * *",
					Message:    "Invalid cron expression",
				},
			},
		}

		// Don't let os.Exit kill the test
		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.outputText(result, check.SeverityError, false)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "Line 5")
		assert.Contains(t, buf.String(), "ERROR")
	})

	t.Run("check text output with issue without expression", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)

		// Create result with issue that has no expression
		result := check.ValidationResult{
			Valid:     false,
			TotalJobs: 0,
			Issues: []check.Issue{
				{
					Severity:   check.SeverityError,
					Code:       check.CodeParseError,
					LineNumber: 0,
					Expression: "",
					Message:    "Failed to read crontab file",
				},
			},
		}

		// Don't let os.Exit kill the test
		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.outputText(result, check.SeverityError, false)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "ERROR")
		assert.Contains(t, buf.String(), "Failed to read")
		// Should not show "Expression:" when expression is empty
		assert.NotContains(t, buf.String(), "Expression:")
	})

	t.Run("check text output with info type issue", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.verbose = true

		// Create result with info type issue
		result := check.ValidationResult{
			Valid:     true,
			TotalJobs: 1,
			ValidJobs: 1,
			Issues: []check.Issue{
				{
					Severity:   check.SeverityInfo,
					Code:       "",
					LineNumber: 1,
					Expression: "0 0 * * *",
					Message:    "Info message",
				},
			},
		}

		// Don't let os.Exit kill the test
		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.outputText(result, check.SeverityError, false)
		require.NoError(t, err)
		// Info issues are only shown when verbose
		assert.Contains(t, buf.String(), "INFO")
		assert.Contains(t, buf.String(), "Info message")
	})

	t.Run("check text output with valid result but no jobs", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)

		// Create result with valid but no jobs
		result := check.ValidationResult{
			Valid:     true,
			TotalJobs: 0,
			ValidJobs: 0,
			Issues:    []check.Issue{},
		}

		// Don't let os.Exit kill the test
		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.outputText(result, check.SeverityError, false)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "All valid")
		// Should not show job count when 0
		assert.NotContains(t, buf.String(), "0 job(s)")
	})

	t.Run("check with --fail-on error (default)", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"0 0 1 * 1"}) // Has warning

		// Don't let os.Exit kill the test
		oldExit := osExit
		exitCode := 0
		osExit = func(code int) { exitCode = code }
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		require.NoError(t, err)
		// With default --fail-on error, warnings don't cause exit
		assert.Equal(t, 0, exitCode, "Should exit with code 0 for warnings with default --fail-on error")
		// Warnings are now shown by default (compact format)
		assert.Contains(t, buf.String(), "warning")
	})

	t.Run("check with --fail-on warn", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"0 0 1 * 1", "--fail-on", "warn", "--verbose"})

		// Don't let os.Exit kill the test
		oldExit := osExit
		exitCode := 0
		osExit = func(code int) { exitCode = code }
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		require.NoError(t, err)
		// With --fail-on warn, warnings cause exit code 2
		assert.Equal(t, 2, exitCode, "Should exit with code 2 for warnings with --fail-on warn")
		assert.Contains(t, buf.String(), "warning")
	})

	t.Run("check with --fail-on warn and errors", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"60 0 * * *", "--fail-on", "warn"})

		// Don't let os.Exit kill the test
		oldExit := osExit
		exitCode := 0
		osExit = func(code int) { exitCode = code }
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		require.NoError(t, err)
		// With --fail-on warn, errors cause exit code 1 (not 2)
		assert.Equal(t, 1, exitCode, "Should exit with code 1 for errors even with --fail-on warn")
	})

	t.Run("check with invalid --fail-on value", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetErr(buf)
		cc.SetArgs([]string{"0 0 * * *", "--fail-on", "invalid"})

		// Don't let os.Exit kill the test
		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid --fail-on value")
	})

	t.Run("calculateExitCode with no issues", func(t *testing.T) {
		result := check.ValidationResult{Valid: true, Issues: []check.Issue{}}
		exitCode := calculateExitCode(result, []check.Issue{}, check.SeverityError)
		assert.Equal(t, 0, exitCode)
	})

	t.Run("calculateExitCode with errors and fail-on error", func(t *testing.T) {
		result := check.ValidationResult{
			Valid: false,
			Issues: []check.Issue{
				{Severity: check.SeverityError, Code: check.CodeParseError},
			},
		}
		exitCode := calculateExitCode(result, result.Issues, check.SeverityError)
		assert.Equal(t, 1, exitCode)
	})

	t.Run("calculateExitCode with warnings and fail-on error", func(t *testing.T) {
		result := check.ValidationResult{
			Valid: true,
			Issues: []check.Issue{
				{Severity: check.SeverityWarn, Code: check.CodeDOMDOWConflict},
			},
		}
		exitCode := calculateExitCode(result, result.Issues, check.SeverityError)
		assert.Equal(t, 0, exitCode, "Warnings should not cause exit with --fail-on error")
	})

	t.Run("calculateExitCode with warnings and fail-on warn", func(t *testing.T) {
		result := check.ValidationResult{
			Valid: true,
			Issues: []check.Issue{
				{Severity: check.SeverityWarn, Code: check.CodeDOMDOWConflict},
			},
		}
		exitCode := calculateExitCode(result, result.Issues, check.SeverityWarn)
		assert.Equal(t, 2, exitCode)
	})

	t.Run("calculateExitCode with warnings and fail-on error (should not exit)", func(t *testing.T) {
		result := check.ValidationResult{
			Valid: true,
			Issues: []check.Issue{
				{Severity: check.SeverityWarn, Code: check.CodeDOMDOWConflict},
			},
		}
		exitCode := calculateExitCode(result, result.Issues, check.SeverityError)
		assert.Equal(t, 0, exitCode, "Warnings should not cause exit with --fail-on error")
	})

	t.Run("calculateExitCode with errors and fail-on warn", func(t *testing.T) {
		result := check.ValidationResult{
			Valid: false,
			Issues: []check.Issue{
				{Severity: check.SeverityError, Code: check.CodeParseError},
			},
		}
		exitCode := calculateExitCode(result, result.Issues, check.SeverityWarn)
		assert.Equal(t, 1, exitCode, "Errors should cause exit code 1 even with --fail-on warn")
	})

	t.Run("calculateExitCode with info and fail-on info", func(t *testing.T) {
		result := check.ValidationResult{
			Valid: true,
			Issues: []check.Issue{
				{Severity: check.SeverityInfo, Code: ""},
			},
		}
		exitCode := calculateExitCode(result, result.Issues, check.SeverityInfo)
		assert.Equal(t, 2, exitCode)
	})

	t.Run("calculateExitCode with mixed severities", func(t *testing.T) {
		result := check.ValidationResult{
			Valid: false,
			Issues: []check.Issue{
				{Severity: check.SeverityWarn, Code: check.CodeDOMDOWConflict},
				{Severity: check.SeverityError, Code: check.CodeParseError},
			},
		}
		exitCode := calculateExitCode(result, result.Issues, check.SeverityError)
		assert.Equal(t, 1, exitCode, "Should return 1 for errors when mixed with warnings")
	})

	t.Run("calculateExitCode default case", func(t *testing.T) {
		result := check.ValidationResult{
			Valid: true,
			Issues: []check.Issue{
				{Severity: check.Severity(999), Code: ""}, // Invalid severity
			},
		}
		exitCode := calculateExitCode(result, result.Issues, check.SeverityError)
		assert.Equal(t, 0, exitCode, "Should return 0 for invalid severity")
	})

	t.Run("check with --group-by severity", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.groupBy = "severity"
		cc.verbose = true

		result := check.ValidationResult{
			Valid:       false,
			TotalJobs:   3,
			ValidJobs:   1,
			InvalidJobs: 2,
			Issues: []check.Issue{
				{Severity: check.SeverityWarn, Code: check.CodeDOMDOWConflict, Message: "Warning 1"},
				{Severity: check.SeverityError, Code: check.CodeParseError, Message: "Error 1"},
				{Severity: check.SeverityWarn, Code: check.CodeDOMDOWConflict, Message: "Warning 2"},
			},
		}

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.outputText(result, check.SeverityError, false)
		require.NoError(t, err)
		output := buf.String()
		// Should have severity group headers
		assert.Contains(t, output, "error Issues")
		assert.Contains(t, output, "warn Issues")
		// Should have grouped issues
		assert.Contains(t, output, "Error 1")
		assert.Contains(t, output, "Warning 1")
		assert.Contains(t, output, "Warning 2")
	})

	t.Run("check with --group-by line", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.groupBy = "line"
		cc.verbose = true

		result := check.ValidationResult{
			Valid:       false,
			TotalJobs:   2,
			ValidJobs:   0,
			InvalidJobs: 2,
			Issues: []check.Issue{
				{Severity: check.SeverityError, Code: check.CodeParseError, LineNumber: 5, Message: "Error on line 5"},
				{Severity: check.SeverityError, Code: check.CodeParseError, LineNumber: 3, Message: "Error on line 3"},
				{Severity: check.SeverityError, Code: check.CodeParseError, LineNumber: 0, Message: "General error"},
			},
		}

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.outputText(result, check.SeverityError, false)
		require.NoError(t, err)
		output := buf.String()
		// Should have line group headers
		assert.Contains(t, output, "Line 3")
		assert.Contains(t, output, "Line 5")
		assert.Contains(t, output, "General Issues")
		// Should have grouped issues
		assert.Contains(t, output, "Error on line 3")
		assert.Contains(t, output, "Error on line 5")
		assert.Contains(t, output, "General error")
	})

	t.Run("check with --group-by job", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.groupBy = "job"
		cc.verbose = true

		result := check.ValidationResult{
			Valid:       false,
			TotalJobs:   2,
			ValidJobs:   0,
			InvalidJobs: 2,
			Issues: []check.Issue{
				{Severity: check.SeverityError, Code: check.CodeParseError, Expression: "0 0 * * *", Message: "Error 1"},
				{Severity: check.SeverityError, Code: check.CodeParseError, Expression: "0 0 * * *", Message: "Error 2"},
				{Severity: check.SeverityError, Code: check.CodeParseError, Expression: "", Message: "General error"},
			},
		}

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.outputText(result, check.SeverityError, false)
		require.NoError(t, err)
		output := buf.String()
		// Should have expression group headers
		assert.Contains(t, output, "Expression: 0 0 * * *")
		assert.Contains(t, output, "General Issues")
		// Should have grouped issues
		assert.Contains(t, output, "Error 1")
		assert.Contains(t, output, "Error 2")
		assert.Contains(t, output, "General error")
	})

	t.Run("check with --group-by none (default)", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.groupBy = "none"

		result := check.ValidationResult{
			Valid:       false,
			TotalJobs:   1,
			ValidJobs:   0,
			InvalidJobs: 1,
			Issues: []check.Issue{
				{Severity: check.SeverityError, Code: check.CodeParseError, Message: "Error 1"},
			},
		}

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.outputText(result, check.SeverityError, false)
		require.NoError(t, err)
		output := buf.String()
		// Should not have group headers
		assert.NotContains(t, output, "━━━")
		// Should have flat display
		assert.Contains(t, output, "Error 1")
	})

	t.Run("parseGroupBy with valid values", func(t *testing.T) {
		assert.Equal(t, GroupByNone, parseGroupBy("none"))
		assert.Equal(t, GroupBySeverity, parseGroupBy("severity"))
		assert.Equal(t, GroupByLine, parseGroupBy("line"))
		assert.Equal(t, GroupByJob, parseGroupBy("job"))
		assert.Equal(t, GroupByNone, parseGroupBy("invalid"))
	})

	t.Run("groupIssues by severity", func(t *testing.T) {
		issues := []check.Issue{
			{Severity: check.SeverityError, Message: "Error 1"},
			{Severity: check.SeverityWarn, Message: "Warning 1"},
			{Severity: check.SeverityError, Message: "Error 2"},
		}
		groups := groupIssues(issues, GroupBySeverity)
		assert.Equal(t, 2, len(groups["error"]))
		assert.Equal(t, 1, len(groups["warn"]))
	})

	t.Run("groupIssues by line", func(t *testing.T) {
		issues := []check.Issue{
			{Severity: check.SeverityError, LineNumber: 5, Message: "Error 1"},
			{Severity: check.SeverityError, LineNumber: 3, Message: "Error 2"},
			{Severity: check.SeverityError, LineNumber: 0, Message: "Error 3"},
		}
		groups := groupIssues(issues, GroupByLine)
		assert.Equal(t, 1, len(groups["line-5"]))
		assert.Equal(t, 1, len(groups["line-3"]))
		assert.Equal(t, 1, len(groups["no-line"]))
	})

	t.Run("groupIssues by job", func(t *testing.T) {
		issues := []check.Issue{
			{Severity: check.SeverityError, Expression: "0 0 * * *", Message: "Error 1"},
			{Severity: check.SeverityError, Expression: "0 0 * * *", Message: "Error 2"},
			{Severity: check.SeverityError, Expression: "", Message: "Error 3"},
		}
		groups := groupIssues(issues, GroupByJob)
		assert.Equal(t, 2, len(groups["0 0 * * *"]))
		assert.Equal(t, 1, len(groups["no-expression"]))
	})
}

// errorWriter is a writer that always returns an error
type errorWriter struct{}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return 0, io.ErrClosedPipe
}

func TestCheckCommand_OutputJSON_Error(t *testing.T) {
	cc := newCheckCommand()
	// Use an error writer to trigger JSON encoding error
	cc.SetOut(&errorWriter{})

	// Create a valid result
	result := check.ValidationResult{
		Valid:     true,
		TotalJobs: 1,
		ValidJobs: 1,
		Issues:    []check.Issue{},
	}

	// Don't let os.Exit kill the test
	oldExit := osExit
	osExit = func(code int) {}
	defer func() { osExit = oldExit }()

	err := cc.outputJSON(result, check.SeverityError, false)
	// Should return error from JSON encoding
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to encode JSON")
}

func TestCheckCommand_OutputJSON_WithIssues(t *testing.T) {
	cc := newCheckCommand()
	buf := new(bytes.Buffer)
	cc.SetOut(buf)
	cc.SetArgs([]string{"0 0 * * *", "--json"})

	// Don't let os.Exit kill the test
	oldExit := osExit
	osExit = func(code int) {}
	defer func() { osExit = oldExit }()

	err := cc.Execute()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.True(t, result["valid"].(bool))
}

func TestCheckCommand_OutputJSON_ExitCode2(t *testing.T) {
	cc := newCheckCommand()
	buf := new(bytes.Buffer)
	cc.SetOut(buf)
	cc.verbose = true

	// Create result with warnings
	result := check.ValidationResult{
		Valid:     true,
		TotalJobs: 1,
		ValidJobs: 1,
		Issues: []check.Issue{
			{
				Severity:   check.SeverityWarn,
				Code:       check.CodeDOMDOWConflict,
				LineNumber: 0,
				Expression: "0 0 1 * 1",
				Message:    "Both day-of-month and day-of-week specified",
				Hint:       check.GetCodeHint(check.CodeDOMDOWConflict),
			},
		},
	}

	// Don't let os.Exit kill the test
	oldExit := osExit
	exitCode := 0
	osExit = func(code int) { exitCode = code }
	defer func() { osExit = oldExit }()

	err := cc.outputJSON(result, check.SeverityError, false)
	require.NoError(t, err)
	// Warnings should not cause exit code unless --fail-on is set
	assert.Equal(t, 0, exitCode, "Warnings should not cause exit with --fail-on error")
}

func TestCheckCommand_OutputJSON_WithWarningsButNotVerbose(t *testing.T) {
	cc := newCheckCommand()
	buf := new(bytes.Buffer)
	cc.SetOut(buf)
	cc.verbose = false

	// Create result with warnings but not verbose
	result := check.ValidationResult{
		Valid:     true,
		TotalJobs: 1,
		ValidJobs: 1,
		Issues: []check.Issue{
			{
				Severity:   check.SeverityWarn,
				Code:       check.CodeDOMDOWConflict,
				LineNumber: 0,
				Expression: "0 0 1 * 1",
				Message:    "Both day-of-month and day-of-week specified",
				Hint:       check.GetCodeHint(check.CodeDOMDOWConflict),
			},
		},
	}

	// Don't let os.Exit kill the test
	oldExit := osExit
	exitCode := 0
	osExit = func(code int) { exitCode = code }
	defer func() { osExit = oldExit }()

	err := cc.outputJSON(result, check.SeverityError, false)
	require.NoError(t, err)
	// Warnings should not cause exit code unless --fail-on is set
	assert.Equal(t, 0, exitCode, "Warnings should not cause exit with --fail-on error")

	var output map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)
	// Warnings are now shown by default (not filtered out)
	issues := output["issues"].([]interface{})
	assert.Equal(t, 1, len(issues), "Warnings should be shown by default")
}

func TestCheckCommand_OutputJSON_ValidWithNoIssues(t *testing.T) {
	cc := newCheckCommand()
	buf := new(bytes.Buffer)
	cc.SetOut(buf)

	// Create result with valid and no issues
	result := check.ValidationResult{
		Valid:     true,
		TotalJobs: 1,
		ValidJobs: 1,
		Issues:    []check.Issue{},
	}

	// Don't let os.Exit kill the test
	oldExit := osExit
	osExit = func(code int) {}
	defer func() { osExit = oldExit }()

	err := cc.outputJSON(result, check.SeverityError, false)
	require.NoError(t, err)

	var output map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)
	// Should be valid when no issues
	assert.True(t, output["valid"].(bool))
	assert.Equal(t, float64(1), output["totalJobs"])
}

func TestCheckCommand_OutputJSON_InvalidResult(t *testing.T) {
	cc := newCheckCommand()
	buf := new(bytes.Buffer)
	cc.SetOut(buf)

	// Create result with invalid (has errors)
	result := check.ValidationResult{
		Valid:       false,
		TotalJobs:   1,
		ValidJobs:   0,
		InvalidJobs: 1,
		Issues: []check.Issue{
			{
				Severity:   check.SeverityError,
				Code:       check.CodeParseError,
				LineNumber: 1,
				Expression: "invalid",
				Message:    "Invalid cron expression",
				Hint:       check.GetCodeHint(check.CodeParseError),
			},
		},
	}

	// Don't let os.Exit kill the test
	oldExit := osExit
	exitCode := 0
	osExit = func(code int) { exitCode = code }
	defer func() { osExit = oldExit }()

	err := cc.outputJSON(result, check.SeverityError, false)
	require.NoError(t, err)

	// Should exit with code 1 for invalid result
	assert.Equal(t, 1, exitCode, "Should exit with code 1 for invalid result")

	var output map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)
	// Should be invalid
	assert.False(t, output["valid"].(bool))
	assert.Equal(t, float64(1), output["totalJobs"])
}

func TestCheckCommand_GroupedOutput(t *testing.T) {
	t.Run("should handle outputCheckText with warnings and verbose", func(t *testing.T) {
		// Test the verbose output path in outputCheckText
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		// Use an expression with DOM/DOW conflict to generate warnings
		cc.SetArgs([]string{"0 0 1 * 1", "--verbose"})

		err := cc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.NotEmpty(t, output)
		// Should show warnings when verbose
		assert.Contains(t, output, "warning")
	})

	t.Run("should handle outputCheckText with grouped issues by severity", func(t *testing.T) {
		// Test the grouping path in outputCheckText
		testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")

		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		cc.SetArgs([]string{"--file", testFile, "--group-by", "severity"})

		err := cc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.NotEmpty(t, output)
		// This tests the grouping logic path
	})

	t.Run("should handle outputCheckText with grouped issues by line", func(t *testing.T) {
		// Test the grouping by line path
		testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")

		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		cc.SetArgs([]string{"--file", testFile, "--group-by", "line"})

		err := cc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.NotEmpty(t, output)
		// This tests the line grouping logic path
	})

	t.Run("should handle outputCheckText with grouped issues by job", func(t *testing.T) {
		// Test the grouping by job path
		testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")

		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		cc.SetArgs([]string{"--file", testFile, "--group-by", "job"})

		err := cc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.NotEmpty(t, output)
		// This tests the job grouping logic path
	})

	t.Run("should handle getSeverityOrder function", func(t *testing.T) {
		// Test that getSeverityOrder is called (line 318)
		// This function is used for sorting issues
		testFile := filepath.Join("..", "..", "testdata", "crontab", "valid", "sample.cron")

		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		cc.SetArgs([]string{"--file", testFile, "--group-by", "severity"})

		err := cc.Execute()
		require.NoError(t, err)
		// This tests that getSeverityOrder is called during grouping
		_ = buf.String()
	})
}

func TestCheckCommand_PrintIssueWithHint(t *testing.T) {
	t.Run("should handle printIssue with hint", func(t *testing.T) {
		// Test the printIssue function when hint is present (line 438-440)
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		// Create a result with an issue that has a hint
		result := check.ValidationResult{
			Valid:     false,
			TotalJobs: 1,
			Issues: []check.Issue{
				{
					Severity:   check.SeverityError,
					Code:       check.CodeParseError,
					LineNumber: 1,
					Expression: "invalid",
					Message:    "Invalid cron expression",
					Hint:       "Use a valid cron expression format",
				},
			},
		}

		cc.verbose = true
		cc.groupBy = "none"

		err := cc.outputText(result, check.SeverityError, false)
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Invalid cron expression")
		assert.Contains(t, output, "Use a valid cron expression format")
		// This tests the hint display path (line 438-440)
	})

	t.Run("should handle printIssue without hint", func(t *testing.T) {
		// Test the printIssue function when hint is empty (line 438 check fails)
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		// Create a result with an issue without hint
		result := check.ValidationResult{
			Valid:     false,
			TotalJobs: 1,
			Issues: []check.Issue{
				{
					Severity:   check.SeverityError,
					Code:       check.CodeParseError,
					LineNumber: 1,
					Expression: "invalid",
					Message:    "Invalid cron expression",
					Hint:       "", // No hint
				},
			},
		}

		cc.verbose = true
		cc.groupBy = "none"

		err := cc.outputText(result, check.SeverityError, false)
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Invalid cron expression")
		// Should not contain hint section when hint is empty
		assert.NotContains(t, output, "Hint:")
	})
}

func TestCheckCommand_PrintIssues(t *testing.T) {
	t.Run("should handle printIssuesFlat", func(t *testing.T) {
		// Test the printIssuesFlat function (line 327-331)
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		// Create a result with issues
		result := check.ValidationResult{
			Valid:     false,
			TotalJobs: 1,
			Issues: []check.Issue{
				{
					Severity:   check.SeverityError,
					Code:       check.CodeParseError,
					LineNumber: 1,
					Expression: "invalid",
					Message:    "Invalid cron expression",
				},
			},
		}

		cc.verbose = true
		cc.groupBy = "none" // This triggers printIssuesFlat

		err := cc.outputText(result, check.SeverityError, false)
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "error")
	})

	t.Run("should handle printIssue with all fields", func(t *testing.T) {
		// Test the printIssue function with all fields populated
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		// Create a result with an issue that has all fields
		result := check.ValidationResult{
			Valid:     false,
			TotalJobs: 1,
			Issues: []check.Issue{
				{
					Severity:   check.SeverityError,
					Code:       check.CodeParseError,
					LineNumber: 5,
					Expression: "0 0 * * *",
					Message:    "Test message",
					Hint:       "Test hint",
				},
			},
		}

		cc.verbose = true
		cc.groupBy = "none"

		err := cc.outputText(result, check.SeverityError, false)
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Test message")
		// This tests printIssue function (line 400-442)
	})

	t.Run("should handle printIssue without expression", func(t *testing.T) {
		// Test the printIssue function when expression is empty
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		// Create a result with an issue without expression
		result := check.ValidationResult{
			Valid:     false,
			TotalJobs: 1,
			Issues: []check.Issue{
				{
					Severity:   check.SeverityError,
					Code:       check.CodeParseError,
					LineNumber: 0,
					Expression: "",
					Message:    "Test message",
				},
			},
		}

		cc.verbose = true
		cc.groupBy = "none"

		err := cc.outputText(result, check.SeverityError, false)
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Test message")
		// This tests the path where expression is empty (line 410-412)
	})
}

func TestCheckCommand_Stdin(t *testing.T) {
	t.Run("check with --stdin flag", func(t *testing.T) {
		// Create a pipe to simulate stdin
		r, w, err := os.Pipe()
		require.NoError(t, err)

		// Write test crontab to pipe
		_, err = w.WriteString("0 2 * * * /usr/bin/backup.sh\n")
		require.NoError(t, err)
		require.NoError(t, w.Close())

		// Replace stdin
		oldStdin := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = oldStdin }()

		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"--stdin"})

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err = cc.Execute()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "All valid")
	})

	t.Run("check with --stdin flag and invalid crontab", func(t *testing.T) {
		// Create a pipe to simulate stdin
		r, w, err := os.Pipe()
		require.NoError(t, err)

		// Write invalid crontab to pipe
		_, err = w.WriteString("60 0 * * * /usr/bin/invalid.sh\n")
		require.NoError(t, err)
		require.NoError(t, w.Close())

		// Replace stdin
		oldStdin := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = oldStdin }()

		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"--stdin"})

		oldExit := osExit
		exitCode := 0
		osExit = func(code int) { exitCode = code }
		defer func() { osExit = oldExit }()

		err = cc.Execute()
		require.NoError(t, err)
		assert.Equal(t, 1, exitCode)
		assert.Contains(t, buf.String(), "error")
	})

	t.Run("check with --stdin flag and JSON output", func(t *testing.T) {
		// Create a pipe to simulate stdin
		r, w, err := os.Pipe()
		require.NoError(t, err)

		// Write test crontab to pipe
		_, err = w.WriteString("0 2 * * * /usr/bin/backup.sh\n")
		require.NoError(t, err)
		require.NoError(t, w.Close())

		// Replace stdin
		oldStdin := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = oldStdin }()

		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"--stdin", "--json"})

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err = cc.Execute()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), `"valid"`)
		assert.Contains(t, buf.String(), `"totalJobs"`)
	})

	t.Run("check with --stdin flag and --fail-on warn", func(t *testing.T) {
		// Create a pipe to simulate stdin
		r, w, err := os.Pipe()
		require.NoError(t, err)

		// Write crontab with DOM/DOW conflict
		_, err = w.WriteString("0 0 1 * 1 /usr/bin/job.sh\n")
		require.NoError(t, err)
		require.NoError(t, w.Close())

		// Replace stdin
		oldStdin := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = oldStdin }()

		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"--stdin", "--fail-on", "warn", "--verbose"})

		oldExit := osExit
		exitCode := 0
		osExit = func(code int) { exitCode = code }
		defer func() { osExit = oldExit }()

		err = cc.Execute()
		require.NoError(t, err)
		assert.Equal(t, 2, exitCode) // Should exit with code 2 for warnings
	})
}

func TestCheckCommand_InvalidFailOn(t *testing.T) {
	t.Run("check with invalid --fail-on value", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetErr(buf)
		cc.SetArgs([]string{"0 0 * * *", "--fail-on", "invalid"})

		err := cc.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid --fail-on value")
	})
}

func TestGroupIssues(t *testing.T) {
	issues := []check.Issue{
		{
			Severity:   check.SeverityError,
			Code:       check.CodeParseError,
			LineNumber: 1,
			Expression: "0 0 * * *",
			Message:    "Error 1",
		},
		{
			Severity:   check.SeverityWarn,
			Code:       check.CodeDOMDOWConflict,
			LineNumber: 2,
			Expression: "0 0 1 * 1",
			Message:    "Warning 1",
		},
		{
			Severity:   check.SeverityError,
			Code:       check.CodeEmptySchedule,
			LineNumber: 1,
			Expression: "0 0 30 2 *",
			Message:    "Error 2",
		},
		{
			Severity:   check.SeverityWarn,
			Code:       check.CodeDOMDOWConflict,
			LineNumber: 3,
			Expression: "0 0 1 * 1",
			Message:    "Warning 2",
		},
		{
			Severity:   check.SeverityError,
			Code:       check.CodeParseError,
			LineNumber: 0,
			Expression: "",
			Message:    "Error without expression",
		},
	}

	t.Run("group by severity", func(t *testing.T) {
		groups := groupIssues(issues, GroupBySeverity)
		assert.Equal(t, 2, len(groups))
		assert.Equal(t, 3, len(groups["error"]))
		assert.Equal(t, 2, len(groups["warn"]))
	})

	t.Run("group by line", func(t *testing.T) {
		groups := groupIssues(issues, GroupByLine)
		assert.Equal(t, 4, len(groups))
		assert.Equal(t, 2, len(groups["line-1"]))
		assert.Equal(t, 1, len(groups["line-2"]))
		assert.Equal(t, 1, len(groups["line-3"]))
		assert.Equal(t, 1, len(groups["no-line"]))
	})

	t.Run("group by job", func(t *testing.T) {
		groups := groupIssues(issues, GroupByJob)
		assert.Equal(t, 4, len(groups))
		assert.Equal(t, 1, len(groups["0 0 * * *"]))
		assert.Equal(t, 2, len(groups["0 0 1 * 1"]))
		assert.Equal(t, 1, len(groups["0 0 30 2 *"]))
		assert.Equal(t, 1, len(groups["no-expression"]))
	})

	t.Run("group by none", func(t *testing.T) {
		groups := groupIssues(issues, GroupByNone)
		assert.Equal(t, 0, len(groups))
	})

	t.Run("group empty issues", func(t *testing.T) {
		groups := groupIssues([]check.Issue{}, GroupBySeverity)
		assert.Equal(t, 0, len(groups))
	})
}

func TestParseGroupBy(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected GroupByMode
	}{
		{
			name:     "severity",
			input:    "severity",
			expected: GroupBySeverity,
		},
		{
			name:     "line",
			input:    "line",
			expected: GroupByLine,
		},
		{
			name:     "job",
			input:    "job",
			expected: GroupByJob,
		},
		{
			name:     "none",
			input:    "none",
			expected: GroupByNone,
		},
		{
			name:     "invalid",
			input:    "invalid",
			expected: GroupByNone,
		},
		{
			name:     "empty",
			input:    "",
			expected: GroupByNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGroupBy(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckCommand_GroupBy(t *testing.T) {
	t.Run("check with group-by severity", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"0 0 1 * 1", "--group-by", "severity", "--verbose"})

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "warn")
	})

	t.Run("check with group-by line", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"0 0 1 * 1", "--group-by", "line", "--verbose"})

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		require.NoError(t, err)
	})

	t.Run("check with group-by line does not merge findings from different files", func(t *testing.T) {
		// Pins that GroupByLine keys on (file, line), not the bare line number.
		inv := inventory.New("", []inventory.Item{
			{Expression: "60 0 * * *", Command: "worker-a", State: inventory.StateActive, Locator: inventory.Locator{File: "ci.yml", Line: 6}},
			{Expression: "60 0 * * *", Command: "worker-b", State: inventory.StateActive, Locator: inventory.Locator{File: "cronjob.yaml", Line: 6}},
		})
		f, err := os.CreateTemp("", "cronkit-inventory-*.json")
		require.NoError(t, err)
		require.NoError(t, inv.Encode(f))
		require.NoError(t, f.Close())
		defer func() { _ = os.Remove(f.Name()) }()

		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"--inventory", f.Name(), "--group-by", "line"})

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err = cc.Execute()
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "ci.yml:6", "each file's finding must keep its own heading")
		assert.Contains(t, output, "cronjob.yaml:6", "each file's finding must keep its own heading")
		assert.Equal(t, 2, strings.Count(output, "issue(s))"), "same-line findings from different files must render as two separate groups")
	})

	t.Run("check with warnings in compact format (not verbose)", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		// Use expression that generates warnings but not verbose mode
		// This should trigger printWarningsCompact
		cc.SetArgs([]string{"0 0 1 * 1"}) // DOM/DOW conflict

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		require.NoError(t, err)
		output := buf.String()
		// Should use compact format (one line per warning)
		assert.Contains(t, output, "⚠")
		// Should include expression in compact format
		assert.Contains(t, output, "0 0 1 * 1")
	})

	t.Run("check printWarningsCompact with all edge cases", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)

		// Create issues with different combinations to test all branches
		warnings := []check.Issue{
			{
				LineNumber: 1,
				Code:       "TEST_CODE",
				Message:    "Test warning",
				Expression: "0 0 1 * 1",
			},
			{
				LineNumber: 0, // No line number
				Code:       "",
				Message:    "Warning without code",
				Expression: "0 * * * *",
			},
			{
				LineNumber: 2,
				Code:       "CODE2",
				Message:    "Warning without expression",
				Expression: "", // No expression
			},
		}

		cc.printWarningsCompact(warnings, false)
		output := buf.String()

		// Should have all three warnings
		assert.Contains(t, output, "Test warning")
		assert.Contains(t, output, "Warning without code")
		assert.Contains(t, output, "Warning without expression")
		// First warning should have line number and code
		assert.Contains(t, output, "Line 1:")
		assert.Contains(t, output, "[TEST_CODE]")
		// Second warning should not have line number prefix
		assert.Contains(t, output, "Warning without code")
		// Third warning should not have expression suffix
		assert.NotContains(t, output, "Warning without expression -")
	})

	t.Run("check with group-by job", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"0 0 1 * 1", "--group-by", "job", "--verbose"})

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		require.NoError(t, err)
	})

	t.Run("check with group-by job renders groups in a deterministic order", func(t *testing.T) {
		// Runs outputText repeatedly to catch nondeterministic map-order regressions.
		result := check.ValidationResult{
			Valid: false,
			Issues: []check.Issue{
				{Severity: check.SeverityError, Code: check.CodeParseError, Expression: "0 0 * * *", Message: "Error A"},
				{Severity: check.SeverityError, Code: check.CodeParseError, Expression: "0 12 * * *", Message: "Error B"},
				{Severity: check.SeverityError, Code: check.CodeParseError, Expression: "0 6 * * *", Message: "Error C"},
				{Severity: check.SeverityError, Code: check.CodeParseError, Expression: "", Message: "Error D"},
			},
		}

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		cc := newCheckCommand()
		cc.groupBy = "job"
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		require.NoError(t, cc.outputText(result, check.SeverityError, false))
		first := buf.String()

		for i := 0; i < 20; i++ {
			cc := newCheckCommand()
			cc.groupBy = "job"
			buf := new(bytes.Buffer)
			cc.SetOut(buf)
			require.NoError(t, cc.outputText(result, check.SeverityError, false))
			assert.Equal(t, first, buf.String(), "group-by job output must not vary between runs")
		}
	})

	t.Run("check with invalid group-by", func(t *testing.T) {
		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"0 0 * * *", "--group-by", "invalid"})

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err := cc.Execute()
		require.NoError(t, err)
		// Should default to no grouping
	})
}

func TestCheckCommand_Inventory(t *testing.T) {
	t.Run("validates items read from an inventory file", func(t *testing.T) {
		inv := inventory.New("", []inventory.Item{
			{Expression: "0 0 * * *", Command: "gcr.io/proj/worker", State: inventory.StateActive},
		})
		f, err := os.CreateTemp("", "cronkit-inventory-*.json")
		require.NoError(t, err)
		require.NoError(t, inv.Encode(f))
		require.NoError(t, f.Close())
		defer func() { _ = os.Remove(f.Name()) }()

		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"--inventory", f.Name()})

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err = cc.Execute()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "All valid")
	})

	t.Run("a container image is never judged by shell hygiene rules", func(t *testing.T) {
		inv := inventory.New("", []inventory.Item{
			{Expression: "0 0 * * *", Command: "worker", Shell: false, State: inventory.StateActive},
		})
		f, err := os.CreateTemp("", "cronkit-inventory-*.json")
		require.NoError(t, err)
		require.NoError(t, inv.Encode(f))
		require.NoError(t, f.Close())
		defer func() { _ = os.Remove(f.Name()) }()

		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"--inventory", f.Name(), "--enable-hygiene-checks", "--verbose"})

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err = cc.Execute()
		require.NoError(t, err)
		assert.NotContains(t, buf.String(), "absolute path",
			"a non-shell command like a container image must never trip CRON-008")
	})

	t.Run("a malformed inventory fails the command", func(t *testing.T) {
		f, err := os.CreateTemp("", "cronkit-inventory-*.json")
		require.NoError(t, err)
		_, err = f.WriteString("not json")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		defer func() { _ = os.Remove(f.Name()) }()

		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetErr(buf)
		cc.SetArgs([]string{"--inventory", f.Name()})

		err = cc.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read inventory")
	})

	t.Run("--inventory - reads from stdin", func(t *testing.T) {
		inv := inventory.New("", []inventory.Item{
			{Expression: "0 0 * * *", Command: "worker", State: inventory.StateActive},
		})
		r, w, err := os.Pipe()
		require.NoError(t, err)
		require.NoError(t, inv.Encode(w))
		require.NoError(t, w.Close())
		oldStdin := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = oldStdin }()

		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"--inventory", "-"})

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err = cc.Execute()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "All valid")
	})

	t.Run("a three-file inventory shows locators even when only one file has findings", func(t *testing.T) {
		// Pins that showLocator is derived from the input, not the issue set.
		inv := inventory.New("", []inventory.Item{
			{Expression: "0 0 * * *", Command: "worker-a", State: inventory.StateActive, Locator: inventory.Locator{File: "site-a/crontab", Line: 1}},
			{Expression: "0 0 * * *", Command: "worker-b", State: inventory.StateActive, Locator: inventory.Locator{File: "site-b/crontab", Line: 1}},
			{Expression: "60 0 * * *", Command: "worker-c", State: inventory.StateActive, Locator: inventory.Locator{File: "site-c/crontab", Line: 1}},
		})
		f, err := os.CreateTemp("", "cronkit-inventory-*.json")
		require.NoError(t, err)
		require.NoError(t, inv.Encode(f))
		require.NoError(t, f.Close())
		defer func() { _ = os.Remove(f.Name()) }()

		cc := newCheckCommand()
		buf := new(bytes.Buffer)
		cc.SetOut(buf)
		cc.SetArgs([]string{"--inventory", f.Name()})

		oldExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldExit }()

		err = cc.Execute()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "site-c/crontab:1:",
			"the sole file with a finding must still be named once the input spans multiple files")
	})
}

// TestCheckCommand_LocatorRendering pins down when a locator is worth showing: only when it disambiguates.
func TestCheckCommand_LocatorRendering(t *testing.T) {
	oldExit := osExit
	osExit = func(code int) {}
	defer func() { osExit = oldExit }()

	t.Run("a single file's issues render exactly as before locators existed", func(t *testing.T) {
		result := check.ValidationResult{
			Valid: false,
			Issues: []check.Issue{
				{
					Severity:   check.SeverityError,
					Code:       check.CodeParseError,
					LineNumber: 6,
					Locator:    inventory.Locator{File: "crontab", Line: 6},
					Expression: "60 0 * * *",
					Message:    "Invalid cron expression: bad",
				},
			},
		}

		cc := newCheckCommand()
		textBuf := new(bytes.Buffer)
		cc.SetOut(textBuf)
		require.NoError(t, cc.outputText(result, check.SeverityError, false))
		assert.Contains(t, textBuf.String(), "Line 6:")
		assert.NotContains(t, textBuf.String(), "crontab:6",
			"a single-source run must not repeat a file the user already named on the command line")

		cc = newCheckCommand()
		jsonBuf := new(bytes.Buffer)
		cc.SetOut(jsonBuf)
		require.NoError(t, cc.outputJSON(result, check.SeverityError, false))
		var decoded map[string]interface{}
		require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &decoded))
		issues := decoded["issues"].([]interface{})
		require.Len(t, issues, 1)
		issue := issues[0].(map[string]interface{})
		assert.NotContains(t, issue, "locator",
			"single-file JSON output must stay byte-identical to before Issue carried a locator")
	})

	t.Run("a multi-file inventory scan names the file", func(t *testing.T) {
		result := check.ValidationResult{
			Valid: false,
			Issues: []check.Issue{
				{
					Severity:   check.SeverityError,
					Code:       check.CodeParseError,
					LineNumber: 2,
					Locator:    inventory.Locator{File: "site-a/crontab", Line: 2},
					Expression: "60 0 * * *",
					Message:    "Invalid cron expression: bad a",
				},
				{
					Severity:   check.SeverityError,
					Code:       check.CodeParseError,
					LineNumber: 6,
					Locator:    inventory.Locator{File: "site-b/k8s.yaml", Line: 6, Path: "spec.schedule"},
					Expression: "60 0 * * *",
					Message:    "Invalid cron expression: bad b",
				},
			},
		}

		cc := newCheckCommand()
		textBuf := new(bytes.Buffer)
		cc.SetOut(textBuf)
		require.NoError(t, cc.outputText(result, check.SeverityError, true))
		output := textBuf.String()
		assert.Contains(t, output, "site-a/crontab:2:")
		assert.Contains(t, output, "site-b/k8s.yaml:6 (spec.schedule):")

		cc = newCheckCommand()
		jsonBuf := new(bytes.Buffer)
		cc.SetOut(jsonBuf)
		require.NoError(t, cc.outputJSON(result, check.SeverityError, true))
		var decoded map[string]interface{}
		require.NoError(t, json.Unmarshal(jsonBuf.Bytes(), &decoded))
		issues := decoded["issues"].([]interface{})
		require.Len(t, issues, 2)
		first := issues[0].(map[string]interface{})
		locator := first["locator"].(map[string]interface{})
		assert.Equal(t, "site-a/crontab", locator["file"])
		assert.Equal(t, float64(2), locator["line"])
		second := issues[1].(map[string]interface{})
		locator = second["locator"].(map[string]interface{})
		assert.Equal(t, "site-b/k8s.yaml", locator["file"])
		assert.Equal(t, "spec.schedule", locator["path"])
	})
}

// TestCheckIssueLabelsDisambiguateSharedLines covers two findings on one line, told apart by path.
func TestCheckIssueLabelsDisambiguateSharedLines(t *testing.T) {
	issues := []check.Issue{
		{
			Severity:   check.SeverityError,
			LineNumber: 3,
			Locator:    inventory.Locator{File: "wf.yaml", Line: 3, Path: "spec.schedules[0]"},
			Message:    "Error 1",
		},
		{
			Severity:   check.SeverityError,
			LineNumber: 3,
			Locator:    inventory.Locator{File: "wf.yaml", Line: 3, Path: "spec.schedules[1]"},
			Message:    "Error 2",
		},
	}

	t.Run("the line prefix names the path when one is set", func(t *testing.T) {
		assert.Equal(t, "Line 3 (spec.schedules[0]): ", issueLinePrefix(issues[0], false))
		assert.Equal(t, "Line 3 (spec.schedules[1]): ", issueLinePrefix(issues[1], false))
	})

	t.Run("grouped output gives the two findings distinguishable headers", func(t *testing.T) {
		cc := newCheckCommand()
		var buf bytes.Buffer
		cc.SetOut(&buf)
		cc.printIssuesGrouped(issues, GroupByLine, false)

		out := buf.String()
		assert.Contains(t, out, "Line 3 (spec.schedules[0])")
		assert.Contains(t, out, "Line 3 (spec.schedules[1])")
	})
}
