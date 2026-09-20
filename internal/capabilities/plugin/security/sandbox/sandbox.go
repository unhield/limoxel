package sandbox

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrSandboxViolation indicates a direct attempt to breach sandbox boundaries.
	ErrSandboxViolation = errors.New("sandbox: security boundary violation")
	// ErrResourceExhausted indicates memory, time, or process quota was exceeded.
	ErrResourceExhausted = errors.New("sandbox: resource quota exhausted")
)

// IsolationLevel defines the depth of isolation applied to a plugin.
type IsolationLevel string

const (
	// IsolationLevelStrict applies OS-level Job Object/Process Group, strict filesystem jail, and network denial.
	IsolationLevelStrict IsolationLevel = "STRICT"
	// IsolationLevelStandard applies process separation, filesystem jail, and permission-based network.
	IsolationLevelStandard IsolationLevel = "STANDARD"
	// IsolationLevelRestricted applies minimal permissions with read-only sandbox.
	IsolationLevelRestricted IsolationLevel = "RESTRICTED"
)

// ResourceLimits specifies hardware and runtime constraints for a plugin instance.
type ResourceLimits struct {
	MaxMemoryBytes   uint64        `json:"max_memory_bytes"`  // e.g. 512 MB (512 * 1024 * 1024)
	MaxProcesses     uint32        `json:"max_processes"`     // Max active child processes (default 1)
	StartupTimeout   time.Duration `json:"startup_timeout"`   // Timeout for plugin to complete handshake
	ShutdownTimeout  time.Duration `json:"shutdown_timeout"`  // Timeout for graceful stop before SIGKILL
	MaxMessageBytes  int           `json:"max_message_bytes"` // Max IPC message size (default 16MB)
	ExecutionTimeout time.Duration `json:"execution_timeout"` // Per-request timeout default
}

// DefaultResourceLimits provides safe default resource limits.
func DefaultResourceLimits() ResourceLimits {
	return ResourceLimits{
		MaxMemoryBytes:   512 * 1024 * 1024, // 512 MB
		MaxProcesses:     1,                 // No child processes allowed by default
		StartupTimeout:   5 * time.Second,
		ShutdownTimeout:  3 * time.Second,
		MaxMessageBytes:  16 * 1024 * 1024, // 16 MB
		ExecutionTimeout: 30 * time.Second,
	}
}

// SandboxConfig encapsulates complete parameters required to initialize a sandbox environment.
type SandboxConfig struct {
	PluginID       string
	WorkspaceRoot  string
	DataRoot       string // Root for private plugin data
	TempRoot       string // Root for ephemeral files
	Isolation      IsolationLevel
	Limits         ResourceLimits
	NetworkAllowed bool
	AllowedHosts   []string
	AllowedPaths   []string // Additional read-only allowed paths
	AllowLocalhost bool
}

// PluginSandbox defines the contract for an isolated execution environment.
type PluginSandbox interface {
	// Initialize prepares the sandbox directories and environment.
	Initialize(ctx context.Context) error

	// Filesystem returns the filesystem guard governing file operations.
	Filesystem() *FilesystemGuard

	// Network returns the network guard governing socket/connection operations.
	Network() *NetworkGuard

	// AttachProcess binds a launched operating system process to the sandbox controls.
	AttachProcess(pid int) error

	// Terminate forcibly terminates all processes assigned to the sandbox.
	Terminate() error

	// Cleanup releases all sandbox resources and cleans ephemeral temporary directories.
	Cleanup() error
}
