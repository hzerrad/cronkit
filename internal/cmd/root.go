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
	Use:   "cronic",
	Short: "cronic - a CLI application",
	Long: `cronic is a command-line interface application built with Go.

Add your application description here.`,
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
	rootCmd.PersistentFlags().StringVar(&locale, "locale", "en", "Locale for parsing day/month names (e.g., en, fr, es)")
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
