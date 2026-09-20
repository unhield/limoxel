package runtime

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestProcessSupervisor_GracefulExit(t *testing.T) {
	cfg := ProcessSupervisorConfig{
		Command: os.Args[0],
		Args:    []string{"-test.run=TestHelperProcess", "--"},
		Env: map[string]string{
			"GO_WANT_HELPER_PROCESS": "1",
			"HELPER_MODE":            "server",
		},
		GracefulTimeout: 2 * time.Second,
	}

	sup := NewProcessSupervisor(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stdout, stdin, err := sup.Start(ctx)
	if err != nil {
		t.Fatalf("supervisor Start failed: %v", err)
	}
	if stdout == nil || stdin == nil {
		t.Fatal("expected non-nil stdout and stdin")
	}

	pid := sup.PID()
	if pid <= 0 {
		t.Fatalf("expected positive PID, got %d", pid)
	}

	if sup.State() != StateRunning {
		t.Fatalf("expected state %s, got %s", StateRunning, sup.State())
	}

	// Trigger graceful stop (closes stdin)
	if err := sup.GracefulStop(); err != nil {
		t.Fatalf("GracefulStop failed: %v", err)
	}

	exited, err := sup.VerifyExited(2 * time.Second)
	if err != nil || !exited {
		t.Fatalf("VerifyExited failed: %v", err)
	}

	if sup.State() != StateStopped {
		t.Errorf("expected state %s, got %s", StateStopped, sup.State())
	}
}

func TestProcessSupervisor_ForcedKillOnTimeout(t *testing.T) {
	cfg := ProcessSupervisorConfig{
		Command: os.Args[0],
		Args:    []string{"-test.run=TestHelperProcess", "--"},
		Env: map[string]string{
			"GO_WANT_HELPER_PROCESS": "1",
			"HELPER_MODE":            "hang", // process sleeps for 10 minutes
		},
		GracefulTimeout: 100 * time.Millisecond, // very short timeout forces kill
	}

	sup := NewProcessSupervisor(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, err := sup.Start(ctx)
	if err != nil {
		t.Fatalf("supervisor Start failed: %v", err)
	}

	pid := sup.PID()
	if pid <= 0 {
		t.Fatalf("expected positive PID, got %d", pid)
	}

	// GracefulStop will exceed 100ms and cascade to ForceKill
	startTime := time.Now()
	_ = sup.GracefulStop()
	duration := time.Since(startTime)

	exited, err := sup.VerifyExited(2 * time.Second)
	if err != nil || !exited {
		t.Fatalf("VerifyExited failed after forced kill: %v", err)
	}

	// Ensure forced kill completed promptly without hanging
	if duration > 3*time.Second {
		t.Errorf("forced kill took too long: %v", duration)
	}
}
