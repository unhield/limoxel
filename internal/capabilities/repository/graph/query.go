package graph

import (
	"sort"
	"strings"
)

// NodeFilter defines functional criteria for filtering nodes.
type NodeFilter func(n *Node) bool

// RelationshipFilter defines functional criteria for filtering relationships.
type RelationshipFilter func(r *Relationship) bool

// QueryEngine provides deterministic read-only graph query, traversal, and filtering capabilities.
type QueryEngine struct {
	graph *KnowledgeGraph
}

// NewQueryEngine constructs a QueryEngine.
func NewQueryEngine(graph *KnowledgeGraph) *QueryEngine {
	return &QueryEngine{graph: graph}
}

// LookupNode retrieves a node by its deterministic canonical ID.
func (qe *QueryEngine) LookupNode(id string) (*Node, error) {
	if qe == nil || qe.graph == nil {
		return nil, ErrNilEngine
	}
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return nil, ErrNodeNotFound
	}
	n := qe.graph.NodeByID(cleanID)
	if n == nil {
		return nil, ErrNodeNotFound
	}
	return n, nil
}

// FindNodesByName returns all nodes matching a display name.
func (qe *QueryEngine) FindNodesByName(name string) []*Node {
	if qe == nil || qe.graph == nil {
		return nil
	}
	return qe.graph.NodesByName(name)
}

// FindNodesByType returns all nodes of a specific NodeType.
func (qe *QueryEngine) FindNodesByType(t NodeType) []*Node {
	if qe == nil || qe.graph == nil {
		return nil
	}
	return qe.graph.NodesByType(t)
}

// LookupRelationships finds relationships between source and target nodes with optional type filter.
func (qe *QueryEngine) LookupRelationships(sourceID, targetID string, relType RelationshipType) []*Relationship {
	if qe == nil || qe.graph == nil {
		return nil
	}
	rels := qe.graph.RelationshipsBetween(sourceID, targetID)
	if relType == "" {
		return rels
	}
	var filtered []*Relationship
	for _, r := range rels {
		if r.Type() == relType {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// Neighbors returns nodes directly connected to a given node in the specified direction.
func (qe *QueryEngine) Neighbors(
	nodeID string,
	dir Direction,
	filterRelTypes ...RelationshipType,
) []*Node {
	if qe == nil || qe.graph == nil {
		return nil
	}
	cleanID := strings.TrimSpace(nodeID)
	if cleanID == "" {
		return nil
	}

	typeSet := make(map[RelationshipType]bool)
	for _, t := range filterRelTypes {
		if t != "" {
			typeSet[t] = true
		}
	}

	neighborIDs := make(map[string]bool)

	if dir == DirOutbound || dir == DirBoth {
		for _, r := range qe.graph.OutboundRelationships(cleanID) {
			if len(typeSet) == 0 || typeSet[r.Type()] {
				if r.TargetID() != cleanID {
					neighborIDs[r.TargetID()] = true
				}
			}
		}
	}

	if dir == DirInbound || dir == DirBoth {
		for _, r := range qe.graph.InboundRelationships(cleanID) {
			if len(typeSet) == 0 || typeSet[r.Type()] {
				if r.SourceID() != cleanID {
					neighborIDs[r.SourceID()] = true
				}
			}
		}
	}

	var results []*Node
	for id := range neighborIDs {
		if n := qe.graph.NodeByID(id); n != nil {
			results = append(results, n)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ID() < results[j].ID()
	})

	return results
}

// TraversePath performs a deterministic breadth-first traversal up to maxDepth with cycle protection.
func (qe *QueryEngine) TraversePath(
	startNodeID string,
	dir Direction,
	maxDepth int,
	filterRelTypes ...RelationshipType,
) ([]*Node, []*Relationship, error) {
	if qe == nil || qe.graph == nil {
		return nil, nil, ErrNilEngine
	}
	cleanID := strings.TrimSpace(startNodeID)
	startNode := qe.graph.NodeByID(cleanID)
	if startNode == nil {
		return nil, nil, ErrNodeNotFound
	}

	if maxDepth < 0 {
		maxDepth = 10 // default depth safeguard when negative
	}
	if maxDepth > 100 {
		return nil, nil, ErrMaxDepthExceeded
	}

	typeSet := make(map[RelationshipType]bool)
	for _, t := range filterRelTypes {
		if t != "" {
			typeSet[t] = true
		}
	}

	visitedNodes := make(map[string]bool)
	visitedRels := make(map[string]bool)

	visitedNodes[cleanID] = true
	var collectedNodes []*Node
	var collectedRels []*Relationship

	collectedNodes = append(collectedNodes, startNode)

	currentQueue := []string{cleanID}
	depth := 0

	for len(currentQueue) > 0 && depth < maxDepth {
		var nextQueue []string
		sort.Strings(currentQueue) // ensure deterministic queue traversal

		for _, currID := range currentQueue {
			var candidateRels []*Relationship

			if dir == DirOutbound || dir == DirBoth {
				candidateRels = append(candidateRels, qe.graph.OutboundRelationships(currID)...)
			}
			if dir == DirInbound || dir == DirBoth {
				candidateRels = append(candidateRels, qe.graph.InboundRelationships(currID)...)
			}

			for _, r := range candidateRels {
				if len(typeSet) > 0 && !typeSet[r.Type()] {
					continue
				}

				if !visitedRels[r.ID()] {
					visitedRels[r.ID()] = true
					collectedRels = append(collectedRels, r)
				}

				var nextNodeID string
				if r.SourceID() == currID {
					nextNodeID = r.TargetID()
				} else {
					nextNodeID = r.SourceID()
				}

				if !visitedNodes[nextNodeID] {
					visitedNodes[nextNodeID] = true
					if targetNode := qe.graph.NodeByID(nextNodeID); targetNode != nil {
						collectedNodes = append(collectedNodes, targetNode)
						nextQueue = append(nextQueue, nextNodeID)
					}
				}
			}
		}

		currentQueue = nextQueue
		depth++
	}

	sort.Slice(collectedNodes, func(i, j int) bool {
		return collectedNodes[i].ID() < collectedNodes[j].ID()
	})
	sort.Slice(collectedRels, func(i, j int) bool {
		return collectedRels[i].ID() < collectedRels[j].ID()
	})

	return collectedNodes, collectedRels, nil
}

// ReverseTraversal performs a reverse (inbound) traversal from targetNodeID up to maxDepth.
func (qe *QueryEngine) ReverseTraversal(
	targetNodeID string,
	maxDepth int,
	filterRelTypes ...RelationshipType,
) ([]*Node, []*Relationship, error) {
	return qe.TraversePath(targetNodeID, DirInbound, maxDepth, filterRelTypes...)
}

// FilterNodes applies a functional filter across all nodes in the graph.
func (qe *QueryEngine) FilterNodes(filter NodeFilter) []*Node {
	if qe == nil || qe.graph == nil || filter == nil {
		return nil
	}
	var matched []*Node
	for _, n := range qe.graph.AllNodes() {
		if filter(n) {
			matched = append(matched, n)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].ID() < matched[j].ID()
	})
	return matched
}

// FilterRelationships applies a functional filter across all relationships in the graph.
func (qe *QueryEngine) FilterRelationships(filter RelationshipFilter) []*Relationship {
	if qe == nil || qe.graph == nil || filter == nil {
		return nil
	}
	var matched []*Relationship
	for _, r := range qe.graph.AllRelationships() {
		if filter(r) {
			matched = append(matched, r)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].ID() < matched[j].ID()
	})
	return matched
}
