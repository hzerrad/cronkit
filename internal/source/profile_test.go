package source

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func testProfile() Profile {
	return Profile{
		ID:          "test",
		Extensions:  []string{".yaml", ".yml"},
		Match:       []FieldMatch{{Path: "kind", Equals: "CronJob"}},
		Schedules:   []Path{"spec.schedule"},
		Timezone:    "spec.timeZone",
		Suspend:     "spec.suspend",
		Concurrency: "spec.concurrencyPolicy",
		Command:     "spec.image",
		Dialect:     "vixie",
	}
}

func extract(t *testing.T, p Profile, name, content string) []inventory.Item {
	t.Helper()
	src, err := NewProfileSource(p)
	require.NoError(t, err)

	fsys := fstest.MapFS{name: &fstest.MapFile{Data: []byte(content)}}
	info, err := fs.Stat(fsys, name)
	require.NoError(t, err)

	unit := Unit{Path: name, Info: info}
	require.True(t, src.Match(unit), "the fixture should match the profile")

	items, err := src.Extract(unit, fsys)
	require.NoError(t, err)
	return items
}

func TestProfileSource_ExtractsAScheduleWithProvenance(t *testing.T) {
	items := extract(t, testProfile(), "k8s/backup.yaml", `apiVersion: batch/v1
kind: CronJob
spec:
  schedule: "0 2 * * *"
  timeZone: Europe/Paris
  concurrencyPolicy: Forbid
  image: busybox
`)

	require.Len(t, items, 1)
	item := items[0]
	assert.Equal(t, "0 2 * * *", item.Expression)
	assert.Equal(t, "test", item.SourceID)
	assert.Equal(t, "vixie", item.Dialect)
	assert.Equal(t, "Europe/Paris", item.Timezone)
	assert.Equal(t, "busybox", item.Command)
	assert.False(t, item.Shell, "a container image is not a shell command")
	assert.Equal(t, inventory.ConcurrencyForbid, item.Concurrency)
	assert.Equal(t, inventory.StateActive, item.State)
	assert.Equal(t, "k8s/backup.yaml", item.Locator.File)
	assert.Equal(t, 4, item.Locator.Line)
	assert.Equal(t, "spec.schedule", item.Locator.Path)
}

func TestProfileSource_SuspendedSchedule(t *testing.T) {
	items := extract(t, testProfile(), "a.yaml", `kind: CronJob
spec:
  schedule: "0 2 * * *"
  suspend: true
`)

	require.Len(t, items, 1)
	assert.Equal(t, inventory.StateSuspended, items[0].State)
	assert.NotEmpty(t, items[0].Reason, "a non-active state must say why")
}

func TestProfileSource_RecognisesEverySuspendSpelling(t *testing.T) {
	spellings := []string{"true", "True", "TRUE", "yes", "Yes", "YES", "on", "On", "ON"}

	for _, spelling := range spellings {
		t.Run(spelling, func(t *testing.T) {
			items := extract(t, testProfile(), "a.yaml", fmt.Sprintf(`kind: CronJob
spec:
  schedule: "0 2 * * *"
  suspend: %s
`, spelling))

			require.Len(t, items, 1)
			assert.Equal(t, inventory.StateSuspended, items[0].State,
				"%q is a genuine boolean true to the YAML parsers these manifests use", spelling)
		})
	}
}

func TestProfileSource_TemplatedSuspendIsUnresolved(t *testing.T) {
	items := extract(t, testProfile(), "a.yaml", `kind: CronJob
spec:
  schedule: "0 2 * * *"
  suspend: "{{ .Values.suspend }}"
`)

	require.Len(t, items, 1)
	assert.Equal(t, inventory.StateUnresolved, items[0].State,
		"a templated suspend flag means we cannot know whether the job runs")
	assert.NotEmpty(t, items[0].Reason)
}

func TestProfileSource_InvalidConcurrencyPolicyIsUnspecified(t *testing.T) {
	items := extract(t, testProfile(), "a.yaml", `kind: CronJob
spec:
  schedule: "0 2 * * *"
  concurrencyPolicy: Nonsense
`)

	require.Len(t, items, 1)
	assert.Equal(t, inventory.ConcurrencyUnspecified, items[0].Concurrency,
		"a policy spelling the source does not recognise must not be reported as a real one")
}

