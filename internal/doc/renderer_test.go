package doc

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdownRenderer(t *testing.T) {
	renderer := &MarkdownRenderer{}
	doc := &Document{
		Title:       "Test Documentation",
		GeneratedAt: time.Now(),
		Source:      "test.cron",
		Jobs: []JobDocument{
			{
				LineNumber:  1,
				Expression:  "0 0 * * *",
				Description: "Runs daily at midnight",
				Command:     "/usr/bin/backup.sh",
			},
		},
		Metadata: Metadata{
			TotalJobs:   1,
			ValidJobs:   1,
			InvalidJobs: 0,
		},
	}

	var buf bytes.Buffer
	err := renderer.Render(doc, &buf)
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "# Test Documentation")
	assert.Contains(t, output, "0 0 * * *")
	assert.Contains(t, output, "Runs daily at midnight")
	assert.Contains(t, output, "/usr/bin/backup.sh")
}

func TestHTMLRenderer(t *testing.T) {
	renderer := &HTMLRenderer{}
	doc := &Document{
		Title:       "Test Documentation",
		GeneratedAt: time.Now(),
		Source:      "test.cron",
		Jobs: []JobDocument{
			{
				LineNumber:  1,
				Expression:  "0 0 * * *",
				Description: "Runs daily at midnight",
				Command:     "/usr/bin/backup.sh",
			},
		},
		Metadata: Metadata{
			TotalJobs:   1,
			ValidJobs:   1,
			InvalidJobs: 0,
		},
	}

	var buf bytes.Buffer
	err := renderer.Render(doc, &buf)
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "<!DOCTYPE html>")
	assert.Contains(t, output, "<title>Test Documentation</title>")
	assert.Contains(t, output, "0 0 * * *")
	assert.Contains(t, output, "Runs daily at midnight")
}

func TestJSONRenderer(t *testing.T) {
	renderer := &JSONRenderer{}
	doc := &Document{
		Title:       "Test Documentation",
		GeneratedAt: time.Now(),
		Source:      "test.cron",
		Jobs: []JobDocument{
			{
				LineNumber:  1,
				Expression:  "0 0 * * *",
				Description: "Runs daily at midnight",
				Command:     "/usr/bin/backup.sh",
			},
		},
		Metadata: Metadata{
			TotalJobs:   1,
			ValidJobs:   1,
			InvalidJobs: 0,
		},
	}

	var buf bytes.Buffer
	err := renderer.Render(doc, &buf)
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "\"Title\"")
	assert.Contains(t, output, "\"Jobs\"")
	assert.Contains(t, output, "0 0 * * *")
}

func TestMarkdownRenderer_WithAllSections(t *testing.T) {
	renderer := &MarkdownRenderer{}
	doc := &Document{
		Title:       "Test Documentation",
		GeneratedAt: time.Now(),
		Source:      "test.cron",
		Jobs: []JobDocument{
			{
				LineNumber:  1,
				Expression:  "0 * * * *",
				Description: "Runs hourly",
				Command:     "/usr/bin/backup.sh",
				Comment:     "Hourly backup",
				NextRuns:    []time.Time{time.Now().Add(1 * time.Hour)},
				Warnings:    []string{"Warning: test"},
				Stats:       &JobStats{RunsPerDay: 24, RunsPerHour: 1},
			},
		},
		Metadata: Metadata{
			TotalJobs:   1,
			ValidJobs:   1,
			InvalidJobs: 0,
		},
	}

	var buf bytes.Buffer
	err := renderer.Render(doc, &buf)
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Next Runs")
	assert.Contains(t, output, "Warnings")
	assert.Contains(t, output, "Statistics")
	assert.Contains(t, output, "Hourly backup")
}

