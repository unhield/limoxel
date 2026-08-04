package cli

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrNilRouter indicates an operation was attempted on a nil Router instance.
	ErrNilRouter = errors.New("cli: router instance is nil")

	// ErrEmptyIdentifier indicates an empty command identifier was provided for routing.
	ErrEmptyIdentifier = errors.New("cli: command identifier cannot be empty")

	// ErrCommandUnresolved indicates a command identifier could not be resolved.
	ErrCommandUnresolved = errors.New("cli: command identifier unresolved")
)

// Router resolves command identifiers to registered immutable CommandDescriptor objects.
type Router struct {
	registry *CommandRegistry
}

// NewRouter constructs a new Router linked to registry.
func NewRouter(registry *CommandRegistry) (*Router, error) {
	if registry == nil {
		return nil, ErrNilCommandRegistry
	}

	return &Router{
		registry: registry,
	}, nil
}

// Registry returns the CommandRegistry associated with the Router.
func (r *Router) Registry() *CommandRegistry {
	if r == nil {
		return nil
	}
	return r.registry
}

// Resolve normalizes identifier and resolves it to a registered CommandDescriptor.
// It matches canonical IDs and registered aliases in O(1) time through the CommandRegistry.
func (r *Router) Resolve(identifier string) (*CommandDescriptor, error) {
	if r == nil {
		return nil, ErrNilRouter
	}

	clean := strings.ToLower(strings.TrimSpace(identifier))
	if clean == "" {
		return nil, ErrEmptyIdentifier
	}

	desc, err := r.registry.Get(clean)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to resolve '%s': %w", ErrCommandUnresolved, clean, err)
	}

	return desc, nil
}
