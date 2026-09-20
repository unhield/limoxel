package runtime

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func TestProtocolCodec_RoundTrip(t *testing.T) {
	var buf bytes.Buffer
	codec := NewProtocolCodec(&buf, &buf)

	envOut := &Envelope{
		ID:      codec.NextID(),
		Type:    MessageTypeHeartbeatRequest,
		Payload: []byte(`{"timestamp":12345678}`),
	}

	if err := codec.WriteEnvelope(envOut); err != nil {
		t.Fatalf("WriteEnvelope failed: %v", err)
	}

	envIn, err := codec.ReadEnvelope()
	if err != nil {
		t.Fatalf("ReadEnvelope failed: %v", err)
	}

	if envIn.ID != envOut.ID {
		t.Errorf("expected ID %s, got %s", envOut.ID, envIn.ID)
	}
	if envIn.Type != envOut.Type {
		t.Errorf("expected Type %s, got %s", envOut.Type, envIn.Type)
	}
	if string(envIn.Payload) != string(envOut.Payload) {
		t.Errorf("expected Payload %s, got %s", string(envOut.Payload), string(envIn.Payload))
	}
}

func TestProtocolCodec_MaxMessageLimit(t *testing.T) {
	hugeData := make([]byte, MaxMessageBytes+100)
	for i := range hugeData {
		hugeData[i] = 'a'
	}
	hugeData[0] = '"'
	hugeData[len(hugeData)-1] = '"'

	var buf bytes.Buffer
	codec := NewProtocolCodec(&buf, &buf)

	env := &Envelope{
		ID:      "huge",
		Type:    "huge.type",
		Payload: hugeData,
	}

	err := codec.WriteEnvelope(env)
	if err == nil {
		t.Fatal("expected error for payload exceeding MaxMessageBytes, got nil")
	}
	if !strings.Contains(err.Error(), "maximum permitted size") {
		t.Errorf("expected maximum permitted size error, got %v", err)
	}
}

func TestHostServer_HandshakeProtocolMismatch(t *testing.T) {
	clientRead, serverWrite := io.Pipe()
	serverRead, clientWrite := io.Pipe()

	server := NewHostServer(serverRead, serverWrite, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = server.Serve(ctx)
	}()

	clientCodec := NewProtocolCodec(clientRead, clientWrite)

	// Send handshake with incompatible protocol version
	reqPayload, _ := EncodePayload(&HandshakeRequest{
		PluginID:        "org.example.test",
		PluginVersion:   "1.0.0",
		ManifestVersion: "1.0.0",
		ProtocolVersion: "99.0.0", // incompatible
	})

	err := clientCodec.WriteEnvelope(&Envelope{
		ID:      clientCodec.NextID(),
		Type:    MessageTypeHandshakeRequest,
		Payload: reqPayload,
	})
	if err != nil {
		t.Fatalf("failed to write handshake envelope: %v", err)
	}

	respEnv, err := clientCodec.ReadEnvelope()
	if err != nil {
		t.Fatalf("failed to read handshake response: %v", err)
	}

	var resp HandshakeResponse
	if err := DecodePayload(respEnv.Payload, &resp); err != nil {
		t.Fatalf("failed to decode response payload: %v", err)
	}

	if resp.Accepted {
		t.Error("expected handshake to be rejected due to version mismatch, but was accepted")
	}
	if !strings.Contains(resp.Error, "incompatible protocol version") {
		t.Errorf("expected incompatible protocol version error message, got: %s", resp.Error)
	}
}

