package check

import (
	"os"
	"testing"

	"github.com/hzerrad/cronic/internal/crontab"
)

func BenchmarkValidateExpression_Simple(b *testing.B) {
	validator := NewValidator("en")
	expr := "0 * * * *"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateExpression(expr)
	}
}

func BenchmarkValidateExpression_Complex(b *testing.B) {
	validator := NewValidator("en")
	expr := "*/15 9-17 * * 1-5"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateExpression(expr)
	}
}

func BenchmarkValidateCrontab_Small(b *testing.B) {
	validator := NewValidator("en")
	reader := crontab.NewReader()
	file := "../../testdata/crontab/valid/sample.cron"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateCrontab(reader, file)
	}
}

func BenchmarkValidateCrontab_Medium(b *testing.B) {
	validator := NewValidator("en")
	reader := crontab.NewReader()

	// Create a medium-sized crontab (100 jobs)
	content := ""
	for i := 0; i < 100; i++ {
		content += "0 * * * * /usr/bin/job" + string(rune('0'+(i%10))) + ".sh\n"
	}

	tmpfile, err := os.CreateTemp("", "bench-validate-*.cron")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_, _ = tmpfile.WriteString(content)
	_ = tmpfile.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateCrontab(reader, tmpfile.Name())
	}
}
