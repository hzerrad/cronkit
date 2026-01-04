package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/hzerrad/cronic/internal/crontab"
	"github.com/hzerrad/cronic/internal/doc"
	"github.com/spf13/cobra"
)

type DocCommand struct {
	*cobra.Command
	file            string
	stdin           bool
	output          string
	format          string
	includeNext     int
	includeWarnings bool
	includeStats    bool
}

func newDocCommand() *DocCommand {
	dc := &DocCommand{}
	dc.Command = &cobra.Command{
		Use:   "doc",
		Short: "Generate documentation from crontab files",
		Long: `Generate human-readable documentation from crontab files.

This command creates markdown, HTML, or JSON documentation that includes:
  - Job summaries with descriptions
  - Schedule details
  - Command information
  - Optional: next runs, warnings, and statistics

Examples:
  cronic doc --file /etc/crontab --output docs.md
  cronic doc --file crontab.txt --format html --output docs.html
  cronic doc --stdin --format json --include-next 5`,
		RunE: dc.runDoc,
		Args: cobra.NoArgs,
	}

	dc.Flags().StringVarP(&dc.file, "file", "f", "", "Path to crontab file (defaults to user's crontab if not specified)")
	dc.Flags().BoolVar(&dc.stdin, "stdin", false, "Read crontab from standard input")
	dc.Flags().StringVarP(&dc.output, "output", "o", "", "Output file path (defaults to stdout)")
	dc.Flags().StringVar(&dc.format, "format", "md", "Output format: 'md' (markdown), 'html', or 'json'")
	dc.Flags().IntVar(&dc.includeNext, "include-next", 0, "Include next N runs per job (0 = disabled)")
	dc.Flags().BoolVar(&dc.includeWarnings, "include-warnings", false, "Include validation warnings")
	dc.Flags().BoolVar(&dc.includeStats, "include-stats", false, "Include frequency statistics")

	return dc
}

func init() {
	rootCmd.AddCommand(newDocCommand().Command)
}

func (dc *DocCommand) runDoc(_ *cobra.Command, _ []string) error {
	// Validate format
	if dc.format != "md" && dc.format != "html" && dc.format != "json" {
		return fmt.Errorf("invalid format: %s (must be 'md', 'html', or 'json')", dc.format)
	}

	// Create generator
	generator := doc.NewGenerator(GetLocale())
	reader := crontab.NewReader()

	var entries []*crontab.Entry
	var source string
	var err error

	// Determine input source
	if dc.stdin {
		// Read from command's input (for testability) or os.Stdin
		inputReader := dc.InOrStdin()
		if inputReader != os.Stdin {
			// Read from command's input stream
			scanner := bufio.NewScanner(inputReader)
			lineNumber := 0
			entries = make([]*crontab.Entry, 0)
			for scanner.Scan() {
				lineNumber++
				line := scanner.Text()
				entry := crontab.ParseLine(line, lineNumber)
				entries = append(entries, entry)
			}
			if err = scanner.Err(); err != nil {
				return fmt.Errorf("failed to read crontab from stdin: %w", err)
			}
		} else {
			entries, err = reader.ParseStdin()
		}
		source = "stdin"
	} else if dc.file != "" {
		entries, err = reader.ParseFile(dc.file)
		source = dc.file
	} else {
		// Read user crontab
		jobs, err := reader.ReadUser()
		if err != nil {
			return fmt.Errorf("failed to read user crontab: %w", err)
		}
		entries = make([]*crontab.Entry, 0, len(jobs))
		for _, job := range jobs {
			entries = append(entries, &crontab.Entry{
				Type:       crontab.EntryTypeJob,
				LineNumber: job.LineNumber,
				Job:        job,
			})
		}
		source = "user crontab"
	}

	if err != nil {
		return fmt.Errorf("failed to read crontab: %w", err)
	}

	// Generate document
	options := doc.GenerateOptions{
		IncludeNext:     dc.includeNext,
		IncludeWarnings: dc.includeWarnings,
		IncludeStats:    dc.includeStats,
	}

	document, err := generator.GenerateDocument(entries, source, options)
	if err != nil {
		return fmt.Errorf("failed to generate document: %w", err)
	}

	// Select renderer
	var renderer doc.Renderer
	switch dc.format {
	case "md":
		renderer = &doc.MarkdownRenderer{}
	case "html":
		renderer = &doc.HTMLRenderer{}
	case "json":
		renderer = &doc.JSONRenderer{}
	}

	// Determine output destination
	var output io.Writer
	if dc.output != "" {
		file, err := os.Create(dc.output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() {
			_ = file.Close()
		}()
		output = file
	} else {
		// Use command's output writer for testability
		output = dc.OutOrStdout()
	}

	// Render document
	if err := renderer.Render(document, output); err != nil {
		return fmt.Errorf("failed to render document: %w", err)
	}

	return nil
}
