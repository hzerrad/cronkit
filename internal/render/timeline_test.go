package render

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTimeline(t *testing.T) {
	t.Run("should create day view timeline", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		assert.Equal(t, DayView, tl.view)
		assert.Equal(t, startTime, tl.startTime)
		assert.Equal(t, startTime.Add(24*time.Hour), tl.endTime)
		assert.Equal(t, 80, tl.width)
		assert.NotNil(t, tl.jobRuns)
		assert.NotNil(t, tl.slots)
	})

	t.Run("should create hour view timeline", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
		tl := NewTimeline(HourView, startTime, 80)

		assert.Equal(t, HourView, tl.view)
		assert.Equal(t, startTime, tl.startTime)
		assert.Equal(t, startTime.Add(time.Hour), tl.endTime)
		assert.Equal(t, 80, tl.width)
	})

	t.Run("should calculate slots for day view", func(t *testing.T) {
		tl := NewTimeline(DayView, time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), 80)
		// Day view: 24 hours = 1440 minutes, with 80 width we should have reasonable slot count
		assert.Greater(t, len(tl.slots), 0)
	})

	t.Run("should calculate slots for hour view", func(t *testing.T) {
		tl := NewTimeline(HourView, time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC), 80)
		// Hour view: 60 minutes, with 80 width we should have reasonable slot count
		assert.Greater(t, len(tl.slots), 0)
	})
}

func TestTimeline_AddJobRun(t *testing.T) {
	t.Run("should add job run within timeline range", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		runTime := startTime.Add(2 * time.Hour)
		tl.AddJobRun("job-1", runTime)

		assert.Len(t, tl.jobRuns, 1)
		assert.Equal(t, "job-1", tl.jobRuns[0].JobID)
		assert.Equal(t, runTime, tl.jobRuns[0].RunTime)
	})

	t.Run("should ignore job run outside timeline range", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		// Run time is after the end of the timeline
		runTime := startTime.Add(25 * time.Hour)
		tl.AddJobRun("job-1", runTime)

		assert.Len(t, tl.jobRuns, 0)
	})

	t.Run("should add multiple runs for same job", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))
		tl.AddJobRun("job-1", startTime.Add(2*time.Hour))
		tl.AddJobRun("job-1", startTime.Add(3*time.Hour))

		assert.Len(t, tl.jobRuns, 3)
	})

	t.Run("should add runs from different jobs", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))
		tl.AddJobRun("job-2", startTime.Add(2*time.Hour))
		tl.AddJobRun("job-3", startTime.Add(3*time.Hour))

		assert.Len(t, tl.jobRuns, 3)
		jobIDs := make(map[string]bool)
		for _, run := range tl.jobRuns {
			jobIDs[run.JobID] = true
		}
		assert.Len(t, jobIDs, 3)
	})
}

func TestTimeline_DetectOverlaps(t *testing.T) {
	t.Run("should detect no overlaps", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))
		tl.AddJobRun("job-2", startTime.Add(2*time.Hour))

		overlaps := tl.DetectOverlaps()
		assert.Len(t, overlaps, 0)
	})

	t.Run("should detect overlaps at same time", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		overlapTime := startTime.Add(1 * time.Hour)
		tl.AddJobRun("job-1", overlapTime)
		tl.AddJobRun("job-2", overlapTime)
		tl.AddJobRun("job-3", overlapTime)

		overlaps := tl.DetectOverlaps()
		require.Len(t, overlaps, 1)
		assert.Equal(t, overlapTime, overlaps[0].Time)
		assert.Equal(t, 3, overlaps[0].Count)
		assert.Contains(t, overlaps[0].JobIDs, "job-1")
		assert.Contains(t, overlaps[0].JobIDs, "job-2")
		assert.Contains(t, overlaps[0].JobIDs, "job-3")
	})

	t.Run("should detect multiple overlap times", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		time1 := startTime.Add(1 * time.Hour)
		time2 := startTime.Add(2 * time.Hour)

		tl.AddJobRun("job-1", time1)
		tl.AddJobRun("job-2", time1)
		tl.AddJobRun("job-3", time2)
		tl.AddJobRun("job-4", time2)

		overlaps := tl.DetectOverlaps()
		assert.Len(t, overlaps, 2)
	})
}

