package render

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// TimelineView represents the type of timeline view
type TimelineView int

const (
	// DayView shows 24 hours
	DayView TimelineView = iota
	// HourView shows 60 minutes
	HourView
)

// String returns the string representation of TimelineView
func (v TimelineView) String() string {
	switch v {
	case DayView:
		return "day"
	case HourView:
		return "hour"
	default:
		return "unknown"
	}
}

// JobRun represents a single job execution at a specific time
type JobRun struct {
	JobID   string
	RunTime time.Time
}

// Overlap represents multiple jobs running at the same time
type Overlap struct {
	Time   time.Time
	Count  int
	JobIDs []string
}

// JobInfo contains metadata about a job
type JobInfo struct {
	Expression  string
	Description string
}

// Timeline represents a timeline with time slots and job runs
type Timeline struct {
	view      TimelineView
	startTime time.Time
	endTime   time.Time
	width     int
	jobRuns   []JobRun
	jobInfo   map[string]JobInfo
	slots     []time.Time
}

// NewTimeline creates a new timeline with the specified view, start time, and width
func NewTimeline(view TimelineView, startTime time.Time, width int) *Timeline {
	var endTime time.Time
	var slots []time.Time

	switch view {
	case DayView:
		endTime = startTime.Add(24 * time.Hour)
		// Create slots for each hour in a day (24 slots)
		slots = make([]time.Time, 24)
		for i := 0; i < 24; i++ {
			slots[i] = startTime.Add(time.Duration(i) * time.Hour)
		}
	case HourView:
		endTime = startTime.Add(time.Hour)
		// Create slots for each minute in an hour (60 slots)
		slots = make([]time.Time, 60)
		for i := 0; i < 60; i++ {
			slots[i] = startTime.Add(time.Duration(i) * time.Minute)
		}
	}

	return &Timeline{
		view:      view,
		startTime: startTime,
		endTime:   endTime,
		width:     width,
		jobRuns:   make([]JobRun, 0),
		jobInfo:   make(map[string]JobInfo),
		slots:     slots,
	}
}

// AddJobRun adds a job run to the timeline if it falls within the timeline range
func (tl *Timeline) AddJobRun(jobID string, runTime time.Time) {
	if runTime.Before(tl.startTime) || !runTime.Before(tl.endTime) {
		return
	}

	tl.jobRuns = append(tl.jobRuns, JobRun{
		JobID:   jobID,
		RunTime: runTime,
	})
}

// SetJobInfo sets metadata for a job
func (tl *Timeline) SetJobInfo(jobID, expression, description string) {
	tl.jobInfo[jobID] = JobInfo{
		Expression:  expression,
		Description: description,
	}
}

// DetectOverlaps finds times where multiple jobs run simultaneously
func (tl *Timeline) DetectOverlaps() []Overlap {
	// Group runs by time
	timeGroups := make(map[time.Time][]string)
	for _, run := range tl.jobRuns {
		// Round to nearest minute for overlap detection
		rounded := run.RunTime.Truncate(time.Minute)
		timeGroups[rounded] = append(timeGroups[rounded], run.JobID)
	}

	overlaps := make([]Overlap, 0)
	for t, jobIDs := range timeGroups {
		if len(jobIDs) > 1 {
			// Remove duplicates
			uniqueJobs := make(map[string]bool)
			uniqueList := make([]string, 0)
			for _, id := range jobIDs {
				if !uniqueJobs[id] {
					uniqueJobs[id] = true
					uniqueList = append(uniqueList, id)
				}
			}

			overlaps = append(overlaps, Overlap{
				Time:   t,
				Count:  len(uniqueList),
				JobIDs: uniqueList,
			})
		}
	}

	// Sort by time
	sort.Slice(overlaps, func(i, j int) bool {
		return overlaps[i].Time.Before(overlaps[j].Time)
	})

	return overlaps
}

