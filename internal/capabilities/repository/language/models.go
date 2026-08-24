package language

import (
	"path/filepath"
	"sort"
	"strings"
)

// DirectoryNode represents a single directory node in the repository hierarchy.
type DirectoryNode struct {
	path             string
	parentPath       string
	childDirectories []string
	files            []string
	isPackage        bool
	isModule         bool
	isVendor         bool
}

// NewDirectoryNode creates an immutable DirectoryNode record.
func NewDirectoryNode(
	path string,
	parentPath string,
	children []string,
	files []string,
	isPackage bool,
	isModule bool,
	isVendor bool,
) *DirectoryNode {
	cleanPath := filepath.ToSlash(filepath.Clean(path))
	cleanParent := filepath.ToSlash(filepath.Clean(parentPath))

	childList := make([]string, len(children))
	for i, c := range children {
		childList[i] = filepath.ToSlash(filepath.Clean(c))
	}
	sort.Strings(childList)

	fileList := make([]string, len(files))
	for i, f := range files {
		fileList[i] = filepath.ToSlash(filepath.Clean(f))
	}
	sort.Strings(fileList)

	return &DirectoryNode{
		path:             cleanPath,
		parentPath:       cleanParent,
		childDirectories: childList,
		files:            fileList,
		isPackage:        isPackage,
		isModule:         isModule,
		isVendor:         isVendor,
	}
}

// Path returns the repository-relative path of the directory.
func (dn *DirectoryNode) Path() string {
	if dn == nil {
		return ""
	}
	return dn.path
}

// ParentPath returns the repository-relative path of the parent directory.
func (dn *DirectoryNode) ParentPath() string {
	if dn == nil {
		return ""
	}
	return dn.parentPath
}

// ChildDirectories returns a defensive copy of child directory paths in deterministic sorted order.
func (dn *DirectoryNode) ChildDirectories() []string {
	if dn == nil || len(dn.childDirectories) == 0 {
		return nil
	}
	cloned := make([]string, len(dn.childDirectories))
	copy(cloned, dn.childDirectories)
	return cloned
}

// Files returns a defensive copy of files located directly within this directory.
func (dn *DirectoryNode) Files() []string {
	if dn == nil || len(dn.files) == 0 {
		return nil
	}
	cloned := make([]string, len(dn.files))
	copy(cloned, dn.files)
	return cloned
}

// IsPackage reports whether this directory represents a detectable code package.
func (dn *DirectoryNode) IsPackage() bool {
	if dn == nil {
		return false
	}
	return dn.isPackage
}

// IsModule reports whether this directory contains a module descriptor.
func (dn *DirectoryNode) IsModule() bool {
	if dn == nil {
		return false
	}
	return dn.isModule
}

// IsVendor reports whether this directory is classified as vendor or third-party storage.
func (dn *DirectoryNode) IsVendor() bool {
	if dn == nil {
		return false
	}
	return dn.isVendor
}

// DirectoryGraph represents the structural directory hierarchy of the repository.
type DirectoryGraph struct {
	rootPath string
	nodes    map[string]*DirectoryNode
	nodeList []*DirectoryNode
}

// NewDirectoryGraph constructs an immutable DirectoryGraph from directory nodes.
func NewDirectoryGraph(rootPath string, nodes []*DirectoryNode) *DirectoryGraph {
	nodeMap := make(map[string]*DirectoryNode, len(nodes))
	sortedNodes := make([]*DirectoryNode, len(nodes))
	copy(sortedNodes, nodes)

	sort.Slice(sortedNodes, func(i, j int) bool {
		return sortedNodes[i].path < sortedNodes[j].path
	})

	for _, n := range sortedNodes {
		nodeMap[n.path] = n
	}

	return &DirectoryGraph{
		rootPath: filepath.ToSlash(filepath.Clean(rootPath)),
		nodes:    nodeMap,
		nodeList: sortedNodes,
	}
}

// RootPath returns the repository root path.
func (dg *DirectoryGraph) RootPath() string {
	if dg == nil {
		return ""
	}
	return dg.rootPath
}

// Node returns the DirectoryNode for the specified repository-relative directory path.
func (dg *DirectoryGraph) Node(relPath string) (*DirectoryNode, bool) {
	if dg == nil || dg.nodes == nil {
		return nil, false
	}
	clean := filepath.ToSlash(filepath.Clean(relPath))
	n, exists := dg.nodes[clean]
	return n, exists
}

