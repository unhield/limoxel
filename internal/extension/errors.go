package extension

import "errors"

var (
	// ErrNilRegistry indicates an operation was attempted on a nil Registry instance.
	ErrNilRegistry = errors.New("extension: registry instance is nil")

	// ErrNilDescriptor indicates an attempt to register a nil Descriptor instance.
	ErrNilDescriptor = errors.New("extension: descriptor instance is nil")

	// ErrInvalidID indicates an extension ID string is empty or invalid.
	ErrInvalidID = errors.New("extension: extension ID is invalid or empty")

	// ErrInvalidName indicates an extension name string is empty or invalid.
	ErrInvalidName = errors.New("extension: extension name is invalid or empty")

	// ErrDuplicateExtension indicates an extension with the same ID is already registered.
	ErrDuplicateExtension = errors.New("extension: extension is already registered")

	// ErrExtensionNotFound indicates the requested extension was not found in the registry.
	ErrExtensionNotFound = errors.New("extension: extension not found")

	// ErrAlreadyActive indicates an attempt to activate an extension that is already active.
	ErrAlreadyActive = errors.New("extension: extension is already active")

	// ErrAlreadyInactive indicates an attempt to deactivate an extension that is already inactive.
	ErrAlreadyInactive = errors.New("extension: extension is already inactive")
)
