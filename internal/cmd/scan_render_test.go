package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testScanWidth is a comfortable, arbitrary width used by tests that are not
// specifically about width fitting.
const testScanWidth = 100

// fixtureInventory builds a hand-authored inventory, with items deliberately
// out of sorted order, so rendering tests pin that renderScanText walks the
// slice forward rather than resorting it.
func fixtureInventory() *inventory.Inventory {
	return inventory.New("/repo", []inventory.Item{
		{
			Expression: "30 3 * * *",
			SourceID:   "k8s",
			Dialect:    "vixie",
			State:      inventory.StateActive,
			Locator:    inventory.Locator{File: "deploy/cronjob.yaml", Line: 6, Path: "spec.schedule"},
		},
		{
			Expression: "0 2 * * *",
			SourceID:   "crontab",
			Dialect:    "vixie",
			Command:    "/usr/bin/backup.sh",
			Shell:      true,
			State:      inventory.StateActive,
			Locator:    inventory.Locator{File: "crontab", Line: 2},
		},
		{
			Expression: "0 1 * * *",
			SourceID:   "argo",
			Dialect:    "vixie",
			State:      inventory.StateActive,
			Locator:    inventory.Locator{File: "workflows/cronworkflow.yaml", Line: 7, Path: "spec.schedules[0]"},
		},
	})
}

func TestRenderScanText_EmptyInventoryPrintsNothingFoundLine(t *testing.T) {
	var buf bytes.Buffer
	renderScanText(&buf, inventory.New("/repo", nil), testScanWidth)

	out := buf.String()
	assert.Equal(t, "No schedules found\n", out, "an empty inventory must render a clear message, not bare headers")
	assert.NotContains(t, out, "LINE", "no table header when there is nothing to show")
}

func TestRenderScanText_EveryItemAppearsWithFileSourceAndDescription(t *testing.T) {
	var buf bytes.Buffer
	renderScanText(&buf, fixtureInventory(), testScanWidth)
	out := buf.String()

	assert.Contains(t, out, "deploy/cronjob.yaml")
	assert.Contains(t, out, "k8s")
	assert.Contains(t, out, "30 3 * * *")
	assert.Contains(t, out, "At 03:30", "the k8s item's schedule must be humanized")

	assert.Contains(t, out, "crontab")
	assert.Contains(t, out, "0 2 * * *")
	assert.Contains(t, out, "At 02:00")

	assert.Contains(t, out, "workflows/cronworkflow.yaml")
	assert.Contains(t, out, "argo")
	assert.Contains(t, out, "0 1 * * *")
	assert.Contains(t, out, "At 01:00")
}

func TestRenderScanText_StructuralPathShownWhenPresentOmittedWhenNot(t *testing.T) {
	var buf bytes.Buffer
	renderScanText(&buf, fixtureInventory(), testScanWidth)
	out := buf.String()

	assert.Contains(t, out, "spec.schedule", "k8s item carries a structural path and it must be shown")
	assert.Contains(t, out, "spec.schedules[0]", "argo item's structural path must be shown")

	// The crontab item has no structural path, so its row must not show a
	// stray path value or repeat the crontab filename in the path column.
	lines := strings.Split(out, "\n")
	var crontabRow string
	for i, l := range lines {
		if l == "crontab" && i+1 < len(lines) {
			crontabRow = lines[i+1]
			break
		}
	}
	require.NotEmpty(t, crontabRow, "expected a data row directly under the crontab file group header")
	assert.NotContains(t, crontabRow, "spec.")
}

func TestRenderScanText_GroupsByFileFollowingInventoryOrder(t *testing.T) {
	var buf bytes.Buffer
	renderScanText(&buf, fixtureInventory(), testScanWidth)
	out := buf.String()

	// fixtureInventory is deliberately not file-sorted; the rendered file
	// headers must appear in that exact order, proving grouping walks the
	// slice forward rather than sorting.
	iDeploy := strings.Index(out, "deploy/cronjob.yaml")
	iCrontab := strings.Index(out, "\ncrontab\n")
	iWorkflows := strings.Index(out, "workflows/cronworkflow.yaml")
	require.NotEqual(t, -1, iDeploy)
	require.NotEqual(t, -1, iCrontab)
	require.NotEqual(t, -1, iWorkflows)
	assert.True(t, iDeploy < iCrontab && iCrontab < iWorkflows,
		"file groups must appear in the inventory's own order, not sorted or map-iterated")
}

