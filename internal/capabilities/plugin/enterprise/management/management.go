package management

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/audit"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/identity"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/monitoring"
)

// DeploymentTarget represents a registered enterprise host, cluster, or container environment.
type DeploymentTarget struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Environment    string    `json:"environment"` // "production", "staging", "dev"
	Platform       string    `json:"platform"`    // "linux/amd64", "windows/amd64"
	RegisteredAt   time.Time `json:"registered_at"`
	Active         bool      `json:"active"`
}

// Manager coordinates centralized plugin management and remote command dispatch.
type Manager struct {
	mu           sync.RWMutex
	targets      map[string]*DeploymentTarget
	targetAgents map[string]*ManagedAgent
	monitor      *monitoring.Monitor
	auditLogger  *audit.Logger
	sequences    map[string]int64 // targetID -> seq
}

// NewManager constructs a central plugin management coordinator.
func NewManager(mon *monitoring.Monitor, auditLogger *audit.Logger) *Manager {
	return &Manager{
		targets:      make(map[string]*DeploymentTarget),
		targetAgents: make(map[string]*ManagedAgent),
		monitor:      mon,
		auditLogger:  auditLogger,
		sequences:    make(map[string]int64),
	}
}

// RegisterTarget registers a deployment target and connects its managed agent adapter.
func (m *Manager) RegisterTarget(target DeploymentTarget, agent *ManagedAgent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if target.ID == "" {
		return errors.New("enterprise management: target ID cannot be empty")
	}

	m.targets[target.ID] = &target
	if agent != nil {
		m.targetAgents[target.ID] = agent
	}
	return nil
}

// GetTarget retrieves a deployment target by ID.
func (m *Manager) GetTarget(targetID string) (*DeploymentTarget, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	t, exists := m.targets[targetID]
	if !exists {
		return nil, fmt.Errorf("target %q not registered", targetID)
	}
	clone := *t
	return &clone, nil
}

func generateID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}

// ExecuteRemoteInstall initiates an installation on a deployment target.
func (m *Manager) ExecuteRemoteInstall(
	ctx context.Context,
	principal *identity.Principal,
	targetID, pluginID, ver, source string,
	archiveData []byte,
) (CommandResult, error) {
	m.mu.Lock()
	target, exists := m.targets[targetID]
	if !exists {
		m.mu.Unlock()
		return CommandResult{}, fmt.Errorf("target %q not found", targetID)
	}
	agent, agentExists := m.targetAgents[targetID]
	if !agentExists {
		m.mu.Unlock()
		return CommandResult{}, fmt.Errorf("agent for target %q not connected", targetID)
	}

	m.sequences[targetID]++
	seq := m.sequences[targetID]
	m.mu.Unlock()

	cmd := RemoteCommand{
		CommandID:      generateID("cmd"),
		Type:           CmdInstall,
		OrganizationID: target.OrganizationID,
		TargetID:       targetID,
		PluginID:       pluginID,
		TargetVersion:  ver,
		Source:         source,
		Timestamp:      time.Now().UTC(),
		Nonce:          generateID("nonce"),
		Sequence:       seq,
		IdempotencyKey: fmt.Sprintf("install:%s:%s:%s", targetID, pluginID, ver),
	}

	res, err := agent.ValidateAndExecute(ctx, cmd, archiveData)

	// Update desired state in monitoring
	if m.monitor != nil {
		_, _ = m.monitor.SetDesiredState(ctx, target.OrganizationID, targetID, pluginID, monitoring.DesiredPluginState{
			Version:   ver,
			Enabled:   true,
			TargetEnv: target.Environment,
		})
		actualState := agent.GenerateTelemetry(pluginID)
		_, _ = m.monitor.RecordTelemetry(ctx, target.OrganizationID, targetID, pluginID, actualState)
	}

	// Audit Logging
	if m.auditLogger != nil {
		outcome := "success"
		if err != nil {
			outcome = "failed"
		}
		actorID := "system"
		actorType := identity.PrincipalServiceAccount
		if principal != nil {
			actorID = principal.ID
			actorType = principal.Type
		}
		_, _ = m.auditLogger.RecordEvent(audit.AuditEvent{
			OrganizationID:   target.OrganizationID,
			ActorID:          actorID,
			ActorType:        actorType,
			Action:           "plugin:install",
			Target:           pluginID,
			PluginID:         pluginID,
			Version:          ver,
			DeploymentTarget: targetID,
			Outcome:          outcome,
			CorrelationID:    cmd.CommandID,
		})
	}

	return res, err
}

