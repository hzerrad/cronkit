package doc

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

const (
	maxCommandLengthDoc  = 50
	maxCommandDisplayDoc = 47 // for truncation
)

// Renderer interface for different output formats
type Renderer interface {
	Render(doc *Document, w io.Writer) error
}

// MarkdownRenderer renders documents in Markdown format
type MarkdownRenderer struct{}

// Render renders a document as Markdown
func (r *MarkdownRenderer) Render(doc *Document, w io.Writer) error {
	// Write header
	_, _ = fmt.Fprintf(w, "# %s\n\n", doc.Title)
	_, _ = fmt.Fprintf(w, "**Generated:** %s\n", doc.GeneratedAt.Format(time.RFC3339))
	_, _ = fmt.Fprintf(w, "**Source:** %s\n\n", doc.Source)

	// Write metadata
	_, _ = fmt.Fprintf(w, "## Summary\n\n")
	_, _ = fmt.Fprintf(w, "- Total Jobs: %d\n", doc.Metadata.TotalJobs)
	_, _ = fmt.Fprintf(w, "- Valid Jobs: %d\n", doc.Metadata.ValidJobs)
	_, _ = fmt.Fprintf(w, "- Invalid Jobs: %d\n\n", doc.Metadata.InvalidJobs)

	// Write jobs table
	_, _ = fmt.Fprintf(w, "## Jobs\n\n")
	_, _ = fmt.Fprintf(w, "| Line | Expression | Description | Command |\n")
	_, _ = fmt.Fprintf(w, "|------|------------|------------|----------|\n")

	for _, job := range doc.Jobs {
		// Truncate command for table display
		command := job.Command
		if len(command) > maxCommandLengthDoc {
			command = command[:maxCommandDisplayDoc] + "..."
		}
		_, _ = fmt.Fprintf(w, "| %d | `%s` | %s | `%s` |\n",
			job.LineNumber, job.Expression, job.Description, command)
	}

	_, _ = fmt.Fprintf(w, "\n")

	// Write detailed job information
	for _, job := range doc.Jobs {
		_, _ = fmt.Fprintf(w, "### Job at Line %d\n\n", job.LineNumber)
		_, _ = fmt.Fprintf(w, "**Expression:** `%s`\n\n", job.Expression)
		_, _ = fmt.Fprintf(w, "**Description:** %s\n\n", job.Description)
		_, _ = fmt.Fprintf(w, "**Command:**\n```bash\n%s\n```\n\n", job.Command)

		if job.Comment != "" {
			_, _ = fmt.Fprintf(w, "**Comment:** %s\n\n", job.Comment)
		}

		if len(job.NextRuns) > 0 {
			_, _ = fmt.Fprintf(w, "**Next Runs:**\n\n")
			for i, t := range job.NextRuns {
				if i >= 10 { // Limit to 10 next runs
					break
				}
				_, _ = fmt.Fprintf(w, "- %s\n", t.Format(time.RFC3339))
			}
			_, _ = fmt.Fprintf(w, "\n")
		}

		if len(job.Warnings) > 0 {
			_, _ = fmt.Fprintf(w, "**Warnings:**\n\n")
			for _, warning := range job.Warnings {
				_, _ = fmt.Fprintf(w, "- ⚠️ %s\n", warning)
			}
			_, _ = fmt.Fprintf(w, "\n")
		}

		if job.Stats != nil {
			_, _ = fmt.Fprintf(w, "**Statistics:**\n\n")
			_, _ = fmt.Fprintf(w, "- Runs per day: %d\n", job.Stats.RunsPerDay)
			_, _ = fmt.Fprintf(w, "- Runs per hour: %d\n\n", job.Stats.RunsPerHour)
		}
	}

	return nil
}

// HTMLRenderer renders documents in HTML format
type HTMLRenderer struct{}

