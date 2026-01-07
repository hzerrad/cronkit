package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hzerrad/cronkit/internal/crontab"
	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/human"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	maxDescriptionLength  = 36
	maxCommandLength      = 40
	maxDescriptionDisplay = 33 // for truncation
	maxCommandDisplay     = 37 // for truncation
)

type ListCommand struct {
	*cobra.Command
	file  string
	all   bool
	json  bool
	stdin bool
}

func newListCommand() *ListCommand {
	lc := &ListCommand{}
	lc.Command = &cobra.Command{
		Use:   "list",
		Short: "List and summarize cron jobs from a crontab file or user's crontab",
		Long: `Parse and display cron jobs from a crontab file or the current user's crontab.

Examples:
  cronkit list                        # List current user's cron jobs
  cronkit list --file /etc/crontab    # List jobs from specific file
  cronkit list --all                  # Include comments and environment variables
  cronkit list --json                 # Output as JSON
  cronkit list --file sample.cron --json > jobs.json`,
		RunE: lc.runList,
	}

	lc.Flags().StringVarP(&lc.file, "file", "f", "", "Path to crontab file (defaults to user's crontab if not specified)")
	lc.Flags().BoolVarP(&lc.all, "all", "a", false, "Show all entries including comments and environment variables")
	lc.Flags().BoolVarP(&lc.json, "json", "j", false, "Output in JSON format")
	lc.Flags().BoolVar(&lc.stdin, "stdin", false, "Read crontab from standard input (automatic if stdin is not a terminal)")

	return lc
}

func init() {
	rootCmd.AddCommand(newListCommand().Command)
}

func (lc *ListCommand) runList(_ *cobra.Command, args []string) error {
	reader := crontab.NewReader()

	var jobs []*crontab.Job
	var entries []*crontab.Entry
	var err error

	// Priority: --file > --stdin > user crontab
	if lc.file != "" {
		if lc.all {
			entries, err = reader.ParseFile(lc.file)
		} else {
			jobs, err = reader.ReadFile(lc.file)
		}
		if err != nil {
			return fmt.Errorf("failed to read crontab file %s: %w", lc.file, err)
		}
	} else if lc.stdin {
		// Read from stdin
		if lc.all {
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
			if lc.all {
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
	if lc.all && entries != nil {
		return lc.outputAllEntries(entries)
	}

	// Handle empty job list
	if len(jobs) == 0 {
		if lc.json {
			return lc.outputJSON(map[string]interface{}{"jobs": []interface{}{}})
		}
		lc.Println("No cron jobs found")
		return nil
	}

	// Output results
	if lc.json {
		return lc.outputJobsJSON(jobs)
	}

	return lc.outputJobsTable(jobs)
}

func (lc *ListCommand) outputJobsJSON(jobs []*crontab.Job) error {
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

	return lc.outputJSON(map[string]interface{}{
		"jobs":   output,
		"locale": GetLocale(),
	})
}

func (lc *ListCommand) outputAllEntries(entries []*crontab.Entry) error {
	if lc.json {
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

		return lc.outputJSON(map[string]interface{}{
			"entries": output,
			"locale":  GetLocale(),
		})
	}

	// Table output for all entries
	for _, entry := range entries {
		typeStr := entryTypeString(entry.Type)
		lc.Printf("%-4d  %-10s  %s\n", entry.LineNumber, typeStr, entry.Raw)
	}

	return nil
}

func (lc *ListCommand) outputJobsTable(jobs []*crontab.Job) error {
	parser := cronx.NewParserWithLocale(GetLocale())
	humanizer := human.NewHumanizer()

	// Print header
	lc.Println("LINE  EXPRESSION        DESCRIPTION                          COMMAND")
	lc.Println("────  ────────────────  ───────────────────────────────────  ────────────────────────")

	for _, job := range jobs {
		description := ""
		schedule, err := parser.Parse(job.Expression)
		if err == nil {
			description = humanizer.Humanize(schedule)
		} else {
			description = "(invalid)"
		}

		// Truncate long descriptions
		if len(description) > maxDescriptionLength {
			description = description[:maxDescriptionDisplay] + "..."
		}

		// Truncate long commands
		command := job.Command
		if len(command) > maxCommandLength {
			command = command[:maxCommandDisplay] + "..."
		}

		lc.Printf("%-4d  %-16s  %-36s  %s\n", job.LineNumber, job.Expression, description, command)
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

func (lc *ListCommand) outputJSON(data interface{}) error {
	encoder := json.NewEncoder(lc.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// isStdinAvailable checks if stdin is available (not a terminal)
func isStdinAvailable() bool {
	return !term.IsTerminal(int(os.Stdin.Fd()))
}
