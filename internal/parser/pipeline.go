package parser

import "fmt"

// PipelineStage represents a generic, implementation-agnostic phase in the parser pipeline.
type PipelineStage int

const (
	// StagePrepare defines the initial pipeline preparation phase.
	StagePrepare PipelineStage = iota

	// StageProcess defines the core parser processing phase.
	StageProcess

	// StageFinalize defines the pipeline finalization phase.
	StageFinalize
)

// String returns the human-readable textual representation of the PipelineStage.
func (s PipelineStage) String() string {
	switch s {
	case StagePrepare:
		return "PREPARE"
	case StageProcess:
		return "PROCESS"
	case StageFinalize:
		return "FINALIZE"
	default:
		return fmt.Sprintf("UNKNOWN_STAGE(%d)", int(s))
	}
}

// CanonicalPipelineStages returns the immutable slice of all pipeline stages in deterministic execution order.
func CanonicalPipelineStages() []PipelineStage {
	return []PipelineStage{
		StagePrepare,
		StageProcess,
		StageFinalize,
	}
}

// Pipeline defines an immutable, generic processing workflow model for a parser descriptor.
type Pipeline struct {
	descriptor *Descriptor
	stages     []PipelineStage
}

// NewPipeline constructs and validates a new immutable Pipeline model for desc using stages.
// If stages is empty, default CanonicalPipelineStages() are used.
func NewPipeline(desc *Descriptor, stages ...PipelineStage) (*Pipeline, error) {
	if desc == nil {
		return nil, ErrNilParser
	}

	activeStages := stages
	if len(activeStages) == 0 {
		activeStages = CanonicalPipelineStages()
	}

	if err := ValidateStageSequence(activeStages); err != nil {
		return nil, err
	}

	clonedStages := make([]PipelineStage, len(activeStages))
	copy(clonedStages, activeStages)

	return &Pipeline{
		descriptor: desc,
		stages:     clonedStages,
	}, nil
}

// Descriptor returns the Descriptor associated with the Pipeline.
func (p *Pipeline) Descriptor() *Descriptor {
	if p == nil {
		return nil
	}
	return p.descriptor
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

// StageCount returns the number of stages defined in the pipeline.
func (p *Pipeline) StageCount() int {
	if p == nil {
		return 0
	}
	return len(p.stages)
}

// ValidateStageSequence checks that the given stage sequence follows strict monotonic canonical ordering.
func ValidateStageSequence(stages []PipelineStage) error {
	if len(stages) == 0 {
		return ErrInvalidPipelineSequence
	}

	lastStage := PipelineStage(-1)
	for _, stage := range stages {
		if stage < StagePrepare || stage > StageFinalize {
			return fmt.Errorf("%w: unknown stage %d", ErrInvalidPipelineSequence, int(stage))
		}
		if stage <= lastStage {
			return fmt.Errorf("%w: stages must be strictly ordered (%s after %s)", ErrInvalidPipelineSequence, stage, lastStage)
		}
		lastStage = stage
	}

	return nil
}
