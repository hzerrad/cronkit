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
	})

	t.Run("should create hour view timeline", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
		tl := NewTimeline(HourView, startTime, 80)

		assert.Equal(t, HourView, tl.view)
		assert.Equal(t, startTime, tl.startTime)
		assert.Equal(t, startTime.Add(time.Hour), tl.endTime)
		assert.Equal(t, 80, tl.width)
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

		tl.SetJobInfo("job-1", "0 9 * * *", "Daily at 9am", "morning-report")
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

		tl.SetJobInfo("job-1", "*/15 * * * *", "Every 15 minutes", "quarter-hour")

		idx, exists := tl.jobIndex["job-1"]
		require.True(t, exists)
		assert.Equal(t, "*/15 * * * *", tl.jobs[idx].expression)
		assert.Equal(t, "Every 15 minutes", tl.jobs[idx].description)
		assert.Equal(t, "quarter-hour", tl.jobs[idx].label)
	})

	t.Run("should keep the first job info on repeated calls", func(t *testing.T) {
		startTime := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, startTime, 80)

		tl.SetJobInfo("job-1", "*/15 * * * *", "Every 15 minutes", "quarter-hour")
		tl.SetJobInfo("job-1", "*/30 * * * *", "Every 30 minutes", "half-hour")

		idx, exists := tl.jobIndex["job-1"]
		require.True(t, exists)
		assert.Equal(t, "*/15 * * * *", tl.jobs[idx].expression)
		assert.Equal(t, "Every 15 minutes", tl.jobs[idx].description)
		assert.Len(t, tl.jobs, 1)
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
