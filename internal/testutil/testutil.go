package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// CreateTempCrontab creates a temporary crontab file with the given content
// and returns the file path and a cleanup function.
func CreateTempCrontab(t *testing.T, content string) (string, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.cron")

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create temp crontab: %v", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}

	return tmpFile, cleanup
}

// LoadTestCrontab loads a test crontab file from the testdata directory.
func LoadTestCrontab(name string) string {
	// Path relative to internal/testutil
	path := filepath.Join("..", "..", "testdata", "crontab", name)
	return path
}

// FileExists checks if a file exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
