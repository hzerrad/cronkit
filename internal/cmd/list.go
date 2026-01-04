package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hzerrad/cronic/internal/crontab"
	"github.com/hzerrad/cronic/internal/cronx"
	"github.com/hzerrad/cronic/internal/human"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	listFile  string
	listAll   bool
	listJSON  bool
	listStdin bool
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List and summarize cron jobs from a crontab file or user's crontab",
	Long: `Parse and display cron jobs from a crontab file or the current user's crontab.

Examples:
  cronic list                        # List current user's cron jobs
  cronic list --file /etc/crontab    # List jobs from specific file
  cronic list --all                  # Include comments and environment variables
  cronic list --json                 # Output as JSON
  cronic list --file sample.cron --json > jobs.json`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&listFile, "file", "f", "", "Path to crontab file (defaults to user's crontab if not specified)")
	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "Show all entries including comments and environment variables")
	listCmd.Flags().BoolVarP(&listJSON, "json", "j", false, "Output in JSON format")
	listCmd.Flags().BoolVar(&listStdin, "stdin", false, "Read crontab from standard input (automatic if stdin is not a terminal)")
}

// newListCommand creates a new list command for testing
func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List and summarize cron jobs from a crontab file or user's crontab",
		Long: `Parse and display cron jobs from a crontab file or the current user's crontab.

Examples:
  cronic list                        # List current user's cron jobs
  cronic list --file /etc/crontab    # List jobs from specific file
  cronic list --all                  # Include comments and environment variables
  cronic list --json                 # Output as JSON
  cronic list --file sample.cron --json > jobs.json`,
		RunE: runList,
	}

	cmd.Flags().StringVarP(&listFile, "file", "f", "", "Path to crontab file (defaults to user's crontab if not specified)")
	cmd.Flags().BoolVarP(&listAll, "all", "a", false, "Show all entries including comments and environment variables")
	cmd.Flags().BoolVarP(&listJSON, "json", "j", false, "Output in JSON format")
	cmd.Flags().BoolVar(&listStdin, "stdin", false, "Read crontab from standard input (automatic if stdin is not a terminal)")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	reader := crontab.NewReader()

	var jobs []*crontab.Job
	var entries []*crontab.Entry
	var err error

	// Priority: --file > --stdin > user crontab
	if listFile != "" {
		if listAll {
			entries, err = reader.ParseFile(listFile)
		} else {
			jobs, err = reader.ReadFile(listFile)
		}
		if err != nil {
			return fmt.Errorf("failed to read crontab file %s: %w", listFile, err)
		}
	} else if listStdin {
		// Read from stdin
		if listAll {
			entries, err = reader.ParseStdin()
		} else {
			jobs, err = reader.ReadStdin()
		}
		if err != nil {
			return fmt.Errorf("failed to read crontab from stdin: %w", err)
		}
	} else {
		// Check if stdin is available (not a terminal)
		if isStdinAvailable() {
			// Read from stdin automatically
			if listAll {
				entries, err = reader.ParseStdin()
			} else {
				jobs, err = reader.ReadStdin()
			}
			if err != nil {
				return fmt.Errorf("failed to read crontab from stdin: %w", err)
			}
		} else {
			// Fall back to user's crontab
			jobs, err = reader.ReadUser()
			if err != nil {
				return fmt.Errorf("failed to read user crontab: %w", err)
			}
		}
	}

	if err != nil {
		return fmt.Errorf("failed to read crontab: %w", err)
	}

	// Handle --all mode
	if listAll && entries != nil {
		return outputAllEntries(cmd, entries)
	}

	// Handle empty job list
	if len(jobs) == 0 {
		if listJSON {
			return outputJSON(cmd, map[string]interface{}{"jobs": []interface{}{}})
		}
		cmd.Println("No cron jobs found")
		return nil
	}

	// Output results
	if listJSON {
		return outputJobsJSON(cmd, jobs)
	}

	return outputJobsTable(cmd, jobs)
}

