package contracts

import (
	"context"

	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// SymbolKind defines normalized public classifications for code symbols.
type SymbolKind string

const (
	// SymbolKindFunction represents a standalone function.
	SymbolKindFunction SymbolKind = "function"

	// SymbolKindMethod represents a method attached to a receiver type.
	SymbolKindMethod SymbolKind = "method"

	// SymbolKindStruct represents a struct data type definition.
	SymbolKindStruct SymbolKind = "struct"

	// SymbolKindInterface represents an interface definition.
	SymbolKindInterface SymbolKind = "interface"

	// SymbolKindType represents a type alias or type definition.
	SymbolKindType SymbolKind = "type"

	// SymbolKindVariable represents a package-level or exported variable.
	SymbolKindVariable SymbolKind = "variable"

	// SymbolKindConstant represents a constant definition.
	SymbolKindConstant SymbolKind = "constant"

	// SymbolKindPackage represents a package namespace.
	SymbolKindPackage SymbolKind = "package"

	// SymbolKindFile represents a source file entity.
	SymbolKindFile SymbolKind = "file"
)

// String returns the string representation of SymbolKind.
func (k SymbolKind) String() string {
	return string(k)
}

// SymbolLocation represents the precise source code location of a symbol.
type SymbolLocation struct {
	FilePath    string `json:"file_path"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	StartColumn int    `json:"start_column,omitempty"`
	EndColumn   int    `json:"end_column,omitempty"`
}

// SymbolInfo represents public information about a code symbol.
type SymbolInfo struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Package    string         `json:"package"`
	Kind       SymbolKind     `json:"kind"`
	Location   SymbolLocation `json:"location"`
	Signature  string         `json:"signature,omitempty"`
	IsExported bool           `json:"is_exported"`
	DocComment string         `json:"doc_comment,omitempty"`
	Ownership  string         `json:"ownership,omitempty"`
}

// SymbolReference represents a reference or call edge to a target symbol.
type SymbolReference struct {
	SourceSymbolID string `json:"source_symbol_id,omitempty"`
	SourceFile     string `json:"source_file"`
	SourceLine     int    `json:"source_line"`
	ReferenceKind  string `json:"reference_kind"`
	Evidence       string `json:"evidence,omitempty"`
}

// SymbolHierarchyNode represents a node in a symbol inheritance or call hierarchy tree.
type SymbolHierarchyNode struct {
	Symbol   SymbolInfo            `json:"symbol"`
	Children []SymbolHierarchyNode `json:"children,omitempty"`
}

// SymbolContract defines the public contract for symbol discovery, lookup, and cross-references.
type SymbolContract interface {
	Contract
	LookupSymbol(ctx context.Context, symbolIDOrName string) (*SymbolInfo, error)
	SymbolHierarchy(ctx context.Context, symbolID string) (*SymbolHierarchyNode, error)
	SymbolReferences(ctx context.Context, symbolID string, opts PaginationOptions) ([]SymbolReference, error)
	SymbolDocumentation(ctx context.Context, symbolID string) (string, error)
	SymbolOwnership(ctx context.Context, symbolID string) (string, error)
}

// DefaultSymbolContractMetadata returns default contract descriptor for Symbol operations.
func DefaultSymbolContractMetadata() BaseContract {
	return NewBaseContract(
		"SymbolContract",
		lifecycle.CapabilitySymbol,
		version.SemVer{Major: 1, Minor: 0, Patch: 0},
		lifecycle.StateSupported,
		"Provides public symbol lookup, hierarchy, reference traversal, and documentation access.",
	)
}
