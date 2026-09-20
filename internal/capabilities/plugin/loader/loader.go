package loader

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	pkgerr "github.com/unhield/limoxel/internal/capabilities/plugin/errors"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/plugin"
)

// Instance represents a running or loaded plugin instance with lifecycle control.
type Instance interface {
	ID() model.Identity
	Manifest() *model.Manifest
	Plugin() plugin.Plugin
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Dispose() error
}

// Loader abstracts runtime plugin instantiation, unloading, and reloading.
type Loader interface {
	Load(ctx context.Context, manifest *model.Manifest, dir string, host plugin.Host) (Instance, error)
	Unload(ctx context.Context, inst Instance) error
	Reload(ctx context.Context, inst Instance, host plugin.Host) (Instance, error)
}

// PluginFactory is a constructor that creates a fresh instance of a plugin.
type PluginFactory func() (plugin.Plugin, error)

// InProcessLoader manages in-process plugin execution with strict panic containment.
type InProcessLoader struct {
	mu        sync.RWMutex
	factories map[model.Identity]PluginFactory
	instances map[model.Identity]*inProcessInstance
}

// NewInProcessLoader constructs an initialized InProcessLoader.
func NewInProcessLoader() *InProcessLoader {
	return &InProcessLoader{
		factories: make(map[model.Identity]PluginFactory),
		instances: make(map[model.Identity]*inProcessInstance),
	}
}

// RegisterFactory registers a constructor for a specific plugin identity.
func (l *InProcessLoader) RegisterFactory(id model.Identity, factory PluginFactory) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.factories[id] = factory
}

// Load instantiates, initializes, and binds a plugin instance.
func (l *InProcessLoader) Load(ctx context.Context, manifest *model.Manifest, dir string, host plugin.Host) (Instance, error) {
	if manifest == nil {
		return nil, pkgerr.New(pkgerr.CodeManifestInvalid, "", "manifest cannot be nil")
	}

	l.mu.Lock()
	factory, ok := l.factories[manifest.ID]
	if !ok {
		l.mu.Unlock()
		return nil, pkgerr.New(pkgerr.CodeLoadingFailed, manifest.ID,
			fmt.Sprintf("no in-process factory registered for plugin %s", manifest.ID))
	}
	l.mu.Unlock()

	plug, err := factory()
	if err != nil {
		return nil, pkgerr.Wrap(pkgerr.CodeLoadingFailed, manifest.ID, "factory failed to create plugin instance", err)
	}

	inst := &inProcessInstance{
		id:       manifest.ID,
		manifest: manifest,
		dir:      dir,
		plugin:   plug,
	}

	// Initialize with panic containment
	if initErr := l.safeCall(func() error {
		return plug.Init(ctx, host)
	}); initErr != nil {
		_ = inst.Dispose()
		return nil, pkgerr.Wrap(pkgerr.CodeInitializationFailed, manifest.ID, "plugin Init() failed", initErr)
	}

	l.mu.Lock()
	l.instances[manifest.ID] = inst
	l.mu.Unlock()

	return inst, nil
}

// Unload stops and disposes a plugin instance, releasing all runtime resources.
func (l *InProcessLoader) Unload(ctx context.Context, inst Instance) error {
	if inst == nil {
		return nil
	}

	l.mu.Lock()
	delete(l.instances, inst.ID())
	l.mu.Unlock()

	stopErr := l.safeCall(func() error {
		return inst.Stop(ctx)
	})

	dispErr := inst.Dispose()
	if stopErr != nil {
		return pkgerr.Wrap(pkgerr.CodeUnloadingFailed, inst.ID(), "failed to stop plugin during unload", stopErr)
	}
	if dispErr != nil {
		return pkgerr.Wrap(pkgerr.CodeUnloadingFailed, inst.ID(), "failed to dispose plugin during unload", dispErr)
	}

	return nil
}

// Reload performs an honest reload by stopping, disposing, and reconstructing the plugin instance.
func (l *InProcessLoader) Reload(ctx context.Context, inst Instance, host plugin.Host) (Instance, error) {
	if inst == nil {
		return nil, pkgerr.New(pkgerr.CodeLoadingFailed, "", "cannot reload nil instance")
	}

	manifest := inst.Manifest()
	dir := ""
	if ip, ok := inst.(*inProcessInstance); ok {
		dir = ip.dir
	}

	// 1. Unload old instance
	if err := l.Unload(ctx, inst); err != nil {
		return nil, pkgerr.Wrap(pkgerr.CodeUnloadingFailed, inst.ID(), "reload failed during unload step", err)
	}

	// 2. Load fresh instance
	newInst, err := l.Load(ctx, manifest, dir, host)
	if err != nil {
		return nil, pkgerr.Wrap(pkgerr.CodeLoadingFailed, manifest.ID, "reload failed during load step", err)
	}

	// 3. Start fresh instance
	if startErr := newInst.Start(ctx); startErr != nil {
		_ = l.Unload(ctx, newInst)
		return nil, pkgerr.Wrap(pkgerr.CodeInitializationFailed, manifest.ID, "reload failed during start step", startErr)
	}

	return newInst, nil
}

func (l *InProcessLoader) safeCall(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("plugin panic caught: %v\nstack:\n%s", r, string(debug.Stack()))
		}
	}()
	return fn()
}

type inProcessInstance struct {
	mu       sync.Mutex
	id       model.Identity
	manifest *model.Manifest
	dir      string
	plugin   plugin.Plugin
	started  bool
	disposed bool
}

func (i *inProcessInstance) ID() model.Identity {
	return i.id
}

func (i *inProcessInstance) Manifest() *model.Manifest {
	return i.manifest
}

func (i *inProcessInstance) Plugin() plugin.Plugin {
	return i.plugin
}

func (i *inProcessInstance) Start(ctx context.Context) (err error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.disposed {
		return pkgerr.New(pkgerr.CodeLoadingFailed, i.id, "cannot start disposed plugin instance")
	}
	if i.started {
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("plugin Start panic caught: %v\nstack:\n%s", r, string(debug.Stack()))
		}
	}()

	if err := i.plugin.Start(ctx); err != nil {
		return err
	}
	i.started = true
	return nil
}

func (i *inProcessInstance) Stop(ctx context.Context) (err error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.started || i.disposed {
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("plugin Stop panic caught: %v\nstack:\n%s", r, string(debug.Stack()))
		}
	}()

	if err := i.plugin.Stop(ctx); err != nil {
		return err
	}
	i.started = false
	return nil
}

func (i *inProcessInstance) Dispose() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.disposed = true
	i.started = false
	i.plugin = nil
	return nil
}
