package query

import (
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/graph"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
)

// GraphAPI provides deterministic querying, traversal, dependency, and call graph lookup operations.
type GraphAPI struct {
	kg        *graph.KnowledgeGraph
	depModel  *dependency.DependencyModel
	xrefModel *xref.XRefModel
}

// NewGraphAPI constructs a GraphAPI instance.
func NewGraphAPI(
	kg *graph.KnowledgeGraph,
	depModel *dependency.DependencyModel,
	xrefModel *xref.XRefModel,
) *GraphAPI {
	return &GraphAPI{
		kg:        kg,
		depModel:  depModel,
		xrefModel: xrefModel,
	}
}

// GetNode retrieves a single graph node by its canonical ID.
func (api *GraphAPI) GetNode(id string) (*GraphNodeDTO, error) {
	if api == nil || api.kg == nil {
		return nil, ErrAnalysisUnavailable
	}
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return nil, ErrInvalidInput
	}

	n := api.kg.NodeByID(cleanID)
	if n == nil {
		return nil, ErrNodeNotFound
	}

	return toGraphNodeDTO(n), nil
}

// LookupRelationships finds direct relationships between source and target nodes with optional type filter.
func (api *GraphAPI) LookupRelationships(sourceID, targetID string, relType graph.RelationshipType) ([]*RelationshipDTO, error) {
	if api == nil || api.kg == nil {
		return nil, ErrAnalysisUnavailable
	}

	sID := strings.TrimSpace(sourceID)
	tID := strings.TrimSpace(targetID)

	var rels []*graph.Relationship
	if sID != "" && tID != "" {
		rels = api.kg.RelationshipsBetween(sID, tID)
	} else if sID != "" {
		rels = api.kg.OutboundRelationships(sID)
	} else if tID != "" {
		rels = api.kg.InboundRelationships(tID)
	} else {
		rels = api.kg.AllRelationships()
	}

	var dtos []*RelationshipDTO
	for _, r := range rels {
		if r != nil {
			if relType == "" || r.Type() == relType {
				dtos = append(dtos, toRelationshipDTO(r))
			}
		}
	}

	sort.Slice(dtos, func(i, j int) bool {
		return dtos[i].ID() < dtos[j].ID()
	})

	return dtos, nil
}

// Traverse performs a bounded graph traversal from a starting node.
func (api *GraphAPI) Traverse(
	startNodeID string,
	dir graph.Direction,
	maxDepth int,
	relTypes ...graph.RelationshipType,
) (*TraversalResultDTO, error) {
	if api == nil || api.kg == nil {
		return nil, ErrAnalysisUnavailable
	}

	cleanID := strings.TrimSpace(startNodeID)
	if cleanID == "" {
		return nil, ErrInvalidInput
	}
	if maxDepth > 100 {
		return nil, ErrMaxDepthExceeded
	}

	nodes, rels, err := api.kg.Query().TraversePath(cleanID, dir, maxDepth, relTypes...)
	if err != nil {
		if err == graph.ErrNodeNotFound {
			return nil, ErrNodeNotFound
		}
		if err == graph.ErrMaxDepthExceeded {
			return nil, ErrMaxDepthExceeded
		}
		return nil, WrapQueryError(ErrCatInternal, "ERR_TRAVERSAL_FAILED", "graph traversal failed", err)
	}

	var nodeDTOs []*GraphNodeDTO
	for _, n := range nodes {
		if n != nil {
			nodeDTOs = append(nodeDTOs, toGraphNodeDTO(n))
		}
	}

	var relDTOs []*RelationshipDTO
	for _, r := range rels {
		if r != nil {
			relDTOs = append(relDTOs, toRelationshipDTO(r))
		}
	}

	return NewTraversalResultDTO(cleanID, dir, maxDepth, nodeDTOs, relDTOs), nil
}

// LookupDependencies retrieves dependencies for the repository or a specific package name.
func (api *GraphAPI) LookupDependencies(pkgOrDepName string) ([]*DependencyDTO, error) {
	if api == nil {
		return nil, ErrAnalysisUnavailable
	}

	cleanName := strings.TrimSpace(pkgOrDepName)

	if api.depModel != nil && api.depModel.Inventory() != nil {
		var dtos []*DependencyDTO
		for _, dep := range api.depModel.Inventory().AllDependencies() {
			if dep == nil {
				continue
			}
			if cleanName == "" || strings.EqualFold(dep.Name(), cleanName) {
				dtos = append(dtos, NewDependencyDTO(
					dep.Name(),
					dep.Version().String(),
					dep.IsDirect(),
					string(dep.Type()),
				))
			}
		}

		sort.Slice(dtos, func(i, j int) bool {
			return dtos[i].Name() < dtos[j].Name()
		})
		return dtos, nil
	}

	// Fallback to KnowledgeGraph if DependencyModel not directly provided
	if api.kg != nil {
		var dtos []*DependencyDTO
		for _, r := range api.kg.AllRelationships() {
			if r.Type() == graph.RelDependsOn {
				targetNode := api.kg.NodeByID(r.TargetID())
				name := r.TargetID()
				if targetNode != nil && targetNode.Name() != "" {
					name = targetNode.Name()
				}
				if cleanName == "" || strings.EqualFold(name, cleanName) {
					version := ""
					isDirect := true
					if targetNode != nil {
						version = targetNode.Metadata()["version"]
						if targetNode.Metadata()["direct"] == "false" {
							isDirect = false
						}
					}
					dtos = append(dtos, NewDependencyDTO(name, version, isDirect, "manifest"))
				}
			}
		}

		sort.Slice(dtos, func(i, j int) bool {
			return dtos[i].Name() < dtos[j].Name()
		})
		return dtos, nil
	}

	return nil, ErrAnalysisUnavailable
}

