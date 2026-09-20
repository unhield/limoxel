package management

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/audit"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/identity"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/monitoring"
)

type mockHostExecutor struct {
	installed       map[string]string // pluginID -> ver
	failOn          string
	failUpdateCount int
}

func newMockHostExecutor() *mockHostExecutor {
	return &mockHostExecutor{
		installed: make(map[string]string),
	}
}

func (m *mockHostExecutor) Install(ctx context.Context, pluginID, version string, archiveData []byte) error {
	if m.failOn == "install" {
		return errors.New("simulated install error")
	}
	m.installed[pluginID] = version
	return nil
}

func (m *mockHostExecutor) Update(ctx context.Context, pluginID, targetVersion string, archiveData []byte) error {
	if m.failUpdateCount > 0 {
		m.failUpdateCount--
		return errors.New("simulated update error")
	}
	if m.failOn == "update" {
		return errors.New("simulated update error")
	}
	m.installed[pluginID] = targetVersion
	return nil
}

func (m *mockHostExecutor) Remove(ctx context.Context, pluginID string, purgeData bool) error {
	delete(m.installed, pluginID)
	return nil
}

func (m *mockHostExecutor) Suspend(ctx context.Context, pluginID string) error {
	return nil
}

func (m *mockHostExecutor) Restart(ctx context.Context, pluginID string) error {
	return nil
}

func (m *mockHostExecutor) GetInstalled() map[string]string {
	clone := make(map[string]string)
	for k, v := range m.installed {
		clone[k] = v
	}
	return clone
}

func TestManagement_InstallUpdateRemove(t *testing.T) {
	tempDir := t.TempDir()
	mon, err := monitoring.NewMonitor(tempDir)
	if err != nil {
		t.Fatalf("NewMonitor failed: %v", err)
	}
	auditLog, err := audit.NewLogger(tempDir)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	mgr := NewManager(mon, auditLog)

	executor := newMockHostExecutor()
	agent := NewManagedAgent("org-acme", "target-host-1", executor)

	target := DeploymentTarget{
		ID:             "target-host-1",
		OrganizationID: "org-acme",
		Name:           "Production Cluster Node 1",
		Environment:    "production",
		Platform:       "linux/amd64",
		RegisteredAt:   time.Now().UTC(),
		Active:         true,
	}

	if err := mgr.RegisterTarget(target, agent); err != nil {
		t.Fatalf("RegisterTarget failed: %v", err)
	}

	ctx := context.Background()
	adminPrincipal := &identity.Principal{
		ID:             "admin-alice",
		OrganizationID: "org-acme",
		Type:           identity.PrincipalUser,
		Roles:          []identity.Role{identity.RoleAdmin},
		Active:         true,
	}

	// 1. Remote Install
	res, err := mgr.ExecuteRemoteInstall(ctx, adminPrincipal, "target-host-1", "acme.auth", "1.0.0", "private_registry", []byte("pkg bytes"))
	if err != nil || !res.Success {
		t.Fatalf("ExecuteRemoteInstall failed: %v, res: %+v", err, res)
	}
	if executor.installed["acme.auth"] != "1.0.0" {
		t.Errorf("expected installed 1.0.0, got %s", executor.installed["acme.auth"])
	}

	// 2. Remote Update
	resUp, err := mgr.ExecuteRemoteUpdate(ctx, adminPrincipal, "target-host-1", "acme.auth", "1.1.0", "1.0.0", []byte("pkg 1.1.0"))
	if err != nil || !resUp.Success {
		t.Fatalf("ExecuteRemoteUpdate failed: %v, res: %+v", err, resUp)
	}
	if executor.installed["acme.auth"] != "1.1.0" {
		t.Errorf("expected updated 1.1.0, got %s", executor.installed["acme.auth"])
	}

	// 3. Remote Removal
	resRem, err := mgr.ExecuteRemoteRemoval(ctx, adminPrincipal, "target-host-1", "acme.auth", true)
	if err != nil || !resRem.Success {
		t.Fatalf("ExecuteRemoteRemoval failed: %v, res: %+v", err, resRem)
	}
	if _, exists := executor.installed["acme.auth"]; exists {
		t.Errorf("expected acme.auth to be removed")
	}

	// 4. Verify Audit Events
	events, err := auditLog.ReadEvents()
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	if len(events) != 3 {
		t.Errorf("expected 3 audit events, got %d", len(events))
	}
	valid, err := auditLog.VerifyIntegrity()
	if err != nil || !valid {
		t.Errorf("audit chain integrity failed: %v", err)
	}
}

