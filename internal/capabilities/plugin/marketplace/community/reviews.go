package community

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/marketplace/storage"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

var (
	// ErrInvalidReview indicates invalid review content or rating.
	ErrInvalidReview = errors.New("community: invalid review data")
	// ErrReviewTooLong indicates title or comment exceeds length limit.
	ErrReviewTooLong = errors.New("community: review text exceeds maximum permitted length")
)

const (
	maxTitleLength   = 200
	maxCommentLength = 5000
)

// ReviewManager handles community reviews, persistence, and sanitization.
type ReviewManager struct {
	mu      sync.RWMutex
	store   storage.Storage
	ratings *RatingManager
}

// NewReviewManager constructs a ReviewManager.
func NewReviewManager(store storage.Storage, ratings *RatingManager) *ReviewManager {
	return &ReviewManager{
		store:   store,
		ratings: ratings,
	}
}

// SubmitReview validates, sanitizes, and records a community review.
func (m *ReviewManager) SubmitReview(rev pubmarket.Review) error {
	if rev.PluginID == "" || rev.AuthorID == "" {
		return fmt.Errorf("%w: plugin_id and author_id are required", ErrInvalidReview)
	}
	if rev.Rating < 1 || rev.Rating > 5 {
		return fmt.Errorf("%w: rating must be 1 to 5 stars", ErrInvalidRating)
	}

	rev.Title = sanitizeText(rev.Title)
	rev.Comment = sanitizeText(rev.Comment)

	if len(rev.Title) > maxTitleLength || len(rev.Comment) > maxCommentLength {
		return ErrReviewTooLong
	}

	if rev.ID == "" {
		rev.ID = generateReviewID()
	}

	now := time.Now().UTC()
	if rev.CreatedAt.IsZero() {
		rev.CreatedAt = now
	}
	rev.UpdatedAt = now

	m.mu.Lock()
	defer m.mu.Unlock()

	existing, err := m.store.GetReviews(rev.PluginID)
	if err != nil {
		existing = []pubmarket.Review{}
	}

	// If author already reviewed this version, update it; otherwise append
	updated := false
	for i, r := range existing {
		if r.AuthorID == rev.AuthorID && r.Version == rev.Version {
			rev.ID = r.ID
			rev.CreatedAt = r.CreatedAt
			existing[i] = rev
			updated = true
			break
		}
	}

	if !updated {
		existing = append(existing, rev)
	}

	if err := m.store.SaveReviews(rev.PluginID, existing); err != nil {
		return fmt.Errorf("failed to persist reviews: %w", err)
	}

	// Also record the star rating in the ratings manager
	if m.ratings != nil {
		_, _ = m.ratings.RecordRating(rev.PluginID, rev.AuthorID, rev.Rating)
	}

	return nil
}

// GetReviews retrieves all recorded reviews for a plugin.
func (m *ReviewManager) GetReviews(pluginID string) ([]pubmarket.Review, error) {
	if pluginID == "" {
		return nil, ErrEmptyPluginID
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	reviews, err := m.store.GetReviews(pluginID)
	if err != nil {
		return []pubmarket.Review{}, nil
	}

	return reviews, nil
}

// sanitizeText strips control characters and trims excess whitespace.
func sanitizeText(s string) string {
	var b strings.Builder
	for _, r := range s {
		// Allow standard printable characters and common whitespace (space, newline, tab)
		if r >= 32 || r == '\n' || r == '\r' || r == '\t' {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// generateReviewID generates a unique hex review identifier.
func generateReviewID() string {
	buf := make([]byte, 12)
	_, _ = rand.Read(buf)
	return fmt.Sprintf("rev-%s", hex.EncodeToString(buf))
}
