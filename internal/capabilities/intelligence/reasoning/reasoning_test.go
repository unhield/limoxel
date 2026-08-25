package reasoning_test

import (
	"sync"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/knowledgegraph"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/reasoning"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
)

// Helper to create a rich test KnowledgeGraphModel
func setupTestKnowledgeGraph() *knowledgegraph.KnowledgeGraphModel {
	pos := symbol.NewSourcePosition("internal/service/user.go", 10, 1, 100)

	repo := knowledgegraph.NewGraphEntity("repository:root", knowledgegraph.EntityRepository, "Limoxel", "", "", nil, nil, "metadata")
	compService := knowledgegraph.NewGraphEntity("arch_component:internal/service", knowledgegraph.EntityArchComponent, "internal/service", "internal/service", "", nil, nil, "arch")
	compStorage := knowledgegraph.NewGraphEntity("arch_component:internal/storage", knowledgegraph.EntityArchComponent, "internal/storage", "internal/storage", "", nil, nil, "arch")

	pkgService := knowledgegraph.NewGraphEntity("package:internal/service", knowledgegraph.EntityPackage, "service", "internal/service", "", nil, nil, "discovery")
	pkgStorage := knowledgegraph.NewGraphEntity("package:internal/storage", knowledgegraph.EntityPackage, "storage", "internal/storage", "", nil, nil, "discovery")
	pkgAuth := knowledgegraph.NewGraphEntity("package:internal/auth", knowledgegraph.EntityPackage, "auth", "internal/auth", "", nil, nil, "discovery")

	fileService := knowledgegraph.NewGraphEntity("file:internal/service/user.go", knowledgegraph.EntityFile, "user.go", "internal/service", "internal/service/user.go", pos, nil, "discovery")
	fileStorage := knowledgegraph.NewGraphEntity("file:internal/storage/db.go", knowledgegraph.EntityFile, "db.go", "internal/storage", "internal/storage/db.go", pos, nil, "discovery")

	symCreateUser := knowledgegraph.NewGraphEntity("symbol:sym:service.CreateUser", knowledgegraph.EntitySymbol, "CreateUser", "internal/service", "internal/service/user.go", pos, map[string]string{"signature": "func CreateUser(name string) error"}, "symbol_db")
	symSaveRecord := knowledgegraph.NewGraphEntity("symbol:sym:storage.SaveRecord", knowledgegraph.EntitySymbol, "SaveRecord", "internal/storage", "internal/storage/db.go", pos, map[string]string{"signature": "func SaveRecord(data string) error"}, "symbol_db")
	symUnused := knowledgegraph.NewGraphEntity("symbol:sym:service.UnusedHelper", knowledgegraph.EntitySymbol, "UnusedHelper", "internal/service", "internal/service/user.go", pos, nil, "symbol_db")

	docSym := knowledgegraph.NewGraphEntity("documentation:symdoc:sym:service.CreateUser", knowledgegraph.EntityDocumentation, "CreateUser_doc", "internal/service", "internal/service/user.go", pos, map[string]string{"content": "CreateUser creates a new user."}, "symbol_doc")

	entities := []*knowledgegraph.GraphEntity{
		repo, compService, compStorage,
		pkgService, pkgStorage, pkgAuth,
		fileService, fileStorage,
		symCreateUser, symSaveRecord, symUnused,
		docSym,
	}

	rels := []*knowledgegraph.GraphRelationship{
		knowledgegraph.NewGraphRelationship(repo.ID(), compService.ID(), knowledgegraph.RelOwns, "owns", "arch", 1.0, nil),
		knowledgegraph.NewGraphRelationship(repo.ID(), compStorage.ID(), knowledgegraph.RelOwns, "owns", "arch", 1.0, nil),
		knowledgegraph.NewGraphRelationship(compService.ID(), pkgService.ID(), knowledgegraph.RelOwns, "contains", "arch", 1.0, nil),
		knowledgegraph.NewGraphRelationship(compStorage.ID(), pkgStorage.ID(), knowledgegraph.RelOwns, "contains", "arch", 1.0, nil),
		knowledgegraph.NewGraphRelationship(pkgService.ID(), fileService.ID(), knowledgegraph.RelOwns, "contains", "discovery", 1.0, nil),
		knowledgegraph.NewGraphRelationship(pkgStorage.ID(), fileStorage.ID(), knowledgegraph.RelOwns, "contains", "discovery", 1.0, nil),

		knowledgegraph.NewGraphRelationship(pkgService.ID(), pkgStorage.ID(), knowledgegraph.RelDependsOn, "imports storage", "dep", 1.0, nil),
		knowledgegraph.NewGraphRelationship(symCreateUser.ID(), symSaveRecord.ID(), knowledgegraph.RelCalls, "CreateUser calls SaveRecord", "xref", 1.0, nil),
		knowledgegraph.NewGraphRelationship(docSym.ID(), symCreateUser.ID(), knowledgegraph.RelDocuments, "documents CreateUser", "doc", 1.0, nil),
	}

	return knowledgegraph.NewKnowledgeGraphModel("d:/limoxel", entities, rels, nil, time.Now().UTC())
}

