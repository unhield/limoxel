package runtime

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
)

func TestTrueProcessReload_GuaranteesTerminationAndReplacement(t *testing.T) {
	manifestJSON := []byte(`{
		"schema_version": "1.0.0",
		"id": "org.limoxel.reloadable",
		"name": "Reloadable Plugin",
		"version": "1.2.3"
	}`)
	manifest, err := model.ParseManifestJSON(manifestJSON)
	if err != nil {
		t.Fatalf("ParseManifestJSON failed: %v", err)
	}

	resolver := func(m *model.Manifest, dir string) (string, []string, map[string]string, error) {
		return os.Args[0], []string{"-test.run=TestHelperProcess", "--"}, map[string]string{
			"GO_WANT_HELPER_PROCESS": "1",
			"HELPER_MODE":            "server",
		}, nil
	}

	ldr := NewProcessLoader(resolver)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Initial Load
	inst1, err := ldr.Load(ctx, manifest, ".", nil)
	if err != nil {
		t.Fatalf("Load inst1 failed: %v", err)
	}
	defer func() {
		_ = ldr.Unload(context.Background(), inst1)
	}()

	pInst1, ok := inst1.(*processInstance)
	if !ok {
		t.Fatalf("expected *processInstance, got %T", inst1)
	}

	oldPID := pInst1.ProcessID()
	oldSession := pInst1.SessionID()

	if oldPID <= 0 {
		t.Fatalf("expected valid old PID > 0, got %d", oldPID)
	}
	if oldSession == "" {
		t.Fatal("expected non-empty old session ID")
	}

	// 2. Verify old runtime responds correctly
	resp1, err := pInst1.Runtime().Invoke(ctx, "echo", []byte("v1-payload"))
	if err != nil {
		t.Fatalf("inst1 invoke failed: %v", err)
	}
	if string(resp1) != "echo: v1-payload" {
		t.Errorf("unexpected inst1 response: %s", string(resp1))
	}

	// 3. Initiate Reload
	reloadedInst, err := ldr.Reload(ctx, inst1, nil)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	defer func() {
		_ = ldr.Unload(context.Background(), reloadedInst)
	}()

	pInst2, ok := reloadedInst.(*processInstance)
	if !ok {
		t.Fatalf("expected *processInstance for reloaded instance, got %T", reloadedInst)
	}

	newPID := pInst2.ProcessID()
	newSession := pInst2.SessionID()

	// 4. Assert old runtime is terminated and no longer reachable
	if pInst1.Runtime().State() != StateStopped {
		t.Errorf("expected old runtime state %s, got %s", StateStopped, pInst1.Runtime().State())
	}
	_, errOldInvoke := pInst1.Runtime().Invoke(ctx, "echo", []byte("dead-check"))
	if errOldInvoke == nil {
		t.Fatal("expected invoke on terminated old runtime to fail, but it succeeded")
	}

	// 5. Assert replacement runtime has distinct identity
	if newPID <= 0 {
		t.Fatalf("expected valid new PID > 0, got %d", newPID)
	}
	if newPID == oldPID {
		t.Errorf("expected replacement process to have different PID, but both were %d", newPID)
	}
	if newSession == oldSession {
		t.Errorf("expected replacement session to have different session ID, but both were %s", newSession)
	}

	// 6. Assert replacement runtime responds correctly
	resp2, err := pInst2.Runtime().Invoke(ctx, "echo", []byte("v2-payload"))
	if err != nil {
		t.Fatalf("reloaded instance invoke failed: %v", err)
	}
	if string(resp2) != "echo: v2-payload" {
		t.Errorf("unexpected reloaded instance response: %s", string(resp2))
	}
}
