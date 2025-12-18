package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/hzerrad/cronic/internal/cronx"
	"github.com/hzerrad/cronic/internal/human"
	"github.com/spf13/cobra"
)

var explainJSON bool

var explainCmd = &cobra.Command{
	Use:   "explain <cron-expression>",
	Short: "Explain a cron expression in plain English",
	Long: `Convert a cron expression to human-readable text.

Supports:
  - Standard 5-field cron expressions
  - Cron aliases (@daily, @hourly, @weekly, @monthly, @yearly)
  - Case-insensitive day and month names

Examples:
  cronic explain "0 0 * * *"
  cronic explain "*/15 9-17 * * 1-5"
  cronic explain "@daily" --json`,
	Args: cobra.ExactArgs(1),
	RunE: runExplain,
}

func init() {
	rootCmd.AddCommand(explainCmd)
	explainCmd.Flags().BoolVarP(&explainJSON, "json", "j", false, "Output as JSON")
}

// newExplainCommand creates a fresh explain command instance for testing
// This avoids state pollution between tests by creating isolated command instances
func newExplainCommand() *cobra.Command {
	var testJSON bool

	cmd := &cobra.Command{
		Use:   "explain <cron-expression>",
		Short: "Explain a cron expression in plain English",
		Long: `Convert a cron expression to human-readable text.

Supports:
  - Standard 5-field cron expressions
  - Cron aliases (@daily, @hourly, @weekly, @monthly, @yearly)
  - Case-insensitive day and month names

Examples:
  cronic explain "0 0 * * *"
  cronic explain "*/15 9-17 * * 1-5"
  cronic explain "@daily" --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			expression := args[0]

			// Parse the cron expression
			parser := cronx.NewParser()
			schedule, err := parser.Parse(expression)
			if err != nil {
				return fmt.Errorf("failed to parse expression: %w", err)
			}

			// Humanize the schedule
			humanizer := human.NewHumanizer()
			description := humanizer.Humanize(schedule)

			// Output based on format flag
			if testJSON {
				return outputJSON(cmd, expression, description)
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), description)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&testJSON, "json", "j", false, "Output as JSON")

	return cmd
}

func runExplain(cmd *cobra.Command, args []string) error {
	expression := args[0]

	// Parse the cron expression
	parser := cronx.NewParser()
	schedule, err := parser.Parse(expression)
	if err != nil {
		return fmt.Errorf("failed to parse expression: %w", err)
	}

	// Humanize the schedule
	humanizer := human.NewHumanizer()
	description := humanizer.Humanize(schedule)

	// Output based on format flag
	if explainJSON {
		return outputJSON(cmd, expression, description)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), description)
	return nil
}

func outputJSON(cmd *cobra.Command, expression, description string) error {
	result := map[string]string{
		"expression":  expression,
		"description": description,
	}

	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}
