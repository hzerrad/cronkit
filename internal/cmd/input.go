package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/hzerrad/cronkit/internal/crontab"
	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/spf13/cobra"
)

// inputFlags holds the --file, --stdin and --inventory flags shared by every schedule-consuming command.
type inputFlags struct {
	file      string
	stdin     bool
	inventory string
	autoStdin bool
}

// register adds --file, --stdin and --inventory to cmd.
func (f *inputFlags) register(cmd *cobra.Command, autoStdin bool) {
	f.autoStdin = autoStdin

	cmd.Flags().StringVarP(&f.file, "file", "f", "", "Path to crontab file (defaults to user's crontab if not specified)")

	stdinHelp := "Read crontab from standard input"
	if autoStdin {
		stdinHelp += " (automatic if stdin is not a terminal)"
	}
	cmd.Flags().BoolVar(&f.stdin, "stdin", false, stdinHelp)

	cmd.Flags().StringVar(&f.inventory, "inventory", "",
		"Path to an inventory JSON file produced by 'cronkit scan', or - to read it from standard input")
}

// inputSource names which tier of the precedence ladder resolveItems resolved to.
// Callers use it for source-specific error handling without re-deriving the ladder.
type inputSource int

const (
	sourceInventory inputSource = iota
	sourceFile
	sourceStdin
	sourceUserCrontab
)

// classify reports which source resolveItems would read from, without reading anything.
func (f *inputFlags) classify() inputSource {
	switch {
	case f.inventory != "":
		return sourceInventory
	case f.file != "":
		return sourceFile
	case f.stdin:
		return sourceStdin
	case f.autoStdin && isStdinAvailable():
		return sourceStdin
	default:
		return sourceUserCrontab
	}
}

// resolveItems reads schedules by flag precedence, then resolves timezones via inventory.ResolveTimezones.
// An unresolvable timezone is marked StateInvalid rather than vanishing silently.
func resolveItems(flags *inputFlags) ([]inventory.Item, error) {
	items, err := readItems(flags)
	if err != nil {
		return nil, err
	}
	return inventory.ResolveTimezones(items), nil
}

// readItems performs the read resolveItems wraps with timezone resolution.
func readItems(flags *inputFlags) ([]inventory.Item, error) {
	if flags.classify() == sourceInventory {
		return readInventoryItems(flags.inventory)
	}

	reader := crontab.NewReader()
	var jobs []*crontab.Job
	var file string
	var err error

	switch flags.classify() {
	case sourceFile:
		// A job id is built from the locator, so it must not carry the spelling the user typed.
		file = displayPath(flags.file)
		jobs, err = reader.ReadFile(flags.file)
	case sourceStdin:
		jobs, err = reader.ReadStdin()
	default:
		jobs, err = reader.ReadUser()
	}
	if err != nil {
		return nil, err
	}

	return inventory.FromCrontabJobs(jobs, file), nil
}

// inputSpansFiles reports whether items carry more than one distinct Locator.File.
// Ask it of the whole input, not a subset that happened to produce a finding.
func inputSpansFiles(items []inventory.Item) bool {
	files := make(map[string]struct{})
	for _, item := range items {
		if item.Locator.File == "" {
			continue
		}
		files[item.Locator.File] = struct{}{}
		if len(files) > 1 {
			return true
		}
	}
	return false
}

// readInventoryItems reads and decodes the inventory JSON contract from
// path, or from standard input when path is "-".
func readInventoryItems(path string) ([]inventory.Item, error) {
	if path == "-" {
		inv, err := inventory.Decode(os.Stdin)
		if err != nil {
			return nil, err
		}
		return inv.Items, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	inv, err := inventory.Decode(f)
	if err != nil {
		return nil, err
	}
	return inv.Items, nil
}

// displayPath renders p relative to the invocation directory when it sits inside it, else absolute.
func displayPath(p string) string {
	if p == "" {
		return p
	}

	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}

	wd, err := os.Getwd()
	if err != nil {
		return filepath.ToSlash(abs)
	}

	rel, err := filepath.Rel(wd, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		// p sits outside wd's subtree, not expressible relative to it, so it stays absolute.
		return filepath.ToSlash(abs)
	}

	return filepath.ToSlash(rel)
}
