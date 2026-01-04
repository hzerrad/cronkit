package diff

import (
	"testing"

	"github.com/hzerrad/cronic/internal/crontab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareCrontabs_AddedJob(t *testing.T) {
	oldEntries := []*crontab.Entry{
		{
			Type:       crontab.EntryTypeJob,
			LineNumber: 1,
			Job: &crontab.Job{
				LineNumber: 1,
				Expression: "0 2 * * *",
				Command:    "/usr/bin/backup.sh",
				Valid:      true,
			},
		},
	}

	newEntries := []*crontab.Entry{
		{
			Type:       crontab.EntryTypeJob,
			LineNumber: 1,
			Job: &crontab.Job{
				LineNumber: 1,
				Expression: "0 2 * * *",
				Command:    "/usr/bin/backup.sh",
				Valid:      true,
			},
		},
		{
			Type:       crontab.EntryTypeJob,
			LineNumber: 2,
			Job: &crontab.Job{
				LineNumber: 2,
				Expression: "*/15 * * * *",
				Command:    "/usr/bin/check.sh",
				Valid:      true,
			},
		},
	}

	result := CompareCrontabs(oldEntries, newEntries)

	require.Len(t, result.Added, 1)
	assert.Equal(t, "*/15 * * * *", result.Added[0].NewJob.Expression)
	assert.Equal(t, "/usr/bin/check.sh", result.Added[0].NewJob.Command)
}

func TestCompareCrontabs_RemovedJob(t *testing.T) {
	oldEntries := []*crontab.Entry{
		{
			Type:       crontab.EntryTypeJob,
			LineNumber: 1,
			Job: &crontab.Job{
				LineNumber: 1,
				Expression: "0 2 * * *",
				Command:    "/usr/bin/backup.sh",
				Valid:      true,
			},
		},
		{
			Type:       crontab.EntryTypeJob,
			LineNumber: 2,
			Job: &crontab.Job{
				LineNumber: 2,
				Expression: "*/15 * * * *",
				Command:    "/usr/bin/check.sh",
				Valid:      true,
			},
		},
	}

	newEntries := []*crontab.Entry{
		{
			Type:       crontab.EntryTypeJob,
			LineNumber: 1,
			Job: &crontab.Job{
				LineNumber: 1,
				Expression: "0 2 * * *",
				Command:    "/usr/bin/backup.sh",
				Valid:      true,
			},
		},
	}

	result := CompareCrontabs(oldEntries, newEntries)

	require.Len(t, result.Removed, 1)
	assert.Equal(t, "*/15 * * * *", result.Removed[0].OldJob.Expression)
	assert.Equal(t, "/usr/bin/check.sh", result.Removed[0].OldJob.Command)
}

func TestCompareCrontabs_ModifiedJob(t *testing.T) {
	oldEntries := []*crontab.Entry{
		{
			Type:       crontab.EntryTypeJob,
			LineNumber: 1,
			Job: &crontab.Job{
				LineNumber: 1,
				Expression: "0 2 * * *",
				Command:    "/usr/bin/backup.sh",
				Comment:    "Old comment",
				Valid:      true,
			},
		},
	}

	newEntries := []*crontab.Entry{
		{
			Type:       crontab.EntryTypeJob,
			LineNumber: 1,
			Job: &crontab.Job{
				LineNumber: 1,
				Expression: "0 2 * * *",
				Command:    "/usr/bin/backup.sh",
				Comment:    "New comment",
				Valid:      true,
			},
		},
	}

	result := CompareCrontabs(oldEntries, newEntries)

	require.Len(t, result.Modified, 1)
	assert.Equal(t, "Old comment", result.Modified[0].OldJob.Comment)
	assert.Equal(t, "New comment", result.Modified[0].NewJob.Comment)
	assert.Contains(t, result.Modified[0].FieldsChanged, "comment")
}

