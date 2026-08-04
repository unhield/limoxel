package parser

import (
	"fmt"
	"strings"
	"time"
)

// Status represents the completion status of a parse result.
type Status int

const (
	// StatusSuccess indicates the parse pipeline completed successfully.
	StatusSuccess Status = iota

	// StatusFailure indicates the parse pipeline failed.
	StatusFailure

	// StatusPartial indicates the parse pipeline completed with partial results.
	StatusPartial
)

// String returns the human-readable textual representation of the Status.
func (s Status) String() string {
	switch s {
	case StatusSuccess:
		return "SUCCESS"
	case StatusFailure:
		return "FAILURE"
	case StatusPartial:
		return "PARTIAL"
	default:
		return fmt.Sprintf("UNKNOWN_STATUS(%d)", int(s))
	}
}

// Result represents an immutable outcome of a parser pipeline evaluation.
type Result struct {
	descriptor *Descriptor
	pipeline   *Pipeline
	status     Status
	timestamp  time.Time
	metadata   map[string]string
}

// NewResult constructs and validates a new immutable Result.
func NewResult(desc *Descriptor, pipe *Pipeline, status Status, timestamp time.Time, metadata map[string]string) (*Result, error) {
	if desc == nil {
		return nil, ErrNilParser
	}
	if pipe == nil {
		return nil, ErrNilPipeline
	}

	if status < StatusSuccess || status > StatusPartial {
		return nil, fmt.Errorf("%w: invalid status %d", ErrInvalidStatus, int(status))
	}

	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	metaCopy := make(map[string]string)
	for k, v := range metadata {
		cleanK := strings.TrimSpace(k)
		if cleanK != "" {
			metaCopy[cleanK] = v
		}
	}

	return &Result{
		descriptor: desc,
		pipeline:   pipe,
		status:     status,
		timestamp:  timestamp,
		metadata:   metaCopy,
	}, nil
}

// Descriptor returns the Descriptor associated with the Result.
func (r *Result) Descriptor() *Descriptor {
	if r == nil {
		return nil
	}
	return r.descriptor
}

// Pipeline returns the Pipeline associated with the Result.
func (r *Result) Pipeline() *Pipeline {
	if r == nil {
		return nil
	}
	return r.pipeline
}

// Status returns the Status of the Result.
func (r *Result) Status() Status {
	if r == nil {
		return StatusFailure
	}
	return r.status
}

// Timestamp returns the completion timestamp of the Result.
func (r *Result) Timestamp() time.Time {
	if r == nil {
		return time.Time{}
	}
	return r.timestamp
}

// Metadata returns a defensive copy of the Result metadata map.
func (r *Result) Metadata() map[string]string {
	if r == nil || len(r.metadata) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(r.metadata))
	for k, v := range r.metadata {
		cloned[k] = v
	}
	return cloned
}

// Successful reports whether the Result status is StatusSuccess.
func (r *Result) Successful() bool {
	if r == nil {
		return false
	}
	return r.status == StatusSuccess
}
