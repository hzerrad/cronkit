package stats

import (
	"fmt"
	"strings"
)

// GenerateHistogram generates a text histogram from hour data
func GenerateHistogram(hourData []int, width int) string {
	if len(hourData) != HoursInDay {
		return ""
	}

	// Find maxCount value for scaling
	maxCount := 0
	for _, v := range hourData {
		if v > maxCount {
			maxCount = v
		}
	}

	if maxCount == 0 {
		return "No runs detected"
	}

	var sb strings.Builder
	sb.WriteString("Hour Distribution:\n")
	sb.WriteString(strings.Repeat("=", width+20) + "\n")

	for hour := 0; hour < HoursInDay; hour++ {
		count := hourData[hour]
		barWidth := int(float64(count) / float64(maxCount) * float64(width))
		bar := strings.Repeat("█", barWidth)
		sb.WriteString(fmt.Sprintf("%02d:00 │%s %d\n", hour, bar, count))
	}

	return sb.String()
}
