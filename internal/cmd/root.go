package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	locale  string // Global locale flag for symbol parsing
)

var rootCmd = &cobra.Command{
	Use:   "cronkit",
	Short: "Make cron human again",
	Long: `Cronkit is a command-line tool that makes cron jobs human-readable, auditable, and visual.

It converts confusing cron syntax into plain English, generates upcoming run schedules,
provides ASCII timeline visualizations, and validates crontabs with severity levels
and diagnostic codes.

Features:
  - Explain cron expressions in plain English
  - Show next scheduled run times
  - List and summarize crontab jobs
  - Visualize job schedules with ASCII timelines
  - Validate crontabs with advanced linting
  - Generate documentation (Markdown, HTML, JSON)
  - Calculate statistics and analyze concurrency budgets
  - Compare crontabs semantically

Read-only and safe by design - never executes or modifies crontabs.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	Run: func(cmd *cobra.Command, args []string) {
		// Default behavior when no subcommand is specified
		_ = cmd.Help()
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags - these apply to all subcommands
	rootCmd.PersistentFlags().StringVar(&locale, "locale", "en", "Locale for parsing day/month names (default: 'en', e.g., 'en', 'fr', 'es')")
}

// GetLocale returns the current locale setting
func GetLocale() string {
	if locale == "" {
		return "en" // Default to English
	}
	return locale
}

// SetOutput sets the output and error writers for the root command
func SetOutput(out, err interface{}) {
	if w, ok := out.(interface{ Write([]byte) (int, error) }); ok {
		rootCmd.SetOut(w)
	}
	if w, ok := err.(interface{ Write([]byte) (int, error) }); ok {
		rootCmd.SetErr(w)
	}
}
