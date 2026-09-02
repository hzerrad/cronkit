package inventory

import (
	"fmt"
	"strings"
	"time"
)

// unresolvableTimezoneReasonPrefix opens every Reason ResolveTimezones writes.
const unresolvableTimezoneReasonPrefix = "unresolvable timezone "

// ResolveTimezones rewrites any StateActive item with an unresolvable Timezone to StateInvalid.
// Call it once at admission, before handing items to any analyzer, so none silently skips it.
func ResolveTimezones(items []Item) []Item {
	out := make([]Item, len(items))
	for i, item := range items {
		out[i] = item
		if item.State != StateActive || item.Timezone == "" {
			continue
		}
		if _, err := time.LoadLocation(item.Timezone); err != nil {
			out[i].State = StateInvalid
			out[i].Reason = fmt.Sprintf("%s%q: %s", unresolvableTimezoneReasonPrefix, item.Timezone, err)
		}
	}
	return out
}

// IsUnresolvableTimezone reports whether item was made StateInvalid by ResolveTimezones specifically.
func IsUnresolvableTimezone(item Item) bool {
	return item.State == StateInvalid && strings.HasPrefix(item.Reason, unresolvableTimezoneReasonPrefix)
}
