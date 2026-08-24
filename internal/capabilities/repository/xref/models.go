package xref

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
)

// Reference represents an immutable, single source-to-target cross-reference.
type Reference struct {
	id             string
	sourceSymbolID string
	targetSymbolID string
	kind           ReferenceKind
	filePath       string
	position       *symbol.SourcePosition
	state          ResolutionState
	evidence       string
}

// NewReference constructs an immutable Reference record.
func NewReference(
	sourceSymbolID string,
	targetSymbolID string,
	kind ReferenceKind,
	filePath string,
	pos *symbol.SourcePosition,
	state ResolutionState,
	evidence string,
) *Reference {
	cleanFile := filepath.ToSlash(filepath.Clean(filePath))
	cleanSrc := strings.TrimSpace(sourceSymbolID)
	cleanTgt := strings.TrimSpace(targetSymbolID)

	line := 0
	col := 0
	if pos != nil {
		line = pos.Line()
		col = pos.Column()
	}

	id := fmt.Sprintf("ref:%s:%d:%d:%s->%s:%s", cleanFile, line, col, cleanSrc, cleanTgt, kind)

	return &Reference{
		id:             id,
		sourceSymbolID: cleanSrc,
		targetSymbolID: cleanTgt,
		kind:           kind,
		filePath:       cleanFile,
		position:       pos,
		state:          state,
		evidence:       strings.TrimSpace(evidence),
	}
}

// ID returns the deterministic identifier of the reference.
func (r *Reference) ID() string {
	if r == nil {
		return ""
	}
	return r.id
}

// SourceSymbolID returns the referencing symbol ID.
func (r *Reference) SourceSymbolID() string {
	if r == nil {
		return ""
	}
	return r.sourceSymbolID
}

// TargetSymbolID returns the referenced symbol ID.
func (r *Reference) TargetSymbolID() string {
	if r == nil {
		return ""
	}
	return r.targetSymbolID
}

// Kind returns the syntactic category of the reference.
func (r *Reference) Kind() ReferenceKind {
	if r == nil {
		return RefUnknown
	}
	return r.kind
}

// FilePath returns the relative repository file path containing the reference.
func (r *Reference) FilePath() string {
	if r == nil {
		return ""
	}
	return r.filePath
}

// Position returns the source coordinate where the reference appears.
func (r *Reference) Position() *symbol.SourcePosition {
	if r == nil {
		return nil
	}
	return r.position
}

// State returns the resolution confidence state.
func (r *Reference) State() ResolutionState {
	if r == nil {
		return StateUnknown
	}
	return r.state
}

// Evidence returns the structural rationale describing the reference.
func (r *Reference) Evidence() string {
	if r == nil {
		return ""
	}
	return r.evidence
}

// ReferenceDatabase represents an immutable, queryable store of deterministic repository references.
type ReferenceDatabase struct {
	references []*Reference
	fromMap    map[string][]*Reference
	toMap      map[string][]*Reference
	fileMap    map[string][]*Reference
}

// NewReferenceDatabase constructs an immutable ReferenceDatabase with deterministic ordering.
func NewReferenceDatabase(refs []*Reference) *ReferenceDatabase {
	sortedRefs := make([]*Reference, len(refs))
	copy(sortedRefs, refs)

	sort.Slice(sortedRefs, func(i, j int) bool {
		if sortedRefs[i].filePath != sortedRefs[j].filePath {
			return sortedRefs[i].filePath < sortedRefs[j].filePath
		}
		var lineI, lineJ, colI, colJ int
		if sortedRefs[i].position != nil {
			lineI = sortedRefs[i].position.Line()
			colI = sortedRefs[i].position.Column()
		}
		if sortedRefs[j].position != nil {
			lineJ = sortedRefs[j].position.Line()
			colJ = sortedRefs[j].position.Column()
		}
		if lineI != lineJ {
			return lineI < lineJ
		}
		if colI != colJ {
			return colI < colJ
		}
		if sortedRefs[i].sourceSymbolID != sortedRefs[j].sourceSymbolID {
			return sortedRefs[i].sourceSymbolID < sortedRefs[j].sourceSymbolID
		}
		if sortedRefs[i].targetSymbolID != sortedRefs[j].targetSymbolID {
			return sortedRefs[i].targetSymbolID < sortedRefs[j].targetSymbolID
		}
		return sortedRefs[i].kind < sortedRefs[j].kind
	})

	fromMap := make(map[string][]*Reference)
	toMap := make(map[string][]*Reference)
	fileMap := make(map[string][]*Reference)

	for _, r := range sortedRefs {
		fromMap[r.sourceSymbolID] = append(fromMap[r.sourceSymbolID], r)
		toMap[r.targetSymbolID] = append(toMap[r.targetSymbolID], r)
		fileMap[r.filePath] = append(fileMap[r.filePath], r)
	}

	return &ReferenceDatabase{
		references: sortedRefs,
		fromMap:    fromMap,
		toMap:      toMap,
		fileMap:    fileMap,
	}
}

