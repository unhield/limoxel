package plugin_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/auth"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/identity"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/management"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/policy"
	"github.com/unhield/limoxel/internal/capabilities/plugin/loader"
	"github.com/unhield/limoxel/internal/capabilities/plugin/marketplace"
	"github.com/unhield/limoxel/internal/capabilities/plugin/marketplace/validation"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security/verification"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	"github.com/unhield/limoxel/plugin"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
	plugtest "github.com/unhield/limoxel/plugin/testing"
	"github.com/unhield/limoxel/plugin/tooling"
)

type mockFlowHostExecutor struct {
	installed map[string]string
	mu        sync.Mutex
}

func (m *mockFlowHostExecutor) Install(ctx context.Context, pluginID, version string, archiveData []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.installed[pluginID] = version
	return nil
}

func (m *mockFlowHostExecutor) Update(ctx context.Context, pluginID, targetVersion string, archiveData []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.installed[pluginID] = targetVersion
	return nil
}

func (m *mockFlowHostExecutor) Remove(ctx context.Context, pluginID string, purgeData bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.installed, pluginID)
	return nil
}

func (m *mockFlowHostExecutor) Suspend(ctx context.Context, pluginID string) error {
	return nil
}

func (m *mockFlowHostExecutor) Restart(ctx context.Context, pluginID string) error {
	return nil
}

func (m *mockFlowHostExecutor) GetInstalled() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	clone := make(map[string]string)
	for k, v := range m.installed {
		clone[k] = v
	}
	return clone
}

type flowTestPlugin struct {
	id     string
	active bool
	mu     sync.Mutex
}

func (p *flowTestPlugin) ID() string                { return p.id }
func (p *flowTestPlugin) Manifest() plugin.Manifest { return plugin.Manifest{ID: p.id} }
func (p *flowTestPlugin) Init(ctx context.Context, host plugin.Host) error {
	return nil
}
func (p *flowTestPlugin) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.active = true
	return nil
}
func (p *flowTestPlugin) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.active = false
	return nil
}

// Flow A: Plugin Developer Flow (SDK -> Build -> Package -> Validate -> Publish -> Discover -> Download)
func TestIntegration_FlowA_PluginDeveloperFlow(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// 1. SDK Project Layout & Manifest
	projectDir := filepath.Join(tempDir, "dev-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	manifest := plugin.Manifest{
		SchemaVersion: "1.0.0",
		ID:            "com.developer.linter",
		Name:          "Dev Linter Plugin",
		Version:       "1.0.0",
		Description:   "A high-performance code linter plugin",
		Publisher:     "acme-devs",
		Entrypoint:    "analyzer.txt",
	}
	mBytes, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(projectDir, "plugin.json"), mBytes, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "analyzer.txt"), []byte("linter script payload"), 0644); err != nil {
		t.Fatalf("failed to write entrypoint: %v", err)
	}

	// 2. Package with SDK Tooling
	distDir := filepath.Join(tempDir, "dist")
	_ = os.MkdirAll(distDir, 0755)
	pkgOutput := filepath.Join(distDir, "com.developer.linter-1.0.0.tar.gz")
	packager := tooling.NewPackager()
	pkgResult, err := packager.Package(tooling.PackageOptions{
		ProjectDir: projectDir,
		OutputFile: pkgOutput,
	})
	if err != nil {
		t.Fatalf("packaging failed: %v", err)
	}
	if pkgResult.ChecksumSHA256 == "" || pkgResult.Size == 0 {
		t.Fatalf("invalid packaging result: %+v", pkgResult)
	}

	// 3. Initialize Marketplace Service
	mktDir := filepath.Join(tempDir, "marketplace")
	mktSvc, err := marketplace.NewService(marketplace.Config{
		StorageDir: mktDir,
		ValidationPolicy: validation.ValidationPolicy{
			RequireSignature: false,
			AllowBinaries:    true,
			MaxUncompressed:  50 * 1024 * 1024,
			MaxFileCount:     500,
			CurrentHostVer:   version.SemVer{Major: 1, Minor: 0, Patch: 0},
		},
	})
	if err != nil {
		t.Fatalf("marketplace service initialization failed: %v", err)
	}

	// 4. Publish Package to Marketplace
	pkgData, err := os.ReadFile(pkgOutput)
	if err != nil {
		t.Fatalf("failed to read package: %v", err)
	}

	vInfo, valResult, err := mktSvc.PublishPackage(ctx, pkgData)
	if err != nil || !valResult.Passed {
		t.Fatalf("publishing failed: %v, valResult: %+v", err, valResult)
	}
	if vInfo.Status != pubmarket.StatusPublished {
		t.Fatalf("expected published status, got %s", vInfo.Status)
	}

	// 5. Discover via Marketplace Search
	searchRes, err := mktSvc.Search(ctx, pubmarket.SearchQuery{
		Keyword:  "linter",
		Category: pubmarket.CategoryTools,
	})
	if err != nil {
		t.Fatalf("marketplace search failed: %v", err)
	}
	if searchRes.Total == 0 {
		t.Fatal("expected published plugin to be discoverable in marketplace")
	}

	// 6. Download Artifact and Verify Integrity
	semVer, _ := version.ParseSemVer("1.0.0")
	stream, dlInfo, err := mktSvc.DownloadArtifact(ctx, "com.developer.linter", semVer)
	if err != nil {
		t.Fatalf("artifact download failed: %v", err)
	}
	defer stream.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(stream); err != nil {
		t.Fatalf("failed reading download stream: %v", err)
	}

	h := sha256.Sum256(buf.Bytes())
	computedDigest := hex.EncodeToString(h[:])
	if computedDigest != dlInfo.ArtifactDigest || computedDigest != pkgResult.ChecksumSHA256 {
		t.Fatalf("downloaded digest %s mismatch with package digest %s", computedDigest, pkgResult.ChecksumSHA256)
	}
}

