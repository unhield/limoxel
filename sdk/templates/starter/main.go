package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/unhield/limoxel/sdk"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize SDK
	client, err := sdk.New(sdk.WithWorkspace("."))
	if err != nil {
		log.Fatalf("Failed to initialize Limoxel SDK: %v", err)
	}
	defer client.Close()

	// Open repository
	repoInfo, err := client.Repository().Open(ctx, ".")
	if err != nil {
		log.Fatalf("Failed to open repository: %v", err)
	}

	fmt.Printf("Successfully initialized Limoxel SDK for: %s (State: %s)\n", repoInfo.Name, client.Repository().State())

	// Example query: Retrieve repository statistics
	stats, err := client.Repository().Statistics(ctx)
	if err != nil {
		log.Fatalf("Failed to query statistics: %v", err)
	}

	fmt.Printf("Discovered %d files and %d packages.\n", stats.TotalFiles, stats.TotalPackages)
}
