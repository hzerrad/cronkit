package inventory

import (
	"testing"

	"github.com/hzerrad/cronkit/internal/crontab"
	"github.com/stretchr/testify/assert"
)

func TestFromCrontabJobs_MapsEveryField(t *testing.T) {
	jobs := []*crontab.Job{
		{
			LineNumber: 7,
			Expression: "0 2 * * *",
			Command:    "/usr/bin/backup.sh",
			Comment:    "nightly backup",
			Valid:      true,
		},
	}

	items := FromCrontabJobs(jobs, "ops/crontab")

	require := assert.New(t)
	require.Len(items, 1)
	got := items[0]
	require.Equal("0 2 * * *", got.Expression)
	require.Equal("crontab", got.SourceID)
	require.Equal("vixie", got.Dialect)
	require.Equal("/usr/bin/backup.sh", got.Command)
	require.True(got.Shell)
	require.Equal("nightly backup", got.Comment)
	require.Equal("", got.RunAs)
	require.Equal("", got.Timezone, "a crontab job carries no timezone of its own")
	require.Equal(StateActive, got.State)
	require.Equal("", got.Reason)
	require.Equal(ConcurrencyUnspecified, got.Concurrency)
	require.Equal(Locator{File: "ops/crontab", Line: 7}, got.Locator)
}

func TestFromCrontabJobs_InvalidJobBecomesStateInvalid(t *testing.T) {
	jobs := []*crontab.Job{
		{
			LineNumber: 3,
			Expression: "99 * * * *",
			Command:    "/usr/bin/broken.sh",
			Valid:      false,
			Error:      "invalid minute value: 99",
		},
	}

	items := FromCrontabJobs(jobs, "ops/crontab")

	require := assert.New(t)
	require.Len(items, 1)
	got := items[0]
	require.Equal(StateInvalid, got.State)
	require.Equal("invalid minute value: 99", got.Reason)
	require.Equal(Locator{File: "ops/crontab", Line: 3}, got.Locator)
}

func TestFromCrontabJobs_EmptyInput(t *testing.T) {
	items := FromCrontabJobs(nil, "ops/crontab")
	assert.Empty(t, items)
}

func TestFromCrontabJobs_OneItemPerJob(t *testing.T) {
	jobs := []*crontab.Job{
		{LineNumber: 1, Expression: "0 * * * *", Valid: true},
		{LineNumber: 2, Expression: "0 0 * * *", Valid: true},
		{LineNumber: 3, Expression: "bad", Valid: false, Error: "boom"},
	}

	items := FromCrontabJobs(jobs, "crontab.txt")
	assert.Len(t, items, 3)
}
