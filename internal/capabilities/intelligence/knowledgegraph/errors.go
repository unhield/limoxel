package knowledgegraph

import (
	"errors"
	"fmt"
)

// Sentinel errors for Stage 5 Knowledge Graph Intelligence.
var (
	ErrNilEngine             = errors.New("knowledge graph engine is nil")
	ErrNilGraph              = errors.New("knowledge graph model is nil")
	ErrEntityNotFound        = errors.New("graph entity not found")
	ErrAmbiguousEntity       = errors.New("ambiguous entity identifier")
	ErrInvalidQuery          = errors.New("invalid graph query specification")
	ErrTraversalLimitReached = errors.New("graph traversal limit exceeded")
	ErrInsufficientEvidence  = errors.New("insufficient repository evidence for relationship")
	ErrCycleDetected         = errors.New("cyclic path detected during acyclic traversal")
)

// ErrorCategory categorizes knowledge graph errors.
type ErrorCategory string

const (
	ErrCatInvalidQuery         ErrorCategory = "INVALID_QUERY"
	ErrCatEntityNotFound       ErrorCategory = "ENTITY_NOT_FOUND"
	ErrCatAmbiguousEntity      ErrorCategory = "AMBIGUOUS_ENTITY"
	ErrCatTraversalLimit       ErrorCategory = "TRAVERSAL_LIMIT"
	ErrCatInsufficientEvidence ErrorCategory = "INSUFFICIENT_EVIDENCE"
	ErrCatCycleDetected        ErrorCategory = "CYCLE_DETECTED"
	ErrCatInternal             ErrorCategory = "INTERNAL_ERROR"
)

// KnowledgeGraphError represents a structured, contextual knowledge graph error.
type KnowledgeGraphError struct {
	Category ErrorCategory `json:"category"`
	Message  string        `json:"message"`
	EntityID string        `json:"entity_id,omitempty"`
	Err      error         `json:"-"`
}

// NewKnowledgeGraphError constructs a structured KnowledgeGraphError.
func NewKnowledgeGraphError(cat ErrorCategory, msg, entityID string, err error) *KnowledgeGraphError {
	return &KnowledgeGraphError{
		Category: cat,
		Message:  msg,
		EntityID: entityID,
		Err:      err,
	}
}

// Error formats the error message.
func (e *KnowledgeGraphError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.EntityID != "" {
		return fmt.Sprintf("[%s] %s (entity: %s)", e.Category, e.Message, e.EntityID)
	}
	return fmt.Sprintf("[%s] %s", e.Category, e.Message)
}

// Unwrap returns the underlying error.
func (e *KnowledgeGraphError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
