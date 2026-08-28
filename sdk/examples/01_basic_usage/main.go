package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/unhield/limoxel/sdk"
)

func main() {
	targetWorkspace := "."
	if len(os.Args) > 1 {
		targetWorkspace = os.Args[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Initializing Limoxel SDK (v%s) for workspace: %s\n", sdk.VersionString(), targetWorkspace)

	// Initialize the Limoxel SDK client with workspace configuration
	client, err := sdk.New(sdk.WithWorkspace(targetWorkspace))
	if err != nil {
		log.Fatalf("Failed to initialize Limoxel SDK: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Warning: error closing SDK client: %v", err)
		}
	}()

	// Open and index the workspace repository
	repoInfo, err := client.Repository().Open(ctx, targetWorkspace)
	if err != nil {
		log.Fatalf("Failed to open repository: %v", err)
	}

	fmt.Printf("\n--- Repository Status ---\n")
	fmt.Printf("Name:        %s\n", repoInfo.Name)
	fmt.Printf("Root Path:   %s\n", repoInfo.RootPath)
	fmt.Printf("State:       %s\n", client.Repository().State())
	fmt.Printf("Is Git:      %t\n", repoInfo.IsGit)

	// Query Quantitative Repository Statistics
	stats, err := client.Repository().Statistics(ctx)
	if err != nil {
		log.Fatalf("Failed to retrieve statistics: %v", err)
	}
	fmt.Printf("\n--- Repository Metrics ---\n")
	fmt.Printf("Total Files:     %d\n", stats.TotalFiles)
	fmt.Printf("Total Packages:  %d\n", stats.TotalPackages)
	fmt.Printf("Total Symbols:   %d\n", stats.TotalSymbols)

	// Discover Source Files
	files, err := client.Files().DiscoverFiles(ctx, sdk.FileFilter{Language: "go"}, sdk.PaginationOptions{Limit: 5})
	if err != nil {
		log.Fatalf("Failed to discover files: %v", err)
	}
	fmt.Printf("\n--- Sample Discovered Files (Top 5) ---\n")
	for i, f := range files {
		fmt.Printf("[%d] %s (%d lines, %d bytes)\n", i+1, f.Path, f.Lines, f.Size)
	}

	// Search for Symbols
	searchResults, err := client.Search().SearchSymbols(ctx, "New", sdk.PaginationOptions{Limit: 5})
	if err != nil {
		log.Fatalf("Failed to search symbols: %v", err)
	}
	fmt.Printf("\n--- Symbol Search Results for 'New' (Found: %d) ---\n", searchResults.TotalMatches)
	for i, m := range searchResults.Matches {
		fmt.Printf("[%d] %s | Package: %s | Location: %s\n", i+1, m.Name, m.Package, m.Location)
	}
}
