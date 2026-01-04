package crontab

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReadFile_SampleCron tests reading a sample crontab file
func TestReadFile_SampleCron(t *testing.T) {
	reader := NewReader()
	jobs, err := reader.ReadFile("../../testdata/crontab/valid/sample.cron")

	require.NoError(t, err)
	assert.Greater(t, len(jobs), 0, "Should find jobs in sample file")

	// Verify we got valid jobs
	validJobs := 0
	for _, job := range jobs {
		if job.Valid {
			validJobs++
		}
	}
	assert.Greater(t, validJobs, 0, "Should have at least one valid job")

	// Check for specific jobs
	foundBackup := false
	foundDiskCheck := false
	for _, job := range jobs {
		if job.Command == "/usr/local/bin/backup.sh" {
			foundBackup = true
			assert.Equal(t, "0 2 * * *", job.Expression)
		}
		if job.Command == "/usr/local/bin/check-disk.sh" {
			foundDiskCheck = true
			assert.Equal(t, "*/15 * * * *", job.Expression)
			assert.Equal(t, "Disk monitoring", job.Comment)
		}
	}
	assert.True(t, foundBackup, "Should find backup job")
	assert.True(t, foundDiskCheck, "Should find disk check job")
}

// TestReadFile_InvalidCron tests reading a crontab with invalid entries
func TestReadFile_InvalidCron(t *testing.T) {
	reader := NewReader()
	jobs, err := reader.ReadFile("../../testdata/crontab/invalid/invalid.cron")

	require.NoError(t, err, "Reading file should not error even with invalid entries")

	// Should have both valid and invalid jobs
	validJobs := 0
	invalidJobs := 0
	for _, job := range jobs {
		if job.Valid {
			validJobs++
		} else {
			invalidJobs++
		}
	}

	assert.Greater(t, validJobs, 0, "Should have at least one valid job")
	assert.Greater(t, invalidJobs, 0, "Should have at least one invalid job")
}

// TestReadFile_EmptyCron tests reading an empty crontab
func TestReadFile_EmptyCron(t *testing.T) {
	reader := NewReader()
	jobs, err := reader.ReadFile("../../testdata/crontab/valid/empty.cron")

	require.NoError(t, err)
	assert.Empty(t, jobs, "Empty crontab should return no jobs")
}

// TestReadFile_NonExistent tests reading a non-existent file
func TestReadFile_NonExistent(t *testing.T) {
	reader := NewReader()
	jobs, err := reader.ReadFile("../../testdata/crontab/does-not-exist.cron")

	assert.Error(t, err, "Should error for non-existent file")
	assert.Nil(t, jobs, "Should return nil jobs on error")
}

// TestReadFile_JobDetails tests that job details are correctly parsed
func TestReadFile_JobDetails(t *testing.T) {
	reader := NewReader()
	jobs, err := reader.ReadFile("../../testdata/crontab/valid/sample.cron")

	require.NoError(t, err)
	require.NotEmpty(t, jobs)

	// Find the weekly report job
	var weeklyJob *Job
	for _, job := range jobs {
		if job.Command == "/usr/local/bin/weekly-report.sh" {
			weeklyJob = job
			break
		}
	}

	require.NotNil(t, weeklyJob, "Should find weekly report job")
	assert.Equal(t, "0 9 * * 1", weeklyJob.Expression)
	assert.True(t, weeklyJob.Valid)
	assert.Empty(t, weeklyJob.Error)
	assert.Greater(t, weeklyJob.LineNumber, 0)
}

// TestReadFile_AliasJobs tests that @alias jobs are parsed correctly
func TestReadFile_AliasJobs(t *testing.T) {
	reader := NewReader()
	jobs, err := reader.ReadFile("../../testdata/crontab/valid/sample.cron")

	require.NoError(t, err)

	// Find alias jobs
	aliases := make(map[string]string)
	for _, job := range jobs {
		if job.Expression[0] == '@' {
			aliases[job.Expression] = job.Command
		}
	}

	assert.NotEmpty(t, aliases, "Should find @alias jobs")
	assert.Contains(t, aliases, "@monthly")
	assert.Contains(t, aliases, "@hourly")
}

