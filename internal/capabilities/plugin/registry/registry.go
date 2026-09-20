package registry

import (
	"fmt"
	"sort"
	"sync"
	"time"

	pkgerr "github.com/unhield/limoxel/internal/capabilities/plugin/errors"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	"github.com/unhield/limoxel/plugin"
)

// PluginRecord stores the registration, lifecycle state, and runtime binding of a managed plugin.
type PluginRecord struct {
	Manifest    *model.Manifest
	Directory   string
	State       plugin.State
	Instance    any
	LastError   error
	InstalledAt time.Time
	ActivatedAt time.Time
	UpdatedAt   time.Time
}

// Clone creates a defensive copy of the record with an isolated manifest clone.
func (r *PluginRecord) Clone() *PluginRecord {
	if r == nil {
		return nil
	}
	return &PluginRecord{
		Manifest:    r.Manifest.Clone(),
		Directory:   r.Directory,
		State:       r.State,
		Instance:    r.Instance,
		LastError:   r.LastError,
		InstalledAt: r.InstalledAt,
		ActivatedAt: r.ActivatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// GetID implements dependency.Resolvable.
func (r *PluginRecord) GetID() model.Identity {
	if r == nil || r.Manifest == nil {
		return ""
	}
	return r.Manifest.ID
}

// GetVersion implements dependency.Resolvable.
func (r *PluginRecord) GetVersion() version.SemVer {
	if r == nil || r.Manifest == nil {
		return version.SemVer{}
	}
	return r.Manifest.Version
}

// GetDependencies implements dependency.Resolvable.
func (r *PluginRecord) GetDependencies() []model.Dependency {
	if r == nil || r.Manifest == nil {
		return nil
	}
	return r.Manifest.Dependencies
}

// Registry provides a thread-safe in-memory store for managed plugins.
type Registry struct {
	mu      sync.RWMutex
	plugins map[model.Identity]*PluginRecord
}

// NewRegistry constructs an initialized, empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[model.Identity]*PluginRecord),
	}
}

// Register adds a new plugin record to the registry. Fails if the identity is already registered.
func (r *Registry) Register(record *PluginRecord) error {
	if record == nil || record.Manifest == nil {
		return pkgerr.New(pkgerr.CodeManifestInvalid, "", "cannot register nil plugin record or manifest")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	id := record.Manifest.ID
	if _, exists := r.plugins[id]; exists {
		return pkgerr.New(pkgerr.CodeDuplicateIdentity, id, fmt.Sprintf("plugin %s is already registered", id))
	}

	r.plugins[id] = record.Clone()
	return nil
}

// Unregister removes a plugin from the registry.
func (r *Registry) Unregister(id model.Identity) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec, exists := r.plugins[id]
	if !exists {
		return pkgerr.New(pkgerr.CodePluginNotFound, id, fmt.Sprintf("plugin %s not found", id))
	}

	if rec.State == plugin.StateActive {
		return pkgerr.New(pkgerr.CodePluginAlreadyActive, id, fmt.Sprintf("cannot unregister active plugin %s without stopping it first", id))
	}

	delete(r.plugins, id)
	return nil
}

// Get retrieves a defensive copy of a plugin record by identity.
func (r *Registry) Get(id model.Identity) (*PluginRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rec, exists := r.plugins[id]
	if !exists {
		return nil, false
	}
	return rec.Clone(), true
}

// List returns all registered plugins sorted deterministically by identity.
func (r *Registry) List() []*PluginRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*PluginRecord, 0, len(r.plugins))
	for _, rec := range r.plugins {
		out = append(out, rec.Clone())
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Manifest.ID.Less(out[j].Manifest.ID)
	})

	return out
}

// ListByState returns plugins matching the given state, sorted deterministically by identity.
func (r *Registry) ListByState(state plugin.State) []*PluginRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []*PluginRecord
	for _, rec := range r.plugins {
		if rec.State == state {
			out = append(out, rec.Clone())
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Manifest.ID.Less(out[j].Manifest.ID)
	})

	return out
}

// UpdateState updates the lifecycle state and optional error of a registered plugin.
func (r *Registry) UpdateState(id model.Identity, newState plugin.State, lastErr error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec, exists := r.plugins[id]
	if !exists {
		return pkgerr.New(pkgerr.CodePluginNotFound, id, fmt.Sprintf("plugin %s not found in registry", id))
	}

	rec.State = newState
	rec.LastError = lastErr
	rec.UpdatedAt = time.Now().UTC()
	if newState == plugin.StateActive {
		rec.ActivatedAt = rec.UpdatedAt
	}
	return nil
}

// UpdateInstance associates a runtime instance with a registered plugin.
func (r *Registry) UpdateInstance(id model.Identity, instance any) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec, exists := r.plugins[id]
	if !exists {
		return pkgerr.New(pkgerr.CodePluginNotFound, id, fmt.Sprintf("plugin %s not found in registry", id))
	}

	rec.Instance = instance
	rec.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateManifest updates the manifest (e.g. during a version update) of a registered plugin.
func (r *Registry) UpdateManifest(id model.Identity, newManifest *model.Manifest) error {
	if newManifest == nil {
		return pkgerr.New(pkgerr.CodeManifestInvalid, id, "cannot update to nil manifest")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	rec, exists := r.plugins[id]
	if !exists {
		return pkgerr.New(pkgerr.CodePluginNotFound, id, fmt.Sprintf("plugin %s not found in registry", id))
	}

	rec.Manifest = newManifest.Clone()
	rec.UpdatedAt = time.Now().UTC()
	return nil
}

// Count returns the total number of registered plugins.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.plugins)
}
