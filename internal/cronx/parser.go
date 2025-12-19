package cronx

import (
	"fmt"
	"strings"

	"github.com/robfig/cron/v3"
)

// Schedule represents a parsed cron schedule with field information
type Schedule interface {
	Original() string
	Minute() Field
	Hour() Field
	DayOfMonth() Field
	Month() Field
	DayOfWeek() Field
}

// Parser is the abstraction layer for cron expression parsing
type Parser interface {
	Parse(expression string) (Schedule, error)
}

// parser implements Parser interface
type parser struct {
	cronParser cron.Parser
	symbols    SymbolRegistry
}

// NewParser creates a new cron expression parser
func NewParser() Parser {
	// Load English locale for now
	symbols, _ := GetSymbolRegistry("en")
	return &parser{
		cronParser: cron.NewParser(
			cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
		),
		symbols: symbols,
	}
}

// Parse parses a cron expression (5-field format or @alias)
func (p *parser) Parse(expression string) (Schedule, error) {
	if expression == "" {
		return nil, fmt.Errorf("empty expression")
	}

	// Store original for reference
	original := expression

	// Don't normalize aliases - robfig/cron expects them as-is
	normalized := expression
	if !strings.HasPrefix(expression, "@") {
		// Only normalize case for regular expressions (for day/month names)
		normalized = strings.ToUpper(expression)
	}

	// Use robfig/cron to parse (BOUNDARY: only place we call external library)
	_, err := p.cronParser.Parse(normalized)
	if err != nil {
		// Simplify error messages for expected cases
		errStr := err.Error()
		if strings.Contains(errStr, "expected exactly 5 fields") {
			return nil, fmt.Errorf("expected 5 fields")
		}
		if strings.Contains(errStr, "above maximum") || strings.Contains(errStr, "below minimum") {
			return nil, fmt.Errorf("value out of range: %w", err)
		}
		return nil, fmt.Errorf("failed to parse expression: %w", err)
	}

	// Parse individual fields
	var fields []string
	if strings.HasPrefix(expression, "@") {
		// Handle aliases (which robfig expands internally)
		fields = aliasToFields(expression)
	} else {
		fields = strings.Fields(normalized)
		if len(fields) != 5 {
			return nil, fmt.Errorf("expected 5 fields, got %d", len(fields))
		}
	}

	return &schedule{
		original:   original,
		minute:     parseField(fields[0], 0, 59, p.symbols),
		hour:       parseField(fields[1], 0, 23, p.symbols),
		dayOfMonth: parseField(fields[2], 1, 31, p.symbols),
		month:      parseField(fields[3], 1, 12, p.symbols),
		dayOfWeek:  parseField(fields[4], 0, 6, p.symbols),
	}, nil
}

// schedule implements Schedule interface
type schedule struct {
	original   string
	minute     Field
	hour       Field
	dayOfMonth Field
	month      Field
	dayOfWeek  Field
}

func (s *schedule) Original() string  { return s.original }
func (s *schedule) Minute() Field     { return s.minute }
func (s *schedule) Hour() Field       { return s.hour }
func (s *schedule) DayOfMonth() Field { return s.dayOfMonth }
func (s *schedule) Month() Field      { return s.month }
func (s *schedule) DayOfWeek() Field  { return s.dayOfWeek }

// aliasToFields converts cron aliases to field representation
func aliasToFields(alias string) []string {
	switch strings.ToLower(alias) {
	case "@yearly", "@annually":
		return []string{"0", "0", "1", "1", "*"}
	case "@monthly":
		return []string{"0", "0", "1", "*", "*"}
	case "@weekly":
		return []string{"0", "0", "*", "*", "0"}
	case "@daily", "@midnight":
		return []string{"0", "0", "*", "*", "*"}
	case "@hourly":
		return []string{"0", "*", "*", "*", "*"}
	default:
		return []string{"*", "*", "*", "*", "*"} // fallback
	}
}
