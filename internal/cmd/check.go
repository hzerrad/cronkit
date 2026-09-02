package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/hzerrad/cronkit/internal/check"
	"github.com/spf13/cobra"
)

type CheckCommand struct {
	*cobra.Command
	inputFlags
	json            bool
	verbose         bool
	failOn          string
	groupBy         string
	enableFrequency bool
	maxRunsPerDay   int
	enableHygiene   bool
	warnOnOverlap   bool
	overlapWindow   string
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
  - Redundant patterns (e.g., */1 instead of *)
  - Excessive run counts (configurable threshold)

Examples:
  cronkit check "0 0 * * *"              # Validate a single expression
  cronkit check --file /etc/crontab       # Validate a crontab file
  cronkit check                           # Validate user's crontab
  cronkit check "0 0 1 * 1" --verbose    # Show warnings (DOM/DOW conflicts)
  cronkit check --file sample.cron --json # JSON output`,
		RunE: cc.runCheck,
		Args: cobra.MaximumNArgs(1),
	}

	cc.register(cc.Command, true)
	cc.Flags().BoolVarP(&cc.json, "json", "j", false, "Output in JSON format")
	cc.Flags().BoolVarP(&cc.verbose, "verbose", "v", false, "Show warnings (DOM/DOW conflicts) as well as errors")
	cc.Flags().StringVar(&cc.failOn, "fail-on", "error", "Severity level to fail on: 'error' (default), 'warn', or 'info'")
	cc.Flags().StringVar(&cc.groupBy, "group-by", "none", "Group issues by: 'none' (default), 'severity', 'line', or 'job'")
	cc.Flags().BoolVar(&cc.enableFrequency, "enable-frequency-checks", true, "Enable frequency analysis (redundant patterns, excessive runs)")
	cc.Flags().IntVar(&cc.maxRunsPerDay, "max-runs-per-day", DefaultMaxRunsPerDay, "Threshold for excessive runs warning (default: 1000)")
	cc.Flags().BoolVar(&cc.enableHygiene, "enable-hygiene-checks", false, "Enable command hygiene checks (absolute paths, redirections, %, quoting)")
	cc.Flags().BoolVar(&cc.warnOnOverlap, "warn-on-overlap", false, "Enable overlap warnings (multiple jobs running simultaneously)")
	cc.Flags().StringVar(&cc.overlapWindow, "overlap-window", "24h", "Time window for overlap analysis (default: 24h, e.g., 1h, 24h, 48h)")

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
	validator.SetFrequencyChecks(cc.enableFrequency)
	validator.SetMaxRunsPerDay(cc.maxRunsPerDay)
	validator.SetHygieneChecks(cc.enableHygiene)

	// Parse overlap window duration
	if cc.warnOnOverlap {
		overlapDuration, err := time.ParseDuration(cc.overlapWindow)
		if err != nil {
			return fmt.Errorf("invalid overlap-window duration: %w", err)
		}
		validator.SetOverlapWindow(overlapDuration)
		validator.SetWarnOnOverlap(true)
	}

	var result check.ValidationResult
	// showLocator must come from the input itself, not from which issues validation happened to produce.
	showLocator := false

	// Priority: expression arg > --inventory > --file > --stdin > user crontab
	if len(args) == 1 {
		// Single expression validation
		result = validator.ValidateExpression(args[0])
	} else {
		items, err := resolveItems(&cc.inputFlags)
		if err != nil {
			switch cc.classify() {
			case sourceStdin:
				return fmt.Errorf("failed to read crontab from stdin: %w", err)
			case sourceInventory:
				return fmt.Errorf("failed to read inventory: %w", err)
			case sourceFile:
				// A bad --file is folded into the result as an issue rather than failing the command.
				result = fileReadErrorResult(fmt.Sprintf("Failed to read crontab file: %s", err.Error()))
			default:
				result = fileReadErrorResult(fmt.Sprintf("Failed to read user crontab: %s", err.Error()))
			}
		} else {
			showLocator = inputSpansFiles(items)
			result = validator.ValidateItems(items)
		}
	}

	// Output based on format
	if cc.json {
		return cc.outputJSON(result, failOnSeverity, showLocator)
	}

	return cc.outputText(result, failOnSeverity, showLocator)
}

