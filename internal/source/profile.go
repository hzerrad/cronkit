package source

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
	"time"

	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/inventory"
)

// templateMarkers are the substrings that mean an expression was not resolved
// by whatever templating layer produced the file.
var templateMarkers = []string{"{{", "${", "$("}

// FieldMatch requires a document field to hold an exact value.
type FieldMatch struct {
	// Path addresses the field
	Path Path `json:"path" yaml:"path"`
	// Equals is the value the field must hold
	Equals string `json:"equals" yaml:"equals"`
}

// Profile describes one ecosystem's file shape as data, so adding an ecosystem costs a struct literal.
type Profile struct {
	// ID names the source this profile becomes
	ID string `json:"id" yaml:"id"`
	// Extensions are the file suffixes to consider, e.g. ".yaml"
	Extensions []string `json:"extensions" yaml:"extensions"`
	// DirPrefix optionally restricts matching to files whose immediate parent directory equals this exactly.
	DirPrefix string `json:"dir_prefix" yaml:"dir_prefix"`
	// Match requires every listed field to hold its value; a path must resolve to exactly one node.
	Match []FieldMatch `json:"match" yaml:"match"`
	// Schedules are the paths a schedule may live at, tried in order
	Schedules []Path `json:"schedules" yaml:"schedules"`
	// Timezone optionally addresses an IANA timezone name
	Timezone Path `json:"timezone" yaml:"timezone"`
	// TimezoneFixed is the timezone for platforms that have no field for one
	TimezoneFixed string `json:"timezone_fixed" yaml:"timezone_fixed"`
	// Suspend optionally addresses a flag meaning the schedule does not run
	Suspend Path `json:"suspend" yaml:"suspend"`
	// Concurrency optionally addresses the overlapping-run policy
	Concurrency Path `json:"concurrency" yaml:"concurrency"`
	// Command optionally addresses a display label for what runs
	Command Path `json:"command" yaml:"command"`
	// Dialect names the grammar the expressions are written in
	Dialect string `json:"dialect" yaml:"dialect"`
	// Shell reports whether Command is a real shell command
	Shell bool `json:"shell" yaml:"shell"`
}

// profileSource is the engine interpreting any Profile.
type profileSource struct {
	profile Profile
	// parser validates expressions; built once here since cronx's parser caches and is safe for concurrent use.
	parser cronx.Parser
}

// NewProfileSource validates a profile, since a malformed path could otherwise address the wrong node.
func NewProfileSource(p Profile) (Source, error) {
	if p.ID == "" {
		return nil, fmt.Errorf("profile has no id")
	}
	if len(p.Extensions) == 0 {
		return nil, fmt.Errorf("profile %q has no extensions", p.ID)
	}
	for _, ext := range p.Extensions {
		if !strings.HasPrefix(ext, ".") || ext == "." {
			return nil, fmt.Errorf("profile %q extension %q must start with a . and have a suffix after it", p.ID, ext)
		}
	}
	if strings.HasPrefix(p.DirPrefix, "/") || strings.HasSuffix(p.DirPrefix, "/") {
		return nil, fmt.Errorf("profile %q dir prefix %q must not have a leading or trailing slash", p.ID, p.DirPrefix)
	}
	if len(p.Schedules) == 0 {
		return nil, fmt.Errorf("profile %q needs at least one schedule path", p.ID)
	}

	for _, schedule := range p.Schedules {
		if err := schedule.Validate(); err != nil {
			return nil, fmt.Errorf("profile %q schedule path: %w", p.ID, err)
		}
	}
	for _, m := range p.Match {
		if err := m.Path.Validate(); err != nil {
			return nil, fmt.Errorf("profile %q match path: %w", p.ID, err)
		}
	}
	optional := []struct {
		name string
		path Path
	}{
		{"timezone", p.Timezone},
		{"suspend", p.Suspend},
		{"concurrency", p.Concurrency},
		{"command", p.Command},
	}
	for _, o := range optional {
		if o.path == "" {
			continue
		}
		if err := o.path.Validate(); err != nil {
			return nil, fmt.Errorf("profile %q %s path: %w", p.ID, o.name, err)
		}
	}

	return &profileSource{profile: p, parser: cronx.NewParser()}, nil
}

// ID names the source.
func (s *profileSource) ID() string { return s.profile.ID }

// Match tests the path only, so a scan skips most files without opening them.
func (s *profileSource) Match(u Unit) bool {
	if s.profile.DirPrefix != "" && path.Dir(u.Path) != s.profile.DirPrefix {
		return false
	}
	ext := path.Ext(u.Path)
	for _, want := range s.profile.Extensions {
		if strings.EqualFold(ext, want) {
			return true
		}
	}
	return false
}

