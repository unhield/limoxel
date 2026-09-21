//go:build !windows && (linux || darwin)

package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

// IsolateProcessCmd configures a command to start in a new, independent process group.
// On Unix systems, this sets SysProcAttr.Setpgid = true so the child process does not
// inherit the caller's process group, ensuring safe sandbox containment.
func IsolateProcessCmd(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// UnixProcessSandbox implements PluginSandbox on Unix-like operating systems (Linux, macOS).
//
// NOTE ON PROCESS GROUP CONTAINMENT:
// Process group termination sends SIGKILL to -pgid, which reaps the leader process and all
// descendants that remain within the assigned process group. However, POSIX process groups do not
// prevent adversarial grandchildren from escaping if they explicitly call setsid() or setpgid() to
// detach from the original group. Complete hierarchical containment without escape requires Linux
// cgroups v2 or PID namespaces.
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
	allowRepoWrite := false
	if cfg.Isolation != IsolationLevelRestricted {
		// By default, only non-restricted sandboxes may be granted repo write
		allowRepoWrite = false
	}

	fs, err := NewFilesystemGuard(
		cfg.WorkspaceRoot,
		cfg.DataRoot,
		cfg.TempRoot,
		allowRepoWrite,
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
//
// INVARIANT: A sandbox must NEVER attach or terminate the process group containing the sandbox host itself.
// If the target process shares the caller's process group, AttachProcess fails with ErrSandboxViolation.
func (s *UnixProcessSandbox) AttachProcess(pid int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return errors.New("cannot attach process to closed sandbox")
	}
	if pid <= 0 {
		return fmt.Errorf("invalid process ID: %d", pid)
	}
	if pid == os.Getpid() {
		return fmt.Errorf("%w: cannot attach current host process to sandbox", ErrSandboxViolation)
	}

	// 1. Atomically reserve process slot against limits
	if err := s.limiter.ReserveProcess(); err != nil {
		return err
	}
	rollback := true
	defer func() {
		if rollback {
			s.limiter.ReleaseProcess()
		}
	}()

	// 2. Verify the process exists and the caller has permissions to signal it
	if err := syscall.Kill(pid, 0); err != nil {
		return fmt.Errorf("target process %d cannot be attached: %w", pid, err)
	}

	// 3. Determine caller's own process group
	callerPgid := syscall.Getpgrp()
	if callerPgid <= 0 {
		var err error
		callerPgid, err = syscall.Getpgid(0)
		if err != nil {
			return fmt.Errorf("failed to determine sandbox host process group: %w", err)
		}
	}

	// 4. Determine target's process group
	targetPgid, err := syscall.Getpgid(pid)
	if err != nil {
		return fmt.Errorf("failed to determine process group for pid %d: %w", pid, err)
	}

	// 5. Enforce fail-closed process group isolation invariants
	if targetPgid <= 1 {
		return fmt.Errorf("%w: invalid target process group %d for pid %d", ErrSandboxViolation, targetPgid, pid)
	}
	if targetPgid == callerPgid {
		return fmt.Errorf("%w: target process %d shares process group %d with sandbox host; refusing to attach unisolated process",
			ErrSandboxViolation, pid, targetPgid)
	}

	s.attachedPids[pid] = struct{}{}
	if s.attachedPgids == nil {
		s.attachedPgids = make(map[int]struct{})
	}
	s.attachedPgids[targetPgid] = struct{}{}
	rollback = false
	return nil
}

// Terminate sends SIGKILL to all attached processes and their isolated process groups.
func (s *UnixProcessSandbox) Terminate() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	callerPgid := syscall.Getpgrp()
	var lastErr error

	for pgid := range s.attachedPgids {
		// Defensive invariant: never signal system init or the caller's process group
		if pgid <= 1 || (callerPgid > 0 && pgid == callerPgid) {
			if lastErr == nil {
				lastErr = fmt.Errorf("%w: refused to terminate unsafe process group %d", ErrSandboxViolation, pgid)
			}
			continue
		}
		err := syscall.Kill(-pgid, syscall.SIGKILL)
		if err != nil && !errors.Is(err, syscall.ESRCH) {
			if lastErr == nil {
				lastErr = fmt.Errorf("failed to terminate process group %d: %w", pgid, err)
			}
		}
	}
	for pid := range s.attachedPids {
		if pid <= 1 {
			if lastErr == nil {
				lastErr = fmt.Errorf("%w: refused to terminate unsafe pid %d", ErrSandboxViolation, pid)
			}
			continue
		}
		err := syscall.Kill(pid, syscall.SIGKILL)
		if err != nil && !errors.Is(err, syscall.ESRCH) {
			if lastErr == nil {
				lastErr = fmt.Errorf("failed to terminate process %d: %w", pid, err)
			}
		}
	}

	for range s.attachedPids {
		s.limiter.ReleaseProcess()
	}
	s.attachedPids = make(map[int]struct{})
	s.attachedPgids = make(map[int]struct{})

	return lastErr
}

// Cleanup terminates processes and removes ephemeral scratch files.
func (s *UnixProcessSandbox) Cleanup() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	callerPgid := syscall.Getpgrp()
	var lastErr error

	for pgid := range s.attachedPgids {
		if pgid <= 1 || (callerPgid > 0 && pgid == callerPgid) {
			if lastErr == nil {
				lastErr = fmt.Errorf("%w: refused to terminate unsafe process group %d", ErrSandboxViolation, pgid)
			}
			continue
		}
		err := syscall.Kill(-pgid, syscall.SIGKILL)
		if err != nil && !errors.Is(err, syscall.ESRCH) {
			if lastErr == nil {
				lastErr = fmt.Errorf("failed to terminate process group %d: %w", pgid, err)
			}
		}
	}
	for pid := range s.attachedPids {
		if pid <= 1 {
			if lastErr == nil {
				lastErr = fmt.Errorf("%w: refused to terminate unsafe pid %d", ErrSandboxViolation, pid)
			}
			continue
		}
		err := syscall.Kill(pid, syscall.SIGKILL)
		if err != nil && !errors.Is(err, syscall.ESRCH) {
			if lastErr == nil {
				lastErr = fmt.Errorf("failed to terminate process %d: %w", pid, err)
			}
		}
	}

	for range s.attachedPids {
		s.limiter.ReleaseProcess()
	}
	s.attachedPids = make(map[int]struct{})
	s.attachedPgids = make(map[int]struct{})

	if err := s.fsGuard.CleanupTempDirectory(); err != nil && lastErr == nil {
		lastErr = err
	}

	return lastErr
}
