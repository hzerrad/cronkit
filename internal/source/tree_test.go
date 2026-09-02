package source

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const cronJobYAML = `apiVersion: batch/v1
kind: CronJob
metadata:
  name: backup
spec:
  schedule: "0 2 * * *"
  timeZone: Europe/Paris
  suspend: false
`

func TestDecodeYAML_MappingWithLineNumbers(t *testing.T) {
	docs, err := DecodeYAML([]byte(cronJobYAML))
	require.NoError(t, err)
	require.Len(t, docs, 1)

	root := docs[0].Root
	require.Equal(t, KindMapping, root.Kind)

	kind, ok := root.Field("kind")
	require.True(t, ok)
	assert.Equal(t, KindScalar, kind.Kind)
	assert.Equal(t, "CronJob", kind.Value)
	assert.Equal(t, 2, kind.Line)

	spec, ok := root.Field("spec")
	require.True(t, ok)
	schedule, ok := spec.Field("schedule")
	require.True(t, ok)
	assert.Equal(t, "0 2 * * *", schedule.Value, "quotes are not part of the value")
	assert.Equal(t, 6, schedule.Line)
}

func TestDecodeYAML_KeysAreDocumentOrdered(t *testing.T) {
	// Deliberately out of alphabetical order, so this fails an implementation that sorts Keys.
	docs, err := DecodeYAML([]byte("zebra: 1\napple: 2\nmango: 3\n"))
	require.NoError(t, err)

	assert.Equal(t, []string{"zebra", "apple", "mango"}, docs[0].Root.Keys,
		"key order follows the document so output never depends on map iteration")
}

func TestDecodeYAML_Sequence(t *testing.T) {
	docs, err := DecodeYAML([]byte(`
on:
  schedule:
    - cron: '30 5 * * 1'
    - cron: '0 0 * * *'
`))
	require.NoError(t, err)

	on, ok := docs[0].Root.Field("on")
	require.True(t, ok, "yaml.v3 keeps the key on as a string, not the boolean true")

	schedule, ok := on.Field("schedule")
	require.True(t, ok)
	require.Equal(t, KindSequence, schedule.Kind)
	require.Len(t, schedule.Items, 2)

	first, ok := schedule.Items[0].Field("cron")
	require.True(t, ok)
	assert.Equal(t, "30 5 * * 1", first.Value)
	assert.Equal(t, 4, first.Line)
}

func TestDecodeYAML_MultipleDocuments(t *testing.T) {
	docs, err := DecodeYAML([]byte("kind: A\n---\nkind: B\n"))
	require.NoError(t, err)
	require.Len(t, docs, 2)

	a, _ := docs[0].Root.Field("kind")
	b, _ := docs[1].Root.Field("kind")
	assert.Equal(t, "A", a.Value)
	assert.Equal(t, "B", b.Value)
	assert.Equal(t, 0, docs[0].Index)
	assert.Equal(t, 1, docs[1].Index)
}

func TestDecodeYAML_SkipsEmptyDocuments(t *testing.T) {
	docs, err := DecodeYAML([]byte("---\n---\nkind: A\n"))
	require.NoError(t, err)
	require.Len(t, docs, 1, "empty documents carry nothing to extract")
}

func TestDecodeYAML_IndexSurvivesSkippedEmptyDocuments(t *testing.T) {
	// "kind: B" is the third document (index 2); a naive slice index would misreport it as document 1.
	docs, err := DecodeYAML([]byte("kind: A\n---\n---\nkind: B\n"))
	require.NoError(t, err)
	require.Len(t, docs, 2)

	assert.Equal(t, 0, docs[0].Index)
	a, _ := docs[0].Root.Field("kind")
	assert.Equal(t, "A", a.Value)

	assert.Equal(t, 2, docs[1].Index, "the skipped empty document still counts toward the index")
	b, _ := docs[1].Root.Field("kind")
	assert.Equal(t, "B", b.Value)
}

func TestDecodeYAML_ResolvesAliases(t *testing.T) {
	docs, err := DecodeYAML([]byte(`
base: &base
  schedule: "0 2 * * *"
copy: *base
`))
	require.NoError(t, err)

	copied, ok := docs[0].Root.Field("copy")
	require.True(t, ok)
	schedule, ok := copied.Field("schedule")
	require.True(t, ok, "an alias resolves to the node it points at")
	assert.Equal(t, "0 2 * * *", schedule.Value)
}

