package enterprise

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/audit"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/auth"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/identity"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/management"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/monitoring"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/organization"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/policy"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/registry"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/validation"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security/verification"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

var (
	// ErrUnauthorized indicates the actor lacks authentication or permission.
	ErrUnauthorized = auth.ErrUnauthorized
)

// MarketplaceCatalogProvider defines an authoritative marketplace lookup interface for provenance verification.
type MarketplaceCatalogProvider interface {
	GetPlugin(ctx context.Context, id string) (*pubmarket.PluginDetail, error)
	GetVersions(ctx context.Context, id string) ([]pubmarket.VersionInfo, error)
}

// Config configures the enterprise service directories and security settings.
type Config struct {
	BaseDir     string
	SigningKey  []byte
	TrustStore  *verification.TrustStore
	Marketplace MarketplaceCatalogProvider
}

// Service provides a unified enterprise management and governance platform.
type Service struct {
	mu          sync.RWMutex
	baseDir     string
	tokenMgr    *identity.TokenManager
	authorizer  auth.Authorizer
	orgMgr      *organization.Manager
	regMgr      *registry.RegistryManager
	policyStore *policy.Storage
	policyEval  *policy.Evaluator
	validator   *validation.EnterpriseValidator
	mgmtMgr     *management.Manager
	monitor     *monitoring.Monitor
	auditLogger *audit.Logger
	marketplace MarketplaceCatalogProvider
}

// NewService constructs and initializes the enterprise service.
func NewService(cfg Config) (*Service, error) {
	if err := os.MkdirAll(cfg.BaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create enterprise base dir: %w", err)
	}

	key := cfg.SigningKey
	if len(key) == 0 {
		var err error
		key, err = identity.GenerateRandomKey()
		if err != nil {
			return nil, err
		}
	}

	tokenMgr, err := identity.NewTokenManager(key)
	if err != nil {
		return nil, err
	}

	authorizer := auth.NewRBACAuthorizer()

	orgMgr, err := organization.NewManager(filepath.Join(cfg.BaseDir, "orgs"))
	if err != nil {
		return nil, err
	}

	regMgr, err := registry.NewRegistryManager(filepath.Join(cfg.BaseDir, "registries"))
	if err != nil {
		return nil, err
	}

	policyStore, err := policy.NewStorage(filepath.Join(cfg.BaseDir, "policies"))
	if err != nil {
		return nil, err
	}

	auditLogger, err := audit.NewLogger(filepath.Join(cfg.BaseDir, "audit"))
	if err != nil {
		return nil, err
	}

	monitor, err := monitoring.NewMonitor(filepath.Join(cfg.BaseDir, "monitoring"))
	if err != nil {
		return nil, err
	}

	validator := validation.NewEnterpriseValidator(cfg.TrustStore)
	mgmtMgr := management.NewManager(monitor, auditLogger)

	return &Service{
		baseDir:     cfg.BaseDir,
		tokenMgr:    tokenMgr,
		authorizer:  authorizer,
		orgMgr:      orgMgr,
		regMgr:      regMgr,
		policyStore: policyStore,
		policyEval:  policy.NewEvaluator(),
		validator:   validator,
		mgmtMgr:     mgmtMgr,
		monitor:     monitor,
		auditLogger: auditLogger,
		marketplace: cfg.Marketplace,
	}, nil
}

// SetMarketplaceCatalog configures or updates the authoritative marketplace catalog provider.
func (s *Service) SetMarketplaceCatalog(cat MarketplaceCatalogProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.marketplace = cat
}

// TokenManager returns the enterprise token manager.
func (s *Service) TokenManager() *identity.TokenManager {
	return s.tokenMgr
}

// Authorizer returns the RBAC authorizer.
func (s *Service) Authorizer() auth.Authorizer {
	return s.authorizer
}

// AuditLogger returns the tamper-evident audit logger.
func (s *Service) AuditLogger() *audit.Logger {
	return s.auditLogger
}

// Monitor returns the enterprise monitoring engine.
func (s *Service) Monitor() *monitoring.Monitor {
	return s.monitor
}

