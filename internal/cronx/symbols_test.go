package cronx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSymbolRegistry_ParseSymbol_DayNames tests parsing day names
func TestSymbolRegistry_ParseSymbol_DayNames(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		found    bool
	}{
		{
			name:     "Sunday uppercase",
			input:    "SUN",
			expected: 0,
			found:    true,
		},
		{
			name:     "Sunday lowercase",
			input:    "sun",
			expected: 0,
			found:    true,
		},
		{
			name:     "Sunday mixed case",
			input:    "Sun",
			expected: 0,
			found:    true,
		},
		{
			name:     "Monday",
			input:    "MON",
			expected: 1,
			found:    true,
		},
		{
			name:     "Tuesday",
			input:    "TUE",
			expected: 2,
			found:    true,
		},
		{
			name:     "Wednesday",
			input:    "WED",
			expected: 3,
			found:    true,
		},
		{
			name:     "Thursday",
			input:    "THU",
			expected: 4,
			found:    true,
		},
		{
			name:     "Friday",
			input:    "FRI",
			expected: 5,
			found:    true,
		},
		{
			name:     "Saturday",
			input:    "SAT",
			expected: 6,
			found:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, found := DefaultSymbolRegistry.ParseSymbol(tt.input)
			assert.Equal(t, tt.found, found)
			if found {
				assert.Equal(t, tt.expected, value)
			}
		})
	}
}

// TestSymbolRegistry_ParseSymbol_MonthNames tests parsing month names
func TestSymbolRegistry_ParseSymbol_MonthNames(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		found    bool
	}{
		{
			name:     "January uppercase",
			input:    "JAN",
			expected: 1,
			found:    true,
		},
		{
			name:     "January lowercase",
			input:    "jan",
			expected: 1,
			found:    true,
		},
		{
			name:     "January mixed case",
			input:    "Jan",
			expected: 1,
			found:    true,
		},
		{
			name:     "February",
			input:    "FEB",
			expected: 2,
			found:    true,
		},
		{
			name:     "March",
			input:    "MAR",
			expected: 3,
			found:    true,
		},
		{
			name:     "April",
			input:    "APR",
			expected: 4,
			found:    true,
		},
		{
			name:     "May",
			input:    "MAY",
			expected: 5,
			found:    true,
		},
		{
			name:     "June",
			input:    "JUN",
			expected: 6,
			found:    true,
		},
		{
			name:     "July",
			input:    "JUL",
			expected: 7,
			found:    true,
		},
		{
			name:     "August",
			input:    "AUG",
			expected: 8,
			found:    true,
		},
		{
			name:     "September",
			input:    "SEP",
			expected: 9,
			found:    true,
		},
		{
			name:     "October",
			input:    "OCT",
			expected: 10,
			found:    true,
		},
		{
			name:     "November",
			input:    "NOV",
			expected: 11,
			found:    true,
		},
		{
			name:     "December",
			input:    "DEC",
			expected: 12,
			found:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, found := DefaultSymbolRegistry.ParseSymbol(tt.input)
			assert.Equal(t, tt.found, found)
			if found {
				assert.Equal(t, tt.expected, value)
			}
		})
	}
}

// TestSymbolRegistry_ParseSymbol_InvalidSymbols tests invalid symbol handling
func TestSymbolRegistry_ParseSymbol_InvalidSymbols(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Invalid day name",
			input: "SUNDAY",
		},
		{
			name:  "Invalid month name",
			input: "JANUARY",
		},
		{
			name:  "Random string",
			input: "INVALID",
		},
		{
			name:  "Empty string",
			input: "",
		},
		{
			name:  "Numeric string",
			input: "123",
		},
		{
			name:  "Special characters",
			input: "@#$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, found := DefaultSymbolRegistry.ParseSymbol(tt.input)
			assert.False(t, found)
			assert.Equal(t, 0, value)
		})
	}
}

// TestSymbolRegistry_Locale tests the Locale method
func TestSymbolRegistry_Locale(t *testing.T) {
	locale := DefaultSymbolRegistry.Locale()
	assert.Equal(t, "en", locale)
}

// TestNewSymbolRegistry tests creating a custom symbol registry
func TestNewSymbolRegistry(t *testing.T) {
	t.Run("create custom registry", func(t *testing.T) {
		customDays := map[string]int{
			"DIM": 0, // French: Dimanche
			"LUN": 1, // French: Lundi
		}
		customMonths := map[string]int{
			"JAN": 1, // Janvier
			"FEV": 2, // Février
		}

		registry := NewSymbolRegistry("fr", customDays, customMonths)

		require.NotNil(t, registry)
		assert.Equal(t, "fr", registry.Locale())

		// Test custom day
		value, found := registry.ParseSymbol("DIM")
		assert.True(t, found)
		assert.Equal(t, 0, value)

		// Test custom month
		value, found = registry.ParseSymbol("FEV")
		assert.True(t, found)
		assert.Equal(t, 2, value)

		// Test non-existent symbol
		_, found = registry.ParseSymbol("MON")
		assert.False(t, found)
	})

	t.Run("case insensitivity", func(t *testing.T) {
		customDays := map[string]int{
			"DIM": 0,
		}
		customMonths := map[string]int{
			"JAN": 1,
		}

		registry := NewSymbolRegistry("test", customDays, customMonths)

		// Should work with different cases
		tests := []string{"DIM", "dim", "Dim", "DiM"}
		for _, input := range tests {
			value, found := registry.ParseSymbol(input)
			assert.True(t, found, "Should find %s", input)
			assert.Equal(t, 0, value)
		}
	})
}

