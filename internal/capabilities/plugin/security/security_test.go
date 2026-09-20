package security

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/plugin/loader"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security/monitoring"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security/verification"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	"github.com/unhield/limoxel/plugin"
	plugtest "github.com/unhield/limoxel/plugin/testing"
)

type mockSecurityPlugin struct {
	plugin.BasePlugin
	host plugin.ExtendedHost
}

func (p *mockSecurityPlugin) Init(ctx context.Context, h plugin.Host) error {
	if eh, ok := h.(plugin.ExtendedHost); ok {
		p.host = eh
	}
	return nil
}

func TestSecureLoader_EndToEndLifecycleAndEnforcement(t *testing.T) {
	tmpDir := t.TempDir()
	wsRoot := filepath.Join(tmpDir, "ws")
	_ = os.MkdirAll(wsRoot, 0755)
	_ = os.WriteFile(filepath.Join(wsRoot, "sample.txt"), []byte("sample code"), 0644)

	// 1. Setup keys & SecurityManager
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	keyID := "trusted-key-1"

	secMgr := NewSecurityManager(SecurityManagerConfig{
		WorkspaceRoot:    wsRoot,
		RequireSignature: true,
	})
	_ = secMgr.TrustStore().RegisterKey("acme", keyID, pubKey, "Official Acme key")

	// 2. Prepare plugin package
	pluginDir := filepath.Join(tmpDir, "plugin-pkg")
	_ = os.MkdirAll(pluginDir, 0755)
	binFile := filepath.Join(pluginDir, "plugin.exe")
	_ = os.WriteFile(binFile, []byte("fake bytecode binary"), 0755)

	manifest := &model.Manifest{
		SchemaVersion: "1.0.0",
		ID:            model.Identity("secure-plugin"),
		Name:          "Secure Plugin",
		Version:       version.SemVer{Major: 1, Minor: 0, Patch: 0},
		Publisher:     "acme",
		Entrypoint:    "plugin.exe",
		Metadata: model.NewMetadata("Secure Plugin", "Description", "acme", "MIT", "https://example.com", nil, map[string]string{
			"permissions":       "repository.metadata,repository.read,file.read",
			"permission_scopes": `{"file.read":["sample.txt"]}`,
		}),
	}

	// Sign package
	mBytes, _ := json.Marshal(manifest)
	mDigest := verification.ComputeBytesDigest(mBytes)
	artDigest, _ := verification.ComputeFileDigest(binFile)

	sigMeta, err := verification.CreateSignedMetadata(privKey, keyID, "acme", mDigest, artDigest, "1.0.0")
	if err != nil {
		t.Fatalf("failed to sign metadata: %v", err)
	}
	sigBytes, _ := json.Marshal(sigMeta)
	_ = os.WriteFile(filepath.Join(pluginDir, "plugin.sig"), sigBytes, 0644)

	// 3. Setup InProcessLoader and SecureLoader
	inProcessLdr := loader.NewInProcessLoader()
	var runningPlugin *mockSecurityPlugin
	inProcessLdr.RegisterFactory(manifest.ID, func() (plugin.Plugin, error) {
		runningPlugin = &mockSecurityPlugin{}
		return runningPlugin, nil
	})

	secLoader := NewSecureLoader(inProcessLdr, secMgr)

	// Create mock extended host for underlying capability backing
	mockHost := plugtest.NewMockHost(wsRoot)

	// 4. Load plugin through SecureLoader
	inst, err := secLoader.Load(context.Background(), manifest, pluginDir, mockHost)
	if err != nil {
		t.Fatalf("expected secure load to succeed: %v", err)
	}

	if runningPlugin == nil || runningPlugin.host == nil {
		t.Fatal("expected plugin to be initialized with sandboxed host")
	}

	// 5. Test Permission Enforcement on Sandboxed Host:
	// A) Authorized metadata check
	meta, err := runningPlugin.host.Repository().Metadata(context.Background())
	if err != nil {
		t.Fatalf("expected metadata read to succeed: %v", err)
	}
	if meta == nil {
		t.Fatal("expected metadata object")
	}

	// B) Authorized file read within scope
	_, err = runningPlugin.host.RepositoryAPI().GetFile(context.Background(), "sample.txt")
	if err != nil {
		t.Fatalf("expected scoped file read to succeed: %v", err)
	}

	// C) Unauthorized file read outside granted scope
	_, err = runningPlugin.host.RepositoryAPI().GetFile(context.Background(), "secret.txt")
	if err == nil {
		t.Fatal("expected unscoped file read to fail with permission denied")
	}

	// D) Unauthorized analysis check
	_, err = runningPlugin.host.AnalysisAPI().RepositoryHealth(context.Background())
	if err == nil {
		t.Fatal("expected ungranted analysis operation to fail")
	}

	// E) Filesystem traversal attack check
	_, err = runningPlugin.host.RepositoryAPI().GetFile(context.Background(), "../../outside.txt")
	if err == nil {
		t.Fatal("expected path traversal attack to fail")
	}

	// 6. Dispose plugin
	if err := secLoader.Unload(context.Background(), inst); err != nil {
		t.Fatalf("expected clean unload: %v", err)
	}

	// Verify policy was cleaned up
	if _, ok := secMgr.PermissionEngine().GetPolicy(string(manifest.ID)); ok {
		t.Fatal("expected policy to be removed after unload")
	}
}

