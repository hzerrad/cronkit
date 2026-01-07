package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/human"
	"github.com/spf13/cobra"
)

type ExplainCommand struct {
	*cobra.Command
	json bool
}

func newExplainCommand() *ExplainCommand {
	ec := &ExplainCommand{}
	ec.Command = &cobra.Command{
		Args:  cobra.ExactArgs(1),
		Use:   "explain <cron-expression>",
		Short: "Explain a cron expression in plain English",
		RunE:  ec.runExplain,
		Long: `Convert a cron expression to human-readable text.

Supports:
  - Standard 5-field cron expressions
  - Cron aliases (@daily, @hourly, @weekly, @monthly, @yearly)
  - Case-insensitive day and month names

Examples:
  cronkit explain "0 0 * * *"
  cronkit explain "*/15 9-17 * * 1-5"
  cronkit explain "@daily" --json`,
	}

	ec.Flags().BoolVarP(&ec.json, "json", "j", false, "Output in JSON format")
	return ec
}

func init() {
	rootCmd.AddCommand(newExplainCommand().Command)
}

func (ec *ExplainCommand) runExplain(_ *cobra.Command, args []string) error {
	expression := args[0]

	// Parse the cron expression with the specified locale
	parser := cronx.NewParserWithLocale(GetLocale())
	schedule, err := parser.Parse(expression)
	if err != nil {
		return fmt.Errorf("failed to parse expression: %w", err)
	}

	// Humanize the schedule
	humanizer := human.NewHumanizer()
	description := humanizer.Humanize(schedule)

	// Output based on format flag
	if ec.json {
		return ec.outputJSON(expression, description)
	}

	ec.Println(description)
	return nil
}

func (ec *ExplainCommand) outputJSON(expression, description string) error {
	result := map[string]interface{}{
		"expression":  expression,
		"description": description,
		"locale":      GetLocale(),
	}

	encoder := json.NewEncoder(ec.OutOrStdout())
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}
