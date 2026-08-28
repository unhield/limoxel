package navigation

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
)

// NavigationTarget represents an immutable, resolved navigation destination.
type NavigationTarget struct {
	id             string
	symbolID       string
	name           string
	kind           string
	filePath       string
	packagePath    string
	modulePath     string
	repositoryPath string
	position       *symbol.SourcePosition
	state          NavigationState
	navKind        NavigationKind
	provenance     string
}

// NewNavigationTarget constructs an immutable NavigationTarget.
func NewNavigationTarget(
	id, symbolID, name, kind string,
	filePath, packagePath, modulePath, repoPath string,
	pos *symbol.SourcePosition,
	state NavigationState,
	navKind NavigationKind,
	provenance string,
) *NavigationTarget {
	cleanFile := filepath.ToSlash(filepath.Clean(filePath))
	cleanPkg := filepath.ToSlash(filepath.Clean(packagePath))
	cleanMod := filepath.ToSlash(filepath.Clean(modulePath))
	cleanRepo := filepath.ToSlash(filepath.Clean(repoPath))

	return &NavigationTarget{
		id:             strings.TrimSpace(id),
		symbolID:       strings.TrimSpace(symbolID),
		name:           strings.TrimSpace(name),
		kind:           strings.TrimSpace(kind),
		filePath:       cleanFile,
		packagePath:    cleanPkg,
		modulePath:     cleanMod,
		repositoryPath: cleanRepo,
		position:       pos,
		state:          state,
		navKind:        navKind,
		provenance:     strings.TrimSpace(provenance),
	}
}

func (t *NavigationTarget) ID() string                       { return t.id }
func (t *NavigationTarget) SymbolID() string                 { return t.symbolID }
func (t *NavigationTarget) Name() string                     { return t.name }
func (t *NavigationTarget) Kind() string                     { return t.kind }
func (t *NavigationTarget) FilePath() string                 { return t.filePath }
func (t *NavigationTarget) PackagePath() string              { return t.packagePath }
func (t *NavigationTarget) ModulePath() string               { return t.modulePath }
func (t *NavigationTarget) RepositoryPath() string           { return t.repositoryPath }
func (t *NavigationTarget) Position() *symbol.SourcePosition { return t.position }
func (t *NavigationTarget) State() NavigationState           { return t.state }
func (t *NavigationTarget) NavKind() NavigationKind          { return t.navKind }
func (t *NavigationTarget) Provenance() string               { return t.provenance }

// DefinitionResult contains the primary resolved target or candidate matches if ambiguous.
type DefinitionResult struct {
	target     *NavigationTarget
	candidates []*NavigationTarget
	state      NavigationState
	provenance string
}

// NewDefinitionResult constructs an immutable DefinitionResult.
func NewDefinitionResult(target *NavigationTarget, candidates []*NavigationTarget, state NavigationState, provenance string) *DefinitionResult {
	cList := make([]*NavigationTarget, len(candidates))
	copy(cList, candidates)
	sort.Slice(cList, func(i, j int) bool {
		return cList[i].ID() < cList[j].ID()
	})

	return &DefinitionResult{
		target:     target,
		candidates: cList,
		state:      state,
		provenance: strings.TrimSpace(provenance),
	}
}

func (r *DefinitionResult) Target() *NavigationTarget { return r.target }
func (r *DefinitionResult) State() NavigationState    { return r.state }
func (r *DefinitionResult) Provenance() string        { return r.provenance }
func (r *DefinitionResult) Candidates() []*NavigationTarget {
	if r == nil || r.candidates == nil {
		return nil
	}
	res := make([]*NavigationTarget, len(r.candidates))
	copy(res, r.candidates)
	return res
}

// ReferenceResult encapsulates all resolved reference destinations for a target symbol.
type ReferenceResult struct {
	targetSymbolID string
	references     []*NavigationTarget
	totalCount     int
}

// NewReferenceResult constructs an immutable ReferenceResult.
func NewReferenceResult(targetSymbolID string, references []*NavigationTarget) *ReferenceResult {
	rList := make([]*NavigationTarget, len(references))
	copy(rList, references)
	sort.Slice(rList, func(i, j int) bool {
		if rList[i].FilePath() == rList[j].FilePath() {
			if rList[i].Position() != nil && rList[j].Position() != nil {
				return rList[i].Position().Line() < rList[j].Position().Line()
			}
			return rList[i].ID() < rList[j].ID()
		}
		return rList[i].FilePath() < rList[j].FilePath()
	})

	return &ReferenceResult{
		targetSymbolID: strings.TrimSpace(targetSymbolID),
		references:     rList,
		totalCount:     len(rList),
	}
}

