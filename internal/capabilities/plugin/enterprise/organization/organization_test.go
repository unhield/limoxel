package organization

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/identity"
)

func TestOrganizationManager_Lifecycle(t *testing.T) {
	tempDir := t.TempDir()
	mgr, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()

	// 1. Create Organization
	org, err := mgr.CreateOrganization(ctx, "org-acme", "Acme Corp", "Main engineering org", "owner-1")
	if err != nil {
		t.Fatalf("CreateOrganization failed: %v", err)
	}
	if org.ID != "org-acme" || org.Status != OrgStatusActive {
		t.Errorf("unexpected org: %+v", org)
	}

	// 2. Prevent duplicate organization
	_, err = mgr.CreateOrganization(ctx, "org-acme", "Acme Duplicate", "", "owner-1")
	if !errors.Is(err, ErrOrgExists) {
		t.Errorf("expected ErrOrgExists, got %v", err)
	}

	// 3. Add Member
	mem, err := mgr.AddMember(ctx, "org-acme", "alice-1", []identity.Role{identity.RolePluginManager})
	if err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}
	if mem.PrincipalID != "alice-1" || len(mem.Roles) != 1 || mem.Roles[0] != identity.RolePluginManager {
		t.Errorf("unexpected member: %+v", mem)
	}

	// 4. Register Plugin Ownership
	ownership, err := mgr.RegisterPluginOwnership(ctx, "acme.auth", "org-acme", "alice-1", []string{"bob-1"})
	if err != nil {
		t.Fatalf("RegisterPluginOwnership failed: %v", err)
	}
	if ownership.PrimaryOwnerID != "alice-1" {
		t.Errorf("expected primary owner alice-1, got %s", ownership.PrimaryOwnerID)
	}

	// 5. Transfer Ownership (add bob-1 as member first)
	_, err = mgr.AddMember(ctx, "org-acme", "bob-1", []identity.Role{identity.RolePluginManager})
	if err != nil {
		t.Fatalf("AddMember bob-1 failed: %v", err)
	}

	transferred, err := mgr.TransferPluginOwnership(ctx, "org-acme", "acme.auth", "alice-1", "bob-1", "Handoff to new tech lead")
	if err != nil {
		t.Fatalf("TransferPluginOwnership failed: %v", err)
	}
	if transferred.PrimaryOwnerID != "bob-1" {
		t.Errorf("expected primary owner bob-1, got %s", transferred.PrimaryOwnerID)
	}
	if len(transferred.TransferHistory) != 1 {
		t.Fatalf("expected 1 transfer history record, got %d", len(transferred.TransferHistory))
	}
	if transferred.TransferHistory[0].FromOwnerID != "alice-1" || transferred.TransferHistory[0].ToOwnerID != "bob-1" {
		t.Errorf("unexpected transfer record: %+v", transferred.TransferHistory[0])
	}

	// 6. Test Persistence & Reload
	mgrReloaded, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("reloading NewManager failed: %v", err)
	}

	loadedOrg, err := mgrReloaded.GetOrganization(ctx, "org-acme")
	if err != nil {
		t.Fatalf("reloaded GetOrganization failed: %v", err)
	}
	if loadedOrg.Name != "Acme Corp" {
		t.Errorf("expected Acme Corp, got %s", loadedOrg.Name)
	}

	loadedOwnership, err := mgrReloaded.GetPluginOwnership(ctx, "org-acme", "acme.auth")
	if err != nil {
		t.Fatalf("reloaded GetPluginOwnership failed: %v", err)
	}
	if loadedOwnership.PrimaryOwnerID != "bob-1" {
		t.Errorf("expected primary owner bob-1 after reload, got %s", loadedOwnership.PrimaryOwnerID)
	}
}

func TestOrganizationManager_UnauthorizedTransfer(t *testing.T) {
	tempDir := filepath.Join(t.TempDir(), "orgs")
	mgr, _ := NewManager(tempDir)
	ctx := context.Background()

	_, _ = mgr.CreateOrganization(ctx, "org-acme", "Acme Corp", "", "owner-1")
	_, _ = mgr.AddMember(ctx, "org-acme", "mallory-1", []identity.Role{identity.RoleMember})
	_, _ = mgr.RegisterPluginOwnership(ctx, "acme.auth", "org-acme", "owner-1", nil)

	// Mallory (regular member) attempts to transfer ownership
	_, err := mgr.TransferPluginOwnership(ctx, "org-acme", "acme.auth", "mallory-1", "mallory-1", "takeover attempt")
	if !errors.Is(err, ErrUnauthorizedTransfer) {
		t.Errorf("expected ErrUnauthorizedTransfer, got %v", err)
	}
}

