package errors

import (
	"fmt"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
)

// ErrorCode identifies the canonical category of a plugin framework failure.
type ErrorCode string

const (
	CodeManifestInvalid         ErrorCode = "ERR_MANIFEST_INVALID"
	CodeIdentityInvalid         ErrorCode = "ERR_IDENTITY_INVALID"
	CodeDuplicateIdentity       ErrorCode = "ERR_DUPLICATE_IDENTITY"
	CodePluginNotFound          ErrorCode = "ERR_PLUGIN_NOT_FOUND"
	CodeDependencyMissing       ErrorCode = "ERR_DEPENDENCY_MISSING"
	CodeDependencyConflict      ErrorCode = "ERR_DEPENDENCY_CONFLICT"
	CodeDependencyCycle         ErrorCode = "ERR_DEPENDENCY_CYCLE"
	CodeIncompatibleVersion     ErrorCode = "ERR_INCOMPATIBLE_VERSION"
	CodeIncompatibleHost        ErrorCode = "ERR_INCOMPATIBLE_HOST"
	CodeInvalidTransition       ErrorCode = "ERR_INVALID_TRANSITION"
	CodeInitializationFailed    ErrorCode = "ERR_INITIALIZATION_FAILED"
	CodeLoadingFailed           ErrorCode = "ERR_LOADING_FAILED"
	CodeUnloadingFailed         ErrorCode = "ERR_UNLOADING_FAILED"
	CodePluginAlreadyActive     ErrorCode = "ERR_PLUGIN_ALREADY_ACTIVE"
	CodePluginNotActive         ErrorCode = "ERR_PLUGIN_NOT_ACTIVE"
	CodeSecurityPolicyViolation ErrorCode = "ERR_SECURITY_POLICY_VIOLATION"
)

// String returns the string representation of ErrorCode.
func (c ErrorCode) String() string {
	return string(c)
}

// PluginError is a structured, contextual error type representing plugin framework failures.
type PluginError struct {
	code     ErrorCode
	message  string
	pluginID model.Identity
	cause    error
	details  map[string]string
}

// New constructs a structured PluginError.
func New(code ErrorCode, pluginID model.Identity, message string) *PluginError {
	return &PluginError{
		code:     code,
		pluginID: pluginID,
		message:  message,
		details:  make(map[string]string),
	}
}

// Wrap constructs a structured PluginError wrapping an underlying cause.
func Wrap(code ErrorCode, pluginID model.Identity, message string, cause error) *PluginError {
	return &PluginError{
		code:     code,
		pluginID: pluginID,
		message:  message,
		cause:    cause,
		details:  make(map[string]string),
	}
}

// Code returns the canonical ErrorCode.
func (e *PluginError) Code() ErrorCode {
	if e == nil {
		return ""
	}
	return e.code
}

// Message returns the human-readable explanation of the error.
func (e *PluginError) Message() string {
	if e == nil {
		return ""
	}
	return e.message
}

// PluginID returns the target plugin identifier associated with the failure, if any.
func (e *PluginError) PluginID() model.Identity {
	if e == nil {
		return ""
	}
	return e.pluginID
}

// Cause returns the underlying cause error, if any.
func (e *PluginError) Cause() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Unwrap enables standard library errors.Unwrap compatibility.
func (e *PluginError) Unwrap() error {
	return e.Cause()
}

// Details returns a copy of the key-value context details.
func (e *PluginError) Details() map[string]string {
	if e == nil || e.details == nil {
		return nil
	}
	cp := make(map[string]string, len(e.details))
	for k, v := range e.details {
		cp[k] = v
	}
	return cp
}

// WithDetail attaches an additional key-value diagnostic attribute.
func (e *PluginError) WithDetail(key, value string) *PluginError {
	if e == nil {
		return nil
	}
	if e.details == nil {
		e.details = make(map[string]string)
	}
	e.details[strings.TrimSpace(key)] = strings.TrimSpace(value)
	return e
}

// Error formats the structured error as a readable string.
func (e *PluginError) Error() string {
	if e == nil {
		return "<nil>"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%s]", e.code))
	if e.pluginID != "" {
		sb.WriteString(fmt.Sprintf(" plugin=%s", e.pluginID))
	}
	sb.WriteString(fmt.Sprintf(": %s", e.message))
	if e.cause != nil {
		sb.WriteString(fmt.Sprintf(" (cause: %v)", e.cause))
	}
	return sb.String()
}