// AllReferences returns a defensive copy of all stored references.
func (db *ReferenceDatabase) AllReferences() []*Reference {
	if db == nil || len(db.references) == 0 {
		return nil
	}
	cloned := make([]*Reference, len(db.references))
	copy(cloned, db.references)
	return cloned
}

// ReferencesFrom returns all references originating from the specified symbol ID.
func (db *ReferenceDatabase) ReferencesFrom(sourceSymbolID string) []*Reference {
	if db == nil || db.fromMap == nil {
		return nil
	}
	list := db.fromMap[strings.TrimSpace(sourceSymbolID)]
	if len(list) == 0 {
		return nil
	}
	cloned := make([]*Reference, len(list))
	copy(cloned, list)
	return cloned
}

// ReferencesTo returns all references targeting the specified symbol ID.
func (db *ReferenceDatabase) ReferencesTo(targetSymbolID string) []*Reference {
	if db == nil || db.toMap == nil {
		return nil
	}
	list := db.toMap[strings.TrimSpace(targetSymbolID)]
	if len(list) == 0 {
		return nil
	}
	cloned := make([]*Reference, len(list))
	copy(cloned, list)
	return cloned
}

// ReferencesInFile returns all references occurring within the specified relative file path.
func (db *ReferenceDatabase) ReferencesInFile(relPath string) []*Reference {
	if db == nil || db.fileMap == nil {
		return nil
	}
	clean := filepath.ToSlash(filepath.Clean(relPath))
	list := db.fileMap[clean]
	if len(list) == 0 {
		return nil
	}
	cloned := make([]*Reference, len(list))
	copy(cloned, list)
	return cloned
}

// TotalCount returns the total number of references.
func (db *ReferenceDatabase) TotalCount() int {
	if db == nil {
		return 0
	}
	return len(db.references)
}

// CallEdge represents a single call relationship edge in the CallGraph.
type CallEdge struct {
	id       string
	callerID string
	calleeID string
	kind     CallKind
	filePath string
	position *symbol.SourcePosition
}

// NewCallEdge constructs an immutable CallEdge.
func NewCallEdge(
	callerID string,
	calleeID string,
	kind CallKind,
	filePath string,
	pos *symbol.SourcePosition,
) *CallEdge {
	cleanFile := filepath.ToSlash(filepath.Clean(filePath))
	cleanCaller := strings.TrimSpace(callerID)
	cleanCallee := strings.TrimSpace(calleeID)

	line := 0
	col := 0
	if pos != nil {
		line = pos.Line()
		col = pos.Column()
	}

	id := fmt.Sprintf("call:%s:%d:%d:%s->%s:%s", cleanFile, line, col, cleanCaller, cleanCallee, kind)

	return &CallEdge{
		id:       id,
		callerID: cleanCaller,
		calleeID: cleanCallee,
		kind:     kind,
		filePath: cleanFile,
		position: pos,
	}
}

// ID returns the edge identifier.
func (e *CallEdge) ID() string {
	if e == nil {
		return ""
	}
	return e.id
}

// CallerID returns the calling symbol identifier.
func (e *CallEdge) CallerID() string {
	if e == nil {
		return ""
	}
	return e.callerID
}

// CalleeID returns the invoked symbol identifier.
func (e *CallEdge) CalleeID() string {
	if e == nil {
		return ""
	}
	return e.calleeID
}

// Kind returns the call kind.
func (e *CallEdge) Kind() CallKind {
	if e == nil {
		return CallDirect
	}
	return e.kind
}

// FilePath returns the file containing the call.
func (e *CallEdge) FilePath() string {
	if e == nil {
		return ""
	}
	return e.filePath
}

// Position returns the position of the call invocation.
func (e *CallEdge) Position() *symbol.SourcePosition {
	if e == nil {
		return nil
	}
	return e.position
}

// CallGraph represents an immutable repository call graph.
type CallGraph struct {
	edges           []*CallEdge
	callersMap      map[string][]string
	calleesMap      map[string][]string
	callerEdgesMap  map[string][]*CallEdge
	calleeEdgesMap  map[string][]*CallEdge
	entryPoints     []string
	exitPoints      []string
	recursiveCycles [][]string
	deadFunctions   []string
	reachability    map[string]ReachabilityState
}

