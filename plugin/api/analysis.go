package api

import "context"

// AnalysisAPI defines the developer-facing interface for repository engineering intelligence analysis.
type AnalysisAPI interface {
	// AnalyzeArchitecture computes components, module layers, and architectural boundary violations.
	AnalyzeArchitecture(ctx context.Context, scope string) (*ArchitectureReport, error)

	// AnalyzeDependencies evaluates direct, transitive, external, and circular dependencies.
	AnalyzeDependencies(ctx context.Context, scope string) (*DependencyReport, error)

	// RepositoryHealth generates a multidimensional evaluation of repository health and maintainability.
	RepositoryHealth(ctx context.Context) (*HealthReport, error)

	// AnalyzeQuality computes maintainability, complexity, and testability metrics.
	AnalyzeQuality(ctx context.Context, scope string) (*QualityReport, error)

	// AnalyzeConfiguration inspects configuration files for syntax, completeness, and drift.
	AnalyzeConfiguration(ctx context.Context) (*ConfigurationReport, error)
}
