package parser

import (
	"fmt"
	"strings"
	"sync"
)

// Registry represents a thread-safe, central catalog of registered parser descriptors and their lifecycle state.
type Registry struct {
	mu          sync.RWMutex
	descriptors map[string]*Descriptor
	states      map[string]State
	order       []string
}

// NewRegistry constructs a new empty Registry instance.
func NewRegistry() *Registry {
	return &Registry{
		descriptors: make(map[string]*Descriptor),
		states:      make(map[string]State),
		order:       make([]string, 0),
	}
}

// Register explicitly registers a Descriptor in the Registry in StateRegistered.
// It returns ErrDuplicateParser if a parser with the same ID is already registered.
func (r *Registry) Register(desc *Descriptor) error {
	if r == nil {
		return ErrNilRegistry
	}
	if desc == nil {
		return ErrNilParser
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	id := desc.ID()
	if existing, exists := r.descriptors[id]; exists {
		return fmt.Errorf("%w: parser ID '%s' is already registered (name: %s)", ErrDuplicateParser, id, existing.Name())
	}

	r.descriptors[id] = desc
	r.states[id] = StateRegistered
	r.order = append(r.order, id)
	return nil
}

// Activate transitions a registered parser from StateRegistered or StateInactive to StateActive.
func (r *Registry) Activate(id string) error {
	if r == nil {
		return ErrNilRegistry
	}

	cleanID := strings.ToLower(strings.TrimSpace(id))
	if cleanID == "" {
		return ErrInvalidID
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.descriptors[cleanID]; !exists {
		return fmt.Errorf("%w: %s", ErrParserNotFound, cleanID)
	}

	if r.states[cleanID] == StateActive {
		return fmt.Errorf("%w: parser '%s' is already active", ErrAlreadyActive, cleanID)
	}

	r.states[cleanID] = StateActive
	return nil
}

// Deactivate transitions an active parser to StateInactive.
func (r *Registry) Deactivate(id string) error {
	if r == nil {
		return ErrNilRegistry
	}

	cleanID := strings.ToLower(strings.TrimSpace(id))
	if cleanID == "" {
		return ErrInvalidID
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.descriptors[cleanID]; !exists {
		return fmt.Errorf("%w: %s", ErrParserNotFound, cleanID)
	}

	if r.states[cleanID] != StateActive {
		return fmt.Errorf("%w: parser '%s' is already inactive", ErrAlreadyInactive, cleanID)
	}

	r.states[cleanID] = StateInactive
	return nil
}

// State returns the current lifecycle state of a registered parser.
func (r *Registry) State(id string) (State, error) {
	if r == nil {
		return StateRegistered, ErrNilRegistry
	}

	cleanID := strings.ToLower(strings.TrimSpace(id))
	if cleanID == "" {
		return StateRegistered, ErrInvalidID
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	state, exists := r.states[cleanID]
	if !exists {
		return StateRegistered, fmt.Errorf("%w: %s", ErrParserNotFound, cleanID)
	}

	return state, nil
}

// IsActive reports whether the parser with the given ID is in StateActive.
func (r *Registry) IsActive(id string) bool {
	if r == nil {
		return false
	}

	cleanID := strings.ToLower(strings.TrimSpace(id))
	if cleanID == "" {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.states[cleanID] == StateActive
}

// Remove unregisters and removes a parser descriptor and its lifecycle state from the Registry.
func (r *Registry) Remove(id string) error {
	if r == nil {
		return ErrNilRegistry
	}

	cleanID := strings.ToLower(strings.TrimSpace(id))
	if cleanID == "" {
		return ErrInvalidID
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.descriptors[cleanID]; !exists {
		return fmt.Errorf("%w: %s", ErrParserNotFound, cleanID)
	}

	delete(r.descriptors, cleanID)
	delete(r.states, cleanID)

	newOrder := make([]string, 0, len(r.order)-1)
	for _, item := range r.order {
		if item != cleanID {
			newOrder = append(newOrder, item)
		}
	}
	r.order = newOrder

	return nil
}

// Get retrieves a registered Descriptor by its ID in O(1) time.
func (r *Registry) Get(id string) (*Descriptor, error) {
	if r == nil {
		return nil, ErrNilRegistry
	}

	cleanID := strings.ToLower(strings.TrimSpace(id))
	if cleanID == "" {
		return nil, ErrInvalidID
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	desc, exists := r.descriptors[cleanID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrParserNotFound, cleanID)
	}

	return desc, nil
}

// Has reports whether a parser with the specified ID is registered.
func (r *Registry) Has(id string) bool {
	if r == nil {
		return false
	}

	cleanID := strings.ToLower(strings.TrimSpace(id))
	if cleanID == "" {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.descriptors[cleanID]
	return exists
}

// Count returns the total number of registered parsers.
func (r *Registry) Count() int {
	if r == nil {
		return 0
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.descriptors)
}

// List returns a slice of all registered Descriptor objects in deterministic registration order.
func (r *Registry) List() []*Descriptor {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.order) == 0 {
		return nil
	}

	result := make([]*Descriptor, len(r.order))
	for i, id := range r.order {
		result[i] = r.descriptors[id]
	}
	return result
}
