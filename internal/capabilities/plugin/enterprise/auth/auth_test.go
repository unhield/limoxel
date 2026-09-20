package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/identity"
)

func TestRBACAuthorizer_RolesAndPermissions(t *testing.T) {
	authorizer := NewRBACAuthorizer()
	ctx := context.Background()

	admin := &identity.Principal{
		ID:             "admin-1",
		OrganizationID: "org-acme",
		Roles:          []identity.Role{identity.RoleAdmin},
		Active:         true,
	}

	member := &identity.Principal{
		ID:             "member-1",
		OrganizationID: "org-acme",
		Roles:          []identity.Role{identity.RoleMember},
		Active:         true,
	}

	secAdmin := &identity.Principal{
		ID:             "sec-1",
		OrganizationID: "org-acme",
		Roles:          []identity.Role{identity.RoleSecurityAdmin},
		Active:         true,
	}

	// Admin can manage policies
	if err := authorizer.Authorize(ctx, admin, PermPolicyManage, "org-acme"); err != nil {
		t.Errorf("admin should be allowed to manage policies, got %v", err)
	}

	// Member cannot manage policies
	if err := authorizer.Authorize(ctx, member, PermPolicyManage, "org-acme"); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("member should be denied from managing policies, got %v", err)
	}

	// Member can view monitoring
	if err := authorizer.Authorize(ctx, member, PermMonitoringView, "org-acme"); err != nil {
		t.Errorf("member should be allowed to view monitoring, got %v", err)
	}

	// Security Admin can manage security policies, but cannot publish plugins
	if err := authorizer.Authorize(ctx, secAdmin, PermSecurityPolicyManage, "org-acme"); err != nil {
		t.Errorf("security admin should be allowed to manage security policy, got %v", err)
	}
	if err := authorizer.Authorize(ctx, secAdmin, PermPluginPublish, "org-acme"); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("security admin should not publish plugins, got %v", err)
	}
}

func TestRBACAuthorizer_TenantIsolation(t *testing.T) {
	authorizer := NewRBACAuthorizer()
	ctx := context.Background()

	ownerA := &identity.Principal{
		ID:             "owner-a",
		OrganizationID: "org-a",
		Roles:          []identity.Role{identity.RoleOwner},
		Active:         true,
	}

	// Owner of Org A attempts to operate on Org B
	err := authorizer.Authorize(ctx, ownerA, PermPluginPublish, "org-b")
	if !errors.Is(err, ErrTenantMismatch) {
		t.Errorf("expected ErrTenantMismatch when operating across org boundary, got %v", err)
	}
}

func TestRBACAuthorizer_InactivePrincipal(t *testing.T) {
	authorizer := NewRBACAuthorizer()
	ctx := context.Background()

	inactiveOwner := &identity.Principal{
		ID:             "owner-inactive",
		OrganizationID: "org-a",
		Roles:          []identity.Role{identity.RoleOwner},
		Active:         false,
	}

	err := authorizer.Authorize(ctx, inactiveOwner, PermPluginPublish, "org-a")
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("inactive principal must be rejected, got %v", err)
	}
}
