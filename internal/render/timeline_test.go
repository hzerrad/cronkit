package render

import (
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

		output := tl.Render()
		assert.Contains(t, output, "Timeline")
		assert.Contains(t, output, "00:00")
	})

	t.Run("should render timeline with job runs", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))
		tl.AddJobRun("job-1", startTime.Add(2*time.Hour))

		output := tl.Render()
		assert.Contains(t, output, "Timeline")
		assert.Contains(t, output, "job-1")
	})

	t.Run("should render hour view", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
		tl := NewTimeline(HourView, startTime, 80)

		tl.AddJobRun("job-1", startTime.Add(5*time.Minute))
		tl.AddJobRun("job-1", startTime.Add(10*time.Minute))

		output := tl.Render()
		assert.Contains(t, output, "Timeline")
		assert.Contains(t, output, "09:00")
		assert.Contains(t, output, "10:00")
	})

	t.Run("should handle narrow width", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 40)

		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))

		output := tl.Render()
		assert.NotEmpty(t, output)
		// Should still render, just with fewer slots
	})

	t.Run("should handle wide width", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 200)

		tl.AddJobRun("job-1", startTime.Add(1*time.Hour))

		output := tl.Render()
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
