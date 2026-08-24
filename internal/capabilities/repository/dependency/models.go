package dependency

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// LicenseInfo holds license metadata associated with a dependency.
type LicenseInfo struct {
	licenseType LicenseType
	expression  string
	source      string
	isAvailable bool
}

// NewLicenseInfo creates a new immutable LicenseInfo record.
func NewLicenseInfo(lType LicenseType, expression, source string, available bool) *LicenseInfo {
	return &LicenseInfo{
		licenseType: lType,
		expression:  strings.TrimSpace(expression),
		source:      strings.TrimSpace(source),
		isAvailable: available,
	}
}

// Type returns the license type classification.
func (li *LicenseInfo) Type() LicenseType {
	if li == nil {
		return LicenseUnavailable
	}
	return li.licenseType
}

// Expression returns the raw license expression.
func (li *LicenseInfo) Expression() string {
	if li == nil {
		return ""
	}
	return li.expression
}

// Source returns the origin source of the license information.
func (li *LicenseInfo) Source() string {
	if li == nil {
		return ""
	}
	return li.source
}

// IsAvailable reports whether license metadata was available.
func (li *LicenseInfo) IsAvailable() bool {
	if li == nil {
		return false
	}
	return li.isAvailable
}

// HealthInfo holds maintenance and activity health signals for a dependency.
type HealthInfo struct {
	status       HealthStatus
	isArchived   bool
	isDeprecated bool
	isAbandoned  bool
	isActive     bool
	healthScore  float64
	signals      []string
}

// NewHealthInfo creates a new immutable HealthInfo record.
func NewHealthInfo(
	status HealthStatus,
	archived bool,
	deprecated bool,
	abandoned bool,
	active bool,
	score float64,
	signals []string,
) *HealthInfo {
	sigList := make([]string, len(signals))
	copy(sigList, signals)
	sort.Strings(sigList)

	return &HealthInfo{
		status:       status,
		isArchived:   archived,
		isDeprecated: deprecated,
		isAbandoned:  abandoned,
		isActive:     active,
		healthScore:  score,
		signals:      sigList,
	}
}

// Status returns the overall health status classification.
func (hi *HealthInfo) Status() HealthStatus {
	if hi == nil {
		return HealthUnknown
	}
	return hi.status
}

// IsArchived reports whether the dependency repository is archived.
func (hi *HealthInfo) IsArchived() bool {
	if hi == nil {
		return false
	}
	return hi.isArchived
}

// IsDeprecated reports whether the dependency is marked deprecated.
func (hi *HealthInfo) IsDeprecated() bool {
	if hi == nil {
		return false
	}
	return hi.isDeprecated
}

// IsAbandoned reports whether the dependency is considered abandoned.
func (hi *HealthInfo) IsAbandoned() bool {
	if hi == nil {
		return false
	}
	return hi.isAbandoned
}

// IsActive reports whether active ongoing maintenance is observed.
func (hi *HealthInfo) IsActive() bool {
	if hi == nil {
		return false
	}
	return hi.isActive
}

// HealthScore returns the composite health score (0.0 to 1.0).
func (hi *HealthInfo) HealthScore() float64 {
	if hi == nil {
		return 0.0
	}
	return hi.healthScore
}

// Signals returns a defensive copy of observed health signals.
func (hi *HealthInfo) Signals() []string {
	if hi == nil || len(hi.signals) == 0 {
		return nil
	}
	cloned := make([]string, len(hi.signals))
	copy(cloned, hi.signals)
	return cloned
}

// Dependency represents an internal or external dependency declaration.
type Dependency struct {
	name            string
	version         *SemanticVersion
	declaredVersion string
	ecosystem       Ecosystem
	depType         DependencyType
	isDirect        bool
	isIndirect      bool
	isInternal      bool
	isExternal      bool
	sourceManifest  string
	modulePath      string
	license         *LicenseInfo
	health          *HealthInfo
}

