package community

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/plugin/marketplace/storage"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security/verification"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

var (
	// ErrInvalidPublisherID indicates an illegal publisher identifier format.
	ErrInvalidPublisherID = errors.New("community: invalid publisher ID format")
	// ErrPublisherNotFound indicates publisher does not exist.
	ErrPublisherNotFound = errors.New("community: publisher not found")
	// ErrInvalidKeyID indicates missing or malformed cryptographic key identifier.
	ErrInvalidKeyID = errors.New("community: invalid key identifier for publisher verification")

	publisherIDRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9\-_]{2,62}[a-z0-9]$`)
)

// PublisherManager administers publisher profiles, trust associations, and verification states.
type PublisherManager struct {
	mu         sync.RWMutex
	store      storage.Storage
	trustStore *verification.TrustStore
}

// NewPublisherManager creates an initialized PublisherManager.
func NewPublisherManager(store storage.Storage) *PublisherManager {
	return &PublisherManager{
		store: store,
	}
}

// SetTrustStore attaches the cryptographic trust store for publisher verification.
func (m *PublisherManager) SetTrustStore(ts *verification.TrustStore) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trustStore = ts
}

// RegisterPublisher validates and stores a publisher profile.
func (m *PublisherManager) RegisterPublisher(pub pubmarket.PublisherProfile) error {
	pub.ID = strings.ToLower(strings.TrimSpace(pub.ID))
	if !publisherIDRegex.MatchString(pub.ID) {
		return fmt.Errorf("%w: ID must be 4-64 alphanumeric chars with hyphens/underscores", ErrInvalidPublisherID)
	}

	if strings.TrimSpace(pub.DisplayName) == "" {
		pub.DisplayName = pub.ID
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.store.SavePublisher(pub)
}

// GetPublisher fetches a publisher profile by identifier.
func (m *PublisherManager) GetPublisher(id string) (*pubmarket.PublisherProfile, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return nil, ErrInvalidPublisherID
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	pub, err := m.store.GetPublisher(id)
	if err != nil {
		return nil, ErrPublisherNotFound
	}
	return pub, nil
}

// ListPublishers returns all registered publisher profiles.
func (m *PublisherManager) ListPublishers() ([]pubmarket.PublisherProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.store.ListPublishers()
}

// VerifyPublisher verifies a publisher profile by binding it to a cryptographic signing key ID.
func (m *PublisherManager) VerifyPublisher(id string, keyID string) error {
	id = strings.ToLower(strings.TrimSpace(id))
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return ErrInvalidKeyID
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	pub, err := m.store.GetPublisher(id)
	if err != nil {
		return ErrPublisherNotFound
	}

	if m.trustStore != nil {
		key, err := m.trustStore.GetKey(keyID)
		if err != nil {
			return fmt.Errorf("cryptographic verification failed: key %q not found or invalid in trust store: %w", keyID, err)
		}
		if key.Publisher != "" && !strings.EqualFold(key.Publisher, id) {
			return fmt.Errorf("cryptographic verification failed: key publisher %q does not match profile ID %q", key.Publisher, id)
		}
	}

	pub.KeyID = keyID
	pub.Verified = true

	return m.store.SavePublisher(*pub)
}
