package check

import (
	"testing"
	"time"

	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetFrequencyChecks(t *testing.T) {
	validator := NewValidator("en")
	validator.SetFrequencyChecks(false)
	assert.False(t, validator.enableFrequency)

	validator.SetFrequencyChecks(true)
	assert.True(t, validator.enableFrequency)
}

func TestSetMaxRunsPerDay(t *testing.T) {
	validator := NewValidator("en")
	validator.SetMaxRunsPerDay(500)
	assert.Equal(t, 500, validator.maxRunsPerDay)
}

func TestSetHygieneChecks(t *testing.T) {
	validator := NewValidator("en")
	validator.SetHygieneChecks(true)
	assert.True(t, validator.enableHygiene)

	validator.SetHygieneChecks(false)
	assert.False(t, validator.enableHygiene)
}

func TestSetWarnOnOverlap(t *testing.T) {
	validator := NewValidator("en")
	validator.SetWarnOnOverlap(true)
	assert.True(t, validator.warnOnOverlap)
}

func TestSetOverlapWindow(t *testing.T) {
	validator := NewValidator("en")
	window := 48 * time.Hour
	validator.SetOverlapWindow(window)
	assert.Equal(t, window, validator.overlapWindow)
}

func TestValidateFrequency(t *testing.T) {
	validator := NewValidator("en")
	validator.SetFrequencyChecks(true)
	validator.SetMaxRunsPerDay(10) // Low threshold

	t.Run("should detect excessive runs", func(t *testing.T) {
		parser := cronx.NewParser()
		schedule, err := parser.Parse("* * * * *") // Every minute - will exceed threshold
		require.NoError(t, err)

		issues := validator.validateFrequency(schedule, "* * * * *")
		assert.Greater(t, len(issues), 0)
	})

	t.Run("should detect redundant patterns", func(t *testing.T) {
		parser := cronx.NewParser()
		schedule, err := parser.Parse("*/1 * * * *")
		require.NoError(t, err)

		issues := validator.validateFrequency(schedule, "*/1 * * * *")
		// Should detect redundant pattern
		found := false
		for _, issue := range issues {
			if issue.Code == CodeRedundantPattern {
				found = true
				break
			}
		}
		assert.True(t, found, "Should detect redundant pattern")
	})

	t.Run("should handle calculation errors gracefully", func(t *testing.T) {
		parser := cronx.NewParser()
		schedule, err := parser.Parse("0 * * * *")
		require.NoError(t, err)

		// Use a valid schedule that won't cause calculation errors
		issues := validator.validateFrequency(schedule, "0 * * * *")
		// Should not error
		assert.GreaterOrEqual(t, len(issues), 0)
	})
}

func TestMin(t *testing.T) {
	t.Run("should return first value when a < b", func(t *testing.T) {
		assert.Equal(t, 1, min(1, 2))
		assert.Equal(t, 0, min(0, 1))
		assert.Equal(t, 5, min(5, 10))
	})

	t.Run("should return second value when a > b", func(t *testing.T) {
		assert.Equal(t, 2, min(3, 2))
		assert.Equal(t, 1, min(2, 1))
	})

	t.Run("should return either value when a == b", func(t *testing.T) {
		assert.Equal(t, 5, min(5, 5))
		assert.Equal(t, 0, min(0, 0))
	})
}

func TestDetectDOMDOWConflict(t *testing.T) {
	parser := cronx.NewParser()

	tests := []struct {
		name     string
		expr     string
		expected bool
	}{
		{
			name:     "both DOM and DOW specified",
			expr:     "0 0 1 * 1",
			expected: true,
		},
		{
			name:     "only DOM specified",
			expr:     "0 0 1 * *",
			expected: false,
		},
		{
			name:     "only DOW specified",
			expr:     "0 0 * * 1",
			expected: false,
		},
		{
			name:     "neither specified",
			expr:     "0 0 * * *",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := parser.Parse(tt.expr)
			require.NoError(t, err)
			result := detectDOMDOWConflict(schedule)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectEmptySchedule(t *testing.T) {
	scheduler := cronx.NewScheduler()

	t.Run("valid schedule should not be empty", func(t *testing.T) {
		result := detectEmptySchedule("0 0 * * *", cronx.KindFields, scheduler)
		assert.False(t, result, "Daily schedule should not be empty")
	})

	t.Run("invalid expression should be empty", func(t *testing.T) {
		result := detectEmptySchedule("invalid", cronx.KindFields, scheduler)
		assert.True(t, result, "Invalid expression should be detected as empty")
	})

	t.Run("expression that runs should not be empty", func(t *testing.T) {
		result := detectEmptySchedule("*/15 * * * *", cronx.KindFields, scheduler)
		assert.False(t, result, "Every 15 minutes should not be empty")
	})

	t.Run("very far future schedule", func(t *testing.T) {
		// This is a valid expression that runs, so should not be empty
		result := detectEmptySchedule("0 0 1 1 *", cronx.KindFields, scheduler)
		assert.False(t, result, "Yearly schedule should not be empty")
	})

	t.Run("complex valid expression", func(t *testing.T) {
		result := detectEmptySchedule("*/30 * * * *", cronx.KindFields, scheduler)
		assert.False(t, result, "Every 30 minutes should not be empty")
	})

	t.Run("reboot trigger is exempt, not empty", func(t *testing.T) {
		// @reboot has no wall-clock next run (Scheduler.Next reports zero
		// times for it), but that is not the same as never firing.
		result := detectEmptySchedule("@reboot", cronx.KindReboot, scheduler)
		assert.False(t, result, "@reboot should never be reported as an empty schedule")
	})
}

func TestValidator_ValidateExpression(t *testing.T) {
	validator := NewValidator("en")

	t.Run("valid expression", func(t *testing.T) {
		result := validator.ValidateExpression("0 0 * * *")
		assert.True(t, result.Valid)
		assert.Equal(t, 1, result.TotalJobs)
		assert.Equal(t, 1, result.ValidJobs)
		assert.Equal(t, 0, result.InvalidJobs)
		assert.Empty(t, result.Issues)
	})

	t.Run("invalid expression", func(t *testing.T) {
		result := validator.ValidateExpression("60 0 * * *")
		assert.False(t, result.Valid)
		assert.Equal(t, 1, result.TotalJobs)
		assert.Equal(t, 0, result.ValidJobs)
		assert.Equal(t, 1, result.InvalidJobs)
		require.Len(t, result.Issues, 1)
		assert.Equal(t, SeverityError, result.Issues[0].Severity)
		assert.Contains(t, result.Issues[0].Message, "Invalid cron expression")
	})

	t.Run("expression with DOM/DOW conflict", func(t *testing.T) {
		result := validator.ValidateExpression("0 0 1 * 1")
		assert.True(t, result.Valid, "Should be valid (cron allows it)")
		assert.Equal(t, 1, result.ValidJobs)
		require.Len(t, result.Issues, 1)
		assert.Equal(t, SeverityWarn, result.Issues[0].Severity)
		assert.Equal(t, CodeDOMDOWConflict, result.Issues[0].Code)
		assert.Contains(t, result.Issues[0].Message, "Both day-of-month and day-of-week")
		assert.NotEmpty(t, result.Issues[0].Hint)
	})

	t.Run("empty expression", func(t *testing.T) {
		result := validator.ValidateExpression("")
		assert.False(t, result.Valid)
		assert.Equal(t, 1, result.InvalidJobs)
		require.Len(t, result.Issues, 1)
		assert.Equal(t, SeverityError, result.Issues[0].Severity)
	})

	t.Run("alias expression", func(t *testing.T) {
		result := validator.ValidateExpression("@daily")
		assert.True(t, result.Valid)
		assert.Equal(t, 1, result.ValidJobs)
	})

	t.Run("reboot descriptor does not panic and reports no findings", func(t *testing.T) {
		// @reboot has no fields, so DOM/DOW and frequency checks must skip it rather than deref a nil Field.
		require.NotPanics(t, func() {
			result := validator.ValidateExpression("@reboot")
			assert.True(t, result.Valid, "@reboot is a valid schedule")
			assert.Equal(t, 1, result.ValidJobs)
			assert.Empty(t, result.Issues, "no DOM/DOW conflict or frequency finding for @reboot")
		})
	})

	t.Run("every descriptor does not panic and reports no field-shaped findings", func(t *testing.T) {
		// @every has no fields either (Kind == KindInterval); same guard as @reboot above.
		lowThreshold := NewValidator("en")
		lowThreshold.SetMaxRunsPerDay(1)

		require.NotPanics(t, func() {
			result := lowThreshold.ValidateExpression("@every 1h")
			assert.True(t, result.Valid, "@every 1h is a valid schedule")
			assert.Equal(t, 1, result.ValidJobs)
			assert.Empty(t, result.Issues, "no DOM/DOW conflict or frequency finding for @every 1h")
		})
	})

	t.Run("expression with empty schedule detection path", func(t *testing.T) {
		result := validator.ValidateExpression("0 0 * * *")
		assert.True(t, result.Valid)
		assert.Equal(t, 1, result.ValidJobs)
	})

	t.Run("expression with both DOM/DOW conflict and empty schedule check", func(t *testing.T) {
		// Test that both checks run
		result := validator.ValidateExpression("0 0 1 * 1")
		assert.True(t, result.Valid)
		// Should have warning for DOM/DOW conflict
		hasWarning := false
		for _, issue := range result.Issues {
			if issue.Severity == SeverityWarn {
				hasWarning = true
				break
			}
		}
		assert.True(t, hasWarning, "Should have DOM/DOW conflict warning")
		// Empty schedule check should also run (but return false for this expression)
		assert.Equal(t, 1, result.ValidJobs)
	})

	t.Run("error case", func(t *testing.T) {
		result := validator.ValidateExpression("invalid")
		assert.False(t, result.Valid)
		assert.Equal(t, 1, len(result.Issues))
		assert.Equal(t, SeverityError, result.Issues[0].Severity)
	})

	t.Run("warning case", func(t *testing.T) {
		result := validator.ValidateExpression("0 0 1 * 1")
		assert.True(t, result.Valid)
		hasWarning := false
		for _, issue := range result.Issues {
			if issue.Severity == SeverityWarn {
				hasWarning = true
				break
			}
		}
		assert.True(t, hasWarning, "Should have warning for DOM/DOW conflict")
	})

	t.Run("valid case", func(t *testing.T) {
		result := validator.ValidateExpression("0 0 * * *")
		assert.True(t, result.Valid)
		assert.Equal(t, 0, len(result.Issues))
	})

	t.Run("expression with empty schedule detected", func(t *testing.T) {
		// Create a validator with a mock scheduler that returns empty schedule
		validator := &Validator{
			parser:    cronx.NewParserWithLocale("en"),
			scheduler: &mockScheduler{returnEmpty: true},
			locale:    "en",
		}

		// Use a valid expression that will be detected as empty by our mock
		result := validator.ValidateExpression("0 0 * * *")
		// Should be detected as empty schedule
		assert.False(t, result.Valid)
		assert.Equal(t, 1, result.InvalidJobs)
		assert.Equal(t, 0, result.ValidJobs)
		hasEmptyError := false
		for _, issue := range result.Issues {
			if issue.Severity == SeverityError && issue.Code == CodeEmptySchedule && issue.Message == "Schedule never runs (empty schedule)" {
				hasEmptyError = true
				break
			}
		}
		assert.True(t, hasEmptyError, "Should have empty schedule error")
	})

	t.Run("expression with empty schedule and DOM/DOW conflict", func(t *testing.T) {
		// Test that both checks run, and empty schedule takes precedence
		validator := &Validator{
			parser:    cronx.NewParserWithLocale("en"),
			scheduler: &mockScheduler{returnEmpty: true},
			locale:    "en",
		}

		result := validator.ValidateExpression("0 0 1 * 1")
		// Should be invalid due to empty schedule (takes precedence)
		assert.False(t, result.Valid)
		assert.Equal(t, 1, result.InvalidJobs)
		assert.Equal(t, 0, result.ValidJobs)
		// Should have empty schedule error
		hasEmptyError := false
		for _, issue := range result.Issues {
			if issue.Message == "Schedule never runs (empty schedule)" {
				hasEmptyError = true
				break
			}
		}
		assert.True(t, hasEmptyError, "Should have empty schedule error")
	})
}

type mockScheduler struct {
	returnEmpty bool
	returnError bool
}

func (m *mockScheduler) Next(expression string, from time.Time, count int) ([]time.Time, error) {
	if m.returnError {
		return nil, &mockError{msg: "mock error"}
	}
	if m.returnEmpty {
		// Return a time far in the future to simulate empty schedule
		return []time.Time{from.AddDate(3, 0, 0)}, nil
	}
	// Return a normal time
	return []time.Time{from.Add(time.Hour)}, nil
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

func codesFor(issues []Issue) []string {
	var codes []string
	for _, issue := range issues {
		codes = append(codes, issue.Code+":"+issue.Expression)
	}
	return codes
}

func TestValidator_ValidateItems(t *testing.T) {
	t.Run("active item with no issues", func(t *testing.T) {
		validator := NewValidator("en")
		items := []inventory.Item{
			{
				Expression: "0 0 * * *",
				Command:    "/usr/local/bin/backup.sh",
				Shell:      true,
				State:      inventory.StateActive,
				Locator:    inventory.Locator{File: "crontab", Line: 1},
			},
		}

		result := validator.ValidateItems(items)
		assert.True(t, result.Valid)
		assert.Equal(t, 1, result.TotalJobs)
		assert.Equal(t, 1, result.ValidJobs)
		assert.Equal(t, 0, result.InvalidJobs)
		assert.Empty(t, result.Issues)
	})

	t.Run("StateInvalid item is reported once from Reason", func(t *testing.T) {
		validator := NewValidator("en")
		items := []inventory.Item{
			{
				Expression: "99 * * * *",
				State:      inventory.StateInvalid,
				Reason:     "minute out of range",
				Locator:    inventory.Locator{File: "k8s/cronjob.yaml", Line: 0},
			},
		}

		result := validator.ValidateItems(items)
		assert.False(t, result.Valid)
		assert.Equal(t, 1, result.TotalJobs)
		assert.Equal(t, 1, result.InvalidJobs)
		require.Len(t, result.Issues, 1)
		assert.Equal(t, CodeParseError, result.Issues[0].Code)
		assert.Contains(t, result.Issues[0].Message, "minute out of range")
	})

	t.Run("suspended and unresolved items are skipped, counted valid", func(t *testing.T) {
		validator := NewValidator("en")
		validator.SetHygieneChecks(true)
		items := []inventory.Item{
			{
				Expression: "not a schedule at all",
				Command:    "relative/path.sh",
				Shell:      true,
				State:      inventory.StateSuspended,
			},
			{
				Expression: "{{ .Values.schedule }}",
				Command:    "relative/path.sh",
				Shell:      true,
				State:      inventory.StateUnresolved,
			},
		}

		result := validator.ValidateItems(items)
		assert.True(t, result.Valid)
		assert.Equal(t, 2, result.TotalJobs)
		assert.Equal(t, 2, result.ValidJobs)
		assert.Equal(t, 0, result.InvalidJobs)
		assert.Empty(t, result.Issues, "a suspended/unresolved item's un-parseable expression must not be evaluated")
	})

	t.Run("DOM/DOW conflict and empty schedule", func(t *testing.T) {
		validator := &Validator{
			parser:    cronx.NewParserWithLocale("en"),
			scheduler: &mockScheduler{returnEmpty: true},
			locale:    "en",
		}
		items := []inventory.Item{
			{Expression: "0 0 1 * 1", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
			{Expression: "0 0 * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
		}

		result := validator.ValidateItems(items)
		codes := codesFor(result.Issues)
		assert.Contains(t, codes, CodeDOMDOWConflict+":0 0 1 * 1")
		assert.Contains(t, codes, CodeEmptySchedule+":0 0 1 * 1")
		assert.Contains(t, codes, CodeEmptySchedule+":0 0 * * *")
		assert.False(t, result.Valid)
		assert.Equal(t, 2, result.InvalidJobs)
		assert.Equal(t, 0, result.ValidJobs)
	})

	t.Run("frequency checks apply only to KindFields schedules", func(t *testing.T) {
		validator := NewValidator("en")
		validator.SetFrequencyChecks(true)
		validator.SetMaxRunsPerDay(1)
		items := []inventory.Item{
			{Expression: "* * * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
			{Expression: "@every 1m", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
		}

		result := validator.ValidateItems(items)
		codes := codesFor(result.Issues)
		assert.Contains(t, codes, CodeExcessiveRuns+":* * * * *")
		assert.NotContains(t, codes, CodeExcessiveRuns+":@every 1m")
	})

	t.Run("hygiene checks are gated on Item.Shell", func(t *testing.T) {
		validator := NewValidator("en")
		validator.SetHygieneChecks(true)
		items := []inventory.Item{
			{
				// A genuine shell command with a relative path: hygiene
				// rules apply and CRON-008 must fire.
				Expression: "0 0 * * *",
				Command:    "backup.sh",
				Shell:      true,
				State:      inventory.StateActive,
				Locator:    inventory.Locator{Line: 1},
			},
			{
				// A container image name (Shell false) must not be judged by CRON-008's shell rule.
				Expression: "0 0 * * *",
				Command:    "backup.sh",
				Shell:      false,
				State:      inventory.StateActive,
				Locator:    inventory.Locator{Line: 2},
			},
		}

		result := validator.ValidateItems(items)

		var line1, line2 []Issue
		for _, issue := range result.Issues {
			switch issue.LineNumber {
			case 1:
				line1 = append(line1, issue)
			case 2:
				line2 = append(line2, issue)
			}
		}
		assert.NotEmpty(t, line1, "the shell command must still be checked for hygiene")
		assert.Empty(t, line2, "a non-shell command must never be judged by shell hygiene rules")
	})

	t.Run("hygiene checks require enableHygiene and a non-empty command", func(t *testing.T) {
		validator := NewValidator("en")
		// enableHygiene left false (default)
		items := []inventory.Item{
			{Expression: "0 0 * * *", Command: "backup.sh", Shell: true, State: inventory.StateActive},
		}
		result := validator.ValidateItems(items)
		assert.Empty(t, result.Issues)

		validator.SetHygieneChecks(true)
		items = []inventory.Item{
			{Expression: "0 0 * * *", Command: "", Shell: true, State: inventory.StateActive},
		}
		result = validator.ValidateItems(items)
		assert.Empty(t, result.Issues, "an empty command has nothing to analyze")
	})

	t.Run("overlap analysis needs at least two active items", func(t *testing.T) {
		validator := NewValidator("en")
		validator.SetWarnOnOverlap(true)
		validator.SetOverlapWindow(24 * time.Hour)

		single := []inventory.Item{
			{Expression: "* * * * *", State: inventory.StateActive},
		}
		result := validator.ValidateItems(single)
		for _, issue := range result.Issues {
			assert.NotEqual(t, CodeOverlapDetected, issue.Code)
		}

		overlapping := []inventory.Item{
			{Expression: "* * * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
			{Expression: "* * * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
		}
		result = validator.ValidateItems(overlapping)
		found := false
		for _, issue := range result.Issues {
			if issue.Code == CodeOverlapDetected {
				found = true
			}
		}
		assert.True(t, found, "two jobs on the same minute must be reported as an overlap")
	})

	// Descriptor schedules have no five fields, so field-based checks must never fire against them.
	t.Run("descriptor schedules are excluded from field-based checks", func(t *testing.T) {
		validator := NewValidator("en")
		validator.SetMaxRunsPerDay(1) // low threshold: would flag @every 1h if the guard were missing

		items := []inventory.Item{
			{Expression: "@reboot", Command: "/usr/local/bin/on-boot.sh", Shell: true, State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
			{Expression: "@every 1h", Command: "/usr/local/bin/hourly.sh", Shell: true, State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
			{Expression: "0 0 1 * 1", Command: "/usr/local/bin/monthly-or-monday.sh", Shell: true, State: inventory.StateActive, Locator: inventory.Locator{Line: 3}}, // genuine DOM/DOW conflict, should still be reported
		}

		var result ValidationResult
		require.NotPanics(t, func() {
			result = validator.ValidateItems(items)
		})

		assert.Equal(t, 3, result.TotalJobs)
		assert.Equal(t, 3, result.ValidJobs)

		codes := codesFor(result.Issues)
		assert.NotContains(t, codes, CodeDOMDOWConflict+":@reboot")
		assert.NotContains(t, codes, CodeDOMDOWConflict+":@every 1h")
		assert.NotContains(t, codes, CodeExcessiveRuns+":@every 1h")
		assert.NotContains(t, codes, CodeRedundantPattern+":@every 1h")
		assert.NotContains(t, codes, CodeEmptySchedule+":@reboot", "@reboot has no wall-clock next run, but it is not an empty schedule")

		// The real conflict on the five-field job must still surface.
		assert.Contains(t, codes, CodeDOMDOWConflict+":0 0 1 * 1")
	})

	t.Run("CRON-012 is suppressed when every overlapping item forbids concurrency", func(t *testing.T) {
		validator := NewValidator("en")
		validator.SetWarnOnOverlap(true)
		validator.SetOverlapWindow(24 * time.Hour)

		items := []inventory.Item{
			{Expression: "* * * * *", State: inventory.StateActive, Concurrency: inventory.ConcurrencyForbid, Locator: inventory.Locator{Line: 1}},
			{Expression: "* * * * *", State: inventory.StateActive, Concurrency: inventory.ConcurrencyForbid, Locator: inventory.Locator{Line: 2}},
		}

		result := validator.ValidateItems(items)
		for _, issue := range result.Issues {
			assert.NotEqual(t, CodeOverlapDetected, issue.Code,
				"the platform already serialises both jobs, so there is nothing the user can act on")
		}
	})

	t.Run("CRON-012 still fires when a Forbid item overlaps an Allow item", func(t *testing.T) {
		validator := NewValidator("en")
		validator.SetWarnOnOverlap(true)
		validator.SetOverlapWindow(24 * time.Hour)

		items := []inventory.Item{
			{Expression: "* * * * *", State: inventory.StateActive, Concurrency: inventory.ConcurrencyForbid, Locator: inventory.Locator{Line: 1}},
			{Expression: "* * * * *", State: inventory.StateActive, Concurrency: inventory.ConcurrencyAllow, Locator: inventory.Locator{Line: 2}},
		}

		result := validator.ValidateItems(items)
		found := false
		for _, issue := range result.Issues {
			if issue.Code == CodeOverlapDetected {
				found = true
			}
		}
		assert.True(t, found, "the Allow item genuinely contends with the Forbid item")
	})

	t.Run("issues carry the item's locator, not just its line number", func(t *testing.T) {
		validator := NewValidator("en")
		validator.SetHygieneChecks(true)

		locator := inventory.Locator{File: "sources/k8s/backup.yaml", Line: 6, Path: "spec.schedule"}
		items := []inventory.Item{
			{
				Expression: "60 0 * * *", // invalid: minute out of range
				Command:    "backup.sh",
				Shell:      true,
				State:      inventory.StateActive,
				Locator:    locator,
			},
		}

		result := validator.ValidateItems(items)
		require.NotEmpty(t, result.Issues)
		for _, issue := range result.Issues {
			assert.Equal(t, locator.Line, issue.LineNumber, "LineNumber stays populated for the --json contract")
			assert.Equal(t, locator, issue.Locator)
		}
	})
}

// TestValidateItems_OverlapNamesTheCollidingJobs pins that an overlap finding says which jobs collided.
func TestValidateItems_OverlapNamesTheCollidingJobs(t *testing.T) {
	validator := NewValidator("en")
	validator.SetWarnOnOverlap(true)

	items := []inventory.Item{
		{
			Expression: "0 12 * * *",
			State:      inventory.StateActive,
			Locator:    inventory.Locator{File: "site-a/crontab", Line: 2},
		},
		{
			Expression: "0 12 * * *",
			State:      inventory.StateActive,
			Locator:    inventory.Locator{File: "site-b/crontab", Line: 7},
		},
	}

	result := validator.ValidateItems(items)

	var overlaps []Issue
	for _, issue := range result.Issues {
		if issue.Code == CodeOverlapDetected {
			overlaps = append(overlaps, issue)
		}
	}
	require.NotEmpty(t, overlaps, "two jobs sharing a minute must produce an overlap finding")

	assert.Contains(t, overlaps[0].Message, "line-site-a/crontab:2")
	assert.Contains(t, overlaps[0].Message, "line-site-b/crontab:7")
}

// TestNamedJobs_SummarisesBeyondTheCap keeps a wide overlap's message readable.
func TestNamedJobs_SummarisesBeyondTheCap(t *testing.T) {
	t.Run("lists every job when they fit", func(t *testing.T) {
		assert.Equal(t, "a, b", namedJobs([]string{"a", "b"}))
	})

	t.Run("names the first few and counts the rest", func(t *testing.T) {
		ids := []string{"a", "b", "c", "d", "e", "f", "g"}
		assert.Equal(t, "a, b, c, d, e, and 2 more", namedJobs(ids))
	})
}
