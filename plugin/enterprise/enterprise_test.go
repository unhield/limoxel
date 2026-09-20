package enterprise

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	intenterprise "github.com/unhield/limoxel/internal/capabilities/plugin/enterprise"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/identity"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/management"
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

func TestPublicEnterpriseClient_EndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	svc, err := intenterprise.NewService(intenterprise.Config{BaseDir: tempDir})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	principal := &identity.Principal{
		ID:             "corp-admin",
		OrganizationID: "org-global",
		Type:           identity.PrincipalUser,
		Roles:          []identity.Role{identity.RoleOwner, identity.RoleAdmin},
		Active:         true,
	}

	client := NewLocalClient(svc, principal)
	ctx := context.Background()

	// 1. Register Organization
	org, err := client.RegisterOrganization(ctx, "org-global", "Global Corp", "Multinational Enterprise")
	if err != nil {
		t.Fatalf("RegisterOrganization failed: %v", err)
	}
	if org.ID != "org-global" {
		t.Errorf("unexpected org: %+v", org)
	}

	// 2. Publish Private Plugin
	v1, _ := version.ParseSemVer("1.0.0")
	manifest := &model.Manifest{
		SchemaVersion: model.ManifestSchemaVersion,
		ID:            model.Identity("corp.analytics"),
		Name:          "Analytics Plugin",
		Version:       v1,
		Publisher:     "org-global",
	}
	payload := []byte("private corp analytics archive bytes")

	entry, err := client.PublishPrivatePlugin(
		ctx,
		"corp.analytics",
		"Analytics Plugin",
		"Proprietary telemetry plugin",
		v1,
		manifest,
		payload,
	)
	if err != nil {
		t.Fatalf("PublishPrivatePlugin failed: %v", err)
	}
	if entry.PluginID != "corp.analytics" || entry.Version != "1.0.0" {
		t.Errorf("unexpected private version entry: %+v", entry)
	}

	// 3. Connect Host Agent and Target
	executor := &mockHostExecutor{installed: make(map[string]string)}
	agent := management.NewManagedAgent("org-global", "node-linux-1", executor)
	target := management.DeploymentTarget{
		ID:             "node-linux-1",
		OrganizationID: "org-global",
		Name:           "Node Linux 1",
		Environment:    "production",
		Platform:       "linux/amd64",
		RegisteredAt:   time.Now().UTC(),
		Active:         true,
	}
	if err := svc.RegisterDeploymentTarget(ctx, principal, target, agent); err != nil {
		t.Fatalf("RegisterDeploymentTarget failed: %v", err)
	}

	// 4. Deploy Plugin via Public Client
	cmdRes, err := client.DeployPlugin(ctx, "node-linux-1", "corp.analytics", v1)
	if err != nil || !cmdRes.Success {
		t.Fatalf("DeployPlugin failed: %v, res: %+v", err, cmdRes)
	}
	if executor.installed["corp.analytics"] != "1.0.0" {
		t.Errorf("expected plugin deployed to host agent")
	}

	// 5. Promote Public Plugin
	pubArchive := []byte("public auth helper bytes")
	hasher := sha256.New()
	hasher.Write(pubArchive)
	digest := hex.EncodeToString(hasher.Sum(nil))

	vPublic, _ := version.ParseSemVer("3.0.0")
	pubPlugin := &pubmarket.PluginDetail{
		PluginSummary: pubmarket.PluginSummary{
			ID:          "public.authhelper",
			Name:        "Auth Helper",
			Description: "OAuth2 client helper",
		},
		Publisher: pubmarket.PublisherProfile{
			ID: "pub-auth-community",
		},
	}
	pubVer := &pubmarket.VersionInfo{
		Version:        vPublic,
		ArtifactDigest: digest,
	}

	promoted, err := client.PromotePublicPlugin(
		ctx,
		pubPlugin,
		pubVer,
		bytes.NewReader(pubArchive),
		"Approved for corporate login flows",
		[]string{"security", "auth"},
	)
	if err != nil {
		t.Fatalf("PromotePublicPlugin failed: %v", err)
	}
	if !promoted.IsPromotedPublic {
		t.Errorf("expected promoted plugin to be flagged IsPromotedPublic")
	}

	// 6. Test AddMember via Public Client
	memSummary, err := client.AddMember(ctx, "org-global", "bob-dev", []Role{RolePluginManager, RoleOperator})
	if err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}
	if memSummary.PrincipalID != "bob-dev" || len(memSummary.Roles) != 2 {
		t.Errorf("unexpected member summary: %+v", memSummary)
	}

	// Unauthorized actor attempting to AddMember must fail
	unauthorizedPrincipal := &identity.Principal{
		ID:             "unauth-user",
		OrganizationID: "org-global",
		Type:           identity.PrincipalUser,
		Roles:          []identity.Role{identity.RoleMember},
		Active:         true,
	}
	unauthClient := NewLocalClient(svc, unauthorizedPrincipal)
	if _, err := unauthClient.AddMember(ctx, "org-global", "eve-hacker", []Role{RoleAdmin}); err == nil {
		t.Fatalf("expected AddMember by unauthorized member to fail")
	}

	// 7. Test UpdatePlugin via Public Client
	v11, _ := version.ParseSemVer("1.1.0")
	manifest11 := &model.Manifest{
		SchemaVersion: model.ManifestSchemaVersion,
		ID:            model.Identity("corp.analytics"),
		Name:          "Analytics Plugin",
		Version:       v11,
		Publisher:     "org-global",
	}
	payload11 := []byte("private corp analytics archive bytes v1.1.0")
	_, err = client.PublishPrivatePlugin(
		ctx,
		"corp.analytics",
		"Analytics Plugin",
		"Proprietary telemetry plugin v1.1.0",
		v11,
		manifest11,
		payload11,
	)
	if err != nil {
		t.Fatalf("PublishPrivatePlugin v1.1.0 failed: %v", err)
	}

	upRes, err := client.UpdatePlugin(ctx, "node-linux-1", "corp.analytics", v11)
	if err != nil || !upRes.Success {
		t.Fatalf("UpdatePlugin failed: %v, res: %+v", err, upRes)
	}
	if executor.installed["corp.analytics"] != "1.1.0" {
		t.Errorf("expected plugin updated to 1.1.0, got: %s", executor.installed["corp.analytics"])
	}

	// 8. Test GetMonitoringStatus (Authorized vs Cross-Tenant)
	monStatus, err := client.GetMonitoringStatus(ctx, "org-global", "node-linux-1", "corp.analytics")
	if err != nil || monStatus == nil {
		t.Fatalf("GetMonitoringStatus failed: %v", err)
	}
	if monStatus.ActualVersion != "1.1.0" {
		t.Errorf("expected actual version 1.1.0 in monitoring, got: %s", monStatus.ActualVersion)
	}

	// Cross-tenant monitoring query must be rejected
	if _, err := client.GetMonitoringStatus(ctx, "org-other-corp", "node-linux-1", "corp.analytics"); err == nil {
		t.Fatalf("expected cross-tenant monitoring query to be rejected")
	}

	// 9. Test RemovePlugin via Public Client
	remRes, err := client.RemovePlugin(ctx, "node-linux-1", "corp.analytics", false)
	if err != nil || !remRes.Success {
		t.Fatalf("RemovePlugin failed: %v, res: %+v", err, remRes)
	}
	if _, exists := executor.installed["corp.analytics"]; exists {
		t.Errorf("expected plugin to be removed from executor")
	}

	// 10. Test Multi-Tenant Audit Isolation
	_, _ = svc.RegisterOrganization(ctx, "org-other", "Other Corp", "Another Org", "other-owner")
	otherOwnerPrincipal := &identity.Principal{
		ID:             "other-owner",
		OrganizationID: "org-other",
		Type:           identity.PrincipalUser,
		Roles:          []identity.Role{identity.RoleOwner},
		Active:         true,
	}
	otherClient := NewLocalClient(svc, otherOwnerPrincipal)

	// ListAuditEvents on org-global client should only return org-global events
	globalAudit, err := client.ListAuditEvents(ctx)
	if err != nil {
		t.Fatalf("ListAuditEvents failed: %v", err)
	}
	for _, ev := range globalAudit {
		if ev.OrganizationID != "org-global" {
			t.Errorf("audit leakage: found event for %q in org-global stream", ev.OrganizationID)
		}
	}

	// ListAuditEvents on org-other client should only return org-other events
	otherAudit, err := otherClient.ListAuditEvents(ctx)
	if err != nil {
		t.Fatalf("otherClient.ListAuditEvents failed: %v", err)
	}
	for _, ev := range otherAudit {
		if ev.OrganizationID != "org-other" {
			t.Errorf("audit leakage: found event for %q in org-other stream", ev.OrganizationID)
		}
	}

	// 11. Test TransferPluginOwnership via Service
	transferRec, err := svc.TransferPluginOwnership(ctx, principal, "corp.analytics", "bob-dev", "reassigning primary ownership to senior engineer")
	if err != nil {
		t.Fatalf("TransferPluginOwnership failed: %v", err)
	}
	if transferRec.PrimaryOwnerID != "bob-dev" {
		t.Errorf("expected primary owner to be bob-dev, got: %s", transferRec.PrimaryOwnerID)
	}
	if len(transferRec.TransferHistory) != 1 {
		t.Fatalf("expected 1 transfer history record, got %d", len(transferRec.TransferHistory))
	}
}

func TestPublicEnterpriseClient_NilPrincipalUnauthorized(t *testing.T) {
	tempDir := t.TempDir()
	svc, err := intenterprise.NewService(intenterprise.Config{BaseDir: tempDir})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	// Client with nil principal must not be allowed to register organizations as admin
	unauthClient := NewLocalClient(svc, nil)
	_, err = unauthClient.RegisterOrganization(context.Background(), "org-rogue", "Rogue Corp", "")
	if err != ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized when registering organization with nil principal, got: %v", err)
	}
}
