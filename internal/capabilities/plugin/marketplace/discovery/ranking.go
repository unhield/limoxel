package discovery

import (
	"sort"
	"strings"
	"time"

	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

const (
	// MaxDiscoveryLimit establishes a hard upper bound on result sets to prevent excessive memory allocation.
	MaxDiscoveryLimit = 100

	// DefaultFeaturedLimit is the fallback limit for featured plugin discovery.
	DefaultFeaturedLimit = 10

	// DefaultTrendingLimit is the fallback limit for trending plugin discovery.
	DefaultTrendingLimit = 10

	// DefaultRecommendedLimit is the fallback limit for recommended plugin discovery.
	DefaultRecommendedLimit = 5
)

// GetFeatured retrieves plugins marked as featured, or highest rated as fallback.
func (e *Engine) GetFeatured(limit int) ([]pubmarket.PluginSummary, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if limit <= 0 {
		limit = DefaultFeaturedLimit
	} else if limit > MaxDiscoveryLimit {
		limit = MaxDiscoveryLimit
	}

	all := e.provider.ListPlugins(true)

	var featured []pubmarket.PluginSummary
	var fallbacks []pubmarket.PluginSummary

	for _, p := range all {
		if !p.Status.IsDiscoverable() {
			continue
		}
		if p.Featured {
			featured = append(featured, p)
		} else if p.AverageRating >= 4.0 && p.RatingCount > 0 {
			fallbacks = append(fallbacks, p)
		}
	}

	// Sort featured by rating descending, then downloads, then deterministic ID
	sort.Slice(featured, func(i, j int) bool {
		if featured[i].AverageRating != featured[j].AverageRating {
			return featured[i].AverageRating > featured[j].AverageRating
		}
		if featured[i].TotalDownloads != featured[j].TotalDownloads {
			return featured[i].TotalDownloads > featured[j].TotalDownloads
		}
		return featured[i].ID < featured[j].ID
	})

	if len(featured) >= limit {
		return featured[:limit], nil
	}

	// Backfill with top-rated fallback items
	sort.Slice(fallbacks, func(i, j int) bool {
		if fallbacks[i].AverageRating != fallbacks[j].AverageRating {
			return fallbacks[i].AverageRating > fallbacks[j].AverageRating
		}
		if fallbacks[i].TotalDownloads != fallbacks[j].TotalDownloads {
			return fallbacks[i].TotalDownloads > fallbacks[j].TotalDownloads
		}
		return fallbacks[i].ID < fallbacks[j].ID
	})

	for _, fb := range fallbacks {
		if len(featured) >= limit {
			break
		}
		featured = append(featured, fb)
	}

	return featured, nil
}

// GetTrending returns plugins calculated via download velocity and positive sentiment.
func (e *Engine) GetTrending(limit int) ([]pubmarket.PluginSummary, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if limit <= 0 {
		limit = DefaultTrendingLimit
	} else if limit > MaxDiscoveryLimit {
		limit = MaxDiscoveryLimit
	}

	all := e.provider.ListPlugins(true)

	type trendItem struct {
		summary pubmarket.PluginSummary
		score   float64
	}

	var candidates []trendItem

	for _, p := range all {
		if !p.Status.IsDiscoverable() {
			continue
		}

		// Trending score: download velocity (recent downloads) weighted heavily + sentiment
		var score float64
		if e.velocity != nil {
			// 7-day velocity window
			velocity := e.velocity.GetVelocity(p.ID, 7*24*time.Hour)
			score = float64(velocity)*10.0 + float64(p.TotalDownloads)*0.05
		} else {
			score = float64(p.TotalDownloads) * 0.7
		}

		if p.RatingCount > 0 {
			score += (p.AverageRating * float64(p.RatingCount)) * 0.3
		}

		candidates = append(candidates, trendItem{
			summary: p,
			score:   score,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].summary.TotalDownloads != candidates[j].summary.TotalDownloads {
			return candidates[i].summary.TotalDownloads > candidates[j].summary.TotalDownloads
		}
		return candidates[i].summary.ID < candidates[j].summary.ID
	})

	count := limit
	if len(candidates) < count {
		count = len(candidates)
	}

	results := make([]pubmarket.PluginSummary, 0)
	for i := 0; i < count; i++ {
		results = append(results, candidates[i].summary)
	}

	return results, nil
}

// GetRecommended returns related plugins for a given target plugin based on category and tags.
func (e *Engine) GetRecommended(pluginID string, limit int) ([]pubmarket.PluginSummary, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if limit <= 0 {
		limit = DefaultRecommendedLimit
	} else if limit > MaxDiscoveryLimit {
		limit = MaxDiscoveryLimit
	}

	target, err := e.provider.GetPlugin(pluginID)
	if err != nil {
		return nil, err
	}

	all := e.provider.ListPlugins(true)

	targetCats := make(map[pubmarket.Category]bool)
	for _, c := range target.Categories {
		targetCats[c] = true
	}

	targetTags := make(map[string]bool)
	for _, t := range target.Tags {
		targetTags[strings.ToLower(t)] = true
	}

	type recItem struct {
		summary pubmarket.PluginSummary
		score   float64
	}

	var scored []recItem

	for _, p := range all {
		if p.ID == pluginID || !p.Status.IsDiscoverable() {
			continue
		}

		score := 0.0

		// Shared category (+5.0 each)
		for _, cat := range p.Categories {
			if targetCats[cat] {
				score += 5.0
			}
		}

		// Shared tags (+3.0 each)
		for _, tag := range p.Tags {
			if targetTags[strings.ToLower(tag)] {
				score += 3.0
			}
		}

		// Same publisher (+2.0)
		if p.PublisherID == target.PublisherID {
			score += 2.0
		}

		// Quality boost from ratings
		if p.AverageRating >= 4.0 {
			score += 1.5
		}

		if score > 0 {
			scored = append(scored, recItem{summary: p, score: score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if scored[i].summary.TotalDownloads != scored[j].summary.TotalDownloads {
			return scored[i].summary.TotalDownloads > scored[j].summary.TotalDownloads
		}
		return scored[i].summary.ID < scored[j].summary.ID
	})

	count := limit
	if len(scored) < count {
		count = len(scored)
	}

	results := make([]pubmarket.PluginSummary, 0)
	for i := 0; i < count; i++ {
		results = append(results, scored[i].summary)
	}

	return results, nil
}
