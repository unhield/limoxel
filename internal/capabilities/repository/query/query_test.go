package query_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/graph"
	"github.com/unhield/limoxel/internal/capabilities/repository/indexing"
	"github.com/unhield/limoxel/internal/capabilities/repository/language"
	"github.com/unhield/limoxel/internal/capabilities/repository/metadata"
	"github.com/unhield/limoxel/internal/capabilities/repository/query"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
	langreg "github.com/unhield/limoxel/internal/language"
)

func setupTestLanguageRegistry(t *testing.T) *langreg.Registry {
	t.Helper()
	reg := langreg.NewRegistry()

	goLang, _ := langreg.New("go", "Go", []string{".go"}, nil, []string{"golang"})
	mdLang, _ := langreg.New("markdown", "Markdown", []string{".md"}, nil, []string{"md"})
	jsonLang, _ := langreg.New("json", "JSON", []string{".json"}, nil, []string{"json"})
	yamlLang, _ := langreg.New("yaml", "YAML", []string{".yaml", ".yml"}, nil, []string{"yaml"})

	_ = reg.Register(goLang)
	_ = reg.Register(mdLang)
	_ = reg.Register(jsonLang)
	_ = reg.Register(yamlLang)

	return reg
}

// Helper: Sets up full upstream analysis pipeline for a test repository
func setupTestRepository(t *testing.T) (*query.RepositoryService, string) {
	t.Helper()
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "sample_repo")
	_ = os.MkdirAll(filepath.Join(repoRoot, "pkg", "math"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "pkg", "auth"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "docs"), 0755)

	// Go source files
	mainSrc := `package main

import "sample_repo/pkg/math"

func main() {
	math.Add(1, 2)
}
`
	mathSrc := `package math

type Calculator interface {
	Compute(a, b int) int
}

type BasicCalc struct{}

func (b *BasicCalc) Compute(a, b int) int {
	return Add(a, b)
}

func Add(a, b int) int {
	return a + b
}

func Multiply(a, b int) int {
	return a * b
}
`
	authSrc := `package auth

type Authenticator struct {
	SecretKey string
}

func (a *Authenticator) Verify(token string) bool {
	return token != ""
}
`
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte(mainSrc), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "pkg", "math", "math.go"), []byte(mathSrc), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "pkg", "auth", "auth.go"), []byte(authSrc), 0644)

	// Config and doc files
	_ = os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Sample Repo\nDocumentation here."), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "docs", "guide.md"), []byte("# User Guide\nDetailed instructions."), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "config.json"), []byte(`{"app": "sample", "port": 8080}`), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "secret_config.yaml"), []byte(`api_key: "super_secret_token"`), 0644)

	// Go modules
	_ = os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module sample_repo\n\ngo 1.21\n"), 0644)

	// Initialize all pipeline engines
	reg := setupTestLanguageRegistry(t)
	disc, err := discovery.New(reg)
	if err != nil {
		t.Fatalf("failed creating discoverer: %v", err)
	}

	metaCollector, _ := metadata.New(disc)
	langAnalyzer, _ := language.New(disc)
	depAnalyzer, _ := dependency.New(disc)
	indexer, _ := indexing.New(disc)
	symEngine, _ := symbol.New(disc)
	xrefEngine, _ := xref.New(disc, symEngine, depAnalyzer)
	graphEngine, _ := graph.New(disc, metaCollector, langAnalyzer, depAnalyzer, indexer, symEngine, xrefEngine)

	svc := query.NewRepositoryService(
		disc,
		metaCollector,
		langAnalyzer,
		depAnalyzer,
		indexer,
		symEngine,
		xrefEngine,
		graphEngine,
	)

	err = svc.Load(repoRoot)
	if err != nil {
		t.Fatalf("failed to load repository service: %v", err)
	}

	return svc, repoRoot
}

