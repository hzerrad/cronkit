package crontab

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Reader provides methods to read crontab files
type Reader interface {
	// ReadFile reads and parses cron jobs from a file
	ReadFile(path string) ([]*Job, error)

	// ReadUser reads cron jobs from the current user's crontab
	ReadUser() ([]*Job, error)

	// ParseFile reads all entries (including comments, env vars) from a file
	ParseFile(path string) ([]*Entry, error)
}

// reader implements the Reader interface
type reader struct{}

// NewReader creates a new crontab reader
func NewReader() Reader {
	return &reader{}
}

// ReadFile reads and parses cron jobs from a file
func (r *reader) ReadFile(path string) ([]*Job, error) {
	entries, err := r.ParseFile(path)
	if err != nil {
		return nil, err
	}

	// Extract only job entries
	var jobs []*Job
	for _, entry := range entries {
		if entry.Type == EntryTypeJob && entry.Job != nil {
			jobs = append(jobs, entry.Job)
		}
	}

	return jobs, nil
}

// ReadUser reads cron jobs from the current user's crontab using `crontab -l`
func (r *reader) ReadUser() ([]*Job, error) {
	// Execute `crontab -l` to get user's crontab
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.Output()
	if err != nil {
		// If exit code is 1, it might mean no crontab exists
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				// No crontab for user - return empty list, not an error
				return []*Job{}, nil
			}
		}
		return nil, fmt.Errorf("failed to read user crontab: %w", err)
	}

	// Parse the output line by line
	lines := strings.Split(string(output), "\n")
	var jobs []*Job

	for lineNum, line := range lines {
		entry := ParseLine(line, lineNum+1)
		if entry.Type == EntryTypeJob && entry.Job != nil {
			jobs = append(jobs, entry.Job)
		}
	}

	return jobs, nil
}

// ParseFile reads all entries from a crontab file
func (r *reader) ParseFile(path string) (entries []*Entry, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("error closing file: %w", closeErr)
		}
	}()

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		entry := ParseLine(line, lineNumber)
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return entries, nil
}
