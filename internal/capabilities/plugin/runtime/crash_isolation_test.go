package runtime

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
)

func TestCrashIsolation_ChildCrashLeavesHostIntact(t *testing.T) {
	manifestJSON := []byte(`{
		"schema_version": "1.0.0",
		"id": "org.limoxel.crashtest",
		"name": "Crash Test Plugin",
		"version": "1.0.0"
	}`)
	manifest, err := model.ParseManifestJSON(manifestJSON)
	if err != nil {
		t.Fatalf("ParseManifestJSON failed: %v", err)
	}

	// 1. Resolver launches a subprocess that immediately terminates with os.Exit(42)
	resolver := func(m *model.Manifest, dir string) (string, []string, map[string]string, error) {
		return os.Args[0], []string{"-test.run=TestHelperProcess", "--"}, map[string]string{
			"GO_WANT_HELPER_PROCESS": "1",
			"HELPER_MODE":            "crash", // exits with code 42
		}, nil
	}

	ldr := NewProcessLoader(resolver)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 2. Attempt to load the crashing plugin
	inst, err := ldr.Load(ctx, manifest, ".", nil)

	// 3. Verify that the load attempt detected failure
	if err == nil {
		t.Fatal("expected Load to fail for crashing child process, but got nil error")
	}

	// 4. Verify that the host process is healthy and did not crash
	if inst != nil {
		pInst, ok := inst.(*processInstance)
		if ok {
			if pInst.Runtime().State() != StateFailed {
				t.Errorf("expected runtime state %s, got %s", StateFailed, pInst.Runtime().State())
			}
		}
	}

	// 5. Verify that another normal plugin can run completely unaffected
	healthyManifestJSON := []byte(`{
		"schema_version": "1.0.0",
		"id": "org.limoxel.healthy",
		"name": "Healthy Plugin",
		"version": "1.0.0"
	}`)
	healthyManifest, _ := model.ParseManifestJSON(healthyManifestJSON)

	healthyResolver := func(m *model.Manifest, dir string) (string, []string, map[string]string, error) {
		return os.Args[0], []string{"-test.run=TestHelperProcess", "--"}, map[string]string{
			"GO_WANT_HELPER_PROCESS": "1",
			"HELPER_MODE":            "server",
		}, nil
	}

	healthyLoader := NewProcessLoader(healthyResolver)
	healthyInst, err := healthyLoader.Load(ctx, healthyManifest, ".", nil)
	if err != nil {
		t.Fatalf("loading healthy plugin after crash failed: %v", err)
	}
	defer func() {
		_ = healthyLoader.Unload(context.Background(), healthyInst)
	}()

	pHealthy, ok := healthyInst.(*processInstance)
	if !ok {
		t.Fatalf("expected *processInstance, got %T", healthyInst)
	}

	resp, err := pHealthy.Runtime().Invoke(ctx, "echo", []byte("isolation-ok"))
	if err != nil {
		t.Fatalf("healthy invoke failed: %v", err)
	}
	if string(resp) != "echo: isolation-ok" {
		t.Errorf("unexpected healthy invoke response: %s", string(resp))
	}
}