// 1. Repository Service Layer & Lifecycle
func TestService_LifecycleAndLoading(t *testing.T) {
	t.Run("unloaded service returns appropriate errors", func(t *testing.T) {
		svc := query.NewRepositoryService(nil, nil, nil, nil, nil, nil, nil, nil)
		if svc.LifecycleState() != query.StateUnloaded {
			t.Errorf("expected UNLOADED state, got %v", svc.LifecycleState())
		}
		_, err := svc.Metadata()
		if err == nil || !query.IsCategory(err, query.ErrCatNotLoaded) {
			t.Errorf("expected NOT_LOADED error on unloaded service, got: %v", err)
		}
		_, err = svc.Statistics()
		if err == nil || !query.IsCategory(err, query.ErrCatNotLoaded) {
			t.Errorf("expected NOT_LOADED error on unloaded service, got: %v", err)
		}
	})

	t.Run("loaded service transitions to READY and exposes metadata and statistics", func(t *testing.T) {
		svc, repoRoot := setupTestRepository(t)
		if svc.LifecycleState() != query.StateReady {
			t.Errorf("expected READY state, got %v", svc.LifecycleState())
		}

		meta, err := svc.Metadata()
		if err != nil {
			t.Fatalf("failed to get metadata: %v", err)
		}
		if meta.Root() != filepath.ToSlash(repoRoot) && meta.Root() != repoRoot {
			t.Errorf("unexpected root: %s vs %s", meta.Root(), repoRoot)
		}
		if len(meta.Capabilities()) == 0 {
			t.Errorf("expected non-empty capabilities list")
		}

		stats, err := svc.Statistics()
		if err != nil {
			t.Fatalf("failed to get statistics: %v", err)
		}
		if !stats.IsAvailable() {
			t.Errorf("expected statistics to be available")
		}
		if stats.FileCount() == 0 {
			t.Errorf("expected >0 file count, got %d", stats.FileCount())
		}
		if stats.SymbolCount() == 0 {
			t.Errorf("expected >0 symbol count, got %d", stats.SymbolCount())
		}
	})

	t.Run("closed service rejects subsequent operations", func(t *testing.T) {
		svc, _ := setupTestRepository(t)
		_ = svc.Close()
		if svc.LifecycleState() != query.StateClosed {
			t.Errorf("expected CLOSED state, got %v", svc.LifecycleState())
		}
		_, err := svc.Metadata()
		if err == nil || !query.IsCategory(err, query.ErrCatLifecycle) {
			t.Errorf("expected lifecycle error on closed service, got: %v", err)
		}
		err = svc.Load("some/path")
		if err == nil || !query.IsCategory(err, query.ErrCatLifecycle) {
			t.Errorf("expected lifecycle error when loading closed service, got: %v", err)
		}
	})
}

// 2. Symbol APIs
func TestSymbols_API(t *testing.T) {
	svc, _ := setupTestRepository(t)
	symAPI := svc.Symbols()
	if symAPI == nil {
		t.Fatalf("expected non-nil SymbolAPI")
	}

	t.Run("FindSymbol by ID", func(t *testing.T) {
		syms, err := symAPI.ListSymbols(query.ScopeRepository, "")
		if err != nil || len(syms) == 0 {
			t.Fatalf("expected symbols, got %v (len=%d)", err, len(syms))
		}

		targetID := syms[0].ID()
		found, err := symAPI.FindSymbol(targetID)
		if err != nil {
			t.Fatalf("FindSymbol failed: %v", err)
		}
		if found.ID() != targetID {
			t.Errorf("expected ID %s, got %s", targetID, found.ID())
		}
	})

	t.Run("ListSymbols by Scopes", func(t *testing.T) {
		allSyms, _ := symAPI.ListSymbols(query.ScopeRepository, "")
		if len(allSyms) == 0 {
			t.Fatalf("expected repository symbols")
		}

		pkgSyms, err := symAPI.ListSymbols(query.ScopePackage, "math")
		if err != nil || len(pkgSyms) == 0 {
			t.Fatalf("expected math package symbols, got %v (len=%d)", err, len(pkgSyms))
		}

		fileSyms, err := symAPI.ListSymbols(query.ScopeFile, "pkg/math/math.go")
		if err != nil || len(fileSyms) == 0 {
			t.Fatalf("expected math.go file symbols, got %v", err)
		}
	})

	t.Run("LookupByType", func(t *testing.T) {
		funcs, err := symAPI.LookupByType(symbol.SymbolKindFunction)
		if err != nil || len(funcs) == 0 {
			t.Fatalf("expected functions, got %v", err)
		}
		for _, f := range funcs {
			if f.Kind() != symbol.SymbolKindFunction {
				t.Errorf("expected kind function, got %v", f.Kind())
			}
		}

		ifaces, err := symAPI.LookupByType(symbol.SymbolKindInterface)
		if err != nil || len(ifaces) == 0 {
			t.Fatalf("expected interfaces, got %v", err)
		}
	})

	t.Run("LookupByName and ambiguity handling", func(t *testing.T) {
		addMatches, err := symAPI.LookupByName("Add")
		if err != nil || len(addMatches) == 0 {
			t.Fatalf("expected match for 'Add', got %v", err)
		}
		if addMatches[0].Name() != "Add" {
			t.Errorf("expected symbol name 'Add', got %s", addMatches[0].Name())
		}

		// Non-existent symbol
		_, err = symAPI.LookupByName("NonExistentSymbol123")
		if err == nil || !query.IsCategory(err, query.ErrCatNotFound) {
			t.Errorf("expected NOT_FOUND error, got: %v", err)
		}
	})
}

