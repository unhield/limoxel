package contracts

import (
	"context"

	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// SearchDomain defines the domain or entity scope of a search operation.
type SearchDomain string

const (
	// SearchDomainUnified searches across all entities and files simultaneously.
	SearchDomainUnified SearchDomain = "unified"

	// SearchDomainSymbol restricts search to code symbols.
	SearchDomainSymbol SearchDomain = "symbol"

	// SearchDomainPackage restricts search to package paths.
	SearchDomainPackage SearchDomain = "package"

	// SearchDomainModule restricts search to Go module declarations.
	SearchDomainModule SearchDomain = "module"

	// SearchDomainFile restricts search to source file paths.
	SearchDomainFile SearchDomain = "file"

	// SearchDomainDependency restricts search to manifest and package dependencies.
	SearchDomainDependency SearchDomain = "dependency"

	// SearchDomainDocumentation restricts search to markdown docs and doc comments.
	SearchDomainDocumentation SearchDomain = "documentation"

	// SearchDomainConfiguration restricts search to configuration keys.
	SearchDomainConfiguration SearchDomain = "configuration"
)

// String returns the string representation of SearchDomain.
func (d SearchDomain) String() string {
	return string(d)
}

// SearchQuery encapsulates query parameters, targeted domain, and pagination.
type SearchQuery struct {
	Query      string            `json:"query"`
	Domain     SearchDomain      `json:"domain"`
	Pagination PaginationOptions `json:"pagination"`
	Filters    map[string]string `json:"filters,omitempty"`
}

// SearchMatch represents an individual result item returned from a search.
type SearchMatch struct {
	Domain   SearchDomain `json:"domain"`
	Name     string       `json:"name"`
	Package  string       `json:"package,omitempty"`
	Location string       `json:"location,omitempty"`
	Snippet  string       `json:"snippet,omitempty"`
	Score    float64      `json:"score"`
}

// SearchResult represents the complete structured response of a search operation.
type SearchResult struct {
	Query        string        `json:"query"`
	Domain       SearchDomain  `json:"domain"`
	TotalMatches int           `json:"total_matches"`
	Matches      []SearchMatch `json:"matches"`
	DurationMs   int64         `json:"duration_ms"`
}

// SearchContract defines the public contract for repository-wide multi-domain search.
type SearchContract interface {
	Contract
	Search(ctx context.Context, q SearchQuery) (*SearchResult, error)
	SearchSymbols(ctx context.Context, query string, pagination PaginationOptions) (*SearchResult, error)
	SearchPackages(ctx context.Context, query string, pagination PaginationOptions) (*SearchResult, error)
	SearchFiles(ctx context.Context, pattern string, pagination PaginationOptions) (*SearchResult, error)
	SearchDocs(ctx context.Context, query string, pagination PaginationOptions) (*SearchResult, error)
	SearchConfigs(ctx context.Context, key string, pagination PaginationOptions) (*SearchResult, error)
}

// DefaultSearchContractMetadata returns default contract descriptor for Search operations.
func DefaultSearchContractMetadata() BaseContract {
	return NewBaseContract(
		"SearchContract",
		lifecycle.CapabilitySearch,
		version.SemVer{Major: 1, Minor: 0, Patch: 0},
		lifecycle.StateSupported,
		"Provides public multi-domain unified and categorized repository search.",
	)
}
