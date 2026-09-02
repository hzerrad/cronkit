package cronx

import (
	"fmt"
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

	// Kind returns the single classification describing this field's shape
	Kind() FieldKind
}

// fieldPart represents a component of a field (a single value, range, etc.)
type fieldPart struct {
	isEvery    bool
	isRange    bool
	rangeStart int
	rangeEnd   int
	isSingle   bool
	value      int
	step       int // 0 or 1 means no step, >1 means step notation
}

// field implements Field interface using composition of parts
type field struct {
	raw   string
	parts []fieldPart
	min   int
	max   int
}

// parseField parses a single cron field using a specific symbol registry
func parseField(raw string, min, max int, registry SymbolRegistry) (Field, error) {
	f := &field{
		raw: raw,
		min: min,
		max: max,
	}

	// Split by comma first - everything can be a list
	rawParts := strings.Split(raw, ",")
	for _, p := range rawParts {
		part, err := parsePart(strings.TrimSpace(p), registry)
		if err != nil {
			return nil, err
		}

		// "N/step" is shorthand for "N-max/step"
		if part.isSingle && part.step > 1 {
			part.isSingle = false
			part.isRange = true
			part.rangeStart = part.value
			part.rangeEnd = max
		}

		f.parts = append(f.parts, part)
	}

	return f, nil
}

// parsePart parses a single component of a field (handles *, ranges, steps, single values)
func parsePart(raw string, registry SymbolRegistry) (fieldPart, error) {
	part := fieldPart{step: 1} // Default: no step

	// Handle Step notation (/)
	if strings.Contains(raw, "/") {
		parts := strings.Split(raw, "/")
		if len(parts) != 2 {
			return fieldPart{}, fmt.Errorf("too many steps in %q", raw)
		}
		stepVal, err := strconv.Atoi(parts[1])
		if err != nil {
			return fieldPart{}, fmt.Errorf("unrecognized step %q", parts[1])
		}
		part.step = stepVal
		raw = parts[0] // Continue parsing the left side
	}

	// Handle Wildcard (* and ?, which cron reads alike)
	if raw == "*" || raw == "?" {
		part.isEvery = true
		return part, nil
	}

	// Handle Range (-)
	if strings.Contains(raw, "-") {
		rangeParts := strings.Split(raw, "-")
		if len(rangeParts) != 2 {
			return fieldPart{}, fmt.Errorf("too many bounds in %q", raw)
		}
		start, err := parseValue(rangeParts[0], registry)
		if err != nil {
			return fieldPart{}, err
		}
		end, err := parseValue(rangeParts[1], registry)
		if err != nil {
			return fieldPart{}, err
		}
		part.isRange = true
		part.rangeStart = start
		part.rangeEnd = end
		return part, nil
	}

	// Handle Single Value
	value, err := parseValue(raw, registry)
	if err != nil {
		return fieldPart{}, err
	}
	part.isSingle = true
	part.value = value
	return part, nil
}

// parseValue converts a string to an integer, supporting both numeric values and symbols
func parseValue(s string, registry SymbolRegistry) (int, error) {
	// Try parsing as integer first
	val, err := strconv.Atoi(s)
	if err == nil {
		return val, nil
	}

	// Try parsing as symbol (day/month name)
	if v, ok := registry.ParseSymbol(s); ok {
		return v, nil
	}

	return 0, fmt.Errorf("unrecognized value %q", s)
}

// IsEvery returns true if the field is "*" without any step (single part that is wildcard with no step)
func (f *field) IsEvery() bool {
	return len(f.parts) == 1 && f.parts[0].isEvery && f.parts[0].step <= 1
}

// IsStep returns true if the field has step notation
func (f *field) IsStep() bool {
	// A field has a step if any part has a step > 1
	for _, p := range f.parts {
		if p.step > 1 {
			return true
		}
	}
	return false
}

// Step returns the step value (for the first part with a step)
func (f *field) Step() int {
	for _, p := range f.parts {
		if p.step > 1 {
			return p.step
		}
	}
	return 1
}

// IsRange returns true if the field is a single range (not a list)
func (f *field) IsRange() bool {
	return len(f.parts) == 1 && f.parts[0].isRange
}

// RangeStart returns the start of the range (first part if it's a range)
func (f *field) RangeStart() int {
	if len(f.parts) > 0 && f.parts[0].isRange {
		return f.parts[0].rangeStart
	}
	return 0
}

// RangeEnd returns the end of the range (first part if it's a range)
func (f *field) RangeEnd() int {
	if len(f.parts) > 0 && f.parts[0].isRange {
		return f.parts[0].rangeEnd
	}
	return 0
}

// IsList returns true if the field has multiple parts
func (f *field) IsList() bool {
	return len(f.parts) > 1
}

// ListValues returns all values from all parts (expanded)
func (f *field) ListValues() []int {
	var values []int
	for _, p := range f.parts {
		step := p.step
		if step < 1 {
			step = 1
		}

		switch {
		case p.isSingle:
			values = append(values, p.value)
		case p.isRange:
			for i := p.rangeStart; i <= p.rangeEnd; i += step {
				values = append(values, i)
			}
		case p.isEvery && step > 1:
			for i := f.min; i <= f.max; i += step {
				values = append(values, i)
			}
		}
	}
	return values
}

// IsSingle returns true if the field is a single value (not a list, range, or wildcard)
func (f *field) IsSingle() bool {
	return len(f.parts) == 1 && f.parts[0].isSingle
}

// Value returns the single value (first part if it's a single value)
func (f *field) Value() int {
	if len(f.parts) > 0 && f.parts[0].isSingle {
		return f.parts[0].value
	}
	return 0
}

// Raw returns the raw field string
func (f *field) Raw() string {
	return f.raw
}
