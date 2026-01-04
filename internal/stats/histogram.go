package stats

import (
	"fmt"
	"strings"
)

// GenerateHistogram generates a text histogram from hour data
func GenerateHistogram(hourData []int, width int) string {
	if len(hourData) != 24 {
		return ""
	}

	// Find max value for scaling
	max := 0
	for _, v := range hourData {
		if v > max {
			max = v
		}
	}

	if max == 0 {
		return "No runs detected"
	}

	var sb strings.Builder
	sb.WriteString("Hour Distribution:\n")
	sb.WriteString(strings.Repeat("=", width+20) + "\n")

	for hour := 0; hour < 24; hour++ {
		count := hourData[hour]
		barWidth := int(float64(count) / float64(max) * float64(width))
		bar := strings.Repeat("█", barWidth)
		sb.WriteString(fmt.Sprintf("%02d:00 │%s %d\n", hour, bar, count))
	}

	return sb.String()
}
