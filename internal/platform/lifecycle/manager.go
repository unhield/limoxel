package lifecycle

import (
	"context"
	"fmt"
	"sync"
)

// Manager coordinates participant lifecycle transitions in deterministic order.
type Manager struct {
	mu           sync.RWMutex
	stage        Stage
	participants []Participant
	names        map[string]struct{}
	active       []Participant
}

// NewManager creates a new Manager instance in StageCreated.
func NewManager() *Manager {
	return &Manager{
		stage:        StageCreated,
		participants: make([]Participant, 0),
		names:        make(map[string]struct{}),
		active:       make([]Participant, 0),
	}
}

// Stage returns the current canonical Stage.
func (m *Manager) Stage() Stage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stage
}

// Register adds a new Participant to the lifecycle manager. Registration is only permitted in StageCreated.
func (m *Manager) Register(p Participant) error {
	if m == nil {
		return ErrManagerNil
	}
	if p == nil {
		return ErrParticipantNil
	}

	name := p.Name()
	if name == "" {
		return fmt.Errorf("%w: name cannot be empty", ErrParticipantNil)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stage != StageCreated {
		return fmt.Errorf("%w: registration only allowed in CREATED stage, current state is %s", ErrInvalidTransition, m.stage)
	}

	if _, exists := m.names[name]; exists {
		return fmt.Errorf("%w: %s", ErrParticipantDuplicate, name)
	}

	m.names[name] = struct{}{}
	m.participants = append(m.participants, p)
	return nil
}

// Initialize transitions through StageInitializing -> StageInitialized across all registered participants sequentially.
func (m *Manager) Initialize(ctx context.Context) error {
	return m.executeForwardStage(ctx, StageInitializing, StageInitialized, func(p Participant, c context.Context) error {
		return p.Initialize(c)
	})
}

// Prepare transitions through StagePreparing -> StagePrepared across all registered participants sequentially.
func (m *Manager) Prepare(ctx context.Context) error {
	return m.executeForwardStage(ctx, StagePreparing, StagePrepared, func(p Participant, c context.Context) error {
		return p.Prepare(c)
	})
}

// Start transitions through StageStarting -> StageRunning across all registered participants sequentially.
func (m *Manager) Start(ctx context.Context) error {
	return m.executeForwardStage(ctx, StageStarting, StageRunning, func(p Participant, c context.Context) error {
		return p.Start(c)
	})
}

// Stop transitions through StageStopping -> StageStopped in REVERSE order of initialization.
func (m *Manager) Stop(ctx context.Context) error {
	if m == nil {
		return ErrManagerNil
	}
	if ctx == nil {
		return ErrNilContext
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.stage.CanTransitionTo(StageStopping) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, m.stage, StageStopping)
	}

	m.stage = StageStopping

	var stopErr error
	// Stop active participants in REVERSE order of startup
	for i := len(m.active) - 1; i >= 0; i-- {
		p := m.active[i]
		if err := p.Stop(ctx); err != nil && stopErr == nil {
			stopErr = fmt.Errorf("%w on participant %s: %v", ErrExecutionFailed, p.Name(), err)
		}
	}

	m.active = m.active[:0]
	m.stage = StageStopped

	return stopErr
}

// Terminate explicitly transitions the Manager from StageStopped to StageTerminated.
func (m *Manager) Terminate(ctx context.Context) error {
	if m == nil {
		return ErrManagerNil
	}
	if ctx == nil {
		return ErrNilContext
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.stage.CanTransitionTo(StageTerminated) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, m.stage, StageTerminated)
	}

	m.stage = StageTerminated
	return nil
}

// executeForwardStage manages forward stage transitions with fail-fast execution and reverse-order rollback cleanup.
func (m *Manager) executeForwardStage(ctx context.Context, transitional, final Stage, action func(Participant, context.Context) error) error {
	if m == nil {
		return ErrManagerNil
	}
	if ctx == nil {
		return ErrNilContext
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.stage.CanTransitionTo(transitional) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, m.stage, transitional)
	}

	m.stage = transitional

	for _, p := range m.participants {
		if err := ctx.Err(); err != nil {
			m.rollbackLocked(ctx)
			return fmt.Errorf("%w: %w", ErrExecutionFailed, err)
		}

		if err := action(p, ctx); err != nil {
			m.rollbackLocked(ctx)
			return fmt.Errorf("%w stage %s on participant %s: %v", ErrExecutionFailed, transitional, p.Name(), err)
		}

		if transitional == StageStarting {
			m.active = append(m.active, p)
		}
	}

	if !m.stage.CanTransitionTo(final) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, m.stage, final)
	}

	m.stage = final
	return nil
}

// rollbackLocked performs reverse-order cleanup when a forward stage fails.
func (m *Manager) rollbackLocked(ctx context.Context) {
	m.stage = StageStopping
	for i := len(m.active) - 1; i >= 0; i-- {
		_ = m.active[i].Stop(ctx)
	}
	m.active = m.active[:0]
	m.stage = StageStopped
}
