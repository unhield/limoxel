package contracts

import (
	"context"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// EventType represents the canonical category of an SDK event.
type EventType string

const (
	// Repository Event Types
	EventTypeRepositoryOpened   EventType = "repository.opened"
	EventTypeRepositoryClosed   EventType = "repository.closed"
	EventTypeRepositoryReloaded EventType = "repository.reloaded"

	// Index Event Types
	EventTypeIndexStarted   EventType = "index.started"
	EventTypeIndexCompleted EventType = "index.completed"
	EventTypeIndexFailed    EventType = "index.failed"

	// Analysis Event Types
	EventTypeAnalysisStarted   EventType = "analysis.started"
	EventTypeAnalysisCompleted EventType = "analysis.completed"

	// Plugin Event Types
	EventTypePluginLoaded   EventType = "plugin.loaded"
	EventTypePluginUnloaded EventType = "plugin.unloaded"

	// Lifecycle Event Types
	EventTypeSDKInitialized EventType = "lifecycle.initialized"
	EventTypeSDKClosed      EventType = "lifecycle.closed"
)

// Event represents an immutable, thread-safe public SDK event.
type Event interface {
	ID() string
	Type() EventType
	Timestamp() time.Time
	Workspace() string
	Payload() any
	Metadata() map[string]string
}

// SDKEvent is the default concrete implementation of Event.
type SDKEvent struct {
	EventID        string            `json:"id"`
	EventTypeValue EventType         `json:"type"`
	EventTimestamp time.Time         `json:"timestamp"`
	WorkspacePath  string            `json:"workspace,omitempty"`
	EventPayload   any               `json:"payload,omitempty"`
	EventMetadata  map[string]string `json:"metadata,omitempty"`
}

func (e SDKEvent) ID() string           { return e.EventID }
func (e SDKEvent) Type() EventType      { return e.EventTypeValue }
func (e SDKEvent) Timestamp() time.Time { return e.EventTimestamp }
func (e SDKEvent) Workspace() string    { return e.WorkspacePath }
func (e SDKEvent) Payload() any         { return e.EventPayload }
func (e SDKEvent) Metadata() map[string]string {
	if e.EventMetadata == nil {
		return nil
	}
	cp := make(map[string]string, len(e.EventMetadata))
	for k, v := range e.EventMetadata {
		cp[k] = v
	}
	return cp
}

// EventHandler defines the callback signature for asynchronous event notifications.
type EventHandler func(evt Event)

// Subscription represents an active event subscription that can be cancelled.
type Subscription interface {
	ID() string
	Unsubscribe() error
}

// EventContract defines the public contract for event publishing, subscription, and streaming.
type EventContract interface {
	Contract
	Publish(ctx context.Context, evt Event) error
	Subscribe(ctx context.Context, eventType EventType, handler EventHandler) (Subscription, error)
	SubscribeAll(ctx context.Context, handler EventHandler) (Subscription, error)
	Events(ctx context.Context, eventType string) (<-chan Event, error)
}

// DefaultEventContractMetadata returns default contract descriptor for Event operations.
func DefaultEventContractMetadata() BaseContract {
	return NewBaseContract(
		"EventContract",
		lifecycle.CapabilityIntelligence,
		version.SemVer{Major: 1, Minor: 0, Patch: 0},
		lifecycle.StateSupported,
		"Provides public subscription, publication, and streaming for repository, index, analysis, plugin, and lifecycle events.",
	)
}
