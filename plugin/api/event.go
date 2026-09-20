package api

import "context"

// EventAPI defines the developer-facing interface for event pub/sub operations.
type EventAPI interface {
	// Publish broadcasts an event across the Limoxel event broker.
	Publish(ctx context.Context, evt Event) error

	// Subscribe attaches a handler for a specific event type or wildcard topic.
	Subscribe(ctx context.Context, eventType string, handler EventHandler) (Subscription, error)

	// SubscribeAll registers a global handler for all dispatched events.
	SubscribeAll(ctx context.Context, handler EventHandler) (Subscription, error)
}
