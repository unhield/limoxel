package crossrepo

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileRelationship represents a directed relationship between two repository files.
type FileRelationship struct {
	id         string
	kind       FileRelationKind
	sourceFile string
	targetFile string
	evidence   string
	provenance string
}

// NewFileRelationship creates an immutable FileRelationship.
func NewFileRelationship(id string, kind FileRelationKind, sourceFile, targetFile, evidence, provenance string) *FileRelationship {
	cleanSrc := filepath.ToSlash(filepath.Clean(sourceFile))
	cleanTgt := filepath.ToSlash(filepath.Clean(targetFile))
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		cleanID = "filerel:" + cleanSrc + "->" + cleanTgt + ":" + string(kind)
	}

	return &FileRelationship{
		id:         cleanID,
		kind:       kind,
		sourceFile: cleanSrc,
		targetFile: cleanTgt,
		evidence:   strings.TrimSpace(evidence),
		provenance: strings.TrimSpace(provenance),
	}
}

func (r *FileRelationship) ID() string             { return r.id }
func (r *FileRelationship) Kind() FileRelationKind { return r.kind }
func (r *FileRelationship) SourceFile() string     { return r.sourceFile }
func (r *FileRelationship) TargetFile() string     { return r.targetFile }
func (r *FileRelationship) Evidence() string       { return r.evidence }
func (r *FileRelationship) Provenance() string     { return r.provenance }

// SymbolPropagation tracks how a symbol travels across file and package boundaries.
type SymbolPropagation struct {
	id                string
	symbolID          string
	symbolName        string
	declaringFile     string
	definingFile      string
	referencingFiles  []string
	exportingPackage  string
	consumingPackages []string
	propagationPath   []string
}

// NewSymbolPropagation creates an immutable SymbolPropagation record.
func NewSymbolPropagation(
	symbolID, symbolName, declaringFile, definingFile, exportingPkg string,
	referencingFiles, consumingPkgs, propPath []string,
) *SymbolPropagation {
	refFiles := make([]string, len(referencingFiles))
	for i, f := range referencingFiles {
		refFiles[i] = filepath.ToSlash(filepath.Clean(f))
	}
	sort.Strings(refFiles)

	conPkgs := make([]string, len(consumingPkgs))
	for i, p := range consumingPkgs {
		conPkgs[i] = filepath.ToSlash(filepath.Clean(p))
	}
	sort.Strings(conPkgs)

	pPath := make([]string, len(propPath))
	copy(pPath, propPath)

	cleanSymID := strings.TrimSpace(symbolID)

	return &SymbolPropagation{
		id:                "prop:" + cleanSymID,
		symbolID:          cleanSymID,
		symbolName:        strings.TrimSpace(symbolName),
		declaringFile:     filepath.ToSlash(filepath.Clean(declaringFile)),
		definingFile:      filepath.ToSlash(filepath.Clean(definingFile)),
		referencingFiles:  refFiles,
		exportingPackage:  filepath.ToSlash(filepath.Clean(exportingPkg)),
		consumingPackages: conPkgs,
		propagationPath:   pPath,
	}
}

func (p *SymbolPropagation) ID() string               { return p.id }
func (p *SymbolPropagation) SymbolID() string         { return p.symbolID }
func (p *SymbolPropagation) SymbolName() string       { return p.symbolName }
func (p *SymbolPropagation) DeclaringFile() string    { return p.declaringFile }
func (p *SymbolPropagation) DefiningFile() string     { return p.definingFile }
func (p *SymbolPropagation) ExportingPackage() string { return p.exportingPackage }

func (p *SymbolPropagation) ReferencingFiles() []string {
	if p == nil || p.referencingFiles == nil {
		return nil
	}
	res := make([]string, len(p.referencingFiles))
	copy(res, p.referencingFiles)
	return res
}

func (p *SymbolPropagation) ConsumingPackages() []string {
	if p == nil || p.consumingPackages == nil {
		return nil
	}
	res := make([]string, len(p.consumingPackages))
	copy(res, p.consumingPackages)
	return res
}

func (p *SymbolPropagation) PropagationPath() []string {
	if p == nil || p.propagationPath == nil {
		return nil
	}
	res := make([]string, len(p.propagationPath))
	copy(res, p.propagationPath)
	return res
}

// CrossFileDependency represents an inter-file dependency path.
type CrossFileDependency struct {
	id              string
	sourceFile      string
	targetFile      string
	symbolMediated  []string
	packageMediated string
	isDirect        bool
}

// NewCrossFileDependency creates an immutable CrossFileDependency.
func NewCrossFileDependency(sourceFile, targetFile, pkgMediated string, symbols []string, isDirect bool) *CrossFileDependency {
	cleanSrc := filepath.ToSlash(filepath.Clean(sourceFile))
	cleanTgt := filepath.ToSlash(filepath.Clean(targetFile))

	symMap := make(map[string]bool)
	var syms []string
	for _, s := range symbols {
		cleanS := strings.TrimSpace(s)
		if cleanS != "" && !symMap[cleanS] {
			symMap[cleanS] = true
			syms = append(syms, cleanS)
		}
	}
	sort.Strings(syms)

	return &CrossFileDependency{
		id:              "filedep:" + cleanSrc + "->" + cleanTgt,
		sourceFile:      cleanSrc,
		targetFile:      cleanTgt,
		symbolMediated:  syms,
		packageMediated: filepath.ToSlash(filepath.Clean(pkgMediated)),
		isDirect:        isDirect,
	}
}

func (d *CrossFileDependency) ID() string              { return d.id }
func (d *CrossFileDependency) SourceFile() string      { return d.sourceFile }
func (d *CrossFileDependency) TargetFile() string      { return d.targetFile }
func (d *CrossFileDependency) PackageMediated() string { return d.packageMediated }
func (d *CrossFileDependency) IsDirect() bool          { return d.isDirect }

func (d *CrossFileDependency) SymbolMediated() []string {
	if d == nil || d.symbolMediated == nil {
		return nil
	}
	res := make([]string, len(d.symbolMediated))
	copy(res, d.symbolMediated)
	return res
}

// SharedConfig represents configuration affecting multiple files, packages, or repositories.
type SharedConfig struct {
	id                   string
	configPath           string
	configFormat         string
	affectedFiles        []string
	affectedPackages     []string
	affectedModules      []string
	affectedRepositories []string
}