func TestImpactAnalysis(t *testing.T) {
	model := setupTestKnowledgeGraph()
	analyzer := reasoning.NewImpactAnalyzer()

	// 1. Analyze impact of modifying SaveRecord (inbound caller is CreateUser)
	res, err := analyzer.Analyze(model, "symbol:sym:storage.SaveRecord")
	if err != nil {
		t.Fatalf("Impact analysis failed: %v", err)
	}

	if len(res.AffectedSymbols) == 0 {
		t.Fatal("Expected affected symbols for SaveRecord impact")
	}

	foundCreateUser := false
	for _, s := range res.AffectedSymbols {
		if s.EntityID == "symbol:sym:service.CreateUser" {
			foundCreateUser = true
			if !s.Direct {
				t.Error("Expected CreateUser to be a direct impact")
			}
		}
	}
	if !foundCreateUser {
		t.Errorf("CreateUser not found in affected symbols: %+v", res.AffectedSymbols)
	}

	if len(res.ImpactPaths) == 0 {
		t.Fatal("Expected non-empty impact paths")
	}
}

func TestRefactoringSafetyRename(t *testing.T) {
	model := setupTestKnowledgeGraph()
	advisor := reasoning.NewRefactoringAdvisor()

	// 1. Safe rename of UnusedHelper
	res, err := advisor.AnalyzeRename(model, "symbol:sym:service.UnusedHelper", "NewUnusedHelper")
	if err != nil {
		t.Fatalf("AnalyzeRename failed: %v", err)
	}
	if !res.Safe {
		t.Errorf("Expected rename of unused helper to be safe, got: %+v", res)
	}

	// 2. Unsafe rename with callers
	res2, err := advisor.AnalyzeRename(model, "symbol:sym:storage.SaveRecord", "SaveData")
	if err != nil {
		t.Fatalf("AnalyzeRename failed: %v", err)
	}
	if len(res2.UnresolvedReferences) == 0 {
		t.Errorf("Expected unresolved references for SaveRecord rename")
	}
}

func TestRefactoringSafetyMove(t *testing.T) {
	model := setupTestKnowledgeGraph()
	advisor := reasoning.NewRefactoringAdvisor()

	// 1. Move to existing package
	res, err := advisor.AnalyzeMove(model, "symbol:sym:service.UnusedHelper", "internal/auth")
	if err != nil {
		t.Fatalf("AnalyzeMove failed: %v", err)
	}
	if !res.Safe {
		t.Errorf("Expected move to auth to be safe: %+v", res)
	}

	// 2. Move to non-existent package
	res2, err := advisor.AnalyzeMove(model, "symbol:sym:service.UnusedHelper", "internal/nonexistent")
	if err != nil {
		t.Fatalf("AnalyzeMove failed: %v", err)
	}
	if res2.Safe || len(res2.BlockingReasons) == 0 {
		t.Errorf("Expected move to non-existent package to be blocked")
	}
}

func TestRefactoringSafetyDeletion(t *testing.T) {
	model := setupTestKnowledgeGraph()
	advisor := reasoning.NewRefactoringAdvisor()

	// 1. Safe deletion of unused symbol
	res, err := advisor.AnalyzeDeletion(model, "symbol:sym:service.UnusedHelper")
	if err != nil {
		t.Fatalf("AnalyzeDeletion failed: %v", err)
	}
	if !res.Safe {
		t.Errorf("Expected deletion of unused symbol to be safe: %+v", res)
	}

	// 2. Blocked deletion of referenced symbol
	res2, err := advisor.AnalyzeDeletion(model, "symbol:sym:storage.SaveRecord")
	if err != nil {
		t.Fatalf("AnalyzeDeletion failed: %v", err)
	}
	if res2.Safe || len(res2.BlockingReasons) == 0 {
		t.Errorf("Expected deletion of referenced symbol to be blocked")
	}
}

