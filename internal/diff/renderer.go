package diff

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// Renderer interface for different output formats
type Renderer interface {
	Render(w io.Writer, diff *Diff, options *RenderOptions) error
}

// RenderOptions configures how the diff is rendered
type RenderOptions struct {
	ShowUnchanged  bool
	IgnoreComments bool
	IgnoreEnv      bool
}

// TextRenderer renders diff in human-readable text format
type TextRenderer struct{}

// Render renders the diff in text format
func (r *TextRenderer) Render(w io.Writer, diff *Diff, options *RenderOptions) error {
	opts := options
	if opts == nil {
		opts = &RenderOptions{}
	}

	_, _ = fmt.Fprintf(w, "Crontab Diff\n")
	_, _ = fmt.Fprintf(w, "═══════════════════════════════════════════════════════════════\n\n")

	// Show added jobs
	if len(diff.Added) > 0 {
		_, _ = fmt.Fprintf(w, "Added Jobs (%d):\n", len(diff.Added))
		_, _ = fmt.Fprintf(w, "─────────────────────────────────────────────────────────────\n")
		for _, change := range diff.Added {
			_, _ = fmt.Fprintf(w, "+ %s  %s\n", change.NewJob.Expression, change.NewJob.Command)
			if change.NewJob.Comment != "" {
				_, _ = fmt.Fprintf(w, "  # %s\n", change.NewJob.Comment)
			}
		}
		_, _ = fmt.Fprintf(w, "\n")
	}

	// Show removed jobs
	if len(diff.Removed) > 0 {
		_, _ = fmt.Fprintf(w, "Removed Jobs (%d):\n", len(diff.Removed))
		_, _ = fmt.Fprintf(w, "─────────────────────────────────────────────────────────────\n")
		for _, change := range diff.Removed {
			_, _ = fmt.Fprintf(w, "- %s  %s\n", change.OldJob.Expression, change.OldJob.Command)
			if change.OldJob.Comment != "" {
				_, _ = fmt.Fprintf(w, "  # %s\n", change.OldJob.Comment)
			}
		}
		_, _ = fmt.Fprintf(w, "\n")
	}

	// Show modified jobs
	if len(diff.Modified) > 0 {
		_, _ = fmt.Fprintf(w, "Modified Jobs (%d):\n", len(diff.Modified))
		_, _ = fmt.Fprintf(w, "─────────────────────────────────────────────────────────────\n")
		for _, change := range diff.Modified {
			_, _ = fmt.Fprintf(w, "~ %s  %s\n", change.NewJob.Expression, change.NewJob.Command)
			_, _ = fmt.Fprintf(w, "  Fields changed: %s\n", strings.Join(change.FieldsChanged, ", "))

			// Show old values for changed fields
			for _, field := range change.FieldsChanged {
				switch field {
				case "expression":
					_, _ = fmt.Fprintf(w, "    Old expression: %s\n", change.OldJob.Expression)
					_, _ = fmt.Fprintf(w, "    New expression: %s\n", change.NewJob.Expression)
				case "command":
					_, _ = fmt.Fprintf(w, "    Old command: %s\n", change.OldJob.Command)
					_, _ = fmt.Fprintf(w, "    New command: %s\n", change.NewJob.Command)
				case "comment":
					_, _ = fmt.Fprintf(w, "    Old comment: %s\n", change.OldJob.Comment)
					_, _ = fmt.Fprintf(w, "    New comment: %s\n", change.NewJob.Comment)
				}
			}
		}
		_, _ = fmt.Fprintf(w, "\n")
	}

	// Show unchanged jobs (if requested)
	if opts.ShowUnchanged && len(diff.Unchanged) > 0 {
		_, _ = fmt.Fprintf(w, "Unchanged Jobs (%d):\n", len(diff.Unchanged))
		_, _ = fmt.Fprintf(w, "─────────────────────────────────────────────────────────────\n")
		for _, change := range diff.Unchanged {
			_, _ = fmt.Fprintf(w, "  %s  %s\n", change.NewJob.Expression, change.NewJob.Command)
		}
		_, _ = fmt.Fprintf(w, "\n")
	}

	// Show environment variable changes
	if !opts.IgnoreEnv && len(diff.EnvChanges) > 0 {
		_, _ = fmt.Fprintf(w, "Environment Variable Changes (%d):\n", len(diff.EnvChanges))
		_, _ = fmt.Fprintf(w, "─────────────────────────────────────────────────────────────\n")
		for _, envChange := range diff.EnvChanges {
			switch envChange.Type {
			case ChangeTypeAdded:
				_, _ = fmt.Fprintf(w, "+ %s=%s\n", envChange.Key, envChange.NewValue)
			case ChangeTypeRemoved:
				_, _ = fmt.Fprintf(w, "- %s=%s\n", envChange.Key, envChange.OldValue)
			case ChangeTypeModified:
				_, _ = fmt.Fprintf(w, "~ %s\n", envChange.Key)
				_, _ = fmt.Fprintf(w, "    Old: %s\n", envChange.OldValue)
				_, _ = fmt.Fprintf(w, "    New: %s\n", envChange.NewValue)
			}
		}
		_, _ = fmt.Fprintf(w, "\n")
	}

	// Show comment changes
	if !opts.IgnoreComments && len(diff.CommentChanges) > 0 {
		_, _ = fmt.Fprintf(w, "Comment Changes (%d):\n", len(diff.CommentChanges))
		_, _ = fmt.Fprintf(w, "─────────────────────────────────────────────────────────────\n")
		for _, commentChange := range diff.CommentChanges {
			switch commentChange.Type {
			case ChangeTypeAdded:
				_, _ = fmt.Fprintf(w, "+ %s\n", commentChange.NewLine)
			case ChangeTypeRemoved:
				_, _ = fmt.Fprintf(w, "- %s\n", commentChange.OldLine)
			}
		}
		_, _ = fmt.Fprintf(w, "\n")
	}

	// Summary
	totalChanges := len(diff.Added) + len(diff.Removed) + len(diff.Modified)
	if totalChanges == 0 {
		_, _ = fmt.Fprintf(w, "No changes detected.\n")
	} else {
		_, _ = fmt.Fprintf(w, "Summary: %d added, %d removed, %d modified\n",
			len(diff.Added), len(diff.Removed), len(diff.Modified))
	}

	return nil
}