// 3. Graph APIs
func TestGraph_API(t *testing.T) {
	svc, _ := setupTestRepository(t)
	graphAPI := svc.Graph()
	if graphAPI == nil {
		t.Fatalf("expected non-nil GraphAPI")
	}

	t.Run("GetNode by ID", func(t *testing.T) {
		node, err := graphAPI.GetNode("doc:README.md")
		if err != nil {
			t.Fatalf("GetNode failed: %v", err)
		}
		if node.ID() != "doc:README.md" {
			t.Errorf("expected doc:README.md, got %s", node.ID())
		}

		_, err = graphAPI.GetNode("non_existent_node_id")
		if err == nil || !query.IsCategory(err, query.ErrCatNotFound) {
			t.Errorf("expected NOT_FOUND error, got: %v", err)
		}
	})

	t.Run("Traverse bounded paths with zero-depth and depth N", func(t *testing.T) {
		// Zero-depth traversal returns exactly the starting node
		zeroRes, err := graphAPI.Traverse("doc:README.md", graph.DirOutbound, 0)
		if err != nil {
			t.Fatalf("zero-depth traversal failed: %v", err)
		}
		if zeroRes.TotalNodes() != 1 || zeroRes.Nodes()[0].ID() != "doc:README.md" {
			t.Errorf("expected start node only at depth 0, got %d nodes", zeroRes.TotalNodes())
		}
		if zeroRes.TotalRelationships() != 0 {
			t.Errorf("expected 0 relationships at depth 0, got %d", zeroRes.TotalRelationships())
		}

		// Depth 2 traversal
		boundedRes, err := graphAPI.Traverse("doc:README.md", graph.DirBoth, 2)
		if err != nil {
			t.Fatalf("bounded traversal failed: %v", err)
		}
		if boundedRes.TotalNodes() < 1 {
			t.Errorf("expected >=1 nodes in bounded traversal")
		}
	})

	t.Run("LookupDependencies and LookupCallGraph", func(t *testing.T) {
		deps, err := graphAPI.LookupDependencies("")
		if err != nil {
			t.Fatalf("LookupDependencies failed: %v", err)
		}
		_ = deps

		calls, err := graphAPI.LookupCallGraph("sym:main.go:main:main:6", query.CallDirectionOutbound)
		if err != nil {
			t.Fatalf("LookupCallGraph failed: %v", err)
		}
		_ = calls
	})
}

