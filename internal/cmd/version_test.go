package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionCommand(t *testing.T) {
	t.Run("version command should be registered", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"version"})
		assert.NoError(t, err)
		assert.Equal(t, "version", cmd.Use)
	})

	t.Run("version command should have description", func(t *testing.T) {
		assert.NotEmpty(t, versionCmd.Short)
		assert.NotEmpty(t, versionCmd.Long)
	})

	t.Run("version command should have run function", func(t *testing.T) {
		assert.NotNil(t, versionCmd.Run)
	})
}
