package cmd

import (
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of cronkit",
	Long:  `All software has versions. This is cronkit's.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("cronkit %s\n", rootCmd.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
