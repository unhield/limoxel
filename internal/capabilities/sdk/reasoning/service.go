package reasoning

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/knowledgegraph"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/reasoning"
	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	sdkerr "github.com/unhield/limoxel/internal/capabilities/sdk/errors"
)

// Service adapts internal Reasoning intelligence capabilities to the public ReasoningContract.
type Service struct {
	mu sync.RWMutex
	contracts.BaseContract
	kgModel *knowledgegraph.KnowledgeGraphModel
	engine  *reasoning.Engine
}

// Ensure Service implements ReasoningContract.
var _ contracts.ReasoningContract = (*Service)(nil)

// NewService constructs a new Reasoning SDK service adapter.
func NewService(kgModel *knowledgegraph.KnowledgeGraphModel) *Service {
	return &Service{
		BaseContract: contracts.DefaultReasoningContractMetadata(),
		kgModel:      kgModel,
		engine:       reasoning.New(),
	}
}

// SetModel updates the active knowledge graph model thread-safely.
func (s *Service) SetModel(model *knowledgegraph.KnowledgeGraphModel) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.kgModel = model
}

// AnalyzeImpact calculates change impact and affected downstream entities for a target.
func (s *Service) AnalyzeImpact(ctx context.Context, targetEntityID string) (*contracts.ImpactResult, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("ReasoningService", "reasoning service is nil")
	}
	if strings.TrimSpace(targetEntityID) == "" {
		return nil, sdkerr.NewInvalidInput("targetEntityID cannot be empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.kgModel == nil {
		return nil, sdkerr.NewUnavailable("KnowledgeGraph", "knowledge graph model is not initialized")
	}

	impactRes, err := s.engine.Impact().Analyze(s.kgModel, targetEntityID)
	if err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_IMPACT_FAILED", fmt.Sprintf("impact analysis failed for: %s", targetEntityID))
	}

	var direct []string
	var indirect []string
	score := 0.0
	risk := "low"

	if impactRes != nil {
		for _, sym := range impactRes.AffectedSymbols {
			if sym != nil {
				if sym.Direct {
					direct = append(direct, sym.EntityID)
				} else {
					indirect = append(indirect, sym.EntityID)
				}
			}
		}
		for _, pkg := range impactRes.AffectedPackages {
			if pkg != nil {
				if pkg.Direct {
					direct = append(direct, pkg.EntityID)
				} else {
					indirect = append(indirect, pkg.EntityID)
				}
			}
		}
		score = float64(impactRes.TotalAffectedCount)
		if impactRes.CrossModuleImpacted {
			risk = "high"
		} else if impactRes.RepositoryImpacted {
			risk = "medium"
		}
	}

	return &contracts.ImpactResult{
		TargetEntity:       targetEntityID,
		DirectlyImpacted:   direct,
		IndirectlyImpacted: indirect,
		ImpactScore:        score,
		RiskLevel:          risk,
	}, nil
}

// GetRecommendations produces actionable engineering recommendations for an entity.
func (s *Service) GetRecommendations(ctx context.Context, targetEntityID string) (*contracts.RecommendationResult, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("ReasoningService", "reasoning service is nil")
	}
	if strings.TrimSpace(targetEntityID) == "" {
		return nil, sdkerr.NewInvalidInput("targetEntityID cannot be empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.kgModel == nil {
		return nil, sdkerr.NewUnavailable("KnowledgeGraph", "knowledge graph model is not initialized")
	}

	recList := s.engine.Recommendations().GenerateRecommendations(s.kgModel)

	var recs []string
	for _, r := range recList {
		if r != nil {
			if targetEntityID == "" || r.TargetEntityID == targetEntityID {
				recs = append(recs, fmt.Sprintf("[%s] %s: %s", r.Priority, r.Title, r.Description))
			}
		}
	}

	if len(recs) == 0 {
		recs = append(recs, "No critical optimizations recommended at this time.")
	}

	return &contracts.RecommendationResult{
		TargetEntity:    targetEntityID,
		Recommendations: recs,
		Confidence:      0.95,
	}, nil
}