func outputJobsJSON(cmd *cobra.Command, jobs []*crontab.Job) error {
	type jobOutput struct {
		LineNumber  int    `json:"lineNumber"`
		Expression  string `json:"expression"`
		Command     string `json:"command"`
		Comment     string `json:"comment,omitempty"`
		Description string `json:"description,omitempty"`
	}

	output := make([]jobOutput, 0, len(jobs))
	parser := cronx.NewParserWithLocale(GetLocale())

	for _, job := range jobs {
		jo := jobOutput{
			LineNumber: job.LineNumber,
			Expression: job.Expression,
			Command:    job.Command,
			Comment:    job.Comment,
		}

		// Try to parse and humanize the expression
		schedule, err := parser.Parse(job.Expression)
		if err == nil {
			humanizer := human.NewHumanizer()
			jo.Description = humanizer.Humanize(schedule)
		}

		output = append(output, jo)
	}

	return outputJSON(cmd, map[string]interface{}{
		"jobs":   output,
		"locale": GetLocale(),
	})
}

func outputAllEntries(cmd *cobra.Command, entries []*crontab.Entry) error {
	if listJSON {
		type entryOutput struct {
			LineNumber int    `json:"lineNumber"`
			Type       string `json:"type"`
			Raw        string `json:"raw"`
			Job        *struct {
				Expression string `json:"expression"`
				Command    string `json:"command"`
				Comment    string `json:"comment,omitempty"`
			} `json:"job,omitempty"`
		}

		output := make([]entryOutput, 0, len(entries))
		for _, entry := range entries {
			eo := entryOutput{
				LineNumber: entry.LineNumber,
				Type:       entryTypeString(entry.Type),
				Raw:        entry.Raw,
			}

			if entry.Type == crontab.EntryTypeJob && entry.Job != nil {
				eo.Job = &struct {
					Expression string `json:"expression"`
					Command    string `json:"command"`
					Comment    string `json:"comment,omitempty"`
				}{
					Expression: entry.Job.Expression,
					Command:    entry.Job.Command,
					Comment:    entry.Job.Comment,
				}
			}

			output = append(output, eo)
		}

		return outputJSON(cmd, map[string]interface{}{
			"entries": output,
			"locale":  GetLocale(),
		})
	}

	// Table output for all entries
	for _, entry := range entries {
		typeStr := entryTypeString(entry.Type)
		cmd.Printf("%-4d  %-10s  %s\n", entry.LineNumber, typeStr, entry.Raw)
	}

	return nil
}

func outputJobsTable(cmd *cobra.Command, jobs []*crontab.Job) error {
	parser := cronx.NewParserWithLocale(GetLocale())
	humanizer := human.NewHumanizer()

	// Print header
	cmd.Println("LINE  EXPRESSION        DESCRIPTION                          COMMAND")
	cmd.Println("────  ────────────────  ───────────────────────────────────  ────────────────────────")

	for _, job := range jobs {
		description := ""
		schedule, err := parser.Parse(job.Expression)
		if err == nil {
			description = humanizer.Humanize(schedule)
		} else {
			description = "(invalid)"
		}

		// Truncate long descriptions
		if len(description) > 36 {
			description = description[:33] + "..."
		}

		// Truncate long commands
		command := job.Command
		if len(command) > 40 {
			command = command[:37] + "..."
		}

		cmd.Printf("%-4d  %-16s  %-36s  %s\n", job.LineNumber, job.Expression, description, command)
	}

	return nil
}

func entryTypeString(t crontab.EntryType) string {
	switch t {
	case crontab.EntryTypeJob:
		return "JOB"
	case crontab.EntryTypeComment:
		return "COMMENT"
	case crontab.EntryTypeEnvVar:
		return "ENV"
	case crontab.EntryTypeEmpty:
		return "EMPTY"
	case crontab.EntryTypeInvalid:
		return "INVALID"
	default:
		return "UNKNOWN"
	}
}

func outputJSON(cmd *cobra.Command, data interface{}) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// isStdinAvailable checks if stdin is available (not a terminal)
func isStdinAvailable() bool {
	return !term.IsTerminal(int(os.Stdin.Fd()))
}
