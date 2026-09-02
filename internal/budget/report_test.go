package budget

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/hzerrad/cronkit/internal/inventory"
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

// TestTextRenderer_UnresolvedExplainsAFailureWithNoViolations confirms the
// Unresolved section explains a "FAILED, Violations: 0" result, always shown, not just under --verbose.
func TestTextRenderer_UnresolvedExplainsAFailureWithNoViolations(t *testing.T) {
	report := &BudgetReport{
		Passed: false,
		Budgets: []BudgetResult{
			{
				Budget:     Budget{Name: "strict", MaxConcurrent: 1, TimeWindow: 1 * time.Hour},
				MaxFound:   2,
				Passed:     false,
				Violations: []Violation{},
				Unresolved: []UnresolvedItem{
					{
						Expression: "* * * * *",
						Locator:    inventory.Locator{File: "ops/crontab", Line: 2},
						Reason:     `unresolvable timezone "Not/AZone": unknown time zone Not/AZone`,
					},
				},
			},
		},
	}

	renderer := &TextRenderer{Verbose: false}
	var buf bytes.Buffer
	err := renderer.Render(&buf, report)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "FAILED")
	assert.Contains(t, output, "Violations: 0")
	assert.Contains(t, output, "Unresolved: 1 item(s)")
	assert.Contains(t, output, "ops/crontab:2")
	assert.Contains(t, output, "Not/AZone")
}

// TestTextRenderer_UnresolvedWithoutFile confirms a locator-less item still renders something legible.
func TestTextRenderer_UnresolvedWithoutFile(t *testing.T) {
	report := &BudgetReport{
		Passed: false,
		Budgets: []BudgetResult{
			{
				Budget:     Budget{Name: "strict", MaxConcurrent: 1, TimeWindow: 1 * time.Hour},
				MaxFound:   1,
				Passed:     false,
				Violations: []Violation{},
				Unresolved: []UnresolvedItem{
					{Expression: "* * * * *", Reason: `unresolvable timezone "Not/AZone": unknown time zone Not/AZone`},
				},
			},
		},
	}

	renderer := &TextRenderer{Verbose: false}
	var buf bytes.Buffer
	err := renderer.Render(&buf, report)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "* * * * * (unresolvable timezone")
	assert.NotContains(t, output, "    - : *")
}

// TestTextRenderer_NoUnresolvedSection confirms a plain failure doesn't grow an empty "Unresolved" section.
func TestTextRenderer_NoUnresolvedSection(t *testing.T) {
	report := &BudgetReport{
		Passed: false,
		Budgets: []BudgetResult{
			{
				Budget:   Budget{Name: "test-budget", MaxConcurrent: 2, TimeWindow: 1 * time.Hour},
				MaxFound: 3,
				Passed:   false,
				Violations: []Violation{
					{Time: time.Now(), Count: 3, Jobs: []string{"line-1", "line-2", "line-3"}},
				},
			},
		},
	}

	renderer := &TextRenderer{Verbose: false}
	var buf bytes.Buffer
	err := renderer.Render(&buf, report)

	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "Unresolved")
}

// TestJSONRenderer_Unresolved confirms unresolved is an array naming the expression, locator, and reason.
func TestJSONRenderer_Unresolved(t *testing.T) {
	report := &BudgetReport{
		Passed: false,
		Budgets: []BudgetResult{
			{
				Budget:     Budget{Name: "strict", MaxConcurrent: 1, TimeWindow: 1 * time.Hour},
				MaxFound:   2,
				Passed:     false,
				Violations: []Violation{},
				Unresolved: []UnresolvedItem{
					{
						Expression: "* * * * *",
						Locator:    inventory.Locator{File: "ops/crontab", Line: 2},
						Reason:     `unresolvable timezone "Not/AZone": unknown time zone Not/AZone`,
					},
				},
			},
		},
	}

	renderer := &JSONRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, report)
	require.NoError(t, err)

	var decoded struct {
		Budgets []struct {
			Unresolved []struct {
				Expression string            `json:"expression"`
				Locator    inventory.Locator `json:"locator"`
				Reason     string            `json:"reason"`
			} `json:"unresolved"`
		} `json:"budgets"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))

	require.Len(t, decoded.Budgets, 1)
	require.Len(t, decoded.Budgets[0].Unresolved, 1)
	u := decoded.Budgets[0].Unresolved[0]
	assert.Equal(t, "* * * * *", u.Expression)
	assert.Equal(t, "ops/crontab", u.Locator.File)
	assert.Equal(t, 2, u.Locator.Line)
	assert.Contains(t, u.Reason, "Not/AZone")
}

// TestJSONRenderer_UnresolvedOmittedWhenEmpty confirms the field is an empty array, not absent or null.
func TestJSONRenderer_UnresolvedOmittedWhenEmpty(t *testing.T) {
	report := &BudgetReport{
		Passed: true,
		Budgets: []BudgetResult{
			{Budget: Budget{Name: "test-budget", MaxConcurrent: 10, TimeWindow: 1 * time.Hour}, MaxFound: 1, Passed: true},
		},
	}

	renderer := &JSONRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, report)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), `"unresolved": []`)
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
