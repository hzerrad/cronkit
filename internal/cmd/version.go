package cmd

import (
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of cronic",
	Long:  `All software has versions. This is cronic's.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("cronic %s\n", rootCmd.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
