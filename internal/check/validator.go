package check

import (
	"fmt"
	"time"

	"github.com/hzerrad/cronic/internal/crontab"
	"github.com/hzerrad/cronic/internal/cronx"
)

// Issue represents a validation issue found in a cron expression or crontab
type Issue struct {
	Severity   Severity // Severity level (info, warn, error)
	Code       string   // Diagnostic code (e.g., "CRON-001")
	LineNumber int      // 0 for single expression checks
	Expression string   // The cron expression (if applicable)
	Message    string   // Human-readable issue description
	Hint       string   // Optional fix suggestion
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
		enableFrequency: true,           // Default: enabled
		maxRunsPerDay:   1000,           // Default threshold
		warnOnOverlap:   false,          // Default: disabled
		overlapWindow:   24 * time.Hour, // Default: 24 hours
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

	// Check for DOM/DOW conflict
	if detectDOMDOWConflict(schedule) {
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
	if detectEmptySchedule(expression, v.scheduler) {
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

	// Frequency analysis (if enabled)
	if v.enableFrequency {
		freqIssues := v.validateFrequency(schedule, expression)
		result.Issues = append(result.Issues, freqIssues...)
	}

	return result
}

// ValidateCrontab validates a crontab file
func (v *Validator) ValidateCrontab(reader crontab.Reader, path string) ValidationResult {
	result := ValidationResult{
		Valid:     true,
		Issues:    []Issue{},
		TotalJobs: 0,
		ValidJobs: 0,
	}

	// Read all entries from the file
	entries, err := reader.ParseFile(path)
	if err != nil {
		result.Valid = false
		result.Issues = append(result.Issues, Issue{
			Severity:   SeverityError,
			Code:       CodeFileReadError,
			LineNumber: 0,
			Expression: "",
			Message:    fmt.Sprintf("Failed to read crontab file: %s", err.Error()),
			Hint:       GetCodeHint(CodeFileReadError),
		})
		return result
	}

	// Validate each job entry
	for _, entry := range entries {
		if entry.Type != crontab.EntryTypeJob || entry.Job == nil {
			continue
		}

		result.TotalJobs++

		// Check if the job is valid
		if !entry.Job.Valid {
			result.Valid = false
			result.InvalidJobs++
			result.Issues = append(result.Issues, Issue{
				Severity:   SeverityError,
				Code:       CodeParseError,
				LineNumber: entry.Job.LineNumber,
				Expression: entry.Job.Expression,
				Message:    fmt.Sprintf("Invalid cron expression: %s", entry.Job.Error),
				Hint:       GetCodeHint(CodeParseError),
			})
			continue
		}

		// Parse the schedule for additional checks
		schedule, err := v.parser.Parse(entry.Job.Expression)
		if err != nil {
			// This shouldn't happen if Valid is true, but handle it anyway
			result.Valid = false
			result.InvalidJobs++
			result.ValidJobs--
			result.Issues = append(result.Issues, Issue{
				Severity:   SeverityError,
				Code:       CodeParseError,
				LineNumber: entry.Job.LineNumber,
				Expression: entry.Job.Expression,
				Message:    fmt.Sprintf("Failed to parse expression: %s", err.Error()),
				Hint:       GetCodeHint(CodeParseError),
			})
			continue
		}

		result.ValidJobs++

		// Check for DOM/DOW conflict
		if detectDOMDOWConflict(schedule) {
			result.Issues = append(result.Issues, Issue{
				Severity:   SeverityWarn,
				Code:       CodeDOMDOWConflict,
				LineNumber: entry.Job.LineNumber,
				Expression: entry.Job.Expression,
				Message:    "Both day-of-month and day-of-week specified (runs if either condition is met)",
				Hint:       GetCodeHint(CodeDOMDOWConflict),
			})
		}

		// Check for empty schedule
		if detectEmptySchedule(entry.Job.Expression, v.scheduler) {
			result.Valid = false
			result.InvalidJobs++
			result.ValidJobs--
			result.Issues = append(result.Issues, Issue{
				Severity:   SeverityError,
				Code:       CodeEmptySchedule,
				LineNumber: entry.Job.LineNumber,
				Expression: entry.Job.Expression,
				Message:    "Schedule never runs (empty schedule)",
				Hint:       GetCodeHint(CodeEmptySchedule),
			})
		}

		// Frequency analysis (if enabled)
		if v.enableFrequency {
			freqIssues := v.validateFrequency(schedule, entry.Job.Expression)
			for i := range freqIssues {
				freqIssues[i].LineNumber = entry.Job.LineNumber
			}
			result.Issues = append(result.Issues, freqIssues...)
		}

		// Command hygiene checks (if enabled)
		if v.enableHygiene && entry.Job.Command != "" {
			hygieneIssues := v.validateCommandHygiene(entry.Job)
			result.Issues = append(result.Issues, hygieneIssues...)
		}
	}

	// Overlap analysis (if enabled) - only for crontab validation
	if v.warnOnOverlap && len(entries) > 1 {
		overlapIssues := v.validateOverlaps(entries)
		result.Issues = append(result.Issues, overlapIssues...)
	}

	return result
}

// ValidateEntries validates a slice of crontab entries (e.g., from stdin)
func (v *Validator) ValidateEntries(entries []*crontab.Entry) ValidationResult {
	result := ValidationResult{
		Valid:     true,
		Issues:    []Issue{},
		TotalJobs: 0,
		ValidJobs: 0,
	}

	// Validate each job entry
	for _, entry := range entries {
		if entry.Type != crontab.EntryTypeJob || entry.Job == nil {
			continue
		}

		result.TotalJobs++

		if !entry.Job.Valid {
			result.Valid = false
			result.InvalidJobs++
			result.Issues = append(result.Issues, Issue{
				Severity:   SeverityError,
				Code:       CodeParseError,
				LineNumber: entry.Job.LineNumber,
				Expression: entry.Job.Expression,
				Message:    fmt.Sprintf("Invalid cron expression: %s", entry.Job.Error),
				Hint:       GetCodeHint(CodeParseError),
			})
			continue
		}

		// Parse the schedule for additional checks
		schedule, err := v.parser.Parse(entry.Job.Expression)
		if err != nil {
			// This shouldn't happen if Valid is true, but handle it anyway
			result.Valid = false
			result.InvalidJobs++
			result.ValidJobs--
			result.Issues = append(result.Issues, Issue{
				Severity:   SeverityError,
				Code:       CodeParseError,
				LineNumber: entry.Job.LineNumber,
				Expression: entry.Job.Expression,
				Message:    fmt.Sprintf("Failed to parse expression: %s", err.Error()),
				Hint:       GetCodeHint(CodeParseError),
			})
			continue
		}

		result.ValidJobs++

		// Check for DOM/DOW conflict
		if detectDOMDOWConflict(schedule) {
			result.Issues = append(result.Issues, Issue{
				Severity:   SeverityWarn,
				Code:       CodeDOMDOWConflict,
				LineNumber: entry.Job.LineNumber,
				Expression: entry.Job.Expression,
				Message:    "Both day-of-month and day-of-week specified (runs if either condition is met)",
				Hint:       GetCodeHint(CodeDOMDOWConflict),
			})
		}

		// Check for empty schedule
		if detectEmptySchedule(entry.Job.Expression, v.scheduler) {
			result.Valid = false
			result.InvalidJobs++
			result.ValidJobs--
			result.Issues = append(result.Issues, Issue{
				Severity:   SeverityError,
				Code:       CodeEmptySchedule,
				LineNumber: entry.Job.LineNumber,
				Expression: entry.Job.Expression,
				Message:    "Schedule never runs (empty schedule)",
				Hint:       GetCodeHint(CodeEmptySchedule),
			})
		}

		// Frequency analysis (if enabled)
		if v.enableFrequency {
			freqIssues := v.validateFrequency(schedule, entry.Job.Expression)
			for i := range freqIssues {
				freqIssues[i].LineNumber = entry.Job.LineNumber
			}
			result.Issues = append(result.Issues, freqIssues...)
		}

		// Command hygiene checks (if enabled)
		if v.enableHygiene && entry.Job.Command != "" {
			hygieneIssues := v.validateCommandHygiene(entry.Job)
			result.Issues = append(result.Issues, hygieneIssues...)
		}
	}

	// Overlap analysis (if enabled) - only for multiple entries
	if v.warnOnOverlap && len(entries) > 1 {
		overlapIssues := v.validateOverlaps(entries)
		result.Issues = append(result.Issues, overlapIssues...)
	}

	return result
}

// ValidateUserCrontab validates the current user's crontab
func (v *Validator) ValidateUserCrontab(reader crontab.Reader) ValidationResult {
	result := ValidationResult{
		Valid:     true,
		Issues:    []Issue{},
		TotalJobs: 0,
		ValidJobs: 0,
	}

	// Read user's crontab
	jobs, err := reader.ReadUser()
	if err != nil {
		result.Valid = false
		result.Issues = append(result.Issues, Issue{
			Severity:   SeverityError,
			Code:       CodeFileReadError,
			LineNumber: 0,
			Expression: "",
			Message:    fmt.Sprintf("Failed to read user crontab: %s", err.Error()),
			Hint:       GetCodeHint(CodeFileReadError),
		})
		return result
	}

	// Validate each job
	for _, job := range jobs {
		result.TotalJobs++

		if !job.Valid {
			result.Valid = false
			result.InvalidJobs++
			result.Issues = append(result.Issues, Issue{
				Severity:   SeverityError,
				Code:       CodeParseError,
				LineNumber: job.LineNumber,
				Expression: job.Expression,
				Message:    fmt.Sprintf("Invalid cron expression: %s", job.Error),
				Hint:       GetCodeHint(CodeParseError),
			})
			continue
		}

		// Parse the schedule for additional checks
		schedule, err := v.parser.Parse(job.Expression)
		if err != nil {
			result.Valid = false
			result.InvalidJobs++
			result.ValidJobs--
			result.Issues = append(result.Issues, Issue{
				Severity:   SeverityError,
				Code:       CodeParseError,
				LineNumber: job.LineNumber,
				Expression: job.Expression,
				Message:    fmt.Sprintf("Failed to parse expression: %s", err.Error()),
				Hint:       GetCodeHint(CodeParseError),
			})
			continue
		}

		result.ValidJobs++

		// Check for DOM/DOW conflict
		if detectDOMDOWConflict(schedule) {
			result.Issues = append(result.Issues, Issue{
				Severity:   SeverityWarn,
				Code:       CodeDOMDOWConflict,
				LineNumber: job.LineNumber,
				Expression: job.Expression,
				Message:    "Both day-of-month and day-of-week specified (runs if either condition is met)",
				Hint:       GetCodeHint(CodeDOMDOWConflict),
			})
		}

		// Check for empty schedule
		if detectEmptySchedule(job.Expression, v.scheduler) {
			result.Valid = false
			result.InvalidJobs++
			result.ValidJobs--
			result.Issues = append(result.Issues, Issue{
				Severity:   SeverityError,
				Code:       CodeEmptySchedule,
				LineNumber: job.LineNumber,
				Expression: job.Expression,
				Message:    "Schedule never runs (empty schedule)",
				Hint:       GetCodeHint(CodeEmptySchedule),
			})
		}

		// Frequency analysis (if enabled)
		if v.enableFrequency {
			freqIssues := v.validateFrequency(schedule, job.Expression)
			for i := range freqIssues {
				freqIssues[i].LineNumber = job.LineNumber
			}
			result.Issues = append(result.Issues, freqIssues...)
		}

		// Command hygiene checks (if enabled)
		if v.enableHygiene && job.Command != "" {
			hygieneIssues := v.validateCommandHygiene(job)
			result.Issues = append(result.Issues, hygieneIssues...)
		}
	}

	// Overlap analysis (if enabled) - only for multiple jobs
	if v.warnOnOverlap && len(jobs) > 1 {
		// Convert jobs to entries for overlap validation
		entries := make([]*crontab.Entry, 0, len(jobs))
		for _, job := range jobs {
			entries = append(entries, &crontab.Entry{
				Type:       crontab.EntryTypeJob,
				LineNumber: job.LineNumber,
				Job:        job,
			})
		}
		overlapIssues := v.validateOverlaps(entries)
		result.Issues = append(result.Issues, overlapIssues...)
	}

	return result
}

// validateOverlaps performs overlap analysis on a set of job entries
func (v *Validator) validateOverlaps(entries []*crontab.Entry) []Issue {
	var issues []Issue

	// Collect valid jobs
	jobs := make([]*crontab.Job, 0)
	for _, entry := range entries {
		if entry.Type == crontab.EntryTypeJob && entry.Job != nil && entry.Job.Valid {
			jobs = append(jobs, entry.Job)
		}
	}

	if len(jobs) < 2 {
		return issues // Need at least 2 jobs for overlaps
	}

	// Analyze overlaps
	_, stats, err := AnalyzeOverlaps(jobs, v.overlapWindow, v.scheduler, v.parser)
	if err != nil {
		return issues // Skip if analysis fails
	}

	// Report overlaps if they exist
	if stats.MaxConcurrent > 1 {
		// Report the most problematic overlaps
		for _, overlap := range stats.MostProblematic[:min(5, len(stats.MostProblematic))] {
			issues = append(issues, Issue{
				Severity:   SeverityWarn,
				Code:       CodeOverlapDetected,
				LineNumber: 0, // Overlap involves multiple jobs
				Expression: "",
				Message:    fmt.Sprintf("Overlap detected: %d jobs scheduled at %s", overlap.Count, overlap.Time.Format("2006-01-02 15:04")),
				Hint:       GetCodeHint(CodeOverlapDetected),
			})
		}
	}

	return issues
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// validateCommandHygiene performs command hygiene analysis
func (v *Validator) validateCommandHygiene(job *crontab.Job) []Issue {
	issues := AnalyzeCommand(job.Command)
	// Set line number and expression for all issues
	for i := range issues {
		issues[i].LineNumber = job.LineNumber
		issues[i].Expression = job.Expression
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

// detectDOMDOWConflict checks if both day-of-month and day-of-week are specified
func detectDOMDOWConflict(schedule *cronx.Schedule) bool {
	// Both DOM and DOW are specified (not wildcards)
	return !schedule.DayOfMonth.IsEvery() && !schedule.DayOfWeek.IsEvery()
}

// detectEmptySchedule checks if a schedule never runs
func detectEmptySchedule(expression string, scheduler cronx.Scheduler) bool {
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
