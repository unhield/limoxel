package graph

import (
	"fmt"
	"time"
)

// ValidationEngine evaluates knowledge graph integrity and consistency.
type ValidationEngine struct {
	graph *KnowledgeGraph
}

// NewValidationEngine constructs a ValidationEngine.
func NewValidationEngine(graph *KnowledgeGraph) *ValidationEngine {
	return &ValidationEngine{graph: graph}
}

// Validate executes all integrity checks across nodes and relationships.
func (ve *ValidationEngine) Validate() *ValidationReport {
	if ve == nil || ve.graph == nil {
		return NewValidationReport(nil)
	}

	var issues []*ValidationIssue

	// 1. Missing Nodes validation
	for _, r := range ve.graph.AllRelationships() {
		if !ve.graph.HasNode(r.SourceID()) {
			issues = append(issues, NewValidationIssue(
				ValMissingNode,
				"GRAPH_MISSING_SOURCE_NODE",
				fmt.Sprintf("relationship %s references non-existent source node %s", r.ID(), r.SourceID()),
				r.ID(),
				r.SourceID(),
				r.TargetID(),
				"",
			))
		}
		if !ve.graph.HasNode(r.TargetID()) {
			issues = append(issues, NewValidationIssue(
				ValMissingNode,
				"GRAPH_MISSING_TARGET_NODE",
				fmt.Sprintf("relationship %s references non-existent target node %s", r.ID(), r.TargetID()),
				r.ID(),
				r.SourceID(),
				r.TargetID(),
				"",
			))
		}
	}

	// 2. Invalid Edges validation (type compatibility rules)
	for _, r := range ve.graph.AllRelationships() {
		srcNode := ve.graph.NodeByID(r.SourceID())
		tgtNode := ve.graph.NodeByID(r.TargetID())

		if srcNode != nil && tgtNode != nil {
			if !isValidEdgeType(r.Type(), srcNode.Type(), tgtNode.Type()) {
				issues = append(issues, NewValidationIssue(
					ValInvalidEdge,
					"GRAPH_INVALID_EDGE_TYPE",
					fmt.Sprintf("invalid %s relationship from %s (%s) to %s (%s)", r.Type(), srcNode.ID(), srcNode.Type(), tgtNode.ID(), tgtNode.Type()),
					r.ID(),
					r.SourceID(),
					r.TargetID(),
					"",
				))
			}
		}
	}

	// 3. Duplicate Edges validation
	seenRels := make(map[string]bool)
	for _, r := range ve.graph.AllRelationships() {
		if seenRels[r.ID()] {
			issues = append(issues, NewValidationIssue(
				ValDuplicateEdge,
				"GRAPH_DUPLICATE_EDGE",
				fmt.Sprintf("duplicate relationship ID %s detected", r.ID()),
				r.ID(),
				r.SourceID(),
				r.TargetID(),
				"",
			))
		}
		seenRels[r.ID()] = true
	}

	// 4. Orphan Nodes validation (isolated nodes with 0 connections)
	for _, n := range ve.graph.AllNodes() {
		outCount := len(ve.graph.OutboundRelationships(n.ID()))
		inCount := len(ve.graph.InboundRelationships(n.ID()))

		if outCount == 0 && inCount == 0 {
			// Repository nodes with 0 children or isolated configs/docs
			issues = append(issues, NewValidationIssue(
				ValOrphanNode,
				"GRAPH_ORPHAN_NODE",
				fmt.Sprintf("node %s (%s) has no incoming or outgoing relationships", n.ID(), n.Type()),
				"",
				"",
				"",
				n.ID(),
			))
		}
	}

	return NewValidationReport(issues)
}

// ValidatePerformance runs basic latency validation for node lookups and neighbor traversal.
func (ve *ValidationEngine) ValidatePerformance() (lookupDuration time.Duration, traversalDuration time.Duration) {
	if ve == nil || ve.graph == nil || ve.graph.TotalNodes() == 0 {
		return 0, 0
	}

	nodes := ve.graph.AllNodes()
	qe := ve.graph.Query()

	// Benchmark node lookup
	startLookup := time.Now()
	for _, n := range nodes {
		_, _ = qe.LookupNode(n.ID())
	}
	lookupDuration = time.Since(startLookup)

	// Benchmark neighbor traversal
	startTrav := time.Now()
	for _, n := range nodes {
		_ = qe.Neighbors(n.ID(), DirBoth)
	}
	traversalDuration = time.Since(startTrav)

	return lookupDuration, traversalDuration
}

func isValidEdgeType(relType RelationshipType, srcType, tgtType NodeType) bool {
	switch relType {
	case RelContains:
		// Repo -> Mod, Mod -> Pkg, Repo -> Pkg, Pkg -> File, File -> Sym, Sym -> Sym
		if srcType == NodeRepository && (tgtType == NodeModule || tgtType == NodePackage || tgtType == NodeDoc || tgtType == NodeConfig) {
			return true
		}
		if srcType == NodeModule && (tgtType == NodePackage || tgtType == NodeFile || tgtType == NodeDoc || tgtType == NodeConfig) {
			return true
		}
		if srcType == NodePackage && (tgtType == NodeFile || tgtType == NodeDoc || tgtType == NodeConfig) {
			return true
		}
		if srcType == NodeFile && (tgtType == NodeSymbol || tgtType == NodeDoc || tgtType == NodeConfig) {
			return true
		}
		if srcType == NodeSymbol && tgtType == NodeSymbol {
			return true
		}
		return false

	case RelImports:
		// File -> Pkg, Pkg -> Pkg, File -> File
		return (srcType == NodeFile || srcType == NodePackage) && (tgtType == NodePackage || tgtType == NodeFile)

	case RelImplements:
		// Symbol -> Symbol (struct -> interface)
		return srcType == NodeSymbol && tgtType == NodeSymbol

	case RelCalls:
		// Symbol -> Symbol (func/method -> func/method)
		return srcType == NodeSymbol && tgtType == NodeSymbol

	case RelReferences:
		// Symbol -> Symbol, File -> Symbol, Pkg -> Symbol
		return (srcType == NodeSymbol || srcType == NodeFile || srcType == NodePackage) && tgtType == NodeSymbol

	case RelDependsOn:
		// Repo/Mod/Pkg/File -> Pkg/Mod/File
		return true

	case RelDocuments:
		// Doc -> Repo, Doc -> Mod, Doc -> Pkg, Doc -> File, Doc -> Symbol
		return srcType == NodeDoc

	case RelConfigures:
		// Config -> Repo, Config -> Mod, Config -> Pkg, Config -> File
		return srcType == NodeConfig
	}

	return false
}