// NewSharedConfig creates an immutable SharedConfig record.
func NewSharedConfig(
	configPath, format string,
	files, pkgs, modules, repos []string,
) *SharedConfig {
	cleanPath := filepath.ToSlash(filepath.Clean(configPath))

	fList := make([]string, len(files))
	for i, f := range files {
		fList[i] = filepath.ToSlash(filepath.Clean(f))
	}
	sort.Strings(fList)

	pList := make([]string, len(pkgs))
	for i, p := range pkgs {
		pList[i] = filepath.ToSlash(filepath.Clean(p))
	}
	sort.Strings(pList)

	mList := make([]string, len(modules))
	for i, m := range modules {
		mList[i] = filepath.ToSlash(filepath.Clean(m))
	}
	sort.Strings(mList)

	rList := make([]string, len(repos))
	for i, r := range repos {
		rList[i] = filepath.ToSlash(filepath.Clean(r))
	}
	sort.Strings(rList)

	return &SharedConfig{
		id:                   "cfg:" + cleanPath,
		configPath:           cleanPath,
		configFormat:         strings.ToLower(strings.TrimSpace(format)),
		affectedFiles:        fList,
		affectedPackages:     pList,
		affectedModules:      mList,
		affectedRepositories: rList,
	}
}

func (c *SharedConfig) ID() string           { return c.id }
func (c *SharedConfig) ConfigPath() string   { return c.configPath }
func (c *SharedConfig) ConfigFormat() string { return c.configFormat }

func (c *SharedConfig) AffectedFiles() []string {
	if c == nil || c.affectedFiles == nil {
		return nil
	}
	res := make([]string, len(c.affectedFiles))
	copy(res, c.affectedFiles)
	return res
}

func (c *SharedConfig) AffectedPackages() []string {
	if c == nil || c.affectedPackages == nil {
		return nil
	}
	res := make([]string, len(c.affectedPackages))
	copy(res, c.affectedPackages)
	return res
}

func (c *SharedConfig) AffectedModules() []string {
	if c == nil || c.affectedModules == nil {
		return nil
	}
	res := make([]string, len(c.affectedModules))
	copy(res, c.affectedModules)
	return res
}

func (c *SharedConfig) AffectedRepositories() []string {
	if c == nil || c.affectedRepositories == nil {
		return nil
	}
	res := make([]string, len(c.affectedRepositories))
	copy(res, c.affectedRepositories)
	return res
}

// PackageCommunication represents interaction between two packages.
type PackageCommunication struct {
	id            string
	sourcePackage string
	targetPackage string
	kind          PackageCommunicationKind
	symbolsUsed   []string
	calls         []string
	direction     string
}

// NewPackageCommunication creates an immutable PackageCommunication record.
func NewPackageCommunication(
	sourcePkg, targetPkg string,
	kind PackageCommunicationKind,
	symbolsUsed, calls []string,
	direction string,
) *PackageCommunication {
	cleanSrc := filepath.ToSlash(filepath.Clean(sourcePkg))
	cleanTgt := filepath.ToSlash(filepath.Clean(targetPkg))

	symMap := make(map[string]bool)
	var syms []string
	for _, s := range symbolsUsed {
		cleanS := strings.TrimSpace(s)
		if cleanS != "" && !symMap[cleanS] {
			symMap[cleanS] = true
			syms = append(syms, cleanS)
		}
	}
	sort.Strings(syms)

	callMap := make(map[string]bool)
	var callList []string
	for _, c := range calls {
		cleanC := strings.TrimSpace(c)
		if cleanC != "" && !callMap[cleanC] {
			callMap[cleanC] = true
			callList = append(callList, cleanC)
		}
	}
	sort.Strings(callList)

	return &PackageCommunication{
		id:            "pkgcomm:" + cleanSrc + "->" + cleanTgt + ":" + string(kind),
		sourcePackage: cleanSrc,
		targetPackage: cleanTgt,
		kind:          kind,
		symbolsUsed:   syms,
		calls:         callList,
		direction:     strings.TrimSpace(direction),
	}
}

func (c *PackageCommunication) ID() string                     { return c.id }
func (c *PackageCommunication) SourcePackage() string          { return c.sourcePackage }
func (c *PackageCommunication) TargetPackage() string          { return c.targetPackage }
func (c *PackageCommunication) Kind() PackageCommunicationKind { return c.kind }
func (c *PackageCommunication) Direction() string              { return c.direction }

func (c *PackageCommunication) SymbolsUsed() []string {
	if c == nil || c.symbolsUsed == nil {
		return nil
	}
	res := make([]string, len(c.symbolsUsed))
	copy(res, c.symbolsUsed)
	return res
}

func (c *PackageCommunication) Calls() []string {
	if c == nil || c.calls == nil {
		return nil
	}
	res := make([]string, len(c.calls))
	copy(res, c.calls)
	return res
}

// PackageContract represents the public contract and interfaces exposed by a package.
type PackageContract struct {
	id              string
	packagePath     string
	exportedSymbols []string
	publicTypes     []string
	interfaces      []string
	documentation   string
}

// NewPackageContract creates an immutable PackageContract record.
func NewPackageContract(packagePath string, exportedSymbols, publicTypes, interfaces []string, doc string) *PackageContract {
	cleanPkg := filepath.ToSlash(filepath.Clean(packagePath))

	syms := make([]string, len(exportedSymbols))
	copy(syms, exportedSymbols)
	sort.Strings(syms)

	types := make([]string, len(publicTypes))
	copy(types, publicTypes)
	sort.Strings(types)

	ifaces := make([]string, len(interfaces))
	copy(ifaces, interfaces)
	sort.Strings(ifaces)

	return &PackageContract{
		id:              "pkgcontract:" + cleanPkg,
		packagePath:     cleanPkg,
		exportedSymbols: syms,
		publicTypes:     types,
		interfaces:      ifaces,
		documentation:   strings.TrimSpace(doc),
	}
}

func (c *PackageContract) ID() string            { return c.id }
func (c *PackageContract) PackagePath() string   { return c.packagePath }
func (c *PackageContract) Documentation() string { return c.documentation }

func (c *PackageContract) ExportedSymbols() []string {
	if c == nil || c.exportedSymbols == nil {
		return nil
	}
	res := make([]string, len(c.exportedSymbols))
	copy(res, c.exportedSymbols)
	return res
}

func (c *PackageContract) PublicTypes() []string {
	if c == nil || c.publicTypes == nil {
		return nil
	}
	res := make([]string, len(c.publicTypes))
	copy(res, c.publicTypes)
	return res
}

func (c *PackageContract) Interfaces() []string {
	if c == nil || c.interfaces == nil {
		return nil
	}
	res := make([]string, len(c.interfaces))
	copy(res, c.interfaces)
	return res
}