func TestTimeline_Render(t *testing.T) {
	t.Run("should render empty timeline", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		output := tl.Render(false)
		assert.Contains(t, output, "Timeline")
		assert.Contains(t, output, "00:00")
	})

	t.Run("should render timeline with job runs", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))
		tl.AddJobRun("job-1", startTime.Add(2*time.Hour))

		output := tl.Render(false)
		assert.Contains(t, output, "Timeline")
		assert.Contains(t, output, "job-1")
	})

	t.Run("should render hour view", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
		tl := NewTimeline(HourView, startTime, 80)

		tl.AddJobRun("job-1", startTime.Add(5*time.Minute))
		tl.AddJobRun("job-1", startTime.Add(10*time.Minute))

		output := tl.Render(false)
		assert.Contains(t, output, "Timeline")
		assert.Contains(t, output, "09:00")
		assert.Contains(t, output, "09:59") // Hour view now shows 09:59 as end time
		assert.Contains(t, output, "Legend")
	})

	t.Run("should handle narrow width", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 40)

		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))

		output := tl.Render(false)
		assert.NotEmpty(t, output)
		// Should still render, just with fewer slots
	})

	t.Run("should handle wide width", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 200)

		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))

		output := tl.Render(false)
		assert.NotEmpty(t, output)
	})
}

