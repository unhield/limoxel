package contracts

import (
	"context"

	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// ArchitectureAnalysisResult encapsulates architecture components, layer metrics, and boundary violations.
type ArchitectureAnalysisResult struct {
	TotalComponents int                `json:"total_components"`
	LayerCount      int                `json:"layer_count"`
	Violations      []Finding          `json:"violations,omitempty"`
	Findings        []Finding          `json:"findings,omitempty"`
	Metrics         map[string]float64 `json:"metrics,omitempty"`
}

// DependencyAnalysisResult encapsulates direct, transitive, circular, and external dependency metrics.
type DependencyAnalysisResult struct {
	TotalDependencies      int                `json:"total_dependencies"`
	DirectDependencies     int                `json:"direct_dependencies"`
	TransitiveDependencies int                `json:"transitive_dependencies"`
	CircularDependencies   int                `json:"circular_dependencies"`
	Findings               []Finding          `json:"findings,omitempty"`
	Metrics                map[string]float64 `json:"metrics,omitempty"`
}

// HealthDimensionResult represents a single scored dimension of repository health.
type HealthDimensionResult struct {
	Name       string             `json:"name"`
	Score      float64            `json:"score"`
	Confidence float64            `json:"confidence"`
	Coverage   float64            `json:"coverage"`
	Weight     float64            `json:"weight"`
	Metrics    map[string]float64 `json:"metrics,omitempty"`
}

// RepositoryHealthReport encapsulates multidimensional repository health evaluation.
type RepositoryHealthReport struct {
	OverallScore float64                 `json:"overall_score"`
	Grade        string                  `json:"grade"`
	Status       string                  `json:"status"`
	Dimensions   []HealthDimensionResult `json:"dimensions"`
	Findings     []Finding               `json:"findings,omitempty"`
}

// CodeQualityReport encapsulates maintainability, complexity, testability, and defect observations.
type CodeQualityReport struct {
	MaintainabilityScore float64            `json:"maintainability_score"`
	ComplexityScore      float64            `json:"complexity_score"`
	TestabilityScore     float64            `json:"testability_score"`
	TotalIssues          int                `json:"total_issues"`
	Findings             []Finding          `json:"findings,omitempty"`
	Metrics              map[string]float64 `json:"metrics,omitempty"`
}

// ConfigurationReport encapsulates configuration completeness, syntax validity, and drift observations.
type ConfigurationReport struct {
	TotalConfigs  int                `json:"total_configs"`
	ValidConfigs  int                `json:"valid_configs"`
	DriftWarnings int                `json:"drift_warnings"`
	Findings      []Finding          `json:"findings,omitempty"`
	Metrics       map[string]float64 `json:"metrics,omitempty"`
}

// AnalysisContract defines the public contract for engineering analysis capabilities.
type AnalysisContract interface {
	Contract
	AnalyzeArchitecture(ctx context.Context, scope string) (*ArchitectureAnalysisResult, error)
	AnalyzeDependencies(ctx context.Context, scope string) (*DependencyAnalysisResult, error)
	RepositoryHealth(ctx context.Context) (*RepositoryHealthReport, error)
	AnalyzeQuality(ctx context.Context, scope string) (*CodeQualityReport, error)
	AnalyzeConfiguration(ctx context.Context) (*ConfigurationReport, error)
	Analyze(ctx context.Context, req AnalysisRequest) (*AnalysisResult, error)
}

// DefaultAnalysisContractMetadata returns default contract descriptor for Analysis operations.
func DefaultAnalysisContractMetadata() BaseContract {
	return NewBaseContract(
		"AnalysisContract",
		lifecycle.CapabilityIntelligence,
		version.SemVer{Major: 1, Minor: 0, Patch: 0},
		lifecycle.StateSupported,
		"Provides public architecture, dependency, health, quality, and configuration engineering analysis.",
	)
}