// AnalyzeBreakingChanges evaluates semantic breaking change risks against baseline models.
func (s *Service) AnalyzeBreakingChanges(ctx context.Context, targetEntityID string) (*contracts.BreakingChangeResult, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("ReasoningService", "reasoning service is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.kgModel == nil {
		return nil, sdkerr.NewUnavailable("KnowledgeGraph", "knowledge graph model is not initialized")
	}

	breakingRes, err := s.engine.BreakingChanges().AnalyzeBreakingChanges(nil, s.kgModel)
	if err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_BREAKING_CHANGES_FAILED", "breaking change analysis failed")
	}

	hasBreaking := false
	var changes []string
	var advice []string
	severity := "none"

	if breakingRes != nil {
		hasBreaking = breakingRes.HasBreakingChanges
		for _, bc := range breakingRes.Findings {
			if bc != nil {
				if targetEntityID == "" || bc.AffectedEntity == targetEntityID {
					changes = append(changes, fmt.Sprintf("%s: %s", bc.Category, bc.ChangeSummary))
					advice = append(advice, bc.Reason)
				}
			}
		}
		if hasBreaking {
			severity = "medium"
		}
	}

	return &contracts.BreakingChangeResult{
		HasBreakingChanges: hasBreaking,
		BreakingChanges:    changes,
		Severity:           severity,
		MigrationAdvice:    advice,
	}, nil
}

// RefactoringAdvice provides automated refactoring guidance for renaming and restructuring.
func (s *Service) RefactoringAdvice(ctx context.Context, targetEntityID string, newName string) (*contracts.RefactoringResult, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("ReasoningService", "reasoning service is nil")
	}
	if strings.TrimSpace(targetEntityID) == "" {
		return nil, sdkerr.NewInvalidInput("targetEntityID cannot be empty")
	}
	if strings.TrimSpace(newName) == "" {
		return nil, sdkerr.NewInvalidInput("newName cannot be empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.kgModel == nil {
		return nil, sdkerr.NewUnavailable("KnowledgeGraph", "knowledge graph model is not initialized")
	}

	refactRes, err := s.engine.Refactoring().AnalyzeRename(s.kgModel, targetEntityID, newName)
	if err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_REFACTORING_FAILED", fmt.Sprintf("refactoring advice failed for: %s", targetEntityID))
	}

	isSafe := true
	var risks []string
	var requiredMods []string

	if refactRes != nil {
		isSafe = refactRes.Safe
		risks = refactRes.BlockingReasons
		requiredMods = refactRes.UnresolvedReferences
	}

	return &contracts.RefactoringResult{
		Operation:             "rename",
		TargetEntity:          targetEntityID,
		IsSafe:                isSafe,
		Risks:                 risks,
		RequiredModifications: requiredMods,
	}, nil
}

// EngineeringInsights returns derived repository-level structural and evolutionary insights.
func (s *Service) EngineeringInsights(ctx context.Context) ([]contracts.EngineeringInsight, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("ReasoningService", "reasoning service is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.kgModel == nil {
		return nil, sdkerr.NewUnavailable("KnowledgeGraph", "knowledge graph model is not initialized")
	}

	var insights []contracts.EngineeringInsight
	for _, ins := range s.kgModel.Insights() {
		if ins != nil {
			var targets []string
			if ins.TargetID() != "" {
				targets = append(targets, ins.TargetID())
			}
			insights = append(insights, contracts.EngineeringInsight{
				ID:               ins.ID(),
				Category:         string(ins.Category()),
				Title:            ins.Title(),
				Description:      ins.Description(),
				Severity:         string(ins.Severity()),
				AffectedEntities: targets,
			})
		}
	}

	return insights, nil
}

// Reason performs a unified reasoning query.
func (s *Service) Reason(ctx context.Context, req contracts.ReasoningRequest) (*contracts.ReasoningResult, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("ReasoningService", "reasoning service is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	if req.TargetEntity != "" {
		impact, err := s.AnalyzeImpact(ctx, req.TargetEntity)
		if err != nil {
			return nil, err
		}
		recs, _ := s.GetRecommendations(ctx, req.TargetEntity)

		var recList []string
		if recs != nil {
			recList = recs.Recommendations
		}

		return &contracts.ReasoningResult{
			Conclusion:      fmt.Sprintf("Impact assessment completed with risk level: %s (blast radius score: %.2f)", impact.RiskLevel, impact.ImpactScore),
			Recommendations: recList,
			Confidence:      0.95,
		}, nil
	}

	insights, err := s.EngineeringInsights(ctx)
	if err != nil {
		return nil, err
	}

	return &contracts.ReasoningResult{
		Conclusion:      fmt.Sprintf("Evaluated repository knowledge graph with %d actionable engineering insights", len(insights)),
		Recommendations: []string{"Review architecture health score", "Address modularity boundaries"},
		Confidence:      0.90,
	}, nil
}
