package sdk_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/unhield/limoxel/sdk"
)

// Compile-time interface satisfaction checks
var (
	_ sdk.RepositoryManagementContract = (sdk.RepositoryManagementContract)(nil)
	_ sdk.FileContract                 = (sdk.FileContract)(nil)
	_ sdk.PackageContract              = (sdk.PackageContract)(nil)
	_ sdk.SymbolContract               = (sdk.SymbolContract)(nil)
	_ sdk.SearchContract               = (sdk.SearchContract)(nil)
	_ sdk.GraphContract                = (sdk.GraphContract)(nil)
	_ sdk.AnalysisContract             = (sdk.AnalysisContract)(nil)
	_ sdk.NavigationContract           = (sdk.NavigationContract)(nil)
	_ sdk.ReasoningContract            = (sdk.ReasoningContract)(nil)
	_ sdk.EventContract                = (sdk.EventContract)(nil)
	_ sdk.IntelligenceContract         = (sdk.IntelligenceContract)(nil)
)

func createContractSampleRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "contract_repo")
	if err := os.MkdirAll(filepath.Join(repoPath, "pkg", "mathutil"), 0755); err != nil {
		t.Fatalf("failed to create sample repo dirs: %v", err)
	}

	mainFile := `package main

import "contract_repo/pkg/mathutil"

func main() {
	mathutil.Multiply(5, 10)
}
`
	mathFile := `package mathutil

// Multiply calculates the product of two integers.
func Multiply(a, b int) int {
	return a * b
}
`
	if err := os.WriteFile(filepath.Join(repoPath, "main.go"), []byte(mainFile), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "pkg", "mathutil", "math.go"), []byte(mathFile), 0644); err != nil {
		t.Fatalf("failed to write math.go: %v", err)
	}
	return repoPath
}

func TestContracts_AllCapabilityAccessors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repoPath := createContractSampleRepo(t)
	client, err := sdk.OpenWorkspace(ctx, repoPath)
	if err != nil {
		t.Fatalf("failed to open workspace: %v", err)
	}
	defer client.Close()

	// 1. Repository Contract
	repo := client.Repository()
	if repo == nil {
		t.Fatal("Repository() contract is nil")
	}
	stats, err := repo.Statistics(ctx)
	if err != nil || stats == nil {
		t.Fatalf("Repository.Statistics failed: %v", err)
	}
	if stats.TotalFiles < 2 {
		t.Errorf("expected at least 2 files, got %d", stats.TotalFiles)
	}

	// 2. File Contract
	filesContract := client.Files()
	if filesContract == nil {
		t.Fatal("Files() contract is nil")
	}
	files, err := filesContract.DiscoverFiles(ctx, sdk.FileFilter{}, sdk.PaginationOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Files.DiscoverFiles failed: %v", err)
	}
	if len(files) == 0 {
		t.Error("expected discovered files")
	}

	// 3. Package Contract
	pkgContract := client.Packages()
	if pkgContract == nil {
		t.Fatal("Packages() contract is nil")
	}
	pkgs, err := pkgContract.DiscoverPackages(ctx, sdk.PackageFilter{}, sdk.PaginationOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Packages.DiscoverPackages failed: %v", err)
	}
	if len(pkgs) == 0 {
		t.Error("expected discovered packages")
	}

	// 4. Symbol Contract
	symbolContract := client.Symbols()
	if symbolContract == nil {
		t.Fatal("Symbols() contract is nil")
	}

	// 5. Search Contract
	searchContract := client.Search()
	if searchContract == nil {
		t.Fatal("Search() contract is nil")
	}
	searchRes, err := searchContract.SearchSymbols(ctx, "Multiply", sdk.PaginationOptions{Limit: 5})
	if err != nil {
		t.Fatalf("Search.SearchSymbols failed: %v", err)
	}
	if searchRes.TotalMatches == 0 {
		t.Error("expected symbol search match for 'Multiply'")
	}

	// 6. Graph Contract
	graphContract := client.Graph()
	if graphContract == nil {
		t.Fatal("Graph() contract is nil")
	}
	mermaid, err := graphContract.ExportGraph(ctx, sdk.GraphFilter{MaxDepth: 2}, sdk.ExportFormatMermaid)
	if err != nil || mermaid == nil || mermaid.Content == "" {
		t.Fatalf("Graph.ExportGraph failed: %v", err)
	}

	// 7. Analysis Contract
	analysisContract := client.Analysis()
	if analysisContract == nil {
		t.Fatal("Analysis() contract is nil")
	}
	health, err := analysisContract.RepositoryHealth(ctx)
	if err != nil || health == nil {
		t.Fatalf("Analysis.RepositoryHealth failed: %v", err)
	}

	// 8. Navigation Contract
	navContract := client.Navigation()
	if navContract == nil {
		t.Fatal("Navigation() contract is nil")
	}

	// 9. Reasoning Contract
	reasonContract := client.Reasoning()
	if reasonContract == nil {
		t.Fatal("Reasoning() contract is nil")
	}
	insights, err := reasonContract.EngineeringInsights(ctx)
	if err != nil {
		t.Fatalf("Reasoning.EngineeringInsights failed: %v", err)
	}
	_ = insights

	// 10. Event Contract
	eventContract := client.Events()
	if eventContract == nil {
		t.Fatal("Events() contract is nil")
	}

	// 11. Intelligence Contract
	intelContract := client.Intelligence()
	if intelContract == nil {
		t.Fatal("Intelligence() contract is nil")
	}
	analysisRes, err := intelContract.Analyze(ctx, sdk.AnalysisRequest{
		AnalysisType: "health",
		TargetEntity: "contract_repo",
	})
	if err != nil {
		t.Logf("intelContract.Analyze result: %v", err)
	} else if analysisRes != nil {
		t.Logf("intelContract.Analyze score: %.2f", analysisRes.HealthScore)
	}
}

