package check

import (
	"fmt"
	"strings"
	"time"

	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/inventory"
)

// Issue represents a validation issue found in a cron expression or crontab
type Issue struct {
	Severity   Severity // Severity level (info, warn, error)
	Code       string   // Diagnostic code (e.g., "CRON-001")
	LineNumber int      // 0 for single expression checks; mirrors Locator.Line for an item-derived issue
	// Locator says where the item came from; zero value for a single-expression or overlap issue.
	// LineNumber stays populated alongside it as part of the --json contract.
	Locator    inventory.Locator
	Expression string // The cron expression (if applicable)
	Message    string // Human-readable issue description
	Hint       string // Optional fix suggestion
}

// ValidationResult contains the results of validating a cron expression or crontab
type ValidationResult struct {
	Valid       bool
	Issues      []Issue
	TotalJobs   int
	ValidJobs   int
	InvalidJobs int
}

// Validator provides validation functionality for cron expressions and crontabs
type Validator struct {
	parser          cronx.Parser
	scheduler       cronx.Scheduler
	locale          string
	enableFrequency bool
	maxRunsPerDay   int
	enableHygiene   bool
	warnOnOverlap   bool
	overlapWindow   time.Duration
}

// NewValidator creates a new validator instance
func NewValidator(locale string) *Validator {
	return &Validator{
		parser:          cronx.NewParserWithLocale(locale),
		scheduler:       cronx.NewScheduler(),
		locale:          locale,
		enableFrequency: true,                 // Default: enabled
		maxRunsPerDay:   1000,                 // Default threshold
		warnOnOverlap:   false,                // Default: disabled
		overlapWindow:   DefaultOverlapWindow, // Default: 24 hours
	}
}

// SetFrequencyChecks enables or disables frequency analysis
func (v *Validator) SetFrequencyChecks(enabled bool) {
	v.enableFrequency = enabled
}

// SetMaxRunsPerDay sets the threshold for excessive runs warning
func (v *Validator) SetMaxRunsPerDay(threshold int) {
	v.maxRunsPerDay = threshold
}

// SetHygieneChecks enables or disables command hygiene checks
func (v *Validator) SetHygieneChecks(enabled bool) {
	v.enableHygiene = enabled
}

// SetWarnOnOverlap enables or disables overlap warnings
func (v *Validator) SetWarnOnOverlap(enabled bool) {
	v.warnOnOverlap = enabled
}

// SetOverlapWindow sets the time window for overlap analysis
func (v *Validator) SetOverlapWindow(window time.Duration) {
	v.overlapWindow = window
}

// ValidateExpression validates a single cron expression
func (v *Validator) ValidateExpression(expression string) ValidationResult {
	result := ValidationResult{
		Valid:     true,
		TotalJobs: 1,
		Issues:    []Issue{},
	}

	// Parse the expression
	schedule, err := v.parser.Parse(expression)
	if err != nil {
		result.Valid = false
		result.InvalidJobs = 1
		result.Issues = append(result.Issues, Issue{
			Severity:   SeverityError,
			Code:       CodeParseError,
			LineNumber: 0,
			Expression: expression,
			Message:    fmt.Sprintf("Invalid cron expression: %s", err.Error()),
			Hint:       GetCodeHint(CodeParseError),
		})
		return result
	}

	// Expression is valid, check for warnings
	result.ValidJobs = 1

	// Check for DOM/DOW conflict -- both fields only exist on five-field
	// schedules; @every and @reboot have no fields to compare.
	if schedule.Kind == cronx.KindFields && detectDOMDOWConflict(schedule) {
		result.Issues = append(result.Issues, Issue{
			Severity:   SeverityWarn,
			Code:       CodeDOMDOWConflict,
			LineNumber: 0,
			Expression: expression,
			Message:    "Both day-of-month and day-of-week specified (runs if either condition is met)",
			Hint:       GetCodeHint(CodeDOMDOWConflict),
		})
	}

	// Check for empty schedule
	if detectEmptySchedule(expression, schedule.Kind, v.scheduler) {
		result.Valid = false
		result.InvalidJobs = 1
		result.ValidJobs = 0
		result.Issues = append(result.Issues, Issue{
			Severity:   SeverityError,
			Code:       CodeEmptySchedule,
			LineNumber: 0,
			Expression: expression,
			Message:    "Schedule never runs (empty schedule)",
			Hint:       GetCodeHint(CodeEmptySchedule),
		})
	}

	// Frequency analysis (if enabled) -- redundant-pattern and excessive-run
	// detection both read the five fields, which @every and @reboot don't have.
	if v.enableFrequency && schedule.Kind == cronx.KindFields {
		freqIssues := v.validateFrequency(schedule, expression)
		result.Issues = append(result.Issues, freqIssues...)
	}

	return result
}