// NewCallGraph constructs an immutable CallGraph with deterministic ordering.
func NewCallGraph(
	edges []*CallEdge,
	entryPoints []string,
	exitPoints []string,
	recursiveCycles [][]string,
	deadFunctions []string,
	reachability map[string]ReachabilityState,
) *CallGraph {
	sortedEdges := make([]*CallEdge, len(edges))
	copy(sortedEdges, edges)

	sort.Slice(sortedEdges, func(i, j int) bool {
		if sortedEdges[i].callerID != sortedEdges[j].callerID {
			return sortedEdges[i].callerID < sortedEdges[j].callerID
		}
		if sortedEdges[i].calleeID != sortedEdges[j].calleeID {
			return sortedEdges[i].calleeID < sortedEdges[j].calleeID
		}
		return sortedEdges[i].id < sortedEdges[j].id
	})

	callersMap := make(map[string][]string)
	calleesMap := make(map[string][]string)
	callerEdgesMap := make(map[string][]*CallEdge)
	calleeEdgesMap := make(map[string][]*CallEdge)

	callerSet := make(map[string]map[string]bool)
	calleeSet := make(map[string]map[string]bool)

	for _, e := range sortedEdges {
		callerEdgesMap[e.calleeID] = append(callerEdgesMap[e.calleeID], e)
		calleeEdgesMap[e.callerID] = append(calleeEdgesMap[e.callerID], e)

		if callerSet[e.calleeID] == nil {
			callerSet[e.calleeID] = make(map[string]bool)
		}
		if !callerSet[e.calleeID][e.callerID] {
			callerSet[e.calleeID][e.callerID] = true
			callersMap[e.calleeID] = append(callersMap[e.calleeID], e.callerID)
		}

		if calleeSet[e.callerID] == nil {
			calleeSet[e.callerID] = make(map[string]bool)
		}
		if !calleeSet[e.callerID][e.calleeID] {
			calleeSet[e.callerID][e.calleeID] = true
			calleesMap[e.callerID] = append(calleesMap[e.callerID], e.calleeID)
		}
	}

	for k := range callersMap {
		sort.Strings(callersMap[k])
	}
	for k := range calleesMap {
		sort.Strings(calleesMap[k])
	}

	entries := make([]string, len(entryPoints))
	copy(entries, entryPoints)
	sort.Strings(entries)

	exits := make([]string, len(exitPoints))
	copy(exits, exitPoints)
	sort.Strings(exits)

	deads := make([]string, len(deadFunctions))
	copy(deads, deadFunctions)
	sort.Strings(deads)

	cycles := make([][]string, len(recursiveCycles))
	for i, c := range recursiveCycles {
		cyc := make([]string, len(c))
		copy(cyc, c)
		cycles[i] = cyc
	}
	sort.Slice(cycles, func(i, j int) bool {
		return strings.Join(cycles[i], "->") < strings.Join(cycles[j], "->")
	})

	reachMap := make(map[string]ReachabilityState, len(reachability))
	for k, v := range reachability {
		reachMap[k] = v
	}

	return &CallGraph{
		edges:           sortedEdges,
		callersMap:      callersMap,
		calleesMap:      calleesMap,
		callerEdgesMap:  callerEdgesMap,
		calleeEdgesMap:  calleeEdgesMap,
		entryPoints:     entries,
		exitPoints:      exits,
		recursiveCycles: cycles,
		deadFunctions:   deads,
		reachability:    reachMap,
	}
}

// AllEdges returns a defensive copy of all call graph edges.
func (cg *CallGraph) AllEdges() []*CallEdge {
	if cg == nil || len(cg.edges) == 0 {
		return nil
	}
	cloned := make([]*CallEdge, len(cg.edges))
	copy(cloned, cg.edges)
	return cloned
}

// Callers returns the deterministic sorted list of symbol IDs that call the given symbol ID.
func (cg *CallGraph) Callers(symbolID string) []string {
	if cg == nil || cg.callersMap == nil {
		return nil
	}
	list := cg.callersMap[strings.TrimSpace(symbolID)]
	if len(list) == 0 {
		return nil
	}
	cloned := make([]string, len(list))
	copy(cloned, list)
	return cloned
}

// Callees returns the deterministic sorted list of symbol IDs invoked by the given symbol ID.
func (cg *CallGraph) Callees(symbolID string) []string {
	if cg == nil || cg.calleesMap == nil {
		return nil
	}
	list := cg.calleesMap[strings.TrimSpace(symbolID)]
	if len(list) == 0 {
		return nil
	}
	cloned := make([]string, len(list))
	copy(cloned, list)
	return cloned
}

// CallerEdges returns the call edges targeting the given symbol ID.
func (cg *CallGraph) CallerEdges(symbolID string) []*CallEdge {
	if cg == nil || cg.callerEdgesMap == nil {
		return nil
	}
	list := cg.callerEdgesMap[strings.TrimSpace(symbolID)]
	if len(list) == 0 {
		return nil
	}
	cloned := make([]*CallEdge, len(list))
	copy(cloned, list)
	return cloned
}

// CalleeEdges returns the call edges originating from the given symbol ID.
func (cg *CallGraph) CalleeEdges(symbolID string) []*CallEdge {
	if cg == nil || cg.calleeEdgesMap == nil {
		return nil
	}
	list := cg.calleeEdgesMap[strings.TrimSpace(symbolID)]
	if len(list) == 0 {
		return nil
	}
	cloned := make([]*CallEdge, len(list))
	copy(cloned, list)
	return cloned
}

