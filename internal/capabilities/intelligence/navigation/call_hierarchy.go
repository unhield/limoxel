package navigation

import (
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
)

// CallHierarchyNavigator provides deterministic call hierarchy traversal, cycle detection, and call depth analysis.
type CallHierarchyNavigator struct {
	symbolDB  *symbol.SymbolDatabase
	xrefModel *xref.XRefModel
}

// NewCallHierarchyNavigator constructs a CallHierarchyNavigator.
func NewCallHierarchyNavigator(symDB *symbol.SymbolDatabase, xrefModel *xref.XRefModel) *CallHierarchyNavigator {
	return &CallHierarchyNavigator{
		symbolDB:  symDB,
		xrefModel: xrefModel,
	}
}

// GetIncomingCalls identifies the functions or methods that call the specified function.
func (n *CallHierarchyNavigator) GetIncomingCalls(symbolID string) ([]*CallHierarchyNode, error) {
	cleanID := strings.TrimSpace(symbolID)
	if cleanID == "" {
		return nil, ErrEmptyTarget
	}

	var callers []*CallHierarchyNode
	callerMap := make(map[string]bool)

	if n.xrefModel != nil && n.xrefModel.CallGraph() != nil {
		for _, edge := range n.xrefModel.CallGraph().AllEdges() {
			if edge == nil || edge.CalleeID() != cleanID {
				continue
			}

			callerID := edge.CallerID()
			if callerMap[callerID] {
				continue
			}
			callerMap[callerID] = true

			name := callerID
			pkgPath := ""
			filePath := edge.FilePath()

			if n.symbolDB != nil {
				s := n.symbolDB.SymbolByID(callerID)
				if s != nil {
					name = s.Name()
					pkgPath = s.PackagePath()
					filePath = s.FilePath()
				}
			}

			node := NewCallHierarchyNode(
				callerID,
				name,
				pkgPath,
				filePath,
				nil,
				[]string{cleanID},
				1,
			)
			callers = append(callers, node)
		}
	}

	sort.Slice(callers, func(i, j int) bool {
		return callers[i].SymbolID() < callers[j].SymbolID()
	})

	return callers, nil
}

// GetOutgoingCalls identifies the functions or methods called by the specified function.
func (n *CallHierarchyNavigator) GetOutgoingCalls(symbolID string) ([]*CallHierarchyNode, error) {
	cleanID := strings.TrimSpace(symbolID)
	if cleanID == "" {
		return nil, ErrEmptyTarget
	}

	var callees []*CallHierarchyNode
	calleeMap := make(map[string]bool)

	if n.xrefModel != nil && n.xrefModel.CallGraph() != nil {
		for _, edge := range n.xrefModel.CallGraph().AllEdges() {
			if edge == nil || edge.CallerID() != cleanID {
				continue
			}

			calleeID := edge.CalleeID()
			if calleeMap[calleeID] {
				continue
			}
			calleeMap[calleeID] = true

			name := calleeID
			pkgPath := ""
			filePath := edge.FilePath()

			if n.symbolDB != nil {
				s := n.symbolDB.SymbolByID(calleeID)
				if s != nil {
					name = s.Name()
					pkgPath = s.PackagePath()
					filePath = s.FilePath()
				}
			}

			node := NewCallHierarchyNode(
				calleeID,
				name,
				pkgPath,
				filePath,
				[]string{cleanID},
				nil,
				1,
			)
			callees = append(callees, node)
		}
	}

	sort.Slice(callees, func(i, j int) bool {
		return callees[i].SymbolID() < callees[j].SymbolID()
	})

	return callees, nil
}

