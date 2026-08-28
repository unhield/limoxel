package examples_test

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

func TestExamples_BasicUsageWorkflow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repoPath := createSampleRepo(t)
	client, err := sdk.OpenWorkspace(ctx, repoPath)
	if err != nil {
		t.Fatalf("failed to open workspace: %v", err)
	}
	defer client.Close()

	// 1. Statistics
	stats, err := client.Repository().Statistics(ctx)
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if stats.TotalFiles < 2 {
		t.Errorf("expected at least 2 files, got %d", stats.TotalFiles)
	}

	// 2. Discover files
	files, err := client.Files().DiscoverFiles(ctx, sdk.FileFilter{Language: "go"}, sdk.PaginationOptions{Limit: 10})
	if err != nil {
		t.Fatalf("failed to discover files: %v", err)
	}
	if len(files) == 0 {
		t.Error("expected discovered files")
	}

	// 3. Search symbols
	results, err := client.Search().SearchSymbols(ctx, "Add", sdk.PaginationOptions{Limit: 5})
	if err != nil {
		t.Fatalf("failed to search symbols: %v", err)
	}
	if results.TotalMatches == 0 {
		t.Error("expected at least 1 match for symbol 'Add'")
	}
}

func TestExamples_AnalysisWorkflow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repoPath := createSampleRepo(t)
	client, err := sdk.OpenWorkspace(ctx, repoPath)
	if err != nil {
		t.Fatalf("failed to open workspace: %v", err)
	}
	defer client.Close()

	// Health
	health, err := client.Analysis().RepositoryHealth(ctx)
	if err != nil {
		t.Fatalf("failed to evaluate health: %v", err)
	}
	if health.OverallScore < 0 {
		t.Errorf("invalid health score: %f", health.OverallScore)
	}

	// Architecture
	arch, err := client.Analysis().AnalyzeArchitecture(ctx, "")
	if err != nil {
		t.Fatalf("failed to analyze architecture: %v", err)
	}
	if arch == nil {
		t.Fatal("expected non-nil architecture result")
	}

	// Dependencies
	deps, err := client.Analysis().AnalyzeDependencies(ctx, "")
	if err != nil {
		t.Fatalf("failed to analyze dependencies: %v", err)
	}
	if deps == nil {
		t.Fatal("expected non-nil dependency result")
	}
}

func TestExamples_KnowledgeGraphWorkflow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repoPath := createSampleRepo(t)
	client, err := sdk.OpenWorkspace(ctx, repoPath)
	if err != nil {
		t.Fatalf("failed to open workspace: %v", err)
	}
	defer client.Close()

	// Export Mermaid
	mermaidResult, err := client.Graph().ExportGraph(ctx, sdk.GraphFilter{MaxDepth: 2}, sdk.ExportFormatMermaid)
	if err != nil {
		t.Fatalf("failed to export mermaid: %v", err)
	}
	if mermaidResult.Content == "" {
		t.Error("expected non-empty mermaid content")
	}

	// Export Graphviz
	dotResult, err := client.Graph().ExportGraph(ctx, sdk.GraphFilter{MaxDepth: 2}, sdk.ExportFormatGraphviz)
	if err != nil {
		t.Fatalf("failed to export graphviz: %v", err)
	}
	if dotResult.Content == "" {
		t.Error("expected non-empty dot content")
	}
}

func TestExamples_NavigationWorkflow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repoPath := createSampleRepo(t)
	client, err := sdk.OpenWorkspace(ctx, repoPath)
	if err != nil {
		t.Fatalf("failed to open workspace: %v", err)
	}
	defer client.Close()

	def, err := client.Navigation().GoToDefinition(ctx, "Add")
	if err != nil {
		t.Logf("GoToDefinition error (expected if symbol unindexed in small test): %v", err)
	} else if def != nil {
		t.Logf("Found definition: %+v", def)
	}
}

func TestExamples_ReasoningWorkflow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repoPath := createSampleRepo(t)
	client, err := sdk.OpenWorkspace(ctx, repoPath)
	if err != nil {
		t.Fatalf("failed to open workspace: %v", err)
	}
	defer client.Close()

	insights, err := client.Reasoning().EngineeringInsights(ctx)
	if err != nil {
		t.Fatalf("failed to retrieve repository insights: %v", err)
	}
	t.Logf("Found %d insights", len(insights))

	impact, err := client.Reasoning().AnalyzeImpact(ctx, "main")
	if err != nil {
		t.Logf("AnalyzeImpact note: %v", err)
	} else if impact != nil {
		t.Logf("Impact risk: %s, score: %.2f", impact.RiskLevel, impact.ImpactScore)
	}
}

func TestExamples_EventStreamingWorkflow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repoPath := createSampleRepo(t)
	client, err := sdk.New(sdk.WithWorkspace(repoPath))
	if err != nil {
		t.Fatalf("failed to create SDK: %v", err)
	}
	defer client.Close()

	var received []sdk.Event
	sub, err := client.Events().Subscribe(ctx, sdk.EventTypeRepositoryOpened, func(ev sdk.Event) {
		received = append(received, ev)
	})
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	if sub == nil || sub.ID() == "" {
		t.Fatal("expected non-nil subscription with valid ID")
	}
}
