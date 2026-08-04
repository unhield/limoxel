package extension

import "fmt"

// State represents the operational lifecycle state of a registered extension descriptor.
type State int

const (
	// StateRegistered indicates the extension descriptor is registered but not active.
	StateRegistered State = iota

	// StateActive indicates the extension is active and available.
	StateActive

	// StateInactive indicates the extension is registered but deactivated.
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
		return fmt.Sprintf("UNKNOWN_STATE(%d)", int(s))
	}
}
