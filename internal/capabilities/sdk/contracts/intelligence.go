package contracts

import (
	"context"

	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// Finding represents an individual quality, maintainability, or architectural finding.
type Finding struct {
	Severity       string `json:"severity"`
	RuleID         string `json:"rule_id"`
	Message        string `json:"message"`
	Location       string `json:"location,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
}

// AnalysisRequest encapsulates parameters for an engineering analysis operation.
type AnalysisRequest struct {
	AnalysisType string            `json:"analysis_type"`
	Scope        string            `json:"scope,omitempty"`
	TargetEntity string            `json:"target_entity,omitempty"`
	Strict       bool              `json:"strict"`
	Parameters   map[string]string `json:"parameters,omitempty"`
}

// AnalysisResult encapsulates structured results from architecture, health, or dependency analysis.
type AnalysisResult struct {
	ID          string             `json:"id"`
	Target      string             `json:"target"`
	HealthScore float64            `json:"health_score"`
	Grade       string             `json:"grade"`
	Status      string             `json:"status"`
	Findings    []Finding          `json:"findings"`
	Metrics     map[string]float64 `json:"metrics,omitempty"`
}

// NavigationTarget represents an entity reached via code or relationship navigation.
type NavigationTarget struct {
	TargetID         string         `json:"target_id"`
	TargetName       string         `json:"target_name"`
	TargetKind       string         `json:"target_kind"`
	Location         SymbolLocation `json:"location"`
	RelationshipKind string         `json:"relationship_kind"`
	Package          string         `json:"package,omitempty"`
}

// NavigationResult represents the outcome of an engineering navigation operation.
type NavigationResult struct {
	SourceID     string             `json:"source_id"`
	Relationship string             `json:"relationship"`
	Targets      []NavigationTarget `json:"targets"`
}

// ReasoningRequest encapsulates parameters for deterministic engineering reasoning.
type ReasoningRequest struct {
	TargetEntity string            `json:"target_entity"`
	Objective    string            `json:"objective"`
	Depth        int               `json:"depth,omitempty"`
	Parameters   map[string]string `json:"parameters,omitempty"`
}

// ReasoningResult represents structured findings and actionable engineering recommendations.
type ReasoningResult struct {
	Conclusion      string    `json:"conclusion"`
	Findings        []Finding `json:"findings,omitempty"`
	Recommendations []string  `json:"recommendations"`
	Confidence      float64   `json:"confidence"`
}

// IntelligenceEvent represents a public event emitted during intelligence processing.
type IntelligenceEvent struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Timestamp int64             `json:"timestamp"`
	EntityID  string            `json:"entity_id,omitempty"`
	Payload   map[string]string `json:"payload,omitempty"`
}

// IntelligenceContract defines the public contract for engineering analysis, navigation, reasoning, and events.
type IntelligenceContract interface {
	Contract
	Analyze(ctx context.Context, req AnalysisRequest) (*AnalysisResult, error)
	Navigate(ctx context.Context, symbolID string, relKind string) (*NavigationResult, error)
	Reason(ctx context.Context, req ReasoningRequest) (*ReasoningResult, error)
	Events(ctx context.Context, eventType string) (<-chan IntelligenceEvent, error)
}

// DefaultIntelligenceContractMetadata returns default contract descriptor for Intelligence operations.
func DefaultIntelligenceContractMetadata() BaseContract {
	return NewBaseContract(
		"IntelligenceContract",
		lifecycle.CapabilityIntelligence,
		version.SemVer{Major: 1, Minor: 0, Patch: 0},
		lifecycle.StateSupported,
		"Provides public engineering analysis, code navigation, deterministic reasoning, and intelligence events.",
	)
}
