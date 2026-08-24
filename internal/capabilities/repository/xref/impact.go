package xref

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
)

// ChangeImpactAnalyzer performs deterministic change-impact analysis over repository symbols and relationships.
type ChangeImpactAnalyzer struct {
	symModel  *symbol.SymbolModel
	refDB     *ReferenceDatabase
	callGraph *CallGraph
	depModel  *dependency.DependencyModel
}

// NewChangeImpactAnalyzer constructs an immutable ChangeImpactAnalyzer.
func NewChangeImpactAnalyzer(
	symModel *symbol.SymbolModel,
	refDB *ReferenceDatabase,
	callGraph *CallGraph,
	depModel *dependency.DependencyModel,
) *ChangeImpactAnalyzer {
	return &ChangeImpactAnalyzer{
		symModel:  symModel,
		refDB:     refDB,
		callGraph: callGraph,
		depModel:  depModel,
	}
}

// AnalyzeFileImpact calculates affected downstream files, packages, and symbols when a file is modified.
func (cia *ChangeImpactAnalyzer) AnalyzeFileImpact(relPath string) *FileImpactResult {
	if cia == nil || cia.symModel == nil || cia.symModel.Symbols() == nil {
		return NewFileImpactResult(relPath, nil, nil, nil, ImpactDirect)
	}

	cleanPath := filepath.ToSlash(filepath.Clean(relPath))
	allSyms := cia.symModel.Symbols().AllSymbols()

	var fileSymbols []*symbol.Symbol
	for _, s := range allSyms {
		if s.FilePath() == cleanPath {
			fileSymbols = append(fileSymbols, s)
		}
	}

	impactedFilesSet := make(map[string]bool)
	impactedPackagesSet := make(map[string]bool)
	impactedSymbolsSet := make(map[string]bool)

	for _, fs := range fileSymbols {
		symImpact := cia.AnalyzeSymbolImpact(fs.ID())
		for _, sym := range symImpact.DirectlyImpactedSymbols() {
			impactedSymbolsSet[sym] = true
		}
		for _, sym := range symImpact.TransitivelyImpactedSymbols() {
			impactedSymbolsSet[sym] = true
		}
		for _, f := range symImpact.ImpactedFiles() {
			if f != cleanPath {
				impactedFilesSet[f] = true
			}
		}
		for _, p := range symImpact.ImpactedPackages() {
			impactedPackagesSet[p] = true
		}
	}

	var files []string
	for f := range impactedFilesSet {
		files = append(files, f)
	}
	var pkgs []string
	for p := range impactedPackagesSet {
		pkgs = append(pkgs, p)
	}
	var syms []string
	for s := range impactedSymbolsSet {
		syms = append(syms, s)
	}

	sev := ImpactDirect
	if len(files) > 0 || len(syms) > 0 {
		sev = ImpactTransitive
	}

	return NewFileImpactResult(
		cleanPath,
		files,
		pkgs,
		syms,
		sev,
	)
}

// AnalyzeSymbolImpact identifies directly and transitively affected callers and referencing symbols.
func (cia *ChangeImpactAnalyzer) AnalyzeSymbolImpact(symbolID string) *SymbolImpactResult {
	if cia == nil || cia.refDB == nil || cia.callGraph == nil {
		return NewSymbolImpactResult(symbolID, nil, nil, nil, nil)
	}

	cleanID := strings.TrimSpace(symbolID)
	if cleanID == "" {
		return NewSymbolImpactResult(symbolID, nil, nil, nil, nil)
	}

	directSymsSet := make(map[string]bool)
	transitiveSymsSet := make(map[string]bool)
	impactedFilesSet := make(map[string]bool)
	impactedPkgsSet := make(map[string]bool)

	// 1. Direct references
	incomingRefs := cia.refDB.ReferencesTo(cleanID)
	for _, r := range incomingRefs {
		if r.SourceSymbolID() != "" && r.SourceSymbolID() != cleanID && r.SourceSymbolID() != "." {
			directSymsSet[r.SourceSymbolID()] = true
			impactedFilesSet[r.FilePath()] = true
			impactedPkgsSet[extractPackageFromSymbolID(r.SourceSymbolID())] = true
		}
	}

	// 2. Direct callers
	callers := cia.callGraph.Callers(cleanID)
	for _, c := range callers {
		if c != cleanID {
			directSymsSet[c] = true
			impactedPkgsSet[extractPackageFromSymbolID(c)] = true
		}
	}

	// 3. Transitive callers via BFS
	visited := make(map[string]bool)
	queue := []string{}
	for s := range directSymsSet {
		visited[s] = true
		queue = append(queue, s)
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		upCallers := cia.callGraph.Callers(curr)
		for _, up := range upCallers {
			if !visited[up] && up != cleanID {
				visited[up] = true
				transitiveSymsSet[up] = true
				impactedPkgsSet[extractPackageFromSymbolID(up)] = true
				queue = append(queue, up)
			}
		}
	}

	var dSyms []string
	for s := range directSymsSet {
		dSyms = append(dSyms, s)
	}
	var tSyms []string
	for s := range transitiveSymsSet {
		tSyms = append(tSyms, s)
	}
	var files []string
	for f := range impactedFilesSet {
		files = append(files, f)
	}
	var pkgs []string
	for p := range impactedPkgsSet {
		pkgs = append(pkgs, p)
	}

	return NewSymbolImpactResult(
		cleanID,
		dSyms,
		tSyms,
		files,
		pkgs,
	)
}