// EntryPoints returns the identified repository entry-point functions.
func (cg *CallGraph) EntryPoints() []string {
	if cg == nil || len(cg.entryPoints) == 0 {
		return nil
	}
	cloned := make([]string, len(cg.entryPoints))
	copy(cloned, cg.entryPoints)
	return cloned
}

// ExitPoints returns the identified terminal exit-point functions.
func (cg *CallGraph) ExitPoints() []string {
	if cg == nil || len(cg.exitPoints) == 0 {
		return nil
	}
	cloned := make([]string, len(cg.exitPoints))
	copy(cloned, cg.exitPoints)
	return cloned
}

// RecursiveCycles returns all detected direct and mutual recursion cycles.
func (cg *CallGraph) RecursiveCycles() [][]string {
	if cg == nil || len(cg.recursiveCycles) == 0 {
		return nil
	}
	cloned := make([][]string, len(cg.recursiveCycles))
	for i, c := range cg.recursiveCycles {
		cyc := make([]string, len(c))
		copy(cyc, c)
		cloned[i] = cyc
	}
	return cloned
}

// DeadFunctions returns functions confirmed unreachable from entry points.
func (cg *CallGraph) DeadFunctions() []string {
	if cg == nil || len(cg.deadFunctions) == 0 {
		return nil
	}
	cloned := make([]string, len(cg.deadFunctions))
	copy(cloned, cg.deadFunctions)
	return cloned
}

// Reachability returns the reachability state for a given symbol ID.
func (cg *CallGraph) Reachability(symbolID string) ReachabilityState {
	if cg == nil || cg.reachability == nil {
		return ReachabilityUnknown
	}
	st, ok := cg.reachability[strings.TrimSpace(symbolID)]
	if !ok {
		return ReachabilityUnknown
	}
	return st
}

// TotalEdges returns the count of call graph edges.
func (cg *CallGraph) TotalEdges() int {
	if cg == nil {
		return 0
	}
	return len(cg.edges)
}

// DefinitionResult represents the result of a Go-to-Definition navigation query.
type DefinitionResult struct {
	targetSymbol *symbol.Symbol
	candidates   []*symbol.Symbol
	location     *symbol.SourcePosition
	state        ResolutionState
	evidence     string
}

// NewDefinitionResult constructs an immutable DefinitionResult.
func NewDefinitionResult(
	target *symbol.Symbol,
	candidates []*symbol.Symbol,
	loc *symbol.SourcePosition,
	state ResolutionState,
	evidence string,
) *DefinitionResult {
	candClones := make([]*symbol.Symbol, len(candidates))
	copy(candClones, candidates)
	return &DefinitionResult{
		targetSymbol: target,
		candidates:   candClones,
		location:     loc,
		state:        state,
		evidence:     evidence,
	}
}

// TargetSymbol returns the resolved declaration symbol, or nil if unresolved/ambiguous.
func (dr *DefinitionResult) TargetSymbol() *symbol.Symbol {
	if dr == nil {
		return nil
	}
	return dr.targetSymbol
}

// Candidates returns all matching candidate symbols in ambiguous resolution cases.
func (dr *DefinitionResult) Candidates() []*symbol.Symbol {
	if dr == nil || len(dr.candidates) == 0 {
		return nil
	}
	cloned := make([]*symbol.Symbol, len(dr.candidates))
	copy(cloned, dr.candidates)
	return cloned
}

// Location returns the source position of the definition.
func (dr *DefinitionResult) Location() *symbol.SourcePosition {
	if dr == nil {
		return nil
	}
	return dr.location
}

// State returns the resolution confidence state.
func (dr *DefinitionResult) State() ResolutionState {
	if dr == nil {
		return StateUnknown
	}
	return dr.state
}

// Evidence returns the resolution description.
func (dr *DefinitionResult) Evidence() string {
	if dr == nil {
		return ""
	}
	return dr.evidence
}

// PackageNavResult represents deterministic package navigation information.
type PackageNavResult struct {
	packagePath string
	packageName string
	symbols     []*symbol.Symbol
	imports     []string
	importedBy  []string
}

// NewPackageNavResult constructs an immutable PackageNavResult.
func NewPackageNavResult(
	pkgPath string,
	pkgName string,
	syms []*symbol.Symbol,
	imports []string,
	importedBy []string,
) *PackageNavResult {
	sortedSyms := make([]*symbol.Symbol, len(syms))
	copy(sortedSyms, syms)
	sort.Slice(sortedSyms, func(i, j int) bool {
		return sortedSyms[i].ID() < sortedSyms[j].ID()
	})

	sortedImports := make([]string, len(imports))
	copy(sortedImports, imports)
	sort.Strings(sortedImports)

	sortedImportedBy := make([]string, len(importedBy))
	copy(sortedImportedBy, importedBy)
	sort.Strings(sortedImportedBy)

	return &PackageNavResult{
		packagePath: pkgPath,
		packageName: pkgName,
		symbols:     sortedSyms,
		imports:     sortedImports,
		importedBy:  sortedImportedBy,
	}
}

