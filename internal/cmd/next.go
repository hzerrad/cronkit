package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hzerrad/cronic/internal/cronx"
	"github.com/hzerrad/cronic/internal/human"
	"github.com/spf13/cobra"
)

// NextCommand wraps cobra.Command with next-specific functionality
type NextCommand struct {
	*cobra.Command
	count int
	json  bool
}

// NextRun represents a single scheduled run time
type NextRun struct {
	Number    int    `json:"number"`
	Timestamp string `json:"timestamp"`
	Relative  string `json:"relative"`
}

// NextResult represents the complete output for the next command
type NextResult struct {
	Expression  string    `json:"expression"`
	Description string    `json:"description"`
	Timezone    string    `json:"timezone"`
	NextRuns    []NextRun `json:"next_runs"`
}

func init() {
	rootCmd.AddCommand(newNextCommand().Command)
}

// newNextCommand creates a fresh next command instance for testing
// This avoids state pollution between tests by creating isolated command instances
func newNextCommand() *NextCommand {
	nc := &NextCommand{}
	nc.Command = &cobra.Command{
		Args:  cobra.ExactArgs(1),
		RunE:  nc.runNext,
		Use:   "next <cron-expression>",
		Short: "Show next scheduled run times for a cron expression",
		Long: `Calculate and display the next scheduled run times for a cron expression.

This command helps you understand when a cron job will actually run in the future.
It shows both the exact timestamps and relative times (e.g., "in 2 hours").

Supports:
  - Standard 5-field cron expressions (minute, hour, day-of-month, month, day-of-week)
  - Cron aliases (@daily, @hourly, @weekly, @monthly, @yearly)
  - Custom count with --count flag (1-100 runs, default: 10)
  - JSON output with --json flag for programmatic use

Examples:
  cronic next "*/15 * * * *"              # Next 10 runs (default)
  cronic next "@daily" --count 5          # Next 5 runs
  cronic next "0 9 * * 1-5" -c 3          # Next 3 runs (short flag)
  cronic next "0 14 * * *" --json         # JSON output
  cronic next "*/5 9-17 * * 1-5" -c 20    # Business hours monitoring`,
	}

	nc.Command.Flags().IntVarP(&nc.count, "count", "c", 10, "Number of runs to show (1-100)")
	nc.Command.Flags().BoolVarP(&nc.json, "json", "j", false, "Output as JSON")

	return nc
}

func (nc *NextCommand) runNext(_ *cobra.Command, args []string) error {
	expression := args[0]

	// Validate count range
	if nc.count < 1 {
		return fmt.Errorf("count must be at least 1")
	}
	if nc.count > 100 {
		return fmt.Errorf("count must be at most 100")
	}

	// Create scheduler and calculate next runs
	scheduler := cronx.NewScheduler()
	now := time.Now()

	times, err := scheduler.Next(expression, now, nc.count)
	if err != nil {
		return fmt.Errorf("failed to calculate next runs: %w", err)
	}

	// Get human description with the specified locale
	parser := cronx.NewParserWithLocale(GetLocale())
	schedule, err := parser.Parse(expression)
	if err != nil {
		return fmt.Errorf("failed to parse expression: %w", err)
	}

	humanizer := human.NewHumanizer()
	description := humanizer.Humanize(schedule)

	// Output based on format
	if nc.json {
		return nc.outputNextJSON(expression, description, times, now)
	}

	return nc.outputNextText(expression, description, times)
}

func (nc *NextCommand) outputNextText(expression, description string, times []time.Time) error {
	// Header with count
	runWord := "runs"
	if len(times) == 1 {
		runWord = "run"
	}
	_, _ = fmt.Fprintf(nc.OutOrStdout(), "Next %d %s for \"%s\" (%s):\n\n",
		len(times), runWord, expression, description)

	// List each run with timestamp
	for i, t := range times {
		_, _ = fmt.Fprintf(nc.OutOrStdout(), "%d. %s\n",
			i+1, t.Format("2006-01-02 15:04:05 MST"))
	}

	return nil
}

func (nc *NextCommand) outputNextJSON(expression, description string, times []time.Time, now time.Time) error {
	// Build next runs array
	runs := make([]NextRun, len(times))
	for i, t := range times {
		runs[i] = NextRun{
			Number:    i + 1,
			Timestamp: t.Format(time.RFC3339),
			Relative:  formatRelativeTime(now, t),
		}
	}

	// Build result structure
	result := NextResult{
		Expression:  expression,
		Description: description,
		Timezone:    times[0].Location().String(),
		NextRuns:    runs,
	}

	// Encode as JSON with indentation
	encoder := json.NewEncoder(nc.OutOrStdout())
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// formatRelativeTime converts a duration between two times to a human-readable format.
func formatRelativeTime(from, to time.Time) string {
	duration := to.Sub(from)

	// Less than a minute
	if duration < time.Minute {
		return "in less than a minute"
	}

	// Minutes (less than an hour)
	if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "in 1 minute"
		}
		return fmt.Sprintf("in %d minutes", minutes)
	}

	// Hours (less than a day)
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "in 1 hour"
		}
		return fmt.Sprintf("in %d hours", hours)
	}

	// Days
	days := int(duration.Hours() / 24)
	if days == 1 {
		return "in 1 day"
	}
	return fmt.Sprintf("in %d days", days)
}