// APIEndpoint describes an API exposed within or outside a module boundary.
type APIEndpoint struct {
	id               string
	symbolID         string
	symbolName       string
	owningPackage    string
	consumerPackages []string
	visibility       APIVisibility
	signature        string
	doc              string
}

// NewAPIEndpoint creates an immutable APIEndpoint record.
func NewAPIEndpoint(
	symbolID, symbolName, owningPkg string,
	consumers []string,
	vis APIVisibility,
	signature, doc string,
) *APIEndpoint {
	cleanSymID := strings.TrimSpace(symbolID)
	cMap := make(map[string]bool)
	var cList []string
	for _, c := range consumers {
		cleanC := filepath.ToSlash(filepath.Clean(c))
		if cleanC != "" && !cMap[cleanC] {
			cMap[cleanC] = true
			cList = append(cList, cleanC)
		}
	}
	sort.Strings(cList)

	return &APIEndpoint{
		id:               "api:" + cleanSymID,
		symbolID:         cleanSymID,
		symbolName:       strings.TrimSpace(symbolName),
		owningPackage:    filepath.ToSlash(filepath.Clean(owningPkg)),
		consumerPackages: cList,
		visibility:       vis,
		signature:        strings.TrimSpace(signature),
		doc:              strings.TrimSpace(doc),
	}
}

func (a *APIEndpoint) ID() string                { return a.id }
func (a *APIEndpoint) SymbolID() string          { return a.symbolID }
func (a *APIEndpoint) SymbolName() string        { return a.symbolName }
func (a *APIEndpoint) OwningPackage() string     { return a.owningPackage }
func (a *APIEndpoint) Visibility() APIVisibility { return a.visibility }
func (a *APIEndpoint) Signature() string         { return a.signature }
func (a *APIEndpoint) Doc() string               { return a.doc }

func (a *APIEndpoint) ConsumerPackages() []string {
	if a == nil || a.consumerPackages == nil {
		return nil
	}
	res := make([]string, len(a.consumerPackages))
	copy(res, a.consumerPackages)
	return res
}

// ModuleRelationship represents a dependency or structural link between modules.
type ModuleRelationship struct {
	id                 string
	sourceModule       string
	targetModule       string
	kind               ModuleRelationKind
	version            string
	dependencyBoundary string
}

// NewModuleRelationship creates an immutable ModuleRelationship record.
func NewModuleRelationship(sourceModule, targetModule string, kind ModuleRelationKind, version, boundary string) *ModuleRelationship {
	cleanSrc := filepath.ToSlash(filepath.Clean(sourceModule))
	cleanTgt := filepath.ToSlash(filepath.Clean(targetModule))

	return &ModuleRelationship{
		id:                 "modrel:" + cleanSrc + "->" + cleanTgt + ":" + string(kind),
		sourceModule:       cleanSrc,
		targetModule:       cleanTgt,
		kind:               kind,
		version:            strings.TrimSpace(version),
		dependencyBoundary: strings.TrimSpace(boundary),
	}
}

func (m *ModuleRelationship) ID() string                 { return m.id }
func (m *ModuleRelationship) SourceModule() string       { return m.sourceModule }
func (m *ModuleRelationship) TargetModule() string       { return m.targetModule }
func (m *ModuleRelationship) Kind() ModuleRelationKind   { return m.kind }
func (m *ModuleRelationship) Version() string            { return m.version }
func (m *ModuleRelationship) DependencyBoundary() string { return m.dependencyBoundary }

// SharedModule represents a module consumed by multiple packages or repositories.
type SharedModule struct {
	id                 string
	modulePath         string
	owningContext      string
	consumingContexts  []string
	sharedSymbols      []string
	sharedDependencies []string
}

// NewSharedModule creates an immutable SharedModule record.
func NewSharedModule(modulePath, owningContext string, consumers, sharedSyms, sharedDeps []string) *SharedModule {
	cleanMod := filepath.ToSlash(filepath.Clean(modulePath))

	cMap := make(map[string]bool)
	var cList []string
	for _, c := range consumers {
		cleanC := filepath.ToSlash(filepath.Clean(c))
		if cleanC != "" && !cMap[cleanC] {
			cMap[cleanC] = true
			cList = append(cList, cleanC)
		}
	}
	sort.Strings(cList)

	sMap := make(map[string]bool)
	var sList []string
	for _, s := range sharedSyms {
		cleanS := strings.TrimSpace(s)
		if cleanS != "" && !sMap[cleanS] {
			sMap[cleanS] = true
			sList = append(sList, cleanS)
		}
	}
	sort.Strings(sList)

	dMap := make(map[string]bool)
	var dList []string
	for _, d := range sharedDeps {
		cleanD := strings.TrimSpace(d)
		if cleanD != "" && !dMap[cleanD] {
			dMap[cleanD] = true
			dList = append(dList, cleanD)
		}
	}
	sort.Strings(dList)

	return &SharedModule{
		id:                 "sharedmod:" + cleanMod,
		modulePath:         cleanMod,
		owningContext:      filepath.ToSlash(filepath.Clean(owningContext)),
		consumingContexts:  cList,
		sharedSymbols:      sList,
		sharedDependencies: dList,
	}
}

func (s *SharedModule) ID() string            { return s.id }
func (s *SharedModule) ModulePath() string    { return s.modulePath }
func (s *SharedModule) OwningContext() string { return s.owningContext }

func (s *SharedModule) ConsumingContexts() []string {
	if s == nil || s.consumingContexts == nil {
		return nil
	}
	res := make([]string, len(s.consumingContexts))
	copy(res, s.consumingContexts)
	return res
}

func (s *SharedModule) SharedSymbols() []string {
	if s == nil || s.sharedSymbols == nil {
		return nil
	}
	res := make([]string, len(s.sharedSymbols))
	copy(res, s.sharedSymbols)
	return res
}

func (s *SharedModule) SharedDependencies() []string {
	if s == nil || s.sharedDependencies == nil {
		return nil
	}
	res := make([]string, len(s.sharedDependencies))
	copy(res, s.sharedDependencies)
	return res
}

// ModuleHierarchyNode represents a node in the module tree.
type ModuleHierarchyNode struct {
	id                string
	modulePath        string
	parentModule      string
	childModules      []string
	containedPackages []string
}

