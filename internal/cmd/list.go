package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hzerrad/cronkit/internal/crontab"
	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/human"
	"github.com/hzerrad/cronkit/internal/inventory"
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
	inputFlags
	all  bool
	json bool
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

	lc.register(lc.Command, true)
	lc.Flags().BoolVarP(&lc.all, "all", "a", false, "Show all entries including comments and environment variables")
	lc.Flags().BoolVarP(&lc.json, "json", "j", false, "Output in JSON format")

	return lc
}

func init() {
	rootCmd.AddCommand(newListCommand().Command)
}

func (lc *ListCommand) runList(_ *cobra.Command, _ []string) error {
	// --all shows comments and env vars, which exist only in a crontab; refuse the combo with --inventory.
	if lc.all && lc.inventory != "" {
		return fmt.Errorf("--all cannot be used with --inventory: comments and environment variables exist only in a crontab")
	}

	source := lc.classify()

	// --all only has raw entries for a file or stdin read; against the user's crontab it's a normal listing.
	if lc.all && (source == sourceFile || source == sourceStdin) {
		entries, err := lc.readAllEntries(source)
		if err != nil {
			return err
		}
		return lc.outputAllEntries(entries)
	}

	items, err := resolveItems(&lc.inputFlags)
	if err != nil {
		switch source {
		case sourceFile:
			return fmt.Errorf("failed to read crontab file %s: %w", lc.file, err)
		case sourceStdin:
			return fmt.Errorf("failed to read crontab from stdin: %w", err)
		case sourceInventory:
			return fmt.Errorf("failed to read inventory: %w", err)
		default:
			return fmt.Errorf("failed to read user crontab: %w", err)
		}
	}

	// Handle empty job list
	if len(items) == 0 {
		if lc.json {
			return lc.outputJSON(map[string]interface{}{"jobs": []interface{}{}})
		}
		lc.Println("No cron jobs found")
		return nil
	}

	// Output results
	if lc.json {
		return lc.outputItemsJSON(items)
	}

	return lc.outputItemsTable(items)
}

// readAllEntries reads the raw entries --all needs for a file or stdin source; ReadUser has no equivalent.
func (lc *ListCommand) readAllEntries(source inputSource) ([]*crontab.Entry, error) {
	reader := crontab.NewReader()

	if source == sourceFile {
		entries, err := reader.ParseFile(lc.file)
		if err != nil {
			return nil, fmt.Errorf("failed to read crontab file %s: %w", lc.file, err)
		}
		return entries, nil
	}

	entries, err := reader.ParseStdin()
	if err != nil {
		return nil, fmt.Errorf("failed to read crontab from stdin: %w", err)
	}
	return entries, nil
}

func (lc *ListCommand) outputItemsJSON(items []inventory.Item) error {
	type jobOutput struct {
		LineNumber  int    `json:"lineNumber"`
		File        string `json:"file,omitempty"`
		Expression  string `json:"expression"`
		Command     string `json:"command"`
		Comment     string `json:"comment,omitempty"`
		Description string `json:"description,omitempty"`
	}

	// File is shown only once it disambiguates, keeping single-source output unchanged from before.
	multiFile := inputSpansFiles(items)

	output := make([]jobOutput, 0, len(items))
	parser := cronx.NewParserWithLocale(GetLocale())

	for _, item := range items {
		jo := jobOutput{
			LineNumber: item.Locator.Line,
			Expression: item.Expression,
			Command:    item.Command,
			Comment:    item.Comment,
		}
		if multiFile {
			jo.File = item.Locator.File
		}

		// Try to parse and humanize the expression
		schedule, err := parser.Parse(item.Expression)
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

// listFileColumnWidth is how many runes the FILE column gets before truncateFieldLeft elides its head.
const listFileColumnWidth = 24

func (lc *ListCommand) outputItemsTable(items []inventory.Item) error {
	parser := cronx.NewParserWithLocale(GetLocale())
	humanizer := human.NewHumanizer()

	// FILE is shown only once the input spans more than one file, where a bare LINE number is ambiguous.
	multiFile := inputSpansFiles(items)

	if multiFile {
		lc.Println("FILE                      LINE  EXPRESSION        DESCRIPTION                          COMMAND")
		lc.Println("────────────────────────  ────  ────────────────  ───────────────────────────────────  ────────────────────────")
	} else {
		lc.Println("LINE  EXPRESSION        DESCRIPTION                          COMMAND")
		lc.Println("────  ────────────────  ───────────────────────────────────  ────────────────────────")
	}

	for _, item := range items {
		description := ""
		schedule, err := parser.Parse(item.Expression)
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
		command := item.Command
		if len(command) > maxCommandLength {
			command = command[:maxCommandDisplay] + "..."
		}

		if multiFile {
			file := truncateFieldLeft(item.Locator.File, listFileColumnWidth)
			lc.Printf("%-*s  %-4d  %-16s  %-36s  %s\n", listFileColumnWidth, file, item.Locator.Line, item.Expression, description, command)
		} else {
			lc.Printf("%-4d  %-16s  %-36s  %s\n", item.Locator.Line, item.Expression, description, command)
		}
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

// isStdinAvailable reports whether stdin is available (not a terminal); a var so tests can override it.
var isStdinAvailable = func() bool {
	return !term.IsTerminal(int(os.Stdin.Fd()))
}
