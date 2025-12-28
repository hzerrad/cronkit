package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/hzerrad/cronic/internal/check"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	err := cc.outputJSON(result)
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

	err := cc.outputJSON(result)
	require.NoError(t, err)
	// Should exit with code 2 for warnings with verbose
	assert.Equal(t, 2, exitCode, "Should exit with code 2 for warnings with --verbose")
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

	err := cc.outputJSON(result)
	require.NoError(t, err)
	// Should not exit with code 2 when not verbose (warnings filtered out)
	assert.Equal(t, 0, exitCode, "Should not exit with code 2 when not verbose")

	var output map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)
	// Issues should be filtered out
	issues := output["issues"].([]interface{})
	assert.Equal(t, 0, len(issues), "Warnings should be filtered out when not verbose")
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

	err := cc.outputJSON(result)
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

	err := cc.outputJSON(result)
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
