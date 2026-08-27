package graph_test

import (
	"context"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/knowledgegraph"
	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	"github.com/unhield/limoxel/internal/capabilities/sdk/graph"
)

func createTestGraphModel() *knowledgegraph.KnowledgeGraphModel {
	e1 := knowledgegraph.NewGraphEntity("repo:root", knowledgegraph.EntityRepository, "sample_repo", "", "root", nil, nil, "test")
	e2 := knowledgegraph.NewGraphEntity("pkg:math", knowledgegraph.EntityPackage, "math", "pkg/math", "pkg/math", nil, nil, "test")
	e3 := knowledgegraph.NewGraphEntity("sym:math.Add", knowledgegraph.EntitySymbol, "Add", "pkg/math", "pkg/math/math.go", nil, nil, "test")

	r1 := knowledgegraph.NewGraphRelationship("repo:root", "pkg:math", knowledgegraph.RelOwns, "discovery", "test", 1.0, nil)
	r2 := knowledgegraph.NewGraphRelationship("pkg:math", "sym:math.Add", knowledgegraph.RelOwns, "symbol_parser", "test", 1.0, nil)

	entities := []*knowledgegraph.GraphEntity{e1, e2, e3}
	rels := []*knowledgegraph.GraphRelationship{r1, r2}

	return knowledgegraph.NewKnowledgeGraphModel("sample_repo", entities, rels, nil, time.Now().UTC())
}

func TestGraphServiceOperations(t *testing.T) {
	ctx := context.Background()
	model := createTestGraphModel()
	svc := graph.NewService(model)

	// 1. GraphInfo
	nodes, edges, err := svc.GraphInfo(ctx)
	if err != nil {
		t.Fatalf("GraphInfo failed: %v", err)
	}
	if nodes != 3 || edges != 2 {
		t.Errorf("got nodes=%d, edges=%d, want 3, 2", nodes, edges)
	}

	// 2. GetNode
	node, err := svc.GetNode(ctx, "pkg:math")
	if err != nil {
		t.Fatalf("GetNode failed: %v", err)
	}
	if node.Name != "math" {
		t.Errorf("got node name %q, want math", node.Name)
	}

	// 3. TraverseNodes
	tNodes, err := svc.TraverseNodes(ctx, "repo:root", contracts.GraphFilter{MaxDepth: 2})
	if err != nil {
		t.Fatalf("TraverseNodes failed: %v", err)
	}
	if len(tNodes) == 0 {
		t.Errorf("expected traversed nodes")
	}

	// 4. TraverseRelationships
	tRels, err := svc.TraverseRelationships(ctx, "", contracts.GraphFilter{})
	if err != nil {
		t.Fatalf("TraverseRelationships failed: %v", err)
	}
	if len(tRels) != 2 {
		t.Errorf("expected 2 relationships, got %d", len(tRels))
	}

	// 5. GetNeighbors
	neighbors, err := svc.GetNeighbors(ctx, "pkg:math", "outbound")
	if err != nil {
		t.Fatalf("GetNeighbors failed: %v", err)
	}
	if len(neighbors) == 0 {
		t.Errorf("expected neighbors for pkg:math")
	}

	// 6. FindPaths
	paths, err := svc.FindPaths(ctx, "repo:root", "sym:math.Add", 3)
	if err != nil {
		t.Fatalf("FindPaths failed: %v", err)
	}
	if len(paths) == 0 {
		t.Errorf("expected at least 1 path between repo:root and sym:math.Add")
	}

	// 7. ExportGraph
	expJSON, err := svc.ExportGraph(ctx, contracts.GraphFilter{}, contracts.ExportFormatJSON)
	if err != nil {
		t.Fatalf("ExportGraph JSON failed: %v", err)
	}
	if expJSON.NodeCount != 3 {
		t.Errorf("expected 3 nodes in export, got %d", expJSON.NodeCount)
	}

	expMermaid, err := svc.ExportGraph(ctx, contracts.GraphFilter{}, contracts.ExportFormatMermaid)
	if err != nil {
		t.Fatalf("ExportGraph Mermaid failed: %v", err)
	}
	if expMermaid.Content == "" {
		t.Errorf("expected non-empty Mermaid export")
	}

	expGraphviz, err := svc.ExportGraph(ctx, contracts.GraphFilter{}, contracts.ExportFormatGraphviz)
	if err != nil {
		t.Fatalf("ExportGraph Graphviz failed: %v", err)
	}
	if expGraphviz.Content == "" {
		t.Errorf("expected non-empty Graphviz export")
	}
}

func TestGraphServiceErrorsAndNil(t *testing.T) {
	ctx := context.Background()
	svc := graph.NewService(nil)

	if _, _, err := svc.GraphInfo(ctx); err == nil {
		t.Errorf("expected error on uninitialized GraphInfo")
	}
	if _, err := svc.GetNode(ctx, ""); err == nil {
		t.Errorf("expected error for empty nodeID")
	}
	if _, err := svc.GetNode(ctx, "nonexistent"); err == nil {
		t.Errorf("expected error on uninitialized GetNode")
	}
	if _, err := svc.GetRelationship(ctx, ""); err == nil {
		t.Errorf("expected error for empty relID")
	}
	if _, err := svc.TraverseNodes(ctx, "", contracts.GraphFilter{}); err == nil {
		t.Errorf("expected error on uninitialized TraverseNodes")
	}
	if _, err := svc.GetNeighbors(ctx, "", "outbound"); err == nil {
		t.Errorf("expected error for empty nodeID")
	}
	if _, err := svc.FindPaths(ctx, "", "", 1); err == nil {
		t.Errorf("expected error for empty path endpoints")
	}

	var nilSvc *graph.Service
	if _, _, err := nilSvc.GraphInfo(ctx); err == nil {
		t.Errorf("expected error on typed nil service GraphInfo")
	}
	if _, err := nilSvc.GetNode(ctx, "node"); err == nil {
		t.Errorf("expected error on typed nil service GetNode")
	}
	if _, err := nilSvc.GetRelationship(ctx, "rel"); err == nil {
		t.Errorf("expected error on typed nil service GetRelationship")
	}
	if _, err := nilSvc.TraverseNodes(ctx, "", contracts.GraphFilter{}); err == nil {
		t.Errorf("expected error on typed nil service TraverseNodes")
	}
	if _, err := nilSvc.TraverseRelationships(ctx, "", contracts.GraphFilter{}); err == nil {
		t.Errorf("expected error on typed nil service TraverseRelationships")
	}
	if _, err := nilSvc.GetNeighbors(ctx, "node", "outbound"); err == nil {
		t.Errorf("expected error on typed nil service GetNeighbors")
	}
	if _, err := nilSvc.FindPaths(ctx, "a", "b", 1); err == nil {
		t.Errorf("expected error on typed nil service FindPaths")
	}
	if _, err := nilSvc.ExportGraph(ctx, contracts.GraphFilter{}, contracts.ExportFormatJSON); err == nil {
		t.Errorf("expected error on typed nil service ExportGraph")
	}
}
