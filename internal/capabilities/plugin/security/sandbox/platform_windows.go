//go:build windows

package sandbox

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"syscall"
	"unsafe"
)

var (
	modkernel32 = syscall.NewLazyDLL("kernel32.dll")

	procCreateJobObjectW         = modkernel32.NewProc("CreateJobObjectW")
	procSetInformationJobObject  = modkernel32.NewProc("SetInformationJobObject")
	procAssignProcessToJobObject = modkernel32.NewProc("AssignProcessToJobObject")
	procTerminateJobObject       = modkernel32.NewProc("TerminateJobObject")
	procOpenProcess              = modkernel32.NewProc("OpenProcess")
	procCloseHandle              = modkernel32.NewProc("CloseHandle")
)

const (
	jobObjectExtendedLimitInformation = 9

	// Job Object Limit Flags
	jobObjectLimitKillOnJobClose = 0x00002000
	jobObjectLimitJobMemory      = 0x00000200
	jobObjectLimitActiveProcess  = 0x00000008

	// Process Access Rights
	processTerminate       = 0x0001
	processSetQuota        = 0x0100
	standardRightsRequired = 0x000F0000
	synchronizeAccess      = 0x00100000
)

type ioCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type jobObjectBasicLimitInformation struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type jobObjectExtendedLimitInformationStruct struct {
	BasicLimitInformation jobObjectBasicLimitInformation
	IoInfo                ioCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

// WindowsProcessSandbox implements PluginSandbox using native Win32 Job Objects.
type WindowsProcessSandbox struct {
	mu           sync.RWMutex
	cfg          SandboxConfig
	jobHandle    syscall.Handle
	fsGuard      *FilesystemGuard
	netGuard     *NetworkGuard
	limiter      *ResourceLimiter
	attachedPids map[int]struct{}
	closed       bool
}

// NewPlatformSandbox instantiates the Windows-native process sandbox.
func NewPlatformSandbox(cfg SandboxConfig) (PluginSandbox, error) {
	fs, err := NewFilesystemGuard(
		cfg.WorkspaceRoot,
		cfg.DataRoot,
		cfg.TempRoot,
		false, // repository write restricted by default
		cfg.AllowedPaths...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize filesystem guard: %w", err)
	}

	netG := NewNetworkGuard(cfg.NetworkAllowed, cfg.AllowLocalhost, cfg.AllowedHosts)
	lim := NewResourceLimiter(cfg.Limits)

	return &WindowsProcessSandbox{
		cfg:          cfg,
		fsGuard:      fs,
		netGuard:     netG,
		limiter:      lim,
		attachedPids: make(map[int]struct{}),
	}, nil
}

// Initialize creates the Win32 Job Object and sets resource / lifetime limits.
func (s *WindowsProcessSandbox) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.jobHandle != 0 {
		return nil
	}

	// 1. Ensure private storage and temp directories exist
	if err := s.fsGuard.EnsureStorageDirectories(); err != nil {
		return fmt.Errorf("failed to prepare storage directories: %w", err)
	}

	// 2. Create Win32 Job Object
	r1, _, err := procCreateJobObjectW.Call(0, 0)
	if r1 == 0 {
		return fmt.Errorf("CreateJobObjectW failed: %w", err)
	}
	hJob := syscall.Handle(r1)

	// 3. Configure Extended Limit Information
	var info jobObjectExtendedLimitInformationStruct
	info.BasicLimitInformation.LimitFlags = jobObjectLimitKillOnJobClose

	if s.cfg.Limits.MaxMemoryBytes > 0 {
		info.BasicLimitInformation.LimitFlags |= jobObjectLimitJobMemory
		info.JobMemoryLimit = uintptr(s.cfg.Limits.MaxMemoryBytes)
	}

	if s.cfg.Limits.MaxProcesses > 0 {
		info.BasicLimitInformation.LimitFlags |= jobObjectLimitActiveProcess
		info.BasicLimitInformation.ActiveProcessLimit = s.cfg.Limits.MaxProcesses
	}

	r2, _, errSet := procSetInformationJobObject.Call(
		uintptr(hJob),
		uintptr(jobObjectExtendedLimitInformation),
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Sizeof(info)),
	)
	if r2 == 0 {
		_, _, _ = procCloseHandle.Call(uintptr(hJob))
		return fmt.Errorf("SetInformationJobObject failed: %w", errSet)
	}

	s.jobHandle = hJob
	return nil
}

// Filesystem returns the sandbox filesystem guard.
func (s *WindowsProcessSandbox) Filesystem() *FilesystemGuard {
	return s.fsGuard
}

// Network returns the sandbox network guard.
func (s *WindowsProcessSandbox) Network() *NetworkGuard {
	return s.netGuard
}

// AttachProcess assigns the specified PID to the Win32 Job Object.
func (s *WindowsProcessSandbox) AttachProcess(pid int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.jobHandle == 0 {
		return errors.New("cannot attach process to closed or uninitialized sandbox")
	}
	if pid <= 0 {
		return fmt.Errorf("invalid process ID: %d", pid)
	}

	// Open process handle with required access rights
	desiredAccess := uintptr(processSetQuota | processTerminate | standardRightsRequired | synchronizeAccess)
	hProc, _, err := procOpenProcess.Call(desiredAccess, 0, uintptr(pid))
	if hProc == 0 {
		return fmt.Errorf("OpenProcess failed for pid %d: %w", pid, err)
	}
	defer func() {
		_, _, _ = procCloseHandle.Call(hProc)
	}()

	// Assign to Job Object
	r, _, errAssign := procAssignProcessToJobObject.Call(uintptr(s.jobHandle), hProc)
	if r == 0 {
		return fmt.Errorf("AssignProcessToJobObject failed for pid %d: %w", pid, errAssign)
	}

	s.attachedPids[pid] = struct{}{}
	s.limiter.IncrementProcesses()
	return nil
}

// Terminate forcibly terminates all processes in the Job Object.
func (s *WindowsProcessSandbox) Terminate() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.jobHandle == 0 {
		return nil
	}

	r, _, err := procTerminateJobObject.Call(uintptr(s.jobHandle), uintptr(1))
	if r == 0 {
		return fmt.Errorf("TerminateJobObject failed: %w", err)
	}

	return nil
}

// Cleanup terminates processes, closes the Job Object, and removes temp files.
func (s *WindowsProcessSandbox) Cleanup() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	var lastErr error
	if s.jobHandle != 0 {
		_, _, _ = procTerminateJobObject.Call(uintptr(s.jobHandle), uintptr(1))
		r, _, err := procCloseHandle.Call(uintptr(s.jobHandle))
		if r == 0 {
			lastErr = fmt.Errorf("CloseHandle failed for job object: %w", err)
		}
		s.jobHandle = 0
	}

	if err := s.fsGuard.CleanupTempDirectory(); err != nil && lastErr == nil {
		lastErr = err
	}

	return lastErr
}
