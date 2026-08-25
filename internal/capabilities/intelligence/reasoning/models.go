package reasoning

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/knowledgegraph"
)

// AffectedEntity represents an engineering entity impacted by a proposed change.
type AffectedEntity struct {
	EntityID     string                    `json:"entity_id"`
	EntityType   knowledgegraph.EntityType `json:"entity_type"`
	Name         string                    `json:"name"`
	PackagePath  string                    `json:"package_path"`
	FilePath     string                    `json:"file_path"`
	ImpactReason string                    `json:"impact_reason"`
	Direct       bool                      `json:"direct"`
	Distance     int                       `json:"distance"`
	Evidence     string                    `json:"evidence"`
	Provenance   string                    `json:"provenance"`
}

// ImpactPath represents a sequence of relationships propagating impact from source to target.
type ImpactPath struct {
	SourceID      string   `json:"source_id"`
	TargetID      string   `json:"target_id"`
	Length        int      `json:"length"`
	HopEntityIDs  []string `json:"hop_entity_ids"`
	Relationships []string `json:"relationships"`
	Evidence      string   `json:"evidence"`
}

// ImpactAnalysisResult encapsulates the complete deterministic impact evaluation.
type ImpactAnalysisResult struct {
	TargetEntityID      string            `json:"target_entity_id"`
	Scope               ImpactScope       `json:"scope"`
	AffectedSymbols     []*AffectedEntity `json:"affected_symbols"`
	AffectedPackages    []*AffectedEntity `json:"affected_packages"`
	AffectedModules     []*AffectedEntity `json:"affected_modules"`
	AffectedFiles       []*AffectedEntity `json:"affected_files"`
	DependencyChain     []string          `json:"dependency_chain"`
	ImpactPaths         []*ImpactPath     `json:"impact_paths"`
	TotalAffectedCount  int               `json:"total_affected_count"`
	RepositoryImpacted  bool              `json:"repository_impacted"`
	CrossModuleImpacted bool              `json:"cross_module_impacted"`
}

// RefactoringSafetyResult encapsulates safety analysis for a proposed structural change.
type RefactoringSafetyResult struct {
	TargetID             string               `json:"target_id"`
	Kind                 RefactoringKind      `json:"kind"`
	Classification       SafetyClassification `json:"classification"`
	Safe                 bool                 `json:"safe"`
	BlockingReasons      []string             `json:"blocking_reasons"`
	UnresolvedReferences []string             `json:"unresolved_references"`
	AffectedContracts    []string             `json:"affected_contracts"`
	Evidence             string               `json:"evidence"`
	Provenance           string               `json:"provenance"`
}

// RefactoringRiskAssessment provides a deterministic risk classification with contributing factors.
type RefactoringRiskAssessment struct {
	TargetID            string    `json:"target_id"`
	Risk                RiskLevel `json:"risk"`
	Score               int       `json:"score"`
	ContributingFactors []string  `json:"contributing_factors"`
	DirectReferences    int       `json:"direct_references"`
	TransitiveDeps      int       `json:"transitive_deps"`
	CrossModuleRefs     int       `json:"cross_module_refs"`
	ExportedAPI         bool      `json:"exported_api"`
}

// BreakingChangeFinding represents an individual breaking or compatible contract modification.
type BreakingChangeFinding struct {
	ID             string                      `json:"id"`
	Category       BreakingChangeCategory      `json:"category"`
	Classification CompatibilityClassification `json:"classification"`
	AffectedEntity string                      `json:"affected_entity"`
	ChangeSummary  string                      `json:"change_summary"`
	Reason         string                      `json:"reason"`
	Evidence       string                      `json:"evidence"`
	Provenance     string                      `json:"provenance"`
}

// BreakingChangeReport encapsulates the complete breaking change analysis findings.
type BreakingChangeReport struct {
	BaselineRoot       string                   `json:"baseline_root,omitempty"`
	TargetRoot         string                   `json:"target_root"`
	HasBreakingChanges bool                     `json:"has_breaking_changes"`
	Findings           []*BreakingChangeFinding `json:"findings"`
	SummaryByCategory  map[string]int           `json:"summary_by_category"`
	SummaryBySeverity  map[string]int           `json:"summary_by_severity"`
}

// Recommendation encapsulates an evidence-backed engineering recommendation.
type Recommendation struct {
	ID                string                 `json:"id"`
	Category          RecommendationCategory `json:"category"`
	Priority          PriorityLevel          `json:"priority"`
	Title             string                 `json:"title"`
	Description       string                 `json:"description"`
	TargetEntityID    string                 `json:"target_entity_id"`
	RuleID            string                 `json:"rule_id"`
	Evidence          string                 `json:"evidence"`
	Consequence       string                 `json:"consequence"`
	RecommendedAction string                 `json:"recommended_action"`
	Provenance        string                 `json:"provenance"`
}