// TestParseFile_AllEntries tests parsing all types of entries
func TestParseFile_AllEntries(t *testing.T) {
	reader := NewReader()
	entries, err := reader.ParseFile("../../testdata/crontab/valid/sample.cron")

	require.NoError(t, err)
	require.NotEmpty(t, entries)

	// Count different entry types
	counts := make(map[EntryType]int)
	for _, entry := range entries {
		counts[entry.Type]++
	}

	assert.Greater(t, counts[EntryTypeJob], 0, "Should have job entries")
	assert.Greater(t, counts[EntryTypeComment], 0, "Should have comment entries")
	assert.Greater(t, counts[EntryTypeEnvVar], 0, "Should have env var entries")
	assert.Greater(t, counts[EntryTypeEmpty], 0, "Should have empty line entries")
}

// TestReadUser tests reading the user's crontab
// This test will work whether the user has a crontab or not
func TestReadUser(t *testing.T) {
	reader := NewReader()
	jobs, err := reader.ReadUser()

	// Should not error (even if no crontab exists, it returns empty list)
	assert.NoError(t, err)
	assert.NotNil(t, jobs, "Should return jobs list (may be empty)")

	// If user has crontab, verify jobs are parsed correctly
	if len(jobs) > 0 {
		// Verify all jobs have required fields
		for _, job := range jobs {
			assert.NotEmpty(t, job.Expression, "Job should have expression")
			assert.NotEmpty(t, job.Command, "Job should have command")
			assert.Greater(t, job.LineNumber, 0, "Job should have line number")
		}
	}
}

// TestReadStdin_ValidContent tests reading valid crontab content from stdin
func TestReadStdin_ValidContent(t *testing.T) {
	reader := NewReader()

	// Create a string reader to simulate stdin
	content := "0 2 * * * /usr/local/bin/backup.sh\n*/15 * * * * /usr/local/bin/check-disk.sh\n"
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Create a temporary file to simulate stdin
	tmpfile, err := os.CreateTemp("", "stdin-test")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	_, err = tmpfile.Seek(0, 0)
	require.NoError(t, err)

	os.Stdin = tmpfile
	_ = tmpfile.Close()

	// Reopen for reading
	tmpfile, err = os.Open(tmpfile.Name())
	require.NoError(t, err)
	os.Stdin = tmpfile
	defer func() { _ = tmpfile.Close() }()

	jobs, err := reader.ReadStdin()
	require.NoError(t, err)
	assert.Len(t, jobs, 2)
	assert.Equal(t, "0 2 * * *", jobs[0].Expression)
	assert.Equal(t, "/usr/local/bin/backup.sh", jobs[0].Command)
	assert.Equal(t, "*/15 * * * *", jobs[1].Expression)
}

// TestReadStdin_EmptyInput tests reading empty stdin
func TestReadStdin_EmptyInput(t *testing.T) {
	reader := NewReader()

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	tmpfile, err := os.CreateTemp("", "stdin-empty")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	os.Stdin = tmpfile
	_ = tmpfile.Close()

	tmpfile, err = os.Open(tmpfile.Name())
	require.NoError(t, err)
	os.Stdin = tmpfile
	defer func() { _ = tmpfile.Close() }()

	jobs, err := reader.ReadStdin()
	require.NoError(t, err)
	assert.Empty(t, jobs)
}