func TestProfileSource_ID(t *testing.T) {
	src, err := NewProfileSource(testProfile())
	require.NoError(t, err)

	assert.Equal(t, "test", src.ID())
}

func TestProfileSource_TemplatedExpressionIsUnresolved(t *testing.T) {
	items := extract(t, testProfile(), "a.yaml", `kind: CronJob
spec:
  schedule: "{{ .Values.backup.cron }}"
`)

	require.Len(t, items, 1)
	assert.Equal(t, inventory.StateUnresolved, items[0].State)
	assert.NotEmpty(t, items[0].Reason)
}

func TestProfileSource_InvalidExpressionIsRecordedNotDropped(t *testing.T) {
	items := extract(t, testProfile(), "a.yaml", `kind: CronJob
spec:
  schedule: "99 99 * * *"
`)

	require.Len(t, items, 1, "an unparseable schedule is still inventoried")
	assert.Equal(t, inventory.StateInvalid, items[0].State)
	assert.NotEmpty(t, items[0].Reason)
}

func TestProfileSource_SuspendedInvalidExpressionStaysSuspended(t *testing.T) {
	items := extract(t, testProfile(), "a.yaml", `kind: CronJob
spec:
  schedule: "99 99 * * *"
  suspend: true
`)

	require.Len(t, items, 1)
	assert.Equal(t, inventory.StateSuspended, items[0].State,
		"a suspended schedule stays suspended even when its expression would also fail validation")
}

func TestProfileSource_TemplatedExpressionIsNotValidated(t *testing.T) {
	items := extract(t, testProfile(), "a.yaml", `kind: CronJob
spec:
  schedule: "{{ .Values.backup.cron }}"
`)

	require.Len(t, items, 1)
	assert.Equal(t, inventory.StateUnresolved, items[0].State,
		"a templated expression is not meant to parse and must not be reported invalid")
}

func TestProfileSource_SkipsDocumentsThatDoNotMatch(t *testing.T) {
	items := extract(t, testProfile(), "a.yaml", `kind: Deployment
spec:
  schedule: "0 2 * * *"
`)

	assert.Empty(t, items, "a document failing the matcher yields nothing, not an error")
}

func TestProfileSource_MultiDocumentRecordsTrueDocumentIndex(t *testing.T) {
	items := extract(t, testProfile(), "a.yaml", `---
---
---
kind: CronJob
spec:
  schedule: "0 2 * * *"
`)

	require.Len(t, items, 1)
	assert.Equal(t, 2, items[0].Locator.Document,
		"empty documents are skipped but still counted")
}

func TestProfileSource_MultiValuedScheduleGetsDistinctLocators(t *testing.T) {
	p := testProfile()
	p.Schedules = []Path{"spec.schedules[]"}

	items := extract(t, p, "a.yaml", `kind: CronJob
spec:
  schedules:
    - "0 2 * * *"
    - "0 3 * * *"
`)

	require.Len(t, items, 2)
	assert.Equal(t, "spec.schedules[0]", items[0].Locator.Path)
	assert.Equal(t, "spec.schedules[1]", items[1].Locator.Path)
	assert.NotEqual(t, items[0].Locator, items[1].Locator,
		"distinct items must never share a locator")
}

func TestProfileSource_FallsBackToTheNextSchedulePath(t *testing.T) {
	p := testProfile()
	p.Schedules = []Path{"spec.schedule", "spec.schedules[]"}

	items := extract(t, p, "a.yaml", `kind: CronJob
spec:
  schedules:
    - "0 3 * * *"
`)

	require.Len(t, items, 1, "a path matching nothing is skipped, the next path still applies")
	assert.Equal(t, "0 3 * * *", items[0].Expression)
}

func TestProfileSource_FirstMatchingSchedulePathWinsOverLaterOnes(t *testing.T) {
	// Schedules is fallback, not a union, so a document with both fields must report the schedule once.
	p := testProfile()
	p.Schedules = []Path{"spec.schedules[]", "spec.schedule"}

	items := extract(t, p, "a.yaml", `kind: CronJob
spec:
  schedule: "0 2 * * *"
  schedules:
    - "0 3 * * *"
`)

	require.Len(t, items, 1, "only the first path that matches anything should contribute items")
	assert.Equal(t, "0 3 * * *", items[0].Expression)
	assert.Equal(t, "spec.schedules[0]", items[0].Locator.Path)
}

