package source

import (
	"fmt"
	"strconv"
	"strings"
)

// Path addresses values in a decoded document: dotted keys descend, [n] selects one, [] selects all.
type Path string

// Located is a node a path matched, with real indices in place of any [] selector.
type Located struct {
	Node *Node
	Path string
}

// Resolve returns every node the path addresses, in document order, with the concrete path found.
// No cycle guard is needed: convertNode always produces a tree with no shared pointers.
func (p Path) Resolve(root *Node) []Located {
	if root == nil {
		return nil
	}

	current := []Located{{Node: root}}
	if p == "" {
		return current
	}

	for _, segment := range strings.Split(string(p), ".") {
		name, selectors := splitSelectors(segment)

		if name != "" {
			current = descend(current, name)
		}
		for _, selector := range selectors {
			current = index(current, selector)
		}
		if len(current) == 0 {
			return nil
		}
	}

	return current
}

// splitSelectors separates "schedule[0][]" into the key and its bracket selectors, in order.
// An unterminated bracket isn't an error here; Validate rejects that shape instead.
func splitSelectors(segment string) (string, []string) {
	open := strings.Index(segment, "[")
	if open == -1 {
		return segment, nil
	}

	name := segment[:open]
	var selectors []string
	for rest := segment[open:]; rest != ""; {
		end := strings.Index(rest, "]")
		if end == -1 {
			break
		}
		selectors = append(selectors, rest[1:end])
		rest = rest[end+1:]
	}
	return name, selectors
}

// descend replaces each located node with the value stored under name,
// dropping nodes that do not have it.
func descend(located []Located, name string) []Located {
	var next []Located
	for _, l := range located {
		field, ok := l.Node.Field(name)
		if !ok {
			continue
		}
		next = append(next, Located{Node: field, Path: joinPath(l.Path, name)})
	}
	return next
}

// index applies one bracket selector: "" selects every element, a number
// selects one. Either way the resulting path records the concrete index.
func index(located []Located, selector string) []Located {
	var next []Located
	for _, l := range located {
		if l.Node.Kind != KindSequence {
			continue
		}
		if selector == "" {
			for i, item := range l.Node.Items {
				next = append(next, Located{Node: item, Path: fmt.Sprintf("%s[%d]", l.Path, i)})
			}
			continue
		}
		i, err := strconv.Atoi(selector)
		if err != nil || i < 0 || i >= len(l.Node.Items) {
			continue
		}
		next = append(next, Located{Node: l.Node.Items[i], Path: fmt.Sprintf("%s[%d]", l.Path, i)})
	}
	return next
}

// joinPath appends a key to a resolved path.
func joinPath(base, name string) string {
	if base == "" {
		return name
	}
	return base + "." + name
}

// Validate reports whether the path is well formed, catching a typo when the profile is built.
func (p Path) Validate() error {
	if p == "" {
		return fmt.Errorf("path is empty")
	}

	for _, segment := range strings.Split(string(p), ".") {
		if segment == "" {
			return fmt.Errorf("path %q has an empty segment", p)
		}
		if err := validateSegment(p, segment); err != nil {
			return err
		}
	}

	return nil
}

// validateSegment checks one dot-separated segment's bracket selectors.
func validateSegment(p Path, segment string) error {
	open := strings.Index(segment, "[")
	if open == -1 {
		return nil
	}

	for rest := segment[open:]; rest != ""; {
		if !strings.HasPrefix(rest, "[") {
			return fmt.Errorf("path %q has trailing text after a selector", p)
		}
		end := strings.Index(rest, "]")
		if end == -1 {
			return fmt.Errorf("path %q has an unterminated selector", p)
		}
		if selector := rest[1:end]; selector != "" {
			if n, err := strconv.Atoi(selector); err != nil || n < 0 {
				return fmt.Errorf("path %q selector %q is not a number", p, selector)
			}
		}
		rest = rest[end+1:]
	}

	return nil
}