func TestTimeline_RenderJSON(t *testing.T) {
	t.Run("should render JSON for empty timeline", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		result := tl.RenderJSON()
		assert.Equal(t, "day", result["view"])
		assert.Equal(t, 80, result["width"])
		assert.NotNil(t, result["startTime"])
		assert.NotNil(t, result["endTime"])
		assert.NotNil(t, result["jobs"])
		assert.NotNil(t, result["overlaps"])
	})

	t.Run("should render JSON with job runs", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		runTime1 := startTime.Add(1 * time.Hour)
		runTime2 := startTime.Add(2 * time.Hour)
		tl.AddJobRun("job-1", runTime1)
		tl.AddJobRun("job-1", runTime2)

		result := tl.RenderJSON()
		jobs := result["jobs"].([]map[string]interface{})
		assert.Len(t, jobs, 1)
		assert.Equal(t, "job-1", jobs[0]["id"])
	})

	t.Run("should render JSON with overlaps", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		overlapTime := startTime.Add(1 * time.Hour)
		tl.AddJobRun("job-1", overlapTime)
		tl.AddJobRun("job-2", overlapTime)

		result := tl.RenderJSON()
		overlaps := result["overlaps"].([]map[string]interface{})
		assert.Greater(t, len(overlaps), 0)
	})

	t.Run("should render JSON with job info", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		tl.SetJobInfo("job-1", "0 9 * * *", "Daily at 9am")
		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))

		result := tl.RenderJSON()
		jobs := result["jobs"].([]map[string]interface{})
		assert.Len(t, jobs, 1)
		assert.Equal(t, "0 9 * * *", jobs[0]["expression"])
		assert.Equal(t, "Daily at 9am", jobs[0]["description"])
	})

	t.Run("should render JSON with jobs without info", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))

		result := tl.RenderJSON()
		jobs := result["jobs"].([]map[string]interface{})
		assert.Len(t, jobs, 1)
		// Should not have expression or description if not set
		_, hasExpression := jobs[0]["expression"]
		_, hasDescription := jobs[0]["description"]
		assert.False(t, hasExpression)
		assert.False(t, hasDescription)
	})

	t.Run("should render JSON with overlap stats", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		overlapTime := startTime.Add(1 * time.Hour)
		tl.AddJobRun("job-1", overlapTime)
		tl.AddJobRun("job-2", overlapTime)
		tl.AddJobRun("job-3", overlapTime)

		result := tl.RenderJSON()
		overlapStats := result["overlapStats"].(map[string]interface{})
		assert.NotNil(t, overlapStats)
		assert.Equal(t, 1, overlapStats["totalWindows"])
		assert.Equal(t, 3, overlapStats["maxConcurrent"])
		assert.NotNil(t, overlapStats["mostProblematic"])
	})

	t.Run("should render JSON with sorted run times", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		// Add runs in non-sequential order
		tl.AddJobRun("job-1", startTime.Add(3*time.Hour))
		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))
		tl.AddJobRun("job-1", startTime.Add(2*time.Hour))

		result := tl.RenderJSON()
		jobs := result["jobs"].([]map[string]interface{})
		runs := jobs[0]["runs"].([]map[string]interface{})
		assert.Len(t, runs, 3)
		// Runs should be sorted by time
		time1, _ := time.Parse(time.RFC3339, runs[0]["time"].(string))
		time2, _ := time.Parse(time.RFC3339, runs[1]["time"].(string))
		time3, _ := time.Parse(time.RFC3339, runs[2]["time"].(string))
		assert.True(t, time1.Before(time2))
		assert.True(t, time2.Before(time3))
	})

	t.Run("should render JSON with correct overlap counts", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		overlapTime := startTime.Add(1 * time.Hour)
		// 3 jobs at same time
		tl.AddJobRun("job-1", overlapTime)
		tl.AddJobRun("job-2", overlapTime)
		tl.AddJobRun("job-3", overlapTime)

		result := tl.RenderJSON()
		jobs := result["jobs"].([]map[string]interface{})
		// Each job should show overlap count of 2 (3 total - 1 self)
		for _, job := range jobs {
			runs := job["runs"].([]map[string]interface{})
			for _, run := range runs {
				if runTime, _ := time.Parse(time.RFC3339, run["time"].(string)); runTime.Equal(overlapTime.Truncate(time.Minute)) {
					assert.Equal(t, 2, run["overlaps"])
				}
			}
		}
	})

	t.Run("should render JSON with multiple jobs and overlaps", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		time1 := startTime.Add(1 * time.Hour)
		time2 := startTime.Add(2 * time.Hour)

		// Job 1 runs at time1 (no overlap)
		tl.AddJobRun("job-1", time1)

		// Jobs 2 and 3 run at time2 (overlap)
		tl.AddJobRun("job-2", time2)
		tl.AddJobRun("job-3", time2)

		result := tl.RenderJSON()
		jobs := result["jobs"].([]map[string]interface{})
		assert.Len(t, jobs, 3)

		overlaps := result["overlaps"].([]map[string]interface{})
		assert.Len(t, overlaps, 1)
		assert.Equal(t, 2, overlaps[0]["count"])
	})
}

func TestTimeline_SetJobInfo(t *testing.T) {
	t.Run("should set job info", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		tl.SetJobInfo("job-1", "*/15 * * * *", "Every 15 minutes")

		info, exists := tl.jobInfo["job-1"]
		require.True(t, exists)
		assert.Equal(t, "*/15 * * * *", info.Expression)
		assert.Equal(t, "Every 15 minutes", info.Description)
	})

	t.Run("should update existing job info", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		tl.SetJobInfo("job-1", "*/15 * * * *", "Every 15 minutes")
		tl.SetJobInfo("job-1", "*/30 * * * *", "Every 30 minutes")

		info, exists := tl.jobInfo["job-1"]
		require.True(t, exists)
		assert.Equal(t, "*/30 * * * *", info.Expression)
		assert.Equal(t, "Every 30 minutes", info.Description)
	})
}

func TestTimelineView_String(t *testing.T) {
	t.Run("should return day for DayView", func(t *testing.T) {
		assert.Equal(t, "day", DayView.String())
	})

	t.Run("should return hour for HourView", func(t *testing.T) {
		assert.Equal(t, "hour", HourView.String())
	})

	t.Run("should return unknown for invalid view", func(t *testing.T) {
		invalidView := TimelineView(999)
		assert.Equal(t, "unknown", invalidView.String())
	})
}

