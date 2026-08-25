package crossrepo

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
)

func createTestSymbol(id, name, pkgPath, filePath string, kind symbol.SymbolKind, signature string, pos *symbol.SourcePosition) *symbol.Symbol {
	cleanPkgPath := filepath.ToSlash(filepath.Clean(pkgPath))
	cleanFilePath := filepath.ToSlash(filepath.Clean(filePath))
	pkgName := filepath.Base(cleanPkgPath)
	return symbol.NewSymbol(
		id,
		kind,
		name,
		pkgName,
		cleanPkgPath,
		cleanFilePath,
		"",
		false,
		signature,
		"",
		false,
		nil,
		nil,
		pos,
		nil,
	)
}

func TestCrossFileAnalysis(t *testing.T) {
	pos1 := symbol.NewSourcePosition("internal/service/user.go", 10, 1, 100)
	pos2 := symbol.NewSourcePosition("internal/service/user_test.go", 15, 1, 150)

	sym1 := createTestSymbol("sym:UserService", "UserService", "internal/service", "internal/service/user.go", symbol.SymbolKindStruct, "type UserService struct", pos1)
	sym2 := createTestSymbol("sym:TestUserService", "TestUserService", "internal/service", "internal/service/user_test.go", symbol.SymbolKindFunction, "func TestUserService(t *testing.T)", pos2)
	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{sym1, sym2})

	refPos := symbol.NewSourcePosition("cmd/server/main.go", 25, 5, 300)
	ref := xref.NewReference("sym:Caller", "sym:UserService", xref.RefStruct, "cmd/server/main.go", refPos, xref.StateResolved, "var svc UserService")
	xrefDB := xref.NewReferenceDatabase([]*xref.Reference{ref})
	xrefModel := xref.NewXRefModel("test_repo", xrefDB, nil, nil, nil, nil, nil)

	analyzer := NewCrossFileAnalyzer()
	fileRels, props, deps, configs := analyzer.Analyze(
		symDB,
		xrefModel,
		nil,
		[]string{".golangci.yml", "go.mod"},
	)

	if len(fileRels) == 0 {
		t.Fatal("Expected file relationships to be generated")
	}

	// Verify test-to-source relationship
	foundTestSource := false
	for _, r := range fileRels {
		if r.Kind() == FileRelTestSource {
			foundTestSource = true
			if r.SourceFile() != "internal/service/user_test.go" || r.TargetFile() != "internal/service/user.go" {
				t.Fatalf("Unexpected test-to-source targets: src=%s, tgt=%s", r.SourceFile(), r.TargetFile())
			}
		}
	}
	if !foundTestSource {
		t.Fatal("Expected test_source relationship to be discovered")
	}

	// Verify symbol propagation
	foundProp := false
	for _, p := range props {
		if p.SymbolID() == "sym:UserService" {
			foundProp = true
			if len(p.ReferencingFiles()) == 0 {
				t.Fatal("Expected referencing files for propagated symbol")
			}
			if p.ReferencingFiles()[0] != "cmd/server/main.go" {
				t.Fatalf("Unexpected referencing file: %v", p.ReferencingFiles())
			}
		}
	}
	if !foundProp {
		t.Fatal("Expected symbol propagation record for UserService")
	}

	// Verify cross-file dependency
	if len(deps) == 0 {
		t.Fatal("Expected cross-file dependencies")
	}
	if deps[0].SourceFile() != "cmd/server/main.go" || deps[0].TargetFile() != "internal/service/user.go" {
		t.Fatalf("Unexpected cross-file dependency: src=%s, tgt=%s", deps[0].SourceFile(), deps[0].TargetFile())
	}

	// Verify shared config
	if len(configs) != 2 {
		t.Fatalf("Expected 2 shared configs, got %d", len(configs))
	}
}

