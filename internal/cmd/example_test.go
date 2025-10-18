package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExampleCommand(t *testing.T) {
	t.Run("example command should be registered", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"example"})
		require.NoError(t, err)
		assert.Equal(t, "example", cmd.Use)
	})

	t.Run("example command should have name flag", func(t *testing.T) {
		flag := exampleCmd.Flags().Lookup("name")
		require.NotNil(t, flag)
		assert.Equal(t, "name", flag.Name)
		assert.Equal(t, "n", flag.Shorthand)
	})

	t.Run("example command should execute with default message", func(t *testing.T) {
		// Capture output
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)

		// Set args to execute example command
		rootCmd.SetArgs([]string{"example"})

		// Execute
		err := rootCmd.Execute()
		require.NoError(t, err)

		// Verify output contains expected message
		output := buf.String()
		assert.Contains(t, output, "Hello from cronic!")
	})

	t.Run("example command should execute with custom name", func(t *testing.T) {
		// Capture output
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)

		// Set args with name flag
		rootCmd.SetArgs([]string{"example", "--name", "Test"})

		// Execute
		err := rootCmd.Execute()
		require.NoError(t, err)

		// Verify output contains custom name
		output := buf.String()
		assert.Contains(t, output, "Hello, Test!")
	})
}