func TestRenderScanText_NonActiveItemsAreDistinguishedWithReason(t *testing.T) {
	inv := inventory.New("/repo", []inventory.Item{
		{
			Expression: "*/15 * * * *",
			SourceID:   "k8s",
			State:      inventory.StateSuspended,
			Reason:     "the schedule is suspended",
			Locator:    inventory.Locator{File: "suspended.yaml", Line: 6, Path: "spec.schedule"},
		},
		{
			Expression: "{{ .Values.backup.cron }}",
			SourceID:   "k8s",
			State:      inventory.StateUnresolved,
			Reason:     "the expression is templated and cannot be read",
			Locator:    inventory.Locator{File: "templated.yaml", Line: 6, Path: "spec.schedule"},
		},
		{
			Expression: "99 99 * * *",
			SourceID:   "k8s",
			State:      inventory.StateInvalid,
			Reason:     "value out of range: end of range (99) above maximum (59): 99",
			Locator:    inventory.Locator{File: "broken.yaml", Line: 6, Path: "spec.schedule"},
		},
		{
			Expression: "0 2 * * *",
			SourceID:   "crontab",
			State:      inventory.StateActive,
			Locator:    inventory.Locator{File: "crontab", Line: 1},
		},
	})

	// Rendered wide enough that reason texts aren't truncated; width-fitting
	// is covered separately by TestRenderScanText_FitsResolvedWidth.
	const wide = 160

	var buf bytes.Buffer
	renderScanText(&buf, inv, wide)
	out := buf.String()

	assert.Contains(t, out, "suspended")
	assert.Contains(t, out, "the schedule is suspended")
	assert.Contains(t, out, "unresolved")
	assert.Contains(t, out, "the expression is templated and cannot be read")
	assert.Contains(t, out, "invalid")
	assert.Contains(t, out, "value out of range")

	// The suspended item's row must not read as a live humanized schedule
	// (e.g. "every 15 minutes"), which would contradict its state.
	assert.NotContains(t, out, "every 15 minutes")

	assert.Contains(t, out, "1 suspended, 1 unresolved, 1 invalid")
}

func TestRenderScanText_UnparseableActiveItemDoesNotCrashOrBlank(t *testing.T) {
	inv := inventory.New("/repo", []inventory.Item{
		{
			Expression: "garbage",
			SourceID:   "crontab",
			State:      inventory.StateActive,
			Locator:    inventory.Locator{File: "crontab", Line: 1},
		},
	})

	var buf bytes.Buffer
	require.NotPanics(t, func() { renderScanText(&buf, inv, testScanWidth) })

	out := buf.String()
	assert.Contains(t, out, "garbage")
	// An active item whose expression fails to parse must not crash or leave
	// the description blank; it falls back to naming the parse error.
	assert.Contains(t, out, "(unparseable:", "an active item that fails to parse must explain why, not print a blank cell")
}

// TestRenderScanText_NonActiveItemWithNoReasonFallsBackToState guards
// describeItem's defensive branch: the renderer must not blank out or panic
// if a non-active item lacks a Reason.
func TestRenderScanText_NonActiveItemWithNoReasonFallsBackToState(t *testing.T) {
	inv := inventory.New("/repo", []inventory.Item{
		{
			Expression: "*/15 * * * *",
			SourceID:   "k8s",
			State:      inventory.StateSuspended,
			Locator:    inventory.Locator{File: "suspended.yaml", Line: 6},
		},
	})

	var buf bytes.Buffer
	renderScanText(&buf, inv, testScanWidth)
	out := buf.String()

	assert.Contains(t, out, "(suspended)")
}

func TestRenderScanText_SummaryCountsMatchInventory(t *testing.T) {
	var buf bytes.Buffer
	renderScanText(&buf, fixtureInventory(), testScanWidth)
	out := buf.String()

	assert.Contains(t, out, "3 schedule(s) across 3 file(s) from 3 source(s)")
	assert.Contains(t, out, "k8s")
	assert.Contains(t, out, "crontab")
	assert.Contains(t, out, "argo")
	assert.Contains(t, out, "0 suspended, 0 unresolved, 0 invalid")
}

func TestRenderScanText_SameInventoryTwiceIsByteIdentical(t *testing.T) {
	inv := fixtureInventory()

	var first, second bytes.Buffer
	renderScanText(&first, inv, testScanWidth)
	renderScanText(&second, inv, testScanWidth)

	assert.Equal(t, first.Bytes(), second.Bytes(), "rendering the same inventory twice must be byte-identical")
}

func TestRenderScanText_RowOrderFollowsInventoryOrderNotSorted(t *testing.T) {
	// Same three items as fixtureInventory but shuffled into a different
	// order; rendered file headers must track the caller's order, proving
	// there is no internal sort.
	inv := inventory.New("/repo", []inventory.Item{
		{Expression: "0 1 * * *", SourceID: "argo", State: inventory.StateActive,
			Locator: inventory.Locator{File: "workflows/cronworkflow.yaml", Line: 7, Path: "spec.schedules[0]"}},
		{Expression: "30 3 * * *", SourceID: "k8s", State: inventory.StateActive,
			Locator: inventory.Locator{File: "deploy/cronjob.yaml", Line: 6, Path: "spec.schedule"}},
		{Expression: "0 2 * * *", SourceID: "crontab", State: inventory.StateActive,
			Locator: inventory.Locator{File: "crontab", Line: 2}},
	})

	var buf bytes.Buffer
	renderScanText(&buf, inv, testScanWidth)
	out := buf.String()

	iWorkflows := strings.Index(out, "workflows/cronworkflow.yaml")
	iDeploy := strings.Index(out, "deploy/cronjob.yaml")
	iCrontab := strings.Index(out, "\ncrontab\n")
	require.NotEqual(t, -1, iWorkflows)
	require.NotEqual(t, -1, iDeploy)
	require.NotEqual(t, -1, iCrontab)
	assert.True(t, iWorkflows < iDeploy && iDeploy < iCrontab,
		"a differently-ordered input inventory must render in that same order")
}

