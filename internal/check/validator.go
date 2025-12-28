package check

import (
	"fmt"
	"time"

	"github.com/hzerrad/cronic/internal/crontab"
	"github.com/hzerrad/cronic/internal/cronx"
)

// Issue represents a validation issue found in a cron expression or crontab
type Issue struct {
	Type       string // "error", "warning", "info"
	LineNumber int    // 0 for single expression checks
	Expression string // The cron expression (if applicable)
	Message    string // Human-readable issue description
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
	parser    cronx.Parser
	scheduler cronx.Scheduler
	locale    string
}

// NewValidator creates a new validator instance
func NewValidator(locale string) *Validator {
	return &Validator{
		parser:    cronx.NewParserWithLocale(locale),
		scheduler: cronx.NewScheduler(),
		locale:    locale,
	}
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
			Type:       "error",
			LineNumber: 0,
			Expression: expression,
			Message:    fmt.Sprintf("Invalid cron expression: %s", err.Error()),
		})
		return result
	}

	// Expression is valid, check for warnings
	result.ValidJobs = 1

	// Check for DOM/DOW conflict
	if detectDOMDOWConflict(schedule) {
		result.Issues = append(result.Issues, Issue{
			Type:       "warning",
			LineNumber: 0,
			Expression: expression,
			Message:    "Both day-of-month and day-of-week specified (runs if either condition is met)",
		})
	}

	// Check for empty schedule
	if detectEmptySchedule(expression, v.scheduler) {
		result.Valid = false
		result.InvalidJobs = 1
		result.ValidJobs = 0
		result.Issues = append(result.Issues, Issue{
			Type:       "error",
			LineNumber: 0,
			Expression: expression,
			Message:    "Schedule never runs (empty schedule)",
		})
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
			Type:       "error",
			LineNumber: 0,
			Expression: "",
			Message:    fmt.Sprintf("Failed to read crontab file: %s", err.Error()),
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
				Type:       "error",
				LineNumber: entry.Job.LineNumber,
				Expression: entry.Job.Expression,
				Message:    fmt.Sprintf("Invalid cron expression: %s", entry.Job.Error),
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
				Type:       "error",
				LineNumber: entry.Job.LineNumber,
				Expression: entry.Job.Expression,
				Message:    fmt.Sprintf("Failed to parse expression: %s", err.Error()),
			})
			continue
		}

		result.ValidJobs++

		// Check for DOM/DOW conflict
		if detectDOMDOWConflict(schedule) {
			result.Issues = append(result.Issues, Issue{
				Type:       "warning",
				LineNumber: entry.Job.LineNumber,
				Expression: entry.Job.Expression,
				Message:    "Both day-of-month and day-of-week specified (runs if either condition is met)",
			})
		}

		// Check for empty schedule
		if detectEmptySchedule(entry.Job.Expression, v.scheduler) {
			result.Valid = false
			result.InvalidJobs++
			result.ValidJobs--
			result.Issues = append(result.Issues, Issue{
				Type:       "error",
				LineNumber: entry.Job.LineNumber,
				Expression: entry.Job.Expression,
				Message:    "Schedule never runs (empty schedule)",
			})
		}
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
			Type:       "error",
			LineNumber: 0,
			Expression: "",
			Message:    fmt.Sprintf("Failed to read user crontab: %s", err.Error()),
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
				Type:       "error",
				LineNumber: job.LineNumber,
				Expression: job.Expression,
				Message:    fmt.Sprintf("Invalid cron expression: %s", job.Error),
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
				Type:       "error",
				LineNumber: job.LineNumber,
				Expression: job.Expression,
				Message:    fmt.Sprintf("Failed to parse expression: %s", err.Error()),
			})
			continue
		}

		result.ValidJobs++

		// Check for DOM/DOW conflict
		if detectDOMDOWConflict(schedule) {
			result.Issues = append(result.Issues, Issue{
				Type:       "warning",
				LineNumber: job.LineNumber,
				Expression: job.Expression,
				Message:    "Both day-of-month and day-of-week specified (runs if either condition is met)",
			})
		}

		// Check for empty schedule
		if detectEmptySchedule(job.Expression, v.scheduler) {
			result.Valid = false
			result.InvalidJobs++
			result.ValidJobs--
			result.Issues = append(result.Issues, Issue{
				Type:       "error",
				LineNumber: job.LineNumber,
				Expression: job.Expression,
				Message:    "Schedule never runs (empty schedule)",
			})
		}
	}

	return result
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
