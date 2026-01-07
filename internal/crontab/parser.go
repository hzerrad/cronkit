package crontab

import (
	"regexp"
	"strings"

	"github.com/hzerrad/cronkit/internal/cronx"
)

var (
	// envVarRegex matches environment variable lines (VAR=value)
	envVarRegex = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*=`)

	// cronAliasRegex matches cron special strings (@hourly, @daily, etc.)
	cronAliasRegex = regexp.MustCompile(`^@(reboot|yearly|annually|monthly|weekly|daily|hourly)`)
)

// ParseLine parses a single line from a crontab file and returns an Entry
func ParseLine(line string, lineNumber int) *Entry {
	entry := &Entry{
		LineNumber: lineNumber,
		Raw:        line,
	}

	// Trim leading/trailing whitespace for analysis
	trimmed := strings.TrimSpace(line)

	// Empty line
	if trimmed == "" {
		entry.Type = EntryTypeEmpty
		return entry
	}

	// Comment line
	if strings.HasPrefix(trimmed, "#") {
		entry.Type = EntryTypeComment
		return entry
	}

	// Environment variable
	if envVarRegex.MatchString(trimmed) {
		entry.Type = EntryTypeEnvVar
		return entry
	}

	// Try to parse as cron job
	job := parseJob(trimmed, lineNumber)
	if job != nil {
		entry.Type = EntryTypeJob
		entry.Job = job
		return entry
	}

	// Invalid/unparseable line
	entry.Type = EntryTypeInvalid
	return entry
}

// parseJob attempts to parse a cron job line
// Returns nil if the line cannot be parsed as a job
func parseJob(line string, lineNumber int) *Job {
	// Check for cron aliases first
	if cronAliasRegex.MatchString(line) {
		return parseAliasJob(line, lineNumber)
	}

	// Split by whitespace (handles both spaces and tabs)
	fields := strings.Fields(line)

	// Need at least 6 fields (5 for expression + 1 for command)
	if len(fields) < 6 {
		return nil
	}

	// Extract cron expression (first 5 fields)
	expression := strings.Join(fields[0:5], " ")

	// Find the command (everything after the expression)
	// We need to find where the command starts in the original line to preserve spacing
	exprEnd := 0
	fieldCount := 0
	for i, char := range line {
		if !isWhitespace(byte(char)) {
			// Check if we've seen 5 non-whitespace segments
			if i > 0 && isWhitespace(line[i-1]) {
				fieldCount++
				if fieldCount == 5 {
					// Find start of command (skip whitespace after 5th field)
					for j := i; j < len(line); j++ {
						if isWhitespace(line[j]) {
							continue
						}
						exprEnd = j
						break
					}
					break
				}
			}
		}
	}

	if exprEnd == 0 {
		return nil
	}

	commandAndComment := line[exprEnd:]

	// Extract inline comment if present
	var command, comment string
	if idx := strings.Index(commandAndComment, "#"); idx != -1 {
		command = strings.TrimSpace(commandAndComment[:idx])
		comment = strings.TrimSpace(commandAndComment[idx+1:])
	} else {
		command = strings.TrimSpace(commandAndComment)
	}

	// Validate the expression using our parser
	parser := cronx.NewParser()
	_, err := parser.Parse(expression)

	job := &Job{
		LineNumber: lineNumber,
		Expression: expression,
		Command:    command,
		Comment:    comment,
		Valid:      err == nil,
	}

	if err != nil {
		job.Error = err.Error()
	}

	return job
}

// parseAliasJob parses a cron job with an alias (@daily, @hourly, etc.)
func parseAliasJob(line string, lineNumber int) *Job {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil
	}

	alias := fields[0]
	commandAndComment := strings.TrimSpace(line[len(alias):])

	// Extract inline comment if present
	var command, comment string
	if idx := strings.Index(commandAndComment, "#"); idx != -1 {
		command = strings.TrimSpace(commandAndComment[:idx])
		comment = strings.TrimSpace(commandAndComment[idx+1:])
	} else {
		command = commandAndComment
	}

	// Validate the alias using our parser
	parser := cronx.NewParser()
	_, err := parser.Parse(alias)

	job := &Job{
		LineNumber: lineNumber,
		Expression: alias,
		Command:    command,
		Comment:    comment,
		Valid:      err == nil,
	}

	if err != nil {
		job.Error = err.Error()
	}

	return job
}

// isWhitespace checks if a byte is whitespace (space or tab)
func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t'
}
