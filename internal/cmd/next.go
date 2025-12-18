package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hzerrad/cronic/internal/cronx"
	"github.com/hzerrad/cronic/internal/human"
	"github.com/spf13/cobra"
)

var (
	nextCount int
	nextJSON  bool
)

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

var nextCmd = &cobra.Command{
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
	Args: cobra.ExactArgs(1),
	RunE: runNext,
}

func init() {
	rootCmd.AddCommand(nextCmd)
	nextCmd.Flags().IntVarP(&nextCount, "count", "c", 10, "Number of runs to show (1-100)")
	nextCmd.Flags().BoolVarP(&nextJSON, "json", "j", false, "Output as JSON")
}

func runNext(cmd *cobra.Command, args []string) error {
	expression := args[0]

	// Validate count range
	if nextCount < 1 {
		return fmt.Errorf("count must be at least 1")
	}
	if nextCount > 100 {
		return fmt.Errorf("count must be at most 100")
	}

	// Create scheduler and calculate next runs
	scheduler := cronx.NewScheduler()
	now := time.Now() // Use system timezone

	times, err := scheduler.Next(expression, now, nextCount)
	if err != nil {
		return fmt.Errorf("failed to calculate next runs: %w", err)
	}

	// Get human-readable description
	parser := cronx.NewParser()
	schedule, err := parser.Parse(expression)
	if err != nil {
		return fmt.Errorf("failed to parse expression: %w", err)
	}

	humanizer := human.NewHumanizer()
	description := humanizer.Humanize(schedule)

	// Output based on format
	if nextJSON {
		return outputNextJSON(cmd, expression, description, times, now)
	}

	return outputNextText(cmd, expression, description, times)
}

func outputNextText(cmd *cobra.Command, expression, description string, times []time.Time) error {
	// Header with count
	runWord := "runs"
	if len(times) == 1 {
		runWord = "run"
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Next %d %s for \"%s\" (%s):\n\n",
		len(times), runWord, expression, description)

	// List each run with timestamp
	for i, t := range times {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%d. %s\n",
			i+1, t.Format("2006-01-02 15:04:05 MST"))
	}

	return nil
}

func outputNextJSON(cmd *cobra.Command, expression, description string, times []time.Time, now time.Time) error {
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
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// formatRelativeTime converts a duration between two times to a human-readable format.
// For v0.1.0, supports minutes, hours, and days only.
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
