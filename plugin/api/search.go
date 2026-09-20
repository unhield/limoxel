package api

import "context"

// SearchAPI defines the developer-facing interface for repository-wide multi-domain search.
type SearchAPI interface {
	// Search executes a structured search query across specified domains with filters.
	Search(ctx context.Context, q SearchQuery) (*SearchResult, error)

	// SearchSymbols searches code symbols matching a query string.
	SearchSymbols(ctx context.Context, query string, opts PaginationOptions) (*SearchResult, error)

	// SearchPackages searches package namespaces matching a query string.
	SearchPackages(ctx context.Context, query string, opts PaginationOptions) (*SearchResult, error)

	// SearchFiles searches file paths and file names matching a query pattern.
	SearchFiles(ctx context.Context, pattern string, opts PaginationOptions) (*SearchResult, error)

	// SearchDocs searches documentation markdown and code comments.
	SearchDocs(ctx context.Context, query string, opts PaginationOptions) (*SearchResult, error)

	// SearchConfigs searches configuration keys, environment variables, and values.
	SearchConfigs(ctx context.Context, key string, opts PaginationOptions) (*SearchResult, error)
}
