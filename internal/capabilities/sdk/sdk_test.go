package sdk_test

import (
	"context"
	"sync"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/sdk"
	"github.com/unhield/limoxel/internal/capabilities/sdk/compatibility"
	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/sdk/testutil"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

func TestSDKInitializationAndDefaults(t *testing.T) {
	_, repoRoot := testutil.SetupTestRepository(t)

	client, err := sdk.New(sdk.WithWorkspace(repoRoot))
	if err != nil {
		t.Fatalf("sdk.New failed: %v", err)
	}

	if client.Workspace() != repoRoot {
		t.Errorf("got workspace %q, want %q", client.Workspace(), repoRoot)
	}

	if client.Version() != version.Current() {
		t.Errorf("got version %v, want %v", client.Version(), version.Current())
	}

	if client.Registry() == nil {
		t.Errorf("expected non-nil registry")
	}

	if client.Validator() == nil {
		t.Errorf("expected non-nil validator")
	}

	// Verify all 11 default contract descriptors are registered
	expectedContracts := []string{
		"RepositoryManagementContract",
		"FileContract",
		"PackageContract",
		"SymbolContract",
		"GraphContract",
		"SearchContract",
		"IntelligenceContract",
		"AnalysisContract",
		"NavigationContract",
		"ReasoningContract",
		"EventContract",
	}

	for _, name := range expectedContracts {
		desc, ok := client.Registry().Lookup(name)
		if !ok {
			t.Errorf("expected contract %q to be registered in lifecycle registry", name)
			continue
		}
		if desc.Name != name {
			t.Errorf("contract descriptor mismatch: got %s, want %s", desc.Name, name)
		}
	}
}

func TestCoreAndIntelligenceSDKServiceAccessors(t *testing.T) {
	ctx := context.Background()
	_, repoRoot := testutil.SetupTestRepository(t)

	client, err := sdk.New(sdk.WithWorkspace(repoRoot))
	if err != nil {
		t.Fatalf("sdk.New failed: %v", err)
	}

	// 1. Repository SDK
	if client.Repository() == nil {
		t.Errorf("expected non-nil Repository service")
	}
	repoInfo, err := client.Repository().Info(ctx)
	if err != nil {
		t.Fatalf("Repository().Info failed: %v", err)
	}
	if repoInfo.RootPath != repoRoot {
		t.Errorf("got repo root %q, want %q", repoInfo.RootPath, repoRoot)
	}

	// 2. File SDK
	if client.Files() == nil {
		t.Errorf("expected non-nil Files service")
	}
	files, err := client.Files().DiscoverFiles(ctx, contracts.FileFilter{}, contracts.PaginationOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Files().DiscoverFiles failed: %v", err)
	}
	if len(files) == 0 {
		t.Errorf("expected files to be discovered")
	}

	// 3. Package SDK
	if client.Packages() == nil {
		t.Errorf("expected non-nil Packages service")
	}
	pkgs, err := client.Packages().DiscoverPackages(ctx, contracts.PackageFilter{}, contracts.PaginationOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Packages().DiscoverPackages failed: %v", err)
	}
	if len(pkgs) == 0 {
		t.Errorf("expected packages to be discovered")
	}

	// 4. Symbol SDK
	if client.Symbols() == nil {
		t.Errorf("expected non-nil Symbols service")
	}
	sym, err := client.Symbols().LookupSymbol(ctx, "Add")
	if err != nil {
		t.Fatalf("Symbols().LookupSymbol failed: %v", err)
	}
	if sym.Name != "Add" {
		t.Errorf("got symbol %q, want Add", sym.Name)
	}

	// 5. Search SDK
	if client.Search() == nil {
		t.Errorf("expected non-nil Search service")
	}
	searchRes, err := client.Search().Search(ctx, contracts.SearchQuery{
		Query:  "Add",
		Domain: contracts.SearchDomainSymbol,
	})
	if err != nil {
		t.Fatalf("Search().Search failed: %v", err)
	}
	if searchRes.TotalMatches == 0 {
		t.Errorf("expected search matches for Add")
	}

	// 6. Graph SDK (Stage 3)
	if client.Graph() == nil {
		t.Errorf("expected non-nil Graph service")
	}
	nodes, edges, err := client.Graph().GraphInfo(ctx)
	if err != nil {
		t.Fatalf("Graph().GraphInfo failed: %v", err)
	}
	if nodes == 0 {
		t.Errorf("expected graph nodes > 0, got %d (edges: %d)", nodes, edges)
	}

	// 7. Analysis SDK (Stage 3)
	if client.Analysis() == nil {
		t.Errorf("expected non-nil Analysis service")
	}
	health, err := client.Analysis().RepositoryHealth(ctx)
	if err != nil {
		t.Fatalf("Analysis().RepositoryHealth failed: %v", err)
	}
	if health.OverallScore == 0 {
		t.Errorf("expected valid health score")
	}

	// 8. Navigation SDK (Stage 3)
	if client.Navigation() == nil {
		t.Errorf("expected non-nil Navigation service")
	}
	defRes, err := client.Navigation().GoToDefinition(ctx, "Add")
	if err != nil {
		t.Fatalf("Navigation().GoToDefinition failed: %v", err)
	}
	if defRes == nil {
		t.Errorf("expected non-nil definition result")
	}

	// 9. Reasoning SDK (Stage 3)
	if client.Reasoning() == nil {
		t.Errorf("expected non-nil Reasoning service")
	}
	insights, err := client.Reasoning().EngineeringInsights(ctx)
	if err != nil {
		t.Fatalf("Reasoning().EngineeringInsights failed: %v", err)
	}
	if len(insights) == 0 {
		t.Logf("no engineering insights generated for small sample repo")
	}

	// 10. Event SDK (Stage 3)
	if client.Events() == nil {
		t.Errorf("expected non-nil Events service")
	}
	sub, err := client.Events().Subscribe(ctx, contracts.EventTypeRepositoryOpened, func(evt contracts.Event) {})
	if err != nil {
		t.Fatalf("Events().Subscribe failed: %v", err)
	}
	_ = sub.Unsubscribe()

	// 11. Intelligence Facade (Stage 3)
	if client.Intelligence() == nil {
		t.Errorf("expected non-nil Intelligence facade")
	}
	anRes, err := client.Intelligence().Analyze(ctx, contracts.AnalysisRequest{AnalysisType: "health", TargetEntity: "sample_repo"})
	if err != nil {
		t.Fatalf("Intelligence().Analyze failed: %v", err)
	}
	if anRes.HealthScore == 0 {
		t.Errorf("expected valid health score from intelligence facade")
	}
}

func TestSDKValidation(t *testing.T) {
	client, err := sdk.New()
	if err != nil {
		t.Fatalf("sdk.New failed: %v", err)
	}

	// Valid evaluation for minor addition in minor bump
	target := contracts.NewBaseContract(
		"FileContract",
		lifecycle.CapabilityRepository,
		version.SemVer{Major: 1, Minor: 1, Patch: 0},
		lifecycle.StateSupported,
		"Updated file contract",
	)

	changes := []compatibility.APIChange{
		{
			APIName:     "DiscoverFiles",
			Kind:        compatibility.ChangeAddition,
			Description: "Added new discovery filter",
		},
	}

	report, err := client.ValidateRelease(target, changes)
	if err != nil {
		t.Fatalf("ValidateRelease failed: %v", err)
	}

	if !report.IsCompatible {
		t.Errorf("expected release to be compatible: %+v", report)
	}
}

func TestSDKLifecycleAndClose(t *testing.T) {
	client, err := sdk.New()
	if err != nil {
		t.Fatalf("sdk.New failed: %v", err)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Subsequent close should be idempotent
	if err := client.Close(); err != nil {
		t.Fatalf("Second Close failed: %v", err)
	}
}

func TestNilSDKReceiverSafety(t *testing.T) {
	var client *sdk.SDK

	if client.Version() != version.Current() {
		t.Errorf("expected current version on nil SDK")
	}
	if client.Workspace() != "." {
		t.Errorf("expected default workspace on nil SDK")
	}
	if client.Registry() != nil {
		t.Errorf("expected nil registry on nil SDK")
	}
	if client.Validator() != nil {
		t.Errorf("expected nil validator on nil SDK")
	}
	if client.Repository() != nil {
		t.Errorf("expected nil Repository on nil SDK")
	}
	if client.Files() != nil {
		t.Errorf("expected nil Files on nil SDK")
	}
	if client.Packages() != nil {
		t.Errorf("expected nil Packages on nil SDK")
	}
	if client.Symbols() != nil {
		t.Errorf("expected nil Symbols on nil SDK")
	}
	if client.Search() != nil {
		t.Errorf("expected nil Search on nil SDK")
	}
	if client.Graph() != nil {
		t.Errorf("expected nil Graph on nil SDK")
	}
	if client.Analysis() != nil {
		t.Errorf("expected nil Analysis on nil SDK")
	}
	if client.Navigation() != nil {
		t.Errorf("expected nil Navigation on nil SDK")
	}
	if client.Reasoning() != nil {
		t.Errorf("expected nil Reasoning on nil SDK")
	}
	if client.Events() != nil {
		t.Errorf("expected nil Events on nil SDK")
	}
	if client.Intelligence() != nil {
		t.Errorf("expected nil Intelligence facade on nil SDK")
	}
	if err := client.Close(); err != nil {
		t.Errorf("unexpected error on nil SDK Close: %v", err)
	}
	if err := client.RegisterContract(contracts.BaseContract{}); err == nil {
		t.Errorf("expected error on nil SDK RegisterContract")
	}
	if _, err := client.ValidateRelease(contracts.BaseContract{}, nil); err == nil {
		t.Errorf("expected error on nil SDK ValidateRelease")
	}
}

func TestSDKIntelligenceFacadeOperations(t *testing.T) {
	ctx := context.Background()
	_, repoRoot := testutil.SetupTestRepository(t)

	client, err := sdk.New(sdk.WithWorkspace(repoRoot))
	if err != nil {
		t.Fatalf("sdk.New failed: %v", err)
	}

	intel := client.Intelligence()
	if intel == nil {
		t.Fatalf("expected non-nil Intelligence facade")
	}

	// 1. Analyze
	anRes, err := intel.Analyze(ctx, contracts.AnalysisRequest{
		AnalysisType: "health",
		TargetEntity: "sample_repo",
	})
	if err != nil {
		t.Fatalf("intel.Analyze failed: %v", err)
	}
	if anRes.HealthScore == 0 {
		t.Errorf("expected non-zero health score")
	}

	// 2. Navigate
	navRes, err := intel.Navigate(ctx, "Add", "definition")
	if err != nil {
		t.Fatalf("intel.Navigate failed: %v", err)
	}
	if len(navRes.Targets) == 0 {
		t.Errorf("expected definition targets for Add")
	}

	// 3. Reason
	reasonRes, err := intel.Reason(ctx, contracts.ReasoningRequest{
		Objective: "Architecture review",
	})
	if err != nil {
		t.Fatalf("intel.Reason failed: %v", err)
	}
	if reasonRes.Conclusion == "" {
		t.Errorf("expected non-empty conclusion from Reason")
	}

	// 4. Events stream
	ch, err := intel.Events(ctx, string(contracts.EventTypeRepositoryOpened))
	if err != nil {
		t.Fatalf("intel.Events failed: %v", err)
	}
	if ch == nil {
		t.Errorf("expected non-nil events channel")
	}
}

func TestSDKConcurrencyAndContextCancellation(t *testing.T) {
	_, repoRoot := testutil.SetupTestRepository(t)

	client, err := sdk.New(sdk.WithWorkspace(repoRoot))
	if err != nil {
		t.Fatalf("sdk.New failed: %v", err)
	}

	// 1. Concurrency stress test across all services
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := context.Background()
			_, _ = client.Repository().Info(c)
			_, _ = client.Files().DiscoverFiles(c, contracts.FileFilter{}, contracts.PaginationOptions{Limit: 5})
			_, _ = client.Packages().DiscoverPackages(c, contracts.PackageFilter{}, contracts.PaginationOptions{Limit: 5})
			_, _ = client.Symbols().LookupSymbol(c, "Add")
			_, _ = client.Search().Search(c, contracts.SearchQuery{Query: "Add", Domain: contracts.SearchDomainSymbol})
			_, _, _ = client.Graph().GraphInfo(c)
			_, _ = client.Analysis().RepositoryHealth(c)
			_, _ = client.Navigation().GoToDefinition(c, "Add")
			_, _ = client.Reasoning().EngineeringInsights(c)
		}()
	}
	wg.Wait()

	// 2. Cancelled context verification
	cancCtx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel

	if _, err := client.Analysis().RepositoryHealth(cancCtx); err == nil {
		t.Errorf("expected error with cancelled context on RepositoryHealth")
	}
	if _, err := client.Navigation().GoToDefinition(cancCtx, "Add"); err == nil {
		t.Errorf("expected error with cancelled context on GoToDefinition")
	}
	if _, err := client.Reasoning().EngineeringInsights(cancCtx); err == nil {
		t.Errorf("expected error with cancelled context on EngineeringInsights")
	}
	if _, _, err := client.Graph().GraphInfo(cancCtx); err == nil {
		t.Errorf("expected error with cancelled context on GraphInfo")
	}
}
