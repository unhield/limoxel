package semantic

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
)

// SemanticRepository represents the repository-level semantic container.
type SemanticRepository struct {
	id              string
	name            string
	root            string
	packages        []*SemanticPackage
	totalSymbols    int
	totalTypes      int
	totalInterfaces int
	totalFunctions  int
	totalVariables  int
	analyzedAt      time.Time
}

// NewSemanticRepository creates an immutable SemanticRepository.
func NewSemanticRepository(
	name, root string,
	pkgs []*SemanticPackage,
	totalSyms, totalTypes, totalIfaces, totalFuncs, totalVars int,
	analyzedAt time.Time,
) *SemanticRepository {
	cleanRoot := filepath.ToSlash(filepath.Clean(strings.TrimSpace(root)))
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		cleanName = filepath.Base(cleanRoot)
	}

	pkgList := make([]*SemanticPackage, len(pkgs))
	copy(pkgList, pkgs)
	sort.Slice(pkgList, func(i, j int) bool {
		return pkgList[i].Path() < pkgList[j].Path()
	})

	return &SemanticRepository{
		id:              "repo:" + cleanRoot,
		name:            cleanName,
		root:            cleanRoot,
		packages:        pkgList,
		totalSymbols:    totalSyms,
		totalTypes:      totalTypes,
		totalInterfaces: totalIfaces,
		totalFunctions:  totalFuncs,
		totalVariables:  totalVars,
		analyzedAt:      analyzedAt,
	}
}

func (r *SemanticRepository) ID() string            { return r.id }
func (r *SemanticRepository) Name() string          { return r.name }
func (r *SemanticRepository) Root() string          { return r.root }
func (r *SemanticRepository) TotalSymbols() int     { return r.totalSymbols }
func (r *SemanticRepository) TotalTypes() int       { return r.totalTypes }
func (r *SemanticRepository) TotalInterfaces() int  { return r.totalInterfaces }
func (r *SemanticRepository) TotalFunctions() int   { return r.totalFunctions }
func (r *SemanticRepository) TotalVariables() int   { return r.totalVariables }
func (r *SemanticRepository) AnalyzedAt() time.Time { return r.analyzedAt }

func (r *SemanticRepository) Packages() []*SemanticPackage {
	if r == nil || r.packages == nil {
		return nil
	}
	res := make([]*SemanticPackage, len(r.packages))
	copy(res, r.packages)
	return res
}

// SemanticPackage represents package-level semantic context.
type SemanticPackage struct {
	id           string
	name         string
	path         string
	symbols      []*SemanticSymbol
	types        []*SemanticType
	interfaces   []*SemanticInterface
	functions    []*SemanticFunction
	variables    []*SemanticVariable
	dependencies []string
}

// NewSemanticPackage constructs an immutable SemanticPackage.
func NewSemanticPackage(
	name, path string,
	symbols []*SemanticSymbol,
	types []*SemanticType,
	interfaces []*SemanticInterface,
	functions []*SemanticFunction,
	variables []*SemanticVariable,
	deps []string,
) *SemanticPackage {
	cleanPath := filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	cleanName := strings.TrimSpace(name)

	syms := make([]*SemanticSymbol, len(symbols))
	copy(syms, symbols)
	sort.Slice(syms, func(i, j int) bool {
		return syms[i].ID() < syms[j].ID()
	})

	tList := make([]*SemanticType, len(types))
	copy(tList, types)
	sort.Slice(tList, func(i, j int) bool {
		return tList[i].ID() < tList[j].ID()
	})

	iList := make([]*SemanticInterface, len(interfaces))
	copy(iList, interfaces)
	sort.Slice(iList, func(i, j int) bool {
		return iList[i].ID() < iList[j].ID()
	})

	fList := make([]*SemanticFunction, len(functions))
	copy(fList, functions)
	sort.Slice(fList, func(i, j int) bool {
		return fList[i].ID() < fList[j].ID()
	})

	vList := make([]*SemanticVariable, len(variables))
	copy(vList, variables)
	sort.Slice(vList, func(i, j int) bool {
		return vList[i].ID() < vList[j].ID()
	})

	dList := make([]string, len(deps))
	copy(dList, deps)
	sort.Strings(dList)

	return &SemanticPackage{
		id:           "pkg:" + cleanPath,
		name:         cleanName,
		path:         cleanPath,
		symbols:      syms,
		types:        tList,
		interfaces:   iList,
		functions:    fList,
		variables:    vList,
		dependencies: dList,
	}
}

