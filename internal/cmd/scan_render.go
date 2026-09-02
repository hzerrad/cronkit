package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/human"
	"github.com/hzerrad/cronkit/internal/inventory"
)

// Column widths for the scan table, following the same fixed-width,
// truncate-with-ellipsis discipline as list.go's table.
const (
	scanColLine   = 4
	scanColSource = 7
	scanColState  = 10
	scanColExpr   = 20

	scanMinPathWidth = 10
	scanMinDescWidth = 15
	scanMaxPathWidth = 40
	scanPathShare    = 0.4 // PATH's cut of the room left over after the fixed columns

	// scanMinExprWidth is EXPRESSION's floor when it and DESCRIPTION are racing for the same room.
	scanMinExprWidth = 10
)

// scanLayout is the resolved column plan for one render, never wider than the width it was asked to fit.
type scanLayout struct {
	showState  bool
	showSource bool
	showPath   bool
	pathW      int
	exprW      int
	descW      int
}

// scanColumnPlan is one candidate set of optional columns to try, richest
// first.
type scanColumnPlan struct {
	showSource bool
	showPath   bool
}

// scanColumnPlans lists the column sets newScanLayout tries, in the order columns are shed as width runs out.
var scanColumnPlans = []scanColumnPlan{
	{showSource: true, showPath: true},
	{showSource: false, showPath: true},
	{showSource: false, showPath: false},
}

// scanColumnPlansNoPath is scanColumnPlans with every PATH-showing plan dropped.
var scanColumnPlansNoPath = []scanColumnPlan{
	{showSource: true, showPath: false},
	{showSource: false, showPath: false},
}

// newScanLayout fits the table to width, shedding optional columns until one plan fits.
func newScanLayout(width int, showState, hasPath bool) scanLayout {
	plans := scanColumnPlans
	if !hasPath {
		plans = scanColumnPlansNoPath
	}

	for _, plan := range plans {
		if layout, ok := fitScanColumnPlan(width, showState, plan); ok {
			return layout
		}
	}

	last := plans[len(plans)-1]
	fixed, gutters := scanColumnCost(showState, last)
	room := width - (fixed - scanColExpr) - gutters
	exprW, descW := splitExprAndDesc(room)
	return scanLayout{showState: showState, showSource: last.showSource, showPath: last.showPath, exprW: exprW, descW: descW}
}

// splitExprAndDesc divides the room left for EXPRESSION and DESCRIPTION combined.
func splitExprAndDesc(room int) (exprW, descW int) {
	if room < 0 {
		room = 0
	}

	exprW = scanColExpr
	if exprW > room {
		exprW = room
	}

	if descW = room - exprW; descW == 0 && exprW > scanMinExprWidth {
		exprW = scanMinExprWidth
		descW = room - exprW
	}

	return exprW, descW
}

// scanColumnCost returns the total width of plan's fixed-shape columns and the total gutter width.
func scanColumnCost(showState bool, plan scanColumnPlan) (fixed, gutters int) {
	fixed = scanColLine + scanColExpr
	numCols := 3 // LINE, EXPRESSION, DESCRIPTION
	if showState {
		fixed += scanColState
		numCols++
	}
	if plan.showSource {
		fixed += scanColSource
		numCols++
	}
	if plan.showPath {
		numCols++
	}
	gutters = (numCols - 1) * 2
	return fixed, gutters
}

// fitScanColumnPlan reports whether plan's columns fit in width, and if so, the resulting layout.
func fitScanColumnPlan(width int, showState bool, plan scanColumnPlan) (scanLayout, bool) {
	fixed, gutters := scanColumnCost(showState, plan)

	minNeeded := fixed + gutters + scanMinDescWidth
	if plan.showPath {
		minNeeded += scanMinPathWidth
	}
	if width < minNeeded {
		return scanLayout{}, false
	}

	layout := scanLayout{showState: showState, showSource: plan.showSource, showPath: plan.showPath, exprW: scanColExpr}
	remaining := width - fixed - gutters

	if !plan.showPath {
		layout.descW = remaining
		return layout, true
	}

	pathW := int(float64(remaining) * scanPathShare)
	if pathW < scanMinPathWidth {
		pathW = scanMinPathWidth
	}
	if pathW > scanMaxPathWidth {
		pathW = scanMaxPathWidth
	}
	layout.pathW = pathW
	layout.descW = remaining - pathW
	return layout, true
}

// row lays out one row across the columns the layout decided to show.
func (l scanLayout) row(line, path, source, state, expression, description string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%-*s  ", scanColLine, line)
	if l.showPath {
		fmt.Fprintf(&b, "%-*s  ", l.pathW, path)
	}
	if l.showSource {
		fmt.Fprintf(&b, "%-*s  ", scanColSource, source)
	}
	if l.showState {
		fmt.Fprintf(&b, "%-*s  ", scanColState, state)
	}
	if l.descW == 0 {
		b.WriteString(expression)
		return b.String()
	}
	fmt.Fprintf(&b, "%-*s  ", l.exprW, expression)
	b.WriteString(description)
	return b.String()
}

