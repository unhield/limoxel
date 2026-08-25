package reasoning

import (
	"errors"
	"fmt"
)

// ErrorCategory classifies the operational or logical domain of a reasoning error.
type ErrorCategory string

const (
	ErrCatInvalidInput          ErrorCategory = "invalid_input"
	ErrCatMissingTarget         ErrorCategory = "missing_target"
	ErrCatAmbiguousTarget       ErrorCategory = "ambiguous_target"
	ErrCatIncompatibleSnapshots ErrorCategory = "incompatible_snapshots"
	ErrCatInsufficientEvidence  ErrorCategory = "insufficient_evidence"
	ErrCatTraversalLimit        ErrorCategory = "traversal_limit"
	ErrCatCycleDetected         ErrorCategory = "cycle_detected"
	ErrCatInternal              ErrorCategory = "internal"
)

// ReasoningError represents a structured, deterministic error in the reasoning engine.
type ReasoningError struct {
	Category ErrorCategory
	Message  string
	TargetID string
	Err      error
}

func (e *ReasoningError) Error() string {
	if e == nil {
		return ""
	}
	if e.TargetID != "" {
		return fmt.Sprintf("[%s] target %s: %s", e.Category, e.TargetID, e.Message)
	}
	return fmt.Sprintf("[%s] %s", e.Category, e.Message)
}

func (e *ReasoningError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NewReasoningError creates a typed ReasoningError.
func NewReasoningError(category ErrorCategory, message, targetID string, err error) *ReasoningError {
	return &ReasoningError{
		Category: category,
		Message:  message,
		TargetID: targetID,
		Err:      err,
	}
}

// Sentinel errors for standard error classification
var (
	ErrNilEngine             = errors.New("reasoning engine is nil")
	ErrNilGraphModel         = errors.New("knowledge graph model is nil")
	ErrMissingTarget         = errors.New("target entity is missing or not found")
	ErrAmbiguousTarget       = errors.New("target entity resolution is ambiguous")
	ErrInsufficientEvidence  = errors.New("insufficient evidence to derive engineering conclusion")
	ErrIncompatibleSnapshots = errors.New("baseline and target models are incompatible for comparison")
	ErrCycleDetected         = errors.New("cyclic dependency structure detected")
	ErrTraversalLimit        = errors.New("traversal depth limit exceeded")
)