func (p *SemanticPackage) ID() string   { return p.id }
func (p *SemanticPackage) Name() string { return p.name }
func (p *SemanticPackage) Path() string { return p.path }

func (p *SemanticPackage) Symbols() []*SemanticSymbol {
	if p == nil || p.symbols == nil {
		return nil
	}
	res := make([]*SemanticSymbol, len(p.symbols))
	copy(res, p.symbols)
	return res
}

func (p *SemanticPackage) Types() []*SemanticType {
	if p == nil || p.types == nil {
		return nil
	}
	res := make([]*SemanticType, len(p.types))
	copy(res, p.types)
	return res
}

func (p *SemanticPackage) Interfaces() []*SemanticInterface {
	if p == nil || p.interfaces == nil {
		return nil
	}
	res := make([]*SemanticInterface, len(p.interfaces))
	copy(res, p.interfaces)
	return res
}

func (p *SemanticPackage) Functions() []*SemanticFunction {
	if p == nil || p.functions == nil {
		return nil
	}
	res := make([]*SemanticFunction, len(p.functions))
	copy(res, p.functions)
	return res
}

func (p *SemanticPackage) Variables() []*SemanticVariable {
	if p == nil || p.variables == nil {
		return nil
	}
	res := make([]*SemanticVariable, len(p.variables))
	copy(res, p.variables)
	return res
}

func (p *SemanticPackage) Dependencies() []string {
	if p == nil || p.dependencies == nil {
		return nil
	}
	res := make([]string, len(p.dependencies))
	copy(res, p.dependencies)
	return res
}

// SemanticSymbol represents the semantic identity and engineering context of a symbol.
type SemanticSymbol struct {
	id          string
	name        string
	kind        symbol.SymbolKind
	packagePath string
	filePath    string
	line        int
	isExported  bool
	visibility  VisibilityKind
	ownership   string
	scopeID     string
	typeID      string
	signature   string
	doc         string
	references  []string
	calls       []string
	calledBy    []string
}

// NewSemanticSymbol constructs an immutable SemanticSymbol.
func NewSemanticSymbol(
	id, name string,
	kind symbol.SymbolKind,
	packagePath, filePath string,
	line int,
	isExported bool,
	visibility VisibilityKind,
	ownership, scopeID, typeID, signature, doc string,
	refs, calls, calledBy []string,
) *SemanticSymbol {
	refList := make([]string, len(refs))
	copy(refList, refs)
	sort.Strings(refList)

	callList := make([]string, len(calls))
	copy(callList, calls)
	sort.Strings(callList)

	calledByList := make([]string, len(calledBy))
	copy(calledByList, calledBy)
	sort.Strings(calledByList)

	return &SemanticSymbol{
		id:          strings.TrimSpace(id),
		name:        strings.TrimSpace(name),
		kind:        kind,
		packagePath: filepath.ToSlash(filepath.Clean(packagePath)),
		filePath:    filepath.ToSlash(filepath.Clean(filePath)),
		line:        line,
		isExported:  isExported,
		visibility:  visibility,
		ownership:   strings.TrimSpace(ownership),
		scopeID:     strings.TrimSpace(scopeID),
		typeID:      strings.TrimSpace(typeID),
		signature:   strings.TrimSpace(signature),
		doc:         strings.TrimSpace(doc),
		references:  refList,
		calls:       callList,
		calledBy:    calledByList,
	}
}

