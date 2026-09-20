package permissions

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/unhield/limoxel/plugin/security"
)

func TestPermissionGrant_Matches(t *testing.T) {
	// Exact match with no scopes
	g1 := PermissionGrant{Action: security.PermRepoRead}
	if !g1.Matches(security.PermRepoRead, "") {
		t.Fatal("expected exact match to succeed")
	}
	if g1.Matches(security.PermRepoWrite, "") {
		t.Fatal("expected different action to fail")
	}

	// Wildcard action
	g2 := PermissionGrant{Action: "repository.*"}
	if !g2.Matches(security.PermRepoRead, "") {
		t.Fatal("expected wildcard action to match repository.read")
	}
	if !g2.Matches(security.PermRepoMetadata, "") {
		t.Fatal("expected wildcard action to match repository.metadata")
	}
	if g2.Matches(security.PermFileRead, "") {
		t.Fatal("expected wildcard action not to match file.read")
	}

	// Scoped match
	g3 := PermissionGrant{
		Action: security.PermFileRead,
		Scopes: []string{"src/**", "config/*.json"},
	}
	if !g3.Matches(security.PermFileRead, "src/pkg/main.go") {
		t.Fatal("expected src/** to match nested file")
	}
	if !g3.Matches(security.PermFileRead, "config/app.json") {
		t.Fatal("expected config/*.json to match file")
	}
	if g3.Matches(security.PermFileRead, "secrets/key.txt") {
		t.Fatal("expected ungranted scope to fail")
	}
}

func TestEngine_EvaluationAndFailClosed(t *testing.T) {
	engine := NewEngine()

	// 1. Unregistered plugin must fail closed
	res := engine.Evaluate(context.Background(), PermissionRequest{
		PluginID: "unregistered",
		Domain:   security.DomainRepository,
		Action:   security.PermRepoRead,
	})
	if res.Allowed() {
		t.Fatal("expected unregistered plugin to fail closed")
	}

	// 2. Register standard policy
	policy := NewStandardPolicy("test-plugin")
	engine.SetPolicy(policy)

	// Authorized read
	err := engine.Check(context.Background(), PermissionRequest{
		PluginID: "test-plugin",
		Domain:   security.DomainRepository,
		Action:   security.PermRepoRead,
	})
	if err != nil {
		t.Fatalf("expected allowed action to succeed: %v", err)
	}

	// Unauthorized write
	err = engine.Check(context.Background(), PermissionRequest{
		PluginID: "test-plugin",
		Domain:   security.DomainRepository,
		Action:   security.PermRepoWrite,
	})
	if err == nil {
		t.Fatal("expected unauthorized write to fail")
	}
}

func TestEngine_ViolationObserver(t *testing.T) {
	engine := NewEngine()
	var violations uint64

	engine.OnViolation(func(req PermissionRequest, res EvaluationResult) {
		atomic.AddUint64(&violations, 1)
	})

	_ = engine.Check(context.Background(), PermissionRequest{
		PluginID: "non-existent",
		Action:   security.PermFileDelete,
		Resource: "critical.db",
	})

	if atomic.LoadUint64(&violations) != 1 {
		t.Fatalf("expected 1 violation recorded, got %d", atomic.LoadUint64(&violations))
	}
}

func TestEngine_Concurrency(t *testing.T) {
	engine := NewEngine()
	policy := NewStandardPolicy("concurrent-plugin")
	engine.SetPolicy(policy)

	var wg sync.WaitGroup
	const goroutines = 50
	const iterations = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				action := security.PermRepoRead
				if j%2 == 0 {
					action = security.PermRepoWrite
				}
				_ = engine.Evaluate(context.Background(), PermissionRequest{
					PluginID: "concurrent-plugin",
					Action:   action,
					Resource: "test.go",
				})
			}
		}(i)
	}

	wg.Wait()
}
