package discovery

import (
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

// CatalogProvider defines the read-only catalog access required by the discovery engine.
type CatalogProvider interface {
	ListPlugins(discoverableOnly bool) []pubmarket.PluginSummary
	GetPlugin(id string) (*pubmarket.PluginDetail, error)
}

// VelocityTracker provides windowed download velocity metrics.
type VelocityTracker interface {
	GetVelocity(pluginID string, window time.Duration) int
}

// Engine coordinates catalog indexing, multi-criteria search, ranking, and recommendations.
type Engine struct {
	mu       sync.RWMutex
	provider CatalogProvider
	velocity VelocityTracker
}

// NewEngine creates an initialized discovery engine.
func NewEngine(provider CatalogProvider) *Engine {
	return &Engine{
		provider: provider,
	}
}

// SetVelocityTracker assigns the download velocity metrics provider.
func (e *Engine) SetVelocityTracker(v VelocityTracker) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.velocity = v
}

// scoredPlugin wraps a summary with its computed relevance score.
type scoredPlugin struct {
	summary pubmarket.PluginSummary
	score   float64
}

// Search executes multi-criteria search with relevance scoring, filtering, and pagination.
func (e *Engine) Search(query pubmarket.SearchQuery) (pubmarket.SearchResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	allPlugins := e.provider.ListPlugins(true)

	tokens := tokenize(query.Keyword)
	var matched []scoredPlugin

	for _, p := range allPlugins {
		// Only discoverable plugins appear in search
		if !p.Status.IsDiscoverable() {
			continue
		}

		// Category filter
		if query.Category != "" && query.Category != pubmarket.CategoryAll {
			hasCat := false
			for _, cat := range p.Categories {
				if cat == query.Category {
					hasCat = true
					break
				}
			}
			if !hasCat {
				continue
			}
		}

		// Publisher filter
		if query.Publisher != "" && !strings.EqualFold(p.PublisherID, query.Publisher) {
			continue
		}

		// Tags filter (all specified tags must match)
		if len(query.Tags) > 0 {
			allTagsMatch := true
			for _, qTag := range query.Tags {
				found := false
				for _, pTag := range p.Tags {
					if strings.EqualFold(pTag, qTag) {
						found = true
						break
					}
				}
				if !found {
					allTagsMatch = false
					break
				}
			}
			if !allTagsMatch {
				continue
			}
		}

		// Featured filter
		if query.Featured != nil && p.Featured != *query.Featured {
			continue
		}

		// MinRating filter
		if query.MinRating > 0 && p.AverageRating < query.MinRating {
			continue
		}

		// Keyword relevance scoring
		score := 0.0
		if len(tokens) > 0 {
			score = computeRelevance(p, tokens)
			if score <= 0 {
				continue
			}
		} else {
			// Base score when no keywords supplied
			score = 1.0
		}

		matched = append(matched, scoredPlugin{summary: p, score: score})
	}

	// Sorting
	sortScoredPlugins(matched, query.Sort, query.Order)

	total := len(matched)
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var items []pubmarket.PluginSummary
	if offset < total {
		end := offset + limit
		if end > total {
			end = total
		}
		items = make([]pubmarket.PluginSummary, 0, end-offset)
		for _, sp := range matched[offset:end] {
			items = append(items, sp.summary)
		}
	} else {
		items = []pubmarket.PluginSummary{}
	}

	return pubmarket.SearchResult{
		Items:  items,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}, nil
}

// tokenize splits a search string into lowercase word tokens.
func tokenize(input string) []string {
	clean := strings.ToLower(strings.TrimSpace(input))
	if clean == "" {
		return nil
	}
	fields := strings.FieldsFunc(clean, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '-' && r != '_'
	})
	return fields
}

// computeRelevance scores a plugin based on query tokens.
func computeRelevance(p pubmarket.PluginSummary, tokens []string) float64 {
	totalScore := 0.0
	idLower := strings.ToLower(p.ID)
	nameLower := strings.ToLower(p.Name)
	descLower := strings.ToLower(p.Description)
	pubLower := strings.ToLower(p.PublisherName)

	for _, token := range tokens {
		matchedToken := false

		// Exact ID match
		if idLower == token {
			totalScore += 100.0
			matchedToken = true
		} else if strings.Contains(idLower, token) {
			totalScore += 40.0
			matchedToken = true
		}

		// Exact Name match
		if nameLower == token {
			totalScore += 80.0
			matchedToken = true
		} else if strings.Contains(nameLower, token) {
			totalScore += 30.0
			matchedToken = true
		}

		// Description match
		if strings.Contains(descLower, token) {
			totalScore += 10.0
			matchedToken = true
		}

		// Tag match
		for _, tag := range p.Tags {
			tagLower := strings.ToLower(tag)
			if tagLower == token {
				totalScore += 25.0
				matchedToken = true
			} else if strings.Contains(tagLower, token) {
				totalScore += 12.0
				matchedToken = true
			}
		}

		// Publisher match
		if strings.Contains(pubLower, token) || strings.EqualFold(p.PublisherID, token) {
			totalScore += 15.0
			matchedToken = true
		}

		// If a token didn't match anywhere, penalize or require match
		if !matchedToken {
			return 0.0
		}
	}

	return totalScore
}

// sortScoredPlugins sorts results according to SortBy and SortOrder.
func sortScoredPlugins(items []scoredPlugin, by pubmarket.SortBy, order pubmarket.SortOrder) {
	isDesc := (order != pubmarket.OrderAsc) // default to descending

	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		switch by {
		case pubmarket.SortByDownloads:
			if a.summary.TotalDownloads != b.summary.TotalDownloads {
				if isDesc {
					return a.summary.TotalDownloads > b.summary.TotalDownloads
				}
				return a.summary.TotalDownloads < b.summary.TotalDownloads
			}
		case pubmarket.SortByRating:
			if a.summary.AverageRating != b.summary.AverageRating {
				if isDesc {
					return a.summary.AverageRating > b.summary.AverageRating
				}
				return a.summary.AverageRating < b.summary.AverageRating
			}
			if a.summary.RatingCount != b.summary.RatingCount {
				return a.summary.RatingCount > b.summary.RatingCount
			}
		case pubmarket.SortByUpdated:
			if isDesc {
				return a.summary.UpdatedAt.After(b.summary.UpdatedAt)
			}
			return a.summary.UpdatedAt.Before(b.summary.UpdatedAt)
		case pubmarket.SortByName:
			cmp := strings.Compare(strings.ToLower(a.summary.Name), strings.ToLower(b.summary.Name))
			if cmp != 0 {
				if isDesc {
					return cmp > 0
				}
				return cmp < 0
			}
		case pubmarket.SortByRelevance:
			fallthrough
		default:
			if a.score != b.score {
				if isDesc {
					return a.score > b.score
				}
				return a.score < b.score
			}
			// Secondary sort by downloads
			if a.summary.TotalDownloads != b.summary.TotalDownloads {
				return a.summary.TotalDownloads > b.summary.TotalDownloads
			}
		}

		// Stable tie-breaker by ID
		return a.summary.ID < b.summary.ID
	})
}