func (s *SemanticSymbol) ID() string                 { return s.id }
func (s *SemanticSymbol) Name() string               { return s.name }
func (s *SemanticSymbol) Kind() symbol.SymbolKind    { return s.kind }
func (s *SemanticSymbol) PackagePath() string        { return s.packagePath }
func (s *SemanticSymbol) FilePath() string           { return s.filePath }
func (s *SemanticSymbol) Line() int                  { return s.line }
func (s *SemanticSymbol) IsExported() bool           { return s.isExported }
func (s *SemanticSymbol) Visibility() VisibilityKind { return s.visibility }
func (s *SemanticSymbol) Ownership() string          { return s.ownership }
func (s *SemanticSymbol) ScopeID() string            { return s.scopeID }
func (s *SemanticSymbol) TypeID() string             { return s.typeID }
func (s *SemanticSymbol) Signature() string          { return s.signature }
func (s *SemanticSymbol) Doc() string                { return s.doc }

func (s *SemanticSymbol) References() []string {
	if s == nil || s.references == nil {
		return nil
	}
	res := make([]string, len(s.references))
	copy(res, s.references)
	return res
}

func (s *SemanticSymbol) Calls() []string {
	if s == nil || s.calls == nil {
		return nil
	}
	res := make([]string, len(s.calls))
	copy(res, s.calls)
	return res
}

func (s *SemanticSymbol) CalledBy() []string {
	if s == nil || s.calledBy == nil {
		return nil
	}
	res := make([]string, len(s.calledBy))
	copy(res, s.calledBy)
	return res
}

// SemanticType represents the semantic classification, relationships, and members of a type.
type SemanticType struct {
	id                    string
	name                  string
	kind                  TypeKind
	packagePath           string
	filePath              string
	underlyingType        string
	isAlias               bool
	aliasTarget           string
	isExported            bool
	fields                []*SemanticVariable
	methods               []*SemanticFunction
	embeddedTypes         []string
	implementedInterfaces []string
	generics              *SemanticGeneric
	resolutionState       ResolutionState
}

// NewSemanticType creates an immutable SemanticType.
func NewSemanticType(
	id, name string,
	kind TypeKind,
	packagePath, filePath, underlyingType string,
	isAlias bool,
	aliasTarget string,
	isExported bool,
	fields []*SemanticVariable,
	methods []*SemanticFunction,
	embeddedTypes, implementedIfaces []string,
	generics *SemanticGeneric,
	state ResolutionState,
) *SemanticType {
	fList := make([]*SemanticVariable, len(fields))
	copy(fList, fields)
	sort.Slice(fList, func(i, j int) bool {
		return fList[i].ID() < fList[j].ID()
	})

	mList := make([]*SemanticFunction, len(methods))
	copy(mList, methods)
	sort.Slice(mList, func(i, j int) bool {
		return mList[i].ID() < mList[j].ID()
	})

	embedList := make([]string, len(embeddedTypes))
	copy(embedList, embeddedTypes)
	sort.Strings(embedList)

	ifaceList := make([]string, len(implementedIfaces))
	copy(ifaceList, implementedIfaces)
	sort.Strings(ifaceList)

	return &SemanticType{
		id:                    strings.TrimSpace(id),
		name:                  strings.TrimSpace(name),
		kind:                  kind,
		packagePath:           filepath.ToSlash(filepath.Clean(packagePath)),
		filePath:              filepath.ToSlash(filepath.Clean(filePath)),
		underlyingType:        strings.TrimSpace(underlyingType),
		isAlias:               isAlias,
		aliasTarget:           strings.TrimSpace(aliasTarget),
		isExported:            isExported,
		fields:                fList,
		methods:               mList,
		embeddedTypes:         embedList,
		implementedInterfaces: ifaceList,
		generics:              generics,
		resolutionState:       state,
	}
}