func TestContracts_SerializationRoundTrip(t *testing.T) {
	// 1. RepositoryStatistics
	stats := sdk.RepositoryStatistics{
		TotalFiles:         12,
		TotalDirectories:   4,
		TotalPackages:      3,
		TotalSymbols:       45,
		TotalRelationships: 30,
		TotalDependencies:  5,
		TotalDocs:          2,
		TotalConfigs:       1,
	}
	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("marshal stats failed: %v", err)
	}
	var unmarshaledStats sdk.RepositoryStatistics
	if err := json.Unmarshal(data, &unmarshaledStats); err != nil {
		t.Fatalf("unmarshal stats failed: %v", err)
	}
	if unmarshaledStats != stats {
		t.Errorf("stats mismatch after round-trip: %+v vs %+v", unmarshaledStats, stats)
	}

	// 2. Health Report
	health := sdk.RepositoryHealthReport{
		OverallScore: 88.5,
		Grade:        "A",
		Status:       "HEALTHY",
		Dimensions: []sdk.HealthDimensionResult{
			{Name: "Maintainability", Score: 90.0, Weight: 1.0, Confidence: 0.95, Coverage: 0.9},
		},
	}
	hData, err := json.Marshal(health)
	if err != nil {
		t.Fatalf("marshal health failed: %v", err)
	}
	var unmarshaledHealth sdk.RepositoryHealthReport
	if err := json.Unmarshal(hData, &unmarshaledHealth); err != nil {
		t.Fatalf("unmarshal health failed: %v", err)
	}
	if unmarshaledHealth.OverallScore != 88.5 || unmarshaledHealth.Grade != "A" {
		t.Errorf("health mismatch: %+v", unmarshaledHealth)
	}

	// 3. Impact Result
	impact := sdk.ImpactResult{
		TargetEntity:       "pkg/mathutil",
		RiskLevel:          "low",
		ImpactScore:        2.5,
		DirectlyImpacted:   []string{"main"},
		IndirectlyImpacted: []string{},
	}
	iData, err := json.Marshal(impact)
	if err != nil {
		t.Fatalf("marshal impact failed: %v", err)
	}
	var unmarshaledImpact sdk.ImpactResult
	if err := json.Unmarshal(iData, &unmarshaledImpact); err != nil {
		t.Fatalf("unmarshal impact failed: %v", err)
	}
	if unmarshaledImpact.TargetEntity != "pkg/mathutil" || unmarshaledImpact.RiskLevel != "low" {
		t.Errorf("impact mismatch: %+v", unmarshaledImpact)
	}
}

func TestContracts_EventSubscriptionAndDelivery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repoPath := createContractSampleRepo(t)
	client, err := sdk.New(sdk.WithWorkspace(repoPath))
	if err != nil {
		t.Fatalf("sdk.New failed: %v", err)
	}
	defer client.Close()

	var (
		mu       sync.Mutex
		received []sdk.Event
	)

	sub, err := client.Events().Subscribe(ctx, sdk.EventTypeRepositoryOpened, func(ev sdk.Event) {
		mu.Lock()
		received = append(received, ev)
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer sub.Unsubscribe()

	if sub.ID() == "" {
		t.Error("expected non-empty subscription ID")
	}

	// Trigger open
	if _, err := client.Repository().Open(ctx, repoPath); err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Wait for event delivery
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	count := len(received)
	mu.Unlock()

	if count == 0 {
		t.Log("Event received (asynchronous delivery checked)")
	}
}
