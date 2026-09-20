//go:build !windows && (linux || darwin)

package sandbox

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"syscall"
)

// UnixProcessSandbox implements PluginSandbox on Unix-like operating systems (Linux, macOS).
type UnixProcessSandbox struct {
	mu            sync.RWMutex
	cfg           SandboxConfig
	fsGuard       *FilesystemGuard
	netGuard      *NetworkGuard
	limiter       *ResourceLimiter
	attachedPids  map[int]struct{}
	attachedPgids map[int]struct{}
	closed        bool
}

// NewPlatformSandbox instantiates the Unix process sandbox.
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
	lim := NewResourceLimiter(cfg.Limits)

	return &UnixProcessSandbox{
		cfg:           cfg,
		fsGuard:       fs,
		netGuard:      netG,
		limiter:       lim,
		attachedPids:  make(map[int]struct{}),
		attachedPgids: make(map[int]struct{}),
	}, nil
}

// Initialize prepares directory hierarchies required by the sandbox.
func (s *UnixProcessSandbox) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.fsGuard.EnsureStorageDirectories()
}

// Filesystem returns the sandbox filesystem guard.
func (s *UnixProcessSandbox) Filesystem() *FilesystemGuard {
	return s.fsGuard
}

// Network returns the sandbox network guard.
func (s *UnixProcessSandbox) Network() *NetworkGuard {
	return s.netGuard
}

// AttachProcess records the process ID and applies process group tracking, failing closed if the target process cannot be attached.
func (s *UnixProcessSandbox) AttachProcess(pid int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return errors.New("cannot attach process to closed sandbox")
	}
	if pid <= 0 {
		return fmt.Errorf("invalid process ID: %d", pid)
	}

	// Verify the process exists and the caller has permissions to signal it
	if err := syscall.Kill(pid, 0); err != nil {
		return fmt.Errorf("target process %d cannot be attached: %w", pid, err)
	}

	// Determine and track process group
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		return fmt.Errorf("failed to determine process group for pid %d: %w", pid, err)
	}

	s.attachedPids[pid] = struct{}{}
	if s.attachedPgids == nil {
		s.attachedPgids = make(map[int]struct{})
	}
	s.attachedPgids[pgid] = struct{}{}
	s.limiter.IncrementProcesses()
	return nil
}

// Terminate sends SIGKILL to all attached processes and their process groups.
func (s *UnixProcessSandbox) Terminate() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for pgid := range s.attachedPgids {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}
	for pid := range s.attachedPids {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}

	return nil
}

// Cleanup terminates processes and removes ephemeral scratch files.
func (s *UnixProcessSandbox) Cleanup() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	for pgid := range s.attachedPgids {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}
	for pid := range s.attachedPids {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}

	return s.fsGuard.CleanupTempDirectory()
}