func TestRenderScanText_LongLocatorsStayAligned(t *testing.T) {
	inv := inventory.New("/repo", []inventory.Item{
		{
			Expression: "0 0 1,15 * 1-5",
			SourceID:   "k8s",
			State:      inventory.StateActive,
			Locator: inventory.Locator{
				File: "deploy/very/deeply/nested/path/to/a/manifest.yaml",
				Line: 42,
				Path: "spec.jobTemplate.spec.template.spec.containers[0].command",
			},
		},
		{
			Expression: "0 2 * * *",
			SourceID:   "crontab",
			State:      inventory.StateActive,
			Locator:    inventory.Locator{File: "crontab", Line: 2},
		},
	})

	var buf bytes.Buffer
	renderScanText(&buf, inv, testScanWidth)
	out := buf.String()

	assert.Contains(t, out, "...", "the overlong structural path must be truncated, not left to blow out the column")

	lines := strings.Split(out, "\n")
	var longRow, shortRow string
	for i, l := range lines {
		if strings.HasPrefix(l, "42") {
			longRow = l
		}
		if l == "crontab" && i+1 < len(lines) {
			shortRow = lines[i+1]
		}
	}
	require.NotEmpty(t, longRow)
	require.NotEmpty(t, shortRow)

	// Both items are active (no STATE column); the fixed-shape columns must
	// place DESCRIPTION at the same offset in both rows regardless of the
	// long row's overlong path.
	layout := newScanLayout(testScanWidth, false, true)
	fixedWidth := scanColLine + 2 + layout.pathW + 2 + scanColSource + 2 + scanColExpr + 2
	require.True(t, len([]rune(longRow)) >= fixedWidth)
	require.True(t, len([]rune(shortRow)) >= fixedWidth)
	assert.Equal(t, string([]rune(longRow)[fixedWidth-2:fixedWidth]), string([]rune(shortRow)[fixedWidth-2:fixedWidth]),
		"the two-space column gutter right before DESCRIPTION must land at the same offset in both rows")
}

// isScanFileGroupHeadingLine reports whether l is a bare file-group heading
// line (or its blank separator), deliberately exempt from the "fits the
// requested width" property since truncating a file path would make it
// unusable.
func isScanFileGroupHeadingLine(l string, files []string) bool {
	if l == "" {
		return true
	}
	for _, f := range files {
		if l == f {
			return true
		}
	}
	return false
}

// TestRenderScanText_FitsResolvedWidthExceptFileHeadings pins that every
// table row, separator, header, and summary line fits the requested width
// (file-group headings excluded), checked against the literal requested
// width across both all-active and suspended fixtures at widths 40 through 200.
func TestRenderScanText_FitsResolvedWidthExceptFileHeadings(t *testing.T) {
	longPath := "spec.jobTemplate.spec.template.spec.containers[0].command"
	longReason := "value out of range: end of range (99) above maximum (59) in a needlessly verbose diagnostic message that keeps going"
	files := []string{"deploy/very/deeply/nested/path/manifest.yaml", "deploy/broken.yaml", "crontab"}

	fixtures := []struct {
		name string
		inv  *inventory.Inventory
	}{
		{"all-active", inventory.New("/repo", []inventory.Item{
			{
				Expression: "0 0 1,15 * 1-5",
				SourceID:   "k8s",
				State:      inventory.StateActive,
				Locator:    inventory.Locator{File: files[0], Line: 42, Path: longPath},
			},
			{
				Expression: "0 2 * * *",
				SourceID:   "crontab",
				State:      inventory.StateActive,
				Locator:    inventory.Locator{File: files[2], Line: 2},
			},
		})},
		{"with-suspended", inventory.New("/repo", []inventory.Item{
			{
				Expression: "0 0 1,15 * 1-5",
				SourceID:   "k8s",
				State:      inventory.StateActive,
				Locator:    inventory.Locator{File: files[0], Line: 42, Path: longPath},
			},
			{
				Expression: "99 99 * * *",
				SourceID:   "k8s",
				State:      inventory.StateInvalid,
				Reason:     longReason,
				Locator:    inventory.Locator{File: files[1], Line: 6, Path: "spec.schedule"},
			},
			{
				Expression: "0 2 * * *",
				SourceID:   "crontab",
				State:      inventory.StateActive,
				Locator:    inventory.Locator{File: files[2], Line: 2},
			},
		})},
	}

	for _, fx := range fixtures {
		name, inv := fx.name, fx.inv
		t.Run(name, func(t *testing.T) {
			for _, width := range []int{40, 60, 72, 80, 100, 200} {
				t.Run(fmt.Sprintf("width=%d", width), func(t *testing.T) {
					var buf bytes.Buffer
					renderScanText(&buf, inv, width)
					out := buf.String()

					for i, l := range strings.Split(out, "\n") {
						if isScanFileGroupHeadingLine(l, files) {
							continue // deliberately exempt; see isScanFileGroupHeadingLine
						}
						assert.LessOrEqualf(t, len([]rune(l)), width,
							"line %d exceeds the requested width %d: %q", i, width, l)
					}
				})
			}
		})
	}
}

