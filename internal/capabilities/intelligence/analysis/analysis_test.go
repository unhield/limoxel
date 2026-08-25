package analysis_test

import (
	"sync"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/analysis"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/crossrepo"
	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/language"
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

func TestCodeQualityAnalyzer(t *testing.T) {
	pos1 := symbol.NewSourcePosition("internal/pkg/util.go", 10, 1, 100)
	pos2 := symbol.NewSourcePosition("internal/pkg/service.go", 20, 1, 200)

	// 1. Dead Code Symbol (private, unreferenced)
	symDead := createTestSymbol("sym:internal/pkg.deadFunc", "deadFunc", "internal/pkg", "internal/pkg/util.go", symbol.SymbolKindFunction, "func deadFunc()", pos1, false)
	// 2. Used Symbol
	symUsed := createTestSymbol("sym:internal/pkg.usedFunc", "usedFunc", "internal/pkg", "internal/pkg/service.go", symbol.SymbolKindFunction, "func usedFunc()", pos2, true)
	// 3. Duplicate Function Pair
	dupSig := "func duplicateLogicMethod(x int, y int) (int, error) {\n\treturn x + y, nil\n}"
	symDup1 := createTestSymbol("sym:pkg/a.Compute", "Compute", "pkg/a", "pkg/a/a.go", symbol.SymbolKindFunction, dupSig, pos1, true)
	symDup2 := createTestSymbol("sym:pkg/b.Calculate", "Calculate", "pkg/b", "pkg/b/b.go", symbol.SymbolKindFunction, dupSig, pos2, true)

	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{symDead, symUsed, symDup1, symDup2})

	// Add reference for symUsed only
	ref1 := xref.NewReference("sym:caller", "sym:internal/pkg.usedFunc", xref.RefFunction, "internal/pkg/service.go", pos2, xref.StateResolved, "call")
	// Add unused package import ref
	refPkg := xref.NewReference("sym:caller", "github.com/unused/pkg", xref.RefUnknown, "internal/pkg/util.go", pos1, xref.StateResolved, "import")
	xrefDB := xref.NewReferenceDatabase([]*xref.Reference{ref1, refPkg})
	xrefModel := xref.NewXRefModel("repo", xrefDB, nil, nil, nil, nil, nil)

	analyzer := analysis.NewCodeQualityAnalyzer(symDB, xrefModel, nil, nil, nil)
	result := analyzer.Analyze()

	if result == nil {
		t.Fatal("Expected non-nil AnalyzerResult")
	}

	// Verify Dead Code Finding
	deadRes := result.RuleResult(analysis.RuleDeadCode)
	if deadRes == nil || deadRes.FindingCount() == 0 {
		t.Fatalf("Expected dead code findings, got %+v", deadRes)
	}
	if deadRes.Findings()[0].SymbolID() != "sym:internal/pkg.deadFunc" {
		t.Fatalf("Unexpected dead code symbol ID: %s", deadRes.Findings()[0].SymbolID())
	}

	// Verify Unused Imports Finding
	importRes := result.RuleResult(analysis.RuleUnusedImports)
	if importRes == nil || importRes.FindingCount() == 0 {
		t.Fatalf("Expected unused import findings, got %+v", importRes)
	}

	// Verify Duplicate Logic Finding
	dupRes := result.RuleResult(analysis.RuleDuplicateLogic)
	if dupRes == nil || dupRes.FindingCount() == 0 {
		t.Fatalf("Expected duplicate logic findings, got %+v", dupRes)
	}
}

