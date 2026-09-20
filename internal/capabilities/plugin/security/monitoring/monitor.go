package monitoring

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var eventSeq uint64

// SecurityListener receives real-time security events.
type SecurityListener func(event SecurityEvent)

// PluginSecurityStats aggregates security metrics for an active plugin.
type PluginSecurityStats struct {
	PluginID        string    `json:"plugin_id"`
	TotalEvents     int       `json:"total_events"`
	Violations      int       `json:"violations"`
	Crashes         int       `json:"crashes"`
	LastViolationAt time.Time `json:"last_violation_at,omitempty"`
	LastCrashAt     time.Time `json:"last_crash_at,omitempty"`
	Quarantined     bool      `json:"quarantined"`
	Suspended       bool      `json:"suspended"`
}

// SecurityMonitor tracks, aggregates, and emits security events across all plugins.
type SecurityMonitor struct {
	mu          sync.RWMutex
	events      []SecurityEvent
	maxEvents   int
	listeners   []SecurityListener
	pluginStats map[string]*PluginSecurityStats
}

// NewSecurityMonitor constructs an initialized security monitor.
func NewSecurityMonitor(maxBufferedEvents int) *SecurityMonitor {
	if maxBufferedEvents <= 0 {
		maxBufferedEvents = 1000
	}
	return &SecurityMonitor{
		maxEvents:   maxBufferedEvents,
		pluginStats: make(map[string]*PluginSecurityStats),
	}
}

// Subscribe registers an observer for security events.
func (m *SecurityMonitor) Subscribe(l SecurityListener) {
	if l == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = append(m.listeners, l)
}

// Record emits and stores a security event.
func (m *SecurityMonitor) Record(event SecurityEvent) {
	seq := atomic.AddUint64(&eventSeq, 1)
	if event.ID == "" {
		event.ID = fmt.Sprintf("sec-%d-%d", seq, time.Now().UnixNano())
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	event.Reason = SanitizeReason(event.Reason)

	m.mu.Lock()

	// Append to bounded circular event buffer
	if len(m.events) >= m.maxEvents {
		m.events = m.events[1:]
	}
	m.events = append(m.events, event)

	// Update aggregated plugin stats
	if event.PluginID != "" {
		stats, exists := m.pluginStats[event.PluginID]
		if !exists {
			stats = &PluginSecurityStats{PluginID: event.PluginID}
			m.pluginStats[event.PluginID] = stats
		}
		stats.TotalEvents++
		if event.Decision == "DENIED" || event.Decision == "VIOLATION" || event.Severity == SeverityCritical {
			stats.Violations++
			stats.LastViolationAt = event.Timestamp
		}
	}

	// Snapshot listeners to call outside main lock
	listeners := make([]SecurityListener, len(m.listeners))
	copy(listeners, m.listeners)
	m.mu.Unlock()

	for _, l := range listeners {
		l(event)
	}
}

// RecordCrash logs an abnormal termination event for a plugin subprocess.
func (m *SecurityMonitor) RecordCrash(pluginID string, pid int, exitCode int, reason string) {
	event := SecurityEvent{
		Category:  CategoryRuntime,
		Severity:  SeverityCritical,
		PluginID:  pluginID,
		ProcessID: pid,
		Action:    "process.exit",
		Decision:  "CRASH",
		Reason:    fmt.Sprintf("plugin process terminated unexpectedly with exit code %d: %s", exitCode, reason),
	}

	m.mu.Lock()
	if stats, ok := m.pluginStats[pluginID]; ok {
		stats.Crashes++
		stats.LastCrashAt = time.Now().UTC()
	}
	m.mu.Unlock()

	m.Record(event)
}

// GetStats returns the security statistics for a specific plugin.
func (m *SecurityMonitor) GetStats(pluginID string) (PluginSecurityStats, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats, ok := m.pluginStats[pluginID]
	if !ok {
		return PluginSecurityStats{PluginID: pluginID}, false
	}
	return *stats, true
}

// GetEvents retrieves a copy of recent events matching the specified filter criteria.
func (m *SecurityMonitor) GetEvents(category EventCategory, pluginID string) []SecurityEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matching []SecurityEvent
	for _, e := range m.events {
		if category != "" && e.Category != category {
			continue
		}
		if pluginID != "" && e.PluginID != pluginID {
			continue
		}
		matching = append(matching, e)
	}
	return matching
}

// MarkSuspended updates the plugin status to suspended.
func (m *SecurityMonitor) MarkSuspended(pluginID string, suspended bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if stats, ok := m.pluginStats[pluginID]; ok {
		stats.Suspended = suspended
	}
}

// MarkQuarantined marks the plugin as quarantined due to high-severity violations.
func (m *SecurityMonitor) MarkQuarantined(pluginID string, quarantined bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if stats, ok := m.pluginStats[pluginID]; ok {
		stats.Quarantined = quarantined
	}
}
