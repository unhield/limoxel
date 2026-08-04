package cli

import (
	"fmt"

	"github.com/unhield/limoxel/internal/engine"
	"github.com/unhield/limoxel/internal/filesystem"
	"github.com/unhield/limoxel/internal/language"
	"github.com/unhield/limoxel/internal/parser"
	"github.com/unhield/limoxel/internal/project"
	"github.com/unhield/limoxel/internal/repository"
	"github.com/unhield/limoxel/internal/workspace"
)

// Bootstrap coordinates deterministic CLI application startup and Engine Foundation initialization.
type Bootstrap struct {
	config      *Config
	engine      *engine.Engine
	initialized bool
}

// NewBootstrap constructs a new Bootstrap instance with config.
func NewBootstrap(config *Config) (*Bootstrap, error) {
	if config == nil {
		return nil, ErrNilConfig
	}

	return &Bootstrap{
		config: config,
	}, nil
}

// Config returns the CLI Bootstrap configuration.
func (b *Bootstrap) Config() *Config {
	if b == nil {
		return nil
	}
	return b.config
}

// Initialized reports whether the CLI bootstrap process has completed successfully.
func (b *Bootstrap) Initialized() bool {
	if b == nil {
		return false
	}
	return b.initialized
}

// Engine returns the initialized engine.Engine coordinator instance, or nil if uninitialized.
func (b *Bootstrap) Engine() *engine.Engine {
	if b == nil {
		return nil
	}
	return b.engine
}

// Initialize performs synchronous, deterministic startup and initializes all required foundation subsystems.
func (b *Bootstrap) Initialize() (*engine.Engine, error) {
	if b == nil {
		return nil, ErrNilBootstrap
	}
	if b.initialized {
		return b.engine, ErrAlreadyInitialized
	}

	fileSer, err := b.initFilesystem()
	if err != nil {
		return nil, err
	}

	langReg := language.NewRegistry()
	parserReg := parser.NewRegistry()

	ws, err := b.initWorkspace(b.config.RootDir())
	if err != nil {
		return nil, err
	}

	proj, err := b.initProject(ws, b.config.RootDir())
	if err != nil {
		return nil, err
	}

	repo, err := b.initRepository(proj, b.config.RootDir())
	if err != nil {
		return nil, err
	}

	eng, err := b.initEngine(fileSer, langReg, parserReg, ws, repo)
	if err != nil {
		return nil, err
	}

	b.engine = eng
	b.initialized = true

	return eng, nil
}

func (b *Bootstrap) initFilesystem() (*filesystem.FileService, error) {
	osFs := filesystem.NewOSFilesystem()
	fileSer, err := filesystem.NewFileService(osFs)
	if err != nil {
		return nil, fmt.Errorf("%w: filesystem service failure: %v", ErrBootstrapFailed, err)
	}
	return fileSer, nil
}

func (b *Bootstrap) initWorkspace(rootDir string) (*workspace.Workspace, error) {
	ws, err := workspace.New("default-workspace", rootDir)
	if err != nil {
		return nil, fmt.Errorf("%w: workspace initialization failure: %v", ErrBootstrapFailed, err)
	}
	return ws, nil
}

func (b *Bootstrap) initProject(ws *workspace.Workspace, rootDir string) (*project.Project, error) {
	proj, err := project.New("default-project", ws, rootDir)
	if err != nil {
		return nil, fmt.Errorf("%w: project initialization failure: %v", ErrBootstrapFailed, err)
	}
	return proj, nil
}

func (b *Bootstrap) initRepository(proj *project.Project, rootDir string) (*repository.Repository, error) {
	repo, err := repository.New("default-repository", proj, rootDir)
	if err != nil {
		return nil, fmt.Errorf("%w: repository initialization failure: %v", ErrBootstrapFailed, err)
	}
	return repo, nil
}

func (b *Bootstrap) initEngine(fileSer *filesystem.FileService, langReg *language.Registry, parserReg *parser.Registry, ws *workspace.Workspace, repo *repository.Repository) (*engine.Engine, error) {
	engCfg, err := engine.NewConfig("limoxel-cli-engine", "Limoxel CLI Coordinator Engine")
	if err != nil {
		return nil, fmt.Errorf("%w: engine config failure: %v", ErrBootstrapFailed, err)
	}

	eng, err := engine.New(engCfg)
	if err != nil {
		return nil, fmt.Errorf("%w: engine instantiation failure: %v", ErrBootstrapFailed, err)
	}

	if err := eng.WithFilesystem(fileSer); err != nil {
		return nil, fmt.Errorf("%w: attaching filesystem failed: %v", ErrBootstrapFailed, err)
	}
	if err := eng.WithLanguageRegistry(langReg); err != nil {
		return nil, fmt.Errorf("%w: attaching language registry failed: %v", ErrBootstrapFailed, err)
	}
	if err := eng.WithParserRegistry(parserReg); err != nil {
		return nil, fmt.Errorf("%w: attaching parser registry failed: %v", ErrBootstrapFailed, err)
	}
	if err := eng.WithWorkspace(ws); err != nil {
		return nil, fmt.Errorf("%w: attaching workspace failed: %v", ErrBootstrapFailed, err)
	}
	if err := eng.WithRepository(repo); err != nil {
		return nil, fmt.Errorf("%w: attaching repository failed: %v", ErrBootstrapFailed, err)
	}

	if err := eng.Configure(); err != nil {
		return nil, fmt.Errorf("%w: engine configure failed: %v", ErrBootstrapFailed, err)
	}
	if err := eng.Prepare(); err != nil {
		return nil, fmt.Errorf("%w: engine prepare failed: %v", ErrBootstrapFailed, err)
	}
	if err := eng.Start(); err != nil {
		return nil, fmt.Errorf("%w: engine start failed: %v", ErrBootstrapFailed, err)
	}

	return eng, nil
}