func TestDecodeYAML_SelfReferentialAliasDoesNotCrash(t *testing.T) {
	// An anchor whose own value aliases back to itself; without cycle detection this overflows the stack.
	docs, err := DecodeYAML([]byte(`
a: &anchor
  b: *anchor
`))
	require.NoError(t, err)
	require.Len(t, docs, 1)

	a, ok := docs[0].Root.Field("a")
	require.True(t, ok)
	assert.Equal(t, KindMapping, a.Kind)

	_, ok = a.Field("b")
	assert.False(t, ok, "the cycle back to the anchor still being expanded is dropped, not followed")
}

func TestDecodeYAML_DiamondAliasExpandsBothReferences(t *testing.T) {
	// The same anchor referenced twice is ordinary reuse and must not be mistaken for a cycle.
	docs, err := DecodeYAML([]byte(`
base: &base
  schedule: "0 2 * * *"
copy1: *base
copy2: *base
`))
	require.NoError(t, err)

	copy1, ok := docs[0].Root.Field("copy1")
	require.True(t, ok)
	schedule1, ok := copy1.Field("schedule")
	require.True(t, ok, "cycle detection must not block ordinary reuse of the same anchor")
	assert.Equal(t, "0 2 * * *", schedule1.Value)

	copy2, ok := docs[0].Root.Field("copy2")
	require.True(t, ok)
	schedule2, ok := copy2.Field("schedule")
	require.True(t, ok)
	assert.Equal(t, "0 2 * * *", schedule2.Value)
}

// aliasBombDocument returns a diamond alias bomb: a small yaml.Node graph that asks for
// roughly 4^15 conversions once convertNode expands it.
func aliasBombDocument() string {
	var b strings.Builder
	b.WriteString("a0: &a0 [\"leaf\", \"leaf\", \"leaf\", \"leaf\"]\n")
	for i := 1; i <= 15; i++ {
		fmt.Fprintf(&b, "a%d: &a%d [*a%d, *a%d, *a%d, *a%d]\n", i, i, i-1, i-1, i-1, i-1)
	}
	b.WriteString("bomb: *a15\n")
	return b.String()
}

func TestDecodeYAML_AliasBombIsBounded(t *testing.T) {
	// maxNodes must stop this in a small fraction of a second, well inside this test's own timeout.
	start := time.Now()
	docs, err := DecodeYAML([]byte(aliasBombDocument()))
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.Len(t, docs, 1, "conversion still returns what it managed to build, not an error")
	assert.Less(t, elapsed, 5*time.Second,
		"the expansion budget must keep conversion fast even for a diamond alias bomb")
}

func TestDecodeYAML_AliasBombAcrossManyDocumentsIsBounded(t *testing.T) {
	// A per-document budget would let many documents defeat maxNodes; DecodeYAML shares one budget instead.
	const documentCount = 200
	docs := make([]string, documentCount)
	for i := range docs {
		docs[i] = aliasBombDocument()
	}
	data := strings.Join(docs, "---\n")

	start := time.Now()
	result, err := DecodeYAML([]byte(data))
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.LessOrEqual(t, len(result), documentCount)
	assert.Less(t, elapsed, 5*time.Second,
		"a shared, per-file budget must keep conversion fast no matter how many documents the file holds")
}

func TestConvert_DepthCeilingStopsRecursion(t *testing.T) {
	// yaml.v3's own parser refuses input nested this deep, so the graph is hand-built rather than decoded.
	current := &yaml.Node{Kind: yaml.ScalarNode, Value: "leaf"}
	for i := 0; i < maxDepth+10; i++ {
		current = &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "next"},
				current,
			},
		}
	}

	var node *Node
	budget := maxNodes
	require.NotPanics(t, func() {
		node = convert(current, &budget)
	})
	require.NotNil(t, node)
	assert.Equal(t, KindMapping, node.Kind)

	// Walk down as far as conversion actually went. If the ceiling didn't
	// stop it, this would walk the full maxDepth+10 chain built above.
	depth := 0
	for {
		next, ok := node.Field("next")
		if !ok {
			break
		}
		node = next
		depth++
	}
	assert.Less(t, depth, maxDepth+10, "the depth ceiling must stop conversion before the full chain is walked")
}

