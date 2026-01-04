package stats

import "time"

// Metrics contains frequency and collision statistics
type Metrics struct {
	TotalRunsPerDay  int
	TotalRunsPerHour int
	JobFrequencies   []JobFrequency
	HourHistogram    []int // 24 elements, index = hour (0-23)
	Collisions       CollisionStats
}

// JobFrequency represents frequency information for a single job
type JobFrequency struct {
	JobID       string
	Expression  string
	RunsPerDay  int
	RunsPerHour int
}

// CollisionStats contains collision analysis results
type CollisionStats struct {
	BusiestHours       []HourStats
	QuietWindows       []TimeWindow
	CollisionFrequency float64 // Percentage of time windows with collisions
	MaxConcurrent      int
}

// HourStats contains statistics for a specific hour
type HourStats struct {
	Hour     int
	RunCount int
	JobCount int
}

// TimeWindow represents a time window with collision information
type TimeWindow struct {
	Start    time.Time
	End      time.Time
	RunCount int
	JobCount int
}
