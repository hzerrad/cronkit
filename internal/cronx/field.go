package cronx

import (
	"strconv"
	"strings"
)

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

// parseField parses a single cron field
func parseField(raw string, min, max int) Field {
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
			start, _ := strconv.Atoi(rangeParts[0])
			end, _ := strconv.Atoi(rangeParts[1])
			f.isRange = true
			f.rangeStart = start
			f.rangeEnd = end
		}
		return f
	}

	// Check for range (N-M)
	if strings.Contains(raw, "-") {
		parts := strings.Split(raw, "-")
		start, _ := strconv.Atoi(parts[0])
		end, _ := strconv.Atoi(parts[1])
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
			val, _ := strconv.Atoi(p)
			f.listValues[i] = val
		}
		return f
	}

	// Single value
	val, _ := strconv.Atoi(raw)
	f.isSingle = true
	f.value = val
	return f
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
