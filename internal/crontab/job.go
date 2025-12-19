package crontab

// Job represents a single cron job entry from a crontab file
type Job struct {
	LineNumber int    // Line number in the crontab file (1-indexed)
	Expression string // Cron expression (e.g., "0 0 * * *")
	Command    string // Command to execute
	Comment    string // Inline or preceding comment (optional)
	Valid      bool   // Whether the expression is valid
	Error      string // Parse error if Valid is false
}

// EntryType represents the type of line in a crontab
type EntryType int

const (
	EntryTypeJob     EntryType = iota // Cron job line
	EntryTypeComment                  // Comment line starting with #
	EntryTypeEnvVar                   // Environment variable (VAR=value)
	EntryTypeEmpty                    // Empty or whitespace-only line
	EntryTypeInvalid                  // Invalid/unparseable line
)

// Entry represents any line in a crontab file
type Entry struct {
	Type       EntryType
	LineNumber int
	Raw        string // Original line content
	Job        *Job   // Non-nil only if Type == EntryTypeJob
}