func (r *ReferenceResult) TargetSymbolID() string { return r.targetSymbolID }
func (r *ReferenceResult) TotalCount() int        { return r.totalCount }
func (r *ReferenceResult) References() []*NavigationTarget {
	if r == nil || r.references == nil {
		return nil
	}
	res := make([]*NavigationTarget, len(r.references))
	copy(res, r.references)
	return res
}

// UsageItem describes an individual contextual usage of an engineering entity.
type UsageItem struct {
	id             string
	sourceSymbolID string
	targetSymbolID string
	kind           UsageKind
	filePath       string
	position       *symbol.SourcePosition
	contextLine    string
	provenance     string
}

// NewUsageItem constructs an immutable UsageItem.
func NewUsageItem(
	sourceSymbolID, targetSymbolID string,
	kind UsageKind,
	filePath string,
	pos *symbol.SourcePosition,
	contextLine, provenance string,
) *UsageItem {
	cleanSrc := strings.TrimSpace(sourceSymbolID)
	cleanTgt := strings.TrimSpace(targetSymbolID)
	cleanFile := filepath.ToSlash(filepath.Clean(filePath))

	line := 0
	if pos != nil {
		line = pos.Line()
	}

	return &UsageItem{
		id:             fmt.Sprintf("usage:%s->%s:%s:%s:%d", cleanSrc, cleanTgt, string(kind), cleanFile, line),
		sourceSymbolID: cleanSrc,
		targetSymbolID: cleanTgt,
		kind:           kind,
		filePath:       cleanFile,
		position:       pos,
		contextLine:    strings.TrimSpace(contextLine),
		provenance:     strings.TrimSpace(provenance),
	}
}

func (u *UsageItem) ID() string                       { return u.id }
func (u *UsageItem) SourceSymbolID() string           { return u.sourceSymbolID }
func (u *UsageItem) TargetSymbolID() string           { return u.targetSymbolID }
func (u *UsageItem) Kind() UsageKind                  { return u.kind }
func (u *UsageItem) FilePath() string                 { return u.filePath }
func (u *UsageItem) Position() *symbol.SourcePosition { return u.position }
func (u *UsageItem) ContextLine() string              { return u.contextLine }
func (u *UsageItem) Provenance() string               { return u.provenance }

// ReverseRelationship represents inbound connections from other entities to a target.
type ReverseRelationship struct {
	id             string
	targetEntityID string
	sourceEntities []string
	relKind        RelationshipKind
	provenance     string
}

// NewReverseRelationship constructs an immutable ReverseRelationship.
func NewReverseRelationship(targetEntityID string, sourceEntities []string, relKind RelationshipKind, provenance string) *ReverseRelationship {
	cleanTgt := strings.TrimSpace(targetEntityID)

	sMap := make(map[string]bool)
	var sList []string
	for _, s := range sourceEntities {
		cleanS := strings.TrimSpace(s)
		if cleanS != "" && !sMap[cleanS] {
			sMap[cleanS] = true
			sList = append(sList, cleanS)
		}
	}
	sort.Strings(sList)

	return &ReverseRelationship{
		id:             "revrel:" + cleanTgt + ":" + string(relKind),
		targetEntityID: cleanTgt,
		sourceEntities: sList,
		relKind:        relKind,
		provenance:     strings.TrimSpace(provenance),
	}
}

func (r *ReverseRelationship) ID() string                { return r.id }
func (r *ReverseRelationship) TargetEntityID() string    { return r.targetEntityID }
func (r *ReverseRelationship) RelKind() RelationshipKind { return r.relKind }
func (r *ReverseRelationship) Provenance() string        { return r.provenance }
func (r *ReverseRelationship) SourceEntities() []string {
	if r == nil || r.sourceEntities == nil {
		return nil
	}
	res := make([]string, len(r.sourceEntities))
	copy(res, r.sourceEntities)
	return res
}

// DependencyNavigationItem describes an edge in the dependency traversal graph.
type DependencyNavigationItem struct {
	id         string
	sourceID   string
	targetID   string
	direction  string // "outbound" or "inbound"
	kind       string
	provenance string
}

