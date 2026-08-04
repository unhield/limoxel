package errors

import "errors"

var (
	// ErrErrorNil indicates an operation was attempted on a nil PlatformError instance.
	ErrErrorNil = errors.New("errors: error instance is nil")

	// ErrCodeInvalid indicates an invalid or empty error code was provided.
	ErrCodeInvalid = errors.New("errors: invalid error code")

	// ErrMessageEmpty indicates an error message string was empty.
	ErrMessageEmpty = errors.New("errors: error message cannot be empty")
)

// Unwrap re-exports standard library errors.Unwrap for platform convenience.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// Is re-exports standard library errors.Is for platform convenience.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As re-exports standard library errors.As for platform convenience.
func As(err error, target any) bool {
	return errors.As(err, target)
}
