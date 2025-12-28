package check

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Severity represents the severity level of a validation issue
type Severity int

const (
	// SeverityInfo represents informational messages
	SeverityInfo Severity = iota
	// SeverityWarn represents warning messages
	SeverityWarn
	// SeverityError represents error messages
	SeverityError
)

// String returns the string representation of the severity
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarn:
		return "warn"
	case SeverityError:
		return "error"
	default:
		return "unknown"
	}
}

// MarshalJSON implements json.Marshaler for Severity
func (s Severity) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON implements json.Unmarshaler for Severity
func (s *Severity) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	parsed := SeverityFromString(str)
	if parsed == -1 {
		return fmt.Errorf("invalid severity: %s", str)
	}

	*s = parsed
	return nil
}

// SeverityFromString converts a string to a Severity value
// Returns -1 if the string is invalid
func SeverityFromString(s string) Severity {
	switch s {
	case "info":
		return SeverityInfo
	case "warn", "warning":
		return SeverityWarn
	case "error":
		return SeverityError
	default:
		return -1
	}
}

// IsError returns true if the severity is Error
func (s Severity) IsError() bool {
	return s == SeverityError
}

// IsWarning returns true if the severity is Warn
func (s Severity) IsWarning() bool {
	return s == SeverityWarn
}

// IsInfo returns true if the severity is Info
func (s Severity) IsInfo() bool {
	return s == SeverityInfo
}

// ParseFailOnLevel parses a fail-on level string and returns the corresponding Severity.
// Valid values: "error", "warn", "info" (case-insensitive).
// Returns an error if the string is invalid.
func ParseFailOnLevel(level string) (Severity, error) {
	normalized := ""
	switch strings.ToLower(level) {
	case "error":
		normalized = "error"
	case "warn", "warning":
		normalized = "warn"
	case "info":
		normalized = "info"
	default:
		return -1, fmt.Errorf("invalid fail-on level: %s (must be 'error', 'warn', or 'info')", level)
	}

	severity := SeverityFromString(normalized)
	if severity == -1 {
		return -1, fmt.Errorf("invalid fail-on level: %s", level)
	}

	return severity, nil
}