// NewDependencyNavigationItem constructs an immutable DependencyNavigationItem.
func NewDependencyNavigationItem(sourceID, targetID, direction, kind, provenance string) *DependencyNavigationItem {
	cleanSrc := strings.TrimSpace(sourceID)
	cleanTgt := strings.TrimSpace(targetID)

	return &DependencyNavigationItem{
		id:         "depnav:" + cleanSrc + "->" + cleanTgt + ":" + direction,
		sourceID:   cleanSrc,
		targetID:   cleanTgt,
		direction:  strings.TrimSpace(direction),
		kind:       strings.TrimSpace(kind),
		provenance: strings.TrimSpace(provenance),
	}
}

func (d *DependencyNavigationItem) ID() string         { return d.id }
func (d *DependencyNavigationItem) SourceID() string   { return d.sourceID }
func (d *DependencyNavigationItem) TargetID() string   { return d.targetID }
func (d *DependencyNavigationItem) Direction() string  { return d.direction }
func (d *DependencyNavigationItem) Kind() string       { return d.kind }
func (d *DependencyNavigationItem) Provenance() string { return d.provenance }

// RelationshipItem describes a general semantic relationship between two entities.
type RelationshipItem struct {
	id         string
	sourceID   string
	targetID   string
	relKind    RelationshipKind
	direction  string
	provenance string
}

// NewRelationshipItem constructs an immutable RelationshipItem.
func NewRelationshipItem(sourceID, targetID string, relKind RelationshipKind, direction, provenance string) *RelationshipItem {
	cleanSrc := strings.TrimSpace(sourceID)
	cleanTgt := strings.TrimSpace(targetID)

	return &RelationshipItem{
		id:         "rel:" + cleanSrc + "->" + cleanTgt + ":" + string(relKind),
		sourceID:   cleanSrc,
		targetID:   cleanTgt,
		relKind:    relKind,
		direction:  strings.TrimSpace(direction),
		provenance: strings.TrimSpace(provenance),
	}
}

func (r *RelationshipItem) ID() string                { return r.id }
func (r *RelationshipItem) SourceID() string          { return r.sourceID }
func (r *RelationshipItem) TargetID() string          { return r.targetID }
func (r *RelationshipItem) RelKind() RelationshipKind { return r.relKind }
func (r *RelationshipItem) Direction() string         { return r.direction }
func (r *RelationshipItem) Provenance() string        { return r.provenance }

// SymbolHierarchyNode models a hierarchical node in the symbol tree.
type SymbolHierarchyNode struct {
	id          string
	symbolID    string
	name        string
	kind        string
	filePath    string
	packagePath string
	parentID    string
	children    []*SymbolHierarchyNode
}

// NewSymbolHierarchyNode constructs an immutable SymbolHierarchyNode.
func NewSymbolHierarchyNode(symbolID, name, kind, filePath, packagePath, parentID string, children []*SymbolHierarchyNode) *SymbolHierarchyNode {
	cleanSym := strings.TrimSpace(symbolID)
	cList := make([]*SymbolHierarchyNode, len(children))
	copy(cList, children)
	sort.Slice(cList, func(i, j int) bool {
		return cList[i].SymbolID() < cList[j].SymbolID()
	})

	return &SymbolHierarchyNode{
		id:          "symnode:" + cleanSym,
		symbolID:    cleanSym,
		name:        strings.TrimSpace(name),
		kind:        strings.TrimSpace(kind),
		filePath:    filepath.ToSlash(filepath.Clean(filePath)),
		packagePath: filepath.ToSlash(filepath.Clean(packagePath)),
		parentID:    strings.TrimSpace(parentID),
		children:    cList,
	}
}

func (n *SymbolHierarchyNode) ID() string          { return n.id }
func (n *SymbolHierarchyNode) SymbolID() string    { return n.symbolID }
func (n *SymbolHierarchyNode) Name() string        { return n.name }
func (n *SymbolHierarchyNode) Kind() string        { return n.kind }
func (n *SymbolHierarchyNode) FilePath() string    { return n.filePath }
func (n *SymbolHierarchyNode) PackagePath() string { return n.packagePath }
func (n *SymbolHierarchyNode) ParentID() string    { return n.parentID }
func (n *SymbolHierarchyNode) Children() []*SymbolHierarchyNode {
	if n == nil || n.children == nil {
		return nil
	}
	res := make([]*SymbolHierarchyNode, len(n.children))
	copy(res, n.children)
	return res
}

