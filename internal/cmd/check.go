package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hzerrad/cronic/internal/check"
	"github.com/hzerrad/cronic/internal/crontab"
	"github.com/spf13/cobra"
)

type CheckCommand struct {
	*cobra.Command
	file    string
	json    bool
	verbose bool
	failOn  string
}

func newCheckCommand() *CheckCommand {
	cc := &CheckCommand{}
	cc.Command = &cobra.Command{
		Use:   "check [cron-expression]",
		Short: "Validate cron expressions and crontab files",
		Long: `Validate cron expressions and crontab files for errors and potential issues.

This command checks for:
  - Invalid cron expressions
  - DOM/DOW conflicts (when both day-of-month and day-of-week are specified)
  - Empty schedules (expressions that never run)
  - Invalid crontab file structure

Examples:
  cronic check "0 0 * * *"              # Validate a single expression
  cronic check --file /etc/crontab       # Validate a crontab file
  cronic check                           # Validate user's crontab
  cronic check "0 0 1 * 1" --verbose    # Show warnings (DOM/DOW conflicts)
  cronic check --file sample.cron --json # JSON output`,
		RunE: cc.runCheck,
		Args: cobra.MaximumNArgs(1),
	}

	cc.Flags().StringVarP(&cc.file, "file", "f", "", "Path to crontab file (defaults to user's crontab)")
	cc.Flags().BoolVarP(&cc.json, "json", "j", false, "Output as JSON")
	cc.Flags().BoolVarP(&cc.verbose, "verbose", "v", false, "Show warnings (DOM/DOW conflicts) as well as errors")
	cc.Flags().StringVar(&cc.failOn, "fail-on", "error", "Severity level to fail on: error (default), warn, or info")

	return cc
}

func init() {
	rootCmd.AddCommand(newCheckCommand().Command)
}

func (cc *CheckCommand) runCheck(_ *cobra.Command, args []string) error {
	// Validate --fail-on flag
	failOnSeverity, err := check.ParseFailOnLevel(cc.failOn)
	if err != nil {
		return fmt.Errorf("invalid --fail-on value: %w", err)
	}

	validator := check.NewValidator(GetLocale())
	reader := crontab.NewReader()

	var result check.ValidationResult

	// Determine what to validate
	if len(args) == 1 {
		// Single expression validation
		result = validator.ValidateExpression(args[0])
	} else if cc.file != "" {
		// File validation
		result = validator.ValidateCrontab(reader, cc.file)
	} else {
		// User crontab validation
		result = validator.ValidateUserCrontab(reader)
	}

	// Output based on format
	if cc.json {
		return cc.outputJSON(result, failOnSeverity)
	}

	return cc.outputText(result, failOnSeverity)
}

func (cc *CheckCommand) outputText(result check.ValidationResult, failOn check.Severity) error {
	// Filter issues based on verbose flag
	issuesToShow := cc.filterIssues(result.Issues)

	// Print summary
	if result.Valid && len(issuesToShow) == 0 {
		_, _ = fmt.Fprintf(cc.OutOrStdout(), "✓ All valid\n")
		if result.TotalJobs > 0 {
			_, _ = fmt.Fprintf(cc.OutOrStdout(), "  %d job(s) validated\n", result.TotalJobs)
		}
		return nil
	}

	// Print error summary
	if !result.Valid {
		_, _ = fmt.Fprintf(cc.OutOrStdout(), "✗ Found %d issue(s)\n", len(issuesToShow))
	} else {
		_, _ = fmt.Fprintf(cc.OutOrStdout(), "⚠ Found %d warning(s)\n", len(issuesToShow))
	}

	if result.TotalJobs > 0 {
		_, _ = fmt.Fprintf(cc.OutOrStdout(), "  Total jobs: %d\n", result.TotalJobs)
		_, _ = fmt.Fprintf(cc.OutOrStdout(), "  Valid: %d\n", result.ValidJobs)
		_, _ = fmt.Fprintf(cc.OutOrStdout(), "  Invalid: %d\n", result.InvalidJobs)
	}

	_, _ = fmt.Fprintln(cc.OutOrStdout())

	// Print issues
	for _, issue := range issuesToShow {
		lineInfo := ""
		if issue.LineNumber > 0 {
			lineInfo = fmt.Sprintf("Line %d: ", issue.LineNumber)
		}

		prefix := ""
		switch issue.Severity {
		case check.SeverityError:
			prefix = "✗ ERROR: "
		case check.SeverityWarn:
			prefix = "⚠ WARNING: "
		case check.SeverityInfo:
			prefix = "ℹ INFO: "
		}

		// Display diagnostic code if available
		codeInfo := ""
		if issue.Code != "" {
			codeInfo = fmt.Sprintf(" [%s]", issue.Code)
		}

		if issue.Expression != "" {
			_, _ = fmt.Fprintf(cc.OutOrStdout(), "%s%s%s%s\n", lineInfo, prefix, issue.Message, codeInfo)
			_, _ = fmt.Fprintf(cc.OutOrStdout(), "  Expression: %s\n", issue.Expression)
		} else {
			_, _ = fmt.Fprintf(cc.OutOrStdout(), "%s%s%s%s\n", lineInfo, prefix, issue.Message, codeInfo)
		}

		// Display hint if available
		if issue.Hint != "" {
			_, _ = fmt.Fprintf(cc.OutOrStdout(), "  Hint: %s\n", issue.Hint)
		}
	}

	// Set exit code based on result and fail-on threshold
	exitCode := calculateExitCode(result, issuesToShow, failOn, cc.verbose)
	if exitCode != 0 {
		osExit(exitCode)
	}

	return nil
}