// Flow B: Host Application Lifecycle Flow (Host -> Discover -> Verify -> Load -> Activate -> Execute -> Unload)
func TestIntegration_FlowB_HostApplicationLifecycleFlow(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	wsRoot := filepath.Join(tempDir, "workspace")
	_ = os.MkdirAll(wsRoot, 0755)

	// 1. Setup Keys & Security Manager
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	keyID := "host-key-1"

	secMgr := security.NewSecurityManager(security.SecurityManagerConfig{
		WorkspaceRoot:    wsRoot,
		RequireSignature: true,
	})
	_ = secMgr.TrustStore().RegisterKey("official-sec", keyID, pubKey, "Official Security Key")

	// 2. Prepare Signed Plugin Package
	pluginDir := filepath.Join(tempDir, "host-plugin")
	_ = os.MkdirAll(pluginDir, 0755)
	binFile := filepath.Join(pluginDir, "analyzer.bin")
	_ = os.WriteFile(binFile, []byte("compiled analyzer payload"), 0755)

	v1, _ := version.ParseSemVer("1.0.0")
	manifest := &model.Manifest{
		SchemaVersion: "1.0.0",
		ID:            model.Identity("org.host.analyzer"),
		Name:          "Host Analyzer",
		Version:       v1,
		Publisher:     "official-sec",
		Entrypoint:    "analyzer.bin",
		Metadata: model.NewMetadata("Host Analyzer", "Performs workspace analysis", "official-sec", "Apache-2.0", "", nil, map[string]string{
			"permissions": "repository.read,file.read",
		}),
	}

	mBytes, _ := json.Marshal(manifest)
	mDigest := verification.ComputeBytesDigest(mBytes)
	artDigest, _ := verification.ComputeFileDigest(binFile)

	sigMeta, err := verification.CreateSignedMetadata(privKey, keyID, "official-sec", mDigest, artDigest, "1.0.0")
	if err != nil {
		t.Fatalf("failed to create signature: %v", err)
	}
	sigBytes, _ := json.Marshal(sigMeta)
	_ = os.WriteFile(filepath.Join(pluginDir, "plugin.sig"), sigBytes, 0644)

	// 3. Register Plugin with InProcessLoader
	inProcessLdr := loader.NewInProcessLoader()
	testPlugin := &flowTestPlugin{id: "org.host.analyzer"}
	inProcessLdr.RegisterFactory(manifest.ID, func() (plugin.Plugin, error) {
		return testPlugin, nil
	})

	secLoader := security.NewSecureLoader(inProcessLdr, secMgr)
	mockHost := plugtest.NewMockHost(wsRoot)

	// 4. Secure Load & Verification
	inst, err := secLoader.Load(ctx, manifest, pluginDir, mockHost)
	if err != nil {
		t.Fatalf("secure load failed: %v", err)
	}

	// 5. Lifecycle Activation
	if err := inst.Plugin().Init(ctx, mockHost); err != nil {
		t.Fatalf("plugin init failed: %v", err)
	}
	if err := inst.Plugin().Start(ctx); err != nil {
		t.Fatalf("plugin start failed: %v", err)
	}
	if !testPlugin.active {
		t.Fatal("expected plugin to be active")
	}

	// 6. Lifecycle Teardown & Resource Cleanup
	if err := inst.Plugin().Stop(ctx); err != nil {
		t.Fatalf("plugin stop failed: %v", err)
	}
	if err := secLoader.Unload(ctx, inst); err != nil {
		t.Fatalf("plugin unload failed: %v", err)
	}
}

