package check

import (
	"strings"
)

// AnalyzeCommand performs static analysis on a command string for common issues
func AnalyzeCommand(command string) []Issue {
	var issues []Issue

	// Check for absolute path
	if !checkAbsolutePath(command) {
		issues = append(issues, Issue{
			Severity:   SeverityInfo,
			Code:       CodeMissingAbsolutePath,
			LineNumber: 0, // Will be set by caller
			Expression: "",
			Message:    "Command may not use absolute path",
			Hint:       GetCodeHint(CodeMissingAbsolutePath),
		})
	}

	// Check for output redirection
	if !checkOutputRedirection(command) {
		issues = append(issues, Issue{
			Severity:   SeverityInfo,
			Code:       CodeMissingRedirection,
			LineNumber: 0, // Will be set by caller
			Expression: "",
			Message:    "Command may not redirect stdout/stderr",
			Hint:       GetCodeHint(CodeMissingRedirection),
		})
	}

	// Check for % character (cron newline semantics)
	if checkPercentCharacter(command) {
		issues = append(issues, Issue{
			Severity:   SeverityWarn,
			Code:       CodePercentCharacter,
			LineNumber: 0, // Will be set by caller
			Expression: "",
			Message:    "Command contains '%' character (cron interprets % as newline)",
			Hint:       GetCodeHint(CodePercentCharacter),
		})
	}

	// Check for quoting/escaping issues
	quotingIssues := checkQuotingEscaping(command)
	issues = append(issues, quotingIssues...)

	return issues
}

// checkAbsolutePath checks if the command uses an absolute path
// This is a heuristic - we look for commands starting with /
func checkAbsolutePath(command string) bool {
	// Trim leading whitespace
	trimmed := strings.TrimSpace(command)
	if len(trimmed) == 0 {
		return false
	}

	// Check if it starts with /
	if strings.HasPrefix(trimmed, "/") {
		return true
	}

	// Check for common absolute paths in the command
	// Look for patterns like /usr/bin/, /bin/, /sbin/, etc.
	parts := strings.Fields(trimmed)
	for _, part := range parts {
		if strings.HasPrefix(part, "/usr/bin/") ||
			strings.HasPrefix(part, "/bin/") ||
			strings.HasPrefix(part, "/sbin/") ||
			strings.HasPrefix(part, "/usr/local/bin/") ||
			strings.HasPrefix(part, "/opt/") {
			return true
		}
	}

	return false
}

// checkOutputRedirection checks if the command redirects stdout or stderr
func checkOutputRedirection(command string) bool {
	// Look for redirection operators
	return strings.Contains(command, ">") ||
		strings.Contains(command, ">>") ||
		strings.Contains(command, "2>") ||
		strings.Contains(command, "&>") ||
		strings.Contains(command, ">>") ||
		strings.Contains(command, "2>>")
}

// checkPercentCharacter checks if the command contains % character
// In cron, % is interpreted as newline, which can cause unexpected behavior
func checkPercentCharacter(command string) bool {
	return strings.Contains(command, "%")
}

// checkQuotingEscaping checks for potential quoting/escaping issues
func checkQuotingEscaping(command string) []Issue {
	var issues []Issue

	// Check for unclosed quotes
	singleQuotes := strings.Count(command, "'")
	doubleQuotes := strings.Count(command, "\"")

	// If odd number of quotes, there might be an issue
	if singleQuotes%2 != 0 {
		issues = append(issues, Issue{
			Severity:   SeverityWarn,
			Code:       CodeQuotingIssue,
			LineNumber: 0, // Will be set by caller
			Expression: "",
			Message:    "Command contains unclosed single quotes",
			Hint:       GetCodeHint(CodeQuotingIssue),
		})
	}

	if doubleQuotes%2 != 0 {
		issues = append(issues, Issue{
			Severity:   SeverityWarn,
			Code:       CodeQuotingIssue,
			LineNumber: 0, // Will be set by caller
			Expression: "",
			Message:    "Command contains unclosed double quotes",
			Hint:       GetCodeHint(CodeQuotingIssue),
		})
	}

	// Check for suspicious patterns like unescaped spaces in arguments
	// This is a simple heuristic - look for patterns that might need quoting
	parts := strings.Fields(command)
	for i, part := range parts {
		// Skip the first part (command itself)
		if i == 0 {
			continue
		}
		// If an argument contains spaces and isn't quoted, it might be an issue
		// This is just a potential issue, not always wrong
		// We'll be conservative and not flag it unless there are other issues
		_ = part // Use part to avoid unused variable warning
	}

	return issues
}
