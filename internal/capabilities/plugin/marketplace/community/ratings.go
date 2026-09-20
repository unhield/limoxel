package community

import (
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/plugin/marketplace/storage"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

var (
	// ErrInvalidRating indicates the star value is outside the 1-5 range.
	ErrInvalidRating = errors.New("community: rating must be between 1 and 5 stars")
	// ErrEmptyAuthor indicates missing author identifier.
	ErrEmptyAuthor = errors.New("community: author ID must not be empty")
	// ErrEmptyPluginID indicates missing plugin identifier.
	ErrEmptyPluginID = errors.New("community: plugin ID must not be empty")
)

// CatalogRatingUpdater allows community components to notify the catalog of updated aggregates.
type CatalogRatingUpdater interface {
	UpdateRating(pluginID string, rating pubmarket.RatingSummary) error
}

// RatingManager manages user ratings and computes aggregate rating summaries.
type RatingManager struct {
	mu      sync.RWMutex
	store   storage.Storage
	updater CatalogRatingUpdater
}

// NewRatingManager creates an initialized RatingManager.
func NewRatingManager(store storage.Storage, updater CatalogRatingUpdater) *RatingManager {
	return &RatingManager{
		store:   store,
		updater: updater,
	}
}

// RecordRating records a user rating (1-5 stars) and updates aggregate metrics.
// One vote per user: subsequent calls by the same author update their existing rating.
func (m *RatingManager) RecordRating(pluginID, authorID string, stars int) (*pubmarket.RatingSummary, error) {
	if pluginID == "" {
		return nil, ErrEmptyPluginID
	}
	if authorID == "" {
		return nil, ErrEmptyAuthor
	}
	if stars < 1 || stars > 5 {
		return nil, ErrInvalidRating
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	ratings, err := m.store.GetRatings(pluginID)
	if err != nil {
		ratings = make(map[string]int)
	}

	// Record or update author rating
	ratings[authorID] = stars

	if err := m.store.SaveRatings(pluginID, ratings); err != nil {
		return nil, fmt.Errorf("failed to save ratings: %w", err)
	}

	summary := CalculateRatingSummary(ratings)

	if m.updater != nil {
		_ = m.updater.UpdateRating(pluginID, summary)
	}

	return &summary, nil
}

// GetRatingSummary retrieves aggregate rating metrics for a plugin.
func (m *RatingManager) GetRatingSummary(pluginID string) (*pubmarket.RatingSummary, error) {
	if pluginID == "" {
		return nil, ErrEmptyPluginID
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	ratings, err := m.store.GetRatings(pluginID)
	if err != nil {
		return &pubmarket.RatingSummary{
			Average: 0.0,
			Count:   0,
			Stars:   map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0},
		}, nil
	}

	summary := CalculateRatingSummary(ratings)
	return &summary, nil
}

// CalculateRatingSummary computes arithmetic average and star breakdowns.
func CalculateRatingSummary(ratings map[string]int) pubmarket.RatingSummary {
	stars := map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0}
	if len(ratings) == 0 {
		return pubmarket.RatingSummary{
			Average: 0.0,
			Count:   0,
			Stars:   stars,
		}
	}

	total := 0
	for _, val := range ratings {
		if val >= 1 && val <= 5 {
			stars[val]++
			total += val
		}
	}

	count := len(ratings)
	avg := float64(total) / float64(count)
	// Round to 1 decimal place
	rounded := math.Round(avg*10) / 10

	return pubmarket.RatingSummary{
		Average: rounded,
		Count:   count,
		Stars:   stars,
	}
}