// OrganizationManager returns the underlying organization manager.
func (s *Service) OrganizationManager() *organization.Manager {
	return s.orgMgr
}

// RegisterOrganization creates an organization and initial owner.
func (s *Service) RegisterOrganization(ctx context.Context, id, name, desc, ownerID string) (*organization.Organization, error) {
	org, err := s.orgMgr.CreateOrganization(ctx, id, name, desc, ownerID)
	if err != nil {
		return nil, err
	}

	// Initialize default open policy
	defaultPolicy := &policy.PolicyDocument{
		OrganizationID: id,
		AuthorID:       ownerID,
		AllowlistRules: []policy.AllowlistRule{
			{RuleID: "default-allow", PluginIDs: []string{"*"}},
		},
	}
	_ = s.policyStore.SavePolicy(ctx, defaultPolicy)

	// Emit audit event
	_, _ = s.auditLogger.RecordEvent(audit.AuditEvent{
		OrganizationID: id,
		ActorID:        ownerID,
		ActorType:      identity.PrincipalUser,
		Action:         "org:register",
		Target:         id,
		Outcome:        "success",
		CorrelationID:  id,
	})

	return org, nil
}

// PublishPrivatePlugin publishes a proprietary enterprise plugin into the organization registry.
func (s *Service) PublishPrivatePlugin(
	ctx context.Context,
	principal *identity.Principal,
	pluginID, name, desc string,
	ver version.SemVer,
	manifest *model.Manifest,
	archiveData []byte,
) (*registry.PrivateVersionEntry, *validation.ValidationReport, error) {
	// 1. Authorize
	if err := s.authorizer.Authorize(ctx, principal, auth.PermPluginPublish, principal.OrganizationID); err != nil {
		return nil, nil, err
	}

	// 2. Fetch Organization Policy
	polDoc, err := s.policyStore.GetPolicy(ctx, principal.OrganizationID)
	if err != nil && !errors.Is(err, policy.ErrPolicyNotFound) {
		return nil, nil, fmt.Errorf("failed to retrieve organization policy: %w", err)
	}

	isSigned, malwareClean, _ := s.validator.VerifyArchive(ctx, archiveData)

	evalCtx := policy.EvaluationContext{
		OrganizationID:     principal.OrganizationID,
		PluginID:           pluginID,
		Version:            ver,
		PublisherID:        principal.OrganizationID,
		DistributionSource: "private_registry",
		IsSigned:           isSigned,
		MalwareScanPassed:  malwareClean,
	}

	// 3. Enterprise Validation
	report := s.validator.ValidateAll(
		ctx,
		manifest,
		"",
		malwareClean,
		polDoc,
		evalCtx,
		"", "",
		nil,
		0,
	)
	if !report.Passed {
		return nil, report, fmt.Errorf("enterprise validation failed for private plugin %s", pluginID)
	}

	// 4. Ingest into Organization Registry
	reg, err := s.regMgr.GetRegistry(principal.OrganizationID)
	if err != nil {
		return nil, report, err
	}

	entry, err := reg.PublishPrivatePackage(ctx, pluginID, name, desc, principal.OrganizationID, ver, archiveData)
	if err != nil {
		return nil, report, err
	}

	// 5. Register initial ownership if first version
	_, _ = s.orgMgr.RegisterPluginOwnership(ctx, pluginID, principal.OrganizationID, principal.ID, nil)

	// 6. Audit
	_, _ = s.auditLogger.RecordEvent(audit.AuditEvent{
		OrganizationID: principal.OrganizationID,
		ActorID:        principal.ID,
		ActorType:      principal.Type,
		Action:         "plugin:publish_private",
		Target:         pluginID,
		PluginID:       pluginID,
		Version:        ver.String(),
		Outcome:        "success",
	})

	return entry, report, nil
}

