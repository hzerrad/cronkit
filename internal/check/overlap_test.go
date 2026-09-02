package check

import (
	"testing"
	"time"

	"github.com/hzerrad/cronkit/internal/crontab"
	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeOverlaps(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("should detect overlaps for jobs running at same time", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true}, // Every hour at :00
			{LineNumber: 2, Expression: "0 * * * *", Valid: true}, // Every hour at :00
		}, "crontab")

		overlaps, stats, err := AnalyzeOverlaps(items, from, 24*time.Hour, scheduler, parser)
		require.NoError(t, err)
		assert.Greater(t, len(overlaps), 0, "Should detect overlaps")
		assert.Greater(t, stats.MaxConcurrent, 1, "Should have max concurrent > 1")
	})

	t.Run("should not detect overlaps for jobs at different times", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true},  // Every hour at :00
			{LineNumber: 2, Expression: "30 * * * *", Valid: true}, // Every hour at :30
		}, "crontab")

		overlaps, stats, err := AnalyzeOverlaps(items, from, 1*time.Hour, scheduler, parser)
		require.NoError(t, err)
		assert.Equal(t, 0, len(overlaps), "Should not detect overlaps for different times")
		assert.Equal(t, 0, stats.MaxConcurrent)
	})

	t.Run("should return empty for single job", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "0 * * * *", Valid: true},
		}, "crontab")

		overlaps, stats, err := AnalyzeOverlaps(items, from, 24*time.Hour, scheduler, parser)
		require.NoError(t, err)
		assert.Equal(t, 0, len(overlaps), "Single job cannot have overlaps")
		assert.Equal(t, 0, stats.MaxConcurrent)
	})

	t.Run("should return empty for empty job list", func(t *testing.T) {
		overlaps, stats, err := AnalyzeOverlaps(nil, from, 24*time.Hour, scheduler, parser)
		require.NoError(t, err)
		assert.Equal(t, 0, len(overlaps))
		assert.Equal(t, 0, stats.MaxConcurrent)
	})

	t.Run("should handle invalid jobs gracefully", func(t *testing.T) {
		items := inventory.FromCrontabJobs([]*crontab.Job{
			{LineNumber: 1, Expression: "invalid", Valid: false},
			{LineNumber: 2, Expression: "0 * * * *", Valid: true},
		}, "crontab")

		overlaps, _, err := AnalyzeOverlaps(items, from, 24*time.Hour, scheduler, parser)
		require.NoError(t, err)
		// Should only analyze valid jobs
		assert.GreaterOrEqual(t, len(overlaps), 0)
	})

	t.Run("should exclude suspended items", func(t *testing.T) {
		// A suspended Kubernetes CronJob does not run, so it cannot collide with anything.
		items := []inventory.Item{
			{Expression: "0 * * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
			{Expression: "0 * * * *", State: inventory.StateSuspended, Locator: inventory.Locator{Line: 2}},
		}

		overlaps, stats, err := AnalyzeOverlaps(items, from, 24*time.Hour, scheduler, parser)
		require.NoError(t, err)
		assert.Equal(t, 0, len(overlaps), "one active item alone cannot overlap")
		assert.Equal(t, 0, stats.MaxConcurrent)
	})

	t.Run("an item with an unresolvable timezone is reported, not silently dropped", func(t *testing.T) {
		// Runs items through the same admission step (inventory.ResolveTimezones) a real caller does.
		items := inventory.ResolveTimezones([]inventory.Item{
			{Expression: "0 * * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
			{Expression: "0 * * * *", Timezone: "Not/AZone", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
		})
		require.Equal(t, inventory.StateInvalid, items[1].State, "sanity check: admission must have marked the bad-zone item")
		require.Contains(t, items[1].Reason, "Not/AZone")

		overlaps, stats, err := AnalyzeOverlaps(items, from, 24*time.Hour, scheduler, parser)
		require.NoError(t, err)
		assert.Equal(t, 0, len(overlaps), "the unanalysable item cannot be shown to collide with anything")
		assert.Equal(t, 0, stats.MaxConcurrent)
	})
}

// TestAnalyzeOverlaps_CrossZoneCollision confirms items at the same instant in different zones collide.
func TestAnalyzeOverlaps_CrossZoneCollision(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()
	from := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)

	items := []inventory.Item{
		{
			Expression: "0 12 * * *",
			Timezone:   "Europe/London",
			State:      inventory.StateActive,
			Locator:    inventory.Locator{Line: 1},
		},
		{
			Expression: "0 13 * * *",
			Timezone:   "Europe/Paris",
			State:      inventory.StateActive,
			Locator:    inventory.Locator{Line: 2},
		},
	}

	overlaps, stats, err := AnalyzeOverlaps(items, from, 24*time.Hour, scheduler, parser)
	require.NoError(t, err)
	require.Len(t, overlaps, 1, "the two schedules land on the same instant exactly once in the window")
	assert.Equal(t, 2, overlaps[0].Count)
	assert.ElementsMatch(t, []string{"line-1", "line-2"}, overlaps[0].JobIDs)
	assert.Equal(t, 2, stats.MaxConcurrent)
}

// TestAnalyzeOverlaps_SameWallClockDifferentZonesDoesNotCollide: different zones, no collision.
func TestAnalyzeOverlaps_SameWallClockDifferentZonesDoesNotCollide(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()
	from := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)

	items := []inventory.Item{
		{Expression: "0 12 * * *", Timezone: "Europe/London", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
		{Expression: "0 12 * * *", Timezone: "Europe/Paris", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
	}

	overlaps, stats, err := AnalyzeOverlaps(items, from, 24*time.Hour, scheduler, parser)
	require.NoError(t, err)
	assert.Empty(t, overlaps, "the same wall-clock hour in different zones is a different instant")
	assert.Equal(t, 0, stats.MaxConcurrent)
}

// TestAnalyzeOverlaps_DaylightSavingTransition confirms each occurrence is evaluated on its own date.
func TestAnalyzeOverlaps_DaylightSavingTransition(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()

	items := []inventory.Item{
		{Expression: "0 9 * * *", Timezone: "America/New_York", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
		{Expression: "0 14 * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
	}

	t.Run("before the transition, 09:00 EST is 14:00 UTC and the items collide", func(t *testing.T) {
		from := time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC)
		overlaps, stats, err := AnalyzeOverlaps(items, from, 24*time.Hour, scheduler, parser)
		require.NoError(t, err)
		require.Len(t, overlaps, 1)
		assert.Equal(t, 2, stats.MaxConcurrent)
	})

	t.Run("after the transition, 09:00 EDT is 13:00 UTC and the items no longer collide", func(t *testing.T) {
		from := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
		overlaps, stats, err := AnalyzeOverlaps(items, from, 24*time.Hour, scheduler, parser)
		require.NoError(t, err)
		assert.Empty(t, overlaps)
		assert.Equal(t, 0, stats.MaxConcurrent)
	})
}

// TestAnalyzeOverlaps_HalfHourOffset pins down that a half-hour UTC offset is honored exactly.
func TestAnalyzeOverlaps_HalfHourOffset(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()
	from := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)

	items := []inventory.Item{
		{Expression: "0 12 * * *", Timezone: "Asia/Kolkata", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
		{Expression: "30 6 * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}}, // exact match: 06:30 UTC
		{Expression: "0 6 * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 3}},  // rounded down: 06:00 UTC
		{Expression: "0 7 * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 4}},  // rounded up: 07:00 UTC
	}

	overlaps, stats, err := AnalyzeOverlaps(items, from, 24*time.Hour, scheduler, parser)
	require.NoError(t, err)
	require.Len(t, overlaps, 1, "only the exact 06:30 UTC match collides with the Kolkata item")
	assert.Equal(t, 2, overlaps[0].Count)
	assert.ElementsMatch(t, []string{"line-1", "line-2"}, overlaps[0].JobIDs)
	assert.Equal(t, 2, stats.MaxConcurrent)
}

// TestAnalyzeOverlaps_EmptyTimezoneUsesInvocationDefault: no Timezone falls back to from's location.
func TestAnalyzeOverlaps_EmptyTimezoneUsesInvocationDefault(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()

	paris, err := time.LoadLocation("Europe/Paris")
	require.NoError(t, err)
	from := time.Date(2026, 1, 5, 0, 0, 0, 0, paris)

	items := []inventory.Item{
		// No Timezone: "0 12 * * *" must be read as 12:00 Paris, not 12:00 UTC.
		{Expression: "0 12 * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
		{Expression: "0 12 * * *", Timezone: "Europe/Paris", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
	}

	overlaps, stats, err := AnalyzeOverlaps(items, from, 24*time.Hour, scheduler, parser)
	require.NoError(t, err)
	require.Len(t, overlaps, 1)
	assert.Equal(t, 2, overlaps[0].Count)
	assert.Equal(t, 2, stats.MaxConcurrent)
}

func TestAnalyzeOverlaps_UsesInjectedOrigin(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	items := inventory.FromCrontabJobs([]*crontab.Job{
		{Expression: "0 * * * *", Command: "/usr/bin/a", Valid: true, LineNumber: 1},
		{Expression: "0 * * * *", Command: "/usr/bin/b", Valid: true, LineNumber: 2},
	}, "crontab")

	overlaps, stats, err := AnalyzeOverlaps(items, from, 3*time.Hour, scheduler, parser)

	require.NoError(t, err)
	// scheduler.Next returns occurrences strictly after from, excluding the origin run itself.
	require.Len(t, overlaps, 2, "one overlap per hour after the origin in a three hour window")
	assert.Equal(t, 2, stats.MaxConcurrent)
	assert.Equal(t, time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC), overlaps[0].Time)
}

func TestAnalyzeOverlaps_IdenticalSchedulesWithoutLineNumbers(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	items := inventory.FromCrontabJobs([]*crontab.Job{
		{Expression: "0 * * * *", Command: "/usr/bin/a", Valid: true},
		{Expression: "0 * * * *", Command: "/usr/bin/b", Valid: true},
	}, "crontab")

	overlaps, stats, err := AnalyzeOverlaps(items, from, 2*time.Hour, scheduler, parser)

	require.NoError(t, err)
	require.Len(t, overlaps, 1, "two jobs on the same schedule overlap")
	assert.Equal(t, 2, overlaps[0].Count)
	assert.Equal(t, 2, stats.MaxConcurrent)
}

// TestAnalyzeOverlaps_ForbidConcurrencySuppression pins CRON-012: all-Forbid overlaps are suppressed.
// Mixing in a non-Forbid item still reports the overlap.
func TestAnalyzeOverlaps_ForbidConcurrencySuppression(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("two Forbid items overlapping is not a finding", func(t *testing.T) {
		items := []inventory.Item{
			{Expression: "0 * * * *", State: inventory.StateActive, Concurrency: inventory.ConcurrencyForbid, Locator: inventory.Locator{Line: 1}},
			{Expression: "0 * * * *", State: inventory.StateActive, Concurrency: inventory.ConcurrencyForbid, Locator: inventory.Locator{Line: 2}},
		}

		overlaps, stats, err := AnalyzeOverlaps(items, from, 2*time.Hour, scheduler, parser)
		require.NoError(t, err)
		assert.Empty(t, overlaps, "both jobs are individually serialised by the platform")
		assert.Equal(t, 0, stats.MaxConcurrent)
	})

	t.Run("a Forbid item overlapping an Allow item is still a finding", func(t *testing.T) {
		items := []inventory.Item{
			{Expression: "0 * * * *", State: inventory.StateActive, Concurrency: inventory.ConcurrencyForbid, Locator: inventory.Locator{Line: 1}},
			{Expression: "0 * * * *", State: inventory.StateActive, Concurrency: inventory.ConcurrencyAllow, Locator: inventory.Locator{Line: 2}},
		}

		overlaps, stats, err := AnalyzeOverlaps(items, from, 2*time.Hour, scheduler, parser)
		require.NoError(t, err)
		require.Len(t, overlaps, 1, "the Allow item genuinely can run at the same time as the Forbid item")
		assert.Equal(t, 2, overlaps[0].Count)
		assert.Equal(t, 2, stats.MaxConcurrent)
	})

	t.Run("ConcurrencyUnspecified -- every crontab item's default -- still contends", func(t *testing.T) {
		items := []inventory.Item{
			{Expression: "0 * * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 1}},
			{Expression: "0 * * * *", State: inventory.StateActive, Locator: inventory.Locator{Line: 2}},
		}

		overlaps, stats, err := AnalyzeOverlaps(items, from, 2*time.Hour, scheduler, parser)
		require.NoError(t, err)
		require.Len(t, overlaps, 1)
		assert.Equal(t, 2, overlaps[0].Count)
		assert.Equal(t, 2, stats.MaxConcurrent)
	})

	t.Run("Replace still contends", func(t *testing.T) {
		items := []inventory.Item{
			{Expression: "0 * * * *", State: inventory.StateActive, Concurrency: inventory.ConcurrencyForbid, Locator: inventory.Locator{Line: 1}},
			{Expression: "0 * * * *", State: inventory.StateActive, Concurrency: inventory.ConcurrencyReplace, Locator: inventory.Locator{Line: 2}},
		}

		overlaps, stats, err := AnalyzeOverlaps(items, from, 2*time.Hour, scheduler, parser)
		require.NoError(t, err)
		require.Len(t, overlaps, 1)
		assert.Equal(t, 2, stats.MaxConcurrent)
	})

	t.Run("three-way overlap: two Forbid plus one Allow is reported whole", func(t *testing.T) {
		items := []inventory.Item{
			{Expression: "0 * * * *", State: inventory.StateActive, Concurrency: inventory.ConcurrencyForbid, Locator: inventory.Locator{Line: 1}},
			{Expression: "0 * * * *", State: inventory.StateActive, Concurrency: inventory.ConcurrencyForbid, Locator: inventory.Locator{Line: 2}},
			{Expression: "0 * * * *", State: inventory.StateActive, Concurrency: inventory.ConcurrencyAllow, Locator: inventory.Locator{Line: 3}},
		}

		overlaps, stats, err := AnalyzeOverlaps(items, from, 2*time.Hour, scheduler, parser)
		require.NoError(t, err)
		require.Len(t, overlaps, 1)
		assert.Equal(t, 3, overlaps[0].Count, "the group is reported whole, not with the Forbid members stripped out")
		assert.Equal(t, 3, stats.MaxConcurrent)
	})
}

func TestAllForbid(t *testing.T) {
	concurrency := map[string]inventory.Concurrency{
		"a": inventory.ConcurrencyForbid,
		"b": inventory.ConcurrencyForbid,
		"c": inventory.ConcurrencyAllow,
	}

	assert.True(t, allForbid([]string{"a", "b"}, concurrency))
	assert.False(t, allForbid([]string{"a", "c"}, concurrency))
	assert.False(t, allForbid([]string{"a", "missing"}, concurrency), "a job missing from the map is treated as not Forbid")
}

func TestUniqueStrings(t *testing.T) {
	t.Run("should remove duplicates", func(t *testing.T) {
		input := []string{"a", "b", "a", "c", "b"}
		result := uniqueStrings(input)
		assert.Equal(t, 3, len(result))
		assert.Contains(t, result, "a")
		assert.Contains(t, result, "b")
		assert.Contains(t, result, "c")
	})

	t.Run("should handle empty slice", func(t *testing.T) {
		result := uniqueStrings([]string{})
		assert.Equal(t, 0, len(result))
	})

	t.Run("should handle single element", func(t *testing.T) {
		result := uniqueStrings([]string{"a"})
		assert.Equal(t, 1, len(result))
		assert.Equal(t, "a", result[0])
	})
}

// TestAnalyzeOverlaps_SameLineDifferentFiles confirms job identity comes from file+line, not line alone.
func TestAnalyzeOverlaps_SameLineDifferentFiles(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	items := []inventory.Item{
		{Expression: "0 * * * *", State: inventory.StateActive, Locator: inventory.Locator{File: "site-a/crontab", Line: 6}},
		{Expression: "0 * * * *", State: inventory.StateActive, Locator: inventory.Locator{File: "site-b/crontab", Line: 6}},
	}

	overlaps, stats, err := AnalyzeOverlaps(items, from, 2*time.Hour, scheduler, parser)
	require.NoError(t, err)
	require.Len(t, overlaps, 1, "two distinct items on the same line in different files must still be reported as colliding")
	assert.Equal(t, 2, overlaps[0].Count)
	assert.Equal(t, 2, stats.MaxConcurrent)
}

// TestAnalyzeOverlaps_JobIDsAreSorted pins the order an overlap's jobs reach the CRON-012 message in.
func TestAnalyzeOverlaps_JobIDsAreSorted(t *testing.T) {
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()
	from := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	// Deliberately supplied in reverse order of the ids they resolve to.
	items := []inventory.Item{
		{Expression: "0 12 * * *", State: inventory.StateActive, Locator: inventory.Locator{File: "site-b/crontab", Line: 7}},
		{Expression: "0 12 * * *", State: inventory.StateActive, Locator: inventory.Locator{File: "site-a/crontab", Line: 2}},
	}

	overlaps, _, err := AnalyzeOverlaps(items, from, 24*time.Hour, scheduler, parser)
	require.NoError(t, err)
	require.NotEmpty(t, overlaps)

	assert.Equal(t, []string{"line-site-a/crontab:2", "line-site-b/crontab:7"}, overlaps[0].JobIDs)
}