// LookupCallGraph retrieves function/method invocation edges for a symbol.
func (api *GraphAPI) LookupCallGraph(symbolID string, dir CallDirection) ([]*CallEdgeDTO, error) {
	if api == nil {
		return nil, ErrAnalysisUnavailable
	}
	cleanID := strings.TrimSpace(symbolID)
	if cleanID == "" {
		return nil, ErrInvalidInput
	}

	if api.xrefModel != nil && api.xrefModel.CallGraph() != nil {
		var edges []*xref.CallEdge
		switch dir {
		case CallDirectionOutbound:
			edges = api.xrefModel.CallGraph().CalleeEdges(cleanID)
		case CallDirectionInbound:
			edges = api.xrefModel.CallGraph().CallerEdges(cleanID)
		case CallDirectionBoth, "":
			callers := api.xrefModel.CallGraph().CallerEdges(cleanID)
			callees := api.xrefModel.CallGraph().CalleeEdges(cleanID)
			edges = append(callers, callees...)
		default:
			return nil, ErrInvalidInput
		}

		var dtos []*CallEdgeDTO
		for _, e := range edges {
			if e != nil {
				line := 0
				if e.Position() != nil {
					line = e.Position().Line()
				}
				dtos = append(dtos, NewCallEdgeDTO(
					e.CallerID(),
					e.CalleeID(),
					string(e.Kind()),
					e.FilePath(),
					line,
				))
			}
		}

		sort.Slice(dtos, func(i, j int) bool {
			if dtos[i].CallerID() != dtos[j].CallerID() {
				return dtos[i].CallerID() < dtos[j].CallerID()
			}
			return dtos[i].CalleeID() < dtos[j].CalleeID()
		})
		return dtos, nil
	}

	// Fallback to KnowledgeGraph
	if api.kg != nil {
		symNodeID := cleanID
		if !strings.HasPrefix(symNodeID, "sym:") {
			symNodeID = "sym:" + cleanID
		}

		var rels []*graph.Relationship
		switch dir {
		case CallDirectionOutbound:
			for _, r := range api.kg.OutboundRelationships(symNodeID) {
				if r.Type() == graph.RelCalls {
					rels = append(rels, r)
				}
			}
		case CallDirectionInbound:
			for _, r := range api.kg.InboundRelationships(symNodeID) {
				if r.Type() == graph.RelCalls {
					rels = append(rels, r)
				}
			}
		case CallDirectionBoth, "":
			for _, r := range api.kg.OutboundRelationships(symNodeID) {
				if r.Type() == graph.RelCalls {
					rels = append(rels, r)
				}
			}
			for _, r := range api.kg.InboundRelationships(symNodeID) {
				if r.Type() == graph.RelCalls {
					rels = append(rels, r)
				}
			}
		}

		var dtos []*CallEdgeDTO
		for _, r := range rels {
			src := strings.TrimPrefix(r.SourceID(), "sym:")
			tgt := strings.TrimPrefix(r.TargetID(), "sym:")
			kind := r.Metadata()["call_kind"]
			if kind == "" {
				kind = "direct"
			}
			dtos = append(dtos, NewCallEdgeDTO(src, tgt, kind, "", 0))
		}

		sort.Slice(dtos, func(i, j int) bool {
			if dtos[i].CallerID() != dtos[j].CallerID() {
				return dtos[i].CallerID() < dtos[j].CallerID()
			}
			return dtos[i].CalleeID() < dtos[j].CalleeID()
		})
		return dtos, nil
	}

	return nil, ErrAnalysisUnavailable
}

// Helpers
func toGraphNodeDTO(n *graph.Node) *GraphNodeDTO {
	if n == nil {
		return nil
	}
	return NewGraphNodeDTO(
		n.ID(),
		n.Type(),
		n.Name(),
		n.Path(),
		n.Module(),
		n.Package(),
		n.Metadata(),
	)
}

func toRelationshipDTO(r *graph.Relationship) *RelationshipDTO {
	if r == nil {
		return nil
	}
	var provs []string
	for _, p := range r.Provenance() {
		provs = append(provs, string(p))
	}
	return NewRelationshipDTO(
		r.ID(),
		r.Type(),
		r.SourceID(),
		r.TargetID(),
		provs,
		r.Metadata(),
	)
}