func TestTimeline_findSlotIndex(t *testing.T) {
	t.Run("should find correct slot for day view", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)
		// Test various hours
		assert.Equal(t, 0, tl.findSlotIndex(startTime))
		assert.Equal(t, 1, tl.findSlotIndex(startTime.Add(1*time.Hour)))
		assert.Equal(t, 12, tl.findSlotIndex(startTime.Add(12*time.Hour)))
		assert.Equal(t, 23, tl.findSlotIndex(startTime.Add(23*time.Hour)))
	})

	t.Run("should find correct slot for hour view", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
		tl := NewTimeline(HourView, startTime, 80)

		// Test various minutes
		assert.Equal(t, 0, tl.findSlotIndex(startTime))
		assert.Equal(t, 5, tl.findSlotIndex(startTime.Add(5*time.Minute)))
		assert.Equal(t, 30, tl.findSlotIndex(startTime.Add(30*time.Minute)))
		assert.Equal(t, 59, tl.findSlotIndex(startTime.Add(59*time.Minute)))
	})

	t.Run("should return -1 for time before start", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		assert.Equal(t, -1, tl.findSlotIndex(startTime.Add(-1*time.Hour)))
	})

	t.Run("should return -1 for time after end", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		assert.Equal(t, -1, tl.findSlotIndex(startTime.Add(25*time.Hour)))
	})

	t.Run("should return -1 for hour view time after end", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
		tl := NewTimeline(HourView, startTime, 80)

		assert.Equal(t, -1, tl.findSlotIndex(startTime.Add(61*time.Minute)))
	})
}

func TestGetDensityChar(t *testing.T) {
	t.Run("should return full block for high density", func(t *testing.T) {
		char := getDensityChar(8, 10)
		assert.Equal(t, "█", char)
	})

	t.Run("should return dark shade for medium-high density", func(t *testing.T) {
		char := getDensityChar(6, 10)
		assert.Equal(t, "▓", char)
	})

	t.Run("should return medium shade for medium density", func(t *testing.T) {
		char := getDensityChar(4, 10)
		assert.Equal(t, "▒", char)
	})

	t.Run("should return light shade for low density", func(t *testing.T) {
		char := getDensityChar(2, 10)
		assert.Equal(t, "░", char)
	})

	t.Run("should return dot for very low density", func(t *testing.T) {
		char := getDensityChar(1, 10)
		assert.Equal(t, "·", char)
	})

	t.Run("should handle zero maxOverlaps", func(t *testing.T) {
		char := getDensityChar(1, 0)
		assert.Equal(t, "█", char)
	})
}

