package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRootCommand tests the root command initialization
func TestRootCommand(t *testing.T) {
	t.Run("root command should have correct name", func(t *testing.T) {
		assert.Equal(t, "cronic", rootCmd.Use)
	})

	t.Run("root command should have version set", func(t *testing.T) {
		require.NotEmpty(t, rootCmd.Version)
	})

	t.Run("root command should have help defined", func(t *testing.T) {
		assert.NotEmpty(t, rootCmd.Short)
		assert.NotEmpty(t, rootCmd.Long)
	})
}

// TestExecute tests the Execute function
func TestExecute(t *testing.T) {
	t.Run("execute should return error for invalid command", func(t *testing.T) {
		// Save original args
		oldArgs := rootCmd.Use

		// Reset command for testing
		rootCmd.SetArgs([]string{"invalid-command"})

		err := Execute()
		assert.Error(t, err)

		// Restore
		rootCmd.Use = oldArgs
	})
}

// Example of table-driven tests
func TestVersionFormat(t *testing.T) {
	t.Run("version string should contain version info", func(t *testing.T) {
		// The version command should have a version string set
		assert.NotEmpty(t, rootCmd.Version)
		// Version should contain "commit" and "built" keywords
		assert.Contains(t, rootCmd.Version, "commit")
		assert.Contains(t, rootCmd.Version, "built")
	})
}
