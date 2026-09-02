package render

import (
	"fmt"
	"strings"
	"time"
)

type budget struct {
	gutter, plot, expr int
}

// maxPlotCells returns the widest useful plot for a view: 15 minutes per cell
// for a day, one minute per cell for an hour.
func maxPlotCells(view TimelineView) int {
	if view == HourView {
		return 60
	}
	return 96
}

// planWidths splits total columns into gutter, plot, and optional expr widths,
// holding the plot to maxPlot so wide terminals stop stretching the chart.
func planWidths(total, label, expr, maxPlot int) budget {
	g := label
	if g < 8 {
		g = 8
	}
	maxGutter := 32
	if total < 60 {
		maxGutter = 10
	} else if total < 100 {
		maxGutter = 17
	}
	if g > maxGutter {
		g = maxGutter
	}
	if total >= 80 && expr > 0 {
		e := expr
		if e > 17 {
			e = 17
		}
		if p := total - g - 2 - 1 - e; p >= 30 {
			return budget{gutter: g, plot: min(p, maxPlot), expr: e}
		}
	}
	return budget{gutter: g, plot: min(total-g-2, maxPlot)}
}

// cellPos maps a time to a clamped column within the plot width.
func cellPos(t, start time.Time, dur time.Duration, plotW int) int {
	p := int(float64(t.Sub(start)) / float64(dur) * float64(plotW))
	if p < 0 {
		p = 0
	}
	if p >= plotW {
		p = plotW - 1
	}
	return p
}

type tick struct {
	col   int
	label string
}

// axisTicks returns evenly spaced time labels for the view plus an end-of-range label.
func axisTicks(view TimelineView, start time.Time, plotW int) []tick {
	step, dur := 6*time.Hour, 24*time.Hour
	if view == HourView {
		step, dur = 15*time.Minute, time.Hour
	}
	var ticks []tick
	for off := time.Duration(0); off < dur; off += step {
		at := start.Add(off)
		ticks = append(ticks, tick{cellPos(at, start, dur, plotW), at.Format("15:04")})
	}
	end := start.Add(dur - time.Minute)
	ticks = append(ticks, tick{plotW - 1, end.Format("15:04")})
	return ticks
}

type glyphset struct {
	run, merged, conflict            rune
	lhs, rhs                         rune
	axis, tickMark, cornerL, cornerR rune
	ellipsis                         rune
	dash, dot, arrow                 string
}

var uniGlyphs = glyphset{'╷', '┃', '▲', '┤', '├', '─', '┬', '└', '┘', '…', "—", "·", "→"}
var asciiGlyphs = glyphset{'|', '#', '^', '|', '|', '-', '+', '+', '+', '~', "-", "-", "->"}

const (
	ansiRed   = "\x1b[31m"
	ansiDim   = "\x1b[2m"
	ansiReset = "\x1b[0m"
)

var laneColors = []string{"\x1b[36m", "\x1b[32m", "\x1b[33m", "\x1b[35m", "\x1b[34m", "\x1b[90m"}

// colorize wraps s with an ANSI code and reset, or leaves it untouched when code is empty.
func colorize(s, code string) string {
	if code == "" {
		return s
	}
	return code + s + ansiReset
}

// padLabel pads s to width w with spaces, or truncates when too long;
// pathLike selects truncPathLeft, which keeps the tail, instead of trailing-ellipsis truncation.
func padLabel(s string, w int, ellipsis rune, pathLike bool) string {
	r := []rune(s)
	if len(r) > w {
		if pathLike {
			return truncPathLeft(s, w)
		}
		return string(r[:w-1]) + string(ellipsis)
	}
	return s + strings.Repeat(" ", w-len(r))
}

func blankCells(w int) []rune {
	c := make([]rune, w)
	for i := range c {
		c[i] = ' '
	}
	return c
}

func laneColor(i int, on bool) string {
	if !on {
		return ""
	}
	return laneColors[i%len(laneColors)]
}

// laneRow renders one lane; exprPathLike and labelPathLike independently say
// whether expr/label is a file path rather than a cron expression, so each truncates from the correct end.
func laneRow(label, expr string, exprPathLike, labelPathLike bool, cells []rune, b budget, g glyphset, code string) string {
	row := padLabel(label, b.gutter, g.ellipsis, labelPathLike) + string(g.lhs) +
		colorize(string(cells), code) + string(g.rhs)
	if b.expr > 0 && expr != "" {
		if exprPathLike {
			row += " " + truncPathLeft(expr, b.expr)
		} else {
			row += " " + truncExpr(expr, b.expr, g.ellipsis)
		}
	}
	return strings.TrimRight(row, " ")
}

// truncExpr truncates s to at most w runes, ending in the ellipsis when it overflows.
func truncExpr(s string, w int, ellipsis rune) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	return string(r[:w-1]) + string(ellipsis)
}

// truncPathLeft truncates s to at most w runes from the front, keeping the
// tail and prefixing "..." since the informative part of a file path sits at the end.
func truncPathLeft(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w <= 3 {
		return string(r[len(r)-w:])
	}
	tail := strings.TrimPrefix(string(r[len(r)-(w-3):]), "/")
	return "..." + tail
}

// distinctFileCount counts distinct non-empty file names registered across
// jobs, so Render can tell a single-file chart from one spanning multiple files.
func distinctFileCount(jobs []jobEntry) int {
	seen := make(map[string]bool)
	for _, j := range jobs {
		if j.file != "" {
			seen[j.file] = true
		}
	}
	return len(seen)
}

