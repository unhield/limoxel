package runtime

import "errors"

var (
	// ErrRuntimeNil indicates an operation was invoked on a nil Runtime instance.
	ErrRuntimeNil = errors.New("runtime: instance is nil")

	// ErrStateNil indicates an operation was invoked with a nil StateManager instance.
	ErrStateNil = errors.New("runtime: state manager is nil")

	// ErrInvalidStateTransition indicates an invalid state transition was attempted.
	ErrInvalidStateTransition = errors.New("runtime: invalid state transition")

	// ErrAlreadyRunning indicates the Runtime is already in a running state.
	ErrAlreadyRunning = errors.New("runtime: runtime is already running")

	// ErrNotRunning indicates an operation requiring a running Runtime was called while not running.
	ErrNotRunning = errors.New("runtime: runtime is not running")

	// ErrAlreadyTerminated indicates the Runtime has already reached the terminated state.
	ErrAlreadyTerminated = errors.New("runtime: runtime is already terminated")

	// ErrValidationFailed indicates Runtime validation failed prior to startup.
	ErrValidationFailed = errors.New("runtime: validation failed")

	// ErrStartupFailed indicates an error occurred during Runtime startup.
	ErrStartupFailed = errors.New("runtime: startup failed")

	// ErrShutdownFailed indicates an error occurred during Runtime shutdown.
	ErrShutdownFailed = errors.New("runtime: shutdown failed")

	// ErrContextCanceled indicates the context was canceled during a Runtime operation.
	ErrContextCanceled = errors.New("runtime: context canceled")
)
