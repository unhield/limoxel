package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/unhield/limoxel/sdk"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Printf("=== Limoxel Event Streaming & Monitoring ===\n\n")

	client, err := sdk.New(sdk.WithWorkspace("."))
	if err != nil {
		log.Fatalf("Failed to initialize SDK: %v", err)
	}
	defer client.Close()

	// Handler function for event notifications
	eventHandler := func(ev sdk.Event) {
		fmt.Printf(" [EVENT RECEIVED] ID: %-8s | Type: %-25s | Time: %s\n",
			ev.ID(), ev.Type(), ev.Timestamp().Format(time.RFC3339))
	}

	// Subscribe to repository, indexing, and analysis events
	sub1, err := client.Events().Subscribe(ctx, sdk.EventTypeRepositoryOpened, eventHandler)
	if err != nil {
		log.Fatalf("Failed to subscribe to repository opened events: %v", err)
	}
	defer sub1.Unsubscribe()

	sub2, err := client.Events().Subscribe(ctx, sdk.EventTypeAnalysisCompleted, eventHandler)
	if err != nil {
		log.Fatalf("Failed to subscribe to analysis events: %v", err)
	}
	defer sub2.Unsubscribe()

	fmt.Println("Active event subscriptions registered.")

	// Trigger lifecycle event by opening workspace
	fmt.Println("Triggering repository lifecycle event by opening workspace...")
	_, _ = client.Repository().Open(ctx, ".")

	// Allow event to process through channel
	time.Sleep(500 * time.Millisecond)

	// Trigger analysis event
	fmt.Println("Triggering analysis event...")
	_, _ = client.Analysis().RepositoryHealth(ctx)

	time.Sleep(500 * time.Millisecond)
	fmt.Println("\nEvent streaming demo completed successfully.")
}
