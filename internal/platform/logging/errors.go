package logging

import "errors"

var (
	// ErrLoggerNil indicates an operation was attempted on a nil Logger instance.
	ErrLoggerNil = errors.New("logging: logger instance is nil")

	// ErrHandlerNil indicates an attempt to register or use a nil Handler instance.
	ErrHandlerNil = errors.New("logging: handler instance is nil")

	// ErrInvalidLevel indicates an invalid logging level was specified.
	ErrInvalidLevel = errors.New("logging: invalid log level")

	// ErrEmptyMessage indicates a log attempt with an empty message string.
	ErrEmptyMessage = errors.New("logging: log message cannot be empty")

	// ErrClosed indicates an operation was attempted on a closed Logger or Handler.
	ErrClosed = errors.New("logging: subsystem is closed")
)
