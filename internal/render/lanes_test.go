package render

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPlanWidths(t *testing.T) {
	t.Run("should give full layout at 80 cols", func(t *testing.T) {
		b := planWidths(80, 13, 16, 96)
		assert.Equal(t, budget{gutter: 13, plot: 48, expr: 16}, b)
		assert.Equal(t, 80, b.gutter+1+b.plot+1+1+b.expr)
	})
	t.Run("should drop expr column below 80", func(t *testing.T) {
		b := planWidths(79, 13, 16, 96)
		assert.Equal(t, budget{gutter: 13, plot: 64, expr: 0}, b)
	})
	t.Run("should cap gutter at 10 below 60", func(t *testing.T) {
		b := planWidths(44, 13, 16, 96)
		assert.Equal(t, budget{gutter: 10, plot: 32, expr: 0}, b)
	})
	t.Run("should clamp tiny labels up to 8", func(t *testing.T) {
		assert.Equal(t, 8, planWidths(80, 3, 0, 96).gutter)
	})
	t.Run("should cap gutter at 17 under 100 cols", func(t *testing.T) {
		assert.Equal(t, 17, planWidths(90, 40, 0, 96).gutter)
	})
	t.Run("should let the gutter reach 32 once there is room", func(t *testing.T) {
		assert.Equal(t, 24, planWidths(160, 24, 0, 96).gutter)
		assert.Equal(t, 32, planWidths(160, 40, 0, 96).gutter)
	})
	t.Run("should hold the plot to the view's useful resolution", func(t *testing.T) {
		assert.Equal(t, 96, planWidths(200, 18, 9, 96).plot)
		assert.Equal(t, 60, planWidths(200, 18, 9, 60).plot)
	})
	t.Run("should drop expr when plot would fall under 30", func(t *testing.T) {
		b := planWidths(80, 17, 17, 96)
		assert.Equal(t, budget{gutter: 17, plot: 43, expr: 17}, b)
		for total := 80; total <= 140; total++ {
			b := planWidths(total, 17, 17, 96)
			if b.expr > 0 {
				assert.GreaterOrEqual(t, b.plot, 30)
			}
		}
	})
}

func TestCellPos(t *testing.T) {
	start := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	t.Run("should map start to col 0", func(t *testing.T) {
		assert.Equal(t, 0, cellPos(start, start, day, 48))
	})
	t.Run("should map noon to the middle", func(t *testing.T) {
		assert.Equal(t, 24, cellPos(start.Add(12*time.Hour), start, day, 48))
	})
	t.Run("should clamp the last minute into the final cell", func(t *testing.T) {
		assert.Equal(t, 47, cellPos(start.Add(day-time.Minute), start, day, 48))
	})
	t.Run("should clamp a time before start to col 0", func(t *testing.T) {
		assert.Equal(t, 0, cellPos(start.Add(-time.Hour), start, day, 48))
	})
}

func TestZoneLabel(t *testing.T) {
	t.Run("should return UTC bare", func(t *testing.T) {
		assert.Equal(t, "UTC", zoneLabel(time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)))
	})
	t.Run("should name a loaded zone with its UTC offset", func(t *testing.T) {
		loc, err := time.LoadLocation("America/New_York")
		if err != nil {
			t.Skip("America/New_York not available")
		}
		got := zoneLabel(time.Date(2026, 8, 6, 12, 0, 0, 0, loc))
		assert.Equal(t, "America/New_York (UTC-04:00)", got)
	})
	t.Run("should fall back to the abbreviation for an unnamed local", func(t *testing.T) {
		loc := time.FixedZone("", -5*3600)
		got := zoneLabel(time.Date(2026, 8, 6, 12, 0, 0, 0, loc))
		assert.Equal(t, "-0500 (UTC-05:00)", got)
	})
}

func TestJobLabel(t *testing.T) {
	tl := laneFixture()
	t.Run("should look up the stored label", func(t *testing.T) {
		assert.Equal(t, "backup.sh", tl.jobLabel("job-1"))
	})
	t.Run("should return the id when unknown", func(t *testing.T) {
		assert.Equal(t, "job-missing", tl.jobLabel("job-missing"))
	})
}

