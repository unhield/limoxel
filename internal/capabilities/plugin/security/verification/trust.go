package verification

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/unhield/limoxel/plugin/security"
)

var (
	// ErrKeyNotFound indicates the requested key ID is not registered in the trust store.
	ErrKeyNotFound = errors.New("public key not found in trust store")
	// ErrKeyRevoked indicates the public key has been revoked.
	ErrKeyRevoked = errors.New("public key has been revoked")
	// ErrInvalidKeyFormat indicates the public key byte representation is invalid.
	ErrInvalidKeyFormat = errors.New("invalid public key format")
)

// TrustedKey represents a registered publisher's public key in the trust store.
type TrustedKey struct {
	KeyID       string            `json:"key_id"`
	Publisher   string            `json:"publisher"`
	PublicKey   ed25519.PublicKey `json:"-"`
	AddedAt     time.Time         `json:"added_at"`
	ExpiresAt   time.Time         `json:"expires_at,omitempty"`
	Revoked     bool              `json:"revoked"`
	Description string            `json:"description,omitempty"`
}

// TrustStore maintains trusted root and publisher keys for plugin signature verification.
type TrustStore struct {
	mu   sync.RWMutex
	keys map[string]*TrustedKey
}

// NewTrustStore creates an initialized, thread-safe trust store.
func NewTrustStore() *TrustStore {
	return &TrustStore{
		keys: make(map[string]*TrustedKey),
	}
}

// RegisterKey adds a trusted Ed25519 public key to the trust store.
func (s *TrustStore) RegisterKey(publisher, keyID string, pubKey ed25519.PublicKey, desc string) error {
	if len(pubKey) != ed25519.PublicKeySize {
		return fmt.Errorf("%w: Ed25519 public key must be %d bytes", ErrInvalidKeyFormat, ed25519.PublicKeySize)
	}
	if keyID == "" {
		keyID = hex.EncodeToString(pubKey[:8])
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.keys[keyID] = &TrustedKey{
		KeyID:       keyID,
		Publisher:   publisher,
		PublicKey:   pubKey,
		AddedAt:     time.Now().UTC(),
		Revoked:     false,
		Description: desc,
	}

	return nil
}

// RevokeKey marks a registered key as revoked.
func (s *TrustStore) RevokeKey(keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	k, ok := s.keys[keyID]
	if !ok {
		return ErrKeyNotFound
	}
	k.Revoked = true
	return nil
}

// GetKey retrieves a key by its ID, checking expiration and revocation status.
func (s *TrustStore) GetKey(keyID string) (*TrustedKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	k, ok := s.keys[keyID]
	if !ok {
		return nil, ErrKeyNotFound
	}
	if k.Revoked {
		return nil, ErrKeyRevoked
	}
	if !k.ExpiresAt.IsZero() && time.Now().UTC().After(k.ExpiresAt) {
		return nil, errors.New("public key has expired")
	}

	return k, nil
}

// EvaluateTrust determines the TrustState based on signature validity, key presence, and revocation.
func (s *TrustStore) EvaluateTrust(publisher, keyID string, sigValid bool) security.TrustState {
	if !sigValid {
		return security.TrustRejected
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	k, ok := s.keys[keyID]
	if !ok {
		// Valid signature, but key not registered in our local trust store
		return security.TrustVerified
	}

	if k.Revoked {
		return security.TrustRejected
	}

	if publisher != "" && k.Publisher != "" && publisher != k.Publisher {
		// Key exists but claims mismatched publisher identity
		return security.TrustRejected
	}

	return security.TrustTrusted
}
