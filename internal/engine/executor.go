package engine

import (
	"fmt"
	"time"

	"github.com/unhield/limoxel/internal/platform/context"
)

var cancelKey = context.NewKey("cancelled")

// Request represents an immutable request to execute an Engine Pipeline.
type Request struct {
	pipeline *Pipeline
	ctx      context.Context
}

// NewRequest constructs and validates a new immutable Request.
func NewRequest(pipeline *Pipeline, ctx context.Context) (*Request, error) {
	if pipeline == nil {
		return nil, ErrNilPipeline
	}
	if ctx == nil {
		return nil, ErrNilContext
	}

	return &Request{
		pipeline: pipeline,
		ctx:      ctx,
	}, nil
}

// Pipeline returns the Pipeline associated with the Request.
func (r *Request) Pipeline() *Pipeline {
	if r == nil {
		return nil
	}
	return r.pipeline
}

// Context returns the platform Context associated with the Request.
func (r *Request) Context() context.Context {
	if r == nil {
		return nil
	}
	return r.ctx
}

// Executor coordinates synchronous, deterministic execution of an Engine Pipeline.
type Executor struct{}

// NewExecutor constructs a new Executor instance.
func NewExecutor() *Executor {
	return &Executor{}
}

// Execute performs synchronous, deterministic execution of req.
func (ex *Executor) Execute(req *Request) (*ExecutionResult, error) {
	if ex == nil {
		return nil, ErrNilExecutor
	}
	if req == nil {
		return nil, ErrNilRequest
	}

	pipe := req.Pipeline()
	if pipe == nil {
		return nil, ErrNilPipeline
	}

	eng := pipe.Engine()
	if eng == nil {
		return nil, ErrNilEngine
	}

	if eng.State() != StateRunning {
		return nil, fmt.Errorf("%w: engine must be in StateRunning to execute, current state: %s", ErrEngineState, eng.State())
	}

	ctx := req.Context()
	startTime := time.Now().UTC()
	stages := pipe.Stages()
	completed := make([]PipelineStage, 0, len(stages))

	for _, stage := range stages {
		if ctx != nil && ctx.Has(cancelKey) {
			err := fmt.Errorf("%w: execution cancelled during stage %s", ErrExecutionCancelled, stage)
			dur := time.Since(startTime)
			return NewExecutionResult(pipe, completed, stage, false, dur, err), err
		}

		if err := executeStage(eng, ctx, stage); err != nil {
			wrappedErr := fmt.Errorf("%w: stage %s failed: %w", ErrExecutionFailed, stage, err)
			dur := time.Since(startTime)
			return NewExecutionResult(pipe, completed, stage, false, dur, wrappedErr), wrappedErr
		}

		completed = append(completed, stage)
	}

	dur := time.Since(startTime)
	return NewExecutionResult(pipe, completed, StageNone, true, dur, nil), nil
}

// executeStage performs synchronous orchestration checks for a single PipelineStage.
func executeStage(eng *Engine, ctx context.Context, stage PipelineStage) error {
	switch stage {
	case StageDiscovery:
		if eng.Workspace() == nil || eng.Filesystem() == nil || eng.Repository() == nil {
			return fmt.Errorf("%w: discovery subsystems uninitialized", ErrMissingSubsystem)
		}
		return nil
	case StageResolution:
		if eng.LanguageRegistry() == nil || eng.ParserRegistry() == nil {
			return fmt.Errorf("%w: resolution registries uninitialized", ErrMissingSubsystem)
		}
		return nil
	case StageExecution:
		return nil
	case StageFinalization:
		return nil
	default:
		return fmt.Errorf("unknown pipeline stage: %s", stage)
	}
}
