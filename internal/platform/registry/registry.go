package registry

import (
	"fmt"
	"sync"
	"time"
)

// Registry is the authoritative, thread-safe catalog of platform components.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]Entry
	order   []string
}

// New creates and initializes a new explicit Registry instance.
func New() *Registry {
	return &Registry{
		entries: make(map[string]Entry),
		order:   make([]string, 0),
	}
}

// Register adds a new component Entry to the registry. Registration order is preserved.
func (r *Registry) Register(entry Entry) error {
	if r == nil {
		return ErrRegistryNil
	}

	if entry.Name == "" {
		return ErrEmptyName
	}

	if entry.Type == "" {
		return ErrEmptyType
	}

	if entry.Instance == nil {
		return ErrEntryNil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.entries[entry.Name]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateComponent, entry.Name)
	}

	record := entry.Clone()
	if record.RegisteredAt.IsZero() {
		record.RegisteredAt = time.Now()
	}

	r.entries[entry.Name] = record
	r.order = append(r.order, entry.Name)
	return nil
}

// RegisterComponent is a helper function to construct and register a component entry.
func (r *Registry) RegisterComponent(name string, compType ComponentType, instance any, meta Metadata) error {
	return r.Register(Entry{
		Name:     name,
		Type:     compType,
		Instance: instance,
		Metadata: meta,
	})
}

// Unregister removes a component by name from the registry while maintaining deterministic order.
func (r *Registry) Unregister(name string) error {
	if r == nil {
		return ErrRegistryNil
	}
	if name == "" {
		return ErrEmptyName
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.entries[name]; !exists {
		return fmt.Errorf("%w: %s", ErrComponentNotFound, name)
	}

	delete(r.entries, name)

	for i, n := range r.order {
		if n == name {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}

	return nil
}

// Get retrieves a defensive copy of a registered component Entry by name.
func (r *Registry) Get(name string) (Entry, error) {
	if r == nil {
		return Entry{}, ErrRegistryNil
	}
	if name == "" {
		return Entry{}, ErrEmptyName
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.entries[name]
	if !exists {
		return Entry{}, fmt.Errorf("%w: %s", ErrComponentNotFound, name)
	}

	return entry.Clone(), nil
}

// GetByType returns defensive copies of all registered entries matching the target ComponentType in registration order.
func (r *Registry) GetByType(targetType ComponentType) []Entry {
	if r == nil || targetType == "" {
		return []Entry{}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make([]Entry, 0)
	for _, name := range r.order {
		entry := r.entries[name]
		if entry.Type == targetType {
			results = append(results, entry.Clone())
		}
	}

	return results
}

// List returns defensive copies of all registered entries in deterministic registration order.
func (r *Registry) List() []Entry {
	if r == nil {
		return []Entry{}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make([]Entry, 0, len(r.order))
	for _, name := range r.order {
		results = append(results, r.entries[name].Clone())
	}

	return results
}

// Has reports whether a component with the given name exists in the registry.
func (r *Registry) Has(name string) bool {
	if r == nil || name == "" {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.entries[name]
	return exists
}

// Count returns the total number of registered components.
func (r *Registry) Count() int {
	if r == nil {
		return 0
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.entries)
}