// TestRenderScanText_StateColumnOnlyWhenNonActive pins that STATE is hidden
// when the whole inventory is active and shown once any item is not.
func TestRenderScanText_StateColumnOnlyWhenNonActive(t *testing.T) {
	allActive := inventory.New("/repo", []inventory.Item{
		{Expression: "0 2 * * *", SourceID: "crontab", State: inventory.StateActive,
			Locator: inventory.Locator{File: "crontab", Line: 1}},
		{Expression: "30 3 * * *", SourceID: "k8s", State: inventory.StateActive,
			Locator: inventory.Locator{File: "deploy/cronjob.yaml", Line: 6, Path: "spec.schedule"}},
	})

	var buf bytes.Buffer
	renderScanText(&buf, allActive, testScanWidth)
	out := buf.String()

	assert.NotContains(t, out, "STATE", "an all-active inventory must not spend a column saying so on every row")
	assert.Contains(t, out, "0 suspended, 0 unresolved, 0 invalid", "the summary still reports the counts even with the column hidden")

	withSuspended := inventory.New("/repo", []inventory.Item{
		{Expression: "0 2 * * *", SourceID: "crontab", State: inventory.StateActive,
			Locator: inventory.Locator{File: "crontab", Line: 1}},
		{Expression: "*/15 * * * *", SourceID: "k8s", State: inventory.StateSuspended,
			Reason:  "the schedule is suspended",
			Locator: inventory.Locator{File: "deploy/cronjob.yaml", Line: 6, Path: "spec.schedule"}},
	})

	buf.Reset()
	renderScanText(&buf, withSuspended, testScanWidth)
	out = buf.String()

	assert.Contains(t, out, "STATE", "at least one non-active item must earn the column back")
	assert.Contains(t, out, "suspended")
	assert.Contains(t, out, "1 suspended, 0 unresolved, 0 invalid")
}

// TestRenderScanText_PathColumnOnlyWhenAnyItemHasOne pins that PATH is
// hidden when no item carries a locator path, even at a width generous
// enough for it to fit (previously it rendered a blank column, following
// STATE's already-correct rule).
func TestRenderScanText_PathColumnOnlyWhenAnyItemHasOne(t *testing.T) {
	noPaths := inventory.New("/repo", []inventory.Item{
		{Expression: "0 2 * * *", SourceID: "crontab", State: inventory.StateActive,
			Command: "/usr/bin/backup.sh", Locator: inventory.Locator{File: "crontab", Line: 1}},
		{Expression: "*/15 * * * *", SourceID: "crontab", State: inventory.StateActive,
			Command: "/usr/bin/poll.sh", Locator: inventory.Locator{File: "crontab", Line: 2}},
	})

	var buf bytes.Buffer
	renderScanText(&buf, noPaths, testScanWidth)
	out := buf.String()

	assert.NotContains(t, out, "PATH", "no item carries a locator path, so the column must not be shown at all")

	layout := newScanLayout(testScanWidth, false, false)
	assert.False(t, layout.showPath)

	withOnePath := inventory.New("/repo", []inventory.Item{
		{Expression: "0 2 * * *", SourceID: "crontab", State: inventory.StateActive,
			Locator: inventory.Locator{File: "crontab", Line: 1}},
		{Expression: "30 3 * * *", SourceID: "k8s", State: inventory.StateActive,
			Locator: inventory.Locator{File: "deploy/cronjob.yaml", Line: 6, Path: "spec.schedule"}},
	})

	buf.Reset()
	renderScanText(&buf, withOnePath, testScanWidth)
	out = buf.String()

	assert.Contains(t, out, "PATH", "at least one item carries a locator path, so the column must earn its place back")
	assert.Contains(t, out, "spec.schedule")
}

