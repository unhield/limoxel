package lifecycle

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/dependency"
	pkgerr "github.com/unhield/limoxel/internal/capabilities/plugin/errors"
	"github.com/unhield/limoxel/internal/capabilities/plugin/loader"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/plugin/registry"
	"github.com/unhield/limoxel/internal/capabilities/plugin/validation"
	"github.com/unhield/limoxel/plugin"
)

// Manager orchestrates atomic plugin lifecycle operations and coordinates registry and loader interactions.
type Manager struct {
	mu   sync.Mutex
	reg  *registry.Registry
	ldr  loader.Loader
	val  *validation.Validator
	res  *dependency.Resolver
	host plugin.Host
}

// NewManager constructs an initialized lifecycle Manager.
func NewManager(reg *registry.Registry, ldr loader.Loader, val *validation.Validator, res *dependency.Resolver, host plugin.Host) *Manager {
	return &Manager{
		reg:  reg,
		ldr:  ldr,
		val:  val,
		res:  res,
		host: host,
	}
}

// Install validates a candidate manifest, checks environmental requirements, and registers the plugin as StateRegistered.
func (m *Manager) Install(ctx context.Context, manifest *model.Manifest, directory string) (*registry.PluginRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Static validation
	if err := m.val.Validate(manifest); err != nil {
		return nil, err
	}

	// 2. Validate required capabilities
	for _, req := range manifest.RequiredCapabilities {
		if !req.Optional && !m.host.HasCapability(req.Name) {
			err := pkgerr.New(pkgerr.CodeDependencyMissing, manifest.ID,
				fmt.Sprintf("mandatory host capability %q required by plugin %s is not available", req.Name, manifest.ID))
			err.WithDetail("required_capability", req.Name)
			return nil, err
		}
	}

	// 3. Register record
	now := time.Now().UTC()
	record := &registry.PluginRecord{
		Manifest:    manifest,
		Directory:   directory,
		State:       plugin.StateRegistered,
		InstalledAt: now,
		UpdatedAt:   now,
	}

	if err := m.reg.Register(record); err != nil {
		return nil, err
	}

	return record, nil
}

// Activate resolves dependencies, initializes the runtime instance, and transitions the plugin to StateActive.
func (m *Manager) Activate(ctx context.Context, id model.Identity) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec, exists := m.reg.Get(id)
	if !exists {
		return pkgerr.New(pkgerr.CodePluginNotFound, id, fmt.Sprintf("plugin %s not found", id))
	}

	if rec.State == plugin.StateActive {
		return nil // Idempotent activation
	}

	if err := ValidateTransition(rec.State, plugin.StateActive); err != nil {
		return err
	}

	// Resolve dependencies against installed plugins
	allPlugins := m.reg.List()
	resolvables := make([]dependency.Resolvable, len(allPlugins))
	for i, p := range allPlugins {
		resolvables[i] = p
	}

	if _, err := m.res.Resolve(resolvables); err != nil {
		return err
	}

	// Load instance
	inst, err := m.ldr.Load(ctx, rec.Manifest, rec.Directory, m.host)
	if err != nil {
		_ = m.reg.UpdateState(id, plugin.StateFailed, err)
		return err
	}

	// Start instance
	if startErr := inst.Start(ctx); startErr != nil {
		_ = m.ldr.Unload(ctx, inst)
		_ = m.reg.UpdateState(id, plugin.StateFailed, startErr)
		return pkgerr.Wrap(pkgerr.CodeInitializationFailed, id, "failed to start plugin", startErr)
	}

	_ = m.reg.UpdateInstance(id, inst)
	_ = m.reg.UpdateState(id, plugin.StateActive, nil)
	return nil
}

// Suspend pauses runtime execution of an active plugin and transitions it to StateSuspended.
func (m *Manager) Suspend(ctx context.Context, id model.Identity) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec, exists := m.reg.Get(id)
	if !exists {
		return pkgerr.New(pkgerr.CodePluginNotFound, id, fmt.Sprintf("plugin %s not found", id))
	}

	if rec.State == plugin.StateSuspended {
		return nil
	}

	if err := ValidateTransition(rec.State, plugin.StateSuspended); err != nil {
		return err
	}

	if inst, ok := rec.Instance.(loader.Instance); ok && inst != nil {
		if err := inst.Stop(ctx); err != nil {
			_ = m.reg.UpdateState(id, plugin.StateFailed, err)
			return pkgerr.Wrap(pkgerr.CodeInitializationFailed, id, "failed to stop plugin during suspension", err)
		}
	}

	return m.reg.UpdateState(id, plugin.StateSuspended, nil)
}

