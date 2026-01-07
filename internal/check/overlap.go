package check

import (
	"fmt"
	"sort"
	"time"

	"github.com/hzerrad/cronkit/internal/crontab"
	"github.com/hzerrad/cronkit/internal/cronx"
)

// Overlap represents multiple jobs running at the same time
type Overlap struct {
	Time   time.Time
	Count  int
	JobIDs []string
}

// OverlapStats contains statistics about job overlaps
type OverlapStats struct {
	TotalWindows    int
	MaxConcurrent   int
	MostProblematic []Overlap // Top N overlaps sorted by count
}

// AnalyzeOverlaps analyzes job overlaps within a time window
func AnalyzeOverlaps(jobs []*crontab.Job, timeWindow time.Duration, scheduler cronx.Scheduler, parser cronx.Parser) ([]Overlap, OverlapStats, error) {
	if len(jobs) == 0 {
		return []Overlap{}, OverlapStats{}, nil
	}

	// Start from current time
	startTime := time.Now().Truncate(time.Minute)
	endTime := startTime.Add(timeWindow)

	// Collect all run times for all jobs
	type jobRun struct {
		time  time.Time
		jobID string
	}
	var allRuns []jobRun

	for _, job := range jobs {
		if !job.Valid {
			continue
		}

		// Get job identifier (use line number or expression)
		jobID := fmt.Sprintf("line-%d", job.LineNumber)
		if job.LineNumber == 0 {
			jobID = job.Expression
		}

		// Get all runs for this job within the time window
		times, err := scheduler.Next(job.Expression, startTime, 10000) // Large limit to get all runs
		if err != nil {
			continue // Skip jobs that can't be scheduled
		}

		for _, t := range times {
			if t.After(endTime) || t.Equal(endTime) {
				break
			}
			if !t.Before(startTime) {
				allRuns = append(allRuns, jobRun{
					time:  t.Truncate(time.Minute), // Round to minute for overlap detection
					jobID: jobID,
				})
			}
		}
	}

	// Group runs by time (minute precision)
	overlapMap := make(map[time.Time][]string)
	for _, run := range allRuns {
		overlapMap[run.time] = append(overlapMap[run.time], run.jobID)
	}

	// Convert to Overlap structs
	var overlaps []Overlap
	for t, jobIDs := range overlapMap {
		// Remove duplicates
		uniqueJobs := uniqueStrings(jobIDs)
		if len(uniqueJobs) > 1 {
			overlaps = append(overlaps, Overlap{
				Time:   t,
				Count:  len(uniqueJobs),
				JobIDs: uniqueJobs,
			})
		}
	}

	// Sort by count (descending) then by time
	sort.Slice(overlaps, func(i, j int) bool {
		if overlaps[i].Count != overlaps[j].Count {
			return overlaps[i].Count > overlaps[j].Count
		}
		return overlaps[i].Time.Before(overlaps[j].Time)
	})

	// Calculate statistics
	stats := OverlapStats{
		TotalWindows:  len(overlaps),
		MaxConcurrent: 0,
	}

	if len(overlaps) > 0 {
		stats.MaxConcurrent = overlaps[0].Count
		// Get top 10 most problematic overlaps
		topN := 10
		if len(overlaps) < topN {
			topN = len(overlaps)
		}
		stats.MostProblematic = overlaps[:topN]
	}

	return overlaps, stats, nil
}

// uniqueStrings removes duplicates from a string slice
func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
