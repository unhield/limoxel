package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HealthStatus defines the health condition of the Runtime.
type HealthStatus string

const (
	// HealthHealthy indicates the Runtime is operating normally.
	HealthHealthy HealthStatus = "HEALTHY"

	// HealthDegraded indicates the Runtime is in transition or shutting down.
	HealthDegraded HealthStatus = "DEGRADED"

	// HealthUnhealthy indicates the Runtime encountered a critical issue.
	HealthUnhealthy HealthStatus = "UNHEALTHY"

	// HealthTerminated indicates the Runtime has been terminated.
	HealthTerminated HealthStatus = "TERMINATED"
)

// Health describes the operational health and lifecycle metric of the Runtime.
type Health struct {
	Status    HealthStatus  `json:"status"`
	State     State         `json:"state"`
	Uptime    time.Duration `json:"uptime"`
	Timestamp time.Time     `json:"timestamp"`
}

// Runtime is the permanent execution coordinator for Limoxel.
type Runtime struct {
	mu        sync.RWMutex
	state     *StateManager
	lifecycle *LifecycleCoordinator
}

// New creates and returns a new Runtime instance in the StateCreated state.
func New() (*Runtime, error) {
	stateMgr := NewStateManager()
	coordinator, err := NewLifecycleCoordinator(stateMgr)
	if err != nil {
		return nil, fmt.Errorf("failed to create runtime lifecycle coordinator: %w", err)
	}

	return &Runtime{
		state:     stateMgr,
		lifecycle: coordinator,
	}, nil
}

// Validate verifies the Runtime configuration and readiness before startup.
func (r *Runtime) Validate() error {
	if r == nil {
		return ErrRuntimeNil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.state == nil {
		return fmt.Errorf("%w: state manager is nil", ErrValidationFailed)
	}

	if r.lifecycle == nil {
		return fmt.Errorf("%w: lifecycle coordinator is nil", ErrValidationFailed)
	}

	return nil
}

// Initialize performs the deterministic initialization phase of the Runtime.
func (r *Runtime) Initialize(ctx context.Context) error {
	if err := r.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	return r.lifecycle.Initialize(ctx)
}

// Prepare performs component readiness coordination prior to operational execution.
func (r *Runtime) Prepare(ctx context.Context) error {
	if err := r.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	return r.lifecycle.Prepare(ctx)
}

// Start initiates platform operational execution under Runtime supervision.
func (r *Runtime) Start(ctx context.Context) error {
	if err := r.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	return r.lifecycle.Start(ctx)
}

// Shutdown performs orderly cleanup and brings the Runtime to StateTerminated.
func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil {
		return ErrRuntimeNil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.lifecycle == nil {
		return nil
	}

	return r.lifecycle.Shutdown(ctx)
}

// State returns the current operational State of the Runtime.
func (r *Runtime) State() State {
	if r == nil || r.state == nil {
		return StateTerminated
	}
	return r.state.Current()
}

// IsRunning reports whether the Runtime is currently in StateRunning.
func (r *Runtime) IsRunning() bool {
	return r.State() == StateRunning
}

// Health evaluates and returns the current Health metrics of the Runtime.
func (r *Runtime) Health() Health {
	now := time.Now()

	if r == nil || r.state == nil {
		return Health{
			Status:    HealthTerminated,
			State:     StateTerminated,
			Uptime:    0,
			Timestamp: now,
		}
	}

	currentState := r.state.Current()
	var status HealthStatus
	var uptime time.Duration

	switch currentState {
	case StateRunning:
		status = HealthHealthy
		if r.lifecycle != nil && !r.lifecycle.StartTime().IsZero() {
			uptime = now.Sub(r.lifecycle.StartTime())
		}
	case StateCreated, StateInitializing, StateInitialized, StatePreparing, StatePrepared, StateShuttingDown:
		status = HealthDegraded
	case StateTerminated:
		status = HealthTerminated
	default:
		status = HealthUnhealthy
	}

	return Health{
		Status:    status,
		State:     currentState,
		Uptime:    uptime,
		Timestamp: now,
	}
}
