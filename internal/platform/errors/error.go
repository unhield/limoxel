package errors

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Metadata represents arbitrary string key-value attributes attached to an Error.
type Metadata map[string]string

// Clone creates a deep copy of Metadata to preserve immutability.
func (m Metadata) Clone() Metadata {
	if m == nil {
		return nil
	}
	cloned := make(Metadata, len(m))
	for k, v := range m {
		cloned[k] = v
	}
	return cloned
}

// Error defines the interface contract for platform-wide structured errors.
type Error interface {
	error
	Code() Code
	Severity() Severity
	Message() string
	Unwrap() error
	Metadata() Metadata
	Timestamp() time.Time
	WithMetadata(key, value string) Error
	WithCause(cause error) Error
	Clone() Error
}

// PlatformError is an immutable, thread-safe implementation of Error.
type PlatformError struct {
	mu        sync.RWMutex
	code      Code
	severity  Severity
	message   string
	cause     error
	metadata  Metadata
	timestamp time.Time
}

// New creates a new immutable PlatformError with SeverityError.
func New(code Code, message string) *PlatformError {
	return NewWithSeverity(code, SeverityError, message)
}

// NewWithSeverity creates a new immutable PlatformError with the given Severity.
func NewWithSeverity(code Code, severity Severity, message string) *PlatformError {
	if code == "" {
		code = CodeInternal
	}
	return &PlatformError{
		code:      code,
		severity:  severity,
		message:   message,
		metadata:  make(Metadata),
		timestamp: time.Now(),
	}
}

// Wrap creates a new immutable PlatformError wrapping an existing underlying cause.
func Wrap(cause error, code Code, message string) *PlatformError {
	err := New(code, message)
	err.cause = cause
	return err
}

// Error returns the formatted string representation of the PlatformError.
func (e *PlatformError) Error() string {
	if e == nil {
		return "<nil>"
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("[")
	sb.WriteString(string(e.code))
	sb.WriteString("] ")
	sb.WriteString(e.message)

	if e.cause != nil {
		sb.WriteString(": ")
		sb.WriteString(e.cause.Error())
	}

	return sb.String()
}

// Code returns the error Code.
func (e *PlatformError) Code() Code {
	if e == nil {
		return CodeInternal
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.code
}

// Severity returns the operational Severity.
func (e *PlatformError) Severity() Severity {
	if e == nil {
		return SeverityError
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.severity
}

// Message returns the error message.
func (e *PlatformError) Message() string {
	if e == nil {
		return ""
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.message
}

// Unwrap returns the underlying wrapped cause error.
func (e *PlatformError) Unwrap() error {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cause
}

// Metadata returns a defensive copy of error Metadata.
func (e *PlatformError) Metadata() Metadata {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.metadata.Clone()
}

// Timestamp returns the creation timestamp of the error.
func (e *PlatformError) Timestamp() time.Time {
	if e == nil {
		return time.Time{}
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.timestamp
}

// WithMetadata returns a new immutable PlatformError with the added key-value metadata.
func (e *PlatformError) WithMetadata(key, value string) Error {
	if e == nil {
		return nil
	}
	if key == "" {
		return e
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	newMeta := e.metadata.Clone()
	if newMeta == nil {
		newMeta = make(Metadata)
	}
	newMeta[key] = value

	return &PlatformError{
		code:      e.code,
		severity:  e.severity,
		message:   e.message,
		cause:     e.cause,
		metadata:  newMeta,
		timestamp: e.timestamp,
	}
}

// WithCause returns a new immutable PlatformError with the updated cause.
func (e *PlatformError) WithCause(cause error) Error {
	if e == nil {
		return nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	return &PlatformError{
		code:      e.code,
		severity:  e.severity,
		message:   e.message,
		cause:     cause,
		metadata:  e.metadata.Clone(),
		timestamp: e.timestamp,
	}
}

// Clone returns a defensive copy of the PlatformError.
func (e *PlatformError) Clone() Error {
	if e == nil {
		return nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	return &PlatformError{
		code:      e.code,
		severity:  e.severity,
		message:   e.message,
		cause:     e.cause,
		metadata:  e.metadata.Clone(),
		timestamp: e.timestamp,
	}
}

// FormatDetails returns a deterministic, formatted string containing code, message, severity, timestamp, metadata, and cause.
func (e *PlatformError) FormatDetails() string {
	if e == nil {
		return "<nil>"
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Code: %s\n", e.code))
	sb.WriteString(fmt.Sprintf("Severity: %s\n", e.severity))
	sb.WriteString(fmt.Sprintf("Message: %s\n", e.message))
	sb.WriteString(fmt.Sprintf("Timestamp: %s\n", e.timestamp.UTC().Format(time.RFC3339Nano)))

	if e.cause != nil {
		sb.WriteString(fmt.Sprintf("Cause: %v\n", e.cause))
	}

	if len(e.metadata) > 0 {
		sb.WriteString("Metadata:\n")
		keys := make([]string, 0, len(e.metadata))
		for k := range e.metadata {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, e.metadata[k]))
		}
	}

	return sb.String()
}
