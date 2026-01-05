package cronx

import "strings"

// SymbolRegistry provides locale-specific mappings for day and month names
type SymbolRegistry interface {
	// ParseSymbol attempts to parse a symbol (day/month name) to its numeric value
	ParseSymbol(s string) (int, bool)

	// Locale returns the locale identifier (e.g., "en", "fr", "es")
	Locale() string
}

// symbolRegistry is the default implementation
type symbolRegistry struct {
	locale     string
	dayNames   map[string]int
	monthNames map[string]int
}

// NewSymbolRegistry creates a new symbol registry with the given mappings
func NewSymbolRegistry(locale string, dayNames, monthNames map[string]int) SymbolRegistry {
	return &symbolRegistry{
		locale:     locale,
		dayNames:   dayNames,
		monthNames: monthNames,
	}
}

// ParseSymbol looks up a symbol in both day and month mappings
func (r *symbolRegistry) ParseSymbol(s string) (int, bool) {
	upperS := strings.ToUpper(s)

	// Try day names first
	if v, ok := r.dayNames[upperS]; ok {
		return v, true
	}

	// Then try month names
	if v, ok := r.monthNames[upperS]; ok {
		return v, true
	}

	return 0, false
}

// Locale returns the locale identifier
func (r *symbolRegistry) Locale() string {
	return r.locale
}

// DefaultSymbolRegistry returns the English symbol registry
var DefaultSymbolRegistry = NewSymbolRegistry(
	"en",
	map[string]int{
		"SUN": 0,
		"MON": 1,
		"TUE": 2,
		"WED": 3,
		"THU": 4,
		"FRI": 5,
		"SAT": 6,
	},
	map[string]int{
		"JAN": 1,
		"FEB": 2,
		"MAR": 3,
		"APR": 4,
		"MAY": 5,
		"JUN": 6,
		"JUL": 7,
		"AUG": 8,
		"SEP": 9,
		"OCT": 10,
		"NOV": 11,
		"DEC": 12,
	},
)

// SymbolRegistryMap holds all available symbol registries by locale
var SymbolRegistryMap = map[string]SymbolRegistry{
	"en": DefaultSymbolRegistry,
	// Future locales can be added here:
	// "fr": FrenchSymbolRegistry,
	// "es": SpanishSymbolRegistry,
}

// GetSymbolRegistry returns a symbol registry for the given locale
// Falls back to English if the locale is not found
func GetSymbolRegistry(locale string) (SymbolRegistry, bool) {
	if registry, ok := SymbolRegistryMap[locale]; ok {
		return registry, true
	}
	return DefaultSymbolRegistry, false
}