// NewModuleHierarchyNode creates an immutable ModuleHierarchyNode.
func NewModuleHierarchyNode(modulePath, parentModule string, children, packages []string) *ModuleHierarchyNode {
	cleanMod := filepath.ToSlash(filepath.Clean(modulePath))

	cList := make([]string, len(children))
	for i, c := range children {
		cList[i] = filepath.ToSlash(filepath.Clean(c))
	}
	sort.Strings(cList)

	pList := make([]string, len(packages))
	for i, p := range packages {
		pList[i] = filepath.ToSlash(filepath.Clean(p))
	}
	sort.Strings(pList)

	return &ModuleHierarchyNode{
		id:                "modnode:" + cleanMod,
		modulePath:        cleanMod,
		parentModule:      filepath.ToSlash(filepath.Clean(parentModule)),
		childModules:      cList,
		containedPackages: pList,
	}
}

func (n *ModuleHierarchyNode) ID() string           { return n.id }
func (n *ModuleHierarchyNode) ModulePath() string   { return n.modulePath }
func (n *ModuleHierarchyNode) ParentModule() string { return n.parentModule }

func (n *ModuleHierarchyNode) ChildModules() []string {
	if n == nil || n.childModules == nil {
		return nil
	}
	res := make([]string, len(n.childModules))
	copy(res, n.childModules)
	return res
}

func (n *ModuleHierarchyNode) ContainedPackages() []string {
	if n == nil || n.containedPackages == nil {
		return nil
	}
	res := make([]string, len(n.containedPackages))
	copy(res, n.containedPackages)
	return res
}

// VersionCompatibility represents compatibility analysis between versioned components.
type VersionCompatibility struct {
	id              string
	modulePath      string
	requiredVersion string
	resolvedVersion string
	state           VersionCompatibilityState
	details         string
}

// NewVersionCompatibility creates an immutable VersionCompatibility record.
func NewVersionCompatibility(modulePath, requiredVersion, resolvedVersion string, state VersionCompatibilityState, details string) *VersionCompatibility {
	cleanMod := filepath.ToSlash(filepath.Clean(modulePath))

	return &VersionCompatibility{
		id:              "compat:" + cleanMod + ":" + strings.TrimSpace(requiredVersion),
		modulePath:      cleanMod,
		requiredVersion: strings.TrimSpace(requiredVersion),
		resolvedVersion: strings.TrimSpace(resolvedVersion),
		state:           state,
		details:         strings.TrimSpace(details),
	}
}

func (v *VersionCompatibility) ID() string                       { return v.id }
func (v *VersionCompatibility) ModulePath() string               { return v.modulePath }
func (v *VersionCompatibility) RequiredVersion() string          { return v.requiredVersion }
func (v *VersionCompatibility) ResolvedVersion() string          { return v.resolvedVersion }
func (v *VersionCompatibility) State() VersionCompatibilityState { return v.state }
func (v *VersionCompatibility) Details() string                  { return v.details }

// WorkspaceRepository represents a repository participating in a workspace.
type WorkspaceRepository struct {
	id           string
	root         string
	name         string
	modules      []string
	packages     []string
	dependencies []string
}

// NewWorkspaceRepository creates an immutable WorkspaceRepository record.
func NewWorkspaceRepository(root, name string, modules, packages, dependencies []string) *WorkspaceRepository {
	cleanRoot := filepath.ToSlash(filepath.Clean(root))
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		cleanName = filepath.Base(cleanRoot)
	}

	mList := make([]string, len(modules))
	for i, m := range modules {
		mList[i] = filepath.ToSlash(filepath.Clean(m))
	}
	sort.Strings(mList)

	pList := make([]string, len(packages))
	for i, p := range packages {
		pList[i] = filepath.ToSlash(filepath.Clean(p))
	}
	sort.Strings(pList)

	dList := make([]string, len(dependencies))
	copy(dList, dependencies)
	sort.Strings(dList)

	return &WorkspaceRepository{
		id:           "repo:" + cleanRoot,
		root:         cleanRoot,
		name:         cleanName,
		modules:      mList,
		packages:     pList,
		dependencies: dList,
	}
}

func (r *WorkspaceRepository) ID() string   { return r.id }
func (r *WorkspaceRepository) Root() string { return r.root }
func (r *WorkspaceRepository) Name() string { return r.name }

func (r *WorkspaceRepository) Modules() []string {
	if r == nil || r.modules == nil {
		return nil
	}
	res := make([]string, len(r.modules))
	copy(res, r.modules)
	return res
}

func (r *WorkspaceRepository) Packages() []string {
	if r == nil || r.packages == nil {
		return nil
	}
	res := make([]string, len(r.packages))
	copy(res, r.packages)
	return res
}

func (r *WorkspaceRepository) Dependencies() []string {
	if r == nil || r.dependencies == nil {
		return nil
	}
	res := make([]string, len(r.dependencies))
	copy(res, r.dependencies)
	return res
}

// WorkspaceRelationship describes an inter-repository link within a workspace.
type WorkspaceRelationship struct {
	id           string
	sourceRepoID string
	targetRepoID string
	kind         WorkspaceRelationKind
	evidence     string
}

// NewWorkspaceRelationship creates an immutable WorkspaceRelationship.
func NewWorkspaceRelationship(sourceRepoID, targetRepoID string, kind WorkspaceRelationKind, evidence string) *WorkspaceRelationship {
	cleanSrc := strings.TrimSpace(sourceRepoID)
	cleanTgt := strings.TrimSpace(targetRepoID)

	return &WorkspaceRelationship{
		id:           "wsrel:" + cleanSrc + "->" + cleanTgt + ":" + string(kind),
		sourceRepoID: cleanSrc,
		targetRepoID: cleanTgt,
		kind:         kind,
		evidence:     strings.TrimSpace(evidence),
	}
}

func (r *WorkspaceRelationship) ID() string                  { return r.id }
func (r *WorkspaceRelationship) SourceRepoID() string        { return r.sourceRepoID }
func (r *WorkspaceRelationship) TargetRepoID() string        { return r.targetRepoID }
func (r *WorkspaceRelationship) Kind() WorkspaceRelationKind { return r.kind }
func (r *WorkspaceRelationship) Evidence() string            { return r.evidence }

// SharedDependency represents a dependency consumed across multiple repositories.
type SharedDependency struct {
	id               string
	dependencyName   string
	version          string
	consumingRepos   []string
	consumingModules []string
}

// NewSharedDependency creates an immutable SharedDependency record.
func NewSharedDependency(name, version string, consumingRepos, consumingModules []string) *SharedDependency {
	cleanName := strings.TrimSpace(name)

	rList := make([]string, len(consumingRepos))
	copy(rList, consumingRepos)
	sort.Strings(rList)

	mList := make([]string, len(consumingModules))
	copy(mList, consumingModules)
	sort.Strings(mList)

	return &SharedDependency{
		id:               "shareddep:" + cleanName + ":" + strings.TrimSpace(version),
		dependencyName:   cleanName,
		version:          strings.TrimSpace(version),
		consumingRepos:   rList,
		consumingModules: mList,
	}
}

