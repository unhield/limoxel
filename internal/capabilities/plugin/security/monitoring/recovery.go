package monitoring

import (
	"fmt"
	"sync"
)

// RecoveryAction defines the automated mitigation response to a security violation.
type RecoveryAction string

const (
	// ActionNone indicates no intervention is needed.
	ActionNone RecoveryAction = "NONE"
	// ActionWarn logs a warning and allows execution to continue.
	ActionWarn RecoveryAction = "WARN"
	// ActionSuspend temporarily halts plugin execution.
	ActionSuspend RecoveryAction = "SUSPEND"
	// ActionTerminate immediately terminates the active plugin process.
	ActionTerminate RecoveryAction = "TERMINATE"
	// ActionQuarantine permanently prohibits the plugin from loading or restarting.
	ActionQuarantine RecoveryAction = "QUARANTINE"
)

// RecoveryConfig defines thresholds for automated incident mitigation.
type RecoveryConfig struct {
	MaxViolationsBeforeSuspend int
	MaxCrashesBeforeQuarantine int
	AutoQuarantineOnEscape     bool
}

// DefaultRecoveryConfig provides sensible mitigation thresholds.
func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{
		MaxViolationsBeforeSuspend: 5,
		MaxCrashesBeforeQuarantine: 3,
		AutoQuarantineOnEscape:     true,
	}
}

// RecoveryManager evaluates incident metrics and decides mitigation actions.
type RecoveryManager struct {
	mu      sync.RWMutex
	cfg     RecoveryConfig
	monitor *SecurityMonitor
}

// NewRecoveryManager constructs an initialized recovery manager.
func NewRecoveryManager(cfg RecoveryConfig, monitor *SecurityMonitor) *RecoveryManager {
	return &RecoveryManager{
		cfg:     cfg,
		monitor: monitor,
	}
}

// Evaluate determines the appropriate recovery action for a given event.
func (r *RecoveryManager) Evaluate(event SecurityEvent) RecoveryAction {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1. Critical sandbox escape attempt triggers immediate quarantine & termination
	if event.Category == CategorySandbox && event.Severity == SeverityCritical {
		if r.cfg.AutoQuarantineOnEscape && event.PluginID != "" {
			r.monitor.MarkQuarantined(event.PluginID, true)
			return ActionQuarantine
		}
		return ActionTerminate
	}

	// 2. Verification failure (tampered bytecode, forged signature) triggers quarantine
	if event.Category == CategoryVerification && (event.Decision == "REJECTED" || event.Severity == SeverityCritical) {
		if event.PluginID != "" {
			r.monitor.MarkQuarantined(event.PluginID, true)
		}
		return ActionQuarantine
	}

	// 3. Evaluate cumulative stats
	stats, ok := r.monitor.GetStats(event.PluginID)
	if !ok {
		return ActionNone
	}

	if stats.Quarantined {
		return ActionQuarantine
	}

	// Repeated crash threshold
	if r.cfg.MaxCrashesBeforeQuarantine > 0 && stats.Crashes >= r.cfg.MaxCrashesBeforeQuarantine {
		r.monitor.MarkQuarantined(event.PluginID, true)
		return ActionQuarantine
	}

	// Repeated violation threshold
	if r.cfg.MaxViolationsBeforeSuspend > 0 && stats.Violations >= r.cfg.MaxViolationsBeforeSuspend {
		r.monitor.MarkSuspended(event.PluginID, true)
		return ActionSuspend
	}

	if event.Decision == "DENIED" || event.Decision == "VIOLATION" {
		return ActionWarn
	}

	return ActionNone
}

// FormatActionMessage produces a structured diagnostic explaining the recovery decision.
func FormatActionMessage(action RecoveryAction, pluginID string, reason string) string {
	return fmt.Sprintf("security recovery action %s applied to plugin %s: %s", action, pluginID, reason)
}
