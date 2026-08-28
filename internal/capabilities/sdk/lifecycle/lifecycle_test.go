package lifecycle_test

import (
	"strings"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

func TestAPIDescriptorValidation(t *testing.T) {
	v1, _ := version.ParseSemVer("1.0.0")
	v2, _ := version.ParseSemVer("2.0.0")

	t.Run("ValidSupportedAPI", func(t *testing.T) {
		desc := lifecycle.APIDescriptor{
			Name:          "RepositoryService.Open",
			Capability:    lifecycle.CapabilityRepository,
			Since:         v1,
			Lifecycle:     lifecycle.StateSupported,
			Documentation: "Opens an existing repository workspace.",
		}
		if err := desc.Validate(); err != nil {
			t.Fatalf("expected valid descriptor, got: %v", err)
		}
	})

	t.Run("ValidDeprecatedAPI", func(t *testing.T) {
		desc := lifecycle.APIDescriptor{
			Name:       "RepositoryService.OpenLegacy",
			Capability: lifecycle.CapabilityRepository,
			Since:      v1,
			Lifecycle:  lifecycle.StateDeprecated,
			Deprecation: &lifecycle.DeprecationInfo{
				Since:             v1,
				PlannedRemoval:    v2,
				Replacement:       "RepositoryService.Open",
				MigrationGuidance: "Replace OpenLegacy with Open(path, opts).",
				Reason:            "Replaced with unified options pattern",
			},
			Documentation: "Deprecated legacy opener.",
		}
		if err := desc.Validate(); err != nil {
			t.Fatalf("expected valid deprecated descriptor, got: %v", err)
		}
		notice := desc.Deprecation.String()
		if !strings.Contains(notice, "Deprecated since v1.0.0") || !strings.Contains(notice, "RepositoryService.Open") {
			t.Errorf("unexpected deprecation notice: %s", notice)
		}
	})

	t.Run("InvalidDeprecatedAPI_MissingInfo", func(t *testing.T) {
		desc := lifecycle.APIDescriptor{
			Name:       "RepositoryService.OpenOld",
			Capability: lifecycle.CapabilityRepository,
			Since:      v1,
			Lifecycle:  lifecycle.StateDeprecated,
		}
		if err := desc.Validate(); err == nil {
			t.Fatalf("expected validation error for deprecated API without DeprecationInfo")
		}
	})

	t.Run("InvalidEmptyName", func(t *testing.T) {
		desc := lifecycle.APIDescriptor{
			Capability: lifecycle.CapabilityRepository,
			Lifecycle:  lifecycle.StateSupported,
		}
		if err := desc.Validate(); err == nil {
			t.Fatalf("expected validation error for empty name")
		}
	})
}

func TestRegistryAndTransitions(t *testing.T) {
	reg := lifecycle.NewRegistry()
	v1, _ := version.ParseSemVer("1.0.0")

	desc := lifecycle.APIDescriptor{
		Name:       "SymbolService.Lookup",
		Capability: lifecycle.CapabilitySymbol,
		Since:      v1,
		Lifecycle:  lifecycle.StateSupported,
	}

	if err := reg.Register(desc); err != nil {
		t.Fatalf("failed to register API: %v", err)
	}

	lookup, ok := reg.Lookup("SymbolService.Lookup")
	if !ok || lookup.Name != "SymbolService.Lookup" {
		t.Errorf("lookup failed or returned wrong descriptor: %v", lookup)
	}

	if len(reg.All()) != 1 {
		t.Errorf("expected 1 registered API, got %d", len(reg.All()))
	}
}

func TestValidateTransition(t *testing.T) {
	// Legal transitions
	if err := lifecycle.ValidateTransition(lifecycle.StateIntroduced, lifecycle.StateSupported); err != nil {
		t.Errorf("expected Introduced -> Supported to be valid: %v", err)
	}
	if err := lifecycle.ValidateTransition(lifecycle.StateSupported, lifecycle.StateDeprecated); err != nil {
		t.Errorf("expected Supported -> Deprecated to be valid: %v", err)
	}
	if err := lifecycle.ValidateTransition(lifecycle.StateDeprecated, lifecycle.StateRemoved); err != nil {
		t.Errorf("expected Deprecated -> Removed to be valid: %v", err)
	}

	// Illegal transitions
	if err := lifecycle.ValidateTransition(lifecycle.StateRemoved, lifecycle.StateSupported); err == nil {
		t.Errorf("expected Removed -> Supported to fail")
	}

	// Same state transition
	if err := lifecycle.ValidateTransition(lifecycle.StateSupported, lifecycle.StateSupported); err != nil {
		t.Errorf("expected same state transition to be valid: %v", err)
	}
}

func TestLifecycleStateProperties(t *testing.T) {
	if !lifecycle.StateIntroduced.IsActive() {
		t.Errorf("expected Introduced to be active")
	}
	if !lifecycle.StateSupported.IsActive() {
		t.Errorf("expected Supported to be active")
	}
	if !lifecycle.StateDeprecated.IsActive() {
		t.Errorf("expected Deprecated to be active")
	}
	if lifecycle.StateRemoved.IsActive() {
		t.Errorf("expected Removed to NOT be active")
	}

	var nilDep *lifecycle.DeprecationInfo
	if nilDep.String() != "" {
		t.Errorf("expected empty string for nil DeprecationInfo")
	}
}