// PromotePublicPlugin approves and mirrors a public marketplace plugin into the enterprise registry.
func (s *Service) PromotePublicPlugin(
	ctx context.Context,
	principal *identity.Principal,
	publicPlugin *pubmarket.PluginDetail,
	publicVersion *pubmarket.VersionInfo,
	archiveReader io.Reader,
	notes string,
	internalTags []string,
) (*registry.PrivateVersionEntry, error) {
	if principal == nil {
		return nil, ErrUnauthorized
	}
	if publicPlugin == nil || publicVersion == nil {
		return nil, errors.New("enterprise service: public plugin and version info cannot be nil")
	}

	if err := s.authorizer.Authorize(ctx, principal, auth.PermPluginPromote, principal.OrganizationID); err != nil {
		return nil, err
	}

	// Authoritative marketplace provenance verification
	if s.marketplace != nil {
		authDetail, err := s.marketplace.GetPlugin(ctx, publicPlugin.ID)
		if err != nil {
			return nil, fmt.Errorf("enterprise service: public marketplace plugin not found: %w", err)
		}
		if authDetail.Status != pubmarket.StatusPublished {
			return nil, fmt.Errorf("enterprise service: cannot promote plugin with marketplace status %s", authDetail.Status)
		}
		if authDetail.Publisher.ID != publicPlugin.Publisher.ID {
			return nil, errors.New("enterprise service: marketplace publisher mismatch")
		}
		versions, err := s.marketplace.GetVersions(ctx, publicPlugin.ID)
		if err != nil {
			return nil, fmt.Errorf("enterprise service: failed to get marketplace versions: %w", err)
		}
		var found bool
		for _, v := range versions {
			if v.Version.Compare(publicVersion.Version) == 0 {
				if v.ArtifactDigest != publicVersion.ArtifactDigest {
					return nil, errors.New("enterprise service: artifact digest mismatch with marketplace catalog")
				}
				if v.Status != pubmarket.StatusPublished {
					return nil, fmt.Errorf("enterprise service: marketplace version status is %s, not published", v.Status)
				}
				found = true
				break
			}
		}
		if !found {
			return nil, errors.New("enterprise service: version not found in marketplace catalog")
		}
	}

	reg, err := s.regMgr.GetRegistry(principal.OrganizationID)
	if err != nil {
		return nil, err
	}

	entry, err := reg.PromotePublicPlugin(ctx, publicPlugin, publicVersion, archiveReader, principal.ID, notes, internalTags)
	if err != nil {
		return nil, err
	}

	_, _ = s.auditLogger.RecordEvent(audit.AuditEvent{
		OrganizationID: principal.OrganizationID,
		ActorID:        principal.ID,
		ActorType:      principal.Type,
		Action:         "plugin:promote",
		Target:         publicPlugin.ID,
		PluginID:       publicPlugin.ID,
		Version:        publicVersion.Version.String(),
		Outcome:        "success",
		Reason:         notes,
	})

	return entry, nil
}

// UpdatePolicy updates the organization's policy document.
func (s *Service) UpdatePolicy(ctx context.Context, principal *identity.Principal, doc *policy.PolicyDocument) error {
	if doc == nil {
		return errors.New("enterprise service: policy document cannot be nil")
	}
	if err := s.authorizer.Authorize(ctx, principal, auth.PermPolicyManage, doc.OrganizationID); err != nil {
		return err
	}

	doc.AuthorID = principal.ID
	if err := s.policyStore.SavePolicy(ctx, doc); err != nil {
		return err
	}

	_, _ = s.auditLogger.RecordEvent(audit.AuditEvent{
		OrganizationID: doc.OrganizationID,
		ActorID:        principal.ID,
		ActorType:      principal.Type,
		Action:         "policy:update",
		Target:         doc.OrganizationID,
		Outcome:        "success",
	})

	return nil
}

// GetPolicy retrieves the policy document for an organization.
func (s *Service) GetPolicy(ctx context.Context, principal *identity.Principal, orgID string) (*policy.PolicyDocument, error) {
	if err := s.authorizer.Authorize(ctx, principal, auth.PermMonitoringView, orgID); err != nil {
		return nil, err
	}
	return s.policyStore.GetPolicy(ctx, orgID)
}

