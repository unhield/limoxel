package security

import (
	"context"
	"fmt"
	"sync"

	pkgerr "github.com/unhield/limoxel/internal/capabilities/plugin/errors"
	"github.com/unhield/limoxel/internal/capabilities/plugin/loader"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security/monitoring"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security/permissions"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security/sandbox"
	"github.com/unhield/limoxel/plugin"
	pubsec "github.com/unhield/limoxel/plugin/security"
)

// ProcessInspector is an optional interface implemented by instances that expose an OS PID.
type ProcessInspector interface {
	ProcessID() int
}

// SecureLoader decorates a loader.Loader with pre-load verification, sandboxing, and host mediation.
type SecureLoader struct {
	mu     sync.RWMutex
	inner  loader.Loader
	secMgr *SecurityManager
}

// NewSecureLoader constructs an initialized SecureLoader wrapping an underlying runtime loader.
func NewSecureLoader(inner loader.Loader, secMgr *SecurityManager) *SecureLoader {
	return &SecureLoader{
		inner:  inner,
		secMgr: secMgr,
	}
}

// Load executes the complete security lifecycle: verification -> sandbox allocation -> sandboxed host -> inner load.
func (l *SecureLoader) Load(ctx context.Context, manifest *model.Manifest, dir string, host plugin.Host) (loader.Instance, error) {
	if manifest == nil {
		return nil, pkgerr.New(pkgerr.CodeManifestInvalid, "", "manifest cannot be nil")
	}

	pluginID := string(manifest.ID)

	// Step 0: Quarantine Check - Fail-closed if quarantined
	if stats, ok := l.secMgr.Monitor().GetStats(pluginID); ok && stats.Quarantined {
		return nil, pkgerr.New(pkgerr.CodeLoadingFailed, manifest.ID, "plugin is quarantined due to critical security violations")
	}

	// Step 1: Pre-Load Cryptographic Verification
	vResult, err := l.secMgr.VerifyPlugin(ctx, manifest, dir)
	if err != nil {
		return nil, pkgerr.Wrap(pkgerr.CodeLoadingFailed, manifest.ID, "security verification failed", err)
	}

	if !vResult.Valid || vResult.State == pubsec.TrustRejected {
		errMsg := "plugin rejected by cryptographic verification"
		if len(vResult.Errors) > 0 {
			errMsg = fmt.Sprintf("plugin rejected by verification: %v", vResult.Errors)
		}
		return nil, pkgerr.New(pkgerr.CodeLoadingFailed, manifest.ID, errMsg)
	}

	// Step 2: Policy Resolution & Registration
	policy := permissions.ParseManifestPolicy(manifest)
	l.secMgr.PermissionEngine().SetPolicy(policy)

	// Step 3: Sandbox Allocation
	sbx, err := l.secMgr.CreateSandbox(ctx, manifest, policy)
	if err != nil {
		l.secMgr.PermissionEngine().RemovePolicy(pluginID)
		return nil, pkgerr.Wrap(pkgerr.CodeLoadingFailed, manifest.ID, "failed to create sandbox", err)
	}

	// Step 4: Wrap Host with SandboxedHostAdapter
	var extendedHost plugin.ExtendedHost
	if eh, ok := host.(plugin.ExtendedHost); ok {
		extendedHost = eh
	}

	sandboxedHost := NewSandboxedHostAdapter(
		extendedHost,
		pluginID,
		l.secMgr.PermissionEngine(),
		sbx.Filesystem(),
		l.secMgr.Monitor(),
	)

	// Step 5: Delegate to Underlying Loader
	innerInst, err := l.inner.Load(ctx, manifest, dir, sandboxedHost)
	if err != nil {
		_ = l.secMgr.DestroySandbox(pluginID)
		l.secMgr.PermissionEngine().RemovePolicy(pluginID)
		return nil, err
	}

	// Step 6: Process isolation boundary check
	if inspector, ok := innerInst.(ProcessInspector); ok && inspector.ProcessID() > 0 {
		pid := inspector.ProcessID()
		if attachErr := sbx.AttachProcess(pid); attachErr != nil {
			_ = innerInst.Dispose()
			_ = sbx.Terminate()
			_ = l.secMgr.DestroySandbox(pluginID)
			l.secMgr.PermissionEngine().RemovePolicy(pluginID)
			l.secMgr.Monitor().Record(monitoring.SecurityEvent{
				Category:  monitoring.CategorySandbox,
				Severity:  monitoring.SeverityCritical,
				PluginID:  pluginID,
				ProcessID: pid,
				Action:    "sandbox.attach",
				Decision:  "VIOLATION",
				Reason:    fmt.Sprintf("failed to attach process %d to sandbox: %v", pid, attachErr),
			})
			return nil, pkgerr.Wrap(pkgerr.CodeLoadingFailed, manifest.ID, fmt.Sprintf("failed to attach process %d to sandbox", pid), attachErr)
		}
	} else if !vResult.State.IsAcceptable() {
		// In-process execution runs directly in the host process memory space without OS process boundaries.
		// Untrusted code is strictly prohibited from running in-process.
		_ = innerInst.Dispose()
		_ = sbx.Terminate()
		_ = l.secMgr.DestroySandbox(pluginID)
		l.secMgr.PermissionEngine().RemovePolicy(pluginID)
		l.secMgr.Monitor().Record(monitoring.SecurityEvent{
			Category: monitoring.CategorySandbox,
			Severity: monitoring.SeverityCritical,
			PluginID: pluginID,
			Action:   "sandbox.inprocess_untrusted",
			Decision: "REJECT",
			Reason:   "untrusted plugin cannot run in host process without process isolation",
		})
		return nil, pkgerr.New(pkgerr.CodeLoadingFailed, manifest.ID, "untrusted plugin cannot run in host process without OS process isolation")
	}

	return &secureInstance{
		inner:    innerInst,
		secMgr:   l.secMgr,
		sandbox:  sbx,
		pluginID: pluginID,
	}, nil
}