// ExecuteRemoteUpdate initiates a version update with rollback support.
func (m *Manager) ExecuteRemoteUpdate(
	ctx context.Context,
	principal *identity.Principal,
	targetID, pluginID, targetVer, rollbackVer string,
	archiveData []byte,
) (CommandResult, error) {
	m.mu.Lock()
	target, exists := m.targets[targetID]
	if !exists {
		m.mu.Unlock()
		return CommandResult{}, fmt.Errorf("target %q not found", targetID)
	}
	agent, agentExists := m.targetAgents[targetID]
	if !agentExists {
		m.mu.Unlock()
		return CommandResult{}, fmt.Errorf("agent for target %q not connected", targetID)
	}

	m.sequences[targetID]++
	seq := m.sequences[targetID]
	m.mu.Unlock()

	cmd := RemoteCommand{
		CommandID:       generateID("cmd"),
		Type:            CmdUpdate,
		OrganizationID:  target.OrganizationID,
		TargetID:        targetID,
		PluginID:        pluginID,
		TargetVersion:   targetVer,
		RollbackVersion: rollbackVer,
		Timestamp:       time.Now().UTC(),
		Nonce:           generateID("nonce"),
		Sequence:        seq,
		IdempotencyKey:  fmt.Sprintf("update:%s:%s:%s", targetID, pluginID, targetVer),
	}

	res, err := agent.ValidateAndExecute(ctx, cmd, archiveData)

	if m.monitor != nil {
		activeVer := targetVer
		if res.RollbackDone {
			activeVer = rollbackVer
		}
		_, _ = m.monitor.SetDesiredState(ctx, target.OrganizationID, targetID, pluginID, monitoring.DesiredPluginState{
			Version:   activeVer,
			Enabled:   true,
			TargetEnv: target.Environment,
		})
		actualState := agent.GenerateTelemetry(pluginID)
		_, _ = m.monitor.RecordTelemetry(ctx, target.OrganizationID, targetID, pluginID, actualState)
	}

	if m.auditLogger != nil {
		outcome := "success"
		if err != nil {
			outcome = "failed"
		}
		actorID := "system"
		actorType := identity.PrincipalServiceAccount
		if principal != nil {
			actorID = principal.ID
			actorType = principal.Type
		}
		_, _ = m.auditLogger.RecordEvent(audit.AuditEvent{
			OrganizationID:   target.OrganizationID,
			ActorID:          actorID,
			ActorType:        actorType,
			Action:           "plugin:update",
			Target:           pluginID,
			PluginID:         pluginID,
			Version:          targetVer,
			DeploymentTarget: targetID,
			Outcome:          outcome,
			CorrelationID:    cmd.CommandID,
		})
	}

	return res, err
}

// ExecuteRemoteRemoval initiates a controlled removal.
func (m *Manager) ExecuteRemoteRemoval(
	ctx context.Context,
	principal *identity.Principal,
	targetID, pluginID string,
	purgeData bool,
) (CommandResult, error) {
	m.mu.Lock()
	target, exists := m.targets[targetID]
	if !exists {
		m.mu.Unlock()
		return CommandResult{}, fmt.Errorf("target %q not found", targetID)
	}
	agent, agentExists := m.targetAgents[targetID]
	if !agentExists {
		m.mu.Unlock()
		return CommandResult{}, fmt.Errorf("agent for target %q not connected", targetID)
	}

	m.sequences[targetID]++
	seq := m.sequences[targetID]
	m.mu.Unlock()

	cmd := RemoteCommand{
		CommandID:      generateID("cmd"),
		Type:           CmdRemove,
		OrganizationID: target.OrganizationID,
		TargetID:       targetID,
		PluginID:       pluginID,
		PurgeData:      purgeData,
		Timestamp:      time.Now().UTC(),
		Nonce:          generateID("nonce"),
		Sequence:       seq,
		IdempotencyKey: fmt.Sprintf("remove:%s:%s", targetID, pluginID),
	}

	res, err := agent.ValidateAndExecute(ctx, cmd, nil)

	if m.monitor != nil {
		_, _ = m.monitor.SetDesiredState(ctx, target.OrganizationID, targetID, pluginID, monitoring.DesiredPluginState{
			Version: "",
			Enabled: false,
		})
		actualState := agent.GenerateTelemetry(pluginID)
		_, _ = m.monitor.RecordTelemetry(ctx, target.OrganizationID, targetID, pluginID, actualState)
	}

	if m.auditLogger != nil {
		outcome := "success"
		if err != nil {
			outcome = "failed"
		}
		actorID := "system"
		actorType := identity.PrincipalServiceAccount
		if principal != nil {
			actorID = principal.ID
			actorType = principal.Type
		}
		_, _ = m.auditLogger.RecordEvent(audit.AuditEvent{
			OrganizationID:   target.OrganizationID,
			ActorID:          actorID,
			ActorType:        actorType,
			Action:           "plugin:remove",
			Target:           pluginID,
			PluginID:         pluginID,
			DeploymentTarget: targetID,
			Outcome:          outcome,
			CorrelationID:    cmd.CommandID,
		})
	}

	return res, err
}