// InterfaceHierarchyNode models an interface contract and its implementors.
type InterfaceHierarchyNode struct {
	id                 string
	interfaceID        string
	name               string
	packagePath        string
	embeddedInterfaces []string
	implementors       []string
}

// NewInterfaceHierarchyNode constructs an immutable InterfaceHierarchyNode.
func NewInterfaceHierarchyNode(interfaceID, name, packagePath string, embeddedIfaces, implementors []string) *InterfaceHierarchyNode {
	cleanID := strings.TrimSpace(interfaceID)

	eMap := make(map[string]bool)
	var eList []string
	for _, e := range embeddedIfaces {
		cleanE := strings.TrimSpace(e)
		if cleanE != "" && !eMap[cleanE] {
			eMap[cleanE] = true
			eList = append(eList, cleanE)
		}
	}
	sort.Strings(eList)

	iMap := make(map[string]bool)
	var iList []string
	for _, imp := range implementors {
		cleanI := strings.TrimSpace(imp)
		if cleanI != "" && !iMap[cleanI] {
			iMap[cleanI] = true
			iList = append(iList, cleanI)
		}
	}
	sort.Strings(iList)

	return &InterfaceHierarchyNode{
		id:                 "ifacenode:" + cleanID,
		interfaceID:        cleanID,
		name:               strings.TrimSpace(name),
		packagePath:        filepath.ToSlash(filepath.Clean(packagePath)),
		embeddedInterfaces: eList,
		implementors:       iList,
	}
}

func (i *InterfaceHierarchyNode) ID() string          { return i.id }
func (i *InterfaceHierarchyNode) InterfaceID() string { return i.interfaceID }
func (i *InterfaceHierarchyNode) Name() string        { return i.name }
func (i *InterfaceHierarchyNode) PackagePath() string { return i.packagePath }
func (i *InterfaceHierarchyNode) EmbeddedInterfaces() []string {
	if i == nil || i.embeddedInterfaces == nil {
		return nil
	}
	res := make([]string, len(i.embeddedInterfaces))
	copy(res, i.embeddedInterfaces)
	return res
}
func (i *InterfaceHierarchyNode) Implementors() []string {
	if i == nil || i.implementors == nil {
		return nil
	}
	res := make([]string, len(i.implementors))
	copy(res, i.implementors)
	return res
}

// TypeHierarchyNode represents type inheritance, embedding, and alias relationships.
type TypeHierarchyNode struct {
	id              string
	typeID          string
	name            string
	packagePath     string
	baseType        string
	aliasedType     string
	embeddedTypes   []string
	implementations []string
}

// NewTypeHierarchyNode constructs an immutable TypeHierarchyNode.
func NewTypeHierarchyNode(typeID, name, packagePath, baseType, aliasedType string, embeddedTypes, implementations []string) *TypeHierarchyNode {
	cleanID := strings.TrimSpace(typeID)

	eMap := make(map[string]bool)
	var eList []string
	for _, e := range embeddedTypes {
		cleanE := strings.TrimSpace(e)
		if cleanE != "" && !eMap[cleanE] {
			eMap[cleanE] = true
			eList = append(eList, cleanE)
		}
	}
	sort.Strings(eList)

	iMap := make(map[string]bool)
	var iList []string
	for _, imp := range implementations {
		cleanI := strings.TrimSpace(imp)
		if cleanI != "" && !iMap[cleanI] {
			iMap[cleanI] = true
			iList = append(iList, cleanI)
		}
	}
	sort.Strings(iList)

	return &TypeHierarchyNode{
		id:              "typenode:" + cleanID,
		typeID:          cleanID,
		name:            strings.TrimSpace(name),
		packagePath:     filepath.ToSlash(filepath.Clean(packagePath)),
		baseType:        strings.TrimSpace(baseType),
		aliasedType:     strings.TrimSpace(aliasedType),
		embeddedTypes:   eList,
		implementations: iList,
	}
}

func (t *TypeHierarchyNode) ID() string          { return t.id }
func (t *TypeHierarchyNode) TypeID() string      { return t.typeID }
func (t *TypeHierarchyNode) Name() string        { return t.name }
func (t *TypeHierarchyNode) PackagePath() string { return t.packagePath }
func (t *TypeHierarchyNode) BaseType() string    { return t.baseType }
func (t *TypeHierarchyNode) AliasedType() string { return t.aliasedType }
func (t *TypeHierarchyNode) EmbeddedTypes() []string {
	if t == nil || t.embeddedTypes == nil {
		return nil
	}
	res := make([]string, len(t.embeddedTypes))
	copy(res, t.embeddedTypes)
	return res
}
func (t *TypeHierarchyNode) Implementations() []string {
	if t == nil || t.implementations == nil {
		return nil
	}
	res := make([]string, len(t.implementations))
	copy(res, t.implementations)
	return res
}

