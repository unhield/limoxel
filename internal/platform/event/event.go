package event

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Event defines the platform event contract interface.
type Event interface {
	ID() string
	Type() Type
	Timestamp() time.Time
	Metadata() Metadata
	Payload() any
	WithMetadata(key, value string) Event
	WithPayload(payload any) Event
	Clone() Event
}

// PlatformEvent is an immutable, thread-safe implementation of Event.
type PlatformEvent struct {
	mu        sync.RWMutex
	id        string
	eventType Type
	timestamp time.Time
	metadata  Metadata
	payload   any
}

// New constructs a new PlatformEvent with an auto-generated timestamp.
func New(id string, eventType Type, payload any) (*PlatformEvent, error) {
	return NewWithMetadata(id, eventType, payload, nil)
}

// NewWithMetadata constructs a PlatformEvent with ID, Type, Payload, and Metadata.
func NewWithMetadata(id string, eventType Type, payload any, meta Metadata) (*PlatformEvent, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrIDEmpty
	}
	if err := ValidateType(eventType); err != nil {
		return nil, err
	}

	var mCopy Metadata
	if meta != nil {
		mCopy = meta.Clone()
	} else {
		mCopy = make(Metadata)
	}

	return &PlatformEvent{
		id:        id,
		eventType: eventType,
		timestamp: time.Now(),
		metadata:  mCopy,
		payload:   payload,
	}, nil
}

// ID returns the event's unique identifier.
func (e *PlatformEvent) ID() string {
	if e == nil {
		return ""
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.id
}

// Type returns the event's classification Type.
func (e *PlatformEvent) Type() Type {
	if e == nil {
		return ""
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.eventType
}

// Timestamp returns the event creation time.
func (e *PlatformEvent) Timestamp() time.Time {
	if e == nil {
		return time.Time{}
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.timestamp
}

// Metadata returns a defensive copy of the event Metadata.
func (e *PlatformEvent) Metadata() Metadata {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.metadata.Clone()
}

// Payload returns the event payload.
func (e *PlatformEvent) Payload() any {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.payload
}

// WithMetadata returns a new immutable PlatformEvent with the added key-value metadata.
func (e *PlatformEvent) WithMetadata(key, value string) Event {
	if e == nil {
		return nil
	}
	if key == "" {
		return e
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	newMeta := e.metadata.Clone()
	if newMeta == nil {
		newMeta = make(Metadata)
	}
	newMeta[key] = value

	return &PlatformEvent{
		id:        e.id,
		eventType: e.eventType,
		timestamp: e.timestamp,
		metadata:  newMeta,
		payload:   e.payload,
	}
}

// WithPayload returns a new immutable PlatformEvent with the updated payload.
func (e *PlatformEvent) WithPayload(payload any) Event {
	if e == nil {
		return nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	return &PlatformEvent{
		id:        e.id,
		eventType: e.eventType,
		timestamp: e.timestamp,
		metadata:  e.metadata.Clone(),
		payload:   payload,
	}
}

// Clone returns a deep defensive copy of the PlatformEvent.
func (e *PlatformEvent) Clone() Event {
	if e == nil {
		return nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	return &PlatformEvent{
		id:        e.id,
		eventType: e.eventType,
		timestamp: e.timestamp,
		metadata:  e.metadata.Clone(),
		payload:   e.payload,
	}
}

// FormatDetails returns a deterministic string representation of the event details.
func (e *PlatformEvent) FormatDetails() string {
	if e == nil {
		return "<nil>"
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("ID: %s\n", e.id))
	sb.WriteString(fmt.Sprintf("Type: %s\n", e.eventType))
	sb.WriteString(fmt.Sprintf("Timestamp: %s\n", e.timestamp.UTC().Format(time.RFC3339Nano)))

	if e.payload != nil {
		sb.WriteString(fmt.Sprintf("Payload: %v\n", e.payload))
	}

	if len(e.metadata) > 0 {
		sb.WriteString("Metadata:\n")
		keys := make([]string, 0, len(e.metadata))
		for k := range e.metadata {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, e.metadata[k]))
		}
	}

	return sb.String()
}
