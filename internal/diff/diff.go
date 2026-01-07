package diff

import (
	"fmt"
	"strings"

	"github.com/hzerrad/cronkit/internal/crontab"
)

// ChangeType represents the type of change detected
type ChangeType int

const (
	ChangeTypeUnchanged ChangeType = iota
	ChangeTypeAdded
	ChangeTypeRemoved
	ChangeTypeModified
)

// Change represents a change between old and new crontab entries
type Change struct {
	Type          ChangeType
	OldEntry      *crontab.Entry
	NewEntry      *crontab.Entry
	OldJob        *crontab.Job
	NewJob        *crontab.Job
	FieldsChanged []string // For modified jobs: which fields changed (expression, command, comment)
}

// Diff represents the semantic differences between two crontabs
type Diff struct {
	Added          []Change
	Removed        []Change
	Modified       []Change
	Unchanged      []Change
	EnvChanges     []EnvChange
	CommentChanges []CommentChange
}

// EnvChange represents a change to an environment variable
type EnvChange struct {
	Type     ChangeType
	OldLine  string
	NewLine  string
	Key      string
	OldValue string
	NewValue string
}

// CommentChange represents a change to a comment
type CommentChange struct {
	Type    ChangeType
	OldLine string
	NewLine string
}

// CompareCrontabs compares two crontabs semantically and returns a Diff
func CompareCrontabs(oldEntries, newEntries []*crontab.Entry) *Diff {
	diff := &Diff{
		Added:          []Change{},
		Removed:        []Change{},
		Modified:       []Change{},
		Unchanged:      []Change{},
		EnvChanges:     []EnvChange{},
		CommentChanges: []CommentChange{},
	}

	// Extract jobs from entries
	oldJobs := extractJobs(oldEntries)
	newJobs := extractJobs(newEntries)

	// Compare jobs semantically
	jobChanges := compareJobs(oldJobs, newJobs)
	for _, change := range jobChanges {
		switch change.Type {
		case ChangeTypeAdded:
			diff.Added = append(diff.Added, change)
		case ChangeTypeRemoved:
			diff.Removed = append(diff.Removed, change)
		case ChangeTypeModified:
			diff.Modified = append(diff.Modified, change)
		case ChangeTypeUnchanged:
			diff.Unchanged = append(diff.Unchanged, change)
		}
	}

	// Compare environment variables
	diff.EnvChanges = compareEnvVars(oldEntries, newEntries)

	// Compare comments
	diff.CommentChanges = compareComments(oldEntries, newEntries)

	return diff
}

// extractJobs extracts all job entries from a list of entries
func extractJobs(entries []*crontab.Entry) []*crontab.Job {
	var jobs []*crontab.Job
	for _, entry := range entries {
		if entry.Type == crontab.EntryTypeJob && entry.Job != nil {
			jobs = append(jobs, entry.Job)
		}
	}
	return jobs
}

// compareJobs compares two lists of jobs semantically
func compareJobs(oldJobs, newJobs []*crontab.Job) []Change {
	var changes []Change

	// Create maps for efficient lookup: key = expression + command
	oldMap := make(map[string]*crontab.Job)
	for _, job := range oldJobs {
		key := jobKey(job)
		oldMap[key] = job
	}

	newMap := make(map[string]*crontab.Job)
	for _, job := range newJobs {
		key := jobKey(job)
		newMap[key] = job
	}

	// Find added jobs (in new but not in old)
	for key, newJob := range newMap {
		if _, exists := oldMap[key]; !exists {
			changes = append(changes, Change{
				Type:   ChangeTypeAdded,
				NewJob: newJob,
			})
		}
	}

	// Find removed jobs (in old but not in new)
	for key, oldJob := range oldMap {
		if _, exists := newMap[key]; !exists {
			changes = append(changes, Change{
				Type:   ChangeTypeRemoved,
				OldJob: oldJob,
			})
		}
	}

	// Find modified jobs (same key but different fields)
	for key, newJob := range newMap {
		if oldJob, exists := oldMap[key]; exists {
			// Check if any fields changed
			fieldsChanged := detectFieldChanges(oldJob, newJob)
			if len(fieldsChanged) > 0 {
				changes = append(changes, Change{
					Type:          ChangeTypeModified,
					OldJob:        oldJob,
					NewJob:        newJob,
					FieldsChanged: fieldsChanged,
				})
			} else {
				changes = append(changes, Change{
					Type:   ChangeTypeUnchanged,
					OldJob: oldJob,
					NewJob: newJob,
				})
			}
		}
	}

	return changes
}