// laneGutter returns what the right gutter should trail a lane with: the item
// count for a collapsed lane, file:line once the chart spans multiple files, otherwise laneExpr's answer.
func laneGutter(j jobEntry, multiFile bool) string {
	if j.count > 0 {
		return plural(j.count, "job")
	}
	if multiFile && j.file != "" {
		if j.line > 0 {
			return fmt.Sprintf("%s:%d", j.file, j.line)
		}
		return j.file
	}
	return laneExpr(j)
}

// gutterIsPath reports whether j's gutter value is a file path, so laneRow truncates it correctly.
func gutterIsPath(j jobEntry, multiFile bool) bool {
	return j.count == 0 && multiFile && j.file != ""
}

// labelIsPath reports whether j's label is a file path, so laneRow truncates it correctly.
func labelIsPath(j jobEntry) bool {
	return j.count > 0
}

// axisRows draws the tick axis and its centered, collision-skipping label row.
func axisRows(view TimelineView, start time.Time, b budget, g glyphset) (string, string) {
	ticks := axisTicks(view, start, b.plot)

	axis := make([]rune, b.plot)
	for i := range axis {
		axis[i] = g.axis
	}
	for _, tk := range ticks {
		axis[tk.col] = g.tickMark
	}
	axisLine := strings.Repeat(" ", b.gutter) + string(g.cornerL) + string(axis) + string(g.cornerR)

	buf := blankCells(b.plot + 2)
	prevEnd := -1
	for _, tk := range ticks {
		label := []rune(tk.label)
		start := tk.col + 1 - len(label)/2
		if start < 0 {
			start = 0
		}
		if end := start + len(label); end > len(buf) {
			start = len(buf) - len(label)
		}
		if start <= prevEnd {
			continue // no room left after the previous label
		}
		copy(buf[start:], label)
		prevEnd = start + len(label)
	}
	labelLine := strings.Repeat(" ", b.gutter) + string(buf)

	return axisLine, labelLine
}

// zoneLabel names the timeline's zone, falling back to abbreviation and offset
// when the location is Go's unnamed Local.
func zoneLabel(t time.Time) string {
	name := t.Location().String()
	if name == "UTC" {
		return name
	}
	if name == "" || name == "Local" {
		name = t.Format("MST")
	}
	return fmt.Sprintf("%s (UTC%s)", name, t.Format("-07:00"))
}

func (tl *Timeline) windowLine(g glyphset) string {
	line := fmt.Sprintf("%s  %s %s %s %s %s",
		tl.startTime.Format("2006-01-02"),
		tl.startTime.Format("15:04"),
		g.arrow,
		tl.endTime.Add(-time.Minute).Format("15:04"),
		g.dot,
		zoneLabel(tl.startTime))
	// Run times above are already converted into the axis zone.
	if len(tl.foreignZones) > 0 {
		line += fmt.Sprintf(" %s converted from %s", g.dot, strings.Join(tl.foreignZones, ", "))
	}
	return line
}

func (tl *Timeline) footerLine(overlapCount int, g glyphset) string {
	parts := []string{plural(len(tl.jobs), "job"), plural(len(tl.jobRuns), "run")}
	if overlapCount == 0 {
		parts = append(parts, "no conflicts")
	} else {
		parts = append(parts,
			plural(overlapCount, "conflict window"),
			fmt.Sprintf("max %d concurrent", tl.GetOverlapStats().MaxConcurrent))
	}
	// The three notes below are all opt-in via their setters and no-ops in their zero state.
	if tl.collapsedLanes > 0 {
		parts = append(parts, fmt.Sprintf("%s collapsed into %s (--expand to show all)",
			plural(tl.collapsedItems, "job"), plural(tl.collapsedLanes, "file lane")))
	}
	if tl.hiddenLanes > 0 {
		parts = append(parts, fmt.Sprintf("%s hidden (--top %d)", plural(tl.hiddenLanes, "lane"), tl.topLimit))
	}
	if excluded := tl.excludedSuspended + tl.excludedUnresolved + tl.excludedInvalid; excluded > 0 {
		var bits []string
		if tl.excludedSuspended > 0 {
			bits = append(bits, plural(tl.excludedSuspended, "suspended job"))
		}
		if tl.excludedUnresolved > 0 {
			bits = append(bits, plural(tl.excludedUnresolved, "unresolved job"))
		}
		if tl.excludedInvalid > 0 {
			bits = append(bits, plural(tl.excludedInvalid, "invalid job"))
		}
		parts = append(parts, strings.Join(bits, ", ")+" excluded")
	}
	return strings.Join(parts, " "+g.dot+" ")
}

// plural formats a count with its noun, singular or plural.
func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func (tl *Timeline) jobLabel(id string) string {
	if idx, ok := tl.jobIndex[id]; ok {
		return tl.jobs[idx].label
	}
	return id
}

func (tl *Timeline) overlapLines(overlaps []Overlap, g glyphset) string {
	n := len(overlaps)
	if n > 50 {
		n = 50
	}
	var sb strings.Builder
	sb.WriteString("\noverlaps:\n")
	for _, o := range overlaps[:n] {
		labels := make([]string, len(o.JobIDs))
		for i, id := range o.JobIDs {
			labels[i] = tl.jobLabel(id)
		}
		fmt.Fprintf(&sb, "  %s  %s\n", o.Time.Format("15:04"), strings.Join(labels, ", "))
	}
	if len(overlaps) > 50 {
		fmt.Fprintf(&sb, "  %s and %d more\n", string(g.ellipsis), len(overlaps)-50)
	}
	return sb.String()
}
