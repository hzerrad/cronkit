package stats

import (
	"sort"
	"time"

	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/hzerrad/cronkit/internal/timeutil"
)

// ReferenceDate is a fixed date used for consistent calculations
// Using 2025-01-01 00:00:00 UTC as a reference point
var ReferenceDate = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

// Calculator calculates statistics for a set of discovered schedules
type Calculator struct {
	scheduler cronx.Scheduler
	parser    cronx.Parser
	now       func() time.Time
}

// NewCalculator creates a new statistics calculator
func NewCalculator() *Calculator {
	return &Calculator{
		scheduler: cronx.NewScheduler(),
		parser:    cronx.NewParser(),
		now:       time.Now,
	}
}

// SetClock replaces the source of the current time, so callers and tests can
// evaluate schedules from a fixed origin.
func (c *Calculator) SetClock(now func() time.Time) {
	c.now = now
}

// CalculateMetrics calculates comprehensive metrics for active items; callers
// must run items through inventory.ResolveTimezones first, or items with an
// unresolved timezone are silently excluded instead of reported.
func (c *Calculator) CalculateMetrics(items []inventory.Item, timeWindow time.Duration) (*Metrics, error) {
	metrics := &Metrics{
		JobFrequencies: []JobFrequency{},
		HourHistogram:  make([]int, HoursInDay),
		Collisions:     CollisionStats{},
	}

	// Calculate per-item frequencies
	for i, item := range items {
		if item.State != inventory.StateActive {
			continue
		}

		runsPerDay, runsPerHour := c.calculateJobFrequency(item.Expression, item.Timezone)
		metrics.JobFrequencies = append(metrics.JobFrequencies, JobFrequency{
			JobID:       item.Locator.Identity(i, "job-", "line-"),
			Expression:  item.Expression,
			RunsPerDay:  runsPerDay,
			RunsPerHour: runsPerHour,
		})

		metrics.TotalRunsPerDay += runsPerDay
		metrics.TotalRunsPerHour += runsPerHour
	}

	// Calculate hour histogram
	c.calculateHourHistogram(items, metrics)

	// Calculate collisions
	collisions := c.CalculateCollisions(items, timeWindow)
	metrics.Collisions = collisions

	return metrics, nil
}

// calculateJobFrequency calculates runs per day and per hour for a schedule,
// evaluated in its own timezone (empty means UTC).
func (c *Calculator) calculateJobFrequency(expression, timezone string) (runsPerDay, runsPerHour int) {
	loc, err := timeutil.ResolveLocation(timezone, ReferenceDate.Location())
	if err != nil {
		return 0, 0
	}

	startTime := ReferenceDate.In(loc)
	endTime := startTime.Add(OneDay)

	// Use optimized counting with smart estimates
	runsPerDay = c.countRunsInWindow(expression, startTime, endTime)

	hourEndTime := startTime.Add(OneHour)
	runsPerHour = c.countRunsInWindow(expression, startTime, hourEndTime)

	return runsPerDay, runsPerHour
}

// countRunsInWindow counts how many times a cron expression runs within a time window
// Uses smart estimation to minimize unnecessary time generation
func (c *Calculator) countRunsInWindow(expression string, startTime, endTime time.Time) int {
	windowDuration := endTime.Sub(startTime)

	// Smart estimate based on window duration to avoid generating excessive times
	// Worst case: every minute
	var maxRuns int
	if windowDuration <= OneHour {
		// For 1 hour: worst case is every minute
		maxRuns = MaxRunsPerHour
	} else if windowDuration <= OneDay {
		// For 24 hours: worst case is every minute
		maxRuns = MaxRunsPerDay
	} else {
		// For longer windows, calculate based on duration
		maxRuns = int(windowDuration.Minutes()) + 1
		if maxRuns > MaxRunsForLongWindow {
			maxRuns = MaxRunsForLongWindow // Cap for very long windows
		}
	}

	times, err := c.scheduler.Next(expression, startTime, maxRuns)
	if err != nil {
		return 0
	}

	count := 0
	for _, t := range times {
		if t.After(endTime) || t.Equal(endTime) {
			break
		}
		if !t.Before(startTime) {
			count++
		}
	}

	return count
}

// calculateHourHistogram buckets runs (converted to UTC) across a fixed
// reference day; it deliberately anchors differently from CalculateCollisions,
// so HourHistogram and Collisions.BusiestHours may disagree for DST-observing
// zones.
func (c *Calculator) calculateHourHistogram(items []inventory.Item, metrics *Metrics) {
	startTime := ReferenceDate
	endTime := startTime.Add(OneDay)

	// Use optimized count: worst case is every minute
	maxRuns := MaxRunsPerDay

	for _, item := range items {
		if item.State != inventory.StateActive {
			continue
		}

		loc, err := timeutil.ResolveLocation(item.Timezone, ReferenceDate.Location())
		if err != nil {
			continue
		}

		times, err := c.scheduler.Next(item.Expression, startTime.In(loc), maxRuns)
		if err != nil {
			continue
		}

		for _, t := range times {
			if t.After(endTime) || t.Equal(endTime) {
				break
			}
			if !t.Before(startTime) {
				hour := t.UTC().Hour()
				metrics.HourHistogram[hour]++
			}
		}
	}
}

