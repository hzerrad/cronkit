package inventory

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocator_String(t *testing.T) {
	assert.Equal(t, "ops/crontab:7",
		Locator{File: "ops/crontab", Line: 7}.String())
	assert.Equal(t, "k8s/backup.yaml:12 (spec.schedule)",
		Locator{File: "k8s/backup.yaml", Line: 12, Path: "spec.schedule"}.String())
	assert.Equal(t, "k8s/all.yaml",
		Locator{File: "k8s/all.yaml"}.String())
}

// TestLocator_Identity pins the shared job-identity rule: an identity is an address, not a position.
func TestLocator_Identity(t *testing.T) {
	t.Run("is exactly linePrefix+line when there is no file", func(t *testing.T) {
		id := Locator{Line: 6}.Identity(0, "job-", "line-")
		assert.Equal(t, "line-6", id, "no file to fold in -- the id stays the bare line, e.g. stdin or the user's own crontab")
	})

	t.Run("folds in the file whenever it is set, regardless of whether the line recurs elsewhere", func(t *testing.T) {
		id := Locator{File: "crontab", Line: 6}.Identity(0, "job-", "line-")
		assert.Equal(t, "line-crontab:6", id)
	})

	t.Run("distinguishes two items on the same line in different files", func(t *testing.T) {
		a := Locator{File: "site-a/crontab", Line: 6}.Identity(0, "job-", "line-")
		b := Locator{File: "site-b/crontab", Line: 6}.Identity(1, "job-", "line-")
		assert.NotEqual(t, a, b, "two distinct items on the same line in different files must not share an identity")
		assert.Equal(t, "line-site-a/crontab:6", a)
		assert.Equal(t, "line-site-b/crontab:6", b)
	})

	t.Run("falls back to index when the locator addresses no position", func(t *testing.T) {
		for name, locator := range map[string]Locator{
			"empty":     {},
			"file only": {File: "crontab"},
		} {
			t.Run(name, func(t *testing.T) {
				a := locator.Identity(0, "job-", "line-")
				b := locator.Identity(1, "job-", "line-")
				assert.NotEqual(t, a, b, "two unaddressable items must not share an identity")
				assert.Equal(t, "job-0", a)
				assert.Equal(t, "job-1", b)
			})
		}
	})

	t.Run("indexPrefix and linePrefix let each caller keep its own id format", func(t *testing.T) {
		// Callers' index and line prefixes must be honoured exactly as given.
		assert.Equal(t, "job-3", Locator{Line: 3}.Identity(0, "job-", "job-"))
		assert.Equal(t, "line-3", Locator{Line: 3}.Identity(0, "job-", "line-"))
	})
}

func TestInventory_SortIsTotal(t *testing.T) {
	inv := New("/repo", []Item{
		{Expression: "e", Locator: Locator{File: "b.yaml", Line: 1}},
		{Expression: "c", Locator: Locator{File: "a.yaml", Document: 1, Line: 1}},
		{Expression: "b", Locator: Locator{File: "a.yaml", Line: 9, Path: "spec.b"}},
		{Expression: "a", Locator: Locator{File: "a.yaml", Line: 9, Path: "spec.a"}},
		{Expression: "d", Locator: Locator{File: "a.yaml", Document: 1, Line: 4}},
	})

	inv.Sort()

	var got []string
	for _, item := range inv.Items {
		got = append(got, item.Expression)
	}
	assert.Equal(t, []string{"a", "b", "c", "d", "e"}, got,
		"sort by file, then document, then line, then path")
}

func TestInventory_EncodeDecodeRoundTrip(t *testing.T) {
	inv := New("/repo", []Item{{
		Expression:  "0 2 * * *",
		SourceID:    "k8s",
		Dialect:     "vixie",
		Command:     "busybox",
		Shell:       false,
		RunAs:       "root",
		Timezone:    "Europe/Paris",
		State:       StateSuspended,
		Reason:      "spec.suspend is true",
		Concurrency: ConcurrencyForbid,
		Locator:     Locator{File: "k8s/backup.yaml", Line: 12, Path: "spec.schedule"},
	}})

	var buf bytes.Buffer
	require.NoError(t, inv.Encode(&buf))

	decoded, err := Decode(&buf)
	require.NoError(t, err)
	assert.Equal(t, inv, decoded)
}

