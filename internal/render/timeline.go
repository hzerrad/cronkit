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

// OverlapStats contains statistics about job overlaps
type OverlapStats struct {
	TotalWindows    int
	MaxConcurrent   int
	MostProblematic []Overlap // Top N overlaps sorted by count
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

// Render generates an ASCII timeline string with optional overlap reporting
func (tl *Timeline) Render(showOverlaps bool) string {
	var sb strings.Builder

	// Collect job descriptions early
	jobIDsSeen := make(map[string]bool)
	jobList := make([]struct {
		jobID       string
		expression  string
		description string
	}, 0)

	for _, run := range tl.jobRuns {
		if !jobIDsSeen[run.JobID] {
			jobIDsSeen[run.JobID] = true
			info, hasInfo := tl.jobInfo[run.JobID]
			if hasInfo {
				jobList = append(jobList, struct {
					jobID       string
					expression  string
					description string
				}{
					jobID:       run.JobID,
					expression:  info.Expression,
					description: info.Description,
				})
			} else {
				jobList = append(jobList, struct {
					jobID       string
					expression  string
					description string
				}{
					jobID:       run.JobID,
					expression:  "",
					description: "",
				})
			}
		}
	}

	// Header
	var timeRange string
	var endTimeDisplay time.Time
	if tl.view == DayView {
		// For day view, show 23:59 as the end time
		endTimeDisplay = tl.endTime.Add(-1 * time.Minute) // Show 23:59 instead of 00:00 next day
		timeRange = fmt.Sprintf("%s ──────────────────────────────────────────────────────────────── %s",
			tl.startTime.Format("15:04"), endTimeDisplay.Format("15:04"))
		sb.WriteString(fmt.Sprintf("Timeline for %s (Day View)\n", tl.startTime.Format("2006-01-02")))
	} else {
		// For hour view, show 59 as the end time
		endTimeDisplay = tl.endTime.Add(-1 * time.Minute) // Show 59 instead of 60
		timeRange = fmt.Sprintf("%s ──────────────────────────────────────────────────────────────── %s",
			tl.startTime.Format("15:04"), endTimeDisplay.Format("15:04"))
		sb.WriteString(fmt.Sprintf("Timeline for %s (Hour View)\n", tl.startTime.Format("2006-01-02 15:04")))
	}

	// Display job descriptions right after the header
	for _, job := range jobList {
		if job.description != "" {
			// For single expressions, show just the description
			if strings.HasPrefix(job.jobID, "expr-") {
				sb.WriteString(fmt.Sprintf("  • %s\n", job.description))
			} else {
				// For crontab jobs, show description with expression in parentheses
				sb.WriteString(fmt.Sprintf("  • %s (%s)\n", job.description, job.expression))
			}
		} else {
			// Fallback to job ID if no description
			sb.WriteString(fmt.Sprintf("  • %s\n", job.jobID))
		}
	}

	sb.WriteString(timeRange + "\n")

	// Calculate available width for timeline bars
	// Account for: "      │" (7 chars) + "  │" (3 chars) = 10 chars fixed
	availableWidth := tl.width - 10
	if availableWidth < 0 {
		availableWidth = 0
	}

	// Draw top border with adaptive width
	sb.WriteString("      │")
	for i := 0; i < availableWidth; i++ {
		sb.WriteString(" ")
	}
	sb.WriteString("  │\n")

	// Group runs by time (rounded to minute for grouping)
	timeRuns := make(map[time.Time][]string) // time -> job IDs
	for _, run := range tl.jobRuns {
		// Round to minute for grouping
		roundedTime := run.RunTime.Truncate(time.Minute)
		timeRuns[roundedTime] = append(timeRuns[roundedTime], run.JobID)
	}

	// Render timeline bars
	maxOverlaps := 1
	for _, jobIDs := range timeRuns {
		uniqueCount := len(uniqueStrings(jobIDs))
		if uniqueCount > maxOverlaps {
			maxOverlaps = uniqueCount
		}
	}

	// Calculate time range for proportional mapping
	durationRange := tl.endTime.Sub(tl.startTime)

	// Draw execution markers for each overlap level
	// Use discrete markers (|) to show individual executions
	for level := 0; level < maxOverlaps; level++ {
		sb.WriteString("      │")

		// Handle edge case when availableWidth is 0 or very small
		if availableWidth <= 0 {
			sb.WriteString("  │\n")
			continue
		}

		// Create a character array for the timeline
		timelineChars := make([]rune, availableWidth)
		for i := range timelineChars {
			timelineChars[i] = ' '
		}

		// Map each execution time directly to its proportional position
		// Sort execution times to handle overlaps better
		sortedTimes := make([]time.Time, 0, len(timeRuns))
		for execTime := range timeRuns {
			if !execTime.Before(tl.startTime) && execTime.Before(tl.endTime) {
				sortedTimes = append(sortedTimes, execTime)
			}
		}
		sort.Slice(sortedTimes, func(i, j int) bool {
			return sortedTimes[i].Before(sortedTimes[j])
		})

		for _, execTime := range sortedTimes {
			jobIDs := timeRuns[execTime]
			uniqueJobs := uniqueStrings(jobIDs)
			if level < len(uniqueJobs) {
				// Calculate position based on time offset from start
				timeOffset := execTime.Sub(tl.startTime)
				if durationRange > 0 {
					// Map time offset proportionally to timeline width
					posFloat := float64(timeOffset) / float64(durationRange) * float64(availableWidth)
					pos := int(posFloat + 0.5) // Round to nearest
					if pos < 0 {
						pos = 0
					}
					if pos >= availableWidth {
						pos = availableWidth - 1
					}

					// Try to place marker, avoiding overlaps when possible
					// If position is occupied, try adjacent positions
					placed := false
					for offset := 0; offset < 3 && !placed; offset++ {
						for direction := -1; direction <= 1 && !placed; direction += 2 {
							if offset == 0 && direction == -1 {
								continue // Skip offset 0, direction -1 (already tried)
							}
							tryPos := pos + (offset * direction)
							if tryPos >= 0 && tryPos < availableWidth {
								if timelineChars[tryPos] == ' ' {
									if len(uniqueJobs) > 1 {
										// Multiple jobs at same time - use density character
										timelineChars[tryPos] = []rune(getDensityChar(len(uniqueJobs), maxOverlaps))[0]
									} else {
										// Single execution - use discrete marker
										timelineChars[tryPos] = '│'
									}
									placed = true
								}
							}
						}
					}
					// If still not placed (all positions occupied), just overwrite
					if !placed {
						if len(uniqueJobs) > 1 {
							timelineChars[pos] = []rune(getDensityChar(len(uniqueJobs), maxOverlaps))[0]
						} else {
							timelineChars[pos] = '│'
						}
					}
				}
			}
		}

		// Write the timeline line
		sb.WriteString(string(timelineChars))
		sb.WriteString("  │\n")
	}

	// Draw bottom border with adaptive width
	sb.WriteString("      │")
	for i := 0; i < availableWidth; i++ {
		sb.WriteString(" ")
	}
	sb.WriteString("  │\n")

	// Draw bottom edge with time markers
	sb.WriteString("      └")
	for i := 0; i < availableWidth; i++ {
		sb.WriteString("─")
	}
	sb.WriteString("──┘\n")

	// Add time markers below the timeline
	if tl.view == DayView && availableWidth >= 40 {
		// Show markers at 0, 6, 12, 18, 24 hours for day view
		// Calculate marker times
		markerTimes := []time.Time{
			tl.startTime,
			tl.startTime.Add(6 * time.Hour),
			tl.startTime.Add(12 * time.Hour),
			tl.startTime.Add(18 * time.Hour),
			tl.startTime.Add(23*time.Hour + 59*time.Minute),
		}
		markerLabels := []string{"00:00", "06:00", "12:00", "18:00", "23:59"}
		sb.WriteString("      ")
		lastPos := 0
		for i, markerTime := range markerTimes {
			if !markerTime.Before(tl.startTime) && markerTime.Before(tl.endTime) {
				// Map time position proportionally to timeline width
				timeOffset := markerTime.Sub(tl.startTime)
				markerX := int(float64(timeOffset) / float64(durationRange) * float64(availableWidth))
				if markerX >= availableWidth {
					markerX = availableWidth - 1
				}
				// Add spaces to reach marker position
				for j := lastPos; j < markerX && j < availableWidth; j++ {
					sb.WriteString(" ")
				}
				// Write marker label
				label := markerLabels[i]
				// Center the label on the marker position if there's room
				labelStart := markerX
				if markerX+len(label) > availableWidth {
					labelStart = availableWidth - len(label)
					if labelStart < 0 {
						labelStart = 0
					}
				}
				// Fill gap if needed
				for j := lastPos; j < labelStart && j < availableWidth; j++ {
					sb.WriteString(" ")
				}
				if labelStart+len(label) <= availableWidth {
					sb.WriteString(label)
					lastPos = labelStart + len(label)
				} else {
					lastPos = markerX
				}
			}
		}
		sb.WriteString("\n")
	} else if tl.view == HourView && availableWidth >= 40 {
		// Show markers at 0, 15, 30, 45, 60 minutes for hour view
		// Calculate marker times
		markerTimes := []time.Time{
			tl.startTime,
			tl.startTime.Add(15 * time.Minute),
			tl.startTime.Add(30 * time.Minute),
			tl.startTime.Add(45 * time.Minute),
			tl.startTime.Add(59 * time.Minute),
		}
		markerLabels := []string{"00", "15", "30", "45", "59"}
		sb.WriteString("      ")
		lastPos := 0
		for i, markerTime := range markerTimes {
			if !markerTime.Before(tl.startTime) && markerTime.Before(tl.endTime) {
				// Map time position proportionally to timeline width
				timeOffset := markerTime.Sub(tl.startTime)
				markerX := int(float64(timeOffset) / float64(durationRange) * float64(availableWidth))
				if markerX >= availableWidth {
					markerX = availableWidth - 1
				}
				// Add spaces to reach marker position
				for j := lastPos; j < markerX && j < availableWidth; j++ {
					sb.WriteString(" ")
				}
				// Write marker label
				label := markerLabels[i]
				labelStart := markerX
				if markerX+len(label) > availableWidth {
					labelStart = availableWidth - len(label)
					if labelStart < 0 {
						labelStart = 0
					}
				}
				// Fill gap if needed
				for j := lastPos; j < labelStart && j < availableWidth; j++ {
					sb.WriteString(" ")
				}
				if labelStart+len(label) <= availableWidth {
					sb.WriteString(label)
					lastPos = labelStart + len(label)
				} else {
					lastPos = markerX
				}
			}
		}
		sb.WriteString("\n")
	}

	// Add legend
	sb.WriteString("\n")
	sb.WriteString("Legend: │ = Job execution time | Each marker represents one execution\n")

	// Add overlap summary if requested
	if showOverlaps {
		overlaps := tl.DetectOverlaps()
		stats := tl.GetOverlapStats()

		sb.WriteString("\n")
		sb.WriteString("━━━ Overlap Summary ━━━\n")

		if len(overlaps) == 0 {
			sb.WriteString("No overlaps detected\n")
		} else {
			sb.WriteString(fmt.Sprintf("Total overlap windows: %d\n", stats.TotalWindows))
			sb.WriteString(fmt.Sprintf("Maximum concurrent jobs: %d\n", stats.MaxConcurrent))
			sb.WriteString("\n")
			sb.WriteString("Overlaps:\n")

			// Show all overlaps, or limit to first 50 if too many
			displayOverlaps := overlaps
			if len(displayOverlaps) > 50 {
				displayOverlaps = displayOverlaps[:50]
				sb.WriteString(fmt.Sprintf("  (showing first 50 of %d overlap windows)\n", len(overlaps)))
			}

			for _, overlap := range displayOverlaps {
				jobList := strings.Join(overlap.JobIDs, ", ")
				sb.WriteString(fmt.Sprintf("  %s: %d job(s) (%s)\n",
					overlap.Time.Format("2006-01-02 15:04:05"),
					overlap.Count,
					jobList))
			}

			if len(overlaps) > 50 {
				sb.WriteString(fmt.Sprintf("  ... and %d more overlap window(s)\n", len(overlaps)-50))
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

// getDensityChar returns a character representing density level
// Higher density = darker/more solid character
func getDensityChar(overlapCount, maxOverlaps int) string {
	if maxOverlaps == 0 {
		return "█"
	}

	// Normalize to 0-1 range
	density := float64(overlapCount) / float64(maxOverlaps)

	// Use different characters based on density
	if density >= 0.8 {
		return "█" // Full block for high density
	} else if density >= 0.6 {
		return "▓" // Dark shade
	} else if density >= 0.4 {
		return "▒" // Medium shade
	} else if density >= 0.2 {
		return "░" // Light shade
	}
	return "·" // Dot for very low density
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