func (d *SharedDependency) ID() string             { return d.id }
func (d *SharedDependency) DependencyName() string { return d.dependencyName }
func (d *SharedDependency) Version() string        { return d.version }

func (d *SharedDependency) ConsumingRepos() []string {
	if d == nil || d.consumingRepos == nil {
		return nil
	}
	res := make([]string, len(d.consumingRepos))
	copy(res, d.consumingRepos)
	return res
}

func (d *SharedDependency) ConsumingModules() []string {
	if d == nil || d.consumingModules == nil {
		return nil
	}
	res := make([]string, len(d.consumingModules))
	copy(res, d.consumingModules)
	return res
}

// SharedArchitecture describes common architectural patterns and service boundaries across repositories.
type SharedArchitecture struct {
	id                 string
	name               string
	description        string
	participatingRepos []string
	commonModules      []string
	serviceBoundaries  []string
}

// NewSharedArchitecture creates an immutable SharedArchitecture record.
func NewSharedArchitecture(name, description string, repos, commonMods, serviceBounds []string) *SharedArchitecture {
	cleanName := strings.TrimSpace(name)

	rList := make([]string, len(repos))
	copy(rList, repos)
	sort.Strings(rList)

	mList := make([]string, len(commonMods))
	copy(mList, commonMods)
	sort.Strings(mList)

	sList := make([]string, len(serviceBounds))
	copy(sList, serviceBounds)
	sort.Strings(sList)

	return &SharedArchitecture{
		id:                 "arch:" + cleanName,
		name:               cleanName,
		description:        strings.TrimSpace(description),
		participatingRepos: rList,
		commonModules:      mList,
		serviceBoundaries:  sList,
	}
}

func (a *SharedArchitecture) ID() string          { return a.id }
func (a *SharedArchitecture) Name() string        { return a.name }
func (a *SharedArchitecture) Description() string { return a.description }

func (a *SharedArchitecture) ParticipatingRepos() []string {
	if a == nil || a.participatingRepos == nil {
		return nil
	}
	res := make([]string, len(a.participatingRepos))
	copy(res, a.participatingRepos)
	return res
}

func (a *SharedArchitecture) CommonModules() []string {
	if a == nil || a.commonModules == nil {
		return nil
	}
	res := make([]string, len(a.commonModules))
	copy(res, a.commonModules)
	return res
}

func (a *SharedArchitecture) ServiceBoundaries() []string {
	if a == nil || a.serviceBoundaries == nil {
		return nil
	}
	res := make([]string, len(a.serviceBoundaries))
	copy(res, a.serviceBoundaries)
	return res
}

// WorkspaceModel represents multi-repository workspace intelligence.
type WorkspaceModel struct {
	id                 string
	workspaceRoot      string
	repositories       []*WorkspaceRepository
	relationships      []*WorkspaceRelationship
	sharedDependencies []*SharedDependency
	sharedConfigs      []*SharedConfig
	sharedArchitecture []*SharedArchitecture
}

// NewWorkspaceModel creates an immutable WorkspaceModel.
func NewWorkspaceModel(
	root string,
	repos []*WorkspaceRepository,
	rels []*WorkspaceRelationship,
	sharedDeps []*SharedDependency,
	sharedConfigs []*SharedConfig,
	sharedArch []*SharedArchitecture,
) *WorkspaceModel {
	cleanRoot := filepath.ToSlash(filepath.Clean(root))

	rList := make([]*WorkspaceRepository, len(repos))
	copy(rList, repos)
	sort.Slice(rList, func(i, j int) bool {
		return rList[i].Root() < rList[j].Root()
	})

	relList := make([]*WorkspaceRelationship, len(rels))
	copy(relList, rels)
	sort.Slice(relList, func(i, j int) bool {
		return relList[i].ID() < relList[j].ID()
	})

	dList := make([]*SharedDependency, len(sharedDeps))
	copy(dList, sharedDeps)
	sort.Slice(dList, func(i, j int) bool {
		return dList[i].ID() < dList[j].ID()
	})

	cList := make([]*SharedConfig, len(sharedConfigs))
	copy(cList, sharedConfigs)
	sort.Slice(cList, func(i, j int) bool {
		return cList[i].ID() < cList[j].ID()
	})

	aList := make([]*SharedArchitecture, len(sharedArch))
	copy(aList, sharedArch)
	sort.Slice(aList, func(i, j int) bool {
		return aList[i].ID() < aList[j].ID()
	})

	return &WorkspaceModel{
		id:                 "workspace:" + cleanRoot,
		workspaceRoot:      cleanRoot,
		repositories:       rList,
		relationships:      relList,
		sharedDependencies: dList,
		sharedConfigs:      cList,
		sharedArchitecture: aList,
	}
}

func (w *WorkspaceModel) ID() string            { return w.id }
func (w *WorkspaceModel) WorkspaceRoot() string { return w.workspaceRoot }

func (w *WorkspaceModel) Repositories() []*WorkspaceRepository {
	if w == nil || w.repositories == nil {
		return nil
	}
	res := make([]*WorkspaceRepository, len(w.repositories))
	copy(res, w.repositories)
	return res
}

func (w *WorkspaceModel) Relationships() []*WorkspaceRelationship {
	if w == nil || w.relationships == nil {
		return nil
	}
	res := make([]*WorkspaceRelationship, len(w.relationships))
	copy(res, w.relationships)
	return res
}

func (w *WorkspaceModel) SharedDependencies() []*SharedDependency {
	if w == nil || w.sharedDependencies == nil {
		return nil
	}
	res := make([]*SharedDependency, len(w.sharedDependencies))
	copy(res, w.sharedDependencies)
	return res
}

func (w *WorkspaceModel) SharedConfigs() []*SharedConfig {
	if w == nil || w.sharedConfigs == nil {
		return nil
	}
	res := make([]*SharedConfig, len(w.sharedConfigs))
	copy(res, w.sharedConfigs)
	return res
}

func (w *WorkspaceModel) SharedArchitecture() []*SharedArchitecture {
	if w == nil || w.sharedArchitecture == nil {
		return nil
	}
	res := make([]*SharedArchitecture, len(w.sharedArchitecture))
	copy(res, w.sharedArchitecture)
	return res
}

// CommitEvent represents a point-in-time change commit in repository history.
type CommitEvent struct {
	commitHash    string
	author        string
	timestamp     time.Time
	message       string
	filesAdded    []string
	filesModified []string
	filesDeleted  []string
}

