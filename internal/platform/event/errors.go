package event

import "errors"

var (
	// ErrEventNil indicates an operation was attempted on a nil Event instance.
	ErrEventNil = errors.New("event: instance is nil")

	// ErrTypeEmpty indicates an empty or invalid Event Type was specified.
	ErrTypeEmpty = errors.New("event: type cannot be empty")

	// ErrIDEmpty indicates an empty Event ID was specified.
	ErrIDEmpty = errors.New("event: ID cannot be empty")

	// ErrMetadataNil indicates a nil Metadata object was passed.
	ErrMetadataNil = errors.New("event: metadata is nil")
)
