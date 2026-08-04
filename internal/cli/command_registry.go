package cli

import (
	"fmt"
	"strings"
	"sync"
)

// CommandRegistry represents a thread-safe, central catalog of registered CommandDescriptor metadata.
type CommandRegistry struct {
	mu       sync.RWMutex
	commands map[string]*CommandDescriptor
	aliases  map[string]*CommandDescriptor
	order    []string
}

// NewCommandRegistry constructs a new empty CommandRegistry instance.
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]*CommandDescriptor),
		aliases:  make(map[string]*CommandDescriptor),
		order:    make([]string, 0),
	}
}

// Register explicitly registers a CommandDescriptor in the registry.
func (r *CommandRegistry) Register(cmd *CommandDescriptor) error {
	if r == nil {
		return ErrNilCommandRegistry
	}
	if cmd == nil {
		return ErrNilCommandDescriptor
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	id := cmd.ID()
	if existing, exists := r.commands[id]; exists {
		return fmt.Errorf("%w: command ID '%s' is already registered (name: %s)", ErrDuplicateCommand, id, existing.Name())
	}
	if existing, exists := r.aliases[id]; exists {
		return fmt.Errorf("%w: command ID '%s' collides with alias of '%s'", ErrDuplicateCommand, id, existing.ID())
	}

	for _, alias := range cmd.Aliases() {
		if existing, exists := r.commands[alias]; exists {
			return fmt.Errorf("%w: command alias '%s' collides with registered command '%s'", ErrDuplicateCommand, alias, existing.ID())
		}
		if existing, exists := r.aliases[alias]; exists {
			return fmt.Errorf("%w: command alias '%s' collides with alias of '%s'", ErrDuplicateCommand, alias, existing.ID())
		}
	}

	r.commands[id] = cmd
	for _, alias := range cmd.Aliases() {
		r.aliases[alias] = cmd
	}
	r.order = append(r.order, id)

	return nil
}

// Get retrieves a registered CommandDescriptor by its ID or alias string.
func (r *CommandRegistry) Get(idOrAlias string) (*CommandDescriptor, error) {
	if r == nil {
		return nil, ErrNilCommandRegistry
	}

	clean := strings.ToLower(strings.TrimSpace(idOrAlias))
	if clean == "" {
		return nil, ErrInvalidID
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if cmd, exists := r.commands[clean]; exists {
		return cmd, nil
	}
	if cmd, exists := r.aliases[clean]; exists {
		return cmd, nil
	}

	return nil, fmt.Errorf("%w: '%s'", ErrCommandNotFound, clean)
}

// Has reports whether a command with the specified ID or alias is registered.
func (r *CommandRegistry) Has(idOrAlias string) bool {
	if r == nil {
		return false
	}

	clean := strings.ToLower(strings.TrimSpace(idOrAlias))
	if clean == "" {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, exists := r.commands[clean]; exists {
		return true
	}
	_, exists := r.aliases[clean]
	return exists
}

// Count returns the total number of primary registered command descriptors.
func (r *CommandRegistry) Count() int {
	if r == nil {
		return 0
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.commands)
}

// List returns a defensive slice copy of all registered CommandDescriptor objects in deterministic registration order.
func (r *CommandRegistry) List() []*CommandDescriptor {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.order) == 0 {
		return nil
	}

	result := make([]*CommandDescriptor, len(r.order))
	for i, id := range r.order {
		result[i] = r.commands[id]
	}
	return result
}
