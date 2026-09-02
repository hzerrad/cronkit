# Test Data

This directory contains test fixtures used by unit, integration, and E2E tests.

## Structure

```
testdata/
├── crontab/
│   ├── valid/          # Valid crontab files
│   ├── invalid/        # Invalid crontab files (for error testing)
│   ├── edge-cases/     # Edge case scenarios
│   ├── performance/    # Large crontabs for performance testing
│   ├── sample.cron     # Sample crontab with various patterns
│   ├── empty.cron      # Empty crontab
│   └── invalid.cron    # Crontab with invalid entries
├── sources/             # Conformance corpus for internal/source sources
│   └── <source>/<case>/
│       ├── input.*      # The file a source is run against
│       └── expected.json # The []inventory.Item it must produce
└── expressions.json    # Test cron expressions
```

## Source Conformance Corpus

`testdata/sources/<source>/<case>/` holds one fixture for `internal/source`'s
`TestConformance`: `input.*` is the file to extract from, and `expected.json`
is the exact `[]inventory.Item` extraction must produce, checked field for
field. The directory name under `sources/` must match a registered source's
ID (e.g. `crontab`, `k8s`, `argo`, `gha`); each case beneath it is an
independent scenario named for what it exercises (e.g. `suspended`,
`legacy-schedule`). Not every source is a `Profile` — `crontab` is a
hand-written `Source`, since crontabs are line-oriented rather than
tree-shaped — but both kinds of source use this same corpus layout.

A case that deliberately spans more than one source — e.g. a multi-document
file where one document matches one ecosystem's `Match` criteria and another
document matches a different one — does not belong under any single source's
directory, since `sources/<source>/` means "cases for this source" and a
cross-source case would misrepresent which source it tests. Such a case
goes under the reserved `testdata/sources/mixed/` directory instead.
`TestConformance` still runs every case beneath it through the full `Default`
registry the same as any other, but exempts `mixed` from the "directory name
matches a registered source ID" check and from the "every source has a
corpus case" completeness check, since it names no single source on purpose.

`input.*` normally sits at the case's root, but a source that restricts
matching to a specific directory (`Profile.DirPrefix`, e.g. GitHub Actions'
`.github/workflows`; or crontab's `cron.d` convention, which also switches on
the parent directory to decide the system vs. user command format) needs its
fixture nested the same way a real repository would lay it out, since
matching depends on the file's real parent directory.

Adding support for a new source — whether it is a data-driven `Profile` or a
hand-written `Source` like `crontab` — is two steps together, not just this
directory:

1. Add the fixture directory here, under `testdata/sources/<id>/`.
2. Register the source in `internal/source/profiles.go`'s `Default()`, so it
   actually runs during a scan.

That is deliberately the whole list: `TestConformance` in
`internal/source/profiles_test.go` reads the set of known IDs from the
registry itself — `Default()`'s own `Sources()` — rather than from a
separately hand-maintained list, so there is no third place to remember to
update. A corpus directory whose name matches no ID the registry actually
produced fails with `corpus directory "<id>" names no registered source`
instead of silently contributing no cases. `TestConformance` checks the
mirror direction too: after walking the corpus it asserts every ID the
registry produced was visited, failing with `sources with no corpus
directory under testdata/sources` and naming them if any registered source
has no case at all. A source registered in `Default()` but never given a
corpus directory is exactly the gap that check exists to close.

## Usage in Tests

### Loading Test Fixtures

```go
import "github.com/hzerrad/cronkit/internal/testutil"

// Load a test crontab
path := testutil.LoadTestCrontab("sample.cron")

// Create temporary crontab
file, cleanup := testutil.CreateTempCrontab(t, "0 2 * * * /usr/bin/backup.sh")
defer cleanup()
```

### Direct File Access

```go
// In tests, use relative paths from test file location
testFile := filepath.Join("..", "..", "testdata", "crontab", "sample.cron")
```

## Test Fixture Guidelines

- **Valid fixtures**: Should contain valid cron expressions for positive testing
- **Invalid fixtures**: Should contain various types of invalid entries for error testing
- **Edge cases**: Should cover boundary conditions and unusual but valid patterns
- **Performance fixtures**: Should be large (100+ jobs) for performance benchmarking

## Adding New Fixtures

1. Place files in appropriate subdirectory
2. Use descriptive names (e.g., `dom-dow-conflict.cron`)
3. Add comments explaining the test scenario
4. Update this README if adding new categories



