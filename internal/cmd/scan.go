package cmd

import (
	"fmt"
	"strings"

	"github.com/hzerrad/cronkit/internal/discover"
	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/hzerrad/cronkit/internal/source"
	"github.com/spf13/cobra"
)

type ScanCommand struct {
	*cobra.Command
	json           bool
	sources        []string
	excludeSources []string
	excludes       []string
	noIgnore       bool
	maxFileSize    int64
	strict         bool
}

func newScanCommand() *ScanCommand {
	sc := &ScanCommand{}
	sc.Command = &cobra.Command{
		Use:   "scan [paths...]",
		Short: "Discover cron schedules across a repository",
		Long: `Discover cron schedules across a repository: crontabs, Kubernetes CronJobs,
GitHub Actions workflows, and Argo CronWorkflows.

scan discovers; it does not audit. There is deliberately no --fail-on
here — severity gating belongs to "cronkit check". The eventual
pipeline is "cronkit scan . | cronkit check --inventory -".

Exit codes:
  0  the walk completed, including when it found nothing at all
     ("no schedules in this repository" is a true answer, and a scan
     that failed on a repo without crons would be unusable in CI)
  1  the walk could not run at all: an unreadable root, an unknown
     --source or --exclude-source ID, or a malformed flag
  1  with --strict, the walk completed but reported one or more
     per-file problems

Examples:
  cronkit scan                          # scan the current directory
  cronkit scan ./services ./infra       # scan multiple roots
  cronkit scan --json                   # emit the inventory JSON contract
  cronkit scan --source crontab,k8s     # only run these sources
  cronkit scan --exclude-source gha     # run every source except this one
  cronkit scan --exclude build,'*.bak'  # skip a directory (any depth) and a suffix, anywhere
  cronkit scan --exclude build/gen.txt  # skip that one path only
  cronkit scan --strict                 # fail the exit code on any per-file problem`,
		RunE: sc.runScan,
	}

	sc.Flags().BoolVarP(&sc.json, "json", "j", false, "Emit the inventory as JSON")
	sc.Flags().StringSliceVar(&sc.sources, "source", nil, "Only run these sources: 'crontab', 'k8s', 'argo', or 'gha' (comma-separated)")
	sc.Flags().StringSliceVar(&sc.excludeSources, "exclude-source", nil, "Run every source except these: 'crontab', 'k8s', 'argo', or 'gha' (comma-separated)")
	sc.Flags().StringSliceVar(&sc.excludes, "exclude", nil, "Skip matching paths (comma-separated): a pattern with a \"/\" matches one full path, a pattern without one matches any path segment (e.g. a directory name, at any depth)")
	sc.Flags().BoolVar(&sc.noIgnore, "no-ignore", false, "Do not honour .gitignore")
	sc.Flags().Int64Var(&sc.maxFileSize, "max-file-size", 0, "Skip files larger than this many bytes (0 means no limit)")
	sc.Flags().BoolVar(&sc.strict, "strict", false, "Exit non-zero if the walk reported any per-file problem")

	return sc
}

func init() {
	rootCmd.AddCommand(newScanCommand().Command)
}

func (sc *ScanCommand) runScan(_ *cobra.Command, args []string) error {
	roots := args
	if len(roots) == 0 {
		roots = []string{"."}
	}

	registry, err := source.Default()
	if err != nil {
		return fmt.Errorf("failed to build source registry: %w", err)
	}

	if err := validateSourceIDs(registry, "--source", sc.sources); err != nil {
		return err
	}
	if err := validateSourceIDs(registry, "--exclude-source", sc.excludeSources); err != nil {
		return err
	}

	opts := discover.Options{
		EnumerateOptions: discover.EnumerateOptions{
			MaxFileSize: sc.maxFileSize,
			NoIgnore:    sc.noIgnore,
			Excludes:    sc.excludes,
		},
		Sources:        sc.sources,
		ExcludeSources: sc.excludeSources,
	}

	inv, problems, err := discover.New(registry, opts).Walk(roots)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	if sc.json {
		inv.Root = displayPath(inv.Root)
		if err := inv.Encode(sc.OutOrStdout()); err != nil {
			return fmt.Errorf("failed to encode inventory: %w", err)
		}
	} else {
		sc.printText(inv)
	}

	// Problems are diagnostics, so they go to stderr, keeping --json piped into jq clean.
	for _, p := range problems {
		sc.PrintErrf("scan: %s: %v\n", p.Path, p.Err)
	}

	if sc.strict && len(problems) > 0 {
		osExit(1)
	}

	return nil
}

// validateSourceIDs checks every id against registry's known sources, naming the bad id and valid ones.
func validateSourceIDs(registry *source.Registry, flag string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	known := registry.Sources()
	valid := make([]string, len(known))
	validSet := make(map[string]bool, len(known))
	for i, s := range known {
		valid[i] = s.ID()
		validSet[s.ID()] = true
	}

	for _, id := range ids {
		if !validSet[id] {
			return fmt.Errorf("unknown %s %q (valid sources: %s)", flag, id, strings.Join(valid, ", "))
		}
	}
	return nil
}

// printText renders a human-readable table of the discovered inventory, fit to the terminal width.
func (sc *ScanCommand) printText(inv *inventory.Inventory) {
	width := resolveWidth(0, stdoutTTY())
	renderScanText(sc.OutOrStdout(), inv, width)
}