// TestNewScanLayout_ShedsOptionalColumnsAsWidthShrinks pins the
// column-shedding order: SOURCE first, then PATH, while LINE, EXPRESSION,
// DESCRIPTION, and STATE never leave; thresholds are derived independently
// rather than read back off a layout.
func TestNewScanLayout_ShedsOptionalColumnsAsWidthShrinks(t *testing.T) {
	for _, showState := range []bool{false, true} {
		extra := 0 // STATE's own width plus its gutter, when shown
		if showState {
			extra = scanColState + 2
		}

		bothMin := scanColLine + scanColSource + scanColExpr + extra + 4*2 + scanMinPathWidth + scanMinDescWidth
		sourceOnlyMin := scanColLine + scanColExpr + extra + 3*2 + scanMinPathWidth + scanMinDescWidth
		neitherMin := scanColLine + scanColExpr + extra + 2*2 + scanMinDescWidth

		wide := newScanLayout(bothMin, showState, true)
		assert.Truef(t, wide.showSource, "showState=%v width=%d: SOURCE must still be shown at its own minimum width", showState, bothMin)
		assert.Truef(t, wide.showPath, "showState=%v width=%d: PATH must still be shown at its own minimum width", showState, bothMin)

		noSource := newScanLayout(sourceOnlyMin, showState, true)
		assert.Falsef(t, noSource.showSource, "showState=%v width=%d: SOURCE must be the first column shed", showState, sourceOnlyMin)
		assert.Truef(t, noSource.showPath, "showState=%v width=%d: PATH must survive once SOURCE alone has been shed", showState, sourceOnlyMin)

		noSourceOrPath := newScanLayout(neitherMin, showState, true)
		assert.Falsef(t, noSourceOrPath.showSource, "showState=%v width=%d: SOURCE must stay shed", showState, neitherMin)
		assert.Falsef(t, noSourceOrPath.showPath, "showState=%v width=%d: PATH must be shed once SOURCE alone is not enough", showState, neitherMin)
		assert.GreaterOrEqualf(t, noSourceOrPath.descW, scanMinDescWidth, "showState=%v", showState)
	}
}

// TestNewScanLayout_DegradesDescriptionRatherThanOverflow covers the last
// resort: EXPRESSION shrinks to scanMinExprWidth before DESCRIPTION gives up
// any room, and the layout must never report a total wider than the
// requested width down to that hard floor.
func TestNewScanLayout_DegradesDescriptionRatherThanOverflow(t *testing.T) {
	for _, showState := range []bool{false, true} {
		fixed, gutters := scanColumnCost(showState, scanColumnPlan{})
		hardFloor := fixed - scanColExpr + scanMinExprWidth + gutters

		for _, width := range []int{hardFloor, hardFloor + 5, hardFloor + 14} {
			layout := newScanLayout(width, showState, true)
			assert.Falsef(t, layout.showSource, "showState=%v width=%d", showState, width)
			assert.Falsef(t, layout.showPath, "showState=%v width=%d", showState, width)
			assert.GreaterOrEqualf(t, layout.descW, 0, "showState=%v width=%d: descW must never go negative", showState, width)
			assert.GreaterOrEqualf(t, layout.exprW, 0, "showState=%v width=%d: exprW must never go negative", showState, width)
			assert.LessOrEqualf(t, fixed-scanColExpr+gutters+layout.exprW+layout.descW, width,
				"showState=%v width=%d: the resulting row must not exceed the requested width", showState, width)
		}

		below := newScanLayout(hardFloor-1, showState, true)
		assert.False(t, below.showSource, "showState=%v", showState)
		assert.False(t, below.showPath, "showState=%v", showState)
		assert.Equal(t, 0, below.descW, "showState=%v: DESCRIPTION shrinks to nothing rather than going negative", showState)
	}
}

// TestTruncateField covers both of truncateField's edges: normal ellipsis
// truncation, and the max<=3 case where there's no room for "..." so the cut is hard.
func TestTruncateField(t *testing.T) {
	assert.Equal(t, "hello", truncateField("hello", 10), "a string that already fits is returned unchanged")
	assert.Equal(t, "he...", truncateField("hello world", 5), "a longer string is cut with a trailing ellipsis")
	assert.Equal(t, "he", truncateField("hello", 2), "too little room for an ellipsis falls back to a hard cut")
}

// TestTruncateField_CountsAndCutsByRunesNotBytes guards against slicing a
// multi-byte string by byte offset, which would both produce invalid UTF-8
// and misalign %-*s padding.
func TestTruncateField_CountsAndCutsByRunesNotBytes(t *testing.T) {
	s := "Расписание резервного копирования" // "Backup schedule", every rune 2 bytes in UTF-8

	got := truncateField(s, 10)
	assert.LessOrEqual(t, len([]rune(got)), 10, "the cut must land on a rune boundary, at or under the requested width")
	assert.True(t, utf8.ValidString(got), "truncating mid-character must never produce invalid UTF-8")
	assert.True(t, strings.HasSuffix(got, "..."), "still cut with the usual ellipsis once it no longer fits")

	gotLeft := truncateFieldLeft(s, 10)
	assert.LessOrEqual(t, len([]rune(gotLeft)), 10)
	assert.True(t, utf8.ValidString(gotLeft), "truncating mid-character must never produce invalid UTF-8")
	assert.True(t, strings.HasPrefix(gotLeft, "..."))

	// A string already short enough is returned as-is, not truncated by a
	// byte-length check that overcounts multi-byte runes.
	assert.Equal(t, "Привет", truncateField("Привет", 10))
	assert.Equal(t, "Привет", truncateFieldLeft("Привет", 10))
}

