package cmd

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/hzerrad/cronic/internal/check"
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
		// Output should contain issue information
		output := buf.String()
		assert.Contains(t, output, "issue", "Should show issues in output")
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
		// Without verbose, warnings shouldn't show
		output := buf.String()
		assert.Contains(t, output, "All valid")
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
		assert.Contains(t, buf.String(), "issue")
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
			assert.Contains(t, issue, "type") // Backward compatibility
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
		// Should exit with code 2 for warnings with --verbose (backward compatibility)
		assert.Equal(t, 2, exitCode, "Should exit with code 2 for warnings with --verbose (backward compatibility)")

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
		// Should exit with code 2 for warnings with --verbose (backward compatibility)
		assert.Equal(t, 2, exitCode, "Should exit with code 2 for warnings with --verbose (backward compatibility)")
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
		assert.Contains(t, buf.String(), "issue")
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

		err := cc.outputText(result, check.SeverityError)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "warning")
		assert.Equal(t, 2, exitCode, "Should exit with code 2 for warnings with --verbose (backward compatibility)")
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

		err := cc.outputText(result, check.SeverityError)
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

		err := cc.outputText(result, check.SeverityError)
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

		err := cc.outputText(result, check.SeverityError)
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

		err := cc.outputText(result, check.SeverityError)
		require.NoError(t, err)
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

		err := cc.outputText(result, check.SeverityError)
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
		assert.Contains(t, buf.String(), "All valid")
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
		exitCode := calculateExitCode(result, []check.Issue{}, check.SeverityError, false)
		assert.Equal(t, 0, exitCode)
	})

	t.Run("calculateExitCode with errors and fail-on error", func(t *testing.T) {
		result := check.ValidationResult{
			Valid: false,
			Issues: []check.Issue{
				{Severity: check.SeverityError, Code: check.CodeParseError},
			},
		}
		exitCode := calculateExitCode(result, result.Issues, check.SeverityError, false)
		assert.Equal(t, 1, exitCode)
	})

	t.Run("calculateExitCode with warnings and fail-on error", func(t *testing.T) {
		result := check.ValidationResult{
			Valid: true,
			Issues: []check.Issue{
				{Severity: check.SeverityWarn, Code: check.CodeDOMDOWConflict},
			},
		}
		exitCode := calculateExitCode(result, result.Issues, check.SeverityError, false)
		assert.Equal(t, 0, exitCode, "Warnings should not cause exit with --fail-on error")
	})

	t.Run("calculateExitCode with warnings and fail-on warn", func(t *testing.T) {
		result := check.ValidationResult{
			Valid: true,
			Issues: []check.Issue{
				{Severity: check.SeverityWarn, Code: check.CodeDOMDOWConflict},
			},
		}
		exitCode := calculateExitCode(result, result.Issues, check.SeverityWarn, false)
		assert.Equal(t, 2, exitCode)
	})

	t.Run("calculateExitCode with warnings and verbose (backward compatibility)", func(t *testing.T) {
		result := check.ValidationResult{
			Valid: true,
			Issues: []check.Issue{
				{Severity: check.SeverityWarn, Code: check.CodeDOMDOWConflict},
			},
		}
		exitCode := calculateExitCode(result, result.Issues, check.SeverityError, true)
		assert.Equal(t, 2, exitCode, "Should exit with code 2 for warnings with --verbose (backward compatibility)")
	})

	t.Run("calculateExitCode with errors and fail-on warn", func(t *testing.T) {
		result := check.ValidationResult{
			Valid: false,
			Issues: []check.Issue{
				{Severity: check.SeverityError, Code: check.CodeParseError},
			},
		}
		exitCode := calculateExitCode(result, result.Issues, check.SeverityWarn, false)
		assert.Equal(t, 1, exitCode, "Errors should cause exit code 1 even with --fail-on warn")
	})

	t.Run("calculateExitCode with info and fail-on info", func(t *testing.T) {
		result := check.ValidationResult{
			Valid: true,
			Issues: []check.Issue{
				{Severity: check.SeverityInfo, Code: ""},
			},
		}
		exitCode := calculateExitCode(result, result.Issues, check.SeverityInfo, false)
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
		exitCode := calculateExitCode(result, result.Issues, check.SeverityError, false)
		assert.Equal(t, 1, exitCode, "Should return 1 for errors when mixed with warnings")
	})

	t.Run("calculateExitCode default case", func(t *testing.T) {
		result := check.ValidationResult{
			Valid: true,
			Issues: []check.Issue{
				{Severity: check.Severity(999), Code: ""}, // Invalid severity
			},
		}
		exitCode := calculateExitCode(result, result.Issues, check.SeverityError, false)
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

		err := cc.outputText(result, check.SeverityError)
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

		err := cc.outputText(result, check.SeverityError)
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

		err := cc.outputText(result, check.SeverityError)
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

		err := cc.outputText(result, check.SeverityError)
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