// PackagePath returns the canonical package import path.
func (pnr *PackageNavResult) PackagePath() string {
	if pnr == nil {
		return ""
	}
	return pnr.packagePath
}

// PackageName returns the package name.
func (pnr *PackageNavResult) PackageName() string {
	if pnr == nil {
		return ""
	}
	return pnr.packageName
}

// Symbols returns all symbols declared in the package.
func (pnr *PackageNavResult) Symbols() []*symbol.Symbol {
	if pnr == nil || len(pnr.symbols) == 0 {
		return nil
	}
	cloned := make([]*symbol.Symbol, len(pnr.symbols))
	copy(cloned, pnr.symbols)
	return cloned
}

// Imports returns packages imported by this package.
func (pnr *PackageNavResult) Imports() []string {
	if pnr == nil || len(pnr.imports) == 0 {
		return nil
	}
	cloned := make([]string, len(pnr.imports))
	copy(cloned, pnr.imports)
	return cloned
}

// ImportedBy returns downstream packages that import this package.
func (pnr *PackageNavResult) ImportedBy() []string {
	if pnr == nil || len(pnr.importedBy) == 0 {
		return nil
	}
	cloned := make([]string, len(pnr.importedBy))
	copy(cloned, pnr.importedBy)
	return cloned
}

// FileImpactResult models the calculated impact of modifying a specific file.
type FileImpactResult struct {
	filePath         string
	impactedFiles    []string
	impactedPackages []string
	impactedSymbols  []string
	severity         ImpactSeverity
}

// NewFileImpactResult constructs an immutable FileImpactResult.
func NewFileImpactResult(
	filePath string,
	impactedFiles []string,
	impactedPackages []string,
	impactedSymbols []string,
	sev ImpactSeverity,
) *FileImpactResult {
	cleanFiles := make([]string, len(impactedFiles))
	copy(cleanFiles, impactedFiles)
	sort.Strings(cleanFiles)

	cleanPkgs := make([]string, len(impactedPackages))
	copy(cleanPkgs, impactedPackages)
	sort.Strings(cleanPkgs)

	cleanSyms := make([]string, len(impactedSymbols))
	copy(cleanSyms, impactedSymbols)
	sort.Strings(cleanSyms)

	return &FileImpactResult{
		filePath:         filepath.ToSlash(filepath.Clean(filePath)),
		impactedFiles:    cleanFiles,
		impactedPackages: cleanPkgs,
		impactedSymbols:  cleanSyms,
		severity:         sev,
	}
}

// FilePath returns the target file path.
func (fir *FileImpactResult) FilePath() string {
	if fir == nil {
		return ""
	}
	return fir.filePath
}

// ImpactedFiles returns the affected downstream files.
func (fir *FileImpactResult) ImpactedFiles() []string {
	if fir == nil || len(fir.impactedFiles) == 0 {
		return nil
	}
	cloned := make([]string, len(fir.impactedFiles))
	copy(cloned, fir.impactedFiles)
	return cloned
}

// ImpactedPackages returns the affected downstream packages.
func (fir *FileImpactResult) ImpactedPackages() []string {
	if fir == nil || len(fir.impactedPackages) == 0 {
		return nil
	}
	cloned := make([]string, len(fir.impactedPackages))
	copy(cloned, fir.impactedPackages)
	return cloned
}

// ImpactedSymbols returns the affected downstream symbols.
func (fir *FileImpactResult) ImpactedSymbols() []string {
	if fir == nil || len(fir.impactedSymbols) == 0 {
		return nil
	}
	cloned := make([]string, len(fir.impactedSymbols))
	copy(cloned, fir.impactedSymbols)
	return cloned
}

// Severity returns the impact severity classification.
func (fir *FileImpactResult) Severity() ImpactSeverity {
	if fir == nil {
		return ImpactDirect
	}
	return fir.severity
}

// SymbolImpactResult models the calculated impact of modifying a specific symbol.
type SymbolImpactResult struct {
	symbolID                    string
	directlyImpactedSymbols     []string
	transitivelyImpactedSymbols []string
	impactedFiles               []string
	impactedPackages            []string
}

// NewSymbolImpactResult constructs an immutable SymbolImpactResult.
func NewSymbolImpactResult(
	symbolID string,
	directSyms []string,
	transitiveSyms []string,
	files []string,
	pkgs []string,
) *SymbolImpactResult {
	dSyms := make([]string, len(directSyms))
	copy(dSyms, directSyms)
	sort.Strings(dSyms)

	tSyms := make([]string, len(transitiveSyms))
	copy(tSyms, transitiveSyms)
	sort.Strings(tSyms)

	fList := make([]string, len(files))
	copy(fList, files)
	sort.Strings(fList)

	pList := make([]string, len(pkgs))
	copy(pList, pkgs)
	sort.Strings(pList)

	return &SymbolImpactResult{
		symbolID:                    symbolID,
		directlyImpactedSymbols:     dSyms,
		transitivelyImpactedSymbols: tSyms,
		impactedFiles:               fList,
		impactedPackages:            pList,
	}
}

