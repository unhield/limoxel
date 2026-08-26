package cli

import (
	"errors"
	"fmt"
)

// ErrorCategory identifies the class of a CLI error.
type ErrorCategory string

const (
	// ErrCatUsage represents invalid command syntax, missing arguments, or invalid options.
	ErrCatUsage ErrorCategory = "USAGE_ERROR"

	// ErrCatContext represents an invalid or missing repository context.
	ErrCatContext ErrorCategory = "CONTEXT_ERROR"

	// ErrCatExecution represents an error during capability execution.
	ErrCatExecution ErrorCategory = "EXECUTION_ERROR"

	// ErrCatNotFound represents an entity, file, or symbol that was not found.
	ErrCatNotFound ErrorCategory = "NOT_FOUND"

	// ErrCatInternal represents an unexpected internal error.
	ErrCatInternal ErrorCategory = "INTERNAL_ERROR"
)

var (
	// ErrUnknownCommand indicates that the specified command is not registered.
	ErrUnknownCommand = errors.New("cli: unknown command")

	// ErrMissingArgument indicates a required positional argument was omitted.
	ErrMissingArgument = errors.New("cli: missing required argument")

	// ErrInvalidOption indicates an invalid option or option value was provided.
	ErrInvalidOption = errors.New("cli: invalid option")

	// ErrRepositoryNotLoaded indicates the operation requires a loaded repository context.
	ErrRepositoryNotLoaded = errors.New("cli: no repository loaded")

	// ErrNilApp indicates an operation was attempted on a nil App.
	ErrNilApp = errors.New("cli: app instance is nil")

	// ErrNilContext indicates an operation was attempted on a nil Context.
	ErrNilContext = errors.New("cli: context instance is nil")

	// ErrNilCommand indicates an operation was attempted on a nil Command.
	ErrNilCommand = errors.New("cli: command instance is nil")
)

// CLIError represents a structured, actionable CLI failure.
type CLIError struct {
	Category ErrorCategory `json:"category"`
	Message  string        `json:"message"`
	Command  string        `json:"command,omitempty"`
	Cause    error         `json:"cause,omitempty"`
	ExitCode ExitCode      `json:"exit_code"`
}

// NewCLIError creates an initialized CLIError with category, message, command, cause, and exit code.
func NewCLIError(category ErrorCategory, message string, command string, cause error, exitCode ExitCode) *CLIError {
	return &CLIError{
		Category: category,
		Message:  message,
		Command:  command,
		Cause:    cause,
		ExitCode: exitCode,
	}
}

// Error returns the human-readable formatted error message.
func (e *CLIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Category, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Category, e.Message)
}

// Unwrap returns the underlying cause error.
func (e *CLIError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// UsageError returns a standardized CLIError for invalid command usage.
func UsageError(cmd string, msg string) *CLIError {
	return NewCLIError(ErrCatUsage, msg, cmd, ErrMissingArgument, ExitUsage)
}

// OptionError returns a standardized CLIError for invalid options.
func OptionError(cmd string, msg string) *CLIError {
	return NewCLIError(ErrCatUsage, msg, cmd, ErrInvalidOption, ExitUsage)
}

// ContextError returns a standardized CLIError for missing or invalid repository context.
func ContextError(cmd string, msg string) *CLIError {
	return NewCLIError(ErrCatContext, msg, cmd, ErrRepositoryNotLoaded, ExitFailure)
}

// ExecutionError returns a standardized CLIError for capability execution failure.
func ExecutionError(cmd string, msg string, cause error) *CLIError {
	return NewCLIError(ErrCatExecution, msg, cmd, cause, ExitFailure)
}
