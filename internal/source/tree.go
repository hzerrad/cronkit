// Package source discovers schedules inside files, independent of the file's format.
package source

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// Kind classifies a node in a decoded document.
type Kind int

const (
	// KindScalar is a leaf value
	KindScalar Kind = iota
	// KindSequence is an ordered list
	KindSequence
	// KindMapping is a set of keyed fields
	KindMapping
)

// Node is one value in a decoded document, in a shape no source format owns.
type Node struct {
	// Kind classifies this node
	Kind Kind
	// Value is the text of a scalar (empty for other kinds); the YAML tag is discarded.
	Value string
	// Line is 1-indexed; 0 when the format cannot attribute one
	Line int
	// Items holds the elements of a sequence
	Items []*Node
	// Keys lists mapping keys in document order, so iteration is deterministic
	Keys []string
	// Fields maps a key to its value
	Fields map[string]*Node
}

// Field returns the value stored under name, and whether this node had it.
func (n *Node) Field(name string) (*Node, bool) {
	if n == nil || n.Kind != KindMapping {
		return nil, false
	}
	v, ok := n.Fields[name]
	return v, ok
}

// Document is one decoded document and its position within the file.
type Document struct {
	// Index is the 0-based position of this document in the file,
	// counting documents that were skipped for being empty.
	Index int
	// Root is the document's top-level node.
	Root *Node
}

// DecodeYAML decodes YAML bytes into one tree per document, sharing one node budget across the file.
func DecodeYAML(data []byte) ([]Document, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))

	budget := maxNodes
	var docs []Document
	for idx := 0; ; idx++ {
		var doc yaml.Node
		err := dec.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to decode yaml: %w", err)
		}
		if node := convert(&doc, &budget); node != nil {
			docs = append(docs, Document{Index: idx, Root: node})
		}
	}

	return docs, nil
}

// maxDepth bounds recursion so pathological nesting can't exhaust the stack.
const maxDepth = 10_000

// maxNodes bounds total Node allocations per file, since a diamond of nested anchors can force
// millions of them; sized to cap worst-case retained memory around 41 MB.
const maxNodes = 200_000

// convert turns a yaml.Node into a format-neutral Node, resolving wrappers and aliases.
func convert(n *yaml.Node, budget *int) *Node {
	return convertNode(n, make(map[*yaml.Node]bool), 0, budget)
}

// convertNode is convert's recursive worker; active guards against a self-referential alias.
func convertNode(n *yaml.Node, active map[*yaml.Node]bool, depth int, budget *int) *Node {
	if n == nil {
		return nil
	}
	if depth > maxDepth {
		return nil
	}
	if active[n] {
		return nil
	}
	active[n] = true
	defer delete(active, n)

	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) == 0 {
			return nil
		}
		content := n.Content[0]
		if content.Kind == yaml.ScalarNode && content.Tag == "!!null" && content.Value == "" {
			// An empty document decodes to an implicit null scalar rather than empty Content.
			return nil
		}
		return convertNode(content, active, depth+1, budget)

	case yaml.AliasNode:
		return convertNode(n.Alias, active, depth+1, budget)

	case yaml.MappingNode:
		if *budget <= 0 {
			return nil
		}
		*budget--
		node := &Node{
			Kind:   KindMapping,
			Line:   n.Line,
			Fields: make(map[string]*Node, len(n.Content)/2),
		}
		var merges []*yaml.Node
		for i := 0; i+1 < len(n.Content); i += 2 {
			keyNode := n.Content[i]
			if isMergeKey(keyNode) {
				merges = append(merges, n.Content[i+1])
				continue
			}
			key := keyNode.Value
			value := convertNode(n.Content[i+1], active, depth+1, budget)
			if value == nil {
				continue
			}
			// A key repeated in the same mapping is last-wins for both its value and its position.
			if _, seen := node.Fields[key]; seen {
				removeKey(node, key)
			}
			node.Keys = append(node.Keys, key)
			node.Fields[key] = value
		}
		// Merge keys apply last, so an explicit key wins over a merged one and earlier merges win over later ones.
		for _, merge := range merges {
			for _, source := range flattenMergeSources(merge, active, depth+1, budget) {
				merged := convertNode(source, active, depth+1, budget)
				if merged == nil || merged.Kind != KindMapping {
					continue
				}
				for _, key := range merged.Keys {
					if _, exists := node.Fields[key]; exists {
						continue
					}
					node.Fields[key] = merged.Fields[key]
					node.Keys = append(node.Keys, key)
				}
			}
		}
		return node

	case yaml.SequenceNode:
		if *budget <= 0 {
			return nil
		}
		*budget--
		node := &Node{Kind: KindSequence, Line: n.Line}
		for _, item := range n.Content {
			if converted := convertNode(item, active, depth+1, budget); converted != nil {
				node.Items = append(node.Items, converted)
			}
		}
		return node

	default:
		if *budget <= 0 {
			return nil
		}
		*budget--
		return &Node{Kind: KindScalar, Value: n.Value, Line: n.Line}
	}
}

// removeKey deletes key's existing entry from node.Keys, so a caller about to
// re-append it ends up with one entry at the new position instead of two.
func removeKey(node *Node, key string) {
	for i, k := range node.Keys {
		if k == key {
			node.Keys = append(node.Keys[:i], node.Keys[i+1:]...)
			return
		}
	}
}

// isMergeKey reports whether n is the YAML merge key "<<", matched by tag so a quoted "<<" is left alone.
func isMergeKey(n *yaml.Node) bool {
	return n.Kind == yaml.ScalarNode && n.Value == "<<" &&
		(n.Tag == "" || n.Tag == "!" || n.Tag == "!!merge")
}

// flattenMergeSources expands a merge key's value into the mapping nodes it draws from, guarding
// against cycles with its own active set, depth ceiling, and budget.
func flattenMergeSources(n *yaml.Node, active map[*yaml.Node]bool, depth int, budget *int) []*yaml.Node {
	if n == nil {
		return nil
	}
	if depth > maxDepth {
		return nil
	}
	if active[n] {
		return nil
	}
	if *budget <= 0 {
		return nil
	}
	active[n] = true
	defer delete(active, n)

	switch n.Kind {
	case yaml.AliasNode:
		return flattenMergeSources(n.Alias, active, depth+1, budget)
	case yaml.SequenceNode:
		var out []*yaml.Node
		for _, item := range n.Content {
			out = append(out, flattenMergeSources(item, active, depth+1, budget)...)
		}
		return out
	case yaml.MappingNode:
		*budget--
		return []*yaml.Node{n}
	default:
		return nil
	}
}
