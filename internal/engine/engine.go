package engine

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/unhield/limoxel/internal/filesystem"
	"github.com/unhield/limoxel/internal/language"
	"github.com/unhield/limoxel/internal/parser"
	"github.com/unhield/limoxel/internal/repository"
	"github.com/unhield/limoxel/internal/workspace"
)

// Engine is the central coordinator orchestrating foundation subsystems.
type Engine struct {
	mu         sync.RWMutex
	config     *Config
	state      State
	fileSer    *filesystem.FileService
	langReg    *language.Registry
	parserReg  *parser.Registry
	workspace  *workspace.Workspace
	repository *repository.Repository
}

// New constructs a new Engine coordinator with cfg in StateCreated.
func New(cfg *Config) (*Engine, error) {
	if cfg == nil {
		return nil, ErrNilConfig
	}

	return &Engine{
		config: cfg,
		state:  StateCreated,
	}, nil
}

// Config returns the Engine configuration.
func (e *Engine) Config() *Config {
	if e == nil {
		return nil
	}
	return e.config
}

// State returns the current operational lifecycle state of the Engine.
func (e *Engine) State() State {
	if e == nil {
		return StateTerminated
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state
}

// attachSubsystem is an unexported helper to eliminate duplicated attachment validation logic.
func (e *Engine) attachSubsystem(component any, assign func()) error {
	if e == nil {
		return ErrNilEngine
	}
	if component == nil || (reflect.ValueOf(component).Kind() == reflect.Pointer && reflect.ValueOf(component).IsNil()) {
		return ErrNilSubsystem
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.state != StateCreated {
		return fmt.Errorf("%w: cannot attach subsystem in state %s", ErrEngineState, e.state)
	}

	assign()
	return nil
}

// WithFilesystem attaches a FileService subsystem component to the Engine.
func (e *Engine) WithFilesystem(fsService *filesystem.FileService) error {
	return e.attachSubsystem(fsService, func() {
		e.fileSer = fsService
	})
}

// WithLanguageRegistry attaches a Language Registry subsystem component to the Engine.
func (e *Engine) WithLanguageRegistry(langReg *language.Registry) error {
	return e.attachSubsystem(langReg, func() {
		e.langReg = langReg
	})
}

// WithParserRegistry attaches a Parser Registry subsystem component to the Engine.
func (e *Engine) WithParserRegistry(parserReg *parser.Registry) error {
	return e.attachSubsystem(parserReg, func() {
		e.parserReg = parserReg
	})
}

// WithWorkspace attaches a Workspace component to the Engine.
func (e *Engine) WithWorkspace(ws *workspace.Workspace) error {
	return e.attachSubsystem(ws, func() {
		e.workspace = ws
	})
}

// WithRepository attaches a Repository component to the Engine.
func (e *Engine) WithRepository(repo *repository.Repository) error {
	return e.attachSubsystem(repo, func() {
		e.repository = repo
	})
}

// Filesystem returns the attached filesystem.FileService component.
func (e *Engine) Filesystem() *filesystem.FileService {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.fileSer
}

// LanguageRegistry returns the attached language.Registry component.
func (e *Engine) LanguageRegistry() *language.Registry {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.langReg
}

// ParserRegistry returns the attached parser.Registry component.
func (e *Engine) ParserRegistry() *parser.Registry {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.parserReg
}

// Workspace returns the attached workspace.Workspace component.
func (e *Engine) Workspace() *workspace.Workspace {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.workspace
}

// Repository returns the attached repository.Repository component.
func (e *Engine) Repository() *repository.Repository {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.repository
}
