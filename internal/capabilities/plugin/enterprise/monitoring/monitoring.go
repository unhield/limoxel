package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	ErrDeploymentTargetNotFound = errors.New("enterprise monitoring: deployment target not found")
	ErrPluginRecordNotFound     = errors.New("enterprise monitoring: plugin monitoring record not found")
)

// HealthState represents the operational status of a managed plugin deployment.
type HealthState string

const (
	HealthHealthy           HealthState = "Healthy"
	HealthDegraded          HealthState = "Degraded"
	HealthUpdating          HealthState = "Updating"
	HealthFailed            HealthState = "Failed"
	HealthBlocked           HealthState = "Blocked"
	HealthPolicyDenied      HealthState = "PolicyDenied"
	HealthSecurityViolation HealthState = "SecurityViolation"
	HealthDisconnected      HealthState = "Disconnected"
	HealthRemoving          HealthState = "Removing"
	HealthRemoved           HealthState = "Removed"
	HealthUnknown           HealthState = "Unknown"
)

// DesiredPluginState defines the intended target state configured by administrators.
type DesiredPluginState struct {
	Version   string `json:"version"`
	Enabled   bool   `json:"enabled"`
	TargetEnv string `json:"target_env,omitempty"`
}

// ActualPluginState defines the current operational state reported by the host agent.
type ActualPluginState struct {
	Version      string      `json:"version"`
	Running      bool        `json:"running"`
	Health       HealthState `json:"health"`
	HealthReason string      `json:"health_reason,omitempty"`
	CrashCount   int         `json:"crash_count"`
	LastCrashAt  *time.Time  `json:"last_crash_at,omitempty"`
	MemoryBytes  int64       `json:"memory_bytes,omitempty"`
	ReportedAt   time.Time   `json:"reported_at"`
}

// ManagedPluginStatus tracks state discrepancies and telemetry for a plugin on a deployment target.
type ManagedPluginStatus struct {
	OrganizationID   string             `json:"organization_id"`
	DeploymentTarget string             `json:"deployment_target"`
	PluginID         string             `json:"plugin_id"`
	Desired          DesiredPluginState `json:"desired"`
	Actual           ActualPluginState  `json:"actual"`
	LastObservedAt   time.Time          `json:"last_observed_at"`
	HasDiscrepancy   bool               `json:"has_discrepancy"`
	DiscrepancyNotes string             `json:"discrepancy_notes,omitempty"`
}

// Monitor coordinates centralized observability and telemetry ingestion across enterprise deployments.
type Monitor struct {
	mu      sync.RWMutex
	baseDir string
	// orgID -> targetID -> pluginID -> ManagedPluginStatus
	data map[string]map[string]map[string]*ManagedPluginStatus
}

// NewMonitor constructs an enterprise monitoring coordinator.
func NewMonitor(baseDir string) (*Monitor, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create monitoring storage dir: %w", err)
	}

	m := &Monitor{
		baseDir: baseDir,
		data:    make(map[string]map[string]map[string]*ManagedPluginStatus),
	}

	if err := m.loadAll(); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *Monitor) stateFile(orgID string) string {
	return filepath.Join(m.baseDir, fmt.Sprintf("%s_monitoring.json", orgID))
}

func (m *Monitor) loadAll() error {
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return fmt.Errorf("failed to read monitoring dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			data, err := os.ReadFile(filepath.Join(m.baseDir, entry.Name()))
			if err != nil {
				continue
			}
			// Parse target map: targetID -> pluginID -> status
			var targetMap map[string]map[string]*ManagedPluginStatus
			if err := json.Unmarshal(data, &targetMap); err == nil {
				for _, plugins := range targetMap {
					for _, status := range plugins {
						if status != nil && status.OrganizationID != "" {
							m.ensureOrgMap(status.OrganizationID)
							if m.data[status.OrganizationID][status.DeploymentTarget] == nil {
								m.data[status.OrganizationID][status.DeploymentTarget] = make(map[string]*ManagedPluginStatus)
							}
							m.data[status.OrganizationID][status.DeploymentTarget][status.PluginID] = status
						}
					}
				}
			}
		}
	}
	return nil
}

func (m *Monitor) ensureOrgMap(orgID string) {
	if m.data[orgID] == nil {
		m.data[orgID] = make(map[string]map[string]*ManagedPluginStatus)
	}
}

