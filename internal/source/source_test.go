package source

import (
	"io/fs"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSource matches any unit whose path equals want.
type fakeSource struct {
	id    string
	want  string
	items []inventory.Item
}

func (f fakeSource) ID() string        { return f.id }
func (f fakeSource) Match(u Unit) bool { return u.Path == f.want }
func (f fakeSource) Extract(u Unit, fsys fs.FS) ([]inventory.Item, error) {
	return f.items, nil
}

func TestRegistry_MatchAllReturnsEveryMatchInRegistrationOrder(t *testing.T) {
	first := fakeSource{id: "first", want: "a.yaml"}
	second := fakeSource{id: "second", want: "a.yaml"}
	registry := NewRegistry(first, second)

	got := registry.MatchAll(Unit{Path: "a.yaml"})

	require.Len(t, got, 2, "a path can legitimately belong to more than one source")
	assert.Equal(t, "first", got[0].ID())
	assert.Equal(t, "second", got[1].ID(), "registration order decides the order results come back in")
}

func TestRegistry_MatchAllReturnsNoneWhenNothingMatches(t *testing.T) {
	registry := NewRegistry(fakeSource{id: "first", want: "a.yaml"})

	got := registry.MatchAll(Unit{Path: "b.toml"})

	assert.Empty(t, got, "an unrecognised file is not an error, it is simply not ours")
}

func TestRegistry_SourcesPreservesRegistrationOrder(t *testing.T) {
	registry := NewRegistry(
		fakeSource{id: "a"}, fakeSource{id: "b"}, fakeSource{id: "c"},
	)

	var ids []string
	for _, s := range registry.Sources() {
		ids = append(ids, s.ID())
	}
	assert.Equal(t, []string{"a", "b", "c"}, ids)
}

func TestRegistry_SourcesCannotBeMutatedByCallers(t *testing.T) {
	registry := NewRegistry(fakeSource{id: "a"}, fakeSource{id: "b"})

	got := registry.Sources()
	got[0] = fakeSource{id: "hijacked"}

	assert.Equal(t, "a", registry.Sources()[0].ID(),
		"Sources returns a copy so a caller cannot rewrite the registry")
}

// statFakerFS reports a chosen size for one path via Stat, without an actual file of that size.
type statFakerFS struct {
	fs.FS
	path string
	size int64
}

func (f statFakerFS) Stat(name string) (fs.FileInfo, error) {
	if name == f.path {
		return fakeFileInfo{name: name, size: f.size}, nil
	}
	return fs.Stat(f.FS, name)
}

func TestDecodeUnit_RefusesOversizedFiles(t *testing.T) {
	fsys := fstest.MapFS{"huge.yaml": &fstest.MapFile{Data: []byte("kind: CronJob\n")}}
	info, err := fs.Stat(fsys, "huge.yaml")
	require.NoError(t, err)
	unit := Unit{Path: "huge.yaml", Info: fakeFileInfo{name: info.Name(), size: maxUnitBytes + 1}}

	_, err = decodeUnit(unit, fsys)

	require.Error(t, err, "a document over the size ceiling must be refused rather than read")
	assert.Contains(t, err.Error(), "huge.yaml")
}

func TestDecodeUnit_AllowsFilesAtTheCeiling(t *testing.T) {
	fsys := fstest.MapFS{"a.yaml": &fstest.MapFile{Data: []byte("kind: CronJob\n")}}
	unit := Unit{Path: "a.yaml", Info: fakeFileInfo{name: "a.yaml", size: maxUnitBytes}}

	_, err := decodeUnit(unit, fsys)

	require.NoError(t, err, "a file exactly at the ceiling is not oversized")
}

func TestDecodeUnit_NilInfoStillEnforcesTheCeiling(t *testing.T) {
	// A Unit built without Info must not bypass the ceiling (see checkUnitSize).
	base := fstest.MapFS{"huge.yaml": &fstest.MapFile{Data: []byte("kind: CronJob\n")}}
	fsys := statFakerFS{FS: base, path: "huge.yaml", size: maxUnitBytes + 1}
	unit := Unit{Path: "huge.yaml"}

	_, err := decodeUnit(unit, fsys)

	require.Error(t, err, "a nil Info must not skip the size check")
	assert.Contains(t, err.Error(), "huge.yaml")
}

func TestDecodeUnit_NilInfoWithinCeilingDecodesNormally(t *testing.T) {
	fsys := fstest.MapFS{"a.yaml": &fstest.MapFile{Data: []byte("kind: CronJob\n")}}
	unit := Unit{Path: "a.yaml"}

	docs, err := decodeUnit(unit, fsys)

	require.NoError(t, err)
	require.Len(t, docs, 1)
}

func TestUnit_CarriesFileInfo(t *testing.T) {
	fsys := fstest.MapFS{"a.yaml": &fstest.MapFile{Data: []byte("kind: A\n")}}
	info, err := fs.Stat(fsys, "a.yaml")
	require.NoError(t, err)

	unit := Unit{Path: "a.yaml", Info: info}

	assert.Equal(t, "a.yaml", unit.Path)
	assert.Equal(t, int64(8), unit.Info.Size())
}

// countingFS counts ReadFile calls so a test can prove decoding happened once.
type countingFS struct {
	fs.FS
	reads map[string]int
}

func (c *countingFS) Open(name string) (fs.File, error) {
	c.reads[name]++
	return c.FS.Open(name)
}

func TestUnit_DocumentsDecodesOncePerCache(t *testing.T) {
	base := fstest.MapFS{"a.yaml": &fstest.MapFile{Data: []byte("kind: CronJob\n")}}
	fsys := &countingFS{FS: base, reads: map[string]int{}}
	info, err := fs.Stat(base, "a.yaml")
	require.NoError(t, err)

	unit := Unit{Path: "a.yaml", Info: info, Cache: NewDocumentCache()}

	first, err := unit.Documents(fsys)
	require.NoError(t, err)
	second, err := unit.Documents(fsys)
	require.NoError(t, err)

	assert.Equal(t, 1, fsys.reads["a.yaml"], "a cached unit reads the file once")
	require.Len(t, first, 1)
	assert.Same(t, first[0].Root, second[0].Root, "both calls return the same tree")
}

func TestUnit_DocumentsWithoutCacheDecodesEveryTime(t *testing.T) {
	base := fstest.MapFS{"a.yaml": &fstest.MapFile{Data: []byte("kind: CronJob\n")}}
	fsys := &countingFS{FS: base, reads: map[string]int{}}
	info, err := fs.Stat(base, "a.yaml")
	require.NoError(t, err)

	unit := Unit{Path: "a.yaml", Info: info}

	_, err = unit.Documents(fsys)
	require.NoError(t, err)
	_, err = unit.Documents(fsys)
	require.NoError(t, err)

	assert.Equal(t, 2, fsys.reads["a.yaml"],
		"a zero-value Unit still works, it simply does not cache")
}

func TestUnit_DocumentsBypassesACacheBoundToADifferentPath(t *testing.T) {
	fsys := fstest.MapFS{
		"a.yaml": &fstest.MapFile{Data: []byte("kind: A\n")},
		"b.yaml": &fstest.MapFile{Data: []byte("kind: B\n")},
	}
	infoA, err := fs.Stat(fsys, "a.yaml")
	require.NoError(t, err)
	infoB, err := fs.Stat(fsys, "b.yaml")
	require.NoError(t, err)

	cache := NewDocumentCache()

	unitA := Unit{Path: "a.yaml", Info: infoA, Cache: cache}
	docsA, err := unitA.Documents(fsys)
	require.NoError(t, err)
	require.Len(t, docsA, 1)
	kindA, _ := docsA[0].Root.Field("kind")
	assert.Equal(t, "A", kindA.Value, "the cache should now be bound to a.yaml")

	unitB := Unit{Path: "b.yaml", Info: infoB, Cache: cache}
	docsB, err := unitB.Documents(fsys)
	require.NoError(t, err)
	require.Len(t, docsB, 1)
	kindB, _ := docsB[0].Root.Field("kind")
	assert.Equal(t, "B", kindB.Value,
		"a cache bound to a different path must decode the requested file fresh, not serve a's contents")

	// The cache itself is left as it was: still bound to a.yaml, still
	// serving a's contents to whoever asks for that path again.
	docsAAgain, err := unitA.Documents(fsys)
	require.NoError(t, err)
	require.Len(t, docsAAgain, 1)
	kindAAgain, _ := docsAAgain[0].Root.Field("kind")
	assert.Equal(t, "A", kindAAgain.Value)
}

func TestUnit_DocumentsCachesFailure(t *testing.T) {
	base := fstest.MapFS{"a.yaml": &fstest.MapFile{Data: []byte("kind: [unclosed\n")}}
	fsys := &countingFS{FS: base, reads: map[string]int{}}
	info, err := fs.Stat(base, "a.yaml")
	require.NoError(t, err)

	unit := Unit{Path: "a.yaml", Info: info, Cache: NewDocumentCache()}

	_, firstErr := unit.Documents(fsys)
	_, secondErr := unit.Documents(fsys)

	require.Error(t, firstErr)
	assert.Equal(t, firstErr.Error(), secondErr.Error(),
		"a failed decode is cached too, so every source sees the same error")
	assert.Equal(t, 1, fsys.reads["a.yaml"])
}

func TestUnit_DocumentsIsSafeForConcurrentUse(t *testing.T) {
	base := fstest.MapFS{"a.yaml": &fstest.MapFile{Data: []byte("kind: CronJob\n")}}
	fsys := &countingFS{FS: base, reads: map[string]int{}}
	info, err := fs.Stat(base, "a.yaml")
	require.NoError(t, err)

	cache := NewDocumentCache()
	unit := Unit{Path: "a.yaml", Info: info, Cache: cache}

	const goroutines = 8
	roots := make([]*Node, goroutines)
	errs := make([]error, goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			docs, err := unit.Documents(fsys)
			errs[i] = err
			if err == nil && len(docs) > 0 {
				roots[i] = docs[0].Root
			}
		}(i)
	}
	wg.Wait()

	for i := range goroutines {
		require.NoError(t, errs[i])
		assert.Same(t, roots[0], roots[i], "every goroutine must see the identical decoded tree")
	}
	assert.Equal(t, 1, fsys.reads["a.yaml"], "concurrent callers sharing a cache must decode the file only once")
}
