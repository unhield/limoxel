package plugin_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	capPlugin "github.com/unhield/limoxel/internal/capabilities/plugin"
	"github.com/unhield/limoxel/internal/capabilities/plugin/loader"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/plugin"
)

type testEnginePlugin struct {
	id         string
	manifest   plugin.Manifest
	received   [][]byte
	sub        plugin.Subscription
	initCalled bool
	active     bool
	mu         sync.Mutex
}

func (p *testEnginePlugin) ID() string                { return p.id }
func (p *testEnginePlugin) Manifest() plugin.Manifest { return p.manifest }

func (p *testEnginePlugin) Init(ctx context.Context, host plugin.Host) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.initCalled = true

	// Test host event subscription
	sub, err := host.Events().Subscribe("workspace.events", func(ctx context.Context, topic string, payload []byte) error {
		p.mu.Lock()
		defer p.mu.Unlock()
		p.received = append(p.received, payload)
		return nil
	})
	if err != nil {
		return err
	}
	p.sub = sub

	// Test host repository access
	_, _ = host.Repository().Metadata(ctx)
	_, _ = host.Repository().ListFiles(ctx, "")
	host.Logger().Info("plugin initialized successfully", "id", p.id)

	return nil
}

func (p *testEnginePlugin) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.active = true
	return nil
}

func (p *testEnginePlugin) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.active = false
	if p.sub != nil {
		_ = p.sub.Unsubscribe()
	}
	return nil
}

func TestService_EndToEndWorkflow(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create workspace root
	wsDir := filepath.Join(tempDir, "workspace")
	_ = os.MkdirAll(filepath.Join(wsDir, "src"), 0755)
	_ = os.WriteFile(filepath.Join(wsDir, "src", "main.go"), []byte("package main\n"), 0644)

	// Create plugin directories
	pluginsRoot := filepath.Join(tempDir, "installed_plugins")
	dirA := filepath.Join(pluginsRoot, "plugin_a")
	dirB := filepath.Join(pluginsRoot, "plugin_b")
	_ = os.MkdirAll(dirA, 0755)
	_ = os.MkdirAll(dirB, 0755)

	// Plugin A: foundational plugin providing "linter.core"
	manifestA := `{
		"schema_version": "1.0.0",
		"id": "org.limoxel.alpha",
		"name": "Alpha Plugin",
		"version": "1.0.0",
		"capabilities": [{"name": "linter.core", "version": "1.0.0"}]
	}`
	_ = os.WriteFile(filepath.Join(dirA, "plugin.json"), []byte(manifestA), 0644)

	// Plugin B: depends on Plugin A
	manifestB := `{
		"schema_version": "1.0.0",
		"id": "org.limoxel.beta",
		"name": "Beta Plugin",
		"version": "1.0.0",
		"dependencies": [{"id": "org.limoxel.alpha", "constraint": ">= 1.0.0"}]
	}`
	_ = os.WriteFile(filepath.Join(dirB, "plugin.json"), []byte(manifestB), 0644)

	// Construct unified plugin Service
	ldr := loader.NewInProcessLoader()
	svc, err := capPlugin.NewService(capPlugin.ServiceOptions{
		WorkspaceRoot:     wsDir,
		SearchDirectories: []string{pluginsRoot},
		Loader:            ldr,
	})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	idA := model.MustParseIdentity("org.limoxel.alpha")
	idB := model.MustParseIdentity("org.limoxel.beta")

	plugA := &testEnginePlugin{id: string(idA)}
	plugB := &testEnginePlugin{id: string(idB)}

	ldr.RegisterFactory(idA, func() (plugin.Plugin, error) { return plugA, nil })
	ldr.RegisterFactory(idB, func() (plugin.Plugin, error) { return plugB, nil })

	// 1. Discover
	candidates, err := svc.Discover(ctx)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 discovered candidates, got %d", len(candidates))
	}

	// 2. Install
	for _, cand := range candidates {
		if _, err := svc.Install(ctx, cand); err != nil {
			t.Fatalf("Install(%s) failed: %v", cand.Manifest.ID, err)
		}
	}

	if len(svc.List()) != 2 {
		t.Fatalf("expected 2 registered plugins, got %d", len(svc.List()))
	}

	// 3. Activate
	if err := svc.Activate(ctx, idA); err != nil {
		t.Fatalf("Activate(A) failed: %v", err)
	}
	if err := svc.Activate(ctx, idB); err != nil {
		t.Fatalf("Activate(B) failed: %v", err)
	}

	active := svc.ListByState(plugin.StateActive)
	if len(active) != 2 {
		t.Errorf("expected 2 active plugins, got %d", len(active))
	}

	// 4. Test Event Routing to Plugin
	if err := svc.Host().Events().Publish(ctx, "workspace.events", []byte("hello-plugin")); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	plugA.mu.Lock()
	if len(plugA.received) != 1 || string(plugA.received[0]) != "hello-plugin" {
		t.Errorf("Plugin A did not receive event payload: %v", plugA.received)
	}
	plugA.mu.Unlock()

	// 5. Suspend and Resume Plugin B
	if err := svc.Suspend(ctx, idB); err != nil {
		t.Fatalf("Suspend(B) failed: %v", err)
	}
	recB, _ := svc.Get(idB)
	if recB.State != plugin.StateSuspended {
		t.Errorf("expected B StateSuspended, got %s", recB.State)
	}

	if err := svc.Activate(ctx, idB); err != nil {
		t.Fatalf("Resume(B) failed: %v", err)
	}
	recB, _ = svc.Get(idB)
	if recB.State != plugin.StateActive {
		t.Errorf("expected B StateActive after resume, got %s", recB.State)
	}

	// 6. Restart Plugin A
	if err := svc.Restart(ctx, idA); err != nil {
		t.Fatalf("Restart(A) failed: %v", err)
	}
	recA, _ := svc.Get(idA)
	if recA.State != plugin.StateActive {
		t.Errorf("expected A StateActive after restart, got %s", recA.State)
	}

	// 7. Close gracefully
	if err := svc.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	for _, rec := range svc.List() {
		if rec.State != plugin.StateUnloaded {
			t.Errorf("expected %s to be StateUnloaded after Close(), got %s", rec.Manifest.ID, rec.State)
		}
	}
}
