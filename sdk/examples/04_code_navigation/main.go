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

	fmt.Printf("=== Limoxel Semantic Code Navigation ===\n")
	fmt.Printf("Workspace: %s\n\n", targetWorkspace)

	client, err := sdk.OpenWorkspace(ctx, targetWorkspace)
	if err != nil {
		log.Fatalf("Failed to open workspace: %v", err)
	}
	defer client.Close()

	// 1. Search for a symbol to navigate
	symbols, err := client.Search().SearchSymbols(ctx, "New", sdk.PaginationOptions{Limit: 1})
	if err != nil || len(symbols.Matches) == 0 {
		fmt.Println("No target symbol found for navigation demo.")
		return
	}

	targetSymbol := symbols.Matches[0].Name
	fmt.Printf("Navigating target symbol: %q\n\n", targetSymbol)

	// 2. Go to Definition
	def, err := client.Navigation().GoToDefinition(ctx, targetSymbol)
	if err != nil {
		log.Printf("GoToDefinition error: %v", err)
	} else if def.Target != nil {
		fmt.Printf("--- Go-To-Definition ---\n")
		fmt.Printf("Symbol:   %s\n", def.Target.TargetName)
		fmt.Printf("Kind:     %s\n", def.Target.TargetKind)
		fmt.Printf("File:     %s\n", def.Target.Location.FilePath)
		fmt.Printf("Location: Line %d, Col %d\n\n", def.Target.Location.StartLine, def.Target.Location.StartColumn)
	}

	// 3. Find References
	refs, err := client.Navigation().FindReferences(ctx, targetSymbol, sdk.PaginationOptions{Limit: 10})
	if err != nil {
		log.Printf("FindReferences error: %v", err)
	} else {
		fmt.Printf("--- Reference Search ---\n")
		fmt.Printf("Found %d references for symbol %q:\n", refs.TotalCount, targetSymbol)
		for i, r := range refs.References {
			if i >= 5 {
				break
			}
			fmt.Printf("  [%d] %s:%d\n", i+1, r.Location.FilePath, r.Location.StartLine)
		}
		fmt.Println()
	}

	// 4. Call Hierarchy
	calls, err := client.Navigation().CallHierarchy(ctx, targetSymbol)
	if err != nil {
		log.Printf("CallHierarchy error: %v", err)
	} else if calls != nil {
		fmt.Printf("--- Call Hierarchy ---\n")
		fmt.Printf("Callers (%d):\n", len(calls.Callers))
		for _, c := range calls.Callers {
			fmt.Printf("  <- %s (%s:%d)\n", c.Name, c.Location.FilePath, c.Location.StartLine)
		}
		fmt.Printf("Callees (%d):\n", len(calls.Callees))
		for _, c := range calls.Callees {
			fmt.Printf("  -> %s (%s:%d)\n", c.Name, c.Location.FilePath, c.Location.StartLine)
		}
	}
}
