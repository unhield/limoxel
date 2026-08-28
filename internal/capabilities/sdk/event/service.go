package event

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	sdkerr "github.com/unhield/limoxel/internal/capabilities/sdk/errors"
)

type subscriber struct {
	id        string
	eventType contracts.EventType
	handler   contracts.EventHandler
	ch        chan contracts.Event
}

type subscription struct {
	id      string
	service *Service
}

func (s *subscription) ID() string {
	return s.id
}

func (s *subscription) Unsubscribe() error {
	if s.service == nil {
		return nil
	}
	return s.service.unsubscribe(s.id)
}

// Service manages thread-safe event publication, subscription, and streaming across SDK capabilities.
type Service struct {
	mu sync.RWMutex
	contracts.BaseContract
	subscribers map[string]*subscriber
	counter     uint64
	closed      bool
}

// Ensure Service implements EventContract.
var _ contracts.EventContract = (*Service)(nil)

// NewService constructs an initialized Event SDK service adapter.
func NewService() *Service {
	return &Service{
		BaseContract: contracts.DefaultEventContractMetadata(),
		subscribers:  make(map[string]*subscriber),
	}
}

// Publish broadcasts an event to all matching subscribers.
func (s *Service) Publish(ctx context.Context, evt contracts.Event) error {
	if s == nil {
		return sdkerr.NewUnavailable("EventService", "event service is nil")
	}
	if evt == nil {
		return sdkerr.NewInvalidInput("event cannot be nil")
	}
	if err := ctx.Err(); err != nil {
		return sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return sdkerr.NewInvalidState("CLOSED", "event service is closed")
	}

	for _, sub := range s.subscribers {
		if sub == nil {
			continue
		}
		if sub.eventType == "" || sub.eventType == evt.Type() {
			if sub.handler != nil {
				h := sub.handler
				go func(handler contracts.EventHandler, event contracts.Event) {
					defer func() {
						_ = recover()
					}()
					handler(event)
				}(h, evt)
			}
			if sub.ch != nil {
				select {
				case sub.ch <- evt:
				default:
					// Non-blocking drop if channel buffer full to prevent deadlocks
				}
			}
		}
	}

	return nil
}

// Subscribe registers a callback handler for a specific event type.
func (s *Service) Subscribe(ctx context.Context, eventType contracts.EventType, handler contracts.EventHandler) (contracts.Subscription, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("EventService", "event service is nil")
	}
	if handler == nil {
		return nil, sdkerr.NewInvalidInput("handler cannot be nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, sdkerr.NewInvalidState("CLOSED", "event service is closed")
	}

	subID := fmt.Sprintf("sub_%d", atomic.AddUint64(&s.counter, 1))
	sub := &subscriber{
		id:        subID,
		eventType: eventType,
		handler:   handler,
	}
	s.subscribers[subID] = sub

	return &subscription{
		id:      subID,
		service: s,
	}, nil
}

// SubscribeAll registers a callback handler receiving all published events.
func (s *Service) SubscribeAll(ctx context.Context, handler contracts.EventHandler) (contracts.Subscription, error) {
	return s.Subscribe(ctx, "", handler)
}

// Events returns a streaming channel receiving events matching the specified event type filter.
func (s *Service) Events(ctx context.Context, eventType string) (<-chan contracts.Event, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("EventService", "event service is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, sdkerr.NewInvalidState("CLOSED", "event service is closed")
	}

	ch := make(chan contracts.Event, 64)
	subID := fmt.Sprintf("chan_sub_%d", atomic.AddUint64(&s.counter, 1))
	sub := &subscriber{
		id:        subID,
		eventType: contracts.EventType(eventType),
		ch:        ch,
	}
	s.subscribers[subID] = sub

	// Automatically clean up subscription when context is done
	go func() {
		<-ctx.Done()
		_ = s.unsubscribe(subID)
	}()

	return ch, nil
}

// Close gracefully closes the event service and removes all subscribers.
func (s *Service) Close() error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed = true
	for _, sub := range s.subscribers {
		if sub != nil && sub.ch != nil {
			close(sub.ch)
		}
	}
	s.subscribers = make(map[string]*subscriber)
	return nil
}

func (s *Service) unsubscribe(subID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, exists := s.subscribers[subID]
	if !exists {
		return nil
	}

	if sub.ch != nil {
		close(sub.ch)
	}
	delete(s.subscribers, subID)
	return nil
}

// EmitLifecycleEvent helper constructs and publishes a lifecycle event.
func EmitLifecycleEvent(ctx context.Context, s *Service, eventType contracts.EventType, workspace string, payload any) error {
	if s == nil {
		return nil
	}
	evt := contracts.SDKEvent{
		EventID:        fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		EventTypeValue: eventType,
		EventTimestamp: time.Now().UTC(),
		WorkspacePath:  workspace,
		EventPayload:   payload,
	}
	return s.Publish(ctx, evt)
}
