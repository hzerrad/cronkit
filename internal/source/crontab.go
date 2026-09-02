package source

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"time"
	"unicode"

	"github.com/hzerrad/cronkit/internal/crontab"
	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/inventory"
)

// crontabNames are the exact file names that are crontabs by convention.
var crontabNames = []string{"crontab"}

// crontabExtensions are the suffixes that mark a crontab.
var crontabExtensions = []string{".cron", ".crontab"}

// crontabDirs are directories whose every file is a crontab fragment.
var crontabDirs = []string{"cron.d"}

// utf8BOM is the UTF-8 encoding of U+FEFF, which breaks parsing if left in place.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// crontabSource reads line-oriented crontabs, which are not tree-shaped so no path language can address them.
type crontabSource struct{}

// NewCrontabSource creates the crontab source.
func NewCrontabSource() Source {
	return &crontabSource{}
}

// ID names the source.
func (s *crontabSource) ID() string { return "crontab" }

// Match tests the path only — never a content sniff, which would defeat cheap skipping.
func (s *crontabSource) Match(u Unit) bool {
	base := path.Base(u.Path)

	for _, name := range crontabNames {
		if strings.EqualFold(base, name) {
			return true
		}
	}
	for _, ext := range crontabExtensions {
		if strings.EqualFold(path.Ext(base), ext) {
			return true
		}
	}
	for _, dir := range crontabDirs {
		if strings.EqualFold(path.Base(path.Dir(u.Path)), dir) {
			return true
		}
	}
	return false
}

// isSystemFormat reports whether p sits under a cron.d-style directory using the system crontab format.
func isSystemFormat(p string) bool {
	dir := path.Base(path.Dir(p))
	for _, d := range crontabDirs {
		if strings.EqualFold(dir, d) {
			return true
		}
	}
	return false
}

// looksLikeUser reports whether token has the shape of a bare system account name.
func looksLikeUser(token string) bool {
	if token == "" || strings.HasPrefix(token, "-") {
		return false
	}
	return !strings.ContainsAny(token, "/=\"'")
}

// splitSystemUser splits a system-format command into its USER field and the command a shell would run.
func splitSystemUser(command string) (user, rest, reason string) {
	first := command
	if i := strings.IndexFunc(command, unicode.IsSpace); i != -1 {
		first = command[:i]
		rest = strings.TrimSpace(command[i:])
	}
	if !looksLikeUser(first) {
		return "", "", "cron.d entry is missing the user field: expected \"minute hour dom month dow user command\""
	}
	if rest == "" {
		return "", "", "cron.d entry has a user but no command: expected \"minute hour dom month dow user command\""
	}
	return first, rest, ""
}

// classifySchedule reports whether expression is valid, and if not, whether to report it as StateInvalid.
func classifySchedule(expression string) (valid, reportable bool) {
	_, err := cronx.NewParser().Parse(expression)
	if err == nil {
		return true, false
	}
	if fields := strings.Fields(expression); len(fields) > 0 && strings.HasPrefix(fields[0], "@") && fields[0] != "@" {
		return false, true
	}
	if strings.Contains(err.Error(), "value out of range") {
		return false, true
	}
	return false, oneFieldAwayFromValid(expression)
}

// oneFieldAwayFromValid reports whether expression would parse if exactly one field were replaced with "*".
func oneFieldAwayFromValid(expression string) bool {
	fields := strings.Fields(expression)
	if len(fields) != 5 {
		return false
	}

	parser := cronx.NewParser()
	for i, field := range fields {
		if field == "*" {
			continue // substituting a field already "*" changes nothing
		}
		trial := make([]string, len(fields))
		copy(trial, fields)
		trial[i] = "*"
		if _, err := parser.Parse(strings.Join(trial, " ")); err == nil {
			return true
		}
	}
	return false
}

// malformedDescriptorItem reports whether raw still attempts an "@" descriptor as StateInvalid.
func malformedDescriptorItem(sourceID, file string, lineNumber int, raw, timezone string, systemFormat bool) (inventory.Item, bool) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "@") {
		return inventory.Item{}, false
	}

	token := strings.Fields(trimmed)[0]
	_, err := cronx.NewParser().Parse(token)
	if err == nil {
		return inventory.Item{}, false
	}

	rest := strings.TrimSpace(trimmed[len(token):])
	command, comment := rest, ""
	if idx := strings.Index(rest, "#"); idx != -1 {
		command = strings.TrimSpace(rest[:idx])
		comment = strings.TrimSpace(rest[idx+1:])
	}

	runAs := ""
	if systemFormat {
		if user, cmdRest, reason := splitSystemUser(command); reason == "" {
			runAs, command = user, cmdRest
		}
	}

	return inventory.Item{
		Expression: token,
		SourceID:   sourceID,
		Dialect:    "vixie",
		Command:    command,
		Shell:      true,
		Comment:    comment,
		RunAs:      runAs,
		Timezone:   timezone,
		State:      inventory.StateInvalid,
		Reason:     err.Error(),
		Locator: inventory.Locator{
			File: file,
			Line: lineNumber,
		},
	}, true
}