func TestCrossPackageAnalysis(t *testing.T) {
	pos1 := symbol.NewSourcePosition("internal/service/auth.go", 10, 1, 100)
	pos2 := symbol.NewSourcePosition("cmd/server/main.go", 20, 1, 200)

	sym1 := createTestSymbol("sym:Authenticate", "Authenticate", "internal/service", "internal/service/auth.go", symbol.SymbolKindFunction, "func Authenticate() bool", pos1)
	sym2 := createTestSymbol("sym:Main", "Main", "cmd/server", "cmd/server/main.go", symbol.SymbolKindFunction, "func Main()", pos2)
	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{sym1, sym2})

	refPos := symbol.NewSourcePosition("cmd/server/main.go", 25, 5, 250)
	ref := xref.NewReference("sym:Main", "sym:Authenticate", xref.RefFunction, "cmd/server/main.go", refPos, xref.StateResolved, "auth.Authenticate()")
	xrefDB := xref.NewReferenceDatabase([]*xref.Reference{ref})

	callEdge := xref.NewCallEdge("sym:Main", "sym:Authenticate", xref.CallDirect, "cmd/server/main.go", refPos)
	callGraph := xref.NewCallGraph([]*xref.CallEdge{callEdge}, nil, nil, nil, nil, nil)

	xrefModel := xref.NewXRefModel("test_repo", xrefDB, callGraph, nil, nil, nil, nil)

	analyzer := NewCrossPackageAnalyzer()
	comms, contracts, apis := analyzer.Analyze(symDB, xrefModel, nil)

	if len(comms) == 0 {
		t.Fatal("Expected package communication to be detected")
	}

	foundCallComm := false
	for _, c := range comms {
		if c.Kind() == PkgCommCall {
			foundCallComm = true
			if c.SourcePackage() != "cmd/server" || c.TargetPackage() != "internal/service" {
				t.Fatalf("Unexpected package comm: src=%s, tgt=%s", c.SourcePackage(), c.TargetPackage())
			}
		}
	}
	if !foundCallComm {
		t.Fatal("Expected call package communication to be found")
	}

	// Verify package contracts
	if len(contracts) == 0 {
		t.Fatal("Expected package contracts")
	}

	// Verify API endpoints
	foundAPI := false
	for _, a := range apis {
		if a.SymbolID() == "sym:Authenticate" {
			foundAPI = true
			if a.Visibility() != APIVisibilityInternal {
				t.Fatalf("Expected internal API visibility for internal/service, got %v", a.Visibility())
			}
			if len(a.ConsumerPackages()) == 0 || a.ConsumerPackages()[0] != "cmd/server" {
				t.Fatalf("Unexpected consumer packages: %v", a.ConsumerPackages())
			}
		}
	}
	if !foundAPI {
		t.Fatal("Expected Authenticate API endpoint to be discovered")
	}
}

func TestCrossModuleAnalysis(t *testing.T) {
	mods := []ModuleInfo{
		{
			Path:         "github.com/example/root",
			Version:      "v1.0.0",
			ParentModule: "",
			Packages:     []string{"github.com/example/root/pkg/api"},
			Dependencies: map[string]string{
				"github.com/example/common": "v1.2.0",
				"github.com/example/core":   "v0.9.0",
			},
		},
		{
			Path:         "github.com/example/core",
			Version:      "v0.9.0",
			ParentModule: "github.com/example/root",
			Packages:     []string{"github.com/example/core/storage"},
			Dependencies: map[string]string{
				"github.com/example/common": "v1.2.0",
			},
		},
	}

	analyzer := NewCrossModuleAnalyzer()
	rels, shared, hierarchy, compats := analyzer.Analyze(mods)

	if len(rels) == 0 {
		t.Fatal("Expected module relationships")
	}

	// Check shared module detection (github.com/example/common consumed by 2 modules)
	foundShared := false
	for _, s := range shared {
		if s.ModulePath() == "github.com/example/common" {
			foundShared = true
			if len(s.ConsumingContexts()) != 2 {
				t.Fatalf("Expected 2 consuming contexts for common module, got %d", len(s.ConsumingContexts()))
			}
		}
	}
	if !foundShared {
		t.Fatal("Expected shared module for github.com/example/common")
	}

	// Check hierarchy
	if len(hierarchy) != 2 {
		t.Fatalf("Expected 2 hierarchy nodes, got %d", len(hierarchy))
	}

	// Check version compatibility
	if len(compats) == 0 {
		t.Fatal("Expected version compatibility entries")
	}
}

