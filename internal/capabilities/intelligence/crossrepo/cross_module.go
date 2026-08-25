package crossrepo

import (
	"path/filepath"
	"sort"
	"strings"
)

// ModuleInfo carries structural information about a discovered module.
type ModuleInfo struct {
	Path         string
	Version      string
	ParentModule string
	Packages     []string
	Dependencies map[string]string
}

// CrossModuleAnalyzer evaluates module relationships, hierarchies, shared modules, and version compatibility.
type CrossModuleAnalyzer struct{}

// NewCrossModuleAnalyzer creates a new CrossModuleAnalyzer.
func NewCrossModuleAnalyzer() *CrossModuleAnalyzer {
	return &CrossModuleAnalyzer{}
}

// Analyze performs cross-module analysis.
func (a *CrossModuleAnalyzer) Analyze(
	modules []ModuleInfo,
) (
	[]*ModuleRelationship,
	[]*SharedModule,
	[]*ModuleHierarchyNode,
	[]*VersionCompatibility,
) {
	var modRels []*ModuleRelationship
	var sharedMods []*SharedModule
	var modHierarchy []*ModuleHierarchyNode
	var versionCompats []*VersionCompatibility

	relMap := make(map[string]*ModuleRelationship)
	sharedMap := make(map[string]*SharedModule)
	hierarchyMap := make(map[string]*ModuleHierarchyNode)
	compatMap := make(map[string]*VersionCompatibility)

	// Map of module children
	childModulesMap := make(map[string][]string)

	for _, mod := range modules {
		cleanMod := filepath.ToSlash(filepath.Clean(mod.Path))
		if mod.ParentModule != "" {
			cleanParent := filepath.ToSlash(filepath.Clean(mod.ParentModule))
			childModulesMap[cleanParent] = append(childModulesMap[cleanParent], cleanMod)

			// Add hierarchy relationship
			hRelID := "modrel:" + cleanParent + "->" + cleanMod + ":hierarchy"
			relMap[hRelID] = NewModuleRelationship(
				cleanParent,
				cleanMod,
				ModuleRelHierarchy,
				mod.Version,
				"internal_hierarchy",
			)
		}

		// Dependencies
		for depPath, depVer := range mod.Dependencies {
			cleanDep := filepath.ToSlash(filepath.Clean(depPath))
			dRelID := "modrel:" + cleanMod + "->" + cleanDep + ":dependency"
			relMap[dRelID] = NewModuleRelationship(
				cleanMod,
				cleanDep,
				ModuleRelDependency,
				depVer,
				"module_boundary",
			)

			// Shared module tracking
			if shared, exists := sharedMap[cleanDep]; exists {
				consumers := append(shared.ConsumingContexts(), cleanMod)
				sharedMap[cleanDep] = NewSharedModule(
					cleanDep,
					shared.OwningContext(),
					consumers,
					shared.SharedSymbols(),
					shared.SharedDependencies(),
				)
			} else {
				sharedMap[cleanDep] = NewSharedModule(
					cleanDep,
					cleanMod,
					[]string{cleanMod},
					nil,
					nil,
				)
			}

			// Version compatibility check
			cID := "compat:" + cleanDep + ":" + depVer
			state := CompatCompatible
			details := "version constraint satisfied"
			if depVer == "" {
				state = CompatUnavailable
				details = "no version constraint specified"
			} else if strings.HasPrefix(depVer, "incompatible") {
				state = CompatIncompatible
				details = "incompatible version constraint"
			}

			compatMap[cID] = NewVersionCompatibility(
				cleanDep,
				depVer,
				depVer,
				state,
				details,
			)
		}
	}

	// Build Hierarchy Nodes
	for _, mod := range modules {
		cleanMod := filepath.ToSlash(filepath.Clean(mod.Path))
		children := childModulesMap[cleanMod]
		cleanParent := ""
		if mod.ParentModule != "" {
			cleanParent = filepath.ToSlash(filepath.Clean(mod.ParentModule))
		}

		hierarchyMap[cleanMod] = NewModuleHierarchyNode(
			cleanMod,
			cleanParent,
			children,
			mod.Packages,
		)
	}

	// Deterministic sorting into result slices
	for _, r := range relMap {
		modRels = append(modRels, r)
	}
	sort.Slice(modRels, func(i, j int) bool { return modRels[i].ID() < modRels[j].ID() })

	for _, s := range sharedMap {
		sharedMods = append(sharedMods, s)
	}
	sort.Slice(sharedMods, func(i, j int) bool { return sharedMods[i].ID() < sharedMods[j].ID() })

	for _, h := range hierarchyMap {
		modHierarchy = append(modHierarchy, h)
	}
	sort.Slice(modHierarchy, func(i, j int) bool { return modHierarchy[i].ID() < modHierarchy[j].ID() })

	for _, v := range compatMap {
		versionCompats = append(versionCompats, v)
	}
	sort.Slice(versionCompats, func(i, j int) bool { return versionCompats[i].ID() < versionCompats[j].ID() })

	return modRels, sharedMods, modHierarchy, versionCompats
}