// 4. Search Engine & Security Boundary
func TestSearch_Engine(t *testing.T) {
	svc, _ := setupTestRepository(t)
	searchEng := svc.Search()
	if searchEng == nil {
		t.Fatalf("expected non-nil SearchEngine")
	}

	t.Run("Search symbols, files, documentation", func(t *testing.T) {
		opts := query.DefaultSearchOptions()

		// Symbol search
		symResults, err := searchEng.SearchSymbols("Add", opts)
		if err != nil || len(symResults) == 0 {
			t.Fatalf("SearchSymbols failed: %v (len=%d)", err, len(symResults))
		}
		if symResults[0].Domain() != query.DomainSymbol {
			t.Errorf("expected SYMBOL domain, got %v", symResults[0].Domain())
		}

		// File search
		fileResults, err := searchEng.SearchFiles("math.go", opts)
		if err != nil || len(fileResults) == 0 {
			t.Fatalf("SearchFiles failed: %v", err)
		}

		// Doc search
		docResults, err := searchEng.SearchDocumentation("guide", opts)
		if err != nil || len(docResults) == 0 {
			t.Fatalf("SearchDocumentation failed: %v", err)
		}
	})

	t.Run("Configuration search masks sensitive secrets", func(t *testing.T) {
		opts := query.DefaultSearchOptions()
		cfgResults, err := searchEng.SearchConfiguration("secret", opts)
		if err != nil {
			t.Fatalf("SearchConfiguration failed: %v", err)
		}
		for _, item := range cfgResults {
			if item.Name() == "secret_config.yaml" {
				if item.Snippet() != "***MASKED_CONFIG***" {
					t.Errorf("expected secret config snippet to be masked, got: %s", item.Snippet())
				}
			}
		}
	})

	t.Run("Fuzzy search produces deterministic ranked results", func(t *testing.T) {
		opts := query.DefaultSearchOptions()
		fuzzy1, err1 := searchEng.FuzzySearch("comput", query.DomainSymbol, opts)
		if err1 != nil || len(fuzzy1) == 0 {
			t.Fatalf("FuzzySearch failed: %v", err1)
		}

		fuzzy2, _ := searchEng.FuzzySearch("comput", query.DomainSymbol, opts)
		if len(fuzzy1) != len(fuzzy2) {
			t.Fatalf("fuzzy search count mismatch: %d vs %d", len(fuzzy1), len(fuzzy2))
		}
		for i := range fuzzy1 {
			if fuzzy1[i].EntityID() != fuzzy2[i].EntityID() || fuzzy1[i].Score() != fuzzy2[i].Score() {
				t.Fatalf("fuzzy search nondeterminism at index %d", i)
			}
		}
	})
}

// 5. Immutability & Defensive Copying
func TestQuery_ImmutabilityAndDefensiveCopying(t *testing.T) {
	svc, _ := setupTestRepository(t)

	meta, _ := svc.Metadata()
	langs := meta.Languages()
	if len(langs) > 0 {
		langs[0] = "MUTATED_LANGUAGE"
		recheckedMeta, _ := svc.Metadata()
		if len(recheckedMeta.Languages()) > 0 && recheckedMeta.Languages()[0] == "MUTATED_LANGUAGE" {
			t.Fatalf("RepositoryMetadataDTO internal slice aliased and mutated!")
		}
	}

	searchRes, _ := svc.Search().Search("math", query.DomainAll, query.DefaultSearchOptions())
	items := searchRes.Items()
	if len(items) > 0 {
		items[0] = nil
		recheckedItems := searchRes.Items()
		if recheckedItems[0] == nil {
			t.Fatalf("SearchResultDTO items slice aliased and mutated!")
		}
	}
}

// 6. Determinism & Concurrency
func TestQuery_DeterminismAndConcurrency(t *testing.T) {
	svc, _ := setupTestRepository(t)

	var wg sync.WaitGroup
	for w := 0; w < 16; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			_, _ = svc.Metadata()
			_, _ = svc.Statistics()
			_, _ = svc.Symbols().ListSymbols(query.ScopeRepository, "")
			_, _ = svc.Symbols().LookupByName("Compute")
			_, _ = svc.Graph().Traverse("doc:README.md", graph.DirBoth, 2)
			_, _ = svc.Search().Search("math", query.DomainAll, query.DefaultSearchOptions())
			_, _ = svc.Search().FuzzySearch("Add", query.DomainSymbol, query.DefaultSearchOptions())
		}(w)
	}
	wg.Wait()
}

