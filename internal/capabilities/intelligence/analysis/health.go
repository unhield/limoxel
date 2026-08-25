package analysis

import (
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/language"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
)

// RepositoryHealthEngine computes deterministic health dimension scores and produces the RepositoryHealthReport.
type RepositoryHealthEngine struct{}

// NewRepositoryHealthEngine constructs a RepositoryHealthEngine.
func NewRepositoryHealthEngine() *RepositoryHealthEngine {
	return &RepositoryHealthEngine{}
}

// ComputeHealth evaluates findings and structural repository metrics to construct an immutable RepositoryHealthReport.
func (e *RepositoryHealthEngine) ComputeHealth(
	findings []*Finding,
	symDB *symbol.SymbolDatabase,
	discResult *discovery.Result,
	langModel *language.StructureModel,
) *RepositoryHealthReport {
	engDim := e.computeEngineeringScore(findings)
	archDim := e.computeArchitectureScore(findings)
	docDim := e.computeDocumentationScore(symDB, langModel)
	testDim := e.computeTestScore(discResult, symDB)
	maintDim := e.computeMaintainabilityScore(findings, symDB, discResult)

	return NewRepositoryHealthReport(engDim, archDim, docDim, testDim, maintDim, time.Now().UTC())
}

// 12.1 Engineering Score
func (e *RepositoryHealthEngine) computeEngineeringScore(findings []*Finding) *HealthDimension {
	score := 100.0
	var deductions []*ScoreDeduction

	for _, f := range findings {
		if f == nil {
			continue
		}
		// Code quality and invalid configuration deductions
		if f.Category() == CategoryQuality || f.Category() == CategoryConfiguration || f.RuleID() == RuleInvalidImports {
			pts := f.Severity().SeverityWeight()
			deductions = append(deductions, NewScoreDeduction(f.RuleID(), f.Severity(), pts, f.Title()))
			score -= pts
		}
	}

	score = math.Max(0.0, math.Min(100.0, score))
	metrics := map[string]float64{
		"finding_count":     float64(len(findings)),
		"total_deductions":  100.0 - score,
		"engineering_index": score,
	}

	return NewHealthDimension("Engineering", score, 1.0, 1.0, 0.30, deductions, metrics)
}

// 12.2 Architecture Score
func (e *RepositoryHealthEngine) computeArchitectureScore(findings []*Finding) *HealthDimension {
	score := 100.0
	var deductions []*ScoreDeduction

	for _, f := range findings {
		if f == nil {
			continue
		}
		if f.Category() == CategoryArchitecture || f.Category() == CategoryDependency {
			pts := f.Severity().SeverityWeight()
			deductions = append(deductions, NewScoreDeduction(f.RuleID(), f.Severity(), pts, f.Title()))
			score -= pts
		}
	}

	score = math.Max(0.0, math.Min(100.0, score))
	metrics := map[string]float64{
		"architecture_deductions": 100.0 - score,
		"architecture_index":      score,
	}

	return NewHealthDimension("Architecture", score, 1.0, 1.0, 0.25, deductions, metrics)
}

// 12.3 Documentation Score
func (e *RepositoryHealthEngine) computeDocumentationScore(symDB *symbol.SymbolDatabase, langModel *language.StructureModel) *HealthDimension {
	totalExported := 0
	documentedExported := 0

	if symDB != nil {
		for _, sym := range symDB.AllSymbols() {
			if sym == nil || !sym.IsExported() {
				continue
			}
			totalExported++
			if sym.Doc() != nil && strings.TrimSpace(sym.Doc().Content()) != "" {
				documentedExported++
			}
		}
	}

	var symbolDocRatio float64 = 0.0
	if totalExported > 0 {
		symbolDocRatio = float64(documentedExported) / float64(totalExported)
	}

	// Doc asset presence (README, architecture docs)
	docAssetRatio := 0.0
	if langModel != nil && len(langModel.DocAssets()) > 0 {
		docAssetRatio = 1.0
	}

	score := (70.0 * symbolDocRatio) + (30.0 * docAssetRatio)
	score = math.Max(0.0, math.Min(100.0, score))

	metrics := map[string]float64{
		"total_exported_symbols":      float64(totalExported),
		"documented_exported_symbols": float64(documentedExported),
		"symbol_doc_ratio":            symbolDocRatio,
		"doc_asset_ratio":             docAssetRatio,
	}

	return NewHealthDimension("Documentation", score, 0.9, 1.0, 0.10, nil, metrics)
}

