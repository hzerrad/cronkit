package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withStdin temporarily replaces os.Stdin, since the commands under test read it directly, not via cobra.
func withStdin(t *testing.T, content string) {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	_, err = w.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = old
	})
}

// writeInventoryFixture writes inv as an inventory JSON file and returns its
// path, cleaned up automatically.
func writeInventoryFixture(t *testing.T, inv *inventory.Inventory) string {
	t.Helper()
	f, err := os.CreateTemp("", "cronkit-inventory-*.json")
	require.NoError(t, err)
	require.NoError(t, inv.Encode(f))
	require.NoError(t, f.Close())
	t.Cleanup(func() { _ = os.Remove(f.Name()) })
	return f.Name()
}

func TestInputFlags_Register(t *testing.T) {
	t.Run("registers file, stdin and inventory flags", func(t *testing.T) {
		cmd := newListCommand()
		assert.NotNil(t, cmd.Flag("file"))
		assert.NotNil(t, cmd.Flag("stdin"))
		assert.NotNil(t, cmd.Flag("inventory"))
	})

	t.Run("stdin help text mentions auto-detection only when autoStdin is set", func(t *testing.T) {
		var withAuto inputFlags
		cmdAuto := &cobra.Command{Use: "auto"}
		withAuto.register(cmdAuto, true)
		assert.Contains(t, cmdAuto.Flag("stdin").Usage, "automatic")

		var withoutAuto inputFlags
		cmdManual := &cobra.Command{Use: "manual"}
		withoutAuto.register(cmdManual, false)
		assert.NotContains(t, cmdManual.Flag("stdin").Usage, "automatic")
	})
}

func TestInputFlags_Classify(t *testing.T) {
	restore := isStdinAvailable
	t.Cleanup(func() { isStdinAvailable = restore })

	t.Run("inventory wins over everything else", func(t *testing.T) {
		isStdinAvailable = func() bool { return true }
		f := inputFlags{inventory: "inv.json", file: "some.cron", stdin: true, autoStdin: true}
		assert.Equal(t, sourceInventory, f.classify())
	})

	t.Run("file wins over stdin and user crontab", func(t *testing.T) {
		isStdinAvailable = func() bool { return true }
		f := inputFlags{file: "some.cron", stdin: true, autoStdin: true}
		assert.Equal(t, sourceFile, f.classify())
	})

	t.Run("explicit stdin wins over user crontab", func(t *testing.T) {
		isStdinAvailable = func() bool { return false }
		f := inputFlags{stdin: true}
		assert.Equal(t, sourceStdin, f.classify())
	})

	t.Run("autoStdin detects a piped stdin when nothing else is set", func(t *testing.T) {
		isStdinAvailable = func() bool { return true }
		f := inputFlags{autoStdin: true}
		assert.Equal(t, sourceStdin, f.classify())
	})

	t.Run("autoStdin does not fire when stdin is a terminal", func(t *testing.T) {
		isStdinAvailable = func() bool { return false }
		f := inputFlags{autoStdin: true}
		assert.Equal(t, sourceUserCrontab, f.classify())
	})

	t.Run("without autoStdin, falls back to user crontab regardless of stdin availability", func(t *testing.T) {
		isStdinAvailable = func() bool { return true }
		f := inputFlags{autoStdin: false}
		assert.Equal(t, sourceUserCrontab, f.classify())
	})
}

func TestResolveItems_Precedence(t *testing.T) {
	restore := isStdinAvailable
	isStdinAvailable = func() bool { return false }
	t.Cleanup(func() { isStdinAvailable = restore })

	cronFile := createTempFile(t, "0 0 * * * /usr/bin/from-file.sh\n")
	inv := inventory.New("", []inventory.Item{
		{Expression: "0 0 * * *", Command: "from-inventory", State: inventory.StateActive},
	})
	invFile := writeInventoryFixture(t, inv)

	t.Run("inventory beats file", func(t *testing.T) {
		f := inputFlags{inventory: invFile, file: cronFile}
		items, err := resolveItems(&f)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "from-inventory", items[0].Command)
	})

	t.Run("file beats stdin and user crontab", func(t *testing.T) {
		f := inputFlags{file: cronFile, stdin: true}
		items, err := resolveItems(&f)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "/usr/bin/from-file.sh", items[0].Command)
	})

	t.Run("explicit stdin reads crontab text from stdin", func(t *testing.T) {
		withStdin(t, "0 0 * * * /usr/bin/from-stdin.sh\n")
		f := inputFlags{stdin: true}
		items, err := resolveItems(&f)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "/usr/bin/from-stdin.sh", items[0].Command)
	})
}