// SymbolID returns the target symbol ID.
func (sir *SymbolImpactResult) SymbolID() string {
	if sir == nil {
		return ""
	}
	return sir.symbolID
}

// DirectlyImpactedSymbols returns directly referencing symbols and callers.
func (sir *SymbolImpactResult) DirectlyImpactedSymbols() []string {
	if sir == nil || len(sir.directlyImpactedSymbols) == 0 {
		return nil
	}
	cloned := make([]string, len(sir.directlyImpactedSymbols))
	copy(cloned, sir.directlyImpactedSymbols)
	return cloned
}

// TransitivelyImpactedSymbols returns transitively affected symbols.
func (sir *SymbolImpactResult) TransitivelyImpactedSymbols() []string {
	if sir == nil || len(sir.transitivelyImpactedSymbols) == 0 {
		return nil
	}
	cloned := make([]string, len(sir.transitivelyImpactedSymbols))
	copy(cloned, sir.transitivelyImpactedSymbols)
	return cloned
}

// ImpactedFiles returns files containing affected symbols.
func (sir *SymbolImpactResult) ImpactedFiles() []string {
	if sir == nil || len(sir.impactedFiles) == 0 {
		return nil
	}
	cloned := make([]string, len(sir.impactedFiles))
	copy(cloned, sir.impactedFiles)
	return cloned
}

// ImpactedPackages returns packages containing affected symbols.
func (sir *SymbolImpactResult) ImpactedPackages() []string {
	if sir == nil || len(sir.impactedPackages) == 0 {
		return nil
	}
	cloned := make([]string, len(sir.impactedPackages))
	copy(cloned, sir.impactedPackages)
	return cloned
}

// PackageImpactResult models the impact of package modifications.
type PackageImpactResult struct {
	packagePath        string
	downstreamPackages []string
	downstreamFiles    []string
	downstreamSymbols  []string
}

// NewPackageImpactResult constructs an immutable PackageImpactResult.
func NewPackageImpactResult(
	pkgPath string,
	downPackages []string,
	downFiles []string,
	downSymbols []string,
) *PackageImpactResult {
	dp := make([]string, len(downPackages))
	copy(dp, downPackages)
	sort.Strings(dp)

	df := make([]string, len(downFiles))
	copy(df, downFiles)
	sort.Strings(df)

	ds := make([]string, len(downSymbols))
	copy(ds, downSymbols)
	sort.Strings(ds)

	return &PackageImpactResult{
		packagePath:        pkgPath,
		downstreamPackages: dp,
		downstreamFiles:    df,
		downstreamSymbols:  ds,
	}
}

// PackagePath returns the package import path.
func (pir *PackageImpactResult) PackagePath() string {
	if pir == nil {
		return ""
	}
	return pir.packagePath
}

// DownstreamPackages returns packages that depend upon this package.
func (pir *PackageImpactResult) DownstreamPackages() []string {
	if pir == nil || len(pir.downstreamPackages) == 0 {
		return nil
	}
	cloned := make([]string, len(pir.downstreamPackages))
	copy(cloned, pir.downstreamPackages)
	return cloned
}

// DownstreamFiles returns files in downstream packages.
func (pir *PackageImpactResult) DownstreamFiles() []string {
	if pir == nil || len(pir.downstreamFiles) == 0 {
		return nil
	}
	cloned := make([]string, len(pir.downstreamFiles))
	copy(cloned, pir.downstreamFiles)
	return cloned
}

// DownstreamSymbols returns symbols in downstream packages.
func (pir *PackageImpactResult) DownstreamSymbols() []string {
	if pir == nil || len(pir.downstreamSymbols) == 0 {
		return nil
	}
	cloned := make([]string, len(pir.downstreamSymbols))
	copy(cloned, pir.downstreamSymbols)
	return cloned
}

// DependencyImpactResult models downstream impact of external dependencies.
type DependencyImpactResult struct {
	dependencyName   string
	impactedPackages []string
	impactedFiles    []string
}

// NewDependencyImpactResult constructs an immutable DependencyImpactResult.
func NewDependencyImpactResult(depName string, pkgs []string, files []string) *DependencyImpactResult {
	pList := make([]string, len(pkgs))
	copy(pList, pkgs)
	sort.Strings(pList)

	fList := make([]string, len(files))
	copy(fList, files)
	sort.Strings(fList)

	return &DependencyImpactResult{
		dependencyName:   depName,
		impactedPackages: pList,
		impactedFiles:    fList,
	}
}

// DependencyName returns the dependency module/import name.
func (dir *DependencyImpactResult) DependencyName() string {
	if dir == nil {
		return ""
	}
	return dir.dependencyName
}

// ImpactedPackages returns packages importing the dependency.
func (dir *DependencyImpactResult) ImpactedPackages() []string {
	if dir == nil || len(dir.impactedPackages) == 0 {
		return nil
	}
	cloned := make([]string, len(dir.impactedPackages))
	copy(cloned, dir.impactedPackages)
	return cloned
}