func TestDecodeYAML_DuplicateKeyIsLastWinsForValueAndPosition(t *testing.T) {
	root := decodeOne(t, "a: 1\nb: 2\na: 3\n")

	assert.Equal(t, []string{"b", "a"}, root.Keys,
		"the repeated key moves to the position of its last occurrence")

	a, ok := root.Field("a")
	require.True(t, ok)
	assert.Equal(t, "3", a.Value, "the last occurrence's value wins")
}

func TestDecodeYAML_ResolvesMergeKey(t *testing.T) {
	docs, err := DecodeYAML([]byte(`
defaults: &defaults
  schedule: "0 2 * * *"
job:
  <<: *defaults
  name: backup
`))
	require.NoError(t, err)

	job, ok := docs[0].Root.Field("job")
	require.True(t, ok)

	schedule, ok := job.Field("schedule")
	require.True(t, ok, "a merged field must be reachable by name")
	assert.Equal(t, "0 2 * * *", schedule.Value)

	name, ok := job.Field("name")
	require.True(t, ok)
	assert.Equal(t, "backup", name.Value)

	assert.ElementsMatch(t, []string{"name", "schedule"}, job.Keys,
		"the merge key itself must not appear in Keys")
}

func TestDecodeYAML_MergeKeySequencePrefersEarlierSource(t *testing.T) {
	docs, err := DecodeYAML([]byte(`
a: &a
  x: "from-a"
  y: "from-a"
b: &b
  y: "from-b"
  z: "from-b"
job:
  <<: [*a, *b]
`))
	require.NoError(t, err)

	job, ok := docs[0].Root.Field("job")
	require.True(t, ok)

	x, ok := job.Field("x")
	require.True(t, ok)
	assert.Equal(t, "from-a", x.Value)

	y, ok := job.Field("y")
	require.True(t, ok)
	assert.Equal(t, "from-a", y.Value, "the earlier merge source wins when both define the same key")

	z, ok := job.Field("z")
	require.True(t, ok)
	assert.Equal(t, "from-b", z.Value)
}

func TestDecodeYAML_LocalKeyOverridesMerge(t *testing.T) {
	// The local key comes before "<<"; a naive sequential merge would overwrite it with the merged value.
	docs, err := DecodeYAML([]byte(`
defaults: &defaults
  schedule: "0 2 * * *"
  name: template
job:
  name: backup
  <<: *defaults
`))
	require.NoError(t, err)

	job, ok := docs[0].Root.Field("job")
	require.True(t, ok)

	name, ok := job.Field("name")
	require.True(t, ok)
	assert.Equal(t, "backup", name.Value, "a local key wins over a merged one regardless of position")

	schedule, ok := job.Field("schedule")
	require.True(t, ok)
	assert.Equal(t, "0 2 * * *", schedule.Value)
}

func TestDecodeYAML_MergeCycleViaSelfReferentialSequence(t *testing.T) {
	// The merge value is a sequence, anchored as m, whose only element aliases back to m itself.
	docs, err := DecodeYAML([]byte(`
job:
  <<: &m
    - *m
`))
	require.NoError(t, err)
	require.Len(t, docs, 1)

	job, ok := docs[0].Root.Field("job")
	require.True(t, ok)
	assert.Equal(t, KindMapping, job.Kind)
	assert.Empty(t, job.Keys, "a merge source that never resolves to a mapping contributes nothing")
}

func TestDecodeYAML_MergeCycleViaAliasedSequence(t *testing.T) {
	// Same hazard, reached through a merge that points at a separately
	// anchored self-referential sequence rather than an inline one.
	docs, err := DecodeYAML([]byte(`
a: &a
  - *a
job:
  <<: *a
`))
	require.NoError(t, err)
	require.Len(t, docs, 1)

	job, ok := docs[0].Root.Field("job")
	require.True(t, ok)
	assert.Equal(t, KindMapping, job.Kind)
	assert.Empty(t, job.Keys)
}

func TestDecodeYAML_Malformed(t *testing.T) {
	_, err := DecodeYAML([]byte("kind: [unclosed\n"))
	require.Error(t, err)
}

func TestNode_FieldOnNonMapping(t *testing.T) {
	scalar := &Node{Kind: KindScalar, Value: "x"}
	_, ok := scalar.Field("anything")
	assert.False(t, ok, "only mappings have fields")
}
