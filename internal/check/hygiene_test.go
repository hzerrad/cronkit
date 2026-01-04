package check

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeCommand(t *testing.T) {
	t.Run("should detect missing absolute path", func(t *testing.T) {
		issues := AnalyzeCommand("backup.sh")
		require.Greater(t, len(issues), 0)
		found := false
		for _, issue := range issues {
			if issue.Code == CodeMissingAbsolutePath {
				found = true
				break
			}
		}
		assert.True(t, found, "Should detect missing absolute path")
	})

	t.Run("should not flag absolute paths", func(t *testing.T) {
		issues := AnalyzeCommand("/usr/bin/backup.sh")
		for _, issue := range issues {
			assert.NotEqual(t, CodeMissingAbsolutePath, issue.Code, "Should not flag absolute paths")
		}
	})

	t.Run("should detect missing redirection", func(t *testing.T) {
		issues := AnalyzeCommand("/usr/bin/backup.sh")
		found := false
		for _, issue := range issues {
			if issue.Code == CodeMissingRedirection {
				found = true
				break
			}
		}
		assert.True(t, found, "Should detect missing redirection")
	})

	t.Run("should not flag commands with redirection", func(t *testing.T) {
		issues := AnalyzeCommand("/usr/bin/backup.sh > /var/log/backup.log 2>&1")
		for _, issue := range issues {
			assert.NotEqual(t, CodeMissingRedirection, issue.Code, "Should not flag commands with redirection")
		}
	})

	t.Run("should detect percent character", func(t *testing.T) {
		issues := AnalyzeCommand("/usr/bin/command --date %Y-%m-%d")
		found := false
		for _, issue := range issues {
			if issue.Code == CodePercentCharacter {
				found = true
				break
			}
		}
		assert.True(t, found, "Should detect % character")
	})

	t.Run("should detect unclosed quotes", func(t *testing.T) {
		issues := AnalyzeCommand("/usr/bin/command 'unclosed quote")
		found := false
		for _, issue := range issues {
			if issue.Code == CodeQuotingIssue {
				found = true
				break
			}
		}
		assert.True(t, found, "Should detect unclosed quotes")
	})

	t.Run("should not flag properly quoted commands", func(t *testing.T) {
		issues := AnalyzeCommand("/usr/bin/command 'properly quoted'")
		for _, issue := range issues {
			assert.NotEqual(t, CodeQuotingIssue, issue.Code, "Should not flag properly quoted commands")
		}
	})
}

func TestCheckAbsolutePath(t *testing.T) {
	t.Run("should detect absolute paths", func(t *testing.T) {
		assert.True(t, checkAbsolutePath("/usr/bin/command"))
		assert.True(t, checkAbsolutePath("/bin/ls"))
		assert.True(t, checkAbsolutePath("  /usr/local/bin/script.sh"))
	})

	t.Run("should not flag relative paths", func(t *testing.T) {
		assert.False(t, checkAbsolutePath("command"))
		assert.False(t, checkAbsolutePath("./script.sh"))
		assert.False(t, checkAbsolutePath("~/bin/script.sh"))
	})

	t.Run("should detect absolute paths in command arguments", func(t *testing.T) {
		assert.True(t, checkAbsolutePath("command /usr/bin/script.sh"))
		assert.True(t, checkAbsolutePath("command /opt/app/bin/run"))
		assert.False(t, checkAbsolutePath("command script.sh"))
	})

	t.Run("should handle empty command", func(t *testing.T) {
		assert.False(t, checkAbsolutePath(""))
		assert.False(t, checkAbsolutePath("   "))
	})
}

func TestCheckOutputRedirection(t *testing.T) {
	t.Run("should detect redirection", func(t *testing.T) {
		assert.True(t, checkOutputRedirection("command > file"))
		assert.True(t, checkOutputRedirection("command >> file"))
		assert.True(t, checkOutputRedirection("command 2> file"))
		assert.True(t, checkOutputRedirection("command &> file"))
		assert.True(t, checkOutputRedirection("command 2>> file"))
	})

	t.Run("should not flag commands without redirection", func(t *testing.T) {
		assert.False(t, checkOutputRedirection("command"))
		assert.False(t, checkOutputRedirection("command arg1 arg2"))
	})
}

func TestCheckPercentCharacter(t *testing.T) {
	t.Run("should detect percent character", func(t *testing.T) {
		assert.True(t, checkPercentCharacter("command --date %Y-%m-%d"))
		assert.True(t, checkPercentCharacter("command %"))
	})

	t.Run("should not flag commands without percent", func(t *testing.T) {
		assert.False(t, checkPercentCharacter("command"))
		assert.False(t, checkPercentCharacter("command --date YYYY-MM-DD"))
	})
}

func TestCheckQuotingEscaping(t *testing.T) {
	t.Run("should detect unclosed single quotes", func(t *testing.T) {
		issues := checkQuotingEscaping("command 'unclosed")
		require.Greater(t, len(issues), 0)
		assert.Equal(t, CodeQuotingIssue, issues[0].Code)
	})

	t.Run("should detect unclosed double quotes", func(t *testing.T) {
		issues := checkQuotingEscaping(`command "unclosed`)
		require.Greater(t, len(issues), 0)
		assert.Equal(t, CodeQuotingIssue, issues[0].Code)
	})

	t.Run("should not flag properly closed quotes", func(t *testing.T) {
		issues := checkQuotingEscaping(`command 'closed' "also closed"`)
		assert.Equal(t, 0, len(issues))
	})
}
