package discovery

import (
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

type mockCatalogProvider struct {
	plugins map[string]pubmarket.PluginDetail
}

func newMockCatalogProvider() *mockCatalogProvider {
	return &mockCatalogProvider{
		plugins: make(map[string]pubmarket.PluginDetail),
	}
}

func (m *mockCatalogProvider) ListPlugins(discoverableOnly bool) []pubmarket.PluginSummary {
	var list []pubmarket.PluginSummary
	for _, p := range m.plugins {
		if discoverableOnly && !p.Status.IsDiscoverable() {
			continue
		}
		list = append(list, p.PluginSummary)
	}
	return list
}

func (m *mockCatalogProvider) GetPlugin(id string) (*pubmarket.PluginDetail, error) {
	p, ok := m.plugins[id]
	if !ok {
		return nil, pubmarket.ErrNotFound
	}
	return &p, nil
}

func setupTestCatalog() *mockCatalogProvider {
	m := newMockCatalogProvider()
	now := time.Now().UTC()

	m.plugins["go-linter"] = pubmarket.PluginDetail{
		PluginSummary: pubmarket.PluginSummary{
			ID:             "go-linter",
			Name:           "Go Deep Linter",
			Description:    "High performance static analysis for Go codebases",
			PublisherID:    "godevs",
			PublisherName:  "Go Devs Org",
			LatestVersion:  version.SemVer{Major: 1, Minor: 2, Patch: 0},
			Categories:     []pubmarket.Category{pubmarket.CategoryAnalysis},
			Tags:           []string{"linter", "static-analysis", "golang"},
			TotalDownloads: 5000,
			AverageRating:  4.8,
			RatingCount:    120,
			Featured:       true,
			Status:         pubmarket.StatusPublished,
			UpdatedAt:      now.Add(-1 * time.Hour),
		},
	}

	m.plugins["ts-formatter"] = pubmarket.PluginDetail{
		PluginSummary: pubmarket.PluginSummary{
			ID:             "ts-formatter",
			Name:           "TypeScript Formatter",
			Description:    "Fast formatting tool for TypeScript and JavaScript",
			PublisherID:    "tsguild",
			PublisherName:  "TypeScript Guild",
			LatestVersion:  version.SemVer{Major: 2, Minor: 0, Patch: 1},
			Categories:     []pubmarket.Category{pubmarket.CategoryTools},
			Tags:           []string{"formatter", "typescript", "javascript"},
			TotalDownloads: 12000,
			AverageRating:  4.5,
			RatingCount:    300,
			Featured:       false,
			Status:         pubmarket.StatusPublished,
			UpdatedAt:      now.Add(-2 * time.Hour),
		},
	}

	m.plugins["sec-audit"] = pubmarket.PluginDetail{
		PluginSummary: pubmarket.PluginSummary{
			ID:             "sec-audit",
			Name:           "Security Auditor",
			Description:    "Automated vulnerability scanner for code and dependencies",
			PublisherID:    "secops",
			PublisherName:  "SecOps Collective",
			LatestVersion:  version.SemVer{Major: 1, Minor: 0, Patch: 0},
			Categories:     []pubmarket.Category{pubmarket.CategorySecurity, pubmarket.CategoryAnalysis},
			Tags:           []string{"security", "audit", "cve"},
			TotalDownloads: 3500,
			AverageRating:  4.9,
			RatingCount:    80,
			Featured:       true,
			Status:         pubmarket.StatusPublished,
			UpdatedAt:      now,
		},
	}

	m.plugins["draft-tool"] = pubmarket.PluginDetail{
		PluginSummary: pubmarket.PluginSummary{
			ID:          "draft-tool",
			Name:        "Draft Tool",
			Description: "Under review plugin",
			PublisherID: "godevs",
			Categories:  []pubmarket.Category{pubmarket.CategoryTools},
			Status:      pubmarket.StatusValidating, // Not discoverable!
			UpdatedAt:   now,
		},
	}

	return m
}

func TestDiscovery_KeywordSearch(t *testing.T) {
	provider := setupTestCatalog()
	engine := NewEngine(provider)

	// 1. Search by exact ID
	res, err := engine.Search(pubmarket.SearchQuery{Keyword: "go-linter"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || res.Items[0].ID != "go-linter" {
		t.Fatalf("expected 1 result for go-linter, got %d", res.Total)
	}

	// 2. Search by keyword in description
	res, err = engine.Search(pubmarket.SearchQuery{Keyword: "vulnerability"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || res.Items[0].ID != "sec-audit" {
		t.Fatalf("expected sec-audit for vulnerability search, got %d", res.Total)
	}

	// 3. Search by tag
	res, err = engine.Search(pubmarket.SearchQuery{Keyword: "typescript"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || res.Items[0].ID != "ts-formatter" {
		t.Fatalf("expected ts-formatter for tag search, got %d", res.Total)
	}

	// 4. Verify undiscoverable plugins are excluded
	res, err = engine.Search(pubmarket.SearchQuery{Keyword: "draft"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 0 {
		t.Fatalf("expected 0 results for undiscoverable plugin, got %d", res.Total)
	}
}

func TestDiscovery_Filters(t *testing.T) {
	provider := setupTestCatalog()
	engine := NewEngine(provider)

	// Category filter: Analysis matches go-linter and sec-audit
	res, err := engine.Search(pubmarket.SearchQuery{Category: pubmarket.CategoryAnalysis})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 2 {
		t.Fatalf("expected 2 analysis plugins, got %d", res.Total)
	}

	// MinRating filter: >= 4.7 matches go-linter (4.8) and sec-audit (4.9)
	res, err = engine.Search(pubmarket.SearchQuery{MinRating: 4.7})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 2 {
		t.Fatalf("expected 2 plugins with rating >= 4.7, got %d", res.Total)
	}

	// Publisher filter: godevs matches go-linter
	res, err = engine.Search(pubmarket.SearchQuery{Publisher: "godevs"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || res.Items[0].ID != "go-linter" {
		t.Fatalf("expected 1 result for publisher godevs, got %d", res.Total)
	}
}

func TestDiscovery_SortingAndPagination(t *testing.T) {
	provider := setupTestCatalog()
	engine := NewEngine(provider)

	// Sort by downloads descending: ts-formatter (12000), go-linter (5000), sec-audit (3500)
	res, err := engine.Search(pubmarket.SearchQuery{
		Sort:  pubmarket.SortByDownloads,
		Order: pubmarket.OrderDesc,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 3 {
		t.Fatalf("expected 3 total plugins, got %d", res.Total)
	}
	if res.Items[0].ID != "ts-formatter" || res.Items[1].ID != "go-linter" || res.Items[2].ID != "sec-audit" {
		t.Fatalf("unexpected order by downloads: %+v", res.Items)
	}

	// Pagination: offset 1, limit 1
	res, err = engine.Search(pubmarket.SearchQuery{
		Sort:   pubmarket.SortByDownloads,
		Order:  pubmarket.OrderDesc,
		Offset: 1,
		Limit:  1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 1 || res.Items[0].ID != "go-linter" {
		t.Fatalf("expected go-linter at page 2, got %+v", res.Items)
	}
}

func TestDiscovery_Ranking(t *testing.T) {
	provider := setupTestCatalog()
	engine := NewEngine(provider)

	// Featured: should return go-linter and sec-audit
	featured, err := engine.GetFeatured(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(featured) < 2 {
		t.Fatalf("expected at least 2 featured, got %d", len(featured))
	}

	// Trending: ts-formatter with 12k downloads should be #1
	trending, err := engine.GetTrending(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(trending) != 2 || trending[0].ID != "ts-formatter" {
		t.Fatalf("expected ts-formatter top trending, got %+v", trending)
	}

	// Recommendations: for go-linter (category: analysis), sec-audit also shares category analysis
	recs, err := engine.GetRecommended("go-linter", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) == 0 {
		t.Fatal("expected recommendation for go-linter")
	}
	if recs[0].ID != "sec-audit" {
		t.Fatalf("expected sec-audit recommended for go-linter, got %s", recs[0].ID)
	}
}

func TestDiscovery_CategoryBreakdown(t *testing.T) {
	provider := setupTestCatalog()
	engine := NewEngine(provider)

	cats, err := engine.GetCategoryBreakdown()
	if err != nil {
		t.Fatal(err)
	}
	foundAnalysis := false
	for _, c := range cats {
		if c.Category == pubmarket.CategoryAnalysis {
			foundAnalysis = true
			if c.PluginCount != 2 {
				t.Fatalf("expected 2 plugins in analysis category, got %d", c.PluginCount)
			}
		}
	}
	if !foundAnalysis {
		t.Fatal("analysis category missing from breakdown")
	}
}
