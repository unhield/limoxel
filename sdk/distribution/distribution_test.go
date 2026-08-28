package distribution_test

import (
	"os"
	"path/filepath"
	"testing"

	canonversion "github.com/unhield/limoxel/internal/version"
	"github.com/unhield/limoxel/sdk/distribution"
)

func createSamplePackage(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, f := range distribution.RequiredPublicFiles {
		fullPath := filepath.Join(dir, f)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("failed to create dir for %s: %v", f, err)
		}
		if err := os.WriteFile(fullPath, []byte("package content for "+f), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", f, err)
		}
	}
	return dir
}

func TestDistribution_IntegrityAndTamperDetection(t *testing.T) {
	tempDir := t.TempDir()
	file1 := "file1.txt"
	file2 := "pkg/file2.txt"

	if err := os.MkdirAll(filepath.Join(tempDir, "pkg"), 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, file1), []byte("Hello Limoxel"), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, file2), []byte("Package Content"), 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}

	// Generate checksums
	manifest, entries, err := distribution.GenerateChecksumManifest(tempDir, []string{file1, file2})
	if err != nil {
		t.Fatalf("GenerateChecksumManifest failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Verify valid manifest
	if err := distribution.VerifyChecksumManifest(tempDir, manifest); err != nil {
		t.Fatalf("VerifyChecksumManifest failed on untouched files: %v", err)
	}

	// Tamper with file1
	if err := os.WriteFile(filepath.Join(tempDir, file1), []byte("Tampered Content"), 0644); err != nil {
		t.Fatalf("failed to tamper file1: %v", err)
	}

	// Verify tampering detection
	if err := distribution.VerifyChecksumManifest(tempDir, manifest); err == nil {
		t.Fatal("expected checksum mismatch error after tampering, got nil")
	}
}

func TestDistribution_PackageValidation(t *testing.T) {
	pkgDir := createSamplePackage(t)

	res, err := distribution.ValidatePackage(pkgDir)
	if err != nil {
		t.Fatalf("ValidatePackage failed: %v", err)
	}
	if !res.IsValid {
		t.Errorf("expected package to be valid")
	}
	if res.Version != canonversion.Version {
		t.Errorf("expected version %s, got %s", canonversion.Version, res.Version)
	}

	// Test missing file failure
	os.Remove(filepath.Join(pkgDir, "LICENSE"))
	resInvalid, err := distribution.ValidatePackage(pkgDir)
	if err == nil {
		t.Fatal("expected validation error for missing LICENSE, got nil")
	}
	if resInvalid.IsValid {
		t.Error("expected IsValid to be false when LICENSE is missing")
	}
}

func TestDistribution_PipelineExecution(t *testing.T) {
	pkgDir := createSamplePackage(t)
	outDir := filepath.Join(t.TempDir(), "dist_output")

	result, err := distribution.RunPipeline(distribution.PipelineOptions{
		RepoRoot:  pkgDir,
		OutputDir: outDir,
	})
	if err != nil {
		t.Fatalf("RunPipeline failed: %v", err)
	}

	if !result.Success {
		t.Fatal("expected pipeline to succeed")
	}

	if _, err := os.Stat(filepath.Join(outDir, "SHA256SUMS")); err != nil {
		t.Errorf("expected SHA256SUMS file to be created: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "release-manifest.json")); err != nil {
		t.Errorf("expected release-manifest.json file to be created: %v", err)
	}
}