func (cc *CheckCommand) outputText(result check.ValidationResult, failOn check.Severity, showLocator bool) error {
	// Filter issues based on verbose flag
	issuesToShow := cc.filterIssues(result.Issues)

	// Separate errors, warnings, and info for display
	var errors []check.Issue
	var warnings []check.Issue
	var info []check.Issue

	for _, issue := range issuesToShow {
		switch issue.Severity {
		case check.SeverityError:
			errors = append(errors, issue)
		case check.SeverityWarn:
			warnings = append(warnings, issue)
		case check.SeverityInfo:
			info = append(info, issue)
		}
	}

	// Print summary
	if len(errors) == 0 && len(warnings) == 0 && len(info) == 0 {
		cc.Printf("✓ All valid\n")
		if result.TotalJobs > 0 {
			cc.Printf("  %d job(s) validated\n", result.TotalJobs)
		}
		return nil
	}

	// Print error summary
	if len(errors) > 0 {
		cc.Printf("✗ Found %d error(s)\n", len(errors))
		if len(warnings) > 0 {
			cc.Printf("⚠ Found %d warning(s)\n", len(warnings))
		}
		if len(info) > 0 {
			cc.Printf("ℹ Found %d info message(s)\n", len(info))
		}
	} else if len(warnings) > 0 {
		cc.Printf("⚠ Found %d warning(s)\n", len(warnings))
		if len(info) > 0 {
			cc.Printf("ℹ Found %d info message(s)\n", len(info))
		}
	} else if len(info) > 0 {
		cc.Printf("ℹ Found %d info message(s)\n", len(info))
	}

	if result.TotalJobs > 0 {
		cc.Printf("  Total jobs: %d\n", result.TotalJobs)
		cc.Printf("  Valid: %d\n", result.ValidJobs)
		cc.Printf("  Invalid: %d\n", result.InvalidJobs)
	}

	cc.Println()

	// Print errors (always full format)
	if len(errors) > 0 {
		groupMode := parseGroupBy(cc.groupBy)
		if groupMode == GroupByNone {
			cc.printIssuesFlat(errors, showLocator)
		} else {
			cc.printIssuesGrouped(errors, groupMode, showLocator)
		}
		if len(warnings) > 0 {
			cc.Println()
		}
	}

	// Print warnings (compact if not verbose, full if verbose)
	if len(warnings) > 0 {
		if cc.verbose {
			// Full format for warnings when verbose
			groupMode := parseGroupBy(cc.groupBy)
			if groupMode == GroupByNone {
				cc.printIssuesFlat(warnings, showLocator)
			} else {
				cc.printIssuesGrouped(warnings, groupMode, showLocator)
			}
		} else {
			// Compact format for warnings when not verbose
			cc.printWarningsCompact(warnings, showLocator)
		}
		if len(info) > 0 {
			cc.Println()
		}
	}

	// Print info (only when verbose, always full format)
	if len(info) > 0 && cc.verbose {
		groupMode := parseGroupBy(cc.groupBy)
		if groupMode == GroupByNone {
			cc.printIssuesFlat(info, showLocator)
		} else {
			cc.printIssuesGrouped(info, groupMode, showLocator)
		}
	}

	// Set exit code based on result and fail-on threshold
	exitCode := calculateExitCode(result, issuesToShow, failOn)
	if exitCode != 0 {
		osExit(exitCode)
	}

	return nil
}

func (cc *CheckCommand) outputJSON(result check.ValidationResult, failOn check.Severity, showLocator bool) error {
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
		}
		if issue.Hint != "" {
			jsonIssue["hint"] = issue.Hint
		}
		// locator is additive: only surfaced once more than one file is in play.
		if showLocator && issue.Locator.File != "" {
			jsonIssue["locator"] = issue.Locator
		}
		jsonIssues[i] = jsonIssue
	}

	output := map[string]interface{}{
		"valid":       result.Valid && len(issuesToShow) == 0,
		"totalJobs":   result.TotalJobs,
		"validJobs":   result.ValidJobs,
		"invalidJobs": result.InvalidJobs,
		"issues":      jsonIssues,
		"locale":      GetLocale(),
	}

	encoder := json.NewEncoder(cc.OutOrStdout())
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	// Set exit code based on result and fail-on threshold
	exitCode := calculateExitCode(result, issuesToShow, failOn)
	if exitCode != 0 {
		osExit(exitCode)
	}

	return nil
}

