package knowledgegraph_test

import (
	"sync"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/crossrepo"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/knowledgegraph"
	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/metadata"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
)

func createTestSymbol(id, name, pkgPath, filePath string, kind symbol.SymbolKind, sig string, pos *symbol.SourcePosition, isExported bool) *symbol.Symbol {
	return symbol.NewSymbol(
		id,
		kind,
		name,
		"pkg",
		pkgPath,
		filePath,
		"",
		isExported,
		sig,
		"",
		false,
		nil,
		nil,
		pos,
		nil,
	)
}

func setupTestRepository() knowledgegraph.GraphBuildParams {
	pos1 := symbol.NewSourcePosition("internal/service/user.go", 10, 1, 100)
	pos2 := symbol.NewSourcePosition("internal/storage/db.go", 20, 1, 200)

	sym1 := createTestSymbol("sym:service.CreateUser", "CreateUser", "internal/service", "internal/service/user.go", symbol.SymbolKindFunction, "func CreateUser()", pos1, true)
	sym2 := createTestSymbol("sym:storage.SaveRecord", "SaveRecord", "internal/storage", "internal/storage/db.go", symbol.SymbolKindFunction, "func SaveRecord()", pos2, true)

	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{sym1, sym2})

	ref1 := xref.NewReference("sym:service.CreateUser", "sym:storage.SaveRecord", xref.RefFunction, "internal/service/user.go", pos1, xref.StateResolved, "call")
	xrefDB := xref.NewReferenceDatabase([]*xref.Reference{ref1})
	xrefModel := xref.NewXRefModel("repo", xrefDB, nil, nil, nil, nil, nil)

	file1 := discovery.NewFileEntry("internal/service/user.go", "d:/limoxel/internal/service/user.go", false, 1500, time.Now(), ".go", nil, false, false, false)
	file2 := discovery.NewFileEntry("internal/storage/db.go", "d:/limoxel/internal/storage/db.go", false, 2000, time.Now(), ".go", nil, false, false, false)
	fileDoc := discovery.NewFileEntry("docs/architecture.md", "d:/limoxel/docs/architecture.md", false, 5000, time.Now(), ".md", nil, false, false, false)
	fileConf := discovery.NewFileEntry("config.yaml", "d:/limoxel/config.yaml", false, 500, time.Now(), ".yaml", nil, false, false, false)
	discResult := discovery.NewResult(nil, "d:/limoxel", []*discovery.FileEntry{file1, file2, fileDoc, fileConf}, nil, nil, nil, nil)

	node1 := dependency.NewGraphNode("internal/service", "service", true, dependency.EcosystemGo)
	node2 := dependency.NewGraphNode("internal/storage", "storage", true, dependency.EcosystemGo)
	depEdge := dependency.NewGraphEdge("internal/service", "internal/storage", dependency.DependencyDirect)
	depGraph := dependency.NewDependencyGraph([]*dependency.GraphNode{node1, node2}, []*dependency.GraphEdge{depEdge})
	depModel := dependency.NewDependencyModel("d:/limoxel", nil, depGraph, nil, nil, nil, 2, nil)

	metaProfile := metadata.NewProfile("Limoxel", "owner", "d:/limoxel", true, "main", "main", nil, nil, nil, nil, nil, time.Now(), 0, 4, 2, 9000, nil, nil, nil)
	comm := crossrepo.NewPackageCommunication("internal/service", "internal/storage", crossrepo.PkgCommCall, []string{"sym:service.CreateUser"}, []string{"call1"}, "outbound")
	crossModel := crossrepo.NewCrossRepoModel(nil, nil, nil, nil, []*crossrepo.PackageCommunication{comm}, nil, nil, nil, nil, nil, nil, nil, nil, nil, time.Time{})

	return knowledgegraph.GraphBuildParams{
		RootPath:        "d:/limoxel",
		DiscoveryResult: discResult,
		SymbolDB:        symDB,
		XRefModel:       xrefModel,
		DependencyModel: depModel,
		MetadataProfile: metaProfile,
		CrossRepoModel:  crossModel,
	}
}

func TestKnowledgeGraphConstruction(t *testing.T) {
	params := setupTestRepository()
	engine := knowledgegraph.New()

	model, err := engine.Build(params)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if model == nil {
		t.Fatal("Expected non-nil model")
	}

	if model.TotalEntities() == 0 {
		t.Fatal("Expected non-zero entities")
	}
	if model.TotalRelationships() == 0 {
		t.Fatal("Expected non-zero relationships")
	}

	// Verify Repository Node
	repoEnt := model.EntityByID("repository:root")
	if repoEnt == nil {
		t.Fatal("Expected repository:root node")
	}
	if repoEnt.Type() != knowledgegraph.EntityRepository {
		t.Fatalf("Unexpected repository type: %s", repoEnt.Type())
	}

	// Verify Package Nodes
	pkgEnt := model.EntityByID("package:internal/service")
	if pkgEnt == nil {
		t.Fatal("Expected package:internal/service node")
	}

	// Verify Symbol Nodes
	symEnt := model.EntityByID("symbol:sym:service.CreateUser")
	if symEnt == nil {
		t.Fatal("Expected symbol:sym:service.CreateUser node")
	}

	// Verify Call Relationship
	calls := model.OutboundRelationships("symbol:sym:service.CreateUser")
	foundCall := false
	for _, c := range calls {
		if c.Kind() == knowledgegraph.RelCalls && c.TargetID() == "symbol:sym:storage.SaveRecord" {
			foundCall = true
			break
		}
	}
	if !foundCall {
		t.Fatal("Expected call relationship between CreateUser and SaveRecord")
	}
}

