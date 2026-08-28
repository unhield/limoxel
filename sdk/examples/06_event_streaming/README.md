# Event Streaming Example

This example demonstrates how to use the Limoxel Event SDK to subscribe to real-time repository lifecycle, indexing, and analysis events via Go channels.

## What It Demonstrates

1. Subscribing to specific event types using `client.Events().Subscribe(ctx, eventTypes...)`
2. Consuming streaming events asynchronously via `<-chan Event`
3. Inspecting event ID, event type, timestamp, and metadata
4. Gracefully canceling subscriptions with the returned `unsubscribe` closure

## Running the Example

```bash
go run ./sdk/examples/06_event_streaming
```

## Expected Output

Outputs real-time log records as repository and analysis events are emitted by the Limoxel core engine.
