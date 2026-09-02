package check

import (
	"sort"
	"time"

	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/hzerrad/cronkit/internal/timeutil"
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

// AnalyzeOverlaps analyzes active item overlaps within a time window starting
// at from, comparing each item's occurrences (evaluated in its own timezone)
// in absolute time; callers must run items through inventory.ResolveTimezones first.
func AnalyzeOverlaps(items []inventory.Item, from time.Time, timeWindow time.Duration, scheduler cronx.Scheduler, parser cronx.Parser) ([]Overlap, OverlapStats, error) {
	if len(items) == 0 {
		return []Overlap{}, OverlapStats{}, nil
	}

	startTime := from.Truncate(time.Minute)
	endTime := startTime.Add(timeWindow)

	// Collect all run times for all items
	type jobRun struct {
		time  time.Time
		jobID string
	}
	var allRuns []jobRun

	// concurrencyByJob records each job's policy so an overlap is suppressed when all participants forbid it.
	concurrencyByJob := make(map[string]inventory.Concurrency)

	for i, item := range items {
		if item.State != inventory.StateActive {
			continue
		}

		jobID := item.Locator.Identity(i, "job-", "line-")
		concurrencyByJob[jobID] = item.Concurrency

		loc, err := timeutil.ResolveLocation(item.Timezone, startTime.Location())
		if err != nil {
			continue // Skip items whose timezone doesn't resolve
		}

		// Get all runs for this item within the time window, evaluated in
		// its own zone
		times, err := scheduler.Next(item.Expression, startTime.In(loc), 10000) // Large limit to get all runs
		if err != nil {
			continue // Skip items that can't be scheduled
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

	// Group runs by minute, keyed by instant so runs in different zones collide
	overlapMap := make(map[int64][]string)
	repTime := make(map[int64]time.Time)
	for _, run := range allRuns {
		k := timeutil.MinuteKey(run.time)
		overlapMap[k] = append(overlapMap[k], run.jobID)
		if _, seen := repTime[k]; !seen {
			repTime[k] = run.time
		}
	}

	// Convert to Overlap structs
	var overlaps []Overlap
	for k, jobIDs := range overlapMap {
		uniqueJobs := uniqueStrings(jobIDs)
		if len(uniqueJobs) > 1 && !allForbid(uniqueJobs, concurrencyByJob) {
			overlaps = append(overlaps, Overlap{
				Time:   repTime[k],
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

// allForbid reports whether every job in an overlapping group has
// ConcurrencyForbid, in which case the overlap is not reported since the
// platform already serialises each of those jobs against itself.
func allForbid(jobIDs []string, concurrencyByJob map[string]inventory.Concurrency) bool {
	for _, id := range jobIDs {
		if concurrencyByJob[id] != inventory.ConcurrencyForbid {
			return false
		}
	}
	return true
}

// uniqueStrings removes duplicates from a string slice and orders what is left
func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	sort.Strings(result)
	return result
}
