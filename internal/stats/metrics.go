package stats

import "time"

// Metrics contains frequency and collision statistics
type Metrics struct {
	TotalRunsPerDay  int
	TotalRunsPerHour int
	JobFrequencies   []JobFrequency
	// HourHistogram is a typical day's run distribution (index i is the
	// count of runs landing in hour i, 0-23), bucketed in UTC against the
	// fixed ReferenceDate rather than Collisions' anchor.
	HourHistogram []int
	// Collisions describes a concrete forward window starting at the moment
	// of evaluation; see CalculateCollisions's doc comment.
	Collisions CollisionStats
}

// JobFrequency represents frequency information for a single job
type JobFrequency struct {
	JobID       string
	Expression  string
	RunsPerDay  int
	RunsPerHour int
}

// CollisionStats contains collision analysis results, computed over a
// concrete forward window starting at the moment of evaluation; see
// CalculateCollisions's doc comment for its exact anchor and zone.
type CollisionStats struct {
	// BusiestHours ranks hours by how many runs land in each, within this
	// analysis window, not HourHistogram's typical day.
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
