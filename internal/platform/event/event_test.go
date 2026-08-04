package event_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/unhield/limoxel/internal/platform/event"
)

func TestEventCreationAndGetters(t *testing.T) {
	evt, err := event.New("evt-001", event.TypePlatformStarted, "payload-data")
	if err != nil {
		t.Fatalf("unexpected error creating event: %v", err)
	}

	if evt.ID() != "evt-001" {
		t.Errorf("got ID %q, want evt-001", evt.ID())
	}
	if evt.Type() != event.TypePlatformStarted {
		t.Errorf("got Type %v, want TypePlatformStarted", evt.Type())
	}
	if evt.Payload() != "payload-data" {
		t.Errorf("got Payload %v, want payload-data", evt.Payload())
	}
	if evt.Timestamp().IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if len(evt.Metadata()) != 0 {
		t.Error("expected empty metadata map")
	}
}

func TestEventValidationErrors(t *testing.T) {
	t.Run("empty ID", func(t *testing.T) {
		_, err := event.New("  ", event.TypePlatformCreated, nil)
		if !errors.Is(err, event.ErrIDEmpty) {
			t.Errorf("got error %v, want ErrIDEmpty", err)
		}
	})

	t.Run("empty Type", func(t *testing.T) {
		_, err := event.New("evt-1", "", nil)
		if !errors.Is(err, event.ErrTypeEmpty) {
			t.Errorf("got error %v, want ErrTypeEmpty", err)
		}
	})

	t.Run("invalid Type with spaces", func(t *testing.T) {
		_, err := event.New("evt-1", "invalid type", nil)
		if err == nil || !errors.Is(err, event.ErrTypeEmpty) {
			t.Errorf("got error %v, want ErrTypeEmpty", err)
		}
	})
}

func TestEventImmutabilityAndMetadata(t *testing.T) {
	meta := event.Metadata{"source": "test", "version": "1.0"}
	evt, err := event.NewWithMetadata("evt-002", event.TypeComponentRegistered, 100, meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meta["source"] = "mutated"
	if evt.Metadata()["source"] == "mutated" {
		t.Error("metadata mutation leaked to event instance")
	}

	withMeta := evt.WithMetadata("new_key", "new_val")
	if evt.Metadata()["new_key"] != "" {
		t.Error("original event should not have new_key")
	}
	if withMeta.Metadata()["new_key"] != "new_val" {
		t.Errorf("got %q, want new_val", withMeta.Metadata()["new_key"])
	}

	withPayload := evt.WithPayload(999)
	if evt.Payload() != 100 {
		t.Errorf("original payload got %v, want 100", evt.Payload())
	}
	if withPayload.Payload() != 999 {
		t.Errorf("withPayload got %v, want 999", withPayload.Payload())
	}

	cloned := evt.Clone()
	if cloned.ID() != evt.ID() || cloned.Type() != evt.Type() || cloned.Payload() != evt.Payload() {
		t.Error("cloned event attributes mismatch original")
	}

	details := evt.FormatDetails()
	for _, substr := range []string{"ID: evt-002", "Type: component.registered", "Payload: 100", "source: test", "version: 1.0"} {
		if !strings.Contains(details, substr) {
			t.Errorf("FormatDetails missing %q in:\n%s", substr, details)
		}
	}
}

func TestNilPlatformEventSafety(t *testing.T) {
	var e *event.PlatformEvent

	if e.ID() != "" {
		t.Errorf("got ID %q, want empty", e.ID())
	}
	if e.Type() != "" {
		t.Errorf("got Type %q, want empty", e.Type())
	}
	if !e.Timestamp().IsZero() {
		t.Error("expected zero timestamp")
	}
	if e.Metadata() != nil {
		t.Error("expected nil metadata")
	}
	if e.Payload() != nil {
		t.Error("expected nil payload")
	}
	if e.WithMetadata("k", "v") != nil {
		t.Error("expected nil WithMetadata")
	}
	if e.WithPayload(1) != nil {
		t.Error("expected nil WithPayload")
	}
	if e.Clone() != nil {
		t.Error("expected nil Clone")
	}
	if e.FormatDetails() != "<nil>" {
		t.Errorf("got FormatDetails %q, want <nil>", e.FormatDetails())
	}
}
