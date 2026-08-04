package engine

import "errors"

var (
	// ErrNilEngine indicates an operation was attempted on a nil Engine instance.
	ErrNilEngine = errors.New("engine: engine instance is nil")

	// ErrNilConfig indicates an attempt to construct an Engine with a nil Config.
	ErrNilConfig = errors.New("engine: config instance is nil")

	// ErrInvalidConfig indicates an engine configuration is empty or invalid.
	ErrInvalidConfig = errors.New("engine: configuration is invalid")

	// ErrNilSubsystem indicates an attempt to attach a nil subsystem component.
	ErrNilSubsystem = errors.New("engine: subsystem component is nil")

	// ErrMissingSubsystem indicates a required subsystem was not registered during preparation.
	ErrMissingSubsystem = errors.New("engine: required subsystem component is missing")

	// ErrEngineState indicates an invalid engine state transition or operation in current state.
	ErrEngineState = errors.New("engine: invalid engine state transition")

	// ErrNilPipeline indicates an operation was attempted on a nil Pipeline instance.
	ErrNilPipeline = errors.New("engine: pipeline instance is nil")

	// ErrInvalidPipelineSequence indicates an engine pipeline stage sequence is empty or out of canonical order.
	ErrInvalidPipelineSequence = errors.New("engine: invalid pipeline stage sequence")

	// ErrNilExecutor indicates an operation was attempted on a nil Executor instance.
	ErrNilExecutor = errors.New("engine: executor instance is nil")

	// ErrNilRequest indicates an operation was attempted on a nil Request instance.
	ErrNilRequest = errors.New("engine: execution request is nil")

	// ErrNilContext indicates an operation was attempted with a nil platform Context.
	ErrNilContext = errors.New("engine: context instance is nil")

	// ErrExecutionFailed indicates an engine pipeline execution flow failed.
	ErrExecutionFailed = errors.New("engine: execution flow failed")

	// ErrExecutionCancelled indicates an engine pipeline execution flow was cancelled by context.
	ErrExecutionCancelled = errors.New("engine: execution flow cancelled")
)