// Extract decodes the unit and returns one item per schedule it holds.
func (s *profileSource) Extract(u Unit, fsys fs.FS) ([]inventory.Item, error) {
	docs, err := u.Documents(fsys)
	if err != nil {
		return nil, err
	}

	var items []inventory.Item
	for _, doc := range docs {
		if !s.matches(doc.Root) {
			continue
		}
		items = append(items, s.itemsFrom(u, doc)...)
	}
	return items, nil
}

// matches reports whether every FieldMatch holds for the document.
func (s *profileSource) matches(root *Node) bool {
	for _, m := range s.profile.Match {
		found := m.Path.Resolve(root)
		if len(found) != 1 || found[0].Node.Value != m.Equals {
			return false
		}
	}
	return true
}

// itemsFrom builds one item per schedule found in the document.
func (s *profileSource) itemsFrom(u Unit, doc Document) []inventory.Item {
	timezone := s.profile.TimezoneFixed
	if v, ok := first(doc.Root, s.profile.Timezone); ok {
		timezone = v
	}

	command, _ := first(doc.Root, s.profile.Command)

	concurrency := inventory.ConcurrencyUnspecified
	if v, ok := first(doc.Root, s.profile.Concurrency); ok {
		if parsed := inventory.ConcurrencyFromString(v); parsed != -1 {
			concurrency = parsed
		}
	}

	suspended := false
	suspendTemplated := false
	if v, ok := first(doc.Root, s.profile.Suspend); ok {
		switch {
		case isTemplated(v):
			suspendTemplated = true
		case isTruthy(v):
			suspended = true
		}
	}

	// Schedules are tried in order and the first path with any matches wins, avoiding double-reporting when a
	// document carries both old and new field shapes.
	var resolved []Located
	for _, schedule := range s.profile.Schedules {
		if found := schedule.Resolve(doc.Root); len(found) > 0 {
			resolved = found
			break
		}
	}

	var items []inventory.Item
	for _, located := range resolved {
		expression := located.Node.Value
		itemTimezone := timezone

		// A schedule's own leading CRON_TZ=/TZ= prefix is lifted out of Expression into itemTimezone and
		// validated here; an explicit profile timezone field takes precedence when already set.
		var zoneErr error
		if zone, rest, ok := cronx.SplitTZPrefix(expression); ok {
			if rest == "" {
				// A bare "CRON_TZ=Asia/Tokyo" is a timezone assignment with no schedule, so the unsplit
				// expression is parsed to surface cronx's specific message for that shape.
				_, zoneErr = s.parser.Parse(expression)
			} else if _, err := time.LoadLocation(zone); err != nil {
				zoneErr = err
			} else {
				expression = rest
				if itemTimezone == "" {
					itemTimezone = zone
				}
			}
		}

		item := inventory.Item{
			Expression:  expression,
			SourceID:    s.profile.ID,
			Dialect:     s.profile.Dialect,
			Command:     command,
			Shell:       s.profile.Shell,
			Timezone:    itemTimezone,
			State:       inventory.StateActive,
			Concurrency: concurrency,
			Locator: inventory.Locator{
				File:     u.Path,
				Line:     located.Node.Line,
				Document: doc.Index,
				Path:     located.Path,
			},
		}

		switch {
		case suspendTemplated:
			item.State = inventory.StateUnresolved
			item.Reason = "the suspend flag is templated and cannot be read"
		case suspended:
			item.State = inventory.StateSuspended
			item.Reason = "the schedule is suspended"
		case isTemplated(item.Expression):
			item.State = inventory.StateUnresolved
			item.Reason = "the expression is templated and cannot be read"
		case zoneErr != nil:
			item.State = inventory.StateInvalid
			item.Reason = zoneErr.Error()
		default:
			// Only an expression that would otherwise be reported active is worth validating.
			if _, err := s.parser.Parse(item.Expression); err != nil {
				item.State = inventory.StateInvalid
				item.Reason = err.Error()
			}
		}

		items = append(items, item)
	}
	return items
}

// first returns the value of the first node a path addresses; an empty path means the field was omitted.
func first(root *Node, p Path) (string, bool) {
	if p == "" {
		return "", false
	}
	found := p.Resolve(root)
	if len(found) == 0 {
		return "", false
	}
	return found[0].Node.Value, true
}

// isTemplated reports whether an expression still holds templating syntax.
func isTemplated(expression string) bool {
	for _, marker := range templateMarkers {
		if strings.Contains(expression, marker) {
			return true
		}
	}
	return false
}

// isTruthy reports whether a raw YAML scalar spells a boolean true, case-insensitively.
func isTruthy(v string) bool {
	switch strings.ToLower(v) {
	case "true", "yes", "on":
		return true
	default:
		return false
	}
}
