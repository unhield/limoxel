package templates_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/unhield/limoxel/sdk"
	"github.com/unhield/limoxel/sdk/templates/cli/cmd"
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

func TestTemplates_StarterPattern(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repoPath := createSampleRepo(t)
	client, err := sdk.OpenWorkspace(ctx, repoPath)
	if err != nil {
		t.Fatalf("starter initialization failed: %v", err)
	}
	defer client.Close()

	if client.Repository().State() != sdk.RepositoryState("READY") {
		t.Errorf("expected READY state, got %s", client.Repository().State())
	}
}

func TestTemplates_CLIPattern(t *testing.T) {
	repoPath := createSampleRepo(t)
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"custom-tool", "-action=inspect", "-repo=" + repoPath}
	if err := cmd.Execute(); err != nil {
		t.Fatalf("CLI template execution failed: %v", err)
	}
}

func TestTemplates_IntegrationPattern(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repoPath := createSampleRepo(t)
	client, err := sdk.OpenWorkspace(ctx, repoPath)
	if err != nil {
		t.Fatalf("integration template client failed: %v", err)
	}
	defer client.Close()

	health, err := client.Analysis().RepositoryHealth(ctx)
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	if health.OverallScore < 0 {
		t.Errorf("invalid score: %f", health.OverallScore)
	}
}

func TestTemplates_ServicePattern(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repoPath := createSampleRepo(t)
	client, err := sdk.OpenWorkspace(ctx, repoPath)
	if err != nil {
		t.Fatalf("failed to open workspace for service: %v", err)
	}
	defer client.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/info", func(w http.ResponseWriter, r *http.Request) {
		info, err := client.Repository().Info(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(info)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/info", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
}

func TestTemplates_EnterprisePattern(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repoPath1 := createSampleRepo(t)
	repoPath2 := createSampleRepo(t)
	paths := []string{repoPath1, repoPath2}

	var (
		mu      sync.Mutex
		results []string
		wg      sync.WaitGroup
	)

	for _, p := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			c, err := sdk.OpenWorkspace(ctx, path)
			if err != nil {
				return
			}
			defer c.Close()
			stats, err := c.Repository().Statistics(ctx)
			if err == nil && stats != nil {
				mu.Lock()
				results = append(results, path)
				mu.Unlock()
			}
		}(p)
	}

	wg.Wait()
	if len(results) != 2 {
		t.Errorf("expected 2 audited repos, got %d", len(results))
	}
}