// IdentifyMostFrequent returns the top N most frequent active items; see
// CalculateMetrics for the admission requirement.
func (c *Calculator) IdentifyMostFrequent(items []inventory.Item, topN int) []JobFrequency {
	frequencies := make([]JobFrequency, 0, len(items))

	for i, item := range items {
		if item.State != inventory.StateActive {
			continue
		}

		runsPerDay, runsPerHour := c.calculateJobFrequency(item.Expression, item.Timezone)
		frequencies = append(frequencies, JobFrequency{
			JobID:       item.Locator.Identity(i, "job-", "line-"),
			Expression:  item.Expression,
			RunsPerDay:  runsPerDay,
			RunsPerHour: runsPerHour,
		})
	}

	// Sort by runs per day (descending)
	sort.Slice(frequencies, func(i, j int) bool {
		return frequencies[i].RunsPerDay > frequencies[j].RunsPerDay
	})

	if topN > 0 && topN < len(frequencies) {
		return frequencies[:topN]
	}

	return frequencies
}

// IdentifyLeastFrequent returns the top N least frequent items
func (c *Calculator) IdentifyLeastFrequent(items []inventory.Item, topN int) []JobFrequency {
	frequencies := c.IdentifyMostFrequent(items, 0) // Get all

	// Sort by runs per day (ascending)
	sort.Slice(frequencies, func(i, j int) bool {
		return frequencies[i].RunsPerDay < frequencies[j].RunsPerDay
	})

	if topN > 0 && topN < len(frequencies) {
		return frequencies[:topN]
	}

	return frequencies
}

// CalculateCollisions calculates collision statistics for active items,
// anchored on c.now() truncated to the minute rather than ReferenceDate, and
// compares each item's occurrences in absolute time so schedules in different
// zones landing on the same moment count as concurrent.
func (c *Calculator) CalculateCollisions(items []inventory.Item, timeWindow time.Duration) CollisionStats {
	stats := CollisionStats{
		BusiestHours:       []HourStats{},
		QuietWindows:       []TimeWindow{},
		CollisionFrequency: 0.0,
		MaxConcurrent:      0,
	}

	startTime := c.now().Truncate(time.Minute)
	endTime := startTime.Add(timeWindow)

	// Group runs by minute, keyed by instant so runs in different zones collide
	minuteRuns := make(map[int64]int)
	// Estimate max runs based on time window (worst case: every minute)
	maxRuns := int(timeWindow.Minutes()) + 1
	if maxRuns > MaxRunsForLongWindow {
		maxRuns = MaxRunsForLongWindow // Cap at reasonable maximum
	}

	for _, item := range items {
		if item.State != inventory.StateActive {
			continue
		}

		loc, err := timeutil.ResolveLocation(item.Timezone, startTime.Location())
		if err != nil {
			continue
		}

		times, err := c.scheduler.Next(item.Expression, startTime.In(loc), maxRuns)
		if err != nil {
			continue
		}

		for _, t := range times {
			if t.After(endTime) || t.Equal(endTime) {
				break
			}
			if !t.Before(startTime) {
				minuteRuns[timeutil.MinuteKey(t)]++
			}
		}
	}

	// Calculate busiest hours in the origin's zone
	hourRuns := make(map[int]int)
	for key, count := range minuteRuns {
		hour := time.Unix(key, 0).In(startTime.Location()).Hour()
		hourRuns[hour] += count
		if count > stats.MaxConcurrent {
			stats.MaxConcurrent = count
		}
	}

	// Convert to HourStats slice
	for hour, count := range hourRuns {
		stats.BusiestHours = append(stats.BusiestHours, HourStats{
			Hour:     hour,
			RunCount: count,
		})
	}

	// Sort by run count (descending), breaking ties by hour (ascending) so the
	// result is deterministic regardless of map iteration order.
	sort.Slice(stats.BusiestHours, func(i, j int) bool {
		if stats.BusiestHours[i].RunCount != stats.BusiestHours[j].RunCount {
			return stats.BusiestHours[i].RunCount > stats.BusiestHours[j].RunCount
		}
		return stats.BusiestHours[i].Hour < stats.BusiestHours[j].Hour
	})

	// Calculate collision frequency
	totalMinutes := int(timeWindow.Minutes())
	collisionMinutes := 0
	for _, count := range minuteRuns {
		if count > 1 {
			collisionMinutes++
		}
	}

	if totalMinutes > 0 {
		stats.CollisionFrequency = float64(collisionMinutes) / float64(totalMinutes) * 100.0
	}

	return stats
}

// IdentifyBusiestHours returns the busiest hours
func (c *Calculator) IdentifyBusiestHours(items []inventory.Item) []HourStats {
	stats := c.CalculateCollisions(items, OneDay)
	return stats.BusiestHours
}