func TestProfileSource_MatchUsesExtensionAndDirectoryPrefix(t *testing.T) {
	p := testProfile()
	p.DirPrefix = ".github/workflows"
	src, err := NewProfileSource(p)
	require.NoError(t, err)

	assert.True(t, src.Match(Unit{Path: ".github/workflows/ci.yml"}))
	assert.False(t, src.Match(Unit{Path: "k8s/backup.yaml"}), "wrong directory")
	assert.False(t, src.Match(Unit{Path: ".github/workflows/README.md"}), "wrong extension")
	assert.False(t, src.Match(Unit{Path: ".github/workflows/nested/ci.yml"}),
		"GitHub does not read workflows from subdirectories of the workflows directory")
}

func TestProfileSource_MatchIsCaseInsensitiveOnExtension(t *testing.T) {
	p := testProfile()
	src, err := NewProfileSource(p)
	require.NoError(t, err)

	assert.True(t, src.Match(Unit{Path: "k8s/backup.YAML"}))
	assert.True(t, src.Match(Unit{Path: "k8s/backup.Yml"}))
}

func TestNewProfileSource_RejectsInvalidPaths(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Profile)
		message string
	}{
		{"empty schedule list", func(p *Profile) { p.Schedules = nil }, "at least one schedule path"},
		{"malformed schedule", func(p *Profile) { p.Schedules = []Path{"spec.schedules[0"} }, "unterminated"},
		{"malformed optional path", func(p *Profile) { p.Timezone = "spec..timeZone" }, "empty segment"},
		{"malformed matcher path", func(p *Profile) { p.Match = []FieldMatch{{Path: "", Equals: "x"}} }, "empty"},
		{"missing id", func(p *Profile) { p.ID = "" }, "id"},
		{"no extensions", func(p *Profile) { p.Extensions = nil }, "extension"},
		{"extension missing leading dot", func(p *Profile) { p.Extensions = []string{"yaml"} }, "extension"},
		{"extension is only a dot", func(p *Profile) { p.Extensions = []string{"."} }, "extension"},
		{"dir prefix with leading slash", func(p *Profile) { p.DirPrefix = "/.github/workflows" }, "dir prefix"},
		{"dir prefix with trailing slash", func(p *Profile) { p.DirPrefix = ".github/workflows/" }, "dir prefix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testProfile()
			tt.mutate(&p)

			_, err := NewProfileSource(p)

			require.Error(t, err, "a malformed profile must fail when built, not silently at scan time")
			assert.Contains(t, err.Error(), tt.message)
		})
	}
}

func TestNewProfileSource_AllowsEmptyOptionalPaths(t *testing.T) {
	p := testProfile()
	p.Timezone = ""
	p.Suspend = ""
	p.Concurrency = ""
	p.Command = ""

	_, err := NewProfileSource(p)

	require.NoError(t, err, "an omitted optional path is not an error")
}

func TestProfileSource_EmptyOptionalPathsMatchNothing(t *testing.T) {
	p := testProfile()
	p.Timezone = ""
	p.Command = ""

	items := extract(t, p, "a.yaml", `kind: CronJob
spec:
  schedule: "0 2 * * *"
  timeZone: Europe/Paris
  image: busybox
`)

	require.Len(t, items, 1)
	assert.Empty(t, items[0].Timezone,
		"an omitted path must match nothing, never the whole document")
	assert.Empty(t, items[0].Command)
}

func TestFirstRejectsEmptyPath(t *testing.T) {
	docs, err := DecodeYAML([]byte("kind: CronJob\nspec:\n  schedule: \"0 2 * * *\"\n"))
	require.NoError(t, err)
	require.Len(t, docs, 1)

	// The document is not empty, so this proves the empty-path guard lives in first.
	v, found := first(docs[0].Root, "")

	assert.False(t, found, "an empty path must never be reported as found")
	assert.Empty(t, v)
}