// NewDependency constructs an immutable Dependency record.
func NewDependency(
	name string,
	declaredVer string,
	eco Ecosystem,
	depType DependencyType,
	isDirect bool,
	isIndirect bool,
	isInternal bool,
	isExternal bool,
	sourceManifest string,
	modulePath string,
	license *LicenseInfo,
	health *HealthInfo,
) *Dependency {
	cleanName := strings.TrimSpace(name)
	cleanManifest := filepath.ToSlash(filepath.Clean(sourceManifest))
	cleanModule := filepath.ToSlash(filepath.Clean(modulePath))

	semVer := ParseSemanticVersion(declaredVer)

	return &Dependency{
		name:            cleanName,
		version:         semVer,
		declaredVersion: strings.TrimSpace(declaredVer),
		ecosystem:       eco,
		depType:         depType,
		isDirect:        isDirect,
		isIndirect:      isIndirect,
		isInternal:      isInternal,
		isExternal:      isExternal,
		sourceManifest:  cleanManifest,
		modulePath:      cleanModule,
		license:         license,
		health:          health,
	}
}

// Name returns the package/module dependency name.
func (d *Dependency) Name() string {
	if d == nil {
		return ""
	}
	return d.name
}

// Version returns the parsed SemanticVersion representation.
func (d *Dependency) Version() *SemanticVersion {
	if d == nil {
		return nil
	}
	return d.version
}

// DeclaredVersion returns the raw unparsed declared version string.
func (d *Dependency) DeclaredVersion() string {
	if d == nil {
		return ""
	}
	return d.declaredVersion
}

// Ecosystem returns the package manager / language ecosystem.
func (d *Dependency) Ecosystem() Ecosystem {
	if d == nil {
		return EcosystemUnknown
	}
	return d.ecosystem
}

// Type returns the dependency classification scope.
func (d *Dependency) Type() DependencyType {
	if d == nil {
		return DependencyUnknown
	}
	return d.depType
}

// IsDirect reports whether the dependency is directly declared in a manifest.
func (d *Dependency) IsDirect() bool {
	if d == nil {
		return false
	}
	return d.isDirect
}

// IsIndirect reports whether the dependency is transitive / indirect.
func (d *Dependency) IsIndirect() bool {
	if d == nil {
		return false
	}
	return d.isIndirect
}

// IsInternal reports whether the dependency is located inside the repository.
func (d *Dependency) IsInternal() bool {
	if d == nil {
		return false
	}
	return d.isInternal
}

// IsExternal reports whether the dependency is a third-party / external package.
func (d *Dependency) IsExternal() bool {
	if d == nil {
		return false
	}
	return d.isExternal
}

// SourceManifest returns the repository-relative path to the manifest declaring this dependency.
func (d *Dependency) SourceManifest() string {
	if d == nil {
		return ""
	}
	return d.sourceManifest
}

// ModulePath returns the repository-relative directory path of the declaring module.
func (d *Dependency) ModulePath() string {
	if d == nil {
		return ""
	}
	return d.modulePath
}

// License returns the license information associated with this dependency.
func (d *Dependency) License() *LicenseInfo {
	if d == nil {
		return nil
	}
	return d.license
}

// Health returns the maintenance health information associated with this dependency.
func (d *Dependency) Health() *HealthInfo {
	if d == nil {
		return nil
	}
	return d.health
}

// String returns a human-readable representation of the Dependency.
func (d *Dependency) String() string {
	if d == nil {
		return ""
	}
	ver := d.declaredVersion
	if ver == "" {
		ver = "latest"
	}
	return fmt.Sprintf("Dependency<%s@%s, eco=%s, direct=%t, internal=%t>", d.name, ver, d.ecosystem, d.isDirect, d.isInternal)
}

// InternalImport represents an internal package-to-package source import.
type InternalImport struct {
	sourcePkg  string
	targetPkg  string
	sourceFile string
	languageID string
}

// NewInternalImport creates an immutable InternalImport record.
func NewInternalImport(sourcePkg, targetPkg, sourceFile, langID string) *InternalImport {
	return &InternalImport{
		sourcePkg:  filepath.ToSlash(filepath.Clean(sourcePkg)),
		targetPkg:  filepath.ToSlash(filepath.Clean(targetPkg)),
		sourceFile: filepath.ToSlash(filepath.Clean(sourceFile)),
		languageID: strings.ToLower(strings.TrimSpace(langID)),
	}
}

// SourcePackage returns the declaring package relative path.
func (ii *InternalImport) SourcePackage() string {
	if ii == nil {
		return ""
	}
	return ii.sourcePkg
}

// TargetPackage returns the imported package relative path or module name.
func (ii *InternalImport) TargetPackage() string {
	if ii == nil {
		return ""
	}
	return ii.targetPkg
}

