package render

import (
	"fmt"
	"strings"
	"time"
)

type budget struct {
	gutter, plot, expr int
}

// planWidths splits total columns into gutter, plot, and optional expr widths.
func planWidths(total, label, expr int) budget {
	g := label
	if g < 8 {
		g = 8
	}
	if g > 17 {
		g = 17
	}
	if total < 60 && g > 10 {
		g = 10
	}
	if total >= 80 && expr > 0 {
		e := expr
		if e > 17 {
			e = 17
		}
		if p := total - g - 2 - 1 - e; p >= 30 {
			return budget{gutter: g, plot: p, expr: e}
		}
	}
	return budget{gutter: g, plot: total - g - 2}
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
	dash, dot                        string
}

var uniGlyphs = glyphset{'╷', '┃', '▲', '┤', '├', '─', '┬', '└', '┘', '…', "—", "·"}
var asciiGlyphs = glyphset{'|', '#', '^', '|', '|', '-', '+', '+', '+', '~', "-", "-"}

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

// padLabel pads s to width w with spaces, or truncates with ellipsis when too long.
func padLabel(s string, w int, ellipsis rune) string {
	r := []rune(s)
	if len(r) > w {
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

func laneRow(label, expr string, cells []rune, b budget, g glyphset, code string) string {
	row := padLabel(label, b.gutter, g.ellipsis) + string(g.lhs) +
		colorize(string(cells), code) + string(g.rhs)
	if b.expr > 0 && expr != "" {
		row += " " + expr
	}
	return strings.TrimRight(row, " ")
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

func (tl *Timeline) headerLine(g glyphset) string {
	date := "2006-01-02"
	if tl.view == HourView {
		date = "2006-01-02 15:04"
	}
	return fmt.Sprintf("cronkit timeline %s %s %s %s %s %s",
		g.dash, tl.startTime.Format(date), g.dot, tl.view.String(), g.dot, tl.startTime.Location().String())
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