// TestTruncateFieldLeft mirrors TestTruncateField: the ellipsis goes on the
// front, and the tail (where a structural path's distinguishing suffix lives) survives.
func TestTruncateFieldLeft(t *testing.T) {
	assert.Equal(t, "hello", truncateFieldLeft("hello", 10), "a string that already fits is returned unchanged")
	assert.Equal(t, "...world", truncateFieldLeft("hello world", 8), "a longer string is cut with a leading ellipsis, keeping the tail")
	assert.Equal(t, "lo", truncateFieldLeft("hello", 2), "too little room for an ellipsis falls back to a hard cut from the front")

	// The retained tail starts right after a ".", so it must not run into
	// the "..." prefix and read as four dots.
	got := truncateFieldLeft("spec.schedules[0]", 16)
	assert.Equal(t, "...schedules[0]", got)
	assert.NotContains(t, got, "....", "a leading dot in the retained tail must be trimmed, not doubled up with the ellipsis")
}

// TestRenderScanText_DescriptionColumnSurvivesAtWidth40 pins that
// DESCRIPTION always keeps some width (previously it could compute to 0
// with STATE shown, leaving trailing whitespace), checked with and without
// a non-active item.
func TestRenderScanText_DescriptionColumnSurvivesAtWidth40(t *testing.T) {
	const width = 40

	assertNoTrailingWhitespace := func(t *testing.T, out string) {
		t.Helper()
		for i, l := range strings.Split(out, "\n") {
			assert.Equalf(t, strings.TrimRight(l, " "), l, "line %d must not carry trailing whitespace: %q", i, l)
		}
	}

	t.Run("with a non-active item", func(t *testing.T) {
		inv := inventory.New("/repo", []inventory.Item{
			{Expression: "0 2 * * *", SourceID: "crontab", State: inventory.StateActive,
				Locator: inventory.Locator{File: "crontab", Line: 1}},
			{Expression: "*/15 * * * *", SourceID: "k8s", State: inventory.StateSuspended,
				Reason:  "the schedule is suspended",
				Locator: inventory.Locator{File: "deploy/cronjob.yaml", Line: 6, Path: "spec.schedule"}},
		})

		layout := newScanLayout(width, true, true)
		require.Greater(t, layout.descW, 0, "sanity check: this is exactly the width/STATE combination that used to force descW to 0")
		require.Less(t, layout.exprW, scanColExpr, "EXPRESSION must actually have given up room for this test to exercise the fix")

		var buf bytes.Buffer
		renderScanText(&buf, inv, width)
		out := buf.String()

		assert.Contains(t, out, "STATE", "sanity check: this fixture must actually earn the STATE column")
		// descW is narrow here, so the header and reason are truncated; what
		// matters is that a nonempty description cell exists at the layout's
		// chosen width.
		assert.Contains(t, out, truncateField("DESCRIPTION", layout.descW), "the header must still carry a description column")
		assert.Contains(t, out, truncateField("the schedule is suspended", layout.descW), "the suspended item's reason must still be visible, even if truncated")
		assertNoTrailingWhitespace(t, out)
	})

	t.Run("without a non-active item", func(t *testing.T) {
		inv := inventory.New("/repo", []inventory.Item{
			{Expression: "0 2 * * *", SourceID: "crontab", State: inventory.StateActive,
				Locator: inventory.Locator{File: "crontab", Line: 1}},
		})

		layout := newScanLayout(width, false, false)
		require.Greater(t, layout.descW, 0)
		assert.Equal(t, scanColExpr, layout.exprW, "EXPRESSION had no need to shrink here; only the STATE-shown case forces it to")

		var buf bytes.Buffer
		renderScanText(&buf, inv, width)
		out := buf.String()

		assert.NotContains(t, out, "STATE")
		assert.Contains(t, out, "At 02:00", "the active item's description must still be shown")
		assertNoTrailingWhitespace(t, out)
	})
}

// TestRenderScanText_LeftTruncatedPathsStayDistinguishable pins that PATH
// truncates from the left, since two paths differing only in their last
// character (e.g. "spec.schedules[0]" vs "[1]") would otherwise collapse to
// identical rows.
func TestRenderScanText_LeftTruncatedPathsStayDistinguishable(t *testing.T) {
	inv := inventory.New("/repo", []inventory.Item{
		{
			Expression: "0 1 * * *",
			SourceID:   "argo",
			State:      inventory.StateActive,
			Locator:    inventory.Locator{File: "workflows/cronworkflow.yaml", Line: 7, Path: "spec.schedules[0]"},
		},
		{
			Expression: "0 13 * * *",
			SourceID:   "argo",
			State:      inventory.StateActive,
			Locator:    inventory.Locator{File: "workflows/cronworkflow.yaml", Line: 8, Path: "spec.schedules[1]"},
		},
	})

	// A width narrow enough that the 18-rune path gets truncated (all-active,
	// so no STATE column, and PATH's share is under 18).
	const narrow = 70

	var buf bytes.Buffer
	renderScanText(&buf, inv, narrow)
	out := buf.String()

	var row0, row1 string
	for _, l := range strings.Split(out, "\n") {
		fields := strings.Fields(l)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "7":
			row0 = l
		case "8":
			row1 = l
		}
	}
	require.NotEmpty(t, row0, "expected to find the line-7 data row")
	require.NotEmpty(t, row1, "expected to find the line-8 data row")

	assert.Contains(t, out, "...", "the overlong path must actually be truncated in this test, or it isn't exercising the bug")
	assert.NotEqual(t, row0, row1, "two distinct schedules must not render as identical rows once their paths are truncated")

	// Truncating from the left keeps the tail, so the distinguishing final
	// character ("[0]" vs "[1]") must still be visible.
	assert.Contains(t, row0, "[0]")
	assert.Contains(t, row1, "[1]")

	// "spec.schedules[N]" is cut right after a ".", so a naive "..." + tail
	// concatenation would read as four dots; no rendered path may do that.
	assert.NotContains(t, out, "....", "a truncated path must never run its ellipsis into a leading dot in the tail")
}

