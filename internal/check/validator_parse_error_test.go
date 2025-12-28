package check

import (
	"fmt"
	"testing"

	"github.com/hzerrad/cronic/internal/crontab"
	"github.com/hzerrad/cronic/internal/cronx"
	"github.com/stretchr/testify/assert"
)

// mockParser is a mock parser that can be configured to fail
type mockParser struct {
	shouldFail bool
}

func (m *mockParser) Parse(expression string) (*cronx.Schedule, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("mock parse error")
	}
	// Use real parser for valid cases
	realParser := cronx.NewParser()
	return realParser.Parse(expression)
}

// TestValidateCrontab_ParseErrorAfterValidation tests the path where
// parse fails even though Valid=true (lines 139-151)
func TestValidateCrontab_ParseErrorAfterValidation(t *testing.T) {
	validator := &Validator{
		parser:    &mockParser{shouldFail: true},
		scheduler: cronx.NewScheduler(),
		locale:    "en",
	}

	// We need to manually create entries with Valid=true
	// Since the real reader will parse and set Valid correctly,
	// we'll use a mock reader instead
	mockReader := &mockReader{
		entries: []*crontab.Entry{
			{
				Type:       crontab.EntryTypeJob,
				LineNumber: 1,
				Job: &crontab.Job{
					LineNumber: 1,
					Expression: "0 0 * * *",
					Command:    "/usr/bin/test.sh",
					Valid:      true, // Marked as valid but will fail on parse
				},
			},
		},
	}

	result := validator.ValidateCrontab(mockReader, "dummy-path")
	// Should detect parse error
	assert.False(t, result.Valid)
	assert.Equal(t, 1, result.InvalidJobs)
	hasParseError := false
	for _, issue := range result.Issues {
		if issue.Severity == SeverityError && issue.Message != "" {
			hasParseError = true
			assert.Contains(t, issue.Message, "Failed to parse expression")
			break
		}
	}
	assert.True(t, hasParseError, "Should have parse error")
}

// TestValidateUserCrontab_ParseErrorAfterValidation tests the parse error path
func TestValidateUserCrontab_ParseErrorAfterValidation(t *testing.T) {
	validator := &Validator{
		parser:    &mockParser{shouldFail: true},
		scheduler: cronx.NewScheduler(),
		locale:    "en",
	}

	mockReader := &mockReader{
		jobs: []*crontab.Job{
			{
				LineNumber: 1,
				Expression: "0 0 * * *",
				Command:    "/usr/bin/test.sh",
				Valid:      true, // Marked as valid but will fail on parse
			},
		},
	}

	result := validator.ValidateUserCrontab(mockReader)
	// Should detect parse error
	assert.False(t, result.Valid)
	assert.Equal(t, 1, result.InvalidJobs)
	hasParseError := false
	for _, issue := range result.Issues {
		if issue.Severity == SeverityError && issue.Message != "" {
			hasParseError = true
			assert.Contains(t, issue.Message, "Failed to parse expression")
			break
		}
	}
	assert.True(t, hasParseError, "Should have parse error")
}

// TestValidateUserCrontab_EmptyScheduleWithDOMDOW tests both checks running
func TestValidateUserCrontab_EmptyScheduleWithDOMDOW(t *testing.T) {
	validator := &Validator{
		parser:    cronx.NewParserWithLocale("en"),
		scheduler: &mockScheduler{returnEmpty: true},
		locale:    "en",
	}

	mockReader := &mockReader{
		jobs: []*crontab.Job{
			{
				LineNumber: 1,
				Expression: "0 0 1 * 1",
				Command:    "/usr/bin/test.sh",
				Valid:      true,
			},
		},
	}

	result := validator.ValidateUserCrontab(mockReader)
	// Should be invalid due to empty schedule
	assert.False(t, result.Valid)
	hasEmptyError := false
	for _, issue := range result.Issues {
		if issue.Message == "Schedule never runs (empty schedule)" {
			hasEmptyError = true
			break
		}
	}
	assert.True(t, hasEmptyError, "Should have empty schedule error")
}
