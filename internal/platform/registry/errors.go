package registry

import "errors"

var (
	// ErrRegistryNil indicates an operation was invoked on a nil Registry instance.
	ErrRegistryNil = errors.New("registry: instance is nil")

	// ErrEntryNil indicates an attempt to register a nil component instance.
	ErrEntryNil = errors.New("registry: component instance is nil")

	// ErrEmptyName indicates an attempt to register a component with an empty name.
	ErrEmptyName = errors.New("registry: component name cannot be empty")

	// ErrEmptyType indicates an attempt to register a component with an empty type.
	ErrEmptyType = errors.New("registry: component type cannot be empty")

	// ErrDuplicateComponent indicates a component with the specified name is already registered.
	ErrDuplicateComponent = errors.New("registry: duplicate component name")

	// ErrComponentNotFound indicates the requested component was not found in the registry.
	ErrComponentNotFound = errors.New("registry: component not found")
)