// AnalyzePackageImpact determines downstream packages, files, and symbols affected by package changes.
func (cia *ChangeImpactAnalyzer) AnalyzePackageImpact(pkgPath string) *PackageImpactResult {
	if cia == nil || cia.symModel == nil || cia.symModel.Symbols() == nil {
		return NewPackageImpactResult(pkgPath, nil, nil, nil)
	}

	cleanPkg := strings.TrimSpace(pkgPath)
	allSyms := cia.symModel.Symbols().AllSymbols()

	downPackagesSet := make(map[string]bool)
	downFilesSet := make(map[string]bool)
	downSymbolsSet := make(map[string]bool)

	for _, s := range allSyms {
		if s.PackagePath() == cleanPkg {
			sImpact := cia.AnalyzeSymbolImpact(s.ID())
			for _, sym := range sImpact.DirectlyImpactedSymbols() {
				if extractPackageFromSymbolID(sym) != cleanPkg {
					downSymbolsSet[sym] = true
					downPackagesSet[extractPackageFromSymbolID(sym)] = true
				}
			}
			for _, sym := range sImpact.TransitivelyImpactedSymbols() {
				if extractPackageFromSymbolID(sym) != cleanPkg {
					downSymbolsSet[sym] = true
					downPackagesSet[extractPackageFromSymbolID(sym)] = true
				}
			}
			for _, f := range sImpact.ImpactedFiles() {
				downFilesSet[f] = true
			}
		}
	}

	var pkgs []string
	for p := range downPackagesSet {
		pkgs = append(pkgs, p)
	}
	var files []string
	for f := range downFilesSet {
		files = append(files, f)
	}
	var syms []string
	for s := range downSymbolsSet {
		syms = append(syms, s)
	}

	return NewPackageImpactResult(
		cleanPkg,
		pkgs,
		files,
		syms,
	)
}

// AnalyzeDependencyImpact determines impacted packages and files when an external dependency changes.
func (cia *ChangeImpactAnalyzer) AnalyzeDependencyImpact(depName string) *DependencyImpactResult {
	if cia == nil || cia.refDB == nil {
		return NewDependencyImpactResult(depName, nil, nil)
	}

	cleanDep := strings.TrimSpace(depName)
	if cleanDep == "" {
		return NewDependencyImpactResult(depName, nil, nil)
	}

	pkgsSet := make(map[string]bool)
	filesSet := make(map[string]bool)

	for _, r := range cia.refDB.AllReferences() {
		if strings.HasPrefix(r.TargetSymbolID(), cleanDep) || strings.Contains(r.TargetSymbolID(), cleanDep) {
			pkgsSet[extractPackageFromSymbolID(r.SourceSymbolID())] = true
			filesSet[r.FilePath()] = true
		}
	}

	var pkgs []string
	for p := range pkgsSet {
		pkgs = append(pkgs, p)
	}
	var files []string
	for f := range filesSet {
		files = append(files, f)
	}

	return NewDependencyImpactResult(
		cleanDep,
		pkgs,
		files,
	)
}

// DetectBreakingChanges compares two SymbolModel states and identifies breaking changes.
func (cia *ChangeImpactAnalyzer) DetectBreakingChanges(
	previous *symbol.SymbolModel,
	current *symbol.SymbolModel,
) *BreakingChangeResult {
	if previous == nil || previous.Symbols() == nil {
		return NewBreakingChangeResult(nil)
	}

	var items []*BreakingChangeItem
	prevSymbols := previous.Symbols().AllSymbols()

	for _, prevSym := range prevSymbols {
		currSym := current.Symbols().SymbolByID(prevSym.ID())

		// 1. Removed symbol check
		if currSym == nil {
			var affectedRefs []*Reference
			if cia != nil && cia.refDB != nil {
				affectedRefs = cia.refDB.ReferencesTo(prevSym.ID())
			}

			cat := BreakingConfirmed
			if len(affectedRefs) == 0 {
				cat = BreakingPotential
			}

			items = append(items, NewBreakingChangeItem(
				prevSym.ID(),
				cat,
				fmt.Sprintf("symbol_removed: %s (%s)", prevSym.ID(), prevSym.Kind()),
				affectedRefs,
			))
			continue
		}

		// 2. Signature change check for functions and methods
		if (prevSym.Kind() == symbol.SymbolKindFunction || prevSym.Kind() == symbol.SymbolKindMethod) &&
			prevSym.Signature() != currSym.Signature() {
			var affectedRefs []*Reference
			if cia != nil && cia.refDB != nil {
				affectedRefs = cia.refDB.ReferencesTo(prevSym.ID())
			}

			items = append(items, NewBreakingChangeItem(
				prevSym.ID(),
				BreakingConfirmed,
				fmt.Sprintf("signature_changed: %s -> %s", prevSym.Signature(), currSym.Signature()),
				affectedRefs,
			))
		}
	}

	return NewBreakingChangeResult(items)
}