func (t *SemanticType) ID() string                       { return t.id }
func (t *SemanticType) Name() string                     { return t.name }
func (t *SemanticType) Kind() TypeKind                   { return t.kind }
func (t *SemanticType) PackagePath() string              { return t.packagePath }
func (t *SemanticType) FilePath() string                 { return t.filePath }
func (t *SemanticType) UnderlyingType() string           { return t.underlyingType }
func (t *SemanticType) IsAlias() bool                    { return t.isAlias }
func (t *SemanticType) AliasTarget() string              { return t.aliasTarget }
func (t *SemanticType) IsExported() bool                 { return t.isExported }
func (t *SemanticType) Generics() *SemanticGeneric       { return t.generics }
func (t *SemanticType) ResolutionState() ResolutionState { return t.resolutionState }

func (t *SemanticType) Fields() []*SemanticVariable {
	if t == nil || t.fields == nil {
		return nil
	}
	res := make([]*SemanticVariable, len(t.fields))
	copy(res, t.fields)
	return res
}

func (t *SemanticType) Methods() []*SemanticFunction {
	if t == nil || t.methods == nil {
		return nil
	}
	res := make([]*SemanticFunction, len(t.methods))
	copy(res, t.methods)
	return res
}

func (t *SemanticType) EmbeddedTypes() []string {
	if t == nil || t.embeddedTypes == nil {
		return nil
	}
	res := make([]string, len(t.embeddedTypes))
	copy(res, t.embeddedTypes)
	return res
}

func (t *SemanticType) ImplementedInterfaces() []string {
	if t == nil || t.implementedInterfaces == nil {
		return nil
	}
	res := make([]string, len(t.implementedInterfaces))
	copy(res, t.implementedInterfaces)
	return res
}

// SemanticInterface represents an interface contract and its implementors.
type SemanticInterface struct {
	id                 string
	name               string
	packagePath        string
	filePath           string
	methods            []*SemanticFunction
	embeddedInterfaces []string
	implementors       []string
	isExported         bool
}

// NewSemanticInterface creates an immutable SemanticInterface.
func NewSemanticInterface(
	id, name, packagePath, filePath string,
	methods []*SemanticFunction,
	embeddedIfaces, implementors []string,
	isExported bool,
) *SemanticInterface {
	mList := make([]*SemanticFunction, len(methods))
	copy(mList, methods)
	sort.Slice(mList, func(i, j int) bool {
		return mList[i].ID() < mList[j].ID()
	})

	embedList := make([]string, len(embeddedIfaces))
	copy(embedList, embeddedIfaces)
	sort.Strings(embedList)

	implList := make([]string, len(implementors))
	copy(implList, implementors)
	sort.Strings(implList)

	return &SemanticInterface{
		id:                 strings.TrimSpace(id),
		name:               strings.TrimSpace(name),
		packagePath:        filepath.ToSlash(filepath.Clean(packagePath)),
		filePath:           filepath.ToSlash(filepath.Clean(filePath)),
		methods:            mList,
		embeddedInterfaces: embedList,
		implementors:       implList,
		isExported:         isExported,
	}
}

func (i *SemanticInterface) ID() string          { return i.id }
func (i *SemanticInterface) Name() string        { return i.name }
func (i *SemanticInterface) PackagePath() string { return i.packagePath }
func (i *SemanticInterface) FilePath() string    { return i.filePath }
func (i *SemanticInterface) IsExported() bool    { return i.isExported }

func (i *SemanticInterface) Methods() []*SemanticFunction {
	if i == nil || i.methods == nil {
		return nil
	}
	res := make([]*SemanticFunction, len(i.methods))
	copy(res, i.methods)
	return res
}

func (i *SemanticInterface) EmbeddedInterfaces() []string {
	if i == nil || i.embeddedInterfaces == nil {
		return nil
	}
	res := make([]string, len(i.embeddedInterfaces))
	copy(res, i.embeddedInterfaces)
	return res
}