func TestAxisTicks(t *testing.T) {
	start := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)
	t.Run("should place day ticks every 6h plus end label", func(t *testing.T) {
		ticks := axisTicks(DayView, start, 48)
		assert.Equal(t, []tick{
			{0, "00:00"}, {12, "06:00"}, {24, "12:00"}, {36, "18:00"}, {47, "23:59"},
		}, ticks)
	})
	t.Run("should place hour ticks every 15m from the real start hour", func(t *testing.T) {
		hs := time.Date(2026, 8, 6, 9, 0, 0, 0, time.UTC)
		ticks := axisTicks(HourView, hs, 60)
		assert.Equal(t, []tick{
			{0, "09:00"}, {15, "09:15"}, {30, "09:30"}, {45, "09:45"}, {59, "09:59"},
		}, ticks)
	})
}

func TestPadLabel(t *testing.T) {
	t.Run("should pad short labels", func(t *testing.T) {
		assert.Equal(t, "backup.sh ", padLabel("backup.sh", 10, '…', false))
	})
	t.Run("should truncate long labels with ellipsis", func(t *testing.T) {
		assert.Equal(t, "At 12:00 …", padLabel("At 12:00 every day", 10, '…', false))
		assert.Equal(t, 10, len([]rune(padLabel("At 12:00 every day", 10, '…', false))))
	})
	t.Run("should truncate a path-like label from the left", func(t *testing.T) {
		got := padLabel("testdata/sources/services/api/cronjob.yaml", 17, '…', true)
		assert.Equal(t, 17, len([]rune(got)))
		assert.True(t, strings.HasPrefix(got, "..."))
		assert.True(t, strings.HasSuffix(got, "cronjob.yaml"), got)
	})
}

func TestColorize(t *testing.T) {
	t.Run("should wrap with reset", func(t *testing.T) {
		assert.Equal(t, "\x1b[31mx\x1b[0m", colorize("x", ansiRed))
	})
	t.Run("should be a no-op without a code", func(t *testing.T) {
		assert.Equal(t, "x", colorize("x", ""))
	})
}

func TestJobRegistryOrder(t *testing.T) {
	t.Run("should keep insertion order", func(t *testing.T) {
		tl := NewTimeline(DayView, time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC), 80)
		tl.SetJobInfo("job-9", "0 2 * * *", "At 02:00 every day", "backup.sh")
		tl.SetJobInfo("job-2", "0 * * * *", "At the start of every hour", "report")
		tl.SetJobInfo("job-9", "0 2 * * *", "At 02:00 every day", "backup.sh")
		assert.Equal(t, []string{"job-9", "job-2"}, []string{tl.jobs[0].id, tl.jobs[1].id})
		assert.Len(t, tl.jobs, 2)
	})
}

func laneFixture() *Timeline {
	start := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)
	tl := NewTimeline(DayView, start, 80)
	tl.SetJobInfo("job-1", "0 2 * * *", "At 02:00 every day", "backup.sh")
	tl.SetJobInfo("job-2", "0 2 * * *", "At 02:00 every day", "verify.sh")
	tl.AddJobRun("job-1", start.Add(2*time.Hour))
	tl.AddJobRun("job-2", start.Add(2*time.Hour))
	return tl
}

// laneFixtureLongExpr adds a third job whose expression exceeds the 17-rune expr budget.
func laneFixtureLongExpr() *Timeline {
	tl := laneFixture()
	tl.SetJobInfo("job-3", "0,15,30,45 2-6 * * 1-5", "every 15m weekday mornings", "cron3")
	tl.AddJobRun("job-3", tl.startTime.Add(2*time.Hour))
	return tl
}

// laneFixtureShortLabels sets up two jobs with 4-rune labels that overlap, so the
// conflicts row renders against a gutter that would otherwise floor below "conflicts".
func laneFixtureShortLabels() *Timeline {
	start := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)
	tl := NewTimeline(DayView, start, 80)
	tl.SetJobInfo("job-1", "0 2 * * *", "At 02:00 every day", "a.sh")
	tl.SetJobInfo("job-2", "0 2 * * *", "At 02:00 every day", "b.sh")
	tl.AddJobRun("job-1", start.Add(2*time.Hour))
	tl.AddJobRun("job-2", start.Add(2*time.Hour))
	return tl
}

