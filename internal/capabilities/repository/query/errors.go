package query

import (
	"errors"
	"fmt"
)

// QueryError represents a structured, categorized query error.
type QueryError struct {
	category ErrorCategory
	code     string
	message  string
	err      error
}

// NewQueryError creates a new structured QueryError.
func NewQueryError(category ErrorCategory, code, message string) *QueryError {
	return &QueryError{
		category: category,
		code:     code,
		message:  message,
	}
}

// WrapQueryError wraps an underlying error in a structured QueryError.
func WrapQueryError(category ErrorCategory, code, message string, err error) *QueryError {
	return &QueryError{
		category: category,
		code:     code,
		message:  message,
		err:      err,
	}
}

// Error implements the standard error interface.
func (e *QueryError) Error() string {
	if e == nil {
		return ""
	}
	if e.err != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.category, e.code, e.message, e.err)
	}
	return fmt.Sprintf("[%s:%s] %s", e.category, e.code, e.message)
}

// Category returns the structured classification of the error.
func (e *QueryError) Category() ErrorCategory {
	if e == nil {
		return ""
	}
	return e.category
}

// Code returns the unique machine-readable error code.
func (e *QueryError) Code() string {
	if e == nil {
		return ""
	}
	return e.code
}

// Message returns the human-readable explanation.
func (e *QueryError) Message() string {
	if e == nil {
		return ""
	}
	return e.message
}

// Unwrap returns the wrapped inner error, if any.
func (e *QueryError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// Sentinel domain errors
var (
	ErrNilService            = NewQueryError(ErrCatInvalidInput, "ERR_NIL_SERVICE", "repository service is nil")
	ErrRepositoryNotLoaded   = NewQueryError(ErrCatNotLoaded, "ERR_NOT_LOADED", "repository has not been loaded into service")
	ErrRepositoryNotFound    = NewQueryError(ErrCatNotFound, "ERR_REPO_NOT_FOUND", "repository was not found at specified path")
	ErrInvalidInput          = NewQueryError(ErrCatInvalidInput, "ERR_INVALID_INPUT", "provided input parameter is invalid")
	ErrInvalidLifecycleState = NewQueryError(ErrCatLifecycle, "ERR_INVALID_LIFECYCLE", "operation invalid for current lifecycle state")
	ErrAnalysisUnavailable   = NewQueryError(ErrCatUnavailable, "ERR_ANALYSIS_UNAVAILABLE", "requested analysis model is unavailable")
	ErrSymbolNotFound        = NewQueryError(ErrCatNotFound, "ERR_SYMBOL_NOT_FOUND", "symbol was not found")
	ErrNodeNotFound          = NewQueryError(ErrCatNotFound, "ERR_NODE_NOT_FOUND", "graph node was not found")
	ErrAmbiguousSymbol       = NewQueryError(ErrCatNotFound, "ERR_AMBIGUOUS_SYMBOL", "multiple symbols matched query")
	ErrInvalidTraversal      = NewQueryError(ErrCatInvalidInput, "ERR_INVALID_TRAVERSAL", "traversal parameters are invalid")
	ErrMaxDepthExceeded      = NewQueryError(ErrCatInvalidInput, "ERR_MAX_DEPTH_EXCEEDED", "traversal max depth exceeds limit (100)")
	ErrEmptyQuery            = NewQueryError(ErrCatInvalidInput, "ERR_EMPTY_QUERY", "search query string cannot be empty")
	ErrServiceClosed         = NewQueryError(ErrCatLifecycle, "ERR_SERVICE_CLOSED", "repository service is closed")
)

// IsCategory checks if an error belongs to a specified ErrorCategory.
func IsCategory(err error, cat ErrorCategory) bool {
	if err == nil {
		return false
	}
	var qErr *QueryError
	if errors.As(err, &qErr) {
		return qErr.Category() == cat
	}
	return false
}
