package navigation

import (
	"errors"
	"fmt"
)

// NavErrorCategory categorizes navigation failure states.
type NavErrorCategory string

const (
	ErrCatInvalidInput      NavErrorCategory = "INVALID_INPUT"
	ErrCatNotFound          NavErrorCategory = "NOT_FOUND"
	ErrCatUnresolved        NavErrorCategory = "UNRESOLVED"
	ErrCatAmbiguous         NavErrorCategory = "AMBIGUOUS"
	ErrCatValidationFailure NavErrorCategory = "VALIDATION_FAILURE"
	ErrCatInternal          NavErrorCategory = "INTERNAL_FAILURE"
)

// NavigationError is the structured error type for the navigation domain.
type NavigationError struct {
	category NavErrorCategory
	code     string
	message  string
	details  map[string]string
	cause    error
}

// NewNavigationError constructs a structured NavigationError.
func NewNavigationError(category NavErrorCategory, code, message string) *NavigationError {
	return &NavigationError{
		category: category,
		code:     code,
		message:  message,
		details:  make(map[string]string),
	}
}

// WithDetail attaches diagnostic key-value context to the error.
func (e *NavigationError) WithDetail(key, value string) *NavigationError {
	if e.details == nil {
		e.details = make(map[string]string)
	}
	e.details[key] = value
	return e
}

// WithCause wraps an underlying root cause error.
func (e *NavigationError) WithCause(cause error) *NavigationError {
	e.cause = cause
	return e
}

func (e *NavigationError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.category, e.code, e.message, e.cause)
	}
	return fmt.Sprintf("[%s:%s] %s", e.category, e.code, e.message)
}

func (e *NavigationError) Category() NavErrorCategory { return e.category }
func (e *NavigationError) Code() string               { return e.code }
func (e *NavigationError) Message() string            { return e.message }
func (e *NavigationError) Cause() error               { return e.cause }
func (e *NavigationError) Details() map[string]string {
	if e.details == nil {
		return nil
	}
	res := make(map[string]string, len(e.details))
	for k, v := range e.details {
		res[k] = v
	}
	return res
}

func (e *NavigationError) Unwrap() error { return e.cause }

// IsCategory checks whether an error belongs to a specified navigation category.
func IsCategory(err error, category NavErrorCategory) bool {
	var navErr *NavigationError
	if errors.As(err, &navErr) {
		return navErr.Category() == category
	}
	return false
}

// Domain Sentinels
var (
	ErrNilEngine        = NewNavigationError(ErrCatInvalidInput, "ERR_NIL_ENGINE", "navigation engine is nil")
	ErrEmptyTarget      = NewNavigationError(ErrCatInvalidInput, "ERR_EMPTY_TARGET", "target identifier cannot be empty")
	ErrSymbolNotFound   = NewNavigationError(ErrCatNotFound, "ERR_SYMBOL_NOT_FOUND", "target symbol not found in knowledge base")
	ErrTargetAmbiguous  = NewNavigationError(ErrCatAmbiguous, "ERR_TARGET_AMBIGUOUS", "multiple candidate navigation targets exist without tiebreaker")
	ErrTargetUnresolved = NewNavigationError(ErrCatUnresolved, "ERR_TARGET_UNRESOLVED", "navigation target could not be resolved from available data")
	ErrInvalidHierarchy = NewNavigationError(ErrCatValidationFailure, "ERR_INVALID_HIERARCHY", "hierarchy structure contains an invalid cycle or missing parent")
)
