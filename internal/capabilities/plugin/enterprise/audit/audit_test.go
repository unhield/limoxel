package audit

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/identity"
)

func TestAuditLogger_ChainAndIntegrity(t *testing.T) {
	logDir := t.TempDir()
	logger, err := NewLogger(logDir)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	// 1. Record series of operations
	e1, err := logger.RecordEvent(AuditEvent{
		OrganizationID:   "org-acme",
		ActorID:          "alice-1",
		ActorType:        identity.PrincipalUser,
		Action:           "plugin:install",
		Target:           "acme.auth",
		PluginID:         "acme.auth",
		Version:          "1.0.0",
		DeploymentTarget: "prod-cluster-1",
		Outcome:          "success",
		CorrelationID:    "corr-001",
	})
	if err != nil {
		t.Fatalf("RecordEvent 1 failed: %v", err)
	}

	e2, err := logger.RecordEvent(AuditEvent{
		OrganizationID: "org-acme",
		ActorID:        "alice-1",
		ActorType:      identity.PrincipalUser,
		Action:         "policy:apply",
		Target:         "policy-allowlist",
		Outcome:        "success",
		CorrelationID:  "corr-002",
	})
	if err != nil {
		t.Fatalf("RecordEvent 2 failed: %v", err)
	}

	if e2.PrevHash != e1.Hash {
		t.Errorf("expected e2.PrevHash to match e1.Hash, got %s vs %s", e2.PrevHash, e1.Hash)
	}

	// 2. Verify Integrity
	valid, err := logger.VerifyIntegrity()
	if err != nil || !valid {
		t.Errorf("expected audit chain to be valid, got valid=%v, err=%v", valid, err)
	}

	// 3. Re-open logger and append more
	reopened, err := NewLogger(logDir)
	if err != nil {
		t.Fatalf("NewLogger reload failed: %v", err)
	}
	e3, err := reopened.RecordEvent(AuditEvent{
		OrganizationID: "org-acme",
		ActorID:        "sec-admin",
		ActorType:      identity.PrincipalUser,
		Action:         "audit:review",
		Target:         "system",
		Outcome:        "success",
		CorrelationID:  "corr-003",
	})
	if err != nil {
		t.Fatalf("RecordEvent 3 failed: %v", err)
	}
	if e3.PrevHash != e2.Hash {
		t.Errorf("expected e3.PrevHash to match e2.Hash across reloads, got %s vs %s", e3.PrevHash, e2.Hash)
	}

	valid, err = reopened.VerifyIntegrity()
	if err != nil || !valid {
		t.Errorf("expected reloaded audit chain to be valid, got valid=%v, err=%v", valid, err)
	}
}

func TestAuditLogger_TamperDetection(t *testing.T) {
	logDir := t.TempDir()
	logger, _ := NewLogger(logDir)

	for i := 0; i < 3; i++ {
		_, _ = logger.RecordEvent(AuditEvent{
			Timestamp:      time.Now().UTC(),
			OrganizationID: "org-acme",
			ActorID:        "admin",
			ActorType:      identity.PrincipalUser,
			Action:         "test:action",
			Outcome:        "success",
			CorrelationID:  "corr-test",
		})
	}

	logFile := filepath.Join(logDir, "enterprise_audit.log")
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}

	// Tamper with second line: replace action with malicious text
	tampered := strings.Replace(string(content), "test:action", "tampered:action", 1)
	if err := os.WriteFile(logFile, []byte(tampered), 0644); err != nil {
		t.Fatalf("failed to write tampered log: %v", err)
	}

	// Verify integrity must fail
	valid, err := logger.VerifyIntegrity()
	if valid || !errors.Is(err, ErrAuditTampered) {
		t.Errorf("expected ErrAuditTampered, got valid=%v, err=%v", valid, err)
	}
}

func TestAuditLogger_ReasonTamperDetection(t *testing.T) {
	logDir := t.TempDir()
	logger, _ := NewLogger(logDir)

	_, _ = logger.RecordEvent(AuditEvent{
		Timestamp:      time.Now().UTC(),
		OrganizationID: "org-acme",
		ActorID:        "admin",
		ActorType:      identity.PrincipalUser,
		Action:         "policy:deny",
		Outcome:        "denied",
		Reason:         "original policy denial reason",
		CorrelationID:  "corr-deny-1",
	})

	logFile := filepath.Join(logDir, "enterprise_audit.log")
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}

	// Tamper with reason field
	tampered := strings.Replace(string(content), "original policy denial reason", "forged approval justification", 1)
	if err := os.WriteFile(logFile, []byte(tampered), 0644); err != nil {
		t.Fatalf("failed to write tampered log: %v", err)
	}

	valid, err := logger.VerifyIntegrity()
	if valid || !errors.Is(err, ErrAuditTampered) {
		t.Errorf("expected ErrAuditTampered when Reason is forged, got valid=%v, err=%v", valid, err)
	}
}

func TestAuditLogger_ReadEventsTamperFails(t *testing.T) {
	logDir := t.TempDir()
	logger, _ := NewLogger(logDir)

	_, _ = logger.RecordEvent(AuditEvent{
		OrganizationID: "org-test",
		ActorID:        "alice",
		ActorType:      identity.PrincipalUser,
		Action:         "test:op",
		Outcome:        "success",
		CorrelationID:  "c1",
	})

	logFile := filepath.Join(logDir, "enterprise_audit.log")
	content, _ := os.ReadFile(logFile)

	// Tamper log on disk
	tampered := strings.Replace(string(content), "alice", "mallory", 1)
	_ = os.WriteFile(logFile, []byte(tampered), 0644)

	// ReadEvents must fail closed with ErrAuditTampered
	_, err := logger.ReadEvents()
	if !errors.Is(err, ErrAuditTampered) {
		t.Fatalf("expected ReadEvents to return ErrAuditTampered, got: %v", err)
	}

	// NewLogger must fail closed when opening tampered log
	_, err = NewLogger(logDir)
	if !errors.Is(err, ErrAuditTampered) {
		t.Fatalf("expected NewLogger to fail on tampered log with ErrAuditTampered, got: %v", err)
	}
}