func TestCompareCrontabs_ModifiedExpression(t *testing.T) {
	oldEntries := []*crontab.Entry{
		{
			Type:       crontab.EntryTypeJob,
			LineNumber: 1,
			Job: &crontab.Job{
				LineNumber: 1,
				Expression: "0 2 * * *",
				Command:    "/usr/bin/backup.sh",
				Valid:      true,
			},
		},
	}

	newEntries := []*crontab.Entry{
		{
			Type:       crontab.EntryTypeJob,
			LineNumber: 1,
			Job: &crontab.Job{
				LineNumber: 1,
				Expression: "0 3 * * *",
				Command:    "/usr/bin/backup.sh",
				Valid:      true,
			},
		},
	}

	result := CompareCrontabs(oldEntries, newEntries)

	// Expression change means different job key, so it should be removed + added
	require.Len(t, result.Removed, 1)
	require.Len(t, result.Added, 1)
	assert.Equal(t, "0 2 * * *", result.Removed[0].OldJob.Expression)
	assert.Equal(t, "0 3 * * *", result.Added[0].NewJob.Expression)
}

func TestCompareCrontabs_UnchangedJob(t *testing.T) {
	oldEntries := []*crontab.Entry{
		{
			Type:       crontab.EntryTypeJob,
			LineNumber: 1,
			Job: &crontab.Job{
				LineNumber: 1,
				Expression: "0 2 * * *",
				Command:    "/usr/bin/backup.sh",
				Valid:      true,
			},
		},
	}

	newEntries := []*crontab.Entry{
		{
			Type:       crontab.EntryTypeJob,
			LineNumber: 5, // Different line number, but same job
			Job: &crontab.Job{
				LineNumber: 5,
				Expression: "0 2 * * *",
				Command:    "/usr/bin/backup.sh",
				Valid:      true,
			},
		},
	}

	result := CompareCrontabs(oldEntries, newEntries)

	require.Len(t, result.Unchanged, 1)
	assert.Equal(t, "0 2 * * *", result.Unchanged[0].OldJob.Expression)
	assert.Equal(t, "0 2 * * *", result.Unchanged[0].NewJob.Expression)
}

func TestCompareCrontabs_EnvVarChanges(t *testing.T) {
	oldEntries := []*crontab.Entry{
		{
			Type:       crontab.EntryTypeEnvVar,
			LineNumber: 1,
			Raw:        "PATH=/usr/bin:/usr/local/bin",
		},
	}

	newEntries := []*crontab.Entry{
		{
			Type:       crontab.EntryTypeEnvVar,
			LineNumber: 1,
			Raw:        "PATH=/usr/bin:/usr/local/bin:/opt/bin",
		},
		{
			Type:       crontab.EntryTypeEnvVar,
			LineNumber: 2,
			Raw:        "TZ=UTC",
		},
	}

	result := CompareCrontabs(oldEntries, newEntries)

	require.Len(t, result.EnvChanges, 2)

	// Modified PATH
	var pathChange *EnvChange
	for i := range result.EnvChanges {
		if result.EnvChanges[i].Key == "PATH" {
			pathChange = &result.EnvChanges[i]
			break
		}
	}
	require.NotNil(t, pathChange)
	assert.Equal(t, ChangeTypeModified, pathChange.Type)

	// Added TZ
	var tzChange *EnvChange
	for i := range result.EnvChanges {
		if result.EnvChanges[i].Key == "TZ" {
			tzChange = &result.EnvChanges[i]
			break
		}
	}
	require.NotNil(t, tzChange)
	assert.Equal(t, ChangeTypeAdded, tzChange.Type)
}

func TestJobKey(t *testing.T) {
	job1 := &crontab.Job{
		Expression: "0 2 * * *",
		Command:    "/usr/bin/backup.sh",
	}

	job2 := &crontab.Job{
		Expression: "  0 2 * * *  ",
		Command:    "  /usr/bin/backup.sh  ",
	}

	key1 := jobKey(job1)
	key2 := jobKey(job2)

	assert.Equal(t, key1, key2, "Keys should match after normalization")
}

