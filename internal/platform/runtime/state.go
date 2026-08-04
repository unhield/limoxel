package runtime

import (
	"fmt"
	"sync"
)

// State represents an operational state within the Runtime lifecycle.
type State uint8

const (
	// StateCreated indicates the Runtime has been created but not initialized.
	StateCreated State = iota

	// StateInitializing indicates the Runtime is undergoing initialization.
	StateInitializing

	// StateInitialized indicates the Runtime has successfully completed initialization.
	StateInitialized

	// StatePreparing indicates the Runtime is preparing component readiness.
	StatePreparing

	// StatePrepared indicates the Runtime is prepared and ready for execution.
	StatePrepared

	// StateRunning indicates the Runtime is actively coordinating execution.
	StateRunning

	// StateShuttingDown indicates the Runtime is performing orderly shutdown.
	StateShuttingDown

	// StateTerminated indicates the Runtime has completed shutdown and terminated.
	StateTerminated
)

// String returns the human-readable string representation of the State.
func (s State) String() string {
	switch s {
	case StateCreated:
		return "CREATED"
	case StateInitializing:
		return "INITIALIZING"
	case StateInitialized:
		return "INITIALIZED"
	case StatePreparing:
		return "PREPARING"
	case StatePrepared:
		return "PREPARED"
	case StateRunning:
		return "RUNNING"
	case StateShuttingDown:
		return "SHUTTING_DOWN"
	case StateTerminated:
		return "TERMINATED"
	default:
		return fmt.Sprintf("UNKNOWN_STATE(%d)", uint8(s))
	}
}

// StateManager manages thread-safe Runtime state transitions and queries.
type StateManager struct {
	mu      sync.RWMutex
	current State
}

// NewStateManager creates a new StateManager initialized to StateCreated.
func NewStateManager() *StateManager {
	return &StateManager{
		current: StateCreated,
	}
}

// Current returns the current operational State of the Runtime.
func (m *StateManager) Current() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// Is returns true if the current State matches the target State.
func (m *StateManager) Is(target State) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current == target
}

// IsRunning returns true if the current state is StateRunning.
func (m *StateManager) IsRunning() bool {
	return m.Is(StateRunning)
}

// IsTerminated returns true if the current state is StateTerminated.
func (m *StateManager) IsTerminated() bool {
	return m.Is(StateTerminated)
}

// CanTransitionTo reports whether a transition from the current state to target is valid.
func (m *StateManager) CanTransitionTo(target State) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.canTransition(m.current, target)
}

// TransitionTo attempts to transition the StateManager to the target State.
func (m *StateManager) TransitionTo(target State) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.canTransition(m.current, target) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStateTransition, m.current, target)
	}

	m.current = target
	return nil
}

// canTransition enforces deterministic state transition rules.
func (m *StateManager) canTransition(from, to State) bool {
	if from == to {
		return false
	}
	if from == StateTerminated {
		return false
	}

	switch from {
	case StateCreated:
		return to == StateInitializing || to == StateShuttingDown || to == StateTerminated
	case StateInitializing:
		return to == StateInitialized || to == StateShuttingDown || to == StateTerminated
	case StateInitialized:
		return to == StatePreparing || to == StateShuttingDown || to == StateTerminated
	case StatePreparing:
		return to == StatePrepared || to == StateShuttingDown || to == StateTerminated
	case StatePrepared:
		return to == StateRunning || to == StateShuttingDown || to == StateTerminated
	case StateRunning:
		return to == StateShuttingDown
	case StateShuttingDown:
		return to == StateTerminated
	default:
		return false
	}
}
