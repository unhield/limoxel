package analysis

import "fmt"

// ErrorCategory classifies the failure domain of an analysis operation.
type ErrorCategory string

const (
	ErrCatInvalidInput       ErrorCategory = "invalid_input"
	ErrCatAnalyzerFailed     ErrorCategory = "analyzer_failed"
	ErrCatUnsupportedRule    ErrorCategory = "unsupported_rule"
	ErrCatInsufficientData   ErrorCategory = "insufficient_data"
	ErrCatConfigurationError ErrorCategory = "configuration_error"
	ErrCatInternal           ErrorCategory = "internal"
)

// AnalysisError represents a structured, categorized error emitted by the analysis engine.
type AnalysisError struct {
	category ErrorCategory
	message  string
	ruleID   RuleID
	detail   string
	cause    error
}

// NewAnalysisError constructs an immutable AnalysisError.
func NewAnalysisError(cat ErrorCategory, msg string, ruleID RuleID) *AnalysisError {
	return &AnalysisError{
		category: cat,
		message:  msg,
		ruleID:   ruleID,
	}
}

// Error implements the error interface.
func (e *AnalysisError) Error() string {
	if e == nil {
		return ""
	}
	rulePart := ""
	if e.ruleID != "" {
		rulePart = fmt.Sprintf("[%s] ", e.ruleID)
	}
	if e.cause != nil {
		return fmt.Sprintf("analysis error (%s): %s%s: %v", e.category, rulePart, e.message, e.cause)
	}
	if e.detail != "" {
		return fmt.Sprintf("analysis error (%s): %s%s (%s)", e.category, rulePart, e.message, e.detail)
	}
	return fmt.Sprintf("analysis error (%s): %s%s", e.category, rulePart, e.message)
}

// Category returns the error category.
func (e *AnalysisError) Category() ErrorCategory {
	if e == nil {
		return ErrCatInternal
	}
	return e.category
}

// RuleID returns the associated rule ID if any.
func (e *AnalysisError) RuleID() RuleID {
	if e == nil {
		return ""
	}
	return e.ruleID
}

// Detail returns the detailed error context.
func (e *AnalysisError) Detail() string {
	if e == nil {
		return ""
	}
	return e.detail
}

// Cause returns the underlying root cause error.
func (e *AnalysisError) Cause() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// WithDetail returns a copy of the error with extra contextual detail.
func (e *AnalysisError) WithDetail(detail string) *AnalysisError {
	if e == nil {
		return nil
	}
	return &AnalysisError{
		category: e.category,
		message:  e.message,
		ruleID:   e.ruleID,
		detail:   detail,
		cause:    e.cause,
	}
}

// WithCause returns a copy of the error wrapping an underlying cause.
func (e *AnalysisError) WithCause(cause error) *AnalysisError {
	if e == nil {
		return nil
	}
	return &AnalysisError{
		category: e.category,
		message:  e.message,
		ruleID:   e.ruleID,
		detail:   e.detail,
		cause:    cause,
	}
}

// IsCategory checks whether an error belongs to a specific ErrorCategory.
func IsCategory(err error, cat ErrorCategory) bool {
	if err == nil {
		return false
	}
	if ae, ok := err.(*AnalysisError); ok {
		return ae.Category() == cat
	}
	return false
}

// Sentinel Errors
var (
	ErrNilEngine         = NewAnalysisError(ErrCatInvalidInput, "analysis engine is nil", "")
	ErrEmptyTarget       = NewAnalysisError(ErrCatInvalidInput, "target identifier is empty", "")
	ErrInvalidRule       = NewAnalysisError(ErrCatUnsupportedRule, "unrecognized or unsupported analysis rule", "")
	ErrInsufficientData  = NewAnalysisError(ErrCatInsufficientData, "insufficient repository knowledge models to complete analysis", "")
	ErrMalformedConfig   = NewAnalysisError(ErrCatConfigurationError, "configuration file contains malformed syntax", "")
	ErrNilAnalyzerResult = NewAnalysisError(ErrCatInternal, "analyzer returned nil result", "")
)
