package knowledgegraph

import (
	"fmt"
	"sort"
)

// InsightEngine derives evidence-backed engineering insights from the knowledge graph model.
type InsightEngine struct{}

// NewInsightEngine constructs an InsightEngine.
func NewInsightEngine() *InsightEngine {
	return &InsightEngine{}
}

// DeriveInsights evaluates graph structures to construct deterministic EngineeringInsights.
func (e *InsightEngine) DeriveInsights(model *KnowledgeGraphModel) []*EngineeringInsight {
	if model == nil {
		return nil
	}

	var insights []*EngineeringInsight

	// 1. Complexity Insights: High Relationship Density on Packages
	pkgs := model.EntitiesByType(EntityPackage)
	for _, pkg := range pkgs {
		outRels := model.OutboundRelationships(pkg.ID())
		inRels := model.InboundRelationships(pkg.ID())
		totalDensity := len(outRels) + len(inRels)

		if totalDensity > 20 {
			in := NewEngineeringInsight(
				InsightComplexity,
				SeverityMedium,
				fmt.Sprintf("High Relationship Density in %s", pkg.Name()),
				fmt.Sprintf("Package %s participates in %d active graph relationships (%d outbound, %d inbound).", pkg.Name(), totalDensity, len(outRels), len(inRels)),
				pkg.ID(),
				fmt.Sprintf("total graph edges %d > threshold 20", totalDensity),
				"knowledge_graph",
				map[string]float64{
					"total_relationships": float64(totalDensity),
					"outbound_count":      float64(len(outRels)),
					"inbound_count":       float64(len(inRels)),
				},
			)
			insights = append(insights, in)
		}
	}

	// 2. Dependency Insights: High Fan-Out & Centrality
	for _, pkg := range pkgs {
		depCount := 0
		for _, r := range model.OutboundRelationships(pkg.ID()) {
			if r.Kind() == RelDependsOn || r.Kind() == RelImports {
				depCount++
			}
		}
		if depCount > 10 {
			in := NewEngineeringInsight(
				InsightDependency,
				SeverityMedium,
				fmt.Sprintf("Broad Dependency Fan-Out in %s", pkg.Name()),
				fmt.Sprintf("Package %s has %d outbound package dependencies.", pkg.Name(), depCount),
				pkg.ID(),
				fmt.Sprintf("outbound dependency count %d > threshold 10", depCount),
				"knowledge_graph",
				map[string]float64{
					"outbound_dependencies": float64(depCount),
				},
			)
			insights = append(insights, in)
		}
	}

	// 3. Architecture Insights: Cross-Component Bridge Saturation
	compEdges := make(map[string]int)
	for _, r := range model.Relationships() {
		src := model.EntityByID(r.SourceID())
		tgt := model.EntityByID(r.TargetID())
		if src != nil && tgt != nil && src.Type() == EntityPackage && tgt.Type() == EntityPackage {
			compEdges[src.PackagePath()+"->"+tgt.PackagePath()]++
		}
	}
	var sortedKeys []string
	for k := range compEdges {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	for _, k := range sortedKeys {
		cnt := compEdges[k]
		if cnt > 15 {
			in := NewEngineeringInsight(
				InsightArchitecture,
				SeverityLow,
				fmt.Sprintf("High Traffic Package Bridge: %s", k),
				fmt.Sprintf("Inter-package communication channel %s sustains %d relationship connections.", k, cnt),
				k,
				fmt.Sprintf("inter-package edge count %d > threshold 15", cnt),
				"knowledge_graph",
				map[string]float64{
					"bridge_edge_count": float64(cnt),
				},
			)
			insights = append(insights, in)
		}
	}

	// 4. Growth & Scale Insights: Monolithic Package Concentration
	totalSymbols := len(model.EntitiesByType(EntitySymbol))
	if totalSymbols > 20 {
		for _, pkg := range pkgs {
			pkgSyms := 0
			for _, sym := range model.EntitiesByType(EntitySymbol) {
				if sym.PackagePath() == pkg.PackagePath() {
					pkgSyms++
				}
			}
			ratio := float64(pkgSyms) / float64(totalSymbols)
			if ratio > 0.40 {
				in := NewEngineeringInsight(
					InsightGrowth,
					SeverityLow,
					fmt.Sprintf("High Symbol Concentration in %s", pkg.Name()),
					fmt.Sprintf("Package %s contains %d symbols (%.1f%% of total repository symbols).", pkg.Name(), pkgSyms, ratio*100),
					pkg.ID(),
					fmt.Sprintf("symbol ratio %.2f > threshold 0.40", ratio),
					"knowledge_graph",
					map[string]float64{
						"package_symbols": float64(pkgSyms),
						"symbol_ratio":    ratio,
					},
				)
				insights = append(insights, in)
			}
		}
	}

	// 5. Risk Insights: Single Point of Failure (High Fan-In Package with High Downstream Reliance)
	for _, pkg := range pkgs {
		inboundCount := 0
		for _, r := range model.InboundRelationships(pkg.ID()) {
			if r.Kind() == RelDependsOn || r.Kind() == RelImports {
				inboundCount++
			}
		}
		if inboundCount > 8 {
			in := NewEngineeringInsight(
				InsightRisk,
				SeverityMedium,
				fmt.Sprintf("Central Single Point of Failure: %s", pkg.Name()),
				fmt.Sprintf("Package %s is imported by %d distinct packages across the repository.", pkg.Name(), inboundCount),
				pkg.ID(),
				fmt.Sprintf("inbound dependency count %d > threshold 8", inboundCount),
				"knowledge_graph",
				map[string]float64{
					"inbound_dependencies": float64(inboundCount),
				},
			)
			insights = append(insights, in)
		}
	}

	return DeduplicateAndSortInsights(insights)
}
