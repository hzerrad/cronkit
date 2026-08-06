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
		b := planWidths(80, 13, 16)
		assert.Equal(t, budget{gutter: 13, plot: 48, expr: 16}, b)
		assert.Equal(t, 80, b.gutter+1+b.plot+1+1+b.expr)
	})
	t.Run("should drop expr column below 80", func(t *testing.T) {
		b := planWidths(79, 13, 16)
		assert.Equal(t, budget{gutter: 13, plot: 64, expr: 0}, b)
	})
	t.Run("should cap gutter at 10 below 60", func(t *testing.T) {
		b := planWidths(44, 13, 16)
		assert.Equal(t, budget{gutter: 10, plot: 32, expr: 0}, b)
	})
	t.Run("should clamp tiny labels up to 8", func(t *testing.T) {
		assert.Equal(t, 8, planWidths(80, 3, 0).gutter)
	})
	t.Run("should cap wide labels at 17", func(t *testing.T) {
		assert.Equal(t, 17, planWidths(120, 40, 0).gutter)
	})
	t.Run("should drop expr when plot would fall under 30", func(t *testing.T) {
		b := planWidths(80, 17, 17)
		assert.Equal(t, budget{gutter: 17, plot: 43, expr: 17}, b)
		for total := 80; total <= 140; total++ {
			b := planWidths(total, 17, 17)
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
		assert.Equal(t, "backup.sh ", padLabel("backup.sh", 10, '…'))
	})
	t.Run("should truncate long labels with ellipsis", func(t *testing.T) {
		assert.Equal(t, "At 12:00 …", padLabel("At 12:00 every day", 10, '…'))
		assert.Equal(t, 10, len([]rune(padLabel("At 12:00 every day", 10, '…'))))
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

func TestRenderLanes(t *testing.T) {
	t.Run("should render one lane per job in order", func(t *testing.T) {
		out := laneFixture().Render(RenderOptions{})
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		assert.True(t, strings.HasPrefix(lines[0], "cronkit timeline — 2026-08-06 · day · UTC"))
		assert.True(t, strings.HasPrefix(lines[2], "backup.sh"))
		assert.True(t, strings.HasPrefix(lines[3], "verify.sh"))
	})
	t.Run("should keep every body line within total width", func(t *testing.T) {
		out := laneFixture().Render(RenderOptions{})
		for _, ln := range strings.Split(out, "\n") {
			assert.LessOrEqual(t, len([]rune(ln)), 80, ln)
		}
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
		want := `cronkit timeline — 2026-08-06 · day · UTC

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
		want := `cronkit timeline — 2026-08-06 09:00 · hour · UTC

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