// JSONRenderer renders diff in JSON format
type JSONRenderer struct{}

// Render renders the diff in JSON format
func (r *JSONRenderer) Render(w io.Writer, diff *Diff, options *RenderOptions) error {
	opts := options
	if opts == nil {
		opts = &RenderOptions{}
	}

	type JobChangeJSON struct {
		Type          string   `json:"type"`
		Expression    string   `json:"expression,omitempty"`
		Command       string   `json:"command,omitempty"`
		Comment       string   `json:"comment,omitempty"`
		LineNumber    int      `json:"lineNumber,omitempty"`
		FieldsChanged []string `json:"fieldsChanged,omitempty"`
		OldExpression string   `json:"oldExpression,omitempty"`
		OldCommand    string   `json:"oldCommand,omitempty"`
		OldComment    string   `json:"oldComment,omitempty"`
		OldLineNumber int      `json:"oldLineNumber,omitempty"`
	}

	type EnvChangeJSON struct {
		Type     string `json:"type"`
		Key      string `json:"key"`
		OldValue string `json:"oldValue,omitempty"`
		NewValue string `json:"newValue,omitempty"`
	}

	type CommentChangeJSON struct {
		Type    string `json:"type"`
		OldLine string `json:"oldLine,omitempty"`
		NewLine string `json:"newLine,omitempty"`
	}

	type DiffJSON struct {
		Added          []JobChangeJSON     `json:"added"`
		Removed        []JobChangeJSON     `json:"removed"`
		Modified       []JobChangeJSON     `json:"modified"`
		Unchanged      []JobChangeJSON     `json:"unchanged,omitempty"`
		EnvChanges     []EnvChangeJSON     `json:"envChanges,omitempty"`
		CommentChanges []CommentChangeJSON `json:"commentChanges,omitempty"`
		Summary        map[string]int      `json:"summary"`
		GeneratedAt    string              `json:"generatedAt"`
	}

	result := DiffJSON{
		Added:          []JobChangeJSON{},
		Removed:        []JobChangeJSON{},
		Modified:       []JobChangeJSON{},
		Unchanged:      []JobChangeJSON{},
		EnvChanges:     []EnvChangeJSON{},
		CommentChanges: []CommentChangeJSON{},
		Summary: map[string]int{
			"added":    len(diff.Added),
			"removed":  len(diff.Removed),
			"modified": len(diff.Modified),
		},
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Convert added jobs
	for _, change := range diff.Added {
		result.Added = append(result.Added, JobChangeJSON{
			Type:       "added",
			Expression: change.NewJob.Expression,
			Command:    change.NewJob.Command,
			Comment:    change.NewJob.Comment,
			LineNumber: change.NewJob.LineNumber,
		})
	}

	// Convert removed jobs
	for _, change := range diff.Removed {
		result.Removed = append(result.Removed, JobChangeJSON{
			Type:       "removed",
			Expression: change.OldJob.Expression,
			Command:    change.OldJob.Command,
			Comment:    change.OldJob.Comment,
			LineNumber: change.OldJob.LineNumber,
		})
	}

	// Convert modified jobs
	for _, change := range diff.Modified {
		result.Modified = append(result.Modified, JobChangeJSON{
			Type:          "modified",
			Expression:    change.NewJob.Expression,
			Command:       change.NewJob.Command,
			Comment:       change.NewJob.Comment,
			LineNumber:    change.NewJob.LineNumber,
			FieldsChanged: change.FieldsChanged,
			OldExpression: change.OldJob.Expression,
			OldCommand:    change.OldJob.Command,
			OldComment:    change.OldJob.Comment,
			OldLineNumber: change.OldJob.LineNumber,
		})
	}

	// Convert unchanged jobs (if requested)
	if opts.ShowUnchanged {
		for _, change := range diff.Unchanged {
			result.Unchanged = append(result.Unchanged, JobChangeJSON{
				Type:       "unchanged",
				Expression: change.NewJob.Expression,
				Command:    change.NewJob.Command,
				Comment:    change.NewJob.Comment,
				LineNumber: change.NewJob.LineNumber,
			})
		}
	}

	// Convert env changes
	if !opts.IgnoreEnv {
		for _, envChange := range diff.EnvChanges {
			envJSON := EnvChangeJSON{
				Key: envChange.Key,
			}
			switch envChange.Type {
			case ChangeTypeAdded:
				envJSON.Type = "added"
				envJSON.NewValue = envChange.NewValue
			case ChangeTypeRemoved:
				envJSON.Type = "removed"
				envJSON.OldValue = envChange.OldValue
			case ChangeTypeModified:
				envJSON.Type = "modified"
				envJSON.OldValue = envChange.OldValue
				envJSON.NewValue = envChange.NewValue
			}
			result.EnvChanges = append(result.EnvChanges, envJSON)
		}
	}

	// Convert comment changes
	if !opts.IgnoreComments {
		for _, commentChange := range diff.CommentChanges {
			commentJSON := CommentChangeJSON{}
			switch commentChange.Type {
			case ChangeTypeAdded:
				commentJSON.Type = "added"
				commentJSON.NewLine = commentChange.NewLine
			case ChangeTypeRemoved:
				commentJSON.Type = "removed"
				commentJSON.OldLine = commentChange.OldLine
			}
			result.CommentChanges = append(result.CommentChanges, commentJSON)
		}
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// UnifiedRenderer renders diff in unified diff format (for git integration)
type UnifiedRenderer struct{}

// Render renders the diff in unified diff format
func (r *UnifiedRenderer) Render(w io.Writer, diff *Diff, options *RenderOptions) error {
	_ = options // Not used in unified format

	_, _ = fmt.Fprintf(w, "--- old crontab\n")
	_, _ = fmt.Fprintf(w, "+++ new crontab\n")
	_, _ = fmt.Fprintf(w, "@@ -1 +1 @@\n")

	// Show removed jobs
	for _, change := range diff.Removed {
		_, _ = fmt.Fprintf(w, "-%s %s", change.OldJob.Expression, change.OldJob.Command)
		if change.OldJob.Comment != "" {
			_, _ = fmt.Fprintf(w, " # %s", change.OldJob.Comment)
		}
		_, _ = fmt.Fprintf(w, "\n")
	}

	// Show added jobs
	for _, change := range diff.Added {
		_, _ = fmt.Fprintf(w, "+%s %s", change.NewJob.Expression, change.NewJob.Command)
		if change.NewJob.Comment != "" {
			_, _ = fmt.Fprintf(w, " # %s", change.NewJob.Comment)
		}
		_, _ = fmt.Fprintf(w, "\n")
	}

	// Show modified jobs (as remove + add)
	for _, change := range diff.Modified {
		_, _ = fmt.Fprintf(w, "-%s %s", change.OldJob.Expression, change.OldJob.Command)
		if change.OldJob.Comment != "" {
			_, _ = fmt.Fprintf(w, " # %s", change.OldJob.Comment)
		}
		_, _ = fmt.Fprintf(w, "\n")
		_, _ = fmt.Fprintf(w, "+%s %s", change.NewJob.Expression, change.NewJob.Command)
		if change.NewJob.Comment != "" {
			_, _ = fmt.Fprintf(w, " # %s", change.NewJob.Comment)
		}
		_, _ = fmt.Fprintf(w, "\n")
	}

	return nil
}

// NewRenderer creates a renderer based on format name
func NewRenderer(format string) (Renderer, error) {
	switch format {
	case "text", "":
		return &TextRenderer{}, nil
	case "json":
		return &JSONRenderer{}, nil
	case "unified":
		return &UnifiedRenderer{}, nil
	default:
		return nil, fmt.Errorf("unknown format: %s (supported: text, json, unified)", format)
	}
}
