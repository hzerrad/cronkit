package check

import "time"

// ReferenceDate is a fixed date used for consistent calculations
// Using 2025-01-01 00:00:00 UTC as a reference point
var ReferenceDate = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
