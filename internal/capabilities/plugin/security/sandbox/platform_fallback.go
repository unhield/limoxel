//go:build !windows && !linux && !darwin

package sandbox

import (
	"context"
	"fmt"
	"sync"
)

// FallbackProcessSandbox implements PluginSandbox for unsupported platforms.
type FallbackProcessSandbox struct {
	mu       sync.RWMutex
	fsGuard  *FilesystemGuard
	netGuard *NetworkGuard
}

// NewPlatformSandbox instantiates the fallback process sandbox.
func NewPlatformSandbox(cfg SandboxConfig) (PluginSandbox, error) {
	fs, err := NewFilesystemGuard(
		cfg.WorkspaceRoot,
		cfg.DataRoot,
		cfg.TempRoot,
		false,
		cfg.AllowedPaths...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize filesystem guard: %w", err)
	}

	netG := NewNetworkGuard(cfg.NetworkAllowed, cfg.AllowLocalhost, cfg.AllowedHosts)

	return &FallbackProcessSandbox{
		fsGuard:  fs,
		netGuard: netG,
	}, nil
}

func (s *FallbackProcessSandbox) Initialize(ctx context.Context) error {
	return s.fsGuard.EnsureStorageDirectories()
}

func (s *FallbackProcessSandbox) Filesystem() *FilesystemGuard { return s.fsGuard }
func (s *FallbackProcessSandbox) Network() *NetworkGuard       { return s.netGuard }
func (s *FallbackProcessSandbox) AttachProcess(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid process ID: %d", pid)
	}
	return nil
}
func (s *FallbackProcessSandbox) Terminate() error { return nil }
func (s *FallbackProcessSandbox) Cleanup() error   { return s.fsGuard.CleanupTempDirectory() }
