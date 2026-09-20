package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var supervisorSeq uint64

// ProcessSupervisorConfig provides parameters for launching and managing an external plugin process.
type ProcessSupervisorConfig struct {
	Command         string
	Args            []string
	Dir             string
	Env             map[string]string
	GracefulTimeout time.Duration
	StderrWriter    io.Writer
}

// ProcessSupervisor manages the operating system process boundary for a plugin runtime.
type ProcessSupervisor struct {
	mu         sync.RWMutex
	cfg        ProcessSupervisorConfig
	instanceID string
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	pid        int
	startTime  time.Time
	exitTime   time.Time
	exitCode   int
	exited     bool
	exitErr    error
	exitChan   chan struct{}
	state      RuntimeState
	stateHooks []func(from, to RuntimeState)
	stateMu    sync.RWMutex
}

// NewProcessSupervisor creates a supervisor instance configured with the specified options.
func NewProcessSupervisor(cfg ProcessSupervisorConfig) *ProcessSupervisor {
	if cfg.GracefulTimeout <= 0 {
		cfg.GracefulTimeout = 3 * time.Second
	}
	seq := atomic.AddUint64(&supervisorSeq, 1)
	instID := fmt.Sprintf("sup-%d-%d", seq, time.Now().UnixNano())
	return &ProcessSupervisor{
		cfg:        cfg,
		instanceID: instID,
		exitChan:   make(chan struct{}),
		state:      StateCreated,
	}
}

// InstanceID returns the unique lifecycle identity of this supervisor instance.
func (s *ProcessSupervisor) InstanceID() string {
	return s.instanceID
}

// OnStateChange registers an observer callback for runtime lifecycle state changes.
func (s *ProcessSupervisor) OnStateChange(fn func(from, to RuntimeState)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stateHooks = append(s.stateHooks, fn)
}

func (s *ProcessSupervisor) transitionState(target RuntimeState) {
	s.stateMu.Lock()
	from := s.state
	s.state = target
	hooks := make([]func(from, to RuntimeState), len(s.stateHooks))
	copy(hooks, s.stateHooks)
	s.stateMu.Unlock()

	for _, h := range hooks {
		h(from, target)
	}
}

// State returns the current lifecycle state of the process supervisor.
func (s *ProcessSupervisor) State() RuntimeState {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.state
}

// PID returns the operating system process identifier, or 0 if not running.
func (s *ProcessSupervisor) PID() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pid
}

// Uptime returns the elapsed runtime duration since process launch.
func (s *ProcessSupervisor) Uptime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.startTime.IsZero() {
		return 0
	}
	if !s.exitTime.IsZero() {
		return s.exitTime.Sub(s.startTime)
	}
	return time.Since(s.startTime)
}

// Start spawns the child process and binds standard streams.
func (s *ProcessSupervisor) Start(ctx context.Context) (io.Reader, io.Writer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != StateCreated {
		return nil, nil, fmt.Errorf("cannot start process supervisor in state: %s", s.state)
	}

	s.transitionState(StateStarting)

	cmd := exec.Command(s.cfg.Command, s.cfg.Args...)
	cmd.Dir = s.cfg.Dir

	// Environment sanitization
	cmd.Env = s.buildSanitizedEnvironment(s.cfg.Env)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		s.transitionState(StateFailed)
		return nil, nil, fmt.Errorf("failed to open stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		s.transitionState(StateFailed)
		return nil, nil, fmt.Errorf("failed to open stdout pipe: %w", err)
	}

	if s.cfg.StderrWriter != nil {
		cmd.Stderr = s.cfg.StderrWriter
	} else {
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		s.transitionState(StateFailed)
		return nil, nil, fmt.Errorf("failed to start plugin process (%s): %w", s.cfg.Command, err)
	}

	s.cmd = cmd
	s.stdin = stdin
	s.stdout = stdout
	s.pid = cmd.Process.Pid
	s.startTime = time.Now().UTC()

	// Monitor child process termination in background
	go s.monitorProcess()

	s.transitionState(StateRunning)
	return stdout, stdin, nil
}

func (s *ProcessSupervisor) monitorProcess() {
	err := s.cmd.Wait()

	s.mu.Lock()
	s.exited = true
	s.exitTime = time.Now().UTC()
	s.exitErr = err

	if s.cmd.ProcessState != nil {
		s.exitCode = s.cmd.ProcessState.ExitCode()
	} else {
		s.exitCode = -1
	}

	currentState := s.State()
	close(s.exitChan)
	s.mu.Unlock()

	// If we were not actively stopping, an unexpected termination transitions state to Failed
	if currentState == StateStopping {
		s.transitionState(StateStopped)
	} else {
		s.transitionState(StateFailed)
	}
}

// GracefulStop attempts graceful shutdown within the configured timeout, cascading to forced termination.
func (s *ProcessSupervisor) GracefulStop() error {
	s.mu.Lock()
	if s.exited {
		s.mu.Unlock()
		return nil
	}

	s.transitionState(StateStopping)

	// Close stdin pipe to signal EOF to the child process
	if s.stdin != nil {
		_ = s.stdin.Close()
	}
	s.mu.Unlock()

	// Wait for exit or timeout
	select {
	case <-s.exitChan:
		return nil
	case <-time.After(s.cfg.GracefulTimeout):
		// Force kill on timeout
		return s.ForceKill()
	}
}

// ForceKill unconditionally terminates the child operating system process.
func (s *ProcessSupervisor) ForceKill() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.exited || s.cmd == nil || s.cmd.Process == nil {
		return nil
	}

	err := s.cmd.Process.Kill()
	return err
}

// VerifyExited confirms that the operating system process has completely ceased execution.
func (s *ProcessSupervisor) VerifyExited(timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)
	for {
		s.mu.RLock()
		exited := s.exited
		cmd := s.cmd
		s.mu.RUnlock()

		if exited && cmd != nil && cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			return true, nil
		}

		if time.Now().After(deadline) {
			return false, errors.New("process exit verification timed out")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// buildSanitizedEnvironment builds a safe environment whitelist preserving system variables while barring leaks.
func (s *ProcessSupervisor) buildSanitizedEnvironment(custom map[string]string) []string {
	var env []string

	// System whitelist for Windows
	windowsWhitelist := map[string]bool{
		"SYSTEMROOT":  true,
		"PATH":        true,
		"PATHEXT":     true,
		"COMSPEC":     true,
		"TEMP":        true,
		"TMP":         true,
		"USERPROFILE": true,
		"HOMEDRIVE":   true,
		"HOMEPATH":    true,
		"WINDIR":      true,
	}

	// System whitelist for POSIX
	posixWhitelist := map[string]bool{
		"PATH":   true,
		"HOME":   true,
		"USER":   true,
		"TMPDIR": true,
		"SHELL":  true,
	}

	for _, entry := range os.Environ() {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		upperKey := strings.ToUpper(key)

		if runtime.GOOS == "windows" {
			if windowsWhitelist[upperKey] {
				env = append(env, entry)
			}
		} else {
			if posixWhitelist[key] {
				env = append(env, entry)
			}
		}
	}

	// Append caller custom variables
	for k, v := range custom {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env
}
