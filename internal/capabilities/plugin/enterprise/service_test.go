package enterprise

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/identity"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/management"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/policy"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

type mockHostExecutor struct {
	installed map[string]string
}

func (m *mockHostExecutor) Install(ctx context.Context, pluginID, version string, archiveData []byte) error {
	m.installed[pluginID] = version
	return nil
}

func (m *mockHostExecutor) Update(ctx context.Context, pluginID, targetVersion string, archiveData []byte) error {
	m.installed[pluginID] = targetVersion
	return nil
}

func (m *mockHostExecutor) Remove(ctx context.Context, pluginID string, purgeData bool) error {
	delete(m.installed, pluginID)
	return nil
}

func (m *mockHostExecutor) Suspend(ctx context.Context, pluginID string) error {
	return nil
}

func (m *mockHostExecutor) Restart(ctx context.Context, pluginID string) error {
	return nil
}

func (m *mockHostExecutor) GetInstalled() map[string]string {
	clone := make(map[string]string)
	for k, v := range m.installed {
		clone[k] = v
	}
	return clone
}

func TestEnterpriseService_EndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	svc, err := NewService(Config{BaseDir: tempDir})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	ctx := context.Background()

	// 1. Register Organization
	org, err := svc.RegisterOrganization(ctx, "org-megacorp", "MegaCorp Inc", "Enterprise Org", "owner-alice")
	if err != nil {
		t.Fatalf("RegisterOrganization failed: %v", err)
	}
	if org.ID != "org-megacorp" {
		t.Errorf("unexpected org: %+v", org)
	}

	ownerPrincipal := &identity.Principal{
		ID:             "owner-alice",
		OrganizationID: "org-megacorp",
		Type:           identity.PrincipalUser,
		Roles:          []identity.Role{identity.RoleOwner},
		Active:         true,
	}

	// 2. Publish Private Plugin
	v1, _ := version.ParseSemVer("1.0.0")
	manifest := &model.Manifest{
		SchemaVersion: model.ManifestSchemaVersion,
		ID:            model.Identity("megacorp.security"),
		Name:          "Security Tool",
		Version:       v1,
		Publisher:     "org-megacorp",
	}
	archivePayload := []byte("private megacorp security package bytes")

	entry, report, err := svc.PublishPrivatePlugin(
		ctx,
		ownerPrincipal,
		"megacorp.security",
		"Security Tool",
		"Proprietary security tool",
		v1,
		manifest,
		archivePayload,
	)
	if err != nil || !report.Passed {
		t.Fatalf("PublishPrivatePlugin failed: %v, report: %+v", err, report)
	}
	if entry.PluginID != "megacorp.security" || entry.Version != "1.0.0" {
		t.Errorf("unexpected version entry: %+v", entry)
	}

	// 3. Connect Managed Agent and Register Deployment Target
	executor := &mockHostExecutor{installed: make(map[string]string)}
	agent := management.NewManagedAgent("org-megacorp", "prod-cluster-1", executor)

	target := management.DeploymentTarget{
		ID:             "prod-cluster-1",
		OrganizationID: "org-megacorp",
		Name:           "Production Cluster Primary",
		Environment:    "production",
		Platform:       "linux/amd64",
		RegisteredAt:   time.Now().UTC(),
		Active:         true,
	}

	if err := svc.RegisterDeploymentTarget(ctx, ownerPrincipal, target, agent); err != nil {
		t.Fatalf("RegisterDeploymentTarget failed: %v", err)
	}

	// 4. Deploy Plugin to Target
	cmdRes, err := svc.DeployPlugin(ctx, ownerPrincipal, "prod-cluster-1", "megacorp.security", v1)
	if err != nil || !cmdRes.Success {
		t.Fatalf("DeployPlugin failed: %v, cmdRes: %+v", err, cmdRes)
	}
	if executor.installed["megacorp.security"] != "1.0.0" {
		t.Errorf("expected plugin installed on agent, got: %s", executor.installed["megacorp.security"])
	}

	// 5. Test Policy Denial: Update Policy with Denylist
	policyDoc, err := svc.GetPolicy(ctx, ownerPrincipal, "org-megacorp")
	if err != nil {
		t.Fatalf("GetPolicy failed: %v", err)
	}
	policyDoc.DenylistRules = append(policyDoc.DenylistRules, policy.DenylistRule{
		RuleID:    "deny-security-tool",
		PluginIDs: []string{"megacorp.security"},
		Reason:    "Emergency freeze on security tool",
	})
	if err := svc.UpdatePolicy(ctx, ownerPrincipal, policyDoc); err != nil {
		t.Fatalf("UpdatePolicy failed: %v", err)
	}

	// Attempting to deploy denied plugin must fail
	_, err = svc.DeployPlugin(ctx, ownerPrincipal, "prod-cluster-1", "megacorp.security", v1)
	if err == nil {
		t.Fatalf("expected deployment to be blocked by policy")
	}

	// 6. Test Promoting Public Marketplace Plugin
	publicArchive := []byte("public database connector archive bytes")
	hasher := sha256.New()
	hasher.Write(publicArchive)
	digest := hex.EncodeToString(hasher.Sum(nil))

	v2, _ := version.ParseSemVer("2.5.0")
	pubPlugin := &pubmarket.PluginDetail{
		PluginSummary: pubmarket.PluginSummary{
			ID:          "public.postgres",
			Name:        "PostgreSQL Driver",
			Description: "Public database connector",
		},
		Publisher: pubmarket.PublisherProfile{
			ID: "pub-postgres-foundation",
		},
	}
	pubVer := &pubmarket.VersionInfo{
		Version:        v2,
		ArtifactDigest: digest,
	}

	promotedEntry, err := svc.PromotePublicPlugin(
		ctx,
		ownerPrincipal,
		pubPlugin,
		pubVer,
		bytes.NewReader(publicArchive),
		"Approved for enterprise cloud deployments",
		[]string{"database", "certified"},
	)
	if err != nil {
		t.Fatalf("PromotePublicPlugin failed: %v", err)
	}
	if !promotedEntry.IsPromotedPublic || promotedEntry.Promotion.OriginalPublisher != "pub-postgres-foundation" {
		t.Errorf("unexpected promoted entry: %+v", promotedEntry)
	}

	// 7. Verify Audit Integrity Chain
	valid, err := svc.AuditLogger().VerifyIntegrity()
	if err != nil || !valid {
		t.Errorf("audit chain integrity failed: %v", err)
	}
}

