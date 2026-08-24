package graph

import (
	"path/filepath"
	"sort"
	"strings"
)

// KnowledgeGraph represents the unified, deterministic engineering knowledge graph.
type KnowledgeGraph struct {
	repoRoot          string
	schemaVersion     string
	nodes             map[string]*Node
	nodesList         []*Node
	nodesByType       map[NodeType][]*Node
	nodesByName       map[string][]*Node
	relationships     map[string]*Relationship
	relationshipsList []*Relationship
	outEdges          map[string][]*Relationship
	inEdges           map[string][]*Relationship
	edgesBetween      map[string][]*Relationship
	validation        *ValidationReport
}

// NewKnowledgeGraph constructs an immutable KnowledgeGraph from nodes and relationships.
func NewKnowledgeGraph(
	repoRoot string,
	nodes []*Node,
	relationships []*Relationship,
) *KnowledgeGraph {
	cleanRoot := filepath.ToSlash(filepath.Clean(strings.TrimSpace(repoRoot)))
	schemaVer := "1.0.0"

	nodeMap := make(map[string]*Node)
	var nList []*Node
	byType := make(map[NodeType][]*Node)
	byName := make(map[string][]*Node)

	for _, n := range nodes {
		if n == nil || n.ID() == "" {
			continue
		}
		if _, exists := nodeMap[n.ID()]; !exists {
			nodeMap[n.ID()] = n
			nList = append(nList, n)
			byType[n.Type()] = append(byType[n.Type()], n)
			if n.Name() != "" {
				byName[n.Name()] = append(byName[n.Name()], n)
			}
		}
	}

	sort.Slice(nList, func(i, j int) bool {
		return nList[i].ID() < nList[j].ID()
	})

	for t := range byType {
		sort.Slice(byType[t], func(i, j int) bool {
			return byType[t][i].ID() < byType[t][j].ID()
		})
	}
	for name := range byName {
		sort.Slice(byName[name], func(i, j int) bool {
			return byName[name][i].ID() < byName[name][j].ID()
		})
	}

	relMap := make(map[string]*Relationship, len(relationships))
	for _, r := range relationships {
		if r == nil || r.ID() == "" {
			continue
		}
		if existing, exists := relMap[r.ID()]; exists {
			// Merge provenance if duplicate relationship ID
			mergedProv := append(existing.Provenance(), r.Provenance()...)
			relMap[r.ID()] = NewRelationship(
				r.Type(),
				r.SourceID(),
				r.TargetID(),
				mergedProv,
				r.Metadata(),
			)
			continue
		}

		relMap[r.ID()] = r
	}

	var rList []*Relationship
	outMap := make(map[string][]*Relationship)
	inMap := make(map[string][]*Relationship)
	betweenMap := make(map[string][]*Relationship)

	for _, r := range relMap {
		rList = append(rList, r)
		outMap[r.SourceID()] = append(outMap[r.SourceID()], r)
		inMap[r.TargetID()] = append(inMap[r.TargetID()], r)
		betweenKey := r.SourceID() + "->" + r.TargetID()
		betweenMap[betweenKey] = append(betweenMap[betweenKey], r)
	}

	sort.Slice(rList, func(i, j int) bool {
		return rList[i].ID() < rList[j].ID()
	})

	for s := range outMap {
		sort.Slice(outMap[s], func(i, j int) bool {
			return outMap[s][i].ID() < outMap[s][j].ID()
		})
	}
	for t := range inMap {
		sort.Slice(inMap[t], func(i, j int) bool {
			return inMap[t][i].ID() < inMap[t][j].ID()
		})
	}
	for k := range betweenMap {
		sort.Slice(betweenMap[k], func(i, j int) bool {
			return betweenMap[k][i].ID() < betweenMap[k][j].ID()
		})
	}

	kg := &KnowledgeGraph{
		repoRoot:          cleanRoot,
		schemaVersion:     schemaVer,
		nodes:             nodeMap,
		nodesList:         nList,
		nodesByType:       byType,
		nodesByName:       byName,
		relationships:     relMap,
		relationshipsList: rList,
		outEdges:          outMap,
		inEdges:           inMap,
		edgesBetween:      betweenMap,
	}

	// Validate graph integrity
	validator := NewValidationEngine(kg)
	kg.validation = validator.Validate()

	return kg
}

// RepositoryRoot returns the normalized repository root path.
func (kg *KnowledgeGraph) RepositoryRoot() string {
	if kg == nil {
		return ""
	}
	return kg.repoRoot
}

// SchemaVersion returns the graph schema version.
func (kg *KnowledgeGraph) SchemaVersion() string {
	if kg == nil {
		return ""
	}
	return kg.schemaVersion
}

