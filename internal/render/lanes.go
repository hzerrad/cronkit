package render

import (
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

type glyphset struct { // nolint:unused // consumed by Task 3's renderer
	run, merged, conflict            rune
	lhs, rhs                         rune
	axis, tickMark, cornerL, cornerR rune
	ellipsis                         rune
	dash, dot                        string
}

var uniGlyphs = glyphset{'╷', '┃', '▲', '┤', '├', '─', '┬', '└', '┘', '…', "—", "·"}   // nolint:unused // consumed by Task 3's renderer
var asciiGlyphs = glyphset{'|', '#', '^', '|', '|', '-', '+', '+', '+', '~', "-", "-"} // nolint:unused // consumed by Task 3's renderer

const (
	ansiRed   = "\x1b[31m"
	ansiDim   = "\x1b[2m"
	ansiReset = "\x1b[0m"
)

var laneColors = []string{"\x1b[36m", "\x1b[32m", "\x1b[33m", "\x1b[35m", "\x1b[34m", "\x1b[90m"} // nolint:unused // consumed by Task 3's renderer

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