func TestDependencyAnalyzer(t *testing.T) {
	// 1. Dependency Cycle: pkg/a -> pkg/b -> pkg/a
	cycles := [][]string{
		{"pkg/a", "pkg/b", "pkg/a"},
	}
	orphans := []string{"pkg/isolated"}
	depModel := dependency.NewDependencyModel("d:/workspace", nil, nil, nil, cycles, orphans, 2, nil)

	// Cross-repo communications showing layer violation: foundation importing cli
	syms10 := []string{"sym1", "sym2", "sym3", "sym4", "sym5", "sym6", "sym7", "sym8", "sym9", "sym10"}
	calls10 := []string{"call1", "call2", "call3", "call4", "call5", "call6", "call7", "call8", "call9", "call10"}
	comm1 := crossrepo.NewPackageCommunication("internal/platform/errors", "internal/cli", crossrepo.PkgCommCall, []string{"Ref1"}, []string{"call1"}, "outbound")
	comm2 := crossrepo.NewPackageCommunication("pkg/serviceA", "pkg/serviceB", crossrepo.PkgCommCall, syms10, calls10, "outbound")
	comm3 := crossrepo.NewPackageCommunication("pkg/serviceB", "pkg/serviceA", crossrepo.PkgCommCall, syms10, calls10, "outbound")
	crossModel := crossrepo.NewCrossRepoModel(nil, nil, nil, nil, []*crossrepo.PackageCommunication{comm1, comm2, comm3}, nil, nil, nil, nil, nil, nil, nil, nil, nil, time.Time{})

	analyzer := analysis.NewDependencyAnalyzer(depModel, crossModel, nil, nil)
	result := analyzer.Analyze()

	if result == nil {
		t.Fatal("Expected non-nil AnalyzerResult")
	}

	// 1. Circular Dependencies
	circRes := result.RuleResult(analysis.RuleCircularDependencies)
	if circRes == nil || circRes.FindingCount() == 0 {
		t.Fatalf("Expected circular dependency findings, got %+v", circRes)
	}
	if circRes.Findings()[0].Severity() != analysis.SeverityCritical {
		t.Fatalf("Expected SeverityCritical for cycle, got %s", circRes.Findings()[0].Severity())
	}

	// 2. Layer Violations
	layerRes := result.RuleResult(analysis.RuleLayerViolations)
	if layerRes == nil || layerRes.FindingCount() == 0 {
		t.Fatalf("Expected layer violation findings, got %+v", layerRes)
	}

	// 3. Tight Coupling
	tightRes := result.RuleResult(analysis.RuleTightCoupling)
	if tightRes == nil || tightRes.FindingCount() == 0 {
		t.Fatalf("Expected tight coupling findings, got %+v", tightRes)
	}

	// 4. Orphan Packages
	orphanRes := result.RuleResult(analysis.RuleOrphanPackages)
	if orphanRes == nil || orphanRes.FindingCount() == 0 {
		t.Fatalf("Expected orphan package findings, got %+v", orphanRes)
	}
}

func TestArchitectureAnalyzer(t *testing.T) {
	// 1. Architecture violation: platform imports intelligence
	comm1 := crossrepo.NewPackageCommunication("internal/platform/bootstrap", "internal/capabilities/intelligence/semantic", crossrepo.PkgCommCall, []string{"Ref1"}, nil, "outbound")
	// 2. Module boundary violation: repo1 accessing internal package of repo2
	comm2 := crossrepo.NewPackageCommunication("repo1/pkg/service", "repo2/internal/secrets", crossrepo.PkgCommCall, []string{"Ref2"}, nil, "outbound")

	repo1 := crossrepo.NewWorkspaceRepository("d:/workspace/repo1", "repo1", nil, []string{"repo1/pkg/service"}, nil)
	repo2 := crossrepo.NewWorkspaceRepository("d:/workspace/repo2", "repo2", nil, []string{"repo2/internal/secrets"}, nil)
	ws := crossrepo.NewWorkspaceModel("d:/workspace", []*crossrepo.WorkspaceRepository{repo1, repo2}, nil, nil, nil, nil)
	crossModel := crossrepo.NewCrossRepoModel(nil, nil, nil, nil, []*crossrepo.PackageCommunication{comm1, comm2}, nil, nil, nil, nil, nil, nil, ws, nil, nil, time.Time{})

	analyzer := analysis.NewArchitectureAnalyzer(crossModel, nil, nil)
	result := analyzer.Analyze()

	if result == nil {
		t.Fatal("Expected non-nil AnalyzerResult")
	}

	// 1. Architecture Violations
	archRes := result.RuleResult(analysis.RuleArchitectureViolations)
	if archRes == nil || archRes.FindingCount() == 0 {
		t.Fatalf("Expected architecture violation findings, got %+v", archRes)
	}
	if archRes.Findings()[0].Severity() != analysis.SeverityCritical {
		t.Fatalf("Expected SeverityCritical, got %s", archRes.Findings()[0].Severity())
	}

	// 2. Module Boundaries
	modRes := result.RuleResult(analysis.RuleModuleBoundaries)
	if modRes == nil || modRes.FindingCount() == 0 {
		t.Fatalf("Expected module boundary findings, got %+v", modRes)
	}
}

