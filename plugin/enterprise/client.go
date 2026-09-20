package enterprise

import (
	"context"
	"io"

	intenterprise "github.com/unhield/limoxel/internal/capabilities/plugin/enterprise"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/identity"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

// Client defines the public enterprise developer and administrator interface.
type Client interface {
	RegisterOrganization(ctx context.Context, id, name, desc string) (*OrganizationSummary, error)
	AddMember(ctx context.Context, orgID, principalID string, roles []Role) (*MemberSummary, error)
	PublishPrivatePlugin(ctx context.Context, pluginID, name, desc string, ver version.SemVer, manifest *model.Manifest, archiveData []byte) (*PrivateVersionSummary, error)
	PromotePublicPlugin(ctx context.Context, publicPlugin *pubmarket.PluginDetail, publicVersion *pubmarket.VersionInfo, archiveReader io.Reader, notes string, tags []string) (*PrivateVersionSummary, error)
	DeployPlugin(ctx context.Context, targetID, pluginID string, ver version.SemVer) (CommandResult, error)
	UpdatePlugin(ctx context.Context, targetID, pluginID string, newVer version.SemVer) (CommandResult, error)
	RemovePlugin(ctx context.Context, targetID, pluginID string, purgeData bool) (CommandResult, error)
	DownloadPrivateArtifact(ctx context.Context, orgID, pluginID string, ver version.SemVer) (io.ReadCloser, *PrivateVersionSummary, error)
	GetMonitoringStatus(ctx context.Context, orgID, targetID, pluginID string) (*MonitoredStatus, error)
	ListAuditEvents(ctx context.Context) ([]AuditSummary, error)
}

// LocalClient provides an in-process client implementation bound to an authenticated principal.
type LocalClient struct {
	svc       *intenterprise.Service
	principal *identity.Principal
}

// NewLocalClient constructs a LocalClient.
func NewLocalClient(svc *intenterprise.Service, principal *identity.Principal) *LocalClient {
	return &LocalClient{
		svc:       svc,
		principal: principal,
	}
}

// RegisterOrganization registers a new enterprise tenant organization.
func (c *LocalClient) RegisterOrganization(ctx context.Context, id, name, desc string) (*OrganizationSummary, error) {
	if c.principal == nil {
		return nil, ErrUnauthorized
	}
	org, err := c.svc.RegisterOrganization(ctx, id, name, desc, c.principal.ID)
	if err != nil {
		return nil, err
	}
	return &OrganizationSummary{
		ID:          org.ID,
		Name:        org.Name,
		Description: org.Description,
		Status:      string(org.Status),
		CreatedAt:   org.CreatedAt,
	}, nil
}

// AddMember adds an enterprise member with authorization and persistent state recording.
func (c *LocalClient) AddMember(ctx context.Context, orgID, principalID string, roles []Role) (*MemberSummary, error) {
	var intRoles []identity.Role
	for _, r := range roles {
		intRoles = append(intRoles, identity.Role(r))
	}
	mem, err := c.svc.AddMember(ctx, c.principal, orgID, principalID, intRoles)
	if err != nil {
		return nil, err
	}
	var pubRoles []Role
	for _, r := range mem.Roles {
		pubRoles = append(pubRoles, Role(r))
	}
	return &MemberSummary{
		PrincipalID:    mem.PrincipalID,
		OrganizationID: mem.OrganizationID,
		Roles:          pubRoles,
		JoinedAt:       mem.JoinedAt,
	}, nil
}

// PublishPrivatePlugin publishes a private plugin.
func (c *LocalClient) PublishPrivatePlugin(
	ctx context.Context,
	pluginID, name, desc string,
	ver version.SemVer,
	manifest *model.Manifest,
	archiveData []byte,
) (*PrivateVersionSummary, error) {
	entry, _, err := c.svc.PublishPrivatePlugin(ctx, c.principal, pluginID, name, desc, ver, manifest, archiveData)
	if err != nil {
		return nil, err
	}
	return &PrivateVersionSummary{
		PluginID:         entry.PluginID,
		Version:          entry.Version,
		ArtifactDigest:   entry.ArtifactDigest,
		ArtifactSize:     entry.ArtifactSize,
		CreatedAt:        entry.CreatedAt,
		IsPromotedPublic: entry.IsPromotedPublic,
		DownloadCount:    entry.DownloadCount,
	}, nil
}

// PromotePublicPlugin approves and mirrors a public plugin.
func (c *LocalClient) PromotePublicPlugin(
	ctx context.Context,
	publicPlugin *pubmarket.PluginDetail,
	publicVersion *pubmarket.VersionInfo,
	archiveReader io.Reader,
	notes string,
	tags []string,
) (*PrivateVersionSummary, error) {
	entry, err := c.svc.PromotePublicPlugin(ctx, c.principal, publicPlugin, publicVersion, archiveReader, notes, tags)
	if err != nil {
		return nil, err
	}
	return &PrivateVersionSummary{
		PluginID:         entry.PluginID,
		Version:          entry.Version,
		ArtifactDigest:   entry.ArtifactDigest,
		ArtifactSize:     entry.ArtifactSize,
		CreatedAt:        entry.CreatedAt,
		IsPromotedPublic: entry.IsPromotedPublic,
		DownloadCount:    entry.DownloadCount,
	}, nil
}

// DeployPlugin deploys a plugin to a target machine.
func (c *LocalClient) DeployPlugin(ctx context.Context, targetID, pluginID string, ver version.SemVer) (CommandResult, error) {
	res, err := c.svc.DeployPlugin(ctx, c.principal, targetID, pluginID, ver)
	return CommandResult{
		CommandID:    res.CommandID,
		Success:      res.Success,
		ErrorMessage: res.ErrorMessage,
		ExecutedAt:   res.ExecutedAt,
		RollbackDone: res.RollbackDone,
	}, err
}

// UpdatePlugin updates a deployed plugin to a new target version with automatic rollback.
func (c *LocalClient) UpdatePlugin(ctx context.Context, targetID, pluginID string, newVer version.SemVer) (CommandResult, error) {
	res, err := c.svc.UpdatePlugin(ctx, c.principal, targetID, pluginID, newVer)
	return CommandResult{
		CommandID:    res.CommandID,
		Success:      res.Success,
		ErrorMessage: res.ErrorMessage,
		ExecutedAt:   res.ExecutedAt,
		RollbackDone: res.RollbackDone,
	}, err
}

// RemovePlugin remotely removes a deployed plugin.
func (c *LocalClient) RemovePlugin(ctx context.Context, targetID, pluginID string, purgeData bool) (CommandResult, error) {
	res, err := c.svc.RemovePlugin(ctx, c.principal, targetID, pluginID, purgeData)
	return CommandResult{
		CommandID:    res.CommandID,
		Success:      res.Success,
		ErrorMessage: res.ErrorMessage,
		ExecutedAt:   res.ExecutedAt,
	}, err
}

// DownloadPrivateArtifact streams private plugin bytes.
func (c *LocalClient) DownloadPrivateArtifact(ctx context.Context, orgID, pluginID string, ver version.SemVer) (io.ReadCloser, *PrivateVersionSummary, error) {
	reader, entry, err := c.svc.DownloadPrivateArtifact(ctx, c.principal, orgID, pluginID, ver)
	if err != nil {
		return nil, nil, err
	}
	summary := &PrivateVersionSummary{
		PluginID:         entry.PluginID,
		Version:          entry.Version,
		ArtifactDigest:   entry.ArtifactDigest,
		ArtifactSize:     entry.ArtifactSize,
		CreatedAt:        entry.CreatedAt,
		IsPromotedPublic: entry.IsPromotedPublic,
		DownloadCount:    entry.DownloadCount,
	}
	return reader, summary, nil
}

// GetMonitoringStatus retrieves telemetry and health for a deployment target with authorization check.
func (c *LocalClient) GetMonitoringStatus(ctx context.Context, orgID, targetID, pluginID string) (*MonitoredStatus, error) {
	status, err := c.svc.GetMonitoringStatus(ctx, c.principal, orgID, targetID, pluginID)
	if err != nil {
		return nil, err
	}
	return &MonitoredStatus{
		OrganizationID:   status.OrganizationID,
		DeploymentTarget: status.DeploymentTarget,
		PluginID:         status.PluginID,
		DesiredVersion:   status.Desired.Version,
		ActualVersion:    status.Actual.Version,
		Health:           string(status.Actual.Health),
		HasDiscrepancy:   status.HasDiscrepancy,
		DiscrepancyNotes: status.DiscrepancyNotes,
		LastObservedAt:   status.LastObservedAt,
	}, nil
}

// ListAuditEvents retrieves organization-scoped audit events with authorization check and tenant isolation.
func (c *LocalClient) ListAuditEvents(ctx context.Context) ([]AuditSummary, error) {
	if c.principal == nil {
		return nil, ErrUnauthorized
	}
	events, err := c.svc.GetAuditEvents(ctx, c.principal, c.principal.OrganizationID)
	if err != nil {
		return nil, err
	}
	var summaries []AuditSummary
	for _, ev := range events {
		summaries = append(summaries, AuditSummary{
			Sequence:       ev.Sequence,
			Timestamp:      ev.Timestamp,
			OrganizationID: ev.OrganizationID,
			ActorID:        ev.ActorID,
			Action:         ev.Action,
			Target:         ev.Target,
			Outcome:        ev.Outcome,
			PrevHash:       ev.PrevHash,
			Hash:           ev.Hash,
		})
	}
	return summaries, nil
}
