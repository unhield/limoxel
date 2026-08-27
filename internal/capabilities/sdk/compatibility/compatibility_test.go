package compatibility_test

import (
	"strings"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/sdk/compatibility"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

func TestCompatibilityEvaluator(t *testing.T) {
	eval := compatibility.NewEvaluator()

	t.Run("PatchCompliant", func(t *testing.T) {
		changes := []compatibility.APIChange{
			{APIName: "Repository.Open", Kind: compatibility.ChangeFix, Description: "Fix race condition during load"},
			{APIName: "Symbol.Lookup", Kind: compatibility.ChangeDocumentation, Description: "Fix typo in doc comment"},
		}

		decision := eval.Evaluate(changes, version.ReleasePatch)
		if !decision.IsCompatible {
			t.Fatalf("expected patch compliant, got violations: %v", decision.Violations)
		}
		if decision.RequiredSemVer != version.ReleasePatch {
			t.Errorf("got %v, want ReleasePatch", decision.RequiredSemVer)
		}
	})

	t.Run("MinorAdditionInPatchViolation", func(t *testing.T) {
		changes := []compatibility.APIChange{
			{APIName: "Repository.Open", Kind: compatibility.ChangeAddition, Description: "Add optional concurrency flag"},
		}

		decision := eval.Evaluate(changes, version.ReleasePatch)
		if decision.IsCompatible {
			t.Fatalf("expected violation when adding feature in PATCH release")
		}
		if decision.RequiredSemVer != version.ReleaseMinor {
			t.Errorf("got %v, want ReleaseMinor", decision.RequiredSemVer)
		}
	})

	t.Run("BreakingInMinorViolation", func(t *testing.T) {
		changes := []compatibility.APIChange{
			{
				APIName:           "Repository.Close",
				Kind:              compatibility.ChangeBreaking,
				Description:       "Remove legacy synchronous close",
				MigrationGuidance: "Use Close(ctx) instead.",
			},
		}

		decision := eval.Evaluate(changes, version.ReleaseMinor)
		if decision.IsCompatible {
			t.Fatalf("expected violation when introducing breaking change in MINOR release")
		}
		if decision.RequiredSemVer != version.ReleaseMajor {
			t.Errorf("got %v, want ReleaseMajor", decision.RequiredSemVer)
		}
	})

	t.Run("BreakingInMajorAllowed", func(t *testing.T) {
		changes := []compatibility.APIChange{
			{
				APIName:           "Repository.Close",
				Kind:              compatibility.ChangeBreaking,
				Description:       "Remove legacy synchronous close",
				MigrationGuidance: "Use Close(ctx) instead.",
			},
		}

		decision := eval.Evaluate(changes, version.ReleaseMajor)
		if !decision.IsCompatible {
			t.Fatalf("expected breaking change to be allowed in MAJOR release: %v", decision.Violations)
		}
	})
}

func TestMigrationGuide(t *testing.T) {
	from, _ := version.ParseSemVer("1.3.0")
	to, _ := version.ParseSemVer("2.0.0")

	changes := []compatibility.APIChange{
		{
			APIName:           "Repository.OpenLegacy",
			Kind:              compatibility.ChangeBreaking,
			Description:       "Removed deprecated method",
			OldSignature:      "OpenLegacy(path string)",
			NewSignature:      "Open(ctx context.Context, path string)",
			MigrationGuidance: "Update call site to pass context and use Open().",
		},
	}

	guide := compatibility.NewMigrationGuide(from, to, changes)
	md := guide.FormatMarkdown()

	if !strings.Contains(md, "# Migration Guide: v1.3.0 -> v2.0.0") ||
		!strings.Contains(md, "Repository.OpenLegacy") ||
		!strings.Contains(md, "Update call site to pass context") {
		t.Errorf("unexpected migration guide content:\n%s", md)
	}
}
