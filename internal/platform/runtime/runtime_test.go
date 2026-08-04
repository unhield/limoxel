package runtime_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/platform/runtime"
)

func TestStateAndStateManager(t *testing.T) {
	t.Run("State String representation", func(t *testing.T) {
		states := map[runtime.State]string{
			runtime.StateCreated:      "CREATED",
			runtime.StateInitializing: "INITIALIZING",
			runtime.StateInitialized:  "INITIALIZED",
			runtime.StatePreparing:    "PREPARING",
			runtime.StatePrepared:     "PREPARED",
			runtime.StateRunning:      "RUNNING",
			runtime.StateShuttingDown: "SHUTTING_DOWN",
			runtime.StateTerminated:   "TERMINATED",
			runtime.State(255):        "UNKNOWN_STATE(255)",
		}
		for st, expected := range states {
			if st.String() != expected {
				t.Errorf("got %q, want %q", st.String(), expected)
			}
		}
	})

	t.Run("StateManager lifecycle transitions", func(t *testing.T) {
		sm := runtime.NewStateManager()
		if !sm.Is(runtime.StateCreated) {
			t.Errorf("expected initial state StateCreated, got %v", sm.Current())
		}

		// Valid sequential transitions
		transitions := []runtime.State{
			runtime.StateInitializing,
			runtime.StateInitialized,
			runtime.StatePreparing,
			runtime.StatePrepared,
			runtime.StateRunning,
			runtime.StateShuttingDown,
			runtime.StateTerminated,
		}

		for _, next := range transitions {
			if !sm.CanTransitionTo(next) {
				t.Fatalf("expected CanTransitionTo(%v) to be true from %v", next, sm.Current())
			}
			if err := sm.TransitionTo(next); err != nil {
				t.Fatalf("unexpected transition error: %v", err)
			}
			if !sm.Is(next) {
				t.Fatalf("expected state %v, got %v", next, sm.Current())
			}
		}

		// Transition from Terminated should fail
		if sm.CanTransitionTo(runtime.StateCreated) {
			t.Error("should not be able to transition from Terminated")
		}
		if err := sm.TransitionTo(runtime.StateRunning); !errors.Is(err, runtime.ErrInvalidStateTransition) {
			t.Errorf("got error %v, want ErrInvalidStateTransition", err)
		}
	})

	t.Run("StateManager illegal transition", func(t *testing.T) {
		sm := runtime.NewStateManager()
		// Cannot jump directly from Created to Running
		if sm.CanTransitionTo(runtime.StateRunning) {
			t.Error("should not be able to transition directly from Created to Running")
		}
		err := sm.TransitionTo(runtime.StateRunning)
		if !errors.Is(err, runtime.ErrInvalidStateTransition) {
			t.Errorf("got %v, want ErrInvalidStateTransition", err)
		}
	})
}

func TestRuntimeFullLifecycle(t *testing.T) {
	r, err := runtime.New()
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}

	ctx := context.Background()

	// Initial checks
	if err := r.Validate(); err != nil {
		t.Errorf("Validate failed: %v", err)
	}
	if r.State() != runtime.StateCreated {
		t.Errorf("got state %v, want Created", r.State())
	}
	if r.IsRunning() {
		t.Error("IsRunning should be false")
	}

	health := r.Health()
	if health.Status != runtime.HealthDegraded {
		t.Errorf("got health status %v, want Degraded", health.Status)
	}

	// Initialize
	if err := r.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if r.State() != runtime.StateInitialized {
		t.Errorf("got state %v, want Initialized", r.State())
	}

	// Prepare
	if err := r.Prepare(ctx); err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}
	if r.State() != runtime.StatePrepared {
		t.Errorf("got state %v, want Prepared", r.State())
	}

	// Start
	if err := r.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if r.State() != runtime.StateRunning {
		t.Errorf("got state %v, want Running", r.State())
	}
	if !r.IsRunning() {
		t.Error("IsRunning should be true")
	}

	healthRunning := r.Health()
	if healthRunning.Status != runtime.HealthHealthy {
		t.Errorf("got health status %v, want Healthy", healthRunning.Status)
	}
	time.Sleep(2 * time.Millisecond)
	if r.Health().Uptime <= 0 {
		t.Error("expected non-zero uptime while running")
	}

	// Shutdown
	if err := r.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
	if r.State() != runtime.StateTerminated {
		t.Errorf("got state %v, want Terminated", r.State())
	}

	healthTerminated := r.Health()
	if healthTerminated.Status != runtime.HealthTerminated {
		t.Errorf("got health status %v, want Terminated", healthTerminated.Status)
	}
}

func TestNilRuntimeSafety(t *testing.T) {
	var r *runtime.Runtime
	if err := r.Validate(); !errors.Is(err, runtime.ErrRuntimeNil) {
		t.Errorf("got %v, want ErrRuntimeNil", err)
	}
	if err := r.Initialize(context.Background()); !errors.Is(err, runtime.ErrRuntimeNil) {
		t.Errorf("got %v, want ErrRuntimeNil", err)
	}
	if err := r.Prepare(context.Background()); !errors.Is(err, runtime.ErrRuntimeNil) {
		t.Errorf("got %v, want ErrRuntimeNil", err)
	}
	if err := r.Start(context.Background()); !errors.Is(err, runtime.ErrRuntimeNil) {
		t.Errorf("got %v, want ErrRuntimeNil", err)
	}
	if err := r.Shutdown(context.Background()); !errors.Is(err, runtime.ErrRuntimeNil) {
		t.Errorf("got %v, want ErrRuntimeNil", err)
	}
	if r.State() != runtime.StateTerminated {
		t.Errorf("got state %v, want Terminated", r.State())
	}
	if r.IsRunning() {
		t.Error("IsRunning should be false for nil runtime")
	}

	health := r.Health()
	if health.Status != runtime.HealthTerminated {
		t.Errorf("got health status %v, want Terminated", health.Status)
	}
}