// AllNodes returns a defensive copy of all directory nodes in deterministic sorted order.
func (dg *DirectoryGraph) AllNodes() []*DirectoryNode {
	if dg == nil || len(dg.nodeList) == 0 {
		return nil
	}
	cloned := make([]*DirectoryNode, len(dg.nodeList))
	copy(cloned, dg.nodeList)
	return cloned
}

// NodeCount returns the total number of directories in the graph.
func (dg *DirectoryGraph) NodeCount() int {
	if dg == nil {
		return 0
	}
	return len(dg.nodeList)
}

// Package represents a discovered structural code package.
type Package struct {
	name       string
	path       string
	languageID string
	files      []string
}

// NewPackage creates a new immutable Package record.
func NewPackage(name, path, languageID string, files []string) *Package {
	fileList := make([]string, len(files))
	for i, f := range files {
		fileList[i] = filepath.ToSlash(filepath.Clean(f))
	}
	sort.Strings(fileList)

	return &Package{
		name:       strings.TrimSpace(name),
		path:       filepath.ToSlash(filepath.Clean(path)),
		languageID: strings.ToLower(strings.TrimSpace(languageID)),
		files:      fileList,
	}
}

// Name returns the package name.
func (p *Package) Name() string {
	if p == nil {
		return ""
	}
	return p.name
}

// Path returns the repository-relative directory path of the package.
func (p *Package) Path() string {
	if p == nil {
		return ""
	}
	return p.path
}

// LanguageID returns the programming language associated with the package.
func (p *Package) LanguageID() string {
	if p == nil {
		return ""
	}
	return p.languageID
}

// Files returns a defensive copy of source files contained in the package.
func (p *Package) Files() []string {
	if p == nil || len(p.files) == 0 {
		return nil
	}
	cloned := make([]string, len(p.files))
	copy(cloned, p.files)
	return cloned
}

// Module represents a detected project module or submodule descriptor.
type Module struct {
	moduleType     ModuleType
	name           string
	path           string
	descriptorFile string
	languageID     string
	buildSystem    BuildSystemType
}

// NewModule creates a new immutable Module record.
func NewModule(
	mType ModuleType,
	name string,
	path string,
	descriptorFile string,
	langID string,
	buildSystem BuildSystemType,
) *Module {
	return &Module{
		moduleType:     mType,
		name:           strings.TrimSpace(name),
		path:           filepath.ToSlash(filepath.Clean(path)),
		descriptorFile: filepath.ToSlash(filepath.Clean(descriptorFile)),
		languageID:     strings.ToLower(strings.TrimSpace(langID)),
		buildSystem:    buildSystem,
	}
}

// Type returns the module category type.
func (m *Module) Type() ModuleType {
	if m == nil {
		return ModuleUnknown
	}
	return m.moduleType
}

// Name returns the module identifier or name.
func (m *Module) Name() string {
	if m == nil {
		return ""
	}
	return m.name
}

// Path returns the repository-relative directory path of the module.
func (m *Module) Path() string {
	if m == nil {
		return ""
	}
	return m.path
}

// DescriptorFile returns the repository-relative path to the module descriptor file.
func (m *Module) DescriptorFile() string {
	if m == nil {
		return ""
	}
	return m.descriptorFile
}

// LanguageID returns the primary language associated with the module.
func (m *Module) LanguageID() string {
	if m == nil {
		return ""
	}
	return m.languageID
}

// BuildSystem returns the build system associated with the module descriptor.
func (m *Module) BuildSystem() BuildSystemType {
	if m == nil {
		return BuildUnknown
	}
	return m.buildSystem
}

// ModuleGraph represents structural relationships between discovered modules.
type ModuleGraph struct {
	modules []*Module
	modMap  map[string]*Module
}

// NewModuleGraph constructs an immutable ModuleGraph from modules.
func NewModuleGraph(modules []*Module) *ModuleGraph {
	modList := make([]*Module, len(modules))
	copy(modList, modules)

	sort.Slice(modList, func(i, j int) bool {
		if modList[i].path != modList[j].path {
			return modList[i].path < modList[j].path
		}
		return modList[i].name < modList[j].name
	})

	modMap := make(map[string]*Module, len(modList))
	for _, m := range modList {
		modMap[m.path] = m
	}

	return &ModuleGraph{
		modules: modList,
		modMap:  modMap,
	}
}

