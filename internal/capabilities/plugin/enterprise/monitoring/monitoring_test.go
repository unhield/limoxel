package monitoring

import (
	"context"
	"testing"
)

func TestMonitor_StateDiscrepancyAndPersistence(t *testing.T) {
	tempDir := t.TempDir()
	mon, err := NewMonitor(tempDir)
	if err != nil {
		t.Fatalf("NewMonitor failed: %v", err)
	}

	ctx := context.Background()

	// 1. Configure Desired State
	status, err := mon.SetDesiredState(ctx, "org-acme", "cluster-prod-1", "acme.auth", DesiredPluginState{
		Version: "2.0.0",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("SetDesiredState failed: %v", err)
	}
	if status.Desired.Version != "2.0.0" {
		t.Errorf("expected desired version 2.0.0, got %s", status.Desired.Version)
	}

	// 2. Report Telemetry with version lag (v1.9.0 vs v2.0.0) -> Discrepancy!
	telemetryStatus, err := mon.RecordTelemetry(ctx, "org-acme", "cluster-prod-1", "acme.auth", ActualPluginState{
		Version: "1.9.0",
		Running: true,
		Health:  HealthHealthy,
	})
	if err != nil {
		t.Fatalf("RecordTelemetry failed: %v", err)
	}
	if !telemetryStatus.HasDiscrepancy {
		t.Errorf("expected discrepancy due to version mismatch, got false")
	}

	// 3. Report Telemetry matching desired state -> Discrepancy resolved!
	resolvedStatus, err := mon.RecordTelemetry(ctx, "org-acme", "cluster-prod-1", "acme.auth", ActualPluginState{
		Version: "2.0.0",
		Running: true,
		Health:  HealthHealthy,
	})
	if err != nil {
		t.Fatalf("RecordTelemetry matching failed: %v", err)
	}
	if resolvedStatus.HasDiscrepancy {
		t.Errorf("expected no discrepancy when versions match, got true: %s", resolvedStatus.DiscrepancyNotes)
	}

	// 4. Persistence Reload
	monReloaded, err := NewMonitor(tempDir)
	if err != nil {
		t.Fatalf("NewMonitor reload failed: %v", err)
	}

	loadedStatus, err := monReloaded.GetStatus(ctx, "org-acme", "cluster-prod-1", "acme.auth")
	if err != nil {
		t.Fatalf("GetStatus after reload failed: %v", err)
	}
	if loadedStatus.Desired.Version != "2.0.0" || loadedStatus.Actual.Version != "2.0.0" {
		t.Errorf("unexpected loaded state: %+v", loadedStatus)
	}
}