func (i *SemanticInterface) Implementors() []string {
	if i == nil || i.implementors == nil {
		return nil
	}
	res := make([]string, len(i.implementors))
	copy(res, i.implementors)
	return res
}

// SemanticFunction represents a function or method in its semantic context.
type SemanticFunction struct {
	id                string
	name              string
	packagePath       string
	filePath          string
	receiverType      string
	isPointerReceiver bool
	parameters        []*SemanticVariable
	returnTypes       []string
	isExported        bool
	visibility        VisibilityKind
	signature         string
	scopeID           string
	calls             []string
	calledBy          []string
	generics          *SemanticGeneric
}

// NewSemanticFunction creates an immutable SemanticFunction.
func NewSemanticFunction(
	id, name, packagePath, filePath, receiverType string,
	isPointerReceiver bool,
	params []*SemanticVariable,
	returnTypes []string,
	isExported bool,
	visibility VisibilityKind,
	signature, scopeID string,
	calls, calledBy []string,
	generics *SemanticGeneric,
) *SemanticFunction {
	paramList := make([]*SemanticVariable, len(params))
	copy(paramList, params)
	sort.Slice(paramList, func(i, j int) bool {
		return paramList[i].ID() < paramList[j].ID()
	})

	retList := make([]string, len(returnTypes))
	copy(retList, returnTypes)

	callList := make([]string, len(calls))
	copy(callList, calls)
	sort.Strings(callList)

	calledByList := make([]string, len(calledBy))
	copy(calledByList, calledBy)
	sort.Strings(calledByList)

	return &SemanticFunction{
		id:                strings.TrimSpace(id),
		name:              strings.TrimSpace(name),
		packagePath:       filepath.ToSlash(filepath.Clean(packagePath)),
		filePath:          filepath.ToSlash(filepath.Clean(filePath)),
		receiverType:      strings.TrimSpace(receiverType),
		isPointerReceiver: isPointerReceiver,
		parameters:        paramList,
		returnTypes:       retList,
		isExported:        isExported,
		visibility:        visibility,
		signature:         strings.TrimSpace(signature),
		scopeID:           strings.TrimSpace(scopeID),
		calls:             callList,
		calledBy:          calledByList,
		generics:          generics,
	}
}

func (f *SemanticFunction) ID() string                 { return f.id }
func (f *SemanticFunction) Name() string               { return f.name }
func (f *SemanticFunction) PackagePath() string        { return f.packagePath }
func (f *SemanticFunction) FilePath() string           { return f.filePath }
func (f *SemanticFunction) ReceiverType() string       { return f.receiverType }
func (f *SemanticFunction) IsPointerReceiver() bool    { return f.isPointerReceiver }
func (f *SemanticFunction) IsExported() bool           { return f.isExported }
func (f *SemanticFunction) Visibility() VisibilityKind { return f.visibility }
func (f *SemanticFunction) Signature() string          { return f.signature }
func (f *SemanticFunction) ScopeID() string            { return f.scopeID }
func (f *SemanticFunction) Generics() *SemanticGeneric { return f.generics }

func (f *SemanticFunction) Parameters() []*SemanticVariable {
	if f == nil || f.parameters == nil {
		return nil
	}
	res := make([]*SemanticVariable, len(f.parameters))
	copy(res, f.parameters)
	return res
}

func (f *SemanticFunction) ReturnTypes() []string {
	if f == nil || f.returnTypes == nil {
		return nil
	}
	res := make([]string, len(f.returnTypes))
	copy(res, f.returnTypes)
	return res
}

func (f *SemanticFunction) Calls() []string {
	if f == nil || f.calls == nil {
		return nil
	}
	res := make([]string, len(f.calls))
	copy(res, f.calls)
	return res
}

func (f *SemanticFunction) CalledBy() []string {
	if f == nil || f.calledBy == nil {
		return nil
	}
	res := make([]string, len(f.calledBy))
	copy(res, f.calledBy)
	return res
}

