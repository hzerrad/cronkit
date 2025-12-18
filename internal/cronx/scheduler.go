package cronx

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler calculates next run times for cron schedules.
// This is the SINGLE BOUNDARY where robfig/cron Schedule.Next() is used.
// No other package in the codebase should use robfig/cron's scheduling capabilities.
type Scheduler interface {
	// Next calculates the next N occurrences of a cron expression starting from the given time.
	// Returns a slice of time.Time values representing when the cron job would run.
	// Returns an error if the expression is invalid or cannot be parsed.
	Next(expression string, from time.Time, count int) ([]time.Time, error)
}

// scheduler implements the Scheduler interface
type scheduler struct {
	parser     Parser
	cronParser cron.Parser
}

// NewScheduler creates a new Scheduler instance.
// The scheduler uses both our internal parser (for validation) and robfig/cron (for calculation).
func NewScheduler() Scheduler {
	return &scheduler{
		parser: NewParser(),
		cronParser: cron.NewParser(
			cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
		),
	}
}

// Next implements the Scheduler interface.
// BOUNDARY: This is the ONLY place in the codebase that calls robfig/cron Schedule.Next().
func (s *scheduler) Next(expression string, from time.Time, count int) ([]time.Time, error) {
	// Step 1: Validate the expression using our internal parser
	// This ensures we catch validation errors with our consistent error messages
	if _, err := s.parser.Parse(expression); err != nil {
		return nil, err
	}

	// Step 2: Parse the expression with robfig/cron to get a Schedule
	// BOUNDARY: Using robfig/cron's parser to create a Schedule object
	schedule, err := s.cronParser.Parse(expression)
	if err != nil {
		// This shouldn't happen if our parser validation is correct,
		// but we handle it just in case
		return nil, fmt.Errorf("failed to parse cron expression: %w", err)
	}

	// Step 3: Calculate the next N occurrences
	// BOUNDARY: This is where we use Schedule.Next() from robfig/cron
	times := make([]time.Time, 0, count)
	current := from

	for i := 0; i < count; i++ {
		// BOUNDARY: Calling robfig/cron's Schedule.Next()
		// This is the ONLY place this method is called in the entire codebase
		next := schedule.Next(current)
		times = append(times, next)
		current = next
	}

	return times, nil
}