// TestReadStdin_InvalidEntries tests reading stdin with invalid entries
func TestReadStdin_InvalidEntries(t *testing.T) {
	reader := NewReader()

	content := "0 2 * * * /usr/local/bin/backup.sh\ninvalid cron line\n*/15 * * * * /usr/local/bin/check-disk.sh\n"
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	tmpfile, err := os.CreateTemp("", "stdin-invalid")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	_, err = tmpfile.Seek(0, 0)
	require.NoError(t, err)

	os.Stdin = tmpfile
	_ = tmpfile.Close()

	tmpfile, err = os.Open(tmpfile.Name())
	require.NoError(t, err)
	os.Stdin = tmpfile
	defer func() { _ = tmpfile.Close() }()

	jobs, err := reader.ReadStdin()
	require.NoError(t, err)
	// Should return valid jobs, skip invalid ones
	assert.GreaterOrEqual(t, len(jobs), 2)

	validCount := 0
	for _, job := range jobs {
		if job.Valid {
			validCount++
		}
	}
	assert.GreaterOrEqual(t, validCount, 2)
}

// TestParseStdin_AllEntryTypes tests parsing all entry types from stdin
func TestParseStdin_AllEntryTypes(t *testing.T) {
	reader := NewReader()

	content := "# This is a comment\nSHELL=/bin/bash\n0 2 * * * /usr/local/bin/backup.sh\n\n*/15 * * * * /usr/local/bin/check-disk.sh\n"
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	tmpfile, err := os.CreateTemp("", "stdin-entries")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	_, err = tmpfile.Seek(0, 0)
	require.NoError(t, err)

	os.Stdin = tmpfile
	_ = tmpfile.Close()

	tmpfile, err = os.Open(tmpfile.Name())
	require.NoError(t, err)
	os.Stdin = tmpfile
	defer func() { _ = tmpfile.Close() }()

	entries, err := reader.ParseStdin()
	require.NoError(t, err)
	assert.Greater(t, len(entries), 0)

	// Count different entry types
	counts := make(map[EntryType]int)
	for _, entry := range entries {
		counts[entry.Type]++
	}

	assert.Greater(t, counts[EntryTypeJob], 0)
	assert.Greater(t, counts[EntryTypeComment], 0)
	assert.Greater(t, counts[EntryTypeEnvVar], 0)
}

// TestParseStdin_MixedContent tests parsing mixed content from stdin
func TestParseStdin_MixedContent(t *testing.T) {
	reader := NewReader()

	content := "# Backup job\n0 2 * * * /usr/local/bin/backup.sh\n# Monitoring\n*/15 * * * * /usr/local/bin/check-disk.sh\n"
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	tmpfile, err := os.CreateTemp("", "stdin-mixed")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	_, err = tmpfile.Seek(0, 0)
	require.NoError(t, err)

	os.Stdin = tmpfile
	_ = tmpfile.Close()

	tmpfile, err = os.Open(tmpfile.Name())
	require.NoError(t, err)
	os.Stdin = tmpfile
	defer func() { _ = tmpfile.Close() }()

	entries, err := reader.ParseStdin()
	require.NoError(t, err)
	assert.Greater(t, len(entries), 0)

	// Should have both comments and jobs
	hasComment := false
	hasJob := false
	for _, entry := range entries {
		if entry.Type == EntryTypeComment {
			hasComment = true
		}
		if entry.Type == EntryTypeJob {
			hasJob = true
		}
	}
	assert.True(t, hasComment)
	assert.True(t, hasJob)
}

// TestReadStdin_LargeContent tests reading large content from stdin
func TestReadStdin_LargeContent(t *testing.T) {
	reader := NewReader()

	// Create content with many jobs
	var builder strings.Builder
	for i := 0; i < 100; i++ {
		builder.WriteString("0 * * * * /usr/local/bin/job")
		builder.WriteString(string(rune('0' + (i % 10))))
		builder.WriteString(".sh\n")
	}
	content := builder.String()

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	tmpfile, err := os.CreateTemp("", "stdin-large")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)
	_, err = tmpfile.Seek(0, 0)
	require.NoError(t, err)

	os.Stdin = tmpfile
	_ = tmpfile.Close()

	tmpfile, err = os.Open(tmpfile.Name())
	require.NoError(t, err)
	os.Stdin = tmpfile
	defer func() { _ = tmpfile.Close() }()

	jobs, err := reader.ReadStdin()
	require.NoError(t, err)
	assert.Len(t, jobs, 100)
}