// 7. Adversarial & Error Handling
func TestQuery_AdversarialAndErrorHandling(t *testing.T) {
	svc, _ := setupTestRepository(t)

	t.Run("empty search query returns ErrEmptyQuery", func(t *testing.T) {
		_, err := svc.Search().Search("   ", query.DomainAll, query.DefaultSearchOptions())
		if err == nil || !query.IsCategory(err, query.ErrCatInvalidInput) {
			t.Errorf("expected INVALID_INPUT error on empty query, got: %v", err)
		}
	})

	t.Run("excessive traversal depth returns ErrMaxDepthExceeded", func(t *testing.T) {
		_, err := svc.Graph().Traverse("doc:README.md", graph.DirOutbound, 999)
		if err == nil || !query.IsCategory(err, query.ErrCatInvalidInput) {
			t.Errorf("expected INVALID_INPUT on depth > 100, got: %v", err)
		}
	})

	t.Run("invalid scope returns ErrInvalidInput", func(t *testing.T) {
		_, err := svc.Symbols().ListSymbols(query.ScopeKind("INVALID_SCOPE"), "")
		if err == nil || !query.IsCategory(err, query.ErrCatInvalidInput) {
			t.Errorf("expected INVALID_INPUT on invalid scope, got: %v", err)
		}
	})
}

// 8. Service LoadFromModels & LoadFromRepository
func TestService_LoadFromModelsAndRepository(t *testing.T) {
	svc, _ := setupTestRepository(t)

	meta, err := svc.Metadata()
	if err != nil {
		t.Fatalf("failed to get metadata: %v", err)
	}
	if meta.Name() == "" {
		t.Errorf("expected non-empty repository name")
	}

	// All-nil models set must be rejected
	emptySvc := query.NewRepositoryService(nil, nil, nil, nil, nil, nil, nil, nil)
	err = emptySvc.LoadFromModels(nil, nil, nil, nil, nil, nil, nil, nil)
	if err == nil || !query.IsCategory(err, query.ErrCatInvalidInput) {
		t.Errorf("expected INVALID_INPUT error when loading all-nil model set, got: %v", err)
	}

	// Loading with valid knowledge graph model should succeed
	kgSvc := query.NewRepositoryService(nil, nil, nil, nil, nil, nil, nil, nil)
	g := graph.NewKnowledgeGraph("/root", nil, nil)
	err = kgSvc.LoadFromModels(nil, nil, nil, nil, nil, nil, nil, g)
	if err != nil {
		t.Fatalf("LoadFromModels failed with valid KG: %v", err)
	}
	if kgSvc.LifecycleState() != query.StateReady {
		t.Errorf("expected READY lifecycle state after LoadFromModels")
	}
}

// 9. Graph LookupRelationships
func TestGraph_LookupRelationships(t *testing.T) {
	svc, _ := setupTestRepository(t)
	graphAPI := svc.Graph()

	rels, err := graphAPI.LookupRelationships("", "", "")
	if err != nil {
		t.Fatalf("LookupRelationships failed: %v", err)
	}
	if len(rels) == 0 {
		t.Errorf("expected relationships to be found in graph")
	}
}

// 10. Search Extended Domains and Options
func TestSearch_ExtendedDomainsAndOptions(t *testing.T) {
	svc, _ := setupTestRepository(t)
	searchEng := svc.Search()

	t.Run("Package Search", func(t *testing.T) {
		opts := query.DefaultSearchOptions()
		res, err := searchEng.Search("math", query.DomainPackage, opts)
		if err != nil {
			t.Fatalf("Package search failed: %v", err)
		}
		if res.TotalMatches() == 0 {
			t.Errorf("expected package match for math")
		}
	})

	t.Run("ExactMatch and CaseSensitive options", func(t *testing.T) {
		opts := query.SearchOptions{
			ExactMatch:    true,
			CaseSensitive: true,
			MaxResults:    10,
		}
		res, err := searchEng.Search("Add", query.DomainSymbol, opts)
		if err != nil || res.TotalMatches() == 0 {
			t.Fatalf("Exact match search failed: %v", err)
		}
	})
}

