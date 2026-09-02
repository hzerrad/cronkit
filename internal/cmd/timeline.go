package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/human"
	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/hzerrad/cronkit/internal/render"
	"github.com/hzerrad/cronkit/internal/timeutil"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// collapseLaneThreshold is where the chart switches from one lane per job to one aggregate lane per file.
const collapseLaneThreshold = 20

// TimelineCommand wraps cobra.Command with timeline-specific functionality
type TimelineCommand struct {
	*cobra.Command
	inputFlags
	json         bool
	view         string
	from         string
	width        int
	timezone     string
	export       string
	locale       string
	showOverlaps bool
	color        string
	ascii        bool
	expand       bool
	top          int
}

func init() {
	rootCmd.AddCommand(newTimelineCommand().Command)
}

// newTimelineCommand creates a fresh timeline command instance for testing
func newTimelineCommand() *TimelineCommand {
	tc := &TimelineCommand{}
	tc.Command = &cobra.Command{
		Args:  cobra.MaximumNArgs(1),
		RunE:  tc.runTimeline,
		Use:   "timeline [cron-expression]",
		Short: "Display ASCII timeline visualization of cron job schedules",
		Long: `Display an ASCII timeline showing when cron jobs will run, including job density and overlaps.

This command helps visualize cron schedules over time, making it easy to see when jobs run
and identify potential conflicts or resource contention.

Supports:
  - Single cron expression (provided as argument)
  - Crontab file (via --file flag)
  - User's crontab (default when no argument or --file provided)
  - Day view (24 hours, default) or hour view (60 minutes) via --view flag
  - JSON output with --json flag for programmatic use

Examples:
  cronkit timeline "*/15 * * * *"              # Timeline for single expression
  cronkit timeline --file /etc/crontab          # Timeline for crontab file
  cronkit timeline "*/5 * * * *" --view hour    # Hour view timeline
  cronkit timeline --file jobs.cron --json       # JSON output
  cronkit timeline                               # Timeline for user's crontab`,
	}

	tc.register(tc.Command, false)
	tc.Command.Flags().BoolVarP(&tc.json, "json", "j", false, "Output in JSON format")
	tc.Command.Flags().StringVar(&tc.view, "view", "day", "Timeline view type: 'day' (24 hours) or 'hour' (60 minutes, default: 'day')")
	tc.Command.Flags().StringVar(&tc.from, "from", "", "Start time for timeline (RFC3339 format, defaults to current time)")
	tc.Command.Flags().IntVar(&tc.width, "width", 0, "Terminal width (0 = auto-detect, defaults to 80 if detection fails)")
	tc.Command.Flags().StringVar(&tc.timezone, "timezone", "", "Timezone for timeline (e.g., 'America/New_York', 'UTC', defaults to local timezone)")
	tc.Command.Flags().StringVar(&tc.export, "export", "", "Export timeline to file (format determined by extension: .txt, .json)")
	tc.Command.Flags().BoolVar(&tc.showOverlaps, "show-overlaps", false, "Show detailed overlap information in output")
	tc.Command.Flags().StringVar(&tc.color, "color", "auto", "Color output: auto, always, or never")
	tc.Command.Flags().BoolVar(&tc.ascii, "ascii", false, "Plain ASCII glyphs")
	tc.Command.Flags().BoolVar(&tc.expand, "expand", false,
		fmt.Sprintf("Always draw one lane per job, even past %d (default: collapse to one lane per file)", collapseLaneThreshold))
	tc.Command.Flags().IntVar(&tc.top, "top", 0,
		"Cap the chart to the busiest N lanes by run count, ties broken by file/line (0 = no cap)")

	return tc
}