// Unload disposes the underlying instance and cleans up sandbox resources.
func (l *SecureLoader) Unload(ctx context.Context, inst loader.Instance) error {
	if inst == nil {
		return nil
	}

	if sInst, ok := inst.(*secureInstance); ok {
		return sInst.Dispose()
	}

	return l.inner.Unload(ctx, inst)
}

// Reload restarts the instance through the secure load pipeline.
func (l *SecureLoader) Reload(ctx context.Context, inst loader.Instance, host plugin.Host) (loader.Instance, error) {
	manifest := inst.Manifest()
	_ = l.Unload(ctx, inst)
	return l.Load(ctx, manifest, "", host)
}

type secureInstance struct {
	inner    loader.Instance
	secMgr   *SecurityManager
	sandbox  sandbox.PluginSandbox
	pluginID string
	closed   bool
	mu       sync.Mutex
}

func (s *secureInstance) ID() model.Identity        { return s.inner.ID() }
func (s *secureInstance) Manifest() *model.Manifest { return s.inner.Manifest() }
func (s *secureInstance) Plugin() plugin.Plugin     { return s.inner.Plugin() }

func (s *secureInstance) Start(ctx context.Context) error {
	return s.inner.Start(ctx)
}

func (s *secureInstance) Stop(ctx context.Context) error {
	err := s.inner.Stop(ctx)
	_ = s.sandbox.Terminate()
	return err
}

func (s *secureInstance) Dispose() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	var firstErr error
	if err := s.inner.Dispose(); err != nil {
		firstErr = err
	}

	_ = s.secMgr.DestroySandbox(s.pluginID)
	s.secMgr.PermissionEngine().RemovePolicy(s.pluginID)

	return firstErr
}

func (s *secureInstance) ProcessID() int {
	if p, ok := s.inner.(ProcessInspector); ok {
		return p.ProcessID()
	}
	return 0
}
