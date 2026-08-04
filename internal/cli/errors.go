package cli

import "errors"

var (
	// ErrNilBootstrap indicates an operation was attempted on a nil Bootstrap instance.
	ErrNilBootstrap = errors.New("cli: bootstrap instance is nil")

	// ErrNilConfig indicates an attempt to construct a Bootstrap with a nil Config.
	ErrNilConfig = errors.New("cli: config instance is nil")

	// ErrInvalidConfig indicates a CLI configuration is empty or invalid.
	ErrInvalidConfig = errors.New("cli: configuration is invalid")

	// ErrBootstrapFailed indicates CLI bootstrap engine initialization failed.
	ErrBootstrapFailed = errors.New("cli: bootstrap initialization failed")

	// ErrAlreadyInitialized indicates an attempt to initialize a Bootstrap instance that is already initialized.
	ErrAlreadyInitialized = errors.New("cli: bootstrap is already initialized")

	// ErrNilCommandRegistry indicates an operation was attempted on a nil CommandRegistry instance.
	ErrNilCommandRegistry = errors.New("cli: command registry instance is nil")

	// ErrNilCommandDescriptor indicates an attempt to register a nil CommandDescriptor instance.
	ErrNilCommandDescriptor = errors.New("cli: command descriptor instance is nil")

	// ErrInvalidID indicates a command ID string is empty or invalid.
	ErrInvalidID = errors.New("cli: command ID is invalid or empty")

	// ErrInvalidName indicates a command name string is empty or invalid.
	ErrInvalidName = errors.New("cli: command name is invalid or empty")

	// ErrDuplicateCommand indicates a command ID or alias is already registered.
	ErrDuplicateCommand = errors.New("cli: command is already registered")

	// ErrCommandNotFound indicates the requested command was not found in the registry.
	ErrCommandNotFound = errors.New("cli: command not found")
)
