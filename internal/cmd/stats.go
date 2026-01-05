package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hzerrad/cronic/internal/crontab"
	"github.com/hzerrad/cronic/internal/stats"
	"github.com/spf13/cobra"
)

type StatsCommand struct {
	*cobra.Command
	file      string
	stdin     bool
	json      bool
	verbose   bool
	top       int
	aggregate bool
}

func newStatsCommand() *StatsCommand {
	sc := &StatsCommand{}
	sc.Command = &cobra.Command{
		Use:   "stats",
		Short: "Calculate and display crontab statistics",
		Long: `Calculate and display statistics about crontab jobs including:
  - Run frequency metrics (runs per day, per hour)
  - Hour distribution histogram
  - Most/least frequent jobs
  - Collision analysis (busiest hours, quiet windows)

Examples:
  cronic stats --file /etc/crontab
  cronic stats --file crontab.txt --json
  cronic stats --top 10 --verbose`,
		RunE: sc.runStats,
		Args: cobra.NoArgs,
	}

	sc.Flags().StringVarP(&sc.file, "file", "f", "", "Path to crontab file (defaults to user's crontab if not specified)")
	sc.Flags().BoolVar(&sc.stdin, "stdin", false, "Read crontab from standard input")
	sc.Flags().BoolVarP(&sc.json, "json", "j", false, "Output in JSON format")
	sc.Flags().BoolVarP(&sc.verbose, "verbose", "v", false, "Show detailed statistics")
	sc.Flags().IntVar(&sc.top, "top", DefaultStatsTopN, "Number of top items to show (default: 5)")
	sc.Flags().BoolVar(&sc.aggregate, "aggregate", false, "Aggregate statistics from multiple sources")

	return sc
}

func init() {
	rootCmd.AddCommand(newStatsCommand().Command)
}

func (sc *StatsCommand) runStats(_ *cobra.Command, _ []string) error {
	reader := crontab.NewReader()
	calculator := stats.NewCalculator()

	var jobs []*crontab.Job
	var err error

	// Determine input source
	if sc.stdin {
		entries, err := reader.ParseStdin()
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
		jobs = extractJobs(entries)
	} else if sc.file != "" {
		entries, err := reader.ParseFile(sc.file)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		jobs = extractJobs(entries)
	} else {
		jobs, err = reader.ReadUser()
		if err != nil {
			return fmt.Errorf("failed to read user crontab: %w", err)
		}
	}

	if err != nil {
		return err
	}

	// Calculate metrics
	metrics, err := calculator.CalculateMetrics(jobs, stats.OneDay)
	if err != nil {
		return fmt.Errorf("failed to calculate metrics: %w", err)
	}

	// Output
	if sc.json {
		return sc.outputJSON(metrics)
	}

	return sc.outputText(metrics, calculator, jobs)
}

func (sc *StatsCommand) outputJSON(metrics *stats.Metrics) error {
	encoder := json.NewEncoder(sc.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(metrics)
}

func (sc *StatsCommand) outputText(metrics *stats.Metrics, calculator *stats.Calculator, jobs []*crontab.Job) error {
	sc.Println("Crontab Statistics")
	sc.Println(strings.Repeat("=", 50))

	// Summary
	sc.Printf("\nSummary:\n")
	sc.Printf("  Total Jobs: %d\n", len(jobs))
	sc.Printf("  Total Runs per Day: %d\n", metrics.TotalRunsPerDay)
	sc.Printf("  Total Runs per Hour: %d\n", metrics.TotalRunsPerHour)

	// Most frequent jobs
	mostFrequent := calculator.IdentifyMostFrequent(jobs, sc.top)
	if len(mostFrequent) > 0 {
		sc.Printf("\nTop %d Most Frequent Jobs:\n", sc.top)
		for i, freq := range mostFrequent {
			sc.Printf("  %d. %s (%d runs/day, %d runs/hour)\n",
				i+1, freq.Expression, freq.RunsPerDay, freq.RunsPerHour)
		}
	}

	// Hour histogram
	if sc.verbose {
		sc.Printf("\n%s\n", stats.GenerateHistogram(metrics.HourHistogram, stats.DefaultHistogramWidth))
	}

	// Collision stats
	if sc.verbose && len(metrics.Collisions.BusiestHours) > 0 {
		sc.Printf("\nBusiest Hours:\n")
		for i, hour := range metrics.Collisions.BusiestHours {
			if i >= sc.top {
				break
			}
			sc.Printf("  %02d:00 - %d runs\n", hour.Hour, hour.RunCount)
		}
		sc.Printf("\nCollision Frequency: %.2f%%\n", metrics.Collisions.CollisionFrequency)
		sc.Printf("Max Concurrent Jobs: %d\n", metrics.Collisions.MaxConcurrent)
	}

	return nil
}

func extractJobs(entries []*crontab.Entry) []*crontab.Job {
	jobs := make([]*crontab.Job, 0)
	for _, entry := range entries {
		if entry.Type == crontab.EntryTypeJob && entry.Job != nil {
			jobs = append(jobs, entry.Job)
		}
	}
	return jobs
}