func TestWorkspaceAnalysis(t *testing.T) {
	repos := []RepositoryInput{
		{
			Root:         "d:/workspace/service-a",
			Name:         "service-a",
			Modules:      []string{"github.com/example/service-a"},
			Packages:     []string{"service-a/handler"},
			Dependencies: []string{"github.com/example/lib-common", "service-b/api"},
		},
		{
			Root:         "d:/workspace/service-b",
			Name:         "service-b",
			Modules:      []string{"github.com/example/service-b"},
			Packages:     []string{"service-b/api"},
			Dependencies: []string{"github.com/example/lib-common"},
		},
	}

	analyzer := NewWorkspaceAnalyzer()
	ws := analyzer.Analyze("d:/workspace", repos, nil)

	if ws == nil {
		t.Fatal("Expected non-nil WorkspaceModel")
	}

	if len(ws.Repositories()) != 2 {
		t.Fatalf("Expected 2 workspace repositories, got %d", len(ws.Repositories()))
	}

	// Verify inter-repo relationship (service-a depends on service-b's package)
	if len(ws.Relationships()) == 0 {
		t.Fatal("Expected inter-repository relationships")
	}

	// Verify shared dependency (lib-common consumed by both service-a and service-b)
	foundSharedDep := false
	for _, dep := range ws.SharedDependencies() {
		if dep.DependencyName() == "github.com/example/lib-common" {
			foundSharedDep = true
			if len(dep.ConsumingRepos()) != 2 {
				t.Fatalf("Expected 2 consuming repos for shared dep, got %d", len(dep.ConsumingRepos()))
			}
		}
	}
	if !foundSharedDep {
		t.Fatal("Expected shared dependency github.com/example/lib-common across repos")
	}

	// Verify shared architecture
	if len(ws.SharedArchitecture()) == 0 {
		t.Fatal("Expected shared architecture for multi-repo workspace")
	}
}

func TestEvolutionAnalysis(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	commits := []*CommitEvent{
		NewCommitEvent("c1", "Alice", "Initial commit", t1, []string{"main.go", "user.go"}, nil, nil),
		NewCommitEvent("c2", "Bob", "Add auth service", t2, []string{"auth.go", "auth_test.go", "f1.go", "f2.go", "f3.go", "f4.go"}, []string{"user.go"}, nil),
	}

	analyzer := NewEvolutionAnalyzer()
	evo := analyzer.Analyze("repo:d:/test", commits, 7, 2, 1, 20, 5)

	if evo == nil {
		t.Fatal("Expected non-nil EvolutionModel")
	}

	if len(evo.Commits()) != 2 {
		t.Fatalf("Expected 2 commits, got %d", len(evo.Commits()))
	}

	if len(evo.StructuralEvolution()) != 2 {
		t.Fatalf("Expected 2 structural evolution points, got %d", len(evo.StructuralEvolution()))
	}

	// Check growth metrics
	if len(evo.GrowthMetrics()) == 0 {
		t.Fatal("Expected growth metrics")
	}
	growth := evo.GrowthMetrics()[0]
	if growth.TotalFiles() != 7 || growth.TotalPackages() != 2 {
		t.Fatalf("Unexpected growth metrics: files=%d, pkgs=%d", growth.TotalFiles(), growth.TotalPackages())
	}
}

func TestCrossRepoValidation(t *testing.T) {
	validator := NewCrossRepoValidator()

	// 1. Invalid cross-file dependency with empty target
	invalidDep := NewCrossFileDependency("main.go", "", "", nil, true)
	// 2. Incompatible version compatibility
	incompat := NewVersionCompatibility("github.com/example/mod", "v2.0.0", "v1.0.0", CompatIncompatible, "major version mismatch")

	report := validator.Validate(
		nil,
		nil,
		[]*CrossFileDependency{invalidDep},
		nil,
		nil,
		nil,
		[]*VersionCompatibility{incompat},
		nil,
	)

	if report == nil {
		t.Fatal("Expected non-nil ValidationReport")
	}

	if report.Status() != StatusInvalid {
		t.Fatalf("Expected StatusInvalid, got %v", report.Status())
	}

	if report.TotalErrors() != 2 {
		t.Fatalf("Expected 2 errors in validation report, got %d", report.TotalErrors())
	}
}

