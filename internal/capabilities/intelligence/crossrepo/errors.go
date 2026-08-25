package crossrepo

import (
	"errors"
	"fmt"
)

// ErrorCategory classifies structured cross-repository errors.
type ErrorCategory string

const (
	ErrCatInvalidInput ErrorCategory = "INVALID_INPUT"
	ErrCatUnavailable  ErrorCategory = "UNAVAILABLE"
	ErrCatUnresolved   ErrorCategory = "UNRESOLVED"
	ErrCatAmbiguous    ErrorCategory = "AMBIGUOUS"
	ErrCatValidation   ErrorCategory = "VALIDATION_FAILURE"
	ErrCatInternal     ErrorCategory = "INTERNAL_FAILURE"
)

// CrossRepoError represents a structured, categorized cross-repository error.
type CrossRepoError struct {
	category ErrorCategory
	code     string
	message  string
	err      error
}

// NewCrossRepoError creates a new structured CrossRepoError.
func NewCrossRepoError(category ErrorCategory, code, message string) *CrossRepoError {
	return &CrossRepoError{
		category: category,
		code:     code,
		message:  message,
	}
}

// WrapCrossRepoError wraps an underlying error in a structured CrossRepoError.
func WrapCrossRepoError(category ErrorCategory, code, message string, err error) *CrossRepoError {
	return &CrossRepoError{
		category: category,
		code:     code,
		message:  message,
		err:      err,
	}
}

// Error implements the standard error interface.
func (e *CrossRepoError) Error() string {
	if e == nil {
		return ""
	}
	if e.err != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.category, e.code, e.message, e.err)
	}
	return fmt.Sprintf("[%s:%s] %s", e.category, e.code, e.message)
}

// Category returns the structured classification of the error.
func (e *CrossRepoError) Category() ErrorCategory {
	if e == nil {
		return ""
	}
	return e.category
}

// Code returns the unique machine-readable error code.
func (e *CrossRepoError) Code() string {
	if e == nil {
		return ""
	}
	return e.code
}

// Message returns the human-readable explanation.
func (e *CrossRepoError) Message() string {
	if e == nil {
		return ""
	}
	return e.message
}

// Unwrap returns the wrapped inner error, if any.
func (e *CrossRepoError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// Sentinel domain errors
var (
	ErrNilEngine         = NewCrossRepoError(ErrCatInvalidInput, "ERR_NIL_ENGINE", "cross repo engine is nil")
	ErrNilModel          = NewCrossRepoError(ErrCatInvalidInput, "ERR_NIL_MODEL", "cross repo model is nil")
	ErrInvalidInput      = NewCrossRepoError(ErrCatInvalidInput, "ERR_INVALID_INPUT", "provided input parameter is invalid")
	ErrWorkspaceNotFound = NewCrossRepoError(ErrCatUnresolved, "ERR_WORKSPACE_NOT_FOUND", "workspace was not found")
	ErrRepoNotFound      = NewCrossRepoError(ErrCatUnresolved, "ERR_REPO_NOT_FOUND", "repository was not found in workspace")
	ErrModuleNotFound    = NewCrossRepoError(ErrCatUnresolved, "ERR_MODULE_NOT_FOUND", "module was not found")
	ErrPackageNotFound   = NewCrossRepoError(ErrCatUnresolved, "ERR_PACKAGE_NOT_FOUND", "package was not found")
	ErrFileNotFound      = NewCrossRepoError(ErrCatUnresolved, "ERR_FILE_NOT_FOUND", "file was not found")
	ErrDataUnavailable   = NewCrossRepoError(ErrCatUnavailable, "ERR_DATA_UNAVAILABLE", "required analysis data is unavailable")
	ErrValidationFailure = NewCrossRepoError(ErrCatValidation, "ERR_VALIDATION_FAILURE", "cross repository validation check failed")
)

// IsCategory checks if an error belongs to a specified ErrorCategory.
func IsCategory(err error, cat ErrorCategory) bool {
	if err == nil {
		return false
	}
	var crErr *CrossRepoError
	if errors.As(err, &crErr) {
		return crErr.Category() == cat
	}
	return false
}
