package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/knowledgegraph"
	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	sdkerr "github.com/unhield/limoxel/internal/capabilities/sdk/errors"
)

// Service adapts internal KnowledgeGraph intelligence capabilities to the public GraphContract.
type Service struct {
	mu sync.RWMutex
	contracts.BaseContract
	kgModel  *knowledgegraph.KnowledgeGraphModel
	queryEng *knowledgegraph.GraphQueryEngine
}

// Ensure Service implements GraphContract.
var _ contracts.GraphContract = (*Service)(nil)

// NewService constructs a new Knowledge Graph SDK service adapter.
func NewService(model *knowledgegraph.KnowledgeGraphModel) *Service {
	var queryEng *knowledgegraph.GraphQueryEngine
	if model != nil {
		queryEng = knowledgegraph.NewGraphQueryEngine(model)
	}
	return &Service{
		BaseContract: contracts.DefaultGraphContractMetadata(),
		kgModel:      model,
		queryEng:     queryEng,
	}
}

// SetModel updates the active knowledge graph model thread-safely.
func (s *Service) SetModel(model *knowledgegraph.KnowledgeGraphModel) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.kgModel = model
	if model != nil {
		s.queryEng = knowledgegraph.NewGraphQueryEngine(model)
	} else {
		s.queryEng = nil
	}
}

// GraphInfo returns the total count of nodes and directed edges in the knowledge graph.
func (s *Service) GraphInfo(ctx context.Context) (int, int, error) {
	if s == nil {
		return 0, 0, sdkerr.NewUnavailable("GraphService", "graph service is nil")
	}
	if err := ctx.Err(); err != nil {
		return 0, 0, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.kgModel == nil {
		return 0, 0, sdkerr.NewUnavailable("KnowledgeGraph", "knowledge graph model is not initialized")
	}

	return s.kgModel.TotalEntities(), s.kgModel.TotalRelationships(), nil
}

// GetNode retrieves a single node by canonical ID.
func (s *Service) GetNode(ctx context.Context, nodeID string) (*contracts.GraphNode, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("GraphService", "graph service is nil")
	}
	if strings.TrimSpace(nodeID) == "" {
		return nil, sdkerr.NewInvalidInput("nodeID cannot be empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.kgModel == nil {
		return nil, sdkerr.NewUnavailable("KnowledgeGraph", "knowledge graph model is not initialized")
	}

	ent := s.kgModel.EntityByID(nodeID)
	if ent == nil {
		return nil, sdkerr.NewNotFound("GraphNode", nodeID)
	}

	return convertEntityToNode(ent), nil
}

// GetRelationship retrieves a single relationship by canonical ID.
func (s *Service) GetRelationship(ctx context.Context, relID string) (*contracts.GraphRelationship, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("GraphService", "graph service is nil")
	}
	if strings.TrimSpace(relID) == "" {
		return nil, sdkerr.NewInvalidInput("relID cannot be empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.kgModel == nil {
		return nil, sdkerr.NewUnavailable("KnowledgeGraph", "knowledge graph model is not initialized")
	}

	rel := s.kgModel.RelationshipByID(relID)
	if rel == nil {
		return nil, sdkerr.NewNotFound("GraphRelationship", relID)
	}

	return convertRelToContract(rel), nil
}

// TraverseNodes traverses from startNodeID respecting the provided filter constraints.
func (s *Service) TraverseNodes(ctx context.Context, startNodeID string, filter contracts.GraphFilter) ([]contracts.GraphNode, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("GraphService", "graph service is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.kgModel == nil {
		return nil, sdkerr.NewUnavailable("KnowledgeGraph", "knowledge graph model is not initialized")
	}

	var results []contracts.GraphNode

	if strings.TrimSpace(startNodeID) == "" {
		// Return all matching nodes
		for _, ent := range s.kgModel.Entities() {
			if ent == nil {
				continue
			}
			if matchesEntityFilter(ent, filter) {
				results = append(results, *convertEntityToNode(ent))
			}
		}
	} else {
		// BFS traversal from start node
		startEnt := s.kgModel.EntityByID(startNodeID)
		if startEnt == nil {
			return nil, sdkerr.NewNotFound("GraphNode", startNodeID)
		}

		maxDepth := filter.MaxDepth
		if maxDepth <= 0 {
			maxDepth = 3
		}

		visited := make(map[string]bool)
		queue := []string{startNodeID}
		visited[startNodeID] = true
		depth := 0

		for len(queue) > 0 && depth <= maxDepth {
			levelSize := len(queue)
			for i := 0; i < levelSize; i++ {
				currID := queue[0]
				queue = queue[1:]

				currEnt := s.kgModel.EntityByID(currID)
				if currEnt != nil && matchesEntityFilter(currEnt, filter) {
					results = append(results, *convertEntityToNode(currEnt))
				}

				if depth < maxDepth {
					for _, rel := range s.kgModel.OutboundRelationships(currID) {
						if rel == nil {
							continue
						}
						if len(filter.RelationshipKinds) > 0 && !containsString(filter.RelationshipKinds, string(rel.Kind())) {
							continue
						}
						tgt := rel.TargetID()
						if !visited[tgt] {
							visited[tgt] = true
							queue = append(queue, tgt)
						}
					}
				}
			}
			depth++
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})

	return results, nil
}

// TraverseRelationships retrieves directed relationships connected to startNodeID or matching filter.
func (s *Service) TraverseRelationships(ctx context.Context, startNodeID string, filter contracts.GraphFilter) ([]contracts.GraphRelationship, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("GraphService", "graph service is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.kgModel == nil {
		return nil, sdkerr.NewUnavailable("KnowledgeGraph", "knowledge graph model is not initialized")
	}

	var results []contracts.GraphRelationship

	var rels []*knowledgegraph.GraphRelationship
	if strings.TrimSpace(startNodeID) != "" {
		rels = s.kgModel.OutboundRelationships(startNodeID)
	} else {
		rels = s.kgModel.Relationships()
	}

	for _, r := range rels {
		if r == nil {
			continue
		}
		if len(filter.RelationshipKinds) > 0 && !containsString(filter.RelationshipKinds, string(r.Kind())) {
			continue
		}
		results = append(results, *convertRelToContract(r))
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})

	return results, nil
}

