package cronx

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Schedule represents a parsed cron schedule with field information.
type Schedule struct {
	Original   string        // The original cron expression string
	Kind       ScheduleKind  // What shape of schedule this is
	Every      time.Duration // Repeat interval; set only when Kind is KindInterval
	Minute     Field         // Minute field, valid only when Kind is KindFields
	Hour       Field         // Hour field (MinHour-MaxHour)
	DayOfMonth Field         // Day of month field (MinDayOfMonth-MaxDayOfMonth)
	Month      Field         // Month field (MinMonth-MaxMonth)
	DayOfWeek  Field         // Day of week field (MinDayOfWeek-MaxDayOfWeek, Sunday=0)
}

// Parser is the abstraction layer for cron expression parsing
type Parser interface {
	Parse(expression string) (*Schedule, error)
}

// fieldSpec names a positional cron field and the value range it accepts
type fieldSpec struct {
	name string
	min  int
	max  int
}

// fieldSpecs lists the five cron fields in the order they appear in an expression
var fieldSpecs = []fieldSpec{
	{"minute", MinMinute, MaxMinute},
	{"hour", MinHour, MaxHour},
	{"day-of-month", MinDayOfMonth, MaxDayOfMonth},
	{"month", MinMonth, MaxMonth},
	{"day-of-week", MinDayOfWeek, MaxDayOfWeek},
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

// parseRobfig recovers a panic from robfig/cron's parser, since cronx parses untrusted input.
func parseRobfig(cp cron.Parser, expression string) (sched cron.Schedule, err error) {
	defer func() {
		if r := recover(); r != nil {
			sched = nil
			err = fmt.Errorf("failed to parse expression %q: %v", expression, r)
		}
	}()
	return cp.Parse(expression)
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

	// body strips a leading CRON_TZ=/TZ= assignment, which robfig treats as a timezone override.
	body := stripTZPrefix(expression)
	if body == "" {
		return nil, fmt.Errorf("%q sets a timezone (CRON_TZ=/TZ=) but has no schedule after it", expression)
	}

	// @reboot and @every do not decompose into fields, so they're handled before the field parser.
	if kind, every, ok, err := parseDescriptor(body); ok {
		if err != nil {
			return nil, err
		}
		schedule := &Schedule{Original: original, Kind: kind, Every: every}
		p.cacheMu.Lock()
		p.cache[expression] = schedule
		p.cacheMu.Unlock()
		return schedule, nil
	}

	// robfig matches named aliases case-sensitively, so lower-case the alias before validating.
	toValidate := expression
	if strings.HasPrefix(body, "@") {
		idx := strings.Index(expression, body)
		toValidate = expression[:idx] + strings.ToLower(body) + expression[idx+len(body):]
	}
	_, err := parseRobfig(p.cronParser, toValidate)
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
	if strings.HasPrefix(body, "@") {
		fields, err = aliasToFields(body)
		if err != nil {
			return nil, err
		}
	} else {
		fields = strings.Fields(body)
		if len(fields) != 5 {
			return nil, fmt.Errorf("expected 5 fields, got %d", len(fields))
		}
	}

	parsed := make([]Field, len(fieldSpecs))
	for i, spec := range fieldSpecs {
		f, err := parseField(fields[i], spec.min, spec.max, p.symbols)
		// robfig accepted the expression, so this only fires when our own decoder disagrees
		if err != nil {
			return nil, fmt.Errorf("%s field: %w", spec.name, err)
		}
		parsed[i] = f
	}

	schedule := &Schedule{
		Original:   original,
		Kind:       KindFields,
		Minute:     parsed[0],
		Hour:       parsed[1],
		DayOfMonth: parsed[2],
		Month:      parsed[3],
		DayOfWeek:  parsed[4],
	}

	// Cache the result (write lock)
	p.cacheMu.Lock()
	p.cache[expression] = schedule
	p.cacheMu.Unlock()

	return schedule, nil
}

// tzPrefixes are the CRON_TZ=/TZ= timezone-override prefixes robfig/cron recognises.
var tzPrefixes = []string{"CRON_TZ=", "TZ="}

// SplitTZPrefix splits a leading CRON_TZ=/TZ= assignment from expression into its zone and remaining text.
func SplitTZPrefix(expression string) (zone, rest string, ok bool) {
	for _, prefix := range tzPrefixes {
		if !strings.HasPrefix(expression, prefix) {
			continue
		}
		body := expression[len(prefix):]
		i := strings.Index(body, " ")
		if i == -1 {
			return StripQuotes(body), "", true
		}
		return StripQuotes(body[:i]), strings.TrimSpace(body[i:]), true
	}
	return "", expression, false
}

// StripQuotes removes one matching pair of leading/trailing quotes from s, leaving a mismatched quote as-is.
func StripQuotes(s string) string {
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// stripTZPrefix removes a leading CRON_TZ=/TZ= assignment from expression.
func stripTZPrefix(expression string) string {
	_, rest, _ := SplitTZPrefix(expression)
	return rest
}

// aliasToFields converts a named alias to its field representation, erroring on anything unrecognised.
func aliasToFields(alias string) ([]string, error) {
	switch strings.ToLower(alias) {
	case "@yearly", "@annually":
		return []string{"0", "0", "1", "1", "*"}, nil
	case "@monthly":
		return []string{"0", "0", "1", "*", "*"}, nil
	case "@weekly":
		return []string{"0", "0", "*", "*", "0"}, nil
	case "@daily", "@midnight":
		return []string{"0", "0", "*", "*", "*"}, nil
	case "@hourly":
		return []string{"0", "*", "*", "*", "*"}, nil
	default:
		return nil, fmt.Errorf("unrecognized descriptor: %s", alias)
	}
}

// parseDescriptor recognises @reboot and @every <duration>, validating @every's duration itself.
func parseDescriptor(body string) (kind ScheduleKind, every time.Duration, ok bool, err error) {
	lower := strings.ToLower(body)

	if lower == "@reboot" {
		return KindReboot, 0, true, nil
	}

	const prefix = "@every"
	if lower == prefix || strings.HasPrefix(lower, prefix+" ") {
		rest := strings.TrimSpace(body[len(prefix):])
		if rest == "" {
			return 0, 0, true, fmt.Errorf("@every requires a duration, e.g. @every 1h30m")
		}
		d, perr := time.ParseDuration(rest)
		if perr != nil {
			return 0, 0, true, fmt.Errorf("invalid @every duration %q: %w", rest, perr)
		}
		if d <= 0 {
			return 0, 0, true, fmt.Errorf("@every duration must be positive, got %q", rest)
		}
		return KindInterval, d, true, nil
	}

	return 0, 0, false, nil
}
