package cronx

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler calculates next run times for cron schedules.
type Scheduler interface {
	// Next calculates the next N occurrences of a cron expression starting from the given time.
	Next(expression string, from time.Time, count int) ([]time.Time, error)
}

// robfigScheduler implements the Scheduler interface using robfig/cron library.
type robfigScheduler struct {
	parser     Parser
	cronParser cron.Parser
}

// NewScheduler creates a new Scheduler instance using the robfig/cron implementation.
func NewScheduler() Scheduler {
	return NewRobfigScheduler()
}

// NewRobfigScheduler creates a new robfig/cron-based scheduler.
func NewRobfigScheduler() Scheduler {
	return &robfigScheduler{
		parser: NewParser(),
		cronParser: cron.NewParser(
			cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
		),
	}
}

// Next implements the Scheduler Next method using robfig/cron library
func (s *robfigScheduler) Next(expression string, from time.Time, count int) ([]time.Time, error) {
	// Step 1: Validate the expression using our internal parser
	// This ensures consistent error messages across all implementations
	if _, err := s.parser.Parse(expression); err != nil {
		return nil, err
	}

	// Step 2: Parse the expression with robfig/cron to get a Schedule
	schedule, err := s.cronParser.Parse(expression)
	if err != nil {
		// This shouldn't happen if our parser validation is correct,
		// but we handle it just in case
		return nil, fmt.Errorf("failed to parse cron expression: %w", err)
	}

	// Step 3: Calculate the next N occurrences using robfig/cron's Schedule.Next()
	times := make([]time.Time, 0, count)
	current := from

	for i := 0; i < count; i++ {
		next := schedule.Next(current)
		times = append(times, next)
		current = next
	}

	return times, nil
}