// parseTZAssignment reports whether raw is a CRON_TZ or TZ assignment, returning key, value, and remainder.
func parseTZAssignment(raw string) (key, value, remainder string, ok bool) {
	trimmed := strings.TrimSpace(raw)
	eq := strings.Index(trimmed, "=")
	if eq == -1 {
		return "", "", "", false
	}
	key = trimmed[:eq]
	if key != "CRON_TZ" && key != "TZ" {
		return "", "", "", false
	}

	rest := strings.TrimLeft(trimmed[eq+1:], " \t")
	value = rest
	if i := strings.IndexFunc(rest, unicode.IsSpace); i != -1 {
		value = rest[:i]
		remainder = strings.TrimSpace(rest[i:])
	}
	if strings.HasPrefix(remainder, "#") {
		remainder = ""
	}
	// Quote-stripping lives in cronx, shared with SplitTZPrefix, so a quoted zone reads the same everywhere.
	return key, cronx.StripQuotes(value), remainder, true
}

// forEachLine walks data as crontab lines, invoking fn per line, avoiding bufio.Scanner's 64 KiB token cap.
func forEachLine(data []byte, fn func(lineNumber int, line string)) {
	if len(data) == 0 {
		return
	}
	data = bytes.TrimSuffix(data, []byte("\n"))

	for lineNumber := 1; ; lineNumber++ {
		idx := bytes.IndexByte(data, '\n')
		line := data
		if idx != -1 {
			line = data[:idx]
		}
		line = bytes.TrimSuffix(line, []byte("\r"))
		fn(lineNumber, string(line))

		if idx == -1 {
			return
		}
		data = data[idx+1:]
	}
}

// Extract parses the file line by line; an unparseable job is inventoried as invalid, not dropped.
func (s *crontabSource) Extract(u Unit, fsys fs.FS) ([]inventory.Item, error) {
	if err := checkUnitSize(u, fsys); err != nil {
		return nil, err
	}

	data, err := fs.ReadFile(fsys, u.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", u.Path, err)
	}
	data = bytes.TrimPrefix(data, utf8BOM)

	systemFormat := isSystemFormat(u.Path)

	// cronTZ and tz track the most recent assignments; CRON_TZ takes precedence over TZ when both are set.
	var cronTZ, tz string

	var items []inventory.Item
	forEachLine(data, func(lineNumber int, line string) {
		entry := crontab.ParseLine(line, lineNumber)

		if entry.Type == crontab.EntryTypeEnvVar {
			key, value, remainder, ok := parseTZAssignment(entry.Raw)
			if !ok {
				return
			}
			// A rejected zone is reported as StateInvalid; cronTZ/tz stay untouched so it can't poison later jobs.
			if _, err := time.LoadLocation(value); err != nil {
				items = append(items, inventory.Item{
					Expression: key + "=" + value,
					SourceID:   s.ID(),
					Dialect:    "vixie",
					State:      inventory.StateInvalid,
					Reason:     err.Error(),
					Locator: inventory.Locator{
						File: u.Path,
						Line: lineNumber,
					},
				})
			} else {
				switch key {
				case "CRON_TZ":
					cronTZ = value
				case "TZ":
					tz = value
				}
			}
			if remainder == "" {
				return
			}
			// An assignment can be followed by a job on the same line, so re-parse remainder as its own line.
			entry = crontab.ParseLine(remainder, lineNumber)
		}

		timezone := tz
		if cronTZ != "" {
			timezone = cronTZ
		}

		if entry.Type != crontab.EntryTypeJob || entry.Job == nil {
			// Lines that never become a Job may still be a malformed "@" descriptor worth reporting.
			if item, ok := malformedDescriptorItem(s.ID(), u.Path, lineNumber, entry.Raw, timezone, systemFormat); ok {
				items = append(items, item)
			}
			return
		}
		valid, reportable := classifySchedule(entry.Job.Expression)
		if !valid && !reportable {
			return
		}

		command := entry.Job.Command
		runAs := ""
		splitReason := ""
		if systemFormat {
			if user, rest, reason := splitSystemUser(command); reason == "" {
				runAs, command = user, rest
			} else {
				splitReason = reason
			}
		}

		item := inventory.Item{
			Expression: entry.Job.Expression,
			SourceID:   s.ID(),
			Dialect:    "vixie",
			Command:    command,
			Shell:      true,
			Comment:    entry.Job.Comment,
			RunAs:      runAs,
			Timezone:   timezone,
			State:      inventory.StateActive,
			Locator: inventory.Locator{
				File: u.Path,
				Line: entry.Job.LineNumber,
			},
		}
		switch {
		case !entry.Job.Valid:
			item.State = inventory.StateInvalid
			item.Reason = entry.Job.Error
		case splitReason != "":
			// splitSystemUser refused the split; Command stays as the original text, not misattributed to RunAs.
			item.State = inventory.StateInvalid
			item.Reason = splitReason
		}

		items = append(items, item)
	})

	return items, nil
}
