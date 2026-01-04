package check

// Diagnostic code constants
const (
	// CodeDOMDOWConflict indicates both day-of-month and day-of-week are specified
	CodeDOMDOWConflict = "CRON-001"
	// CodeEmptySchedule indicates the schedule never runs
	CodeEmptySchedule = "CRON-002"
	// CodeParseError indicates a cron expression parsing error
	CodeParseError = "CRON-003"
	// CodeFileReadError indicates an error reading a crontab file
	CodeFileReadError = "CRON-004"
	// CodeInvalidStructure indicates invalid crontab structure
	CodeInvalidStructure = "CRON-005"
	// CodeRedundantPattern indicates a redundant step pattern (e.g., */1 instead of *)
	CodeRedundantPattern = "CRON-006"
	// CodeExcessiveRuns indicates a schedule that runs too frequently (exceeds threshold)
	CodeExcessiveRuns = "CRON-007"
	// CodeMissingAbsolutePath indicates command may not use absolute path
	CodeMissingAbsolutePath = "CRON-008"
	// CodeMissingRedirection indicates command may not redirect stdout/stderr
	CodeMissingRedirection = "CRON-009"
	// CodePercentCharacter indicates command contains % character (cron newline semantics)
	CodePercentCharacter = "CRON-010"
	// CodeQuotingIssue indicates potential quoting/escaping issues in command
	CodeQuotingIssue = "CRON-011"
	// CodeOverlapDetected indicates multiple jobs running at the same time
	CodeOverlapDetected = "CRON-012"
)

// GetCodeSeverity returns the severity level for a given diagnostic code
func GetCodeSeverity(code string) Severity {
	switch code {
	case CodeDOMDOWConflict, CodeRedundantPattern, CodeExcessiveRuns, CodePercentCharacter, CodeQuotingIssue, CodeOverlapDetected:
		return SeverityWarn
	case CodeMissingAbsolutePath, CodeMissingRedirection:
		return SeverityInfo
	case CodeEmptySchedule, CodeParseError, CodeFileReadError, CodeInvalidStructure:
		return SeverityError
	default:
		return SeverityError // Default to error for unknown codes
	}
}

// GetCodeHint returns a hint/suggestion for fixing an issue with the given code
func GetCodeHint(code string) string {
	switch code {
	case CodeDOMDOWConflict:
		return "Consider using only day-of-month OR day-of-week, not both. Cron uses OR logic (runs if either condition is met)."
	case CodeEmptySchedule:
		return "This expression never runs. Check for conflicting constraints or impossible date combinations."
	case CodeParseError:
		return "Fix the syntax error in the cron expression. Ensure all 5 fields are present and valid."
	case CodeFileReadError:
		return "Check that the file exists and is readable. Verify file permissions."
	case CodeInvalidStructure:
		return "Ensure the crontab file follows the correct format with valid cron expressions."
	case CodeRedundantPattern:
		return "Use '*' instead of '*/1' for better readability. They are functionally equivalent."
	case CodeExcessiveRuns:
		return "This schedule runs very frequently. Consider if this is necessary, as it may impact system resources."
	case CodeMissingAbsolutePath:
		return "Consider using absolute paths for commands to avoid PATH-related issues. Example: /usr/bin/command instead of command"
	case CodeMissingRedirection:
		return "Consider redirecting stdout and stderr to log files to capture output and errors. Example: command > /var/log/command.log 2>&1"
	case CodePercentCharacter:
		return "The '%' character in cron commands is interpreted as a newline. Escape it as '\\%' if you need a literal % character."
	case CodeQuotingIssue:
		return "Check that all quotes are properly closed and escaped. Use single quotes for literal strings, double quotes for variable expansion."
	case CodeOverlapDetected:
		return "Multiple jobs are scheduled to run at the same time. This may cause resource contention. Consider adjusting schedules to distribute load."
	default:
		return ""
	}
}
