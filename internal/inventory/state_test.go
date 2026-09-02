package inventory

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestState_String(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateActive, "active"},
		{StateSuspended, "suspended"},
		{StateUnresolved, "unresolved"},
		{StateInvalid, "invalid"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestStateFromString(t *testing.T) {
	assert.Equal(t, StateActive, StateFromString("active"))
	assert.Equal(t, StateSuspended, StateFromString("suspended"))
	assert.Equal(t, StateUnresolved, StateFromString("unresolved"))
	assert.Equal(t, StateInvalid, StateFromString("invalid"))
	assert.Equal(t, State(-1), StateFromString("nonsense"))
}

func TestState_JSONRoundTrip(t *testing.T) {
	data, err := json.Marshal(StateSuspended)
	require.NoError(t, err)
	assert.JSONEq(t, `"suspended"`, string(data))

	var decoded State
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, StateSuspended, decoded)
}

func TestState_UnmarshalRejectsUnknown(t *testing.T) {
	var s State
	err := json.Unmarshal([]byte(`"nonsense"`), &s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonsense")
}

func TestConcurrency_String(t *testing.T) {
	tests := []struct {
		value    Concurrency
		expected string
	}{
		{ConcurrencyUnspecified, ""},
		{ConcurrencyAllow, "allow"},
		{ConcurrencyForbid, "forbid"},
		{ConcurrencyReplace, "replace"},
		{Concurrency(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.value.String())
		})
	}
}

func TestConcurrencyFromString(t *testing.T) {
	assert.Equal(t, ConcurrencyAllow, ConcurrencyFromString("Allow"))
	assert.Equal(t, ConcurrencyForbid, ConcurrencyFromString("Forbid"))
	assert.Equal(t, ConcurrencyReplace, ConcurrencyFromString("Replace"))
	assert.Equal(t, ConcurrencyUnspecified, ConcurrencyFromString(""))
	assert.Equal(t, Concurrency(-1), ConcurrencyFromString("nonsense"))
}

func TestConcurrency_JSONRoundTrip(t *testing.T) {
	data, err := json.Marshal(ConcurrencyForbid)
	require.NoError(t, err)
	assert.JSONEq(t, `"forbid"`, string(data))

	var decoded Concurrency
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, ConcurrencyForbid, decoded)
}

func TestConcurrency_UnmarshalRejectsUnknown(t *testing.T) {
	var c Concurrency
	err := json.Unmarshal([]byte(`"nonsense"`), &c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonsense")
}

func TestState_UnmarshalInvalidJSON(t *testing.T) {
	var s State
	err := json.Unmarshal([]byte(`123`), &s)
	require.Error(t, err)
}

func TestConcurrency_UnmarshalInvalidJSON(t *testing.T) {
	var c Concurrency
	err := json.Unmarshal([]byte(`123`), &c)
	require.Error(t, err)
}

func TestState_MarshalJSON(t *testing.T) {
	data, err := json.Marshal(StateActive)
	require.NoError(t, err)
	assert.JSONEq(t, `"active"`, string(data))
}

func TestConcurrency_MarshalJSON(t *testing.T) {
	data, err := json.Marshal(ConcurrencyUnspecified)
	require.NoError(t, err)
	assert.JSONEq(t, `""`, string(data))
}