func TestManagement_ReplayProtectionAndIdempotency(t *testing.T) {
	executor := newMockHostExecutor()
	agent := NewManagedAgent("org-acme", "target-1", executor)
	ctx := context.Background()

	now := time.Now().UTC()
	cmd1 := RemoteCommand{
		CommandID:      "cmd-001",
		Type:           CmdInstall,
		OrganizationID: "org-acme",
		TargetID:       "target-1",
		PluginID:       "acme.tool",
		TargetVersion:  "1.0.0",
		Timestamp:      now,
		Nonce:          "nonce-abc",
		Sequence:       1,
		IdempotencyKey: "idem-key-1",
	}

	// 1. Initial execution succeeds
	res1, err := agent.ValidateAndExecute(ctx, cmd1, nil)
	if err != nil || !res1.Success {
		t.Fatalf("first execution failed: %v", err)
	}

	// 2. Replay with identical idempotency key returns cached success without re-executing
	resIdem, err := agent.ValidateAndExecute(ctx, cmd1, nil)
	if err != nil || !resIdem.Success {
		t.Fatalf("idempotent execution failed: %v", err)
	}

	// 3. Replay with same nonce but different command ID fails
	cmdReplayNonce := RemoteCommand{
		CommandID:      "cmd-002",
		Type:           CmdInstall,
		OrganizationID: "org-acme",
		TargetID:       "target-1",
		PluginID:       "acme.tool",
		TargetVersion:  "1.0.0",
		Timestamp:      now,
		Nonce:          "nonce-abc", // Reused!
		Sequence:       2,
	}
	_, err = agent.ValidateAndExecute(ctx, cmdReplayNonce, nil)
	if !errors.Is(err, ErrReplayDetected) {
		t.Errorf("expected ErrReplayDetected on reused nonce, got %v", err)
	}

	// 4. Stale timestamp (> 5 minutes ago) fails
	cmdStale := RemoteCommand{
		CommandID:      "cmd-003",
		Type:           CmdInstall,
		OrganizationID: "org-acme",
		TargetID:       "target-1",
		PluginID:       "acme.tool",
		TargetVersion:  "1.0.0",
		Timestamp:      now.Add(-10 * time.Minute), // 10 minutes ago
		Nonce:          "nonce-fresh",
		Sequence:       3,
	}
	_, err = agent.ValidateAndExecute(ctx, cmdStale, nil)
	if !errors.Is(err, ErrReplayDetected) {
		t.Errorf("expected ErrReplayDetected on stale timestamp, got %v", err)
	}

	// 5. Monotonic sequence violation fails (sequence 1 <= current 1)
	cmdBadSeq := RemoteCommand{
		CommandID:      "cmd-004",
		Type:           CmdInstall,
		OrganizationID: "org-acme",
		TargetID:       "target-1",
		PluginID:       "acme.tool",
		TargetVersion:  "1.0.0",
		Timestamp:      now,
		Nonce:          "nonce-fresh-2",
		Sequence:       1, // Sequence 1 already used!
	}
	_, err = agent.ValidateAndExecute(ctx, cmdBadSeq, nil)
	if !errors.Is(err, ErrReplayDetected) {
		t.Errorf("expected ErrReplayDetected on sequence regression, got %v", err)
	}
}

func TestManagement_RollbackOnUpdateFailure(t *testing.T) {
	executor := newMockHostExecutor()
	executor.installed["acme.tool"] = "1.0.0"
	executor.failUpdateCount = 1 // force initial update to fail, rollback will succeed

	agent := NewManagedAgent("org-acme", "target-1", executor)
	ctx := context.Background()

	cmd := RemoteCommand{
		CommandID:       "cmd-update-fail",
		Type:            CmdUpdate,
		OrganizationID:  "org-acme",
		TargetID:        "target-1",
		PluginID:        "acme.tool",
		TargetVersion:   "2.0.0",
		RollbackVersion: "1.0.0",
		Timestamp:       time.Now().UTC(),
		Nonce:           "nonce-rb",
		Sequence:        1,
	}

	res, err := agent.ValidateAndExecute(ctx, cmd, nil)
	if err == nil {
		t.Fatalf("expected update to fail")
	}
	if !res.RollbackDone {
		t.Errorf("expected rollback to be performed on failure")
	}
	if executor.installed["acme.tool"] != "1.0.0" {
		t.Errorf("expected plugin to be rolled back to 1.0.0, got %s", executor.installed["acme.tool"])
	}
}