func TestRenderLanes(t *testing.T) {
	t.Run("should render one lane per job in order", func(t *testing.T) {
		out := laneFixture().Render(RenderOptions{})
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		assert.True(t, strings.HasPrefix(lines[0], "2026-08-06  00:00 → 23:59 · UTC"))
		assert.True(t, strings.HasPrefix(lines[2], "backup.sh"))
		assert.True(t, strings.HasPrefix(lines[3], "verify.sh"))
	})
	t.Run("should keep every body line within total width", func(t *testing.T) {
		out := laneFixtureLongExpr().Render(RenderOptions{})
		for _, ln := range strings.Split(out, "\n") {
			assert.LessOrEqual(t, len([]rune(ln)), 80, ln)
		}
	})
	t.Run("should print the source above the window line", func(t *testing.T) {
		tl := laneFixture()
		tl.SetSource("/etc/crontab", "")
		lines := strings.Split(tl.Render(RenderOptions{}), "\n")
		assert.Equal(t, "/etc/crontab", lines[0])
		assert.Equal(t, "2026-08-06  00:00 → 23:59 · UTC", lines[1])
		assert.Empty(t, lines[2])
	})
	t.Run("should name the window range and zone without a source", func(t *testing.T) {
		lines := strings.Split(laneFixture().Render(RenderOptions{}), "\n")
		assert.Equal(t, "2026-08-06  00:00 → 23:59 · UTC", lines[0])
	})
	t.Run("should not trail a lane with an expression its label already carries", func(t *testing.T) {
		start := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, start, 80)
		tl.SetJobInfo("job-1", "0 2 * * *", "At 02:00 every day", "0 2 * * *")
		tl.AddJobRun("job-1", start.Add(2*time.Hour))

		lane := strings.Split(tl.Render(RenderOptions{}), "\n")[2]
		assert.Equal(t, 1, strings.Count(lane, "0 2 * * *"))
	})
	t.Run("should stop stretching the plot on a very wide terminal", func(t *testing.T) {
		start := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)
		tl := NewTimeline(DayView, start, 200)
		tl.SetJobInfo("job-1", "0 2 * * *", "At 02:00 every day", "At 02:00 every day")
		tl.AddJobRun("job-1", start.Add(2*time.Hour))

		lines := strings.Split(tl.Render(RenderOptions{}), "\n")
		assert.True(t, strings.HasPrefix(lines[2], "At 02:00 every day┤"), lines[2])
		assert.Equal(t, 96, strings.Count(lines[3], "─")+strings.Count(lines[3], "┬"))
	})
	t.Run("should truncate an expression that overflows the expr budget", func(t *testing.T) {
		out := laneFixtureLongExpr().Render(RenderOptions{})
		var row string
		for _, ln := range strings.Split(out, "\n") {
			if strings.HasPrefix(ln, "cron3") {
				row = ln
			}
		}
		r := []rune(row)
		assert.NotEmpty(t, r)
		assert.Equal(t, '…', r[len(r)-1])
	})
	t.Run("should put both 02:00 markers in the same column", func(t *testing.T) {
		out := laneFixture().Render(RenderOptions{})
		lines := strings.Split(out, "\n")
		c1 := strings.IndexRune(lines[2], '╷')
		c2 := strings.IndexRune(lines[3], '╷')
		assert.Equal(t, c1, c2)
		assert.Positive(t, c1)
	})
	t.Run("should flag the conflict column", func(t *testing.T) {
		out := laneFixture().Render(RenderOptions{})
		assert.Contains(t, out, "conflicts")
		assert.Contains(t, out, "▲")
	})
	t.Run("should keep the conflicts label whole even with short job labels", func(t *testing.T) {
		out := laneFixtureShortLabels().Render(RenderOptions{})
		var row string
		for _, ln := range strings.Split(out, "\n") {
			if strings.HasPrefix(ln, "conflic") {
				row = ln
			}
		}
		assert.True(t, strings.HasPrefix(row, "conflicts"), row)
	})
	t.Run("should merge same-job same-cell runs into a heavy mark", func(t *testing.T) {
		tl := laneFixture()
		tl.AddJobRun("job-1", tl.startTime.Add(2*time.Hour+time.Minute))
		assert.Contains(t, tl.Render(RenderOptions{}), "┃")
	})
	t.Run("should say when the window is empty", func(t *testing.T) {
		tl := NewTimeline(DayView, time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC), 80)
		tl.SetJobInfo("job-1", "0 2 * * 0", "At 02:00 on Sunday", "weekly.sh")
		assert.Contains(t, tl.Render(RenderOptions{}), "no runs in this window")
	})
	t.Run("should render an empty lane for a job with no runs", func(t *testing.T) {
		tl := laneFixture()
		tl.SetJobInfo("job-3", "0 2 * * 0", "At 02:00 on Sunday", "weekly.sh")
		assert.Contains(t, tl.Render(RenderOptions{}), "weekly.sh")
	})
	t.Run("should show the footer stats", func(t *testing.T) {
		out := laneFixture().Render(RenderOptions{})
		assert.Contains(t, out, "2 jobs · 2 runs · 1 conflict window · max 2 concurrent")
	})
	t.Run("should list overlap windows when asked", func(t *testing.T) {
		out := laneFixture().Render(RenderOptions{ShowOverlaps: true})
		assert.Contains(t, out, "overlaps:")
		assert.Contains(t, out, "02:00  backup.sh, verify.sh")
	})
	t.Run("should match the frozen 80-col day view", func(t *testing.T) {
		want := `2026-08-06  00:00 → 23:59 · UTC

backup.sh┤    ╷                                                      ├ 0 2 * * *
verify.sh┤    ╷                                                      ├ 0 2 * * *
conflicts┤    ▲                                                      ├
         └┬─────────────┬──────────────┬──────────────┬─────────────┬┘
         00:00        06:00          12:00          18:00        23:59

2 jobs · 2 runs · 1 conflict window · max 2 concurrent
`
		assert.Equal(t, want, laneFixture().Render(RenderOptions{}))
	})
	t.Run("should match the frozen 44-col hour view", func(t *testing.T) {
		want := `2026-08-06  09:00 → 09:59 · UTC

backup.sh┤           ╷                     ├
verify.sh┤           ╷                     ├
conflicts┤           ▲                     ├
         └┬───────┬───────┬───────┬───────┬┘
         09:00  09:15   09:30   09:45  09:59

2 jobs · 2 runs · 1 conflict window · max 2 concurrent
`
		assert.Equal(t, want, laneFixtureHour().Render(RenderOptions{}))
	})
}

