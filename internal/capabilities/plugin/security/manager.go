package security

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security/monitoring"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security/permissions"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security/sandbox"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security/verification"
	pubsec "github.com/unhield/limoxel/plugin/security"
)

// SecurityManagerConfig provides initialization options for the central security manager.
type SecurityManagerConfig struct {
	WorkspaceRoot    string
	DataRoot         string // Root directory for plugin private storage
	TempRoot         string // Root directory for ephemeral scratch files
	RequireSignature bool   // Enforce digital signatures for all plugins
	DefaultLimits    sandbox.ResourceLimits
	RecoveryConfig   monitoring.RecoveryConfig
}

// SecurityManager coordinates verification, permission enforcement, sandboxing, and security monitoring.
type SecurityManager struct {
	mu         sync.RWMutex
	cfg        SecurityManagerConfig
	trustStore *verification.TrustStore
	verifier   *verification.Verifier
	permEngine *permissions.Engine
	monitor    *monitoring.SecurityMonitor
	recovery   *monitoring.RecoveryManager
	sandboxes  map[string]sandbox.PluginSandbox
}

// NewSecurityManager constructs an initialized central security manager.
func NewSecurityManager(cfg SecurityManagerConfig) *SecurityManager {
	if cfg.WorkspaceRoot == "" || cfg.WorkspaceRoot == "." {
		cfg.WorkspaceRoot = "."
	}
	cfg.WorkspaceRoot = filepath.Clean(cfg.WorkspaceRoot)

	if cfg.DataRoot == "" {
		cfg.DataRoot = filepath.Join(cfg.WorkspaceRoot, ".limoxel", "plugins", "data")
	}
	if cfg.TempRoot == "" {
		cfg.TempRoot = filepath.Join(cfg.WorkspaceRoot, ".limoxel", "plugins", "temp")
	}

	trustStore := verification.NewTrustStore()
	verifier := verification.NewVerifier(verification.VerifierConfig{
		TrustStore:       trustStore,
		RequireSignature: cfg.RequireSignature,
	})
	engine := permissions.NewEngine()
	monitor := monitoring.NewSecurityMonitor(2000)
	recovery := monitoring.NewRecoveryManager(cfg.RecoveryConfig, monitor)

	mgr := &SecurityManager{
		cfg:        cfg,
		trustStore: trustStore,
		verifier:   verifier,
		permEngine: engine,
		monitor:    monitor,
		recovery:   recovery,
		sandboxes:  make(map[string]sandbox.PluginSandbox),
	}

	// Connect permission engine violations directly into the security monitor
	engine.OnViolation(func(req permissions.PermissionRequest, res permissions.EvaluationResult) {
		monitor.Record(monitoring.SecurityEvent{
			Category: monitoring.CategoryPermission,
			Severity: monitoring.SeverityWarning,
			PluginID: req.PluginID,
			Action:   req.Action,
			Resource: req.Resource,
			Decision: string(res.Decision),
			Reason:   res.Reason,
		})
	})

	return mgr
}

// TrustStore returns the verification trust store.
func (m *SecurityManager) TrustStore() *verification.TrustStore {
	return m.trustStore
}

// Verifier returns the plugin integrity and signature verifier.
func (m *SecurityManager) Verifier() *verification.Verifier {
	return m.verifier
}

// PermissionEngine returns the centralized permission evaluation engine.
func (m *SecurityManager) PermissionEngine() *permissions.Engine {
	return m.permEngine
}

// Monitor returns the security monitoring and audit event recorder.
func (m *SecurityManager) Monitor() *monitoring.SecurityMonitor {
	return m.monitor
}

// Recovery returns the automated failure recovery manager.
func (m *SecurityManager) Recovery() *monitoring.RecoveryManager {
	return m.recovery
}

// VerifyPlugin checks manifest validity, checksum integrity, signatures, and publisher trust.
func (m *SecurityManager) VerifyPlugin(ctx context.Context, manifest *model.Manifest, dir string) (*pubsec.VerificationResult, error) {
	res, err := m.verifier.VerifyPluginDirectory(ctx, manifest, dir)
	if err != nil {
		return nil, err
	}

	pluginID := ""
	if manifest != nil {
		pluginID = string(manifest.ID)
	}

	// Record security event
	severity := monitoring.SeverityInfo
	if !res.Valid || res.State == pubsec.TrustRejected {
		severity = monitoring.SeverityError
	}

	reason := fmt.Sprintf("verification state: %s", res.State)
	if len(res.Errors) > 0 {
		reason = fmt.Sprintf("verification errors: %v", res.Errors)
	}

	m.monitor.Record(monitoring.SecurityEvent{
		Category: monitoring.CategoryVerification,
		Severity: severity,
		PluginID: pluginID,
		Action:   "plugin.verify",
		Decision: string(res.State),
		Reason:   reason,
	})

	return res, nil
}

// CreateSandbox allocates and initializes an isolated sandbox environment for a plugin.
func (m *SecurityManager) CreateSandbox(ctx context.Context, manifest *model.Manifest, policy *permissions.PluginPolicy) (sandbox.PluginSandbox, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pluginID := string(manifest.ID)
	pluginDataDir := filepath.Join(m.cfg.DataRoot, pluginID)
	sessionID := fmt.Sprintf("%d", time.Now().UnixNano())
	if ctx != nil {
		if s, ok := ctx.Value("session_id").(string); ok && s != "" {
			sessionID = s
		}
	}
	pluginTempDir := filepath.Join(m.cfg.TempRoot, fmt.Sprintf("%s-%s", pluginID, sessionID))

	limits := m.cfg.DefaultLimits
	if limits.MaxMemoryBytes == 0 {
		limits = sandbox.DefaultResourceLimits()
	}

	// Check if network is granted
	networkAllowed := false
	var allowedHosts []string
	if policy != nil {
		for _, g := range policy.Grants {
			if g.Action == pubsec.PermNetworkOutbound {
				networkAllowed = true
				allowedHosts = g.Scopes
				break
			}
		}
	}

	sbx, err := sandbox.NewPlatformSandbox(sandbox.SandboxConfig{
		PluginID:       pluginID,
		WorkspaceRoot:  m.cfg.WorkspaceRoot,
		DataRoot:       pluginDataDir,
		TempRoot:       pluginTempDir,
		Limits:         limits,
		NetworkAllowed: networkAllowed,
		AllowedHosts:   allowedHosts,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox for plugin %s: %w", pluginID, err)
	}

	if err := sbx.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize sandbox for plugin %s: %w", pluginID, err)
	}

	m.sandboxes[pluginID] = sbx
	return sbx, nil
}

// DestroySandbox terminates and cleans up an allocated plugin sandbox.
func (m *SecurityManager) DestroySandbox(pluginID string) error {
	m.mu.Lock()
	sbx, ok := m.sandboxes[pluginID]
	delete(m.sandboxes, pluginID)
	m.mu.Unlock()

	if !ok || sbx == nil {
		return nil
	}

	_ = sbx.Terminate()
	return sbx.Cleanup()
}
