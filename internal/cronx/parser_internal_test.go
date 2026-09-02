package cronx

import (
	"testing"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseRobfig_RecoversFromAPanic exercises parseRobfig's recover with an input confirmed to panic robfig.
func TestParseRobfig_RecoversFromAPanic(t *testing.T) {
	cp := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

	sched, err := parseRobfig(cp, "CRON_TZ=Asia/Tokyo")

	require.Error(t, err, "a panic inside robfig must come back as an error, not crash the test")
	assert.Nil(t, sched)
	assert.Contains(t, err.Error(), "CRON_TZ=Asia/Tokyo")
}

// TestParseRobfig_StillWorksForValidInput guards against recover swallowing a real result.
func TestParseRobfig_StillWorksForValidInput(t *testing.T) {
	cp := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

	sched, err := parseRobfig(cp, "0 2 * * *")

	require.NoError(t, err)
	assert.NotNil(t, sched)
}
