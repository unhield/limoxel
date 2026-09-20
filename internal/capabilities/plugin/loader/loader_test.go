package loader_test

import (
	"context"
	"errors"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/plugin/adapter"
	pkgerr "github.com/unhield/limoxel/internal/capabilities/plugin/errors"
	"github.com/unhield/limoxel/internal/capabilities/plugin/loader"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/plugin"
)

type mockPluginInstance struct {
	id         string
	panicInit  bool
	panicStart bool
	panicStop  bool
	failInit   bool
	initCalled bool
	startCount int
	stopCount  int
}

func (m *mockPluginInstance) ID() string                { return m.id }
func (m *mockPluginInstance) Manifest() plugin.Manifest { return plugin.Manifest{ID: m.id} }
func (m *mockPluginInstance) Init(ctx context.Context, host plugin.Host) error {
	m.initCalled = true
	if m.panicInit {
		panic("boom init")
	}
	if m.failInit {
		return errors.New("init error")
	}
	return nil
}
func (m *mockPluginInstance) Start(ctx context.Context) error {
	m.startCount++
	if m.panicStart {
		panic("boom start")
	}
	return nil
}
func (m *mockPluginInstance) Stop(ctx context.Context) error {
	m.stopCount++
	if m.panicStop {
		panic("boom stop")
	}
	return nil
}

func TestInProcessLoader_LoadAndLifecycle(t *testing.T) {
	ctx := context.Background()
	ldr := loader.NewInProcessLoader()
	host := adapter.NewHostAdapter(t.TempDir())

	id := model.MustParseIdentity("org.limoxel.testplug")
	m, _ := model.ParseManifestJSON([]byte(`{
		"schema_version": "1.0.0",
		"id": "org.limoxel.testplug",
		"name": "Test",
		"version": "1.0.0"
	}`))

	instObj := &mockPluginInstance{id: string(id)}
	ldr.RegisterFactory(id, func() (plugin.Plugin, error) {
		return instObj, nil
	})

	inst, err := ldr.Load(ctx, m, t.TempDir(), host)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !instObj.initCalled {
		t.Error("expected Init() to be called on load")
	}

	if err := inst.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if instObj.startCount != 1 {
		t.Errorf("expected startCount 1, got %d", instObj.startCount)
	}

	if err := inst.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if instObj.stopCount != 1 {
		t.Errorf("expected stopCount 1, got %d", instObj.stopCount)
	}

	if err := ldr.Unload(ctx, inst); err != nil {
		t.Fatalf("Unload failed: %v", err)
	}
}

func TestInProcessLoader_PanicContainment(t *testing.T) {
	ctx := context.Background()
	ldr := loader.NewInProcessLoader()
	host := adapter.NewHostAdapter(t.TempDir())

	id := model.MustParseIdentity("org.limoxel.panicplug")
	m, _ := model.ParseManifestJSON([]byte(`{
		"schema_version": "1.0.0",
		"id": "org.limoxel.panicplug",
		"name": "Panic Plug",
		"version": "1.0.0"
	}`))

	// 1. Panic in Init
	instPanicInit := &mockPluginInstance{id: string(id), panicInit: true}
	ldr.RegisterFactory(id, func() (plugin.Plugin, error) {
		return instPanicInit, nil
	})

	_, err := ldr.Load(ctx, m, t.TempDir(), host)
	if err == nil {
		t.Fatal("expected error on plugin panic in Init(), got nil")
	}
	pErr, ok := err.(*pkgerr.PluginError)
	if !ok || pErr.Code() != pkgerr.CodeInitializationFailed {
		t.Errorf("expected CodeInitializationFailed, got %v", err)
	}

	// 2. Panic in Start
	instPanicStart := &mockPluginInstance{id: string(id), panicStart: true}
	ldr.RegisterFactory(id, func() (plugin.Plugin, error) {
		return instPanicStart, nil
	})

	inst, err := ldr.Load(ctx, m, t.TempDir(), host)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	err = inst.Start(ctx)
	if err == nil {
		t.Fatal("expected error on plugin panic in Start(), got nil")
	}

	// 3. Panic in Stop
	instPanicStop := &mockPluginInstance{id: string(id), panicStop: true}
	ldr.RegisterFactory(id, func() (plugin.Plugin, error) {
		return instPanicStop, nil
	})

	inst, _ = ldr.Load(ctx, m, t.TempDir(), host)
	_ = inst.Start(ctx)
	err = inst.Stop(ctx)
	if err == nil {
		t.Fatal("expected error on plugin panic in Stop(), got nil")
	}
}

func TestInProcessLoader_HonestReload(t *testing.T) {
	ctx := context.Background()
	ldr := loader.NewInProcessLoader()
	host := adapter.NewHostAdapter(t.TempDir())

	id := model.MustParseIdentity("org.limoxel.reloadable")
	m, _ := model.ParseManifestJSON([]byte(`{
		"schema_version": "1.0.0",
		"id": "org.limoxel.reloadable",
		"name": "Reloadable",
		"version": "1.0.0"
	}`))

	callCount := 0
	ldr.RegisterFactory(id, func() (plugin.Plugin, error) {
		callCount++
		return &mockPluginInstance{id: string(id)}, nil
	})

	inst, err := ldr.Load(ctx, m, t.TempDir(), host)
	if err != nil {
		t.Fatalf("initial Load failed: %v", err)
	}
	_ = inst.Start(ctx)

	if callCount != 1 {
		t.Fatalf("expected 1 factory invocation, got %d", callCount)
	}

	// Reload
	newInst, err := ldr.Reload(ctx, inst, host)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 factory invocations after reload, got %d", callCount)
	}

	_ = ldr.Unload(ctx, newInst)
}
