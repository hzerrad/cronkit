package inventory

import (
	"fmt"
	"strconv"
)

// Item is one discovered schedule, wherever it was found.
type Item struct {
	// Expression is the schedule exactly as written in the source
	Expression string `json:"expression"`
	// SourceID is the ID of the source that produced this item
	SourceID string `json:"source"`
	// Dialect names the grammar Expression is written in
	Dialect string `json:"dialect"`
	// Command is a shell command when Shell is true, and a display label otherwise
	Command string `json:"command,omitempty"`
	// Shell reports whether Command is a real shell command, which is what
	// gates the shell-specific hygiene rules
	Shell bool `json:"shell"`
	// Comment is the crontab inline comment; empty for other sources
	Comment string `json:"comment,omitempty"`
	// RunAs is the account a schedule runs as where the source expresses one, and empty otherwise.
	RunAs string `json:"runAs,omitempty"`
	// Timezone is an IANA name; empty means inherit the invocation default
	Timezone string `json:"timezone,omitempty"`
	// State reports whether this schedule actually runs
	State State `json:"state"`
	// Reason explains State when it is not StateActive
	Reason string `json:"reason,omitempty"`
	// Concurrency is the platform's overlapping-run policy
	Concurrency Concurrency `json:"concurrency,omitempty"`
	// Locator says where this item was found
	Locator Locator `json:"locator"`
}

// Locator identifies where in a source an item was found.
type Locator struct {
	// File is slash-separated and relative to the scan root
	File string `json:"file"`
	// Line is 1-indexed; 0 when the format cannot attribute one
	Line int `json:"line,omitempty"`
	// Document is the index of the document within a multi-document file
	Document int `json:"document,omitempty"`
	// Path is the structural path within the document, e.g. "spec.schedule"
	Path string `json:"path,omitempty"`
}

// String renders the locator for display, naming the structural path when there is one.
func (l Locator) String() string {
	base := l.File
	if l.Line > 0 {
		base = fmt.Sprintf("%s:%d", l.File, l.Line)
	}
	if l.Path != "" {
		return fmt.Sprintf("%s (%s)", base, l.Path)
	}
	return base
}

// Identity returns a stable id for the item, keyed by its address so it survives a reordering.
func (l Locator) Identity(index int, indexPrefix, linePrefix string) string {
	address := l.address()
	if address == "" {
		return fmt.Sprintf("%s%d", indexPrefix, index)
	}
	return linePrefix + address
}

// address renders what pins down one schedule; a file name alone is not one, so it returns "".
func (l Locator) address() string {
	var address string
	switch {
	case l.Line > 0 && l.File != "":
		address = fmt.Sprintf("%s:%d", l.File, l.Line)
	case l.Line > 0:
		address = strconv.Itoa(l.Line)
	case l.Path != "":
		address = l.File
	}
	if l.Path == "" {
		return address
	}
	// A flow-style sequence puts several schedules on one line; the path is what separates them.
	if address == "" {
		return l.Path
	}
	return address + "#" + l.Path
}