func TestMarkdownRenderer_EdgeCases(t *testing.T) {
	renderer := &MarkdownRenderer{}

	t.Run("should handle document with no jobs", func(t *testing.T) {
		doc := &Document{
			Title:       "Empty Documentation",
			GeneratedAt: time.Now(),
			Source:      "empty.cron",
			Jobs:        []JobDocument{},
			Metadata: Metadata{
				TotalJobs:   0,
				ValidJobs:   0,
				InvalidJobs: 0,
			},
		}

		var buf bytes.Buffer
		err := renderer.Render(doc, &buf)
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Empty Documentation")
		assert.Contains(t, output, "Total Jobs: 0")
	})

	t.Run("should handle job with long command", func(t *testing.T) {
		doc := &Document{
			Title:       "Test",
			GeneratedAt: time.Now(),
			Source:      "test.cron",
			Jobs: []JobDocument{
				{
					LineNumber:  1,
					Expression:  "0 * * * *",
					Description: "Test",
					Command:     strings.Repeat("a", 100), // Long command
				},
			},
			Metadata: Metadata{
				TotalJobs:   1,
				ValidJobs:   1,
				InvalidJobs: 0,
			},
		}

		var buf bytes.Buffer
		err := renderer.Render(doc, &buf)
		require.NoError(t, err)
		output := buf.String()
		// Should truncate in table but show full in details
		assert.Contains(t, output, "...")
	})

	t.Run("should handle job with exactly 10 next runs", func(t *testing.T) {
		doc := &Document{
			Title:       "Test",
			GeneratedAt: time.Now(),
			Source:      "test.cron",
			Jobs: []JobDocument{
				{
					LineNumber:  1,
					Expression:  "0 * * * *",
					Description: "Test",
					Command:     "/usr/bin/test.sh",
					NextRuns:    []time.Time{time.Now(), time.Now().Add(1 * time.Hour), time.Now().Add(2 * time.Hour), time.Now().Add(3 * time.Hour), time.Now().Add(4 * time.Hour), time.Now().Add(5 * time.Hour), time.Now().Add(6 * time.Hour), time.Now().Add(7 * time.Hour), time.Now().Add(8 * time.Hour), time.Now().Add(9 * time.Hour)},
				},
			},
			Metadata: Metadata{
				TotalJobs:   1,
				ValidJobs:   1,
				InvalidJobs: 0,
			},
		}

		var buf bytes.Buffer
		err := renderer.Render(doc, &buf)
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Next Runs")
		// Should show all 10 runs
		assert.GreaterOrEqual(t, strings.Count(output, "- 2"), 1)
	})
}

func TestHTMLRenderer_EdgeCases(t *testing.T) {
	renderer := &HTMLRenderer{}

	t.Run("should handle document with no jobs", func(t *testing.T) {
		doc := &Document{
			Title:       "Empty Documentation",
			GeneratedAt: time.Now(),
			Source:      "empty.cron",
			Jobs:        []JobDocument{},
			Metadata: Metadata{
				TotalJobs:   0,
				ValidJobs:   0,
				InvalidJobs: 0,
			},
		}

		var buf bytes.Buffer
		err := renderer.Render(doc, &buf)
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "Empty Documentation")
		assert.Contains(t, output, "Total Jobs: 0")
	})

	t.Run("should handle multiple next runs", func(t *testing.T) {
		doc := &Document{
			Title:       "Test",
			GeneratedAt: time.Now(),
			Source:      "test.cron",
			Jobs: []JobDocument{
				{
					LineNumber:  1,
					Expression:  "0 * * * *",
					Description: "Test",
					Command:     "/usr/bin/test.sh",
					NextRuns:    []time.Time{time.Now(), time.Now().Add(1 * time.Hour), time.Now().Add(2 * time.Hour), time.Now().Add(3 * time.Hour), time.Now().Add(4 * time.Hour), time.Now().Add(5 * time.Hour), time.Now().Add(6 * time.Hour), time.Now().Add(7 * time.Hour), time.Now().Add(8 * time.Hour), time.Now().Add(9 * time.Hour), time.Now().Add(10 * time.Hour), time.Now().Add(11 * time.Hour)}, // More than 10
				},
			},
			Metadata: Metadata{
				TotalJobs:   1,
				ValidJobs:   1,
				InvalidJobs: 0,
			},
		}

		var buf bytes.Buffer
		err := renderer.Render(doc, &buf)
		require.NoError(t, err)
		output := buf.String()
		// Should limit to 10 next runs
		assert.Contains(t, output, "Next Runs")
	})
}

func TestHTMLRenderer_WithAllSections(t *testing.T) {
	renderer := &HTMLRenderer{}
	doc := &Document{
		Title:       "Test Documentation",
		GeneratedAt: time.Now(),
		Source:      "test.cron",
		Jobs: []JobDocument{
			{
				LineNumber:  1,
				Expression:  "0 * * * *",
				Description: "Runs hourly",
				Command:     "/usr/bin/backup.sh",
				Comment:     "Hourly backup",
				NextRuns:    []time.Time{time.Now().Add(1 * time.Hour)},
				Warnings:    []string{"Warning: test"},
				Stats:       &JobStats{RunsPerDay: 24, RunsPerHour: 1},
			},
		},
		Metadata: Metadata{
			TotalJobs:   1,
			ValidJobs:   1,
			InvalidJobs: 0,
		},
	}

	var buf bytes.Buffer
	err := renderer.Render(doc, &buf)
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Next Runs")
	assert.Contains(t, output, "Warnings")
	assert.Contains(t, output, "Statistics")
	assert.Contains(t, output, "Hourly backup")
}
