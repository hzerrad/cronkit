package budget

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Renderer interface for different output formats
type Renderer interface {
	Render(w io.Writer, report *BudgetReport) error
}

// TextRenderer renders budget report in human-readable text format
type TextRenderer struct {
	Verbose bool
}

// Render renders the budget report in text format
func (r *TextRenderer) Render(w io.Writer, report *BudgetReport) error {
	_, _ = fmt.Fprintf(w, "Budget Analysis\n")
	_, _ = fmt.Fprintf(w, "═══════════════════════════════════════════════════════════════\n\n")

	// Overall status
	if report.Passed {
		_, _ = fmt.Fprintf(w, "✓ All budgets passed\n\n")
	} else {
		_, _ = fmt.Fprintf(w, "✗ Budget violations detected\n\n")
	}

	// Show results for each budget
	for _, budgetResult := range report.Budgets {
		_, _ = fmt.Fprintf(w, "Budget: %s\n", budgetResult.Budget.Name)
		if budgetResult.Budget.Name == "" {
			_, _ = fmt.Fprintf(w, "Budget: Max %d concurrent jobs per %s\n",
				budgetResult.Budget.MaxConcurrent, formatDuration(budgetResult.Budget.TimeWindow))
		}
		_, _ = fmt.Fprintf(w, "  Limit: %d concurrent jobs\n", budgetResult.Budget.MaxConcurrent)
		_, _ = fmt.Fprintf(w, "  Found: %d concurrent jobs (max)\n", budgetResult.MaxFound)

		if budgetResult.Passed {
			_, _ = fmt.Fprintf(w, "  Status: ✓ PASSED\n\n")
		} else {
			_, _ = fmt.Fprintf(w, "  Status: ✗ FAILED\n")
			_, _ = fmt.Fprintf(w, "  Violations: %d\n", len(budgetResult.Violations))

			if r.Verbose && len(budgetResult.Violations) > 0 {
				_, _ = fmt.Fprintf(w, "\n  Violation Details:\n")
				// Show top 10 violations
				maxShow := 10
				if len(budgetResult.Violations) < maxShow {
					maxShow = len(budgetResult.Violations)
				}
				for i := 0; i < maxShow; i++ {
					v := budgetResult.Violations[i]
					_, _ = fmt.Fprintf(w, "    - %s: %d jobs running concurrently\n",
						v.Time.Format("2006-01-02 15:04:05"), v.Count)
					if r.Verbose && len(v.Jobs) > 0 {
						_, _ = fmt.Fprintf(w, "      Jobs: %v\n", v.Jobs)
					}
				}
				if len(budgetResult.Violations) > maxShow {
					_, _ = fmt.Fprintf(w, "    ... and %d more violations\n",
						len(budgetResult.Violations)-maxShow)
				}
			}
			_, _ = fmt.Fprintf(w, "\n")
		}
	}

	// Summary
	totalViolations := len(report.Violations)
	if totalViolations > 0 {
		_, _ = fmt.Fprintf(w, "Summary: %d violation(s) across %d budget(s)\n",
			totalViolations, len(report.Budgets))
	}

	return nil
}

// JSONRenderer renders budget report in JSON format
type JSONRenderer struct{}

// Render renders the budget report in JSON format
func (r *JSONRenderer) Render(w io.Writer, report *BudgetReport) error {
	type BudgetResultJSON struct {
		Name          string      `json:"name"`
		MaxConcurrent int         `json:"maxConcurrent"`
		TimeWindow    string      `json:"timeWindow"`
		MaxFound      int         `json:"maxFound"`
		Passed        bool        `json:"passed"`
		Violations    []Violation `json:"violations"`
	}

	type ViolationJSON struct {
		Time   string   `json:"time"`
		Count  int      `json:"count"`
		Jobs   []string `json:"jobs"`
		Budget struct {
			Name          string `json:"name"`
			MaxConcurrent int    `json:"maxConcurrent"`
			TimeWindow    string `json:"timeWindow"`
		} `json:"budget"`
	}

	type BudgetReportJSON struct {
		Passed      bool               `json:"passed"`
		Budgets     []BudgetResultJSON `json:"budgets"`
		Violations  []ViolationJSON    `json:"violations"`
		GeneratedAt string             `json:"generatedAt"`
	}

	result := BudgetReportJSON{
		Passed:      report.Passed,
		Budgets:     []BudgetResultJSON{},
		Violations:  []ViolationJSON{},
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Convert budgets
	for _, budgetResult := range report.Budgets {
		budgetJSON := BudgetResultJSON{
			Name:          budgetResult.Budget.Name,
			MaxConcurrent: budgetResult.Budget.MaxConcurrent,
			TimeWindow:    formatDuration(budgetResult.Budget.TimeWindow),
			MaxFound:      budgetResult.MaxFound,
			Passed:        budgetResult.Passed,
			Violations:    budgetResult.Violations,
		}
		result.Budgets = append(result.Budgets, budgetJSON)
	}

	// Convert violations
	for _, violation := range report.Violations {
		violationJSON := ViolationJSON{
			Time:  violation.Time.Format(time.RFC3339),
			Count: violation.Count,
			Jobs:  violation.Jobs,
		}
		violationJSON.Budget.Name = violation.Budget.Name
		violationJSON.Budget.MaxConcurrent = violation.Budget.MaxConcurrent
		violationJSON.Budget.TimeWindow = formatDuration(violation.Budget.TimeWindow)
		result.Violations = append(result.Violations, violationJSON)
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// NewRenderer creates a renderer based on format name
func NewRenderer(format string, verbose bool) (Renderer, error) {
	switch format {
	case "text", "":
		return &TextRenderer{Verbose: verbose}, nil
	case "json":
		return &JSONRenderer{}, nil
	default:
		return nil, fmt.Errorf("unknown format: %s (supported: text, json)", format)
	}
}