// GetRecursivePaths detects direct, mutual, and multi-hop recursive call cycles involving the target.
func (n *CallHierarchyNavigator) GetRecursivePaths(symbolID string) ([]*RecursivePath, error) {
	cleanID := strings.TrimSpace(symbolID)
	if cleanID == "" {
		return nil, ErrEmptyTarget
	}

	if n.xrefModel == nil || n.xrefModel.CallGraph() == nil {
		return nil, nil
	}

	// Build adjacency map
	adj := make(map[string][]string)
	for _, edge := range n.xrefModel.CallGraph().AllEdges() {
		if edge != nil {
			adj[edge.CallerID()] = append(adj[edge.CallerID()], edge.CalleeID())
		}
	}

	// Deterministically sort adjacency lists
	for k := range adj {
		sort.Strings(adj[k])
	}

	var paths []*RecursivePath
	visited := make(map[string]bool)
	recStack := make([]string, 0)
	cycleSet := make(map[string]bool)

	var dfs func(curr string, depth int)
	dfs = func(curr string, depth int) {
		if depth > 32 {
			return // Bound traversal to prevent overflow
		}

		recStack = append(recStack, curr)

		for _, neighbor := range adj[curr] {
			if neighbor == cleanID {
				// Cycle detected leading back to root target
				cycle := append([]string{}, recStack...)
				cycle = append(cycle, cleanID)
				cycleKey := strings.Join(cycle, "->")
				if !cycleSet[cycleKey] {
					cycleSet[cycleKey] = true
					isDirect := (len(cycle) == 2 && cycle[0] == cycle[1]) || (curr == cleanID && neighbor == cleanID)
					paths = append(paths, NewRecursivePath(cycle, isDirect))
				}
			} else {
				// Check if neighbor is in current recursion stack
				inStack := false
				for _, s := range recStack {
					if s == neighbor {
						inStack = true
						break
					}
				}
				if !inStack && !visited[neighbor] {
					dfs(neighbor, depth+1)
				}
			}
		}

		recStack = recStack[:len(recStack)-1]
		visited[curr] = true
	}

	dfs(cleanID, 1)

	sort.Slice(paths, func(i, j int) bool {
		return paths[i].ID() < paths[j].ID()
	})

	return paths, nil
}

// GetDependencyChains resolves deterministic sequences of call or dependency steps connecting source to target.
func (n *CallHierarchyNavigator) GetDependencyChains(sourceID, targetID string, maxDepth int) ([]*DependencyChain, error) {
	cleanSrc := strings.TrimSpace(sourceID)
	cleanTgt := strings.TrimSpace(targetID)

	if cleanSrc == "" || cleanTgt == "" {
		return nil, ErrEmptyTarget
	}

	if maxDepth <= 0 {
		maxDepth = 10
	}

	if n.xrefModel == nil || n.xrefModel.CallGraph() == nil {
		return nil, nil
	}

	adj := make(map[string][]string)
	for _, edge := range n.xrefModel.CallGraph().AllEdges() {
		if edge != nil {
			adj[edge.CallerID()] = append(adj[edge.CallerID()], edge.CalleeID())
		}
	}
	for k := range adj {
		sort.Strings(adj[k])
	}

	var chains []*DependencyChain
	var currentPath []string

	var search func(curr string, depth int)
	search = func(curr string, depth int) {
		if depth > maxDepth {
			return
		}

		currentPath = append(currentPath, curr)

		if curr == cleanTgt && len(currentPath) > 1 {
			steps := make([]string, len(currentPath))
			copy(steps, currentPath)
			chains = append(chains, NewDependencyChain(steps, false))
		} else {
			for _, nxt := range adj[curr] {
				// Avoid simple cycles in single path
				alreadyInPath := false
				for _, p := range currentPath {
					if p == nxt {
						alreadyInPath = true
						break
					}
				}
				if !alreadyInPath {
					search(nxt, depth+1)
				}
			}
		}

		currentPath = currentPath[:len(currentPath)-1]
	}

	search(cleanSrc, 1)

	sort.Slice(chains, func(i, j int) bool {
		return chains[i].ID() < chains[j].ID()
	})

	return chains, nil
}

// CalculateCallDepth calculates the maximum call depth reachable from a symbol without infinite recursion.
func (n *CallHierarchyNavigator) CalculateCallDepth(symbolID string, maxDepth int) (int, error) {
	cleanID := strings.TrimSpace(symbolID)
	if cleanID == "" {
		return 0, ErrEmptyTarget
	}

	if maxDepth <= 0 {
		maxDepth = 32
	}

	if n.xrefModel == nil || n.xrefModel.CallGraph() == nil {
		return 0, nil
	}

	adj := make(map[string][]string)
	for _, edge := range n.xrefModel.CallGraph().AllEdges() {
		if edge != nil {
			adj[edge.CallerID()] = append(adj[edge.CallerID()], edge.CalleeID())
		}
	}

	maxFoundDepth := 0
	visited := make(map[string]bool)

	var traverse func(curr string, currentDepth int)
	traverse = func(curr string, currentDepth int) {
		if currentDepth > maxFoundDepth {
			maxFoundDepth = currentDepth
		}
		if currentDepth >= maxDepth {
			return
		}

		visited[curr] = true
		for _, callee := range adj[curr] {
			if !visited[callee] {
				traverse(callee, currentDepth+1)
			}
		}
		visited[curr] = false
	}

	traverse(cleanID, 0)
	return maxFoundDepth, nil
}
