package enterprise

import (
	"errors"
	"time"
)

// Common public enterprise errors.
var (
	ErrUnauthorized         = errors.New("enterprise: unauthorized action")
	ErrTenantMismatch       = errors.New("enterprise: organization tenant boundary mismatch")
	ErrOrganizationNotFound = errors.New("enterprise: organization not found")
	ErrPluginNotFound       = errors.New("enterprise: plugin not found in organization registry")
	ErrPolicyViolation      = errors.New("enterprise: operation blocked by organization policy")
	ErrReplayDetected       = errors.New("enterprise: command replay or stale timestamp detected")
	ErrIntegrityMismatch    = errors.New("enterprise: artifact integrity checksum mismatch")
	ErrInvalidCommand       = errors.New("enterprise: invalid remote command specification")
)

// Role defines the role assigned to an enterprise principal.
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

// OrganizationSummary represents basic organization details.
type OrganizationSummary struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// MemberSummary represents an organization member.
type MemberSummary struct {
	PrincipalID    string    `json:"principal_id"`
	OrganizationID string    `json:"organization_id"`
	Roles          []Role    `json:"roles"`
	JoinedAt       time.Time `json:"joined_at"`
}

// PrivateVersionSummary summarizes a private or promoted version in an organization registry.
type PrivateVersionSummary struct {
	PluginID         string    `json:"plugin_id"`
	Version          string    `json:"version"`
	ArtifactDigest   string    `json:"artifact_digest"`
	ArtifactSize     int64     `json:"artifact_size"`
	CreatedAt        time.Time `json:"created_at"`
	IsPromotedPublic bool      `json:"is_promoted_public"`
	DownloadCount    int64     `json:"download_count"`
}

// DeploymentTargetSummary summarizes a managed execution host or cluster.
type DeploymentTargetSummary struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Environment    string    `json:"environment"`
	Platform       string    `json:"platform"`
	RegisteredAt   time.Time `json:"registered_at"`
	Active         bool      `json:"active"`
}

// CommandResult conveys the outcome of a remote deployment operation.
type CommandResult struct {
	CommandID    string    `json:"command_id"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`
	ExecutedAt   time.Time `json:"executed_at"`
	RollbackDone bool      `json:"rollback_done,omitempty"`
}

// MonitoredStatus conveys desired vs actual states and health.
type MonitoredStatus struct {
	OrganizationID   string    `json:"organization_id"`
	DeploymentTarget string    `json:"deployment_target"`
	PluginID         string    `json:"plugin_id"`
	DesiredVersion   string    `json:"desired_version"`
	ActualVersion    string    `json:"actual_version"`
	Health           string    `json:"health"`
	HasDiscrepancy   bool      `json:"has_discrepancy"`
	DiscrepancyNotes string    `json:"discrepancy_notes,omitempty"`
	LastObservedAt   time.Time `json:"last_observed_at"`
}

// AuditSummary conveys an entry from the tamper-evident audit log.
type AuditSummary struct {
	Sequence       int64     `json:"sequence"`
	Timestamp      time.Time `json:"timestamp"`
	OrganizationID string    `json:"organization_id"`
	ActorID        string    `json:"actor_id"`
	Action         string    `json:"action"`
	Target         string    `json:"target"`
	Outcome        string    `json:"outcome"`
	PrevHash       string    `json:"prev_hash"`
	Hash           string    `json:"hash"`
}