// SourceFile returns the relative path of the source file containing the import.
func (ii *InternalImport) SourceFile() string {
	if ii == nil {
		return ""
	}
	return ii.sourceFile
}

// LanguageID returns the language of the source file.
func (ii *InternalImport) LanguageID() string {
	if ii == nil {
		return ""
	}
	return ii.languageID
}

// GraphNode represents a node in the dependency graph.
type GraphNode struct {
	id         string
	name       string
	isInternal bool
	ecosystem  Ecosystem
}

// NewGraphNode creates an immutable GraphNode record.
func NewGraphNode(id, name string, isInternal bool, eco Ecosystem) *GraphNode {
	return &GraphNode{
		id:         strings.TrimSpace(id),
		name:       strings.TrimSpace(name),
		isInternal: isInternal,
		ecosystem:  eco,
	}
}

// ID returns the unique deterministic node identifier.
func (gn *GraphNode) ID() string {
	if gn == nil {
		return ""
	}
	return gn.id
}

// Name returns the display name of the node.
func (gn *GraphNode) Name() string {
	if gn == nil {
		return ""
	}
	return gn.name
}

// IsInternal reports whether the node represents an internal repository component.
func (gn *GraphNode) IsInternal() bool {
	if gn == nil {
		return false
	}
	return gn.isInternal
}

// Ecosystem returns the node ecosystem.
func (gn *GraphNode) Ecosystem() Ecosystem {
	if gn == nil {
		return EcosystemUnknown
	}
	return gn.ecosystem
}

// GraphEdge represents a directed relationship between two dependency graph nodes.
type GraphEdge struct {
	sourceID string
	targetID string
	relType  DependencyType
}

// NewGraphEdge creates an immutable GraphEdge record.
func NewGraphEdge(sourceID, targetID string, relType DependencyType) *GraphEdge {
	return &GraphEdge{
		sourceID: strings.TrimSpace(sourceID),
		targetID: strings.TrimSpace(targetID),
		relType:  relType,
	}
}

// SourceID returns the identifier of the dependent node (A in A -> B).
func (ge *GraphEdge) SourceID() string {
	if ge == nil {
		return ""
	}
	return ge.sourceID
}

// TargetID returns the identifier of the dependency node (B in A -> B).
func (ge *GraphEdge) TargetID() string {
	if ge == nil {
		return ""
	}
	return ge.targetID
}

// RelationshipType returns the type of relationship (direct, indirect, internal).
func (ge *GraphEdge) RelationshipType() DependencyType {
	if ge == nil {
		return DependencyUnknown
	}
	return ge.relType
}

// DependencyGraph represents the consolidated directed graph of dependencies.
type DependencyGraph struct {
	nodes []*GraphNode
	edges []*GraphEdge
}

// NewDependencyGraph constructs an immutable DependencyGraph with deterministic sorting.
func NewDependencyGraph(nodes []*GraphNode, edges []*GraphEdge) *DependencyGraph {
	nodeList := make([]*GraphNode, len(nodes))
	copy(nodeList, nodes)
	sort.Slice(nodeList, func(i, j int) bool {
		return nodeList[i].id < nodeList[j].id
	})

	edgeList := make([]*GraphEdge, len(edges))
	copy(edgeList, edges)
	sort.Slice(edgeList, func(i, j int) bool {
		if edgeList[i].sourceID != edgeList[j].sourceID {
			return edgeList[i].sourceID < edgeList[j].sourceID
		}
		return edgeList[i].targetID < edgeList[j].targetID
	})

	return &DependencyGraph{
		nodes: nodeList,
		edges: edgeList,
	}
}

// Nodes returns a defensive copy of all graph nodes in deterministic sorted order.
func (dg *DependencyGraph) Nodes() []*GraphNode {
	if dg == nil || len(dg.nodes) == 0 {
		return nil
	}
	cloned := make([]*GraphNode, len(dg.nodes))
	copy(cloned, dg.nodes)
	return cloned
}

// Edges returns a defensive copy of all directed graph edges in deterministic sorted order.
func (dg *DependencyGraph) Edges() []*GraphEdge {
	if dg == nil || len(dg.edges) == 0 {
		return nil
	}
	cloned := make([]*GraphEdge, len(dg.edges))
	copy(cloned, dg.edges)
	return cloned
}

