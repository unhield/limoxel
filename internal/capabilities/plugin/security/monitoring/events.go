package monitoring

import (
	"fmt"
	"strings"
	"time"
)

// EventCategory defines the classification of security events.
type EventCategory string

const (
	// CategoryVerification covers signature, checksum, publisher, and trust validation.
	CategoryVerification EventCategory = "VERIFICATION"
	// CategoryPermission covers permission requests, grants, and denials.
	CategoryPermission EventCategory = "PERMISSION"
	// CategorySandbox covers filesystem, network, and process jail enforcement.
	CategorySandbox EventCategory = "SANDBOX"
	// CategoryResource covers memory quotas, process limits, and timeouts.
	CategoryResource EventCategory = "RESOURCE"
	// CategoryRuntime covers subprocess lifecycle, crashes, and abnormal exits.
	CategoryRuntime EventCategory = "RUNTIME"
)

// EventSeverity represents the urgency level of a security event.
type EventSeverity string

const (
	SeverityInfo     EventSeverity = "INFO"
	SeverityWarning  EventSeverity = "WARNING"
	SeverityError    EventSeverity = "ERROR"
	SeverityCritical EventSeverity = "CRITICAL"
)

// SecurityEvent encapsulates an auditable security action or violation.
type SecurityEvent struct {
	ID        string            `json:"id"`
	Timestamp time.Time         `json:"timestamp"`
	Category  EventCategory     `json:"category"`
	Severity  EventSeverity     `json:"severity"`
	PluginID  string            `json:"plugin_id"`
	ProcessID int               `json:"process_id,omitempty"`
	Action    string            `json:"action"`             // e.g. "file.write", "signature.verify"
	Resource  string            `json:"resource,omitempty"` // e.g. "C:\\Windows\\cmd.exe", "repo.read"
	Decision  string            `json:"decision"`           // "GRANTED", "DENIED", "VIOLATION", "CONTAINED"
	Reason    string            `json:"reason"`
	Context   map[string]string `json:"context,omitempty"`
}

// String formats the security event into a clean audit string.
func (e SecurityEvent) String() string {
	return fmt.Sprintf("[%s] [%s] [%s] plugin=%s action=%s decision=%s: %s",
		e.Timestamp.Format(time.RFC3339),
		e.Severity,
		e.Category,
		e.PluginID,
		e.Action,
		e.Decision,
		e.Reason,
	)
}

// SanitizeReason ensures no passwords, tokens, or secret credentials leak into audit logs.
func SanitizeReason(msg string) string {
	sensitiveKeywords := []string{
		"password", "secret", "token", "apikey", "bearer", "private_key",
	}

	lower := strings.ToLower(msg)
	for _, kw := range sensitiveKeywords {
		if strings.Contains(lower, kw) {
			// Redact potential values following '=' or ':'
			idx := strings.Index(lower, kw)
			if idx != -1 && len(msg) > idx+len(kw)+1 {
				return msg[:idx+len(kw)] + "=[REDACTED]"
			}
		}
	}
	return msg
}