func (m *Monitor) saveOrgLocked(orgID string) error {
	orgData := m.data[orgID]
	if orgData == nil {
		return nil
	}

	data, err := json.MarshalIndent(orgData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal monitoring data: %w", err)
	}

	tempPath := filepath.Join(m.baseDir, fmt.Sprintf(".mon-%s-%d.tmp", orgID, time.Now().UnixNano()))
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp monitoring file: %w", err)
	}

	if err := os.Rename(tempPath, m.stateFile(orgID)); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to commit monitoring file: %w", err)
	}

	return nil
}

// SetDesiredState configures the administrator's intended deployment state.
func (m *Monitor) SetDesiredState(ctx context.Context, orgID, targetID, pluginID string, desired DesiredPluginState) (*ManagedPluginStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ensureOrgMap(orgID)
	if m.data[orgID][targetID] == nil {
		m.data[orgID][targetID] = make(map[string]*ManagedPluginStatus)
	}

	status := m.data[orgID][targetID][pluginID]
	now := time.Now().UTC()
	if status == nil {
		status = &ManagedPluginStatus{
			OrganizationID:   orgID,
			DeploymentTarget: targetID,
			PluginID:         pluginID,
			Actual: ActualPluginState{
				Health: HealthUnknown,
			},
		}
		m.data[orgID][targetID][pluginID] = status
	}

	status.Desired = desired
	status.LastObservedAt = now
	m.calculateDiscrepancy(status)

	if err := m.saveOrgLocked(orgID); err != nil {
		return nil, err
	}

	clone := *status
	return &clone, nil
}

// RecordTelemetry reports actual execution telemetry received from a managed host agent.
func (m *Monitor) RecordTelemetry(ctx context.Context, orgID, targetID, pluginID string, actual ActualPluginState) (*ManagedPluginStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ensureOrgMap(orgID)
	if m.data[orgID][targetID] == nil {
		m.data[orgID][targetID] = make(map[string]*ManagedPluginStatus)
	}

	status := m.data[orgID][targetID][pluginID]
	now := time.Now().UTC()
	if status == nil {
		status = &ManagedPluginStatus{
			OrganizationID:   orgID,
			DeploymentTarget: targetID,
			PluginID:         pluginID,
		}
		m.data[orgID][targetID][pluginID] = status
	}

	actual.ReportedAt = now
	status.Actual = actual
	status.LastObservedAt = now
	m.calculateDiscrepancy(status)

	if err := m.saveOrgLocked(orgID); err != nil {
		return nil, err
	}

	clone := *status
	return &clone, nil
}

func (m *Monitor) calculateDiscrepancy(s *ManagedPluginStatus) {
	if s.Desired.Version != "" && s.Actual.Version != "" && s.Desired.Version != s.Actual.Version {
		s.HasDiscrepancy = true
		s.DiscrepancyNotes = fmt.Sprintf("version mismatch: desired %s, actual %s", s.Desired.Version, s.Actual.Version)
		return
	}
	if s.Desired.Enabled != s.Actual.Running {
		s.HasDiscrepancy = true
		s.DiscrepancyNotes = fmt.Sprintf("execution state mismatch: desired enabled=%v, actual running=%v", s.Desired.Enabled, s.Actual.Running)
		return
	}
	if s.Actual.Health != HealthHealthy && s.Actual.Health != HealthUnknown {
		s.HasDiscrepancy = true
		s.DiscrepancyNotes = fmt.Sprintf("unhealthy plugin: %s (%s)", s.Actual.Health, s.Actual.HealthReason)
		return
	}

	s.HasDiscrepancy = false
	s.DiscrepancyNotes = ""
}

// GetStatus returns the deployment status for a specific plugin on a deployment target.
func (m *Monitor) GetStatus(ctx context.Context, orgID, targetID, pluginID string) (*ManagedPluginStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	orgMap := m.data[orgID]
	if orgMap == nil || orgMap[targetID] == nil || orgMap[targetID][pluginID] == nil {
		return nil, ErrPluginRecordNotFound
	}

	clone := *orgMap[targetID][pluginID]
	return &clone, nil
}

// ListTargetStatus returns all monitored plugin records on a target.
func (m *Monitor) ListTargetStatus(ctx context.Context, orgID, targetID string) ([]ManagedPluginStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	orgMap := m.data[orgID]
	if orgMap == nil || orgMap[targetID] == nil {
		return nil, nil
	}

	var list []ManagedPluginStatus
	for _, status := range orgMap[targetID] {
		list = append(list, *status)
	}
	return list, nil
}
