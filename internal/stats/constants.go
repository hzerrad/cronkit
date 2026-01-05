package stats

import "time"

// Time window constants
const (
	// HoursPerDay is the number of hours in a day
	HoursPerDay = 24
	// MinutesPerHour is the number of minutes in an hour
	MinutesPerHour = 60
	// MinutesPerDay is the number of minutes in a day (24 * 60)
	MinutesPerDay = 1440
)

// Time duration constants
const (
	// OneHour represents one hour duration
	OneHour = 1 * time.Hour
	// OneDay represents one day duration (24 hours)
	OneDay = 24 * time.Hour
)

// Scheduler run count limits
const (
	// MaxRunsPerHour is the maximum possible runs in one hour (every minute)
	MaxRunsPerHour = MinutesPerHour
	// MaxRunsPerDay is the maximum possible runs in one day (every minute)
	MaxRunsPerDay = MinutesPerDay
	// MaxRunsForLongWindow is the cap for very long time windows
	MaxRunsForLongWindow = 10000
)

// Histogram constants
const (
	// HoursInDay is the number of hours in a day (for histogram array size)
	HoursInDay = 24
	// DefaultHistogramWidth is the default width for histogram bars
	DefaultHistogramWidth = 40
)
