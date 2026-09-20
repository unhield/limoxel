package dependency

import (
	"fmt"
	"sort"
	"strings"

	pkgerr "github.com/unhield/limoxel/internal/capabilities/plugin/errors"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// Resolvable represents a plugin candidate with identity, version, and dependencies.
type Resolvable interface {
	GetID() model.Identity
	GetVersion() version.SemVer
	GetDependencies() []model.Dependency
}

// ResolutionPlan represents the outcome of a successful dependency resolution.
type ResolutionPlan struct {
	ActivationOrder   []model.Identity
	DeactivationOrder []model.Identity
}

// Resolver resolves inter-plugin dependencies and detects cycles and version conflicts.
type Resolver struct{}

// NewResolver constructs an initialized Resolver.
func NewResolver() *Resolver {
	return &Resolver{}
}

// Resolve computes an activation order for a set of plugins, validating all dependencies.
func (r *Resolver) Resolve(plugins []Resolvable) (*ResolutionPlan, error) {
	if len(plugins) == 0 {
		return &ResolutionPlan{}, nil
	}

	lookup := make(map[model.Identity]Resolvable, len(plugins))
	adjList := make(map[model.Identity][]model.Identity, len(plugins))
	allIDs := make([]model.Identity, 0, len(plugins))

	for _, p := range plugins {
		id := p.GetID()
		if _, exists := lookup[id]; exists {
			return nil, pkgerr.New(pkgerr.CodeDuplicateIdentity, id, fmt.Sprintf("duplicate identity %s in resolution set", id))
		}
		lookup[id] = p
		allIDs = append(allIDs, id)
		adjList[id] = nil
	}

	sort.Slice(allIDs, func(i, j int) bool {
		return allIDs[i].Less(allIDs[j])
	})

	// 1. Verify dependency existence and version compatibility
	for _, p := range plugins {
		pID := p.GetID()
		for _, dep := range p.GetDependencies() {
			target, exists := lookup[dep.ID]
			if !exists {
				if dep.Optional {
					continue // Optional dependencies may be absent
				}
				err := pkgerr.New(pkgerr.CodeDependencyMissing, pID,
					fmt.Sprintf("mandatory dependency %s required by %s is not installed", dep.ID, pID))
				err.WithDetail("missing_dependency", string(dep.ID))
				return nil, err
			}

			if !dep.MatchesVersion(target.GetVersion()) {
				err := pkgerr.New(pkgerr.CodeIncompatibleVersion, pID,
					fmt.Sprintf("dependency %s version %s does not satisfy constraint %q required by %s",
						dep.ID, target.GetVersion(), dep.Constraint, pID))
				err.WithDetail("dependency", string(dep.ID))
				err.WithDetail("installed_version", target.GetVersion().String())
				err.WithDetail("required_constraint", dep.Constraint)
				return nil, err
			}

			adjList[pID] = append(adjList[pID], dep.ID)
		}
		// Sort adjacency list for deterministic traversal
		sort.Slice(adjList[pID], func(i, j int) bool {
			return adjList[pID][i].Less(adjList[pID][j])
		})
	}

	// 2. Detect cycles using 3-color DFS
	const (
		white = 0 // unvisited
		gray  = 1 // currently visiting (in recursion stack)
		black = 2 // completely visited
	)
	colors := make(map[model.Identity]int, len(allIDs))
	parent := make(map[model.Identity]model.Identity)

	for _, id := range allIDs {
		if colors[id] == white {
			if cycleErr := r.detectCycleDFS(id, adjList, colors, parent); cycleErr != nil {
				return nil, cycleErr
			}
		}
	}

	// 3. Compute topological order (dependencies first)
	visited := make(map[model.Identity]bool, len(allIDs))
	var order []model.Identity

	var topoVisit func(id model.Identity)
	topoVisit = func(id model.Identity) {
		if visited[id] {
			return
		}
		visited[id] = true
		for _, depID := range adjList[id] {
			topoVisit(depID)
		}
		order = append(order, id)
	}

	for _, id := range allIDs {
		if !visited[id] {
			topoVisit(id)
		}
	}

	// Build deactivation order (reverse of activation order)
	deact := make([]model.Identity, len(order))
	for i, id := range order {
		deact[len(order)-1-i] = id
	}

	return &ResolutionPlan{
		ActivationOrder:   order,
		DeactivationOrder: deact,
	}, nil
}

func (r *Resolver) detectCycleDFS(node model.Identity, adj map[model.Identity][]model.Identity, colors map[model.Identity]int, parent map[model.Identity]model.Identity) error {
	colors[node] = 1 // gray
	for _, neighbor := range adj[node] {
		if colors[neighbor] == 1 {
			// Found cycle! Reconstruct path
			cyclePath := []string{string(neighbor), string(node)}
			curr := node
			for curr != neighbor && parent[curr] != "" {
				curr = parent[curr]
				cyclePath = append(cyclePath, string(curr))
			}
			// Reverse for readability
			for i, j := 0, len(cyclePath)-1; i < j; i, j = i+1, j-1 {
				cyclePath[i], cyclePath[j] = cyclePath[j], cyclePath[i]
			}
			return pkgerr.New(pkgerr.CodeDependencyCycle, node,
				fmt.Sprintf("dependency cycle detected: %s", strings.Join(cyclePath, " -> ")))
		}
		if colors[neighbor] == 0 {
			parent[neighbor] = node
			if err := r.detectCycleDFS(neighbor, adj, colors, parent); err != nil {
				return err
			}
		}
	}
	colors[node] = 2 // black
	return nil
}
