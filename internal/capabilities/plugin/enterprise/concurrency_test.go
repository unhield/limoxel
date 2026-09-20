package enterprise

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/identity"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/management"
	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/policy"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

func TestEnterprise_ConcurrentOperations(t *testing.T) {
	tempDir := t.TempDir()
	svc, err := NewService(Config{BaseDir: tempDir})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	ctx := context.Background()
	_, err = svc.RegisterOrganization(ctx, "org-concurrent", "Concurrent Org", "", "owner-1")
	if err != nil {
		t.Fatalf("RegisterOrganization failed: %v", err)
	}

	ownerPrincipal := &identity.Principal{
		ID:             "owner-1",
		OrganizationID: "org-concurrent",
		Type:           identity.PrincipalUser,
		Roles:          []identity.Role{identity.RoleOwner},
		Active:         true,
	}

	executor := &mockHostExecutor{installed: make(map[string]string)}
	agent := management.NewManagedAgent("org-concurrent", "node-con-1", executor)
	_ = svc.RegisterDeploymentTarget(ctx, ownerPrincipal, management.DeploymentTarget{
		ID:             "node-con-1",
		OrganizationID: "org-concurrent",
		Environment:    "production",
		Active:         true,
	}, agent)

	// Concurrently publish and deploy multiple plugins
	const workerCount = 10
	var wg sync.WaitGroup
	errCh := make(chan error, workerCount*2)

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			pluginID := fmt.Sprintf("concurrent.plugin.%d", idx)
			v, _ := version.ParseSemVer("1.0.0")
			manifest := &model.Manifest{
				SchemaVersion: model.ManifestSchemaVersion,
				ID:            model.Identity(pluginID),
				Name:          fmt.Sprintf("Plugin %d", idx),
				Version:       v,
				Publisher:     "org-concurrent",
			}
			payload := fmt.Appendf(nil, "archive payload %d", idx)

			// 1. Publish
			_, rep, pubErr := svc.PublishPrivatePlugin(
				ctx,
				ownerPrincipal,
				pluginID,
				fmt.Sprintf("Plugin %d", idx),
				"",
				v,
				manifest,
				payload,
			)
			if pubErr != nil || (rep != nil && !rep.Passed) {
				errCh <- fmt.Errorf("worker %d publish failed: %v", idx, pubErr)
				return
			}

			// 2. Deploy
			_, depErr := svc.DeployPlugin(ctx, ownerPrincipal, "node-con-1", pluginID, v)
			if depErr != nil {
				errCh <- fmt.Errorf("worker %d deploy failed: %v", idx, depErr)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent worker error: %v", err)
	}

	// Verify Audit Chain Integrity after all concurrent writes
	valid, err := svc.AuditLogger().VerifyIntegrity()
	if err != nil || !valid {
		t.Errorf("audit chain integrity compromised after concurrent execution: %v", err)
	}
}

func TestEnterprise_PolicyDenialBlocksDeployment(t *testing.T) {
	tempDir := t.TempDir()
	svc, _ := NewService(Config{BaseDir: tempDir})
	ctx := context.Background()

	_, _ = svc.RegisterOrganization(ctx, "org-blocked", "Blocked Org", "", "owner-1")
	owner := &identity.Principal{
		ID:             "owner-1",
		OrganizationID: "org-blocked",
		Type:           identity.PrincipalUser,
		Roles:          []identity.Role{identity.RoleOwner},
		Active:         true,
	}

	v1, _ := version.ParseSemVer("1.0.0")
	manifest := &model.Manifest{
		SchemaVersion: model.ManifestSchemaVersion,
		ID:            model.Identity("blocked.tool"),
		Name:          "Blocked Tool",
		Version:       v1,
		Publisher:     "org-blocked",
	}

	_, _, err := svc.PublishPrivatePlugin(ctx, owner, "blocked.tool", "Tool", "", v1, manifest, []byte("pkg"))
	if err != nil {
		t.Fatalf("PublishPrivatePlugin failed: %v", err)
	}

	// Configure denylist policy
	doc := &policy.PolicyDocument{
		OrganizationID: "org-blocked",
		DenylistRules: []policy.DenylistRule{
			{RuleID: "deny-1", PluginIDs: []string{"blocked.*"}, Reason: "banned prefix"},
		},
	}
	_ = svc.UpdatePolicy(ctx, owner, doc)

	executor := &mockHostExecutor{installed: make(map[string]string)}
	agent := management.NewManagedAgent("org-blocked", "node-1", executor)
	_ = svc.RegisterDeploymentTarget(ctx, owner, management.DeploymentTarget{
		ID:             "node-1",
		OrganizationID: "org-blocked",
		Active:         true,
	}, agent)

	// Deployment must be blocked by policy
	res, err := svc.DeployPlugin(ctx, owner, "node-1", "blocked.tool", v1)
	if err == nil || res.Success {
		t.Errorf("expected policy to block deployment of blocked.tool")
	}
}