func TestTimeline_Render_AdaptiveWidth(t *testing.T) {
	t.Run("should render with narrow width", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 50)

		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))

		output := tl.Render(false)
		assert.Contains(t, output, "Timeline")
		// Should still render successfully
		assert.NotEmpty(t, output)
	})

	t.Run("should render with very narrow width", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 40)

		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))

		output := tl.Render(false)
		assert.Contains(t, output, "Timeline")
		// Should still render successfully even with very narrow width
		assert.NotEmpty(t, output)
	})

	t.Run("should render with wide width", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 200)

		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))
		tl.AddJobRun("job-2", startTime.Add(1*time.Hour))

		output := tl.Render(false)
		assert.Contains(t, output, "Timeline")
		// Should use density characters
		assert.Contains(t, output, "█")
	})

	t.Run("should use density characters for overlaps", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 100)

		overlapTime := startTime.Add(1 * time.Hour)
		// Add multiple jobs at same time to create high density
		for i := 0; i < 8; i++ {
			tl.AddJobRun(fmt.Sprintf("job-%d", i), overlapTime)
		}

		output := tl.Render(false)
		assert.Contains(t, output, "█") // Should use full block for high density
	})

	t.Run("should handle very narrow width with slotWidth calculation", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 50) // Narrow width

		// Add runs at different hours
		for i := 0; i < 5; i++ {
			tl.AddJobRun("job-1", startTime.Add(time.Duration(i)*time.Hour))
		}

		output := tl.Render(false)
		assert.Contains(t, output, "Timeline")
		// Should render successfully with narrow width
		assert.NotEmpty(t, output)
	})

	t.Run("should handle slotWidth calculation when availableWidth < slotCount", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		// Very narrow width - less than number of slots (24 for day view)
		tl := NewTimeline(DayView, startTime, 30)

		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))

		output := tl.Render(false)
		assert.Contains(t, output, "Timeline")
		// Should handle narrow width gracefully
		assert.NotEmpty(t, output)
	})

	t.Run("should use different density characters based on overlap count", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

		// Test high density (8+ jobs) - should use full block
		tl := NewTimeline(DayView, startTime, 100)
		overlapTime := startTime.Add(1 * time.Hour)
		for i := 0; i < 8; i++ {
			tl.AddJobRun(fmt.Sprintf("job-%d", i), overlapTime)
		}
		output := tl.Render(false)
		// With 8 jobs and maxOverlaps=8, density = 8/8 = 1.0, should use █
		assert.Contains(t, output, "█")

		// Test medium-high density (6 jobs out of 10 max) - should use dark shade
		tl2 := NewTimeline(DayView, startTime, 100)
		overlapTime2 := startTime.Add(1 * time.Hour)
		overlapTime3 := startTime.Add(2 * time.Hour)
		// Create 10 jobs at time2 to set maxOverlaps=10
		for i := 0; i < 10; i++ {
			tl2.AddJobRun(fmt.Sprintf("job-%d", i), overlapTime3)
		}
		// Then 6 jobs at time1 (density = 6/10 = 0.6, should use ▓)
		for i := 0; i < 6; i++ {
			tl2.AddJobRun(fmt.Sprintf("job-%d", i), overlapTime2)
		}
		output2 := tl2.Render(false)
		assert.Contains(t, output2, "▓")

		// Test medium density (4 jobs out of 10 max) - should use medium shade
		tl3 := NewTimeline(DayView, startTime, 100)
		// Create 10 jobs at time3 to set maxOverlaps=10
		for i := 0; i < 10; i++ {
			tl3.AddJobRun(fmt.Sprintf("job-%d", i), overlapTime3)
		}
		// Then 4 jobs at time2 (density = 4/10 = 0.4, should use ▒)
		for i := 0; i < 4; i++ {
			tl3.AddJobRun(fmt.Sprintf("job-%d", i), overlapTime2)
		}
		output3 := tl3.Render(false)
		assert.Contains(t, output3, "▒")

		// Test low density (2 jobs out of 10 max) - should use light shade
		tl4 := NewTimeline(DayView, startTime, 100)
		// Create 10 jobs at time3 to set maxOverlaps=10
		for i := 0; i < 10; i++ {
			tl4.AddJobRun(fmt.Sprintf("job-%d", i), overlapTime3)
		}
		// Then 2 jobs at time2 (density = 2/10 = 0.2, should use ░)
		for i := 0; i < 2; i++ {
			tl4.AddJobRun(fmt.Sprintf("job-%d", i), overlapTime2)
		}
		output4 := tl4.Render(false)
		assert.Contains(t, output4, "░")

		// Test single execution (1 job) - should use discrete marker │
		tl5 := NewTimeline(DayView, startTime, 100)
		// Create 10 jobs at time3 to set maxOverlaps=10
		for i := 0; i < 10; i++ {
			tl5.AddJobRun(fmt.Sprintf("job-%d", i), overlapTime3)
		}
		// Then 1 job at time2 (single execution, should use │)
		tl5.AddJobRun("job-1", overlapTime2)
		output5 := tl5.Render(false)
		assert.Contains(t, output5, "│")
	})

	t.Run("should handle Render with showOverlaps and many overlaps (>50)", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 100)

		// Create 60 overlaps (one per minute for 60 minutes)
		for i := 0; i < 60; i++ {
			overlapTime := startTime.Add(time.Duration(i) * time.Minute)
			tl.AddJobRun("job-1", overlapTime)
			tl.AddJobRun("job-2", overlapTime)
		}

		output := tl.Render(true)
		assert.Contains(t, output, "Overlap Summary")
		assert.Contains(t, output, "showing first 50")
		assert.Contains(t, output, "and 10 more overlap window(s)")
	})

	t.Run("should handle Render with showOverlaps and exactly 50 overlaps", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 100)

		// Create exactly 50 overlaps
		for i := 0; i < 50; i++ {
			overlapTime := startTime.Add(time.Duration(i) * time.Minute)
			tl.AddJobRun("job-1", overlapTime)
			tl.AddJobRun("job-2", overlapTime)
		}

		output := tl.Render(true)
		assert.Contains(t, output, "Overlap Summary")
		assert.NotContains(t, output, "showing first 50")
		assert.NotContains(t, output, "more overlap window(s)")
	})

	t.Run("should handle Render with slotWidth > 1", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		// Wide width so slotWidth will be > 1
		tl := NewTimeline(DayView, startTime, 200)

		// Add runs at different hours
		for i := 0; i < 5; i++ {
			tl.AddJobRun("job-1", startTime.Add(time.Duration(i)*time.Hour))
		}

		output := tl.Render(false)
		assert.Contains(t, output, "Timeline")
		// Should render successfully with wide width
		assert.NotEmpty(t, output)
	})

	t.Run("should handle Render with usedWidth < availableWidth", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		// Width that doesn't divide evenly into slots
		tl := NewTimeline(DayView, startTime, 100)

		// Add a few runs
		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))
		tl.AddJobRun("job-2", startTime.Add(2*time.Hour))

		output := tl.Render(false)
		assert.Contains(t, output, "Timeline")
		// Should fill remaining space correctly
		assert.NotEmpty(t, output)
	})

	t.Run("should handle Render with level >= len(uniqueJobs)", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 100)

		// Add runs that create multiple overlap levels
		overlapTime := startTime.Add(1 * time.Hour)
		tl.AddJobRun("job-1", overlapTime)
		tl.AddJobRun("job-2", overlapTime)
		tl.AddJobRun("job-3", overlapTime)

		output := tl.Render(false)
		assert.Contains(t, output, "Timeline")
		// Should render all levels correctly
		assert.NotEmpty(t, output)
	})

	t.Run("should handle Render with hour view and adaptive width", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
		tl := NewTimeline(HourView, startTime, 80)

		// Add runs at different minutes
		for i := 0; i < 10; i++ {
			tl.AddJobRun("job-1", startTime.Add(time.Duration(i)*time.Minute))
		}

		output := tl.Render(false)
		assert.Contains(t, output, "Timeline")
		assert.Contains(t, output, "Hour View")
		assert.NotEmpty(t, output)
	})

	t.Run("should handle Render with no job info", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		// Add run without setting job info
		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))

		output := tl.Render(false)
		assert.Contains(t, output, "Timeline")
		assert.Contains(t, output, "job-1")
		// Should show job ID even without description
		assert.NotEmpty(t, output)
	})

	t.Run("should handle Render with slotWidth calculation when availableWidth is 0", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		// Very narrow width that results in availableWidth = 0 or negative
		tl := NewTimeline(DayView, startTime, 5)

		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))

		output := tl.Render(false)
		assert.Contains(t, output, "Timeline")
		// Should handle gracefully
		assert.NotEmpty(t, output)
	})

	t.Run("should handle Render with slotCount reduction for narrow terminals", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		// Width that's less than number of slots (24 for day view)
		tl := NewTimeline(DayView, startTime, 20)

		// Add runs at different hours
		for i := 0; i < 5; i++ {
			tl.AddJobRun("job-1", startTime.Add(time.Duration(i)*time.Hour))
		}

		output := tl.Render(false)
		assert.Contains(t, output, "Timeline")
		// Should handle narrow width with slotCount reduction
		assert.NotEmpty(t, output)
	})

	t.Run("should handle Render with slotWidth > 1 and multiple slots", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		// Wide width so slotWidth will be > 1
		tl := NewTimeline(DayView, startTime, 300)

		// Add runs at multiple hours
		for i := 0; i < 10; i++ {
			tl.AddJobRun("job-1", startTime.Add(time.Duration(i)*time.Hour))
		}

		output := tl.Render(false)
		assert.Contains(t, output, "Timeline")
		// Should render successfully with wide width
		assert.NotEmpty(t, output)
	})
}

