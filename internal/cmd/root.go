package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "cronic",
	Short: "cronic - a CLI application",
	Long: `cronic is a command-line interface application built with Go.

Add your application description here.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	Run: func(cmd *cobra.Command, args []string) {
		// Default behavior when no subcommand is specified
		cmd.Help()
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags can be added here
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cronic.yaml)")

	// Local flags
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