// fileReadErrorResult builds the ValidationResult for a failed --file or user-crontab read.
func fileReadErrorResult(message string) check.ValidationResult {
	return check.ValidationResult{
		Valid: false,
		Issues: []check.Issue{
			{
				Severity: check.SeverityError,
				Code:     check.CodeFileReadError,
				Message:  message,
				Hint:     check.GetCodeHint(check.CodeFileReadError),
			},
		},
	}
}

// osExit is a variable that can be overridden in tests
var osExit = os.Exit

// calculateExitCode returns 0 if issuesToShow is empty or below failOn, else 1 for error, 2 for warn or info.
func calculateExitCode(result check.ValidationResult, issuesToShow []check.Issue, failOn check.Severity) int {
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
	// Always show errors and warnings, filter info only if not verbose
	filtered := []check.Issue{}
	for _, issue := range issues {
		if issue.Severity == check.SeverityError || issue.Severity == check.SeverityWarn {
			filtered = append(filtered, issue)
		} else if issue.Severity == check.SeverityInfo && cc.verbose {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// issueLinePrefix renders the leading "Line N: " or "file:line: " token used by both issue renderers.
func issueLinePrefix(issue check.Issue, showLocator bool) string {
	if label := issueLabel(issue, showLocator); label != "" {
		return label + ": "
	}
	return ""
}

// issueLabel names where an issue was found, naming the path since one line can hold several schedules.
func issueLabel(issue check.Issue, showLocator bool) string {
	if showLocator && issue.Locator.File != "" {
		return issue.Locator.String()
	}
	if issue.LineNumber == 0 {
		return ""
	}
	if issue.Locator.Path != "" {
		return fmt.Sprintf("Line %d (%s)", issue.LineNumber, issue.Locator.Path)
	}
	return fmt.Sprintf("Line %d", issue.LineNumber)
}

// GroupByMode represents the grouping mode for issues
type GroupByMode int

const (
	GroupByNone GroupByMode = iota
	GroupBySeverity
	GroupByLine
	GroupByJob
)

// parseGroupBy parses the group-by string and returns the corresponding mode
func parseGroupBy(groupBy string) GroupByMode {
	switch groupBy {
	case "severity":
		return GroupBySeverity
	case "line":
		return GroupByLine
	case "job":
		return GroupByJob
	default:
		return GroupByNone
	}
}

// groupIssues groups issues by the specified mode
func groupIssues(issues []check.Issue, mode GroupByMode) map[string][]check.Issue {
	groups := make(map[string][]check.Issue)

	switch mode {
	case GroupBySeverity:
		for _, issue := range issues {
			key := issue.Severity.String()
			groups[key] = append(groups[key], issue)
		}
	case GroupByLine:
		for _, issue := range issues {
			if issue.LineNumber == 0 {
				groups["no-line"] = append(groups["no-line"], issue)
				continue
			}
			// LineNumber, not issue.Locator.Line, is the field every Issue carries a line on.
			loc := issue.Locator
			loc.Line = issue.LineNumber
			key := loc.Identity(0, "issue-", "line-")
			groups[key] = append(groups[key], issue)
		}
	case GroupByJob:
		for _, issue := range issues {
			key := issue.Expression
			if key == "" {
				key = "no-expression"
			}
			groups[key] = append(groups[key], issue)
		}
	default:
		// No grouping - return empty map, caller will handle flat display
		return groups
	}

	return groups
}

// getSeverityOrder returns the order for displaying severity groups
func getSeverityOrder() []check.Severity {
	return []check.Severity{
		check.SeverityError,
		check.SeverityWarn,
		check.SeverityInfo,
	}
}

// printIssuesFlat prints issues in a flat list (default behavior)
func (cc *CheckCommand) printIssuesFlat(issues []check.Issue, showLocator bool) {
	for _, issue := range issues {
		cc.printIssue(issue, showLocator)
	}
}

// printIssuesGrouped prints issues grouped by the specified mode
func (cc *CheckCommand) printIssuesGrouped(issues []check.Issue, mode GroupByMode, showLocator bool) {
	groups := groupIssues(issues, mode)

	switch mode {
	case GroupBySeverity:
		// Print groups in severity order: error, warn, info
		for _, severity := range getSeverityOrder() {
			key := severity.String()
			if severityIssues, ok := groups[key]; ok {
				cc.printGroupHeader(fmt.Sprintf("%s Issues", severity.String()), len(severityIssues))
				for _, issue := range severityIssues {
					cc.printIssue(issue, showLocator)
				}
				cc.Println()
			}
		}
	case GroupByLine:
		// Order groups by (line number, file) rather than parsing the map key back apart.
		type lineGroup struct {
			key    string
			line   int
			file   string
			issues []check.Issue
		}
		lgs := make([]lineGroup, 0, len(groups))
		for key, groupIssues := range groups {
			lg := lineGroup{key: key}
			if len(groupIssues) > 0 {
				lg.line = groupIssues[0].LineNumber
				lg.file = groupIssues[0].Locator.File
			}
			lg.issues = groupIssues
			lgs = append(lgs, lg)
		}
		sort.Slice(lgs, func(i, j int) bool {
			if lgs[i].line != lgs[j].line {
				return lgs[i].line < lgs[j].line
			}
			return lgs[i].file < lgs[j].file
		})
		for _, lg := range lgs {
			if lg.line == 0 {
				cc.printGroupHeader("General Issues", len(lg.issues))
			} else {
				cc.printGroupHeader(issueLabel(lg.issues[0], showLocator), len(lg.issues))
			}
			for _, issue := range lg.issues {
				cc.printIssue(issue, showLocator)
			}
			cc.Println()
		}
	case GroupByJob:
		// Sort expressions rather than ranging over the map directly, which iterates in Go's randomized order.
		var expressions []string
		for key := range groups {
			if key != "no-expression" {
				expressions = append(expressions, key)
			}
		}
		sort.Strings(expressions)
		if generalIssues, ok := groups["no-expression"]; ok {
			cc.printGroupHeader("General Issues", len(generalIssues))
			for _, issue := range generalIssues {
				cc.printIssue(issue, showLocator)
			}
			cc.Println()
		}
		for _, key := range expressions {
			groupIssues := groups[key]
			cc.printGroupHeader(fmt.Sprintf("Expression: %s", key), len(groupIssues))
			for _, issue := range groupIssues {
				cc.printIssue(issue, showLocator)
			}
			cc.Println()
		}
	default:
		// GroupByNone or unexpected mode - no-op, caller handles flat display
		// This should not print anything as groupIssues returns an empty map for GroupByNone
	}
}

// printGroupHeader prints a header for a group of issues
func (cc *CheckCommand) printGroupHeader(title string, count int) {
	cc.Printf("━━━ %s (%d issue(s)) ━━━\n", title, count)
}

// printIssue prints a single issue with all its details
func (cc *CheckCommand) printIssue(issue check.Issue, showLocator bool) {
	lineInfo := issueLinePrefix(issue, showLocator)

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
		cc.Printf("  %s%s%s%s\n", lineInfo, prefix, issue.Message, codeInfo)
		cc.Printf("    Expression: %s\n", issue.Expression)
	} else {
		cc.Printf("  %s%s%s%s\n", lineInfo, prefix, issue.Message, codeInfo)
	}

	// Display hint if available
	if issue.Hint != "" {
		cc.Printf("    Hint: %s\n", issue.Hint)
	}
}

// printWarningsCompact prints warnings in a compact format (one line per warning)
func (cc *CheckCommand) printWarningsCompact(warnings []check.Issue, showLocator bool) {
	for _, issue := range warnings {
		lineInfo := issueLinePrefix(issue, showLocator)

		codeInfo := ""
		if issue.Code != "" {
			codeInfo = fmt.Sprintf(" [%s]", issue.Code)
		}

		if issue.Expression != "" {
			cc.Printf("  ⚠ %s%s%s - %s\n", lineInfo, issue.Message, codeInfo, issue.Expression)
		} else {
			cc.Printf("  ⚠ %s%s%s\n", lineInfo, issue.Message, codeInfo)
		}
	}
}
