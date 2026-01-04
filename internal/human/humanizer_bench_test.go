package human

import (
	"testing"

	"github.com/hzerrad/cronic/internal/cronx"
)

func BenchmarkHumanize_Simple(b *testing.B) {
	humanizer := NewHumanizer()
	parser := cronx.NewParser()
	schedule, _ := parser.Parse("0 * * * *")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = humanizer.Humanize(schedule)
	}
}

func BenchmarkHumanize_Complex(b *testing.B) {
	humanizer := NewHumanizer()
	parser := cronx.NewParser()
	schedule, _ := parser.Parse("*/15 9-17 * * 1-5")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = humanizer.Humanize(schedule)
	}
}

func BenchmarkHumanize_WithRanges(b *testing.B) {
	humanizer := NewHumanizer()
	parser := cronx.NewParser()
	schedule, _ := parser.Parse("0 0 1-15 * MON-FRI")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = humanizer.Humanize(schedule)
	}
}
