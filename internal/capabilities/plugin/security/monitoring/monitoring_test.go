package monitoring

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestSecurityEvent_SanitizeReason(t *testing.T) {
	raw := "connection failed with password=supersecret123 on host"
	sanitized := SanitizeReason(raw)
	if sanitized == raw {
		t.Fatal("expected sensitive password to be redacted")
	}

	rawToken := "auth token=xyz987 failed"
	sanitizedToken := SanitizeReason(rawToken)
	if sanitizedToken == rawToken {
		t.Fatal("expected token to be redacted")
	}

	clean := "file read access denied for path /etc/passwd"
	if SanitizeReason(clean) != clean {
		t.Fatal("expected clean message to remain unchanged")
	}
}

func TestSecurityMonitor_RecordAndSubscribe(t *testing.T) {
	monitor := NewSecurityMonitor(100)

	var received uint64
	monitor.Subscribe(func(event SecurityEvent) {
		atomic.AddUint64(&received, 1)
	})

	monitor.Record(SecurityEvent{
		Category: CategoryPermission,
		Severity: SeverityWarning,
		PluginID: "plugin-alpha",
		Action:   "file.write",
		Decision: "DENIED",
		Reason:   "write not permitted",
	})

	if atomic.LoadUint64(&received) != 1 {
		t.Fatalf("expected 1 event delivered to subscriber, got %d", atomic.LoadUint64(&received))
	}

	stats, ok := monitor.GetStats("plugin-alpha")
	if !ok {
		t.Fatal("expected stats for plugin-alpha")
	}
	if stats.Violations != 1 {
		t.Fatalf("expected 1 violation, got %d", stats.Violations)
	}

	// Test Crash recording
	monitor.RecordCrash("plugin-alpha", 1234, -1, "segmentation fault")
	stats, _ = monitor.GetStats("plugin-alpha")
	if stats.Crashes != 1 {
		t.Fatalf("expected 1 crash recorded, got %d", stats.Crashes)
	}
}

func TestRecoveryManager_ThresholdEvaluation(t *testing.T) {
	monitor := NewSecurityMonitor(100)
	cfg := RecoveryConfig{
		MaxViolationsBeforeSuspend: 3,
		MaxCrashesBeforeQuarantine: 2,
		AutoQuarantineOnEscape:     true,
	}
	recovery := NewRecoveryManager(cfg, monitor)

	pluginID := "violator-plugin"

	// 1. Critical sandbox escape triggers immediate Quarantine
	escapeEvent := SecurityEvent{
		Category: CategorySandbox,
		Severity: SeverityCritical,
		PluginID: pluginID,
		Action:   "path.traversal",
		Decision: "VIOLATION",
		Reason:   "attempted traversal escape",
	}
	action := recovery.Evaluate(escapeEvent)
	if action != ActionQuarantine {
		t.Fatalf("expected ActionQuarantine for critical sandbox escape, got %s", action)
	}

	// 2. Normal violations increment towards suspension
	monitor.MarkQuarantined(pluginID, false) // Reset quarantine
	for i := 0; i < 3; i++ {
		monitor.Record(SecurityEvent{
			Category: CategoryPermission,
			Severity: SeverityWarning,
			PluginID: pluginID,
			Decision: "DENIED",
			Reason:   "access denied",
		})
	}

	permEvent := SecurityEvent{
		Category: CategoryPermission,
		PluginID: pluginID,
		Decision: "DENIED",
	}
	action = recovery.Evaluate(permEvent)
	if action != ActionSuspend {
		t.Fatalf("expected ActionSuspend after 3 violations, got %s", action)
	}
}

func TestSecurityMonitor_Concurrency(t *testing.T) {
	monitor := NewSecurityMonitor(500)
	var wg sync.WaitGroup
	const goroutines = 30
	const eventsPerGoroutine = 50

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				monitor.Record(SecurityEvent{
					Category: CategoryPermission,
					PluginID: "concurrent-plugin",
					Action:   "file.read",
					Decision: "GRANTED",
				})
			}
		}(i)
	}

	wg.Wait()

	events := monitor.GetEvents(CategoryPermission, "concurrent-plugin")
	if len(events) == 0 {
		t.Fatal("expected recorded events")
	}
}
