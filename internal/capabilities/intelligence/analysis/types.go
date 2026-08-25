package analysis

// FindingCategory classifies the primary domain of an engineering finding.
type FindingCategory string

const (
	CategoryQuality       FindingCategory = "code_quality"
	CategoryDependency    FindingCategory = "dependency"
	CategoryArchitecture  FindingCategory = "architecture"
	CategoryConfiguration FindingCategory = "configuration"
	CategoryHealth        FindingCategory = "health"
)

// Severity indicates the operational or structural severity of a finding.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// SeverityWeight returns the numerical deduction weight associated with a severity level.
func (s Severity) SeverityWeight() float64 {
	switch s {
	case SeverityCritical:
		return 15.0
	case SeverityHigh:
		return 8.0
	case SeverityMedium:
		return 4.0
	case SeverityLow:
		return 1.0
	case SeverityInfo:
		return 0.2
	default:
		return 0.0
	}
}

// SeverityRank returns an integer rank for deterministic ordering (higher severity = lower rank value).
func (s Severity) SeverityRank() int {
	switch s {
	case SeverityCritical:
		return 1
	case SeverityHigh:
		return 2
	case SeverityMedium:
		return 3
	case SeverityLow:
		return 4
	case SeverityInfo:
		return 5
	default:
		return 6
	}
}

// Confidence indicates detection confidence based on empirical repository evidence.
type Confidence string

const (
	ConfidenceTentative Confidence = "tentative"
	ConfidenceLikely    Confidence = "likely"
	ConfidenceDefinite  Confidence = "definite"
)

// EvaluationStatus indicates the evaluation outcome of an analyzer or individual rule.
type EvaluationStatus string

const (
	StatusEvaluated            EvaluationStatus = "evaluated"
	StatusNoFindings           EvaluationStatus = "no_findings"
	StatusFindingsPresent      EvaluationStatus = "findings_present"
	StatusUnsupported          EvaluationStatus = "unsupported"
	StatusInsufficientEvidence EvaluationStatus = "insufficient_evidence"
	StatusFailed               EvaluationStatus = "failed"
)

// RuleID represents canonical identifiers for analysis rules.
type RuleID string

const (
	// Task 1: Code Quality Rules
	RuleDeadCode       RuleID = "QUAL-001"
	RuleUnusedImports  RuleID = "QUAL-002"
	RuleUnusedExports  RuleID = "QUAL-003"
	RuleDuplicateLogic RuleID = "QUAL-004"
	RuleLargeFiles     RuleID = "QUAL-005"
	RuleLargeFunctions RuleID = "QUAL-006"

	// Task 2: Dependency Rules
	RuleCircularDependencies RuleID = "DEP-001"
	RuleLayerViolations      RuleID = "DEP-002"
	RuleInvalidImports       RuleID = "DEP-003"
	RuleTightCoupling        RuleID = "DEP-004"
	RuleOrphanPackages       RuleID = "DEP-005"

	// Task 3: Architecture Rules
	RuleArchitectureViolations RuleID = "ARCH-001"
	RuleModuleBoundaries       RuleID = "ARCH-002"
	RuleLayerConsistency       RuleID = "ARCH-003"
	RuleRepositoryOrganization RuleID = "ARCH-004"
	RulePackageCohesion        RuleID = "ARCH-005"

	// Task 4: Configuration Rules
	RuleInvalidConfiguration    RuleID = "CONF-001"
	RuleDuplicateConfiguration  RuleID = "CONF-002"
	RuleMissingConfiguration    RuleID = "CONF-003"
	RuleDeprecatedConfiguration RuleID = "CONF-004"
	RuleConfigurationConflicts  RuleID = "CONF-005"
)