func TestConfigurationAnalyzer(t *testing.T) {
	entries := []*analysis.RawConfigEntry{
		{FilePath: "config.yaml", Key: "host", Value: "", Line: 1},                      // Invalid empty
		{FilePath: "config.yaml", Key: "port", Value: "8080", Line: 2},                  // Valid
		{FilePath: "config.yaml", Key: "port", Value: "8081", Line: 3},                  // Duplicate & conflict
		{FilePath: "env.yaml", Key: "port", Value: "9000", Line: 1},                     // Conflict with config.yaml
		{FilePath: "config.yaml", Key: "legacy_mode", Value: "true", Line: 4},           // Deprecated
		{FilePath: "config.yaml", Key: "api_key", Value: "secret-token-12345", Line: 5}, // Secret to redact
	}

	analyzer := analysis.NewConfigurationAnalyzer(nil, entries)
	result := analyzer.Analyze()

	if result == nil {
		t.Fatal("Expected non-nil AnalyzerResult")
	}

	// 1. Invalid Config
	invRes := result.RuleResult(analysis.RuleInvalidConfiguration)
	if invRes == nil || invRes.FindingCount() == 0 {
		t.Fatalf("Expected invalid config findings, got %+v", invRes)
	}

	// 2. Duplicate Config
	dupRes := result.RuleResult(analysis.RuleDuplicateConfiguration)
	if dupRes == nil || dupRes.FindingCount() == 0 {
		t.Fatalf("Expected duplicate config findings, got %+v", dupRes)
	}

	// 3. Deprecated Config
	depRes := result.RuleResult(analysis.RuleDeprecatedConfiguration)
	if depRes == nil || depRes.FindingCount() == 0 {
		t.Fatalf("Expected deprecated config findings, got %+v", depRes)
	}

	// 4. Configuration Conflicts & Secret Redaction
	confRes := result.RuleResult(analysis.RuleConfigurationConflicts)
	if confRes == nil || confRes.FindingCount() == 0 {
		t.Fatalf("Expected configuration conflict findings, got %+v", confRes)
	}

	// Verify Secret Redaction
	redacted := analysis.RedactSecret("api_key", "secret-token-12345")
	if redacted != "[REDACTED]" {
		t.Fatalf("Expected secret value to be redacted, got %s", redacted)
	}
}

func TestRepositoryHealthEngine(t *testing.T) {
	findingCrit := analysis.NewFinding("dependency", analysis.RuleCircularDependencies, analysis.CategoryDependency, analysis.SeverityCritical, analysis.ConfidenceDefinite, "Cycle", "desc", "", "", "pkg/a", "", "", nil, "cycle", nil, "", "dep")
	findingHigh := analysis.NewFinding("dependency", analysis.RuleLayerViolations, analysis.CategoryDependency, analysis.SeverityHigh, analysis.ConfidenceDefinite, "Violation", "desc", "", "", "pkg/b", "", "", nil, "layer", nil, "", "dep")
	findingMed := analysis.NewFinding("code_quality", analysis.RuleDeadCode, analysis.CategoryQuality, analysis.SeverityMedium, analysis.ConfidenceDefinite, "Dead", "desc", "", "", "pkg/c", "", "", nil, "dead", nil, "", "sym")

	file1 := discovery.NewFileEntry("pkg/a/a.go", "d:/workspace/pkg/a/a.go", false, 500, time.Now(), ".go", nil, false, false, false)
	fileTest := discovery.NewFileEntry("pkg/a/a_test.go", "d:/workspace/pkg/a/a_test.go", false, 300, time.Now(), ".go", nil, false, false, false)
	discResult := discovery.NewResult(nil, "d:/workspace", []*discovery.FileEntry{file1, fileTest}, nil, nil, nil, nil)
	doc1 := language.NewDocAsset(language.DocReadme, "README.md", "general")
	langModel := language.NewStructureModel("d:/workspace", nil, nil, nil, nil, nil, nil, nil, []*language.DocAsset{doc1}, nil, nil)

	engine := analysis.NewRepositoryHealthEngine()
	report := engine.ComputeHealth([]*analysis.Finding{findingCrit, findingHigh, findingMed}, nil, discResult, langModel)

	if report == nil {
		t.Fatal("Expected non-nil RepositoryHealthReport")
	}

	if report.OverallScore() <= 0.0 || report.OverallScore() > 100.0 {
		t.Fatalf("Overall score out of bounds: %f", report.OverallScore())
	}
	if report.Grade() == "" {
		t.Fatal("Expected non-empty Grade")
	}
	if report.Engineering() == nil || report.Architecture() == nil || report.Documentation() == nil || report.Test() == nil || report.Maintainability() == nil {
		t.Fatal("Expected all 5 health dimensions to be initialized")
	}

	// Verify deductions were applied
	if report.Engineering().Score() >= 100.0 {
		t.Fatalf("Expected Engineering score to reflect deductions, got %f", report.Engineering().Score())
	}
	if report.Architecture().Score() >= 100.0 {
		t.Fatalf("Expected Architecture score to reflect deductions, got %f", report.Architecture().Score())
	}
}