// renderScanText writes a width-fit rendering of inv to w, grouped by file with a summary line.
func renderScanText(w io.Writer, inv *inventory.Inventory, width int) {
	if len(inv.Items) == 0 {
		_, _ = fmt.Fprintln(w, "No schedules found")
		return
	}

	// Uses the global --locale flag, matching the other commands.
	parser := cronx.NewParserWithLocale(GetLocale())
	humanizer := human.NewHumanizer()

	showState := false
	hasPath := false
	for _, item := range inv.Items {
		if item.State != inventory.StateActive {
			showState = true
		}
		if item.Locator.Path != "" {
			hasPath = true
		}
	}

	layout := newScanLayout(width, showState, hasPath)

	// EXPRESSION and DESCRIPTION headers get the same truncation as their column data.
	_, _ = fmt.Fprintln(w, layout.row("LINE", "PATH", "SOURCE", "STATE", truncateField("EXPRESSION", layout.exprW), truncateField("DESCRIPTION", layout.descW)))
	_, _ = fmt.Fprintln(w, layout.row(
		strings.Repeat("─", scanColLine),
		strings.Repeat("─", layout.pathW),
		strings.Repeat("─", scanColSource),
		strings.Repeat("─", scanColState),
		strings.Repeat("─", layout.exprW),
		strings.Repeat("─", layout.descW),
	))

	summary := newScanSummary()
	currentFile := ""
	for _, item := range inv.Items {
		if item.Locator.File != currentFile {
			// The file-group heading is exempt from width fitting since a path must stay complete to be useful.
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, item.Locator.File)
			currentFile = item.Locator.File
		}

		line := ""
		if item.Locator.Line > 0 {
			line = fmt.Sprintf("%d", item.Locator.Line)
		}

		var path string
		if layout.showPath {
			path = truncateFieldLeft(item.Locator.Path, layout.pathW)
		}
		expression := truncateField(item.Expression, layout.exprW)
		description := truncateField(describeItem(parser, humanizer, item), layout.descW)

		_, _ = fmt.Fprintln(w, layout.row(line, path, item.SourceID, item.State.String(), expression, description))

		summary.add(item)
	}

	_, _ = fmt.Fprintln(w)
	for _, line := range summary.render(width) {
		_, _ = fmt.Fprintln(w, line)
	}
}

// truncateField shortens s to max runes from the tail, replacing it with "...".
func truncateField(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}

// truncateFieldLeft shortens s to max runes from the front, keeping the tail and prefixing "...".
func truncateFieldLeft(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[len(r)-max:])
	}
	// Trims a leading "." from the retained tail so it doesn't run into the "..." prefix and read as four dots.
	tail := strings.TrimPrefix(string(r[len(r)-(max-3):]), ".")
	return "..." + tail
}

// wrapText greedily word-wraps s to width without breaking words; width<=0 disables wrapping.
func wrapText(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}

	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{s}
	}

	var lines []string
	cur := words[0]
	for _, word := range words[1:] {
		if len(cur)+1+len(word) > width {
			lines = append(lines, cur)
			cur = word
			continue
		}
		cur += " " + word
	}
	return append(lines, cur)
}

// describeItem says what a schedule means in English, falling back to the item's Reason when inactive.
func describeItem(parser cronx.Parser, humanizer human.Humanizer, item inventory.Item) string {
	if item.State != inventory.StateActive {
		if item.Reason != "" {
			return item.Reason
		}
		return fmt.Sprintf("(%s)", item.State)
	}

	schedule, err := parser.Parse(item.Expression)
	if err != nil {
		return fmt.Sprintf("(unparseable: %s)", err)
	}
	return humanizer.Humanize(schedule)
}

// scanSummary accumulates the counts renderScanText's closing line reports.
type scanSummary struct {
	total      int
	files      map[string]struct{}
	sourceSeen map[string]bool
	sources    []string
	suspended  int
	unresolved int
	invalid    int
}

func newScanSummary() *scanSummary {
	return &scanSummary{
		files:      map[string]struct{}{},
		sourceSeen: map[string]bool{},
	}
}

func (s *scanSummary) add(item inventory.Item) {
	s.total++
	s.files[item.Locator.File] = struct{}{}
	if !s.sourceSeen[item.SourceID] {
		s.sourceSeen[item.SourceID] = true
		s.sources = append(s.sources, item.SourceID)
	}

	switch item.State {
	case inventory.StateSuspended:
		s.suspended++
	case inventory.StateUnresolved:
		s.unresolved++
	case inventory.StateInvalid:
		s.invalid++
	}
}

// clauses splits the summary into its two natural clauses for render's line-splitting fallback.
func (s *scanSummary) clauses() (found, breakdown string) {
	found = fmt.Sprintf("%d schedule(s) across %d file(s) from %d source(s) (%s)",
		s.total, len(s.files), len(s.sources), strings.Join(s.sources, ", "))
	breakdown = fmt.Sprintf("%d suspended, %d unresolved, %d invalid", s.suspended, s.unresolved, s.invalid)
	return found, breakdown
}

func (s *scanSummary) String() string {
	found, breakdown := s.clauses()
	return found + "; " + breakdown
}

// render fits the summary to width, falling back to splitting at the clause boundary and then to wrapText.
func (s *scanSummary) render(width int) []string {
	combined := s.String()
	if width <= 0 || len(combined) <= width {
		return []string{combined}
	}

	found, breakdown := s.clauses()
	lines := wrapText(found, width)
	return append(lines, wrapText(breakdown, width)...)
}
