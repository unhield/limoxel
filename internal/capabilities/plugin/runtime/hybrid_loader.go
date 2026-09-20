package runtime

import (
	"context"
	"strings"
	"sync"

	pkgerr "github.com/unhield/limoxel/internal/capabilities/plugin/errors"
	"github.com/unhield/limoxel/internal/capabilities/plugin/loader"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/plugin"
)

// HybridLoader multiplexes between in-process execution and external process execution based on manifest configuration.
type HybridLoader struct {
	mu           sync.RWMutex
	inProcessLdr *loader.InProcessLoader
	processLdr   *ProcessLoader
	loadedTypes  map[model.Identity]bool // true if external process, false if in-process
}

// NewHybridLoader creates a hybrid loader combining an InProcessLoader and a ProcessLoader.
func NewHybridLoader(inProcess *loader.InProcessLoader, process *ProcessLoader) *HybridLoader {
	if inProcess == nil {
		inProcess = loader.NewInProcessLoader()
	}
	if process == nil {
		process = NewProcessLoader(nil)
	}
	return &HybridLoader{
		inProcessLdr: inProcess,
		processLdr:   process,
		loadedTypes:  make(map[model.Identity]bool),
	}
}

// InProcessLoader returns the underlying InProcessLoader for factory registrations.
func (l *HybridLoader) InProcessLoader() *loader.InProcessLoader {
	return l.inProcessLdr
}

// ProcessLoader returns the underlying ProcessLoader.
func (l *HybridLoader) ProcessLoader() *ProcessLoader {
	return l.processLdr
}

// ShouldUseExternalProcess evaluates whether a manifest requests or requires out-of-process isolation.
func (l *HybridLoader) ShouldUseExternalProcess(manifest *model.Manifest) bool {
	if manifest == nil {
		return false
	}

	attrs := manifest.Metadata.CustomAttributes()
	if mode, ok := attrs["mode"]; ok {
		m := strings.ToLower(strings.TrimSpace(mode))
		if m == "process" || m == "external" || m == "isolated" {
			return true
		}
	}

	if exe, ok := attrs["executable"]; ok && exe != "" {
		return true
	}

	if manifest.Entrypoint != "" {
		return true
	}

	return false
}

// Load routes instantiation to the appropriate runtime engine.
func (l *HybridLoader) Load(ctx context.Context, manifest *model.Manifest, dir string, host plugin.Host) (loader.Instance, error) {
	if manifest == nil {
		return nil, pkgerr.New(pkgerr.CodeManifestInvalid, "", "manifest cannot be nil")
	}

	isExternal := l.ShouldUseExternalProcess(manifest)

	var inst loader.Instance
	var err error

	if isExternal {
		inst, err = l.processLdr.Load(ctx, manifest, dir, host)
	} else {
		inst, err = l.inProcessLdr.Load(ctx, manifest, dir, host)
	}

	if err != nil {
		return nil, err
	}

	l.mu.Lock()
	l.loadedTypes[manifest.ID] = isExternal
	l.mu.Unlock()

	return inst, nil
}

// Unload dispatches uninstallation to the engine managing the instance.
func (l *HybridLoader) Unload(ctx context.Context, inst loader.Instance) error {
	if inst == nil {
		return nil
	}

	l.mu.Lock()
	isExternal, found := l.loadedTypes[inst.ID()]
	delete(l.loadedTypes, inst.ID())
	l.mu.Unlock()

	if !found {
		// Fallback detection based on concrete type
		if _, ok := inst.(*processInstance); ok {
			return l.processLdr.Unload(ctx, inst)
		}
		return l.inProcessLdr.Unload(ctx, inst)
	}

	if isExternal {
		return l.processLdr.Unload(ctx, inst)
	}
	return l.inProcessLdr.Unload(ctx, inst)
}

// Reload dispatches reload to the corresponding engine.
func (l *HybridLoader) Reload(ctx context.Context, inst loader.Instance, host plugin.Host) (loader.Instance, error) {
	if inst == nil {
		return nil, pkgerr.New(pkgerr.CodeManifestInvalid, "", "cannot reload nil instance")
	}

	l.mu.RLock()
	isExternal := l.loadedTypes[inst.ID()]
	l.mu.RUnlock()

	if isExternal {
		return l.processLdr.Reload(ctx, inst, host)
	}
	return l.inProcessLdr.Reload(ctx, inst, host)
}
