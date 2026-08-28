package analysis_test

import (
	"context"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/analysis"
	analysissdk "github.com/unhield/limoxel/internal/capabilities/sdk/analysis"
	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
)

func createTestAnalysisModel() *analysis.AnalysisModel {
	f1 := analysis.NewFinding(
		"code_quality",
		analysis.RuleLargeFunctions,
		analysis.CategoryQuality,
		analysis.SeverityMedium,
		analysis.ConfidenceLikely,
		"Complex Function",
		"Function Add exceeds cyclomatic complexity threshold",
		"sample_repo",
		"sample_repo",
		"pkg/math",
		"pkg/math/math.go",
		"sym:math.Add",
		nil,
		"cyclomatic complexity = 12",
		nil,
		"Decompose function into smaller helpers",
		"analysis_engine",
	)

	fArch := analysis.NewFinding(
		"architecture",
		analysis.RuleArchitectureViolations,
		analysis.CategoryArchitecture,
		analysis.SeverityHigh,
		analysis.ConfidenceDefinite,
		"Modularity Warning",
		"Direct dependency on internal package",
		"sample_repo",
		"sample_repo",
		"pkg/math",
		"pkg/math/math.go",
		"sym:math.Add",
		nil,
		"package boundary crossed",
		nil,
		"Refactor module boundaries",
		"analysis_engine",
	)

	dim := analysis.NewHealthDimension("Quality", 88.0, 0.9, 0.85, 1.0, nil, map[string]float64{"issues": 1.0})
	healthReport := analysis.NewRepositoryHealthReport(dim, dim, dim, dim, dim, time.Now().UTC())

	rMapQual := map[analysis.RuleID]*analysis.AnalysisRuleResult{
		analysis.RuleLargeFunctions: analysis.NewAnalysisRuleResult(analysis.RuleLargeFunctions, analysis.StatusFindingsPresent, []*analysis.Finding{f1}, "found complex function"),
	}
	qualRes := analysis.NewAnalyzerResult("code_quality", rMapQual)

	rMapArch := map[analysis.RuleID]*analysis.AnalysisRuleResult{
		analysis.RuleArchitectureViolations: analysis.NewAnalysisRuleResult(analysis.RuleArchitectureViolations, analysis.StatusFindingsPresent, []*analysis.Finding{fArch}, "found arch violation"),
	}
	archRes := analysis.NewAnalyzerResult("architecture", rMapArch)

	rMapDep := map[analysis.RuleID]*analysis.AnalysisRuleResult{
		analysis.RuleCircularDependencies: analysis.NewAnalysisRuleResult(analysis.RuleCircularDependencies, analysis.StatusNoFindings, nil, "no circular deps"),
	}
	depRes := analysis.NewAnalyzerResult("dependency", rMapDep)

	rMapConf := map[analysis.RuleID]*analysis.AnalysisRuleResult{
		analysis.RuleInvalidConfiguration: analysis.NewAnalysisRuleResult(analysis.RuleInvalidConfiguration, analysis.StatusNoFindings, nil, "valid config"),
	}
	confRes := analysis.NewAnalyzerResult("configuration", rMapConf)

	return analysis.NewAnalysisModel(qualRes, depRes, archRes, confRes, healthReport)
}

func TestAnalysisServiceOperations(t *testing.T) {
	ctx := context.Background()
	model := createTestAnalysisModel()
	svc := analysissdk.NewService(model)

	// 1. AnalyzeArchitecture
	archRes, err := svc.AnalyzeArchitecture(ctx, "")
	if err != nil {
		t.Fatalf("AnalyzeArchitecture failed: %v", err)
	}
	if archRes.LayerCount == 0 {
		t.Errorf("expected layer count > 0")
	}

	// 2. AnalyzeDependencies
	depRes, err := svc.AnalyzeDependencies(ctx, "")
	if err != nil {
		t.Fatalf("AnalyzeDependencies failed: %v", err)
	}
	if depRes.TotalDependencies == 0 {
		t.Errorf("expected dependencies > 0")
	}

	// 3. RepositoryHealth
	health, err := svc.RepositoryHealth(ctx)
	if err != nil {
		t.Fatalf("RepositoryHealth failed: %v", err)
	}
	if health.OverallScore == 0 {
		t.Errorf("expected health score > 0, got %f", health.OverallScore)
	}
	if len(health.Dimensions) == 0 {
		t.Errorf("expected at least 1 health dimension")
	}

	// 4. AnalyzeQuality
	qual, err := svc.AnalyzeQuality(ctx, "")
	if err != nil {
		t.Fatalf("AnalyzeQuality failed: %v", err)
	}
	if len(qual.Findings) == 0 {
		t.Errorf("expected at least 1 finding in quality analysis")
	}

	// 5. AnalyzeConfiguration
	conf, err := svc.AnalyzeConfiguration(ctx)
	if err != nil {
		t.Fatalf("AnalyzeConfiguration failed: %v", err)
	}
	if conf.TotalConfigs == 0 {
		t.Errorf("expected TotalConfigs > 0")
	}

	// 6. Unified Analyze
	res, err := svc.Analyze(ctx, contracts.AnalysisRequest{
		AnalysisType: "health",
		TargetEntity: "sample_repo",
		Strict:       true,
	})
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if res.HealthScore == 0 {
		t.Errorf("expected valid health score")
	}
}

func TestAnalysisServiceErrorsAndNil(t *testing.T) {
	ctx := context.Background()
	svc := analysissdk.NewService(nil)

	if _, err := svc.AnalyzeArchitecture(ctx, ""); err == nil {
		t.Errorf("expected error on uninitialized AnalyzeArchitecture")
	}
	if _, err := svc.AnalyzeDependencies(ctx, ""); err == nil {
		t.Errorf("expected error on uninitialized AnalyzeDependencies")
	}
	if _, err := svc.RepositoryHealth(ctx); err == nil {
		t.Errorf("expected error on uninitialized RepositoryHealth")
	}
	if _, err := svc.AnalyzeQuality(ctx, ""); err == nil {
		t.Errorf("expected error on uninitialized AnalyzeQuality")
	}
	if _, err := svc.AnalyzeConfiguration(ctx); err == nil {
		t.Errorf("expected error on uninitialized AnalyzeConfiguration")
	}
	if _, err := svc.Analyze(ctx, contracts.AnalysisRequest{}); err == nil {
		t.Errorf("expected error on uninitialized Analyze")
	}

	var nilSvc *analysissdk.Service
	if _, err := nilSvc.AnalyzeArchitecture(ctx, ""); err == nil {
		t.Errorf("expected error on typed nil service AnalyzeArchitecture")
	}
	if _, err := nilSvc.AnalyzeDependencies(ctx, ""); err == nil {
		t.Errorf("expected error on typed nil service AnalyzeDependencies")
	}
	if _, err := nilSvc.RepositoryHealth(ctx); err == nil {
		t.Errorf("expected error on typed nil service RepositoryHealth")
	}
	if _, err := nilSvc.AnalyzeQuality(ctx, ""); err == nil {
		t.Errorf("expected error on typed nil service AnalyzeQuality")
	}
	if _, err := nilSvc.AnalyzeConfiguration(ctx); err == nil {
		t.Errorf("expected error on typed nil service AnalyzeConfiguration")
	}
	if _, err := nilSvc.Analyze(ctx, contracts.AnalysisRequest{}); err == nil {
		t.Errorf("expected error on typed nil service Analyze")
	}
}