// jobKey creates a semantic key for a job (expression + command, normalized)
func jobKey(job *crontab.Job) string {
	// Normalize expression and command by trimming whitespace
	expr := strings.TrimSpace(job.Expression)
	cmd := strings.TrimSpace(job.Command)
	return fmt.Sprintf("%s|||%s", expr, cmd)
}

// detectFieldChanges detects which fields changed between two jobs
func detectFieldChanges(oldJob, newJob *crontab.Job) []string {
	var fields []string

	// Note: We don't check expression/command because if those changed,
	// it would be a different job key. We only check comment here.
	oldExpr := strings.TrimSpace(oldJob.Expression)
	newExpr := strings.TrimSpace(newJob.Expression)
	if oldExpr != newExpr {
		fields = append(fields, "expression")
	}

	oldCmd := strings.TrimSpace(oldJob.Command)
	newCmd := strings.TrimSpace(newJob.Command)
	if oldCmd != newCmd {
		fields = append(fields, "command")
	}

	oldComment := strings.TrimSpace(oldJob.Comment)
	newComment := strings.TrimSpace(newJob.Comment)
	if oldComment != newComment {
		fields = append(fields, "comment")
	}

	return fields
}

// compareEnvVars compares environment variables between two crontabs
func compareEnvVars(oldEntries, newEntries []*crontab.Entry) []EnvChange {
	var changes []EnvChange

	oldEnv := extractEnvVars(oldEntries)
	newEnv := extractEnvVars(newEntries)

	// Find added env vars
	for key, newValue := range newEnv {
		if oldValue, exists := oldEnv[key]; !exists {
			changes = append(changes, EnvChange{
				Type:     ChangeTypeAdded,
				Key:      key,
				NewValue: newValue,
			})
		} else if oldValue != newValue {
			changes = append(changes, EnvChange{
				Type:     ChangeTypeModified,
				Key:      key,
				OldValue: oldValue,
				NewValue: newValue,
			})
		}
	}

	// Find removed env vars
	for key, oldValue := range oldEnv {
		if _, exists := newEnv[key]; !exists {
			changes = append(changes, EnvChange{
				Type:     ChangeTypeRemoved,
				Key:      key,
				OldValue: oldValue,
			})
		}
	}

	return changes
}

// extractEnvVars extracts environment variables from entries
func extractEnvVars(entries []*crontab.Entry) map[string]string {
	env := make(map[string]string)
	for _, entry := range entries {
		if entry.Type == crontab.EntryTypeEnvVar {
			// Parse VAR=value format
			parts := strings.SplitN(entry.Raw, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				env[key] = value
			}
		}
	}
	return env
}

// compareComments compares comments between two crontabs
func compareComments(oldEntries, newEntries []*crontab.Entry) []CommentChange {
	var changes []CommentChange

	oldComments := extractComments(oldEntries)
	newComments := extractComments(newEntries)

	// Simple comparison: if comment count or content changed
	// For a full semantic diff, we'd need more sophisticated matching
	// For now, we'll just track additions/removals
	if len(newComments) > len(oldComments) {
		// Comments added
		for i := len(oldComments); i < len(newComments); i++ {
			changes = append(changes, CommentChange{
				Type:    ChangeTypeAdded,
				NewLine: newComments[i],
			})
		}
	} else if len(oldComments) > len(newComments) {
		// Comments removed
		for i := len(newComments); i < len(oldComments); i++ {
			changes = append(changes, CommentChange{
				Type:    ChangeTypeRemoved,
				OldLine: oldComments[i],
			})
		}
	}

	return changes
}

// extractComments extracts comment lines from entries
func extractComments(entries []*crontab.Entry) []string {
	var comments []string
	for _, entry := range entries {
		if entry.Type == crontab.EntryTypeComment {
			comments = append(comments, entry.Raw)
		}
	}
	return comments
}
