package reasoning

import (
	"fmt"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/knowledgegraph"
)

// ImpactAnalyzer performs deterministic impact analysis across symbols, packages, modules, and dependencies.
type ImpactAnalyzer struct {
	maxDepth int
}

// NewImpactAnalyzer constructs an initialized ImpactAnalyzer.
func NewImpactAnalyzer() *ImpactAnalyzer {
	return &ImpactAnalyzer{
		maxDepth: 10,
	}
}

// SetMaxDepth configures the maximum traversal depth for impact paths.
func (a *ImpactAnalyzer) SetMaxDepth(depth int) {
	if depth > 0 {
		a.maxDepth = depth
	}
}

// Analyze evaluates the full engineering impact of modifying or removing the target entity.
func (a *ImpactAnalyzer) Analyze(model *knowledgegraph.KnowledgeGraphModel, targetID string) (*ImpactAnalysisResult, error) {
	if model == nil {
		return nil, ErrNilGraphModel
	}
	if strings.TrimSpace(targetID) == "" {
		return nil, NewReasoningError(ErrCatMissingTarget, "target ID cannot be empty", "", ErrMissingTarget)
	}

	// Resolve target entity
	target := model.EntityByID(targetID)
	if target == nil {
		// Try canonical resolution if prefix was omitted
		for _, ent := range model.Entities() {
			if ent.Name() == targetID || ent.PackagePath() == targetID || ent.FilePath() == targetID {
				target = ent
				targetID = ent.ID()
				break
			}
		}
	}
	if target == nil {
		return nil, NewReasoningError(ErrCatMissingTarget, fmt.Sprintf("target entity %q not found in graph", targetID), targetID, ErrMissingTarget)
	}

	var affectedSymbols []*AffectedEntity
	var affectedPackages []*AffectedEntity
	var affectedModules []*AffectedEntity
	var affectedFiles []*AffectedEntity
	var impactPaths []*ImpactPath
	var depChain []string

	visited := make(map[string]int)
	visited[targetID] = 0

	type queueItem struct {
		entityID string
		distance int
		path     []string
		rels     []string
	}

	queue := []queueItem{{
		entityID: targetID,
		distance: 0,
		path:     []string{targetID},
		rels:     nil,
	}}

	crossModule := false
	repoWide := false
	sourcePkg := target.PackagePath()

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.distance >= a.maxDepth {
			continue
		}

		// Inbound relationships: callers, importers, dependents, implementing entities
		inRels := model.InboundRelationships(curr.entityID)
		for _, rel := range inRels {
			srcID := rel.SourceID()
			srcEnt := model.EntityByID(srcID)
			if srcEnt == nil {
				continue
			}

			// Filter out documentation or purely hierarchical parent ownership from downstream impact
			if rel.Kind() == knowledgegraph.RelDocuments || rel.Kind() == knowledgegraph.RelConfigures {
				continue
			}

			dist, seen := visited[srcID]
			nextDist := curr.distance + 1

			if !seen || nextDist < dist {
				visited[srcID] = nextDist

				// Record affected entity
				aff := &AffectedEntity{
					EntityID:     srcEnt.ID(),
					EntityType:   srcEnt.Type(),
					Name:         srcEnt.Name(),
					PackagePath:  srcEnt.PackagePath(),
					FilePath:     srcEnt.FilePath(),
					ImpactReason: fmt.Sprintf("inbound %s from %s (%s)", rel.Kind(), curr.entityID, rel.Evidence()),
					Direct:       nextDist == 1,
					Distance:     nextDist,
					Evidence:     rel.Evidence(),
					Provenance:   rel.Provenance(),
				}

				switch srcEnt.Type() {
				case knowledgegraph.EntitySymbol:
					affectedSymbols = append(affectedSymbols, aff)
				case knowledgegraph.EntityPackage:
					affectedPackages = append(affectedPackages, aff)
					depChain = append(depChain, srcEnt.PackagePath())
				case knowledgegraph.EntityModule:
					affectedModules = append(affectedModules, aff)
				case knowledgegraph.EntityFile:
					affectedFiles = append(affectedFiles, aff)
				}

				if srcEnt.PackagePath() != "" && sourcePkg != "" && srcEnt.PackagePath() != sourcePkg {
					crossModule = true
				}

				newPath := append([]string(nil), curr.path...)
				newPath = append(newPath, srcID)
				newRels := append([]string(nil), curr.rels...)
				newRels = append(newRels, string(rel.Kind()))

				impactPaths = append(impactPaths, &ImpactPath{
					SourceID:      targetID,
					TargetID:      srcID,
					Length:        nextDist,
					HopEntityIDs:  newPath,
					Relationships: newRels,
					Evidence:      fmt.Sprintf("impact propagated across %d hops via %s", nextDist, strings.Join(newRels, "->")),
				})

				queue = append(queue, queueItem{
					entityID: srcID,
					distance: nextDist,
					path:     newPath,
					rels:     newRels,
				})
			}
		}

		// Also evaluate outbound relationships for structural calls or dependencies
		outRels := model.OutboundRelationships(curr.entityID)
		for _, rel := range outRels {
			if rel.Kind() == knowledgegraph.RelCalls || rel.Kind() == knowledgegraph.RelDependsOn || rel.Kind() == knowledgegraph.RelImports {
				tgtID := rel.TargetID()
				tgtEnt := model.EntityByID(tgtID)
				if tgtEnt == nil {
					continue
				}

				dist, seen := visited[tgtID]
				nextDist := curr.distance + 1

				if !seen || nextDist < dist {
					visited[tgtID] = nextDist

					aff := &AffectedEntity{
						EntityID:     tgtEnt.ID(),
						EntityType:   tgtEnt.Type(),
						Name:         tgtEnt.Name(),
						PackagePath:  tgtEnt.PackagePath(),
						FilePath:     tgtEnt.FilePath(),
						ImpactReason: fmt.Sprintf("outbound %s to %s", rel.Kind(), tgtID),
						Direct:       nextDist == 1,
						Distance:     nextDist,
						Evidence:     rel.Evidence(),
						Provenance:   rel.Provenance(),
					}

					switch tgtEnt.Type() {
					case knowledgegraph.EntitySymbol:
						affectedSymbols = append(affectedSymbols, aff)
					case knowledgegraph.EntityPackage:
						affectedPackages = append(affectedPackages, aff)
						depChain = append(depChain, tgtEnt.PackagePath())
					case knowledgegraph.EntityModule:
						affectedModules = append(affectedModules, aff)
					case knowledgegraph.EntityFile:
						affectedFiles = append(affectedFiles, aff)
					}

					newPath := append([]string(nil), curr.path...)
					newPath = append(newPath, tgtID)
					newRels := append([]string(nil), curr.rels...)
					newRels = append(newRels, string(rel.Kind()))

					impactPaths = append(impactPaths, &ImpactPath{
						SourceID:      targetID,
						TargetID:      tgtID,
						Length:        nextDist,
						HopEntityIDs:  newPath,
						Relationships: newRels,
						Evidence:      fmt.Sprintf("impact propagated across %d hops via %s", nextDist, strings.Join(newRels, "->")),
					})

					queue = append(queue, queueItem{
						entityID: tgtID,
						distance: nextDist,
						path:     newPath,
						rels:     newRels,
					})
				}
			}
		}
	}

	// Deduplicate and sort all collections
	dedupSymbols := DeduplicateAndSortAffectedEntities(affectedSymbols)
	dedupPackages := DeduplicateAndSortAffectedEntities(affectedPackages)
	dedupModules := DeduplicateAndSortAffectedEntities(affectedModules)
	dedupFiles := DeduplicateAndSortAffectedEntities(affectedFiles)
	dedupPaths := DeduplicateAndSortImpactPaths(impactPaths)

	// Clean dep chain
	sort.Strings(depChain)
	var cleanDepChain []string
	seenDep := make(map[string]bool)
	for _, d := range depChain {
		if d != "" && !seenDep[d] {
			seenDep[d] = true
			cleanDepChain = append(cleanDepChain, d)
		}
	}

	totalAffected := len(dedupSymbols) + len(dedupPackages) + len(dedupModules) + len(dedupFiles)

	// Determine scope
	var scope ImpactScope
	if totalAffected == 0 {
		scope = ScopeLocal
	} else if len(dedupPackages) > 1 || crossModule {
		scope = ScopeModule
		if len(dedupPackages) >= 5 || len(dedupSymbols) >= 20 {
			scope = ScopeRepository
			repoWide = true
		}
	} else if len(dedupPackages) == 1 {
		scope = ScopePackage
	} else {
		scope = ScopeLocal
	}

	return &ImpactAnalysisResult{
		TargetEntityID:      targetID,
		Scope:               scope,
		AffectedSymbols:     dedupSymbols,
		AffectedPackages:    dedupPackages,
		AffectedModules:     dedupModules,
		AffectedFiles:       dedupFiles,
		DependencyChain:     cleanDepChain,
		ImpactPaths:         dedupPaths,
		TotalAffectedCount:  totalAffected,
		RepositoryImpacted:  repoWide,
		CrossModuleImpacted: crossModule,
	}, nil
}
