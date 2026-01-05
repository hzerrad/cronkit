package check

import "time"

// Time window constants
const (
	// DefaultOverlapWindow is the default time window for overlap detection
	DefaultOverlapWindow = 24 * time.Hour
)

// Scheduler run count limits for frequency calculations
const (
	// MaxRunsForDailyCalculation is the maximum runs to request for daily frequency calculation
	// Worst case: every minute = 1440 runs/day, using 2000 for safety margin
	MaxRunsForDailyCalculation = 2000
	// MaxRunsForHourlyCalculation is the maximum runs to request for hourly frequency calculation
	// Worst case: every minute = 60 runs/hour, using 100 for safety margin
	MaxRunsForHourlyCalculation = 100
)