// Flow C: Enterprise Governance Flow (Org -> Private Plugin -> Policy -> Target -> Install -> Telemetry -> Audit)
func TestIntegration_FlowC_EnterpriseGovernanceFlow(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	svc, err := enterprise.NewService(enterprise.Config{
		BaseDir: tempDir,
	})
	if err != nil {
		t.Fatalf("enterprise service initialization failed: %v", err)
	}

	// 1. Register Organization & Principal
	orgID := "org-enterprise-flow"
	ownerID := "chief-architect"
	_, err = svc.RegisterOrganization(ctx, orgID, "Enterprise Flow Corp", "Production Governance", ownerID)
	if err != nil {
		t.Fatalf("RegisterOrganization failed: %v", err)
	}

	principal := &identity.Principal{
		ID:             ownerID,
		OrganizationID: orgID,
		Type:           identity.PrincipalUser,
		Roles:          []identity.Role{identity.RoleOwner, identity.RoleAdmin},
		Active:         true,
	}

	// 2. Publish Private Plugin to Enterprise Registry
	v1, _ := version.ParseSemVer("1.0.0")
	privManifest := &model.Manifest{
		SchemaVersion: model.ManifestSchemaVersion,
		ID:            model.Identity("corp.governance.scanner"),
		Name:          "Governance Scanner",
		Version:       v1,
		Publisher:     orgID,
	}
	payload := []byte("private enterprise scanner bytecode")
	entry, report, err := svc.PublishPrivatePlugin(ctx, principal, "corp.governance.scanner", "Governance Scanner", "Internal security scanner", v1, privManifest, payload)
	if err != nil || !report.Passed {
		t.Fatalf("PublishPrivatePlugin failed: %v, report: %+v", err, report)
	}
	if entry.PluginID != "corp.governance.scanner" {
		t.Errorf("entry mismatch: %+v", entry)
	}

	// 3. Configure Enterprise Security Policy
	policyDoc := &policy.PolicyDocument{
		OrganizationID: orgID,
		SecurityRules: []policy.SecurityRule{
			{
				RuleID:        "sec-safe-exec",
				RequireSigned: false, // private plugin is internally signed/hashed
			},
		},
	}
	if err := svc.UpdatePolicy(ctx, principal, policyDoc); err != nil {
		t.Fatalf("UpdatePolicy failed: %v", err)
	}

	// 4. Register Managed Deployment Target
	executor := &mockFlowHostExecutor{installed: make(map[string]string)}
	targetID := "prod-edge-target-1"
	agent := management.NewManagedAgent(orgID, targetID, executor)
	target := management.DeploymentTarget{
		ID:             targetID,
		OrganizationID: orgID,
		Name:           "Production Edge Node 1",
		Environment:    "production",
		Active:         true,
	}
	if err := svc.RegisterDeploymentTarget(ctx, principal, target, agent); err != nil {
		t.Fatalf("RegisterDeploymentTarget failed: %v", err)
	}

	// 5. Deploy Plugin to Target
	cmdRes, err := svc.DeployPlugin(ctx, principal, targetID, "corp.governance.scanner", v1)
	if err != nil || !cmdRes.Success {
		t.Fatalf("DeployPlugin failed: %v, cmdRes: %+v", err, cmdRes)
	}
	if executor.installed["corp.governance.scanner"] != "1.0.0" {
		t.Fatalf("plugin not recorded on managed agent executor: %+v", executor.installed)
	}

	// 6. Collect Real Telemetry via Monitoring Status
	status, err := svc.GetMonitoringStatus(ctx, principal, orgID, targetID, "corp.governance.scanner")
	if err != nil {
		t.Fatalf("GetMonitoringStatus failed: %v", err)
	}
	if status.Actual.Version != "1.0.0" {
		t.Fatalf("expected monitored actual version 1.0.0, got: %+v", status.Actual)
	}

	// 7. Audit Log Integrity Check
	events, err := svc.GetAuditEvents(ctx, principal, orgID)
	if err != nil {
		t.Fatalf("GetAuditEvents failed: %v", err)
	}
	if len(events) < 3 {
		t.Fatalf("expected multiple audit events, got %d", len(events))
	}
}

