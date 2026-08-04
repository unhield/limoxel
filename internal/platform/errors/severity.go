package errors

import (
	"fmt"
	"strings"
)

// Severity defines the operational impact level of a platform error.
type Severity uint8

const (
	// SeverityInfo indicates an informational or non-disruptive error condition.
	SeverityInfo Severity = iota

	// SeverityWarning indicates a non-critical error or warning condition.
	SeverityWarning

	// SeverityError indicates a standard operational error condition.
	SeverityError

	// SeverityCritical indicates a critical subsystem failure.
	SeverityCritical

	// SeverityFatal indicates an unrecoverable platform failure.
	SeverityFatal
)

// String returns the canonical uppercase string representation of Severity.
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityCritical:
		return "CRITICAL"
	case SeverityFatal:
		return "FATAL"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", uint8(s))
	}
}

// ParseSeverity parses a string representation into a Severity value.
func ParseSeverity(str string) (Severity, error) {
	switch strings.ToUpper(strings.TrimSpace(str)) {
	case "INFO":
		return SeverityInfo, nil
	case "WARN", "WARNING":
		return SeverityWarning, nil
	case "ERROR":
		return SeverityError, nil
	case "CRITICAL":
		return SeverityCritical, nil
	case "FATAL":
		return SeverityFatal, nil
	default:
		return SeverityError, fmt.Errorf("errors: unknown severity %q", str)
	}
}