// PackageHierarchyNode represents the package containment tree within modules and repositories.
type PackageHierarchyNode struct {
	id              string
	packagePath     string
	modulePath      string
	repositoryRoot  string
	containedFiles  []string
	childPackages   []string
	exportedSymbols []string
}

// NewPackageHierarchyNode constructs an immutable PackageHierarchyNode.
func NewPackageHierarchyNode(packagePath, modulePath, repoRoot string, files, childPkgs, exportedSyms []string) *PackageHierarchyNode {
	cleanPkg := filepath.ToSlash(filepath.Clean(packagePath))

	fMap := make(map[string]bool)
	var fList []string
	for _, f := range files {
		cleanF := filepath.ToSlash(filepath.Clean(f))
		if cleanF != "" && !fMap[cleanF] {
			fMap[cleanF] = true
			fList = append(fList, cleanF)
		}
	}
	sort.Strings(fList)

	cMap := make(map[string]bool)
	var cList []string
	for _, c := range childPkgs {
		cleanC := filepath.ToSlash(filepath.Clean(c))
		if cleanC != "" && !cMap[cleanC] {
			cMap[cleanC] = true
			cList = append(cList, cleanC)
		}
	}
	sort.Strings(cList)

	sMap := make(map[string]bool)
	var sList []string
	for _, s := range exportedSyms {
		cleanS := strings.TrimSpace(s)
		if cleanS != "" && !sMap[cleanS] {
			sMap[cleanS] = true
			sList = append(sList, cleanS)
		}
	}
	sort.Strings(sList)

	return &PackageHierarchyNode{
		id:              "pkghierarchy:" + cleanPkg,
		packagePath:     cleanPkg,
		modulePath:      filepath.ToSlash(filepath.Clean(modulePath)),
		repositoryRoot:  filepath.ToSlash(filepath.Clean(repoRoot)),
		containedFiles:  fList,
		childPackages:   cList,
		exportedSymbols: sList,
	}
}

func (p *PackageHierarchyNode) ID() string             { return p.id }
func (p *PackageHierarchyNode) PackagePath() string    { return p.packagePath }
func (p *PackageHierarchyNode) ModulePath() string     { return p.modulePath }
func (p *PackageHierarchyNode) RepositoryRoot() string { return p.repositoryRoot }
func (p *PackageHierarchyNode) ContainedFiles() []string {
	if p == nil || p.containedFiles == nil {
		return nil
	}
	res := make([]string, len(p.containedFiles))
	copy(res, p.containedFiles)
	return res
}
func (p *PackageHierarchyNode) ChildPackages() []string {
	if p == nil || p.childPackages == nil {
		return nil
	}
	res := make([]string, len(p.childPackages))
	copy(res, p.childPackages)
	return res
}
func (p *PackageHierarchyNode) ExportedSymbols() []string {
	if p == nil || p.exportedSymbols == nil {
		return nil
	}
	res := make([]string, len(p.exportedSymbols))
	copy(res, p.exportedSymbols)
	return res
}

// CallHierarchyNode describes caller/callee connections for a function/method.
type CallHierarchyNode struct {
	id              string
	symbolID        string
	name            string
	packagePath     string
	filePath        string
	incomingCallers []string
	outgoingCallees []string
	depth           int
}

// NewCallHierarchyNode constructs an immutable CallHierarchyNode.
func NewCallHierarchyNode(symbolID, name, packagePath, filePath string, callers, callees []string, depth int) *CallHierarchyNode {
	cleanSym := strings.TrimSpace(symbolID)

	cInMap := make(map[string]bool)
	var inList []string
	for _, c := range callers {
		cleanC := strings.TrimSpace(c)
		if cleanC != "" && !cInMap[cleanC] {
			cInMap[cleanC] = true
			inList = append(inList, cleanC)
		}
	}
	sort.Strings(inList)

	cOutMap := make(map[string]bool)
	var outList []string
	for _, c := range callees {
		cleanC := strings.TrimSpace(c)
		if cleanC != "" && !cOutMap[cleanC] {
			cOutMap[cleanC] = true
			outList = append(outList, cleanC)
		}
	}
	sort.Strings(outList)

	return &CallHierarchyNode{
		id:              "callnode:" + cleanSym,
		symbolID:        cleanSym,
		name:            strings.TrimSpace(name),
		packagePath:     filepath.ToSlash(filepath.Clean(packagePath)),
		filePath:        filepath.ToSlash(filepath.Clean(filePath)),
		incomingCallers: inList,
		outgoingCallees: outList,
		depth:           depth,
	}
}

