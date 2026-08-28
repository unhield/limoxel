package reasoning_test

import (
	"context"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/knowledgegraph"
	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	"github.com/unhield/limoxel/internal/capabilities/sdk/reasoning"
)

func createTestGraphModel() *knowledgegraph.KnowledgeGraphModel {
	e1 := knowledgegraph.NewGraphEntity("repo:root", knowledgegraph.EntityRepository, "sample_repo", "", "root", nil, nil, "test")
	e2 := knowledgegraph.NewGraphEntity("pkg:math", knowledgegraph.EntityPackage, "math", "pkg/math", "pkg/math", nil, nil, "test")
	e3 := knowledgegraph.NewGraphEntity("sym:math.Add", knowledgegraph.EntitySymbol, "Add", "pkg/math", "pkg/math/math.go", nil, nil, "test")

	r1 := knowledgegraph.NewGraphRelationship("repo:root", "pkg:math", knowledgegraph.RelOwns, "discovery", "test", 1.0, nil)
	r2 := knowledgegraph.NewGraphRelationship("pkg:math", "sym:math.Add", knowledgegraph.RelOwns, "symbol_parser", "test", 1.0, nil)

	entities := []*knowledgegraph.GraphEntity{e1, e2, e3}
	rels := []*knowledgegraph.GraphRelationship{r1, r2}

	insight := knowledgegraph.NewEngineeringInsight(
		knowledgegraph.InsightArchitecture,
		knowledgegraph.SeverityLow,
		"Good Modularity",
		"Package structure is well-formed",
		"pkg:math",
		"structural_analysis",
		"test",
		map[string]float64{"confidence": 0.9},
	)

	return knowledgegraph.NewKnowledgeGraphModel("sample_repo", entities, rels, []*knowledgegraph.EngineeringInsight{insight}, time.Now().UTC())
}

func TestReasoningServiceOperations(t *testing.T) {
	ctx := context.Background()
	model := createTestGraphModel()
	svc := reasoning.NewService(model)

	// 1. AnalyzeImpact
	impact, err := svc.AnalyzeImpact(ctx, "pkg:math")
	if err != nil {
		t.Fatalf("AnalyzeImpact failed: %v", err)
	}
	if impact.TargetEntity != "pkg:math" {
		t.Errorf("got %q, want pkg:math", impact.TargetEntity)
	}

	// 2. GetRecommendations
	recs, err := svc.GetRecommendations(ctx, "pkg:math")
	if err != nil {
		t.Fatalf("GetRecommendations failed: %v", err)
	}
	if len(recs.Recommendations) == 0 {
		t.Errorf("expected recommendations for pkg:math")
	}

	// 3. AnalyzeBreakingChanges
	breaking, err := svc.AnalyzeBreakingChanges(ctx, "pkg:math")
	if err != nil {
		t.Fatalf("AnalyzeBreakingChanges failed: %v", err)
	}
	if breaking == nil {
		t.Errorf("expected non-nil BreakingChangeResult")
	}

	// 4. RefactoringAdvice
	refact, err := svc.RefactoringAdvice(ctx, "sym:math.Add", "AddNumbers")
	if err != nil {
		t.Fatalf("RefactoringAdvice failed: %v", err)
	}
	if !refact.IsSafe {
		t.Errorf("expected safe refactoring for test entity")
	}

	// 5. EngineeringInsights
	insights, err := svc.EngineeringInsights(ctx)
	if err != nil {
		t.Fatalf("EngineeringInsights failed: %v", err)
	}
	if len(insights) == 0 {
		t.Errorf("expected at least 1 engineering insight")
	}

	// 6. Reason
	reasonRes, err := svc.Reason(ctx, contracts.ReasoningRequest{
		TargetEntity: "pkg:math",
		Objective:    "Evaluate blast radius",
	})
	if err != nil {
		t.Fatalf("Reason failed: %v", err)
	}
	if reasonRes.Conclusion == "" {
		t.Errorf("expected valid conclusion")
	}
}

func TestReasoningServiceErrorsAndNil(t *testing.T) {
	ctx := context.Background()
	svc := reasoning.NewService(nil)

	if _, err := svc.AnalyzeImpact(ctx, ""); err == nil {
		t.Errorf("expected error for empty targetEntityID")
	}
	if _, err := svc.AnalyzeImpact(ctx, "target"); err == nil {
		t.Errorf("expected error on uninitialized AnalyzeImpact")
	}
	if _, err := svc.GetRecommendations(ctx, ""); err == nil {
		t.Errorf("expected error for empty targetEntityID")
	}
	if _, err := svc.RefactoringAdvice(ctx, "", ""); err == nil {
		t.Errorf("expected error for empty args")
	}
	if _, err := svc.EngineeringInsights(ctx); err == nil {
		t.Errorf("expected error on uninitialized EngineeringInsights")
	}
	if _, err := svc.Reason(ctx, contracts.ReasoningRequest{}); err == nil {
		t.Errorf("expected error on uninitialized Reason")
	}

	var nilSvc *reasoning.Service
	if _, err := nilSvc.AnalyzeImpact(ctx, "target"); err == nil {
		t.Errorf("expected error on typed nil service AnalyzeImpact")
	}
	if _, err := nilSvc.GetRecommendations(ctx, "target"); err == nil {
		t.Errorf("expected error on typed nil service GetRecommendations")
	}
	if _, err := nilSvc.AnalyzeBreakingChanges(ctx, "target"); err == nil {
		t.Errorf("expected error on typed nil service AnalyzeBreakingChanges")
	}
	if _, err := nilSvc.RefactoringAdvice(ctx, "target", "new"); err == nil {
		t.Errorf("expected error on typed nil service RefactoringAdvice")
	}
	if _, err := nilSvc.EngineeringInsights(ctx); err == nil {
		t.Errorf("expected error on typed nil service EngineeringInsights")
	}
	if _, err := nilSvc.Reason(ctx, contracts.ReasoningRequest{}); err == nil {
		t.Errorf("expected error on typed nil service Reason")
	}
}