// NodeByID returns the node matching the given ID, or nil if not found.
func (kg *KnowledgeGraph) NodeByID(id string) *Node {
	if kg == nil || kg.nodes == nil {
		return nil
	}
	return kg.nodes[strings.TrimSpace(id)]
}

// HasNode returns true if a node with the given ID exists.
func (kg *KnowledgeGraph) HasNode(id string) bool {
	return kg.NodeByID(id) != nil
}

// NodesByType returns all nodes of a specific type, deterministically sorted.
func (kg *KnowledgeGraph) NodesByType(t NodeType) []*Node {
	if kg == nil || kg.nodesByType == nil {
		return nil
	}
	list := kg.nodesByType[t]
	if list == nil {
		return nil
	}
	cloned := make([]*Node, len(list))
	copy(cloned, list)
	return cloned
}

// NodesByName returns nodes matching a display name, deterministically sorted.
func (kg *KnowledgeGraph) NodesByName(name string) []*Node {
	if kg == nil || kg.nodesByName == nil {
		return nil
	}
	list := kg.nodesByName[strings.TrimSpace(name)]
	if list == nil {
		return nil
	}
	cloned := make([]*Node, len(list))
	copy(cloned, list)
	return cloned
}

// AllNodes returns a defensive copy of all graph nodes, deterministically sorted.
func (kg *KnowledgeGraph) AllNodes() []*Node {
	if kg == nil || kg.nodesList == nil {
		return nil
	}
	cloned := make([]*Node, len(kg.nodesList))
	copy(cloned, kg.nodesList)
	return cloned
}

// TotalNodes returns the total number of nodes in the graph.
func (kg *KnowledgeGraph) TotalNodes() int {
	if kg == nil {
		return 0
	}
	return len(kg.nodesList)
}

// RelationshipByID returns the relationship with the given ID, or nil.
func (kg *KnowledgeGraph) RelationshipByID(id string) *Relationship {
	if kg == nil || kg.relationships == nil {
		return nil
	}
	return kg.relationships[strings.TrimSpace(id)]
}

// AllRelationships returns a defensive copy of all relationships, deterministically sorted.
func (kg *KnowledgeGraph) AllRelationships() []*Relationship {
	if kg == nil || kg.relationshipsList == nil {
		return nil
	}
	cloned := make([]*Relationship, len(kg.relationshipsList))
	copy(cloned, kg.relationshipsList)
	return cloned
}

// TotalRelationships returns the total number of relationships in the graph.
func (kg *KnowledgeGraph) TotalRelationships() int {
	if kg == nil {
		return 0
	}
	return len(kg.relationshipsList)
}

// OutboundRelationships returns all outgoing relationships from a source node.
func (kg *KnowledgeGraph) OutboundRelationships(sourceID string) []*Relationship {
	if kg == nil || kg.outEdges == nil {
		return nil
	}
	list := kg.outEdges[strings.TrimSpace(sourceID)]
	if list == nil {
		return nil
	}
	cloned := make([]*Relationship, len(list))
	copy(cloned, list)
	return cloned
}

// InboundRelationships returns all incoming relationships to a target node.
func (kg *KnowledgeGraph) InboundRelationships(targetID string) []*Relationship {
	if kg == nil || kg.inEdges == nil {
		return nil
	}
	list := kg.inEdges[strings.TrimSpace(targetID)]
	if list == nil {
		return nil
	}
	cloned := make([]*Relationship, len(list))
	copy(cloned, list)
	return cloned
}

// RelationshipsBetween returns relationships directly connecting sourceID -> targetID.
func (kg *KnowledgeGraph) RelationshipsBetween(sourceID, targetID string) []*Relationship {
	if kg == nil || kg.edgesBetween == nil {
		return nil
	}
	key := strings.TrimSpace(sourceID) + "->" + strings.TrimSpace(targetID)
	list := kg.edgesBetween[key]
	if list == nil {
		return nil
	}
	cloned := make([]*Relationship, len(list))
	copy(cloned, list)
	return cloned
}

// Validation returns the graph validation report.
func (kg *KnowledgeGraph) Validation() *ValidationReport {
	if kg == nil {
		return nil
	}
	return kg.validation
}

// Query returns a new QueryEngine bound to this graph.
func (kg *KnowledgeGraph) Query() *QueryEngine {
	return NewQueryEngine(kg)
}

// Export returns a new ExportEngine bound to this graph.
func (kg *KnowledgeGraph) Export() *ExportEngine {
	return NewExportEngine(kg)
}