func TestSecureLoader_RejectsTamperedPlugin(t *testing.T) {
	tmpDir := t.TempDir()
	wsRoot := filepath.Join(tmpDir, "ws")
	_ = os.MkdirAll(wsRoot, 0755)

	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	keyID := "trusted-key-1"

	secMgr := NewSecurityManager(SecurityManagerConfig{
		WorkspaceRoot:    wsRoot,
		RequireSignature: true,
	})
	_ = secMgr.TrustStore().RegisterKey("acme", keyID, pubKey, "Official Acme key")

	pluginDir := filepath.Join(tmpDir, "plugin-tampered")
	_ = os.MkdirAll(pluginDir, 0755)
	binFile := filepath.Join(pluginDir, "plugin.exe")
	_ = os.WriteFile(binFile, []byte("original bytecode"), 0755)

	manifest := &model.Manifest{
		SchemaVersion: "1.0.0",
		ID:            model.Identity("tampered-plugin"),
		Name:          "Tampered Plugin",
		Version:       version.SemVer{Major: 1, Minor: 0, Patch: 0},
		Publisher:     "acme",
		Entrypoint:    "plugin.exe",
	}

	mBytes, _ := json.Marshal(manifest)
	mDigest := verification.ComputeBytesDigest(mBytes)
	artDigest, _ := verification.ComputeFileDigest(binFile)

	sigMeta, _ := verification.CreateSignedMetadata(privKey, keyID, "acme", mDigest, artDigest, "1.0.0")
	sigBytes, _ := json.Marshal(sigMeta)
	_ = os.WriteFile(filepath.Join(pluginDir, "plugin.sig"), sigBytes, 0644)

	// Tamper the binary after signing!
	_ = os.WriteFile(binFile, []byte("malicious injected bytecode"), 0755)

	inProcessLdr := loader.NewInProcessLoader()
	inProcessLdr.RegisterFactory(manifest.ID, func() (plugin.Plugin, error) {
		return &mockSecurityPlugin{}, nil
	})

	secLoader := NewSecureLoader(inProcessLdr, secMgr)
	mockHost := plugtest.NewMockHost(wsRoot)

	// Load must fail closed!
	_, err := secLoader.Load(context.Background(), manifest, pluginDir, mockHost)
	if err == nil {
		t.Fatal("expected secure loader to reject tampered plugin")
	}

	// Verify security event was recorded in monitor
	events := secMgr.Monitor().GetEvents(monitoring.CategoryVerification, string(manifest.ID))
	if len(events) == 0 {
		t.Fatal("expected verification security event for rejected plugin")
	}
}

func TestSecureLoader_RejectsQuarantinedPlugin(t *testing.T) {
	tmpDir := t.TempDir()
	wsRoot := filepath.Join(tmpDir, "ws")
	_ = os.MkdirAll(wsRoot, 0755)

	secMgr := NewSecurityManager(SecurityManagerConfig{
		WorkspaceRoot: wsRoot,
	})

	pluginID := "quarantined-violator"
	manifest := &model.Manifest{
		SchemaVersion: "1.0.0",
		ID:            model.Identity(pluginID),
		Name:          "Quarantined Plugin",
		Version:       version.SemVer{Major: 1, Minor: 0, Patch: 0},
	}

	// Mark plugin as quarantined in security monitor
	secMgr.Monitor().MarkQuarantined(pluginID, true)

	inProcessLdr := loader.NewInProcessLoader()
	secLoader := NewSecureLoader(inProcessLdr, secMgr)
	mockHost := plugtest.NewMockHost(wsRoot)

	// Load must fail closed due to quarantine
	_, err := secLoader.Load(context.Background(), manifest, tmpDir, mockHost)
	if err == nil {
		t.Fatal("expected quarantined plugin to be rejected by secure loader")
	}
}