func TestCrossRepoEngine_AnalyzeAndConcurrency(t *testing.T) {
	engine := NewEngine()

	pos := symbol.NewSourcePosition("pkg/user.go", 10, 1, 100)
	sym := createTestSymbol("sym:User", "User", "pkg", "pkg/user.go", symbol.SymbolKindStruct, "type User struct", pos)
	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{sym})

	params := AnalysisParams{
		WorkspaceRoot: "d:/workspace",
		Repositories: []RepositoryInput{
			{
				Root:         "d:/workspace/repo1",
				Name:         "repo1",
				Packages:     []string{"pkg"},
				Dependencies: []string{"lib"},
			},
		},
		SymbolDB:          symDB,
		TotalFiles:        5,
		TotalPackages:     1,
		TotalModules:      1,
		TotalSymbols:      1,
		TotalDependencies: 1,
	}

	model, err := engine.Analyze(params)
	if err != nil {
		t.Fatalf("Engine.Analyze failed: %v", err)
	}

	if model == nil || engine.Model() == nil {
		t.Fatal("Expected non-nil CrossRepoModel")
	}

	if engine.CrossFileAnalyzer() == nil || engine.CrossPackageAnalyzer() == nil ||
		engine.CrossModuleAnalyzer() == nil || engine.WorkspaceAnalyzer() == nil ||
		engine.EvolutionAnalyzer() == nil || engine.Validator() == nil {
		t.Fatal("Expected all sub-analyzers to be initialized")
	}

	// Concurrency test
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = engine.Model().FileRelationships()
			_ = engine.Model().SymbolPropagations()
			_ = engine.Model().CrossFileDependencies()
			_ = engine.Model().PackageCommunications()
			_ = engine.Model().ModuleRelationships()
			_ = engine.Model().Workspace()
			_ = engine.Model().Evolution()
		}(i)
	}
	wg.Wait()
}

func TestCrossRepoErrors(t *testing.T) {
	err := NewCrossRepoError(ErrCatInvalidInput, "ERR_TEST", "test crossrepo error")
	if err.Category() != ErrCatInvalidInput || err.Code() != "ERR_TEST" {
		t.Fatalf("Unexpected error properties: %v", err)
	}
	if !IsCategory(err, ErrCatInvalidInput) {
		t.Fatal("IsCategory returned false for matching category")
	}
	if IsCategory(err, ErrCatInternal) {
		t.Fatal("IsCategory returned true for non-matching category")
	}
}

func TestMultiRepoIsolationAndCollisions(t *testing.T) {
	// 2 distinct repositories with same package and symbol names
	repo1 := RepositoryInput{
		Root:         "d:/workspace/repoA",
		Name:         "repoA",
		Modules:      []string{"github.com/org/service"},
		Packages:     []string{"github.com/org/service/pkg/handler"},
		Dependencies: []string{"github.com/org/shared-lib"},
	}
	repo2 := RepositoryInput{
		Root:         "d:/workspace/repoB",
		Name:         "repoB",
		Modules:      []string{"github.com/org/service"},
		Packages:     []string{"github.com/org/service/pkg/handler"},
		Dependencies: []string{"github.com/org/shared-lib"},
	}

	analyzer := NewWorkspaceAnalyzer()
	ws := analyzer.Analyze("d:/workspace", []RepositoryInput{repo1, repo2}, nil)

	if len(ws.Repositories()) != 2 {
		t.Fatalf("Expected 2 repositories, got %d", len(ws.Repositories()))
	}
	if ws.Repositories()[0].ID() == ws.Repositories()[1].ID() {
		t.Fatalf("Repository identity collision: repo0=%s, repo1=%s", ws.Repositories()[0].ID(), ws.Repositories()[1].ID())
	}

	// Verify shared dependency is correctly associated with both distinct repo IDs
	if len(ws.SharedDependencies()) != 1 {
		t.Fatalf("Expected 1 shared dependency, got %d", len(ws.SharedDependencies()))
	}
	sDep := ws.SharedDependencies()[0]
	if len(sDep.ConsumingRepos()) != 2 {
		t.Fatalf("Expected 2 distinct consumers for shared dependency, got %d", len(sDep.ConsumingRepos()))
	}
	if sDep.ConsumingRepos()[0] == sDep.ConsumingRepos()[1] {
		t.Fatal("Consuming repos must be distinct")
	}
}