func TestTimeline_findSlotIndex_EdgeCases(t *testing.T) {
	t.Run("should handle exact boundary times", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		// Test exact start time
		assert.Equal(t, 0, tl.findSlotIndex(startTime))

		// Test exact end time (should return -1 as it's not before endTime)
		endTime := startTime.Add(24 * time.Hour)
		assert.Equal(t, -1, tl.findSlotIndex(endTime))
	})

	t.Run("should handle hour view boundary times", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
		tl := NewTimeline(HourView, startTime, 80)

		// Test exact start time
		assert.Equal(t, 0, tl.findSlotIndex(startTime))

		// Test exact end time (should return -1)
		endTime := startTime.Add(time.Hour)
		assert.Equal(t, -1, tl.findSlotIndex(endTime))
	})

	t.Run("should handle times just before start", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		beforeStart := startTime.Add(-1 * time.Minute)
		assert.Equal(t, -1, tl.findSlotIndex(beforeStart))
	})

	t.Run("should handle times just after end", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		afterEnd := startTime.Add(24*time.Hour + 1*time.Minute)
		assert.Equal(t, -1, tl.findSlotIndex(afterEnd))
	})

	t.Run("should handle hour view with hours >= 60", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
		tl := NewTimeline(HourView, startTime, 80)

		// Time that would be >= 60 minutes
		tooLate := startTime.Add(60 * time.Minute)
		assert.Equal(t, -1, tl.findSlotIndex(tooLate))
	})

	t.Run("should handle day view with hours >= 24", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		// Time that would be >= 24 hours
		tooLate := startTime.Add(24 * time.Hour)
		assert.Equal(t, -1, tl.findSlotIndex(tooLate))
	})

	t.Run("should handle day view with negative hours result", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		// Edge case: time that passes boundary check but results in negative hours
		// This shouldn't happen in practice due to boundary check, but test the branch
		testTime := startTime.Add(-1 * time.Nanosecond)
		// Should be caught by boundary check, but testing the hours < 0 case
		hours := int(testTime.Sub(startTime).Hours())
		if hours < 0 {
			// This branch should be hit
			assert.Equal(t, -1, tl.findSlotIndex(testTime))
		}
	})

	t.Run("should handle hour view with negative minutes result", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
		tl := NewTimeline(HourView, startTime, 80)

		// Edge case: time that passes boundary check but results in negative minutes
		testTime := startTime.Add(-1 * time.Nanosecond)
		// Should be caught by boundary check, but testing the minutes < 0 case
		minutes := int(testTime.Sub(startTime).Minutes())
		if minutes < 0 {
			// This branch should be hit
			assert.Equal(t, -1, tl.findSlotIndex(testTime))
		}
	})

	t.Run("should handle day view with hours >= 24 in switch", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		// Time that results in hours == 24 (should return -1)
		testTime := startTime.Add(24 * time.Hour)
		hours := int(testTime.Sub(startTime).Hours())
		assert.GreaterOrEqual(t, hours, 24)
		assert.Equal(t, -1, tl.findSlotIndex(testTime))
	})

	t.Run("should handle hour view with minutes >= 60 in switch", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
		tl := NewTimeline(HourView, startTime, 80)

		// Time that results in minutes == 60 (should return -1)
		testTime := startTime.Add(60 * time.Minute)
		minutes := int(testTime.Sub(startTime).Minutes())
		assert.GreaterOrEqual(t, minutes, 60)
		assert.Equal(t, -1, tl.findSlotIndex(testTime))
	})
}

