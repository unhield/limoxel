package knowledgegraph

import (
	"fmt"
	"sort"
)

// GraphReasoningEngine executes deterministic inference rules over the knowledge graph.
type GraphReasoningEngine struct{}

// NewGraphReasoningEngine constructs a GraphReasoningEngine.
func NewGraphReasoningEngine() *GraphReasoningEngine {
	return &GraphReasoningEngine{}
}

// InferredDependencyChain represents an inferred transitive dependency path.
type InferredDependencyChain struct {
	SourceID string
	TargetID string
	Hops     int
	Path     []string
	Evidence string
}

// InferTransitiveDependencies computes bounded, cycle-safe transitive dependencies.
func (r *GraphReasoningEngine) InferTransitiveDependencies(
	model *KnowledgeGraphModel,
	maxDepth int,
) []*InferredDependencyChain {
	if model == nil || maxDepth <= 0 {
		return nil
	}

	var chains []*InferredDependencyChain
	pkgEntities := model.EntitiesByType(EntityPackage)

	for _, src := range pkgEntities {
		visited := make(map[string]bool)
		var currentPath []string

		var dfs func(currID string, depth int)
		dfs = func(currID string, depth int) {
			if depth > maxDepth {
				return
			}
			visited[currID] = true
			currentPath = append(currentPath, currID)

			for _, edge := range model.OutboundRelationships(currID) {
				if edge.Kind() != RelDependsOn && edge.Kind() != RelImports {
					continue
				}
				tgtID := edge.TargetID()
				tgtEnt := model.EntityByID(tgtID)
				if tgtEnt == nil || tgtEnt.Type() != EntityPackage {
					continue
				}

				if !visited[tgtID] {
					// Discovered transitive dependency
					fullPath := append([]string(nil), currentPath...)
					fullPath = append(fullPath, tgtID)
					chains = append(chains, &InferredDependencyChain{
						SourceID: src.ID(),
						TargetID: tgtID,
						Hops:     depth,
						Path:     fullPath,
						Evidence: fmt.Sprintf("inferred transitive dependency across %d hops: %v", depth, fullPath),
					})
					dfs(tgtID, depth+1)
				}
			}

			currentPath = currentPath[:len(currentPath)-1]
			visited[currID] = false
		}

		dfs(src.ID(), 1)
	}

	// Sort chains deterministically
	sort.Slice(chains, func(i, j int) bool {
		if chains[i].SourceID != chains[j].SourceID {
			return chains[i].SourceID < chains[j].SourceID
		}
		if chains[i].TargetID != chains[j].TargetID {
			return chains[i].TargetID < chains[j].TargetID
		}
		return chains[i].Hops < chains[j].Hops
	})

	return chains
}

// InferredOwnership represents an inferred ownership hierarchy mapping.
type InferredOwnership struct {
	EntityID     string
	OwnerChain   []string
	TopLevelUnit string
}

// InferOwnershipHierarchy traces the complete ownership chain from a symbol or file up to the repository root.
func (r *GraphReasoningEngine) InferOwnershipHierarchy(
	model *KnowledgeGraphModel,
	entityID string,
) *InferredOwnership {
	if model == nil || entityID == "" {
		return nil
	}

	var chain []string
	currID := entityID
	visited := make(map[string]bool)

	for currID != "" && !visited[currID] {
		visited[currID] = true
		chain = append(chain, currID)

		var parentID string
		for _, edge := range model.OutboundRelationships(currID) {
			if edge.Kind() == RelBelongsTo {
				parentID = edge.TargetID()
				break
			}
		}
		currID = parentID
	}

	topLevel := ""
	if len(chain) > 0 {
		topLevel = chain[len(chain)-1]
	}

	return &InferredOwnership{
		EntityID:     entityID,
		OwnerChain:   chain,
		TopLevelUnit: topLevel,
	}
}
