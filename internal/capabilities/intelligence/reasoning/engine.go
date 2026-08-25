package reasoning

import (
	"fmt"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/knowledgegraph"
)

// ReasoningParams defines parameters for comprehensive deterministic reasoning.
type ReasoningParams struct {
	Model           *knowledgegraph.KnowledgeGraphModel
	TargetEntityID  string
	RefactorNewName string
	BaselineModel   *knowledgegraph.KnowledgeGraphModel
}

// Engine is the thread-safe coordinator for Stage 6 Deterministic Reasoning Engine.
type Engine struct {
	mu          sync.RWMutex
	impact      *ImpactAnalyzer
	refactoring *RefactoringAdvisor
	breaking    *BreakingChangeAnalyzer
	recs        *RecommendationEngine
}

// New constructs an initialized Deterministic Reasoning Engine.
func New() *Engine {
	return &Engine{
		impact:      NewImpactAnalyzer(),
		refactoring: NewRefactoringAdvisor(),
		breaking:    NewBreakingChangeAnalyzer(),
		recs:        NewRecommendationEngine(),
	}
}

// Impact returns the ImpactAnalyzer.
func (e *Engine) Impact() *ImpactAnalyzer {
	if e == nil {
		return nil
	}
	return e.impact
}

// Refactoring returns the RefactoringAdvisor.
func (e *Engine) Refactoring() *RefactoringAdvisor {
	if e == nil {
		return nil
	}
	return e.refactoring
}

// BreakingChanges returns the BreakingChangeAnalyzer.
func (e *Engine) BreakingChanges() *BreakingChangeAnalyzer {
	if e == nil {
		return nil
	}
	return e.breaking
}

// Recommendations returns the RecommendationEngine.
func (e *Engine) Recommendations() *RecommendationEngine {
	if e == nil {
		return nil
	}
	return e.recs
}

// Reason performs unified deterministic reasoning across impact, refactoring safety, risk, recommendations, and chains.
func (e *Engine) Reason(params ReasoningParams) (*ReasoningReport, error) {
	if e == nil {
		return nil, ErrNilEngine
	}
	if params.Model == nil {
		return nil, ErrNilGraphModel
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	report := &ReasoningReport{
		RootPath:    params.Model.RootPath(),
		GeneratedAt: time.Now().UTC(),
	}

	// 1. Evaluate Impact if TargetEntityID provided
	if params.TargetEntityID != "" {
		impactRes, err := e.impact.Analyze(params.Model, params.TargetEntityID)
		if err != nil {
			return nil, err
		}
		report.ImpactResult = impactRes

		// 2. Evaluate Refactoring Safety & Risk
		if params.RefactorNewName != "" {
			refactRes, err := e.refactoring.AnalyzeRename(params.Model, params.TargetEntityID, params.RefactorNewName)
			if err != nil {
				return nil, err
			}
			report.RefactoringSafety = refactRes
		} else {
			delRes, err := e.refactoring.AnalyzeDeletion(params.Model, params.TargetEntityID)
			if err != nil {
				return nil, err
			}
			report.RefactoringSafety = delRes
		}

		riskRes, err := e.refactoring.AssessRisk(params.Model, params.TargetEntityID)
		if err != nil {
			return nil, err
		}
		report.RiskAssessment = riskRes

		// 3. Build Reasoning Chain
		var facts []*ReasoningFact
		for _, s := range impactRes.AffectedSymbols {
			facts = append(facts, &ReasoningFact{
				Subject:    params.TargetEntityID,
				Predicate:  "affects_symbol",
				Object:     s.EntityID,
				Evidence:   s.Evidence,
				Provenance: s.Provenance,
			})
		}
		for _, p := range impactRes.AffectedPackages {
			facts = append(facts, &ReasoningFact{
				Subject:    params.TargetEntityID,
				Predicate:  "affects_package",
				Object:     p.EntityID,
				Evidence:   p.Evidence,
				Provenance: p.Provenance,
			})
		}

		conclusion := fmt.Sprintf("Target %s impacts %d symbols and %d packages with %s scope", params.TargetEntityID, len(impactRes.AffectedSymbols), len(impactRes.AffectedPackages), impactRes.Scope)
		report.ReasoningChains = append(report.ReasoningChains, &ReasoningChain{
			TargetID:    params.TargetEntityID,
			Conclusion:  conclusion,
			Facts:       facts,
			DerivedStep: fmt.Sprintf("Graph traversal depth = %d", len(impactRes.ImpactPaths)),
		})
	}

	// 4. Evaluate Breaking Changes if BaselineModel provided
	if params.BaselineModel != nil {
		breakReport, err := e.breaking.AnalyzeBreakingChanges(params.BaselineModel, params.Model)
		if err != nil {
			return nil, err
		}
		report.BreakingChanges = breakReport
	}

	// 5. Derive Recommendations
	report.Recommendations = e.recs.GenerateRecommendations(params.Model)

	return report, nil
}
