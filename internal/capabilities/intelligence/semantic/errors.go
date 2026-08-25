package semantic

import (
	"errors"
	"fmt"
)

// ErrorCategory identifies the classification of a semantic error.
type ErrorCategory string

const (
	ErrCatInvalidInput ErrorCategory = "INVALID_INPUT"
	ErrCatUnavailable  ErrorCategory = "UNAVAILABLE"
	ErrCatUnresolved   ErrorCategory = "UNRESOLVED"
	ErrCatAmbiguous    ErrorCategory = "AMBIGUOUS"
	ErrCatValidation   ErrorCategory = "VALIDATION_FAILURE"
	ErrCatInternal     ErrorCategory = "INTERNAL_FAILURE"
)

// SemanticError represents a structured, categorized semantic error.
type SemanticError struct {
	category ErrorCategory
	code     string
	message  string
	err      error
}

// NewSemanticError creates a new structured SemanticError.
func NewSemanticError(category ErrorCategory, code, message string) *SemanticError {
	return &SemanticError{
		category: category,
		code:     code,
		message:  message,
	}
}

// WrapSemanticError wraps an underlying error in a structured SemanticError.
func WrapSemanticError(category ErrorCategory, code, message string, err error) *SemanticError {
	return &SemanticError{
		category: category,
		code:     code,
		message:  message,
		err:      err,
	}
}

// Error implements the standard error interface.
func (e *SemanticError) Error() string {
	if e == nil {
		return ""
	}
	if e.err != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.category, e.code, e.message, e.err)
	}
	return fmt.Sprintf("[%s:%s] %s", e.category, e.code, e.message)
}

// Category returns the structured classification of the error.
func (e *SemanticError) Category() ErrorCategory {
	if e == nil {
		return ""
	}
	return e.category
}

// Code returns the unique machine-readable error code.
func (e *SemanticError) Code() string {
	if e == nil {
		return ""
	}
	return e.code
}

// Message returns the human-readable explanation.
func (e *SemanticError) Message() string {
	if e == nil {
		return ""
	}
	return e.message
}

// Unwrap returns the wrapped inner error, if any.
func (e *SemanticError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// Sentinel domain errors
var (
	ErrNilEngine       = NewSemanticError(ErrCatInvalidInput, "ERR_NIL_ENGINE", "semantic engine is nil")
	ErrNilModel        = NewSemanticError(ErrCatInvalidInput, "ERR_NIL_MODEL", "semantic model is nil")
	ErrInvalidInput    = NewSemanticError(ErrCatInvalidInput, "ERR_INVALID_INPUT", "provided input parameter is invalid")
	ErrSymbolNotFound  = NewSemanticError(ErrCatUnresolved, "ERR_SYMBOL_NOT_FOUND", "semantic symbol was not found")
	ErrTypeNotFound    = NewSemanticError(ErrCatUnresolved, "ERR_TYPE_NOT_FOUND", "semantic type was not found")
	ErrScopeNotFound   = NewSemanticError(ErrCatUnresolved, "ERR_SCOPE_NOT_FOUND", "semantic scope was not found")
	ErrAmbiguousEntity = NewSemanticError(ErrCatAmbiguous, "ERR_AMBIGUOUS_ENTITY", "multiple entities matched query")
	ErrInvalidType     = NewSemanticError(ErrCatValidation, "ERR_INVALID_TYPE", "semantic type is invalid or inconsistent")
	ErrInvalidScope    = NewSemanticError(ErrCatValidation, "ERR_INVALID_SCOPE", "scope hierarchy or reference is invalid")
	ErrDataUnavailable = NewSemanticError(ErrCatUnavailable, "ERR_DATA_UNAVAILABLE", "required repository analysis data is unavailable")
)

// IsCategory checks if an error belongs to a specified ErrorCategory.
func IsCategory(err error, cat ErrorCategory) bool {
	if err == nil {
		return false
	}
	var semErr *SemanticError
	if errors.As(err, &semErr) {
		return semErr.Category() == cat
	}
	return false
}
