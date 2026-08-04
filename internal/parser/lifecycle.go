package parser

import "fmt"

// State represents the operational lifecycle state of a registered parser descriptor.
type State int

const (
	// StateRegistered indicates the parser descriptor is registered but not active.
	StateRegistered State = iota

	// StateActive indicates the parser is active and available.
	StateActive

	// StateInactive indicates the parser is registered but deactivated.
	StateInactive
)

// String returns the human-readable textual representation of the State.
func (s State) String() string {
	switch s {
	case StateRegistered:
		return "REGISTERED"
	case StateActive:
		return "ACTIVE"
	case StateInactive:
		return "INACTIVE"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int(s))
	}
}