func TestRefactoringRiskAssessment(t *testing.T) {
	model := setupTestKnowledgeGraph()
	advisor := reasoning.NewRefactoringAdvisor()

	risk, err := advisor.AssessRisk(model, "symbol:sym:storage.SaveRecord")
	if err != nil {
		t.Fatalf("AssessRisk failed: %v", err)
	}
	if risk.DirectReferences == 0 {
		t.Errorf("Expected direct references for SaveRecord risk")
	}
	if risk.Risk == "" {
		t.Errorf("Expected non-empty risk tier")
	}
}

func TestBreakingChangeDetection(t *testing.T) {
	baseModel := setupTestKnowledgeGraph()

	// Create target model with removed symbol and changed signature
	pos := symbol.NewSourcePosition("internal/service/user.go", 10, 1, 100)
	symCreateUserModified := knowledgegraph.NewGraphEntity("symbol:sym:service.CreateUser", knowledgegraph.EntitySymbol, "CreateUser", "internal/service", "internal/service/user.go", pos, map[string]string{"signature": "func CreateUser(name string, age int) error"}, "symbol_db")

	targetEntities := []*knowledgegraph.GraphEntity{
		symCreateUserModified,
		// SaveRecord was removed!
	}

	targetModel := knowledgegraph.NewKnowledgeGraphModel("d:/limoxel", targetEntities, nil, nil, time.Now().UTC())

	analyzer := reasoning.NewBreakingChangeAnalyzer()
	report, err := analyzer.AnalyzeBreakingChanges(baseModel, targetModel)
	if err != nil {
		t.Fatalf("AnalyzeBreakingChanges failed: %v", err)
	}

	if !report.HasBreakingChanges {
		t.Fatal("Expected breaking changes to be reported")
	}
	if len(report.Findings) < 2 {
		t.Fatalf("Expected at least 2 findings (removal + signature change), got %d", len(report.Findings))
	}
}

func TestRecommendationEngine(t *testing.T) {
	// Build graph with a circular dependency to test recommendation engine
	pkgA := knowledgegraph.NewGraphEntity("package:internal/a", knowledgegraph.EntityPackage, "a", "internal/a", "", nil, nil, "dep")
	pkgB := knowledgegraph.NewGraphEntity("package:internal/b", knowledgegraph.EntityPackage, "b", "internal/b", "", nil, nil, "dep")

	relAB := knowledgegraph.NewGraphRelationship(pkgA.ID(), pkgB.ID(), knowledgegraph.RelDependsOn, "a imports b", "dep", 1.0, nil)
	relBA := knowledgegraph.NewGraphRelationship(pkgB.ID(), pkgA.ID(), knowledgegraph.RelDependsOn, "b imports a", "dep", 1.0, nil)

	model := knowledgegraph.NewKnowledgeGraphModel("d:/limoxel", []*knowledgegraph.GraphEntity{pkgA, pkgB}, []*knowledgegraph.GraphRelationship{relAB, relBA}, nil, time.Now().UTC())

	engine := reasoning.NewRecommendationEngine()
	recs := engine.GenerateRecommendations(model)

	if len(recs) == 0 {
		t.Fatal("Expected circular dependency recommendation")
	}
	if recs[0].Category != reasoning.RecDependency || recs[0].Priority != reasoning.PriorityCritical {
		t.Errorf("Unexpected top recommendation: %+v", recs[0])
	}
}

func TestReasoningEngineCoordinator(t *testing.T) {
	model := setupTestKnowledgeGraph()
	engine := reasoning.New()

	report, err := engine.Reason(reasoning.ReasoningParams{
		Model:          model,
		TargetEntityID: "symbol:sym:storage.SaveRecord",
	})
	if err != nil {
		t.Fatalf("Reason failed: %v", err)
	}

	if report.ImpactResult == nil {
		t.Fatal("Expected non-nil ImpactResult in report")
	}
	if report.RiskAssessment == nil {
		t.Fatal("Expected non-nil RiskAssessment in report")
	}
	if len(report.ReasoningChains) == 0 {
		t.Fatal("Expected reasoning chains in report")
	}
}

