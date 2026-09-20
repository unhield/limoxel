package identity

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	// ErrInvalidToken indicates the enterprise authentication token is malformed or invalid.
	ErrInvalidToken = errors.New("enterprise identity: invalid token format or signature")
	// ErrTokenExpired indicates the token has expired.
	ErrTokenExpired = errors.New("enterprise identity: token has expired")
	// ErrTokenRevoked indicates the token has been revoked.
	ErrTokenRevoked = errors.New("enterprise identity: token has been revoked")
	// ErrPrincipalNotFound indicates the principal does not exist.
	ErrPrincipalNotFound = errors.New("enterprise identity: principal not found")
	// ErrOrganizationNotFound indicates the requested organization does not exist.
	ErrOrganizationNotFound = errors.New("enterprise identity: organization not found")
)

// Role defines the enterprise role assigned to a principal.
type Role string

const (
	RoleOwner         Role = "owner"
	RoleAdmin         Role = "admin"
	RolePluginManager Role = "plugin_manager"
	RoleSecurityAdmin Role = "security_admin"
	RoleOperator      Role = "operator"
	RoleAuditor       Role = "auditor"
	RoleMember        Role = "member"
	RoleManagedAgent  Role = "managed_agent"
)

// PrincipalType defines the category of an enterprise actor.
type PrincipalType string

const (
	PrincipalUser           PrincipalType = "user"
	PrincipalServiceAccount PrincipalType = "service_account"
	PrincipalAgent          PrincipalType = "managed_agent"
)

// Principal represents an authenticated enterprise identity.
type Principal struct {
	ID             string        `json:"id"`
	OrganizationID string        `json:"organization_id"`
	Type           PrincipalType `json:"type"`
	DisplayName    string        `json:"display_name"`
	Email          string        `json:"email,omitempty"`
	Roles          []Role        `json:"roles"`
	CreatedAt      time.Time     `json:"created_at"`
	Active         bool          `json:"active"`
}

// TokenClaims holds the payload of a cryptographically signed enterprise token.
type TokenClaims struct {
	TokenID        string    `json:"jti"`
	PrincipalID    string    `json:"sub"`
	OrganizationID string    `json:"org"`
	Roles          []Role    `json:"roles"`
	IssuedAt       time.Time `json:"iat"`
	ExpiresAt      time.Time `json:"exp"`
	Nonce          string    `json:"nonce"`
}

// TokenManager issues, validates, and revokes enterprise bearer tokens.
type TokenManager struct {
	mu           sync.RWMutex
	signingKey   []byte
	revokedToken map[string]time.Time
}

// NewTokenManager constructs a token manager with a secure signing key.
func NewTokenManager(signingKey []byte) (*TokenManager, error) {
	if len(signingKey) < 32 {
		return nil, errors.New("enterprise identity: signing key must be at least 32 bytes")
	}
	return &TokenManager{
		signingKey:   signingKey,
		revokedToken: make(map[string]time.Time),
	}, nil
}

// GenerateRandomKey generates a cryptographically strong 32-byte secret key.
func GenerateRandomKey() ([]byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}
	return b, nil
}

// IssueToken mints a cryptographically signed token for a principal.
func (m *TokenManager) IssueToken(principal *Principal, ttl time.Duration) (string, error) {
	if principal == nil {
		return "", errors.New("enterprise identity: principal is nil")
	}
	if !principal.Active {
		return "", errors.New("enterprise identity: inactive principal cannot receive tokens")
	}

	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	tokenIDBytes := make([]byte, 16)
	if _, err := rand.Read(tokenIDBytes); err != nil {
		return "", fmt.Errorf("failed to generate token ID: %w", err)
	}

	now := time.Now().UTC()
	claims := TokenClaims{
		TokenID:        hex.EncodeToString(tokenIDBytes),
		PrincipalID:    principal.ID,
		OrganizationID: principal.OrganizationID,
		Roles:          principal.Roles,
		IssuedAt:       now,
		ExpiresAt:      now.Add(ttl),
		Nonce:          hex.EncodeToString(nonceBytes),
	}

	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token claims: %w", err)
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)

	mac := hmac.New(sha256.New, m.signingKey)
	mac.Write([]byte(encodedPayload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return encodedPayload + "." + signature, nil
}

// ValidateToken parses, cryptographically verifies, and checks expiration/revocation of a token.
func (m *TokenManager) ValidateToken(tokenStr string) (*TokenClaims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 2 {
		return nil, ErrInvalidToken
	}

	encodedPayload := parts[0]
	signatureStr := parts[1]

	// Verify HMAC signature
	mac := hmac.New(sha256.New, m.signingKey)
	mac.Write([]byte(encodedPayload))
	expectedSig := mac.Sum(nil)

	providedSig, err := base64.RawURLEncoding.DecodeString(signatureStr)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if !hmac.Equal(expectedSig, providedSig) {
		return nil, ErrInvalidToken
	}

	// Decode claims
	payloadJSON, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims TokenClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	now := time.Now().UTC()
	if now.After(claims.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	// Check revocation
	m.mu.RLock()
	_, exists := m.revokedToken[claims.TokenID]
	m.mu.RUnlock()

	if exists {
		return nil, ErrTokenRevoked
	}

	return &claims, nil
}

// RevokeToken invalidates a token by its unique identifier.
func (m *TokenManager) RevokeToken(tokenID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.revokedToken[tokenID] = time.Now().UTC()
}

// PruneRevoked removes expired tokens from the revocation map.
func (m *TokenManager) PruneRevoked() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	for id, revTime := range m.revokedToken {
		if now.Sub(revTime) > 24*time.Hour {
			delete(m.revokedToken, id)
		}
	}
}
