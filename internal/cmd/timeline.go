package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/hzerrad/cronic/internal/crontab"
	"github.com/hzerrad/cronic/internal/cronx"
	"github.com/hzerrad/cronic/internal/human"
	"github.com/hzerrad/cronic/internal/render"
	"github.com/spf13/cobra"
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
  cronic timeline "*/15 * * * *"              # Timeline for single expression
  cronic timeline --file /etc/crontab          # Timeline for crontab file
  cronic timeline "*/5 * * * *" --view hour    # Hour view timeline
  cronic timeline --file jobs.cron --json       # JSON output
  cronic timeline                               # Timeline for user's crontab`,
	}

	tc.Command.Flags().StringVarP(&tc.file, "file", "f", "", "Path to crontab file (defaults to user's crontab if not specified)")
	tc.Command.Flags().BoolVarP(&tc.json, "json", "j", false, "Output in JSON format")
	tc.Command.Flags().StringVar(&tc.view, "view", "day", "Timeline view type: 'day' (24 hours) or 'hour' (60 minutes, default: 'day')")
	tc.Command.Flags().StringVar(&tc.from, "from", "", "Start time for timeline (RFC3339 format, defaults to current time)")
	tc.Command.Flags().IntVar(&tc.width, "width", 0, "Terminal width (0 = auto-detect, defaults to 80 if detection fails)")
	tc.Command.Flags().StringVar(&tc.timezone, "timezone", "", "Timezone for timeline (e.g., 'America/New_York', 'UTC', defaults to local timezone)")
	tc.Command.Flags().StringVar(&tc.export, "export", "", "Export timeline to file (format determined by extension: .txt, .json)")
	tc.Command.Flags().BoolVar(&tc.showOverlaps, "show-overlaps", false, "Show detailed overlap information in output")

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
	width := detectTerminalWidth()
	if tc.width > 0 {
		width = tc.width
	}
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
	var err error

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
		} else {
			jobs, err = reader.ReadUser()
			if err != nil {
				return fmt.Errorf("failed to read user crontab: %w", err)
			}
		}
	}

	// Process jobs and add runs to timeline
	parser := cronx.NewParserWithLocale(locale)
	humanizer := human.NewHumanizer()
	scheduler := cronx.NewScheduler()

	// Calculate how many runs to get based on view
	var runCount int
	var timeRange time.Duration
	if timelineView == render.DayView {
		timeRange = 24 * time.Hour
		runCount = 200 // Enough to cover a day for most schedules
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

		// Set job info
		timeline.SetJobInfo(jobID, job.Expression, description)

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
	output = timeline.Render(tc.showOverlaps)

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

// detectTerminalWidth attempts to detect the terminal width
func detectTerminalWidth() int {
	// Try COLUMNS environment variable first
	if colsStr := os.Getenv("COLUMNS"); colsStr != "" {
		if cols, err := strconv.Atoi(colsStr); err == nil && cols > 0 {
			return cols
		}
	}

	// Try to get terminal size (Unix-like systems)
	// Note: This is a simple implementation; for cross-platform support,
	// we'd need a library like golang.org/x/term
	// For now, default to 80
	return 80
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
