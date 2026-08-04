package engine

import (
	"fmt"
	"time"
)

// ExecutionResult represents an immutable result of an engine pipeline execution flow.
type ExecutionResult struct {
	pipeline    *Pipeline
	completed   []PipelineStage
	failedStage PipelineStage
	success     bool
	duration    time.Duration
	err         error
}

// NewExecutionResult constructs a new immutable ExecutionResult instance.
func NewExecutionResult(pipeline *Pipeline, completed []PipelineStage, failedStage PipelineStage, success bool, duration time.Duration, err error) *ExecutionResult {
	clonedCompleted := make([]PipelineStage, len(completed))
	copy(clonedCompleted, completed)

	return &ExecutionResult{
		pipeline:    pipeline,
		completed:   clonedCompleted,
		failedStage: failedStage,
		success:     success,
		duration:    duration,
		err:         err,
	}
}

// Pipeline returns the Pipeline associated with the ExecutionResult.
func (r *ExecutionResult) Pipeline() *Pipeline {
	if r == nil {
		return nil
	}
	return r.pipeline
}

// CompletedStages returns a defensive copy of completed PipelineStage items.
func (r *ExecutionResult) CompletedStages() []PipelineStage {
	if r == nil || len(r.completed) == 0 {
		return nil
	}
	cloned := make([]PipelineStage, len(r.completed))
	copy(cloned, r.completed)
	return cloned
}

// FailedStage returns the stage where execution failed, or StageNone if successful.
func (r *ExecutionResult) FailedStage() PipelineStage {
	if r == nil {
		return StageNone
	}
	return r.failedStage
}

// Success reports whether the execution completed all stages successfully.
func (r *ExecutionResult) Success() bool {
	if r == nil {
		return false
	}
	return r.success
}

// Duration returns the total execution time duration.
func (r *ExecutionResult) Duration() time.Duration {
	if r == nil {
		return 0
	}
	return r.duration
}

// Err returns the underlying execution error, if any.
func (r *ExecutionResult) Err() error {
	if r == nil {
		return nil
	}
	return r.err
}

// String returns a concise human-readable representation of the ExecutionResult.
func (r *ExecutionResult) String() string {
	if r == nil {
		return "ExecutionResult<nil>"
	}
	if r.success {
		return fmt.Sprintf("ExecutionResult<SUCCESS>(completed=%d, dur=%v)", len(r.completed), r.duration)
	}
	return fmt.Sprintf("ExecutionResult<FAILED>(stage=%s, err=%v)", r.failedStage, r.err)
}
