package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hzerrad/cronkit/internal/crontab"
	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/human"
	"github.com/hzerrad/cronkit/internal/render"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// TimelineCommand wraps cobra.Command with timeline-specific functionality
type TimelineCommand struct {
	*cobra.Command
	file         string
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

	tc.Command.Flags().StringVarP(&tc.file, "file", "f", "", "Path to crontab file (defaults to user's crontab if not specified)")
	tc.Command.Flags().BoolVarP(&tc.json, "json", "j", false, "Output in JSON format")
	tc.Command.Flags().StringVar(&tc.view, "view", "day", "Timeline view type: 'day' (24 hours) or 'hour' (60 minutes, default: 'day')")
	tc.Command.Flags().StringVar(&tc.from, "from", "", "Start time for timeline (RFC3339 format, defaults to current time)")
	tc.Command.Flags().IntVar(&tc.width, "width", 0, "Terminal width (0 = auto-detect, defaults to 80 if detection fails)")
	tc.Command.Flags().StringVar(&tc.timezone, "timezone", "", "Timezone for timeline (e.g., 'America/New_York', 'UTC', defaults to local timezone)")
	tc.Command.Flags().StringVar(&tc.export, "export", "", "Export timeline to file (format determined by extension: .txt, .json)")
	tc.Command.Flags().BoolVar(&tc.showOverlaps, "show-overlaps", false, "Show detailed overlap information in output")
	tc.Command.Flags().StringVar(&tc.color, "color", "auto", "Color output: auto, always, or never")
	tc.Command.Flags().BoolVar(&tc.ascii, "ascii", false, "Plain ASCII glyphs")

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

	// Parse jobs
	var jobs []*crontab.Job

	if len(args) > 0 {
		// Single expression provided
		expression := args[0]
		parser := cronx.NewParserWithLocale(locale)
		_, err = parser.Parse(expression)
		if err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}

		// Create a job for the expression
		jobs = []*crontab.Job{
			{
				LineNumber: 0,
				Expression: expression,
				Command:    "(single expression)",
				Valid:      true,
			},
		}
	} else {
		// Read from file or user crontab
		reader := crontab.NewReader()
		if tc.file != "" {
			jobs, err = reader.ReadFile(tc.file)
			if err != nil {
				return fmt.Errorf("failed to read crontab file: %w", err)
			}
			timeline.SetSource(absPath(tc.file), "")
		} else {
			jobs, err = reader.ReadUser()
			if err != nil {
				return fmt.Errorf("failed to read user crontab: %w", err)
			}
			timeline.SetSource(userCrontabSource(), "")
		}
	}

	// Process jobs and add runs to timeline
	parser := cronx.NewParserWithLocale(locale)
	humanizer := human.NewHumanizer()
	scheduler := cronx.NewScheduler()

	// Pre-count lane label basenames so duplicates can be disambiguated with a :LINE suffix.
	labelCounts := make(map[string]int)
	if len(args) == 0 {
		for _, job := range jobs {
			if job.Valid {
				labelCounts[laneLabel(job)]++
			}
		}
	}

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

	for _, job := range jobs {
		if !job.Valid {
			continue
		}

		// Parse expression
		schedule, err := parser.Parse(job.Expression)
		if err != nil {
			continue // Skip invalid expressions
		}

		// Get human description
		description := humanizer.Humanize(schedule)

		// Generate job ID
		jobID := fmt.Sprintf("job-%d", job.LineNumber)
		if job.LineNumber == 0 {
			jobID = fmt.Sprintf("expr-%s", job.Expression)
		}

		// Lane label: the expression itself for the single-expression path (its
		// description goes above the chart), else the command basename deduped
		// with a :LINE suffix when two jobs share one.
		label := job.Expression
		if len(args) == 0 {
			label = laneLabel(job)
			if labelCounts[label] > 1 {
				label = label + ":" + strconv.Itoa(job.LineNumber)
			}
		} else {
			timeline.SetSource(job.Expression, description)
		}

		// Set job info
		timeline.SetJobInfo(jobID, job.Expression, description, label)

		// Calculate next runs
		times, err := scheduler.Next(job.Expression, startTime, runCount)
		if err != nil {
			continue // Skip if we can't calculate runs
		}

		// Add runs that fall within the timeline range
		endTime := startTime.Add(timeRange)
		for _, runTime := range times {
			if runTime.Before(endTime) && !runTime.Before(startTime) {
				timeline.AddJobRun(jobID, runTime)
			}
			// Stop if we've gone past the end time
			if !runTime.Before(endTime) {
				break
			}
		}
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

// laneLabel derives a job's lane label from the basename of its first command token.
func laneLabel(job *crontab.Job) string {
	f := strings.Fields(job.Command)
	if len(f) == 0 {
		return fmt.Sprintf("job-%d", job.LineNumber)
	}
	return filepath.Base(f[0])
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