func TestAnalysisEngineEndToEndAndConcurrency(t *testing.T) {
	pos := symbol.NewSourcePosition("internal/pkg/a.go", 10, 1, 100)
	sym1 := createTestSymbol("sym:internal/pkg.FuncA", "FuncA", "internal/pkg", "internal/pkg/a.go", symbol.SymbolKindFunction, "func FuncA()", pos, true)
	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{sym1})

	engine := analysis.New()

	params := analysis.AnalysisParams{
		SymbolDB: symDB,
		ConfigEntries: []*analysis.RawConfigEntry{
			{FilePath: "config.yaml", Key: "port", Value: "8080", Line: 1},
		},
	}

	model, err := engine.Analyze(params)
	if err != nil || model == nil {
		t.Fatalf("Analyze failed: err=%v, model=%+v", err, model)
	}

	if model.HealthReport() == nil {
		t.Fatal("Expected non-nil HealthReport in AnalysisModel")
	}

	// Concurrent read queries
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m := engine.Model()
			if m == nil || m.HealthReport() == nil {
				t.Errorf("Concurrent engine.Model() returned nil")
			}
			_ = m.AllFindings()
			_ = m.FindingsBySeverity(analysis.SeverityCritical)
		}()
	}
	wg.Wait()
}

func TestFalsePositiveResistance(t *testing.T) {
	pos := symbol.NewSourcePosition("pkg/api/client.go", 10, 1, 100)

	// 1. Exported public API in external package pkg/api (no local refs) -> NOT dead code
	symPublic := createTestSymbol("sym:pkg/api.Client", "Client", "pkg/api", "pkg/api/client.go", symbol.SymbolKindStruct, "type Client struct", pos, true)
	// 2. Entry point main -> NOT dead code
	symMain := createTestSymbol("sym:main.main", "main", "cmd/app", "cmd/app/main.go", symbol.SymbolKindFunction, "func main()", pos, false)
	// 3. Test helper in _test.go -> NOT dead code
	symTestHelper := createTestSymbol("sym:pkg/util.helper", "helper", "pkg/util", "pkg/util/util_test.go", symbol.SymbolKindFunction, "func helper()", pos, false)

	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{symPublic, symMain, symTestHelper})
	analyzer := analysis.NewCodeQualityAnalyzer(symDB, nil, nil, nil, nil)
	result := analyzer.Analyze()

	deadRes := result.RuleResult(analysis.RuleDeadCode)
	if deadRes != nil && deadRes.FindingCount() > 0 {
		t.Fatalf("False positive: legitimate public/entry/test symbols flagged as dead code: %+v", deadRes.Findings())
	}
}

