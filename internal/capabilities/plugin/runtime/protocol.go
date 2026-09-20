package runtime

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

// ProtocolVersion is the authoritative version of the plugin runtime communication protocol.
const ProtocolVersion = "1.0.0"

// MaxMessageBytes enforces a strict ceiling on incoming message frames to prevent memory exhaustion (16MB).
const MaxMessageBytes = 16 * 1024 * 1024

// Canonical Message Types
const (
	MessageTypeHandshakeRequest  = "handshake.request"
	MessageTypeHandshakeResponse = "handshake.response"
	MessageTypeInitRequest       = "init.request"
	MessageTypeInitResponse      = "init.response"
	MessageTypeStartRequest      = "start.request"
	MessageTypeStartResponse     = "start.response"
	MessageTypeStopRequest       = "stop.request"
	MessageTypeStopResponse      = "stop.response"
	MessageTypeShutdownRequest   = "shutdown.request"
	MessageTypeShutdownResponse  = "shutdown.response"
	MessageTypeHeartbeatRequest  = "heartbeat.request"
	MessageTypeHeartbeatResponse = "heartbeat.response"
	MessageTypeInvokeRequest     = "invoke.request"
	MessageTypeInvokeResponse    = "invoke.response"
	MessageTypeEventNotification = "event.notification"
	MessageTypeError             = "protocol.error"
)

// Envelope represents the framed wire transport container for all protocol interactions.
type Envelope struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// HandshakeRequest is sent by the host to negotiate capabilities and establish a runtime session.
type HandshakeRequest struct {
	PluginID        string            `json:"plugin_id"`
	PluginVersion   string            `json:"plugin_version"`
	ManifestVersion string            `json:"manifest_version"`
	ProtocolVersion string            `json:"protocol_version"`
	Capabilities    []string          `json:"capabilities,omitempty"`
	PlatformOS      string            `json:"platform_os"`
	PlatformArch    string            `json:"platform_arch"`
	Environment     map[string]string `json:"environment,omitempty"`
}

// HandshakeResponse is returned by the plugin host indicating acceptance or rejection.
type HandshakeResponse struct {
	Accepted              bool     `json:"accepted"`
	SessionID             string   `json:"session_id"`
	HostVersion           string   `json:"host_version"`
	ProtocolVersion       string   `json:"protocol_version"`
	SupportedCapabilities []string `json:"supported_capabilities,omitempty"`
	Error                 string   `json:"error,omitempty"`
}

// InitializeRequest passes runtime configuration and context to the plugin.
type InitializeRequest struct {
	Config        map[string]string `json:"config,omitempty"`
	WorkspaceRoot string            `json:"workspace_root"`
}

// InitializeResponse indicates whether plugin initialization succeeded.
type InitializeResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// StartRequest instructs the plugin to transition to the active state.
type StartRequest struct {
	TimeoutMs int64 `json:"timeout_ms"`
}

// StartResponse reports startup outcome.
type StartResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// StopRequest signals graceful deactivation.
type StopRequest struct {
	GracefulTimeoutMs int64 `json:"graceful_timeout_ms"`
}

// StopResponse reports deactivation outcome.
type StopResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ShutdownRequest directs the plugin host process to exit.
type ShutdownRequest struct {
	Force bool `json:"force"`
}

// ShutdownResponse acknowledges shutdown receipt.
type ShutdownResponse struct {
	Success bool `json:"success"`
}

// HeartbeatRequest queries health and resource utilization.
type HeartbeatRequest struct {
	Timestamp int64 `json:"timestamp"`
}

// HeartbeatResponse reports current telemetry.
type HeartbeatResponse struct {
	Status      string `json:"status"`
	ProcessID   int    `json:"process_id"`
	MemoryBytes uint64 `json:"memory_bytes"`
}

// InvokeRequest dispatches a functional invocation to the plugin.
type InvokeRequest struct {
	Method  string `json:"method"`
	Payload []byte `json:"payload,omitempty"`
}

// InvokeResponse delivers the result of an invocation.
type InvokeResponse struct {
	Result []byte `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// EventNotification broadcasts an asynchronous event.
type EventNotification struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// ProtocolCodec manages thread-safe framing and serialization of envelopes over streams.
type ProtocolCodec struct {
	r       *bufio.Reader
	w       *bufio.Writer
	muRead  sync.Mutex
	muWrite sync.Mutex
	seq     uint64
}

// NewProtocolCodec creates an initialized codec for the given input and output streams.
func NewProtocolCodec(r io.Reader, w io.Writer) *ProtocolCodec {
	return &ProtocolCodec{
		r: bufio.NewReaderSize(r, 64*1024),
		w: bufio.NewWriterSize(w, 64*1024),
	}
}

// NextID returns a monotonically increasing identifier string for request correlation.
func (c *ProtocolCodec) NextID() string {
	val := atomic.AddUint64(&c.seq, 1)
	return fmt.Sprintf("msg-%d", val)
}

// ReadEnvelope reads the next newline-delimited envelope from the stream with strict allocation bounds.
func (c *ProtocolCodec) ReadEnvelope() (*Envelope, error) {
	c.muRead.Lock()
	defer c.muRead.Unlock()

	var line []byte
	for {
		chunk, isPrefix, err := c.r.ReadLine()
		if err != nil {
			return nil, err
		}
		if len(line)+len(chunk) > MaxMessageBytes {
			return nil, errors.New("message frame exceeds maximum permitted size")
		}
		line = append(line, chunk...)
		if !isPrefix {
			break
		}
	}

	var env Envelope
	if err := json.Unmarshal(line, &env); err != nil {
		return nil, fmt.Errorf("failed to decode protocol envelope: %w", err)
	}

	return &env, nil
}

// WriteEnvelope writes a framed newline-delimited envelope to the stream.
func (c *ProtocolCodec) WriteEnvelope(env *Envelope) error {
	c.muWrite.Lock()
	defer c.muWrite.Unlock()

	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("failed to encode protocol envelope: %w", err)
	}

	if len(data) > MaxMessageBytes {
		return errors.New("serialized envelope exceeds maximum permitted size")
	}

	if _, err := c.w.Write(data); err != nil {
		return err
	}
	if err := c.w.WriteByte('\n'); err != nil {
		return err
	}
	return c.w.Flush()
}

// EncodePayload marshals a typed value into json.RawMessage.
func EncodePayload(v interface{}) (json.RawMessage, error) {
	if v == nil {
		return nil, nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// DecodePayload unmarshals raw payload bytes into a typed destination.
func DecodePayload(raw json.RawMessage, dest interface{}) error {
	if len(raw) == 0 {
		return errors.New("empty payload")
	}
	return json.Unmarshal(raw, dest)
}
