package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveTimezones_UnresolvableZoneBecomesInvalid(t *testing.T) {
	items := []Item{
		{Expression: "* * * * *", Timezone: "Not/AZone", State: StateActive, Locator: Locator{Line: 1}},
	}

	got := ResolveTimezones(items)

	require := assert.New(t)
	require.Len(got, 1)
	require.Equal(StateInvalid, got[0].State)
	require.Contains(got[0].Reason, "Not/AZone")
}

func TestResolveTimezones_EmptyZoneStaysActive(t *testing.T) {
	items := []Item{
		{Expression: "* * * * *", State: StateActive, Locator: Locator{Line: 1}},
	}

	got := ResolveTimezones(items)

	assert.Equal(t, StateActive, got[0].State)
	assert.Empty(t, got[0].Reason)
}

func TestResolveTimezones_ValidZoneStaysActive(t *testing.T) {
	items := []Item{
		{Expression: "* * * * *", Timezone: "Europe/Paris", State: StateActive, Locator: Locator{Line: 1}},
	}

	got := ResolveTimezones(items)

	assert.Equal(t, StateActive, got[0].State)
	assert.Empty(t, got[0].Reason)
}

func TestResolveTimezones_NonActiveItemsUntouched(t *testing.T) {
	// A suspended or already-invalid item is left alone: its Timezone is not
	// consulted, and its existing Reason must not be overwritten.
	items := []Item{
		{Expression: "* * * * *", Timezone: "Not/AZone", State: StateSuspended, Reason: "spec.suspend is true"},
		{Expression: "bad", State: StateInvalid, Reason: "unparseable expression"},
	}

	got := ResolveTimezones(items)

	assert.Equal(t, StateSuspended, got[0].State)
	assert.Equal(t, "spec.suspend is true", got[0].Reason)
	assert.Equal(t, StateInvalid, got[1].State)
	assert.Equal(t, "unparseable expression", got[1].Reason)
}

func TestResolveTimezones_DoesNotMutateInput(t *testing.T) {
	items := []Item{
		{Expression: "* * * * *", Timezone: "Not/AZone", State: StateActive},
	}

	_ = ResolveTimezones(items)

	assert.Equal(t, StateActive, items[0].State, "the original slice must not be mutated")
}

func TestResolveTimezones_EmptyInput(t *testing.T) {
	assert.Empty(t, ResolveTimezones(nil))
}

func TestIsUnresolvableTimezone(t *testing.T) {
	t.Run("true for an item ResolveTimezones invalidated", func(t *testing.T) {
		items := []Item{{Expression: "* * * * *", Timezone: "Not/AZone", State: StateActive}}
		resolved := ResolveTimezones(items)
		assert.True(t, IsUnresolvableTimezone(resolved[0]))
	})

	t.Run("false for an item invalid for some other reason", func(t *testing.T) {
		assert.False(t, IsUnresolvableTimezone(Item{Expression: "bad", State: StateInvalid, Reason: "unparseable expression"}))
	})

	t.Run("false for an active item", func(t *testing.T) {
		assert.False(t, IsUnresolvableTimezone(Item{Expression: "* * * * *", State: StateActive}))
	})

	t.Run("false for a suspended item", func(t *testing.T) {
		assert.False(t, IsUnresolvableTimezone(Item{Expression: "* * * * *", State: StateSuspended, Reason: "spec.suspend is true"}))
	})
}
