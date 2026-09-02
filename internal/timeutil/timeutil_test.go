package timeutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinuteKey_IgnoresLocation(t *testing.T) {
	paris, err := time.LoadLocation("Europe/Paris")
	require.NoError(t, err)

	utc := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	same := utc.In(paris)

	assert.NotEqual(t, utc, same, "the two values differ as time.Time")
	assert.Equal(t, MinuteKey(utc), MinuteKey(same), "but they are the same instant")
}

func TestMinuteKey_TruncatesToTheMinute(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 30, 0, 0, time.UTC)

	assert.Equal(t, MinuteKey(base), MinuteKey(base.Add(59*time.Second)),
		"seconds within the same minute share a key")
	assert.NotEqual(t, MinuteKey(base), MinuteKey(base.Add(time.Minute)),
		"the next minute is a different key")
}

func TestResolveLocation_EmptyZoneUsesFallback(t *testing.T) {
	fallback := time.FixedZone("fallback", 3600)

	loc, err := ResolveLocation("", fallback)

	require.NoError(t, err)
	assert.Same(t, fallback, loc)
}

func TestResolveLocation_NamedZoneOverridesFallback(t *testing.T) {
	fallback := time.UTC

	loc, err := ResolveLocation("Europe/Paris", fallback)

	require.NoError(t, err)
	require.NotNil(t, loc)
	assert.Equal(t, "Europe/Paris", loc.String())
}

func TestResolveLocation_UnknownZoneErrors(t *testing.T) {
	_, err := ResolveLocation("Not/AZone", time.UTC)
	assert.Error(t, err)
}