func TestKnowledgeEnrichment(t *testing.T) {
	params := setupTestRepository()
	engine := knowledgegraph.New()

	model, err := engine.Build(params)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify Architectural Components Enrichment
	archComps := model.EntitiesByType(knowledgegraph.EntityArchComponent)
	if len(archComps) == 0 {
		t.Fatal("Expected enriched architectural components")
	}

	// Verify Documentation Entities
	docs := model.EntitiesByType(knowledgegraph.EntityDocumentation)
	if len(docs) == 0 {
		t.Fatal("Expected enriched documentation entities")
	}

	// Verify Configuration Entities
	confs := model.EntitiesByType(knowledgegraph.EntityConfiguration)
	if len(confs) == 0 {
		t.Fatal("Expected enriched configuration entities")
	}
}

func TestGraphReasoningAndInference(t *testing.T) {
	params := setupTestRepository()
	engine := knowledgegraph.New()

	model, err := engine.Build(params)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	chains := engine.Reasoning().InferTransitiveDependencies(model, 3)
	if len(chains) == 0 {
		t.Fatal("Expected inferred dependency chains")
	}
	if chains[0].SourceID != "package:internal/service" || chains[0].TargetID != "package:internal/storage" {
		t.Fatalf("Unexpected chain: %+v", chains[0])
	}

	// Verify Ownership Hierarchy Inference
	ownership := engine.Reasoning().InferOwnershipHierarchy(model, "symbol:sym:service.CreateUser")
	if ownership == nil {
		t.Fatal("Expected non-nil ownership hierarchy")
	}
	if len(ownership.OwnerChain) == 0 {
		t.Fatal("Expected non-empty ownership chain")
	}
}

func TestContextGeneration(t *testing.T) {
	params := setupTestRepository()
	engine := knowledgegraph.New()

	model, err := engine.Build(params)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// 1. Repository Context
	repoCtx := engine.Context().GenerateRepositoryContext(model)
	if repoCtx == nil || repoCtx.TotalFiles == 0 {
		t.Fatalf("Invalid repository context: %+v", repoCtx)
	}

	// 2. Package Context
	pkgCtx := engine.Context().GeneratePackageContext(model, "internal/service")
	if pkgCtx == nil || len(pkgCtx.ContainedFiles) == 0 {
		t.Fatalf("Invalid package context: %+v", pkgCtx)
	}

	// 3. Symbol Context
	symCtx := engine.Context().GenerateSymbolContext(model, "sym:service.CreateUser")
	if symCtx == nil || len(symCtx.Callees) == 0 {
		t.Fatalf("Invalid symbol context: %+v", symCtx)
	}

	// 4. Module Context
	modCtx := engine.Context().GenerateModuleContext(model, "internal/service")
	if modCtx == nil || len(modCtx.ContainedFiles) == 0 {
		t.Fatalf("Invalid module context: %+v", modCtx)
	}

	// 5. Architecture Context
	archCtx := engine.Context().GenerateArchitectureContext(model)
	if archCtx == nil || len(archCtx.Components) == 0 {
		t.Fatalf("Invalid architecture context: %+v", archCtx)
	}
}

func TestGraphQueryAndTraversal(t *testing.T) {
	params := setupTestRepository()
	engine := knowledgegraph.New()

	model, err := engine.Build(params)
	if err != nil || model == nil {
		t.Fatalf("Build failed: %v", err)
	}

	query := engine.Query()

	// 1. Neighbors query
	neighbors := query.Neighbors("symbol:sym:service.CreateUser", knowledgegraph.DirOutbound, knowledgegraph.RelCalls)
	if len(neighbors) == 0 {
		t.Fatal("Expected outgoing call neighbors")
	}
	if neighbors[0].ID() != "symbol:sym:storage.SaveRecord" {
		t.Fatalf("Unexpected neighbor: %s", neighbors[0].ID())
	}

	// 2. Path query
	paths := query.FindPaths("package:internal/service", "package:internal/storage", 3)
	if len(paths) == 0 {
		t.Fatal("Expected path between service and storage packages")
	}

	// 3. Subgraph extraction
	subgraph, err := query.ExtractSubgraph("package:internal/service", 2)
	if err != nil || subgraph == nil {
		t.Fatalf("Subgraph extraction failed: %v", err)
	}
	if subgraph.TotalEntities() == 0 {
		t.Fatal("Expected non-empty subgraph")
	}

	// 4. Search query
	matches := query.SearchEntities("CreateUser", knowledgegraph.EntitySymbol)
	if len(matches) == 0 {
		t.Fatal("Expected search matches for CreateUser")
	}
}

