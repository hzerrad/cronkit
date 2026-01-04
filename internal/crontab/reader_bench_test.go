package crontab

import (
	"os"
	"testing"
)

func BenchmarkReadFile_Small(b *testing.B) {
	reader := NewReader()
	file := "../../testdata/crontab/valid/sample.cron"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reader.ReadFile(file)
	}
}

func BenchmarkReadFile_Medium(b *testing.B) {
	reader := NewReader()
	// Create a medium-sized crontab (100 jobs)
	content := ""
	for i := 0; i < 100; i++ {
		content += "0 * * * * /usr/bin/job" + string(rune('0'+(i%10))) + ".sh\n"
	}

	tmpfile, err := os.CreateTemp("", "bench-medium-*.cron")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_, _ = tmpfile.WriteString(content)
	_ = tmpfile.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reader.ReadFile(tmpfile.Name())
	}
}

func BenchmarkReadFile_Large(b *testing.B) {
	reader := NewReader()
	// Create a large crontab (500 jobs)
	content := ""
	for i := 0; i < 500; i++ {
		content += "0 * * * * /usr/bin/job" + string(rune('0'+(i%10))) + ".sh\n"
	}

	tmpfile, err := os.CreateTemp("", "bench-large-*.cron")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_, _ = tmpfile.WriteString(content)
	_ = tmpfile.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reader.ReadFile(tmpfile.Name())
	}
}

func BenchmarkParseFile_AllEntries(b *testing.B) {
	reader := NewReader()
	file := "../../testdata/crontab/valid/sample.cron"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reader.ParseFile(file)
	}
}