// 12.4 Test Score
func (e *RepositoryHealthEngine) computeTestScore(discResult *discovery.Result, symDB *symbol.SymbolDatabase) *HealthDimension {
	totalPackages := make(map[string]bool)
	packagesWithTests := make(map[string]bool)
	totalSourceFiles := 0
	testFiles := 0

	if discResult != nil && len(discResult.Files()) > 0 {
		for _, f := range discResult.Files() {
			if f == nil || f.IsIgnored() || f.IsDir() {
				continue
			}
			pkg := filepath.ToSlash(filepath.Dir(f.RelPath()))
			if pkg != "" {
				totalPackages[pkg] = true
			}
			if strings.HasSuffix(f.RelPath(), "_test.go") {
				testFiles++
				if pkg != "" {
					packagesWithTests[pkg] = true
				}
			} else {
				totalSourceFiles++
			}
		}
	} else if symDB != nil {
		for _, sym := range symDB.AllSymbols() {
			if sym == nil {
				continue
			}
			totalPackages[sym.PackagePath()] = true
			if strings.HasSuffix(sym.FilePath(), "_test.go") {
				packagesWithTests[sym.PackagePath()] = true
			}
		}
	}

	pkgCoverageRatio := 0.0
	if len(totalPackages) > 0 {
		pkgCoverageRatio = float64(len(packagesWithTests)) / float64(len(totalPackages))
	}

	fileTestRatio := 0.0
	if totalSourceFiles > 0 {
		fileTestRatio = math.Min(1.0, float64(testFiles)/float64(totalSourceFiles))
	}

	score := (60.0 * pkgCoverageRatio) + (40.0 * fileTestRatio)
	score = math.Max(0.0, math.Min(100.0, score))

	metrics := map[string]float64{
		"total_packages":      float64(len(totalPackages)),
		"packages_with_tests": float64(len(packagesWithTests)),
		"package_test_ratio":  pkgCoverageRatio,
		"file_test_ratio":     fileTestRatio,
		"test_file_count":     float64(testFiles),
		"source_file_count":   float64(totalSourceFiles),
	}

	return NewHealthDimension("Test", score, 0.95, 1.0, 0.15, nil, metrics)
}

// 12.5 Maintainability Score
func (e *RepositoryHealthEngine) computeMaintainabilityScore(
	findings []*Finding,
	symDB *symbol.SymbolDatabase,
	discResult *discovery.Result,
) *HealthDimension {
	score := 100.0
	var deductions []*ScoreDeduction

	for _, f := range findings {
		if f == nil {
			continue
		}
		// Maintainability deductions for large files, large functions, duplicate logic, and tight coupling
		switch f.RuleID() {
		case RuleLargeFiles:
			pts := 2.0
			deductions = append(deductions, NewScoreDeduction(RuleLargeFiles, SeverityLow, pts, f.Title()))
			score -= pts
		case RuleLargeFunctions:
			pts := 3.0
			deductions = append(deductions, NewScoreDeduction(RuleLargeFunctions, SeverityMedium, pts, f.Title()))
			score -= pts
		case RuleDuplicateLogic:
			pts := 4.0
			deductions = append(deductions, NewScoreDeduction(RuleDuplicateLogic, SeverityMedium, pts, f.Title()))
			score -= pts
		case RuleTightCoupling:
			pts := 4.0
			deductions = append(deductions, NewScoreDeduction(RuleTightCoupling, SeverityMedium, pts, f.Title()))
			score -= pts
		case RuleDeadCode:
			pts := 2.0
			deductions = append(deductions, NewScoreDeduction(RuleDeadCode, SeverityMedium, pts, f.Title()))
			score -= pts
		}
	}

	score = math.Max(0.0, math.Min(100.0, score))
	metrics := map[string]float64{
		"maintainability_deductions": 100.0 - score,
		"maintainability_index":      score,
	}

	return NewHealthDimension("Maintainability", score, 1.0, 1.0, 0.20, deductions, metrics)
}
