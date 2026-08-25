package crossrepo

import (
	"path/filepath"
	"sort"
)

// RepositoryInput provides the data model for an individual repository in a workspace.
type RepositoryInput struct {
	Root         string
	Name         string
	Modules      []string
	Packages     []string
	Dependencies []string
}

// WorkspaceAnalyzer analyzes multi-repository relationships, shared dependencies, and shared architectures.
type WorkspaceAnalyzer struct{}

// NewWorkspaceAnalyzer creates a new WorkspaceAnalyzer.
func NewWorkspaceAnalyzer() *WorkspaceAnalyzer {
	return &WorkspaceAnalyzer{}
}

// Analyze performs multi-repository workspace analysis.
func (a *WorkspaceAnalyzer) Analyze(
	workspaceRoot string,
	repos []RepositoryInput,
	sharedConfigs []*SharedConfig,
) *WorkspaceModel {
	cleanWsRoot := filepath.ToSlash(filepath.Clean(workspaceRoot))

	var wsRepos []*WorkspaceRepository
	var wsRels []*WorkspaceRelationship
	var sharedDeps []*SharedDependency
	var sharedArch []*SharedArchitecture

	depConsumers := make(map[string][]string)
	pkgToRepo := make(map[string]string)
	modToRepo := make(map[string]string)

	// 1. Build Workspace Repositories
	for _, r := range repos {
		cleanRoot := filepath.ToSlash(filepath.Clean(r.Root))
		repoObj := NewWorkspaceRepository(
			cleanRoot,
			r.Name,
			r.Modules,
			r.Packages,
			r.Dependencies,
		)
		wsRepos = append(wsRepos, repoObj)

		for _, pkg := range r.Packages {
			pkgToRepo[pkg] = repoObj.ID()
		}
		for _, mod := range r.Modules {
			modToRepo[mod] = repoObj.ID()
		}

		for _, dep := range r.Dependencies {
			depConsumers[dep] = append(depConsumers[dep], repoObj.ID())
		}
	}

	// 2. Identify Cross-Repository Dependencies & Relationships
	relMap := make(map[string]*WorkspaceRelationship)

	for _, r := range wsRepos {
		for _, dep := range r.Dependencies() {
			// Check if dependency is another repository or package in another repository
			if targetRepoID, exists := pkgToRepo[dep]; exists && targetRepoID != r.ID() {
				relID := "wsrel:" + r.ID() + "->" + targetRepoID + ":package_dependency"
				relMap[relID] = NewWorkspaceRelationship(
					r.ID(),
					targetRepoID,
					WorkspaceRelSharedPackage,
					"dependency on package "+dep,
				)
			}
			if targetRepoID, exists := modToRepo[dep]; exists && targetRepoID != r.ID() {
				relID := "wsrel:" + r.ID() + "->" + targetRepoID + ":module_dependency"
				relMap[relID] = NewWorkspaceRelationship(
					r.ID(),
					targetRepoID,
					WorkspaceRelSharedModule,
					"dependency on module "+dep,
				)
			}
		}
	}

	for _, rel := range relMap {
		wsRels = append(wsRels, rel)
	}

	// 3. Shared Dependencies Across Multiple Repositories
	for depName, consumers := range depConsumers {
		if len(consumers) > 1 {
			// Deduplicate consumers
			cMap := make(map[string]bool)
			var uniqueConsumers []string
			for _, c := range consumers {
				if !cMap[c] {
					cMap[c] = true
					uniqueConsumers = append(uniqueConsumers, c)
				}
			}
			sort.Strings(uniqueConsumers)

			sharedDeps = append(sharedDeps, NewSharedDependency(
				depName,
				"workspace_shared",
				uniqueConsumers,
				nil,
			))
		}
	}

	// 4. Shared Architecture Discovery
	if len(wsRepos) > 1 {
		var participatingRepoIDs []string
		for _, r := range wsRepos {
			participatingRepoIDs = append(participatingRepoIDs, r.ID())
		}
		sort.Strings(participatingRepoIDs)

		sharedArch = append(sharedArch, NewSharedArchitecture(
			"multi_service_workspace",
			"Cooperating repository workspace architecture",
			participatingRepoIDs,
			nil,
			[]string{"workspace_boundary", "service_boundary"},
		))
	}

	return NewWorkspaceModel(
		cleanWsRoot,
		wsRepos,
		wsRels,
		sharedDeps,
		sharedConfigs,
		sharedArch,
	)
}
