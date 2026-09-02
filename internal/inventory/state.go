package inventory

import (
	"encoding/json"
	"fmt"
	"strings"
)

// State describes whether a discovered schedule actually runs.
type State int

const (
	// StateActive means the schedule is live and parseable
	StateActive State = iota
	// StateSuspended means the platform is configured not to run it
	StateSuspended
	// StateUnresolved means the expression is templated and cannot be read
	StateUnresolved
	// StateInvalid means the expression is not valid for its dialect
	StateInvalid
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateActive:
		return "active"
	case StateSuspended:
		return "suspended"
	case StateUnresolved:
		return "unresolved"
	case StateInvalid:
		return "invalid"
	default:
		return "unknown"
	}
}

// MarshalJSON implements json.Marshaler for State
func (s State) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON implements json.Unmarshaler for State
func (s *State) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	parsed := StateFromString(str)
	if parsed == -1 {
		return fmt.Errorf("invalid state: %s", str)
	}

	*s = parsed
	return nil
}

// StateFromString converts a string to a State value
// Returns -1 if the string is invalid
func StateFromString(s string) State {
	switch s {
	case "active":
		return StateActive
	case "suspended":
		return StateSuspended
	case "unresolved":
		return StateUnresolved
	case "invalid":
		return StateInvalid
	default:
		return -1
	}
}

// Concurrency describes what the platform does when a run is still going and
// the next one is due.
type Concurrency int

const (
	// ConcurrencyUnspecified means the source does not express a policy
	ConcurrencyUnspecified Concurrency = iota
	// ConcurrencyAllow means overlapping runs are permitted
	ConcurrencyAllow
	// ConcurrencyForbid means the platform serialises runs
	ConcurrencyForbid
	// ConcurrencyReplace means a new run cancels the one in flight
	ConcurrencyReplace
)

// String returns the string representation of the concurrency policy
func (c Concurrency) String() string {
	switch c {
	case ConcurrencyUnspecified:
		return ""
	case ConcurrencyAllow:
		return "allow"
	case ConcurrencyForbid:
		return "forbid"
	case ConcurrencyReplace:
		return "replace"
	default:
		return "unknown"
	}
}

// MarshalJSON implements json.Marshaler for Concurrency
func (c Concurrency) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

// UnmarshalJSON implements json.Unmarshaler for Concurrency
func (c *Concurrency) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	parsed := ConcurrencyFromString(str)
	if parsed == -1 {
		return fmt.Errorf("invalid concurrency policy: %s", str)
	}

	*c = parsed
	return nil
}

// ConcurrencyFromString converts a string, accepting Kubernetes/Argo capitalised spellings, or -1 if invalid.
func ConcurrencyFromString(s string) Concurrency {
	switch strings.ToLower(s) {
	case "":
		return ConcurrencyUnspecified
	case "allow":
		return ConcurrencyAllow
	case "forbid":
		return ConcurrencyForbid
	case "replace":
		return ConcurrencyReplace
	default:
		return -1
	}
}