func laneFixtureHour() *Timeline {
	start := time.Date(2026, 8, 6, 9, 0, 0, 0, time.UTC)
	tl := NewTimeline(HourView, start, 44)
	tl.SetJobInfo("job-1", "0 2 * * *", "At 02:00 every day", "backup.sh")
	tl.SetJobInfo("job-2", "0 2 * * *", "At 02:00 every day", "verify.sh")
	tl.AddJobRun("job-1", start.Add(20*time.Minute))
	tl.AddJobRun("job-2", start.Add(20*time.Minute))
	return tl
}

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestRenderModes(t *testing.T) {
	t.Run("should emit identical text with color stripped", func(t *testing.T) {
		plain := laneFixture().Render(RenderOptions{})
		colored := laneFixture().Render(RenderOptions{Color: true})
		assert.NotEqual(t, plain, colored)
		assert.Equal(t, plain, ansiRE.ReplaceAllString(colored, ""))
	})
	t.Run("should color conflicts red", func(t *testing.T) {
		out := laneFixture().Render(RenderOptions{Color: true})
		assert.Contains(t, out, ansiRed)
	})
	t.Run("should stay pure ascii in ascii mode", func(t *testing.T) {
		out := laneFixture().Render(RenderOptions{ASCII: true})
		for _, r := range out {
			assert.Less(t, r, rune(128), "non-ascii rune %q", r)
		}
	})
	t.Run("should keep ascii marks aligned with unicode marks", func(t *testing.T) {
		uni := laneFixture().Render(RenderOptions{})
		asc := laneFixture().Render(RenderOptions{ASCII: true})
		uniLines, ascLines := strings.Split(uni, "\n"), strings.Split(asc, "\n")
		uniCol := runeIndex(uniLines[2], '╷')
		assert.Positive(t, uniCol)
		assert.Equal(t, '|', []rune(ascLines[2])[uniCol]) // same cell, different glyph
	})
}

