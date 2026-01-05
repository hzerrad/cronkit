package cronx

import (
	"fmt"
	"strings"
	"sync"

	"github.com/robfig/cron/v3"
)

// Schedule represents a parsed cron schedule with field information.
type Schedule struct {
	Original   string // The original cron expression string
	Minute     Field  // Minute field (MinMinute-MaxMinute)
	Hour       Field  // Hour field (MinHour-MaxHour)
	DayOfMonth Field  // Day of month field (MinDayOfMonth-MaxDayOfMonth)
	Month      Field  // Month field (MinMonth-MaxMonth)
	DayOfWeek  Field  // Day of week field (MinDayOfWeek-MaxDayOfWeek, Sunday=0)
}

// Parser is the abstraction layer for cron expression parsing
type Parser interface {
	Parse(expression string) (*Schedule, error)
}

// parser implements Parser interface
type parser struct {
	cronParser cron.Parser
	symbols    SymbolRegistry
	cache      map[string]*Schedule
	cacheMu    sync.RWMutex
}

// NewParser creates a new cron expression parser with English locale (default)
func NewParser() Parser {
	return NewParserWithLocale("en")
}

// NewParserWithLocale creates a new cron expression parser with a specific locale
func NewParserWithLocale(locale string) Parser {
	symbols, _ := GetSymbolRegistry(locale)
	return &parser{
		cronParser: cron.NewParser(
			cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
		),
		symbols: symbols,
		cache:   make(map[string]*Schedule),
	}
}

// Parse parses a cron expression (5-field format or @alias)
// Results are cached to improve performance when parsing the same expression multiple times
func (p *parser) Parse(expression string) (*Schedule, error) {
	if expression == "" {
		return nil, fmt.Errorf("empty expression")
	}

	// Check cache first (read lock)
	p.cacheMu.RLock()
	if cached, ok := p.cache[expression]; ok {
		p.cacheMu.RUnlock()
		return cached, nil
	}
	p.cacheMu.RUnlock()

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

	schedule := &Schedule{
		Original:   original,
		Minute:     parseField(fields[0], MinMinute, MaxMinute, p.symbols),
		Hour:       parseField(fields[1], MinHour, MaxHour, p.symbols),
		DayOfMonth: parseField(fields[2], MinDayOfMonth, MaxDayOfMonth, p.symbols),
		Month:      parseField(fields[3], MinMonth, MaxMonth, p.symbols),
		DayOfWeek:  parseField(fields[4], MinDayOfWeek, MaxDayOfWeek, p.symbols),
	}

	// Cache the result (write lock)
	p.cacheMu.Lock()
	p.cache[expression] = schedule
	p.cacheMu.Unlock()

	return schedule, nil
}

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