// SemanticVariable represents a variable, parameter, or field in its semantic context.
type SemanticVariable struct {
	id             string
	name           string
	packagePath    string
	filePath       string
	scopeKind      ScopeKind
	scopeID        string
	typeExpression string
	typeID         string
	isExported     bool
	visibility     VisibilityKind
	line           int
}

// NewSemanticVariable creates an immutable SemanticVariable.
func NewSemanticVariable(
	id, name, packagePath, filePath string,
	scopeKind ScopeKind,
	scopeID, typeExpr, typeID string,
	isExported bool,
	visibility VisibilityKind,
	line int,
) *SemanticVariable {
	return &SemanticVariable{
		id:             strings.TrimSpace(id),
		name:           strings.TrimSpace(name),
		packagePath:    filepath.ToSlash(filepath.Clean(packagePath)),
		filePath:       filepath.ToSlash(filepath.Clean(filePath)),
		scopeKind:      scopeKind,
		scopeID:        strings.TrimSpace(scopeID),
		typeExpression: strings.TrimSpace(typeExpr),
		typeID:         strings.TrimSpace(typeID),
		isExported:     isExported,
		visibility:     visibility,
		line:           line,
	}
}

func (v *SemanticVariable) ID() string                 { return v.id }
func (v *SemanticVariable) Name() string               { return v.name }
func (v *SemanticVariable) PackagePath() string        { return v.packagePath }
func (v *SemanticVariable) FilePath() string           { return v.filePath }
func (v *SemanticVariable) ScopeKind() ScopeKind       { return v.scopeKind }
func (v *SemanticVariable) ScopeID() string            { return v.scopeID }
func (v *SemanticVariable) TypeExpression() string     { return v.typeExpression }
func (v *SemanticVariable) TypeID() string             { return v.typeID }
func (v *SemanticVariable) IsExported() bool           { return v.isExported }
func (v *SemanticVariable) Visibility() VisibilityKind { return v.visibility }
func (v *SemanticVariable) Line() int                  { return v.line }

// SemanticGeneric represents generic type parameter declarations, constraints, and instantiations.
type SemanticGeneric struct {
	id              string
	targetSymbolID  string
	typeParameters  []string
	constraints     map[string]string
	typeArguments   map[string]string
	resolutionState ResolutionState
}

// NewSemanticGeneric creates an immutable SemanticGeneric.
func NewSemanticGeneric(
	id, targetSymbolID string,
	typeParams []string,
	constraints, typeArgs map[string]string,
	state ResolutionState,
) *SemanticGeneric {
	tpList := make([]string, len(typeParams))
	copy(tpList, typeParams)
	sort.Strings(tpList)

	cMap := make(map[string]string, len(constraints))
	for k, v := range constraints {
		cMap[k] = v
	}

	aMap := make(map[string]string, len(typeArgs))
	for k, v := range typeArgs {
		aMap[k] = v
	}

	return &SemanticGeneric{
		id:              strings.TrimSpace(id),
		targetSymbolID:  strings.TrimSpace(targetSymbolID),
		typeParameters:  tpList,
		constraints:     cMap,
		typeArguments:   aMap,
		resolutionState: state,
	}
}

func (g *SemanticGeneric) ID() string                       { return g.id }
func (g *SemanticGeneric) TargetSymbolID() string           { return g.targetSymbolID }
func (g *SemanticGeneric) ResolutionState() ResolutionState { return g.resolutionState }

func (g *SemanticGeneric) TypeParameters() []string {
	if g == nil || g.typeParameters == nil {
		return nil
	}
	res := make([]string, len(g.typeParameters))
	copy(res, g.typeParameters)
	return res
}

func (g *SemanticGeneric) Constraints() map[string]string {
	if g == nil || g.constraints == nil {
		return nil
	}
	res := make(map[string]string, len(g.constraints))
	for k, v := range g.constraints {
		res[k] = v
	}
	return res
}