func (tc *TimelineCommand) runTimeline(_ *cobra.Command, args []string) error {
	// Determine timeline view
	var timelineView render.TimelineView
	switch tc.view {
	case "day":
		timelineView = render.DayView
	case "hour":
		timelineView = render.HourView
	default:
		return fmt.Errorf("invalid view type: %s (must be 'day' or 'hour')", tc.view)
	}

	tty := stdoutTTY()
	colorOn, err := resolveColor(tc.color, tty)
	if err != nil {
		return err
	}
	if tc.export != "" {
		colorOn = false // one render feeds both the export file and the stdout echo; keep the file clean
	}

	// Determine timezone
	loc := time.Local
	if tc.timezone != "" {
		parsedLoc, err := time.LoadLocation(tc.timezone)
		if err != nil {
			return fmt.Errorf("invalid timezone: %w (use IANA timezone name like 'America/New_York' or 'UTC')", err)
		}
		loc = parsedLoc
	}

	// Determine start time
	startTime := time.Now().In(loc)
	if tc.from != "" {
		parsed, err := time.Parse(time.RFC3339, tc.from)
		if err != nil {
			return fmt.Errorf("invalid --from time format: %w (expected RFC3339)", err)
		}
		startTime = parsed.In(loc)
	}

	// Round down start time based on view
	if timelineView == render.DayView {
		startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())
	} else {
		startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(), startTime.Hour(), 0, 0, 0, startTime.Location())
	}

	// Determine width (auto-detect if not specified)
	width := resolveWidth(tc.width, tty)
	if width < 40 {
		width = 40 // Minimum width for readability
	}

	// Create timeline
	timeline := render.NewTimeline(timelineView, startTime, width)

	// Get locale
	locale := GetLocale()
	if tc.locale != "" {
		locale = tc.locale
	}

	// Resolve schedules
	var items []inventory.Item

	if len(args) > 0 {
		// Single expression provided
		expression := args[0]
		parser := cronx.NewParserWithLocale(locale)
		_, err = parser.Parse(expression)
		if err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}

		// Create an item for the expression
		items = []inventory.Item{
			{
				Expression: expression,
				Command:    "(single expression)",
				Shell:      true,
				State:      inventory.StateActive,
			},
		}
	} else {
		items, err = resolveItems(&tc.inputFlags)
		if err != nil {
			switch tc.classify() {
			case sourceStdin:
				return fmt.Errorf("failed to read crontab from stdin: %w", err)
			case sourceInventory:
				return fmt.Errorf("failed to read inventory: %w", err)
			case sourceFile:
				return fmt.Errorf("failed to read crontab file: %w", err)
			default:
				return fmt.Errorf("failed to read user crontab: %w", err)
			}
		}
		timeline.SetSource(timelineSourceLabel(&tc.inputFlags), "")
	}

	// Sort by locator (file, then line) and finally expression, matching inventory.Inventory.Sort.
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.Locator.File != b.Locator.File {
			return a.Locator.File < b.Locator.File
		}
		if a.Locator.Line != b.Locator.Line {
			return a.Locator.Line < b.Locator.Line
		}
		return a.Expression < b.Expression
	})

	// Process items and add runs to timeline
	parser := cronx.NewParserWithLocale(locale)
	humanizer := human.NewHumanizer()
	scheduler := cronx.NewScheduler()

	// Calculate how many runs to get based on view
	var runCount int
	var timeRange time.Duration
	if timelineView == render.DayView {
		timeRange = 24 * time.Hour // Using literal for comparison, OneDay constant is in stats package
		runCount = 200             // Enough to cover a day for most schedules
	} else {
		timeRange = time.Hour
		runCount = 100 // Enough to cover an hour for most schedules
	}
	endTime := startTime.Add(timeRange)

	if len(args) > 0 {
		// A single-expression argument is always one item, using the expression itself as source and label.
		item := items[0]
		if schedule, err := parser.Parse(item.Expression); err == nil {
			description := humanizer.Humanize(schedule)
			jobID := fmt.Sprintf("expr-%s", item.Expression)
			timeline.SetSource(item.Expression, description)
			timeline.SetJobInfo(jobID, item.Expression, description, item.Expression)
			addLaneRuns(timeline, scheduler, item, jobID, startTime, endTime, runCount)
		}
	} else {
		renderInventoryLanes(timeline, items, parser, humanizer, scheduler, startTime, endTime, runCount, tc.expand, tc.top)
	}

	// Output based on format
	var output string
	if tc.json {
		result := timeline.RenderJSON()
		// Add timezone and locale to JSON output
		result["timezone"] = loc.String()
		result["locale"] = locale

		// If exporting JSON, write to file, otherwise to stdout
		if tc.export != "" {
			file, err := os.Create(tc.export)
			if err != nil {
				return fmt.Errorf("failed to create export file: %w", err)
			}
			encoder := json.NewEncoder(file)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(result); err != nil {
				_ = file.Close()
				return fmt.Errorf("failed to encode JSON: %w", err)
			}
			if err := file.Close(); err != nil {
				return fmt.Errorf("failed to close export file: %w", err)
			}
		} else {
			encoder := json.NewEncoder(tc.OutOrStdout())
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(result); err != nil {
				return fmt.Errorf("failed to encode JSON: %w", err)
			}
		}
		return nil
	}

	// Text output
	output = timeline.Render(render.RenderOptions{Color: colorOn, ASCII: tc.ascii, ShowOverlaps: tc.showOverlaps})

	// Handle export if specified
	if tc.export != "" {
		if err := tc.exportTimeline(output, timeline); err != nil {
			return fmt.Errorf("failed to export timeline: %w", err)
		}
		// Also print to stdout when exporting
		tc.Print(output)
	} else {
		// Normal output
		tc.Print(output)
	}

	return nil
}

