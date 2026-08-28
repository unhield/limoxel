package contracts

import (
	"context"

	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// ImpactResult encapsulates blast-radius evaluation for an engineering entity.
type ImpactResult struct {
	TargetEntity       string    `json:"target_entity"`
	DirectlyImpacted   []string  `json:"directly_impacted"`
	IndirectlyImpacted []string  `json:"indirectly_impacted"`
	ImpactScore        float64   `json:"impact_score"`
	RiskLevel          string    `json:"risk_level"`
	Findings           []Finding `json:"findings,omitempty"`
}

// RecommendationResult encapsulates actionable engineering recommendations for an entity.
type RecommendationResult struct {
	TargetEntity    string    `json:"target_entity"`
	Recommendations []string  `json:"recommendations"`
	Confidence      float64   `json:"confidence"`
	Findings        []Finding `json:"findings,omitempty"`
}

// BreakingChangeResult encapsulates semantic breaking change risk analysis.
type BreakingChangeResult struct {
	HasBreakingChanges bool      `json:"has_breaking_changes"`
	BreakingChanges    []string  `json:"breaking_changes,omitempty"`
	Severity           string    `json:"severity"`
	MigrationAdvice    []string  `json:"migration_advice,omitempty"`
	Findings           []Finding `json:"findings,omitempty"`
}

// RefactoringResult encapsulates automated refactoring safety and impact guidance.
type RefactoringResult struct {
	Operation             string   `json:"operation"`
	TargetEntity          string   `json:"target_entity"`
	IsSafe                bool     `json:"is_safe"`
	Risks                 []string `json:"risks,omitempty"`
	RequiredModifications []string `json:"required_modifications,omitempty"`
}

// EngineeringInsight represents an actionable repository-level intelligence insight.
type EngineeringInsight struct {
	ID               string   `json:"id"`
	Category         string   `json:"category"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Severity         string   `json:"severity"`
	AffectedEntities []string `json:"affected_entities,omitempty"`
}

// ReasoningContract defines the public contract for deterministic engineering reasoning.
type ReasoningContract interface {
	Contract
	AnalyzeImpact(ctx context.Context, targetEntityID string) (*ImpactResult, error)
	GetRecommendations(ctx context.Context, targetEntityID string) (*RecommendationResult, error)
	AnalyzeBreakingChanges(ctx context.Context, targetEntityID string) (*BreakingChangeResult, error)
	RefactoringAdvice(ctx context.Context, targetEntityID string, newName string) (*RefactoringResult, error)
	EngineeringInsights(ctx context.Context) ([]EngineeringInsight, error)
	Reason(ctx context.Context, req ReasoningRequest) (*ReasoningResult, error)
}

// DefaultReasoningContractMetadata returns default contract descriptor for Reasoning operations.
func DefaultReasoningContractMetadata() BaseContract {
	return NewBaseContract(
		"ReasoningContract",
		lifecycle.CapabilityIntelligence,
		version.SemVer{Major: 1, Minor: 0, Patch: 0},
		lifecycle.StateSupported,
		"Provides public impact analysis, recommendation engine, breaking change evaluation, and refactoring advisor.",
	)
}
