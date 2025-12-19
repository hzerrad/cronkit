package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExplainCommand(t *testing.T) {
	t.Run("explain command should be registered", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"explain"})
		assert.NoError(t, err)
		assert.Equal(t, "explain", cmd.Name())
	})

	t.Run("explain command should have metadata", func(t *testing.T) {
		ec := newExplainCommand()
		assert.NotEmpty(t, ec.Short)
		assert.NotEmpty(t, ec.Long)
		assert.Contains(t, ec.Use, "explain")
	})

	t.Run("explain standard cron expression", func(t *testing.T) {
		ec := newExplainCommand()
		buf := new(bytes.Buffer)
		ec.SetOut(buf)
		ec.SetArgs([]string{"0 0 * * *"})

		err := ec.Execute()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "At midnight")
	})

	t.Run("explain cron alias", func(t *testing.T) {
		ec := newExplainCommand()
		buf := new(bytes.Buffer)
		ec.SetOut(buf)
		ec.SetArgs([]string{"@daily"})

		err := ec.Execute()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "At midnight every day")
	})

	t.Run("explain case-insensitive day names", func(t *testing.T) {
		ec := newExplainCommand()
		buf := new(bytes.Buffer)
		ec.SetOut(buf)
		ec.SetArgs([]string{"0 9 * * mon-fri"})

		err := ec.Execute()
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "weekdays")
	})

	t.Run("explain with JSON output", func(t *testing.T) {
		ec := newExplainCommand()
		buf := new(bytes.Buffer)
		ec.SetOut(buf)
		ec.SetArgs([]string{"0 0 * * *", "--json"})

		err := ec.Execute()
		require.NoError(t, err)

		var result map[string]string
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		assert.Equal(t, "0 0 * * *", result["expression"])
		assert.Contains(t, result["description"], "midnight")
	})

	t.Run("fail on invalid cron expression", func(t *testing.T) {
		ec := newExplainCommand()
		ec.SetArgs([]string{"invalid"})

		err := ec.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse expression")
	})

	t.Run("fail on missing argument", func(t *testing.T) {
		ec := newExplainCommand()
		ec.SetArgs([]string{})

		err := ec.Execute()
		assert.Error(t, err)
	})

	t.Run("fail on too many arguments", func(t *testing.T) {
		ec := newExplainCommand()
		ec.SetArgs([]string{"0 0 * * *", "extra"})

		err := ec.Execute()
		assert.Error(t, err)
	})
}
