package model_test

import (
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

func TestEvaluateConstraint(t *testing.T) {
	tests := []struct {
		constraint string
		ver        string
		wantMatch  bool
	}{
		{"*", "1.2.3", true},
		{"", "1.2.3", true},
		{"latest", "2.0.0", true},
		{"1.2.3", "1.2.3", true},
		{"1.2.3", "1.2.4", false},
		{">= 1.0.0", "1.0.0", true},
		{">= 1.0.0", "1.5.0", true},
		{">= 1.0.0", "0.9.9", false},
		{"< 2.0.0", "1.9.9", true},
		{"< 2.0.0", "2.0.0", false},
		{">= 1.0.0, < 2.0.0", "1.4.0", true},
		{">= 1.0.0, < 2.0.0", "2.0.0", false},
		{">= 1.0.0, < 2.0.0", "0.9.0", false},
		{"^1.2.3", "1.2.3", true},
		{"^1.2.3", "1.9.0", true},
		{"^1.2.3", "2.0.0", false},
		{"^1.2.3", "1.2.2", false},
		{"~1.2.3", "1.2.3", true},
		{"~1.2.3", "1.2.9", true},
		{"~1.2.3", "1.3.0", false},
	}

	for _, tc := range tests {
		sv, err := version.ParseSemVer(tc.ver)
		if err != nil {
			t.Fatalf("invalid semver in test case %q: %v", tc.ver, err)
		}
		got := model.EvaluateConstraint(tc.constraint, sv)
		if got != tc.wantMatch {
			t.Errorf("EvaluateConstraint(%q, %q) = %v, want %v", tc.constraint, tc.ver, got, tc.wantMatch)
		}
	}
}

func TestNewDependency_Validation(t *testing.T) {
	id := model.MustParseIdentity("com.example.dep")

	// Valid
	if _, err := model.NewDependency(id, ">= 1.0.0", false); err != nil {
		t.Errorf("unexpected error on valid dependency: %v", err)
	}

	// Invalid constraint syntax
	if _, err := model.NewDependency(id, "not-a-valid-constraint", false); err == nil {
		t.Error("expected error for invalid constraint, got nil")
	}

	// Empty ID
	if _, err := model.NewDependency("", "1.0.0", false); err == nil {
		t.Error("expected error for empty identity, got nil")
	}
}