// GetNeighbors returns neighboring nodes in the given direction.
func (s *Service) GetNeighbors(ctx context.Context, nodeID string, direction string, kinds ...string) ([]contracts.GraphNode, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("GraphService", "graph service is nil")
	}
	if strings.TrimSpace(nodeID) == "" {
		return nil, sdkerr.NewInvalidInput("nodeID cannot be empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.kgModel == nil || s.queryEng == nil {
		return nil, sdkerr.NewUnavailable("KnowledgeGraph", "knowledge graph query engine is not initialized")
	}

	dir := knowledgegraph.DirOutbound
	switch strings.ToLower(direction) {
	case "in", "inbound":
		dir = knowledgegraph.DirInbound
	case "both", "bidirectional":
		dir = knowledgegraph.DirBidirectional
	}

	var kKinds []knowledgegraph.RelationshipKind
	for _, k := range kinds {
		kKinds = append(kKinds, knowledgegraph.RelationshipKind(k))
	}

	ents := s.queryEng.Neighbors(nodeID, dir, kKinds...)
	var results []contracts.GraphNode
	for _, e := range ents {
		if e != nil {
			results = append(results, *convertEntityToNode(e))
		}
	}

	return results, nil
}

// FindPaths discovers all paths between startID and endID up to maxDepth.
func (s *Service) FindPaths(ctx context.Context, startID, endID string, maxDepth int) ([][]contracts.GraphNode, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("GraphService", "graph service is nil")
	}
	if strings.TrimSpace(startID) == "" || strings.TrimSpace(endID) == "" {
		return nil, sdkerr.NewInvalidInput("startID and endID cannot be empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.kgModel == nil || s.queryEng == nil {
		return nil, sdkerr.NewUnavailable("KnowledgeGraph", "knowledge graph query engine is not initialized")
	}

	if maxDepth <= 0 {
		maxDepth = 5
	}

	paths := s.queryEng.FindPaths(startID, endID, maxDepth)
	var out [][]contracts.GraphNode
	for _, p := range paths {
		if p == nil {
			continue
		}
		var pathNodes []contracts.GraphNode
		for _, e := range p.Entities() {
			if e != nil {
				pathNodes = append(pathNodes, *convertEntityToNode(e))
			}
		}
		out = append(out, pathNodes)
	}

	return out, nil
}

