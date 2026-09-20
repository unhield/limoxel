package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	pkgerr "github.com/unhield/limoxel/internal/capabilities/plugin/errors"
	"github.com/unhield/limoxel/internal/capabilities/plugin/loader"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/plugin"
)

// ExecutableResolver determines the command line and environment to launch a plugin's host process.
type ExecutableResolver func(manifest *model.Manifest, dir string) (command string, args []string, env map[string]string, err error)

// ProcessLoader implements loader.Loader for out-of-process plugin execution.
type ProcessLoader struct {
	mu        sync.RWMutex
	resolver  ExecutableResolver
	instances map[model.Identity]*processInstance
}

// NewProcessLoader creates an initialized ProcessLoader with an optional executable resolver.
func NewProcessLoader(resolver ExecutableResolver) *ProcessLoader {
	if resolver == nil {
		resolver = defaultExecutableResolver
	}
	return &ProcessLoader{
		resolver:  resolver,
		instances: make(map[model.Identity]*processInstance),
	}
}

func defaultExecutableResolver(manifest *model.Manifest, dir string) (string, []string, map[string]string, error) {
	if manifest == nil {
		return "", nil, nil, errors.New("nil manifest")
	}

	// 1. Check custom attributes in metadata
	attrs := manifest.Metadata.CustomAttributes()
	if exe, ok := attrs["executable"]; ok && exe != "" {
		if !filepath.IsAbs(exe) && dir != "" {
			exe = filepath.Join(dir, exe)
		}
		return exe, nil, nil, nil
	}

	// 2. Check manifest entrypoint
	if manifest.Entrypoint != "" {
		exe := manifest.Entrypoint
		if !filepath.IsAbs(exe) && dir != "" {
			exe = filepath.Join(dir, exe)
		}
		return exe, nil, nil, nil
	}

	// 2. Check environment variable override
	if envBin := os.Getenv("LIMOXEL_PLUGIN_HOST_BIN"); envBin != "" {
		return envBin, []string{string(manifest.ID)}, nil, nil
	}

	// 3. Fallback: executable must be provided via resolver
	return "", nil, nil, fmt.Errorf("no executable configured for external plugin %s", manifest.ID)
}

// Load launches an external plugin process, performs handshake, initializes it, and returns a loader.Instance.
func (l *ProcessLoader) Load(ctx context.Context, manifest *model.Manifest, dir string, host plugin.Host) (loader.Instance, error) {
	if manifest == nil {
		return nil, pkgerr.New(pkgerr.CodeManifestInvalid, "", "manifest cannot be nil")
	}

	cmd, args, env, err := l.resolver(manifest, dir)
	if err != nil {
		return nil, pkgerr.Wrap(pkgerr.CodeLoadingFailed, manifest.ID, "failed to resolve plugin executable", err)
	}

	rt := NewProcessRuntime(ProcessRuntimeConfig{
		Manifest:        manifest,
		Command:         cmd,
		Args:            args,
		Dir:             dir,
		Env:             env,
		StartupTimeout:  5 * time.Second,
		ShutdownTimeout: 3 * time.Second,
	})

	if err := rt.Start(ctx); err != nil {
		return nil, err
	}

	wsRoot := ""
	if host != nil && host.Repository() != nil {
		wsRoot = host.Repository().WorkspacePath()
	}

	if err := rt.Initialize(ctx, wsRoot, nil); err != nil {
		_ = rt.Close()
		return nil, pkgerr.Wrap(pkgerr.CodeInitializationFailed, manifest.ID, "plugin host initialization failed", err)
	}

	inst := &processInstance{
		id:       manifest.ID,
		manifest: manifest,
		dir:      dir,
		runtime:  rt,
		host:     host,
	}

	l.mu.Lock()
	l.instances[manifest.ID] = inst
	l.mu.Unlock()

	return inst, nil
}

// Unload gracefully terminates the external plugin process, verifies termination, and releases resources.
func (l *ProcessLoader) Unload(ctx context.Context, inst loader.Instance) error {
	if inst == nil {
		return nil
	}

	l.mu.Lock()
	delete(l.instances, inst.ID())
	l.mu.Unlock()

	stopErr := inst.Stop(ctx)
	dispErr := inst.Dispose()

	if stopErr != nil {
		return pkgerr.Wrap(pkgerr.CodeUnloadingFailed, inst.ID(), "failed to stop process instance during unload", stopErr)
	}
	if dispErr != nil {
		return pkgerr.Wrap(pkgerr.CodeUnloadingFailed, inst.ID(), "failed to dispose process instance during unload", dispErr)
	}
	return nil
}

