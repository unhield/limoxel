package registry_test

import (
	"sync"
	"testing"

	pkgerr "github.com/unhield/limoxel/internal/capabilities/plugin/errors"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/plugin/registry"
	"github.com/unhield/limoxel/plugin"
)

func mockRecord(idStr string) *registry.PluginRecord {
	m, _ := model.ParseManifestJSON([]byte(`{
		"schema_version": "1.0.0",
		"id": "` + idStr + `",
		"name": "Test Plugin",
		"version": "1.0.0"
	}`))
	return &registry.PluginRecord{
		Manifest:  m,
		Directory: "/tmp/" + idStr,
		State:     plugin.StateRegistered,
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := registry.NewRegistry()
	rec := mockRecord("org.limoxel.alpha")

	if err := reg.Register(rec); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	got, found := reg.Get(rec.Manifest.ID)
	if !found {
		t.Fatalf("expected to find %s", rec.Manifest.ID)
	}
	if got.Manifest.Name != "Test Plugin" {
		t.Errorf("unexpected record content: %+v", got)
	}

	// Duplicate registration
	err := reg.Register(rec)
	if err == nil {
		t.Fatal("expected error on duplicate registration, got nil")
	}
	pErr, ok := err.(*pkgerr.PluginError)
	if !ok || pErr.Code() != pkgerr.CodeDuplicateIdentity {
		t.Errorf("expected CodeDuplicateIdentity, got %v", err)
	}
}

func TestRegistry_ListAndSort(t *testing.T) {
	reg := registry.NewRegistry()

	_ = reg.Register(mockRecord("z.plugin"))
	_ = reg.Register(mockRecord("a.plugin"))
	_ = reg.Register(mockRecord("m.plugin"))

	list := reg.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(list))
	}

	// Must be deterministically sorted
	if string(list[0].Manifest.ID) != "a.plugin" ||
		string(list[1].Manifest.ID) != "m.plugin" ||
		string(list[2].Manifest.ID) != "z.plugin" {
		t.Errorf("list not sorted: %v", list)
	}
}

func TestRegistry_StateTransitions(t *testing.T) {
	reg := registry.NewRegistry()
	rec := mockRecord("a.plugin")
	_ = reg.Register(rec)

	if err := reg.UpdateState(rec.Manifest.ID, plugin.StateActive, nil); err != nil {
		t.Fatalf("UpdateState failed: %v", err)
	}

	active := reg.ListByState(plugin.StateActive)
	if len(active) != 1 || active[0].Manifest.ID != rec.Manifest.ID {
		t.Errorf("expected active plugin, got: %v", active)
	}

	// Cannot unregister active plugin
	err := reg.Unregister(rec.Manifest.ID)
	if err == nil {
		t.Fatal("expected error unregistering active plugin, got nil")
	}

	// Deactivate and unregister
	_ = reg.UpdateState(rec.Manifest.ID, plugin.StateUnloaded, nil)
	if err := reg.Unregister(rec.Manifest.ID); err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}
	if reg.Count() != 0 {
		t.Errorf("expected registry count 0, got %d", reg.Count())
	}
}

func TestRegistry_Concurrency(t *testing.T) {
	reg := registry.NewRegistry()
	rec := mockRecord("concurrent.plugin")
	_ = reg.Register(rec)

	var wg sync.WaitGroup
	workers := 20
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if w%2 == 0 {
					_, _ = reg.Get(rec.Manifest.ID)
					_ = reg.List()
					_ = reg.ListByState(plugin.StateRegistered)
				} else {
					_ = reg.UpdateState(rec.Manifest.ID, plugin.StateRegistered, nil)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestRegistry_DefensiveCopying(t *testing.T) {
	reg := registry.NewRegistry()
	rec := mockRecord("defensive.plugin")
	if err := reg.Register(rec); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Retrieve defensive copy
	got, ok := reg.Get("defensive.plugin")
	if !ok {
		t.Fatal("expected record to exist")
	}

	// Mutate retrieved copy's name and dependencies
	got.Manifest.Name = "MUTATED"
	got.Manifest.Dependencies = append(got.Manifest.Dependencies, model.Dependency{})

	// Re-fetch from registry and verify it was NOT mutated
	fresh, ok := reg.Get("defensive.plugin")
	if !ok {
		t.Fatal("expected record to exist")
	}
	if fresh.Manifest.Name == "MUTATED" {
		t.Errorf("manifest name in registry was mutated externally through leaked pointer!")
	}
	if len(fresh.Manifest.Dependencies) != 0 {
		t.Errorf("manifest dependencies in registry was mutated externally!")
	}

	// Test UpdateManifest with nil manifest
	if err := reg.UpdateManifest("defensive.plugin", nil); err == nil {
		t.Error("expected UpdateManifest with nil manifest to fail, got nil")
	}
}