func TestProfileSource_FixedTimezoneWins(t *testing.T) {
	p := testProfile()
	p.Timezone = ""
	p.TimezoneFixed = "UTC"

	items := extract(t, p, "a.yaml", "kind: CronJob\nspec:\n  schedule: \"0 2 * * *\"\n")

	require.Len(t, items, 1)
	assert.Equal(t, "UTC", items[0].Timezone,
		"platforms with no timezone field declare a fixed one")
}

func TestProfileSource_ResolvedTimezoneWinsOverFixed(t *testing.T) {
	p := testProfile()
	p.TimezoneFixed = "UTC"

	items := extract(t, p, "a.yaml", `kind: CronJob
spec:
  schedule: "0 2 * * *"
  timeZone: Europe/Paris
`)

	require.Len(t, items, 1)
	assert.Equal(t, "Europe/Paris", items[0].Timezone,
		"a timezone actually found in the document must win over the fixed default")
}

func TestProfileSource_LiftsAnInlineTimezoneOutOfTheExpression(t *testing.T) {
	p := testProfile()
	p.Timezone = ""

	items := extract(t, p, "a.yaml", `kind: CronJob
spec:
  schedule: "CRON_TZ=Asia/Tokyo 0 2 * * *"
`)

	require.Len(t, items, 1)
	assert.Equal(t, "0 2 * * *", items[0].Expression,
		"the assignment must not remain part of the schedule")
	assert.Equal(t, "Asia/Tokyo", items[0].Timezone)
	assert.Equal(t, inventory.StateActive, items[0].State)
}

func TestProfileSource_RejectsAnInlineTimezoneThatDoesNotExist(t *testing.T) {
	// A zone lifted out of the expression must be validated at the point it is lifted.
	p := testProfile()
	p.Timezone = ""

	items := extract(t, p, "a.yaml", `kind: CronJob
spec:
  schedule: "CRON_TZ=Not/AZone 0 2 * * *"
`)

	require.Len(t, items, 1)
	assert.Equal(t, inventory.StateInvalid, items[0].State)
	assert.Contains(t, items[0].Reason, "Not/AZone")
	assert.Equal(t, "CRON_TZ=Not/AZone 0 2 * * *", items[0].Expression,
		"the original text is kept, not the schedule with the bad assignment stripped off")
	assert.Empty(t, items[0].Timezone,
		"an invalid zone must never become the item's timezone")
}

func TestProfileSource_TimezoneAssignmentWithNoScheduleNamesTheRealProblem(t *testing.T) {
	// A bare timezone assignment must report cronx.Parse's specific message, not "empty expression".
	p := testProfile()
	p.Timezone = ""

	items := extract(t, p, "a.yaml", `kind: CronJob
spec:
  schedule: "CRON_TZ=Asia/Tokyo"
`)

	require.Len(t, items, 1)
	assert.Equal(t, inventory.StateInvalid, items[0].State)
	assert.Contains(t, items[0].Reason, "no schedule after it")
	assert.NotEqual(t, "empty expression", items[0].Reason)
	assert.Equal(t, "CRON_TZ=Asia/Tokyo", items[0].Expression,
		"the original text is kept, not blanked to an empty schedule")
	assert.Empty(t, items[0].Timezone,
		"a timezone with no schedule must never become the item's timezone")
}

func TestProfileSource_QuotedInlineTimezoneIsUnquoted(t *testing.T) {
	// A manifest's CRON_TZ='Asia/Tokyo' must unquote to the same clean zone as an unquoted assignment.
	p := testProfile()
	p.Timezone = ""

	items := extract(t, p, "a.yaml", `kind: CronJob
spec:
  schedule: "CRON_TZ='Asia/Tokyo' 0 2 * * *"
`)

	require.Len(t, items, 1)
	assert.Equal(t, "0 2 * * *", items[0].Expression)
	assert.Equal(t, "Asia/Tokyo", items[0].Timezone)
	assert.Equal(t, inventory.StateActive, items[0].State)
}

func TestProfileSource_ExplicitTimezoneFieldWinsOverAnInlinePrefix(t *testing.T) {
	// The explicit timezone field wins over an inline CRON_TZ= prefix, which is still stripped from Expression.
	items := extract(t, testProfile(), "a.yaml", `kind: CronJob
spec:
  schedule: "CRON_TZ=Asia/Tokyo 0 2 * * *"
  timeZone: Europe/Paris
`)

	require.Len(t, items, 1)
	assert.Equal(t, "0 2 * * *", items[0].Expression)
	assert.Equal(t, "Europe/Paris", items[0].Timezone,
		"the explicit field wins over the inline prefix")
}