// NewCommitEvent creates an immutable CommitEvent record.
func NewCommitEvent(hash, author, message string, ts time.Time, added, modified, deleted []string) *CommitEvent {
	aList := make([]string, len(added))
	copy(aList, added)
	sort.Strings(aList)

	mList := make([]string, len(modified))
	copy(mList, modified)
	sort.Strings(mList)

	dList := make([]string, len(deleted))
	copy(dList, deleted)
	sort.Strings(dList)

	return &CommitEvent{
		commitHash:    strings.TrimSpace(hash),
		author:        strings.TrimSpace(author),
		timestamp:     ts,
		message:       strings.TrimSpace(message),
		filesAdded:    aList,
		filesModified: mList,
		filesDeleted:  dList,
	}
}

func (c *CommitEvent) CommitHash() string   { return c.commitHash }
func (c *CommitEvent) Author() string       { return c.author }
func (c *CommitEvent) Timestamp() time.Time { return c.timestamp }
func (c *CommitEvent) Message() string      { return c.message }

func (c *CommitEvent) FilesAdded() []string {
	if c == nil || c.filesAdded == nil {
		return nil
	}
	res := make([]string, len(c.filesAdded))
	copy(res, c.filesAdded)
	return res
}

func (c *CommitEvent) FilesModified() []string {
	if c == nil || c.filesModified == nil {
		return nil
	}
	res := make([]string, len(c.filesModified))
	copy(res, c.filesModified)
	return res
}

func (c *CommitEvent) FilesDeleted() []string {
	if c == nil || c.filesDeleted == nil {
		return nil
	}
	res := make([]string, len(c.filesDeleted))
	copy(res, c.filesDeleted)
	return res
}

// StructuralEvolution captures point-in-time structural counts and changes.
type StructuralEvolution struct {
	id           string
	timestamp    time.Time
	addedFiles   int
	removedFiles int
	movedFiles   int
	packageCount int
	moduleCount  int
}

// NewStructuralEvolution creates an immutable StructuralEvolution record.
func NewStructuralEvolution(id string, ts time.Time, added, removed, moved, pkgs, mods int) *StructuralEvolution {
	return &StructuralEvolution{
		id:           strings.TrimSpace(id),
		timestamp:    ts,
		addedFiles:   added,
		removedFiles: removed,
		movedFiles:   moved,
		packageCount: pkgs,
		moduleCount:  mods,
	}
}

func (s *StructuralEvolution) ID() string           { return s.id }
func (s *StructuralEvolution) Timestamp() time.Time { return s.timestamp }
func (s *StructuralEvolution) AddedFiles() int      { return s.addedFiles }
func (s *StructuralEvolution) RemovedFiles() int    { return s.removedFiles }
func (s *StructuralEvolution) MovedFiles() int      { return s.movedFiles }
func (s *StructuralEvolution) PackageCount() int    { return s.packageCount }
func (s *StructuralEvolution) ModuleCount() int     { return s.moduleCount }

// ArchitectureEvolution captures boundary and directional shifts over time.
type ArchitectureEvolution struct {
	id               string
	timestamp        time.Time
	boundaryChanges  []string
	directionChanges []string
	reorganizations  []string
}

// NewArchitectureEvolution creates an immutable ArchitectureEvolution record.
func NewArchitectureEvolution(id string, ts time.Time, boundaries, directions, reorgs []string) *ArchitectureEvolution {
	bList := make([]string, len(boundaries))
	copy(bList, boundaries)
	sort.Strings(bList)

	dList := make([]string, len(directions))
	copy(dList, directions)
	sort.Strings(dList)

	rList := make([]string, len(reorgs))
	copy(rList, reorgs)
	sort.Strings(rList)

	return &ArchitectureEvolution{
		id:               strings.TrimSpace(id),
		timestamp:        ts,
		boundaryChanges:  bList,
		directionChanges: dList,
		reorganizations:  rList,
	}
}

func (a *ArchitectureEvolution) ID() string           { return a.id }
func (a *ArchitectureEvolution) Timestamp() time.Time { return a.timestamp }

func (a *ArchitectureEvolution) BoundaryChanges() []string {
	if a == nil || a.boundaryChanges == nil {
		return nil
	}
	res := make([]string, len(a.boundaryChanges))
	copy(res, a.boundaryChanges)
	return res
}

func (a *ArchitectureEvolution) DirectionChanges() []string {
	if a == nil || a.directionChanges == nil {
		return nil
	}
	res := make([]string, len(a.directionChanges))
	copy(res, a.directionChanges)
	return res
}

func (a *ArchitectureEvolution) Reorganizations() []string {
	if a == nil || a.reorganizations == nil {
		return nil
	}
	res := make([]string, len(a.reorganizations))
	copy(res, a.reorganizations)
	return res
}

// DependencyEvolution records dependency shifts and version changes over time.
type DependencyEvolution struct {
	id             string
	timestamp      time.Time
	addedDeps      []string
	removedDeps    []string
	versionChanges map[string]string
}

// NewDependencyEvolution creates an immutable DependencyEvolution record.
func NewDependencyEvolution(id string, ts time.Time, added, removed []string, versionChanges map[string]string) *DependencyEvolution {
	aList := make([]string, len(added))
	copy(aList, added)
	sort.Strings(aList)

	rList := make([]string, len(removed))
	copy(rList, removed)
	sort.Strings(rList)

	vMap := make(map[string]string, len(versionChanges))
	for k, v := range versionChanges {
		vMap[k] = v
	}

	return &DependencyEvolution{
		id:             strings.TrimSpace(id),
		timestamp:      ts,
		addedDeps:      aList,
		removedDeps:    rList,
		versionChanges: vMap,
	}
}

func (d *DependencyEvolution) ID() string           { return d.id }
func (d *DependencyEvolution) Timestamp() time.Time { return d.timestamp }

func (d *DependencyEvolution) AddedDeps() []string {
	if d == nil || d.addedDeps == nil {
		return nil
	}
	res := make([]string, len(d.addedDeps))
	copy(res, d.addedDeps)
	return res
}

func (d *DependencyEvolution) RemovedDeps() []string {
	if d == nil || d.removedDeps == nil {
		return nil
	}
	res := make([]string, len(d.removedDeps))
	copy(res, d.removedDeps)
	return res
}

func (d *DependencyEvolution) VersionChanges() map[string]string {
	if d == nil || d.versionChanges == nil {
		return nil
	}
	res := make(map[string]string, len(d.versionChanges))
	for k, v := range d.versionChanges {
		res[k] = v
	}
	return res
}