// runeIndex finds target's rune (not byte) offset in s, or -1 if absent.
func runeIndex(s string, target rune) int {
	for i, r := range []rune(s) {
		if r == target {
			return i
		}
	}
	return -1
}

func TestDistinctFileCount(t *testing.T) {
	t.Run("should count zero for jobs with no file", func(t *testing.T) {
		assert.Equal(t, 0, distinctFileCount([]jobEntry{{file: ""}, {file: ""}}))
	})
	t.Run("should count one shared file once", func(t *testing.T) {
		assert.Equal(t, 1, distinctFileCount([]jobEntry{{file: "a/crontab"}, {file: "a/crontab"}}))
	})
	t.Run("should count distinct files", func(t *testing.T) {
		assert.Equal(t, 2, distinctFileCount([]jobEntry{{file: "a/crontab"}, {file: "b/crontab"}}))
	})
}

func TestLaneGutter(t *testing.T) {
	t.Run("should fall back to laneExpr when not multi-file and not aggregated", func(t *testing.T) {
		j := jobEntry{expression: "0 2 * * *", label: "backup.sh"}
		assert.Equal(t, "0 2 * * *", laneGutter(j, false))
	})
	t.Run("should show file:line once the chart spans multiple files", func(t *testing.T) {
		j := jobEntry{expression: "0 2 * * *", label: "backup.sh", file: "site-a/crontab", line: 3}
		assert.Equal(t, "site-a/crontab:3", laneGutter(j, true))
	})
	t.Run("should show the bare file when there is no line", func(t *testing.T) {
		j := jobEntry{expression: "0 2 * * *", label: "backup.sh", file: "site-a/crontab"}
		assert.Equal(t, "site-a/crontab", laneGutter(j, true))
	})
	t.Run("should still show the expression when multi-file but this lane has no file", func(t *testing.T) {
		j := jobEntry{expression: "0 2 * * *", label: "backup.sh"}
		assert.Equal(t, "0 2 * * *", laneGutter(j, true))
	})
	t.Run("should show the job count for an aggregate lane regardless of file state", func(t *testing.T) {
		j := jobEntry{expression: "0 2 * * *", label: "site-a/crontab", file: "site-a/crontab", count: 12}
		assert.Equal(t, "12 jobs", laneGutter(j, true))
		assert.Equal(t, "12 jobs", laneGutter(j, false))
	})
	t.Run("should singularize a one-job aggregate", func(t *testing.T) {
		j := jobEntry{count: 1}
		assert.Equal(t, "1 job", laneGutter(j, true))
	})
}

func TestGutterIsPath(t *testing.T) {
	t.Run("false for an ordinary expression lane", func(t *testing.T) {
		assert.False(t, gutterIsPath(jobEntry{expression: "0 2 * * *"}, false))
	})
	t.Run("true once multi-file and this lane has a file", func(t *testing.T) {
		assert.True(t, gutterIsPath(jobEntry{file: "site-a/crontab"}, true))
	})
	t.Run("false when multi-file but this lane has no file", func(t *testing.T) {
		assert.False(t, gutterIsPath(jobEntry{}, true))
	})
	t.Run("false for an aggregate lane even with a file", func(t *testing.T) {
		assert.False(t, gutterIsPath(jobEntry{file: "site-a/crontab", count: 5}, true))
	})
}