// ImpactedFiles returns files importing the dependency.
func (dir *DependencyImpactResult) ImpactedFiles() []string {
	if dir == nil || len(dir.impactedFiles) == 0 {
		return nil
	}
	cloned := make([]string, len(dir.impactedFiles))
	copy(cloned, dir.impactedFiles)
	return cloned
}

// BreakingChangeItem describes a single breaking change occurrence.
type BreakingChangeItem struct {
	symbolID           string
	category           BreakingCategory
	description        string
	affectedReferences []*Reference
}

// NewBreakingChangeItem constructs an immutable BreakingChangeItem.
func NewBreakingChangeItem(
	symbolID string,
	category BreakingCategory,
	desc string,
	affectedRefs []*Reference,
) *BreakingChangeItem {
	refs := make([]*Reference, len(affectedRefs))
	copy(refs, affectedRefs)
	return &BreakingChangeItem{
		symbolID:           symbolID,
		category:           category,
		description:        desc,
		affectedReferences: refs,
	}
}

// SymbolID returns the symbol that changed.
func (bci *BreakingChangeItem) SymbolID() string {
	if bci == nil {
		return ""
	}
	return bci.symbolID
}

// Category returns the breaking change certainty category.
func (bci *BreakingChangeItem) Category() BreakingCategory {
	if bci == nil {
		return BreakingUnknown
	}
	return bci.category
}

// Description returns the structural explanation.
func (bci *BreakingChangeItem) Description() string {
	if bci == nil {
		return ""
	}
	return bci.description
}

// AffectedReferences returns references invalidated by this breaking change.
func (bci *BreakingChangeItem) AffectedReferences() []*Reference {
	if bci == nil || len(bci.affectedReferences) == 0 {
		return nil
	}
	cloned := make([]*Reference, len(bci.affectedReferences))
	copy(cloned, bci.affectedReferences)
	return cloned
}

// BreakingChangeResult aggregates all detected breaking changes between two states.
type BreakingChangeResult struct {
	items          []*BreakingChangeItem
	confirmedCount int
	potentialCount int
	unknownCount   int
}

// NewBreakingChangeResult constructs an immutable BreakingChangeResult.
func NewBreakingChangeResult(items []*BreakingChangeItem) *BreakingChangeResult {
	sortedItems := make([]*BreakingChangeItem, len(items))
	copy(sortedItems, items)
	sort.Slice(sortedItems, func(i, j int) bool {
		return sortedItems[i].symbolID < sortedItems[j].symbolID
	})

	var confirmed, potential, unknown int
	for _, it := range sortedItems {
		switch it.category {
		case BreakingConfirmed:
			confirmed++
		case BreakingPotential:
			potential++
		case BreakingUnknown:
			unknown++
		}
	}

	return &BreakingChangeResult{
		items:          sortedItems,
		confirmedCount: confirmed,
		potentialCount: potential,
		unknownCount:   unknown,
	}
}

// Items returns a defensive copy of all breaking change items.
func (bcr *BreakingChangeResult) Items() []*BreakingChangeItem {
	if bcr == nil || len(bcr.items) == 0 {
		return nil
	}
	cloned := make([]*BreakingChangeItem, len(bcr.items))
	copy(cloned, bcr.items)
	return cloned
}

// ConfirmedCount returns the number of confirmed breaking changes.
func (bcr *BreakingChangeResult) ConfirmedCount() int {
	if bcr == nil {
		return 0
	}
	return bcr.confirmedCount
}

// PotentialCount returns the number of potential breaking changes.
func (bcr *BreakingChangeResult) PotentialCount() int {
	if bcr == nil {
		return 0
	}
	return bcr.potentialCount
}

// UnknownCount returns the number of unknown breaking changes.
func (bcr *BreakingChangeResult) UnknownCount() int {
	if bcr == nil {
		return 0
	}
	return bcr.unknownCount
}

// TotalCount returns the total number of breaking change items.
func (bcr *BreakingChangeResult) TotalCount() int {
	if bcr == nil {
		return 0
	}
	return len(bcr.items)
}

// ValidationIssue models a single relationship integrity issue.
type ValidationIssue struct {
	severity ValidationSeverity
	message  string
	sourceID string
	targetID string
	filePath string
	position *symbol.SourcePosition
}

// NewValidationIssue constructs an immutable ValidationIssue.
func NewValidationIssue(
	sev ValidationSeverity,
	msg string,
	srcID string,
	tgtID string,
	filePath string,
	pos *symbol.SourcePosition,
) *ValidationIssue {
	return &ValidationIssue{
		severity: sev,
		message:  msg,
		sourceID: srcID,
		targetID: tgtID,
		filePath: filepath.ToSlash(filepath.Clean(filePath)),
		position: pos,
	}
}

