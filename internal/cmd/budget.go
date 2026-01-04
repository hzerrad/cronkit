package cmd

import (
	"fmt"
	"time"

	"github.com/hzerrad/cronic/internal/budget"
	"github.com/hzerrad/cronic/internal/crontab"
	"github.com/hzerrad/cronic/internal/cronx"
	"github.com/spf13/cobra"
)

type BudgetCommand struct {
	*cobra.Command
	file          string
	stdin         bool
	maxConcurrent int
	window        string
	enforce       bool
	json          bool
	verbose       bool
}

func newBudgetCommand() *BudgetCommand {
	bc := &BudgetCommand{}
	bc.Command = &cobra.Command{
		Use:   "budget",
		Short: "Analyze crontab against concurrency budgets",
		Long: `Analyze crontab jobs against concurrency budgets to prevent resource exhaustion.

This command checks if the crontab violates concurrency limits by analyzing
how many jobs run simultaneously within a given time window.

Examples:
  cronic budget --file /etc/crontab --max-concurrent 10 --window 1m
  cronic budget --file crontab.txt --max-concurrent 50 --window 1h --json
  cronic budget --file jobs.cron --max-concurrent 10 --window 1m --enforce
  cronic budget --stdin --max-concurrent 5 --window 1h --verbose`,
		RunE: bc.runBudget,
	}

	bc.Flags().StringVarP(&bc.file, "file", "f", "", "Path to crontab file (defaults to user's crontab if not specified)")
	bc.Flags().BoolVar(&bc.stdin, "stdin", false, "Read crontab from standard input")
	bc.Flags().IntVar(&bc.maxConcurrent, "max-concurrent", 0, "Maximum concurrent jobs allowed (required)")
	bc.Flags().StringVar(&bc.window, "window", "", "Time window for budget (e.g., 1m, 1h, 24h) (required)")
	bc.Flags().BoolVar(&bc.enforce, "enforce", false, "Exit with error code if budget is violated (default: report only)")
	bc.Flags().BoolVarP(&bc.json, "json", "j", false, "Output in JSON format")
	bc.Flags().BoolVarP(&bc.verbose, "verbose", "v", false, "Show detailed violation information")

	return bc
}

func init() {
	rootCmd.AddCommand(newBudgetCommand().Command)
}

func (bc *BudgetCommand) runBudget(_ *cobra.Command, args []string) error {
	// Validate required flags
	if bc.maxConcurrent <= 0 {
		return fmt.Errorf("--max-concurrent must be greater than 0")
	}
	if bc.window == "" {
		return fmt.Errorf("--window is required (e.g., 1m, 1h, 24h)")
	}

	// Parse time window
	timeWindow, err := time.ParseDuration(bc.window)
	if err != nil {
		return fmt.Errorf("invalid --window duration: %w (expected format: 1m, 1h, 24h, etc.)", err)
	}

	// Read crontab
	reader := crontab.NewReader()
	var jobs []*crontab.Job

	if bc.stdin {
		jobs, err = reader.ReadStdin()
		if err != nil {
			return fmt.Errorf("failed to read crontab from stdin: %w", err)
		}
	} else if bc.file != "" {
		jobs, err = reader.ReadFile(bc.file)
		if err != nil {
			return fmt.Errorf("failed to read crontab file: %w", err)
		}
	} else {
		// Read user crontab
		jobs, err = reader.ReadUser()
		if err != nil {
			return fmt.Errorf("failed to read user crontab: %w", err)
		}
	}

	// Create budget
	budgets := []budget.Budget{
		{
			MaxConcurrent: bc.maxConcurrent,
			TimeWindow:    timeWindow,
			Name:          fmt.Sprintf("max-%d-per-%s", bc.maxConcurrent, bc.window),
		},
	}

	// Analyze budget
	scheduler := cronx.NewScheduler()
	parser := cronx.NewParser()
	report, err := budget.AnalyzeBudget(jobs, budgets, scheduler, parser)
	if err != nil {
		return fmt.Errorf("failed to analyze budget: %w", err)
	}

	// Render output
	format := "text"
	if bc.json {
		format = "json"
	}

	renderer, err := budget.NewRenderer(format, bc.verbose)
	if err != nil {
		return fmt.Errorf("failed to create renderer: %w", err)
	}

	output := bc.OutOrStdout()
	if err := renderer.Render(output, report); err != nil {
		return fmt.Errorf("failed to render budget report: %w", err)
	}

	// Exit code handling
	if bc.enforce && !report.Passed {
		return fmt.Errorf("budget violation detected")
	}

	return nil
}