// Modules returns a defensive copy of all discovered modules in deterministic sorted order.
func (mg *ModuleGraph) Modules() []*Module {
	if mg == nil || len(mg.modules) == 0 {
		return nil
	}
	cloned := make([]*Module, len(mg.modules))
	copy(cloned, mg.modules)
	return cloned
}

// ModuleByPath returns the module located at the specified repository-relative directory path.
func (mg *ModuleGraph) ModuleByPath(relPath string) (*Module, bool) {
	if mg == nil || mg.modMap == nil {
		return nil, false
	}
	clean := filepath.ToSlash(filepath.Clean(relPath))
	m, exists := mg.modMap[clean]
	return m, exists
}

// Count returns the total number of discovered modules.
func (mg *ModuleGraph) Count() int {
	if mg == nil {
		return 0
	}
	return len(mg.modules)
}

// WorkspaceStructure represents multi-module workspace or monorepository structural layout.
type WorkspaceStructure struct {
	isMonorepo    bool
	rootModules   []*Module
	nestedModules []*Module
}

// NewWorkspaceStructure creates an immutable WorkspaceStructure descriptor.
func NewWorkspaceStructure(rootModules, nestedModules []*Module) *WorkspaceStructure {
	rMods := make([]*Module, len(rootModules))
	copy(rMods, rootModules)
	sort.Slice(rMods, func(i, j int) bool { return rMods[i].path < rMods[j].path })

	nMods := make([]*Module, len(nestedModules))
	copy(nMods, nestedModules)
	sort.Slice(nMods, func(i, j int) bool { return nMods[i].path < nMods[j].path })

	isMono := len(nMods) > 0 || len(rMods) > 1

	return &WorkspaceStructure{
		isMonorepo:    isMono,
		rootModules:   rMods,
		nestedModules: nMods,
	}
}

// IsMonorepo reports whether the repository contains multiple modules or a monorepository layout.
func (ws *WorkspaceStructure) IsMonorepo() bool {
	if ws == nil {
		return false
	}
	return ws.isMonorepo
}

// RootModules returns a defensive copy of top-level modules.
func (ws *WorkspaceStructure) RootModules() []*Module {
	if ws == nil || len(ws.rootModules) == 0 {
		return nil
	}
	cloned := make([]*Module, len(ws.rootModules))
	copy(cloned, ws.rootModules)
	return cloned
}

// NestedModules returns a defensive copy of nested/sub-directory modules.
func (ws *WorkspaceStructure) NestedModules() []*Module {
	if ws == nil || len(ws.nestedModules) == 0 {
		return nil
	}
	cloned := make([]*Module, len(ws.nestedModules))
	copy(cloned, ws.nestedModules)
	return cloned
}

// VendorEntry represents a discovered vendor or third-party dependency storage directory.
type VendorEntry struct {
	path      string
	ecosystem string
}

// NewVendorEntry creates an immutable VendorEntry record.
func NewVendorEntry(path, ecosystem string) *VendorEntry {
	return &VendorEntry{
		path:      filepath.ToSlash(filepath.Clean(path)),
		ecosystem: strings.TrimSpace(ecosystem),
	}
}

// Path returns the repository-relative path to the vendor directory.
func (v *VendorEntry) Path() string {
	if v == nil {
		return ""
	}
	return v.path
}

// Ecosystem returns the package ecosystem associated with the vendor directory (e.g. "go", "npm").
func (v *VendorEntry) Ecosystem() string {
	if v == nil {
		return ""
	}
	return v.ecosystem
}

// BuildConfig represents a discovered build system configuration asset.
type BuildConfig struct {
	buildType  BuildSystemType
	path       string
	configFile string
	modulePath string
}

// NewBuildConfig creates an immutable BuildConfig record.
func NewBuildConfig(bType BuildSystemType, path, configFile, modulePath string) *BuildConfig {
	return &BuildConfig{
		buildType:  bType,
		path:       filepath.ToSlash(filepath.Clean(path)),
		configFile: filepath.ToSlash(filepath.Clean(configFile)),
		modulePath: filepath.ToSlash(filepath.Clean(modulePath)),
	}
}