func (cc *CheckCommand) outputJSON(result check.ValidationResult, failOn check.Severity) error {
	// Filter issues based on verbose flag
	issuesToShow := cc.filterIssues(result.Issues)

	// Convert issues to JSON format with all fields
	jsonIssues := make([]map[string]interface{}, len(issuesToShow))
	for i, issue := range issuesToShow {
		jsonIssue := map[string]interface{}{
			"severity":   issue.Severity.String(),
			"code":       issue.Code,
			"lineNumber": issue.LineNumber,
			"expression": issue.Expression,
			"message":    issue.Message,
			"type":       issue.Type(), // Deprecated: for backward compatibility
		}
		if issue.Hint != "" {
			jsonIssue["hint"] = issue.Hint
		}
		jsonIssues[i] = jsonIssue
	}

	output := map[string]interface{}{
		"valid":       result.Valid && len(issuesToShow) == 0,
		"totalJobs":   result.TotalJobs,
		"validJobs":   result.ValidJobs,
		"invalidJobs": result.InvalidJobs,
		"issues":      jsonIssues,
	}

	encoder := json.NewEncoder(cc.OutOrStdout())
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	// Set exit code based on result and fail-on threshold
	exitCode := calculateExitCode(result, issuesToShow, failOn, cc.verbose)
	if exitCode != 0 {
		osExit(exitCode)
	}

	return nil
}

// osExit is a variable that can be overridden in tests
var osExit = os.Exit

// calculateExitCode determines the appropriate exit code based on validation result,
// issues shown, fail-on threshold, and verbose flag.
// Returns:
//   - 0: No issues, or only issues below the fail-on threshold
//   - 1: Errors present (or configured severity level reached)
//   - 2: Warnings present (only if fail-on is warn or info, or if verbose is set for backward compatibility)
func calculateExitCode(result check.ValidationResult, issuesToShow []check.Issue, failOn check.Severity, verbose bool) int {
	if len(issuesToShow) == 0 {
		return 0
	}

	// Find the highest severity in the issues shown
	highestSeverity := check.SeverityInfo
	for _, issue := range issuesToShow {
		if issue.Severity > highestSeverity {
			highestSeverity = issue.Severity
		}
	}

	// Backward compatibility: if verbose is set and we have warnings, exit with code 2
	// This maintains the old behavior where --verbose would cause exit 2 for warnings
	if verbose && highestSeverity == check.SeverityWarn && failOn == check.SeverityError {
		return 2
	}

	// If highest severity is below the fail-on threshold, return 0
	if highestSeverity < failOn {
		return 0
	}

	// Determine exit code based on highest severity
	switch highestSeverity {
	case check.SeverityError:
		return 1
	case check.SeverityWarn:
		return 2
	case check.SeverityInfo:
		return 2
	default:
		return 0
	}
}

// filterIssues filters issues based on the verbose flag
func (cc *CheckCommand) filterIssues(issues []check.Issue) []check.Issue {
	if cc.verbose {
		return issues
	}
	// Only show errors if not verbose
	filtered := []check.Issue{}
	for _, issue := range issues {
		if issue.Severity == check.SeverityError {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}
