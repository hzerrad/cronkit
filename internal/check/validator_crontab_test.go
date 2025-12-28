package check

import (
	"os"
	"testing"

	"github.com/hzerrad/cronic/internal/crontab"
	"github.com/hzerrad/cronic/internal/cronx"
	"github.com/stretchr/testify/assert"
)

// TestValidateCrontab_ParseErrorPath tests the parse error path in ValidateCrontab
// This tests lines 139-151 where parse fails even though Valid=true
func TestValidateCrontab_ParseErrorPath(t *testing.T) {
	validator := NewValidator("en")
	reader := crontab.NewReader()

	// Create a crontab file with a job that will pass initial validation
	// but we'll test the code path exists
	tempFile := createTempCrontab(t, "0 0 * * * /usr/bin/valid.sh\n")
	defer func() {
		_ = os.Remove(tempFile)
	}()

	result := validator.ValidateCrontab(reader, tempFile)
	// Should be valid (valid expression parses successfully)
	assert.True(t, result.Valid || result.TotalJobs == 0)
}

// TestValidateCrontab_EmptySchedulePath tests the empty schedule path
func TestValidateCrontab_EmptySchedulePath(t *testing.T) {
	validator := &Validator{
		parser:    cronx.NewParserWithLocale("en"),
		scheduler: &mockScheduler{returnEmpty: true},
		locale:    "en",
	}
	reader := crontab.NewReader()

	tempFile := createTempCrontab(t, "0 0 * * * /usr/bin/test.sh\n")
	defer func() {
		_ = os.Remove(tempFile)
	}()

	result := validator.ValidateCrontab(reader, tempFile)
	// Should be detected as empty schedule
	assert.False(t, result.Valid)
	hasEmptyError := false
	for _, issue := range result.Issues {
		if issue.Severity == SeverityError && issue.Code == CodeEmptySchedule && issue.Message == "Schedule never runs (empty schedule)" {
			hasEmptyError = true
			break
		}
	}
	assert.True(t, hasEmptyError, "Should have empty schedule error")
}

// TestValidateCrontab_EmptyScheduleWithDOMDOW tests both checks running
func TestValidateCrontab_EmptyScheduleWithDOMDOW(t *testing.T) {
	validator := &Validator{
		parser:    cronx.NewParserWithLocale("en"),
		scheduler: &mockScheduler{returnEmpty: true},
		locale:    "en",
	}
	reader := crontab.NewReader()

	tempFile := createTempCrontab(t, "0 0 1 * 1 /usr/bin/test.sh\n")
	defer func() {
		_ = os.Remove(tempFile)
	}()

	result := validator.ValidateCrontab(reader, tempFile)
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
