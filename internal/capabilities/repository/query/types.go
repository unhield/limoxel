package query

import (
	"time"

	"github.com/unhield/limoxel/internal/capabilities/repository/graph"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
)

// LifecycleState represents the operational state of the RepositoryService.
type LifecycleState string

const (
	StateUnloaded LifecycleState = "UNLOADED"
	StateLoading  LifecycleState = "LOADING"
	StateReady    LifecycleState = "READY"
	StateClosed   LifecycleState = "CLOSED"
)

// ScopeKind defines the scoping granularity for symbol listing and queries.
type ScopeKind string

const (
	ScopeRepository ScopeKind = "REPOSITORY"
	ScopeModule     ScopeKind = "MODULE"
	ScopePackage    ScopeKind = "PACKAGE"
	ScopeFile       ScopeKind = "FILE"
)

// SearchDomain defines the repository entities targetable by the SearchEngine.
type SearchDomain string

const (
	DomainAll           SearchDomain = "ALL"
	DomainSymbol        SearchDomain = "SYMBOL"
	DomainFile          SearchDomain = "FILE"
	DomainPackage       SearchDomain = "PACKAGE"
	DomainDocumentation SearchDomain = "DOCUMENTATION"
	DomainConfiguration SearchDomain = "CONFIGURATION"
)

// CallDirection defines the navigation direction for call graph lookups.
type CallDirection string

const (
	CallDirectionOutbound CallDirection = "OUTBOUND" // Caller -> Callee
	CallDirectionInbound  CallDirection = "INBOUND"  // Callee <- Caller
	CallDirectionBoth     CallDirection = "BOTH"
)

// ErrorCategory identifies the structured classification of a query error.
type ErrorCategory string

const (
	ErrCatInvalidInput ErrorCategory = "INVALID_INPUT"
	ErrCatNotFound     ErrorCategory = "NOT_FOUND"
	ErrCatUnavailable  ErrorCategory = "UNAVAILABLE"
	ErrCatNotLoaded    ErrorCategory = "NOT_LOADED"
	ErrCatLifecycle    ErrorCategory = "INVALID_LIFECYCLE"
	ErrCatInternal     ErrorCategory = "INTERNAL_FAILURE"
)

// SearchOptions configures deterministic search execution.
type SearchOptions struct {
	CaseSensitive bool
	ExactMatch    bool
	MaxResults    int
	MinScore      float64
}

// DefaultSearchOptions returns sensible deterministic defaults.
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		CaseSensitive: false,
		ExactMatch:    false,
		MaxResults:    50,
		MinScore:      0.0,
	}
}

// GraphQueryCriteria defines multidimensional criteria for querying graph structures.
type GraphQueryCriteria struct {
	NodeID           string
	NodeType         graph.NodeType
	RelationshipType graph.RelationshipType
	PackagePath      string
	FilePath         string
}

// PipelineModels contains pre-computed immutable models for loading the service directly.
type PipelineModels struct {
	AnalyzedAt time.Time
}

// SymbolKind is an alias to the authoritative symbol.SymbolKind for consumer convenience.
type SymbolKind = symbol.SymbolKind
