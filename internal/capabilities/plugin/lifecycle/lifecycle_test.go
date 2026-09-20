package lifecycle_test

import (
	"context"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/plugin/adapter"
	"github.com/unhield/limoxel/internal/capabilities/plugin/dependency"
	pkgerr "github.com/unhield/limoxel/internal/capabilities/plugin/errors"
	"github.com/unhield/limoxel/internal/capabilities/plugin/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/plugin/loader"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/plugin/registry"
	"github.com/unhield/limoxel/internal/capabilities/plugin/validation"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	"github.com/unhield/limoxel/plugin"
)

type dummyPlugin struct {
	id       string
	initFn   func() error
	startFn  func() error
	stopFn   func() error
	started  bool
	stopped  bool
	manifest plugin.Manifest
}

func (d *dummyPlugin) ID() string                { return d.id }
func (d *dummyPlugin) Manifest() plugin.Manifest { return d.manifest }
func (d *dummyPlugin) Init(ctx context.Context, host plugin.Host) error {
	if d.initFn != nil {
		return d.initFn()
	}
	return nil
}
func (d *dummyPlugin) Start(ctx context.Context) error {
	d.started = true
	if d.startFn != nil {
		return d.startFn()
	}
	return nil
}
func (d *dummyPlugin) Stop(ctx context.Context) error {
	d.stopped = true
	if d.stopFn != nil {
		return d.stopFn()
	}
	return nil
}

func TestLifecycleStateTransitions(t *testing.T) {
	// Valid transitions
	validPairs := [][2]plugin.State{
		{plugin.StateDiscovered, plugin.StateValidated},
		{plugin.StateValidated, plugin.StateInstalled},
		{plugin.StateInstalled, plugin.StateRegistered},
		{plugin.StateRegistered, plugin.StateActive},
		{plugin.StateActive, plugin.StateSuspended},
		{plugin.StateSuspended, plugin.StateActive},
		{plugin.StateActive, plugin.StateUnloaded},
		{plugin.StateUnloaded, plugin.StateRegistered},
		{plugin.StateRegistered, plugin.StateRemoving},
		{plugin.StateRemoving, plugin.StateRemoved},
		{plugin.StateFailed, plugin.StateRegistered},
	}

	for _, pair := range validPairs {
		if err := lifecycle.ValidateTransition(pair[0], pair[1]); err != nil {
			t.Errorf("expected valid transition from %s to %s, got %v", pair[0], pair[1], err)
		}
	}

	// Invalid transitions
	invalidPairs := [][2]plugin.State{
		{plugin.StateDiscovered, plugin.StateActive},
		{plugin.StateRemoved, plugin.StateActive},
		{plugin.StateRegistered, plugin.StateSuspended},
		{plugin.StateInstalled, plugin.StateActive},
	}

	for _, pair := range invalidPairs {
		if err := lifecycle.ValidateTransition(pair[0], pair[1]); err == nil {
			t.Errorf("expected error for invalid transition from %s to %s, got nil", pair[0], pair[1])
		}
	}
}

func TestLifecycleManager_FullWorkflow(t *testing.T) {
	ctx := context.Background()
	reg := registry.NewRegistry()
	ldr := loader.NewInProcessLoader()
	host := adapter.NewHostAdapter(t.TempDir())
	hostVer, _ := version.ParseSemVer("1.4.0")
	val := validation.NewValidator(hostVer)
	res := dependency.NewResolver()

	mgr := lifecycle.NewManager(reg, ldr, val, res, host)

	manifestData := []byte(`{
		"schema_version": "1.0.0",
		"id": "org.limoxel.workflow",
		"name": "Workflow Plugin",
		"version": "1.0.0"
	}`)
	m, err := model.ParseManifestJSON(manifestData)
	if err != nil {
		t.Fatal(err)
	}

	plug := &dummyPlugin{id: "org.limoxel.workflow", manifest: m.ToPublic()}
	ldr.RegisterFactory(m.ID, func() (plugin.Plugin, error) {
		return plug, nil
	})

	// 1. Install
	rec, err := mgr.Install(ctx, m, t.TempDir())
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	if rec.State != plugin.StateRegistered {
		t.Errorf("expected state REGISTERED, got %s", rec.State)
	}

	// 2. Activate
	if err := mgr.Activate(ctx, m.ID); err != nil {
		t.Fatalf("Activate failed: %v", err)
	}
	if !plug.started {
		t.Error("expected plugin to be started")
	}
	rec, _ = reg.Get(m.ID)
	if rec.State != plugin.StateActive {
		t.Errorf("expected state ACTIVE, got %s", rec.State)
	}

	// 3. Suspend
	if err := mgr.Suspend(ctx, m.ID); err != nil {
		t.Fatalf("Suspend failed: %v", err)
	}
	if !plug.stopped {
		t.Error("expected plugin to be stopped on suspension")
	}
	rec, _ = reg.Get(m.ID)
	if rec.State != plugin.StateSuspended {
		t.Errorf("expected state SUSPENDED, got %s", rec.State)
	}

	// 4. Resume (Activate from Suspended)
	plug.started = false
	if err := mgr.Activate(ctx, m.ID); err != nil {
		t.Fatalf("Activate from Suspended failed: %v", err)
	}
	rec, _ = reg.Get(m.ID)
	if rec.State != plugin.StateActive {
		t.Errorf("expected state ACTIVE, got %s", rec.State)
	}

	// 5. Restart
	if err := mgr.Restart(ctx, m.ID); err != nil {
		t.Fatalf("Restart failed: %v", err)
	}
	rec, _ = reg.Get(m.ID)
	if rec.State != plugin.StateActive {
		t.Errorf("expected state ACTIVE after restart, got %s", rec.State)
	}

	// 6. Unload
	if err := mgr.Unload(ctx, m.ID); err != nil {
		t.Fatalf("Unload failed: %v", err)
	}
	rec, _ = reg.Get(m.ID)
	if rec.State != plugin.StateUnloaded {
		t.Errorf("expected state UNLOADED, got %s", rec.State)
	}

	// 7. Remove
	if err := mgr.Remove(ctx, m.ID); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if _, exists := reg.Get(m.ID); exists {
		t.Errorf("expected plugin to be unregistered from registry")
	}
}

