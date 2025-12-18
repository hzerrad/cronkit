package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var exampleCmd = &cobra.Command{
	Use:   "example",
	Short: "An example command",
	Long: `This is an example command to demonstrate how to add new commands to cronic.

You can remove this file or modify it to create your own commands.`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		if name != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Hello, %s!\n", name)
		} else {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Hello from cronic!")
		}
	},
}

func init() {
	rootCmd.AddCommand(exampleCmd)
	exampleCmd.Flags().StringP("name", "n", "", "Name to greet")
}
