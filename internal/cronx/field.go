package cronx

import (
	"strconv"
	"strings"
)

// Field represents a single cron field (minute, hour, etc.)
type Field interface {
	// IsEvery returns true if field is "*" (every value)
	IsEvery() bool

	// IsStep returns true if field has step notation (*/N)
	IsStep() bool

	// Step returns the step value (e.g., 15 for "*/15")
	Step() int

	// IsRange returns true if field is a range (e.g., "1-5")
	IsRange() bool

	// RangeStart returns the start of a range
	RangeStart() int

	// RangeEnd returns the end of a range
	RangeEnd() int

	// IsList returns true if field is a comma-separated list
	IsList() bool

	// ListValues returns the list values
	ListValues() []int

	// IsSingle returns true if field is a single value
	IsSingle() bool

	// Value returns the single value
	Value() int

	// Raw returns the raw field string
	Raw() string
}

// field implements Field interface
type field struct {
	raw        string
	min        int
	max        int
	isEvery    bool
	isStep     bool
	step       int
	isRange    bool
	rangeStart int
	rangeEnd   int
	isList     bool
	listValues []int
	isSingle   bool
	value      int
}

// parseField parses a single cron field using a specific symbol registry
func parseField(raw string, min, max int, registry SymbolRegistry) Field {
	f := &field{
		raw: raw,
		min: min,
		max: max,
	}

	// Check for wildcard (every)
	if raw == "*" {
		f.isEvery = true
		return f
	}

	// Check for step notation (*/N or N-M/S)
	if strings.Contains(raw, "/") {
		parts := strings.Split(raw, "/")
		stepVal, _ := strconv.Atoi(parts[1])
		f.isStep = true
		f.step = stepVal

		// Check if it's a range with step (N-M/S)
		if strings.Contains(parts[0], "-") && parts[0] != "*" {
			rangeParts := strings.Split(parts[0], "-")
			start := parseValue(rangeParts[0], registry)
			end := parseValue(rangeParts[1], registry)
			f.isRange = true
			f.rangeStart = start
			f.rangeEnd = end
		}
		return f
	}

	// Check for range (N-M)
	if strings.Contains(raw, "-") {
		parts := strings.Split(raw, "-")
		start := parseValue(parts[0], registry)
		end := parseValue(parts[1], registry)
		f.isRange = true
		f.rangeStart = start
		f.rangeEnd = end
		return f
	}

	// Check for list (N,M,O)
	if strings.Contains(raw, ",") {
		parts := strings.Split(raw, ",")
		f.isList = true
		f.listValues = make([]int, len(parts))
		for i, p := range parts {
			f.listValues[i] = parseValue(p, registry)
		}
		return f
	}

	// Single value
	val := parseValue(raw, registry)
	f.isSingle = true
	f.value = val
	return f
}

// parseValue converts a string to an integer, supporting both numeric values and symbols
func parseValue(s string, registry SymbolRegistry) int {
	// Try parsing as integer first
	val, err := strconv.Atoi(s)
	if err == nil {
		return val
	}

	// Try parsing as symbol (day/month name)
	if v, ok := registry.ParseSymbol(s); ok {
		return v
	}

	// Return 0 if unable to parse
	return 0
}

func (f *field) IsEvery() bool     { return f.isEvery }
func (f *field) IsStep() bool      { return f.isStep }
func (f *field) Step() int         { return f.step }
func (f *field) IsRange() bool     { return f.isRange }
func (f *field) RangeStart() int   { return f.rangeStart }
func (f *field) RangeEnd() int     { return f.rangeEnd }
func (f *field) IsList() bool      { return f.isList }
func (f *field) ListValues() []int { return f.listValues }
func (f *field) IsSingle() bool    { return f.isSingle }
func (f *field) Value() int        { return f.value }
func (f *field) Raw() string       { return f.raw }