func TestCycleSafety(t *testing.T) {
	// A -> B -> A
	symA := knowledgegraph.NewGraphEntity("symbol:sym:A", knowledgegraph.EntitySymbol, "A", "internal/a", "a.go", nil, nil, "test")
	symB := knowledgegraph.NewGraphEntity("symbol:sym:B", knowledgegraph.EntitySymbol, "B", "internal/b", "b.go", nil, nil, "test")

	relAB := knowledgegraph.NewGraphRelationship(symA.ID(), symB.ID(), knowledgegraph.RelCalls, "A calls B", "test", 1.0, nil)
	relBA := knowledgegraph.NewGraphRelationship(symB.ID(), symA.ID(), knowledgegraph.RelCalls, "B calls A", "test", 1.0, nil)

	model := knowledgegraph.NewKnowledgeGraphModel("d:/limoxel", []*knowledgegraph.GraphEntity{symA, symB}, []*knowledgegraph.GraphRelationship{relAB, relBA}, nil, time.Now().UTC())

	analyzer := reasoning.NewImpactAnalyzer()
	res, err := analyzer.Analyze(model, symA.ID())
	if err != nil {
		t.Fatalf("Analyze failed on cyclic graph: %v", err)
	}

	if len(res.ImpactPaths) == 0 {
		t.Fatal("Expected paths in cyclic graph")
	}
}

func TestDeterminismAndOrdering(t *testing.T) {
	model := setupTestKnowledgeGraph()
	engine := reasoning.New()

	report1, err := engine.Reason(reasoning.ReasoningParams{
		Model:          model,
		TargetEntityID: "symbol:sym:storage.SaveRecord",
	})
	if err != nil {
		t.Fatalf("Reason run 1 failed: %v", err)
	}

	report2, err := engine.Reason(reasoning.ReasoningParams{
		Model:          model,
		TargetEntityID: "symbol:sym:storage.SaveRecord",
	})
	if err != nil {
		t.Fatalf("Reason run 2 failed: %v", err)
	}

	if len(report1.ImpactResult.AffectedSymbols) != len(report2.ImpactResult.AffectedSymbols) {
		t.Fatal("Nondeterministic affected symbols count")
	}
	for i := range report1.ImpactResult.AffectedSymbols {
		if report1.ImpactResult.AffectedSymbols[i].EntityID != report2.ImpactResult.AffectedSymbols[i].EntityID {
			t.Fatalf("Nondeterministic entity ID at %d: %s vs %s", i, report1.ImpactResult.AffectedSymbols[i].EntityID, report2.ImpactResult.AffectedSymbols[i].EntityID)
		}
	}
}

func TestAdversarialEmptyAndMissingData(t *testing.T) {
	engine := reasoning.New()

	// 1. Nil model
	_, err := engine.Reason(reasoning.ReasoningParams{Model: nil})
	if err != reasoning.ErrNilGraphModel {
		t.Errorf("Expected ErrNilGraphModel, got: %v", err)
	}

	// 2. Missing target
	model := setupTestKnowledgeGraph()
	_, err = engine.Impact().Analyze(model, "nonexistent_target_id")
	if err == nil {
		t.Error("Expected error on non-existent target")
	}
}

func TestDeterminismAcrossConcurrency(t *testing.T) {
	model := setupTestKnowledgeGraph()
	engine := reasoning.New()

	const routines = 10
	var wg sync.WaitGroup
	results := make([]int, routines)

	for i := 0; i < routines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			rep, err := engine.Reason(reasoning.ReasoningParams{
				Model:          model,
				TargetEntityID: "symbol:sym:storage.SaveRecord",
			})
			if err != nil {
				t.Errorf("Goroutine %d failed: %v", idx, err)
				return
			}
			results[idx] = len(rep.ImpactResult.AffectedSymbols)
		}(i)
	}
	wg.Wait()

	for i := 1; i < routines; i++ {
		if results[i] != results[0] {
			t.Fatalf("Concurrency result mismatch at %d: %d vs %d", i, results[i], results[0])
		}
	}
}

func BenchmarkDeterministicReasoning(b *testing.B) {
	model := setupTestKnowledgeGraph()
	engine := reasoning.New()

	b.ResetTimer()
	for b.Loop() {
		_, _ = engine.Reason(reasoning.ReasoningParams{
			Model:          model,
			TargetEntityID: "symbol:sym:storage.SaveRecord",
		})
	}
}
