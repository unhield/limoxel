package engine

import "fmt"

// State represents the operational lifecycle state of the Engine coordinator.
type State int

const (
	// StateCreated indicates the engine instance has been instantiated.
	StateCreated State = iota

	// StateConfigured indicates the engine configuration has been validated and frozen.
	StateConfigured

	// StateReady indicates all required subsystem components have been registered and validated.
	StateReady

	// StateRunning indicates the engine coordinator is actively running.
	StateRunning

	// StateStopped indicates the engine coordinator has stopped execution.
	StateStopped

	// StateTerminated indicates the engine coordinator has terminated and released all references.
	StateTerminated
)

// String returns the human-readable textual representation of the State.
func (s State) String() string {
	switch s {
	case StateCreated:
		return "CREATED"
	case StateConfigured:
		return "CONFIGURED"
	case StateReady:
		return "READY"
	case StateRunning:
		return "RUNNING"
	case StateStopped:
		return "STOPPED"
	case StateTerminated:
		return "TERMINATED"
	default:
		return fmt.Sprintf("UNKNOWN_STATE(%d)", int(s))
	}
}
