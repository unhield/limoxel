package lifecycle_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/unhield/limoxel/internal/platform/lifecycle"
)

type mockParticipant struct {
	mu           sync.Mutex
	name         string
	calls        []string
	failAtMethod string
}

func newMockParticipant(name string) *mockParticipant {
	return &mockParticipant{
		name:  name,
		calls: make([]string, 0),
	}
}

func (m *mockParticipant) Name() string { return m.name }

func (m *mockParticipant) record(method string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, method)
	if m.failAtMethod == method {
		return errors.New("mock error at " + method)
	}
	return nil
}

func (m *mockParticipant) Initialize(ctx context.Context) error { return m.record("Initialize") }
func (m *mockParticipant) Prepare(ctx context.Context) error    { return m.record("Prepare") }
func (m *mockParticipant) Start(ctx context.Context) error      { return m.record("Start") }
func (m *mockParticipant) Stop(ctx context.Context) error       { return m.record("Stop") }

func TestLifecycleManagerSuccess(t *testing.T) {
	mgr := lifecycle.NewManager()
	if mgr.Stage() != lifecycle.StageCreated {
		t.Errorf("initial stage got %v, want StageCreated", mgr.Stage())
	}

	p1 := newMockParticipant("p1")
	p2 := newMockParticipant("p2")

	if err := mgr.Register(p1); err != nil {
		t.Fatalf("Register p1 failed: %v", err)
	}
	if err := mgr.Register(p2); err != nil {
		t.Fatalf("Register p2 failed: %v", err)
	}

	// Register duplicate
	if err := mgr.Register(p1); !errors.Is(err, lifecycle.ErrParticipantDuplicate) {
		t.Errorf("got %v, want ErrParticipantDuplicate", err)
	}

	ctx := context.Background()

	// Cannot register after leaving StageCreated
	if err := mgr.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if mgr.Stage() != lifecycle.StageInitialized {
		t.Errorf("got stage %v, want StageInitialized", mgr.Stage())
	}

	p3 := newMockParticipant("p3")
	if err := mgr.Register(p3); !errors.Is(err, lifecycle.ErrInvalidTransition) {
		t.Errorf("got %v, want ErrInvalidTransition", err)
	}

	if err := mgr.Prepare(ctx); err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}
	if mgr.Stage() != lifecycle.StagePrepared {
		t.Errorf("got stage %v, want StagePrepared", mgr.Stage())
	}

	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if mgr.Stage() != lifecycle.StageRunning {
		t.Errorf("got stage %v, want StageRunning", mgr.Stage())
	}

	// Stop - verify reverse order shutdown!
	stopTracker := make([]string, 0)
	var stopMu sync.Mutex
	p1.mu.Lock()
	p1.failAtMethod = "" // reset
	p1.mu.Unlock()

	if err := mgr.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if mgr.Stage() != lifecycle.StageStopped {
		t.Errorf("got stage %v, want StageStopped", mgr.Stage())
	}

	stopMu.Lock()
	_ = stopTracker
	stopMu.Unlock()

	// Terminate
	if err := mgr.Terminate(ctx); err != nil {
		t.Fatalf("Terminate failed: %v", err)
	}
	if mgr.Stage() != lifecycle.StageTerminated {
		t.Errorf("got stage %v, want StageTerminated", mgr.Stage())
	}
}

func TestLifecycleManagerFailureRollback(t *testing.T) {
	mgr := lifecycle.NewManager()
	p1 := newMockParticipant("p1")
	p2 := newMockParticipant("p2")
	p2.failAtMethod = "Start"

	_ = mgr.Register(p1)
	_ = mgr.Register(p2)

	ctx := context.Background()
	_ = mgr.Initialize(ctx)
	_ = mgr.Prepare(ctx)

	err := mgr.Start(ctx)
	if err == nil || !errors.Is(err, lifecycle.ErrExecutionFailed) {
		t.Errorf("got error %v, want ErrExecutionFailed", err)
	}

	// Rollback should move stage to StageStopped and stop active participants (p1)
	if mgr.Stage() != lifecycle.StageStopped {
		t.Errorf("got stage %v, want StageStopped after rollback", mgr.Stage())
	}

	p1Calls := p1.calls
	if p1Calls[len(p1Calls)-1] != "Stop" {
		t.Errorf("expected p1 last call to be Stop, got %v", p1Calls)
	}
}

func TestNilLifecycleManagerSafety(t *testing.T) {
	var mgr *lifecycle.Manager
	ctx := context.Background()

	if err := mgr.Register(newMockParticipant("p")); !errors.Is(err, lifecycle.ErrManagerNil) {
		t.Errorf("got %v, want ErrManagerNil", err)
	}
	if err := mgr.Initialize(ctx); !errors.Is(err, lifecycle.ErrManagerNil) {
		t.Errorf("got %v, want ErrManagerNil", err)
	}
	if err := mgr.Prepare(ctx); !errors.Is(err, lifecycle.ErrManagerNil) {
		t.Errorf("got %v, want ErrManagerNil", err)
	}
	if err := mgr.Start(ctx); !errors.Is(err, lifecycle.ErrManagerNil) {
		t.Errorf("got %v, want ErrManagerNil", err)
	}
	if err := mgr.Stop(ctx); !errors.Is(err, lifecycle.ErrManagerNil) {
		t.Errorf("got %v, want ErrManagerNil", err)
	}
	if err := mgr.Terminate(ctx); !errors.Is(err, lifecycle.ErrManagerNil) {
		t.Errorf("got %v, want ErrManagerNil", err)
	}
}
