package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/identity"
)

var (
	// ErrUnauthorized indicates the actor lacks permission to perform the requested operation.
	ErrUnauthorized = errors.New("enterprise auth: principal is unauthorized for this action")
	// ErrTenantMismatch indicates an operation attempted to cross tenant organization boundaries.
	ErrTenantMismatch = errors.New("enterprise auth: organization boundary violation / tenant mismatch")
)

// Permission defines a specific enterprise action.
type Permission string

const (
	PermOrgRegister           Permission = "org:register"
	PermOrgManage             Permission = "org:manage"
	PermPluginCreate          Permission = "plugin:create"
	PermPluginPublish         Permission = "plugin:publish"
	PermPluginMetadataModify  Permission = "plugin:modify_metadata"
	PermPluginOwnershipManage Permission = "plugin:manage_ownership"
	PermPluginPromote         Permission = "plugin:promote"
	PermPluginInstall         Permission = "plugin:install"
	PermPluginUpdate          Permission = "plugin:update"
	PermPluginRemove          Permission = "plugin:remove"
	PermPolicyManage          Permission = "policy:manage"
	PermSecurityPolicyManage  Permission = "policy:manage_security"
	PermDeploymentManage      Permission = "deployment:manage"
	PermMonitoringView        Permission = "monitoring:view"
	PermAuditView             Permission = "audit:view"
	PermEmergencyAction       Permission = "emergency:action"
)

// Authorizer validates permissions and tenant boundaries.
type Authorizer interface {
	Authorize(ctx context.Context, principal *identity.Principal, action Permission, targetOrgID string) error
	HasPermission(role identity.Role, action Permission) bool
}

// RBACAuthorizer implements role-based access control with organization isolation.
type RBACAuthorizer struct {
	rolePermissions map[identity.Role]map[Permission]bool
}

// NewRBACAuthorizer constructs an enterprise RBAC authorization engine.
func NewRBACAuthorizer() *RBACAuthorizer {
	auth := &RBACAuthorizer{
		rolePermissions: make(map[identity.Role]map[Permission]bool),
	}
	auth.initDefaultPermissions()
	return auth
}

func (a *RBACAuthorizer) initDefaultPermissions() {
	allPerms := []Permission{
		PermOrgRegister, PermOrgManage, PermPluginCreate, PermPluginPublish,
		PermPluginMetadataModify, PermPluginOwnershipManage, PermPluginPromote,
		PermPluginInstall, PermPluginUpdate, PermPluginRemove,
		PermPolicyManage, PermSecurityPolicyManage, PermDeploymentManage,
		PermMonitoringView, PermAuditView, PermEmergencyAction,
	}

	// Owner has all permissions
	a.rolePermissions[identity.RoleOwner] = make(map[Permission]bool)
	for _, p := range allPerms {
		a.rolePermissions[identity.RoleOwner][p] = true
	}

	// Admin
	a.rolePermissions[identity.RoleAdmin] = map[Permission]bool{
		PermOrgManage:             true,
		PermPluginCreate:          true,
		PermPluginPublish:         true,
		PermPluginMetadataModify:  true,
		PermPluginOwnershipManage: true,
		PermPluginPromote:         true,
		PermPluginInstall:         true,
		PermPluginUpdate:          true,
		PermPluginRemove:          true,
		PermPolicyManage:          true,
		PermDeploymentManage:      true,
		PermMonitoringView:        true,
		PermAuditView:             true,
	}

	// Security Admin
	a.rolePermissions[identity.RoleSecurityAdmin] = map[Permission]bool{
		PermPolicyManage:         true,
		PermSecurityPolicyManage: true,
		PermPluginPromote:        true,
		PermMonitoringView:       true,
		PermAuditView:            true,
		PermEmergencyAction:      true,
	}

	// Plugin Manager
	a.rolePermissions[identity.RolePluginManager] = map[Permission]bool{
		PermPluginCreate:          true,
		PermPluginPublish:         true,
		PermPluginMetadataModify:  true,
		PermPluginOwnershipManage: true,
		PermPluginPromote:         true,
		PermMonitoringView:        true,
	}

	// Operator
	a.rolePermissions[identity.RoleOperator] = map[Permission]bool{
		PermPluginInstall:    true,
		PermPluginUpdate:     true,
		PermPluginRemove:     true,
		PermDeploymentManage: true,
		PermMonitoringView:   true,
		PermEmergencyAction:  true,
	}

	// Auditor
	a.rolePermissions[identity.RoleAuditor] = map[Permission]bool{
		PermMonitoringView: true,
		PermAuditView:      true,
	}

	// Member
	a.rolePermissions[identity.RoleMember] = map[Permission]bool{
		PermMonitoringView: true,
	}

	// Managed Agent
	a.rolePermissions[identity.RoleManagedAgent] = map[Permission]bool{
		PermMonitoringView: true,
	}
}

// HasPermission checks if a single role possesses the given permission.
func (a *RBACAuthorizer) HasPermission(role identity.Role, action Permission) bool {
	perms, ok := a.rolePermissions[role]
	if !ok {
		return false
	}
	return perms[action]
}

// Authorize checks whether the principal possesses the required permission and belongs to the target organization.
func (a *RBACAuthorizer) Authorize(ctx context.Context, principal *identity.Principal, action Permission, targetOrgID string) error {
	if principal == nil {
		return errors.New("enterprise auth: principal is nil")
	}
	if !principal.Active {
		return fmt.Errorf("%w: principal is disabled", ErrUnauthorized)
	}

	// Enforce strict multi-tenant organization boundary
	if targetOrgID != "" && principal.OrganizationID != targetOrgID {
		return fmt.Errorf("%w: principal org %q cannot operate on org %q", ErrTenantMismatch, principal.OrganizationID, targetOrgID)
	}

	// Check if any role of the principal grants the permission
	for _, role := range principal.Roles {
		if a.HasPermission(role, action) {
			return nil
		}
	}

	return fmt.Errorf("%w: principal %q lacking permission %q", ErrUnauthorized, principal.ID, action)
}