func TestVersionCompatibilityEdgeCases(t *testing.T) {
	mods := []ModuleInfo{
		{
			Path:    "modA",
			Version: "v1.0.0",
			Dependencies: map[string]string{
				"modB": "v1.2.3",
				"modC": "incompatible_v2",
				"modD": "",
			},
		},
	}

	analyzer := NewCrossModuleAnalyzer()
	_, _, _, compats := analyzer.Analyze(mods)

	compatMap := make(map[string]VersionCompatibilityState)
	for _, c := range compats {
		compatMap[c.ModulePath()+":"+c.RequiredVersion()] = c.State()
	}

	if compatMap["modB:v1.2.3"] != CompatCompatible {
		t.Fatalf("Expected modB to be Compatible, got %v", compatMap["modB:v1.2.3"])
	}
	if compatMap["modC:incompatible_v2"] != CompatIncompatible {
		t.Fatalf("Expected modC to be Incompatible, got %v", compatMap["modC:incompatible_v2"])
	}
	if compatMap["modD:"] != CompatUnavailable {
		t.Fatalf("Expected modD to be Unavailable, got %v", compatMap["modD:"])
	}
}

func TestDeterminismRepeatedExecution(t *testing.T) {
	engine1 := NewEngine()
	engine2 := NewEngine()

	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{
		createTestSymbol("sym:1", "Alpha", "pkg1", "pkg1/a.go", symbol.SymbolKindStruct, "", nil),
		createTestSymbol("sym:2", "Beta", "pkg2", "pkg2/b.go", symbol.SymbolKindFunction, "", nil),
	})

	params := AnalysisParams{
		WorkspaceRoot: "d:/workspace",
		Repositories: []RepositoryInput{
			{
				Root:         "d:/workspace/r1",
				Name:         "r1",
				Packages:     []string{"pkg1", "pkg2"},
				Dependencies: []string{"depX"},
			},
		},
		SymbolDB:          symDB,
		KnownConfigs:      []string{"go.mod", ".golangci.yml"},
		TotalFiles:        2,
		TotalPackages:     2,
		TotalModules:      1,
		TotalSymbols:      2,
		TotalDependencies: 1,
	}

	m1, err1 := engine1.Analyze(params)
	m2, err2 := engine2.Analyze(params)

	if err1 != nil || err2 != nil {
		t.Fatalf("Engine analysis failed: err1=%v, err2=%v", err1, err2)
	}

	if len(m1.FileRelationships()) != len(m2.FileRelationships()) {
		t.Fatalf("File relationship count mismatch: %d vs %d", len(m1.FileRelationships()), len(m2.FileRelationships()))
	}
	for i := range m1.FileRelationships() {
		if m1.FileRelationships()[i].ID() != m2.FileRelationships()[i].ID() {
			t.Fatalf("File relationship ordering mismatch at %d: %s vs %s", i, m1.FileRelationships()[i].ID(), m2.FileRelationships()[i].ID())
		}
	}

	if len(m1.SharedConfigs()) != len(m2.SharedConfigs()) {
		t.Fatalf("Shared configs count mismatch: %d vs %d", len(m1.SharedConfigs()), len(m2.SharedConfigs()))
	}
	for i := range m1.SharedConfigs() {
		if m1.SharedConfigs()[i].ID() != m2.SharedConfigs()[i].ID() {
			t.Fatalf("Shared config ordering mismatch at %d: %s vs %s", i, m1.SharedConfigs()[i].ID(), m2.SharedConfigs()[i].ID())
		}
	}
}