// NodeCount returns the total number of nodes in the graph.
func (dg *DependencyGraph) NodeCount() int {
	if dg == nil {
		return 0
	}
	return len(dg.nodes)
}

// EdgeCount returns the total number of directed edges in the graph.
func (dg *DependencyGraph) EdgeCount() int {
	if dg == nil {
		return 0
	}
	return len(dg.edges)
}

// DetectCycles finds all circular dependency paths in the graph deterministically.
func (dg *DependencyGraph) DetectCycles() [][]string {
	if dg == nil || len(dg.edges) == 0 {
		return nil
	}

	adj := make(map[string][]string)
	for _, edge := range dg.edges {
		adj[edge.sourceID] = append(adj[edge.sourceID], edge.targetID)
	}
	for k := range adj {
		sort.Strings(adj[k])
	}

	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var path []string

	var dfs func(u string)
	dfs = func(u string) {
		visited[u] = true
		recStack[u] = true
		path = append(path, u)

		for _, v := range adj[u] {
			if !visited[v] {
				dfs(v)
			} else if recStack[v] {
				// Found cycle: slice path from v to current u, plus v to close cycle
				var cycle []string
				cycleStart := -1
				for idx, node := range path {
					if node == v {
						cycleStart = idx
						break
					}
				}
				if cycleStart != -1 {
					cycle = append(cycle, path[cycleStart:]...)
					cycle = append(cycle, v)
					cycles = append(cycles, cycle)
				}
			}
		}

		path = path[:len(path)-1]
		recStack[u] = false
	}

	// Iterate nodes in deterministic order
	for _, node := range dg.nodes {
		if !visited[node.id] {
			dfs(node.id)
		}
	}

	// Sort cycles deterministically
	sort.Slice(cycles, func(i, j int) bool {
		c1 := strings.Join(cycles[i], "->")
		c2 := strings.Join(cycles[j], "->")
		return c1 < c2
	})

	return cycles
}

// DetectOrphans identifies packages/nodes with zero inbound and zero outbound edges.
func (dg *DependencyGraph) DetectOrphans() []string {
	if dg == nil || len(dg.nodes) == 0 {
		return nil
	}

	inDegree := make(map[string]int)
	outDegree := make(map[string]int)

	for _, edge := range dg.edges {
		outDegree[edge.sourceID]++
		inDegree[edge.targetID]++
	}

	var orphans []string
	for _, node := range dg.nodes {
		if inDegree[node.id] == 0 && outDegree[node.id] == 0 {
			orphans = append(orphans, node.id)
		}
	}

	sort.Strings(orphans)
	return orphans
}

// CalculateMaxDepth computes the maximum dependency depth in the graph using BFS/DFS with cycle protection.
func (dg *DependencyGraph) CalculateMaxDepth() int {
	if dg == nil || len(dg.nodes) == 0 || len(dg.edges) == 0 {
		return 0
	}

	adj := make(map[string][]string)
	inDegree := make(map[string]int)
	for _, node := range dg.nodes {
		inDegree[node.id] = 0
	}
	for _, edge := range dg.edges {
		adj[edge.sourceID] = append(adj[edge.sourceID], edge.targetID)
		inDegree[edge.targetID]++
	}

	// Find root nodes (in-degree == 0)
	var roots []string
	for _, node := range dg.nodes {
		if inDegree[node.id] == 0 {
			roots = append(roots, node.id)
		}
	}
	if len(roots) == 0 && len(dg.nodes) > 0 {
		// Graph is all cycles or connected; pick first node
		roots = []string{dg.nodes[0].id}
	}

	maxDepth := 0
	for _, root := range roots {
		depthMap := make(map[string]int)
		depthMap[root] = 0
		queue := []string{root}

		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			currDepth := depthMap[curr]
			if currDepth > maxDepth {
				maxDepth = currDepth
			}

			for _, neighbor := range adj[curr] {
				if _, seen := depthMap[neighbor]; !seen {
					depthMap[neighbor] = currDepth + 1
					queue = append(queue, neighbor)
				}
			}
		}
	}

	return maxDepth
}

// DependencyInventory represents the organized inventory of dependencies.
type DependencyInventory struct {
	directDependencies   []*Dependency
	indirectDependencies []*Dependency
	internalDependencies []*Dependency
	externalDependencies []*Dependency
	allDependencies      []*Dependency
}

