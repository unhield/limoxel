package api

import "context"

// SymbolAPI defines the developer-facing interface for symbol discovery and code relationships.
type SymbolAPI interface {
	// LookupSymbol resolves a symbol by its canonical ID or fully-qualified name.
	LookupSymbol(ctx context.Context, symbolIDOrName string) (*SymbolInfo, error)

	// FindSymbols searches for symbols matching a name pattern within a package scope.
	FindSymbols(ctx context.Context, pattern string, pkgScope string, opts PaginationOptions) ([]SymbolInfo, error)

	// SymbolHierarchy builds the inheritance, interface, or call hierarchy tree for a symbol.
	SymbolHierarchy(ctx context.Context, symbolID string) (*SymbolHierarchyNode, error)

	// SymbolReferences locates source reference sites and usages for a symbol.
	SymbolReferences(ctx context.Context, symbolID string, opts PaginationOptions) ([]SymbolReference, error)

	// SymbolDocumentation returns extracted documentation comments for a symbol.
	SymbolDocumentation(ctx context.Context, symbolID string) (string, error)
}
