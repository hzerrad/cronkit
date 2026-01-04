package cmd

import (
	"bufio"
	"fmt"

	"github.com/hzerrad/cronic/internal/crontab"
	"github.com/hzerrad/cronic/internal/diff"
	"github.com/spf13/cobra"
)

type DiffCommand struct {
	*cobra.Command
	oldFile        string
	newFile        string
	oldStdin       bool
	newStdin       bool
	format         string
	json           bool
	ignoreComments bool
	ignoreEnv      bool
	showUnchanged  bool
}

func newDiffCommand() *DiffCommand {
	dc := &DiffCommand{}
	dc.Command = &cobra.Command{
		Use:   "diff [old-file] [new-file]",
		Short: "Compare crontabs semantically",
		Long: `Compare two crontabs semantically, showing what actually changed.

This command performs semantic comparison (not just line-by-line), identifying:
  - Jobs added, removed, or modified
  - Schedule changes (expression modifications)
  - Command changes
  - Comment changes
  - Environment variable changes

Examples:
  cronic diff old.cron new.cron
  cronic diff --old-file old.cron --new-file new.cron --json
  cronic diff --old-stdin --new-file new.cron
  cronic diff old.cron new.cron --format unified`,
		RunE: dc.runDiff,
		Args: cobra.MaximumNArgs(2),
	}

	dc.Flags().StringVar(&dc.oldFile, "old-file", "", "Path to old crontab file")
	dc.Flags().StringVar(&dc.newFile, "new-file", "", "Path to new crontab file")
	dc.Flags().BoolVar(&dc.oldStdin, "old-stdin", false, "Read old crontab from standard input")
	dc.Flags().BoolVar(&dc.newStdin, "new-stdin", false, "Read new crontab from standard input")
	dc.Flags().StringVar(&dc.format, "format", "text", "Output format: 'text' (default), 'json', or 'unified'")
	dc.Flags().BoolVarP(&dc.json, "json", "j", false, "Output in JSON format (shorthand for --format json)")
	dc.Flags().BoolVar(&dc.ignoreComments, "ignore-comments", false, "Ignore comment-only changes")
	dc.Flags().BoolVar(&dc.ignoreEnv, "ignore-env", false, "Ignore environment variable changes")
	dc.Flags().BoolVar(&dc.showUnchanged, "show-unchanged", false, "Show unchanged jobs (default: false)")

	return dc
}

func init() {
	rootCmd.AddCommand(newDiffCommand().Command)
}

func (dc *DiffCommand) runDiff(_ *cobra.Command, args []string) error {
	reader := crontab.NewReader()

	// Determine old crontab source
	var oldEntries []*crontab.Entry
	var err error

	if dc.oldStdin {
		// Read from stdin manually to support command input
		inputReader := dc.InOrStdin()
		scanner := bufio.NewScanner(inputReader)
		lineNumber := 0
		oldEntries = make([]*crontab.Entry, 0)
		for scanner.Scan() {
			lineNumber++
			line := scanner.Text()
			entry := crontab.ParseLine(line, lineNumber)
			oldEntries = append(oldEntries, entry)
		}
		if err = scanner.Err(); err != nil {
			return fmt.Errorf("failed to read old crontab from stdin: %w", err)
		}
	} else if dc.oldFile != "" {
		oldEntries, err = reader.ParseFile(dc.oldFile)
		if err != nil {
			return fmt.Errorf("failed to read old crontab file: %w", err)
		}
	} else if len(args) >= 1 {
		oldEntries, err = reader.ParseFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read old crontab file: %w", err)
		}
	} else {
		return fmt.Errorf("must specify old crontab source (--old-file, --old-stdin, or positional argument)")
	}

	// Determine new crontab source
	var newEntries []*crontab.Entry

	if dc.newStdin {
		// Read from stdin manually to support command input
		inputReader := dc.InOrStdin()
		scanner := bufio.NewScanner(inputReader)
		lineNumber := 0
		newEntries = make([]*crontab.Entry, 0)
		for scanner.Scan() {
			lineNumber++
			line := scanner.Text()
			entry := crontab.ParseLine(line, lineNumber)
			newEntries = append(newEntries, entry)
		}
		if err = scanner.Err(); err != nil {
			return fmt.Errorf("failed to read new crontab from stdin: %w", err)
		}
	} else if dc.newFile != "" {
		newEntries, err = reader.ParseFile(dc.newFile)
		if err != nil {
			return fmt.Errorf("failed to read new crontab file: %w", err)
		}
	} else if len(args) >= 2 {
		newEntries, err = reader.ParseFile(args[1])
		if err != nil {
			return fmt.Errorf("failed to read new crontab file: %w", err)
		}
	} else if len(args) == 1 && !dc.oldStdin && dc.oldFile == "" {
		// If only one arg and old wasn't specified, treat it as new
		newEntries, err = reader.ParseFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read new crontab file: %w", err)
		}
	} else {
		return fmt.Errorf("must specify new crontab source (--new-file, --new-stdin, or positional argument)")
	}

	// Perform semantic diff
	result := diff.CompareCrontabs(oldEntries, newEntries)

	// Determine output format
	outputFormat := dc.format
	if dc.json {
		outputFormat = "json"
	}

	// Create renderer
	renderer, err := diff.NewRenderer(outputFormat)
	if err != nil {
		return fmt.Errorf("failed to create renderer: %w", err)
	}

	// Render output
	options := &diff.RenderOptions{
		ShowUnchanged:  dc.showUnchanged,
		IgnoreComments: dc.ignoreComments,
		IgnoreEnv:      dc.ignoreEnv,
	}

	output := dc.OutOrStdout()
	if err := renderer.Render(output, result, options); err != nil {
		return fmt.Errorf("failed to render diff: %w", err)
	}

	return nil
}