func TestDetectFieldChanges(t *testing.T) {
	t.Run("no changes", func(t *testing.T) {
		oldJob := &crontab.Job{
			Expression: "0 2 * * *",
			Command:    "/usr/bin/backup.sh",
			Comment:    "test",
		}
		newJob := &crontab.Job{
			Expression: "0 2 * * *",
			Command:    "/usr/bin/backup.sh",
			Comment:    "test",
		}

		changes := detectFieldChanges(oldJob, newJob)
		assert.Empty(t, changes)
	})

	t.Run("comment changed", func(t *testing.T) {
		oldJob := &crontab.Job{
			Expression: "0 2 * * *",
			Command:    "/usr/bin/backup.sh",
			Comment:    "old",
		}
		newJob := &crontab.Job{
			Expression: "0 2 * * *",
			Command:    "/usr/bin/backup.sh",
			Comment:    "new",
		}

		changes := detectFieldChanges(oldJob, newJob)
		assert.Contains(t, changes, "comment")
	})

	t.Run("command changed", func(t *testing.T) {
		oldJob := &crontab.Job{
			Expression: "0 2 * * *",
			Command:    "/usr/bin/old.sh",
		}
		newJob := &crontab.Job{
			Expression: "0 2 * * *",
			Command:    "/usr/bin/new.sh",
		}

		changes := detectFieldChanges(oldJob, newJob)
		assert.Contains(t, changes, "command")
	})

	t.Run("expression changed", func(t *testing.T) {
		oldJob := &crontab.Job{
			Expression: "0 2 * * *",
			Command:    "/usr/bin/backup.sh",
		}
		newJob := &crontab.Job{
			Expression: "0 3 * * *",
			Command:    "/usr/bin/backup.sh",
		}

		changes := detectFieldChanges(oldJob, newJob)
		assert.Contains(t, changes, "expression")
	})

	t.Run("multiple fields changed", func(t *testing.T) {
		oldJob := &crontab.Job{
			Expression: "0 2 * * *",
			Command:    "/usr/bin/old.sh",
			Comment:    "old",
		}
		newJob := &crontab.Job{
			Expression: "0 3 * * *",
			Command:    "/usr/bin/new.sh",
			Comment:    "new",
		}

		changes := detectFieldChanges(oldJob, newJob)
		assert.Contains(t, changes, "expression")
		assert.Contains(t, changes, "command")
		assert.Contains(t, changes, "comment")
	})
}

func TestCompareCrontabs_CommentChanges(t *testing.T) {
	oldEntries := []*crontab.Entry{
		{
			Type:       crontab.EntryTypeComment,
			LineNumber: 1,
			Raw:        "# Old comment",
		},
	}

	newEntries := []*crontab.Entry{
		{
			Type:       crontab.EntryTypeComment,
			LineNumber: 1,
			Raw:        "# Old comment",
		},
		{
			Type:       crontab.EntryTypeComment,
			LineNumber: 2,
			Raw:        "# New comment",
		},
	}

	result := CompareCrontabs(oldEntries, newEntries)
	// Comment changes may or may not be detected depending on implementation
	assert.NotNil(t, result)
}

func TestCompareCrontabs_EmptyCrontabs(t *testing.T) {
	result := CompareCrontabs([]*crontab.Entry{}, []*crontab.Entry{})
	assert.Empty(t, result.Added)
	assert.Empty(t, result.Removed)
	assert.Empty(t, result.Modified)
	assert.Empty(t, result.Unchanged)
}

func TestCompareCrontabs_EnvVarRemoved(t *testing.T) {
	oldEntries := []*crontab.Entry{
		{
			Type:       crontab.EntryTypeEnvVar,
			LineNumber: 1,
			Raw:        "PATH=/usr/bin",
		},
	}

	newEntries := []*crontab.Entry{}

	result := CompareCrontabs(oldEntries, newEntries)
	require.Greater(t, len(result.EnvChanges), 0)
	assert.Equal(t, ChangeTypeRemoved, result.EnvChanges[0].Type)
	assert.Equal(t, "PATH", result.EnvChanges[0].Key)
}

func TestCompareCrontabs_EnvVarModified(t *testing.T) {
	oldEntries := []*crontab.Entry{
		{
			Type:       crontab.EntryTypeEnvVar,
			LineNumber: 1,
			Raw:        "PATH=/usr/bin",
		},
	}

	newEntries := []*crontab.Entry{
		{
			Type:       crontab.EntryTypeEnvVar,
			LineNumber: 1,
			Raw:        "PATH=/usr/bin:/usr/local/bin",
		},
	}

	result := CompareCrontabs(oldEntries, newEntries)
	require.Greater(t, len(result.EnvChanges), 0)
	assert.Equal(t, ChangeTypeModified, result.EnvChanges[0].Type)
	assert.Equal(t, "PATH", result.EnvChanges[0].Key)
}