func TestInventory_EncodeOmitsEmptyOptionalFields(t *testing.T) {
	inv := New("", []Item{{
		Expression: "0 2 * * *",
		SourceID:   "crontab",
		Dialect:    "vixie",
		Shell:      true,
		State:      StateActive,
		Locator:    Locator{File: "ops/crontab", Line: 3},
	}})

	var buf bytes.Buffer
	require.NoError(t, inv.Encode(&buf))
	out := buf.String()

	assert.Contains(t, out, `"schemaVersion"`)
	assert.NotContains(t, out, `"root"`)
	assert.NotContains(t, out, `"reason"`)
	assert.NotContains(t, out, `"runAs"`)
	assert.NotContains(t, out, `"timezone"`)
	assert.NotContains(t, out, `"concurrency"`)
	assert.NotContains(t, out, `"document"`)
}

func TestDecode_RejectsSchemaVersionMismatch(t *testing.T) {
	_, err := Decode(strings.NewReader(`{"schemaVersion":"99","items":[]}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "99")
	assert.Contains(t, err.Error(), SchemaVersion,
		"the error must name both the version found and the version supported")
}

func TestDecode_IgnoresUnknownFields(t *testing.T) {
	inv, err := Decode(strings.NewReader(
		`{"schemaVersion":"` + SchemaVersion + `","futureField":true,"items":[]}`))
	require.NoError(t, err, "unknown fields are ignored for forward compatibility")
	assert.Empty(t, inv.Items)
}

func TestDecode_RejectsTrailingContent(t *testing.T) {
	_, err := Decode(strings.NewReader(`{"schemaVersion":"1","items":[]} garbage`))
	require.Error(t, err)
}

func TestDecode_RejectsMissingSchemaVersion(t *testing.T) {
	_, err := Decode(strings.NewReader(`{"items":[]}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not look like an inventory",
		"an absent schemaVersion should not surface the confusing empty-quotes mismatch message")
}

func TestDecode_RejectsEmptySchemaVersion(t *testing.T) {
	_, err := Decode(strings.NewReader(`{"schemaVersion":"","items":[]}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not look like an inventory")
}

func TestDecode_InvalidJSON(t *testing.T) {
	_, err := Decode(strings.NewReader(`{invalid json}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode inventory")
}

func TestDecode_TablePipedInWithoutJSON(t *testing.T) {
	// The exact shape of `cronkit scan` output when --json is forgotten:
	// a table starting with the header line, not JSON at all.
	table := "LINE  PATH              SOURCE   EXPRESSION            DESCRIPTION\n" +
		"────  ────────────────  ───────  ────────────────────  ─────────────────────────\n"

	_, err := Decode(strings.NewReader(table))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not look like an inventory",
		"input that never had a chance of being JSON gets the same courtesy as a missing schemaVersion")
	assert.Contains(t, err.Error(), "--json",
		"the message should name the likely cause: scan run without --json")
	assert.Contains(t, err.Error(), "invalid character",
		"the underlying JSON decode error must still be reachable, not discarded")
	assert.ErrorContains(t, err, "invalid character 'L'")
}

func TestDecode_MalformedJSONStillReportsRealError(t *testing.T) {
	// Starts with '{', so this must report the real parse failure, not the "forgot --json" diagnosis.
	_, err := Decode(strings.NewReader(`{"schemaVersion": "1", "items": [`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode inventory")
	assert.NotContains(t, err.Error(), "does not look like an inventory")
	assert.NotContains(t, err.Error(), "--json")
}

func TestDecode_LeadingWhitespaceBeforeJSONIsStillJSON(t *testing.T) {
	// Whitespace-then-'{' must not be misdiagnosed as "not JSON" -- only the
	// first non-whitespace byte decides.
	_, err := Decode(strings.NewReader("\n\t  {\"schemaVersion\": \"1\", \"items\": [}"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode inventory")
	assert.NotContains(t, err.Error(), "does not look like an inventory")
}

func TestDecode_EmptyInputGetsTheNotAnInventoryDiagnosis(t *testing.T) {
	_, err := Decode(strings.NewReader(""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not look like an inventory")
}

func TestDecode_ValidInventoryStillDecodesNormally(t *testing.T) {
	// Sanity: the new peek-ahead must not disturb the ordinary success path.
	inv, err := Decode(strings.NewReader(`{"schemaVersion":"1","items":[]}`))
	require.NoError(t, err)
	assert.Equal(t, SchemaVersion, inv.SchemaVersion)
}

func TestInventory_EncodeWithNilItemsIsEmptyList(t *testing.T) {
	inv := New("/repo", nil)

	var buf bytes.Buffer
	require.NoError(t, inv.Encode(&buf))

	assert.Contains(t, buf.String(), `"items": []`,
		"a scan that finds nothing must emit [], not null")
}

func TestInventory_Sort_Empty(t *testing.T) {
	inv := New("", []Item{})
	inv.Sort()
	assert.Empty(t, inv.Items)
}

func TestInventory_New(t *testing.T) {
	items := []Item{{Expression: "0 2 * * *", SourceID: "crontab", Dialect: "vixie", State: StateActive, Locator: Locator{File: "crontab"}}}
	inv := New("/root", items)
	assert.Equal(t, SchemaVersion, inv.SchemaVersion)
	assert.Equal(t, "/root", inv.Root)
	assert.Equal(t, items, inv.Items)
}

func TestLocator_String_OnlyFile(t *testing.T) {
	assert.Equal(t, "ops/crontab", Locator{File: "ops/crontab"}.String())
}

func TestDecode_PreservesItems(t *testing.T) {
	input := `{
		"schemaVersion":"1",
		"root":"/repo",
		"items":[
			{
				"expression":"0 2 * * *",
				"source":"crontab",
				"dialect":"vixie",
				"shell":true,
				"state":"active",
				"locator":{"file":"crontab","line":1}
			}
		]
	}`

	inv, err := Decode(strings.NewReader(input))
	require.NoError(t, err)
	assert.Equal(t, "/repo", inv.Root)
	assert.Len(t, inv.Items, 1)
	assert.Equal(t, "0 2 * * *", inv.Items[0].Expression)
}

func TestInventory_EncodeAlwaysIncludesStateAndShell(t *testing.T) {
	inv := New("", []Item{{
		Expression: "0 2 * * *",
		SourceID:   "crontab",
		Dialect:    "vixie",
		Shell:      false,
		State:      StateActive,
		Locator:    Locator{File: "ops/crontab", Line: 3},
	}})

	var buf bytes.Buffer
	require.NoError(t, inv.Encode(&buf))
	out := buf.String()

	assert.Contains(t, out, `"state": "active"`, "state must be present even when StateActive (zero value)")
	assert.Contains(t, out, `"shell": false`, "shell must be present even when false (zero value)")

	buf.Reset()
	require.NoError(t, inv.Encode(&buf))
	decoded, err := Decode(&buf)
	require.NoError(t, err)
	assert.Equal(t, StateActive, decoded.Items[0].State)
	assert.Equal(t, false, decoded.Items[0].Shell)
}

func TestInventory_EncodeIsByteIdentical(t *testing.T) {
	inv := New("/repo", []Item{
		{Expression: "0 2 * * *", SourceID: "k8s", Dialect: "vixie", State: StateActive,
			Locator: Locator{File: "k8s/backup.yaml", Line: 12, Path: "spec.schedule"}},
		{Expression: "0 3 * * *", SourceID: "crontab", Dialect: "vixie", State: StateActive,
			Locator: Locator{File: "ops/crontab", Line: 3}},
	})

	var first, second bytes.Buffer
	require.NoError(t, inv.Encode(&first))
	require.NoError(t, inv.Encode(&second))

	assert.Equal(t, first.Bytes(), second.Bytes(),
		"encoding the same inventory twice must produce byte-identical output")
}

func TestInventory_SortIsDeterministicAcrossInputOrders(t *testing.T) {
	// The fixed target order exercises every sort key, so a comparator missing any one key fails.
	want := []Item{
		{Expression: "x", SourceID: "src-a", Locator: Locator{File: "a.yaml", Line: 1}},
		{Expression: "x", SourceID: "src-z", Locator: Locator{File: "a.yaml", Line: 1}},
		{Expression: "y", Locator: Locator{File: "a.yaml", Line: 1}},
		{Expression: "p", Locator: Locator{File: "a.yaml", Line: 2, Path: "spec.p"}},
		{Expression: "p", Locator: Locator{File: "a.yaml", Line: 5, Path: "spec.p"}},
		{Expression: "a", Locator: Locator{File: "a.yaml", Line: 9, Path: "spec.a"}},
		{Expression: "b", Locator: Locator{File: "a.yaml", Line: 9, Path: "spec.b"}},
		{Expression: "q", Locator: Locator{File: "a.yaml", Line: 20, Path: "spec.q1"}},
		{Expression: "q", Locator: Locator{File: "a.yaml", Line: 20, Path: "spec.q2"}},
		{Expression: "c", Locator: Locator{File: "a.yaml", Document: 1, Line: 1}},
		{Expression: "d", Locator: Locator{File: "a.yaml", Document: 1, Line: 4}},
		{Expression: "e", Locator: Locator{File: "b.yaml", Line: 1}},
	}

	orders := [][]int{
		{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}, // already sorted
		{11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}, // fully reversed
		{5, 11, 2, 8, 0, 9, 4, 1, 10, 6, 3, 7}, // shuffled
	}

	for i, order := range orders {
		items := make([]Item, len(order))
		for j, idx := range order {
			items[j] = want[idx]
		}

		inv := New("", items)
		inv.Sort()

		assert.Equal(t, want, inv.Items, "input order %d must sort to the same fixed order", i)
	}
}

// TestLocator_IdentityAddressesStructuralPath covers a flow-style sequence: several schedules, one line.
func TestLocator_IdentityAddressesStructuralPath(t *testing.T) {
	t.Run("distinguishes two schedules sharing a line by their path", func(t *testing.T) {
		a := Locator{File: "wf.yaml", Line: 3, Path: "spec.schedules[0]"}.Identity(0, "job-", "line-")
		b := Locator{File: "wf.yaml", Line: 3, Path: "spec.schedules[1]"}.Identity(1, "job-", "line-")

		assert.Equal(t, "line-wf.yaml:3#spec.schedules[0]", a)
		assert.Equal(t, "line-wf.yaml:3#spec.schedules[1]", b)
	})

	t.Run("never folds the item's position into an addressable identity", func(t *testing.T) {
		locator := Locator{File: "wf.yaml", Line: 3, Path: "spec.schedules[1]"}

		first := locator.Identity(1, "job-", "line-")
		later := locator.Identity(7, "job-", "line-")

		assert.Equal(t, first, later, "an addressable locator's identity must not depend on its position")
		assert.NotContains(t, first, "#7")
	})
}

// TestLocator_IdentityWithoutALine covers a schedule addressed structurally rather than by line.
func TestLocator_IdentityWithoutALine(t *testing.T) {
	t.Run("uses the file and path together", func(t *testing.T) {
		id := Locator{File: "wf.yaml", Path: "spec.schedule"}.Identity(0, "job-", "line-")
		assert.Equal(t, "line-wf.yaml#spec.schedule", id)
	})

	t.Run("uses the path alone when that is all there is", func(t *testing.T) {
		id := Locator{Path: "spec.schedule"}.Identity(0, "job-", "line-")
		assert.Equal(t, "line-spec.schedule", id)
	})
}
