package human

import (
	"fmt"
	"strings"

	"github.com/hzerrad/cronkit/internal/cronx"
)

// Humanizer converts cron schedules to human-readable descriptions
type Humanizer interface {
	Humanize(schedule *cronx.Schedule) string
}

type humanizer struct {
	// Could add locale/language support here in future
}

// NewHumanizer creates a new humanizer with English templates (v1)
func NewHumanizer() Humanizer {
	return &humanizer{}
}

// Humanize converts a parsed cron schedule to human-readable text
func (h *humanizer) Humanize(schedule *cronx.Schedule) string {
	var parts []string

	minute := schedule.Minute
	hour := schedule.Hour
	dayOfWeek := schedule.DayOfWeek
	dayOfMonth := schedule.DayOfMonth

	// Build the human-readable description by analyzing each field
	timePart := h.buildTimePart(minute, hour)
	dayPart := h.buildDayPart(dayOfWeek, dayOfMonth)
	monthPart := h.buildMonthPart(schedule.Month)

	parts = append(parts, timePart)

	// A wildcard hour already covers every day
	isSimplePattern := hour.Kind() == cronx.KindEvery && dayOfWeek.IsEvery() && dayOfMonth.IsEvery()

	// Special case: specific day + specific month (e.g., @yearly)
	month := schedule.Month
	if dayOfMonth.IsSingle() && month.IsSingle() && dayOfWeek.IsEvery() {
		parts = append(parts, fmt.Sprintf("on %s %d%s",
			formatMonth(month.Value()),
			dayOfMonth.Value(),
			ordinalSuffix(dayOfMonth.Value())))
		return strings.Join(parts, " ")
	}

	if dayPart != "" && !isSimplePattern {
		parts = append(parts, dayPart)
	}

	if monthPart != "" {
		parts = append(parts, monthPart)
	}

	return strings.Join(parts, " ")
}

// buildTimePart constructs the time portion of the description
func (h *humanizer) buildTimePart(minute, hour cronx.Field) string {
	minuteKind, hourKind := minute.Kind(), hour.Kind()

	if isClockPoint(minuteKind) && isClockPoint(hourKind) {
		return h.fusedClockTimes(minute, hour)
	}

	clause := h.minuteClause(minute, hourKind)
	if suffix := h.hourSuffix(hour, minuteKind); suffix != "" {
		return clause + " " + suffix
	}
	return clause
}

func isClockPoint(k cronx.FieldKind) bool {
	return k == cronx.KindSingle || k == cronx.KindList
}

func (h *humanizer) fusedClockTimes(minute, hour cronx.Field) string {
	if minute.Kind() == cronx.KindSingle && hour.Kind() == cronx.KindSingle &&
		minute.Value() == 0 && hour.Value() == 0 {
		return "At midnight"
	}

	var times []string
	for _, hr := range hour.ListValues() {
		for _, min := range minute.ListValues() {
			times = append(times, formatTime(hr, min))
		}
	}
	return fmt.Sprintf("At %s", formatList(times))
}

func (h *humanizer) minuteClause(minute cronx.Field, hourKind cronx.FieldKind) string {
	switch minute.Kind() {
	case cronx.KindEvery:
		return "Every minute"

	case cronx.KindStep:
		return fmt.Sprintf("Every %d minutes", minute.Step())

	case cronx.KindSingle:
		if minute.Value() == 0 && hourKind == cronx.KindEvery {
			return "At the start of every hour"
		}
		return fmt.Sprintf("At %d minutes past the hour", minute.Value())

	case cronx.KindRange:
		return fmt.Sprintf("At minutes %d through %d past the hour",
			minute.RangeStart(), minute.RangeEnd())

	default:
		return fmt.Sprintf("At %s minutes past the hour", formatList(formatInts(minute.ListValues())))
	}
}

func (h *humanizer) hourSuffix(hour cronx.Field, minuteKind cronx.FieldKind) string {
	switch hour.Kind() {
	case cronx.KindEvery:
		return ""

	case cronx.KindStep:
		return fmt.Sprintf("every %d hours", hour.Step())

	case cronx.KindRange:
		return fmt.Sprintf("between %s and %s",
			formatHour(hour.RangeStart()), formatHourEnd(hour.RangeEnd()))

	case cronx.KindBoundedStep:
		return fmt.Sprintf("every %d hours between %s and %s",
			hour.Step(), formatHour(hour.RangeStart()), formatHourEnd(hour.RangeEnd()))

	default:
		hours := hour.ListValues()
		formatted := make([]string, len(hours))
		for i, hr := range hours {
			formatted[i] = formatHour(hr)
		}
		if minuteKind == cronx.KindEvery {
			return fmt.Sprintf("during the %s %s", formatList(formatted), pluralize("hour", len(hours)))
		}
		return fmt.Sprintf("at %s", formatList(formatted))
	}
}

// buildDayPart constructs the day portion of the description
func (h *humanizer) buildDayPart(dayOfWeek, dayOfMonth cronx.Field) string {
	// If both are wildcard, return empty (every day)
	if dayOfWeek.IsEvery() && dayOfMonth.IsEvery() {
		return "every day"
	}

	// Day of week has priority
	if !dayOfWeek.IsEvery() {
		return h.formatDayOfWeek(dayOfWeek)
	}

	// Day of month
	if !dayOfMonth.IsEvery() {
		return h.formatDayOfMonth(dayOfMonth)
	}

	return "every day"
}

// buildMonthPart constructs the month portion of the description
func (h *humanizer) buildMonthPart(month cronx.Field) string {
	if month.IsEvery() {
		return ""
	}

	if month.IsSingle() {
		return fmt.Sprintf("in %s", formatMonth(month.Value()))
	}

	if month.IsRange() {
		return fmt.Sprintf("from %s to %s",
			formatMonth(month.RangeStart()),
			formatMonth(month.RangeEnd()))
	}

	if month.IsList() {
		months := make([]string, len(month.ListValues()))
		for i, m := range month.ListValues() {
			months[i] = formatMonth(m)
		}
		return fmt.Sprintf("in %s", formatList(months))
	}

	return ""
}

// formatDayOfWeek formats day of week field
func (h *humanizer) formatDayOfWeek(dow cronx.Field) string {
	if dow.IsRange() {
		// Special case for Mon-Fri (1-5)
		if dow.RangeStart() == 1 && dow.RangeEnd() == 5 {
			return "on weekdays (Mon-Fri)"
		}
		return fmt.Sprintf("on %s-%s",
			dayName(dow.RangeStart()),
			dayName(dow.RangeEnd()))
	}

	if dow.IsList() {
		days := make([]string, len(dow.ListValues()))
		for i, d := range dow.ListValues() {
			days[i] = dayName(d)
		}
		return fmt.Sprintf("on %s", formatList(days))
	}

	if dow.IsSingle() {
		// Special case for Sunday (0)
		if dow.Value() == 0 {
			return "every Sunday"
		}
		return fmt.Sprintf("every %s", dayName(dow.Value()))
	}

	return ""
}

// formatDayOfMonth formats day of month field
func (h *humanizer) formatDayOfMonth(dom cronx.Field) string {
	if dom.IsSingle() {
		if dom.Value() == 1 {
			return "on the first day of every month"
		}
		return fmt.Sprintf("on day %d of every month", dom.Value())
	}

	if dom.IsRange() {
		return fmt.Sprintf("on days %d-%d of every month",
			dom.RangeStart(), dom.RangeEnd())
	}

	return ""
}
