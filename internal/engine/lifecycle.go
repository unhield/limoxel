package engine

import "fmt"

// Configure transitions the Engine from StateCreated to StateConfigured.
// Configure is idempotent.
func (e *Engine) Configure() error {
	if e == nil {
		return ErrNilEngine
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	switch e.state {
	case StateConfigured:
		return nil
	case StateCreated:
		e.state = StateConfigured
		return nil
	default:
		return fmt.Errorf("%w: cannot configure engine in state %s", ErrEngineState, e.state)
	}
}

// Prepare validates that all required subsystem components are attached and transitions from StateConfigured to StateReady.
// Prepare is idempotent.
func (e *Engine) Prepare() error {
	if e == nil {
		return ErrNilEngine
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.state == StateReady {
		return nil
	}

	if e.state != StateConfigured {
		return fmt.Errorf("%w: cannot prepare engine in state %s", ErrEngineState, e.state)
	}

	if e.fileSer == nil {
		return fmt.Errorf("%w: filesystem.FileService", ErrMissingSubsystem)
	}
	if e.langReg == nil {
		return fmt.Errorf("%w: language.Registry", ErrMissingSubsystem)
	}
	if e.parserReg == nil {
		return fmt.Errorf("%w: parser.Registry", ErrMissingSubsystem)
	}
	if e.workspace == nil {
		return fmt.Errorf("%w: workspace.Workspace", ErrMissingSubsystem)
	}
	if e.repository == nil {
		return fmt.Errorf("%w: repository.Repository", ErrMissingSubsystem)
	}

	e.state = StateReady
	return nil
}

// Start transitions the Engine from StateReady or StateStopped to StateRunning.
// Start is idempotent.
func (e *Engine) Start() error {
	if e == nil {
		return ErrNilEngine
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	switch e.state {
	case StateRunning:
		return nil
	case StateReady, StateStopped:
		e.state = StateRunning
		return nil
	default:
		return fmt.Errorf("%w: cannot start engine in state %s", ErrEngineState, e.state)
	}
}

// Stop transitions the Engine from StateRunning to StateStopped.
// Stop is idempotent.
func (e *Engine) Stop() error {
	if e == nil {
		return ErrNilEngine
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	switch e.state {
	case StateStopped:
		return nil
	case StateRunning:
		e.state = StateStopped
		return nil
	default:
		return fmt.Errorf("%w: cannot stop engine in state %s", ErrEngineState, e.state)
	}
}

// Terminate transitions the Engine to StateTerminated and releases all subsystem references.
// Terminate is idempotent.
func (e *Engine) Terminate() error {
	if e == nil {
		return ErrNilEngine
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.state == StateTerminated {
		return nil
	}

	e.state = StateTerminated
	e.fileSer = nil
	e.langReg = nil
	e.parserReg = nil
	e.workspace = nil
	e.repository = nil
	return nil
}
