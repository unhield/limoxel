package dependency

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
)

// DependencyModel represents the consolidated, immutable result of repository dependency analysis.
type DependencyModel struct {
	root        string
	inventory   *DependencyInventory
	graph       *DependencyGraph
	licenses    *LicenseInventory
	cycles      [][]string
	orphans     []string
	maxDepth    int
	diagnostics []*discovery.Diagnostic
}

// NewDependencyModel constructs an immutable DependencyModel with defensively copied collections.
func NewDependencyModel(
	root string,
	inventory *DependencyInventory,
	graph *DependencyGraph,
	licenses *LicenseInventory,
	cycles [][]string,
	orphans []string,
	maxDepth int,
	diagnostics []*discovery.Diagnostic,
) *DependencyModel {
	// Defensive copy & deterministic sort for cycles
	cycleList := make([][]string, len(cycles))
	for i, c := range cycles {
		cycleCopy := make([]string, len(c))
		copy(cycleCopy, c)
		cycleList[i] = cycleCopy
	}
	sort.Slice(cycleList, func(i, j int) bool {
		c1 := fmt.Sprintf("%v", cycleList[i])
		c2 := fmt.Sprintf("%v", cycleList[j])
		return c1 < c2
	})

	// Defensive copy & deterministic sort for orphans
	orphanList := make([]string, len(orphans))
	copy(orphanList, orphans)
	sort.Strings(orphanList)

	// Defensive copy for diagnostics
	diagList := make([]*discovery.Diagnostic, len(diagnostics))
	copy(diagList, diagnostics)

	return &DependencyModel{
		root:        filepath.ToSlash(filepath.Clean(root)),
		inventory:   inventory,
		graph:       graph,
		licenses:    licenses,
		cycles:      cycleList,
		orphans:     orphanList,
		maxDepth:    maxDepth,
		diagnostics: diagList,
	}
}

// Root returns the cleaned repository root path.
func (dm *DependencyModel) Root() string {
	if dm == nil {
		return ""
	}
	return dm.root
}

// Inventory returns the dependency inventory.
func (dm *DependencyModel) Inventory() *DependencyInventory {
	if dm == nil {
		return nil
	}
	return dm.inventory
}

// Graph returns the directed dependency graph.
func (dm *DependencyModel) Graph() *DependencyGraph {
	if dm == nil {
		return nil
	}
	return dm.graph
}

// Licenses returns the license inventory.
func (dm *DependencyModel) Licenses() *LicenseInventory {
	if dm == nil {
		return nil
	}
	return dm.licenses
}

// Cycles returns a defensive copy of detected dependency cycles.
func (dm *DependencyModel) Cycles() [][]string {
	if dm == nil || len(dm.cycles) == 0 {
		return nil
	}
	cloned := make([][]string, len(dm.cycles))
	for i, c := range dm.cycles {
		cCopy := make([]string, len(c))
		copy(cCopy, c)
		cloned[i] = cCopy
	}
	return cloned
}

// Orphans returns a defensive copy of detected orphan packages / components.
func (dm *DependencyModel) Orphans() []string {
	if dm == nil || len(dm.orphans) == 0 {
		return nil
	}
	cloned := make([]string, len(dm.orphans))
	copy(cloned, dm.orphans)
	return cloned
}

// MaxDepth returns the maximum dependency depth in the graph.
func (dm *DependencyModel) MaxDepth() int {
	if dm == nil {
		return 0
	}
	return dm.maxDepth
}

// Diagnostics returns a defensive copy of diagnostics generated during analysis.
func (dm *DependencyModel) Diagnostics() []*discovery.Diagnostic {
	if dm == nil || len(dm.diagnostics) == 0 {
		return nil
	}
	cloned := make([]*discovery.Diagnostic, len(dm.diagnostics))
	copy(cloned, dm.diagnostics)
	return cloned
}

// String returns a human-readable summary of the DependencyModel.
func (dm *DependencyModel) String() string {
	if dm == nil {
		return ""
	}
	totalDeps := 0
	if dm.inventory != nil {
		totalDeps = dm.inventory.TotalCount()
	}
	nodeCount := 0
	edgeCount := 0
	if dm.graph != nil {
		nodeCount = dm.graph.NodeCount()
		edgeCount = dm.graph.EdgeCount()
	}
	return fmt.Sprintf("DependencyModel<%s>[dependencies=%d, nodes=%d, edges=%d, cycles=%d, orphans=%d, maxDepth=%d]",
		dm.root, totalDeps, nodeCount, edgeCount, len(dm.cycles), len(dm.orphans), dm.maxDepth)
}
