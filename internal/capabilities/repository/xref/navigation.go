package xref

import (
	"fmt"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
)

// NavigationEngine provides deterministic code navigation across verified repository symbols and relationships.
type NavigationEngine struct {
	symModel *symbol.SymbolModel
	refDB    *ReferenceDatabase
	depModel *dependency.DependencyModel
}

// NewNavigationEngine constructs a NavigationEngine.
func NewNavigationEngine(
	symModel *symbol.SymbolModel,
	refDB *ReferenceDatabase,
	depModel *dependency.DependencyModel,
) *NavigationEngine {
	return &NavigationEngine{
		symModel: symModel,
		refDB:    refDB,
		depModel: depModel,
	}
}

// GoToDefinition resolves a symbol identifier to its authoritative declaration, or returns ambiguity where multiple candidates exist.
func (ne *NavigationEngine) GoToDefinition(symbolID string) (*DefinitionResult, error) {
	if ne == nil || ne.symModel == nil || ne.symModel.Symbols() == nil {
		return nil, ErrNilSymbolModel
	}

	cleanID := strings.TrimSpace(symbolID)
	if cleanID == "" {
		return nil, ErrSymbolNotFound
	}

	// 1. Exact lookup by canonical Symbol ID
	if sym := ne.symModel.Symbols().SymbolByID(cleanID); sym != nil {
		return NewDefinitionResult(
			sym,
			[]*symbol.Symbol{sym},
			sym.Position(),
			StateResolved,
			fmt.Sprintf("exact_symbol_match:%s", sym.ID()),
		), nil
	}

	// 2. Lookup by Name across repository
	candidates := ne.symModel.Symbols().SymbolsByName(cleanID)
	if len(candidates) == 1 {
		sym := candidates[0]
		return NewDefinitionResult(
			sym,
			candidates,
			sym.Position(),
			StateResolved,
			fmt.Sprintf("unique_name_match:%s", sym.ID()),
		), nil
	}

	if len(candidates) > 1 {
		return NewDefinitionResult(
			nil,
			candidates,
			nil,
			StateAmbiguous,
			fmt.Sprintf("ambiguous_candidates_count:%d", len(candidates)),
		), ErrAmbiguousDefinition
	}

	// 3. Check if target is external
	if strings.Contains(cleanID, "/") || strings.Contains(cleanID, ".") {
		return NewDefinitionResult(
			nil,
			nil,
			nil,
			StateUnresolvedExternal,
			"external_target_symbol",
		), ErrSymbolNotFound
	}

	return NewDefinitionResult(
		nil,
		nil,
		nil,
		StateBroken,
		"unresolved_symbol_target",
	), ErrSymbolNotFound
}

// FindReferences returns all source references targeting the given symbol ID.
func (ne *NavigationEngine) FindReferences(symbolID string) []*Reference {
	if ne == nil || ne.refDB == nil {
		return nil
	}
	cleanID := strings.TrimSpace(symbolID)
	if cleanID == "" {
		return nil
	}
	return ne.refDB.ReferencesTo(cleanID)
}

// FindImplementations returns all structs or types implementing the specified interface symbol ID.
func (ne *NavigationEngine) FindImplementations(interfaceSymbolID string) []*symbol.Symbol {
	if ne == nil || ne.symModel == nil || ne.symModel.Relationships() == nil || ne.symModel.Symbols() == nil {
		return nil
	}

	cleanID := strings.TrimSpace(interfaceSymbolID)
	if cleanID == "" {
		return nil
	}

	rels := ne.symModel.Relationships().RelationshipsForTarget(cleanID)
	var implSymbols []*symbol.Symbol

	for _, r := range rels {
		if r.Kind() == symbol.RelInterfaceImplementation {
			if sym := ne.symModel.Symbols().SymbolByID(r.SourceID()); sym != nil {
				implSymbols = append(implSymbols, sym)
			}
		}
	}

	return implSymbols
}

// PackageNavigation returns package-level navigation data including declared symbols, imports, and downstream importers.
func (ne *NavigationEngine) PackageNavigation(pkgPath string) *PackageNavResult {
	if ne == nil || ne.symModel == nil || ne.symModel.Symbols() == nil {
		return nil
	}

	cleanPkg := strings.TrimSpace(pkgPath)
	allSyms := ne.symModel.Symbols().AllSymbols()

	var pkgSyms []*symbol.Symbol
	pkgName := ""

	for _, s := range allSyms {
		if s.PackagePath() == cleanPkg {
			pkgSyms = append(pkgSyms, s)
			if pkgName == "" {
				pkgName = s.PackageName()
			}
		}
	}

	var imports []string
	var importedBy []string

	if ne.depModel != nil {
		importedBy = ne.ReverseDependencyLookup(cleanPkg)
	}

	return NewPackageNavResult(
		cleanPkg,
		pkgName,
		pkgSyms,
		imports,
		importedBy,
	)
}

// ReverseDependencyLookup returns downstream internal packages that import the specified package path.
func (ne *NavigationEngine) ReverseDependencyLookup(pkgPath string) []string {
	if ne == nil || ne.symModel == nil || ne.refDB == nil {
		return nil
	}

	cleanPkg := strings.TrimSpace(pkgPath)
	if cleanPkg == "" {
		return nil
	}

	importersMap := make(map[string]bool)

	// Sourced from references database
	for _, ref := range ne.refDB.AllReferences() {
		targetPkg := extractPackageFromSymbolID(ref.TargetSymbolID())
		if targetPkg == cleanPkg {
			sourcePkg := extractPackageFromSymbolID(ref.SourceSymbolID())
			if sourcePkg != "" && sourcePkg != cleanPkg {
				importersMap[sourcePkg] = true
			}
		}
	}

	var result []string
	for p := range importersMap {
		result = append(result, p)
	}

	return result
}

// extractPackageFromSymbolID extracts the package component from a symbol ID.
func extractPackageFromSymbolID(symbolID string) string {
	clean := strings.TrimSpace(symbolID)
	if clean == "" || clean == "." {
		return "."
	}
	if idx := strings.LastIndex(clean, ".("); idx != -1 {
		return clean[:idx]
	}
	if idx := strings.LastIndex(clean, "."); idx != -1 {
		return clean[:idx]
	}
	return "."
}
