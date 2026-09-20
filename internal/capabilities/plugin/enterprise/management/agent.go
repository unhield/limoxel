package management

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/monitoring"
)

// HostExecutor defines local plugin lifecycle hooks implemented by the host agent.
type HostExecutor interface {
	Install(ctx context.Context, pluginID, version string, archiveData []byte) error
	Update(ctx context.Context, pluginID, targetVersion string, archiveData []byte) error
	Remove(ctx context.Context, pluginID string, purgeData bool) error
	Suspend(ctx context.Context, pluginID string) error
	Restart(ctx context.Context, pluginID string) error
	GetInstalled() map[string]string // pluginID -> version
}

// TelemetryProvider optionally reports real execution metrics from the host.
type TelemetryProvider interface {
	GetPluginTelemetry(pluginID string) (crashCount int, memoryBytes int64, err error)
}

// ManagedAgent resides on the managed host and executes validated remote commands.
type ManagedAgent struct {
	mu           sync.RWMutex
	targetID     string
	orgID        string
	executor     HostExecutor
	seenNonces   map[string]time.Time
	lastSequence int64
	idempotency  map[string]CommandResult
}

// NewManagedAgent constructs an agent with replay protection.
func NewManagedAgent(orgID, targetID string, executor HostExecutor) *ManagedAgent {
	return &ManagedAgent{
		targetID:    targetID,
		orgID:       orgID,
		executor:    executor,
		seenNonces:  make(map[string]time.Time),
		idempotency: make(map[string]CommandResult),
	}
}

// ValidateAndExecute checks replay defense headers and executes the typed command.
func (a *ManagedAgent) ValidateAndExecute(ctx context.Context, cmd RemoteCommand, archiveData []byte) (CommandResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// 0. Target ID Boundary Check
	if cmd.TargetID != "" && cmd.TargetID != a.targetID {
		res := CommandResult{
			CommandID:    cmd.CommandID,
			Success:      false,
			ErrorMessage: fmt.Sprintf("target mismatch: agent target %q, command target %q", a.targetID, cmd.TargetID),
			ExecutedAt:   time.Now().UTC(),
		}
		return res, ErrReplayDetected
	}

	// 1. Organization Boundary Check
	if cmd.OrganizationID != a.orgID {
		res := CommandResult{
			CommandID:    cmd.CommandID,
			Success:      false,
			ErrorMessage: fmt.Sprintf("tenant mismatch: agent org %q, command org %q", a.orgID, cmd.OrganizationID),
			ExecutedAt:   time.Now().UTC(),
		}
		return res, ErrReplayDetected
	}

	// 2. Idempotency Check
	if cmd.IdempotencyKey != "" {
		if cached, exists := a.idempotency[cmd.IdempotencyKey]; exists {
			return cached, nil
		}
	}

	// 3. Timestamp Freshness Check (Max 5 minutes clock skew)
	now := time.Now().UTC()
	if math.Abs(now.Sub(cmd.Timestamp).Seconds()) > 300 {
		res := CommandResult{
			CommandID:    cmd.CommandID,
			Success:      false,
			ErrorMessage: "command timestamp outside acceptable 5-minute freshness window",
			ExecutedAt:   now,
		}
		return res, ErrReplayDetected
	}

	// 4. Nonce Replay Check
	if cmd.Nonce != "" {
		if _, exists := a.seenNonces[cmd.Nonce]; exists {
			res := CommandResult{
				CommandID:    cmd.CommandID,
				Success:      false,
				ErrorMessage: "command nonce already used",
				ExecutedAt:   now,
			}
			return res, ErrReplayDetected
		}
		a.seenNonces[cmd.Nonce] = now
	}

	// 5. Sequence Monotonicity Check
	if cmd.Sequence <= a.lastSequence {
		res := CommandResult{
			CommandID:    cmd.CommandID,
			Success:      false,
			ErrorMessage: fmt.Sprintf("invalid command sequence %d <= current %d", cmd.Sequence, a.lastSequence),
			ExecutedAt:   now,
		}
		return res, ErrReplayDetected
	}
	a.lastSequence = cmd.Sequence

	// 6. Execute Domain Command
	var execErr error
	var rollbackDone bool
	var rollbackFailed bool

	switch cmd.Type {
	case CmdInstall:
		if a.executor != nil {
			execErr = a.executor.Install(ctx, cmd.PluginID, cmd.TargetVersion, archiveData)
		}
	case CmdUpdate:
		if a.executor != nil {
			execErr = a.executor.Update(ctx, cmd.PluginID, cmd.TargetVersion, archiveData)
			if execErr != nil && cmd.RollbackVersion != "" {
				// Attempt rollback
				if rbErr := a.executor.Update(ctx, cmd.PluginID, cmd.RollbackVersion, nil); rbErr == nil {
					rollbackDone = true
				} else {
					rollbackFailed = true
					execErr = fmt.Errorf("update failed: %w (rollback to %s failed: %v)", execErr, cmd.RollbackVersion, rbErr)
				}
			}
		}
	case CmdRemove:
		if a.executor != nil {
			execErr = a.executor.Remove(ctx, cmd.PluginID, cmd.PurgeData)
		}
	case CmdSuspend:
		if a.executor != nil {
			execErr = a.executor.Suspend(ctx, cmd.PluginID)
		}
	case CmdRestart:
		if a.executor != nil {
			execErr = a.executor.Restart(ctx, cmd.PluginID)
		}
	default:
		execErr = fmt.Errorf("%w: unknown command type %q", ErrInvalidCommand, cmd.Type)
	}

	res := CommandResult{
		CommandID:      cmd.CommandID,
		Success:        execErr == nil,
		ExecutedAt:     now,
		RollbackDone:   rollbackDone,
		RollbackFailed: rollbackFailed,
	}
	if execErr != nil {
		res.ErrorMessage = execErr.Error()
	}

	// Store idempotency result
	if cmd.IdempotencyKey != "" {
		a.idempotency[cmd.IdempotencyKey] = res
	}

	return res, execErr
}

// GenerateTelemetry collects local plugin execution status to report back to monitoring.
func (a *ManagedAgent) GenerateTelemetry(pluginID string) monitoring.ActualPluginState {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var ver string
	var running bool
	if a.executor != nil {
		installed := a.executor.GetInstalled()
		if v, ok := installed[pluginID]; ok {
			ver = v
			running = true
		}
	}

	health := monitoring.HealthHealthy
	if ver == "" {
		health = monitoring.HealthRemoved
	}

	var crashCount int
	var memoryBytes int64
	if tp, ok := a.executor.(TelemetryProvider); ok {
		if crashes, mem, err := tp.GetPluginTelemetry(pluginID); err == nil {
			crashCount = crashes
			memoryBytes = mem
		}
	}

	return monitoring.ActualPluginState{
		Version:     ver,
		Running:     running,
		Health:      health,
		ReportedAt:  time.Now().UTC(),
		CrashCount:  crashCount,
		MemoryBytes: memoryBytes,
	}
}