// Render generates an ASCII timeline string
func (tl *Timeline) Render() string {
	var sb strings.Builder

	// Header
	var timeRange string
	if tl.view == DayView {
		timeRange = fmt.Sprintf("%s ──────────────────────────────────────────────────────────────── %s",
			tl.startTime.Format("15:04"), tl.endTime.Format("15:04"))
		sb.WriteString(fmt.Sprintf("Timeline for %s (Day View)\n", tl.startTime.Format("2006-01-02")))
	} else {
		timeRange = fmt.Sprintf("%s ──────────────────────────────────────────────────────────────── %s",
			tl.startTime.Format("15:04"), tl.endTime.Format("15:04"))
		sb.WriteString(fmt.Sprintf("Timeline for %s (Hour View)\n", tl.startTime.Format("2006-01-02 15:04")))
	}

	sb.WriteString(timeRange + "\n")
	sb.WriteString("      │                                                                    │\n")

	// Group runs by slot
	slotRuns := make(map[int][]string) // slot index -> job IDs
	for _, run := range tl.jobRuns {
		slotIdx := tl.findSlotIndex(run.RunTime)
		if slotIdx >= 0 && slotIdx < len(tl.slots) {
			slotRuns[slotIdx] = append(slotRuns[slotIdx], run.JobID)
		}
	}

	// Render timeline bars
	maxOverlaps := 1
	for _, jobIDs := range slotRuns {
		uniqueCount := len(uniqueStrings(jobIDs))
		if uniqueCount > maxOverlaps {
			maxOverlaps = uniqueCount
		}
	}

	// Draw bars for each overlap level
	for level := 0; level < maxOverlaps; level++ {
		sb.WriteString("      │")
		for i := 0; i < len(tl.slots); i++ {
			if jobIDs, hasRuns := slotRuns[i]; hasRuns {
				uniqueJobs := uniqueStrings(jobIDs)
				if level < len(uniqueJobs) {
					sb.WriteString("  ████")
				} else {
					sb.WriteString("      ")
				}
			} else {
				sb.WriteString("      ")
			}
		}
		sb.WriteString("  │\n")
	}

	sb.WriteString("      │                                                                    │\n")
	sb.WriteString("      └──────────────────────────────────────────────────────────────────┘\n")

	// List jobs
	jobIDsSeen := make(map[string]bool)
	for _, run := range tl.jobRuns {
		if !jobIDsSeen[run.JobID] {
			jobIDsSeen[run.JobID] = true
			info, hasInfo := tl.jobInfo[run.JobID]
			if hasInfo {
				sb.WriteString(fmt.Sprintf("      %s: %s\n", run.JobID, info.Description))
			} else {
				sb.WriteString(fmt.Sprintf("      %s\n", run.JobID))
			}
		}
	}

	return sb.String()
}

// RenderJSON generates a JSON representation of the timeline
func (tl *Timeline) RenderJSON() map[string]interface{} {
	// Group runs by job ID
	jobRunsMap := make(map[string][]time.Time)
	for _, run := range tl.jobRuns {
		jobRunsMap[run.JobID] = append(jobRunsMap[run.JobID], run.RunTime)
	}

	// Build jobs array
	jobs := make([]map[string]interface{}, 0)
	for jobID, runTimes := range jobRunsMap {
		// Sort run times
		sort.Slice(runTimes, func(i, j int) bool {
			return runTimes[i].Before(runTimes[j])
		})

		jobData := map[string]interface{}{
			"id":   jobID,
			"runs": make([]map[string]interface{}, 0),
		}

		// Add job info if available
		if info, hasInfo := tl.jobInfo[jobID]; hasInfo {
			jobData["expression"] = info.Expression
			jobData["description"] = info.Description
		}

		// Add runs
		overlaps := tl.DetectOverlaps()
		overlapMap := make(map[time.Time]int)
		for _, overlap := range overlaps {
			overlapMap[overlap.Time.Truncate(time.Minute)] = overlap.Count
		}

		for _, runTime := range runTimes {
			overlapCount := 0
			if count, hasOverlap := overlapMap[runTime.Truncate(time.Minute)]; hasOverlap {
				overlapCount = count - 1 // Subtract 1 because the job itself is included
			}

			jobData["runs"] = append(jobData["runs"].([]map[string]interface{}), map[string]interface{}{
				"time":     runTime.Format(time.RFC3339),
				"overlaps": overlapCount,
			})
		}

		jobs = append(jobs, jobData)
	}

	// Build overlaps array
	overlaps := tl.DetectOverlaps()
	overlapsJSON := make([]map[string]interface{}, 0, len(overlaps))
	for _, overlap := range overlaps {
		overlapsJSON = append(overlapsJSON, map[string]interface{}{
			"time":  overlap.Time.Format(time.RFC3339),
			"count": overlap.Count,
			"jobs":  overlap.JobIDs,
		})
	}

	return map[string]interface{}{
		"view":      tl.view.String(),
		"startTime": tl.startTime.Format(time.RFC3339),
		"endTime":   tl.endTime.Format(time.RFC3339),
		"width":     tl.width,
		"jobs":      jobs,
		"overlaps":  overlapsJSON,
	}
}

// findSlotIndex finds the slot index for a given time
func (tl *Timeline) findSlotIndex(t time.Time) int {
	if t.Before(tl.startTime) || !t.Before(tl.endTime) {
		return -1
	}

	switch tl.view {
	case DayView:
		// Find which hour slot
		hours := int(t.Sub(tl.startTime).Hours())
		if hours >= 0 && hours < 24 {
			return hours
		}
	case HourView:
		// Find which minute slot
		minutes := int(t.Sub(tl.startTime).Minutes())
		if minutes >= 0 && minutes < 60 {
			return minutes
		}
	}

	return -1
}

// uniqueStrings returns unique strings from a slice
func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