// ExportGraph exports the knowledge graph in JSON, Mermaid, or Graphviz format.
func (s *Service) ExportGraph(ctx context.Context, filter contracts.GraphFilter, format contracts.GraphExportFormat) (*contracts.GraphExportResult, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("GraphService", "graph service is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	nodes, err := s.TraverseNodes(ctx, "", filter)
	if err != nil {
		return nil, err
	}

	rels, err := s.TraverseRelationships(ctx, "", filter)
	if err != nil {
		return nil, err
	}

	var content string
	switch format {
	case contracts.ExportFormatMermaid:
		var sb strings.Builder
		sb.WriteString("graph TD\n")
		for _, n := range nodes {
			sb.WriteString(fmt.Sprintf("    %s[\"%s (%s)\"]\n", sanitizeID(n.ID), n.Name, n.Kind))
		}
		for _, r := range rels {
			sb.WriteString(fmt.Sprintf("    %s -->|%s| %s\n", sanitizeID(r.SourceID), r.Kind, sanitizeID(r.TargetID)))
		}
		content = sb.String()

	case contracts.ExportFormatGraphviz:
		var sb strings.Builder
		sb.WriteString("digraph LimoxelGraph {\n")
		sb.WriteString("    rankdir=LR;\n")
		for _, n := range nodes {
			sb.WriteString(fmt.Sprintf("    \"%s\" [label=\"%s\\n(%s)\"];\n", n.ID, n.Name, n.Kind))
		}
		for _, r := range rels {
			sb.WriteString(fmt.Sprintf("    \"%s\" -> \"%s\" [label=\"%s\"];\n", r.SourceID, r.TargetID, r.Kind))
		}
		sb.WriteString("}\n")
		content = sb.String()

	default: // JSON
		payload := struct {
			Nodes []contracts.GraphNode         `json:"nodes"`
			Edges []contracts.GraphRelationship `json:"edges"`
		}{
			Nodes: nodes,
			Edges: rels,
		}
		bytes, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_SERIALIZATION_FAILED", "failed to serialize graph json")
		}
		content = string(bytes)
	}

	return &contracts.GraphExportResult{
		Format:    format,
		Content:   content,
		NodeCount: len(nodes),
		EdgeCount: len(rels),
	}, nil
}

// Helpers

func convertEntityToNode(ent *knowledgegraph.GraphEntity) *contracts.GraphNode {
	if ent == nil {
		return nil
	}
	loc := ent.FilePath()
	if ent.Position() != nil {
		loc = fmt.Sprintf("%s:%d:%d", loc, ent.Position().Line(), ent.Position().Column())
	}
	return &contracts.GraphNode{
		ID:         ent.ID(),
		Kind:       string(ent.Type()),
		Name:       ent.Name(),
		Package:    ent.PackagePath(),
		Location:   loc,
		Properties: ent.Attributes(),
	}
}

func convertRelToContract(rel *knowledgegraph.GraphRelationship) *contracts.GraphRelationship {
	if rel == nil {
		return nil
	}
	return &contracts.GraphRelationship{
		ID:         rel.ID(),
		SourceID:   rel.SourceID(),
		TargetID:   rel.TargetID(),
		Kind:       string(rel.Kind()),
		Evidence:   rel.Evidence(),
		Properties: rel.Attributes(),
	}
}

func matchesEntityFilter(ent *knowledgegraph.GraphEntity, filter contracts.GraphFilter) bool {
	if ent == nil {
		return false
	}
	if len(filter.EntityTypes) > 0 && !containsString(filter.EntityTypes, string(ent.Type())) {
		return false
	}
	if filter.PackageScope != "" && !strings.HasPrefix(ent.PackagePath(), filter.PackageScope) {
		return false
	}
	return true
}

func containsString(list []string, val string) bool {
	for _, item := range list {
		if strings.EqualFold(item, val) {
			return true
		}
	}
	return false
}

func sanitizeID(id string) string {
	id = strings.ReplaceAll(id, ":", "_")
	id = strings.ReplaceAll(id, "/", "_")
	id = strings.ReplaceAll(id, "\\", "_")
	id = strings.ReplaceAll(id, ".", "_")
	id = strings.ReplaceAll(id, "-", "_")
	return id
}
