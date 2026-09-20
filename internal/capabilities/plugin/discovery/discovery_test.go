package discovery_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/plugin/discovery"
)

const manifestOne = `{
  "schema_version": "1.0.0",
  "id": "org.limoxel.alpha",
  "name": "Alpha Plugin",
  "version": "1.0.0"
}`

const manifestTwo = `{
  "schema_version": "1.0.0",
  "id": "org.limoxel.beta",
  "name": "Beta Plugin",
  "version": "2.0.0"
}`

func TestDiscoverer_Discover(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Direct plugin dir
	dirA := filepath.Join(tempDir, "plugin_a")
	if err := os.MkdirAll(dirA, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirA, "plugin.json"), []byte(manifestOne), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. Subdirectory plugin with limoxel-plugin.json
	dirB := filepath.Join(tempDir, "plugin_b")
	if err := os.MkdirAll(dirB, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirB, "limoxel-plugin.json"), []byte(manifestTwo), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Ignored directory without manifest
	dirEmpty := filepath.Join(tempDir, "ignored_dir")
	if err := os.MkdirAll(dirEmpty, 0755); err != nil {
		t.Fatal(err)
	}

	disc := discovery.NewDiscoverer()
	candidates, err := disc.Discover(context.Background(), tempDir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(candidates) != 2 {
		t.Fatalf("expected 2 discovered candidates, got %d", len(candidates))
	}

	// Confirm deterministic order (alpha before beta)
	if string(candidates[0].Manifest.ID) != "org.limoxel.alpha" {
		t.Errorf("expected first candidate org.limoxel.alpha, got %s", candidates[0].Manifest.ID)
	}
	if string(candidates[1].Manifest.ID) != "org.limoxel.beta" {
		t.Errorf("expected second candidate org.limoxel.beta, got %s", candidates[1].Manifest.ID)
	}
}

func TestDiscoverer_DuplicateIdentityHandling(t *testing.T) {
	tempDir := t.TempDir()

	dir1 := filepath.Join(tempDir, "root1", "p1")
	dir2 := filepath.Join(tempDir, "root2", "p2")
	_ = os.MkdirAll(dir1, 0755)
	_ = os.MkdirAll(dir2, 0755)

	// Both have identity "org.limoxel.alpha"
	_ = os.WriteFile(filepath.Join(dir1, "plugin.json"), []byte(manifestOne), 0644)
	_ = os.WriteFile(filepath.Join(dir2, "plugin.json"), []byte(manifestOne), 0644)

	disc := discovery.NewDiscoverer()
	candidates, err := disc.Discover(context.Background(), filepath.Join(tempDir, "root1"), filepath.Join(tempDir, "root2"))
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Should deduplicate and record diagnostic
	if len(candidates) != 1 {
		t.Errorf("expected 1 deduplicated candidate, got %d", len(candidates))
	}
	if len(disc.Diagnostics()) == 0 {
		t.Error("expected diagnostic record for duplicate identity collision, got none")
	}
}