func (g *SemanticGeneric) TypeArguments() map[string]string {
	if g == nil || g.typeArguments == nil {
		return nil
	}
	res := make(map[string]string, len(g.typeArguments))
	for k, v := range g.typeArguments {
		res[k] = v
	}
	return res
}

// SemanticRelationship represents a derived engineering connection between semantic entities.
type SemanticRelationship struct {
	id         string
	kind       SemanticRelationKind
	sourceID   string
	targetID   string
	evidence   string
	provenance string
	metadata   map[string]string
}

// NewSemanticRelationship creates an immutable SemanticRelationship.
func NewSemanticRelationship(
	id string,
	kind SemanticRelationKind,
	sourceID, targetID, evidence, provenance string,
	metadata map[string]string,
) *SemanticRelationship {
	meta := make(map[string]string, len(metadata))
	for k, v := range metadata {
		meta[k] = v
	}

	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		cleanID = fmt.Sprintf("semrel:%s->%s:%s", sourceID, targetID, kind)
	}

	return &SemanticRelationship{
		id:         cleanID,
		kind:       kind,
		sourceID:   strings.TrimSpace(sourceID),
		targetID:   strings.TrimSpace(targetID),
		evidence:   strings.TrimSpace(evidence),
		provenance: strings.TrimSpace(provenance),
		metadata:   meta,
	}
}

func (r *SemanticRelationship) ID() string                 { return r.id }
func (r *SemanticRelationship) Kind() SemanticRelationKind { return r.kind }
func (r *SemanticRelationship) SourceID() string           { return r.sourceID }
func (r *SemanticRelationship) TargetID() string           { return r.targetID }
func (r *SemanticRelationship) Evidence() string           { return r.evidence }
func (r *SemanticRelationship) Provenance() string         { return r.provenance }

func (r *SemanticRelationship) Metadata() map[string]string {
	if r == nil || r.metadata == nil {
		return nil
	}
	res := make(map[string]string, len(r.metadata))
	for k, v := range r.metadata {
		res[k] = v
	}
	return res
}

// SemanticModel represents the complete immutable snapshot of repository semantic intelligence.
type SemanticModel struct {
	repository       *SemanticRepository
	symbols          map[string]*SemanticSymbol
	types            map[string]*SemanticType
	interfaces       map[string]*SemanticInterface
	functions        map[string]*SemanticFunction
	variables        map[string]*SemanticVariable
	generics         map[string]*SemanticGeneric
	relationships    []*SemanticRelationship
	scopes           map[string]*SemanticScope
	validationReport *ValidationReport
	analyzedAt       time.Time
}

// NewSemanticModel constructs an immutable SemanticModel.
func NewSemanticModel(
	repo *SemanticRepository,
	syms map[string]*SemanticSymbol,
	types map[string]*SemanticType,
	ifaces map[string]*SemanticInterface,
	funcs map[string]*SemanticFunction,
	vars map[string]*SemanticVariable,
	generics map[string]*SemanticGeneric,
	rels []*SemanticRelationship,
	scopes map[string]*SemanticScope,
	report *ValidationReport,
	analyzedAt time.Time,
) *SemanticModel {
	symMap := make(map[string]*SemanticSymbol, len(syms))
	for k, v := range syms {
		symMap[k] = v
	}

	typeMap := make(map[string]*SemanticType, len(types))
	for k, v := range types {
		typeMap[k] = v
	}

	ifaceMap := make(map[string]*SemanticInterface, len(ifaces))
	for k, v := range ifaces {
		ifaceMap[k] = v
	}

	funcMap := make(map[string]*SemanticFunction, len(funcs))
	for k, v := range funcs {
		funcMap[k] = v
	}

	varMap := make(map[string]*SemanticVariable, len(vars))
	for k, v := range vars {
		varMap[k] = v
	}

	genMap := make(map[string]*SemanticGeneric, len(generics))
	for k, v := range generics {
		genMap[k] = v
	}

	relList := make([]*SemanticRelationship, len(rels))
	copy(relList, rels)
	sort.Slice(relList, func(i, j int) bool {
		return relList[i].ID() < relList[j].ID()
	})

	scopeMap := make(map[string]*SemanticScope, len(scopes))
	for k, v := range scopes {
		scopeMap[k] = v
	}

	return &SemanticModel{
		repository:       repo,
		symbols:          symMap,
		types:            typeMap,
		interfaces:       ifaceMap,
		functions:        funcMap,
		variables:        varMap,
		generics:         genMap,
		relationships:    relList,
		scopes:           scopeMap,
		validationReport: report,
		analyzedAt:       analyzedAt,
	}
}