// computeItemRuns resolves item's timezone and returns its next runs, converted to the axis zone, that
// fall in [startTime, endTime). Returns nil rather than an error so a bad item is skipped, not fatal.
func computeItemRuns(scheduler cronx.Scheduler, item inventory.Item, startTime, endTime time.Time, runCount int) []time.Time {
	loc, err := timeutil.ResolveLocation(item.Timezone, startTime.Location())
	if err != nil {
		return nil
	}
	times, err := scheduler.Next(item.Expression, startTime.In(loc), runCount)
	if err != nil {
		return nil
	}
	var runs []time.Time
	for _, runTime := range times {
		converted := runTime.In(startTime.Location())
		if converted.Before(endTime) && !converted.Before(startTime) {
			runs = append(runs, converted)
		}
		if !converted.Before(endTime) {
			break
		}
	}
	return runs
}

// addLaneRuns adds every run computeItemRuns finds for item to timeline under jobID.
func addLaneRuns(timeline *render.Timeline, scheduler cronx.Scheduler, item inventory.Item, jobID string, startTime, endTime time.Time, runCount int) {
	for _, rt := range computeItemRuns(scheduler, item, startTime, endTime, runCount) {
		timeline.AddJobRun(jobID, rt)
	}
}

// capLanesByRunCount decides which of n lanes survive a --top cap, keeping the busiest by run count.
func capLanesByRunCount(n int, runs [][]time.Time, top int, locator func(i int) (file string, line int, expr string)) (keep []int, hidden int) {
	if top <= 0 || top >= n {
		keep = make([]int, n)
		for i := range keep {
			keep[i] = i
		}
		return keep, 0
	}

	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		ia, ib := order[a], order[b]
		if ra, rb := len(runs[ia]), len(runs[ib]); ra != rb {
			return ra > rb
		}
		fa, la, ea := locator(ia)
		fb, lb, eb := locator(ib)
		if fa != fb {
			return fa < fb
		}
		if la != lb {
			return la < lb
		}
		return ea < eb
	})

	kept := make(map[int]bool, top)
	for _, idx := range order[:top] {
		kept[idx] = true
	}
	for i := 0; i < n; i++ {
		if kept[i] {
			keep = append(keep, i)
		}
	}
	return keep, n - top
}

// renderInventoryLanes turns a resolved, locator-sorted set of items into timeline lanes, one per job unless
// past collapseLaneThreshold (then one per file, unless expand).
func renderInventoryLanes(timeline *render.Timeline, items []inventory.Item, parser cronx.Parser, humanizer human.Humanizer, scheduler cronx.Scheduler, startTime, endTime time.Time, runCount int, expand bool, top int) {
	var excludedSuspended, excludedUnresolved, excludedInvalid int
	activeItems := make([]inventory.Item, 0, len(items))
	for _, item := range items {
		switch item.State {
		case inventory.StateActive:
			activeItems = append(activeItems, item)
		case inventory.StateSuspended:
			excludedSuspended++
		case inventory.StateUnresolved:
			excludedUnresolved++
		case inventory.StateInvalid:
			excludedInvalid++
		}
	}
	if excludedSuspended+excludedUnresolved+excludedInvalid > 0 {
		timeline.SetExcluded(excludedSuspended, excludedUnresolved, excludedInvalid)
	}

	// Note every distinct item zone that differs from the axis, for the window line's conversion note.
	axisZone := startTime.Location().String()
	foreignSeen := make(map[string]bool)
	var foreignZones []string
	for _, item := range activeItems {
		if item.Timezone != "" && item.Timezone != axisZone && !foreignSeen[item.Timezone] {
			foreignSeen[item.Timezone] = true
			foreignZones = append(foreignZones, item.Timezone)
		}
	}
	if len(foreignZones) > 0 {
		sort.Strings(foreignZones)
		timeline.SetForeignZones(foreignZones)
	}

	if !expand && len(activeItems) > collapseLaneThreshold {
		renderAggregateLanes(timeline, activeItems, scheduler, startTime, endTime, runCount, top)
		return
	}
	renderPerJobLanes(timeline, activeItems, parser, humanizer, scheduler, startTime, endTime, runCount, top)
}

