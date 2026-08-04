package bootstrap

import "errors"

var (
	// ErrBootstrapperNil indicates an operation was invoked on a nil Bootstrapper.
	ErrBootstrapperNil = errors.New("bootstrap: bootstrapper instance is nil")

	// ErrNilContext indicates a nil context was passed to the bootstrap execution.
	ErrNilContext = errors.New("bootstrap: context is nil")

	// ErrPrerequisitesFailed indicates environment or prerequisite validation failed during startup.
	ErrPrerequisitesFailed = errors.New("bootstrap: prerequisite validation failed")

	// ErrRuntimeCreationFailed indicates Runtime instantiation failed.
	ErrRuntimeCreationFailed = errors.New("bootstrap: runtime creation failed")

	// ErrRuntimeValidationFailed indicates Runtime pre-startup validation failed.
	ErrRuntimeValidationFailed = errors.New("bootstrap: runtime validation failed")

	// ErrRuntimeInitializationFailed indicates Runtime initialization stage failed.
	ErrRuntimeInitializationFailed = errors.New("bootstrap: runtime initialization failed")

	// ErrRuntimePreparationFailed indicates Runtime preparation stage failed.
	ErrRuntimePreparationFailed = errors.New("bootstrap: runtime preparation failed")

	// ErrRuntimeStartupFailed indicates Runtime startup stage failed.
	ErrRuntimeStartupFailed = errors.New("bootstrap: runtime startup failed")

	// ErrRuntimeVerificationFailed indicates Runtime verification failed post-startup.
	ErrRuntimeVerificationFailed = errors.New("bootstrap: runtime verification failed")
)