// 11. Error Structures, Categories, and Unwrap
func TestError_StructuresAndUnwrap(t *testing.T) {
	innerErr := os.ErrNotExist
	wrapped := query.WrapQueryError(query.ErrCatNotFound, "CODE_TEST", "entity missing", innerErr)

	if wrapped.Category() != query.ErrCatNotFound {
		t.Errorf("expected category %v, got %v", query.ErrCatNotFound, wrapped.Category())
	}
	if wrapped.Code() != "CODE_TEST" {
		t.Errorf("expected code CODE_TEST, got %v", wrapped.Code())
	}
	if wrapped.Message() != "entity missing" {
		t.Errorf("expected message 'entity missing', got %v", wrapped.Message())
	}
	if wrapped.Unwrap() != innerErr {
		t.Errorf("expected unwrap to match innerErr")
	}
	if !query.IsCategory(wrapped, query.ErrCatNotFound) {
		t.Errorf("IsCategory should return true")
	}
	if query.IsCategory(wrapped, query.ErrCatInternal) {
		t.Errorf("IsCategory should return false for mismatched category")
	}
}

// 12. Models Defensive Copies and Getters
func TestModels_DefensiveCopiesAndGetters(t *testing.T) {
	rel := query.NewRelationshipDTO(
		"rel:1",
		graph.RelContains,
		"src:1",
		"tgt:1",
		[]string{"prov1", "prov2"},
		map[string]string{"k1": "v1"},
	)
	if rel.ID() != "rel:1" || rel.Type() != graph.RelContains {
		t.Errorf("unexpected relationship getters")
	}
	provs := rel.Provenance()
	if len(provs) != 2 {
		t.Errorf("expected 2 provenance items")
	}
	provs[0] = "MUTATED"
	if rel.Provenance()[0] == "MUTATED" {
		t.Errorf("RelationshipDTO provenance slice leaked internal reference")
	}

	meta := rel.Metadata()
	meta["k1"] = "MUTATED"
	if rel.Metadata()["k1"] == "MUTATED" {
		t.Errorf("RelationshipDTO metadata map leaked internal reference")
	}

	node := query.NewGraphNodeDTO(
		"node:1",
		graph.NodeFile,
		"file.go",
		"pkg/file.go",
		"main",
		"pkg",
		map[string]string{"attr": "val"},
	)
	if node.ID() != "node:1" || node.Type() != graph.NodeFile {
		t.Errorf("unexpected node getters")
	}
	nodeMeta := node.Metadata()
	nodeMeta["attr"] = "MUTATED"
	if node.Metadata()["attr"] == "MUTATED" {
		t.Errorf("GraphNodeDTO metadata map leaked internal reference")
	}

	srItem := query.NewSearchResultItem(
		"ent:1",
		query.DomainSymbol,
		"SymName",
		"file.go",
		"pkg",
		"func",
		0.95,
		"snippet",
		[]int{1, 2},
	)
	if srItem.String() == "" {
		t.Errorf("expected non-empty String() for SearchResultItem")
	}
	hl := srItem.Highlights()
	hl[0] = 999
	if srItem.Highlights()[0] == 999 {
		t.Errorf("SearchResultItem highlights slice leaked internal reference")
	}
}

// 13. Metadata Unavailable Facts
func TestService_MetadataUnavailableFacts(t *testing.T) {
	// Service loaded with KnowledgeGraph only (no metadata profile)
	kgSvc := query.NewRepositoryService(nil, nil, nil, nil, nil, nil, nil, nil)
	g := graph.NewKnowledgeGraph("/root", nil, nil)
	_ = kgSvc.LoadFromModels(nil, nil, nil, nil, nil, nil, nil, g)

	meta, err := kgSvc.Metadata()
	if err != nil {
		t.Fatalf("failed to get metadata: %v", err)
	}
	if meta.DefaultBranch() != "" {
		t.Errorf("expected empty default branch when metadata profile unavailable, got %q", meta.DefaultBranch())
	}
	if meta.CurrentBranch() != "" {
		t.Errorf("expected empty current branch when metadata profile unavailable, got %q", meta.CurrentBranch())
	}
}