func (m *SemanticModel) Repository() *SemanticRepository     { return m.repository }
func (m *SemanticModel) ValidationReport() *ValidationReport { return m.validationReport }
func (m *SemanticModel) AnalyzedAt() time.Time               { return m.analyzedAt }

func (m *SemanticModel) SymbolByID(id string) *SemanticSymbol {
	if m == nil || m.symbols == nil {
		return nil
	}
	return m.symbols[strings.TrimSpace(id)]
}

func (m *SemanticModel) TypeByID(id string) *SemanticType {
	if m == nil || m.types == nil {
		return nil
	}
	return m.types[strings.TrimSpace(id)]
}

func (m *SemanticModel) InterfaceByID(id string) *SemanticInterface {
	if m == nil || m.interfaces == nil {
		return nil
	}
	return m.interfaces[strings.TrimSpace(id)]
}

func (m *SemanticModel) FunctionByID(id string) *SemanticFunction {
	if m == nil || m.functions == nil {
		return nil
	}
	return m.functions[strings.TrimSpace(id)]
}

func (m *SemanticModel) VariableByID(id string) *SemanticVariable {
	if m == nil || m.variables == nil {
		return nil
	}
	return m.variables[strings.TrimSpace(id)]
}

func (m *SemanticModel) ScopeByID(id string) *SemanticScope {
	if m == nil || m.scopes == nil {
		return nil
	}
	return m.scopes[strings.TrimSpace(id)]
}

func (m *SemanticModel) AllSymbols() []*SemanticSymbol {
	if m == nil || m.symbols == nil {
		return nil
	}
	res := make([]*SemanticSymbol, 0, len(m.symbols))
	for _, s := range m.symbols {
		res = append(res, s)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].ID() < res[j].ID()
	})
	return res
}

func (m *SemanticModel) AllTypes() []*SemanticType {
	if m == nil || m.types == nil {
		return nil
	}
	res := make([]*SemanticType, 0, len(m.types))
	for _, t := range m.types {
		res = append(res, t)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].ID() < res[j].ID()
	})
	return res
}

func (m *SemanticModel) AllInterfaces() []*SemanticInterface {
	if m == nil || m.interfaces == nil {
		return nil
	}
	res := make([]*SemanticInterface, 0, len(m.interfaces))
	for _, iface := range m.interfaces {
		res = append(res, iface)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].ID() < res[j].ID()
	})
	return res
}

func (m *SemanticModel) AllFunctions() []*SemanticFunction {
	if m == nil || m.functions == nil {
		return nil
	}
	res := make([]*SemanticFunction, 0, len(m.functions))
	for _, f := range m.functions {
		res = append(res, f)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].ID() < res[j].ID()
	})
	return res
}

func (m *SemanticModel) AllVariables() []*SemanticVariable {
	if m == nil || m.variables == nil {
		return nil
	}
	res := make([]*SemanticVariable, 0, len(m.variables))
	for _, v := range m.variables {
		res = append(res, v)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].ID() < res[j].ID()
	})
	return res
}

func (m *SemanticModel) AllRelationships() []*SemanticRelationship {
	if m == nil || m.relationships == nil {
		return nil
	}
	res := make([]*SemanticRelationship, len(m.relationships))
	copy(res, m.relationships)
	return res
}