// ValidateItems validates a slice of inventory items from any source cronkit
// scan can discover; callers must run items through inventory.ResolveTimezones
// first, and a suspended or unresolved item is counted valid and skipped
// without further checks since its schedule isn't evaluable right now.
func (v *Validator) ValidateItems(items []inventory.Item) ValidationResult {
	result := ValidationResult{
		Valid:     true,
		Issues:    []Issue{},
		TotalJobs: 0,
		ValidJobs: 0,
	}

	for _, item := range items {
		result.TotalJobs++

		if item.State == inventory.StateInvalid {
			result.Valid = false
			result.InvalidJobs++
			result.Issues = append(result.Issues, Issue{
				Severity:   SeverityError,
				Code:       CodeParseError,
				LineNumber: item.Locator.Line,
				Locator:    item.Locator,
				Expression: item.Expression,
				Message:    fmt.Sprintf("Invalid cron expression: %s", item.Reason),
				Hint:       GetCodeHint(CodeParseError),
			})
			continue
		}

		if item.State != inventory.StateActive {
			result.ValidJobs++
			continue
		}

		// Parse the schedule for additional checks
		schedule, err := v.parser.Parse(item.Expression)
		if err != nil {
			// This shouldn't happen for a StateActive item, but handle it anyway
			result.Valid = false
			result.InvalidJobs++
			result.Issues = append(result.Issues, Issue{
				Severity:   SeverityError,
				Code:       CodeParseError,
				LineNumber: item.Locator.Line,
				Locator:    item.Locator,
				Expression: item.Expression,
				Message:    fmt.Sprintf("Failed to parse expression: %s", err.Error()),
				Hint:       GetCodeHint(CodeParseError),
			})
			continue
		}

		result.ValidJobs++

		// Check for DOM/DOW conflict -- both fields only exist on
		// five-field schedules; @every and @reboot have no fields to compare.
		if schedule.Kind == cronx.KindFields && detectDOMDOWConflict(schedule) {
			result.Issues = append(result.Issues, Issue{
				Severity:   SeverityWarn,
				Code:       CodeDOMDOWConflict,
				LineNumber: item.Locator.Line,
				Locator:    item.Locator,
				Expression: item.Expression,
				Message:    "Both day-of-month and day-of-week specified (runs if either condition is met)",
				Hint:       GetCodeHint(CodeDOMDOWConflict),
			})
		}

		// Check for empty schedule
		if detectEmptySchedule(item.Expression, schedule.Kind, v.scheduler) {
			result.Valid = false
			result.InvalidJobs++
			result.ValidJobs--
			result.Issues = append(result.Issues, Issue{
				Severity:   SeverityError,
				Code:       CodeEmptySchedule,
				LineNumber: item.Locator.Line,
				Locator:    item.Locator,
				Expression: item.Expression,
				Message:    "Schedule never runs (empty schedule)",
				Hint:       GetCodeHint(CodeEmptySchedule),
			})
		}

		// Frequency analysis reads the five fields, which @every and @reboot don't have.
		if v.enableFrequency && schedule.Kind == cronx.KindFields {
			freqIssues := v.validateFrequency(schedule, item.Expression)
			for i := range freqIssues {
				freqIssues[i].LineNumber = item.Locator.Line
				freqIssues[i].Locator = item.Locator
			}
			result.Issues = append(result.Issues, freqIssues...)
		}

		// Command hygiene checks (CRON-008..CRON-011) apply only when Item.Shell marks Command as shell.
		if v.enableHygiene && item.Shell && item.Command != "" {
			result.Issues = append(result.Issues, hygieneIssues(item.Command, item.Locator, item.Expression)...)
		}
	}

	// Overlap analysis (if enabled) -- only for multiple active items
	if v.warnOnOverlap && len(items) > 1 {
		result.Issues = append(result.Issues, v.validateOverlapItems(items)...)
	}

	return result
}

