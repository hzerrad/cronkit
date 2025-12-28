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
)

// GetCodeSeverity returns the severity level for a given diagnostic code
func GetCodeSeverity(code string) Severity {
	switch code {
	case CodeDOMDOWConflict:
		return SeverityWarn
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
	default:
		return ""
	}
}
