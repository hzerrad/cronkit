package render

import (
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

// OverlapStats contains statistics about job overlaps
type OverlapStats struct {
	TotalWindows    int
	MaxConcurrent   int
	MostProblematic []Overlap // Top N overlaps sorted by count
}

// jobEntry holds a job's metadata in first-seen order.
type jobEntry struct {
	id, expression, description, label string
}

// Timeline represents a timeline with time slots and job runs
type Timeline struct {
	view      TimelineView
	startTime time.Time
	endTime   time.Time
	width     int
	jobRuns   []JobRun
	jobs      []jobEntry
	jobIndex  map[string]int
}

// NewTimeline creates a new timeline with the specified view, start time, and width
func NewTimeline(view TimelineView, startTime time.Time, width int) *Timeline {
	var endTime time.Time

	switch view {
	case DayView:
		endTime = startTime.Add(24 * time.Hour)
	case HourView:
		endTime = startTime.Add(time.Hour)
	}

	return &Timeline{
		view:      view,
		startTime: startTime,
		endTime:   endTime,
		width:     width,
		jobRuns:   make([]JobRun, 0),
		jobs:      make([]jobEntry, 0),
		jobIndex:  make(map[string]int),
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

// SetJobInfo registers a job's metadata and lane label, keeping first-seen order.
func (tl *Timeline) SetJobInfo(jobID, expression, description, label string) {
	if _, ok := tl.jobIndex[jobID]; ok {
		return
	}
	tl.jobIndex[jobID] = len(tl.jobs)
	tl.jobs = append(tl.jobs, jobEntry{jobID, expression, description, label})
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

// GetOverlapStats returns statistics about overlaps
func (tl *Timeline) GetOverlapStats() OverlapStats {
	overlaps := tl.DetectOverlaps()

	if len(overlaps) == 0 {
		return OverlapStats{
			TotalWindows:    0,
			MaxConcurrent:   0,
			MostProblematic: []Overlap{},
		}
	}

	maxConcurrent := 0
	for _, overlap := range overlaps {
		if overlap.Count > maxConcurrent {
			maxConcurrent = overlap.Count
		}
	}

	// Sort overlaps by count (descending) for most problematic
	mostProblematic := make([]Overlap, len(overlaps))
	copy(mostProblematic, overlaps)
	sort.Slice(mostProblematic, func(i, j int) bool {
		if mostProblematic[i].Count != mostProblematic[j].Count {
			return mostProblematic[i].Count > mostProblematic[j].Count
		}
		return mostProblematic[i].Time.Before(mostProblematic[j].Time)
	})

	// Limit to top 10 most problematic
	if len(mostProblematic) > 10 {
		mostProblematic = mostProblematic[:10]
	}

	return OverlapStats{
		TotalWindows:    len(overlaps),
		MaxConcurrent:   maxConcurrent,
		MostProblematic: mostProblematic,
	}
}

// RenderOptions controls text rendering; JSON output is unaffected.
type RenderOptions struct {
	Color, ASCII, ShowOverlaps bool
}

// Render draws the lane chart timeline as plain text.
func (tl *Timeline) Render(opts RenderOptions) string {
	g := uniGlyphs
	if opts.ASCII {
		g = asciiGlyphs
	}
	var sb strings.Builder
	sb.WriteString(tl.headerLine(g) + "\n\n")
	if len(tl.jobRuns) == 0 {
		sb.WriteString("no runs in this window\n")
		return sb.String()
	}

	overlaps := tl.DetectOverlaps()
	hasConflictRow := len(overlaps) > 0 && len(tl.jobs) > 1

	maxLabel, maxExpr := 0, 0
	if hasConflictRow {
		maxLabel = len([]rune("conflicts"))
	}
	for _, j := range tl.jobs {
		if n := len([]rune(j.label)); n > maxLabel {
			maxLabel = n
		}
		if n := len([]rune(j.expression)); n > maxExpr {
			maxExpr = n
		}
	}
	b := planWidths(tl.width, maxLabel, maxExpr)
	dur := tl.endTime.Sub(tl.startTime)

	counts := make(map[string]map[int]int)
	for _, r := range tl.jobRuns {
		if counts[r.JobID] == nil {
			counts[r.JobID] = make(map[int]int)
		}
		counts[r.JobID][cellPos(r.RunTime, tl.startTime, dur, b.plot)]++
	}

	for i, j := range tl.jobs {
		cells := blankCells(b.plot)
		for col, n := range counts[j.id] {
			cells[col] = g.run
			if n > 1 {
				cells[col] = g.merged
			}
		}
		sb.WriteString(laneRow(j.label, j.expression, cells, b, g, laneColor(i, opts.Color)) + "\n")
	}

	if hasConflictRow {
		cells := blankCells(b.plot)
		for _, o := range overlaps {
			cells[cellPos(o.Time, tl.startTime, dur, b.plot)] = g.conflict
		}
		code := ""
		if opts.Color {
			code = ansiRed
		}
		sb.WriteString(laneRow("conflicts", "", cells, b, g, code) + "\n")
	}

	axis, labels := axisRows(tl.view, tl.startTime, b, g)
	if opts.Color {
		axis, labels = colorize(axis, ansiDim), colorize(labels, ansiDim)
	}
	sb.WriteString(axis + "\n" + labels + "\n\n")
	sb.WriteString(tl.footerLine(len(overlaps), g) + "\n")
	if opts.ShowOverlaps && len(overlaps) > 0 {
		sb.WriteString(tl.overlapLines(overlaps, g))
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
		if idx, hasInfo := tl.jobIndex[jobID]; hasInfo {
			jobData["expression"] = tl.jobs[idx].expression
			jobData["description"] = tl.jobs[idx].description
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

	// Add overlap statistics
	stats := tl.GetOverlapStats()
	mostProblematicJSON := make([]map[string]interface{}, 0, len(stats.MostProblematic))
	for _, overlap := range stats.MostProblematic {
		mostProblematicJSON = append(mostProblematicJSON, map[string]interface{}{
			"time":  overlap.Time.Format(time.RFC3339),
			"count": overlap.Count,
			"jobs":  overlap.JobIDs,
		})
	}

	overlapStatsJSON := map[string]interface{}{
		"totalWindows":    stats.TotalWindows,
		"maxConcurrent":   stats.MaxConcurrent,
		"mostProblematic": mostProblematicJSON,
	}

	return map[string]interface{}{
		"view":         tl.view.String(),
		"startTime":    tl.startTime.Format(time.RFC3339),
		"endTime":      tl.endTime.Format(time.RFC3339),
		"width":        tl.width,
		"jobs":         jobs,
		"overlaps":     overlapsJSON,
		"overlapStats": overlapStatsJSON,
	}
}
