package management

import (
	"errors"
	"time"
)

var (
	ErrReplayDetected     = errors.New("enterprise management: command replay detected (stale timestamp, nonce reused, or invalid sequence)")
	ErrInvalidCommand     = errors.New("enterprise management: invalid command specification")
	ErrExecutionFailed    = errors.New("enterprise management: remote command execution failed")
	ErrDependencyConflict = errors.New("enterprise management: removal would violate active dependencies")
)

// CommandType defines the typed domain operation executed remotely.
type CommandType string

const (
	CmdInstall CommandType = "INSTALL"
	CmdUpdate  CommandType = "UPDATE"
	CmdRemove  CommandType = "REMOVE"
	CmdSuspend CommandType = "SUSPEND"
	CmdRestart CommandType = "RESTART"
)

// RemoteCommand holds typed execution parameters and replay protection headers.
type RemoteCommand struct {
	CommandID       string      `json:"command_id"`
	Type            CommandType `json:"type"`
	OrganizationID  string      `json:"organization_id"`
	TargetID        string      `json:"target_id"`
	PluginID        string      `json:"plugin_id"`
	TargetVersion   string      `json:"target_version,omitempty"`
	RollbackVersion string      `json:"rollback_version,omitempty"`
	Source          string      `json:"source,omitempty"` // "private_registry" or "marketplace"
	PurgeData       bool        `json:"purge_data,omitempty"`
	Timestamp       time.Time   `json:"timestamp"`
	Nonce           string      `json:"nonce"`
	Sequence        int64       `json:"sequence"`
	IdempotencyKey  string      `json:"idempotency_key"`
}

// CommandResult documents the outcome of a remote command execution.
type CommandResult struct {
	CommandID      string    `json:"command_id"`
	Success        bool      `json:"success"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	ExecutedAt     time.Time `json:"executed_at"`
	RollbackDone   bool      `json:"rollback_done,omitempty"`
	RollbackFailed bool      `json:"rollback_failed,omitempty"`
}
