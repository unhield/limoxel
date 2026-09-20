package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	pkgerr "github.com/unhield/limoxel/internal/capabilities/plugin/errors"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
)

// ProcessRuntimeConfig configures an external process plugin runtime.
type ProcessRuntimeConfig struct {
	Manifest        *model.Manifest
	Command         string
	Args            []string
	Dir             string
	Env             map[string]string
	StartupTimeout  time.Duration
	ShutdownTimeout time.Duration
	StderrWriter    io.Writer
}

// ProcessRuntime implements PluginRuntime managing an out-of-process plugin instance.
type ProcessRuntime struct {
	mu         sync.RWMutex
	cfg        ProcessRuntimeConfig
	supervisor *ProcessSupervisor
	codec      *ProtocolCodec
	sessionID  string
	state      RuntimeState
	stopChan   chan struct{}
}

// NewProcessRuntime constructs an initialized ProcessRuntime.
func NewProcessRuntime(cfg ProcessRuntimeConfig) *ProcessRuntime {
	if cfg.StartupTimeout <= 0 {
		cfg.StartupTimeout = 5 * time.Second
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = 3 * time.Second
	}

	sup := NewProcessSupervisor(ProcessSupervisorConfig{
		Command:         cfg.Command,
		Args:            cfg.Args,
		Dir:             cfg.Dir,
		Env:             cfg.Env,
		GracefulTimeout: cfg.ShutdownTimeout,
		StderrWriter:    cfg.StderrWriter,
	})

	pr := &ProcessRuntime{
		cfg:        cfg,
		supervisor: sup,
		state:      StateCreated,
		stopChan:   make(chan struct{}),
	}

	sup.OnStateChange(func(from, to RuntimeState) {
		pr.mu.Lock()
		pr.state = to
		pr.mu.Unlock()
	})

	return pr
}

// SessionID returns the unique identifier for the current runtime execution session.
func (r *ProcessRuntime) SessionID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sessionID
}

// ProcessID returns the operating system process identifier of the child process.
func (r *ProcessRuntime) ProcessID() int {
	return r.supervisor.PID()
}

// InstanceID returns the unique lifecycle identity of the underlying process supervisor.
func (r *ProcessRuntime) InstanceID() string {
	return r.supervisor.InstanceID()
}

// State returns the current lifecycle state of the runtime.
func (r *ProcessRuntime) State() RuntimeState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state
}

// Start launches the child process, performs handshake, and transitions state to Running.
func (r *ProcessRuntime) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.state != StateCreated {
		r.mu.Unlock()
		return fmt.Errorf("cannot start runtime in state: %s", r.state)
	}
	r.state = StateStarting
	r.mu.Unlock()

	startCtx, cancel := context.WithTimeout(ctx, r.cfg.StartupTimeout)
	defer cancel()

	stdout, stdin, err := r.supervisor.Start(startCtx)
	if err != nil {
		r.mu.Lock()
		r.state = StateFailed
		r.mu.Unlock()
		return pkgerr.Wrap(pkgerr.CodeLoadingFailed, r.cfg.Manifest.ID, "failed to start process supervisor", err)
	}

	r.codec = NewProtocolCodec(stdout, stdin)

	// Perform Handshake
	if err := r.performHandshake(startCtx); err != nil {
		_ = r.supervisor.ForceKill()
		r.mu.Lock()
		r.state = StateFailed
		r.mu.Unlock()
		return err
	}

	r.mu.Lock()
	r.state = StateRunning
	r.mu.Unlock()
	return nil
}

func (r *ProcessRuntime) performHandshake(ctx context.Context) error {
	var caps []string
	if r.cfg.Manifest != nil {
		for _, c := range r.cfg.Manifest.Capabilities {
			caps = append(caps, c.Name)
		}
	}

	pluginID := ""
	pluginVer := ""
	manifestVer := "1.0.0"
	if r.cfg.Manifest != nil {
		pluginID = string(r.cfg.Manifest.ID)
		pluginVer = r.cfg.Manifest.Version.String()
		manifestVer = r.cfg.Manifest.SchemaVersion
	}

	req := HandshakeRequest{
		PluginID:        pluginID,
		PluginVersion:   pluginVer,
		ManifestVersion: manifestVer,
		ProtocolVersion: ProtocolVersion,
		Capabilities:    caps,
		PlatformOS:      runtime.GOOS,
		PlatformArch:    runtime.GOARCH,
	}

	payload, err := EncodePayload(req)
	if err != nil {
		return pkgerr.Wrap(pkgerr.CodeLoadingFailed, model.Identity(pluginID), "handshake encoding failed", err)
	}

	env := &Envelope{
		ID:      r.codec.NextID(),
		Type:    MessageTypeHandshakeRequest,
		Payload: payload,
	}

	if err := r.codec.WriteEnvelope(env); err != nil {
		return pkgerr.Wrap(pkgerr.CodeLoadingFailed, model.Identity(pluginID), "handshake send failed", err)
	}

	respEnv, err := r.readEnvelopeWithContext(ctx)
	if err != nil {
		return pkgerr.Wrap(pkgerr.CodeLoadingFailed, model.Identity(pluginID), "handshake receive failed", err)
	}

	var resp HandshakeResponse
	if err := DecodePayload(respEnv.Payload, &resp); err != nil {
		return pkgerr.Wrap(pkgerr.CodeLoadingFailed, model.Identity(pluginID), "handshake response decoding failed", err)
	}

	if !resp.Accepted {
		return pkgerr.New(pkgerr.CodeIncompatibleHost, model.Identity(pluginID),
			fmt.Sprintf("handshake rejected by plugin host: %s", resp.Error))
	}

	r.mu.Lock()
	r.sessionID = resp.SessionID
	r.mu.Unlock()

	return nil
}

