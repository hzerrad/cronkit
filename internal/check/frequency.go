package check

import (
	"fmt"
	"strings"
	"time"

	"github.com/hzerrad/cronic/internal/cronx"
)

// CalculateRunsPerDay calculates the number of times a cron expression runs per day
func CalculateRunsPerDay(expression string, scheduler cronx.Scheduler) (int, error) {
	// Start from just before midnight to capture the first run at midnight
	// scheduler.Next returns times AFTER the given time, so we need to start slightly before
	startTime := ReferenceDate
	queryTime := startTime.Add(-1 * time.Second) // Query from just before midnight
	endTime := startTime.Add(DefaultOverlapWindow)

	// Get all runs for a 24-hour period
	// For every-minute schedules, we need 1440 runs. Use a larger limit to be safe.
	times, err := scheduler.Next(expression, queryTime, MaxRunsForDailyCalculation)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate runs: %w", err)
	}

	// Count runs that fall within the 24-hour window [startTime, endTime)
	count := 0
	for _, t := range times {
		if !t.Before(endTime) {
			break
		}
		// Include all times >= startTime and < endTime
		if !t.Before(startTime) {
			count++
		}
	}

	return count, nil
}

// DetectRedundantPattern detects if a schedule uses redundant step patterns like */1
func DetectRedundantPattern(schedule *cronx.Schedule) bool {
	// Check each field for */1 pattern by examining the raw field value
	fields := []cronx.Field{
		schedule.Minute,
		schedule.Hour,
		schedule.DayOfMonth,
		schedule.Month,
		schedule.DayOfWeek,
	}

	for _, field := range fields {
		raw := field.Raw()
		// Check for exact */1 pattern (could be */1 or 0-59/1, etc.)
		// Note: IsStep() returns true only for step > 1, so */1 has IsStep() = false
		// We need to check the raw string directly
		if strings.HasSuffix(raw, "/1") {
			return true
		}
	}

	return false
}

// EstimateRunFrequency estimates the run frequency for a cron expression
// Returns runs per day and runs per hour
func EstimateRunFrequency(expression string, scheduler cronx.Scheduler) (runsPerDay, runsPerHour int, err error) {
	runsPerDay, err = CalculateRunsPerDay(expression, scheduler)
	if err != nil {
		return 0, 0, err
	}

	// Calculate runs per hour by getting runs for a 1-hour window
	// Start from just before the hour to capture the first run at the start of the hour
	startTime := ReferenceDate
	queryTime := startTime.Add(-1 * time.Second) // Query from just before the hour
	endTime := startTime.Add(1 * time.Hour)

	times, err := scheduler.Next(expression, queryTime, MaxRunsForHourlyCalculation)
	if err != nil {
		return runsPerDay, 0, fmt.Errorf("failed to calculate hourly runs: %w", err)
	}

	count := 0
	for _, t := range times {
		if !t.Before(endTime) {
			break
		}
		// Include all times >= startTime and < endTime
		if !t.Before(startTime) {
			count++
		}
	}

	runsPerHour = count
	return runsPerDay, runsPerHour, nil
}

// GetRedundantPatternSuggestion returns a suggestion for simplifying a redundant pattern
func GetRedundantPatternSuggestion(expression string, schedule *cronx.Schedule) string {
	parts := strings.Fields(expression)
	if len(parts) != 5 {
		return expression // Can't simplify if not standard format
	}

	simplified := make([]string, 5)
	copy(simplified, parts)

	// Replace */1 with * in each field
	// Check both the schedule field and the raw string to handle various formats
	fields := []struct {
		field cronx.Field
		index int
	}{
		{schedule.Minute, 0},
		{schedule.Hour, 1},
		{schedule.DayOfMonth, 2},
		{schedule.Month, 3},
		{schedule.DayOfWeek, 4},
	}

	for _, f := range fields {
		// Check if the raw part contains */1 pattern
		// Note: IsStep() returns true only for step > 1, so */1 has IsStep() = false
		// We need to check the raw string directly
		raw := parts[f.index]
		if strings.HasSuffix(raw, "/1") {
			simplified[f.index] = "*"
		}
	}

	return strings.Join(simplified, " ")
}