// GrowthMetrics represents measurable changes in repository size and complexity.
type GrowthMetrics struct {
	id                string
	timestamp         time.Time
	totalFiles        int
	totalPackages     int
	totalModules      int
	totalSymbols      int
	totalDependencies int
	fileGrowthRate    float64
	packageGrowthRate float64
}

// NewGrowthMetrics creates an immutable GrowthMetrics record.
func NewGrowthMetrics(id string, ts time.Time, files, pkgs, mods, syms, deps int, fileRate, pkgRate float64) *GrowthMetrics {
	return &GrowthMetrics{
		id:                strings.TrimSpace(id),
		timestamp:         ts,
		totalFiles:        files,
		totalPackages:     pkgs,
		totalModules:      mods,
		totalSymbols:      syms,
		totalDependencies: deps,
		fileGrowthRate:    fileRate,
		packageGrowthRate: pkgRate,
	}
}

func (g *GrowthMetrics) ID() string                 { return g.id }
func (g *GrowthMetrics) Timestamp() time.Time       { return g.timestamp }
func (g *GrowthMetrics) TotalFiles() int            { return g.totalFiles }
func (g *GrowthMetrics) TotalPackages() int         { return g.totalPackages }
func (g *GrowthMetrics) TotalModules() int          { return g.totalModules }
func (g *GrowthMetrics) TotalSymbols() int          { return g.totalSymbols }
func (g *GrowthMetrics) TotalDependencies() int     { return g.totalDependencies }
func (g *GrowthMetrics) FileGrowthRate() float64    { return g.fileGrowthRate }
func (g *GrowthMetrics) PackageGrowthRate() float64 { return g.packageGrowthRate }

// EvolutionModel represents historical evolution across a repository.
type EvolutionModel struct {
	repoID                string
	commits               []*CommitEvent
	structuralEvolution   []*StructuralEvolution
	architectureEvolution []*ArchitectureEvolution
	dependencyEvolution   []*DependencyEvolution
	growthMetrics         []*GrowthMetrics
}

// NewEvolutionModel creates an immutable EvolutionModel.
func NewEvolutionModel(
	repoID string,
	commits []*CommitEvent,
	structural []*StructuralEvolution,
	arch []*ArchitectureEvolution,
	deps []*DependencyEvolution,
	growth []*GrowthMetrics,
) *EvolutionModel {
	cList := make([]*CommitEvent, len(commits))
	copy(cList, commits)
	sort.Slice(cList, func(i, j int) bool {
		return cList[i].Timestamp().Before(cList[j].Timestamp())
	})

	sList := make([]*StructuralEvolution, len(structural))
	copy(sList, structural)
	sort.Slice(sList, func(i, j int) bool {
		return sList[i].Timestamp().Before(sList[j].Timestamp())
	})

	aList := make([]*ArchitectureEvolution, len(arch))
	copy(aList, arch)
	sort.Slice(aList, func(i, j int) bool {
		return aList[i].Timestamp().Before(aList[j].Timestamp())
	})

	dList := make([]*DependencyEvolution, len(deps))
	copy(dList, deps)
	sort.Slice(dList, func(i, j int) bool {
		return dList[i].Timestamp().Before(dList[j].Timestamp())
	})

	gList := make([]*GrowthMetrics, len(growth))
	copy(gList, growth)
	sort.Slice(gList, func(i, j int) bool {
		return gList[i].Timestamp().Before(gList[j].Timestamp())
	})

	return &EvolutionModel{
		repoID:                strings.TrimSpace(repoID),
		commits:               cList,
		structuralEvolution:   sList,
		architectureEvolution: aList,
		dependencyEvolution:   dList,
		growthMetrics:         gList,
	}
}

func (e *EvolutionModel) RepoID() string { return e.repoID }

func (e *EvolutionModel) Commits() []*CommitEvent {
	if e == nil || e.commits == nil {
		return nil
	}
	res := make([]*CommitEvent, len(e.commits))
	copy(res, e.commits)
	return res
}

func (e *EvolutionModel) StructuralEvolution() []*StructuralEvolution {
	if e == nil || e.structuralEvolution == nil {
		return nil
	}
	res := make([]*StructuralEvolution, len(e.structuralEvolution))
	copy(res, e.structuralEvolution)
	return res
}

func (e *EvolutionModel) ArchitectureEvolution() []*ArchitectureEvolution {
	if e == nil || e.architectureEvolution == nil {
		return nil
	}
	res := make([]*ArchitectureEvolution, len(e.architectureEvolution))
	copy(res, e.architectureEvolution)
	return res
}

func (e *EvolutionModel) DependencyEvolution() []*DependencyEvolution {
	if e == nil || e.dependencyEvolution == nil {
		return nil
	}
	res := make([]*DependencyEvolution, len(e.dependencyEvolution))
	copy(res, e.dependencyEvolution)
	return res
}

func (e *EvolutionModel) GrowthMetrics() []*GrowthMetrics {
	if e == nil || e.growthMetrics == nil {
		return nil
	}
	res := make([]*GrowthMetrics, len(e.growthMetrics))
	copy(res, e.growthMetrics)
	return res
}

// CrossRepoModel represents the consolidated immutable snapshot of Cross Repository Intelligence.
type CrossRepoModel struct {
	fileRelationships      []*FileRelationship
	symbolPropagations     []*SymbolPropagation
	crossFileDependencies  []*CrossFileDependency
	sharedConfigs          []*SharedConfig
	packageCommunications  []*PackageCommunication
	packageContracts       []*PackageContract
	apiEndpoints           []*APIEndpoint
	moduleRelationships    []*ModuleRelationship
	sharedModules          []*SharedModule
	moduleHierarchy        []*ModuleHierarchyNode
	versionCompatibilities []*VersionCompatibility
	workspace              *WorkspaceModel
	evolution              *EvolutionModel
	validationReport       *ValidationReport
	analyzedAt             time.Time
}

