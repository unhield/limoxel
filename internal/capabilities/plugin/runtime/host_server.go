package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/unhield/limoxel/plugin"
)

// HostServer runs inside the plugin host process, handling communication with the Limoxel parent process.
type HostServer struct {
	mu           sync.RWMutex
	codec        *ProtocolCodec
	plugin       plugin.Plugin
	host         plugin.Host
	handshaked   bool
	sessionID    string
	manifestID   string
	version      string
	exitChan     chan struct{}
	stopped      bool
	invokeRoutes map[string]func(payload []byte) ([]byte, error)
}

// NewHostServer constructs an initialized host server using the supplied reader and writer.
func NewHostServer(r io.Reader, w io.Writer, p plugin.Plugin, h plugin.Host) *HostServer {
	return &HostServer{
		codec:        NewProtocolCodec(r, w),
		plugin:       p,
		host:         h,
		exitChan:     make(chan struct{}),
		invokeRoutes: make(map[string]func(payload []byte) ([]byte, error)),
	}
}

// RegisterRoute adds a custom invocation route handler.
func (s *HostServer) RegisterRoute(method string, handler func(payload []byte) ([]byte, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invokeRoutes[method] = handler
}

// Serve runs the protocol dispatch loop until shutdown or stream closure.
func (s *HostServer) Serve(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.exitChan:
			return nil
		default:
		}

		env, err := s.codec.ReadEnvelope()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("protocol read error: %w", err)
		}

		respEnv := s.handleEnvelope(ctx, env)
		if respEnv != nil {
			if err := s.codec.WriteEnvelope(respEnv); err != nil {
				return fmt.Errorf("protocol write error: %w", err)
			}
		}

		s.mu.RLock()
		stopped := s.stopped
		s.mu.RUnlock()
		if stopped {
			return nil
		}
	}
}

func (s *HostServer) handleEnvelope(ctx context.Context, env *Envelope) (resp *Envelope) {
	// Panic containment for message handling
	defer func() {
		if r := recover(); r != nil {
			resp = &Envelope{
				ID:    env.ID,
				Type:  MessageTypeError,
				Error: fmt.Sprintf("server panic caught: %v\nstack:\n%s", r, string(debug.Stack())),
			}
		}
	}()

	switch env.Type {
	case MessageTypeHandshakeRequest:
		return s.handleHandshake(env)

	case MessageTypeInitRequest:
		return s.handleInit(ctx, env)

	case MessageTypeStartRequest:
		return s.handleStart(ctx, env)

	case MessageTypeStopRequest:
		return s.handleStop(ctx, env)

	case MessageTypeShutdownRequest:
		return s.handleShutdown(env)

	case MessageTypeHeartbeatRequest:
		return s.handleHeartbeat(env)

	case MessageTypeInvokeRequest:
		return s.handleInvoke(env)

	default:
		return &Envelope{
			ID:    env.ID,
			Type:  MessageTypeError,
			Error: fmt.Sprintf("unsupported message type: %s", env.Type),
		}
	}
}

func (s *HostServer) handleHandshake(env *Envelope) *Envelope {
	var req HandshakeRequest
	if err := DecodePayload(env.Payload, &req); err != nil {
		return &Envelope{
			ID:    env.ID,
			Type:  MessageTypeHandshakeResponse,
			Error: fmt.Sprintf("invalid handshake payload: %v", err),
		}
	}

	// Validate protocol version compatibility
	if req.ProtocolVersion != ProtocolVersion {
		respPayload, _ := EncodePayload(&HandshakeResponse{
			Accepted:        false,
			ProtocolVersion: ProtocolVersion,
			Error: fmt.Sprintf("incompatible protocol version: client requested %q, server supports %q",
				req.ProtocolVersion, ProtocolVersion),
		})
		return &Envelope{
			ID:      env.ID,
			Type:    MessageTypeHandshakeResponse,
			Payload: respPayload,
		}
	}

	s.mu.Lock()
	s.handshaked = true
	s.manifestID = req.PluginID
	s.version = req.PluginVersion
	s.sessionID = fmt.Sprintf("session-%s-%d", req.PluginID, time.Now().UnixNano())
	sessID := s.sessionID
	s.mu.Unlock()

	respPayload, _ := EncodePayload(&HandshakeResponse{
		Accepted:              true,
		SessionID:             sessID,
		HostVersion:           "1.0.0",
		ProtocolVersion:       ProtocolVersion,
		SupportedCapabilities: req.Capabilities,
	})

	return &Envelope{
		ID:      env.ID,
		Type:    MessageTypeHandshakeResponse,
		Payload: respPayload,
	}
}

