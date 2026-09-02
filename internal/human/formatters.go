package human

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// formatHour formats hour as HH:00
func formatHour(hour int) string {
	return fmt.Sprintf("%02d:00", hour)
}

// formatHourEnd formats hour as HH:59 (end of hour range)
func formatHourEnd(hour int) string {
	return fmt.Sprintf("%02d:59", hour)
}

// formatTime formats hour and minute as HH:MM
func formatTime(hour, minute int) string {
	return fmt.Sprintf("%02d:%02d", hour, minute)
}

// formatList formats a slice of strings with Oxford comma
func formatList(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return fmt.Sprintf("%s and %s", items[0], items[1])
	default:
		last := items[len(items)-1]
		rest := items[:len(items)-1]
		return fmt.Sprintf("%s, and %s", strings.Join(rest, ", "), last)
	}
}

func formatInts(values []int) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = strconv.Itoa(v)
	}
	return out
}

func pluralize(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "s"
}

// formatInterval renders an @every duration as a plain English phrase (e.g.
// "Every hour", "Every 30 minutes"); a positive duration under a second is
// clamped up to exactly one second first, mirroring robfig/cron's own ConstantDelaySchedule.
func formatInterval(d time.Duration) string {
	if d > 0 && d < time.Second {
		d = time.Second
	}

	hours := int(d / time.Hour)
	d -= time.Duration(hours) * time.Hour
	minutes := int(d / time.Minute)
	d -= time.Duration(minutes) * time.Minute
	seconds := int(d / time.Second)

	units := []struct {
		count int
		name  string
	}{
		{hours, "hour"},
		{minutes, "minute"},
		{seconds, "second"},
	}

	present := 0
	for _, u := range units {
		if u.count > 0 {
			present++
		}
	}

	var parts []string
	for _, u := range units {
		switch {
		case u.count == 0:
			continue
		case present == 1 && u.count == 1:
			parts = append(parts, u.name)
		default:
			parts = append(parts, fmt.Sprintf("%d %s", u.count, pluralize(u.name, u.count)))
		}
	}

	if len(parts) == 0 {
		return "Every moment"
	}
	return "Every " + strings.Join(parts, " ")
}

// dayName returns the name for a day of week (0=Sunday, 6=Saturday)
func dayName(day int) string {
	days := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	if day >= 0 && day < len(days) {
		return days[day]
	}
	return fmt.Sprintf("day%d", day)
}

// formatMonth returns the name for a month (1=January, 12=December)
func formatMonth(month int) string {
	months := []string{
		"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}
	if month >= 1 && month <= 12 {
		return months[month-1]
	}
	return fmt.Sprintf("month%d", month)
}

// ordinalSuffix returns the ordinal suffix for a day number (1st, 2nd, 3rd, etc.)
func ordinalSuffix(day int) string {
	// Numbers ending in 11, 12, or 13 always use "th" (e.g., 11th, 12th, 13th, 111th, 112th, 113th)
	lastTwoDigits := day % 100
	if lastTwoDigits >= 11 && lastTwoDigits <= 13 {
		return "th"
	}
	switch day % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	default:
		return "th"
	}
}