func TestManagement_TenantCheckBeforeIdempotency(t *testing.T) {
	executor := newMockHostExecutor()
	agent := NewManagedAgent("org-legit", "target-1", executor)
	ctx := context.Background()

	// First command succeeds and caches idempotency key
	cmdLegit := RemoteCommand{
		CommandID:      "cmd-100",
		Type:           CmdInstall,
		OrganizationID: "org-legit",
		TargetID:       "target-1",
		PluginID:       "legit.tool",
		TargetVersion:  "1.0.0",
		Timestamp:      time.Now().UTC(),
		Nonce:          "nonce-100",
		Sequence:       1,
		IdempotencyKey: "idem-key-1",
	}
	res1, err := agent.ValidateAndExecute(ctx, cmdLegit, nil)
	if err != nil || !res1.Success {
		t.Fatalf("first command failed: %v", err)
	}

	// Adversarial command uses same IdempotencyKey but foreign OrganizationID
	cmdAttacker := RemoteCommand{
		CommandID:      "cmd-200",
		Type:           CmdInstall,
		OrganizationID: "org-attacker",
		TargetID:       "target-1",
		PluginID:       "legit.tool",
		TargetVersion:  "1.0.0",
		Timestamp:      time.Now().UTC(),
		Nonce:          "nonce-200",
		Sequence:       2,
		IdempotencyKey: "idem-key-1",
	}
	res2, err := agent.ValidateAndExecute(ctx, cmdAttacker, nil)
	if !errors.Is(err, ErrReplayDetected) || res2.Success {
		t.Errorf("expected cross-tenant command with cached idempotency key to be rejected, got res: %+v, err: %v", res2, err)
	}
}

func TestManagement_TargetMismatch(t *testing.T) {
	executor := newMockHostExecutor()
	agent := NewManagedAgent("org-acme", "target-alpha", executor)
	ctx := context.Background()

	cmd := RemoteCommand{
		CommandID:      "cmd-mismatch",
		Type:           CmdInstall,
		OrganizationID: "org-acme",
		TargetID:       "target-beta", // mismatch!
		PluginID:       "test.plugin",
		TargetVersion:  "1.0.0",
		Timestamp:      time.Now().UTC(),
		Nonce:          "nonce-target-mismatch",
		Sequence:       1,
	}

	res, err := agent.ValidateAndExecute(ctx, cmd, nil)
	if !errors.Is(err, ErrReplayDetected) || res.Success {
		t.Fatalf("expected target mismatch to be rejected with ErrReplayDetected, got res=%+v, err=%v", res, err)
	}
}

func TestManagement_DoubleFailureRollbackFailed(t *testing.T) {
	executor := newMockHostExecutor()
	executor.installed["acme.tool"] = "1.0.0"
	// Fail 2 consecutive update calls (both update and rollback)
	executor.failUpdateCount = 2

	agent := NewManagedAgent("org-acme", "target-1", executor)
	ctx := context.Background()

	cmd := RemoteCommand{
		CommandID:       "cmd-double-fail",
		Type:            CmdUpdate,
		OrganizationID:  "org-acme",
		TargetID:        "target-1",
		PluginID:        "acme.tool",
		TargetVersion:   "2.0.0",
		RollbackVersion: "1.0.0",
		Timestamp:       time.Now().UTC(),
		Nonce:           "nonce-double-fail",
		Sequence:        1,
	}

	res, err := agent.ValidateAndExecute(ctx, cmd, nil)
	if err == nil {
		t.Fatalf("expected update to fail")
	}
	if !res.RollbackFailed {
		t.Errorf("expected RollbackFailed to be true when rollback also fails")
	}
	if res.RollbackDone {
		t.Errorf("expected RollbackDone to be false when rollback fails")
	}
}

type mockTelemetryExecutor struct {
	*mockHostExecutor
	crashes int
	mem     int64
}

func (m *mockTelemetryExecutor) GetPluginTelemetry(pluginID string) (int, int64, error) {
	return m.crashes, m.mem, nil
}

func TestManagement_TelemetryRealMetrics(t *testing.T) {
	// 1. Basic executor without TelemetryProvider: should NOT have synthetic 16MB
	baseExec := newMockHostExecutor()
	baseExec.installed["p1"] = "1.0.0"
	agent1 := NewManagedAgent("org-1", "t1", baseExec)

	telem1 := agent1.GenerateTelemetry("p1")
	if telem1.MemoryBytes != 0 {
		t.Errorf("expected 0 memory bytes (no synthetic metric) when not supported, got %d", telem1.MemoryBytes)
	}

	// 2. Executor with real TelemetryProvider
	telExec := &mockTelemetryExecutor{
		mockHostExecutor: baseExec,
		crashes:          3,
		mem:              42 * 1024 * 1024,
	}
	agent2 := NewManagedAgent("org-1", "t1", telExec)
	telem2 := agent2.GenerateTelemetry("p1")
	if telem2.CrashCount != 3 || telem2.MemoryBytes != 42*1024*1024 {
		t.Errorf("expected real metrics (3 crashes, 42MB), got %+v", telem2)
	}
}