// validateOverlapItems runs overlap analysis directly on items that have
// already been admitted (run through inventory.ResolveTimezones).
func (v *Validator) validateOverlapItems(items []inventory.Item) []Issue {
	var issues []Issue

	activeCount := 0
	for _, item := range items {
		if item.State == inventory.StateActive {
			activeCount++
		}
	}
	if activeCount < 2 {
		return issues // Need at least 2 active items for overlaps
	}

	_, stats, err := AnalyzeOverlaps(items, time.Now(), v.overlapWindow, v.scheduler, v.parser)
	if err != nil {
		return issues // Skip if analysis fails
	}

	if stats.MaxConcurrent > 1 {
		for _, overlap := range stats.MostProblematic[:min(5, len(stats.MostProblematic))] {
			issues = append(issues, Issue{
				Severity:   SeverityWarn,
				Code:       CodeOverlapDetected,
				LineNumber: 0, // Overlap involves multiple jobs
				Expression: "",
				Message: fmt.Sprintf("Overlap detected: %d jobs scheduled at %s: %s",
					overlap.Count, overlap.Time.Format("2006-01-02 15:04"), namedJobs(overlap.JobIDs)),
				Hint: GetCodeHint(CodeOverlapDetected),
			})
		}
	}

	return issues
}

// maxNamedJobs bounds how many job ids one overlap message spells out.
const maxNamedJobs = 5

// namedJobs lists the jobs an overlap involves, since the count alone does not say which.
func namedJobs(jobIDs []string) string {
	named := jobIDs
	suffix := ""
	if len(named) > maxNamedJobs {
		named = named[:maxNamedJobs]
		suffix = fmt.Sprintf(", and %d more", len(jobIDs)-maxNamedJobs)
	}
	return strings.Join(named, ", ") + suffix
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// hygieneIssues runs command hygiene analysis and stamps each issue with its locator and expression.
func hygieneIssues(command string, locator inventory.Locator, expression string) []Issue {
	issues := AnalyzeCommand(command)
	for i := range issues {
		issues[i].LineNumber = locator.Line
		issues[i].Locator = locator
		issues[i].Expression = expression
	}
	return issues
}

// validateFrequency performs frequency analysis on a schedule
func (v *Validator) validateFrequency(schedule *cronx.Schedule, expression string) []Issue {
	var issues []Issue

	// Check for redundant patterns (e.g., */1)
	if DetectRedundantPattern(schedule) {
		suggestion := GetRedundantPatternSuggestion(expression, schedule)
		issues = append(issues, Issue{
			Severity:   SeverityWarn,
			Code:       CodeRedundantPattern,
			LineNumber: 0, // Will be set by caller
			Expression: expression,
			Message:    "Redundant step pattern detected (e.g., */1 can be simplified to *)",
			Hint:       fmt.Sprintf("%s Consider using: %s", GetCodeHint(CodeRedundantPattern), suggestion),
		})
	}

	// Check for excessive run counts
	runsPerDay, err := CalculateRunsPerDay(expression, v.scheduler)
	if err == nil && runsPerDay > v.maxRunsPerDay {
		issues = append(issues, Issue{
			Severity:   SeverityWarn,
			Code:       CodeExcessiveRuns,
			LineNumber: 0, // Will be set by caller
			Expression: expression,
			Message:    fmt.Sprintf("Schedule runs %d times per day (exceeds threshold of %d)", runsPerDay, v.maxRunsPerDay),
			Hint:       GetCodeHint(CodeExcessiveRuns),
		})
	}

	return issues
}

// detectDOMDOWConflict checks if both day-of-month and day-of-week are
// specified; call only for schedule.Kind == cronx.KindFields — those fields are nil for @every/@reboot.
func detectDOMDOWConflict(schedule *cronx.Schedule) bool {
	// Both DOM and DOW are specified (not wildcards)
	return !schedule.DayOfMonth.IsEvery() && !schedule.DayOfWeek.IsEvery()
}

// detectEmptySchedule checks if a schedule never runs; KindReboot is exempt (Next reports zero times).
func detectEmptySchedule(expression string, kind cronx.ScheduleKind, scheduler cronx.Scheduler) bool {
	if kind == cronx.KindReboot {
		return false
	}

	now := time.Now()
	future := now.AddDate(2, 0, 0) // Check 2 years ahead

	times, err := scheduler.Next(expression, now, 1)
	if err != nil {
		return true // Invalid = empty
	}

	// If no times found or first time is beyond our check window
	if len(times) == 0 || times[0].After(future) {
		return true
	}

	return false
}
