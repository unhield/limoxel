package validation

import (
	"context"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/policy"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security/verification"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

func TestEnterpriseValidator_AllDimensions(t *testing.T) {
	trustStore := verification.NewTrustStore()
	validator := NewEnterpriseValidator(trustStore)
	ctx := context.Background()

	v1, _ := version.ParseSemVer("1.0.0")
	v2, _ := version.ParseSemVer("2.0.0")
	manifest := &model.Manifest{
		SchemaVersion: model.ManifestSchemaVersion,
		ID:            model.Identity("acme.tool"),
		Name:          "Acme Tool",
		Version:       v2,
		Publisher:     "org-acme",
	}

	doc := &policy.PolicyDocument{
		OrganizationID: "org-acme",
		AllowlistRules: []policy.AllowlistRule{
			{RuleID: "allow-all", PluginIDs: []string{"acme.*"}},
		},
	}

	evalCtx := policy.EvaluationContext{
		OrganizationID:     "org-acme",
		PluginID:           "acme.tool",
		Version:            v2,
		PublisherID:        "org-acme",
		DistributionSource: "private_registry",
		IsSigned:           true,
		MalwareScanPassed:  true,
	}

	// 1. Fully Valid Plugin
	report := validator.ValidateAll(
		ctx,
		manifest,
		"", // directory check skipped when empty
		true,
		doc,
		evalCtx,
		"linux", "amd64",
		&v1, // upgrade from v1 to v2
		0,   // crashes
	)

	if !report.Passed {
		t.Fatalf("expected validation to pass, issues: %+v", report.Issues)
	}
	if !report.CompliancePassed || !report.SecurityPassed || !report.CompatibilityPassed || !report.UpgradePassed || !report.StabilityPassed {
		t.Errorf("all 5 dimensions should be passed, got: %+v", report)
	}

	// 2. Test Downgrade Rejection
	v3, _ := version.ParseSemVer("3.0.0")
	reportDowngrade := validator.ValidateAll(
		ctx,
		manifest, // version is 2.0.0
		"",
		true,
		doc,
		evalCtx,
		"linux", "amd64",
		&v3, // current is 3.0.0, target is 2.0.0 -> Downgrade!
		0,
	)

	if reportDowngrade.Passed || reportDowngrade.UpgradePassed {
		t.Errorf("downgrade validation should fail, got: %+v", reportDowngrade)
	}

	// 3. Test Stability Crash Limit
	reportCrashes := validator.ValidateAll(
		ctx,
		manifest,
		"",
		true,
		doc,
		evalCtx,
		"linux", "amd64",
		&v1,
		5, // 5 crashes exceeds limit 3
	)

	if reportCrashes.Passed || reportCrashes.StabilityPassed {
		t.Errorf("excessive crashes should fail stability validation, got: %+v", reportCrashes)
	}
}

func TestEnterpriseValidator_VerifyArchive(t *testing.T) {
	trustStore := verification.NewTrustStore()
	validator := NewEnterpriseValidator(trustStore)
	ctx := context.Background()

	// 1. Empty data
	isSigned, clean, err := validator.VerifyArchive(ctx, nil)
	if err != nil || isSigned || !clean {
		t.Errorf("expected isSigned=false, clean=true for nil data, got isSigned=%v, clean=%v, err=%v", isSigned, clean, err)
	}

	// 2. Benign raw payload (not an archive)
	isSigned, clean, err = validator.VerifyArchive(ctx, []byte("normal log message"))
	if err != nil || isSigned || !clean {
		t.Errorf("expected isSigned=false, clean=true for benign text, got isSigned=%v, clean=%v, err=%v", isSigned, clean, err)
	}

	// 3. Raw payload with PE header executable signature
	peData := []byte{0x4D, 0x5A, 0x90, 0x00}
	isSigned, clean, _ = validator.VerifyArchive(ctx, peData)
	if isSigned || clean {
		t.Errorf("expected clean=false for PE executable header, got clean=%v", clean)
	}
}