func TestDeterminismAndOrdering(t *testing.T) {
	finding1 := analysis.NewFinding("code_quality", analysis.RuleDeadCode, analysis.CategoryQuality, analysis.SeverityLow, analysis.ConfidenceDefinite, "Dead1", "desc", "", "", "pkg/a", "pkg/a/a.go", "sym:a", nil, "ev1", nil, "", "p1")
	finding2 := analysis.NewFinding("dependency", analysis.RuleCircularDependencies, analysis.CategoryDependency, analysis.SeverityCritical, analysis.ConfidenceDefinite, "Cycle", "desc", "", "", "pkg/b", "pkg/b/b.go", "sym:b", nil, "ev2", nil, "", "p2")
	finding3 := analysis.NewFinding("architecture", analysis.RuleArchitectureViolations, analysis.CategoryArchitecture, analysis.SeverityCritical, analysis.ConfidenceDefinite, "Violation", "desc", "", "", "pkg/c", "pkg/c/c.go", "sym:c", nil, "ev3", nil, "", "p3")

	// Order 1
	sorted1 := analysis.SortFindings([]*analysis.Finding{finding1, finding2, finding3})
	// Order 2 (different input order)
	sorted2 := analysis.SortFindings([]*analysis.Finding{finding3, finding1, finding2})

	if len(sorted1) != len(sorted2) {
		t.Fatalf("Length mismatch: %d vs %d", len(sorted1), len(sorted2))
	}

	for i := range sorted1 {
		if sorted1[i].ID() != sorted2[i].ID() {
			t.Fatalf("Deterministic sorting mismatch at index %d: %s vs %s", i, sorted1[i].ID(), sorted2[i].ID())
		}
	}

	// Critical severity must appear first
	if sorted1[0].Severity() != analysis.SeverityCritical {
		t.Fatalf("Expected SeverityCritical first, got %s", sorted1[0].Severity())
	}
}

func TestAdversarialEmptyAndMissingData(t *testing.T) {
	// Nil / empty engine analyze
	engine := analysis.New()
	model, err := engine.Analyze(analysis.AnalysisParams{})
	if err != nil || model == nil {
		t.Fatalf("Engine Analyze on empty inputs failed: %v", err)
	}

	if model.TotalFindings() != 0 {
		t.Fatalf("Expected 0 findings on empty repository, got %d", model.TotalFindings())
	}

	if model.HealthReport() == nil {
		t.Fatal("Expected non-nil HealthReport on empty repository")
	}
}

func TestAdversarialSecretRedactionCompleteness(t *testing.T) {
	testCases := []struct {
		key      string
		val      string
		expected string
	}{
		{"DATABASE_PASSWORD", "super_secret_123", "[REDACTED]"},
		{"api_key", "key_abc_xyz", "[REDACTED]"},
		{"auth_token", "jwt_token_payload", "[REDACTED]"},
		{"private_key", "-----BEGIN RSA PRIVATE KEY-----", "[REDACTED]"},
		{"app_port", "8080", "8080"},
		{"server_host", "localhost", "localhost"},
	}

	for _, tc := range testCases {
		res := analysis.RedactSecret(tc.key, tc.val)
		if res != tc.expected {
			t.Errorf("RedactSecret(%s, %s) = %s; expected %s", tc.key, tc.val, res, tc.expected)
		}
	}
}