// TestRenderScanText_NonASCIIContentStaysValidUTF8AndAligned pins that
// non-ASCII text (Cyrillic here) truncates on rune boundaries, not byte
// offsets, which would otherwise corrupt UTF-8 and misalign %-*s padding.
func TestRenderScanText_NonASCIIContentStaysValidUTF8AndAligned(t *testing.T) {
	inv := inventory.New("/repo", []inventory.Item{
		{
			Expression: "0 2 * * *",
			SourceID:   "k8s",
			State:      inventory.StateInvalid,
			Reason:     "Расписание резервного копирования базы данных повреждено",
			Locator:    inventory.Locator{File: "deploy/cronjob.yaml", Line: 6, Path: "spec.параметры.расписание"},
		},
		{
			Expression: "0 3 * * *",
			SourceID:   "crontab",
			State:      inventory.StateActive,
			Locator:    inventory.Locator{File: "crontab", Line: 2},
		},
	})

	const narrow = 70
	var buf bytes.Buffer
	renderScanText(&buf, inv, narrow)
	out := buf.String()

	require.True(t, utf8.ValidString(out), "truncating non-ASCII content must never emit invalid UTF-8")
	assert.Contains(t, out, "...", "sanity check: this fixture must actually force truncation to exercise the bug")

	// Every row must still line up: the gutter before DESCRIPTION lands at
	// the same rune offset on both the Cyrillic and plain-ASCII rows.
	layout := newScanLayout(narrow, true, true)
	require.False(t, layout.showSource, "sanity check: SOURCE must be shed at this width so the offset below only counts PATH, STATE and EXPRESSION")
	fixedWidth := scanColLine + 2 + layout.pathW + 2 + scanColState + 2 + scanColExpr + 2

	// Located by file-group heading, not by leading LINE number, since the
	// crontab item's line "2" collides with the summary's "2 schedule(s)...".
	var cyrillicRow, plainRow string
	lines := strings.Split(out, "\n")
	for i, l := range lines {
		if l == "deploy/cronjob.yaml" && i+1 < len(lines) {
			cyrillicRow = lines[i+1]
		}
		if l == "crontab" && i+1 < len(lines) {
			plainRow = lines[i+1]
		}
	}
	require.NotEmpty(t, cyrillicRow, "expected to find the Cyrillic item's data row")
	require.NotEmpty(t, plainRow, "expected to find the plain-ASCII item's data row")

	cyrillicRunes := []rune(cyrillicRow)
	plainRunes := []rune(plainRow)
	require.GreaterOrEqual(t, len(cyrillicRunes), fixedWidth)
	require.GreaterOrEqual(t, len(plainRunes), fixedWidth)
	assert.Equal(t, string(plainRunes[fixedWidth-2:fixedWidth]), string(cyrillicRunes[fixedWidth-2:fixedWidth]),
		"the gutter right before DESCRIPTION must land at the same rune offset regardless of multi-byte content earlier in the row")
}

