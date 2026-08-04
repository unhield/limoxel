package language

import "fmt"

// State represents the operational lifecycle state of the Language Registry.
type State int

const (
	// StateCreated indicates the registry is active and accepting language registrations.
	StateCreated State = iota

	// StateOperational indicates the registry is frozen, read-only, and operational.
	StateOperational

	// StateTerminated indicates the registry has been shutdown and is no longer accessible.
	StateTerminated
)

// String returns the human-readable textual representation of the State.
func (s State) String() string {
	switch s {
	case StateCreated:
		return "CREATED"
	case StateOperational:
		return "OPERATIONAL"
	case StateTerminated:
		return "TERMINATED"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int(s))
	}
}