func (s *HostServer) checkHandshake(env *Envelope) *Envelope {
	s.mu.RLock()
	hs := s.handshaked
	s.mu.RUnlock()

	if !hs {
		return &Envelope{
			ID:    env.ID,
			Type:  MessageTypeError,
			Error: "protocol violation: handshake required prior to operation execution",
		}
	}
	return nil
}

func (s *HostServer) handleInit(ctx context.Context, env *Envelope) *Envelope {
	if errEnv := s.checkHandshake(env); errEnv != nil {
		return errEnv
	}

	var req InitializeRequest
	_ = DecodePayload(env.Payload, &req)

	var initErr error
	if s.plugin != nil {
		initErr = s.plugin.Init(ctx, s.host)
	}

	resp := &InitializeResponse{
		Success: initErr == nil,
	}
	if initErr != nil {
		resp.Error = initErr.Error()
	}

	payload, _ := EncodePayload(resp)
	return &Envelope{
		ID:      env.ID,
		Type:    MessageTypeInitResponse,
		Payload: payload,
	}
}

func (s *HostServer) handleStart(ctx context.Context, env *Envelope) *Envelope {
	if errEnv := s.checkHandshake(env); errEnv != nil {
		return errEnv
	}

	var startErr error
	if s.plugin != nil {
		startErr = s.plugin.Start(ctx)
	}

	resp := &StartResponse{
		Success: startErr == nil,
	}
	if startErr != nil {
		resp.Error = startErr.Error()
	}

	payload, _ := EncodePayload(resp)
	return &Envelope{
		ID:      env.ID,
		Type:    MessageTypeStartResponse,
		Payload: payload,
	}
}

func (s *HostServer) handleStop(ctx context.Context, env *Envelope) *Envelope {
	if errEnv := s.checkHandshake(env); errEnv != nil {
		return errEnv
	}

	var stopErr error
	if s.plugin != nil {
		stopErr = s.plugin.Stop(ctx)
	}

	resp := &StopResponse{
		Success: stopErr == nil,
	}
	if stopErr != nil {
		resp.Error = stopErr.Error()
	}

	payload, _ := EncodePayload(resp)
	return &Envelope{
		ID:      env.ID,
		Type:    MessageTypeStopResponse,
		Payload: payload,
	}
}

func (s *HostServer) handleShutdown(env *Envelope) *Envelope {
	s.mu.Lock()
	s.stopped = true
	close(s.exitChan)
	s.mu.Unlock()

	payload, _ := EncodePayload(&ShutdownResponse{Success: true})
	return &Envelope{
		ID:      env.ID,
		Type:    MessageTypeShutdownResponse,
		Payload: payload,
	}
}

func (s *HostServer) handleHeartbeat(env *Envelope) *Envelope {
	resp := &HeartbeatResponse{
		Status:    "ok",
		ProcessID: os.Getpid(),
	}
	payload, _ := EncodePayload(resp)
	return &Envelope{
		ID:      env.ID,
		Type:    MessageTypeHeartbeatResponse,
		Payload: payload,
	}
}

func (s *HostServer) handleInvoke(env *Envelope) *Envelope {
	if errEnv := s.checkHandshake(env); errEnv != nil {
		return errEnv
	}

	var req InvokeRequest
	if err := DecodePayload(env.Payload, &req); err != nil {
		return &Envelope{
			ID:    env.ID,
			Type:  MessageTypeInvokeResponse,
			Error: fmt.Sprintf("invalid invoke payload: %v", err),
		}
	}

	s.mu.RLock()
	handler, ok := s.invokeRoutes[req.Method]
	s.mu.RUnlock()

	if !ok {
		return &Envelope{
			ID:    env.ID,
			Type:  MessageTypeInvokeResponse,
			Error: fmt.Sprintf("unknown invoke method: %s", req.Method),
		}
	}

	res, err := handler(req.Payload)
	resp := &InvokeResponse{
		Result: res,
	}
	if err != nil {
		resp.Error = err.Error()
	}

	payload, _ := EncodePayload(resp)
	return &Envelope{
		ID:      env.ID,
		Type:    MessageTypeInvokeResponse,
		Payload: payload,
	}
}
