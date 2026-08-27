package version_test

import (
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

func TestParseSemVer(t *testing.T) {
	tests := []struct {
		input       string
		wantErr     bool
		wantMajor   int
		wantMinor   int
		wantPatch   int
		wantPre     string
		wantBuild   string
		wantCoreStr string
	}{
		{input: "1.3.0", wantMajor: 1, wantMinor: 3, wantPatch: 0, wantCoreStr: "1.3.0"},
		{input: "v1.3.0", wantMajor: 1, wantMinor: 3, wantPatch: 0, wantCoreStr: "1.3.0"},
		{input: "0.1.0-alpha.1", wantMajor: 0, wantMinor: 1, wantPatch: 0, wantPre: "alpha.1", wantCoreStr: "0.1.0"},
		{input: "2.0.0+build.123", wantMajor: 2, wantMinor: 0, wantPatch: 0, wantBuild: "build.123", wantCoreStr: "2.0.0"},
		{input: "2.1.3-beta.2+20260827", wantMajor: 2, wantMinor: 1, wantPatch: 3, wantPre: "beta.2", wantBuild: "20260827", wantCoreStr: "2.1.3"},
		{input: "", wantErr: true},
		{input: "invalid", wantErr: true},
		{input: "1.2", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			sv, err := version.ParseSemVer(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseSemVer(%q) expected error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSemVer(%q) unexpected error: %v", tc.input, err)
			}
			if sv.Major != tc.wantMajor || sv.Minor != tc.wantMinor || sv.Patch != tc.wantPatch {
				t.Errorf("got %d.%d.%d, want %d.%d.%d", sv.Major, sv.Minor, sv.Patch, tc.wantMajor, tc.wantMinor, tc.wantPatch)
			}
			if sv.PreRelease != tc.wantPre {
				t.Errorf("got prerelease %q, want %q", sv.PreRelease, tc.wantPre)
			}
			if sv.BuildMetadata != tc.wantBuild {
				t.Errorf("got build %q, want %q", sv.BuildMetadata, tc.wantBuild)
			}
			if sv.CoreString() != tc.wantCoreStr {
				t.Errorf("got CoreString %q, want %q", sv.CoreString(), tc.wantCoreStr)
			}
		})
	}
}

func TestNewSemVer(t *testing.T) {
	sv, err := version.NewSemVer(1, 4, 2, "rc.1", "sha.abc")
	if err != nil {
		t.Fatalf("NewSemVer failed: %v", err)
	}
	if sv.String() != "1.4.2-rc.1+sha.abc" {
		t.Errorf("unexpected string: %s", sv.String())
	}

	_, err = version.NewSemVer(-1, 0, 0, "", "")
	if err == nil {
		t.Error("expected error for negative major version")
	}
}

func TestCurrentVersion(t *testing.T) {
	current := version.Current()
	if current.Major != 1 || current.Minor != 3 || current.Patch != 0 {
		t.Errorf("unexpected Current() version: %v", current)
	}
	if current.String() != "1.3.0" {
		t.Errorf("unexpected current string: %s", current.String())
	}
}

func TestSemVerCompare(t *testing.T) {
	v1, _ := version.ParseSemVer("1.0.0")
	v1_0_1, _ := version.ParseSemVer("1.0.1")
	v1_1_0, _ := version.ParseSemVer("1.1.0")
	v2_0_0, _ := version.ParseSemVer("2.0.0")
	v1_0_0_alpha, _ := version.ParseSemVer("1.0.0-alpha")

	if v1.Compare(v1) != 0 {
		t.Error("expected v1 == v1")
	}
	if v1.Compare(v1_0_1) >= 0 {
		t.Error("expected v1 < v1.0.1")
	}
	if v1_0_1.Compare(v1) <= 0 {
		t.Error("expected v1.0.1 > v1")
	}
	if v1_1_0.Compare(v1_0_1) <= 0 {
		t.Error("expected v1.1.0 > v1.0.1")
	}
	if v2_0_0.Compare(v1_1_0) <= 0 {
		t.Error("expected v2.0.0 > v1.1.0")
	}
	if v1_0_0_alpha.Compare(v1) >= 0 {
		t.Error("expected 1.0.0-alpha < 1.0.0")
	}
}

func TestSemVerCompatibility(t *testing.T) {
	v1_2_0, _ := version.ParseSemVer("1.2.0")
	v1_3_0, _ := version.ParseSemVer("1.3.0")
	v1_1_0, _ := version.ParseSemVer("1.1.0")
	v2_0_0, _ := version.ParseSemVer("2.0.0")

	// v1.3.0 is compatible with requirement v1.2.0 (same major, newer minor)
	if !v1_2_0.IsCompatibleWith(v1_3_0) {
		t.Error("expected 1.3.0 to be backward compatible with 1.2.0")
	}

	// v1.1.0 is NOT compatible with requirement v1.2.0 (missing features introduced in 1.2.0)
	if v1_2_0.IsCompatibleWith(v1_1_0) {
		t.Error("expected 1.1.0 to NOT satisfy requirement 1.2.0")
	}

	// v2.0.0 is NOT compatible with requirement v1.2.0 (breaking major change)
	if v1_2_0.IsCompatibleWith(v2_0_0) {
		t.Error("expected 2.0.0 to NOT be compatible with 1.2.0")
	}
}

func TestDiffAndClassifyRelease(t *testing.T) {
	v1_0_0, _ := version.ParseSemVer("1.0.0")
	v1_0_1, _ := version.ParseSemVer("1.0.1")
	v1_1_0, _ := version.ParseSemVer("1.1.0")
	v2_0_0, _ := version.ParseSemVer("2.0.0")

	if v1_0_0.Diff(v1_0_1) != version.DiffPatch {
		t.Errorf("got %v, want DiffPatch", v1_0_0.Diff(v1_0_1))
	}
	if v1_0_0.Diff(v1_1_0) != version.DiffMinor {
		t.Errorf("got %v, want DiffMinor", v1_0_0.Diff(v1_1_0))
	}
	if v1_0_0.Diff(v2_0_0) != version.DiffMajor {
		t.Errorf("got %v, want DiffMajor", v1_0_0.Diff(v2_0_0))
	}

	if version.ClassifyRelease(v1_0_0, v1_0_1) != version.ReleasePatch {
		t.Error("expected ReleasePatch")
	}
	if version.ClassifyRelease(v1_0_0, v1_1_0) != version.ReleaseMinor {
		t.Error("expected ReleaseMinor")
	}
	if version.ClassifyRelease(v1_0_0, v2_0_0) != version.ReleaseMajor {
		t.Error("expected ReleaseMajor")
	}
}

func TestBuildMetadataPrecedence(t *testing.T) {
	v1, _ := version.ParseSemVer("1.3.0+build.1")
	v2, _ := version.ParseSemVer("1.3.0+build.2")
	if v1.Compare(v2) != 0 {
		t.Errorf("SemVer 2.0.0 requires build metadata to be ignored in precedence comparison")
	}
}

func TestZeroVersionCompatibility(t *testing.T) {
	v0_1_0, _ := version.ParseSemVer("0.1.0")
	v0_1_1, _ := version.ParseSemVer("0.1.1")
	v0_2_0, _ := version.ParseSemVer("0.2.0")

	if !v0_1_0.IsCompatibleWith(v0_1_1) {
		t.Errorf("expected 0.1.1 to be compatible with 0.1.0")
	}
	if v0_1_0.IsCompatibleWith(v0_2_0) {
		t.Errorf("expected 0.2.0 to be incompatible with 0.1.0 in 0.y.z phase")
	}
}
