package discovery

import "fmt"

// DiagnosticSeverity defines the severity level of a discovery diagnostic.
type DiagnosticSeverity string

const (
	// SeverityInfo indicates an informational discovery diagnostic.
	SeverityInfo DiagnosticSeverity = "INFO"

	// SeverityWarning indicates a non-fatal warning during discovery.
	SeverityWarning DiagnosticSeverity = "WARNING"

	// SeverityError indicates a recoverable error condition encountered during discovery.
	SeverityError DiagnosticSeverity = "ERROR"
)

// Diagnostic represents a structured, deterministic observation produced during discovery.
type Diagnostic struct {
	severity DiagnosticSeverity
	code     string
	message  string
	path     string
	isFatal  bool
}

// NewDiagnostic creates an immutable Diagnostic instance.
func NewDiagnostic(severity DiagnosticSeverity, code string, message string, path string, isFatal bool) *Diagnostic {
	return &Diagnostic{
		severity: severity,
		code:     code,
		message:  message,
		path:     path,
		isFatal:  isFatal,
	}
}

// Severity returns the diagnostic severity level.
func (d *Diagnostic) Severity() DiagnosticSeverity {
	if d == nil {
		return SeverityInfo
	}
	return d.severity
}

// Code returns the diagnostic code identifier.
func (d *Diagnostic) Code() string {
	if d == nil {
		return ""
	}
	return d.code
}

// Message returns the diagnostic message text.
func (d *Diagnostic) Message() string {
	if d == nil {
		return ""
	}
	return d.message
}

// Path returns the repository-relative path associated with the diagnostic, if applicable.
func (d *Diagnostic) Path() string {
	if d == nil {
		return ""
	}
	return d.path
}

// IsFatal reports whether the diagnostic condition prevents complete repository discovery.
func (d *Diagnostic) IsFatal() bool {
	if d == nil {
		return false
	}
	return d.isFatal
}

// String returns a human-readable representation of the Diagnostic.
func (d *Diagnostic) String() string {
	if d == nil {
		return ""
	}
	if d.path != "" {
		return fmt.Sprintf("[%s] %s (%s): %s", d.severity, d.code, d.path, d.message)
	}
	return fmt.Sprintf("[%s] %s: %s", d.severity, d.code, d.message)
}
