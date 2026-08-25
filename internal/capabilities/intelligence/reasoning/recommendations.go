package reasoning

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/knowledgegraph"
)

// RecommendationEngine produces deterministic, evidence-backed engineering recommendations across 5 dimensions.
type RecommendationEngine struct{}

// NewRecommendationEngine constructs an initialized RecommendationEngine.
func NewRecommendationEngine() *RecommendationEngine {
	return &RecommendationEngine{}
}

// GenerateRecommendations evaluates the complete knowledge graph model and derives prioritized engineering recommendations.
func (e *RecommendationEngine) GenerateRecommendations(model *knowledgegraph.KnowledgeGraphModel) []*Recommendation {
	if model == nil {
		return nil
	}

	var recs []*Recommendation

	// 1. Dependency Recommendations
	pkgs := model.EntitiesByType(knowledgegraph.EntityPackage)
	for _, p := range pkgs {
		outbound := model.OutboundRelationships(p.ID())
		inbound := model.InboundRelationships(p.ID())

		// Check circular dependencies
		for _, out := range outbound {
			if out.Kind() == knowledgegraph.RelDependsOn || out.Kind() == knowledgegraph.RelImports {
				targetPkgID := out.TargetID()
				// Check if targetPkgID has edge back to p.ID()
				for _, back := range model.OutboundRelationships(targetPkgID) {
					if (back.Kind() == knowledgegraph.RelDependsOn || back.Kind() == knowledgegraph.RelImports) && back.TargetID() == p.ID() {
						ruleID := "DEP-CIRCULAR-01"
						recs = append(recs, &Recommendation{
							ID:                CanonicalRecommendationID(RecDependency, p.ID(), ruleID),
							Category:          RecDependency,
							Priority:          PriorityCritical,
							Title:             fmt.Sprintf("Circular Dependency between %s and %s", p.PackagePath(), targetPkgID),
							Description:       "Direct cyclic dependency detected between two packages.",
							TargetEntityID:    p.ID(),
							RuleID:            ruleID,
							Evidence:          fmt.Sprintf("%s -> %s and %s -> %s", p.ID(), targetPkgID, targetPkgID, p.ID()),
							Consequence:       "Causes tight coupling, prevents independent compilation/testing, and complicates dependency management.",
							RecommendedAction: "Extract shared types/interfaces into a common lower-level foundation package.",
							Provenance:        "dependency_graph_topology",
						})
					}
				}
			}
		}

		// Check high outbound dependency coupling
		depCount := 0
		for _, r := range outbound {
			if r.Kind() == knowledgegraph.RelDependsOn || r.Kind() == knowledgegraph.RelImports {
				depCount++
			}
		}
		if depCount >= 8 {
			ruleID := "DEP-HIGH-COUPLING-02"
			recs = append(recs, &Recommendation{
				ID:                CanonicalRecommendationID(RecDependency, p.ID(), ruleID),
				Category:          RecDependency,
				Priority:          PriorityHigh,
				Title:             fmt.Sprintf("High Outbound Coupling in Package %s", p.PackagePath()),
				Description:       fmt.Sprintf("Package depends directly on %d other packages.", depCount),
				TargetEntityID:    p.ID(),
				RuleID:            ruleID,
				Evidence:          fmt.Sprintf("outbound package dependency count = %d (threshold = 8)", depCount),
				Consequence:       "High fragility and risk of cascade rebuilds when upstream packages change.",
				RecommendedAction: "Invert dependencies using interfaces or consolidate related domain logic.",
				Provenance:        "dependency_fanout_metric",
			})
		}

		// Check orphan package
		if len(outbound) == 0 && len(inbound) == 0 && p.PackagePath() != "cmd" && !strings.HasPrefix(p.PackagePath(), "cmd/") {
			ruleID := "DEP-ORPHAN-PKG-03"
			recs = append(recs, &Recommendation{
				ID:                CanonicalRecommendationID(RecDependency, p.ID(), ruleID),
				Category:          RecDependency,
				Priority:          PriorityLow,
				Title:             fmt.Sprintf("Orphan Package %s", p.PackagePath()),
				Description:       "Package has no inbound or outbound graph dependencies.",
				TargetEntityID:    p.ID(),
				RuleID:            ruleID,
				Evidence:          "inbound = 0, outbound = 0",
				Consequence:       "May represent dead code or unintegrated utility package.",
				RecommendedAction: "Verify whether package is active or consider deprecating/removing.",
				Provenance:        "dependency_isolation_metric",
			})
		}
	}

	// 2. Architecture Recommendations
	// Check layer violations (e.g. storage calling service)
	for _, rel := range model.Relationships() {
		if rel.Kind() == knowledgegraph.RelCalls || rel.Kind() == knowledgegraph.RelDependsOn {
			src := model.EntityByID(rel.SourceID())
			tgt := model.EntityByID(rel.TargetID())
			if src != nil && tgt != nil {
				// storage calling service or platform calling capability
				if strings.Contains(src.PackagePath(), "storage") && strings.Contains(tgt.PackagePath(), "service") {
					ruleID := "ARCH-INVERTED-LAYER-01"
					recs = append(recs, &Recommendation{
						ID:                CanonicalRecommendationID(RecArchitecture, src.ID(), ruleID),
						Category:          RecArchitecture,
						Priority:          PriorityHigh,
						Title:             fmt.Sprintf("Inverted Layer Relationship: %s calls %s", src.PackagePath(), tgt.PackagePath()),
						Description:       "Lower-level storage component has dependency on higher-level service layer.",
						TargetEntityID:    src.ID(),
						RuleID:            ruleID,
						Evidence:          fmt.Sprintf("edge: %s -> %s (%s)", src.ID(), tgt.ID(), rel.Evidence()),
						Consequence:       "Violates layered architecture principles and creates inverted dependency cycles.",
						RecommendedAction: "Decouple persistence from service coordination by passing domain parameters or events.",
						Provenance:        "architecture_layer_inspection",
					})
				}
			}
		}
	}

	// 3. Performance Recommendations (Hotspots and Hub bottlenecks)
	for _, s := range model.EntitiesByType(knowledgegraph.EntitySymbol) {
		inbound := model.InboundRelationships(s.ID())
		callCount := 0
		distinctCallers := make(map[string]bool)
		for _, r := range inbound {
			if r.Kind() == knowledgegraph.RelCalls {
				callCount++
				distinctCallers[r.SourceID()] = true
			}
		}

		if len(distinctCallers) >= 10 {
			ruleID := "PERF-HUB-BOTTLENECK-01"
			recs = append(recs, &Recommendation{
				ID:                CanonicalRecommendationID(RecPerformance, s.ID(), ruleID),
				Category:          RecPerformance,
				Priority:          PriorityMedium,
				Title:             fmt.Sprintf("High-Traffic Central Symbol %s", s.Name()),
				Description:       fmt.Sprintf("Symbol is called by %d distinct callers across the codebase.", len(distinctCallers)),
				TargetEntityID:    s.ID(),
				RuleID:            ruleID,
				Evidence:          fmt.Sprintf("inbound distinct caller count = %d, total calls = %d", len(distinctCallers), callCount),
				Consequence:       "Centralized hotspot where performance regressions or lock contention affect broad subsystems.",
				RecommendedAction: "Ensure optimized algorithms, minimize mutex contention, or provide localized caching.",
				Provenance:        "call_graph_centrality",
			})
		}
	}

	// 4. Repository Organization Recommendations
	files := model.EntitiesByType(knowledgegraph.EntityFile)
	filesPerPkg := make(map[string]int)
	for _, f := range files {
		if f.PackagePath() != "" {
			filesPerPkg[f.PackagePath()]++
		}
	}
	for pkgPath, count := range filesPerPkg {
		if count >= 30 {
			pkgID := knowledgegraph.CanonicalEntityID(knowledgegraph.EntityPackage, pkgPath)
			ruleID := "ORG-OVERSIZED-PKG-01"
			recs = append(recs, &Recommendation{
				ID:                CanonicalRecommendationID(RecRepoOrganization, pkgID, ruleID),
				Category:          RecRepoOrganization,
				Priority:          PriorityMedium,
				Title:             fmt.Sprintf("Oversized Package %s", pkgPath),
				Description:       fmt.Sprintf("Package contains %d files in a single flat directory.", count),
				TargetEntityID:    pkgID,
				RuleID:            ruleID,
				Evidence:          fmt.Sprintf("file count = %d (threshold = 30)", count),
				Consequence:       "Reduced maintainability, long compilation times, and unclear sub-domain boundaries.",
				RecommendedAction: "Decompose package into focused domain subpackages or capability modules.",
				Provenance:        "file_distribution_analysis",
			})
		}
	}

	// 5. Engineering Recommendations (Unused exported symbols & Documentation)
	for _, s := range model.EntitiesByType(knowledgegraph.EntitySymbol) {
		isExported := len(s.Name()) > 0 && unicode.IsUpper([]rune(s.Name())[0])
		if isExported {
			inbound := model.InboundRelationships(s.ID())
			hasCallers := false
			hasDoc := false
			for _, r := range inbound {
				if r.Kind() == knowledgegraph.RelCalls {
					hasCallers = true
				}
				if r.Kind() == knowledgegraph.RelDocuments {
					hasDoc = true
				}
			}

			if !hasDoc && s.PackagePath() != "" && !strings.HasSuffix(s.FilePath(), "_test.go") {
				ruleID := "ENG-UNDOCUMENTED-API-01"
				recs = append(recs, &Recommendation{
					ID:                CanonicalRecommendationID(RecEngineering, s.ID(), ruleID),
					Category:          RecEngineering,
					Priority:          PriorityInfo,
					Title:             fmt.Sprintf("Undocumented Exported Symbol %s", s.Name()),
					Description:       "Exported symbol in public API has no attached doc comment.",
					TargetEntityID:    s.ID(),
					RuleID:            ruleID,
					Evidence:          fmt.Sprintf("symbol %s in %s has no RelDocuments edge", s.ID(), s.PackagePath()),
					Consequence:       "Decreased API discoverability and code readability.",
					RecommendedAction: "Add clear Go doc comment explaining purpose, parameters, and return contracts.",
					Provenance:        "documentation_graph_coverage",
				})
			}

			if !hasCallers && !strings.HasSuffix(s.FilePath(), "_test.go") && s.PackagePath() != "main" && !strings.HasPrefix(s.PackagePath(), "cmd") {
				ruleID := "ENG-UNUSED-EXPORT-02"
				recs = append(recs, &Recommendation{
					ID:                CanonicalRecommendationID(RecEngineering, s.ID(), ruleID),
					Category:          RecEngineering,
					Priority:          PriorityLow,
					Title:             fmt.Sprintf("Exported Symbol with Zero Internal Callers: %s", s.Name()),
					Description:       "Exported symbol is not referenced anywhere within the repository.",
					TargetEntityID:    s.ID(),
					RuleID:            ruleID,
					Evidence:          "inbound RelCalls count = 0",
					Consequence:       "May represent unused legacy API or unnecessary export increasing surface area.",
					RecommendedAction: "If internal-only, unexport symbol; if public library API, ensure test coverage.",
					Provenance:        "cross_reference_graph_audit",
				})
			}
		}
	}

	return DeduplicateAndSortRecommendations(recs)
}