func (c *CallHierarchyNode) ID() string          { return c.id }
func (c *CallHierarchyNode) SymbolID() string    { return c.symbolID }
func (c *CallHierarchyNode) Name() string        { return c.name }
func (c *CallHierarchyNode) PackagePath() string { return c.packagePath }
func (c *CallHierarchyNode) FilePath() string    { return c.filePath }
func (c *CallHierarchyNode) Depth() int          { return c.depth }
func (c *CallHierarchyNode) IncomingCallers() []string {
	if c == nil || c.incomingCallers == nil {
		return nil
	}
	res := make([]string, len(c.incomingCallers))
	copy(res, c.incomingCallers)
	return res
}
func (c *CallHierarchyNode) OutgoingCallees() []string {
	if c == nil || c.outgoingCallees == nil {
		return nil
	}
	res := make([]string, len(c.outgoingCallees))
	copy(res, c.outgoingCallees)
	return res
}

// RecursivePath describes a confirmed recursive call cycle.
type RecursivePath struct {
	id           string
	cycleSymbols []string
	length       int
	isDirect     bool
}

// NewRecursivePath constructs an immutable RecursivePath.
func NewRecursivePath(cycleSymbols []string, isDirect bool) *RecursivePath {
	sList := make([]string, len(cycleSymbols))
	copy(sList, cycleSymbols)

	id := "cycle:" + strings.Join(sList, "->")

	return &RecursivePath{
		id:           id,
		cycleSymbols: sList,
		length:       len(sList),
		isDirect:     isDirect,
	}
}

func (r *RecursivePath) ID() string     { return r.id }
func (r *RecursivePath) Length() int    { return r.length }
func (r *RecursivePath) IsDirect() bool { return r.isDirect }
func (r *RecursivePath) CycleSymbols() []string {
	if r == nil || r.cycleSymbols == nil {
		return nil
	}
	res := make([]string, len(r.cycleSymbols))
	copy(res, r.cycleSymbols)
	return res
}

// DependencyChain describes a deterministic sequential path through dependencies.
type DependencyChain struct {
	id          string
	steps       []string
	totalLength int
	isCyclic    bool
}

// NewDependencyChain constructs an immutable DependencyChain.
func NewDependencyChain(steps []string, isCyclic bool) *DependencyChain {
	sList := make([]string, len(steps))
	copy(sList, steps)

	id := "depchain:" + strings.Join(sList, "->")

	return &DependencyChain{
		id:          id,
		steps:       sList,
		totalLength: len(sList),
		isCyclic:    isCyclic,
	}
}

func (d *DependencyChain) ID() string       { return d.id }
func (d *DependencyChain) TotalLength() int { return d.totalLength }
func (d *DependencyChain) IsCyclic() bool   { return d.isCyclic }
func (d *DependencyChain) Steps() []string {
	if d == nil || d.steps == nil {
		return nil
	}
	res := make([]string, len(d.steps))
	copy(res, d.steps)
	return res
}

// NavigationPath represents the sequence of entities through which a destination is reached.
type NavigationPath struct {
	id            string
	sourceID      string
	targetID      string
	steps         []string
	totalDistance int
}

// NewNavigationPath constructs an immutable NavigationPath.
func NewNavigationPath(sourceID, targetID string, steps []string) *NavigationPath {
	cleanSrc := strings.TrimSpace(sourceID)
	cleanTgt := strings.TrimSpace(targetID)

	sList := make([]string, len(steps))
	copy(sList, steps)

	return &NavigationPath{
		id:            "navpath:" + cleanSrc + "->" + cleanTgt,
		sourceID:      cleanSrc,
		targetID:      cleanTgt,
		steps:         sList,
		totalDistance: len(sList),
	}
}