// RegisterDeploymentTarget registers a target machine/cluster and its agent.
func (s *Service) RegisterDeploymentTarget(ctx context.Context, principal *identity.Principal, target management.DeploymentTarget, agent *management.ManagedAgent) error {
	if principal == nil {
		return ErrUnauthorized
	}
	if err := organization.ValidateOrganizationID(target.OrganizationID); err != nil {
		return fmt.Errorf("enterprise service: invalid target organization: %w", err)
	}
	if err := s.authorizer.Authorize(ctx, principal, auth.PermDeploymentManage, target.OrganizationID); err != nil {
		return err
	}
	return s.mgmtMgr.RegisterTarget(target, agent)
}

// DeployPlugin coordinates enterprise policy check, registry retrieval, and remote agent installation.
func (s *Service) DeployPlugin(
	ctx context.Context,
	principal *identity.Principal,
	targetID, pluginID string,
	ver version.SemVer,
) (management.CommandResult, error) {
	target, err := s.mgmtMgr.GetTarget(targetID)
	if err != nil {
		return management.CommandResult{}, err
	}

	// 1. Authorize
	if err := s.authorizer.Authorize(ctx, principal, auth.PermPluginInstall, target.OrganizationID); err != nil {
		return management.CommandResult{}, err
	}

	// 2. Fetch Registry & Artifact
	reg, err := s.regMgr.GetRegistry(target.OrganizationID)
	if err != nil {
		return management.CommandResult{}, err
	}

	reader, entry, err := reg.DownloadArtifact(ctx, pluginID, ver)
	if err != nil {
		return management.CommandResult{}, fmt.Errorf("failed to retrieve plugin artifact: %w", err)
	}
	defer reader.Close()

	archiveBytes, err := io.ReadAll(reader)
	if err != nil {
		return management.CommandResult{}, fmt.Errorf("failed to read artifact data: %w", err)
	}

	// 3. Evaluate Policy for Target Environment
	polDoc, err := s.policyStore.GetPolicy(ctx, target.OrganizationID)
	if err != nil && !errors.Is(err, policy.ErrPolicyNotFound) {
		return management.CommandResult{}, fmt.Errorf("failed to retrieve organization policy: %w", err)
	}

	isSigned, malwareClean, _ := s.validator.VerifyArchive(ctx, archiveBytes)

	evalCtx := policy.EvaluationContext{
		OrganizationID:     target.OrganizationID,
		PluginID:           pluginID,
		Version:            ver,
		PublisherID:        entry.PluginID,
		DistributionSource: "private_registry",
		IsSigned:           isSigned,
		MalwareScanPassed:  malwareClean,
		Environment:        target.Environment,
		Platform:           target.Platform,
	}

	decision := s.policyEval.Evaluate(polDoc, evalCtx)
	if !decision.Allowed {
		_, _ = s.auditLogger.RecordEvent(audit.AuditEvent{
			OrganizationID:   target.OrganizationID,
			ActorID:          principal.ID,
			ActorType:        principal.Type,
			Action:           "plugin:install_blocked",
			Target:           pluginID,
			DeploymentTarget: targetID,
			Outcome:          "denied",
			Reason:           decision.Reason,
		})
		return management.CommandResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("policy denial: %s", decision.Reason),
			ExecutedAt:   time.Now().UTC(),
		}, fmt.Errorf("policy denial: %s", decision.Reason)
	}

	// 4. Remote Install
	return s.mgmtMgr.ExecuteRemoteInstall(ctx, principal, targetID, pluginID, ver.String(), "private_registry", archiveBytes)
}

// DownloadPrivateArtifact retrieves the artifact reader for authorized internal distribution.
func (s *Service) DownloadPrivateArtifact(
	ctx context.Context,
	principal *identity.Principal,
	orgID, pluginID string,
	ver version.SemVer,
) (io.ReadCloser, *registry.PrivateVersionEntry, error) {
	if err := s.authorizer.Authorize(ctx, principal, auth.PermMonitoringView, orgID); err != nil {
		return nil, nil, err
	}

	reg, err := s.regMgr.GetRegistry(orgID)
	if err != nil {
		return nil, nil, err
	}

	return reg.DownloadArtifact(ctx, pluginID, ver)
}

