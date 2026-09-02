// Package inventory holds the data model for a discovered cron schedule and its JSON envelope.
// The JSON is a published contract; see SchemaVersion for what counts as a breaking change.
package inventory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// SchemaVersion is the version of the inventory JSON contract this build reads and writes.
// It identifies breaking changes only; an additive, ignorable field never bumps it.
const SchemaVersion = "1"

// Inventory is a set of discovered schedules and the envelope they travel in.
type Inventory struct {
	// SchemaVersion identifies the JSON contract
	SchemaVersion string `json:"schemaVersion"`
	// Root is the directory items are relative to, when there is one
	Root string `json:"root,omitempty"`
	// Items are the discovered schedules
	Items []Item `json:"items"`
}

// New creates an inventory carrying the current schema version.
func New(root string, items []Item) *Inventory {
	if items == nil {
		items = []Item{}
	}
	return &Inventory{
		SchemaVersion: SchemaVersion,
		Root:          root,
		Items:         items,
	}
}

// Sort orders items by file, document, line, path, expression, then source for stable output.
func (inv *Inventory) Sort() {
	sort.SliceStable(inv.Items, func(i, j int) bool {
		a, b := inv.Items[i], inv.Items[j]
		aLoc, bLoc := a.Locator, b.Locator
		if aLoc.File != bLoc.File {
			return aLoc.File < bLoc.File
		}
		if aLoc.Document != bLoc.Document {
			return aLoc.Document < bLoc.Document
		}
		if aLoc.Line != bLoc.Line {
			return aLoc.Line < bLoc.Line
		}
		if aLoc.Path != bLoc.Path {
			return aLoc.Path < bLoc.Path
		}
		if a.Expression != b.Expression {
			return a.Expression < b.Expression
		}
		return a.SourceID < b.SourceID
	})
}

// Encode writes the inventory as indented JSON.
func (inv *Inventory) Encode(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(inv)
}

// Decode reads an inventory, rejecting a schema version this build doesn't understand.
// If input isn't JSON, the error points at the likely cause: scan run without --json.
func Decode(r io.Reader) (*Inventory, error) {
	br := bufio.NewReader(r)
	looksLikeJSON := startsLikeJSON(br)

	dec := json.NewDecoder(br)
	var inv Inventory
	if err := dec.Decode(&inv); err != nil {
		if !looksLikeJSON {
			return nil, fmt.Errorf(
				"input does not look like an inventory: scan writes a table by default; use --json to produce one (%w)", err)
		}
		return nil, fmt.Errorf("failed to decode inventory: %w", err)
	}
	if dec.More() {
		return nil, fmt.Errorf("failed to decode inventory: unexpected content after the JSON document")
	}
	if inv.SchemaVersion == "" {
		return nil, fmt.Errorf("input does not look like an inventory: schemaVersion is missing")
	}
	if inv.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("unsupported inventory schema version %q (this build reads %q)",
			inv.SchemaVersion, SchemaVersion)
	}
	return &inv, nil
}

// startsLikeJSON reports whether the first non-whitespace byte is '{' or '['; it peeks, not reads.
func startsLikeJSON(br *bufio.Reader) bool {
	for {
		b, err := br.Peek(1)
		if err != nil {
			return false
		}
		switch b[0] {
		case ' ', '\t', '\n', '\r':
			_, _ = br.Discard(1)
			continue
		case '{', '[':
			return true
		default:
			return false
		}
	}
}