func (p *NavigationPath) ID() string         { return p.id }
func (p *NavigationPath) SourceID() string   { return p.sourceID }
func (p *NavigationPath) TargetID() string   { return p.targetID }
func (p *NavigationPath) TotalDistance() int { return p.totalDistance }
func (p *NavigationPath) Steps() []string {
	if p == nil || p.steps == nil {
		return nil
	}
	res := make([]string, len(p.steps))
	copy(res, p.steps)
	return res
}

// ValidationFinding represents a diagnosed navigation issue.
type ValidationFinding struct {
	id         string
	kind       ValidationFindingType
	severity   Severity
	sourceID   string
	targetID   string
	message    string
	provenance string
}

// NewValidationFinding constructs an immutable ValidationFinding.
func NewValidationFinding(kind ValidationFindingType, severity Severity, sourceID, targetID, message, provenance string) *ValidationFinding {
	cleanSrc := strings.TrimSpace(sourceID)
	cleanTgt := strings.TrimSpace(targetID)

	return &ValidationFinding{
		id:         "valfinding:" + string(kind) + ":" + cleanSrc + "->" + cleanTgt,
		kind:       kind,
		severity:   severity,
		sourceID:   cleanSrc,
		targetID:   cleanTgt,
		message:    strings.TrimSpace(message),
		provenance: strings.TrimSpace(provenance),
	}
}

func (v *ValidationFinding) ID() string                  { return v.id }
func (v *ValidationFinding) Kind() ValidationFindingType { return v.kind }
func (v *ValidationFinding) Severity() Severity          { return v.severity }
func (v *ValidationFinding) SourceID() string            { return v.sourceID }
func (v *ValidationFinding) TargetID() string            { return v.targetID }
func (v *ValidationFinding) Message() string             { return v.message }
func (v *ValidationFinding) Provenance() string          { return v.provenance }

// NavigationValidationReport aggregates all diagnosed validation issues.
type NavigationValidationReport struct {
	findings           []*ValidationFinding
	missingTargetCount int
	brokenRefCount     int
	duplicatePathCount int
	isValid            bool
}

// NewNavigationValidationReport constructs an immutable NavigationValidationReport.
func NewNavigationValidationReport(findings []*ValidationFinding) *NavigationValidationReport {
	fList := make([]*ValidationFinding, len(findings))
	copy(fList, findings)
	sort.Slice(fList, func(i, j int) bool {
		return fList[i].ID() < fList[j].ID()
	})

	missing := 0
	broken := 0
	duplicates := 0
	hasErrors := false

	for _, f := range fList {
		if f == nil {
			continue
		}
		if f.Severity() == SeverityError {
			hasErrors = true
		}
		switch f.Kind() {
		case NavValMissingTarget:
			missing++
		case NavValBrokenReference:
			broken++
		case NavValDuplicatePath:
			duplicates++
		}
	}

	return &NavigationValidationReport{
		findings:           fList,
		missingTargetCount: missing,
		brokenRefCount:     broken,
		duplicatePathCount: duplicates,
		isValid:            !hasErrors,
	}
}

func (r *NavigationValidationReport) Findings() []*ValidationFinding {
	if r == nil || r.findings == nil {
		return nil
	}
	res := make([]*ValidationFinding, len(r.findings))
	copy(res, r.findings)
	return res
}
func (r *NavigationValidationReport) MissingTargetCount() int { return r.missingTargetCount }
func (r *NavigationValidationReport) BrokenRefCount() int     { return r.brokenRefCount }
func (r *NavigationValidationReport) DuplicatePathCount() int { return r.duplicatePathCount }
func (r *NavigationValidationReport) IsValid() bool           { return r.isValid }

// NavigationModel is the root immutable snapshot of all computed navigation structures.
type NavigationModel struct {
	definitions      map[string]*DefinitionResult
	references       map[string]*ReferenceResult
	usages           map[string][]*UsageItem
	symbolHierarchy  map[string]*SymbolHierarchyNode
	interfaceNodes   map[string]*InterfaceHierarchyNode
	typeNodes        map[string]*TypeHierarchyNode
	packageHierarchy map[string]*PackageHierarchyNode
	callNodes        map[string]*CallHierarchyNode
	validation       *NavigationValidationReport
}