// AddMember adds or updates a member's roles within an organization.
func (s *Service) AddMember(ctx context.Context, principal *identity.Principal, orgID, principalID string, roles []identity.Role) (*organization.Member, error) {
	if err := s.authorizer.Authorize(ctx, principal, auth.PermOrgManage, orgID); err != nil {
		return nil, err
	}

	mem, err := s.orgMgr.AddMember(ctx, orgID, principalID, roles)
	if err != nil {
		return nil, err
	}

	actorID := "system"
	actorType := identity.PrincipalServiceAccount
	if principal != nil {
		actorID = principal.ID
		actorType = principal.Type
	}
	_, _ = s.auditLogger.RecordEvent(audit.AuditEvent{
		OrganizationID: orgID,
		ActorID:        actorID,
		ActorType:      actorType,
		Action:         "org:member_add",
		Target:         principalID,
		Outcome:        "success",
		CorrelationID:  orgID,
	})

	return mem, nil
}

// TransferPluginOwnership reassigns the primary ownership of a plugin.
func (s *Service) TransferPluginOwnership(ctx context.Context, principal *identity.Principal, pluginID, newOwnerID, reason string) (*organization.PluginOwnership, error) {
	if principal == nil {
		return nil, ErrUnauthorized
	}

	ownership, err := s.orgMgr.GetPluginOwnership(ctx, principal.OrganizationID, pluginID)
	if err != nil {
		return nil, err
	}

	if err := s.authorizer.Authorize(ctx, principal, auth.PermPluginOwnershipManage, ownership.OrganizationID); err != nil {
		return nil, err
	}

	actorID := principal.ID
	actorType := principal.Type

	updated, err := s.orgMgr.TransferPluginOwnership(ctx, principal.OrganizationID, pluginID, actorID, newOwnerID, reason)
	if err != nil {
		return nil, err
	}

	_, _ = s.auditLogger.RecordEvent(audit.AuditEvent{
		OrganizationID: ownership.OrganizationID,
		ActorID:        actorID,
		ActorType:      actorType,
		Action:         "plugin:ownership_transfer",
		Target:         pluginID,
		PluginID:       pluginID,
		Outcome:        "success",
		Reason:         reason,
		CorrelationID:  pluginID,
	})

	return updated, nil
}