// Render renders a document as HTML
func (r *HTMLRenderer) Render(doc *Document, w io.Writer) error {
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>%s</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 40px; }
        h1 { color: #333; }
        h2 { color: #666; margin-top: 30px; }
        table { border-collapse: collapse; width: 100%%; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        code { background-color: #f4f4f4; padding: 2px 4px; border-radius: 3px; }
        pre { background-color: #f4f4f4; padding: 10px; border-radius: 5px; overflow-x: auto; }
        .warning { color: #ff9800; }
    </style>
</head>
<body>
`, doc.Title)

	_, _ = fmt.Fprintf(w, "<h1>%s</h1>\n", doc.Title)
	_, _ = fmt.Fprintf(w, "<p><strong>Generated:</strong> %s</p>\n", doc.GeneratedAt.Format(time.RFC3339))
	_, _ = fmt.Fprintf(w, "<p><strong>Source:</strong> %s</p>\n", doc.Source)

	_, _ = fmt.Fprintf(w, "<h2>Summary</h2>\n<ul>\n")
	_, _ = fmt.Fprintf(w, "<li>Total Jobs: %d</li>\n", doc.Metadata.TotalJobs)
	_, _ = fmt.Fprintf(w, "<li>Valid Jobs: %d</li>\n", doc.Metadata.ValidJobs)
	_, _ = fmt.Fprintf(w, "<li>Invalid Jobs: %d</li>\n</ul>\n", doc.Metadata.InvalidJobs)

	_, _ = fmt.Fprintf(w, "<h2>Jobs</h2>\n<table>\n<thead>\n<tr><th>Line</th><th>Expression</th><th>Description</th><th>Command</th></tr>\n</thead>\n<tbody>\n")
	for _, job := range doc.Jobs {
		command := job.Command
		if len(command) > maxCommandLengthDoc {
			command = command[:maxCommandDisplayDoc] + "..."
		}
		_, _ = fmt.Fprintf(w, "<tr><td>%d</td><td><code>%s</code></td><td>%s</td><td><code>%s</code></td></tr>\n",
			job.LineNumber, job.Expression, job.Description, command)
	}
	_, _ = fmt.Fprintf(w, "</tbody>\n</table>\n")

	for _, job := range doc.Jobs {
		_, _ = fmt.Fprintf(w, "<h3>Job at Line %d</h3>\n", job.LineNumber)
		_, _ = fmt.Fprintf(w, "<p><strong>Expression:</strong> <code>%s</code></p>\n", job.Expression)
		_, _ = fmt.Fprintf(w, "<p><strong>Description:</strong> %s</p>\n", job.Description)
		_, _ = fmt.Fprintf(w, "<p><strong>Command:</strong></p><pre>%s</pre>\n", job.Command)

		if job.Comment != "" {
			_, _ = fmt.Fprintf(w, "<p><strong>Comment:</strong> %s</p>\n", job.Comment)
		}

		if len(job.NextRuns) > 0 {
			_, _ = fmt.Fprintf(w, "<p><strong>Next Runs:</strong></p><ul>\n")
			for i, t := range job.NextRuns {
				if i >= 10 {
					break
				}
				_, _ = fmt.Fprintf(w, "<li>%s</li>\n", t.Format(time.RFC3339))
			}
			_, _ = fmt.Fprintf(w, "</ul>\n")
		}

		if len(job.Warnings) > 0 {
			_, _ = fmt.Fprintf(w, "<p><strong>Warnings:</strong></p><ul class=\"warning\">\n")
			for _, warning := range job.Warnings {
				_, _ = fmt.Fprintf(w, "<li>⚠️ %s</li>\n", warning)
			}
			_, _ = fmt.Fprintf(w, "</ul>\n")
		}

		if job.Stats != nil {
			_, _ = fmt.Fprintf(w, "<p><strong>Statistics:</strong></p><ul>\n")
			_, _ = fmt.Fprintf(w, "<li>Runs per day: %d</li>\n", job.Stats.RunsPerDay)
			_, _ = fmt.Fprintf(w, "<li>Runs per hour: %d</li>\n</ul>\n", job.Stats.RunsPerHour)
		}
	}

	_, _ = fmt.Fprintf(w, "</body>\n</html>\n")
	return nil
}

// JSONRenderer renders documents in JSON format
type JSONRenderer struct{}

// Render renders a document as JSON
func (r *JSONRenderer) Render(doc *Document, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(doc)
}
