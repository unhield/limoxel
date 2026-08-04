package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// LifecycleCoordinator manages the deterministic lifecycle transitions of a Runtime.
type LifecycleCoordinator struct {
	mu        sync.Mutex
	state     *StateManager
	startTime time.Time
}

// NewLifecycleCoordinator initializes a new LifecycleCoordinator with the given StateManager.
func NewLifecycleCoordinator(state *StateManager) (*LifecycleCoordinator, error) {
	if state == nil {
		return nil, ErrStateNil
	}
	return &LifecycleCoordinator{
		state: state,
	}, nil
}

// Initialize transitions the Runtime from Created -> Initializing -> Initialized.
func (lc *LifecycleCoordinator) Initialize(ctx context.Context) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrContextCanceled, err)
	}

	if err := lc.state.TransitionTo(StateInitializing); err != nil {
		return err
	}

	if err := lc.state.TransitionTo(StateInitialized); err != nil {
		return err
	}

	return nil
}

// Prepare transitions the Runtime from Initialized -> Preparing -> Prepared.
func (lc *LifecycleCoordinator) Prepare(ctx context.Context) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrContextCanceled, err)
	}

	if err := lc.state.TransitionTo(StatePreparing); err != nil {
		return err
	}

	if err := lc.state.TransitionTo(StatePrepared); err != nil {
		return err
	}

	return nil
}

// Start transitions the Runtime from Prepared -> Running.
func (lc *LifecycleCoordinator) Start(ctx context.Context) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrContextCanceled, err)
	}

	if lc.state.Is(StateRunning) {
		return ErrAlreadyRunning
	}

	if lc.state.Is(StateTerminated) {
		return ErrAlreadyTerminated
	}

	if err := lc.state.TransitionTo(StateRunning); err != nil {
		return fmt.Errorf("%w: %v", ErrStartupFailed, err)
	}

	lc.startTime = time.Now()
	return nil
}

// Shutdown transitions the Runtime from Running -> ShuttingDown -> Terminated.
func (lc *LifecycleCoordinator) Shutdown(ctx context.Context) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	current := lc.state.Current()
	if current == StateTerminated {
		return nil
	}

	if err := lc.state.TransitionTo(StateShuttingDown); err != nil {
		return fmt.Errorf("%w: %v", ErrShutdownFailed, err)
	}

	if err := lc.state.TransitionTo(StateTerminated); err != nil {
		return fmt.Errorf("%w: %v", ErrShutdownFailed, err)
	}

	return nil
}

// StartTime returns the timestamp when the Runtime entered StateRunning.
func (lc *LifecycleCoordinator) StartTime() time.Time {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.startTime
}
