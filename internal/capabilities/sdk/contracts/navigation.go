package contracts

import (
	"context"

	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// DefinitionResult encapsulates the outcome of a go-to-definition operation.
type DefinitionResult struct {
	Target     *NavigationTarget  `json:"target,omitempty"`
	Candidates []NavigationTarget `json:"candidates,omitempty"`
	State      string             `json:"state"`
}

// ReferenceResult encapsulates symbol reference lookup results.
type ReferenceResult struct {
	SymbolID   string             `json:"symbol_id"`
	References []NavigationTarget `json:"references"`
	TotalCount int                `json:"total_count"`
}

// CallHierarchyItem represents a single callable entity in a call hierarchy.
type CallHierarchyItem struct {
	SymbolID string         `json:"symbol_id"`
	Name     string         `json:"name"`
	Kind     string         `json:"kind"`
	Location SymbolLocation `json:"location"`
	Package  string         `json:"package,omitempty"`
}

// CallHierarchyNode represents callers and callees connected to a symbol.
type CallHierarchyNode struct {
	Item    CallHierarchyItem   `json:"item"`
	Callers []CallHierarchyItem `json:"callers,omitempty"`
	Callees []CallHierarchyItem `json:"callees,omitempty"`
}

// NavSymbolHierarchyNode represents structural parent/child symbol relationships.
type NavSymbolHierarchyNode struct {
	SymbolID string                   `json:"symbol_id"`
	Name     string                   `json:"name"`
	Kind     string                   `json:"kind"`
	ParentID string                   `json:"parent_id,omitempty"`
	Children []NavSymbolHierarchyNode `json:"children,omitempty"`
}

// NavigationContextResult encapsulates enriched contextual intelligence surrounding a target symbol.
type NavigationContextResult struct {
	Target            NavigationTarget   `json:"target"`
	ContainingFile    string             `json:"containing_file"`
	ContainingPackage string             `json:"containing_package"`
	RelatedSymbols    []NavigationTarget `json:"related_symbols,omitempty"`
	Relationships     []string           `json:"relationships,omitempty"`
}

// NavigationContract defines the public contract for code navigation, hierarchy, and context discovery.
type NavigationContract interface {
	Contract
	GoToDefinition(ctx context.Context, symbolIDOrName string) (*DefinitionResult, error)
	FindReferences(ctx context.Context, symbolIDOrName string, opts PaginationOptions) (*ReferenceResult, error)
	CallHierarchy(ctx context.Context, symbolIDOrName string) (*CallHierarchyNode, error)
	SymbolHierarchy(ctx context.Context, symbolIDOrName string) (*NavSymbolHierarchyNode, error)
	NavigationContext(ctx context.Context, symbolIDOrName string) (*NavigationContextResult, error)
	Navigate(ctx context.Context, symbolID string, relKind string) (*NavigationResult, error)
}

// DefaultNavigationContractMetadata returns default contract descriptor for Navigation operations.
func DefaultNavigationContractMetadata() BaseContract {
	return NewBaseContract(
		"NavigationContract",
		lifecycle.CapabilityIntelligence,
		version.SemVer{Major: 1, Minor: 0, Patch: 0},
		lifecycle.StateSupported,
		"Provides public definition resolution, reference discovery, call hierarchy, and navigation context.",
	)
}
