package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decodeOne(t *testing.T, src string) *Node {
	t.Helper()
	docs, err := DecodeYAML([]byte(src))
	require.NoError(t, err)
	require.Len(t, docs, 1)
	return docs[0].Root
}

func values(located []Located) []string {
	out := make([]string, 0, len(located))
	for _, l := range located {
		out = append(out, l.Node.Value)
	}
	return out
}

func TestPath_DottedKeys(t *testing.T) {
	root := decodeOne(t, "spec:\n  schedule: \"0 2 * * *\"\n")

	assert.Equal(t, []string{"0 2 * * *"}, values(Path("spec.schedule").Resolve(root)))
}

func TestPath_EveryElement(t *testing.T) {
	root := decodeOne(t, `
on:
  schedule:
    - cron: '30 5 * * 1'
    - cron: '0 0 * * *'
`)

	assert.Equal(t, []string{"30 5 * * 1", "0 0 * * *"},
		values(Path("on.schedule[].cron").Resolve(root)))
}

func TestPath_Index(t *testing.T) {
	root := decodeOne(t, "spec:\n  schedules:\n    - \"0 2 * * *\"\n    - \"0 3 * * *\"\n")

	assert.Equal(t, []string{"0 3 * * *"}, values(Path("spec.schedules[1]").Resolve(root)))
	assert.Empty(t, Path("spec.schedules[9]").Resolve(root), "an out-of-range index yields nothing")
}

func TestPath_BareSequence(t *testing.T) {
	root := decodeOne(t, "spec:\n  schedules:\n    - \"0 2 * * *\"\n    - \"0 3 * * *\"\n")

	assert.Equal(t, []string{"0 2 * * *", "0 3 * * *"},
		values(Path("spec.schedules[]").Resolve(root)))
}

func TestPath_MissingAndMismatched(t *testing.T) {
	root := decodeOne(t, "spec:\n  schedule: \"0 2 * * *\"\n")

	assert.Empty(t, Path("spec.nothing").Resolve(root), "a missing key yields nothing")
	assert.Empty(t, Path("spec.schedule.deeper").Resolve(root), "descending into a scalar yields nothing")
	assert.Empty(t, Path("spec.schedule[]").Resolve(root), "indexing a scalar yields nothing")
	assert.Empty(t, Path("spec.schedule").Resolve(nil), "a nil root yields nothing")
}

func TestPath_Empty(t *testing.T) {
	root := decodeOne(t, "spec:\n  schedule: \"0 2 * * *\"\n")

	got := Path("").Resolve(root)
	require.Len(t, got, 1)
	assert.Same(t, root, got[0].Node, "an empty path is the root itself")
}

func TestPath_NonNumericSelector(t *testing.T) {
	root := decodeOne(t, "spec:\n  schedules:\n    - \"0 2 * * *\"\n    - \"0 3 * * *\"\n")

	assert.Empty(t, Path("spec.schedules[abc]").Resolve(root),
		"a non-numeric selector hits strconv.Atoi's error branch and selects nothing")
}

func TestPath_MalformedSelector(t *testing.T) {
	root := decodeOne(t, "spec:\n  schedules:\n    - \"0 2 * * *\"\n    - \"0 3 * * *\"\n")

	// An unterminated bracket drops the malformed selector, resolving the bare key (see splitSelectors).
	// Validate rejects this shape outright; Resolve is just permissive about it.
	got := Path("spec.schedules[0").Resolve(root)
	require.Len(t, got, 1)
	assert.Equal(t, KindSequence, got[0].Node.Kind, "the malformed selector is dropped, leaving the bare key")
	assert.Empty(t, got[0].Node.Value, "a sequence node carries no scalar value")
}

func TestPath_PreservesLineNumbers(t *testing.T) {
	root := decodeOne(t, "spec:\n  schedule: \"0 2 * * *\"\n")

	got := Path("spec.schedule").Resolve(root)
	require.Len(t, got, 1)
	assert.Equal(t, 2, got[0].Node.Line, "the located node still knows where it came from")
}

func TestPath_ResolveRecordsConcreteIndices(t *testing.T) {
	root := decodeOne(t, `
on:
  schedule:
    - cron: '30 5 * * 1'
    - cron: '0 0 * * *'
`)

	got := Path("on.schedule[].cron").Resolve(root)
	require.Len(t, got, 2)
	assert.Equal(t, "on.schedule[0].cron", got[0].Path)
	assert.Equal(t, "on.schedule[1].cron", got[1].Path)
}

func TestPath_ResolveKeepsExplicitIndex(t *testing.T) {
	root := decodeOne(t, "spec:\n  schedules:\n    - \"0 2 * * *\"\n    - \"0 3 * * *\"\n")

	got := Path("spec.schedules[1]").Resolve(root)
	require.Len(t, got, 1)
	assert.Equal(t, "spec.schedules[1]", got[0].Path)
}

func TestPath_ResolveOnPlainPath(t *testing.T) {
	root := decodeOne(t, "spec:\n  schedule: \"0 2 * * *\"\n")

	got := Path("spec.schedule").Resolve(root)
	require.Len(t, got, 1)
	assert.Equal(t, "spec.schedule", got[0].Path)
	assert.Equal(t, 2, got[0].Node.Line)
}

func TestPath_Validate(t *testing.T) {
	valid := []Path{
		"spec.schedule",
		"spec.schedules[]",
		"spec.schedules[1]",
		"on.schedule[].cron",
	}
	for _, p := range valid {
		t.Run(string(p), func(t *testing.T) {
			assert.NoError(t, p.Validate())
		})
	}
}

func TestPath_ValidateRejects(t *testing.T) {
	tests := []struct {
		path     Path
		contains string
	}{
		{"", "empty"},
		{"spec..schedule", "empty segment"},
		{"spec.schedule.", "empty segment"},
		{"spec.schedules[0", "unterminated"},
		{"spec.schedules[abc]", "not a number"},
		{"spec.schedules[-1]", "not a number"},
		{"spec.schedules[0]junk", "trailing"},
	}

	for _, tt := range tests {
		t.Run(string(tt.path), func(t *testing.T) {
			err := tt.path.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.contains)
		})
	}
}
