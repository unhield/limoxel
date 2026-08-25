package knowledgegraph

import (
	"sort"
	"strings"
	"time"
)

// GraphQueryEngine executes deterministic graph traversal, pathfinding, and subgraph queries.
type GraphQueryEngine struct {
	model *KnowledgeGraphModel
}

// NewGraphQueryEngine constructs a GraphQueryEngine.
func NewGraphQueryEngine(model *KnowledgeGraphModel) *GraphQueryEngine {
	return &GraphQueryEngine{model: model}
}

// Neighbors returns immediate neighbor entities filtered by direction and relationship kinds.
func (q *GraphQueryEngine) Neighbors(
	entityID string,
	dir Direction,
	kinds ...RelationshipKind,
) []*GraphEntity {
	if q.model == nil || entityID == "" {
		return nil
	}

	kindFilter := make(map[RelationshipKind]bool)
	for _, k := range kinds {
		kindFilter[k] = true
	}

	var neighbors []*GraphEntity

	if dir == DirOutbound || dir == DirBidirectional {
		for _, r := range q.model.OutboundRelationships(entityID) {
			if len(kindFilter) > 0 && !kindFilter[r.Kind()] {
				continue
			}
			tgt := q.model.EntityByID(r.TargetID())
			if tgt != nil {
				neighbors = append(neighbors, tgt)
			}
		}
	}

	if dir == DirInbound || dir == DirBidirectional {
		for _, r := range q.model.InboundRelationships(entityID) {
			if len(kindFilter) > 0 && !kindFilter[r.Kind()] {
				continue
			}
			src := q.model.EntityByID(r.SourceID())
			if src != nil {
				neighbors = append(neighbors, src)
			}
		}
	}

	return DeduplicateAndSortEntities(neighbors)
}

// FindPaths discovers all paths between startID and endID up to maxDepth with cycle protection.
func (q *GraphQueryEngine) FindPaths(
	startID, endID string,
	maxDepth int,
) []*GraphPath {
	if q.model == nil || startID == "" || endID == "" || maxDepth <= 0 {
		return nil
	}

	startEnt := q.model.EntityByID(startID)
	endEnt := q.model.EntityByID(endID)
	if startEnt == nil || endEnt == nil {
		return nil
	}

	var paths []*GraphPath
	visited := make(map[string]bool)

	var currentEntities []*GraphEntity
	var currentRels []*GraphRelationship

	var dfs func(currID string, depth int)
	dfs = func(currID string, depth int) {
		if depth > maxDepth {
			return
		}
		if currID == endID && len(currentRels) > 0 {
			paths = append(paths, NewGraphPath(currentEntities, currentRels))
			return
		}

		visited[currID] = true
		for _, rel := range q.model.OutboundRelationships(currID) {
			tgtID := rel.TargetID()
			if !visited[tgtID] {
				tgtEnt := q.model.EntityByID(tgtID)
				if tgtEnt == nil {
					continue
				}

				currentEntities = append(currentEntities, tgtEnt)
				currentRels = append(currentRels, rel)

				dfs(tgtID, depth+1)

				currentEntities = currentEntities[:len(currentEntities)-1]
				currentRels = currentRels[:len(currentRels)-1]
			}
		}
		visited[currID] = false
	}

	currentEntities = append(currentEntities, startEnt)
	dfs(startID, 1)

	// Sort paths deterministically by length, then by node sequence
	sort.Slice(paths, func(i, j int) bool {
		if paths[i].Length() != paths[j].Length() {
			return paths[i].Length() < paths[j].Length()
		}
		for k := 0; k < len(paths[i].Entities()) && k < len(paths[j].Entities()); k++ {
			if paths[i].Entities()[k].ID() != paths[j].Entities()[k].ID() {
				return paths[i].Entities()[k].ID() < paths[j].Entities()[k].ID()
			}
		}
		return false
	})

	return paths
}

// ExtractSubgraph extracts a radius-k neighborhood subgraph centered at rootID.
func (q *GraphQueryEngine) ExtractSubgraph(rootID string, radius int) (*KnowledgeGraphModel, error) {
	if q.model == nil {
		return nil, ErrNilGraph
	}
	rootEnt := q.model.EntityByID(rootID)
	if rootEnt == nil {
		return nil, NewKnowledgeGraphError(ErrCatEntityNotFound, "root entity not found", rootID, ErrEntityNotFound)
	}

	includedEntities := make(map[string]*GraphEntity)
	includedRels := make(map[string]*GraphRelationship)
	queue := []string{rootID}
	includedEntities[rootID] = rootEnt

	currentDepth := 0
	for len(queue) > 0 && currentDepth < radius {
		levelSize := len(queue)
		for i := 0; i < levelSize; i++ {
			currID := queue[0]
			queue = queue[1:]

			for _, rel := range q.model.OutboundRelationships(currID) {
				includedRels[rel.ID()] = rel
				tgt := q.model.EntityByID(rel.TargetID())
				if tgt != nil && includedEntities[tgt.ID()] == nil {
					includedEntities[tgt.ID()] = tgt
					queue = append(queue, tgt.ID())
				}
			}

			for _, rel := range q.model.InboundRelationships(currID) {
				includedRels[rel.ID()] = rel
				src := q.model.EntityByID(rel.SourceID())
				if src != nil && includedEntities[src.ID()] == nil {
					includedEntities[src.ID()] = src
					queue = append(queue, src.ID())
				}
			}
		}
		currentDepth++
	}

	var entList []*GraphEntity
	for _, e := range includedEntities {
		entList = append(entList, e)
	}
	var relList []*GraphRelationship
	for _, r := range includedRels {
		relList = append(relList, r)
	}

	return NewKnowledgeGraphModel(q.model.RootPath(), entList, relList, nil, time.Now().UTC()), nil
}

// SearchEntities finds entities matching a name query and optional entity type filter.
func (q *GraphQueryEngine) SearchEntities(query string, entityType EntityType) []*GraphEntity {
	if q.model == nil || query == "" {
		return nil
	}

	lowerQuery := strings.ToLower(query)
	var matches []*GraphEntity

	for _, e := range q.model.Entities() {
		if entityType != "" && e.Type() != entityType {
			continue
		}
		if strings.Contains(strings.ToLower(e.Name()), lowerQuery) || strings.Contains(strings.ToLower(e.ID()), lowerQuery) {
			matches = append(matches, e)
		}
	}

	return DeduplicateAndSortEntities(matches)
}