// renderPerJobLanes registers one lane per active item, capped to top when positive.
func renderPerJobLanes(timeline *render.Timeline, activeItems []inventory.Item, parser cronx.Parser, humanizer human.Humanizer, scheduler cronx.Scheduler, startTime, endTime time.Time, runCount, top int) {
	// Runs are computed once, up front, so ranking by --top never costs a second pass over the scheduler.
	runsByItem := make([][]time.Time, len(activeItems))
	for i, item := range activeItems {
		runsByItem[i] = computeItemRuns(scheduler, item, startTime, endTime, runCount)
	}

	keepIdx, hidden := capLanesByRunCount(len(activeItems), runsByItem, top, func(i int) (string, int, string) {
		it := activeItems[i]
		return it.Locator.File, it.Locator.Line, it.Expression
	})
	if hidden > 0 {
		timeline.SetHiddenLanes(hidden, top)
	}
	shown := make([]inventory.Item, len(keepIdx))
	shownRuns := make([][]time.Time, len(keepIdx))
	for i, idx := range keepIdx {
		shown[i] = activeItems[idx]
		shownRuns[i] = runsByItem[idx]
	}

	// Pre-count lane label basenames so duplicates can be disambiguated with a :LINE suffix.
	labelCounts := make(map[string]int)
	labelLineCounts := make(map[string]int)
	for _, item := range shown {
		label := laneLabel(item)
		labelCounts[label]++
		if item.Locator.Line > 0 {
			labelLineCounts[label+":"+strconv.Itoa(item.Locator.Line)]++
		}
	}

	for i, item := range shown {
		// Parse expression
		schedule, err := parser.Parse(item.Expression)
		if err != nil {
			continue // Skip invalid expressions
		}

		// Get human description
		description := humanizer.Humanize(schedule)

		// Generate job ID, keyed on the locator (via inventory.Locator.Identity)
		// so it can never collide.
		jobID := item.Locator.Identity(i, "job-", "job-")

		// Lane label: command basename, deduped with :LINE, escalated to :file:LINE if that still collides.
		label := laneLabel(item)
		if labelCounts[label] > 1 {
			if item.Locator.Line == 0 {
				// No line to key on, so fall back to this item's position, unique by construction.
				label = label + ":" + strconv.Itoa(i)
			} else {
				lineSuffix := strconv.Itoa(item.Locator.Line)
				if labelLineCounts[label+":"+lineSuffix] > 1 {
					label = label + ":" + item.Locator.File + ":" + lineSuffix
				} else {
					label = label + ":" + lineSuffix
				}
			}
		}

		timeline.SetJobInfo(jobID, item.Expression, description, label)
		// SetJobLocator is harmless unconditionally: Render only switches to a file:line gutter past one file.
		timeline.SetJobLocator(jobID, item.Locator.File, item.Locator.Line, item.Locator.Path)
		for _, rt := range shownRuns[i] {
			timeline.AddJobRun(jobID, rt)
		}
	}
}

// fileGroup collects the active items that share one Locator.File.
type fileGroup struct {
	file  string
	items []inventory.Item
}

// groupByFile buckets items by Locator.File, in the order each file is first seen.
func groupByFile(items []inventory.Item) []fileGroup {
	var groups []fileGroup
	index := make(map[string]int)
	for _, item := range items {
		idx, ok := index[item.Locator.File]
		if !ok {
			idx = len(groups)
			index[item.Locator.File] = idx
			groups = append(groups, fileGroup{file: item.Locator.File})
		}
		groups[idx].items = append(groups[idx].items, item)
	}
	return groups
}

