package api

import "context"

// GraphAPI defines the developer-facing interface for Knowledge Graph inspection and traversal.
type GraphAPI interface {
	// GraphInfo returns the total node and relationship counts in the graph.
	GraphInfo(ctx context.Context) (totalNodes int, totalRelationships int, err error)

	// GetNode retrieves a single graph node by identifier.
	GetNode(ctx context.Context, nodeID string) (*GraphNode, error)

	// GetRelationship retrieves a single graph relationship edge by identifier.
	GetRelationship(ctx context.Context, relID string) (*GraphRelationship, error)

	// TraverseNodes performs bounded graph traversal starting from a node.
	TraverseNodes(ctx context.Context, startNodeID string, filter GraphFilter) ([]GraphNode, error)

	// TraverseRelationships retrieves edges along a traversal path.
	TraverseRelationships(ctx context.Context, startNodeID string, filter GraphFilter) ([]GraphRelationship, error)

	// GetNeighbors returns immediately connected nodes in the specified direction.
	GetNeighbors(ctx context.Context, nodeID string, direction string, kinds ...string) ([]GraphNode, error)

	// ExportGraph serializes the knowledge graph or filtered subgraph into JSON, Mermaid, or Graphviz.
	ExportGraph(ctx context.Context, filter GraphFilter, format GraphExportFormat) (*GraphExportResult, error)
}
