package language

import "errors"

var (
	// ErrNilRegistry indicates an operation was attempted on a nil Registry instance.
	ErrNilRegistry = errors.New("language: registry instance is nil")

	// ErrNilLanguage indicates an attempt to register a nil Language instance.
	ErrNilLanguage = errors.New("language: language instance is nil")

	// ErrInvalidID indicates a language ID string is empty or invalid.
	ErrInvalidID = errors.New("language: language ID is invalid or empty")

	// ErrInvalidName indicates a language name string is empty or invalid.
	ErrInvalidName = errors.New("language: language name is invalid or empty")

	// ErrInvalidExtension indicates an extension string is empty or invalid.
	ErrInvalidExtension = errors.New("language: extension is invalid or empty")

	// ErrInvalidAlias indicates an alias string is empty or invalid.
	ErrInvalidAlias = errors.New("language: alias is invalid or empty")

	// ErrInvalidFilename indicates a filename string is empty or invalid.
	ErrInvalidFilename = errors.New("language: filename is invalid or empty")

	// ErrDuplicateLanguage indicates a language with the same ID is already registered.
	ErrDuplicateLanguage = errors.New("language: language is already registered")

	// ErrLanguageNotFound indicates the requested language was not found in the registry.
	ErrLanguageNotFound = errors.New("language: language not found")

	// ErrRegistryFrozen indicates an attempt to register a language in a frozen registry.
	ErrRegistryFrozen = errors.New("language: registry is frozen and read-only")

	// ErrRegistryTerminated indicates an operation was attempted on a terminated registry.
	ErrRegistryTerminated = errors.New("language: registry is terminated")
)
