package errors

import (
	"fmt"
	"strings"
)

// Code represents a standardized, canonical error code classification string.
type Code string

const (
	// CodeInternal designates an internal system or platform failure.
	CodeInternal Code = "ERR_INTERNAL"

	// CodeInvalidInput designates invalid argument or input parameters.
	CodeInvalidInput Code = "ERR_INVALID_INPUT"

	// CodeNotFound designates a requested resource or key was not found.
	CodeNotFound Code = "ERR_NOT_FOUND"

	// CodeAlreadyExists designates a duplicate resource or entity.
	CodeAlreadyExists Code = "ERR_ALREADY_EXISTS"

	// CodeUnauthorized designates an unauthenticated operation attempt.
	CodeUnauthorized Code = "ERR_UNAUTHORIZED"

	// CodeForbidden designates an unauthorized operation attempt.
	CodeForbidden Code = "ERR_FORBIDDEN"

	// CodeTimeout designates an operation timeout.
	CodeTimeout Code = "ERR_TIMEOUT"

	// CodeCanceled designates an operation context cancellation.
	CodeCanceled Code = "ERR_CANCELED"

	// CodeNotImplemented designates an unimplemented feature or method.
	CodeNotImplemented Code = "ERR_NOT_IMPLEMENTED"

	// CodeUnavailable designates a temporary system or service unavailability.
	CodeUnavailable Code = "ERR_UNAVAILABLE"
)

// String returns the string representation of Code.
func (c Code) String() string {
	return string(c)
}

// ValidateCode verifies that a Code string is non-empty and well-formed.
func ValidateCode(code Code) error {
	if code == "" {
		return ErrCodeInvalid
	}
	if strings.Contains(string(code), " ") {
		return fmt.Errorf("%w: code cannot contain spaces", ErrCodeInvalid)
	}
	return nil
}
