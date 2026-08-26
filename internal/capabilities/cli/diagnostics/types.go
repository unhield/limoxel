package diagnostics

import (
	"fmt"
	"strings"
	"time"
)

// LogLevel represents the operational significance of a log event.
type LogLevel int

const (
	// LevelDebug indicates verbose diagnostic/troubleshooting information.
	LevelDebug LogLevel = iota
	// LevelInfo indicates normal operational progress events.
	LevelInfo
	// LevelWarn indicates anomalous conditions that do not prevent execution.
	LevelWarn
	// LevelError indicates operational failures in specific operations.
	LevelError
	// LevelCritical indicates severe platform/subsystem operational conditions.
	LevelCritical
)

// String returns human-readable uppercase representation of LogLevel.
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel parses a string into a LogLevel.
func ParseLogLevel(str string) (LogLevel, error) {
	switch strings.ToUpper(strings.TrimSpace(str)) {
	case "DEBUG", "TRACE":
		return LevelDebug, nil
	case "INFO":
		return LevelInfo, nil
	case "WARN", "WARNING":
		return LevelWarn, nil
	case "ERROR":
		return LevelError, nil
	case "CRITICAL", "FATAL":
		return LevelCritical, nil
	default:
		return LevelInfo, fmt.Errorf("unknown log level %q (valid: debug, info, warn, error, critical)", str)
	}
}

// LogFormat defines output serialization for log events.
type LogFormat string

const (
	// LogFormatText is human-readable console format with optional ANSI colors.
	LogFormatText LogFormat = "text"
	// LogFormatJSON is structured newline-delimited JSON.
	LogFormatJSON LogFormat = "json"
)

// LogEvent represents an immutable, structured operational log record.
type LogEvent struct {
	Timestamp  time.Time      `json:"timestamp"`
	Level      LogLevel       `json:"level"`
	LevelName  string         `json:"level_name"`
	Component  string         `json:"component,omitempty"`
	Operation  string         `json:"operation,omitempty"`
	Event      string         `json:"event,omitempty"`
	Message    string         `json:"message"`
	Context    map[string]any `json:"context,omitempty"`
	Error      string         `json:"error,omitempty"`
	RequestID  string         `json:"request_id,omitempty"`
	DurationMs float64        `json:"duration_ms,omitempty"`
}

// DiagnosticSeverity communicates the significance of an observed condition.
type DiagnosticSeverity string

const (
	// DiagInfo represents benign or informational operational observations.
	DiagInfo DiagnosticSeverity = "info"
	// DiagWarn represents conditions that may degrade performance or require attention.
	DiagWarn DiagnosticSeverity = "warn"
	// DiagError represents operational failures or misconfigurations.
	DiagError DiagnosticSeverity = "error"
	// DiagCritical represents severe conditions preventing subsystem function.
	DiagCritical DiagnosticSeverity = "critical"
)

// DiagnosticCategory identifies the domain of an observed condition.
type DiagnosticCategory string

const (
	// DiagCatSystem covers host OS, architecture, memory, and runtime conditions.
	DiagCatSystem DiagnosticCategory = "system"
	// DiagCatRepository covers repository accessibility, git metadata, and permissions.
	DiagCatRepository DiagnosticCategory = "repository"
	// DiagCatConfiguration covers configuration validity, schema, and profiles.
	DiagCatConfiguration DiagnosticCategory = "configuration"
	// DiagCatDependency covers external tools, toolchains, and module files.
	DiagCatDependency DiagnosticCategory = "dependency"
	// DiagCatPerformance covers runtime latencies, GC pressure, and allocations.
	DiagCatPerformance DiagnosticCategory = "performance"
	// DiagCatRuntime covers goroutines, scheduler, and general execution.
	DiagCatRuntime DiagnosticCategory = "runtime"
)

// DiagnosticEntry represents an observed condition requiring attention or understanding.
type DiagnosticEntry struct {
	ID          string             `json:"id"`
	Timestamp   time.Time          `json:"timestamp"`
	Severity    DiagnosticSeverity `json:"severity"`
	Category    DiagnosticCategory `json:"category"`
	Component   string             `json:"component"`
	Operation   string             `json:"operation,omitempty"`
	Location    string             `json:"location,omitempty"`
	Message     string             `json:"message"`
	Details     map[string]any     `json:"details,omitempty"`
	Error       string             `json:"error,omitempty"`
	Remediation string             `json:"remediation,omitempty"`
}

// HealthStatus represents operational availability of a component.
type HealthStatus string

const (
	// HealthHealthy indicates component is fully operational.
	HealthHealthy HealthStatus = "healthy"
	// HealthDegraded indicates component is operational with impaired capabilities.
	HealthDegraded HealthStatus = "degraded"
	// HealthUnavailable indicates component cannot currently be accessed.
	HealthUnavailable HealthStatus = "unavailable"
	// HealthFailed indicates component has encountered an unrecoverable failure.
	HealthFailed HealthStatus = "failed"
)

// HealthCheckResult represents the outcome of an individual health check.
type HealthCheckResult struct {
	Name      string             `json:"name"`
	Category  DiagnosticCategory `json:"category"`
	Status    HealthStatus       `json:"status"`
	Message   string             `json:"message"`
	Details   map[string]any     `json:"details,omitempty"`
	Timestamp time.Time          `json:"timestamp"`
	LatencyMs float64            `json:"latency_ms"`
}

// HealthReport summarizes platform and component operational readiness.
type HealthReport struct {
	OverallStatus HealthStatus        `json:"overall_status"`
	Timestamp     time.Time           `json:"timestamp"`
	DurationMs    float64             `json:"duration_ms"`
	Checks        []HealthCheckResult `json:"checks"`
}

// ProfileType identifies the type of runtime profiling being performed.
type ProfileType string

const (
	// ProfileCPU tracks CPU execution bottlenecks via pprof.
	ProfileCPU ProfileType = "cpu"
	// ProfileMemory tracks heap allocation and in-use memory via pprof.
	ProfileMemory ProfileType = "memory"
	// ProfileTrace captures detailed runtime execution traces via runtime/trace.
	ProfileTrace ProfileType = "trace"
	// ProfileExecution measures operation latency and resource deltas.
	ProfileExecution ProfileType = "execution"
)

// ProfileResult summarizes a completed profiling session.
type ProfileResult struct {
	Type       ProfileType    `json:"type"`
	FilePath   string         `json:"file_path,omitempty"`
	DurationMs float64        `json:"duration_ms"`
	Size       int64          `json:"size_bytes,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// ResourceStats captures current host and runtime resource metrics.
type ResourceStats struct {
	NumGoroutine int     `json:"num_goroutine"`
	AllocMB      float64 `json:"alloc_mb"`
	TotalAllocMB float64 `json:"total_alloc_mb"`
	SysMB        float64 `json:"sys_mb"`
	NumGC        uint32  `json:"num_gc"`
	PauseTotalMs float64 `json:"pause_total_ms"`
	CPUCount     int     `json:"cpu_count"`
	GoVersion    string  `json:"go_version"`
	OS           string  `json:"os"`
	Arch         string  `json:"arch"`
}

// TraceSpan represents a single instrumented operational block.
type TraceSpan struct {
	ID         string            `json:"id"`
	ParentID   string            `json:"parent_id,omitempty"`
	Name       string            `json:"name"`
	Component  string            `json:"component"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time"`
	DurationMs float64           `json:"duration_ms"`
	Tags       map[string]string `json:"tags,omitempty"`
}