// NewNavigationModel constructs an immutable NavigationModel.
func NewNavigationModel(
	defs map[string]*DefinitionResult,
	refs map[string]*ReferenceResult,
	usages map[string][]*UsageItem,
	symHier map[string]*SymbolHierarchyNode,
	ifaceNodes map[string]*InterfaceHierarchyNode,
	typeNodes map[string]*TypeHierarchyNode,
	pkgHier map[string]*PackageHierarchyNode,
	callNodes map[string]*CallHierarchyNode,
	val *NavigationValidationReport,
) *NavigationModel {
	dMap := make(map[string]*DefinitionResult, len(defs))
	for k, v := range defs {
		dMap[k] = v
	}

	rMap := make(map[string]*ReferenceResult, len(refs))
	for k, v := range refs {
		rMap[k] = v
	}

	uMap := make(map[string][]*UsageItem, len(usages))
	for k, v := range usages {
		itemList := make([]*UsageItem, len(v))
		copy(itemList, v)
		uMap[k] = itemList
	}

	shMap := make(map[string]*SymbolHierarchyNode, len(symHier))
	for k, v := range symHier {
		shMap[k] = v
	}

	iMap := make(map[string]*InterfaceHierarchyNode, len(ifaceNodes))
	for k, v := range ifaceNodes {
		iMap[k] = v
	}

	tMap := make(map[string]*TypeHierarchyNode, len(typeNodes))
	for k, v := range typeNodes {
		tMap[k] = v
	}

	pMap := make(map[string]*PackageHierarchyNode, len(pkgHier))
	for k, v := range pkgHier {
		pMap[k] = v
	}

	cMap := make(map[string]*CallHierarchyNode, len(callNodes))
	for k, v := range callNodes {
		cMap[k] = v
	}

	return &NavigationModel{
		definitions:      dMap,
		references:       rMap,
		usages:           uMap,
		symbolHierarchy:  shMap,
		interfaceNodes:   iMap,
		typeNodes:        tMap,
		packageHierarchy: pMap,
		callNodes:        cMap,
		validation:       val,
	}
}

func (m *NavigationModel) Definition(symbolID string) *DefinitionResult {
	if m == nil {
		return nil
	}
	return m.definitions[symbolID]
}

func (m *NavigationModel) Definitions() map[string]*DefinitionResult {
	if m == nil || m.definitions == nil {
		return nil
	}
	res := make(map[string]*DefinitionResult, len(m.definitions))
	for k, v := range m.definitions {
		res[k] = v
	}
	return res
}

func (m *NavigationModel) References(symbolID string) *ReferenceResult {
	if m == nil {
		return nil
	}
	return m.references[symbolID]
}

func (m *NavigationModel) Usages(symbolID string) []*UsageItem {
	if m == nil || m.usages == nil {
		return nil
	}
	items := m.usages[symbolID]
	if items == nil {
		return nil
	}
	res := make([]*UsageItem, len(items))
	copy(res, items)
	return res
}

func (m *NavigationModel) SymbolHierarchyNode(symbolID string) *SymbolHierarchyNode {
	if m == nil {
		return nil
	}
	return m.symbolHierarchy[symbolID]
}

func (m *NavigationModel) InterfaceHierarchyNode(interfaceID string) *InterfaceHierarchyNode {
	if m == nil {
		return nil
	}
	return m.interfaceNodes[interfaceID]
}

func (m *NavigationModel) TypeHierarchyNode(typeID string) *TypeHierarchyNode {
	if m == nil {
		return nil
	}
	return m.typeNodes[typeID]
}

func (m *NavigationModel) PackageHierarchyNode(packagePath string) *PackageHierarchyNode {
	if m == nil {
		return nil
	}
	return m.packageHierarchy[packagePath]
}

func (m *NavigationModel) SymbolHierarchyNodes() map[string]*SymbolHierarchyNode {
	if m == nil || m.symbolHierarchy == nil {
		return nil
	}
	res := make(map[string]*SymbolHierarchyNode, len(m.symbolHierarchy))
	for k, v := range m.symbolHierarchy {
		res[k] = v
	}
	return res
}

func (m *NavigationModel) CallHierarchyNode(symbolID string) *CallHierarchyNode {
	if m == nil {
		return nil
	}
	return m.callNodes[symbolID]
}

func (m *NavigationModel) CallHierarchyNodes() map[string]*CallHierarchyNode {
	if m == nil || m.callNodes == nil {
		return nil
	}
	res := make(map[string]*CallHierarchyNode, len(m.callNodes))
	for k, v := range m.callNodes {
		res[k] = v
	}
	return res
}

func (m *NavigationModel) ValidationReport() *NavigationValidationReport {
	if m == nil {
		return nil
	}
	return m.validation
}
