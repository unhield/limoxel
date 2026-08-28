package compatibility_test

import (
	"strings"
	"testing"

	"github.com/unhield/limoxel/sdk/compatibility"
)

func TestCompatibility_CurrentVersion(t *testing.T) {
	sv := compatibility.CurrentVersion()
	str := compatibility.CurrentVersionString()
	if sv.String() != str {
		t.Errorf("expected CurrentVersion().String() %q to equal CurrentVersionString() %q", sv.String(), str)
	}
	if str == "" {
		t.Error("expected non-empty version string")
	}
}

func TestCompatibility_DeprecationTracker(t *testing.T) {
	tracker := compatibility.NewDeprecationTracker()

	rec := compatibility.DeprecationRecord{
		APIName:           "Client.LegacySearch",
		Since:             "1.3.0",
		PlannedRemoval:    "2.0.0",
		Replacement:       "Client.Search().SearchSymbols",
		Reason:            "Replaced by unified SearchContract",
		MigrationGuidance: "Update call site to client.Search().SearchSymbols(ctx, query, opts)",
	}

	tracker.Register(rec)

	found, ok := tracker.Lookup("Client.LegacySearch")
	if !ok {
		t.Fatal("expected to find registered deprecation record")
	}
	if found.Replacement != "Client.Search().SearchSymbols" {
		t.Errorf("unexpected replacement: %q", found.Replacement)
	}

	notice := compatibility.FormatDeprecationNotice(found)
	if !strings.Contains(notice, "[DEPRECATION]") || !strings.Contains(notice, "Client.LegacySearch") {
		t.Errorf("unexpected formatted notice: %s", notice)
	}

	all := tracker.All()
	if len(all) != 1 {
		t.Errorf("expected 1 record in All(), got %d", len(all))
	}
}

func TestCompatibility_UpgradeValidator(t *testing.T) {
	val := compatibility.NewUpgradeValidator()

	v130 := compatibility.SemVer{Major: 1, Minor: 3, Patch: 0}
	v140 := compatibility.SemVer{Major: 1, Minor: 4, Patch: 0}
	v200 := compatibility.SemVer{Major: 2, Minor: 0, Patch: 0}

	// Minor upgrade with additions
	minorChanges := []compatibility.APIChange{
		{
			APIName:     "SymbolContract.SymbolOwnership",
			Kind:        compatibility.ChangeAddition,
			Description: "Added code ownership metadata query",
		},
	}

	decisionMinor := val.ValidateUpgrade(v130, v140, minorChanges)
	if !decisionMinor.IsCompatible {
		t.Errorf("expected minor addition to be compatible with minor release: %+v", decisionMinor)
	}

	// Breaking change in minor release should fail
	breakingChanges := []compatibility.APIChange{
		{
			APIName:      "RepositoryManagementContract.Open",
			Kind:         compatibility.ChangeBreaking,
			Description:  "Altered signature parameters",
			OldSignature: "Open(ctx, path)",
			NewSignature: "Open(ctx, path, flags)",
		},
	}

	decisionInvalid := val.ValidateUpgrade(v130, v140, breakingChanges)
	if decisionInvalid.IsCompatible {
		t.Error("expected breaking change in minor release to be marked incompatible")
	}

	// Breaking change in major release should pass
	decisionMajor := val.ValidateUpgrade(v130, v200, breakingChanges)
	if !decisionMajor.IsCompatible {
		t.Errorf("expected breaking change to be valid in major release: %+v", decisionMajor)
	}

	// Migration guide generation
	guide := val.GenerateMigrationGuide(v130, v200, breakingChanges)
	md := guide.FormatMarkdown()
	if !strings.Contains(md, "Migration Guide: v1.3.0 -> v2.0.0") || !strings.Contains(md, "Breaking Changes") {
		t.Errorf("unexpected migration guide content: %s", md)
	}
}