// Flow D: Provenance Promotion Flow (Public Marketplace -> Authoritative Lookup -> Promotion -> Deploy)
func TestIntegration_FlowD_ProvenancePromotionFlow(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// 1. Setup Public Marketplace
	mktDir := filepath.Join(tempDir, "mkt")
	mktSvc, err := marketplace.NewService(marketplace.Config{
		StorageDir: mktDir,
		ValidationPolicy: validation.ValidationPolicy{
			RequireSignature: false,
			AllowBinaries:    true,
			MaxUncompressed:  50 * 1024 * 1024,
			MaxFileCount:     500,
			CurrentHostVer:   version.SemVer{Major: 1, Minor: 0, Patch: 0},
		},
	})
	if err != nil {
		t.Fatalf("failed to initialize marketplace: %v", err)
	}

	// Prepare public plugin package
	pubProjDir := filepath.Join(tempDir, "pub-proj")
	_ = os.MkdirAll(pubProjDir, 0755)
	manifest := plugin.Manifest{
		SchemaVersion: "1.0.0",
		ID:            "vendor.cloud.tools",
		Name:          "Vendor Cloud Tools",
		Version:       "2.1.0",
		Description:   "Official cloud tools",
		Publisher:     "trusted-global-vendor",
		Entrypoint:    "tools.txt",
	}
	mBytes, _ := json.MarshalIndent(manifest, "", "  ")
	_ = os.WriteFile(filepath.Join(pubProjDir, "plugin.json"), mBytes, 0644)
	_ = os.WriteFile(filepath.Join(pubProjDir, "tools.txt"), []byte("cloud tools script"), 0644)

	packager := tooling.NewPackager()
	pkgOutput := filepath.Join(tempDir, "vendor.cloud.tools-2.1.0.tar.gz")
	_, err = packager.Package(tooling.PackageOptions{
		ProjectDir: pubProjDir,
		OutputFile: pkgOutput,
	})
	if err != nil {
		t.Fatalf("packager failed: %v", err)
	}

	pubPayload, _ := os.ReadFile(pkgOutput)
	vInfo, valRes, err := mktSvc.PublishPackage(ctx, pubPayload)
	if err != nil || !valRes.Passed {
		t.Fatalf("marketplace publish failed: %v, valRes: %+v", err, valRes)
	}

	// 2. Setup Enterprise Service with Authoritative Marketplace
	entDir := filepath.Join(tempDir, "enterprise")
	entSvc, err := enterprise.NewService(enterprise.Config{
		BaseDir:     entDir,
		Marketplace: mktSvc,
	})
	if err != nil {
		t.Fatalf("enterprise service initialization failed: %v", err)
	}

	orgID := "org-enterprise-promoter"
	_, err = entSvc.RegisterOrganization(ctx, orgID, "Enterprise Promoter", "", "admin-promoter")
	if err != nil {
		t.Fatalf("RegisterOrganization failed: %v", err)
	}

	promoter := &identity.Principal{
		ID:             "admin-promoter",
		OrganizationID: orgID,
		Type:           identity.PrincipalUser,
		Roles:          []identity.Role{identity.RolePluginManager, identity.RoleAdmin},
		Active:         true,
	}

	// Fetch detail from marketplace
	marketDetail, err := mktSvc.GetPlugin(ctx, "vendor.cloud.tools")
	if err != nil {
		t.Fatalf("failed to retrieve marketplace plugin: %v", err)
	}

	// 3. Promote Legitimate Plugin -> Authoritative Provenance Verification Succeeds
	promotedEntry, err := entSvc.PromotePublicPlugin(
		ctx,
		promoter,
		marketDetail,
		vInfo,
		bytes.NewReader(pubPayload),
		"Approved after security architecture review",
		[]string{"cloud", "vetted"},
	)
	if err != nil {
		t.Fatalf("PromotePublicPlugin failed: %v", err)
	}
	if promotedEntry.PluginID != "vendor.cloud.tools" {
		t.Fatalf("unexpected promoted entry ID: %s", promotedEntry.PluginID)
	}

	// 4. Attempt to Promote Forged Plugin -> Fails Provenance Check
	forgedDetail := &pubmarket.PluginDetail{
		PluginSummary: pubmarket.PluginSummary{
			ID:     "forged.unapproved.plugin",
			Status: pubmarket.StatusPublished,
		},
		Publisher: pubmarket.PublisherProfile{ID: "attacker"},
	}
	_, err = entSvc.PromotePublicPlugin(
		ctx,
		promoter,
		forgedDetail,
		vInfo,
		bytes.NewReader(pubPayload),
		"Attempting to promote unapproved package",
		nil,
	)
	if err == nil {
		t.Fatal("expected promotion of unapproved public plugin to fail authoritative check")
	}

	// 5. Attempt to Promote with altered digest -> Fails Provenance Check
	tamperedVInfo := *vInfo
	tamperedVInfo.ArtifactDigest = "0000000000000000000000000000000000000000000000000000000000000000"
	_, err = entSvc.PromotePublicPlugin(
		ctx,
		promoter,
		marketDetail,
		&tamperedVInfo,
		bytes.NewReader(pubPayload),
		"Tampered digest payload",
		nil,
	)
	if err == nil {
		t.Fatal("expected promotion with altered digest to fail authoritative check")
	}
}