// NewDependencyInventory constructs an immutable DependencyInventory with deterministic sorting.
func NewDependencyInventory(deps []*Dependency) *DependencyInventory {
	allList := make([]*Dependency, len(deps))
	copy(allList, deps)

	sort.Slice(allList, func(i, j int) bool {
		if allList[i].name != allList[j].name {
			return allList[i].name < allList[j].name
		}
		if allList[i].modulePath != allList[j].modulePath {
			return allList[i].modulePath < allList[j].modulePath
		}
		return allList[i].declaredVersion < allList[j].declaredVersion
	})

	var directs, indirects, internals, externals []*Dependency
	for _, d := range allList {
		if d.isDirect {
			directs = append(directs, d)
		}
		if d.isIndirect {
			indirects = append(indirects, d)
		}
		if d.isInternal {
			internals = append(internals, d)
		}
		if d.isExternal {
			externals = append(externals, d)
		}
	}

	return &DependencyInventory{
		directDependencies:   directs,
		indirectDependencies: indirects,
		internalDependencies: internals,
		externalDependencies: externals,
		allDependencies:      allList,
	}
}

// DirectDependencies returns a defensive copy of directly declared dependencies.
func (di *DependencyInventory) DirectDependencies() []*Dependency {
	if di == nil || len(di.directDependencies) == 0 {
		return nil
	}
	cloned := make([]*Dependency, len(di.directDependencies))
	copy(cloned, di.directDependencies)
	return cloned
}

// IndirectDependencies returns a defensive copy of indirect dependencies.
func (di *DependencyInventory) IndirectDependencies() []*Dependency {
	if di == nil || len(di.indirectDependencies) == 0 {
		return nil
	}
	cloned := make([]*Dependency, len(di.indirectDependencies))
	copy(cloned, di.indirectDependencies)
	return cloned
}

// InternalDependencies returns a defensive copy of internal dependencies.
func (di *DependencyInventory) InternalDependencies() []*Dependency {
	if di == nil || len(di.internalDependencies) == 0 {
		return nil
	}
	cloned := make([]*Dependency, len(di.internalDependencies))
	copy(cloned, di.internalDependencies)
	return cloned
}

// ExternalDependencies returns a defensive copy of external dependencies.
func (di *DependencyInventory) ExternalDependencies() []*Dependency {
	if di == nil || len(di.externalDependencies) == 0 {
		return nil
	}
	cloned := make([]*Dependency, len(di.externalDependencies))
	copy(cloned, di.externalDependencies)
	return cloned
}

// AllDependencies returns a defensive copy of all dependencies in deterministic order.
func (di *DependencyInventory) AllDependencies() []*Dependency {
	if di == nil || len(di.allDependencies) == 0 {
		return nil
	}
	cloned := make([]*Dependency, len(di.allDependencies))
	copy(cloned, di.allDependencies)
	return cloned
}

// TotalCount returns the total number of dependencies.
func (di *DependencyInventory) TotalCount() int {
	if di == nil {
		return 0
	}
	return len(di.allDependencies)
}

// LicenseInventory represents an organized inventory of dependency licenses.
type LicenseInventory struct {
	licenses []*LicenseInfo
}

// NewLicenseInventory constructs an immutable LicenseInventory.
func NewLicenseInventory(licenses []*LicenseInfo) *LicenseInventory {
	licList := make([]*LicenseInfo, len(licenses))
	copy(licList, licenses)

	sort.Slice(licList, func(i, j int) bool {
		if licList[i].licenseType != licList[j].licenseType {
			return licList[i].licenseType < licList[j].licenseType
		}
		return licList[i].source < licList[j].source
	})

	return &LicenseInventory{licenses: licList}
}

// Licenses returns a defensive copy of all discovered licenses in deterministic order.
func (li *LicenseInventory) Licenses() []*LicenseInfo {
	if li == nil || len(li.licenses) == 0 {
		return nil
	}
	cloned := make([]*LicenseInfo, len(li.licenses))
	copy(cloned, li.licenses)
	return cloned
}

// Count returns the count of license records.
func (li *LicenseInventory) Count() int {
	if li == nil {
		return 0
	}
	return len(li.licenses)
}
