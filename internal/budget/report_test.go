package budget

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextRenderer_Render(t *testing.T) {
	report := &BudgetReport{
		Passed: false,
		Budgets: []BudgetResult{
			{
				Budget: Budget{
					Name:          "test-budget",
					MaxConcurrent: 2,
					TimeWindow:    1 * time.Hour,
				},
				MaxFound: 3,
				Passed:   false,
				Violations: []Violation{
					{
						Time:  time.Now(),
						Count: 3,
						Jobs:  []string{"job1", "job2", "job3"},
						Budget: Budget{
							Name:          "test-budget",
							MaxConcurrent: 2,
							TimeWindow:    1 * time.Hour,
						},
					},
				},
			},
		},
		Violations: []Violation{
			{
				Time:  time.Now(),
				Count: 3,
				Jobs:  []string{"job1", "job2", "job3"},
			},
		},
	}

	renderer := &TextRenderer{Verbose: false}
	var buf bytes.Buffer
	err := renderer.Render(&buf, report)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Budget Analysis")
	assert.Contains(t, output, "Budget violations detected")
	assert.Contains(t, output, "test-budget")
	assert.Contains(t, output, "FAILED")
}

func TestJSONRenderer_Render(t *testing.T) {
	report := &BudgetReport{
		Passed: true,
		Budgets: []BudgetResult{
			{
				Budget: Budget{
					Name:          "test-budget",
					MaxConcurrent: 10,
					TimeWindow:    1 * time.Hour,
				},
				MaxFound: 5,
				Passed:   true,
			},
		},
	}

	renderer := &JSONRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, report)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, `"passed"`)
	assert.Contains(t, output, `"budgets"`)
	assert.Contains(t, output, `"test-budget"`)
}

func TestNewRenderer(t *testing.T) {
	t.Run("text format", func(t *testing.T) {
		renderer, err := NewRenderer("text", false)
		require.NoError(t, err)
		assert.IsType(t, &TextRenderer{}, renderer)
	})

	t.Run("json format", func(t *testing.T) {
		renderer, err := NewRenderer("json", false)
		require.NoError(t, err)
		assert.IsType(t, &JSONRenderer{}, renderer)
	})

	t.Run("default format", func(t *testing.T) {
		renderer, err := NewRenderer("", false)
		require.NoError(t, err)
		assert.IsType(t, &TextRenderer{}, renderer)
	})

	t.Run("invalid format", func(t *testing.T) {
		_, err := NewRenderer("invalid", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown format")
	})
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"30 seconds", 30 * time.Second, "30s"},
		{"5 minutes", 5 * time.Minute, "5m"},
		{"2 hours", 2 * time.Hour, "2h"},
		{"1 day", 24 * time.Hour, "1d"},
		{"2 days", 48 * time.Hour, "2d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTextRenderer_Verbose(t *testing.T) {
	report := &BudgetReport{
		Passed: false,
		Budgets: []BudgetResult{
			{
				Budget: Budget{
					Name:          "test-budget",
					MaxConcurrent: 2,
					TimeWindow:    1 * time.Hour,
				},
				MaxFound: 3,
				Passed:   false,
				Violations: []Violation{
					{
						Time:  time.Now(),
						Count: 3,
						Jobs:  []string{"job1", "job2", "job3"},
						Budget: Budget{
							Name:          "test-budget",
							MaxConcurrent: 2,
							TimeWindow:    1 * time.Hour,
						},
					},
				},
			},
		},
		Violations: []Violation{
			{
				Time:  time.Now(),
				Count: 3,
				Jobs:  []string{"job1", "job2", "job3"},
			},
		},
	}

	renderer := &TextRenderer{Verbose: true}
	var buf bytes.Buffer
	err := renderer.Render(&buf, report)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Violation Details")
	assert.Contains(t, output, "job1")
}

func TestTextRenderer_Passed(t *testing.T) {
	report := &BudgetReport{
		Passed: true,
		Budgets: []BudgetResult{
			{
				Budget: Budget{
					Name:          "test-budget",
					MaxConcurrent: 10,
					TimeWindow:    1 * time.Hour,
				},
				MaxFound: 5,
				Passed:   true,
			},
		},
	}

	renderer := &TextRenderer{Verbose: false}
	var buf bytes.Buffer
	err := renderer.Render(&buf, report)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "All budgets passed")
	assert.Contains(t, output, "PASSED")
}

func TestTextRenderer_NoName(t *testing.T) {
	report := &BudgetReport{
		Passed: true,
		Budgets: []BudgetResult{
			{
				Budget: Budget{
					Name:          "",
					MaxConcurrent: 10,
					TimeWindow:    1 * time.Hour,
				},
				MaxFound: 5,
				Passed:   true,
			},
		},
	}

	renderer := &TextRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, report)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Max 10 concurrent jobs")
}

func TestJSONRenderer_WithViolations(t *testing.T) {
	report := &BudgetReport{
		Passed: false,
		Budgets: []BudgetResult{
			{
				Budget: Budget{
					Name:          "test-budget",
					MaxConcurrent: 2,
					TimeWindow:    1 * time.Hour,
				},
				MaxFound: 3,
				Passed:   false,
				Violations: []Violation{
					{
						Time:  time.Now(),
						Count: 3,
						Jobs:  []string{"job1", "job2", "job3"},
						Budget: Budget{
							Name:          "test-budget",
							MaxConcurrent: 2,
							TimeWindow:    1 * time.Hour,
						},
					},
				},
			},
		},
		Violations: []Violation{
			{
				Time:  time.Now(),
				Count: 3,
				Jobs:  []string{"job1", "job2", "job3"},
				Budget: Budget{
					Name:          "test-budget",
					MaxConcurrent: 2,
					TimeWindow:    1 * time.Hour,
				},
			},
		},
	}

	renderer := &JSONRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, report)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, `"passed"`)
	assert.Contains(t, output, `false`)
	assert.Contains(t, output, `"violations"`)
	assert.Contains(t, output, `"job1"`)
}