// ReasoningFact represents an atomic evidence-backed fact in a reasoning chain.
type ReasoningFact struct {
	Subject    string `json:"subject"`
	Predicate  string `json:"predicate"`
	Object     string `json:"object"`
	Evidence   string `json:"evidence"`
	Provenance string `json:"provenance"`
}

// ReasoningChain represents a logical derivation trace connecting facts to a conclusion.
type ReasoningChain struct {
	TargetID    string           `json:"target_id"`
	Conclusion  string           `json:"conclusion"`
	Facts       []*ReasoningFact `json:"facts"`
	DerivedStep string           `json:"derived_step"`
}

// ReasoningReport is the aggregate container returned by the Reasoning Engine.
type ReasoningReport struct {
	RootPath          string                     `json:"root_path"`
	ImpactResult      *ImpactAnalysisResult      `json:"impact_result,omitempty"`
	RefactoringSafety *RefactoringSafetyResult   `json:"refactoring_safety,omitempty"`
	RiskAssessment    *RefactoringRiskAssessment `json:"risk_assessment,omitempty"`
	BreakingChanges   *BreakingChangeReport      `json:"breaking_changes,omitempty"`
	Recommendations   []*Recommendation          `json:"recommendations"`
	ReasoningChains   []*ReasoningChain          `json:"reasoning_chains"`
	GeneratedAt       time.Time                  `json:"generated_at"`
}

// Deterministic ID builders
func CanonicalBreakingFindingID(category BreakingChangeCategory, entityID, change string) string {
	h := sha256.Sum256(fmt.Appendf(nil, "%s:%s:%s", category, entityID, change))
	return fmt.Sprintf("break:%s:%s", category, hex.EncodeToString(h[:8]))
}

func CanonicalRecommendationID(category RecommendationCategory, targetID, ruleID string) string {
	h := sha256.Sum256(fmt.Appendf(nil, "%s:%s:%s", category, targetID, ruleID))
	return fmt.Sprintf("rec:%s:%s", category, hex.EncodeToString(h[:8]))
}

// Deterministic Sorting Helpers
func DeduplicateAndSortAffectedEntities(entities []*AffectedEntity) []*AffectedEntity {
	seen := make(map[string]*AffectedEntity)
	for _, e := range entities {
		if e == nil || e.EntityID == "" {
			continue
		}
		if _, exists := seen[e.EntityID]; !exists {
			seen[e.EntityID] = e
		}
	}
	result := make([]*AffectedEntity, 0, len(seen))
	for _, e := range seen {
		result = append(result, e)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].EntityID < result[j].EntityID
	})
	return result
}

func DeduplicateAndSortBreakingFindings(findings []*BreakingChangeFinding) []*BreakingChangeFinding {
	seen := make(map[string]*BreakingChangeFinding)
	for _, f := range findings {
		if f == nil || f.ID == "" {
			continue
		}
		if _, exists := seen[f.ID]; !exists {
			seen[f.ID] = f
		}
	}
	result := make([]*BreakingChangeFinding, 0, len(seen))
	for _, f := range seen {
		result = append(result, f)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

func DeduplicateAndSortRecommendations(recs []*Recommendation) []*Recommendation {
	seen := make(map[string]*Recommendation)
	for _, r := range recs {
		if r == nil || r.ID == "" {
			continue
		}
		if _, exists := seen[r.ID]; !exists {
			seen[r.ID] = r
		}
	}
	result := make([]*Recommendation, 0, len(seen))
	for _, r := range seen {
		result = append(result, r)
	}

	priorityRank := map[PriorityLevel]int{
		PriorityCritical: 1,
		PriorityHigh:     2,
		PriorityMedium:   3,
		PriorityLow:      4,
		PriorityInfo:     5,
	}

	sort.Slice(result, func(i, j int) bool {
		rI := priorityRank[result[i].Priority]
		rJ := priorityRank[result[j].Priority]
		if rI != rJ {
			return rI < rJ
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func DeduplicateAndSortImpactPaths(paths []*ImpactPath) []*ImpactPath {
	seen := make(map[string]*ImpactPath)
	for _, p := range paths {
		if p == nil {
			continue
		}
		key := fmt.Sprintf("%s->%s:%s", p.SourceID, p.TargetID, strings.Join(p.HopEntityIDs, ">"))
		if _, exists := seen[key]; !exists {
			seen[key] = p
		}
	}
	result := make([]*ImpactPath, 0, len(seen))
	for _, p := range seen {
		result = append(result, p)
	}
	sort.Slice(result, func(i, j int) bool {
		kI := fmt.Sprintf("%s->%s:%d", result[i].SourceID, result[i].TargetID, result[i].Length)
		kJ := fmt.Sprintf("%s->%s:%d", result[j].SourceID, result[j].TargetID, result[j].Length)
		return kI < kJ
	})
	return result
}