func TestTruncPathLeft(t *testing.T) {
	t.Run("returns s unchanged when it already fits", func(t *testing.T) {
		assert.Equal(t, "crontab:3", truncPathLeft("crontab:3", 17))
	})
	t.Run("keeps the tail and prefixes ASCII dots, not a unicode ellipsis", func(t *testing.T) {
		got := truncPathLeft("services/api/deploy/backup.yaml:3", 17)
		assert.True(t, strings.HasPrefix(got, "..."), got)
		assert.True(t, strings.HasSuffix(got, "backup.yaml:3"), got)
		assert.LessOrEqual(t, len([]rune(got)), 17)
		assert.NotContains(t, got, "…")
	})
	t.Run("keeps two long paths sharing a prefix distinguishable", func(t *testing.T) {
		a := truncPathLeft("services/api/deploy/backup.yaml:3", 17)
		b := truncPathLeft("services/api/deploy/restore.yaml:3", 17)
		assert.NotEqual(t, a, b)
	})
	t.Run("trims a leading separator so the cut doesn't double up", func(t *testing.T) {
		// len(r) - (w-3) lands exactly on a "/" here; without the trim the
		// result would read "..." + "/restore.yaml:9", a doubled marker.
		got := truncPathLeft("a/very/long/path/to/restore.yaml:9", 18)
		assert.Equal(t, "...restore.yaml:9", got)
		assert.False(t, strings.HasPrefix(got, "...//"), got)
	})
	t.Run("falls back to a bare tail when the budget is too small for the marker", func(t *testing.T) {
		got := truncPathLeft("backup.yaml:3", 3)
		assert.Equal(t, ":3", got[len(got)-2:])
		assert.Equal(t, 3, len([]rune(got)))
	})
}

func TestWindowLineForeignZones(t *testing.T) {
	t.Run("should say nothing extra when no foreign zones were declared", func(t *testing.T) {
		tl := laneFixture()
		assert.NotContains(t, tl.windowLine(uniGlyphs), "converted from")
	})
	t.Run("should name the declared zones converted onto the axis", func(t *testing.T) {
		tl := laneFixture()
		tl.SetForeignZones([]string{"Asia/Tokyo", "Europe/London"})
		line := tl.windowLine(uniGlyphs)
		assert.Contains(t, line, "converted from Asia/Tokyo, Europe/London")
	})
}

func TestFooterLineNotes(t *testing.T) {
	t.Run("should stay unchanged with nothing set", func(t *testing.T) {
		out := laneFixture().Render(RenderOptions{})
		assert.NotContains(t, out, "excluded")
		assert.NotContains(t, out, "collapsed")
		assert.NotContains(t, out, "hidden")
	})
	t.Run("should report a collapse", func(t *testing.T) {
		tl := laneFixture()
		tl.SetCollapsed(42, 3)
		out := tl.Render(RenderOptions{})
		assert.Contains(t, out, "42 jobs collapsed into 3 file lanes (--expand to show all)")
	})
	t.Run("should report a single collapsed file lane in the singular", func(t *testing.T) {
		tl := laneFixture()
		tl.SetCollapsed(2, 1)
		out := tl.Render(RenderOptions{})
		assert.Contains(t, out, "2 jobs collapsed into 1 file lane (--expand to show all)")
	})
	t.Run("should report hidden lanes from --top", func(t *testing.T) {
		tl := laneFixture()
		tl.SetHiddenLanes(5, 10)
		out := tl.Render(RenderOptions{})
		assert.Contains(t, out, "5 lanes hidden (--top 10)")
	})
	t.Run("should report excluded non-active items", func(t *testing.T) {
		tl := laneFixture()
		tl.SetExcluded(1, 2, 3)
		out := tl.Render(RenderOptions{})
		assert.Contains(t, out, "1 suspended job, 2 unresolved jobs, 3 invalid jobs excluded")
	})
	t.Run("should omit zero categories from the excluded note", func(t *testing.T) {
		tl := laneFixture()
		tl.SetExcluded(1, 0, 0)
		out := tl.Render(RenderOptions{})
		assert.Contains(t, out, "1 suspended job excluded")
		assert.NotContains(t, out, "unresolved")
		assert.NotContains(t, out, "invalid")
	})
	t.Run("should report excluded items even when the window has no runs", func(t *testing.T) {
		tl := NewTimeline(DayView, time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC), 80)
		tl.SetExcluded(1, 1, 0)
		out := tl.Render(RenderOptions{})
		assert.Contains(t, out, "no runs in this window")
		assert.Contains(t, out, "1 suspended job, 1 unresolved job excluded")
	})
}