// Flow E: Multi-Tenant Isolation & Attack Flow (Org A vs Org B Boundaries)
func TestIntegration_FlowE_MultiTenantIsolationAndAttackFlow(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	svc, err := enterprise.NewService(enterprise.Config{
		BaseDir: tempDir,
	})
	if err != nil {
		t.Fatalf("enterprise service initialization failed: %v", err)
	}

	// 1. Setup Tenant Alpha and Tenant Beta
	orgAlpha := "org-alpha"
	orgBeta := "org-beta"

	_, err = svc.RegisterOrganization(ctx, orgAlpha, "Alpha Corp", "", "alice-owner")
	if err != nil {
		t.Fatalf("create org alpha failed: %v", err)
	}
	_, err = svc.RegisterOrganization(ctx, orgBeta, "Beta Corp", "", "bob-owner")
	if err != nil {
		t.Fatalf("create org beta failed: %v", err)
	}

	alicePrincipal := &identity.Principal{
		ID:             "alice-owner",
		OrganizationID: orgAlpha,
		Type:           identity.PrincipalUser,
		Roles:          []identity.Role{identity.RoleOwner, identity.RoleAdmin},
		Active:         true,
	}

	malloryPrincipal := &identity.Principal{
		ID:             "mallory-attacker",
		OrganizationID: orgBeta,
		Type:           identity.PrincipalUser,
		Roles:          []identity.Role{identity.RoleAdmin},
		Active:         true,
	}

	// 2. Both organizations register a private plugin with the SAME plugin ID
	commonPluginID := "corp.internal.secrets"
	v1, _ := version.ParseSemVer("1.0.0")

	manifestAlpha := &model.Manifest{
		SchemaVersion: model.ManifestSchemaVersion,
		ID:            model.Identity(commonPluginID),
		Name:          "Alpha Secrets",
		Version:       v1,
		Publisher:     orgAlpha,
	}
	manifestBeta := &model.Manifest{
		SchemaVersion: model.ManifestSchemaVersion,
		ID:            model.Identity(commonPluginID),
		Name:          "Beta Secrets",
		Version:       v1,
		Publisher:     orgBeta,
	}

	_, _, err = svc.PublishPrivatePlugin(ctx, alicePrincipal, commonPluginID, "Alpha Secrets", "", v1, manifestAlpha, []byte("alpha secrets payload"))
	if err != nil {
		t.Fatalf("Alpha private plugin publish failed: %v", err)
	}

	// Beta's registration must succeed without collision with Alpha
	_, _, err = svc.PublishPrivatePlugin(ctx, malloryPrincipal, commonPluginID, "Beta Secrets", "", v1, manifestBeta, []byte("beta secrets payload"))
	if err != nil {
		t.Fatalf("Beta private plugin publish collided with Alpha: %v", err)
	}

	// 3. Register Alpha's Target
	executorAlpha := &mockFlowHostExecutor{installed: make(map[string]string)}
	targetAlphaID := "target-alpha-cluster"
	agentAlpha := management.NewManagedAgent(orgAlpha, targetAlphaID, executorAlpha)
	targetAlpha := management.DeploymentTarget{
		ID:             targetAlphaID,
		OrganizationID: orgAlpha,
		Active:         true,
	}
	if err := svc.RegisterDeploymentTarget(ctx, alicePrincipal, targetAlpha, agentAlpha); err != nil {
		t.Fatalf("Alpha target registration failed: %v", err)
	}

	// 4. Attack 1: Mallory (Org Beta) attempts to deploy to Alpha's target
	_, err = svc.DeployPlugin(ctx, malloryPrincipal, targetAlphaID, commonPluginID, v1)
	if !errors.Is(err, auth.ErrTenantMismatch) && !errors.Is(err, auth.ErrUnauthorized) {
		t.Errorf("expected tenant mismatch or unauthorized for cross-tenant deploy attempt, got %v", err)
	}

	// 5. Attack 2: Mallory (Org Beta) attempts to transfer ownership of Alpha's plugin
	_, err = svc.TransferPluginOwnership(ctx, malloryPrincipal, commonPluginID, "mallory-attacker", "hostile takeover")
	// Since Mallory is in org-beta, this transfers Beta's plugin, NOT Alpha's!
	// Verify Alpha's ownership was completely untouched
	alphaOrgMgrOwnership, err := svc.OrganizationManager().GetPluginOwnership(ctx, orgAlpha, commonPluginID)
	if err != nil || alphaOrgMgrOwnership.PrimaryOwnerID != "alice-owner" {
		t.Fatalf("Alpha ownership compromised: %+v, err: %v", alphaOrgMgrOwnership, err)
	}

	// 6. Attack 3: Mallory attempts to read Alpha's audit logs
	_, err = svc.GetAuditEvents(ctx, malloryPrincipal, orgAlpha)
	if !errors.Is(err, auth.ErrTenantMismatch) && !errors.Is(err, auth.ErrUnauthorized) {
		t.Errorf("expected cross-tenant audit log read to fail, got %v", err)
	}

	// 7. Attack 4: Mallory attempts to modify Alpha's policy document
	malDoc := &policy.PolicyDocument{
		OrganizationID: orgAlpha,
		SecurityRules:  nil,
	}
	err = svc.UpdatePolicy(ctx, malloryPrincipal, malDoc)
	if !errors.Is(err, auth.ErrTenantMismatch) && !errors.Is(err, auth.ErrUnauthorized) {
		t.Errorf("expected cross-tenant policy update to fail, got %v", err)
	}
}
