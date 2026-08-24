package language

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
)

// StructureModel represents a consolidated, immutable repository structural analysis model.
type StructureModel struct {
	root        string
	dirGraph    *DirectoryGraph
	packages    []*Package
	modGraph    *ModuleGraph
	workspace   *WorkspaceStructure
	vendors     []*VendorEntry
	buildGraph  *BuildGraph
	configs     []*ConfigAsset
	docs        []*DocAsset
	languages   []*discovery.LanguageDistribution
	diagnostics []*discovery.Diagnostic
}

// NewStructureModel constructs an immutable StructureModel record with defensively copied collections.
func NewStructureModel(
	root string,
	dirGraph *DirectoryGraph,
	packages []*Package,
	modGraph *ModuleGraph,
	workspace *WorkspaceStructure,
	vendors []*VendorEntry,
	buildGraph *BuildGraph,
	configs []*ConfigAsset,
	docs []*DocAsset,
	languages []*discovery.LanguageDistribution,
	diagnostics []*discovery.Diagnostic,
) *StructureModel {
	// Defensive copy & deterministic sort for packages: path asc, name asc
	pkgList := make([]*Package, len(packages))
	copy(pkgList, packages)
	sort.Slice(pkgList, func(i, j int) bool {
		if pkgList[i].path != pkgList[j].path {
			return pkgList[i].path < pkgList[j].path
		}
		return pkgList[i].name < pkgList[j].name
	})

	// Defensive copy & deterministic sort for vendor entries: path asc
	vendorList := make([]*VendorEntry, len(vendors))
	copy(vendorList, vendors)
	sort.Slice(vendorList, func(i, j int) bool {
		return vendorList[i].path < vendorList[j].path
	})

	// Defensive copy & deterministic sort for config assets: path asc
	cfgList := make([]*ConfigAsset, len(configs))
	copy(cfgList, configs)
	sort.Slice(cfgList, func(i, j int) bool {
		return cfgList[i].path < cfgList[j].path
	})

	// Defensive copy & deterministic sort for doc assets: path asc
	docList := make([]*DocAsset, len(docs))
	copy(docList, docs)
	sort.Slice(docList, func(i, j int) bool {
		return docList[i].path < docList[j].path
	})

	// Defensive copy for languages
	langList := make([]*discovery.LanguageDistribution, len(languages))
	copy(langList, languages)

	// Defensive copy for diagnostics
	diagList := make([]*discovery.Diagnostic, len(diagnostics))
	copy(diagList, diagnostics)

	return &StructureModel{
		root:        filepath.ToSlash(filepath.Clean(root)),
		dirGraph:    dirGraph,
		packages:    pkgList,
		modGraph:    modGraph,
		workspace:   workspace,
		vendors:     vendorList,
		buildGraph:  buildGraph,
		configs:     cfgList,
		docs:        docList,
		languages:   langList,
		diagnostics: diagList,
	}
}

// Root returns the cleaned absolute or relative root path.
func (sm *StructureModel) Root() string {
	if sm == nil {
		return ""
	}
	return sm.root
}

// DirectoryGraph returns the hierarchical directory graph.
func (sm *StructureModel) DirectoryGraph() *DirectoryGraph {
	if sm == nil {
		return nil
	}
	return sm.dirGraph
}

// Packages returns a defensive copy of discovered code packages in deterministic sorted order.
func (sm *StructureModel) Packages() []*Package {
	if sm == nil || len(sm.packages) == 0 {
		return nil
	}
	cloned := make([]*Package, len(sm.packages))
	copy(cloned, sm.packages)
	return cloned
}

// ModuleGraph returns the graph of detected modules.
func (sm *StructureModel) ModuleGraph() *ModuleGraph {
	if sm == nil {
		return nil
	}
	return sm.modGraph
}

// WorkspaceStructure returns multi-module workspace or monorepository structural information.
func (sm *StructureModel) WorkspaceStructure() *WorkspaceStructure {
	if sm == nil {
		return nil
	}
	return sm.workspace
}

// VendorEntries returns a defensive copy of detected vendor directories.
func (sm *StructureModel) VendorEntries() []*VendorEntry {
	if sm == nil || len(sm.vendors) == 0 {
		return nil
	}
	cloned := make([]*VendorEntry, len(sm.vendors))
	copy(cloned, sm.vendors)
	return cloned
}

// BuildGraph returns the build-system configuration graph.
func (sm *StructureModel) BuildGraph() *BuildGraph {
	if sm == nil {
		return nil
	}
	return sm.buildGraph
}

// ConfigAssets returns a defensive copy of configuration file assets in deterministic sorted order.
func (sm *StructureModel) ConfigAssets() []*ConfigAsset {
	if sm == nil || len(sm.configs) == 0 {
		return nil
	}
	cloned := make([]*ConfigAsset, len(sm.configs))
	copy(cloned, sm.configs)
	return cloned
}

// DocAssets returns a defensive copy of documentation assets in deterministic sorted order.
func (sm *StructureModel) DocAssets() []*DocAsset {
	if sm == nil || len(sm.docs) == 0 {
		return nil
	}
	cloned := make([]*DocAsset, len(sm.docs))
	copy(cloned, sm.docs)
	return cloned
}

// Languages returns a defensive copy of language distributions.
func (sm *StructureModel) Languages() []*discovery.LanguageDistribution {
	if sm == nil || len(sm.languages) == 0 {
		return nil
	}
	cloned := make([]*discovery.LanguageDistribution, len(sm.languages))
	copy(cloned, sm.languages)
	return cloned
}

// Diagnostics returns a defensive copy of diagnostics produced during analysis.
func (sm *StructureModel) Diagnostics() []*discovery.Diagnostic {
	if sm == nil || len(sm.diagnostics) == 0 {
		return nil
	}
	cloned := make([]*discovery.Diagnostic, len(sm.diagnostics))
	copy(cloned, sm.diagnostics)
	return cloned
}

// String returns a human-readable representation of the StructureModel.
func (sm *StructureModel) String() string {
	if sm == nil {
		return ""
	}
	modCount := 0
	if sm.modGraph != nil {
		modCount = sm.modGraph.Count()
	}
	buildCount := 0
	if sm.buildGraph != nil {
		buildCount = sm.buildGraph.Count()
	}
	return fmt.Sprintf("StructureModel<%s>[packages=%d, modules=%d, buildConfigs=%d, configs=%d, docs=%d]",
		sm.root, len(sm.packages), modCount, buildCount, len(sm.configs), len(sm.docs))
}
