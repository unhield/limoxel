package validation_test

import (
	"strings"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/sdk/compatibility"
	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/sdk/validation"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

func TestValidationSuite(t *testing.T) {
	val := validation.NewValidator()
	reg := lifecycle.NewRegistry()
	sv := version.Current()

	_ = reg.Register(lifecycle.APIDescriptor{
		Name:          "RepositoryService.Open",
		Capability:    lifecycle.CapabilityRepository,
		Since:         sv,
		Lifecycle:     lifecycle.StateSupported,
		Documentation: "Opens an existing workspace.",
	})

	report := val.ValidateAll(reg, sv, nil, version.ReleaseMinor)
	if !report.IsValid {
		t.Fatalf("expected validation report to be valid, got: %s", report.Summary)
	}
	if report.PassedChecks != report.TotalChecks {
		t.Errorf("expected all checks to pass (%d/%d)", report.PassedChecks, report.TotalChecks)
	}

	md := report.FormatReportMarkdown()
	if !strings.Contains(md, "# SDK Validation Report") || !strings.Contains(md, "PASSED") {
		t.Errorf("unexpected report markdown:\n%s", md)
	}
}

func TestValidationFailureOnIncompatibleChange(t *testing.T) {
	val := validation.NewValidator()
	reg := lifecycle.NewRegistry()
	sv := version.Current()

	changes := []compatibility.APIChange{
		{
			APIName:     "SymbolService.Lookup",
			Kind:        compatibility.ChangeBreaking,
			Description: "Signature changed incompatibly",
		},
	}

	report := val.ValidateAll(reg, sv, changes, version.ReleaseMinor)
	if report.IsValid {
		t.Fatalf("expected validation to fail for breaking change in MINOR release")
	}
	if report.FailedChecks == 0 {
		t.Errorf("expected failed checks > 0")
	}
}