// TestGetSymbolRegistry tests getting symbol registries by locale
func TestGetSymbolRegistry(t *testing.T) {
	t.Run("get English registry", func(t *testing.T) {
		registry, ok := GetSymbolRegistry("en")
		require.True(t, ok)
		assert.NotNil(t, registry)
		assert.Equal(t, "en", registry.Locale())

		// Verify it works
		value, found := registry.ParseSymbol("MON")
		assert.True(t, found)
		assert.Equal(t, 1, value)
	})

	t.Run("fallback to default for unknown locale", func(t *testing.T) {
		registry, ok := GetSymbolRegistry("fr")
		assert.False(t, ok, "Should indicate locale not found")
		require.NotNil(t, registry, "Should still return default registry")
		assert.Equal(t, "en", registry.Locale(), "Should fallback to English")

		// Verify it works with English symbols
		value, found := registry.ParseSymbol("MON")
		assert.True(t, found)
		assert.Equal(t, 1, value)
	})

	t.Run("fallback to default for empty locale", func(t *testing.T) {
		registry, ok := GetSymbolRegistry("")
		assert.False(t, ok)
		require.NotNil(t, registry)
		assert.Equal(t, "en", registry.Locale())
	})
}

// TestDefaultSymbolRegistry_Completeness tests that all expected symbols are present
func TestDefaultSymbolRegistry_Completeness(t *testing.T) {
	t.Run("all days of week present", func(t *testing.T) {
		expectedDays := []struct {
			name  string
			value int
		}{
			{"SUN", 0},
			{"MON", 1},
			{"TUE", 2},
			{"WED", 3},
			{"THU", 4},
			{"FRI", 5},
			{"SAT", 6},
		}

		for _, day := range expectedDays {
			value, found := DefaultSymbolRegistry.ParseSymbol(day.name)
			assert.True(t, found, "Day %s should be found", day.name)
			assert.Equal(t, day.value, value, "Day %s should have value %d", day.name, day.value)
		}
	})

	t.Run("all months present", func(t *testing.T) {
		expectedMonths := []struct {
			name  string
			value int
		}{
			{"JAN", 1},
			{"FEB", 2},
			{"MAR", 3},
			{"APR", 4},
			{"MAY", 5},
			{"JUN", 6},
			{"JUL", 7},
			{"AUG", 8},
			{"SEP", 9},
			{"OCT", 10},
			{"NOV", 11},
			{"DEC", 12},
		}

		for _, month := range expectedMonths {
			value, found := DefaultSymbolRegistry.ParseSymbol(month.name)
			assert.True(t, found, "Month %s should be found", month.name)
			assert.Equal(t, month.value, value, "Month %s should have value %d", month.name, month.value)
		}
	})
}

// TestSymbolRegistryMap_Contains tests the global registry map
func TestSymbolRegistryMap_Contains(t *testing.T) {
	t.Run("contains English registry", func(t *testing.T) {
		registry, ok := SymbolRegistryMap["en"]
		require.True(t, ok, "Map should contain 'en' key")
		assert.NotNil(t, registry)
		assert.Equal(t, "en", registry.Locale())
	})

	t.Run("English registry same as default", func(t *testing.T) {
		assert.Equal(t, DefaultSymbolRegistry, SymbolRegistryMap["en"])
	})
}

// TestSymbolRegistry_EdgeCases tests edge cases and boundary conditions
func TestSymbolRegistry_EdgeCases(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		value, found := DefaultSymbolRegistry.ParseSymbol("")
		assert.False(t, found)
		assert.Equal(t, 0, value)
	})

	t.Run("whitespace only", func(t *testing.T) {
		value, found := DefaultSymbolRegistry.ParseSymbol("   ")
		assert.False(t, found)
		assert.Equal(t, 0, value)
	})

	t.Run("with surrounding whitespace", func(t *testing.T) {
		// Note: Current implementation doesn't trim whitespace
		// This documents the current behavior
		value, found := DefaultSymbolRegistry.ParseSymbol(" MON ")
		assert.False(t, found)
		assert.Equal(t, 0, value)
	})

	t.Run("partial match", func(t *testing.T) {
		// Should not match partial strings
		value, found := DefaultSymbolRegistry.ParseSymbol("MO")
		assert.False(t, found)
		assert.Equal(t, 0, value)
	})

	t.Run("full day name instead of abbreviation", func(t *testing.T) {
		value, found := DefaultSymbolRegistry.ParseSymbol("MONDAY")
		assert.False(t, found)
		assert.Equal(t, 0, value)
	})
}
