package language

import (
	"fmt"
	"strings"
	"sync"
)

// Registry represents a thread-safe, central catalog of supported programming languages and metadata.
type Registry struct {
	mu         sync.RWMutex
	state      State
	languages  map[string]*Language
	extensions map[string]*Language
	filenames  map[string]*Language
	aliases    map[string]*Language
	order      []string
}

// NewRegistry constructs a new empty Registry instance in StateCreated.
func NewRegistry() *Registry {
	return &Registry{
		state:      StateCreated,
		languages:  make(map[string]*Language),
		extensions: make(map[string]*Language),
		filenames:  make(map[string]*Language),
		aliases:    make(map[string]*Language),
		order:      make([]string, 0),
	}
}

// State returns the current operational lifecycle state of the Registry.
func (r *Registry) State() State {
	if r == nil {
		return StateTerminated
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state
}

// Freeze transitions the Registry to StateOperational, freezing all registrations and making it read-only.
// Freeze is idempotent.
func (r *Registry) Freeze() error {
	if r == nil {
		return ErrNilRegistry
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	switch r.state {
	case StateOperational:
		return nil
	case StateTerminated:
		return ErrRegistryTerminated
	case StateCreated:
		r.state = StateOperational
		return nil
	default:
		return fmt.Errorf("language: invalid registry state %s", r.state)
	}
}

// Close transitions the Registry to StateTerminated and releases all internal state.
// Close is idempotent.
func (r *Registry) Close() error {
	if r == nil {
		return ErrNilRegistry
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state == StateTerminated {
		return nil
	}

	r.state = StateTerminated
	r.languages = nil
	r.extensions = nil
	r.filenames = nil
	r.aliases = nil
	r.order = nil
	return nil
}

// Register explicitly registers a Language descriptor and its metadata in the Registry.
// It returns ErrRegistryFrozen if the registry is not in StateCreated.
func (r *Registry) Register(lang *Language) error {
	if r == nil {
		return ErrNilRegistry
	}
	if lang == nil {
		return ErrNilLanguage
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state == StateTerminated {
		return ErrRegistryTerminated
	}
	if r.state != StateCreated {
		return fmt.Errorf("%w: cannot register language in state %s", ErrRegistryFrozen, r.state)
	}

	id := lang.ID()
	if _, exists := r.languages[id]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateLanguage, id)
	}

	r.languages[id] = lang
	r.order = append(r.order, id)

	for _, ext := range lang.Extensions() {
		if _, exists := r.extensions[ext]; !exists {
			r.extensions[ext] = lang
		}
	}

	for _, fn := range lang.Filenames() {
		lowerFn := strings.ToLower(fn)
		if _, exists := r.filenames[lowerFn]; !exists {
			r.filenames[lowerFn] = lang
		}
	}

	for _, alias := range lang.Aliases() {
		if _, exists := r.aliases[alias]; !exists {
			r.aliases[alias] = lang
		}
	}

	return nil
}

// Get retrieves a registered Language by its ID.
func (r *Registry) Get(id string) (*Language, error) {
	if r == nil {
		return nil, ErrNilRegistry
	}

	cleanID := strings.ToLower(strings.TrimSpace(id))
	if cleanID == "" {
		return nil, ErrInvalidID
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.state == StateTerminated {
		return nil, ErrRegistryTerminated
	}

	lang, exists := r.languages[cleanID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrLanguageNotFound, cleanID)
	}

	return lang, nil
}

// GetByExtension retrieves a registered Language associated with the given file extension (e.g. ".go" or "go").
func (r *Registry) GetByExtension(ext string) (*Language, error) {
	if r == nil {
		return nil, ErrNilRegistry
	}

	cleaned := strings.ToLower(strings.TrimSpace(ext))
	if cleaned == "" {
		return nil, ErrInvalidExtension
	}
	if !strings.HasPrefix(cleaned, ".") {
		cleaned = "." + cleaned
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.state == StateTerminated {
		return nil, ErrRegistryTerminated
	}

	lang, exists := r.extensions[cleaned]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrLanguageNotFound, cleaned)
	}

	return lang, nil
}

// GetByAlias retrieves a registered Language associated with the given alias (e.g. "golang").
func (r *Registry) GetByAlias(alias string) (*Language, error) {
	if r == nil {
		return nil, ErrNilRegistry
	}

	cleaned := strings.ToLower(strings.TrimSpace(alias))
	if cleaned == "" {
		return nil, ErrInvalidAlias
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.state == StateTerminated {
		return nil, ErrRegistryTerminated
	}

	lang, exists := r.aliases[cleaned]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrLanguageNotFound, cleaned)
	}

	return lang, nil
}

// Has reports whether a language with the specified ID is registered.
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

	if r.state == StateTerminated {
		return false
	}

	_, exists := r.languages[cleanID]
	return exists
}

// Count returns the total number of registered languages.
func (r *Registry) Count() int {
	if r == nil {
		return 0
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.state == StateTerminated {
		return 0
	}

	return len(r.languages)
}

// List returns a slice of all registered Language descriptors in deterministic insertion order.
func (r *Registry) List() []*Language {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.state == StateTerminated || len(r.order) == 0 {
		return nil
	}

	result := make([]*Language, len(r.order))
	for i, id := range r.order {
		result[i] = r.languages[id]
	}
	return result
}