// UpdatePlugin coordinates upgrade validation, policy checks, artifact retrieval, and remote execution with rollback.
func (s *Service) UpdatePlugin(
	ctx context.Context,
	principal *identity.Principal,
	targetID, pluginID string,
	newVer version.SemVer,
) (management.CommandResult, error) {
	target, err := s.mgmtMgr.GetTarget(targetID)
	if err != nil {
		return management.CommandResult{}, err
	}

	// 1. Authorize
	if err := s.authorizer.Authorize(ctx, principal, auth.PermPluginUpdate, target.OrganizationID); err != nil {
		return management.CommandResult{}, err
	}

	// 2. Query currently installed version for rollback feasibility
	var curVerStr string
	if monStatus, err := s.monitor.GetStatus(ctx, target.OrganizationID, targetID, pluginID); err == nil && monStatus != nil {
		curVerStr = monStatus.Actual.Version
	}

	polDoc, err := s.policyStore.GetPolicy(ctx, target.OrganizationID)
	if err != nil && !errors.Is(err, policy.ErrPolicyNotFound) {
		return management.CommandResult{}, fmt.Errorf("failed to retrieve organization policy: %w", err)
	}

	// 3. Validate Upgrade if previous version exists
	if curVerStr != "" {
		if curVer, err := version.ParseSemVer(curVerStr); err == nil {
			upOk, issues := s.validator.ValidateUpgrade(ctx, curVer, newVer, polDoc)
			if !upOk {
				var msg string
				if len(issues) > 0 {
					msg = issues[0].Message
				} else {
					msg = "upgrade validation rejected"
				}
				_, _ = s.auditLogger.RecordEvent(audit.AuditEvent{
					OrganizationID:   target.OrganizationID,
					ActorID:          principal.ID,
					ActorType:        principal.Type,
					Action:           "plugin:update_blocked",
					Target:           pluginID,
					DeploymentTarget: targetID,
					Outcome:          "denied",
					Reason:           msg,
				})
				return management.CommandResult{
					Success:      false,
					ErrorMessage: msg,
					ExecutedAt:   time.Now().UTC(),
				}, fmt.Errorf("upgrade validation failed: %s", msg)
			}
		}
	}

	// 4. Retrieve New Artifact
	reg, err := s.regMgr.GetRegistry(target.OrganizationID)
	if err != nil {
		return management.CommandResult{}, err
	}

	reader, entry, err := reg.DownloadArtifact(ctx, pluginID, newVer)
	if err != nil {
		return management.CommandResult{}, fmt.Errorf("failed to retrieve plugin artifact: %w", err)
	}
	defer reader.Close()

	archiveBytes, err := io.ReadAll(reader)
	if err != nil {
		return management.CommandResult{}, fmt.Errorf("failed to read artifact data: %w", err)
	}

	// 5. Evaluate Policy
	isSigned, malwareClean, _ := s.validator.VerifyArchive(ctx, archiveBytes)

	evalCtx := policy.EvaluationContext{
		OrganizationID:     target.OrganizationID,
		PluginID:           pluginID,
		Version:            newVer,
		PublisherID:        entry.PluginID,
		DistributionSource: "private_registry",
		IsSigned:           isSigned,
		MalwareScanPassed:  malwareClean,
		Environment:        target.Environment,
		Platform:           target.Platform,
	}

	decision := s.policyEval.Evaluate(polDoc, evalCtx)
	if !decision.Allowed {
		_, _ = s.auditLogger.RecordEvent(audit.AuditEvent{
			OrganizationID:   target.OrganizationID,
			ActorID:          principal.ID,
			ActorType:        principal.Type,
			Action:           "plugin:update_blocked",
			Target:           pluginID,
			DeploymentTarget: targetID,
			Outcome:          "denied",
			Reason:           decision.Reason,
		})
		return management.CommandResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("policy denial: %s", decision.Reason),
			ExecutedAt:   time.Now().UTC(),
		}, fmt.Errorf("policy denial: %s", decision.Reason)
	}

	// 6. Execute Remote Update with Rollback Version
	return s.mgmtMgr.ExecuteRemoteUpdate(ctx, principal, targetID, pluginID, newVer.String(), curVerStr, archiveBytes)
}

// RemovePlugin coordinates authorized remote plugin removal.
func (s *Service) RemovePlugin(
	ctx context.Context,
	principal *identity.Principal,
	targetID, pluginID string,
	purgeData bool,
) (management.CommandResult, error) {
	target, err := s.mgmtMgr.GetTarget(targetID)
	if err != nil {
		return management.CommandResult{}, err
	}

	if err := s.authorizer.Authorize(ctx, principal, auth.PermPluginRemove, target.OrganizationID); err != nil {
		return management.CommandResult{}, err
	}

	return s.mgmtMgr.ExecuteRemoteRemoval(ctx, principal, targetID, pluginID, purgeData)
}

// GetMonitoringStatus retrieves the monitored plugin status with authorization check.
func (s *Service) GetMonitoringStatus(
	ctx context.Context,
	principal *identity.Principal,
	orgID, targetID, pluginID string,
) (*monitoring.ManagedPluginStatus, error) {
	if err := s.authorizer.Authorize(ctx, principal, auth.PermMonitoringView, orgID); err != nil {
		return nil, err
	}
	return s.monitor.GetStatus(ctx, orgID, targetID, pluginID)
}

// GetAuditEvents returns audit events for an organization with authorization check and tenant filtering.
func (s *Service) GetAuditEvents(
	ctx context.Context,
	principal *identity.Principal,
	orgID string,
) ([]audit.AuditEvent, error) {
	if err := s.authorizer.Authorize(ctx, principal, auth.PermAuditView, orgID); err != nil {
		return nil, err
	}

	allEvents, err := s.auditLogger.ReadEvents()
	if err != nil {
		return nil, err
	}

	var filtered []audit.AuditEvent
	for _, ev := range allEvents {
		if ev.OrganizationID == orgID {
			filtered = append(filtered, ev)
		}
	}
	return filtered, nil
}
