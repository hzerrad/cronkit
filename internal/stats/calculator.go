package stats

import (
	"fmt"
	"sort"
	"time"

	"github.com/hzerrad/cronic/internal/crontab"
	"github.com/hzerrad/cronic/internal/cronx"
)

// Calculator calculates statistics for crontab jobs
type Calculator struct {
	scheduler cronx.Scheduler
	parser    cronx.Parser
}

// NewCalculator creates a new statistics calculator
func NewCalculator() *Calculator {
	return &Calculator{
		scheduler: cronx.NewScheduler(),
		parser:    cronx.NewParser(),
	}
}

// CalculateMetrics calculates comprehensive metrics for a set of jobs
func (c *Calculator) CalculateMetrics(jobs []*crontab.Job, timeWindow time.Duration) (*Metrics, error) {
	metrics := &Metrics{
		JobFrequencies: []JobFrequency{},
		HourHistogram:  make([]int, 24),
		Collisions:     CollisionStats{},
	}

	// Calculate per-job frequencies
	for _, job := range jobs {
		if !job.Valid {
			continue
		}

		jobID := fmt.Sprintf("line-%d", job.LineNumber)
		if job.LineNumber == 0 {
			jobID = job.Expression
		}

		runsPerDay, runsPerHour := c.calculateJobFrequency(job.Expression)
		metrics.JobFrequencies = append(metrics.JobFrequencies, JobFrequency{
			JobID:       jobID,
			Expression:  job.Expression,
			RunsPerDay:  runsPerDay,
			RunsPerHour: runsPerHour,
		})

		metrics.TotalRunsPerDay += runsPerDay
		metrics.TotalRunsPerHour += runsPerHour
	}

	// Calculate hour histogram
	c.calculateHourHistogram(jobs, metrics)

	// Calculate collisions
	collisions := c.CalculateCollisions(jobs, timeWindow)
	metrics.Collisions = collisions

	return metrics, nil
}

// calculateJobFrequency calculates runs per day and per hour for a job
func (c *Calculator) calculateJobFrequency(expression string) (runsPerDay, runsPerHour int) {
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.Add(24 * time.Hour)

	times, err := c.scheduler.Next(expression, startTime, 2000)
	if err != nil {
		return 0, 0
	}

	for _, t := range times {
		if t.After(endTime) || t.Equal(endTime) {
			break
		}
		if !t.Before(startTime) {
			runsPerDay++
		}
	}

	hourEndTime := startTime.Add(1 * time.Hour)
	hourTimes, err := c.scheduler.Next(expression, startTime, 100)
	if err == nil {
		for _, t := range hourTimes {
			if t.After(hourEndTime) || t.Equal(hourEndTime) {
				break
			}
			if !t.Before(startTime) {
				runsPerHour++
			}
		}
	}

	return runsPerDay, runsPerHour
}

// calculateHourHistogram calculates the distribution of runs across hours
func (c *Calculator) calculateHourHistogram(jobs []*crontab.Job, metrics *Metrics) {
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.Add(24 * time.Hour)

	for _, job := range jobs {
		if !job.Valid {
			continue
		}

		times, err := c.scheduler.Next(job.Expression, startTime, 2000)
		if err != nil {
			continue
		}

		for _, t := range times {
			if t.After(endTime) || t.Equal(endTime) {
				break
			}
			if !t.Before(startTime) {
				hour := t.Hour()
				metrics.HourHistogram[hour]++
			}
		}
	}
}

// IdentifyMostFrequent returns the top N most frequent jobs
func (c *Calculator) IdentifyMostFrequent(jobs []*crontab.Job, topN int) []JobFrequency {
	frequencies := make([]JobFrequency, 0, len(jobs))

	for _, job := range jobs {
		if !job.Valid {
			continue
		}

		jobID := fmt.Sprintf("line-%d", job.LineNumber)
		if job.LineNumber == 0 {
			jobID = job.Expression
		}

		runsPerDay, runsPerHour := c.calculateJobFrequency(job.Expression)
		frequencies = append(frequencies, JobFrequency{
			JobID:       jobID,
			Expression:  job.Expression,
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

// IdentifyLeastFrequent returns the top N least frequent jobs
func (c *Calculator) IdentifyLeastFrequent(jobs []*crontab.Job, topN int) []JobFrequency {
	frequencies := c.IdentifyMostFrequent(jobs, 0) // Get all

	// Sort by runs per day (ascending)
	sort.Slice(frequencies, func(i, j int) bool {
		return frequencies[i].RunsPerDay < frequencies[j].RunsPerDay
	})

	if topN > 0 && topN < len(frequencies) {
		return frequencies[:topN]
	}

	return frequencies
}

// CalculateCollisions calculates collision statistics
func (c *Calculator) CalculateCollisions(jobs []*crontab.Job, timeWindow time.Duration) CollisionStats {
	stats := CollisionStats{
		BusiestHours:       []HourStats{},
		QuietWindows:       []TimeWindow{},
		CollisionFrequency: 0.0,
		MaxConcurrent:      0,
	}

	// Use overlap analysis from check package
	// For now, simplified implementation
	startTime := time.Now().Truncate(time.Minute)
	endTime := startTime.Add(timeWindow)

	// Group runs by minute
	minuteRuns := make(map[time.Time]int)
	for _, job := range jobs {
		if !job.Valid {
			continue
		}

		times, err := c.scheduler.Next(job.Expression, startTime, 10000)
		if err != nil {
			continue
		}

		for _, t := range times {
			if t.After(endTime) || t.Equal(endTime) {
				break
			}
			if !t.Before(startTime) {
				minute := t.Truncate(time.Minute)
				minuteRuns[minute]++
			}
		}
	}

	// Calculate busiest hours
	hourRuns := make(map[int]int)
	for minute, count := range minuteRuns {
		hour := minute.Hour()
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

	// Sort by run count (descending)
	sort.Slice(stats.BusiestHours, func(i, j int) bool {
		return stats.BusiestHours[i].RunCount > stats.BusiestHours[j].RunCount
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
func (c *Calculator) IdentifyBusiestHours(jobs []*crontab.Job) []HourStats {
	stats := c.CalculateCollisions(jobs, 24*time.Hour)
	return stats.BusiestHours
}