func (r *ProcessRuntime) readEnvelopeWithContext(ctx context.Context) (*Envelope, error) {
	type result struct {
		env *Envelope
		err error
	}
	ch := make(chan result, 1)

	go func() {
		env, err := r.codec.ReadEnvelope()
		ch <- result{env: env, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		return res.env, res.err
	}
}

// Initialize passes workspace context and configuration to the plugin.
func (r *ProcessRuntime) Initialize(ctx context.Context, wsRoot string, config map[string]string) error {
	payload, _ := EncodePayload(&InitializeRequest{
		WorkspaceRoot: wsRoot,
		Config:        config,
	})

	env := &Envelope{
		ID:      r.codec.NextID(),
		Type:    MessageTypeInitRequest,
		Payload: payload,
	}

	if err := r.codec.WriteEnvelope(env); err != nil {
		return err
	}

	respEnv, err := r.readEnvelopeWithContext(ctx)
	if err != nil {
		return err
	}

	var resp InitializeResponse
	if err := DecodePayload(respEnv.Payload, &resp); err != nil {
		return err
	}

	if !resp.Success {
		return errors.New(resp.Error)
	}
	return nil
}

// Stop gracefully shuts down the external plugin process.
func (r *ProcessRuntime) Stop(ctx context.Context) error {
	r.mu.Lock()
	if r.state == StateStopped || r.state == StateStopping {
		r.mu.Unlock()
		return nil
	}
	r.state = StateStopping
	r.mu.Unlock()

	// 1. Send StopRequest
	stopPayload, _ := EncodePayload(&StopRequest{GracefulTimeoutMs: r.cfg.ShutdownTimeout.Milliseconds()})
	_ = r.codec.WriteEnvelope(&Envelope{
		ID:      r.codec.NextID(),
		Type:    MessageTypeStopRequest,
		Payload: stopPayload,
	})

	// 2. Send ShutdownRequest
	shutdownPayload, _ := EncodePayload(&ShutdownRequest{Force: false})
	_ = r.codec.WriteEnvelope(&Envelope{
		ID:      r.codec.NextID(),
		Type:    MessageTypeShutdownRequest,
		Payload: shutdownPayload,
	})

	// 3. Graceful stop with fallback to forced kill
	err := r.supervisor.GracefulStop()

	// 4. Verify process exit
	_, verifyErr := r.supervisor.VerifyExited(r.cfg.ShutdownTimeout)

	r.mu.Lock()
	r.state = StateStopped
	r.mu.Unlock()

	if err != nil {
		return err
	}
	return verifyErr
}

// Restart stops the current process and re-launches a fresh instance.
func (r *ProcessRuntime) Restart(ctx context.Context) error {
	if err := r.Stop(ctx); err != nil {
		_ = r.Close()
	}
	sup := NewProcessSupervisor(ProcessSupervisorConfig{
		Command:         r.cfg.Command,
		Args:            r.cfg.Args,
		Dir:             r.cfg.Dir,
		Env:             r.cfg.Env,
		GracefulTimeout: r.cfg.ShutdownTimeout,
		StderrWriter:    r.cfg.StderrWriter,
	})
	r.supervisor = sup
	r.state = StateCreated

	sup.OnStateChange(func(from, to RuntimeState) {
		r.mu.Lock()
		r.state = to
		r.mu.Unlock()
	})

	return r.Start(ctx)
}

// Status queries observable metrics from the runtime and child process.
func (r *ProcessRuntime) Status(ctx context.Context) (RuntimeStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return RuntimeStatus{
		State:           r.state,
		SessionID:       r.sessionID,
		ProcessID:       r.supervisor.PID(),
		ProtocolVersion: ProtocolVersion,
		Uptime:          r.supervisor.Uptime(),
	}, nil
}

// Invoke dispatches a functional call to the plugin over the protocol.
func (r *ProcessRuntime) Invoke(ctx context.Context, method string, req []byte) ([]byte, error) {
	r.mu.RLock()
	if r.state != StateRunning {
		r.mu.RUnlock()
		return nil, fmt.Errorf("cannot invoke method %s: runtime is in state %s", method, r.state)
	}
	r.mu.RUnlock()

	payload, _ := EncodePayload(&InvokeRequest{
		Method:  method,
		Payload: req,
	})

	env := &Envelope{
		ID:      r.codec.NextID(),
		Type:    MessageTypeInvokeRequest,
		Payload: payload,
	}

	if err := r.codec.WriteEnvelope(env); err != nil {
		return nil, fmt.Errorf("invoke write error: %w", err)
	}

	respEnv, err := r.readEnvelopeWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("invoke read error: %w", err)
	}

	if respEnv.Error != "" {
		return nil, errors.New(respEnv.Error)
	}

	var resp InvokeResponse
	if err := DecodePayload(respEnv.Payload, &resp); err != nil {
		return nil, fmt.Errorf("invoke response decode error: %w", err)
	}

	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}

	return resp.Result, nil
}

// Close terminates the child process and cleans up supervisor resources.
func (r *ProcessRuntime) Close() error {
	_ = r.supervisor.ForceKill()
	r.mu.Lock()
	r.state = StateStopped
	r.mu.Unlock()
	return nil
}