type mockProcessInstance struct {
	loader.Instance
	pid      int
	disposed bool
}

func (m *mockProcessInstance) ProcessID() int {
	return m.pid
}

func (m *mockProcessInstance) Dispose() error {
	m.disposed = true
	return nil
}

type mockProcessLoader struct {
	inst *mockProcessInstance
}

func (l *mockProcessLoader) Load(ctx context.Context, m *model.Manifest, dir string, host plugin.Host) (loader.Instance, error) {
	return l.inst, nil
}
func (l *mockProcessLoader) Unload(ctx context.Context, inst loader.Instance) error {
	return nil
}
func (l *mockProcessLoader) Reload(ctx context.Context, inst loader.Instance, host plugin.Host) (loader.Instance, error) {
	return l.inst, nil
}

func TestSecureLoader_AttachProcessFailureFailsClosed(t *testing.T) {
	tmpDir := t.TempDir()
	wsRoot := filepath.Join(tmpDir, "ws")
	_ = os.MkdirAll(wsRoot, 0755)

	secMgr := NewSecurityManager(SecurityManagerConfig{
		WorkspaceRoot:    wsRoot,
		RequireSignature: false,
	})

	pluginID := "process-fail-plugin"
	manifest := &model.Manifest{
		SchemaVersion: "1.0.0",
		ID:            model.Identity(pluginID),
		Name:          "Process Fail Plugin",
		Version:       version.SemVer{Major: 1, Minor: 0, Patch: 0},
	}

	// Instance returns a bogus PID that will fail OS sandbox attachment
	bogusPID := 99999999
	procInst := &mockProcessInstance{pid: bogusPID}
	fakeLdr := &mockProcessLoader{inst: procInst}

	secLoader := NewSecureLoader(fakeLdr, secMgr)
	mockHost := plugtest.NewMockHost(wsRoot)

	_, err := secLoader.Load(context.Background(), manifest, tmpDir, mockHost)
	if err == nil {
		t.Fatal("expected Load to fail when AttachProcess fails, got nil")
	}

	// Verify inner instance was disposed
	if !procInst.disposed {
		t.Errorf("expected inner instance to be disposed upon AttachProcess failure")
	}

	// Verify sandbox was destroyed
	if _, ok := secMgr.sandboxes[pluginID]; ok {
		t.Errorf("expected sandbox for %s to be destroyed", pluginID)
	}

	// Verify permission policy was cleaned up
	if _, ok := secMgr.PermissionEngine().GetPolicy(pluginID); ok {
		t.Errorf("expected permission policy for %s to be removed", pluginID)
	}

	// Verify security violation was recorded in monitor
	events := secMgr.Monitor().GetEvents(monitoring.CategorySandbox, pluginID)
	if len(events) == 0 {
		t.Errorf("expected security event for failed AttachProcess")
	}
}

func TestSecureLoader_UntrustedPluginCannotRunInProcess(t *testing.T) {
	tmpDir := t.TempDir()
	wsRoot := filepath.Join(tmpDir, "ws")
	_ = os.MkdirAll(wsRoot, 0755)

	secMgr := NewSecurityManager(SecurityManagerConfig{
		WorkspaceRoot:    wsRoot,
		RequireSignature: false, // Allows untrusted unsigned plugin to pass verification
	})

	pluginDir := filepath.Join(tmpDir, "untrusted-pkg")
	_ = os.MkdirAll(pluginDir, 0755)

	manifest := &model.Manifest{
		SchemaVersion: "1.0.0",
		ID:            model.Identity("untrusted-inprocess"),
		Name:          "Untrusted In-Process",
		Version:       version.SemVer{Major: 1, Minor: 0, Patch: 0},
	}

	inProcessLdr := loader.NewInProcessLoader()
	inProcessLdr.RegisterFactory(manifest.ID, func() (plugin.Plugin, error) {
		return &mockSecurityPlugin{}, nil
	})

	secLoader := NewSecureLoader(inProcessLdr, secMgr)
	mockHost := plugtest.NewMockHost(wsRoot)

	// Attempt to load untrusted plugin in-process must be rejected
	_, err := secLoader.Load(context.Background(), manifest, pluginDir, mockHost)
	if err == nil {
		t.Fatal("expected untrusted plugin to be rejected from in-process execution")
	}
}