func TestEnterpriseService_RequireSignedAndMalwareScanPolicy(t *testing.T) {
	tempDir := t.TempDir()
	svc, err := NewService(Config{BaseDir: tempDir})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	ctx := context.Background()

	// 1. Register Org & Target
	_, err = svc.RegisterOrganization(ctx, "org-sec", "Sec Org", "", "sec-owner")
	if err != nil {
		t.Fatalf("RegisterOrganization failed: %v", err)
	}

	ownerPrincipal := &identity.Principal{
		ID:             "sec-owner",
		OrganizationID: "org-sec",
		Type:           identity.PrincipalUser,
		Roles:          []identity.Role{identity.RoleOwner, identity.RoleAdmin},
		Active:         true,
	}

	executor := &mockHostExecutor{installed: make(map[string]string)}
	agent := management.NewManagedAgent("org-sec", "sec-target-1", executor)
	target := management.DeploymentTarget{
		ID:             "sec-target-1",
		OrganizationID: "org-sec",
		Name:           "Sec Target 1",
		Environment:    "production",
		Platform:       "linux/amd64",
		RegisteredAt:   time.Now().UTC(),
		Active:         true,
	}
	if err := svc.RegisterDeploymentTarget(ctx, ownerPrincipal, target, agent); err != nil {
		t.Fatalf("RegisterDeploymentTarget failed: %v", err)
	}

	// 2. Publish unsigned plugin without policy yet
	v1, _ := version.ParseSemVer("1.0.0")
	manifest := &model.Manifest{
		SchemaVersion: model.ManifestSchemaVersion,
		ID:            model.Identity("unsigned.plugin"),
		Name:          "Unsigned Plugin",
		Version:       v1,
		Publisher:     "org-sec",
	}
	payload := []byte("plain unsigned bytes")
	_, _, err = svc.PublishPrivatePlugin(ctx, ownerPrincipal, "unsigned.plugin", "Unsigned Plugin", "", v1, manifest, payload)
	if err != nil {
		t.Fatalf("PublishPrivatePlugin failed: %v", err)
	}

	// 3. Enforce RequireSigned policy
	policyDoc := &policy.PolicyDocument{
		OrganizationID: "org-sec",
		SecurityRules: []policy.SecurityRule{
			{
				RuleID:        "sec-signed-only",
				RequireSigned: true,
			},
		},
	}
	if err := svc.UpdatePolicy(ctx, ownerPrincipal, policyDoc); err != nil {
		t.Fatalf("UpdatePolicy failed: %v", err)
	}

	// 4. Deploying unsigned plugin MUST be blocked by security policy
	_, err = svc.DeployPlugin(ctx, ownerPrincipal, "sec-target-1", "unsigned.plugin", v1)
	if err == nil {
		t.Fatal("expected DeployPlugin to fail when policy requires signed plugins and plugin is unsigned")
	}

	// 5. Test malware detection during publish
	peData := []byte{0x4D, 0x5A, 0x90, 0x00}
	malManifest := &model.Manifest{
		SchemaVersion: model.ManifestSchemaVersion,
		ID:            model.Identity("malware.plugin"),
		Name:          "Malware Plugin",
		Version:       v1,
		Publisher:     "org-sec",
	}
	_, report, err := svc.PublishPrivatePlugin(ctx, ownerPrincipal, "malware.plugin", "Malware Plugin", "", v1, malManifest, peData)
	if err == nil || report.Passed || report.SecurityPassed {
		t.Fatal("expected PublishPrivatePlugin to fail for payload containing suspicious PE executable header")
	}
}