// TestRenderScanText_HonoursTheLocaleFlag pins that scan resolves
// cronx.NewParserWithLocale(GetLocale()) like list/explain/next/timeline/check,
// using a throwaway locale that maps "MON" to a different weekday (3,
// Wednesday) than English (1, Monday) to make the difference observable.
func TestRenderScanText_HonoursTheLocaleFlag(t *testing.T) {
	const testLocale = "zz-test"
	cronx.SymbolRegistryMap[testLocale] = cronx.NewSymbolRegistry(testLocale,
		map[string]int{"SUN": 0, "MON": 3, "TUE": 2, "WED": 1, "THU": 4, "FRI": 5, "SAT": 6},
		map[string]int{"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4, "MAY": 5, "JUN": 6,
			"JUL": 7, "AUG": 8, "SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12},
	)
	t.Cleanup(func() { delete(cronx.SymbolRegistryMap, testLocale) })

	origLocale := locale
	locale = testLocale
	t.Cleanup(func() { locale = origLocale })

	inv := inventory.New("/repo", []inventory.Item{
		{
			Expression: "0 9 * * MON",
			SourceID:   "crontab",
			State:      inventory.StateActive,
			Locator:    inventory.Locator{File: "crontab", Line: 1},
		},
	})

	var buf bytes.Buffer
	renderScanText(&buf, inv, testScanWidth)
	out := buf.String()

	assert.Contains(t, out, "Wednesday",
		"scan's own parser must resolve \"MON\" through the active locale's registry (3, Wednesday here), not silently default to English")
	assert.NotContains(t, out, "Monday",
		"the English reading of \"MON\" must not win once a different locale is active")
}

// TestSplitExprAndDesc_NegativeRoomClampsToZero covers the defensive guard:
// with no room left for EXPRESSION or DESCRIPTION, both must come back
// zero, never negative.
func TestSplitExprAndDesc_NegativeRoomClampsToZero(t *testing.T) {
	exprW, descW := splitExprAndDesc(-5)
	assert.Equal(t, 0, exprW)
	assert.Equal(t, 0, descW)
}

// TestRenderScanText_ExtremelyNarrowWidthDropsDescriptionEntirely exercises
// row's descW==0 branch: EXPRESSION renders as the last column with no
// trailing gutter or whitespace, below the width where DESCRIPTION has any
// room at all.
func TestRenderScanText_ExtremelyNarrowWidthDropsDescriptionEntirely(t *testing.T) {
	inv := inventory.New("/repo", []inventory.Item{
		{Expression: "0 2 * * *", SourceID: "crontab", State: inventory.StateActive,
			Locator: inventory.Locator{File: "crontab", Line: 1}},
	})

	const extremelyNarrow = 15
	layout := newScanLayout(extremelyNarrow, false, false)
	require.Equal(t, 0, layout.descW, "sanity check: this width must actually be below the true hard floor")

	var buf bytes.Buffer
	renderScanText(&buf, inv, extremelyNarrow)
	out := buf.String()

	assert.NotContains(t, out, "DESCRIPTION")
	for i, l := range strings.Split(out, "\n") {
		assert.Equalf(t, strings.TrimRight(l, " "), l, "line %d must not carry trailing whitespace: %q", i, l)
	}
}

// TestScanSummary_RenderWrapsInsteadOfOverflowing pins that the summary
// fits on one line when there's room and otherwise wraps at its natural
// clause boundary, never dropping a count.
func TestScanSummary_RenderWrapsInsteadOfOverflowing(t *testing.T) {
	s := newScanSummary()
	for _, item := range []inventory.Item{
		{SourceID: "gha", State: inventory.StateActive, Locator: inventory.Locator{File: "a"}},
		{SourceID: "crontab", State: inventory.StateActive, Locator: inventory.Locator{File: "b"}},
		{SourceID: "k8s", State: inventory.StateSuspended, Locator: inventory.Locator{File: "c"}},
		{SourceID: "argo", State: inventory.StateInvalid, Locator: inventory.Locator{File: "d"}},
	} {
		s.add(item)
	}

	full := s.String()
	require.Greater(t, len(full), 80, "sanity check: this fixture's summary must actually be too long for an 80-column render to matter")

	// Wide enough that the whole sentence fits on one line.
	wide := s.render(200)
	require.Len(t, wide, 1)
	assert.Equal(t, full, wide[0])

	// Narrow: must split rather than overflow, and every count must still
	// be present somewhere in the output.
	narrow := s.render(80)
	assert.Greater(t, len(narrow), 1, "a summary that does not fit on one line must wrap, not overflow it")
	for _, l := range narrow {
		assert.LessOrEqualf(t, len(l), 80, "wrapped summary line exceeds the width it was wrapped to: %q", l)
	}

	joined := strings.Join(narrow, " ")
	assert.Contains(t, joined, "4 schedule(s)")
	assert.Contains(t, joined, "4 file(s)")
	assert.Contains(t, joined, "4 source(s)")
	assert.Contains(t, joined, "gha")
	assert.Contains(t, joined, "crontab")
	assert.Contains(t, joined, "k8s")
	assert.Contains(t, joined, "argo")
	assert.Contains(t, joined, "1 suspended")
	assert.Contains(t, joined, "0 unresolved")
	assert.Contains(t, joined, "1 invalid")
}

// TestWrapText exercises wrapText directly: non-positive width disables
// wrapping, empty string is unchanged, ordinary text wraps at word
// boundaries, and a word longer than width is kept whole rather than split.
func TestWrapText(t *testing.T) {
	assert.Equal(t, []string{"hello"}, wrapText("hello", 0), "a non-positive width disables wrapping")
	assert.Equal(t, []string{""}, wrapText("", 10), "no words at all returns the (empty) input unwrapped")
	assert.Equal(t, []string{"a", "bb", "ccc"}, wrapText("a bb ccc", 2), "ordinary text wraps at word boundaries")
	assert.Equal(t, []string{"hi", "extraordinarily"}, wrapText("hi extraordinarily", 5),
		"a word longer than width is never split, only placed on its own line")
}

// TestScanSummary_RenderZeroOrNegativeWidthDisablesWrapping guards render's
// defensive branch: a non-positive width returns the plain unwrapped
// summary rather than panicking.
func TestScanSummary_RenderZeroOrNegativeWidthDisablesWrapping(t *testing.T) {
	s := newScanSummary()
	s.add(inventory.Item{SourceID: "crontab", State: inventory.StateActive, Locator: inventory.Locator{File: "a"}})

	assert.Equal(t, []string{s.String()}, s.render(0))
	assert.Equal(t, []string{s.String()}, s.render(-5))
}
