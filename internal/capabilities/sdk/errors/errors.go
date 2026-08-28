package errors

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// SemanticCategory defines stable public error classifications for programmatic handling by external consumers.
type SemanticCategory string

const (
	// CategoryInvalidInput represents malformed arguments, out-of-range values, or invalid query options.
	CategoryInvalidInput SemanticCategory = "ERR_INVALID_INPUT"

	// CategoryNotFound represents missing repository entities, symbols, packages, files, or nodes.
	CategoryNotFound SemanticCategory = "ERR_NOT_FOUND"

	// CategoryUnsupported represents operations, formats, or query types not supported by the capability.
	CategoryUnsupported SemanticCategory = "ERR_UNSUPPORTED"

	// CategoryInvalidState represents operations attempted when the repository or service is in an unsuitable state.
	CategoryInvalidState SemanticCategory = "ERR_INVALID_STATE"

	// CategoryLifecycleViolation represents calls to deprecated/removed APIs or out-of-order lifecycle calls.
	CategoryLifecycleViolation SemanticCategory = "ERR_LIFECYCLE_VIOLATION"

	// CategoryUnavailable represents temporary resource unavailability or uninitialized subsystems.
	CategoryUnavailable SemanticCategory = "ERR_UNAVAILABLE"

	// CategoryInternal represents unexpected internal execution failures without leaking implementation secrets.
	CategoryInternal SemanticCategory = "ERR_INTERNAL"
)

// String returns the string representation of SemanticCategory.
func (c SemanticCategory) String() string {
	return string(c)
}

// SDKError is the public interface contract for all errors emitted by Limoxel SDK capabilities.
type SDKError interface {
	error
	Category() SemanticCategory
	Code() string
	Message() string
	SafePublicMessage() string
	Details() map[string]string
	Unwrap() error
}

// Error represents an immutable, thread-safe SDK error instance conforming to SDKError.
type Error struct {
	mu          sync.RWMutex
	category    SemanticCategory
	code        string
	message     string
	safeMessage string
	cause       error
	details     map[string]string
}

// Ensure Error implements SDKError.
var _ SDKError = (*Error)(nil)

// New constructs a base Error with the given semantic category, code, and message.
func New(category SemanticCategory, code, message string) *Error {
	if category == "" {
		category = CategoryInternal
	}
	if code == "" {
		code = string(category)
	}
	return &Error{
		category:    category,
		code:        strings.TrimSpace(code),
		message:     strings.TrimSpace(message),
		safeMessage: strings.TrimSpace(message),
		details:     make(map[string]string),
	}
}

// NewInvalidInput creates a standard ERR_INVALID_INPUT error.
func NewInvalidInput(message string) *Error {
	return New(CategoryInvalidInput, "ERR_INVALID_INPUT", message)
}

// NewNotFound creates a standard ERR_NOT_FOUND error for an entity and identity.
func NewNotFound(entityType, identity string) *Error {
	msg := fmt.Sprintf("%s %q not found", strings.TrimSpace(entityType), strings.TrimSpace(identity))
	err := New(CategoryNotFound, "ERR_NOT_FOUND", msg)
	return err.WithDetail("entity_type", entityType).WithDetail("identity", identity)
}

// NewUnsupported creates a standard ERR_UNSUPPORTED error.
func NewUnsupported(feature, message string) *Error {
	msg := fmt.Sprintf("unsupported %s: %s", strings.TrimSpace(feature), strings.TrimSpace(message))
	err := New(CategoryUnsupported, "ERR_UNSUPPORTED", msg)
	return err.WithDetail("feature", feature)
}

// NewInvalidState creates a standard ERR_INVALID_STATE error.
func NewInvalidState(state, message string) *Error {
	msg := fmt.Sprintf("invalid state %q: %s", strings.TrimSpace(state), strings.TrimSpace(message))
	err := New(CategoryInvalidState, "ERR_INVALID_STATE", msg)
	return err.WithDetail("state", state)
}

// NewLifecycleViolation creates a standard ERR_LIFECYCLE_VIOLATION error.
func NewLifecycleViolation(apiName, state, message string) *Error {
	msg := fmt.Sprintf("API %q lifecycle violation (%s): %s", strings.TrimSpace(apiName), strings.TrimSpace(state), strings.TrimSpace(message))
	err := New(CategoryLifecycleViolation, "ERR_LIFECYCLE_VIOLATION", msg)
	return err.WithDetail("api", apiName).WithDetail("lifecycle_state", state)
}

// NewUnavailable creates a standard ERR_UNAVAILABLE error.
func NewUnavailable(resource, message string) *Error {
	msg := fmt.Sprintf("resource %q unavailable: %s", strings.TrimSpace(resource), strings.TrimSpace(message))
	err := New(CategoryUnavailable, "ERR_UNAVAILABLE", msg)
	return err.WithDetail("resource", resource)
}

// NewInternal creates a standard ERR_INTERNAL error wrapping an underlying cause.
func NewInternal(message string, cause error) *Error {
	err := New(CategoryInternal, "ERR_INTERNAL", message)
	err.safeMessage = "an internal error occurred during SDK operation"
	if cause != nil {
		err.cause = cause
	}
	return err
}