func TestEnterpriseService_RegisterDeploymentTarget_Authorization(t *testing.T) {
	tempDir := t.TempDir()
	svc, err := NewService(Config{BaseDir: tempDir})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	ctx := context.Background()

	_, err = svc.RegisterOrganization(ctx, "org-authorized", "Authorized Org", "", "admin-user")
	if err != nil {
		t.Fatalf("RegisterOrganization failed: %v", err)
	}

	executor := &mockHostExecutor{installed: make(map[string]string)}
	agent := management.NewManagedAgent("org-authorized", "target-1", executor)
	validTarget := management.DeploymentTarget{
		ID:             "target-1",
		OrganizationID: "org-authorized",
		Active:         true,
	}

	// 1. Unauthenticated registration must fail
	if err := svc.RegisterDeploymentTarget(ctx, nil, validTarget, agent); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for nil principal, got %v", err)
	}

	// 2. Invalid target organization ID format must fail
	invalidTarget := management.DeploymentTarget{
		ID:             "target-inv",
		OrganizationID: "../invalid/org",
		Active:         true,
	}
	authPrincipal := &identity.Principal{
		ID:             "admin-user",
		OrganizationID: "org-authorized",
		Type:           identity.PrincipalUser,
		Roles:          []identity.Role{identity.RoleAdmin},
		Active:         true,
	}
	if err := svc.RegisterDeploymentTarget(ctx, authPrincipal, invalidTarget, agent); err == nil {
		t.Error("expected error for invalid organization id format")
	}

	// 3. Principal from different tenant must fail authorization
	crossTenantPrincipal := &identity.Principal{
		ID:             "hacker",
		OrganizationID: "org-other",
		Type:           identity.PrincipalUser,
		Roles:          []identity.Role{identity.RoleAdmin},
		Active:         true,
	}
	if err := svc.RegisterDeploymentTarget(ctx, crossTenantPrincipal, validTarget, agent); err == nil {
		t.Error("expected cross-tenant target registration to be blocked")
	}

	// 4. Authorized registration must succeed
	if err := svc.RegisterDeploymentTarget(ctx, authPrincipal, validTarget, agent); err != nil {
		t.Fatalf("authorized target registration failed: %v", err)
	}
}

type mockMarketplaceCatalog struct {
	plugins  map[string]*pubmarket.PluginDetail
	versions map[string][]pubmarket.VersionInfo
}

func (m *mockMarketplaceCatalog) GetPlugin(ctx context.Context, id string) (*pubmarket.PluginDetail, error) {
	p, ok := m.plugins[id]
	if !ok {
		return nil, errors.New("plugin not found in catalog")
	}
	return p, nil
}