// Reload executes genuine out-of-process replacement: stopping the old process, verifying exit, and spawning a fresh replacement.
func (l *ProcessLoader) Reload(ctx context.Context, inst loader.Instance, host plugin.Host) (loader.Instance, error) {
	if inst == nil {
		return nil, pkgerr.New(pkgerr.CodeManifestInvalid, "", "cannot reload nil instance")
	}

	manifest := inst.Manifest()
	pInst, ok := inst.(*processInstance)
	if !ok {
		return nil, pkgerr.New(pkgerr.CodeLoadingFailed, inst.ID(), "instance is not a processInstance")
	}

	oldPID := pInst.ProcessID()
	oldSession := pInst.SessionID()
	oldInstanceID := pInst.InstanceID()
	dir := pInst.dir

	// 1. Unload old process completely
	if err := l.Unload(ctx, inst); err != nil {
		return nil, pkgerr.Wrap(pkgerr.CodeUnloadingFailed, inst.ID(), "reload failed during unload step", err)
	}

	// 2. Load fresh replacement process
	newInst, err := l.Load(ctx, manifest, dir, host)
	if err != nil {
		return nil, pkgerr.Wrap(pkgerr.CodeLoadingFailed, manifest.ID, "reload failed during load step", err)
	}

	newPInst, ok := newInst.(*processInstance)
	if ok {
		newPID := newPInst.ProcessID()
		newSession := newPInst.SessionID()
		newInstanceID := newPInst.InstanceID()

		if newInstanceID == oldInstanceID && oldInstanceID != "" {
			_ = l.Unload(ctx, newInst)
			return nil, pkgerr.New(pkgerr.CodeLoadingFailed, manifest.ID, "reload failed: replacement instance reused old supervisor instance ID")
		}
		if newSession == oldSession && oldSession != "" {
			_ = l.Unload(ctx, newInst)
			return nil, pkgerr.New(pkgerr.CodeLoadingFailed, manifest.ID, "reload failed: replacement session reused old session ID")
		}
		if newPID == oldPID && oldPID > 0 {
			_ = l.Unload(ctx, newInst)
			return nil, pkgerr.New(pkgerr.CodeLoadingFailed, manifest.ID, "reload failed: replacement process reused old PID without genuine replacement")
		}
	}

	// 3. Start fresh process
	if startErr := newInst.Start(ctx); startErr != nil {
		_ = l.Unload(ctx, newInst)
		return nil, pkgerr.Wrap(pkgerr.CodeInitializationFailed, manifest.ID, "reload failed during start step", startErr)
	}

	return newInst, nil
}

// processInstance wraps ProcessRuntime to satisfy loader.Instance.
type processInstance struct {
	id       model.Identity
	manifest *model.Manifest
	dir      string
	runtime  *ProcessRuntime
	host     plugin.Host
}

func (i *processInstance) ID() model.Identity {
	return i.id
}

func (i *processInstance) Manifest() *model.Manifest {
	return i.manifest
}

func (i *processInstance) Plugin() plugin.Plugin {
	var pub plugin.Manifest
	if i.manifest != nil {
		pub = i.manifest.ToPublic()
	}
	return &proxyPlugin{
		id:       string(i.id),
		manifest: pub,
		rt:       i.runtime,
	}
}

func (i *processInstance) Start(ctx context.Context) error {
	if i.runtime.State() == StateRunning {
		return nil
	}
	return i.runtime.Start(ctx)
}

func (i *processInstance) Stop(ctx context.Context) error {
	return i.runtime.Stop(ctx)
}

func (i *processInstance) Dispose() error {
	return i.runtime.Close()
}

func (i *processInstance) ProcessID() int {
	return i.runtime.ProcessID()
}

func (i *processInstance) SessionID() string {
	return i.runtime.SessionID()
}

func (i *processInstance) InstanceID() string {
	return i.runtime.InstanceID()
}

func (i *processInstance) Runtime() *ProcessRuntime {
	return i.runtime
}

// proxyPlugin provides a plugin.Plugin facade forwarding calls over the runtime protocol.
type proxyPlugin struct {
	id       string
	manifest plugin.Manifest
	rt       *ProcessRuntime
}

func (p *proxyPlugin) ID() string {
	return p.id
}

func (p *proxyPlugin) Manifest() plugin.Manifest {
	return p.manifest
}

func (p *proxyPlugin) Init(ctx context.Context, host plugin.Host) error {
	wsRoot := ""
	if host != nil && host.Repository() != nil {
		wsRoot = host.Repository().WorkspacePath()
	}
	return p.rt.Initialize(ctx, wsRoot, nil)
}

func (p *proxyPlugin) Start(ctx context.Context) error {
	if p.rt.State() == StateRunning {
		return nil
	}
	return p.rt.Start(ctx)
}

func (p *proxyPlugin) Stop(ctx context.Context) error {
	return p.rt.Stop(ctx)
}