func TestAdversarialHealthScoreBoundaries(t *testing.T) {
	engine := analysis.NewRepositoryHealthEngine()

	// 1. Catastrophic repository (10 critical findings) -> score clamped to 0.0, grade F
	var critFindings []*analysis.Finding
	for i := 0; i < 10; i++ {
		critFindings = append(critFindings, analysis.NewFinding(
			"dependency",
			analysis.RuleCircularDependencies,
			analysis.CategoryDependency,
			analysis.SeverityCritical,
			analysis.ConfidenceDefinite,
			"Cycle",
			"desc",
			"",
			"",
			"pkg/a",
			"",
			"",
			nil,
			"cycle",
			nil,
			"",
			"dep",
		))
	}

	reportCatastrophic := engine.ComputeHealth(critFindings, nil, nil, nil)
	if reportCatastrophic.OverallScore() > 50.0 {
		t.Fatalf("Catastrophic score too high: %f", reportCatastrophic.OverallScore())
	}
	if reportCatastrophic.Grade() != "F" && reportCatastrophic.Grade() != "D" {
		t.Fatalf("Expected low grade for catastrophic repository, got %s", reportCatastrophic.Grade())
	}

	// 2. Perfect repository (0 findings, 100% doc, 100% tests)
	pos := symbol.NewSourcePosition("pkg/a/a.go", 10, 1, 100)
	doc := symbol.NewDocEntry("doc1", "sym:Alpha", symbol.DocKindStruct, "Alpha does important things.", "// Alpha does important things.", pos)
	symDoc := symbol.NewSymbol("sym:Alpha", symbol.SymbolKindStruct, "Alpha", "pkg", "pkg/a", "pkg/a/a.go", "", true, "type Alpha struct", "", false, nil, nil, pos, doc)
	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{symDoc})

	file1 := discovery.NewFileEntry("pkg/a/a.go", "d:/workspace/pkg/a/a.go", false, 500, time.Now(), ".go", nil, false, false, false)
	fileTest := discovery.NewFileEntry("pkg/a/a_test.go", "d:/workspace/pkg/a/a_test.go", false, 300, time.Now(), ".go", nil, false, false, false)
	discResult := discovery.NewResult(nil, "d:/workspace", []*discovery.FileEntry{file1, fileTest}, nil, nil, nil, nil)

	docAsset := language.NewDocAsset(language.DocADR, "docs/ARCH.md", "architecture")
	langModel := language.NewStructureModel("d:/workspace", nil, nil, nil, nil, nil, nil, nil, []*language.DocAsset{docAsset}, nil, nil)

	reportPerfect := engine.ComputeHealth(nil, symDB, discResult, langModel)
	if reportPerfect.OverallScore() < 95.0 {
		t.Fatalf("Expected near-perfect score for healthy repository, got %f", reportPerfect.OverallScore())
	}
	if reportPerfect.Grade() != "A" {
		t.Fatalf("Expected grade A, got %s", reportPerfect.Grade())
	}
}

func TestDeterminismAcrossConcurrency(t *testing.T) {
	pos := symbol.NewSourcePosition("internal/pkg/a.go", 10, 1, 100)
	sym1 := createTestSymbol("sym:internal/pkg.FuncA", "FuncA", "internal/pkg", "internal/pkg/a.go", symbol.SymbolKindFunction, "func FuncA()", pos, false)
	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{sym1})

	params := analysis.AnalysisParams{
		SymbolDB: symDB,
		ConfigEntries: []*analysis.RawConfigEntry{
			{FilePath: "config.yaml", Key: "port", Value: "8080", Line: 1},
			{FilePath: "config.yaml", Key: "port", Value: "8081", Line: 2},
		},
	}

	engine := analysis.New()
	baselineModel, err := engine.Analyze(params)
	if err != nil || baselineModel == nil {
		t.Fatalf("Baseline analyze failed: %v", err)
	}

	baselineScore := baselineModel.HealthReport().OverallScore()
	baselineCount := baselineModel.TotalFindings()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e := analysis.New()
			m, err := e.Analyze(params)
			if err != nil || m == nil {
				t.Errorf("Concurrent analyze failed: %v", err)
				return
			}
			if m.TotalFindings() != baselineCount {
				t.Errorf("Finding count mismatch under concurrency: %d vs baseline %d", m.TotalFindings(), baselineCount)
			}
			if m.HealthReport().OverallScore() != baselineScore {
				t.Errorf("Score mismatch under concurrency: %f vs baseline %f", m.HealthReport().OverallScore(), baselineScore)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkAnalysisEngine(b *testing.B) {
	pos := symbol.NewSourcePosition("internal/pkg/a.go", 10, 1, 100)
	sym1 := createTestSymbol("sym:internal/pkg.FuncA", "FuncA", "internal/pkg", "internal/pkg/a.go", symbol.SymbolKindFunction, "func FuncA()", pos, false)
	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{sym1})

	params := analysis.AnalysisParams{
		SymbolDB: symDB,
		ConfigEntries: []*analysis.RawConfigEntry{
			{FilePath: "config.yaml", Key: "port", Value: "8080", Line: 1},
			{FilePath: "config.yaml", Key: "port", Value: "8081", Line: 2},
		},
	}

	engine := analysis.New()
	b.ReportAllocs()

	for b.Loop() {
		_, err := engine.Analyze(params)
		if err != nil {
			b.Fatalf("Analyze failed: %v", err)
		}
	}
}