// Type returns the build system type.
func (bc *BuildConfig) Type() BuildSystemType {
	if bc == nil {
		return BuildUnknown
	}
	return bc.buildType
}

// Path returns the repository-relative directory path containing the build configuration.
func (bc *BuildConfig) Path() string {
	if bc == nil {
		return ""
	}
	return bc.path
}

// ConfigFile returns the repository-relative path to the configuration file (e.g. "Makefile").
func (bc *BuildConfig) ConfigFile() string {
	if bc == nil {
		return ""
	}
	return bc.configFile
}

// ModulePath returns the associated module directory path, if applicable.
func (bc *BuildConfig) ModulePath() string {
	if bc == nil {
		return ""
	}
	return bc.modulePath
}

// BuildGraph represents all detected build configurations across the repository.
type BuildGraph struct {
	configs []*BuildConfig
}

// NewBuildGraph constructs an immutable BuildGraph.
func NewBuildGraph(configs []*BuildConfig) *BuildGraph {
	cfgList := make([]*BuildConfig, len(configs))
	copy(cfgList, configs)

	sort.Slice(cfgList, func(i, j int) bool {
		if cfgList[i].path != cfgList[j].path {
			return cfgList[i].path < cfgList[j].path
		}
		return cfgList[i].configFile < cfgList[j].configFile
	})

	return &BuildGraph{configs: cfgList}
}

// Configs returns a defensive copy of all build configurations in deterministic sorted order.
func (bg *BuildGraph) Configs() []*BuildConfig {
	if bg == nil || len(bg.configs) == 0 {
		return nil
	}
	cloned := make([]*BuildConfig, len(bg.configs))
	copy(cloned, bg.configs)
	return cloned
}

// Count returns the count of discovered build configurations.
func (bg *BuildGraph) Count() int {
	if bg == nil {
		return 0
	}
	return len(bg.configs)
}

// ConfigAsset represents a discovered configuration file asset.
type ConfigAsset struct {
	configType ConfigType
	path       string
	filename   string
	isHidden   bool
}

// NewConfigAsset creates an immutable ConfigAsset record.
func NewConfigAsset(cType ConfigType, path string, isHidden bool) *ConfigAsset {
	cleanPath := filepath.ToSlash(filepath.Clean(path))
	baseName := filepath.Base(cleanPath)

	return &ConfigAsset{
		configType: cType,
		path:       cleanPath,
		filename:   baseName,
		isHidden:   isHidden,
	}
}

// Type returns the configuration category type.
func (ca *ConfigAsset) Type() ConfigType {
	if ca == nil {
		return ConfigUnknown
	}
	return ca.configType
}

// Path returns the repository-relative path of the configuration asset.
func (ca *ConfigAsset) Path() string {
	if ca == nil {
		return ""
	}
	return ca.path
}

// Filename returns the base filename of the configuration asset.
func (ca *ConfigAsset) Filename() string {
	if ca == nil {
		return ""
	}
	return ca.filename
}

// IsHidden reports whether the configuration asset is a hidden dotfile (e.g. ".env").
func (ca *ConfigAsset) IsHidden() bool {
	if ca == nil {
		return false
	}
	return ca.isHidden
}

// DocAsset represents an engineering documentation file asset.
type DocAsset struct {
	docType  DocType
	path     string
	filename string
	category string
}

// NewDocAsset creates an immutable DocAsset record.
func NewDocAsset(dType DocType, path string, category string) *DocAsset {
	cleanPath := filepath.ToSlash(filepath.Clean(path))
	baseName := filepath.Base(cleanPath)

	return &DocAsset{
		docType:  dType,
		path:     cleanPath,
		filename: baseName,
		category: strings.TrimSpace(category),
	}
}

// Type returns the documentation asset category type.
func (da *DocAsset) Type() DocType {
	if da == nil {
		return DocUnknown
	}
	return da.docType
}

// Path returns the repository-relative path of the documentation asset.
func (da *DocAsset) Path() string {
	if da == nil {
		return ""
	}
	return da.path
}

// Filename returns the base filename of the documentation asset.
func (da *DocAsset) Filename() string {
	if da == nil {
		return ""
	}
	return da.filename
}

// Category returns the documentation category classification.
func (da *DocAsset) Category() string {
	if da == nil {
		return ""
	}
	return da.category
}