func TestHostServer_HandshakeRequiredBeforeOperations(t *testing.T) {
	clientRead, serverWrite := io.Pipe()
	serverRead, clientWrite := io.Pipe()

	server := NewHostServer(serverRead, serverWrite, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = server.Serve(ctx)
	}()

	clientCodec := NewProtocolCodec(clientRead, clientWrite)

	// Try sending Init without prior Handshake
	err := clientCodec.WriteEnvelope(&Envelope{
		ID:      clientCodec.NextID(),
		Type:    MessageTypeInitRequest,
		Payload: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("failed to write envelope: %v", err)
	}

	respEnv, err := clientCodec.ReadEnvelope()
	if err != nil {
		t.Fatalf("failed to read envelope: %v", err)
	}

	if respEnv.Type != MessageTypeError {
		t.Errorf("expected response type %s, got %s", MessageTypeError, respEnv.Type)
	}
	if !strings.Contains(respEnv.Error, "handshake required") {
		t.Errorf("expected handshake required error, got: %s", respEnv.Error)
	}
}

func TestHostServer_LifecycleAndInvoke(t *testing.T) {
	clientRead, serverWrite := io.Pipe()
	serverRead, clientWrite := io.Pipe()

	plug := &helperTestPlugin{}
	server := NewHostServer(serverRead, serverWrite, plug, nil)
	server.RegisterRoute("uppercase", func(payload []byte) ([]byte, error) {
		return []byte(strings.ToUpper(string(payload))), nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		_ = server.Serve(ctx)
	}()

	clientCodec := NewProtocolCodec(clientRead, clientWrite)

	// 1. Handshake
	hsPayload, _ := EncodePayload(&HandshakeRequest{
		PluginID:        "org.test.plugin",
		PluginVersion:   "1.0.0",
		ManifestVersion: "1.0.0",
		ProtocolVersion: ProtocolVersion,
	})
	_ = clientCodec.WriteEnvelope(&Envelope{
		ID:      clientCodec.NextID(),
		Type:    MessageTypeHandshakeRequest,
		Payload: hsPayload,
	})
	hsRespEnv, err := clientCodec.ReadEnvelope()
	if err != nil {
		t.Fatalf("handshake read failed: %v", err)
	}
	var hsResp HandshakeResponse
	_ = DecodePayload(hsRespEnv.Payload, &hsResp)
	if !hsResp.Accepted {
		t.Fatalf("handshake not accepted: %s", hsResp.Error)
	}

	// 2. Init
	initPayload, _ := EncodePayload(&InitializeRequest{WorkspaceRoot: "/test"})
	_ = clientCodec.WriteEnvelope(&Envelope{
		ID:      clientCodec.NextID(),
		Type:    MessageTypeInitRequest,
		Payload: initPayload,
	})
	initRespEnv, _ := clientCodec.ReadEnvelope()
	var initResp InitializeResponse
	_ = DecodePayload(initRespEnv.Payload, &initResp)
	if !initResp.Success {
		t.Fatalf("init failed: %s", initResp.Error)
	}
	if !plug.initialized {
		t.Error("plugin was not marked initialized")
	}

	// 3. Start
	_ = clientCodec.WriteEnvelope(&Envelope{
		ID:      clientCodec.NextID(),
		Type:    MessageTypeStartRequest,
		Payload: []byte(`{}`),
	})
	startRespEnv, _ := clientCodec.ReadEnvelope()
	var startResp StartResponse
	_ = DecodePayload(startRespEnv.Payload, &startResp)
	if !startResp.Success {
		t.Fatalf("start failed: %s", startResp.Error)
	}
	if !plug.started {
		t.Error("plugin was not marked started")
	}

	// 4. Invoke
	invPayload, _ := EncodePayload(&InvokeRequest{
		Method:  "uppercase",
		Payload: []byte(`"hello world"`),
	})
	_ = clientCodec.WriteEnvelope(&Envelope{
		ID:      clientCodec.NextID(),
		Type:    MessageTypeInvokeRequest,
		Payload: invPayload,
	})
	invRespEnv, _ := clientCodec.ReadEnvelope()
	var invResp InvokeResponse
	_ = DecodePayload(invRespEnv.Payload, &invResp)
	if invResp.Error != "" {
		t.Fatalf("invoke error: %s", invResp.Error)
	}
	if string(invResp.Result) != `"HELLO WORLD"` {
		t.Errorf("expected \"HELLO WORLD\", got %s", string(invResp.Result))
	}

	// 5. Stop
	_ = clientCodec.WriteEnvelope(&Envelope{
		ID:      clientCodec.NextID(),
		Type:    MessageTypeStopRequest,
		Payload: []byte(`{}`),
	})
	stopRespEnv, _ := clientCodec.ReadEnvelope()
	var stopResp StopResponse
	_ = DecodePayload(stopRespEnv.Payload, &stopResp)
	if !stopResp.Success {
		t.Fatalf("stop failed: %s", stopResp.Error)
	}
	if !plug.stopped {
		t.Error("plugin was not marked stopped")
	}

	// 6. Shutdown
	_ = clientCodec.WriteEnvelope(&Envelope{
		ID:      clientCodec.NextID(),
		Type:    MessageTypeShutdownRequest,
		Payload: []byte(`{}`),
	})
	shutdownRespEnv, _ := clientCodec.ReadEnvelope()
	var shutdownResp ShutdownResponse
	_ = DecodePayload(shutdownRespEnv.Payload, &shutdownResp)
	if !shutdownResp.Success {
		t.Fatal("shutdown acknowledge failed")
	}
}
