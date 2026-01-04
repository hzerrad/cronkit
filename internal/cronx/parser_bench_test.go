package cronx

import (
	"testing"
	"time"
)

func BenchmarkParser_Parse_Simple(b *testing.B) {
	parser := NewParser()
	expr := "0 * * * *"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(expr)
	}
}

func BenchmarkParser_Parse_Complex(b *testing.B) {
	parser := NewParser()
	expr := "*/15 9-17 * * 1-5"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(expr)
	}
}

func BenchmarkParser_Parse_WithRanges(b *testing.B) {
	parser := NewParser()
	expr := "0 0 1-15 * MON-FRI"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(expr)
	}
}

func BenchmarkParser_Parse_Alias(b *testing.B) {
	parser := NewParser()
	expr := "@daily"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(expr)
	}
}

func BenchmarkScheduler_Next_Single(b *testing.B) {
	scheduler := NewScheduler()
	expr := "0 * * * *"
	from := parseTime("2025-01-01T00:00:00Z")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scheduler.Next(expr, from, 10)
	}
}

func BenchmarkScheduler_Next_Multiple(b *testing.B) {
	scheduler := NewScheduler()
	expr := "*/5 * * * *"
	from := parseTime("2025-01-01T00:00:00Z")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scheduler.Next(expr, from, 100)
	}
}

// Helper function for benchmarks
func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
