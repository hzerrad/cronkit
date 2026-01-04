package crontab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJob(t *testing.T) {
	t.Run("should create job with all fields", func(t *testing.T) {
		job := &Job{
			LineNumber: 1,
			Expression: "0 2 * * *",
			Command:    "/usr/bin/backup.sh",
			Comment:    "Daily backup",
			Valid:      true,
			Error:      "",
		}

		assert.Equal(t, 1, job.LineNumber)
		assert.Equal(t, "0 2 * * *", job.Expression)
		assert.Equal(t, "/usr/bin/backup.sh", job.Command)
		assert.Equal(t, "Daily backup", job.Comment)
		assert.True(t, job.Valid)
		assert.Empty(t, job.Error)
	})

	t.Run("should create invalid job with error", func(t *testing.T) {
		job := &Job{
			LineNumber: 2,
			Expression: "60 0 * * *",
			Command:    "/usr/bin/invalid.sh",
			Comment:    "",
			Valid:      false,
			Error:      "minute out of range",
		}

		assert.Equal(t, 2, job.LineNumber)
		assert.Equal(t, "60 0 * * *", job.Expression)
		assert.False(t, job.Valid)
		assert.Equal(t, "minute out of range", job.Error)
	})

	t.Run("should create job without comment", func(t *testing.T) {
		job := &Job{
			LineNumber: 3,
			Expression: "*/15 * * * *",
			Command:    "/usr/bin/check.sh",
			Comment:    "",
			Valid:      true,
			Error:      "",
		}

		assert.Equal(t, 3, job.LineNumber)
		assert.Equal(t, "*/15 * * * *", job.Expression)
		assert.Empty(t, job.Comment)
		assert.True(t, job.Valid)
	})
}

func TestEntryType(t *testing.T) {
	t.Run("should have correct entry type constants", func(t *testing.T) {
		assert.Equal(t, EntryType(0), EntryTypeJob)
		assert.Equal(t, EntryType(1), EntryTypeComment)
		assert.Equal(t, EntryType(2), EntryTypeEnvVar)
		assert.Equal(t, EntryType(3), EntryTypeEmpty)
		assert.Equal(t, EntryType(4), EntryTypeInvalid)
	})

	t.Run("should create entry with job type", func(t *testing.T) {
		job := &Job{
			LineNumber: 1,
			Expression: "0 2 * * *",
			Command:    "/usr/bin/backup.sh",
			Valid:      true,
		}

		entry := &Entry{
			Type:       EntryTypeJob,
			LineNumber: 1,
			Raw:        "0 2 * * * /usr/bin/backup.sh",
			Job:        job,
		}

		assert.Equal(t, EntryTypeJob, entry.Type)
		assert.Equal(t, 1, entry.LineNumber)
		assert.Equal(t, "0 2 * * * /usr/bin/backup.sh", entry.Raw)
		assert.NotNil(t, entry.Job)
		assert.Equal(t, job, entry.Job)
	})

	t.Run("should create entry with comment type", func(t *testing.T) {
		entry := &Entry{
			Type:       EntryTypeComment,
			LineNumber: 1,
			Raw:        "# This is a comment",
			Job:        nil,
		}

		assert.Equal(t, EntryTypeComment, entry.Type)
		assert.Equal(t, 1, entry.LineNumber)
		assert.Equal(t, "# This is a comment", entry.Raw)
		assert.Nil(t, entry.Job)
	})

	t.Run("should create entry with env var type", func(t *testing.T) {
		entry := &Entry{
			Type:       EntryTypeEnvVar,
			LineNumber: 1,
			Raw:        "PATH=/usr/bin:/bin",
			Job:        nil,
		}

		assert.Equal(t, EntryTypeEnvVar, entry.Type)
		assert.Equal(t, "PATH=/usr/bin:/bin", entry.Raw)
		assert.Nil(t, entry.Job)
	})

	t.Run("should create entry with empty type", func(t *testing.T) {
		entry := &Entry{
			Type:       EntryTypeEmpty,
			LineNumber: 1,
			Raw:        "",
			Job:        nil,
		}

		assert.Equal(t, EntryTypeEmpty, entry.Type)
		assert.Empty(t, entry.Raw)
		assert.Nil(t, entry.Job)
	})

	t.Run("should create entry with invalid type", func(t *testing.T) {
		entry := &Entry{
			Type:       EntryTypeInvalid,
			LineNumber: 1,
			Raw:        "invalid line",
			Job:        nil,
		}

		assert.Equal(t, EntryTypeInvalid, entry.Type)
		assert.Equal(t, "invalid line", entry.Raw)
		assert.Nil(t, entry.Job)
	})
}