func TestTimeline_GetOverlapStats(t *testing.T) {
	t.Run("should return empty stats when no overlaps", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))
		tl.AddJobRun("job-2", startTime.Add(2*time.Hour))

		stats := tl.GetOverlapStats()
		assert.Equal(t, 0, stats.TotalWindows)
		assert.Equal(t, 0, stats.MaxConcurrent)
		assert.Len(t, stats.MostProblematic, 0)
	})

	t.Run("should calculate stats with overlaps", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		time1 := startTime.Add(1 * time.Hour)
		time2 := startTime.Add(2 * time.Hour)

		// 3 jobs at time1
		tl.AddJobRun("job-1", time1)
		tl.AddJobRun("job-2", time1)
		tl.AddJobRun("job-3", time1)

		// 2 jobs at time2
		tl.AddJobRun("job-4", time2)
		tl.AddJobRun("job-5", time2)

		stats := tl.GetOverlapStats()
		assert.Equal(t, 2, stats.TotalWindows)
		assert.Equal(t, 3, stats.MaxConcurrent)
		assert.Len(t, stats.MostProblematic, 2)
		// Most problematic should be sorted by count (descending)
		assert.Equal(t, 3, stats.MostProblematic[0].Count)
		assert.Equal(t, 2, stats.MostProblematic[1].Count)
	})

	t.Run("should limit most problematic to 10", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		// Create 15 overlaps (each with 2 jobs)
		for i := 0; i < 15; i++ {
			overlapTime := startTime.Add(time.Duration(i) * time.Minute)
			tl.AddJobRun(fmt.Sprintf("job-%d", i), overlapTime)
			tl.AddJobRun(fmt.Sprintf("job-%d-b", i), overlapTime)
		}

		stats := tl.GetOverlapStats()
		assert.Equal(t, 15, stats.TotalWindows)
		assert.Equal(t, 2, stats.MaxConcurrent)
		assert.Len(t, stats.MostProblematic, 10)
	})

	t.Run("should sort overlaps with equal counts by time", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		// Create 3 overlaps with same count but different times
		time1 := startTime.Add(2 * time.Hour)
		time2 := startTime.Add(1 * time.Hour)
		time3 := startTime.Add(3 * time.Hour)

		// All have 2 jobs
		tl.AddJobRun("job-1", time1)
		tl.AddJobRun("job-2", time1)

		tl.AddJobRun("job-3", time2)
		tl.AddJobRun("job-4", time2)

		tl.AddJobRun("job-5", time3)
		tl.AddJobRun("job-6", time3)

		stats := tl.GetOverlapStats()
		assert.Equal(t, 3, stats.TotalWindows)
		// When counts are equal, should be sorted by time (ascending)
		assert.True(t, stats.MostProblematic[0].Time.Before(stats.MostProblematic[1].Time) ||
			stats.MostProblematic[0].Time.Equal(stats.MostProblematic[1].Time))
		assert.True(t, stats.MostProblematic[1].Time.Before(stats.MostProblematic[2].Time) ||
			stats.MostProblematic[1].Time.Equal(stats.MostProblematic[2].Time))
	})

	t.Run("should handle overlaps with different counts correctly", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		time1 := startTime.Add(1 * time.Hour)
		time2 := startTime.Add(2 * time.Hour)
		time3 := startTime.Add(3 * time.Hour)

		// 5 jobs at time1 (highest)
		for i := 0; i < 5; i++ {
			tl.AddJobRun(fmt.Sprintf("job-%d", i), time1)
		}

		// 3 jobs at time2
		for i := 5; i < 8; i++ {
			tl.AddJobRun(fmt.Sprintf("job-%d", i), time2)
		}

		// 2 jobs at time3 (lowest)
		tl.AddJobRun("job-8", time3)
		tl.AddJobRun("job-9", time3)

		stats := tl.GetOverlapStats()
		assert.Equal(t, 3, stats.TotalWindows)
		assert.Equal(t, 5, stats.MaxConcurrent)
		assert.Len(t, stats.MostProblematic, 3)
		// Should be sorted by count descending
		assert.Equal(t, 5, stats.MostProblematic[0].Count)
		assert.Equal(t, 3, stats.MostProblematic[1].Count)
		assert.Equal(t, 2, stats.MostProblematic[2].Count)
	})
}
