package semantic

import (
	"path/filepath"
	"strings"
	"unicode"

	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
)

// SymbolResolutionResult represents the outcome of resolving a symbol identifier or reference.
type SymbolResolutionResult struct {
	Identifier      string
	ResolvedSymbol  *SemanticSymbol
	ResolutionState ResolutionState
	Ownership       string
	Visibility      VisibilityKind
	ErrorMessage    string
}

// SymbolResolver performs deterministic symbol resolution, ownership mapping, and visibility checks.
type SymbolResolver struct {
	symbols       map[string]*SemanticSymbol
	scopeResolver *ScopeResolver
}

// NewSymbolResolver creates an initialized SymbolResolver.
func NewSymbolResolver(symbols map[string]*SemanticSymbol, scopeResolver *ScopeResolver) *SymbolResolver {
	symMap := make(map[string]*SemanticSymbol, len(symbols))
	for k, v := range symbols {
		symMap[k] = v
	}

	return &SymbolResolver{
		symbols:       symMap,
		scopeResolver: scopeResolver,
	}
}

// ResolveSymbol resolves a symbol name from a given scope context.
func (r *SymbolResolver) ResolveSymbol(name string, scope *SemanticScope) *SymbolResolutionResult {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return &SymbolResolutionResult{
			Identifier:      name,
			ResolutionState: StateUnresolved,
			ErrorMessage:    "empty symbol identifier",
		}
	}

	// Direct ID match (with or without prefix)
	if sym, exists := r.symbols[cleanName]; exists && sym != nil {
		return &SymbolResolutionResult{
			Identifier:      cleanName,
			ResolvedSymbol:  sym,
			ResolutionState: StateResolved,
			Ownership:       sym.Ownership(),
			Visibility:      sym.Visibility(),
		}
	}
	if sym, exists := r.symbols["sym:"+cleanName]; exists && sym != nil {
		return &SymbolResolutionResult{
			Identifier:      cleanName,
			ResolvedSymbol:  sym,
			ResolutionState: StateResolved,
			Ownership:       sym.Ownership(),
			Visibility:      sym.Visibility(),
		}
	}

	// If a scope is provided, perform hierarchical scope lookup
	if scope != nil && r.scopeResolver != nil {
		lookup := r.scopeResolver.ResolveInScope(scope.ID(), cleanName)
		if lookup.ResolutionState == StateResolved && lookup.Symbol != nil {
			return &SymbolResolutionResult{
				Identifier:      cleanName,
				ResolvedSymbol:  lookup.Symbol,
				ResolutionState: StateResolved,
				Ownership:       lookup.Symbol.Ownership(),
				Visibility:      lookup.Symbol.Visibility(),
			}
		} else if lookup.ResolutionState == StateAmbiguous {
			return &SymbolResolutionResult{
				Identifier:      cleanName,
				ResolutionState: StateAmbiguous,
				ErrorMessage:    "multiple symbols with matching name in scope",
			}
		}
	}

	// Package-qualified lookup (e.g. "pkg.SymbolName")
	if strings.Contains(cleanName, ".") {
		parts := strings.Split(cleanName, ".")
		pkgOrType := parts[0]
		symName := parts[1]

		for _, sym := range r.symbols {
			if sym.Name() == symName {
				if filepath.Base(sym.PackagePath()) == pkgOrType ||
					strings.HasSuffix(sym.PackagePath(), pkgOrType) ||
					sym.Ownership() == pkgOrType ||
					strings.EqualFold(filepath.Base(sym.PackagePath()), pkgOrType) ||
					strings.EqualFold(sym.PackagePath(), pkgOrType) {
					return &SymbolResolutionResult{
						Identifier:      cleanName,
						ResolvedSymbol:  sym,
						ResolutionState: StateResolved,
						Ownership:       sym.Ownership(),
						Visibility:      sym.Visibility(),
					}
				}
			}
		}
	}

	// Global lookup across all symbols
	var matches []*SemanticSymbol
	for _, sym := range r.symbols {
		if sym.Name() == cleanName {
			matches = append(matches, sym)
		}
	}

	if len(matches) == 1 {
		return &SymbolResolutionResult{
			Identifier:      cleanName,
			ResolvedSymbol:  matches[0],
			ResolutionState: StateResolved,
			Ownership:       matches[0].Ownership(),
			Visibility:      matches[0].Visibility(),
		}
	} else if len(matches) > 1 {
		return &SymbolResolutionResult{
			Identifier:      cleanName,
			ResolutionState: StateAmbiguous,
			ErrorMessage:    "multiple global symbols matching name",
		}
	}

	return &SymbolResolutionResult{
		Identifier:      cleanName,
		ResolutionState: StateUnresolved,
		ErrorMessage:    "symbol not found in repository",
	}
}

// GetSymbolOwnership determines the owning package or enclosing struct/type for a symbol.
func (r *SymbolResolver) GetSymbolOwnership(symID string) string {
	sym := r.symbols[symID]
	if sym == nil {
		return ""
	}
	return sym.Ownership()
}

// CheckVisibility verifies whether a target symbol is visible and accessible from a caller package.
func (r *SymbolResolver) CheckVisibility(symID string, fromPkgPath string) bool {
	sym := r.symbols[symID]
	if sym == nil {
		return false
	}

	cleanFrom := filepath.ToSlash(filepath.Clean(fromPkgPath))
	cleanTarget := filepath.ToSlash(filepath.Clean(sym.PackagePath()))

	// Same package is always visible
	if cleanFrom == cleanTarget {
		return true
	}

	// Exported symbols (capitalized identifier) are publicly visible
	if sym.IsExported() || sym.Visibility() == VisibilityPublic {
		return true
	}

	// Local or unexported symbols in other packages are not visible
	return false
}

// DetermineVisibility calculates VisibilityKind from symbol name and package context.
func DetermineVisibility(name string, kind symbol.SymbolKind) VisibilityKind {
	cleanName := strings.TrimSpace(name)
	if len(cleanName) == 0 {
		return VisibilityPackagePrivate
	}

	// Go export rule: first rune is uppercase
	firstRune := []rune(cleanName)[0]
	if unicode.IsUpper(firstRune) {
		return VisibilityPublic
	}

	return VisibilityPackagePrivate
}