// Severity returns the issue severity classification.
func (vi *ValidationIssue) Severity() ValidationSeverity {
	if vi == nil {
		return ValidationBrokenRef
	}
	return vi.severity
}

// Message returns the issue description.
func (vi *ValidationIssue) Message() string {
	if vi == nil {
		return ""
	}
	return vi.message
}

// SourceID returns the referencing/source symbol ID if applicable.
func (vi *ValidationIssue) SourceID() string {
	if vi == nil {
		return ""
	}
	return vi.sourceID
}

// TargetID returns the referenced target symbol ID if applicable.
func (vi *ValidationIssue) TargetID() string {
	if vi == nil {
		return ""
	}
	return vi.targetID
}

// FilePath returns the file where the issue occurs.
func (vi *ValidationIssue) FilePath() string {
	if vi == nil {
		return ""
	}
	return vi.filePath
}

// Position returns the coordinates of the issue.
func (vi *ValidationIssue) Position() *symbol.SourcePosition {
	if vi == nil {
		return nil
	}
	return vi.position
}

// ValidationReport aggregates relationship validation findings.
type ValidationReport struct {
	issues             []*ValidationIssue
	brokenRefs         []*ValidationIssue
	missingSymbols     []*ValidationIssue
	duplicateSymbols   []*ValidationIssue
	invalidImports     []*ValidationIssue
	circularReferences []*ValidationIssue
}

// NewValidationReport constructs an immutable ValidationReport.
func NewValidationReport(issues []*ValidationIssue) *ValidationReport {
	sortedIssues := make([]*ValidationIssue, len(issues))
	copy(sortedIssues, issues)
	sort.Slice(sortedIssues, func(i, j int) bool {
		if sortedIssues[i].filePath != sortedIssues[j].filePath {
			return sortedIssues[i].filePath < sortedIssues[j].filePath
		}
		if sortedIssues[i].severity != sortedIssues[j].severity {
			return sortedIssues[i].severity < sortedIssues[j].severity
		}
		return sortedIssues[i].message < sortedIssues[j].message
	})

	var broken, missing, duplicates, imports, circular []*ValidationIssue
	for _, it := range sortedIssues {
		switch it.severity {
		case ValidationBrokenRef:
			broken = append(broken, it)
		case ValidationMissingSymbol:
			missing = append(missing, it)
		case ValidationDuplicateSymbol:
			duplicates = append(duplicates, it)
		case ValidationInvalidImport:
			imports = append(imports, it)
		case ValidationCircularRef:
			circular = append(circular, it)
		}
	}

	return &ValidationReport{
		issues:             sortedIssues,
		brokenRefs:         broken,
		missingSymbols:     missing,
		duplicateSymbols:   duplicates,
		invalidImports:     imports,
		circularReferences: circular,
	}
}

// Issues returns a defensive copy of all validation issues.
func (vr *ValidationReport) Issues() []*ValidationIssue {
	if vr == nil || len(vr.issues) == 0 {
		return nil
	}
	cloned := make([]*ValidationIssue, len(vr.issues))
	copy(cloned, vr.issues)
	return cloned
}

// BrokenReferences returns broken reference issues.
func (vr *ValidationReport) BrokenReferences() []*ValidationIssue {
	if vr == nil || len(vr.brokenRefs) == 0 {
		return nil
	}
	cloned := make([]*ValidationIssue, len(vr.brokenRefs))
	copy(cloned, vr.brokenRefs)
	return cloned
}

// MissingSymbols returns missing symbol issues.
func (vr *ValidationReport) MissingSymbols() []*ValidationIssue {
	if vr == nil || len(vr.missingSymbols) == 0 {
		return nil
	}
	cloned := make([]*ValidationIssue, len(vr.missingSymbols))
	copy(cloned, vr.missingSymbols)
	return cloned
}

// DuplicateSymbols returns duplicate symbol issues.
func (vr *ValidationReport) DuplicateSymbols() []*ValidationIssue {
	if vr == nil || len(vr.duplicateSymbols) == 0 {
		return nil
	}
	cloned := make([]*ValidationIssue, len(vr.duplicateSymbols))
	copy(cloned, vr.duplicateSymbols)
	return cloned
}

// InvalidImports returns invalid import issues.
func (vr *ValidationReport) InvalidImports() []*ValidationIssue {
	if vr == nil || len(vr.invalidImports) == 0 {
		return nil
	}
	cloned := make([]*ValidationIssue, len(vr.invalidImports))
	copy(cloned, vr.invalidImports)
	return cloned
}

// CircularReferences returns circular reference issues.
func (vr *ValidationReport) CircularReferences() []*ValidationIssue {
	if vr == nil || len(vr.circularReferences) == 0 {
		return nil
	}
	cloned := make([]*ValidationIssue, len(vr.circularReferences))
	copy(cloned, vr.circularReferences)
	return cloned
}

// TotalIssues returns the total count of validation issues.
func (vr *ValidationReport) TotalIssues() int {
	if vr == nil {
		return 0
	}
	return len(vr.issues)
}
