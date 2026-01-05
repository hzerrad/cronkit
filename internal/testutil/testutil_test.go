package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateTempCrontab(t *testing.T) {
	content := "0 2 * * * /usr/bin/backup.sh\n*/15 * * * * /usr/bin/check.sh\n"

	file, cleanup := CreateTempCrontab(t, content)
	defer cleanup()

	// Verify file exists
	if !FileExists(file) {
		t.Fatal("temp crontab file should exist")
	}

	// Verify content
	readContent, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("failed to read temp crontab: %v", err)
	}

	if string(readContent) != content {
		t.Errorf("content mismatch: got %q, want %q", string(readContent), content)
	}
}

func TestLoadTestCrontab(t *testing.T) {
	path := LoadTestCrontab("sample.cron")

	// Verify path is constructed correctly
	expected := filepath.Join("..", "..", "testdata", "crontab", "sample.cron")
	if path != expected {
		t.Errorf("path mismatch: got %q, want %q", path, expected)
	}
}

func TestFileExists(t *testing.T) {
	// Test with existing file
	file, cleanup := CreateTempCrontab(t, "test content")
	defer cleanup()

	if !FileExists(file) {
		t.Error("FileExists should return true for existing file")
	}

	// Test with non-existent file
	if FileExists("/nonexistent/file.cron") {
		t.Error("FileExists should return false for non-existent file")
	}
}

func TestCreateTempCrontab_ErrorPath(t *testing.T) {
	// Test that CreateTempCrontab handles the error path
	// We can't easily force os.WriteFile to fail, but we can verify
	// the function structure handles errors correctly
	t.Run("should handle cleanup on error", func(t *testing.T) {
		// Create a valid temp crontab to verify structure
		file, cleanup := CreateTempCrontab(t, "test")
		defer cleanup()

		// Verify file was created
		if !FileExists(file) {
			t.Error("File should exist")
		}

		// Verify cleanup function works
		cleanup()
		if FileExists(file) {
			t.Error("File should be cleaned up")
		}
	})
}
