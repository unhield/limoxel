package model_test

import (
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
)

func TestParseIdentity(t *testing.T) {
	tests := []struct {
		input     string
		wantValid bool
		wantNS    string
		wantName  string
	}{
		{"com.example.linter", true, "com.example", "linter"},
		{"org.limoxel.analysis-core", true, "org.limoxel", "analysis-core"},
		{"simple", true, "", "simple"},
		{"my-plugin", true, "", "my-plugin"},
		{"a.b.c.d", true, "a.b.c", "d"},
		{"", false, "", ""},
		{"   ", false, "", ""},
		{"ab", false, "", ""}, // too short (< 3 chars)
		{"UPPERCASE.NOT.ALLOWED", false, "", ""},
		{"spaces not allowed", false, "", ""},
		{"has/slashes", false, "", ""},
		{"has\\backslashes", false, "", ""},
		{"..consecutive.dots", false, "", ""},
		{"trailing.dot.", false, "", ""},
		{".leading.dot", false, "", ""},
		{"-leading-hyphen", false, "", ""},
		{"trailing-hyphen-", false, "", ""},
	}

	for _, tc := range tests {
		id, err := model.ParseIdentity(tc.input)
		if tc.wantValid && err != nil {
			t.Errorf("ParseIdentity(%q) expected valid, got error: %v", tc.input, err)
		} else if !tc.wantValid && err == nil {
			t.Errorf("ParseIdentity(%q) expected error, got valid: %s", tc.input, id)
		}

		if tc.wantValid {
			if id.Namespace() != tc.wantNS {
				t.Errorf("Namespace() for %q got %q, want %q", tc.input, id.Namespace(), tc.wantNS)
			}
			if id.Name() != tc.wantName {
				t.Errorf("Name() for %q got %q, want %q", tc.input, id.Name(), tc.wantName)
			}
		}
	}
}

func TestIdentityOrdering(t *testing.T) {
	idA := model.MustParseIdentity("a.plugin")
	idB := model.MustParseIdentity("b.plugin")

	if idA.Compare(idB) >= 0 {
		t.Errorf("expected idA < idB, got compare %d", idA.Compare(idB))
	}
	if !idA.Less(idB) {
		t.Errorf("expected idA.Less(idB) to be true")
	}
	if idB.Less(idA) {
		t.Errorf("expected idB.Less(idA) to be false")
	}
	if idA.Compare(idA) != 0 {
		t.Errorf("expected idA.Compare(idA) to be 0")
	}
}