func TestProfileSource_ScheduleWithNoPrefixIsUnchanged(t *testing.T) {
	items := extract(t, testProfile(), "a.yaml", `kind: CronJob
spec:
  schedule: "0 2 * * *"
  timeZone: Europe/Paris
`)

	require.Len(t, items, 1)
	assert.Equal(t, "0 2 * * *", items[0].Expression)
	assert.Equal(t, "Europe/Paris", items[0].Timezone)
}

func TestProfileSource_MalformedYAMLIsAnError(t *testing.T) {
	src, err := NewProfileSource(testProfile())
	require.NoError(t, err)

	fsys := fstest.MapFS{"a.yaml": &fstest.MapFile{Data: []byte("kind: [unclosed\n")}}
	info, err := fs.Stat(fsys, "a.yaml")
	require.NoError(t, err)

	_, err = src.Extract(Unit{Path: "a.yaml", Info: info}, fsys)

	require.Error(t, err, "a file we claimed to recognise and then could not read is reportable")
}

// fullyPopulatedProfile sets every field, including the ones the built-in
// profiles happen to leave zero, so a round trip exercises every tag.
func fullyPopulatedProfile() Profile {
	return Profile{
		ID:            "test",
		Extensions:    []string{".yaml", ".yml"},
		DirPrefix:     ".github/workflows",
		Match:         []FieldMatch{{Path: "kind", Equals: "CronJob"}},
		Schedules:     []Path{"spec.schedule", "spec.schedules[]"},
		Timezone:      "spec.timeZone",
		TimezoneFixed: "UTC",
		Suspend:       "spec.suspend",
		Concurrency:   "spec.concurrencyPolicy",
		Command:       "spec.image",
		Dialect:       "vixie",
		Shell:         true,
	}
}

func TestProfile_JSONRoundTrip(t *testing.T) {
	want := fullyPopulatedProfile()

	data, err := json.Marshal(want)
	require.NoError(t, err)

	var got Profile
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, want, got)
}

func TestProfile_YAMLRoundTrip(t *testing.T) {
	want := fullyPopulatedProfile()

	data, err := yaml.Marshal(want)
	require.NoError(t, err)

	var got Profile
	require.NoError(t, yaml.Unmarshal(data, &got))

	assert.Equal(t, want, got)
}

// wantProfileKeys and wantFieldMatchKeys pin the emitted wire keys; a round trip can't catch a wrong tag.
var (
	wantProfileKeys = []string{
		"id", "extensions", "dir_prefix", "match", "schedules",
		"timezone", "timezone_fixed", "suspend", "concurrency",
		"command", "dialect", "shell",
	}
	wantFieldMatchKeys = []string{"path", "equals"}
)

// keys returns a map's keys in no particular order, for comparing against an
// expected set with assert.ElementsMatch.
func keys[K comparable, V any](m map[K]V) []K {
	ks := make([]K, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func TestProfile_JSONWireNames(t *testing.T) {
	data, err := json.Marshal(fullyPopulatedProfile())
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.ElementsMatch(t, wantProfileKeys, keys(raw), "JSON keys emitted for Profile")

	var matches []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw["match"], &matches))
	require.Len(t, matches, 1)
	assert.ElementsMatch(t, wantFieldMatchKeys, keys(matches[0]), "JSON keys emitted for FieldMatch")
}

func TestProfile_YAMLWireNames(t *testing.T) {
	data, err := yaml.Marshal(fullyPopulatedProfile())
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &raw))
	assert.ElementsMatch(t, wantProfileKeys, keys(raw), "YAML keys emitted for Profile")

	matches, ok := raw["match"].([]interface{})
	require.True(t, ok, "match must decode as a sequence")
	require.Len(t, matches, 1)
	matchEntry, ok := matches[0].(map[string]interface{})
	require.True(t, ok, "a match entry must decode as a mapping")
	assert.ElementsMatch(t, wantFieldMatchKeys, keys(matchEntry), "YAML keys emitted for FieldMatch")
}
