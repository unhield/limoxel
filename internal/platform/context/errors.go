package context

import "errors"

var (
	// ErrContextNil indicates an operation was attempted on a nil Context instance.
	ErrContextNil = errors.New("context: instance is nil")

	// ErrKeyEmpty indicates an empty key was provided.
	ErrKeyEmpty = errors.New("context: key name cannot be empty")

	// ErrKeyNotFound indicates the requested key was not found in the context or its parent chain.
	ErrKeyNotFound = errors.New("context: key not found")

	// ErrTypeMismatch indicates a value conversion failed due to a type mismatch.
	ErrTypeMismatch = errors.New("context: value type mismatch")
)