// 14. Fuzzy Search Domain Validation
func TestSearch_FuzzyDomainsAndValidation(t *testing.T) {
	svc, _ := setupTestRepository(t)
	searchEng := svc.Search()
	opts := query.DefaultSearchOptions()

	// Valid domains
	validDomains := []query.SearchDomain{
		query.DomainSymbol,
		query.DomainFile,
		query.DomainPackage,
		query.DomainDocumentation,
		query.DomainConfiguration,
		query.DomainAll,
	}

	for _, d := range validDomains {
		_, err := searchEng.FuzzySearch("math", d, opts)
		if err != nil {
			t.Errorf("expected valid domain %v to succeed, got: %v", d, err)
		}
	}

	// Invalid domain
	_, err := searchEng.FuzzySearch("math", query.SearchDomain("INVALID_DOMAIN"), opts)
	if err == nil || !query.IsCategory(err, query.ErrCatInvalidInput) {
		t.Errorf("expected INVALID_INPUT error for invalid fuzzy domain, got: %v", err)
	}
}

// 15. Security Secrets Adversarial Testing
func TestSearch_SecuritySecretsAdversarial(t *testing.T) {
	svc, _ := setupTestRepository(t)
	searchEng := svc.Search()
	opts := query.DefaultSearchOptions()

	secretQueries := []string{"secret", "api_key", "password", "token", "credential", "auth"}
	for _, q := range secretQueries {
		res, err := searchEng.SearchConfiguration(q, opts)
		if err != nil {
			t.Fatalf("SearchConfiguration(%q) failed: %v", q, err)
		}
		for _, item := range res {
			if strings.Contains(strings.ToLower(item.Path()), "secret") {
				if item.Snippet() != "***MASKED_CONFIG***" {
					t.Errorf("secret config snippet was not masked for %s: %s", item.Path(), item.Snippet())
				}
			}
		}
	}
}

// Benchmarks
func BenchmarkSymbolAPI_FindSymbol(b *testing.B) {
	t := &testing.T{}
	svc, _ := setupTestRepository(t)
	syms, _ := svc.Symbols().ListSymbols(query.ScopeRepository, "")
	if len(syms) == 0 {
		b.Skip("no symbols found")
	}
	targetID := syms[0].ID()

	for b.Loop() {
		_, _ = svc.Symbols().FindSymbol(targetID)
	}
}

func BenchmarkSymbolAPI_ListSymbols(b *testing.B) {
	t := &testing.T{}
	svc, _ := setupTestRepository(t)

	for b.Loop() {
		_, _ = svc.Symbols().ListSymbols(query.ScopeRepository, "")
	}
}

func BenchmarkGraphAPI_Traverse(b *testing.B) {
	t := &testing.T{}
	svc, _ := setupTestRepository(t)

	for b.Loop() {
		_, _ = svc.Graph().Traverse("doc:README.md", graph.DirBoth, 2)
	}
}

func BenchmarkGraphAPI_LookupDependencies(b *testing.B) {
	t := &testing.T{}
	svc, _ := setupTestRepository(t)

	for b.Loop() {
		_, _ = svc.Graph().LookupDependencies("")
	}
}

func BenchmarkGraphAPI_LookupCallGraph(b *testing.B) {
	t := &testing.T{}
	svc, _ := setupTestRepository(t)

	for b.Loop() {
		_, _ = svc.Graph().LookupCallGraph("sym:main.go:main:main:6", query.CallDirectionBoth)
	}
}

func BenchmarkSearchEngine_Search(b *testing.B) {
	t := &testing.T{}
	svc, _ := setupTestRepository(t)
	opts := query.DefaultSearchOptions()

	for b.Loop() {
		_, _ = svc.Search().Search("math", query.DomainAll, opts)
	}
}

func BenchmarkSearchEngine_FuzzySearch(b *testing.B) {
	t := &testing.T{}
	svc, _ := setupTestRepository(t)
	opts := query.DefaultSearchOptions()

	for b.Loop() {
		_, _ = svc.Search().FuzzySearch("comput", query.DomainSymbol, opts)
	}
}
