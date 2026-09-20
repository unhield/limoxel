package runtime

import (
	"context"
	"fmt"
	"time"
)

// RuntimeState defines the deterministic lifecycle states of an execution runtime.
type RuntimeState string

const (
	// StateCreated indicates the runtime has been instantiated but not yet started.
	StateCreated RuntimeState = "Created"
	// StateStarting indicates the runtime process or environment is initializing.
	StateStarting RuntimeState = "Starting"
	// StateRunning indicates the runtime is active, healthy, and ready for execution.
	StateRunning RuntimeState = "Running"
	// StateStopping indicates graceful termination has been requested.
	StateStopping RuntimeState = "Stopping"
	// StateStopped indicates the runtime has exited cleanly and all resources are released.
	StateStopped RuntimeState = "Stopped"
	// StateFailed indicates an unexpected crash, timeout, or unrecoverable error occurred.
	StateFailed RuntimeState = "Failed"
)

// IsTerminal returns true if the state represents an ended runtime.
func (s RuntimeState) IsTerminal() bool {
	return s == StateStopped || s == StateFailed
}

// RuntimeStatus encapsulates observable telemetry and state of an active runtime.
type RuntimeStatus struct {
	State           RuntimeState      `json:"state"`
	SessionID       string            `json:"session_id"`
	ProcessID       int               `json:"process_id"`
	ProtocolVersion string            `json:"protocol_version"`
	Uptime          time.Duration     `json:"uptime"`
	Error           string            `json:"error,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// PluginRuntime defines the technology-agnostic execution boundary for a plugin.
type PluginRuntime interface {
	// Start starts the underlying execution environment and establishes communication.
	Start(ctx context.Context) error

	// Stop gracefully terminates execution within the specified timeout.
	Stop(ctx context.Context) error

	// Restart stops the runtime and restarts a fresh execution session.
	Restart(ctx context.Context) error

	// Status queries current telemetry, state, and health of the runtime.
	Status(ctx context.Context) (RuntimeStatus, error)

	// Invoke dispatches an execution request to the plugin and returns the response payload.
	Invoke(ctx context.Context, method string, req []byte) ([]byte, error)

	// Close releases all underlying resources, terminating processes if still active.
	Close() error

	// SessionID returns the unique identifier for the current runtime execution session.
	SessionID() string

	// ProcessID returns the operating system process ID, or 0 if in-process.
	ProcessID() int

	// State returns the current lifecycle state of the runtime.
	State() RuntimeState
}

// StateTransitionError reports illegal runtime lifecycle transitions.
type StateTransitionError struct {
	From RuntimeState
	To   RuntimeState
	Msg  string
}

func (e *StateTransitionError) Error() string {
	return fmt.Sprintf("invalid runtime state transition from %s to %s: %s", e.From, e.To, e.Msg)
}
