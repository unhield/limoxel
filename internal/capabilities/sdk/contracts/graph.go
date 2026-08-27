package contracts

import (
	"context"

	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// GraphNode represents a public node in the Limoxel Knowledge Graph.
type GraphNode struct {
	ID         string            `json:"id"`
	Kind       string            `json:"kind"`
	Name       string            `json:"name"`
	Package    string            `json:"package,omitempty"`
	Location   string            `json:"location,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

// GraphRelationship represents a directed edge between two knowledge graph nodes.
type GraphRelationship struct {
	ID         string            `json:"id"`
	SourceID   string            `json:"source_id"`
	TargetID   string            `json:"target_id"`
	Kind       string            `json:"kind"`
	Evidence   string            `json:"evidence,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

// GraphFilter defines filtering constraints for graph queries and export operations.
type GraphFilter struct {
	EntityTypes       []string `json:"entity_types,omitempty"`
	RelationshipKinds []string `json:"relationship_kinds,omitempty"`
	PackageScope      string   `json:"package_scope,omitempty"`
	MaxDepth          int      `json:"max_depth,omitempty"`
}

// GraphExportFormat defines supported serialization formats for knowledge graph export.
type GraphExportFormat string

const (
	// ExportFormatJSON exports graph entities and edges in structured JSON.
	ExportFormatJSON GraphExportFormat = "json"

	// ExportFormatMermaid exports the graph as a Mermaid flowchart definition.
	ExportFormatMermaid GraphExportFormat = "mermaid"

	// ExportFormatGraphviz exports the graph as a Graphviz DOT digraph.
	ExportFormatGraphviz GraphExportFormat = "graphviz"
)

// GraphExportResult encapsulates the output of a graph export operation.
type GraphExportResult struct {
	Format    GraphExportFormat `json:"format"`
	Content   string            `json:"content"`
	NodeCount int               `json:"node_count"`
	EdgeCount int               `json:"edge_count"`
}

// GraphContract defines the public contract for knowledge graph traversal, filtering, and export.
type GraphContract interface {
	Contract
	GraphInfo(ctx context.Context) (totalNodes int, totalEdges int, err error)
	GetNode(ctx context.Context, nodeID string) (*GraphNode, error)
	GetRelationship(ctx context.Context, relID string) (*GraphRelationship, error)
	TraverseNodes(ctx context.Context, startNodeID string, filter GraphFilter) ([]GraphNode, error)
	TraverseRelationships(ctx context.Context, startNodeID string, filter GraphFilter) ([]GraphRelationship, error)
	GetNeighbors(ctx context.Context, nodeID string, direction string, kinds ...string) ([]GraphNode, error)
	FindPaths(ctx context.Context, startID, endID string, maxDepth int) ([][]GraphNode, error)
	ExportGraph(ctx context.Context, filter GraphFilter, format GraphExportFormat) (*GraphExportResult, error)
}

// DefaultGraphContractMetadata returns default contract descriptor for Knowledge Graph operations.
func DefaultGraphContractMetadata() BaseContract {
	return NewBaseContract(
		"GraphContract",
		lifecycle.CapabilityGraph,
		version.SemVer{Major: 1, Minor: 0, Patch: 0},
		lifecycle.StateSupported,
		"Provides public knowledge graph inspection, traversal, filtering, and visual diagram export.",
	)
}