// renderAggregateLanes registers one lane per file, carrying the union of runs of every item it contains.
func renderAggregateLanes(timeline *render.Timeline, activeItems []inventory.Item, scheduler cronx.Scheduler, startTime, endTime time.Time, runCount, top int) {
	groups := groupByFile(activeItems)
	timeline.SetCollapsed(len(activeItems), len(groups))

	// Each group's runs are computed once so --top can rank by total run count without a second pass.
	groupRuns := make([][]time.Time, len(groups))
	for gi, group := range groups {
		for _, item := range group.items {
			groupRuns[gi] = append(groupRuns[gi], computeItemRuns(scheduler, item, startTime, endTime, runCount)...)
		}
	}

	keepIdx, hidden := capLanesByRunCount(len(groups), groupRuns, top, func(i int) (string, int, string) {
		return groups[i].file, 0, ""
	})
	if hidden > 0 {
		timeline.SetHiddenLanes(hidden, top)
	}

	for i, idx := range keepIdx {
		group := groups[idx]
		jobID := fmt.Sprintf("file-%d", i)
		label := group.file
		if label == "" {
			label = "(no file)"
		}
		description := fmt.Sprintf("%d schedules", len(group.items))

		timeline.SetJobInfo(jobID, "", description, label)
		timeline.SetJobLocator(jobID, group.file, 0, "")
		timeline.SetJobAggregateCount(jobID, len(group.items))
		for _, rt := range groupRuns[idx] {
			timeline.AddJobRun(jobID, rt)
		}
	}
}

// stdoutTTY reports whether stdout is attached to a terminal.
func stdoutTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// resolveWidth picks the timeline width: flag, then terminal size, then $COLUMNS, then 80.
func resolveWidth(flagW int, tty bool) int {
	if flagW > 0 {
		return flagW
	}
	if tty {
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
			return w
		}
	}
	if c, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && c > 0 {
		return c
	}
	return 80
}

// resolveColor decides whether color output is on for the given --color mode.
func resolveColor(mode string, tty bool) (bool, error) {
	switch mode {
	case "always":
		return true, nil
	case "never":
		return false, nil
	case "auto":
		return tty && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb", nil
	}
	return false, fmt.Errorf("invalid --color value: %s (must be auto, always, or never)", mode)
}

// absPath resolves a path for display, falling back to the input when resolution fails.
func absPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

// userCrontabSource names whose crontab is being read.
func userCrontabSource() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return "crontab for " + u.Username
	}
	return "user crontab"
}

// timelineSourceLabel names what resolveItems read, for the window line
// SetSource prints above the chart.
func timelineSourceLabel(flags *inputFlags) string {
	switch flags.classify() {
	case sourceInventory:
		if flags.inventory == "-" {
			return "inventory from stdin"
		}
		return absPath(flags.inventory)
	case sourceFile:
		return absPath(flags.file)
	case sourceStdin:
		return "stdin"
	default:
		return userCrontabSource()
	}
}

// laneLabel derives an item's lane label: command basename, else file base name, else path's last segment,
// else "job-<line>".
func laneLabel(item inventory.Item) string {
	if f := strings.Fields(item.Command); len(f) > 0 {
		return filepath.Base(f[0])
	}
	if item.Locator.File != "" {
		base := filepath.Base(item.Locator.File)
		if name := strings.TrimSuffix(base, filepath.Ext(base)); name != "" {
			return name
		}
	}
	if item.Locator.Path != "" {
		segments := strings.Split(item.Locator.Path, ".")
		if last := segments[len(segments)-1]; last != "" {
			return last
		}
	}
	return fmt.Sprintf("job-%d", item.Locator.Line)
}

// exportTimeline exports the timeline to a file (text format only, JSON handled separately)
func (tc *TimelineCommand) exportTimeline(textOutput string, timeline *render.Timeline) error {
	file, err := os.Create(tc.export)
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	if _, err := file.WriteString(textOutput); err != nil {
		return fmt.Errorf("failed to write text output: %w", err)
	}

	return nil
}
