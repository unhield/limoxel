package sdk_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/unhield/limoxel/sdk"
)

func createSampleRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "test_repo")
	if err := os.MkdirAll(filepath.Join(repoPath, "pkg", "calc"), 0755); err != nil {
		t.Fatalf("failed to create sample repo dirs: %v", err)
	}

	mainFile := `package main

import "test_repo/pkg/calc"

func main() {
	calc.Add(10, 20)
}
`
	calcFile := `package calc

// Add sums two integers.
func Add(a, b int) int {
	return a + b
}
`
	if err := os.WriteFile(filepath.Join(repoPath, "main.go"), []byte(mainFile), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "pkg", "calc", "calc.go"), []byte(calcFile), 0644); err != nil {
		t.Fatalf("failed to write calc.go: %v", err)
	}
	return repoPath
}

func TestClient_Initialization(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repoPath := createSampleRepo(t)

	client, err := sdk.New(sdk.WithWorkspace(repoPath))
	if err != nil {
		t.Fatalf("sdk.New failed: %v", err)
	}
	defer client.Close()

	if client.Workspace() != repoPath {
		t.Errorf("expected workspace %q, got %q", repoPath, client.Workspace())
	}

	if client.Version().String() != sdk.VersionString() {
		t.Errorf("expected client.Version %q to equal sdk.VersionString %q", client.Version().String(), sdk.VersionString())
	}
	if sdk.VersionString() == "" {
		t.Error("expected non-empty VersionString")
	}

	// Verify all capability accessors are non-nil
	if client.Repository() == nil {
		t.Error("Repository() is nil")
	}
	if client.Files() == nil {
		t.Error("Files() is nil")
	}
	if client.Packages() == nil {
		t.Error("Packages() is nil")
	}
	if client.Symbols() == nil {
		t.Error("Symbols() is nil")
	}
	if client.Search() == nil {
		t.Error("Search() is nil")
	}
	if client.Graph() == nil {
		t.Error("Graph() is nil")
	}
	if client.Analysis() == nil {
		t.Error("Analysis() is nil")
	}
	if client.Navigation() == nil {
		t.Error("Navigation() is nil")
	}
	if client.Reasoning() == nil {
		t.Error("Reasoning() is nil")
	}
	if client.Events() == nil {
		t.Error("Events() is nil")
	}
	if client.Intelligence() == nil {
		t.Error("Intelligence() is nil")
	}
	if client.Registry() == nil {
		t.Error("Registry() is nil")
	}
	if client.Validator() == nil {
		t.Error("Validator() is nil")
	}

	// Test repository opening
	repoInfo, err := client.Repository().Open(ctx, repoPath)
	if err != nil {
		t.Fatalf("Repository().Open failed: %v", err)
	}
	if repoInfo == nil {
		t.Fatal("expected non-nil repoInfo")
	}

	state := client.Repository().State()
	if state != sdk.RepositoryState("READY") {
		t.Errorf("expected repository state READY, got %q", state)
	}
}

func TestOpenWorkspace(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repoPath := createSampleRepo(t)

	client, err := sdk.OpenWorkspace(ctx, repoPath)
	if err != nil {
		t.Fatalf("OpenWorkspace failed: %v", err)
	}
	defer client.Close()

	info, err := client.Repository().Info(ctx)
	if err != nil {
		t.Fatalf("Repository().Info failed: %v", err)
	}
	if info == nil || info.RootPath == "" {
		t.Error("expected non-empty repository root path in Info")
	}
}