// Restart stops, disposes, and re-initializes an existing plugin instance.
func (m *Manager) Restart(ctx context.Context, id model.Identity) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec, exists := m.reg.Get(id)
	if !exists {
		return pkgerr.New(pkgerr.CodePluginNotFound, id, fmt.Sprintf("plugin %s not found", id))
	}

	inst, ok := rec.Instance.(loader.Instance)
	if ok && inst != nil {
		newInst, err := m.ldr.Reload(ctx, inst, m.host)
		if err != nil {
			_ = m.reg.UpdateState(id, plugin.StateFailed, err)
			return err
		}
		_ = m.reg.UpdateInstance(id, newInst)
		_ = m.reg.UpdateState(id, plugin.StateActive, nil)
		return nil
	}

	// If no existing instance (e.g. registered or failed), perform fresh activation
	freshInst, err := m.ldr.Load(ctx, rec.Manifest, rec.Directory, m.host)
	if err != nil {
		_ = m.reg.UpdateState(id, plugin.StateFailed, err)
		return err
	}

	if startErr := freshInst.Start(ctx); startErr != nil {
		_ = m.ldr.Unload(ctx, freshInst)
		_ = m.reg.UpdateState(id, plugin.StateFailed, startErr)
		return startErr
	}

	_ = m.reg.UpdateInstance(id, freshInst)
	_ = m.reg.UpdateState(id, plugin.StateActive, nil)
	return nil
}

// Update upgrades or reconfigures a plugin with a new manifest.
func (m *Manager) Update(ctx context.Context, id model.Identity, newManifest *model.Manifest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec, exists := m.reg.Get(id)
	if !exists {
		return pkgerr.New(pkgerr.CodePluginNotFound, id, fmt.Sprintf("plugin %s not found", id))
	}

	if newManifest == nil {
		return pkgerr.New(pkgerr.CodeManifestInvalid, id, "update manifest cannot be nil")
	}

	if newManifest.ID != id {
		return pkgerr.New(pkgerr.CodeIdentityInvalid, id,
			fmt.Sprintf("update manifest identity %s does not match existing %s", newManifest.ID, id))
	}

	if err := ValidateTransition(rec.State, plugin.StateUpdating); err != nil {
		return err
	}

	if err := m.val.Validate(newManifest); err != nil {
		return err
	}

	wasActive := rec.State == plugin.StateActive
	if err := m.reg.UpdateState(id, plugin.StateUpdating, nil); err != nil {
		return err
	}

	// Stop existing instance if active
	if inst, ok := rec.Instance.(loader.Instance); ok && inst != nil {
		if err := m.ldr.Unload(ctx, inst); err != nil {
			_ = m.reg.UpdateState(id, plugin.StateFailed, err)
			return err
		}
		_ = m.reg.UpdateInstance(id, nil)
	}

	// Update manifest
	if err := m.reg.UpdateManifest(id, newManifest); err != nil {
		_ = m.reg.UpdateState(id, plugin.StateFailed, err)
		return err
	}

	if wasActive {
		// Re-activate with new manifest
		newInst, err := m.ldr.Load(ctx, newManifest, rec.Directory, m.host)
		if err != nil {
			_ = m.reg.UpdateState(id, plugin.StateFailed, err)
			return err
		}
		if startErr := newInst.Start(ctx); startErr != nil {
			_ = m.ldr.Unload(ctx, newInst)
			_ = m.reg.UpdateState(id, plugin.StateFailed, startErr)
			return startErr
		}
		_ = m.reg.UpdateInstance(id, newInst)
		_ = m.reg.UpdateState(id, plugin.StateActive, nil)
	} else {
		_ = m.reg.UpdateState(id, plugin.StateRegistered, nil)
	}

	return nil
}

// Unload releases runtime resources of an active plugin, returning it to StateUnloaded.
func (m *Manager) Unload(ctx context.Context, id model.Identity) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec, exists := m.reg.Get(id)
	if !exists {
		return pkgerr.New(pkgerr.CodePluginNotFound, id, fmt.Sprintf("plugin %s not found", id))
	}

	if rec.State == plugin.StateUnloaded {
		return nil
	}

	if err := ValidateTransition(rec.State, plugin.StateUnloaded); err != nil {
		return err
	}

	if inst, ok := rec.Instance.(loader.Instance); ok && inst != nil {
		if err := m.ldr.Unload(ctx, inst); err != nil {
			_ = m.reg.UpdateState(id, plugin.StateFailed, err)
			return err
		}
		_ = m.reg.UpdateInstance(id, nil)
	}

	return m.reg.UpdateState(id, plugin.StateUnloaded, nil)
}

// Remove shuts down, unloads, and unregisters a plugin from the registry.
func (m *Manager) Remove(ctx context.Context, id model.Identity) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec, exists := m.reg.Get(id)
	if !exists {
		return pkgerr.New(pkgerr.CodePluginNotFound, id, fmt.Sprintf("plugin %s not found", id))
	}

	// Unload if running
	if inst, ok := rec.Instance.(loader.Instance); ok && inst != nil {
		if unloadErr := m.ldr.Unload(ctx, inst); unloadErr != nil {
			_ = m.reg.UpdateState(id, plugin.StateFailed, unloadErr)
			return pkgerr.Wrap(pkgerr.CodeUnloadingFailed, id, "failed to unload plugin during removal", unloadErr)
		}
		_ = m.reg.UpdateInstance(id, nil)
	}

	if err := m.reg.UpdateState(id, plugin.StateRemoving, nil); err != nil {
		return err
	}
	if err := m.reg.UpdateState(id, plugin.StateRemoved, nil); err != nil {
		return err
	}

	return m.reg.Unregister(id)
}
