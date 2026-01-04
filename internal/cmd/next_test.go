package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextCommand(t *testing.T) {
	t.Run("next command should be registered", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"next"})
		assert.NoError(t, err)
		assert.Equal(t, "next", cmd.Name())
	})

	t.Run("next command should have metadata", func(t *testing.T) {
		nc := newNextCommand()
		assert.NotEmpty(t, nc.Short)
		assert.NotEmpty(t, nc.Long)
		assert.Contains(t, nc.Use, "next")
	})

	t.Run("next standard cron expression (text)", func(t *testing.T) {
		nc := newNextCommand()
		buf := new(bytes.Buffer)
		nc.SetOut(buf)
		nc.SetArgs([]string{"*/15 * * * *"})

		err := nc.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Next 10 runs")
		assert.Contains(t, output, "*/15 * * * *")
		assert.Contains(t, output, "Every 15 minutes")
		assert.Contains(t, output, "1.")
		assert.Contains(t, output, "10.")
	})

	t.Run("next with custom count", func(t *testing.T) {
		nc := newNextCommand()
		buf := new(bytes.Buffer)
		nc.SetOut(buf)
		nc.SetArgs([]string{"@daily", "--count", "5"})

		err := nc.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Next 5 runs")
		assert.Contains(t, output, "5.")
		assert.NotContains(t, output, "6.")
	})

	t.Run("next with JSON output", func(t *testing.T) {
		nc := newNextCommand()
		buf := new(bytes.Buffer)
		nc.SetOut(buf)
		nc.SetArgs([]string{"@hourly", "--json", "-c", "3"})

		err := nc.Execute()
		require.NoError(t, err)

		var result NextResult
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		assert.Equal(t, "@hourly", result.Expression)
		assert.Contains(t, result.Description, "hour")
		assert.Len(t, result.NextRuns, 3)
		assert.Equal(t, 1, result.NextRuns[0].Number)
		assert.NotEmpty(t, result.NextRuns[0].Timestamp)
		assert.NotEmpty(t, result.NextRuns[0].Relative)
	})

	t.Run("fail on invalid cron expression", func(t *testing.T) {
		nc := newNextCommand()
		nc.SetArgs([]string{"invalid"})

		err := nc.Execute()
		assert.Error(t, err)
	})

	t.Run("fail on out of range count (low)", func(t *testing.T) {
		nc := newNextCommand()
		nc.SetArgs([]string{"@daily", "--count", "0"})

		err := nc.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "count must be at least 1")
	})

	t.Run("fail on out of range count (high)", func(t *testing.T) {
		nc := newNextCommand()
		nc.SetArgs([]string{"@daily", "--count", "101"})

		err := nc.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "count must be at most 100")
	})

	t.Run("fail on missing argument", func(t *testing.T) {
		nc := newNextCommand()
		nc.SetArgs([]string{})

		err := nc.Execute()
		assert.Error(t, err)
	})

	t.Run("formatRelativeTime", func(t *testing.T) {
		from := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

		tests := []struct {
			name     string
			to       time.Time
			expected string
		}{
			{
				"less than a minute",
				from.Add(30 * time.Second),
				"in less than a minute",
			},
			{
				"exactly 1 minute",
				from.Add(1 * time.Minute),
				"in 1 minute",
			},
			{
				"multiple minutes",
				from.Add(15 * time.Minute),
				"in 15 minutes",
			},
			{
				"exactly 1 hour",
				from.Add(1 * time.Hour),
				"in 1 hour",
			},
			{
				"multiple hours",
				from.Add(5 * time.Hour),
				"in 5 hours",
			},
			{
				"exactly 1 day",
				from.Add(24 * time.Hour),
				"in 1 day",
			},
			{
				"multiple days",
				from.Add(48 * time.Hour),
				"in 2 days",
			},
			{
				"many days",
				from.Add(10 * 24 * time.Hour),
				"in 10 days",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.expected, formatRelativeTime(from, tt.to))
			})
		}
	})

	t.Run("outputNextText with singular run", func(t *testing.T) {
		nc := newNextCommand()
		buf := new(bytes.Buffer)
		nc.SetOut(buf)
		nc.SetArgs([]string{"@daily", "--count", "1"})

		err := nc.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Next 1 run", "Should use singular 'run' for count of 1")
		assert.Contains(t, output, "1.")
		assert.NotContains(t, output, "2.")
	})

	t.Run("next with --timezone UTC", func(t *testing.T) {
		nc := newNextCommand()
		buf := new(bytes.Buffer)
		nc.SetOut(buf)
		nc.SetArgs([]string{"0 * * * *", "--timezone", "UTC", "--count", "3"})

		err := nc.Execute()
		require.NoError(t, err)

		output := buf.String()
		// Verify timestamps are in UTC
		assert.Contains(t, output, "UTC")
	})

	t.Run("next with --timezone America/New_York", func(t *testing.T) {
		nc := newNextCommand()
		buf := new(bytes.Buffer)
		nc.SetOut(buf)
		nc.SetArgs([]string{"0 * * * *", "--timezone", "America/New_York", "--count", "3"})

		err := nc.Execute()
		require.NoError(t, err)

		output := buf.String()
		// Should contain timezone abbreviation (EST or EDT) or timezone name
		assert.True(t, strings.Contains(output, "EST") || strings.Contains(output, "EDT") || strings.Contains(output, "America/New_York"))
	})

	t.Run("next with invalid timezone", func(t *testing.T) {
		nc := newNextCommand()
		nc.SetArgs([]string{"0 * * * *", "--timezone", "Invalid/Timezone"})

		err := nc.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid timezone")
	})

	t.Run("next with timezone and --json", func(t *testing.T) {
		nc := newNextCommand()
		buf := new(bytes.Buffer)
		nc.SetOut(buf)
		nc.SetArgs([]string{"@hourly", "--timezone", "UTC", "--json", "-c", "2"})

		err := nc.Execute()
		require.NoError(t, err)

		var result NextResult
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		assert.Equal(t, "UTC", result.Timezone)
		assert.Len(t, result.NextRuns, 2)
	})

	t.Run("next with timezone and various cron expressions", func(t *testing.T) {
		expressions := []string{"*/15 * * * *", "@daily", "0 9 * * 1-5", "0 0 1 * *"}

		for _, expr := range expressions {
			t.Run(expr, func(t *testing.T) {
				nc := newNextCommand()
				buf := new(bytes.Buffer)
				nc.SetOut(buf)
				nc.SetArgs([]string{expr, "--timezone", "Europe/London", "--count", "1"})

				err := nc.Execute()
				require.NoError(t, err)

				output := buf.String()
				assert.Contains(t, output, expr)
			})
		}
	})

	t.Run("next default behavior (local timezone when not specified)", func(t *testing.T) {
		nc := newNextCommand()
		buf := new(bytes.Buffer)
		nc.SetOut(buf)
		nc.SetArgs([]string{"0 * * * *", "--count", "1"})

		err := nc.Execute()
		require.NoError(t, err)

		output := buf.String()
		// Should not error and should produce output
		assert.Contains(t, output, "Next 1 run")
	})
}
