package extension_test

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/unhield/limoxel/internal/extension"
)

func TestDescriptorConstructorAndGetters(t *testing.T) {
	meta := map[string]string{
		"category": "linter",
		"  ":       "invalid",
		"env":      "prod",
	}

	t.Run("valid descriptor creation", func(t *testing.T) {
		desc, err := extension.NewDescriptor("Ext-01", "Extension One", "2.0.0", "Limoxel Team", "Sample Extension", meta)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if desc.ID() != "ext-01" {
			t.Errorf("got ID %q, want ext-01", desc.ID())
		}
		if desc.Name() != "Extension One" {
			t.Errorf("got Name %q, want Extension One", desc.Name())
		}
		if desc.Version() != "2.0.0" {
			t.Errorf("got Version %q, want 2.0.0", desc.Version())
		}
		if desc.Author() != "Limoxel Team" {
			t.Errorf("got Author %q, want Limoxel Team", desc.Author())
		}
		if desc.Description() != "Sample Extension" {
			t.Errorf("got Description %q", desc.Description())
		}
		if desc.Metadata()["category"] != "linter" || desc.Metadata()["env"] != "prod" {
			t.Errorf("got Metadata %v", desc.Metadata())
		}
		if desc.String() != "Extension<ext-01>(name=Extension One, v=2.0.0)" {
			t.Errorf("unexpected String(): %q", desc.String())
		}

		// Defensive copy check of metadata map
		meta["category"] = "mutated"
		if desc.Metadata()["category"] == "mutated" {
			t.Error("metadata map mutation leaked into Descriptor")
		}
	})

	t.Run("default version fallback", func(t *testing.T) {
		desc, err := extension.NewDescriptor("ext-02", "Ext Two", "  ", "", "", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if desc.Version() != "1.0.0" {
			t.Errorf("got Version %q, want 1.0.0", desc.Version())
		}
	})

	t.Run("invalid ID errors", func(t *testing.T) {
		_, err := extension.NewDescriptor("  ", "Ext", "1.0", "", "", nil)
		if !errors.Is(err, extension.ErrInvalidID) {
			t.Errorf("got %v, want ErrInvalidID", err)
		}
		_, err = extension.NewDescriptor("ext 01", "Ext", "1.0", "", "", nil)
		if err == nil || !errors.Is(err, extension.ErrInvalidID) {
			t.Errorf("got %v, want ErrInvalidID for spaces", err)
		}
	})

	t.Run("invalid Name error", func(t *testing.T) {
		_, err := extension.NewDescriptor("ext-01", "  ", "1.0", "", "", nil)
		if !errors.Is(err, extension.ErrInvalidName) {
			t.Errorf("got %v, want ErrInvalidName", err)
		}
	})

	t.Run("nil descriptor getters", func(t *testing.T) {
		var desc *extension.Descriptor
		if desc.ID() != "" || desc.Name() != "" || desc.Version() != "" || desc.Author() != "" || desc.Description() != "" || desc.Metadata() != nil {
			t.Error("expected zero values for nil Descriptor getters")
		}
		if desc.String() != "Extension<nil>" {
			t.Errorf("got %q, want Extension<nil>", desc.String())
		}
	})
}

func TestRegistryRegistrationAndLifecycle(t *testing.T) {
	reg := extension.NewRegistry()
	d1, _ := extension.NewDescriptor("ext-1", "Extension 1", "1.0.0", "Author", "Desc", nil)
	d2, _ := extension.NewDescriptor("ext-2", "Extension 2", "1.0.0", "Author", "Desc", nil)

	// Register
	if err := reg.Register(d1); err != nil {
		t.Fatalf("Register d1 failed: %v", err)
	}
	if err := reg.Register(d2); err != nil {
		t.Fatalf("Register d2 failed: %v", err)
	}

	if reg.Count() != 2 {
		t.Errorf("got count %d, want 2", reg.Count())
	}
	if !reg.Has("ext-1") || !reg.Has("ext-2") {
		t.Error("expected Has to return true")
	}

	// Duplicate registration error
	err := reg.Register(d1)
	if !errors.Is(err, extension.ErrDuplicateExtension) {
		t.Errorf("got %v, want ErrDuplicateExtension", err)
	}

	// Initial State is StateRegistered
	st, err := reg.State("ext-1")
	if err != nil || st != extension.StateRegistered {
		t.Errorf("got state %v, %v; want StateRegistered", st, err)
	}
	if reg.IsActive("ext-1") {
		t.Error("should not be active yet")
	}

	// Activate
	if err := reg.Activate("ext-1"); err != nil {
		t.Fatalf("Activate failed: %v", err)
	}
	if !reg.IsActive("ext-1") {
		t.Error("expected IsActive to be true")
	}
	// Double activate error
	if err := reg.Activate("ext-1"); !errors.Is(err, extension.ErrAlreadyActive) {
		t.Errorf("got %v, want ErrAlreadyActive", err)
	}

	// Deactivate
	if err := reg.Deactivate("ext-1"); err != nil {
		t.Fatalf("Deactivate failed: %v", err)
	}
	if reg.IsActive("ext-1") {
		t.Error("should be inactive")
	}
	// Double deactivate error
	if err := reg.Deactivate("ext-1"); !errors.Is(err, extension.ErrAlreadyInactive) {
		t.Errorf("got %v, want ErrAlreadyInactive", err)
	}

	// Re-activate
	if err := reg.Activate("ext-1"); err != nil {
		t.Errorf("Re-activate failed: %v", err)
	}

	// List ordering
	list := reg.List()
	if len(list) != 2 || list[0].ID() != "ext-1" || list[1].ID() != "ext-2" {
		t.Errorf("List() ordering mismatch: %v", list)
	}

	// Remove
	if err := reg.Remove("ext-1"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if reg.Has("ext-1") {
		t.Error("ext-1 should be removed")
	}
	if reg.Count() != 1 {
		t.Errorf("got count %d, want 1", reg.Count())
	}
}

func TestNilRegistrySafety(t *testing.T) {
	var reg *extension.Registry
	d, _ := extension.NewDescriptor("ext", "Ext", "1.0", "", "", nil)

	if err := reg.Register(d); !errors.Is(err, extension.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if err := reg.Activate("ext"); !errors.Is(err, extension.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if err := reg.Deactivate("ext"); !errors.Is(err, extension.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if _, err := reg.State("ext"); !errors.Is(err, extension.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if reg.IsActive("ext") {
		t.Error("expected false for nil IsActive")
	}
	if err := reg.Remove("ext"); !errors.Is(err, extension.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if _, err := reg.Get("ext"); !errors.Is(err, extension.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if reg.Has("ext") {
		t.Error("expected false for nil Has")
	}
	if reg.Count() != 0 {
		t.Error("expected 0 for nil Count")
	}
	if reg.List() != nil {
		t.Error("expected nil for nil List")
	}
}

func TestDiscovery(t *testing.T) {
	root1 := filepath.Join("/path", "ext-b")
	root2 := filepath.Join("/path", "ext-a")

	t.Run("no roots error", func(t *testing.T) {
		_, err := extension.NewDiscoverer()
		if !errors.Is(err, extension.ErrNoRootsProvided) {
			t.Errorf("got %v, want ErrNoRootsProvided", err)
		}
	})

	disc, err := extension.NewDiscoverer(root1, root2, root1)
	if err != nil {
		t.Fatalf("NewDiscoverer failed: %v", err)
	}

	// Verify deduplicated roots
	if len(disc.Roots()) != 2 {
		t.Errorf("got %d roots, want 2", len(disc.Roots()))
	}

	res, err := disc.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if res.Count() != 2 {
		t.Errorf("got count %d, want 2", res.Count())
	}

	// Verify deterministic lexicographical sorting by ID (ext-a before ext-b)
	descs := res.Descriptors()
	if len(descs) != 2 || descs[0].ID() != "ext-a" || descs[1].ID() != "ext-b" {
		t.Errorf("descriptors out of order: %v", descs)
	}

	// Nil discoverer receiver
	var nilDisc *extension.Discoverer
	if nilDisc.Roots() != nil {
		t.Error("expected nil Roots for nil Discoverer")
	}
	if _, err := nilDisc.Discover(); !errors.Is(err, extension.ErrNilDiscoverer) {
		t.Errorf("got %v, want ErrNilDiscoverer", err)
	}

	// Nil DiscoveryResult receiver
	var nilRes *extension.DiscoveryResult
	if nilRes.Roots() != nil || nilRes.Descriptors() != nil || nilRes.Count() != 0 {
		t.Error("expected zero values for nil DiscoveryResult getters")
	}
}

func TestScopeAndIsolation(t *testing.T) {
	scope1, err := extension.NewScope("Ext-1", "namespace-a")
	if err != nil {
		t.Fatalf("NewScope failed: %v", err)
	}
	if scope1.ID() != "ext-1" {
		t.Errorf("got ID %q, want ext-1", scope1.ID())
	}
	if scope1.Namespace() != "namespace-a" {
		t.Errorf("got Namespace %q, want namespace-a", scope1.Namespace())
	}

	// Default namespace fallback to ID
	scope2, _ := extension.NewScope("Ext-2", "  ")
	if scope2.Namespace() != "ext-2" {
		t.Errorf("got Namespace %q, want ext-2", scope2.Namespace())
	}

	validator := extension.NewIsolationValidator()
	d1, _ := extension.NewDescriptor("ext-1", "Ext 1", "1.0", "Author", "Desc", map[string]string{"namespace": "ns-a"})
	d2, _ := extension.NewDescriptor("ext-2", "Ext 2", "1.0", "Author", "Desc", map[string]string{"namespace": "ns-b"})
	dCollidingID, _ := extension.NewDescriptor("ext-1", "Ext Collide", "1.0", "Author", "Desc", map[string]string{"namespace": "ns-c"})
	dCollidingNS, _ := extension.NewDescriptor("ext-3", "Ext Collide NS", "1.0", "Author", "Desc", map[string]string{"namespace": "ns-a"})

	existing := []*extension.Descriptor{d1, d2}

	// Valid target
	dValidTarget, _ := extension.NewDescriptor("ext-3", "Ext 3", "1.0", "Author", "Desc", map[string]string{"namespace": "ns-c"})
	if err := validator.ValidateIsolation(dValidTarget, existing); err != nil {
		t.Errorf("valid isolation check failed: %v", err)
	}

	// ID collision error
	if err := validator.ValidateIsolation(dCollidingID, existing); !errors.Is(err, extension.ErrIsolationViolation) {
		t.Errorf("got %v, want ErrIsolationViolation for ID collision", err)
	}

	// Namespace collision error
	if err := validator.ValidateIsolation(dCollidingNS, existing); !errors.Is(err, extension.ErrIsolationViolation) {
		t.Errorf("got %v, want ErrIsolationViolation for namespace collision", err)
	}

	// Nil validator / descriptor safety
	var nilVal *extension.IsolationValidator
	if err := nilVal.ValidateIsolation(dValidTarget, existing); !errors.Is(err, extension.ErrNilIsolationValidator) {
		t.Errorf("got %v, want ErrNilIsolationValidator", err)
	}
	if err := validator.ValidateIsolation(nil, existing); !errors.Is(err, extension.ErrNilDescriptor) {
		t.Errorf("got %v, want ErrNilDescriptor", err)
	}
	var nilScope *extension.Scope
	if nilScope.ID() != "" || nilScope.Namespace() != "" {
		t.Error("expected empty string getters on nil Scope")
	}
}

func TestStateStringRepresentation(t *testing.T) {
	states := map[extension.State]string{
		extension.StateRegistered: "REGISTERED",
		extension.StateActive:     "ACTIVE",
		extension.StateInactive:   "INACTIVE",
		extension.State(99):       "UNKNOWN_STATE(99)",
	}

	for st, str := range states {
		if st.String() != str {
			t.Errorf("got %q, want %q", st.String(), str)
		}
	}
}

func TestConcurrentRegistryReads(t *testing.T) {
	reg := extension.NewRegistry()
	d, _ := extension.NewDescriptor("ext-1", "Ext 1", "1.0", "Author", "Desc", nil)
	_ = reg.Register(d)
	_ = reg.Activate("ext-1")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = reg.Get("ext-1")
				_, _ = reg.State("ext-1")
				_ = reg.IsActive("ext-1")
				_ = reg.Has("ext-1")
				_ = reg.Count()
				_ = reg.List()
			}
		}()
	}
	wg.Wait()
}