func TestResolveItems_Inventory(t *testing.T) {
	t.Run("reads from a path", func(t *testing.T) {
		inv := inventory.New("/repo", []inventory.Item{
			{Expression: "0 0 * * *", Command: "gcr.io/proj/image", SourceID: "k8s", State: inventory.StateActive},
		})
		path := writeInventoryFixture(t, inv)

		f := inputFlags{inventory: path}
		items, err := resolveItems(&f)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "gcr.io/proj/image", items[0].Command)
		assert.False(t, items[0].Shell)
	})

	t.Run("- reads from stdin, not sniffed as crontab text", func(t *testing.T) {
		inv := inventory.New("", []inventory.Item{
			{Expression: "*/5 * * * *", Command: "worker", State: inventory.StateActive},
		})
		var buf strings.Builder
		require.NoError(t, inv.Encode(&buf))
		withStdin(t, buf.String())

		f := inputFlags{inventory: "-"}
		items, err := resolveItems(&f)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "worker", items[0].Command)
	})

	t.Run("malformed inventory fails clearly", func(t *testing.T) {
		path := createTempFile(t, "{ this is not valid json")
		f := inputFlags{inventory: path}
		_, err := resolveItems(&f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode inventory")
	})

	t.Run("schema version mismatch names both versions", func(t *testing.T) {
		path := createTempFile(t, `{"schemaVersion":"999","items":[]}`)
		f := inputFlags{inventory: path}
		_, err := resolveItems(&f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "999")
		assert.Contains(t, err.Error(), inventory.SchemaVersion)
	})

	t.Run("missing file surfaces the open error", func(t *testing.T) {
		f := inputFlags{inventory: "/no/such/path/inventory.json"}
		_, err := resolveItems(&f)
		require.Error(t, err)
	})

	t.Run("resolves timezones at the admission boundary", func(t *testing.T) {
		inv := inventory.New("", []inventory.Item{
			{Expression: "0 0 * * *", Command: "worker", State: inventory.StateActive, Timezone: "Not/AZone"},
		})
		path := writeInventoryFixture(t, inv)

		f := inputFlags{inventory: path}
		items, err := resolveItems(&f)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, inventory.StateInvalid, items[0].State)
		assert.True(t, inventory.IsUnresolvableTimezone(items[0]))
	})
}

// TestReadItems_FileLocatorIsIndependentOfHowTheFileWasNamed pins the locator against how --file was typed.
func TestReadItems_FileLocatorIsIndependentOfHowTheFileWasNamed(t *testing.T) {
	restore := isStdinAvailable
	isStdinAvailable = func() bool { return false }
	t.Cleanup(func() { isStdinAvailable = restore })

	dir := t.TempDir()
	path := filepath.Join(dir, "jobs.cron")
	require.NoError(t, os.WriteFile(path, []byte("0 0 * * * /usr/bin/backup.sh\n"), 0o644))

	orig, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(orig) })
	require.NoError(t, os.Chdir(dir))

	locatorFor := func(t *testing.T, spelling string) string {
		t.Helper()
		f := inputFlags{file: spelling}
		items, err := resolveItems(&f)
		require.NoError(t, err)
		require.Len(t, items, 1)
		return items[0].Locator.File
	}

	bare := locatorFor(t, "jobs.cron")
	assert.Equal(t, bare, locatorFor(t, "./jobs.cron"), "a leading ./ must not change the schedule's identity")
	assert.Equal(t, bare, locatorFor(t, path), "an absolute path must not change the schedule's identity")
	assert.Equal(t, "jobs.cron", bare)
}