// NewCrossRepoModel creates an immutable CrossRepoModel.
func NewCrossRepoModel(
	fileRels []*FileRelationship,
	symbolProps []*SymbolPropagation,
	crossFileDeps []*CrossFileDependency,
	sharedConfigs []*SharedConfig,
	pkgComms []*PackageCommunication,
	pkgContracts []*PackageContract,
	apis []*APIEndpoint,
	modRels []*ModuleRelationship,
	sharedMods []*SharedModule,
	modHierarchy []*ModuleHierarchyNode,
	versionCompats []*VersionCompatibility,
	ws *WorkspaceModel,
	evo *EvolutionModel,
	report *ValidationReport,
	analyzedAt time.Time,
) *CrossRepoModel {
	fRels := make([]*FileRelationship, len(fileRels))
	copy(fRels, fileRels)
	sort.Slice(fRels, func(i, j int) bool { return fRels[i].ID() < fRels[j].ID() })

	sProps := make([]*SymbolPropagation, len(symbolProps))
	copy(sProps, symbolProps)
	sort.Slice(sProps, func(i, j int) bool { return sProps[i].ID() < sProps[j].ID() })

	cfDeps := make([]*CrossFileDependency, len(crossFileDeps))
	copy(cfDeps, crossFileDeps)
	sort.Slice(cfDeps, func(i, j int) bool { return cfDeps[i].ID() < cfDeps[j].ID() })

	sConfigs := make([]*SharedConfig, len(sharedConfigs))
	copy(sConfigs, sharedConfigs)
	sort.Slice(sConfigs, func(i, j int) bool { return sConfigs[i].ID() < sConfigs[j].ID() })

	pComms := make([]*PackageCommunication, len(pkgComms))
	copy(pComms, pkgComms)
	sort.Slice(pComms, func(i, j int) bool { return pComms[i].ID() < pComms[j].ID() })

	pContracts := make([]*PackageContract, len(pkgContracts))
	copy(pContracts, pkgContracts)
	sort.Slice(pContracts, func(i, j int) bool { return pContracts[i].ID() < pContracts[j].ID() })

	apiList := make([]*APIEndpoint, len(apis))
	copy(apiList, apis)
	sort.Slice(apiList, func(i, j int) bool { return apiList[i].ID() < apiList[j].ID() })

	mRels := make([]*ModuleRelationship, len(modRels))
	copy(mRels, modRels)
	sort.Slice(mRels, func(i, j int) bool { return mRels[i].ID() < mRels[j].ID() })

	sMods := make([]*SharedModule, len(sharedMods))
	copy(sMods, sharedMods)
	sort.Slice(sMods, func(i, j int) bool { return sMods[i].ID() < sMods[j].ID() })

	mHierarchy := make([]*ModuleHierarchyNode, len(modHierarchy))
	copy(mHierarchy, modHierarchy)
	sort.Slice(mHierarchy, func(i, j int) bool { return mHierarchy[i].ID() < mHierarchy[j].ID() })

	vCompats := make([]*VersionCompatibility, len(versionCompats))
	copy(vCompats, versionCompats)
	sort.Slice(vCompats, func(i, j int) bool { return vCompats[i].ID() < vCompats[j].ID() })

	return &CrossRepoModel{
		fileRelationships:      fRels,
		symbolPropagations:     sProps,
		crossFileDependencies:  cfDeps,
		sharedConfigs:          sConfigs,
		packageCommunications:  pComms,
		packageContracts:       pContracts,
		apiEndpoints:           apiList,
		moduleRelationships:    mRels,
		sharedModules:          sMods,
		moduleHierarchy:        mHierarchy,
		versionCompatibilities: vCompats,
		workspace:              ws,
		evolution:              evo,
		validationReport:       report,
		analyzedAt:             analyzedAt,
	}
}

func (m *CrossRepoModel) Workspace() *WorkspaceModel          { return m.workspace }
func (m *CrossRepoModel) Evolution() *EvolutionModel          { return m.evolution }
func (m *CrossRepoModel) ValidationReport() *ValidationReport { return m.validationReport }
func (m *CrossRepoModel) AnalyzedAt() time.Time               { return m.analyzedAt }

func (m *CrossRepoModel) FileRelationships() []*FileRelationship {
	if m == nil || m.fileRelationships == nil {
		return nil
	}
	res := make([]*FileRelationship, len(m.fileRelationships))
	copy(res, m.fileRelationships)
	return res
}

func (m *CrossRepoModel) SymbolPropagations() []*SymbolPropagation {
	if m == nil || m.symbolPropagations == nil {
		return nil
	}
	res := make([]*SymbolPropagation, len(m.symbolPropagations))
	copy(res, m.symbolPropagations)
	return res
}

func (m *CrossRepoModel) CrossFileDependencies() []*CrossFileDependency {
	if m == nil || m.crossFileDependencies == nil {
		return nil
	}
	res := make([]*CrossFileDependency, len(m.crossFileDependencies))
	copy(res, m.crossFileDependencies)
	return res
}

func (m *CrossRepoModel) SharedConfigs() []*SharedConfig {
	if m == nil || m.sharedConfigs == nil {
		return nil
	}
	res := make([]*SharedConfig, len(m.sharedConfigs))
	copy(res, m.sharedConfigs)
	return res
}

func (m *CrossRepoModel) PackageCommunications() []*PackageCommunication {
	if m == nil || m.packageCommunications == nil {
		return nil
	}
	res := make([]*PackageCommunication, len(m.packageCommunications))
	copy(res, m.packageCommunications)
	return res
}

func (m *CrossRepoModel) PackageContracts() []*PackageContract {
	if m == nil || m.packageContracts == nil {
		return nil
	}
	res := make([]*PackageContract, len(m.packageContracts))
	copy(res, m.packageContracts)
	return res
}

func (m *CrossRepoModel) APIEndpoints() []*APIEndpoint {
	if m == nil || m.apiEndpoints == nil {
		return nil
	}
	res := make([]*APIEndpoint, len(m.apiEndpoints))
	copy(res, m.apiEndpoints)
	return res
}

func (m *CrossRepoModel) ModuleRelationships() []*ModuleRelationship {
	if m == nil || m.moduleRelationships == nil {
		return nil
	}
	res := make([]*ModuleRelationship, len(m.moduleRelationships))
	copy(res, m.moduleRelationships)
	return res
}

func (m *CrossRepoModel) SharedModules() []*SharedModule {
	if m == nil || m.sharedModules == nil {
		return nil
	}
	res := make([]*SharedModule, len(m.sharedModules))
	copy(res, m.sharedModules)
	return res
}

func (m *CrossRepoModel) ModuleHierarchy() []*ModuleHierarchyNode {
	if m == nil || m.moduleHierarchy == nil {
		return nil
	}
	res := make([]*ModuleHierarchyNode, len(m.moduleHierarchy))
	copy(res, m.moduleHierarchy)
	return res
}

func (m *CrossRepoModel) VersionCompatibilities() []*VersionCompatibility {
	if m == nil || m.versionCompatibilities == nil {
		return nil
	}
	res := make([]*VersionCompatibility, len(m.versionCompatibilities))
	copy(res, m.versionCompatibilities)
	return res
}
