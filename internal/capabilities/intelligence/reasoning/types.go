package reasoning

// ImpactScope defines the boundary scale of an engineering impact.
type ImpactScope string

const (
	ScopeLocal      ImpactScope = "local"
	ScopePackage    ImpactScope = "package"
	ScopeModule     ImpactScope = "module"
	ScopeRepository ImpactScope = "repository"
)

// RefactoringKind represents the category of structural refactoring analyzed.
type RefactoringKind string

const (
	RefactorRename     RefactoringKind = "rename"
	RefactorMove       RefactoringKind = "move"
	RefactorExtraction RefactoringKind = "extraction"
	RefactorDeletion   RefactoringKind = "deletion"
)

// SafetyClassification indicates whether a proposed refactoring is structurally safe.
type SafetyClassification string

const (
	SafetySafe                 SafetyClassification = "safe"
	SafetyUnsafe               SafetyClassification = "unsafe"
	SafetyBlocked              SafetyClassification = "blocked"
	SafetyInsufficientEvidence SafetyClassification = "insufficient_evidence"
)

// RiskLevel defines the deterministic engineering risk tier.
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// BreakingChangeCategory classifies breaking changes across API, package, symbol, interface, and versioning.
type BreakingChangeCategory string

const (
	BreakAPIChange            BreakingChangeCategory = "api_change"
	BreakPackageChange        BreakingChangeCategory = "package_change"
	BreakSymbolRemoval        BreakingChangeCategory = "symbol_removal"
	BreakInterfaceChange      BreakingChangeCategory = "interface_change"
	BreakVersionCompatibility BreakingChangeCategory = "version_compatibility"
)

// CompatibilityClassification indicates the severity and compatibility nature of a change.
type CompatibilityClassification string

const (
	CompatAdditive            CompatibilityClassification = "additive"
	CompatCompatible          CompatibilityClassification = "compatible"
	CompatPotentiallyBreaking CompatibilityClassification = "potentially_breaking"
	CompatBreaking            CompatibilityClassification = "breaking"
)

// RecommendationCategory represents the domain of an engineering recommendation.
type RecommendationCategory string

const (
	RecDependency       RecommendationCategory = "dependency"
	RecArchitecture     RecommendationCategory = "architecture"
	RecPerformance      RecommendationCategory = "performance"
	RecRepoOrganization RecommendationCategory = "repo_organization"
	RecEngineering      RecommendationCategory = "engineering"
)

// PriorityLevel represents the deterministic urgency of an engineering recommendation.
type PriorityLevel string

const (
	PriorityCritical PriorityLevel = "critical"
	PriorityHigh     PriorityLevel = "high"
	PriorityMedium   PriorityLevel = "medium"
	PriorityLow      PriorityLevel = "low"
	PriorityInfo     PriorityLevel = "info"
)