// Wrap wraps an existing error into a categorized SDKError.
func Wrap(cause error, category SemanticCategory, code, message string) *Error {
	err := New(category, code, message)
	err.cause = cause
	return err
}

// Error returns the formatted error string.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%s] %s", e.code, e.message))
	if e.cause != nil {
		sb.WriteString(": ")
		sb.WriteString(e.cause.Error())
	}
	return sb.String()
}

// Category returns the semantic category.
func (e *Error) Category() SemanticCategory {
	if e == nil {
		return CategoryInternal
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.category
}

// Code returns the error code.
func (e *Error) Code() string {
	if e == nil {
		return string(CategoryInternal)
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.code
}

// Message returns the error message.
func (e *Error) Message() string {
	if e == nil {
		return ""
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.message
}

// SafePublicMessage returns a sanitized public message safe for external exposure without leaking implementation details.
func (e *Error) SafePublicMessage() string {
	if e == nil {
		return ""
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.safeMessage != "" {
		return e.safeMessage
	}
	return e.message
}

// Details returns a defensive copy of structured error details.
func (e *Error) Details() map[string]string {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	res := make(map[string]string, len(e.details))
	for k, v := range e.details {
		res[k] = v
	}
	return res
}

// Unwrap returns the underlying cause error.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cause
}

// WithDetail returns a new copy of Error with an added key-value detail entry.
func (e *Error) WithDetail(key, value string) *Error {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	clonedDetails := make(map[string]string, len(e.details)+1)
	for k, v := range e.details {
		clonedDetails[k] = v
	}
	clonedDetails[strings.TrimSpace(key)] = strings.TrimSpace(value)

	return &Error{
		category:    e.category,
		code:        e.code,
		message:     e.message,
		safeMessage: e.safeMessage,
		cause:       e.cause,
		details:     clonedDetails,
	}
}

// WithCause returns a new copy of Error with the updated underlying cause.
func (e *Error) WithCause(cause error) *Error {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	clonedDetails := make(map[string]string, len(e.details))
	for k, v := range e.details {
		clonedDetails[k] = v
	}

	return &Error{
		category:    e.category,
		code:        e.code,
		message:     e.message,
		safeMessage: e.safeMessage,
		cause:       cause,
		details:     clonedDetails,
	}
}

// WithSafePublicMessage returns a new copy of Error with a custom sanitized public message.
func (e *Error) WithSafePublicMessage(safeMsg string) *Error {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	clonedDetails := make(map[string]string, len(e.details))
	for k, v := range e.details {
		clonedDetails[k] = v
	}

	return &Error{
		category:    e.category,
		code:        e.code,
		message:     e.message,
		safeMessage: strings.TrimSpace(safeMsg),
		cause:       e.cause,
		details:     clonedDetails,
	}
}

// FormatDetails returns a formatted multiline inspection string of the error.
func (e *Error) FormatDetails() string {
	if e == nil {
		return "<nil>"
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Category : %s\n", e.category))
	sb.WriteString(fmt.Sprintf("Code     : %s\n", e.code))
	sb.WriteString(fmt.Sprintf("Message  : %s\n", e.message))
	if e.cause != nil {
		sb.WriteString(fmt.Sprintf("Cause    : %v\n", e.cause))
	}
	if len(e.details) > 0 {
		sb.WriteString("Details  :\n")
		keys := make([]string, 0, len(e.details))
		for k := range e.details {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, e.details[k]))
		}
	}
	return sb.String()
}

// GetSDKError checks if err or any error in its chain implements SDKError.
func GetSDKError(err error) (SDKError, bool) {
	if err == nil {
		return nil, false
	}
	var sdkErr SDKError
	if errors.As(err, &sdkErr) {
		return sdkErr, true
	}
	return nil, false
}

// IsCategory checks whether err matches the given SemanticCategory.
func IsCategory(err error, category SemanticCategory) bool {
	if sdkErr, ok := GetSDKError(err); ok {
		return sdkErr.Category() == category
	}
	return false
}

// IsInvalidInput returns true if err represents CategoryInvalidInput.
func IsInvalidInput(err error) bool {
	return IsCategory(err, CategoryInvalidInput)
}

// IsNotFound returns true if err represents CategoryNotFound.
func IsNotFound(err error) bool {
	return IsCategory(err, CategoryNotFound)
}

// IsUnsupported returns true if err represents CategoryUnsupported.
func IsUnsupported(err error) bool {
	return IsCategory(err, CategoryUnsupported)
}

// IsInvalidState returns true if err represents CategoryInvalidState.
func IsInvalidState(err error) bool {
	return IsCategory(err, CategoryInvalidState)
}

// IsLifecycleViolation returns true if err represents CategoryLifecycleViolation.
func IsLifecycleViolation(err error) bool {
	return IsCategory(err, CategoryLifecycleViolation)
}

// IsUnavailable returns true if err represents CategoryUnavailable.
func IsUnavailable(err error) bool {
	return IsCategory(err, CategoryUnavailable)
}

// IsInternal returns true if err represents CategoryInternal.
func IsInternal(err error) bool {
	return IsCategory(err, CategoryInternal)
}
