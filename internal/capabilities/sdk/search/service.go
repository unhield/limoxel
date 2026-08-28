package search

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/repository/query"
	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	sdkerr "github.com/unhield/limoxel/internal/capabilities/sdk/errors"
)

// Service provides the concrete SDK adapter implementation for SearchContract.
type Service struct {
	mu sync.RWMutex
	contracts.BaseContract
	repoService *query.RepositoryService
}

// Ensure Service implements SearchContract.
var _ contracts.SearchContract = (*Service)(nil)

// NewService constructs an initialized Search SDK service adapter.
func NewService(repoService *query.RepositoryService) *Service {
	return &Service{
		BaseContract: contracts.DefaultSearchContractMetadata(),
		repoService:  repoService,
	}
}

// Search executes a multi-domain or categorized search across repository data.
func (s *Service) Search(ctx context.Context, q contracts.SearchQuery) (*contracts.SearchResult, error) {
	if s == nil || s.repoService == nil {
		return nil, sdkerr.NewUnavailable("SearchService", "underlying repository service is unavailable")
	}
	clean := strings.TrimSpace(q.Query)
	if clean == "" {
		return nil, sdkerr.NewInvalidInput("search query string cannot be empty")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	searchEng := s.repoService.Search()
	if searchEng == nil {
		return nil, sdkerr.NewInvalidState("UNINITIALIZED", "repository search engine is uninitialized")
	}

	normOpts := q.Pagination.Normalize(50, 500)
	queryDomain := toInternalSearchDomain(q.Domain)

	start := time.Now()
	dto, err := searchEng.Search(clean, queryDomain, query.SearchOptions{
		MaxResults: normOpts.Limit,
	})
	duration := time.Since(start).Milliseconds()

	if err != nil && !strings.Contains(err.Error(), "no matches") {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_SEARCH_FAILED", "search execution failed")
	}

	matches := make([]contracts.SearchMatch, 0)
	totalMatches := 0
	if dto != nil {
		totalMatches = dto.TotalMatches()
		for _, m := range dto.Items() {
			matches = append(matches, contracts.SearchMatch{
				Domain:   toContractSearchDomain(m.Domain()),
				Name:     m.Name(),
				Package:  m.PackageName(),
				Location: m.Path(),
				Snippet:  m.Snippet(),
				Score:    m.Score(),
			})
		}
	}

	return &contracts.SearchResult{
		Query:        clean,
		Domain:       q.Domain,
		TotalMatches: totalMatches,
		Matches:      matches,
		DurationMs:   duration,
	}, nil
}

// SearchSymbols executes a search restricted to code symbols.
func (s *Service) SearchSymbols(ctx context.Context, queryStr string, pagination contracts.PaginationOptions) (*contracts.SearchResult, error) {
	return s.Search(ctx, contracts.SearchQuery{
		Query:      queryStr,
		Domain:     contracts.SearchDomainSymbol,
		Pagination: pagination,
	})
}

// SearchPackages executes a search restricted to package declarations.
func (s *Service) SearchPackages(ctx context.Context, queryStr string, pagination contracts.PaginationOptions) (*contracts.SearchResult, error) {
	return s.Search(ctx, contracts.SearchQuery{
		Query:      queryStr,
		Domain:     contracts.SearchDomainPackage,
		Pagination: pagination,
	})
}

// SearchFiles executes a search restricted to repository files.
func (s *Service) SearchFiles(ctx context.Context, pattern string, pagination contracts.PaginationOptions) (*contracts.SearchResult, error) {
	return s.Search(ctx, contracts.SearchQuery{
		Query:      pattern,
		Domain:     contracts.SearchDomainFile,
		Pagination: pagination,
	})
}

// SearchDocs executes a search restricted to markdown documentation and doc comments.
func (s *Service) SearchDocs(ctx context.Context, queryStr string, pagination contracts.PaginationOptions) (*contracts.SearchResult, error) {
	return s.Search(ctx, contracts.SearchQuery{
		Query:      queryStr,
		Domain:     contracts.SearchDomainDocumentation,
		Pagination: pagination,
	})
}

// SearchConfigs executes a search restricted to repository configuration elements.
func (s *Service) SearchConfigs(ctx context.Context, key string, pagination contracts.PaginationOptions) (*contracts.SearchResult, error) {
	return s.Search(ctx, contracts.SearchQuery{
		Query:      key,
		Domain:     contracts.SearchDomainConfiguration,
		Pagination: pagination,
	})
}

func toInternalSearchDomain(d contracts.SearchDomain) query.SearchDomain {
	switch d {
	case contracts.SearchDomainSymbol:
		return query.DomainSymbol
	case contracts.SearchDomainPackage:
		return query.DomainPackage
	case contracts.SearchDomainFile:
		return query.DomainFile
	case contracts.SearchDomainDocumentation:
		return query.DomainDocumentation
	case contracts.SearchDomainConfiguration:
		return query.DomainConfiguration
	default:
		return query.DomainAll
	}
}

func toContractSearchDomain(d query.SearchDomain) contracts.SearchDomain {
	switch d {
	case query.DomainSymbol:
		return contracts.SearchDomainSymbol
	case query.DomainPackage:
		return contracts.SearchDomainPackage
	case query.DomainFile:
		return contracts.SearchDomainFile
	case query.DomainDocumentation:
		return contracts.SearchDomainDocumentation
	case query.DomainConfiguration:
		return contracts.SearchDomainConfiguration
	default:
		return contracts.SearchDomainUnified
	}
}