func (m *mockMarketplaceCatalog) GetVersions(ctx context.Context, id string) ([]pubmarket.VersionInfo, error) {
	v, ok := m.versions[id]
	if !ok {
		return nil, errors.New("versions not found in catalog")
	}
	return v, nil
}

func TestEnterpriseService_PromotePublicPlugin_AuthoritativeProvenance(t *testing.T) {
	tempDir := t.TempDir()
	mockCat := &mockMarketplaceCatalog{
		plugins:  make(map[string]*pubmarket.PluginDetail),
		versions: make(map[string][]pubmarket.VersionInfo),
	}

	svc, err := NewService(Config{
		BaseDir:     tempDir,
		Marketplace: mockCat,
	})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	ctx := context.Background()

	_, err = svc.RegisterOrganization(ctx, "org-promo", "Promo Org", "", "promoter-1")
	if err != nil {
		t.Fatalf("RegisterOrganization failed: %v", err)
	}

	promoter := &identity.Principal{
		ID:             "promoter-1",
		OrganizationID: "org-promo",
		Type:           identity.PrincipalUser,
		Roles:          []identity.Role{identity.RolePluginManager},
		Active:         true,
	}

	v1, _ := version.ParseSemVer("1.0.0")
	validPayload := []byte("trusted public plugin archive data")
	h := sha256.Sum256(validPayload)
	validDigest := hex.EncodeToString(h[:])

	// Populate authoritative marketplace catalog
	mockCat.plugins["pub.verified"] = &pubmarket.PluginDetail{
		PluginSummary: pubmarket.PluginSummary{
			ID:     "pub.verified",
			Status: pubmarket.StatusPublished,
		},
		Publisher: pubmarket.PublisherProfile{
			ID: "official-pub",
		},
	}
	mockCat.versions["pub.verified"] = []pubmarket.VersionInfo{
		{
			Version:        v1,
			ArtifactDigest: validDigest,
			Status:         pubmarket.StatusPublished,
		},
	}

	pubPlugin := &pubmarket.PluginDetail{
		PluginSummary: pubmarket.PluginSummary{
			ID:     "pub.verified",
			Status: pubmarket.StatusPublished,
		},
		Publisher: pubmarket.PublisherProfile{
			ID: "official-pub",
		},
	}
	pubVer := &pubmarket.VersionInfo{
		Version:        v1,
		ArtifactDigest: validDigest,
		Status:         pubmarket.StatusPublished,
	}

	// 1. Legitimate promotion matching marketplace catalog succeeds
	entry, err := svc.PromotePublicPlugin(ctx, promoter, pubPlugin, pubVer, bytes.NewReader(validPayload), "Approved for corp use", []string{"verified"})
	if err != nil {
		t.Fatalf("expected promotion to succeed, got %v", err)
	}
	if entry.PluginID != "pub.verified" {
		t.Errorf("expected promoted plugin ID pub.verified, got %s", entry.PluginID)
	}

	// 2. Tampered digest must fail authoritative provenance check
	tamperedVer := &pubmarket.VersionInfo{
		Version:        v1,
		ArtifactDigest: "forged-digest-00000000000000000000000000000000000000000000000000000000",
		Status:         pubmarket.StatusPublished,
	}
	_, err = svc.PromotePublicPlugin(ctx, promoter, pubPlugin, tamperedVer, bytes.NewReader(validPayload), "tampered digest attempt", nil)
	if err == nil {
		t.Fatal("expected promotion of tampered digest to fail authoritative provenance check")
	}

	// 3. Forged plugin not in catalog must fail
	forgedPlugin := &pubmarket.PluginDetail{
		PluginSummary: pubmarket.PluginSummary{
			ID:     "pub.forged",
			Status: pubmarket.StatusPublished,
		},
		Publisher: pubmarket.PublisherProfile{
			ID: "attacker",
		},
	}
	_, err = svc.PromotePublicPlugin(ctx, promoter, forgedPlugin, pubVer, bytes.NewReader(validPayload), "forged plugin attempt", nil)
	if err == nil {
		t.Fatal("expected promotion of unlisted public plugin to fail")
	}
}
