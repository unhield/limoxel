package identity

import (
	"testing"
	"time"
)

func TestTokenManager_IssueAndValidate(t *testing.T) {
	key, err := GenerateRandomKey()
	if err != nil {
		t.Fatalf("GenerateRandomKey failed: %v", err)
	}

	mgr, err := NewTokenManager(key)
	if err != nil {
		t.Fatalf("NewTokenManager failed: %v", err)
	}

	p := &Principal{
		ID:             "user-123",
		OrganizationID: "org-acme",
		Type:           PrincipalUser,
		DisplayName:    "Alice Admin",
		Email:          "alice@acme.com",
		Roles:          []Role{RoleAdmin, RolePluginManager},
		CreatedAt:      time.Now().UTC(),
		Active:         true,
	}

	token, err := mgr.IssueToken(p, 1*time.Hour)
	if err != nil {
		t.Fatalf("IssueToken failed: %v", err)
	}

	claims, err := mgr.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.PrincipalID != p.ID {
		t.Errorf("expected principal ID %s, got %s", p.ID, claims.PrincipalID)
	}
	if claims.OrganizationID != p.OrganizationID {
		t.Errorf("expected org ID %s, got %s", p.OrganizationID, claims.OrganizationID)
	}
	if len(claims.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(claims.Roles))
	}
	if claims.Roles[0] != RoleAdmin || claims.Roles[1] != RolePluginManager {
		t.Errorf("unexpected roles: %v", claims.Roles)
	}
}

func TestTokenManager_TamperedToken(t *testing.T) {
	key, _ := GenerateRandomKey()
	mgr, _ := NewTokenManager(key)

	p := &Principal{
		ID:             "user-123",
		OrganizationID: "org-acme",
		Type:           PrincipalUser,
		Active:         true,
	}

	token, _ := mgr.IssueToken(p, 1*time.Hour)
	tampered := token[:len(token)-5] + "XXXXX"

	_, err := mgr.ValidateToken(tampered)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken on tampered token, got %v", err)
	}
}

func TestTokenManager_ExpiredToken(t *testing.T) {
	key, _ := GenerateRandomKey()
	mgr, _ := NewTokenManager(key)

	p := &Principal{
		ID:             "user-123",
		OrganizationID: "org-acme",
		Type:           PrincipalUser,
		Active:         true,
	}

	// Issue token with negative TTL
	token, _ := mgr.IssueToken(p, -1*time.Minute)

	_, err := mgr.ValidateToken(token)
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestTokenManager_Revocation(t *testing.T) {
	key, _ := GenerateRandomKey()
	mgr, _ := NewTokenManager(key)

	p := &Principal{
		ID:             "user-123",
		OrganizationID: "org-acme",
		Type:           PrincipalUser,
		Active:         true,
	}

	token, _ := mgr.IssueToken(p, 1*time.Hour)
	claims, err := mgr.ValidateToken(token)
	if err != nil {
		t.Fatalf("initial validation failed: %v", err)
	}

	mgr.RevokeToken(claims.TokenID)

	_, err = mgr.ValidateToken(token)
	if err != ErrTokenRevoked {
		t.Errorf("expected ErrTokenRevoked, got %v", err)
	}
}
