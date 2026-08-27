package event_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	eventsdk "github.com/unhield/limoxel/internal/capabilities/sdk/event"
)

func TestEventServiceOperations(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc := eventsdk.NewService()

	var count uint64

	// 1. Subscribe specific event type
	sub, err := svc.Subscribe(ctx, contracts.EventTypeRepositoryOpened, func(evt contracts.Event) {
		atomic.AddUint64(&count, 1)
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	if sub.ID() == "" {
		t.Errorf("expected non-empty subscription ID")
	}

	// 2. Publish matching event
	evt1 := contracts.SDKEvent{
		EventID:        "evt_1",
		EventTypeValue: contracts.EventTypeRepositoryOpened,
		EventTimestamp: time.Now().UTC(),
		WorkspacePath:  "sample_repo",
	}
	if err := svc.Publish(ctx, evt1); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if atomic.LoadUint64(&count) != 1 {
		t.Errorf("expected count=1, got %d", atomic.LoadUint64(&count))
	}

	// 3. Publish non-matching event
	evt2 := contracts.SDKEvent{
		EventID:        "evt_2",
		EventTypeValue: contracts.EventTypeIndexStarted,
		EventTimestamp: time.Now().UTC(),
	}
	if err := svc.Publish(ctx, evt2); err != nil {
		t.Fatalf("Publish non-matching failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if atomic.LoadUint64(&count) != 1 {
		t.Errorf("expected count to remain 1, got %d", atomic.LoadUint64(&count))
	}

	// 4. SubscribeAll
	var allCount uint64
	allSub, err := svc.SubscribeAll(ctx, func(evt contracts.Event) {
		atomic.AddUint64(&allCount, 1)
	})
	if err != nil {
		t.Fatalf("SubscribeAll failed: %v", err)
	}

	if err := svc.Publish(ctx, evt1); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	if err := svc.Publish(ctx, evt2); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if atomic.LoadUint64(&allCount) != 2 {
		t.Errorf("expected allCount=2, got %d", atomic.LoadUint64(&allCount))
	}

	// 5. Unsubscribe
	if err := sub.Unsubscribe(); err != nil {
		t.Fatalf("Unsubscribe failed: %v", err)
	}
	if err := allSub.Unsubscribe(); err != nil {
		t.Fatalf("allSub Unsubscribe failed: %v", err)
	}

	// 6. Events streaming channel
	ch, err := svc.Events(ctx, string(contracts.EventTypeIndexStarted))
	if err != nil {
		t.Fatalf("Events channel failed: %v", err)
	}

	if err := svc.Publish(ctx, evt2); err != nil {
		t.Fatalf("Publish to stream failed: %v", err)
	}

	select {
	case received := <-ch:
		if received.ID() != "evt_2" {
			t.Errorf("expected evt_2, got %s", received.ID())
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("timeout waiting for event on streaming channel")
	}

	// 7. Close
	if err := svc.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Publish after close should fail
	if err := svc.Publish(ctx, evt1); err == nil {
		t.Errorf("expected error on Publish after Close")
	}
}

func TestEventServiceErrorsAndNil(t *testing.T) {
	ctx := context.Background()
	svc := eventsdk.NewService()

	if err := svc.Publish(ctx, nil); err == nil {
		t.Errorf("expected error for nil event")
	}
	if _, err := svc.Subscribe(ctx, "test", nil); err == nil {
		t.Errorf("expected error for nil handler")
	}

	var nilSvc *eventsdk.Service
	if err := nilSvc.Publish(ctx, contracts.SDKEvent{}); err == nil {
		t.Errorf("expected error on typed nil service Publish")
	}
	if _, err := nilSvc.Subscribe(ctx, "test", func(contracts.Event) {}); err == nil {
		t.Errorf("expected error on typed nil service Subscribe")
	}
	if _, err := nilSvc.SubscribeAll(ctx, func(contracts.Event) {}); err == nil {
		t.Errorf("expected error on typed nil service SubscribeAll")
	}
	if _, err := nilSvc.Events(ctx, "test"); err == nil {
		t.Errorf("expected error on typed nil service Events")
	}
	if err := nilSvc.Close(); err != nil {
		t.Errorf("unexpected error on nil service Close: %v", err)
	}
}

func TestEventServicePanicIsolationAndConcurrency(t *testing.T) {
	ctx := context.Background()
	svc := eventsdk.NewService()

	var successCount uint64

	// Sub 1: Panics intentionally
	_, err := svc.Subscribe(ctx, contracts.EventTypePluginLoaded, func(evt contracts.Event) {
		panic("intentional subscriber panic")
	})
	if err != nil {
		t.Fatalf("Subscribe panicking handler failed: %v", err)
	}

	// Sub 2: Should still receive events despite Sub 1 panicking
	_, err = svc.Subscribe(ctx, contracts.EventTypePluginLoaded, func(evt contracts.Event) {
		atomic.AddUint64(&successCount, 1)
	})
	if err != nil {
		t.Fatalf("Subscribe normal handler failed: %v", err)
	}

	evt := contracts.SDKEvent{
		EventID:        "plugin_1",
		EventTypeValue: contracts.EventTypePluginLoaded,
		EventTimestamp: time.Now().UTC(),
	}

	if err := svc.Publish(ctx, evt); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if atomic.LoadUint64(&successCount) != 1 {
		t.Errorf("expected successCount=1 despite sibling panic, got %d", atomic.LoadUint64(&successCount))
	}
}
