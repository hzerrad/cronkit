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

	// For simple patterns (minute-based with wildcard hours/days),
	// skip "every day" as it's implied
	minuteBasedPattern := (minute.IsEvery() || minute.IsStep() ||
		(minute.IsSingle() && minute.Value() == 0)) && hour.IsEvery()
	isSimplePattern := minuteBasedPattern && dayOfWeek.IsEvery() && dayOfMonth.IsEvery()

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
	// Case 1: Every minute (*, *)
	if minute.IsEvery() && hour.IsEvery() {
		return "Every minute"
	}

	// Case 2: Minute intervals with wildcard hour (*/N, *)
	if minute.IsStep() && hour.IsEvery() {
		return fmt.Sprintf("Every %d minutes", minute.Step())
	}

	// Case 3: Minute intervals within hour range (*/N, N-M)
	if minute.IsStep() && hour.IsRange() {
		return fmt.Sprintf("Every %d minutes between %s and %s",
			minute.Step(),
			formatHour(hour.RangeStart()),
			formatHourEnd(hour.RangeEnd()))
	}

	// Case 4: Start of every hour (0, *)
	if minute.IsSingle() && minute.Value() == 0 && hour.IsEvery() {
		return "At the start of every hour"
	}

	// Case 5: Specific minute of every hour (N, *)
	if minute.IsSingle() && hour.IsEvery() {
		return fmt.Sprintf("At minute %d of every hour", minute.Value())
	}

	// Case 6: Specific time (N, M)
	if minute.IsSingle() && hour.IsSingle() {
		if minute.Value() == 0 && hour.Value() == 0 {
			return "At midnight"
		}
		return fmt.Sprintf("At %s", formatTime(hour.Value(), minute.Value()))
	}

	// Case 7: Specific time with multiple hours (N, M,N,O)
	if minute.IsSingle() && hour.IsList() {
		times := make([]string, len(hour.ListValues()))
		for i, h := range hour.ListValues() {
			times[i] = formatTime(h, minute.Value())
		}
		return fmt.Sprintf("At %s", formatList(times))
	}

	// Case 8: Step minutes with single hour (*/N, M)
	if minute.IsStep() && hour.IsSingle() {
		return fmt.Sprintf("Every %d minutes at %s", minute.Step(), formatHour(hour.Value()))
	}

	// Case 9: Step minutes with list hour (*/N, M,N,O)
	if minute.IsStep() && hour.IsList() {
		times := make([]string, len(hour.ListValues()))
		for i, h := range hour.ListValues() {
			times[i] = formatHour(h)
		}
		return fmt.Sprintf("Every %d minutes at %s", minute.Step(), formatList(times))
	}

	// Case 10: Single minute with range hour (N, M-O)
	if minute.IsSingle() && hour.IsRange() {
		return fmt.Sprintf("At %d minutes past the hour between %s and %s",
			minute.Value(),
			formatHour(hour.RangeStart()),
			formatHourEnd(hour.RangeEnd()))
	}

	// Case 11: List minute with single hour (N,M,O, H)
	if minute.IsList() && hour.IsSingle() {
		times := make([]string, len(minute.ListValues()))
		for i, m := range minute.ListValues() {
			times[i] = formatTime(hour.Value(), m)
		}
		return fmt.Sprintf("At %s", formatList(times))
	}

	// Case 12: List minute with range hour (N,M,O, H-J)
	if minute.IsList() && hour.IsRange() {
		minutes := minute.ListValues()
		minuteStrs := make([]string, len(minutes))
		for i, m := range minutes {
			minuteStrs[i] = fmt.Sprintf("%d", m)
		}
		return fmt.Sprintf("At %s minutes past the hour between %s and %s",
			formatList(minuteStrs),
			formatHour(hour.RangeStart()),
			formatHourEnd(hour.RangeEnd()))
	}

	// Case 13: List minute with list hour (N,M,O, H,J,K) - cartesian product
	if minute.IsList() && hour.IsList() {
		times := h.generateTimeCombinations(minute.ListValues(), hour.ListValues())
		return fmt.Sprintf("At %s", formatList(times))
	}

	// Default fallback
	return "Runs periodically"
}

// generateTimeCombinations creates a cartesian product of minutes and hours
// and returns formatted time strings sorted by hour then minute
func (h *humanizer) generateTimeCombinations(minutes, hours []int) []string {
	var times []string
	for _, hour := range hours {
		for _, minute := range minutes {
			times = append(times, formatTime(hour, minute))
		}
	}
	return times
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
