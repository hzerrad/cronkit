// Package timeutil provides time helpers shared by the schedule analyzers.
package timeutil

import "time"

// MinuteKey buckets by absolute instant, not time.Time, since == on time.Time also compares Location.
func MinuteKey(t time.Time) int64 {
	return t.UTC().Truncate(time.Minute).Unix()
}

// ResolveLocation returns an item's own IANA zone when it names one, and fallback when the zone is empty.
func ResolveLocation(zone string, fallback *time.Location) (*time.Location, error) {
	if zone == "" {
		return fallback, nil
	}
	return time.LoadLocation(zone)
}