func TestCycleSafety(t *testing.T) {
	// Construct cyclic graph: A -> B -> A
	n1 := knowledgegraph.NewGraphEntity("package:pkg/a", knowledgegraph.EntityPackage, "a", "pkg/a", "", nil, nil, "test")
	n2 := knowledgegraph.NewGraphEntity("package:pkg/b", knowledgegraph.EntityPackage, "b", "pkg/b", "", nil, nil, "test")

	r1 := knowledgegraph.NewGraphRelationship("package:pkg/a", "package:pkg/b", knowledgegraph.RelDependsOn, "a->b", "test", 1.0, nil)
	r2 := knowledgegraph.NewGraphRelationship("package:pkg/b", "package:pkg/a", knowledgegraph.RelDependsOn, "b->a", "test", 1.0, nil)

	model := knowledgegraph.NewKnowledgeGraphModel("/root", []*knowledgegraph.GraphEntity{n1, n2}, []*knowledgegraph.GraphRelationship{r1, r2}, nil, time.Now().UTC())
	query := knowledgegraph.NewGraphQueryEngine(model)

	paths := query.FindPaths("package:pkg/a", "package:pkg/b", 5)
	if len(paths) == 0 {
		t.Fatal("Expected paths in cyclic graph without infinite recursion")
	}

	reasoning := knowledgegraph.NewGraphReasoningEngine()
	chains := reasoning.InferTransitiveDependencies(model, 5)
	if len(chains) == 0 {
		t.Fatal("Expected transitive dependency chains in cyclic graph")
	}
}

func TestDeterminismAndOrdering(t *testing.T) {
	params := setupTestRepository()

	engine1 := knowledgegraph.New()
	m1, err := engine1.Build(params)
	if err != nil || m1 == nil {
		t.Fatalf("Build 1 failed: %v", err)
	}

	engine2 := knowledgegraph.New()
	m2, err := engine2.Build(params)
	if err != nil || m2 == nil {
		t.Fatalf("Build 2 failed: %v", err)
	}

	if m1.TotalEntities() != m2.TotalEntities() {
		t.Fatalf("Entity count mismatch: %d vs %d", m1.TotalEntities(), m2.TotalEntities())
	}
	if m1.TotalRelationships() != m2.TotalRelationships() {
		t.Fatalf("Relationship count mismatch: %d vs %d", m1.TotalRelationships(), m2.TotalRelationships())
	}

	for i := range m1.Entities() {
		if m1.Entities()[i].ID() != m2.Entities()[i].ID() {
			t.Fatalf("Entity ordering mismatch at index %d: %s vs %s", i, m1.Entities()[i].ID(), m2.Entities()[i].ID())
		}
	}
}

func TestAdversarialEmptyAndMissingData(t *testing.T) {
	emptyParams := knowledgegraph.GraphBuildParams{
		RootPath: "/empty",
	}
	engine := knowledgegraph.New()
	model, err := engine.Build(emptyParams)
	if err != nil || model == nil {
		t.Fatalf("Empty build failed: %v", err)
	}

	// Search in empty model
	matches := engine.Query().SearchEntities("nonexistent", knowledgegraph.EntitySymbol)
	if len(matches) != 0 {
		t.Fatalf("Expected 0 matches in empty model, got %d", len(matches))
	}

	// Missing entity query
	neighbors := engine.Query().Neighbors("missing:id", knowledgegraph.DirOutbound)
	if len(neighbors) != 0 {
		t.Fatalf("Expected 0 neighbors for missing entity, got %d", len(neighbors))
	}
}

func TestDeterminismAcrossConcurrency(t *testing.T) {
	params := setupTestRepository()

	engine := knowledgegraph.New()
	baselineModel, err := engine.Build(params)
	if err != nil || baselineModel == nil {
		t.Fatalf("Baseline build failed: %v", err)
	}

	baselineEnts := baselineModel.TotalEntities()
	baselineRels := baselineModel.TotalRelationships()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e := knowledgegraph.New()
			m, err := e.Build(params)
			if err != nil || m == nil {
				t.Errorf("Concurrent build failed: %v", err)
				return
			}
			if m.TotalEntities() != baselineEnts {
				t.Errorf("Entity count mismatch under concurrency: %d vs %d", m.TotalEntities(), baselineEnts)
			}
			if m.TotalRelationships() != baselineRels {
				t.Errorf("Relationship count mismatch under concurrency: %d vs %d", m.TotalRelationships(), baselineRels)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkKnowledgeGraphBuildAndQuery(b *testing.B) {
	params := setupTestRepository()
	engine := knowledgegraph.New()
	b.ReportAllocs()

	for b.Loop() {
		model, err := engine.Build(params)
		if err != nil || model == nil {
			b.Fatalf("Build failed: %v", err)
		}
		_ = engine.Query().Neighbors("symbol:sym:service.CreateUser", knowledgegraph.DirOutbound)
	}
}
