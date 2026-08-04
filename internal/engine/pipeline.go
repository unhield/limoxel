package engine

import "fmt"

// PipelineStage represents an implementation-neutral orchestration stage in the Engine processing pipeline.
type PipelineStage int

const (
	// StageNone represents an uninitialized or non-existent pipeline stage.
	StageNone PipelineStage = -1

	// StageDiscovery defines the subsystem workspace and repository discovery orchestration stage.
	StageDiscovery PipelineStage = iota

	// StageResolution defines the subsystem language and parser descriptor resolution stage.
	StageResolution

	// StageExecution defines the subsystem processing and pipeline execution stage.
	StageExecution

	// StageFinalization defines the subsystem finalization and completion stage.
	StageFinalization
)

// String returns the human-readable textual representation of the PipelineStage.
func (s PipelineStage) String() string {
	switch s {
	case StageNone:
		return "NONE"
	case StageDiscovery:
		return "DISCOVERY"
	case StageResolution:
		return "RESOLUTION"
	case StageExecution:
		return "EXECUTION"
	case StageFinalization:
		return "FINALIZATION"
	default:
		return fmt.Sprintf("UNKNOWN_STAGE(%d)", int(s))
	}
}

// CanonicalPipelineStages returns an immutable slice of all Engine pipeline stages in deterministic order.
func CanonicalPipelineStages() []PipelineStage {
	return []PipelineStage{
		StageDiscovery,
		StageResolution,
		StageExecution,
		StageFinalization,
	}
}

// Pipeline defines an immutable processing workflow model for the Engine coordinator.
type Pipeline struct {
	engine *Engine
	stages []PipelineStage
}

// NewPipeline constructs and validates a new immutable Pipeline for eng using stages.
// If stages is empty, default CanonicalPipelineStages() are used.
func NewPipeline(eng *Engine, stages ...PipelineStage) (*Pipeline, error) {
	if eng == nil {
		return nil, ErrNilEngine
	}

	activeStages := stages
	if len(activeStages) == 0 {
		activeStages = CanonicalPipelineStages()
	}

	if err := ValidatePipelineSequence(activeStages); err != nil {
		return nil, err
	}

	cloned := make([]PipelineStage, len(activeStages))
	copy(cloned, activeStages)

	return &Pipeline{
		engine: eng,
		stages: cloned,
	}, nil
}

// Engine returns the Engine associated with the Pipeline.
func (p *Pipeline) Engine() *Engine {
	if p == nil {
		return nil
	}
	return p.engine
}

// Stages returns a defensive copy of the PipelineStage sequence.
func (p *Pipeline) Stages() []PipelineStage {
	if p == nil || len(p.stages) == 0 {
		return nil
	}
	cloned := make([]PipelineStage, len(p.stages))
	copy(cloned, p.stages)
	return cloned
}

// StageCount returns the total number of stages defined in the pipeline.
func (p *Pipeline) StageCount() int {
	if p == nil {
		return 0
	}
	return len(p.stages)
}

// ValidatePipelineSequence checks that the given stage sequence follows strict monotonic canonical ordering.
func ValidatePipelineSequence(stages []PipelineStage) error {
	if len(stages) == 0 {
		return ErrInvalidPipelineSequence
	}

	lastStage := StageNone
	for _, stage := range stages {
		if stage < StageDiscovery || stage > StageFinalization {
			return fmt.Errorf("%w: unknown stage %d", ErrInvalidPipelineSequence, int(stage))
		}
		if stage <= lastStage {
			return fmt.Errorf("%w: stages must be strictly ordered (%s after %s)", ErrInvalidPipelineSequence, stage, lastStage)
		}
		lastStage = stage
	}

	return nil
}
