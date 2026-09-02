package source

import (
	"fmt"
	"io/fs"
	"sync"

	"github.com/hzerrad/cronkit/internal/inventory"
)

// maxUnitBytes is a conservative ceiling on how large a file this package will load into memory.
const maxUnitBytes = 16 * 1024 * 1024 // 16 MiB

// checkUnitSize enforces maxUnitBytes before a caller reads u's file, statting the file if u.Info is absent.
func checkUnitSize(u Unit, fsys fs.FS) error {
	size := int64(-1)
	if u.Info != nil {
		size = u.Info.Size()
	} else if info, err := fs.Stat(fsys, u.Path); err == nil {
		size = info.Size()
	}
	if size > maxUnitBytes {
		return fmt.Errorf("%s is %d bytes, over the %d byte size ceiling", u.Path, size, maxUnitBytes)
	}
	return nil
}

// Unit is one candidate file a source may recognise.
type Unit struct {
	// Path is slash-separated and relative to the root being scanned
	Path string
	// Info describes the file, so a source can decline without opening it
	Info fs.FileInfo
	// Cache optionally memoises this file's decoded documents. A zero-value
	// Unit is valid and simply decodes on every call.
	Cache *DocumentCache
}

// DocumentCache memoises one file's decoded documents, safe for concurrent use, so every source matching that
// file shares a single read and parse. A Documents call whose Unit names a different path bypasses the cache.
type DocumentCache struct {
	once sync.Once
	path string
	docs []Document
	err  error
}

// NewDocumentCache creates an empty cache for one file.
func NewDocumentCache() *DocumentCache {
	return &DocumentCache{}
}

// Documents decodes the unit's file, reusing the cached result when bound to this path — see DocumentCache.
func (u Unit) Documents(fsys fs.FS) ([]Document, error) {
	if u.Cache == nil {
		return decodeUnit(u, fsys)
	}

	u.Cache.once.Do(func() {
		u.Cache.path = u.Path
		u.Cache.docs, u.Cache.err = decodeUnit(u, fsys)
	})

	if u.Cache.path != u.Path {
		return decodeUnit(u, fsys)
	}
	return u.Cache.docs, u.Cache.err
}

// decodeUnit reads and decodes one unit's file.
func decodeUnit(u Unit, fsys fs.FS) ([]Document, error) {
	if err := checkUnitSize(u, fsys); err != nil {
		return nil, err
	}

	data, err := fs.ReadFile(fsys, u.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", u.Path, err)
	}

	docs, err := DecodeYAML(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", u.Path, err)
	}
	return docs, nil
}

// Source extracts schedules from files it recognises; implementations are data-driven where possible.
type Source interface {
	// ID names the source in output and in --source filters
	ID() string

	// Match reports whether this source recognises the unit; it must not read the file.
	Match(u Unit) bool

	// Extract reads the unit and returns the schedules it holds; it receives the filesystem so a source can
	// consult sibling files.
	Extract(u Unit, fsys fs.FS) ([]inventory.Item, error)
}

// Registry holds the sources a scan will try, in the order they were given.
type Registry struct {
	sources []Source
}

// NewRegistry creates a registry from sources, which are tried in order.
func NewRegistry(sources ...Source) *Registry {
	return &Registry{sources: sources}
}

// Sources returns the registered sources in registration order.
func (r *Registry) Sources() []Source {
	out := make([]Source, len(r.sources))
	copy(out, r.sources)
	return out
}

// MatchAll returns every source recognising the unit, since a path can belong to more than one source.
func (r *Registry) MatchAll(u Unit) []Source {
	var matched []Source
	for _, s := range r.sources {
		if s.Match(u) {
			matched = append(matched, s)
		}
	}
	return matched
}