func TestOrganizationManager_ValidationAndMembershipBoundaries(t *testing.T) {
	tempDir := filepath.Join(t.TempDir(), "orgs-boundary")
	mgr, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	ctx := context.Background()

	// 1. Organization ID validation checks
	invalidIDs := []string{
		"",
		"a",
		"ab",
		"abc",
		"../traversal",
		"org/nested",
		"org\\escaped",
		"org@invalid",
		"org:name",
	}
	for _, id := range invalidIDs {
		if err := ValidateOrganizationID(id); !errors.Is(err, ErrInvalidOrgID) {
			t.Errorf("expected ErrInvalidOrgID for '%s', got %v", id, err)
		}
		if _, err := mgr.CreateOrganization(ctx, id, "Test Org", "", "owner-1"); !errors.Is(err, ErrInvalidOrgID) {
			t.Errorf("expected CreateOrganization to fail for '%s', got %v", id, err)
		}
	}

	// 2. Valid organization creation
	_, err = mgr.CreateOrganization(ctx, "org-valid", "Valid Org", "", "owner-1")
	if err != nil {
		t.Fatalf("CreateOrganization failed: %v", err)
	}

	// 3. Register ownership with non-member owner must fail with ErrMemberNotFound
	_, err = mgr.RegisterPluginOwnership(ctx, "plugin-1", "org-valid", "non-member-user", nil)
	if !errors.Is(err, ErrMemberNotFound) {
		t.Errorf("expected ErrMemberNotFound for non-member initial owner, got %v", err)
	}

	// 4. Register with actual member succeeds
	_, err = mgr.RegisterPluginOwnership(ctx, "plugin-1", "org-valid", "owner-1", nil)
	if err != nil {
		t.Fatalf("RegisterPluginOwnership failed: %v", err)
	}

	// 5. Transfer ownership to non-member must fail with ErrMemberNotFound
	_, err = mgr.TransferPluginOwnership(ctx, "org-valid", "plugin-1", "owner-1", "non-existent-user", "handover")
	if !errors.Is(err, ErrMemberNotFound) {
		t.Errorf("expected ErrMemberNotFound for non-member recipient, got %v", err)
	}
}

func TestOrganizationManager_MultiTenantOwnershipIsolation(t *testing.T) {
	tempDir := filepath.Join(t.TempDir(), "orgs-multitenant")
	mgr, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	ctx := context.Background()

	// Create Org A and Org B
	_, err = mgr.CreateOrganization(ctx, "org-alpha", "Alpha Corp", "", "owner-alpha")
	if err != nil {
		t.Fatalf("Create Org Alpha failed: %v", err)
	}
	_, err = mgr.CreateOrganization(ctx, "org-bravo", "Bravo Corp", "", "owner-bravo")
	if err != nil {
		t.Fatalf("Create Org Bravo failed: %v", err)
	}

	// Both register private plugin with the EXACT SAME pluginID "corp.analytics"
	commonPluginID := "corp.analytics"
	ownAlpha, err := mgr.RegisterPluginOwnership(ctx, commonPluginID, "org-alpha", "owner-alpha", nil)
	if err != nil {
		t.Fatalf("Org Alpha RegisterPluginOwnership failed: %v", err)
	}

	// Org Bravo registration must SUCCEED without collision
	ownBravo, err := mgr.RegisterPluginOwnership(ctx, commonPluginID, "org-bravo", "owner-bravo", nil)
	if err != nil {
		t.Fatalf("Org Bravo RegisterPluginOwnership failed with collision: %v", err)
	}

	// Verify isolated owners
	if ownAlpha.PrimaryOwnerID != "owner-alpha" || ownAlpha.OrganizationID != "org-alpha" {
		t.Errorf("unexpected ownAlpha: %+v", ownAlpha)
	}
	if ownBravo.PrimaryOwnerID != "owner-bravo" || ownBravo.OrganizationID != "org-bravo" {
		t.Errorf("unexpected ownBravo: %+v", ownBravo)
	}

	// Query each org's ownership independently
	gotAlpha, err := mgr.GetPluginOwnership(ctx, "org-alpha", commonPluginID)
	if err != nil || gotAlpha.PrimaryOwnerID != "owner-alpha" {
		t.Errorf("GetPluginOwnership org-alpha mismatch: %+v, err: %v", gotAlpha, err)
	}

	gotBravo, err := mgr.GetPluginOwnership(ctx, "org-bravo", commonPluginID)
	if err != nil || gotBravo.PrimaryOwnerID != "owner-bravo" {
		t.Errorf("GetPluginOwnership org-bravo mismatch: %+v, err: %v", gotBravo, err)
	}

	// Org Alpha duplicate within same org must still fail
	_, err = mgr.RegisterPluginOwnership(ctx, commonPluginID, "org-alpha", "owner-alpha", nil)
	if !errors.Is(err, ErrPluginOwnershipExists) {
		t.Errorf("expected ErrPluginOwnershipExists for intra-tenant duplicate, got %v", err)
	}
}