func TestLifecycleManager_MissingCapabilityFailsInstall(t *testing.T) {
	ctx := context.Background()
	reg := registry.NewRegistry()
	ldr := loader.NewInProcessLoader()
	host := adapter.NewHostAdapter(t.TempDir()) // has default capabilities, not custom.unavailable
	val := validation.NewValidator(host.HostVersion())
	res := dependency.NewResolver()
	mgr := lifecycle.NewManager(reg, ldr, val, res, host)

	manifestData := []byte(`{
		"schema_version": "1.0.0",
		"id": "org.limoxel.capreq",
		"name": "Capability Required Plugin",
		"version": "1.0.0",
		"required_capabilities": [
			{"name": "custom.unavailable", "optional": false}
		]
	}`)
	m, _ := model.ParseManifestJSON(manifestData)

	_, err := mgr.Install(ctx, m, t.TempDir())
	if err == nil {
		t.Fatal("expected Install to fail for missing mandatory capability, got nil")
	}

	pErr, ok := err.(*pkgerr.PluginError)
	if !ok || pErr.Code() != pkgerr.CodeDependencyMissing {
		t.Errorf("expected CodeDependencyMissing, got %v", err)
	}
}

func TestLifecycleManager_UpdateNilManifestFails(t *testing.T) {
	ctx := context.Background()
	reg := registry.NewRegistry()
	ldr := loader.NewInProcessLoader()
	host := adapter.NewHostAdapter(t.TempDir())
	val := validation.NewValidator(host.HostVersion())
	res := dependency.NewResolver()
	mgr := lifecycle.NewManager(reg, ldr, val, res, host)

	manifestData := []byte(`{
		"schema_version": "1.0.0",
		"id": "org.limoxel.niltest",
		"name": "Nil Test",
		"version": "1.0.0"
	}`)
	m, _ := model.ParseManifestJSON(manifestData)
	_, err := mgr.Install(ctx, m, t.TempDir())
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Calling Update with nil manifest should return an error without panicking
	err = mgr.Update(ctx, m.ID, nil)
	if err == nil {
		t.Fatal("expected error on Update with nil manifest, got nil")
	}
}

func TestLifecycleManager_RemoveUnloadFailureFailsClosed(t *testing.T) {
	ctx := context.Background()
	reg := registry.NewRegistry()
	ldr := loader.NewInProcessLoader()
	host := adapter.NewHostAdapter(t.TempDir())
	val := validation.NewValidator(host.HostVersion())
	res := dependency.NewResolver()
	mgr := lifecycle.NewManager(reg, ldr, val, res, host)

	manifestData := []byte(`{
		"schema_version": "1.0.0",
		"id": "org.limoxel.unloadfail",
		"name": "Unload Fail Test",
		"version": "1.0.0"
	}`)
	m, _ := model.ParseManifestJSON(manifestData)

	plug := &dummyPlugin{
		id:       "org.limoxel.unloadfail",
		manifest: m.ToPublic(),
		stopFn: func() error {
			return pkgerr.New(pkgerr.CodeUnloadingFailed, m.ID, "simulated stop failure")
		},
	}
	ldr.RegisterFactory(m.ID, func() (plugin.Plugin, error) {
		return plug, nil
	})

	_, err := mgr.Install(ctx, m, t.TempDir())
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	if err := mgr.Activate(ctx, m.ID); err != nil {
		t.Fatalf("Activate failed: %v", err)
	}

	// Remove should attempt unload, fail, and leave plugin in StateFailed without unregistering
	err = mgr.Remove(ctx, m.ID)
	if err == nil {
		t.Fatal("expected Remove to fail when instance unload fails, got nil")
	}

	rec, exists := reg.Get(m.ID)
	if !exists {
		t.Fatal("plugin was unregistered despite unload failure!")
	}
	if rec.State != plugin.StateFailed {
		t.Errorf("expected plugin state FAILED, got %s", rec.State)
	}
}
