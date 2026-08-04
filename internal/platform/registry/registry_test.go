package registry_test

import (
	"errors"
	"testing"

	"github.com/unhield/limoxel/internal/platform/registry"
)

func TestRegistryRegistrationAndLookup(t *testing.T) {
	reg := registry.New()

	meta := registry.Metadata{"version": "1.0"}
	err := reg.RegisterComponent("comp-a", "service", "instance-a", meta)
	if err != nil {
		t.Fatalf("RegisterComponent failed: %v", err)
	}

	if !reg.Has("comp-a") {
		t.Error("expected reg.Has('comp-a') to be true")
	}
	if reg.Count() != 1 {
		t.Errorf("got count %d, want 1", reg.Count())
	}

	entry, err := reg.Get("comp-a")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if entry.Name != "comp-a" || entry.Type != "service" || entry.Instance != "instance-a" {
		t.Errorf("unexpected entry values: %v", entry)
	}

	// Verify Metadata defensive copy
	meta["version"] = "mutated"
	if entry.Metadata["version"] == "mutated" {
		t.Error("metadata mutation leaked to registered entry")
	}
}

func TestRegistryDuplicateAndValidationErrors(t *testing.T) {
	reg := registry.New()

	t.Run("empty name", func(t *testing.T) {
		err := reg.Register(registry.Entry{Name: "", Type: "svc", Instance: "inst"})
		if !errors.Is(err, registry.ErrEmptyName) {
			t.Errorf("got %v, want ErrEmptyName", err)
		}
	})

	t.Run("empty type", func(t *testing.T) {
		err := reg.Register(registry.Entry{Name: "n", Type: "", Instance: "inst"})
		if !errors.Is(err, registry.ErrEmptyType) {
			t.Errorf("got %v, want ErrEmptyType", err)
		}
	})

	t.Run("nil instance", func(t *testing.T) {
		err := reg.Register(registry.Entry{Name: "n", Type: "svc", Instance: nil})
		if !errors.Is(err, registry.ErrEntryNil) {
			t.Errorf("got %v, want ErrEntryNil", err)
		}
	})

	t.Run("duplicate component", func(t *testing.T) {
		_ = reg.RegisterComponent("c1", "svc", "inst1", nil)
		err := reg.RegisterComponent("c1", "svc", "inst2", nil)
		if !errors.Is(err, registry.ErrDuplicateComponent) {
			t.Errorf("got %v, want ErrDuplicateComponent", err)
		}
	})
}

func TestRegistryOrderingUnregisterAndGetByType(t *testing.T) {
	reg := registry.New()

	_ = reg.RegisterComponent("c1", "service", "inst1", nil)
	_ = reg.RegisterComponent("c2", "driver", "inst2", nil)
	_ = reg.RegisterComponent("c3", "service", "inst3", nil)

	// Test List ordering
	list := reg.List()
	if len(list) != 3 {
		t.Fatalf("got list len %d, want 3", len(list))
	}
	if list[0].Name != "c1" || list[1].Name != "c2" || list[2].Name != "c3" {
		t.Errorf("list ordering mismatch: %v", list)
	}

	// Test GetByType
	svcs := reg.GetByType("service")
	if len(svcs) != 2 {
		t.Fatalf("got %d service entries, want 2", len(svcs))
	}
	if svcs[0].Name != "c1" || svcs[1].Name != "c3" {
		t.Errorf("GetByType mismatch: %v", svcs)
	}

	// Test Unregister
	if err := reg.Unregister("c2"); err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}
	if reg.Has("c2") {
		t.Error("c2 should be unregistered")
	}
	if reg.Count() != 2 {
		t.Errorf("got count %d, want 2", reg.Count())
	}

	// Unregister non-existent
	if err := reg.Unregister("missing"); !errors.Is(err, registry.ErrComponentNotFound) {
		t.Errorf("got %v, want ErrComponentNotFound", err)
	}
}

func TestNilRegistrySafety(t *testing.T) {
	var reg *registry.Registry

	if err := reg.RegisterComponent("name", "type", "inst", nil); !errors.Is(err, registry.ErrRegistryNil) {
		t.Errorf("got %v, want ErrRegistryNil", err)
	}
	if err := reg.Unregister("name"); !errors.Is(err, registry.ErrRegistryNil) {
		t.Errorf("got %v, want ErrRegistryNil", err)
	}
	if _, err := reg.Get("name"); !errors.Is(err, registry.ErrRegistryNil) {
		t.Errorf("got %v, want ErrRegistryNil", err)
	}
	if len(reg.GetByType("type")) != 0 {
		t.Error("expected empty slice from nil GetByType")
	}
	if len(reg.List()) != 0 {
		t.Error("expected empty slice from nil List")
	}
	if reg.Has("name") {
		t.Error("expected false for nil Has")
	}
	if reg.Count() != 0 {
		t.Error("expected 0 for nil Count")
	}
}
