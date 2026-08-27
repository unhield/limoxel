package search_test

import (
	"context"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	"github.com/unhield/limoxel/internal/capabilities/sdk/search"
	"github.com/unhield/limoxel/internal/capabilities/sdk/testutil"
)

func TestSearchServiceOperations(t *testing.T) {
	ctx := context.Background()
	querySvc, _ := testutil.SetupTestRepository(t)

	svc := search.NewService(querySvc)

	// 1. Unified Search
	res, err := svc.Search(ctx, contracts.SearchQuery{
		Query:      "Add",
		Domain:     contracts.SearchDomainUnified,
		Pagination: contracts.PaginationOptions{Limit: 10},
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if res.Query != "Add" {
		t.Errorf("got query %q, want Add", res.Query)
	}

	// 2. SearchSymbols
	symRes, err := svc.SearchSymbols(ctx, "Add", contracts.PaginationOptions{})
	if err != nil {
		t.Fatalf("SearchSymbols failed: %v", err)
	}
	if symRes.TotalMatches == 0 {
		t.Errorf("expected matches for symbol Add")
	}

	// 3. SearchPackages
	pkgRes, err := svc.SearchPackages(ctx, "math", contracts.PaginationOptions{})
	if err != nil {
		t.Fatalf("SearchPackages failed: %v", err)
	}
	if pkgRes.TotalMatches == 0 {
		t.Errorf("expected matches for package math")
	}

	// 4. SearchFiles
	fileRes, err := svc.SearchFiles(ctx, "main.go", contracts.PaginationOptions{})
	if err != nil {
		t.Fatalf("SearchFiles failed: %v", err)
	}
	if fileRes.TotalMatches == 0 {
		t.Errorf("expected matches for file main.go")
	}

	// 5. SearchDocs
	docRes, err := svc.SearchDocs(ctx, "Documentation", contracts.PaginationOptions{})
	if err != nil {
		t.Fatalf("SearchDocs failed: %v", err)
	}
	if docRes.TotalMatches == 0 {
		t.Logf("no doc matches found (acceptable for sample repo)")
	}

	// 6. SearchConfigs
	cfgRes, err := svc.SearchConfigs(ctx, "app_name", contracts.PaginationOptions{})
	if err != nil {
		t.Fatalf("SearchConfigs failed: %v", err)
	}
	if cfgRes.TotalMatches == 0 {
		t.Logf("no config matches found (acceptable for sample repo)")
	}
}

func TestSearchServiceErrors(t *testing.T) {
	ctx := context.Background()
	svc := search.NewService(nil)

	if _, err := svc.Search(ctx, contracts.SearchQuery{Query: "test"}); err == nil {
		t.Errorf("expected error on nil service Search")
	}
	if _, err := svc.Search(ctx, contracts.SearchQuery{Query: ""}); err == nil {
		t.Errorf("expected error for empty search query")
	}

	var nilSvc *search.Service
	if _, err := nilSvc.Search(ctx, contracts.SearchQuery{Query: "test"}); err == nil {
		t.Errorf("expected error on typed nil service Search")
	}
	if _, err := nilSvc.SearchSymbols(ctx, "test", contracts.PaginationOptions{}); err == nil {
		t.Errorf("expected error on typed nil service SearchSymbols")
	}
	if _, err := nilSvc.SearchPackages(ctx, "test", contracts.PaginationOptions{}); err == nil {
		t.Errorf("expected error on typed nil service SearchPackages")
	}
	if _, err := nilSvc.SearchFiles(ctx, "test", contracts.PaginationOptions{}); err == nil {
		t.Errorf("expected error on typed nil service SearchFiles")
	}
	if _, err := nilSvc.SearchDocs(ctx, "test", contracts.PaginationOptions{}); err == nil {
		t.Errorf("expected error on typed nil service SearchDocs")
	}
	if _, err := nilSvc.SearchConfigs(ctx, "test", contracts.PaginationOptions{}); err == nil {
		t.Errorf("expected error on typed nil service SearchConfigs")
	}
}
